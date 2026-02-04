# Conformance Test Harness

This package provides a conformance test harness for Cardano ledger rule validation using Amaru test vectors. It can be used by any implementation that provides the required state interfaces.

## Test Status

| Metric | Count |
|--------|-------|
| Total vector files | 320 |
| Parsed vectors | 314 |
| Passing (MockStateManager) | 314 |
| Pass rate | 100% |

Note: 6 files are skipped (README files, documentation).

---

## Lessons Learned

### 1. Withdraw Zero Trick

Cardano allows zero-value withdrawals with Plutus scripts. This is the "withdraw zero trick" used to trigger script validation without actually withdrawing funds.

```go
// Skip amount validation for script-based withdrawals
// RedeemerTagReward = 3 indicates script-based withdrawal
if hasWithdrawalRedeemer {
    return nil // Let Plutus script handle validation
}
```

### 2. Governance Action Lifecycle

Proposals follow a strict lifecycle:
1. **Submission** - Added to active proposals with ExpiresAfter = currentEpoch + govActionLifetime
2. **Voting** - Votes are recorded during the epoch
3. **Ratification** - At epoch boundary, proposals meeting thresholds are marked with RatifiedEpoch
4. **Enactment** - In the NEXT epoch, ratified proposals are enacted (roots updated)
5. **Expiration** - Proposals past ExpiresAfter are removed

**Key insight**: Enactment happens at epoch E+1 for proposals ratified at epoch E.

### 3. Proposal Parent Chain (PrevGovId)

Each governance action type has a "root" tracking the last enacted proposal:
- `Roots.Constitution` - NewConstitution
- `Roots.ProtocolParameters` - ParameterChange
- `Roots.HardFork` - HardForkInitiation
- `Roots.ConstitutionalCommittee` - UpdateCommittee or NoConfidence

When submitting a new proposal:
- If root exists: PrevGovId MUST be non-nil and match root OR an active proposal
- If no root: PrevGovId CAN be nil (genesis proposal)

### 4. Committee Member Resign

The Cardano spec requires the credential in a `ResignCommitteeCold` certificate to be either a current committee member or a proposed member in a pending `UpdateCommittee` governance action. Resigning a credential that is neither a current nor proposed member fails validation.

### 5. Credential Encoding

Test vectors may encode credentials in different formats:
- Binary bytes (28 bytes for Blake2b224)
- Hex-encoded text (56 characters)

The parser must handle both:
```go
case string:
    if len(v) == 56 {
        if decoded, err := hex.DecodeString(v); err == nil && len(decoded) == 28 {
            copy(hash[:], decoded)
            return &hash
        }
    }
```

### 6. Cost Model Application

When a ParameterChange governance action is enacted, cost models must be applied to the active protocol parameters. The harness needs to retrieve updated parameters after each epoch boundary:

```go
// After ProcessEpochBoundary
h.protocolParams = h.stateManager.GetProtocolParameters()
```

### 7. Pool Deposit Refund

Pool retirement triggers a deposit refund. In gouroboros value conservation:
- **Consumed** should include PoolDeposit for `PoolRetirementCertificate` (similar to KeyDeposit for `StakeDeregistrationCertificate`)

### 8. Vote Tracking

Votes are stored within ProposalState, keyed by "voterType:credHash":
```go
voterKey := fmt.Sprintf("%d:%s", voter.Type, hex.EncodeToString(voter.Hash[:]))
proposal.Votes[voterKey] = votingProc.Vote
```

Vote values: 0=No, 1=Yes, 2=Abstain

---

## Test Vectors

