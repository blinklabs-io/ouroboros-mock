# StateManager Implementation Guide

This guide is for downstream projects — gouroboros, dingo, adder, or any third party — that want to run the conformance test harness against their own ledger or database implementation rather than the built-in `MockStateManager`.

---

## Architecture

```
conformance.Harness
    │
    │  StateManager interface (mutations)
    ▼
YourStateManager          ← you implement this
    │  GetStateProvider()
    ▼
YourBackend               ← wraps your real database / ledger
    │  implements ledger.StateProvider (read-only queries)
    ▼
your real DB / ledger code
```

The harness owns the test-execution loop: it parses vectors, drives events, and validates results. You supply the implementation underneath. The network protocol layer (handshake, chainsync, blockfetch) is not affected — only the persistence layer changes.

---

## Step 1 — Implement `ledger.StateProvider`

`ledger.StateProvider` ([ledger/backend.go](ledger/backend.go)) is the read-only surface the harness uses during transaction validation. It composes seven gouroboros interfaces:

```go
type StateProvider interface {
    lcommon.LedgerState   // NetworkId()
    lcommon.UtxoState     // UtxoById(), CostModels()
    lcommon.CertState     // StakeRegistration(), IsStakeCredentialRegistered(), IsRewardAccountRegistered()
    lcommon.SlotState     // SlotToTime(), TimeToSlot()
    lcommon.PoolState     // PoolCurrentState(), IsPoolRegistered(), IsVrfKeyInUse()
    lcommon.RewardState   // CalculateRewards(), GetAdaPots(), UpdateAdaPots(), GetRewardSnapshot(), RewardAccountBalance()
    lcommon.GovState      // CommitteeMember(), CommitteeMembers(), DRepRegistration(), DRepRegistrations(),
                          // Constitution(), TreasuryValue(), GovActionById(), GovActionExists()
}
```

Add a compile-time check to catch missing methods early:

```go
var _ ledger.StateProvider = (*YourBackend)(nil)
```

Each method should read from your real database. Start with stub delegations to `MockStateManager` and replace them one at a time as your implementation matures (see [example_backend_test.go](example_backend_test.go) for the delegation pattern).

---

## Step 2 — Implement `conformance.StateManager`

`conformance.StateManager` ([state.go](state.go)) handles state mutations during test execution. Add the same compile-time check:

```go
var _ conformance.StateManager = (*YourStateManager)(nil)
```

### `LoadInitialState(state *ParsedInitialState, pp common.ProtocolParameters) error`

Called once per vector, before any events are processed. Your job is to hydrate your database from the parsed state so that subsequent reads through `GetStateProvider()` reflect that starting point.

`ParsedInitialState` fields:

| Field | Type | Description |
|-------|------|-------------|
| `CurrentEpoch` | `uint64` | Epoch number at vector start |
| `Utxos` | `map[string]ParsedUtxo` | UTxO set, keyed by `"txHash#index"` |
| `StakeRegistrations` | `map[Blake2b224]bool` | Registered stake credentials |
| `RewardAccounts` | `map[Blake2b224]uint64` | Reward account balances |
| `PoolRegistrations` | `map[Blake2b224]bool` | Registered pools |
| `CommitteeMembers` | `map[Blake2b224]uint64` | Cold key → expiry epoch |
| `HotKeyAuthorizations` | `map[Blake2b224]Blake2b224` | Cold key → hot key |
| `DRepRegistrations` | `[]Blake2b224` | Registered DRep credential hashes |
| `Proposals` | `map[string]GovActionInfo` | Active governance proposals |
| `ProposalRoots` | `ProposalRoots` | Last-enacted root for each action type |
| `Constitution` | `*ConstitutionInfo` | Current constitution (may be nil) |
| `PParamsHash` | `[]byte` | 32-byte hash of current protocol parameters |
| `CostModels` | `map[uint][]int64` | Plutus version → cost array |

Each `ParsedUtxo` carries the full `common.TransactionOutput` (decoded as `babbage.BabbageTransactionOutput`, which covers all fields through Conway).

The `pp` parameter is a deep copy of the protocol parameters loaded from `pparams-by-hash/` using `PParamsHash`. Store it; it may be updated later when `ParameterChange` proposals are enacted.

