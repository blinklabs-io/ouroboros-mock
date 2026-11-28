# Reproducible peer experiments

The `benchmark` package and `ouroboros-mock benchmark` command run reactive
node-to-node peers from a checksummed experiment. The controller owns node
startup, implementation configuration, process/resource sampling, clock
calibration and outcome analysis. No node implementation is imported here.
Existing YAML conversations, fixture constructors and ordinary peer `Close`
remain available.

## Experiment inputs

`testdata/reference` contains capture-once Conway data from Cardano node 11.2.0:
164 full blocks on two independently produced branches, 12 distinct transactions,
funded genesis state, staking configuration, public test credentials, the exact
protocol parameters, real Plutus cost models and a Plutus V3 loop workload.
The workload includes four confirmed value transfers and four confirmed script
spends with successful phase-2 results. Every script spend has collateral and
explicit memory/step budgets. `node-expected.json` preserves transaction inputs,
collateral, redeemers, fees, outputs, inclusion and branch order. Pausing one
producer for five seconds created an actual empty-slot range. The two short
fork prefixes stay within security parameter 20.

These are immutable captures, not a claim that a random seed regenerates block
signatures, transaction IDs or timings. Every artifact has its byte length and
SHA-256 identity in each manifest. `node-oracle.json` identifies the reference
version, image digest, method and input identities. Genesis, parameter and
transaction bytes are authoritative; linkage and CBOR decoding alone do not
establish ledger or consensus validity. Manifests distinguish `decoder`,
`ledger` and `consensus` expectations and require independent matching oracles.

The credentials are deliberately public keys for this isolated test network,
including signing material. They exist only to reproduce this funded state.
The fresh-state policy means a new database derived from the declared genesis,
not a database image shared between implementations. Historical signed chains
retain their original genesis clock and serve validation/replay experiments;
they cannot be retimed into a live forging or slot-propagation trial. Live
trials require their own validated network inputs and freshly produced blocks. The cache declaration
specifies the process/database policy; the controller must also record its host
cache treatment. Diagnostic selection explicitly requests snapshot and ledger
measurements; the controller must configure the target accordingly and verify
the resulting observations.

All files are rooted at the manifest's directory. Loading rejects escaping
paths and symlinks, nonregular files, wrong sizes/digests, duplicate identities,
unknown fields or versions, dependency cycles, missing expected cases, and
inconsistent/empty cost models. Payloads are reverified when read. Actions are
validated against the complete selected chain, including earlier append/fork
operations. Transaction artifact dependencies name funding and collateral
predecessors; an offer action does not assert that the target has accepted them.
The controller supplies equivalent prerequisite state before those offers.

## Controller protocol

Build the existing command with Go, then run a role-specific manifest:

```sh
go build -o ouroboros-mock ./cmd/ouroboros-mock
./ouroboros-mock benchmark \
  --manifest benchmark/testdata/reference/upstream.json \
  --run-id trial-001 --endpoint upstream=127.0.0.1:0
```

Benchmark control input uses an interruptible POSIX pipe. Keep stdin open for
the trial. Read JSON lines from stdout continuously and send one JSON line on
stdin after all `ready` records have arrived:

```json
{"version":1,"kind":"start","run":"trial-001","start":"2026-01-01T00:00:00.000000000Z"}
```

Supply the actual intended UTC start time, normally a future time shared with
other participants. Scheduling uses that absolute start plus each `at_ns`.
Action offsets are strictly before `duration_ns`; dispatch at or after the end
is an error for every action kind. Fetch action records retain the request ID.
Records retain both scheduled offsets and actual call times, so a late
controller start is visible. The example timestamp illustrates the format.
An explicit `stop` control, input EOF, malformed/mismatched control, signal,
connection failure or output failure terminates with an error and invalidates
the recording. A successful run lasts the manifest's entire duration.

Every role declares its target identity and TCP transport. Endpoint flags assign
actual addresses to those roles and cannot name unknown roles. Upstream accepts
multiple sessions; `ready.endpoint` contains its bound address. Other roles
dial the node, each on its own connection:

| Manifest | Role | Behavior |
| --- | --- | --- |
| `upstream.json` | `upstream` | Serve the full selected value branch reactively. |
| `fork.json` | `upstream` | Switch from eight blocks to a competing twelve-block branch. |
| `follower.json` | `follower` | Observe headers and rollbacks without fetching bodies. |
| `fetcher.json` | `fetcher` | Fetch three blocks spanning real empty slots. |
| `txpeer.json` | `txpeer` | Offer a funded import transaction and observe actual node pulls. |

The example manifests share complete corpus and metric expectations. A
controller can construct another checked manifest to select the Plutus branch,
combine roles, repeat offers, request overlapping ranges or append blocks.
`after` refers to completion of an earlier scheduled API action. It does not
mean asynchronous range completion, transaction acceptance or chain adoption.
Those have separate observations/outcome expectations. Actions at the same
scheduled time run in declaration order. The command does not reconnect
implicitly; a new process has a new run identity, and each upstream connection
has a distinct session identity.

