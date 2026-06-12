# Test Vector Format Reference

This document describes the CBOR binary format of the Amaru conformance test vectors consumed by this package. It is intended for implementors who need to parse vectors directly or understand how the harness extracts state.

Vectors are stored in `testdata/eras/conway/impl/dump/Conway/Imp/` (binary CBOR, no extension). Protocol parameter files are in `testdata/eras/conway/impl/dump/pparams-by-hash/` (one file per hash, named by hex-encoded hash).

---

## Top-Level Structure

Each vector file decodes to a 5-element CBOR array:

```
vector = [
    config,         ; [0] network/epoch configuration
    initial_state,  ; [1] NewEpochState before events
    final_state,    ; [2] NewEpochState after events
    events,         ; [3] array of events
    title,          ; [4] test name (text string)
]
```

The `title` field identifies the Haskell test that generated this vector (e.g. `"Conway/Imp/GOV/vote on committee member"`). It is also used to detect "No cost model" vectors (see [Protocol Parameters](#protocol-parameters)).

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
    cert_state,  ; [0] array[5] — certificates, DReps, committee
    utxo_state,  ; [1] array[4] — UTxOs, deposits, fees, governance
]
```

### cert\_state (index 3→1→0)

```
cert_state = [
    voting_state,  ; [0] [drep_state, committee_state, ...]
    _deleg_state,  ; [1]
    _pool_state,   ; [2] — pool registrations / stake distributions
    _reward_state, ; [3] — deposit map (stake registrations, pool deposits)
    _stake_state,  ; [4]
]
```

The harness reads DRep registrations and stake registrations from this subtree:
- DReps: `initial_state[3][1][0][0][0]`
- Stake registrations: `initial_state[3][1][0][2][0]`

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
| Stake registrations | `initial_state[3][1][0][2][0]` |
| Final reward balances | `final_state[3][1][1][3]` (reward accounts within gov state) |

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
    proposals_tree,     ; [0] map GovActionId -> ProposalState
    root_params,        ; [1] GovActionId | null — last enacted ParameterChange
    root_hard_fork,     ; [2] GovActionId | null — last enacted HardForkInitiation
    root_cc,            ; [3] GovActionId | null — last enacted NoConfidence/UpdateCommittee
    root_constitution,  ; [4] GovActionId | null — last enacted NewConstitution
]
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