**Also initialize `GovernanceState` here.** The easiest way is to call `GovernanceState.LoadFromParsedState(state)` on a `NewGovernanceState()` instance — this populates all the committee, DRep, stake, pool, proposal, and root fields needed for harness pre-validation.

### `ApplyTransaction(tx common.Transaction, slot uint64) error`

Called after each event of type `Transaction` whose `success` field is `true`. Update your database with the transaction's effects:

1. **Remove consumed UTxOs**: iterate `tx.Consumed()` and delete each input.
2. **Add produced UTxOs**: iterate `tx.Produced()` and insert each output.
3. **Process certificates**: iterate `tx.Certificates()` and apply each:
   - `StakeRegistration` / `Registration` (types 0, 7): register the stake credential.
   - `StakeDeregistration` / `Deregistration` (types 1, 8): deregister the credential and remove its reward account.
   - `PoolRegistration` (type 3): register the pool; cancel any pending retirement.
   - `PoolRetirement` (type 4): schedule pool retirement at the stated epoch.
   - `AuthCommitteeHot` (type 14): record cold→hot key authorization.
   - `ResignCommitteeCold` (type 15): mark member as resigned, clear hot key.
   - `RegistrationDrep` (type 16): register the DRep credential.
   - `DeregistrationDrep` (type 17): deregister the DRep credential.
   - Types 9–13: combined stake/vote delegation — register and/or delegate as indicated.
4. **Record governance proposals**: iterate `tx.ProposalProcedures()` and store each with `ExpiresAfter = currentEpoch + govActionLifetime`.
5. **Record votes**: iterate `tx.VotingProcedures()` and update each proposal's vote map.
6. **Process withdrawals**: iterate `tx.Withdrawals()` and reduce each reward account balance by the withdrawn amount.

For **phase-2 invalid transactions** (`tx.IsValid() == false`): skip steps 1–6. Instead, consume only the collateral inputs (`tx.CollateralInputs()`) and add the collateral return output (`tx.CollateralReturnOutput()`) if present.

Also update your `GovernanceState` to match: register/deregister credentials, update pool retirements, add proposals and votes, apply withdrawals. The harness reads governance state directly from `GetGovernanceState()` for pre-validation.

### `ProcessEpochBoundary(newEpoch uint64) error`

Called for each `PassEpoch` event. Perform standard epoch-transition bookkeeping:

1. **Advance epoch**: update `GovernanceState.CurrentEpoch` to `newEpoch`.
2. **Enact previously ratified proposals**: for any proposal whose `RatifiedEpoch < newEpoch`, call your enactment logic and update proposal roots. Enactment happens one epoch *after* ratification.
3. **Ratify eligible proposals**: for each active proposal, check if voting thresholds are met (see `mock_state_manager.go:ratifyProposals` for the reference implementation). If ratified, set `RatifiedEpoch = newEpoch`.
4. **Expire old proposals**: remove proposals where `ExpiresAfter < newEpoch`.
5. **Process pool retirements**: remove pools whose retirement epoch has arrived.

**Enactment effects by action type:**

| Action type | What to update |
|-------------|----------------|
| `ParameterChange` | Apply the parameter update to your stored protocol parameters. Call `GetProtocolParameters()` to get the current value; apply the update; store it. |
| `UpdateCommittee` | Merge proposed members into the committee; remove any removed members. |
| `NewConstitution` | Replace the constitution anchor and policy hash. |
| `NoConfidence` | Record in the `ConstitutionalCommittee` root. |
| `HardForkInitiation` | Record in the `HardFork` root (no state change beyond the root). |
| `TreasuryWithdrawal` | Record in governance history (no in-vector treasury effect expected). |
| `InfoAction` | Auto-ratified; no state change. |

After enacting a `ParameterChange`, refresh your in-memory protocol parameters so `GetProtocolParameters()` reflects the update. The harness calls `GetProtocolParameters()` after each epoch boundary.

### `GetStateProvider() StateProvider`

Return the `ledger.StateProvider` that the harness uses for read-only validation queries. This should reflect all state applied by prior `LoadInitialState`, `ApplyTransaction`, and `ProcessEpochBoundary` calls.

Return your real backend here. The harness passes it to `common.VerifyTransaction` and the governance pre-validation layer.

### `GetGovernanceState() *GovernanceState`

Return a pointer to your current `conformance.GovernanceState`. The harness uses this for governance pre-validation (certificate checks, proposal validation, voting procedure checks) before calling the core ledger verifier.