**Source**: Amaru test vectors from [pragma-org/amaru](https://github.com/pragma-org/amaru)

The vectors are stored in `testdata/eras/conway/impl/dump/` and include:
- **320 test vector files** (CBOR binary) in `Conway/Imp/` (314 parsed, 6 skipped)
- **44 protocol parameter files** in `pparams-by-hash/`

---

## Test Vector CBOR Structure

### Top-Level Array
```text
[0] config:        array[13]  - Network/protocol configuration
[1] initial_state: array[7]   - NewEpochState before events
[2] final_state:   array[7]   - NewEpochState after events
[3] events:        array[N]   - Transaction/epoch events
[4] title:         string     - Test name/path
```

### Config Array (index 0)
The config array contains simplified network parameters, not full protocol parameters:
```text
[0]  start_slot:     uint64   - Epoch start slot
[1]  slot_length:    uint64   - Slot duration (milliseconds)
[2]  epoch_length:   uint64   - Slots per epoch
[3]  security_param: uint64   - Security parameter (k)
[4]  active_slots:   uint64   - Active slots coefficient denominator
[5]  network_id:     uint64   - Network ID (0=testnet, 1=mainnet)
[6]  pool_stake:     uint64   - Pool stake (scaled)
[7]  unknown_7:      uint64   - Unknown
[8]  unknown_8:      uint64   - Unknown
[9]  max_lovelace:   uint64   - Maximum lovelace (for rational encoding)
[10] rational:       tag(30)  - Rational number [numerator, denominator]
[11] unknown_11:     uint64   - Unknown
[12] ex_units:       array    - [mem, steps, price] for script execution
```

Note: Full protocol parameters including cost models are extracted from the
initial_state via pparams hash lookup, not from this config array.

### NewEpochState Structure
```text
[0] epoch_no
[3] begin_epoch_state: array[2]
    [0] account_state: [treasury, reserves]
    [1] ledger_state: array[2]
        [0] cert_state: array[5]
            [0] voting_state (dreps, committee)
        [1] utxo_state: array[4]
            [0] utxos: map[TxIn]TxOut
            [1] deposits
            [2] fees
            [3] gov_state: array[7]
                [0] proposals
                [1] committee
                [2] constitution
                [3] current_pparams_hash
```

### Event Types
Events are CBOR arrays where the first element is the variant tag:
```text
Transaction: [0, tx_cbor:bytes, success:bool, slot:uint64]
PassTick:    [1, slot:uint64]
PassEpoch:   [2, epoch:uint64]
```

The `success` field in Transaction events indicates:
- `true` = Transaction should be accepted (even if IsValid=false for phase-2 failures)
- `false` = Transaction should be rejected by phase-1 validation

Note: A transaction with `IsValid=false` may still have `success=true` if it was
correctly identified as a phase-2 failure. The transaction will be included in
the block but its effects (other than collateral consumption) will be reverted.

---

## Key CBOR Paths

| Data | CBOR Path |
|------|-----------|
| UTxOs | `initial_state[3][1][1][0]` |
| Gov State | `initial_state[3][1][1][3]` |
| Proposals | `initial_state[3][1][1][3][0]` |
| Committee | `initial_state[3][1][1][3][1]` |
| Constitution | `initial_state[3][1][1][3][2]` |
| PParams Hash | `initial_state[3][1][1][3][3]` |
| DReps | `initial_state[3][1][0][0][0]` |
| Stake Registrations | `initial_state[3][1][0][2][0]` |

---

## Multi-Transaction Handling

Many test vectors contain multiple transactions that build on each other:
1. TX 0 creates initial UTxOs
2. TX 1+ may spend outputs from prior TXs

The test harness updates the UTxO set after each transaction using:
- `tx.Consumed()` - UTxOs removed
- `tx.Produced()` - UTxOs created

---

## UTxO Encoding Formats

The harness handles multiple UTxO encodings:
1. `map[UtxoId]Output` - Typed keys
2. `map[string]Output` - String keys ("txid#index")
3. `map[rawBytes]Output` - Byte-string keys
4. `[[UtxoId, Output], ...]` - Array of pairs

UTxO outputs are decoded as `babbage.BabbageTransactionOutput` which includes
all fields: address, amount, assets, datum, datumHash, and scriptRef.

---

## Governance State Structure

### Gov State Array
The gov_state at `initial_state[3][1][1][3]` contains 7 elements:
```text
[0] proposals           - Proposal tracking
[1] committee           - Constitutional committee
[2] constitution        - Current constitution anchor and policy
[3] current_pparams_hash - Hash of current protocol parameters (32 bytes)
[4] prev_pparams_hash   - Hash of previous epoch's protocol parameters (32 bytes)
[5] future_pparams      - Future protocol parameters (if any)
[6] drep_state          - DRep-related state
```

Note: The current_pparams_hash at [3] is used to look up protocol parameters
from the `pparams-by-hash/` directory.

### Proposals Array
```text
proposals = [
    [0] proposals_tree,     - Map of GovActionId -> ProposalState
    [1] root_params,        - Last enacted ParameterChange (or null)
    [2] root_hard_fork,     - Last enacted HardFork (or null)
    [3] root_cc,            - Last enacted NoConfidence/UpdateCommittee (or null)
    [4] root_constitution   - Last enacted NewConstitution (or null)
]
```

### ProposalState CBOR
```text
ProposalState = [
    [0] id,                 - GovActionId [txHash, index]
    [1] committee_votes,    - map[StakeCredential]Vote
    [2] dreps_votes,        - map[StakeCredential]Vote
    [3] pools_votes,        - map[PoolId]Vote
    [4] procedure,          - Proposal (contains action type)
    [5] proposed_in,        - Epoch
    [6] expires_after       - Epoch
]
```

### Vote Values
- `0` = No
- `1` = Yes
- `2` = Abstain

---

## Certificate Types

The harness handles all Conway-era certificate types:

| Type | ID | Description |
|------|----|-------------|
| StakeRegistration | 0 | Shelley-era stake registration |
| StakeDeregistration | 1 | Shelley-era stake deregistration |
| StakeDelegation | 2 | Delegate stake to pool |
| PoolRegistration | 3 | Register stake pool |
| PoolRetirement | 4 | Retire stake pool |
| GenesisKeyDelegation | 5 | Genesis key delegation |
| MoveInstantaneousRewards | 6 | MIR certificate |
| Registration | 7 | Conway-era stake registration with deposit |
| Deregistration | 8 | Conway-era stake deregistration with refund |
| VoteDelegation | 9 | Delegate vote to DRep |
| StakeVoteDelegation | 10 | Delegate stake and vote |
| StakeRegistrationDelegation | 11 | Register + delegate stake |
| VoteRegistrationDelegation | 12 | Register + delegate vote |
| StakeVoteRegistrationDelegation | 13 | Register + delegate stake + vote |
| AuthCommitteeHot | 14 | Authorize CC hot key |
| ResignCommitteeCold | 15 | Resign from CC |
| RegistrationDrep | 16 | Register DRep |
| DeregistrationDrep | 17 | Deregister DRep |
| UpdateDrep | 18 | Update DRep metadata |

---

## Usage

### Basic Usage with MockStateManager

```go
import (
    "testing"
    "github.com/blinklabs-io/ouroboros-mock/conformance"
)

func TestConformance(t *testing.T) {
    sm := conformance.NewMockStateManager()
    harness := conformance.NewHarness(sm, conformance.HarnessConfig{
        TestdataRoot: "testdata",
    })
    harness.RunAllVectors(t)
}
```

### With Custom State Manager

Implement the `StateManager` interface:

```go
type StateManager interface {
    LoadInitialState(state *ParsedInitialState, pp common.ProtocolParameters) error
    ApplyTransaction(tx common.Transaction, slot uint64) error
    ProcessEpochBoundary(newEpoch uint64) error
    GetStateProvider() StateProvider
    GetGovernanceState() *GovernanceState
    SetRewardBalances(balances map[common.Blake2b224]uint64)
    GetProtocolParameters() common.ProtocolParameters
    Reset() error
}
```

---

## Implementation Notes

### ScriptDataHash Validation

The ScriptDataHash is computed as `Blake2b256(redeemers || datums || language_views)`:
- Redeemers: Original CBOR bytes preserved via `ConwayRedeemers.Cbor()`
- Datums: Original CBOR bytes preserved via `SetType[Datum].Cbor()` (only if non-empty)
- Language views: Encoded per Cardano spec with version-specific formats

**PlutusV1** (double-bagged for historical compatibility):
- Tag: `serialize(serialize(0))` = `0x4100` (bytestring containing 0x00)
- Params: indefinite-length list of cost model values, wrapped in bytestring

**PlutusV2/V3**:
- Tag: `serialize(version)` = `0x01` or `0x02`
- Params: definite-length list of cost model values (no bytestring wrapper)

### Cost Model Handling

The Haskell test suite modifies protocol parameters in memory via `modifyPParams`,
but test vectors store the original (unmodified) pparams hash. For "No cost model"
tests, the harness clears cost models to simulate the Haskell behavior.

### Malformed Reference Scripts

Transaction outputs with reference scripts must contain well-formed Plutus bytecode.
The validation uses plutigo's `syn.Decode[syn.DeBruijn]` to verify scripts are valid UPLC.

---

## Common Pitfalls

1. **Reference inputs**: Resolved but never consumed
2. **Collateral**: Only consumed when IsValid=false
3. **Datum lookup**: Check witness set, inline datum, and reference inputs
4. **Cost models**: Must exist for each Plutus version used
5. **Network ID**: All vectors use Preview/Testnet (network ID 0)
6. **Proposal enactment**: Happens at epoch boundaries, not immediately
7. **Vote tracking**: Votes are stored within ProposalState, not separately
8. **Ratification timing**: Proposals ratified in epoch N are enacted in epoch N+1
9. **Combined certificates**: Types 11-13 register AND delegate in one step
10. **Credential encoding**: May be binary or hex-encoded text
11. **CC resign**: Resignation requires current or proposed CC membership
12. **Zero withdrawals**: Valid with Plutus scripts ("withdraw zero trick")
13. **Pool deposit refund**: Retirement adds deposit to consumed value

---

## Test Categories

| Directory | Tests | Focus |
|-----------|-------|-------|
| GOV | 55 | Proposals, voting, policies |
| GOVCERT | 9 | DRep/CC certificates |
| ENACT | 16 | Proposal enactment |
| DELEG | 24 | Delegation operations |
| EPOCH | 12 | Epoch boundary logic |
| RATIFY | 46 | Ratification thresholds |
| AlonzoImpSpec | ~50 | Plutus V1/V2/V3 scripts |
| BabbageImpSpec | ~20 | Reference scripts, inline datums |
| ShelleyImpSpec | ~30 | Basic TX, witnesses, metadata |

---

## Files

| File | Purpose |
|------|---------|
| `vector.go` | Test vector parsing |
| `state_parser.go` | Initial state CBOR extraction |
| `pparams.go` | Protocol parameters loading |
| `harness.go` | Test harness core |
| `validation.go` | Pre-validation (governance, certificates) |
| `state.go` | StateManager interface, GovernanceState |
| `mock_state_manager.go` | In-memory StateManager implementation |
| `harness_test.go` | Harness self-tests |
