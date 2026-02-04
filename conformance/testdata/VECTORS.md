# Conformance Test Vector Index

Quick reference for finding test vectors by scenario. Click the detail links for full state change diagrams.

## Shelley Era (11 tests) → [shelley-era.md](shelley-era.md)

| Scenario | File | Expected |
|----------|------|----------|
| Basic transaction across epoch boundary | `EPOCH/Runs_basic_transaction` | ✓ Success |
| UTxO set updates after transactions | `LEDGER/Transactions_update_UTxO` | ✓ Success |
| Transaction doesn't balance (value not conserved) | `UTXO/ValueNotConservedUTxO` | ✗ Failure |
| Invalid Byron bootstrap witness | `UTXOW/Bootstrap_Witness/InvalidWitnessesUTXOW` | ✗ Failure |
| Valid Byron bootstrap witness | `UTXOW/Bootstrap_Witness/Valid_Witnesses` | ✓ Success |
| Metadata hash doesn't match metadata | `UTXOW/ConflictingMetadataHash` | ✗ Failure |
| Extra unnecessary script witness | `UTXOW/ExtraneousScriptWitnessesUTXOW` | ✗ Failure |
| Missing required script witness | `UTXOW/MissingScriptWitnessesUTXOW` | ✗ Failure |
| Metadata present but hash missing | `UTXOW/MissingTxBodyMetadataHash` | ✗ Failure |
| Metadata hash present but metadata missing | `UTXOW/MissingTxMetadata` | ✗ Failure |
| Missing required signature | `UTXOW/MissingVKeyWitnessesUTXOW` | ✗ Failure |

## Mary Era (2 tests) → [mary-era.md](mary-era.md)

| Scenario | File | Expected |
|----------|------|----------|
| Mint a native token | `UTXO/Mint_a_Token` | ✓ Success |
| Burn more tokens than exist | `UTXO/ValueNotConservedUTxO` | ✗ Failure |

## Allegra Era (1 test) → [allegra-era.md](allegra-era.md)

| Scenario | File | Expected |
|----------|------|----------|
| Invalid/malformed metadata | `UTXOW/InvalidMetadata` | ✗ Failure |

## Alonzo Era - UTXO (7 tests) → [alonzo-utxo.md](alonzo-utxo.md)

| Scenario | File | Expected |
|----------|------|----------|
| PlutusV1 insufficient collateral | `PlutusV1/Insufficient_collateral` | ✗ Failure |
| PlutusV1 too many execution units | `PlutusV1/Too_many_execution_units_for_tx` | ✗ Failure |
| PlutusV2 insufficient collateral | `PlutusV2/Insufficient_collateral` | ✗ Failure |
| PlutusV2 too many execution units | `PlutusV2/Too_many_execution_units_for_tx` | ✗ Failure |
| PlutusV3 insufficient collateral | `PlutusV3/Insufficient_collateral` | ✗ Failure |
| PlutusV3 too many execution units | `PlutusV3/Too_many_execution_units_for_tx` | ✗ Failure |
| Wrong network ID | `Wrong_network_ID` | ✗ Failure |

## Alonzo Era - UTXOS (33 tests) → [alonzo-utxos.md](alonzo-utxos.md)

| Scenario | File | Expected |
|----------|------|----------|
| PlutusV1 script passes phase 2 | `PlutusV1/Scripts_pass_in_phase_2` | ✓ Success |
| PlutusV1 spending with datum | `PlutusV1/Spending_scripts_with_a_Datum` | ✓ Success |
| PlutusV1 no cost model available | `PlutusV1/No_cost_model` | ✗ Failure |
| PlutusV1 malformed script | `PlutusV1/Malformed_scripts` | ✗ Failure |
| PlutusV2 script passes phase 2 | `PlutusV2/Scripts_pass_in_phase_2` | ✓ Success |
| PlutusV2 spending with datum | `PlutusV2/Spending_scripts_with_a_Datum` | ✓ Success |
| PlutusV2 no cost model available | `PlutusV2/No_cost_model` | ✗ Failure |
| PlutusV2 malformed script | `PlutusV2/Malformed_scripts` | ✗ Failure |
| PlutusV3 script passes phase 2 | `PlutusV3/Scripts_pass_in_phase_2` | ✓ Success |
| PlutusV3 spending with datum | `PlutusV3/Spending_scripts_with_a_Datum` | ✓ Success |
| PlutusV3 no cost model available | `PlutusV3/No_cost_model` | ✗ Failure |
| PlutusV3 malformed script | `PlutusV3/Malformed_scripts` | ✗ Failure |