The simplest approach: maintain a `*GovernanceState` alongside your database and keep it in sync inside `ApplyTransaction` and `ProcessEpochBoundary`.

### `SetRewardBalances(balances map[Blake2b224]uint64)`

The harness calls this before each transaction with adjusted reward balances for that transaction's withdrawal validation. The adjustment accounts for future withdrawals within the same vector:

```
adjusted[cred] = finalStateBalance[cred] + sum(withdrawals[cred] from tx+1 onward)
```

Write these values into your `GovernanceState.RewardAccounts` map (and your database if your withdrawal validation reads from there). Do not persist them permanently; they are replaced before every transaction.

### `GetProtocolParameters() common.ProtocolParameters`

Return the current protocol parameters. Initially these come from `LoadInitialState`. After a `ParameterChange` proposal is enacted in `ProcessEpochBoundary`, they should reflect the updated values.

Return the parameters by value (or a copy) — the caller may modify them.

### `Reset() error`

Clear all state so the next test vector starts from scratch. The harness calls this at the start of every vector (before `LoadInitialState`). Clear your database tables (or equivalent in-memory structures) and reinitialize `GovernanceState`.

---

## Step 3 — Wire into the harness

```go
func TestConformance(t *testing.T) {
    sm := newYourStateManager()

    h := conformance.NewHarness(sm, conformance.HarnessConfig{
        TestdataRoot: "path/to/conformance/testdata",
    })
    h.RunAllVectors(t)
}
```

`TestdataRoot` must point to the directory that contains `eras/conway/impl/dump/`. If you are running inside the `conformance` package itself, `"testdata"` is the correct value. External consumers should use an absolute path or embed the testdata.

---

## Incremental adoption

You do not have to implement everything at once. The delegation pattern from [example_backend_test.go](example_backend_test.go) lets you forward individual methods to `MockStateManager` while you build out your real implementation:

```go
type YourBackend struct {
    db    *yourdb.DB
    inner func() ledger.StateProvider // fallback to mock
}

func (b *YourBackend) UtxoById(id common.TransactionInput) (common.Utxo, error) {
    // Once your DB is ready, replace this with:
    // return b.db.GetUtxo(id)
    return b.inner().UtxoById(id)
}
```

Start with `ApplyTransaction` → UTxO changes → certificates → governance. Add `ProcessEpochBoundary` last once the epoch lifecycle is clear.

---

## Key behaviors to match

**Reward balance injection** — `SetRewardBalances` is called by the harness before every transaction. The values already account for future withdrawals. Your withdrawal validation must use these values, not your database's current reward balance, or multi-withdrawal vectors will fail.

**Phase-2 invalid transactions** — `ApplyTransaction` is called even when `tx.IsValid() == false`. Apply only collateral effects; do not apply outputs or certificates.

**Pool re-registration cancels retirement** — when a pool registers again (`PoolRegistration`), any scheduled retirement for that pool is cancelled.

**Enactment is E+1** — a proposal ratified in epoch N is enacted in epoch N+1. Do not enact proposals in the same epoch they are ratified.

**ParameterChange refreshes pparams** — the harness calls `GetProtocolParameters()` after each `ProcessEpochBoundary`. Ensure the returned value is updated after enactment.

**"No cost model" vectors** — the harness clears cost models from the protocol parameters before passing them to `LoadInitialState` for any vector whose title matches `nocostmodel` (case/separator-insensitive). Your implementation receives already-cleared parameters and does not need to detect these vectors itself.

**Protocol parameters are per-vector** — the harness provides a deep copy of pparams to `LoadInitialState`. Never share a pparams object between vectors; a `ParameterChange` enactment in one vector must not affect the next.

---

## See also

- [VECTOR_FORMAT.md](VECTOR_FORMAT.md) — CBOR layout of test vectors, event types, initial-state extraction paths, `pparams-by-hash/` lookup.
- [example_backend_test.go](example_backend_test.go) — compilable stub showing the full delegation pattern.
- [mock_state_manager.go](mock_state_manager.go) — reference implementation of every method above.
- [validation.go](validation.go) — pre-validation logic that runs against `GetGovernanceState()`.
- [state.go](state.go) — `StateManager` interface, `GovernanceState`, and `ParsedInitialState` types.
