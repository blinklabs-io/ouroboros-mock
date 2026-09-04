# Test Vector Format Reference

This document describes the legacy CBOR envelope and the Cardano Blueprint
JSON records consumed by this package. See [CORPUS.md](CORPUS.md) for the
pinned source and refresh procedure.

Vectors are generated under `testdata/eras/conway/impl/dump/`. Blueprint
records are JSON files with hex-encoded `cbor`, `oldLedgerState`, and
`newLedgerState` fields, plus `success` and `testState`. Protocol parameter
files are in `testdata/eras/conway/impl/dump/pparams-by-hash/`.

---

## Legacy CBOR envelope

The legacy CBOR files decode to a 5-element array:

```
vector = [
    config,         ; [0] network/epoch configuration
    initial_state,  ; [1] NewEpochState before events
    final_state,    ; [2] NewEpochState after events
    events,         ; [3] array of events
    title,          ; [4] test name (text string)
]
```

The `title` field identifies the source test. This envelope remains supported
for locally authored synthetic fixtures.

## Cardano Blueprint JSON records

The pinned Blueprint archive is the primary ledger corpus. Each record contains
hex-encoded `cbor`, `oldLedgerState`, and `newLedgerState` fields, a boolean
`success`, and a `testState` title. Blueprint exports `LedgerState`, not
`NewEpochState`, so the adapter wraps each state in the legacy shape while
preserving the source CBOR bytes. The export does not contain
`NewEpochState.epoch_no` or its event timeline. The adapter restores the
imported corpus's default execution epoch (899) and records the known
timeline-derived exception for `GOV.Voting.expired_gov-actions/5` (epoch 902).
These values are adapter metadata, not changes to the Blueprint bytes; refresh
them from the legacy event envelope when the pinned Blueprint revision changes.

Blueprint UTxO maps use the ledger's compact binary representation. The parser
decodes its tagged, length-prefixed address and variable-length coin directly;
it must not materialize the complete state as `cbor.Value`, because reference
script vectors can exceed the Go stack's recursive decoding limit.

---

## Config Array (index 0)

The config array holds simplified network parameters — not full protocol parameters. Full protocol parameters are loaded separately from `pparams-by-hash/` using the hash embedded in `initial_state`.

```
config = [
    start_slot,        ; [0]  uint64 — epoch start slot
    slot_length,       ; [1]  uint64 — milliseconds per slot
    epoch_length,      ; [2]  uint64 — slots per epoch
    security_param,    ; [3]  uint64 — security parameter (k)
    active_slots,      ; [4]  uint64 — active slots coefficient denominator
    network_id,        ; [5]  uint64 — 0 = testnet/preview, 1 = mainnet
    pool_stake,        ; [6]  uint64 — pool stake (scaled)
    _unknown_7,        ; [7]  uint64
    _unknown_8,        ; [8]  uint64
    max_lovelace,      ; [9]  uint64 — maximum lovelace
    _rational,         ; [10] tag(30) [numerator, denominator]
    _unknown_11,       ; [11] uint64
    ex_units,          ; [12] array — [mem, steps, price] for script execution
]
```

The harness reads `start_slot` (index 0) and `epoch_length` (index 2) from config. All other config fields are currently unused by the harness.

---

## NewEpochState Structure (initial\_state and final\_state)

Both `initial_state` and `final_state` are 7-element CBOR arrays representing a Cardano `NewEpochState`. The harness reads from `initial_state`; `final_state` is used to extract final reward balances.

```
NewEpochState = [
    epoch_no,          ; [0] uint64 — current epoch number
    _prev_blocks,      ; [1] map    — (unused)
    _curr_blocks,      ; [2] map    — (unused)
    begin_epoch_state, ; [3] array[2]
    _snap_shots,       ; [4]        — (unused)
    _reward_update,    ; [5]        — (unused)
    _pool_distr,       ; [6]        — (unused)
]
```

### begin\_epoch\_state (index 3)

```
begin_epoch_state = [
    account_state,  ; [0] [treasury, reserves]
    ledger_state,   ; [1] array[2]
]
```

### ledger\_state (index 3→1)

```
ledger_state = [
    cert_state,  ; [0] array[3] — certificates, DReps, stake and pool state
    utxo_state,  ; [1] array[4] — UTxOs, deposits, fees, governance
]
```

### cert\_state (index 3→1→0)