## Alonzo Era - UTXOW (67 tests) → [alonzo-utxow.md](alonzo-utxow.md)

| Scenario | File | Expected |
|----------|------|----------|
| Valid PlutusV1 transaction | `Valid_transactions/PlutusV1/*` | ✓ Success |
| Valid PlutusV2 transaction | `Valid_transactions/PlutusV2/*` | ✓ Success |
| Valid PlutusV3 transaction | `Valid_transactions/PlutusV3/*` | ✓ Success |
| PlutusV1 extra redeemer | `Invalid_transactions/PlutusV1/Extra_Redeemer` | ✗ Failure |
| PlutusV1 PP hash mismatch | `Invalid_transactions/PlutusV1/PPViewHashesDontMatch` | ✗ Failure |
| PlutusV2 extra redeemer | `Invalid_transactions/PlutusV2/Extra_Redeemer` | ✗ Failure |
| PlutusV2 PP hash mismatch | `Invalid_transactions/PlutusV2/PPViewHashesDontMatch` | ✗ Failure |
| PlutusV3 extra redeemer | `Invalid_transactions/PlutusV3/Extra_Redeemer` | ✗ Failure |
| PlutusV3 PP hash mismatch | `Invalid_transactions/PlutusV3/PPViewHashesDontMatch` | ✗ Failure |

## Babbage Era (3 tests) → [babbage-era.md](babbage-era.md)

| Scenario | File | Expected |
|----------|------|----------|
| Malformed reference script in output | `UTXOW/MalformedReferenceScripts` | ✗ Failure |
| Malformed script in witness set | `UTXOW/MalformedScriptWitnesses` | ✗ Failure |
| Redeemer points to nothing | `UTXOW/ExtraRedeemers-RedeemerPointerPointsToNothing` | ✗ Failure |

## Conway Era - CERTS (2 tests) → [conway-certs.md](conway-certs.md)

| Scenario | File | Expected |
|----------|------|----------|
| Withdraw from unregistered reward account | `Withdrawals/Unregistered` | ✗ Failure |
| Withdraw wrong amount | `Withdrawals/Wrong_amount` | ✗ Failure |

## Conway Era - DELEG (24 tests) → [conway-deleg.md](conway-deleg.md)

| Scenario | File | Expected |
|----------|------|----------|
| Register stake with correct deposit | `Register/With_correct_deposit` | ✓ Success |
| Register stake with wrong deposit | `Register/With_incorrect_deposit` | ✗ Failure |
| Register already-registered stake | `Register/When_already_registered` | ✗ Failure |
| Duplicate registration in same tx | `Register/Twice_same_certificate` | ✗ Failure |
| Unregister registered stake | `Unregister/When_registered` | ✓ Success |
| Unregister non-registered stake | `Unregister/When_not_registered` | ✗ Failure |
| Unregister with non-zero rewards | `Unregister/With_non_zero_balance` | ✗ Failure |
| Delegate to registered pool | `Delegate_stake/To_registered_pool` | ✓ Success |
| Delegate unregistered credential | `Delegate_stake/Unregistered_credentials` | ✗ Failure |
| Delegate to unregistered pool | `Delegate_stake/To_unregistered_pool` | ✗ Failure |
| Delegate vote to registered DRep | `Delegate_vote/To_registered_DRep` | ✓ Success |
| Delegate vote - unregistered credential | `Delegate_vote/Unregistered_credentials` | ✗ Failure |
| Delegate vote to unregistered DRep | `Delegate_vote/To_unregistered_DRep` | ✗ Failure |
| Combined stake+vote delegation | `Delegate_both/*` | ✓ Success |

## Conway Era - GOVCERT (9 tests) → [conway-govcert.md](conway-govcert.md)

