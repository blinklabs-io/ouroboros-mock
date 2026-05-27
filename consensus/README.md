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

The dispatcher resolves `scenarios/<name>/run.sh` and execs it. Each
scenario owns its own orchestration shape (number of cardano peers,
configurator behavior, number of sidecar invocations, whether a
composer + golden diff runs at the end) so the dispatcher itself
stays trivial.

See each scenario's `README.md` for what it captures and how to run it
directly. Existing scenarios:

| Scenario | Peers | What it tests |
|---|---|---|
| `intersect_origin_one_rollforward` | 1 | Smoke-test: handshake → find_intersect[origin] → roll_backward → roll_forward |
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
```

The wrapper passes `--skip-golden` to every scenario so existing
goldens don't block regeneration — that's the whole point of running
it. Scenarios with no golden accept the flag as a no-op.

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
import "github.com/blinklabs-io/ouroboros-mock/consensus"

// 1. Adapt your chain-selection implementation to the harness's
//    ChainSelector interface.
type myAdapter struct{ /* ... */ }
func (a *myAdapter) UpdatePeerTip(id ConnectionId, tip Tip, vrf []byte) bool { ... }
func (a *myAdapter) EvaluateAndSwitch() { ... }
func (a *myAdapter) BestPeerTip() (Tip, bool) { ... }

// 2. Iterate the committed corpus.
for _, path := range consensus.CapturedVectorPaths() {
    t.Run(filepath.Base(path), func(t *testing.T) {
        v, err := consensus.LoadVector(path)
        if err != nil { t.Fatal(err) }
        if err := consensus.RunConsensusVector(t, v, &myAdapter{}); err != nil {
            t.Fatalf("%s: %v", v.Title, err)
        }
    })
}
```

The harness derives each peer's last `roll_forward` tip from the
served trace, feeds it to the adapter via `UpdatePeerTip`, calls
`EvaluateAndSwitch`, and asserts the adapter's best tip matches
`expected_output.final_tip` (slot + hash + block_number).

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