```
cert_state = [
    voting_state,     ; [0] [drep_state, committee_state, ...]
    pool_state,       ; [1] pool registrations / retirements
    delegation_state, ; [2] unified stake-credential state
]

delegation_state = [
    unified_map_wrapper, ; [0]
    _future_gen_delegs,  ; [1]
    _gen_delegs,         ; [2]
    _instantaneous,      ; [3]
]

unified_map_wrapper = [
    reward_accounts, ; [0] map[Credential]AccountState
    _pointer_map,    ; [1]
]
```

The harness reads DRep registrations and stake registrations from this subtree:
- DReps: `initial_state[3][1][0][0][0]`
- Stake registrations: `initial_state[3][1][0][2][0][0]`

### utxo\_state (index 3→1→1)

```
utxo_state = [
    utxos,     ; [0] map[TxIn]TxOut — the UTxO set
    deposits,  ; [1] coin
    fees,      ; [2] coin
    gov_state, ; [3] array[7]
]
```

---

## CBOR Extraction Paths

| Data | CBOR Path |
|------|-----------|
| Epoch number | `initial_state[0]` |
| UTxO set | `initial_state[3][1][1][0]` |
| Governance state array | `initial_state[3][1][1][3]` |
| Active proposals | `initial_state[3][1][1][3][0]` |
| Committee | `initial_state[3][1][1][3][1]` |
| Constitution | `initial_state[3][1][1][3][2]` |
| Current pparams hash | `initial_state[3][1][1][3][3]` |
| Previous pparams hash | `initial_state[3][1][1][3][4]` |
| DRep registrations | `initial_state[3][1][0][0][0]` |
| Stake registrations | `initial_state[3][1][0][2][0][0]` |
| Final reward balances | `final_state[3][1][0][2][0][0]` (reward accounts within delegation state) |

---

Reward-account keys are stake credentials encoded as `[type, hash]`. The type
is part of the identity: type `0` (verification-key hash) and type `1` (script
hash) remain distinct even when the 28-byte hashes are equal.

Vendored UMap account values wrap `[reward, deposit]` in their first field, so
the reward balance is the first value of that nested pair. Modern Conway
AccountState values encode `[balance, deposit, poolDelegation,
drepDelegation]`, with the balance directly in the first field.

---

## Governance State Array (index 3→1→1→3)

The `gov_state` array has 7 elements:

```
gov_state = [
    proposals,           ; [0] proposals container
    committee,           ; [1] constitutional committee
    constitution,        ; [2] constitution anchor and optional policy
    current_pparams_hash ; [3] bytes(32) — used for pparams-by-hash/ lookup
    prev_pparams_hash,   ; [4] bytes(32) — previous epoch's pparams hash
    future_pparams,      ; [5] optional — future pparams if any
    drep_state,          ; [6] DRep-related state
]
```

### Proposals Container (gov\_state\[0\])

```
proposals_container = [
    roots,               ; [0] StrictMaybe [PParam, HardFork, Committee, Constitution]
    proposal_sequence,   ; [1] flat OMap sequence of ProposalState records
]

roots = [
    root_params,         ; [] or [[GovActionId]]
    root_hard_fork,      ; [] or [[GovActionId]]
    root_cc,             ; [] or [[GovActionId]]
    root_constitution,   ; [] or [[GovActionId]]
]

proposal_sequence = [ ProposalState, ... ]
```

Roots are used for parent-chain validation when new proposals are submitted. A `null` root means no proposal of that type has ever been enacted (genesis state).

### ProposalState CBOR

```
ProposalState = [
    id,               ; [0] GovActionId — [txHash(bytes), index(uint)]
    committee_votes,  ; [1] map StakeCredential -> Vote
    drep_votes,       ; [2] map StakeCredential -> Vote
    pool_votes,       ; [3] map PoolId -> Vote
    procedure,        ; [4] Proposal — contains the governance action
    proposed_in,      ; [5] uint64 — epoch submitted
    expires_after,    ; [6] uint64 — epoch at which proposal expires
]
```

Vote values: `0` = No, `1` = Yes, `2` = Abstain.

---

## Event Types (index 3)

Events are CBOR arrays whose first element is a variant tag:

### Transaction (tag 0)

```
[0, tx_cbor, success, slot]
```

| Field | Type | Description |
|-------|------|-------------|
| `tx_cbor` | `bytes` | Raw CBOR of the transaction |
| `success` | `bool` | Whether the transaction should be accepted |
| `slot` | `uint64` | Slot number of this transaction |