| Scenario | File | Expected |
|----------|------|----------|
| Register and unregister DRep | `succeeds_for/registering_and_unregistering_a_DRep` | ✓ Success |
| Re-register CC hot key | `succeeds_for/re-registering_a_CC_hot_key` | ✓ Success |
| Resign non-CC key | `succeeds_for/resigning_a_non-CC_key` | ✓ Success |
| Resigning proposed CC key | `Resigning_proposed_CC_key` | ✓ Success |
| DRep already registered | `fails_for/DRep_already_registered` | ✗ Failure |
| Invalid DRep registration deposit | `fails_for/invalid_deposit_provided_with_DRep_registration_cert` | ✗ Failure |
| Invalid DRep deregistration refund | `fails_for/invalid_refund_provided_with_DRep_deregistration_cert` | ✗ Failure |
| Register resigned CC member hotkey | `fails_for/registering_a_resigned_CC_member_hotkey` | ✗ Failure |
| Unregister nonexistent DRep | `fails_for/unregistering_a_nonexistent_DRep` | ✗ Failure |

## Conway Era - GOV (55 tests) → [conway-gov.md](conway-gov.md)

| Scenario | File | Expected |
|----------|------|----------|
| Submit valid proposal | `Proposing_and_voting/*` | ✓ Success |
| Vote on proposal | `Voting/*` | ✓ Success |
| Constitution proposal accepted | `Constitution_proposals/accepted_for/*` | ✓ Success |
| Constitution proposal rejected | `Constitution_proposals/rejected_for/*` | ✗ Failure |
| First hard fork proposal | `HardFork/First_hard_fork` | ✓ Success |
| Invalid hard fork version | `HardFork/Invalid_version` | ✗ Failure |
| Parameter update proposal | `PParamUpdate/*` | Mixed |
| Treasury withdrawal proposal | `Withdrawals/*` | Mixed |
| Proposal with wrong deposit | `Predicate_failures/Wrong_deposit` | ✗ Failure |
| Proposal with invalid return address | `Predicate_failures/Invalid_return_address` | ✗ Failure |

## Conway Era - ENACT (16 tests) → [conway-enact.md](conway-enact.md)

| Scenario | File | Expected |
|----------|------|----------|
| Treasury withdrawal enacted | `Treasury_withdrawals/*` | ✓ Success |
| Committee update enacted | `Committee_enactment/*` | ✓ Success |
| Competing proposals - one wins | `Competing_proposals/*` | ✓ Success |

## Conway Era - RATIFY (46 tests) → [conway-ratify.md](conway-ratify.md)

| Scenario | File | Expected |
|----------|------|----------|
| DRep voting stake calculation | `Active_voting_stake/DRep/*` | ✓ Success |
| SPO voting stake calculation | `Active_voting_stake/StakePool/*` | ✓ Success |
| Predefined DReps (AlwaysAbstain) | `Predefined_DReps/*` | ✓ Success |
| SPO default votes in bootstrap | `SPO_default_votes/*` | ✓ Success |
| Committee term limits | `Committee_members_can_serve_full_term/*` | ✓ Success |
| Expired committee member handling | `When_CC_expired/*` | ✓ Success |
| Resigned committee quorum discount | `Expired_and_resigned_committee/*` | ✓ Success |
| Zero committee threshold | `When_CC_threshold_is_0/*` | ✓ Success |
| Parameter change affects proposals | `ParameterChange_affects_existing/*` | ✓ Success |

## Conway Era - UTXO (3 tests) → [conway-utxo.md](conway-utxo.md)

| Scenario | File | Expected |
|----------|------|----------|
| Reference script in Conway | `Reference_scripts/*` | Mixed |

## Conway Era - UTXOS (38 tests) → [conway-utxos.md](conway-utxos.md)

| Scenario | File | Expected |
|----------|------|----------|
| PlutusV1 accessing Conway fields | `Conway_features_fail_in_v1_v2/PlutusV1/*` | ✗ Failure |
| PlutusV2 accessing Conway fields | `Conway_features_fail_in_v1_v2/PlutusV2/*` | ✗ Failure |
| PlutusV3 accessing governance fields | `PlutusV3_Initialization/*` | ✓ Success |
| Governance policy script | `Gov_policy_scripts/*` | ✓ Success |
| Spending script without datum (V2+) | `Spending_script_without_a_Datum/*` | ✓ Success |
