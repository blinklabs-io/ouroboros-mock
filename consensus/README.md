# consensus

Consensus-conformance capture + replay harness for Ouroboros nodes.

Records cardano-node's served chainsync `roll_forward` and
`roll_backward` responses into JSON test vectors. cardano-node is the
oracle. The format defines schema for `find_intersect`,
`intersect_found`, `await_reply`, blockfetch messages, etc., but the
recorder only wires the two `RollForward*` / `RollBackward*` chainsync
callbacks today.

A small replay harness loads those vectors and drives them against a
caller-supplied `ChainSelector` implementation. Each node
implementation adapts its chain-selection surface to the harness
interface and uses the harness from its own test suite.

## Layout

```text
consensus/
  format/                    Go package: TestVector type + JSON codecs
  recorder.go                Recorder: callback-driven capture buffer
  conversation.go            capture-conversation.json loader + steps
                             (find_intersect / request_next / drain_to_tip)
  sidecar.go                 Sidecar runtime (connection + driver loop)
  emit.go                    WriteVector helper
  compose.go                 Multi-peer vector composition (used by
                             cmd/compose-consensus-vector)
  diff.go                    Structural-tolerance golden diff
  harness.go                 Replay harness: LoadVector + RunVector
                             against a caller-supplied ChainSelector
  sidecar_test.go            Offline round-trip + recorder tests
  live_capture_test.go       Build-tag-gated end-to-end smoke test
  golden_test.go             Build-tag-gated golden-corpus assertions
  cmd/
    capture-sidecar/         Binary that does one capture run
    compose-consensus-vector/ Binary that merges N single-peer captures
                              into one multi-peer vector + golden diff
  Dockerfile.configurator    Shared base image (genesis toolchain)
  Dockerfile.capture_sidecar Shared base image (Go build of
                             cmd/capture-sidecar)
  Dockerfile.compose_consensus_vector
                             Shared base image (Go build of
                             cmd/compose-consensus-vector)
  capture-scenario.sh        Dispatcher: forwards to scenarios/<n>/run.sh
  capture-all.sh             Bulk wrapper: runs every scenario, writes
                             each to testdata/captured/<n>.json
  scenarios/
    intersect_origin_one_rollforward/   Single-peer smoke test
    fork_and_select_v1/                 Two-peer fork-and-select scenario
  testdata/
    fixtures/                Hand-crafted vectors for format/ tests
    captured/                Committed goldens from live captures
```

Adding a scenario means dropping in a new `scenarios/<name>/`
directory. The shared base does not change.

## Running a scenario

```bash
./capture-scenario.sh intersect_origin_one_rollforward -out /tmp/vector.json
```

The dispatcher resolves `scenarios/<name>/run.sh` and runs it. Each
scenario owns its own orchestration shape (number of cardano peers,
configurator behavior, number of sidecar invocations, whether a
composer + golden diff runs at the end) so the dispatcher itself
stays thin — its one job beyond dispatch is to **re-roll the forge**:
because captures are nondeterministic, set `CAPTURE_RETRIES=N` to retry
`run.sh` up to N times until it produces a shape-valid vector (each
attempt is a fresh forge; `run.sh` tears down with `down -v` on exit).

See each scenario's `README.md` for what it captures and how to run it
directly. Existing scenarios:

| Scenario | Peers | What it tests |
|---|---|---|
| `intersect_origin_one_rollforward` | 1 | Smoke-test: chainsync from origin captures the standard roll_backward (to origin) followed by one roll_forward (the first forged block) |
| `fork_and_select_v1` | 2 | Praos chain selection + rollback to non-genesis intersect across two divergent chains with a shared prefix |

Multi-peer scenarios use the `cmd/compose-consensus-vector` binary to
merge per-peer captures into the multi-peer vector and diff against
the committed golden (structural-tolerance match per
`consensus/diff.go`).

To regenerate the entire committed corpus in one go (every scenario,
each writing to `testdata/captured/<name>.json`), use the bulk
wrapper:

```bash
./capture-all.sh                                   # all scenarios
./capture-all.sh --only intersect_origin_one_rollforward
./capture-all.sh --fail-fast                       # stop on first failure
./capture-all.sh --retries 50                      # re-roll harder
```

The wrapper passes `--skip-golden` to every scenario so existing
goldens don't block regeneration — that's the whole point of running
it. Scenarios with no golden accept the flag as a no-op.