The `success` flag:
- `true` — transaction passes phase-1 validation and is applied. If `tx.IsValid() == false`, it is a phase-2 failure: collateral is consumed, all other effects are reverted, but the vector still expects `success = true` because the failure was identified correctly.
- `false` — transaction fails phase-1 validation and must be rejected.

### PassTick (tag 1)

```
[1, slot]
```

Advance the current slot without applying a transaction. The harness uses this to move time forward (relevant for slot-based expiry or TTL checks). No state change is required unless your implementation has slot-sensitive state.

### PassEpoch (tag 2)

```
[2, epoch_delta]
```

Advance by `epoch_delta` epochs. The harness computes the new epoch number and calls `StateManager.ProcessEpochBoundary(newEpoch)` for each epoch in the delta. Most vectors use a delta of 1.

### Rollback (tag 3)

```
[3, target_slot]
```

Roll back state to `target_slot`. The harness resets the state manager and replays all retained events (those with `slot <= target_slot`) from the beginning of the vector. Your `Reset()` must fully clear state; `LoadInitialState` and subsequent `ApplyTransaction` / `ProcessEpochBoundary` calls rebuild it.

---

## Protocol Parameters

### Loading

Protocol parameters are **not** embedded in the vector itself. The harness extracts the 32-byte hash from `initial_state[3][1][1][3][3]` and looks up the corresponding file in `testdata/eras/conway/impl/dump/pparams-by-hash/`.

Files are named by their hex-encoded hash (64 hex characters, no extension). The file contains CBOR-encoded `ConwayProtocolParameters`.

Lookup logic:
1. Exact filename match (most common).
2. Substring match as a fallback for non-standard naming.

### Deep Copy Requirement

Every vector receives its own deep copy of the loaded parameters. A `ParameterChange` governance action enacted during one vector must not affect the parameters seen by any other vector. The `PParamsLoader` handles this automatically; callers should not share `common.ProtocolParameters` objects across vectors.

### "No cost model" Handling

Some Haskell test cases call `modifyPParams (ppCostModelsL .~ mempty)` in memory but export the unmodified pparams hash. The harness detects these vectors by checking whether the title contains `nocostmodel` (after stripping case and non-alphanumeric characters) and clears all cost models from the loaded parameters before passing them to `LoadInitialState`.

Your `StateManager` implementation receives already-cleared parameters and does not need to detect these vectors.

### Cost Model Format

Cost models are stored as `map[uint][]int64` within `ConwayProtocolParameters.CostModels`:

| Key | Plutus version |
|-----|----------------|
| `1` | PlutusV1 |
| `2` | PlutusV2 |
| `3` | PlutusV3 |

Each value is a flat array of integer cost parameters in the order defined by the Cardano ledger spec for that version.

---

## UTxO Encoding

The UTxO map at `initial_state[3][1][1][0]` may appear in several encodings depending on how the Haskell exporter serialized it. The parser handles all variants:

| Encoding | Description |
|----------|-------------|
| `map[UtxoId]Output` | Typed key struct `[txHash, index]` |
| `map[string]Output` | String key `"txHash#index"` (hex txHash, decimal index) |
| `map[bytes]Output` | Raw 32-byte txHash key (index encoded separately) |
| `[[UtxoId, Output], ...]` | Array of two-element pairs |

Outputs are decoded as `babbage.BabbageTransactionOutput`, which carries all fields (address, value, assets, datum, datumHash, scriptRef) and is valid for all eras through Conway.

---

## Credential Encoding

Stake credentials and pool keys appear as 28-byte `Blake2b224` hashes throughout the vector. The parser handles two encodings:

- **Binary**: raw 28-byte CBOR bytestring.
- **Hex text**: 56-character text string (hex-encoded 28 bytes).

When implementing your own parser, check the CBOR type and handle both cases.

---

## See also

- [IMPLEMENTING_STATE_MANAGER.md](IMPLEMENTING_STATE_MANAGER.md) — how to wrap your ledger backend and implement the `StateManager` interface.
- [state_parser.go](state_parser.go) — reference Go implementation of initial-state extraction.
- [vector.go](vector.go) — reference Go implementation of top-level vector and event decoding.
- [pparams.go](pparams.go) — reference Go implementation of pparams-by-hash lookup, deep copy, and "No cost model" clearing.