## Observation and finalization semantics

`prepared` records identify the exact manifest, executable digest source, module
and Go toolchain. `go_vcs_revision` and `go_vcs_modified` retain the compiler's
reported VCS diagnostics, which can describe an enclosing checkout in linked
worktrees. On Linux, `executable_sha256` hashes the running image through
`/proc/self/exe`,
even if its pathname is replaced. Other platforms report
`executable_path_sha256` with `digest_source: "pathname"`; this is a pathname
diagnostic and does not attest the running image. The controller records the
build artifact hash and links that digest to its independently recorded source/build provenance.
Peer records carry occurrence time, publication
sequence, session, connection and request identities where the protocol has an
exchange, full points/parents, block number, byte size and actual item counts.
`transaction_refs` retains era plus full hash; the hash-only `transactions`
list is not the complete wire identity. UTC timestamps always retain nine fractional digits.
Publication order and
occurrence order are separate, especially across concurrent connections.

Boundaries describe what is observable at this layer:

- `roll_forward` is a decoded-header callback for a follower, or a server header
  send-call boundary for an upstream. The advertised tip is never substituted
  for that header's identity.
- `fetch_api_start` is API entry, not the eventual mux write. `block_fetched`
  follows successful decoding, before any target ledger validation. Fetch
  completion retains its request ID, including cancellation and no-block cases.
- `block_send_call` records the server API call and its result, not a claimed
  completed socket write. `batch_done.count` counts blocks, not slots.
- `action` is the scheduled operation's call/result. `txs_offered` is actual
  insertion into the local offer queue. `tx_ids_requested` and `txs_requested`
  are the node's request callbacks. `tx_bodies_returned` separately records the
  prepared body reply and its actual item population. It is not a wire-write
  completion.
  `tx_ids_offered` and `txs_received` describe the upstream relay side.
  `tx_receive_failed` retains the body-request error and wanted identities when
  protocol validation prevents a successful body reply.
  Serving a body is distinct from target mempool acceptance or inclusion.

Wire-write, target admission, chain selection and native forging phases belong
to their actual owners. The coverage contract requires those sources; an API
timestamp cannot satisfy a wire-write requirement. Native phases retain their
own meaning, precomputed intervals and ordinary nonleader/losing-candidate
outcomes. Go GC data is never renamed as GHC major-GC live bytes.

The queue remains bounded at 256 observations. Each attempted publication gets
a sequence number; overflow increments drops. Source payloads are copied before
queuing. Nothing writes files or blocks on export inside a protocol callback.
Opt-in `Finish(ctx)` stops producers/connections and drains while the caller
consumes `Events`. `source_final` is written only after collectors have drained
and reports produced/delivered/dropped/discarded counts. Control monitoring stays
active through this drain. The runner owns its control and output streams;
closing either must interrupt pending I/O. Finalization has a ten-second budget
bounded further by the caller deadline; an interrupted write invalidates it.
Ordinary `Close` still
discards unread observations. Overflow, an interrupted tail or an early ordinary
close cannot certify complete observations.

`final.recording_complete` certifies this process's capture, action execution,
minimum declared event/item populations and successful output writes. These
minimums cover the entire process capture, including preparation and drain.
Measurement-window populations are checked by the controller using occurrence
timestamps and the declared window. It is not a ledger verdict or a full experiment pass. The controller must additionally
validate node outcomes, all required observables, cross-process finalization,
clock quality and resource windows. A missing final record or nonzero process
exit is always an incomplete recording. A failed writer may prevent even the
failure record from being emitted, so exit status must also be checked.

## Measurement corpus

`observables.schema.json`, `envelope.schema.json` and `canonical.jsonl` retain
the common specification's typed catalogue and examples, including both
snapshot edges. `coverage.json` specifies source authority, unit, boundary, native meaning and
positive/error/quality cases for all 25 observable entries and all 23 derived
families. `observations.json` and `metric-expected.json` contain 44 constructed
metric cases with independent arithmetic/state/population derivations in
`metric-oracle.json`. They are exact semantic goldens, not measured performance
numbers. `raw-events.json` and `normalized-expected.json` independently fix the
native peer conversion, full identities, duplicate-peer population and
occurrence versus serialization ordering.

The goldens include signed-candidate outcomes, precomputation, batching,
empty-slot ranges, retries, duplicate peers/transactions, fork re-inclusion,
missing pairs, censored nonadopters, CPU windows and zero denominators,
snapshot completion/duration, major-GC windows, atomic mempool occupancy and
same-generation ledger size/density. Lost records, writer failure, absent final
records, restart boundaries, clock uncertainty, sampler overruns and checker
panics are explicit invalid cases. Resource and timing values from an actual
run must be measured afresh rather than copied from these goldens.