Regeneration is **safe**. Before committing, each `run.sh` validates the
captured vector against its scenario's intended SHAPE (switch / no-switch /
VRF tie / single) with `cmd/check-consensus-vector`, which decodes the real
block headers and checks rollback depth, tip lead, the 5-slot restricted-
tiebreaker window, peer feed order, and `local_tip` / `expected_rollback`
presence. A capture that drifts out of shape — a length fork forged where a
tie was intended, a switch the SUT can't reach, an exceeds-k incumbent that
isn't actually > k deep — **fails without overwriting the committed golden**,
and the forge is re-rolled (up to `--retries N`, default 30). So
`./capture-all.sh` either refreshes the corpus with shape-correct vectors or
leaves it untouched; it cannot silently regenerate a wrong-but-green vector
the way a bare composer run could. The composer also feeds the
winner/incumbent peer in the order the harness's switch assertion needs, so
ordering no longer depends on which pool the VRF lottery favored.

## Vector format

JSON, schema-versioned. Binary fields (header bytes, block bytes,
hashes) are hex-encoded into JSON strings so vectors stay diffable.

Each vector carries per-peer captured `chainsync` / `blockfetch`
traces as inputs and an `expected_output` with two sides: a wire-level
chainsync trace and a structured chain tip.

See `format/vector.go` for the Go shape and `testdata/fixtures/` for
hand-crafted examples.

## Replay harness

```go
import (
    "testing"

    "github.com/blinklabs-io/ouroboros-mock/consensus"
    "github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

// 1. Adapt your chain-selection implementation to the harness's Replayer
//    interface. peerID is the vector's peer_id (a uint64); your adapter is
//    free to map it to whatever internal peer-routing type your selector uses.
type myAdapter struct{ /* ... */ }

func (a *myAdapter) RollForward(peerID uint64, era uint, headerCbor []byte, tip format.Tip) error { /* ... */ }
func (a *myAdapter) RollBackward(peerID uint64, point format.Point, tip format.Tip) error { /* ... */ }
func (a *myAdapter) Stabilize()                              { /* drive the selector to a quiescent decision */ }
func (a *myAdapter) BestTip() (format.Tip, bool)             { /* ... */ }
func (a *myAdapter) DrainSwitchEvents() []format.SwitchEvent { /* ... */ }

// 2. Run the embedded corpus. The factory is called once per subtest with
//    that vector's capture, so the adapter can configure k (security_param)
//    and any pre-seeded local_tip before replay; each vector replays against
//    a fresh selector.
func TestConsensusConformance(t *testing.T) {
    consensus.RunAllCapturedVectors(t,
        func(capture *format.ConsensusCapture) consensus.Replayer {
            return newMyAdapter(capture.SecurityParam, capture.LocalTip)
        })
}
```

The harness replays every peer's full served trace in order — each
`roll_forward` / `roll_backward` delivered through the matching `Replayer`
method — then calls `Stabilize` and asserts `BestTip` matches
`expected_output.final_tip` (slot + hash + block_number). When the vector
carries an `expected_rollback`, it additionally asserts the SUT emitted a
switch onto `final_tip` off a shorter-or-equal-length peer (via
`DrainSwitchEvents`). The rollback *point* itself is not replayed-verified, and
`expected_output.downstream_chainsync` is recorded but not asserted during
replay — both are checked structurally at capture/compose time only.

For ad-hoc iteration outside the harness's subtest loop, use
`consensus.CapturedVectors()` to get the embedded corpus as
`[]CapturedVector` and `consensus.LoadVector(path)` to decode a
vector from disk.

## Tests

```bash
# Fast: offline format + recorder + composer + diff tests, plus the
# golden-corpus structural assertions.
go test ./consensus/...

# Slow: live end-to-end capture (docker required).
go test -tags consensuscapture -run TestCaptureScenarioLiveStack \
    ./consensus/...
```

## Recording layer

The capture sidecar records via gouroboros's decoded protocol
callbacks (`RollForwardRawFunc`, `RollBackwardFunc`) so it does not
touch gouroboros internals. Raw header / block bytes flow through to
the vector's `header_cbor` / `block_cbor` fields untouched; envelope
fields (slot, hash, era, tip, points) are populated from the callback
arguments as structured JSON.
