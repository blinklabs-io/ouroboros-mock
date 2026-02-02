# Conformance Test Vectors

This directory contains Cardano ledger conformance test vectors sourced from the
[Amaru](https://github.com/pragma-org/amaru) project.

## Source

**Original location:** `crates/amaru-ledger/tests/data/rules-conformance/`

**Current version:** Extracted from gouroboros internal tarball (commit 930c14b6bdf8197bc7d9397d872949e108b28eb4)

**Path cleanup:** File and directory names containing spaces have been replaced with underscores for better compatibility.

## Structure

```
testdata/
├── eras/
│   └── conway/
│       └── impl/
│           └── dump/
│               ├── Conway/          # Test vector files (CBOR)
│               │   └── Imp/
│               │       ├── AllegraImpSpec/
│               │       ├── AlonzoImpSpec/
│               │       ├── BabbageImpSpec/
│               │       ├── ConwayImpSpec_-_Version_10/
│               │       ├── MaryImpSpec/
│               │       └── ShelleyImpSpec/
│               └── pparams-by-hash/ # Protocol parameter files
```

## Test Vector Format

Each test vector is a CBOR-encoded file containing a 5-element array:

```
[0] config:        array[13]  - Network/protocol configuration
[1] initial_state: array[7]   - NewEpochState before events
[2] final_state:   array[7]   - NewEpochState after events
[3] events:        array[N]   - Transaction/epoch events
[4] title:         string     - Test name/path
```

### Event Types

```
Transaction: [0, tx_cbor:bytes, success:bool, slot:uint64]
PassTick:    [1, slot:uint64]
PassEpoch:   [2, epoch_delta:uint64]
```

## Statistics

- **Test vectors:** 314
- **Protocol parameter versions:** 44
- **Total files:** 358

---

## Test Vector Documentation

**[VECTORS.md](VECTORS.md)** - Quick scenario index (find what you need fast)

**Detailed documentation by category:**

| File | Era | Tests | Description |
|------|-----|-------|-------------|
| [shelley-era.md](shelley-era.md) | Shelley | 11 | Basic UTxO, witnesses, epoch transitions |
| [mary-era.md](mary-era.md) | Mary | 2 | Native token minting |
| [allegra-era.md](allegra-era.md) | Allegra | 1 | Metadata validation |
| [alonzo-utxo.md](alonzo-utxo.md) | Alonzo | 7 | Collateral, execution units |
| [alonzo-utxos.md](alonzo-utxos.md) | Alonzo | 33 | Plutus script execution |
| [alonzo-utxow.md](alonzo-utxow.md) | Alonzo | 67 | Script witnesses, redeemers |
| [babbage-era.md](babbage-era.md) | Babbage | 3 | Reference scripts, inline datums |
| [conway-certs.md](conway-certs.md) | Conway | 2 | Withdrawal certificates |
| [conway-deleg.md](conway-deleg.md) | Conway | 24 | Stake/vote delegation |
| [conway-govcert.md](conway-govcert.md) | Conway | 9 | DRep/committee certificates |
| [conway-gov.md](conway-gov.md) | Conway | 55 | Proposals and voting |
| [conway-enact.md](conway-enact.md) | Conway | 16 | Action enactment |
| [conway-ratify.md](conway-ratify.md) | Conway | 46 | Vote counting, thresholds |
| [conway-utxo.md](conway-utxo.md) | Conway | 3 | Reference scripts |
| [conway-utxos.md](conway-utxos.md) | Conway | 38 | PlutusV3, governance scripts |

---

## Test Vector Summary

The test vectors are organized by Cardano era, testing era-specific ledger rules.

### Shelley Era (11 vectors)

Foundation rules for the Cardano ledger.

| Category | Count | What It Tests |
|----------|-------|---------------|
| **EPOCH** | 1 | Basic epoch boundary processing and transaction execution |
| **LEDGER** | 1 | Core ledger state transition rules |
| **UTXO** | 1 | Basic UTXO validation (ShelleyUtxoPredFailure) |
| **UTXOW** | 8 | UTXO with witness validation, bootstrap address witnesses |

**Specific tests:**
- `Runs_basic_transaction` - Verifies basic transaction processing works
- `ShelleyUtxoPredFailure` - Tests UTXO predicate failure detection
- Bootstrap witness handling for legacy Byron-era addresses

### Mary Era (2 vectors)

Multi-asset (native token) functionality introduced in Mary.

| Category | Count | What It Tests |
|----------|-------|---------------|
| **UTXO** | 2 | Native asset minting and transfer rules |

**What's validated:**
- Multi-asset value preservation across transactions
- Token minting policy enforcement
- Asset bundle serialization

### Allegra Era (1 vector)

Metadata and time-lock improvements.

| Category | Count | What It Tests |
|----------|-------|---------------|
| **UTXOW** | 1 | `InvalidMetadata` - Transaction metadata validation |

**What's validated:**
- Metadata hash matches attached metadata
- Metadata structure requirements

### Alonzo Era (98 vectors)

Smart contract (Plutus) execution rules. This is the largest pre-Conway test set.

#### UTXO (7 vectors)

| Subcategory | Count | What It Tests |
|-------------|-------|---------------|
| PlutusV1/Insufficient_collateral | 2 | Collateral amount validation for V1 scripts |
| PlutusV2/Insufficient_collateral | 2 | Collateral amount validation for V2 scripts |
| PlutusV3/Insufficient_collateral | 2 | Collateral amount validation for V3 scripts |
| Generic | 1 | General UTXO validation with scripts |

**What's validated:**
- Collateral must cover potential script failure costs
- Each Plutus version has specific collateral requirements

#### UTXOS (24 vectors)

Script execution during the spending phase.

| Subcategory | What It Tests |
|-------------|---------------|
| `Scripts_pass_in_phase_2` | Successful Plutus script execution |
| `Spending_scripts_with_a_Datum` | Datum presence and hash validation |
| `No_cost_model` | Rejection when cost model is missing |
| `Malformed_scripts` | Script structure validation |

**Version breakdown:** Each test exists for PlutusV1, V2, and V3.

**What's validated:**
- Phase 2 script execution succeeds with valid inputs
- Datums must be present for spending scripts (pre-Babbage)
- Cost models must exist for the script version used
- Script CBOR structure must be well-formed

#### UTXOW (67 vectors)

Transaction witness validation with scripts.

| Subcategory | What It Tests |
|-------------|---------------|
| `Valid_transactions/PlutusV*` | Complete valid script transactions |
| `Invalid_transactions/*/Extra_Redeemer` | Redeemers without matching scripts |
| `Invalid_transactions/*/PPViewHashesDontMatch` | Protocol parameter hash mismatch |

**What's validated:**
- All required script witnesses are present
- Redeemers match script executions
- Protocol parameter hash in tx body matches chain state
- Datum hashes resolve to provided datums

### Babbage Era (3 vectors)

Reference scripts and inline datums introduced in Babbage.

| Category | Count | What It Tests |
|----------|-------|---------------|
| **UTXOW** | 3 | Reference script and inline datum witness rules |

**What's validated:**
- Reference scripts can satisfy witness requirements
- Inline datums don't require separate datum witnesses
- Correct interaction between reference and regular scripts

### Conway Era (205 vectors)

Governance (CIP-1694) implementation. The largest test category.

#### CERTS - Certificate Processing (2 vectors)

| Subcategory | What It Tests |
|-------------|---------------|
| Withdrawals | Stake credential withdrawal authorization |

**What's validated:**
- Withdrawal transactions require stake key signature
- Withdrawal amounts match reward account balances

#### DELEG - Stake Delegation (24 vectors)

| Subcategory | Count | What It Tests |
|-------------|-------|---------------|
| Register_stake_credential | 4 | Registration deposit, duplicate prevention |
| Unregister_stake_credentials | 6 | Deregistration rules, deposit refund |
| Delegate_stake | 6 | Pool delegation mechanics |
| Delegate_vote | 5 | DRep delegation |
| Delegate_both_stake_and_vote | 3 | Combined stake+vote delegation |

**Specific tests:**
- `With_correct_deposit_or_without_any_deposit` - Deposit validation
- `When_already_registered` - Duplicate registration rejection
- `Twice_the_same_certificate_in_the_same_transaction` - Intra-tx duplicate detection

**What's validated:**
- Key deposit amounts match protocol parameters
- Cannot register already-registered credentials
- Delegation targets must exist (pools/DReps)
- Combined delegation certificates work atomically

#### GOVCERT - Governance Certificates (9 vectors)

| Subcategory | Count | What It Tests |
|-------------|-------|---------------|
| succeeds_for | 3 | Valid DRep/committee certificate operations |
| fails_for | 5 | Invalid governance certificate rejection |

**What's validated:**
- DRep registration deposit requirements
- Committee hot key authorization
- Cold key resignation processing
- DRep update certificate validation

#### GOV - Governance Actions (55 vectors)

| Subcategory | Count | What It Tests |
|-------------|-------|---------------|
| Proposing_and_voting | 7 | Proposal submission mechanics |
| Voting | 8 | Vote casting and recording |
| Constitution_proposals | varies | Constitution change proposals |
| HardFork | 2 | First vs subsequent hard fork rules |
| PParamUpdate | 2+ | Protocol parameter change proposals |
| Network_ID | 1 | Network identifier in proposals |
| Unknown_CostModels | 1 | Cost model validation in updates |
| Withdrawals | 3 | Treasury withdrawal requests |
| Predicate_failures | 4 | Governance rule violation detection |

**Specific tests:**
- `accepted_for_*` - Proposals that should pass validation
- `rejected_for_*` - Proposals that should fail validation
- `First_hard_fork` vs subsequent - Different validation rules apply

**What's validated:**
- Proposal deposits meet requirements
- Return addresses are valid
- Anchor URLs/hashes present when required
- Previous action references are valid
- Hard fork version increments correctly

#### ENACT - Governance Enactment (16 vectors)

| Subcategory | Count | What It Tests |
|-------------|-------|---------------|
| Treasury_withdrawals | 4 | Withdrawal enactment to reward accounts |
| Committee_enactment | 2 | Committee membership updates |
| Competing_proposals | 3 | Priority when multiple proposals ratify |

**What's validated:**
- Treasury funds transfer correctly on enactment
- Committee additions and removals take effect
- When competing proposals ratify, only one enacts
- Enactment order follows protocol rules

#### RATIFY - Governance Ratification (46 vectors)

| Subcategory | What It Tests |
|-------------|---------------|
| Active_voting_stake | Stake-weighted vote counting |
| SPO_default_votes | Stake pool operator default voting behavior |
| Interaction_between_governing_bodies | Cross-body voting effects |
| Committee_members_can_serve_full_CommitteeMaxTermLength | Term limit enforcement |
| When_CC_expired | Handling of expired committee members |
| CommitteeMinSize_affects_in-flight_proposals | Quorum changes mid-proposal |
| Expired_and_resigned_committee_members | Quorum discount rules |
| When_CC_threshold_is_0 | Zero threshold special case |
| Delaying_actions | Action delay mechanics |
| ParameterChange_affects_existing_proposals | Dynamic parameter effects |

**Specific tests:**
- `DRep` - DRep voting power calculation
- `StakePool` - SPO voting weight
- `Predefined_DReps` - AlwaysAbstain, AlwaysNoConfidence handling

**What's validated:**
- Voting thresholds calculated correctly per body type
- Expired members excluded from quorum
- Resigned members excluded from quorum
- Parameter changes affect in-flight proposals
- Delay enactment works correctly

#### UTXO (3 vectors)

| Subcategory | What It Tests |
|-------------|---------------|
| Reference_scripts | Reference script handling in Conway context |

#### UTXOS - Script Execution (38 vectors)

| Subcategory | Count | What It Tests |
|-------------|-------|---------------|
| Conway_features_fail_in_Plutus_v1_and_v2 | 32 | Backwards compatibility |
| Gov_policy_scripts | 3 | Governance proposal policy scripts |
| PlutusV3_Initialization | 3 | V3-specific script setup |
| Spending_script_without_a_Datum | 3 | Datumless spending (V2+ feature) |

**Conway features unsupported in V1/V2:**
- Certificates: `RegDepositTxCert`, `UnRegDepositTxCert`
- Fields: `CurrentTreasuryValue`, `ProposalProcedures`, `TreasuryDonation`, `VotingProcedures`

**What's validated:**
- Old Plutus versions fail when accessing new fields
- PlutusV3 can access all Conway features
- Governance policy scripts execute correctly

---

## Naming Conventions

### Success Indicators
- `accepted_for`, `succeeds_for`, `valid`, `Valid_transactions`
- `With_correct_deposit_or_without_any_deposit`
- `Scripts_pass_in_phase_2`

### Failure Indicators
- `rejected_for`, `fails_for`, `invalid`, `Invalid_transactions`
- `ShelleyUtxoPredFailure`, `PPViewHashesDontMatch`, `Extra_Redeemer`
- `InvalidMetadata`, `Insufficient_collateral`, `Malformed_scripts`
- `When_already_registered`, `Twice_the_same_certificate`

### Version-Specific
- `PlutusV1`, `PlutusV2`, `PlutusV3` - Plutus language version tests
- `Version_10` - Conway protocol version (10)

---

## Ledger Rules Reference

The test vectors map to specific ledger rules from the Cardano specification:

| Rule | Description | Vector Count |
|------|-------------|--------------|
| UTXO | Unspent transaction output validation | 13 |
| UTXOS | UTXO with script validation | 62 |
| UTXOW | UTXO with witness validation | 79 |
| DELEG | Delegation certificate processing | 24 |
| GOVCERT | Governance certificate processing | 9 |
| GOV | Governance proposal/voting rules | 55 |
| ENACT | Governance action enactment | 16 |
| RATIFY | Governance ratification rules | 46 |
| CERTS | General certificate processing | 2 |
| EPOCH | Epoch boundary processing | 1 |
| LEDGER | Top-level ledger rules | 1 |

---

## Protocol Parameters

The `pparams-by-hash/` directory contains 44 protocol parameter configurations,
each named by its Blake2b-256 hash. These are CBOR-encoded
`ConwayProtocolParameters` structures containing:

- **Fee parameters:** minFeeA, minFeeB
- **Size limits:** maxTxSize, maxBlockBodySize, maxBlockHeaderSize
- **Deposit amounts:** keyDeposit, poolDeposit, drepDeposit, govActionDeposit
- **Cost models:** Plutus V1, V2, V3 execution costs
- **Governance parameters:** Voting thresholds, committee settings

Test vectors reference protocol parameters by hash in their initial state,
allowing the same pparams to be shared across multiple vectors.

---

## Updating

To update from the latest Amaru main branch:

```bash
make download-conformance-tests
```

This target will:
1. Clone/fetch the Amaru repository
2. Copy files from `crates/amaru-ledger/tests/data/rules-conformance/`
3. Clean up paths (replace spaces with underscores)
4. Update this README with the new commit hash

See [gouroboros conformance README](https://github.com/blinklabs-io/gouroboros/blob/main/internal/test/conformance/README.md)
for additional documentation on the test vector format.
