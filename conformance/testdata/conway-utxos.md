
# Conway UTXOS Conformance Test Vectors

This directory contains conformance test vectors for the Conway-era UTXOS (UTxO with Scripts) transition rule. These tests validate Plutus script execution, Conway governance feature compatibility, and script version constraints.

## Overview

| Category | Test Count | Focus Area |
|----------|------------|------------|
| Reference Scripts | 3 | Reference script usage and input overlap |
| Spending Scripts without Datum | 3 | PlutusV1/V2/V3 datum requirements |
| Inline Datums | 1 | PlutusV1 inline datum restrictions |
| Conway Features V1/V2 Failure | 25 | PlutusV1/V2 Conway governance access |
| Gov Policy Scripts | 3 | Governance policy script validation |
| PlutusV3 Initialization | 3 | CostModel and govPolicy setup |

**Total: 38 test vectors**

---

## Category 1: Reference Script Tests

### Test: Can Use Reference Scripts

**File:** `can_use_reference_scripts`
**Rule:** UTXOS
**Expected:** Success

```
Initial State:
|-- UTxOs: Empty (genesis funding only)
|-- Reference Scripts: None
|-- Cost Models: PlutusV2/V3 available

Event 1: Setup Transaction (slot 3883681)
|-- Inputs: 1 UTxO
|-- Outputs: 4 UTxOs (including reference script UTxO)
|__ Tx Result: Success

Event 2: Fund Transaction (slot 3883681)
|-- Inputs: 1 UTxO
|-- Outputs: 2 UTxOs
|__ Tx Result: Success

Event 3: Reference Script Spend (slot 3883681)
|-- Inputs: 2 UTxOs (including script-locked)
|-- Reference Inputs: 1 UTxO (containing reference script)
|-- Outputs: 1 UTxO
|-- Script Execution:
|   |-- Uses reference script (not witness)
|   |__ Redeemers: 1
|__ Tx Result: Success
```

**What This Tests:** Validates that scripts can be provided via reference inputs rather than in the transaction witness set.

---

### Test: Can Use Regular Inputs for Reference

**File:** `can_use_regular_inputs_for_reference`
**Rule:** UTXOS
**Expected:** Success

```
Event 3: Script Spend (slot 3883681)
|-- Inputs: 3 UTxOs (one contains the script being used)
|-- Reference Inputs: 0 (script in regular input)
|-- Outputs: 1 UTxO
|-- Script Execution:
|   |-- Script sourced from consumed input
|   |__ Redeemers: 1
|__ Tx Result: Success
```

**What This Tests:** Validates that reference scripts can be located in regular inputs (that are being spent), not just reference inputs.

---

### Test: Fails with Same TxIn in Regular and Reference Inputs

**File:** `fails_with_same_txIn_in_regular_inputs_and_reference_inputs`
**Rule:** UTXOS
**Expected:** Failure

```
Event 3: Invalid Transaction (slot 3883681)
|-- Inputs: 3 UTxOs
|-- Reference Inputs: 1 UTxO (SAME AS REGULAR INPUT)
|-- Outputs: 1 UTxO
|__ Tx Result: FAILURE (duplicate input)
```

**What This Tests:** Ensures the UTXOS rule rejects transactions where the same UTxO appears in both regular inputs and reference inputs.

---

## Category 2: Spending Scripts Without Datum

These tests validate that PlutusV1 and PlutusV2 require datums for spending, while PlutusV3 allows datum-less spending.

### Test: PlutusV1 Spending Without Datum

**File:** `Spending_script_without_a_Datum/PlutusV1`
**Rule:** UTXOS
**Plutus Version:** V1
**Expected:** Failure

```
Event 3: Invalid Spend (slot 3883681)
|-- Inputs: 2 UTxOs (script-locked without datum)
|-- Witness Scripts: 1 (PlutusV1)
|-- Redeemers: 1
|__ Tx Result: FAILURE (no datum)
```

**What This Tests:** PlutusV1 scripts MUST have a datum attached to script outputs. Spending without datum must fail.

---

### Test: PlutusV2 Spending Without Datum

**File:** `Spending_script_without_a_Datum/PlutusV2`
**Rule:** UTXOS
**Plutus Version:** V2
**Expected:** Failure

```
Event 3: Invalid Spend (slot 3883681)
|-- Inputs: 2 UTxOs (script-locked without datum)
|-- Witness Scripts: 1 (PlutusV2)
|-- Redeemers: 1
|__ Tx Result: FAILURE (no datum)
```

**What This Tests:** PlutusV2 scripts also require datums for spending script outputs.

---

### Test: PlutusV3 Spending Without Datum

**File:** `Spending_script_without_a_Datum/PlutusV3`
**Rule:** UTXOS
**Plutus Version:** V3
**Expected:** Success

```
Event 3: Valid Spend (slot 3883681)
|-- Inputs: 2 UTxOs (script-locked without datum)
|-- Witness Scripts: 1 (PlutusV3)
|-- Redeemers: 1
|__ Tx Result: SUCCESS (V3 allows no datum)
```

**What This Tests:** PlutusV3 introduces datum-less spending capability. Scripts can validate without a datum argument.

---

## Category 3: Inline Datum Restrictions

### Test: Fails When Using Inline Datums for PlutusV1

**File:** `fails_when_using_inline_datums_for_PlutusV1`
**Rule:** UTXOS
**Plutus Version:** V1
**Expected:** Failure

```
Event 3: Invalid Transaction (slot 3883681)
|-- Inputs: 2 UTxOs (script-locked with INLINE datum)
|-- Reference Inputs: 1 (with reference script)
|-- Witness Scripts: 0 (using reference)
|-- Redeemers: 1
|__ Tx Result: FAILURE (V1 cannot use inline datums)
```

**What This Tests:** PlutusV1 scripts cannot use inline datums introduced in Babbage. They must use datum hashes with witness datums.

---

## Category 4: Conway Features Fail in PlutusV1 and V2

These tests validate that PlutusV1 and PlutusV2 scripts **cannot access** Conway-era governance features. This is a critical backward compatibility constraint.

### Translated Certificates

Conway introduces new certificate representations. Some certificates translate to equivalent Babbage representations.

#### Test: RegDepositTxCert V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Translated/RegDepositTxCert/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Translated/RegDepositTxCert/V2`

**Expected:** The final transaction with the translated certificate succeeds (certificates translate to V1/V2-compatible form).

```
Event Sequence:
|-- Setup transactions (success)
|__ Final: PlutusV1/V2 script with RegDRepTxCert
    |-- Script uses Conway DRep registration certificate
    |__ Tx Result: SUCCESS (certificate translates)
```

**What This Tests:** RegDepositTxCert can be translated to a form compatible with PlutusV1/V2 scripts.

---

#### Test: UnRegDepositTxCert V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Translated/UnRegDepositTxCert/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Translated/UnRegDepositTxCert/V2`

Similar pattern - unregistration certificates that translate to V1/V2-compatible forms.

---

### Unsupported Certificates

These certificates have NO translation to PlutusV1/V2 and MUST fail.

#### Test: AuthCommitteeHotKeyTxCert V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/AuthCommitteeHotKeyTxCert/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/AuthCommitteeHotKeyTxCert/V2`

**Expected:** Failure

```
Event 3: Invalid Transaction (slot 3883681)
|-- Inputs: 2 UTxOs (script-locked)
|-- Witness Scripts: 1 (PlutusV1 or V2)
|-- Certificates: AuthCommitteeHotKeyTxCert
|-- Redeemers: 1
|-- Conway Features: Committee Certificates
|__ Tx Result: FAILURE (V1/V2 cannot see committee certs)
```

**What This Tests:** PlutusV1/V2 scripts cannot access Conway committee authorization certificates.

---

#### Test: ResignCommitteeColdTxCert V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/ResignCommitteeColdTxCert/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/ResignCommitteeColdTxCert/V2`

**Expected:** Failure (committee resignation not visible to V1/V2)

---

#### Test: DelegTxCert V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/DelegTxCert/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/DelegTxCert/V2`

**Expected:** Failure (Conway delegation format not visible to V1/V2)

---

#### Test: RegDRepTxCert V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/RegDRepTxCert/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/RegDRepTxCert/V2`

**Expected:** Failure (DRep registration not visible to V1/V2)

---

#### Test: UnRegDRepTxCert V1

**File:** `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/UnRegDRepTxCert/V1`

**Expected:** Failure (DRep unregistration not visible to V1)

---

#### Test: UpdateDRepTxCert V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/UpdateDRepTxCert/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/UpdateDRepTxCert/V2`

**Expected:** Failure (DRep update not visible to V1/V2)

---

#### Test: RegDepositDelegTxCert V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/RegDepositDelegTxCert/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Certificates/Unsupported/RegDepositDelegTxCert/V2`

**Expected:** Failure (combined registration/delegation not visible to V1/V2)

---

### Unsupported Transaction Fields

These fields are new in Conway and MUST NOT be accessible to PlutusV1/V2 scripts.

#### Test: VotingProcedures V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Unsupported_Fields/VotingProcedures/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Unsupported_Fields/VotingProcedures/V2`

**Expected:** Failure

```
Event 7: Invalid Transaction (slot 3883681)
|-- Inputs: 2 UTxOs (script-locked)
|-- Witness Scripts: 1 (PlutusV1/V2)
|-- Voting Procedures: 1 voter
|-- Redeemers: 1
|-- Conway Features: Voting Procedures
|__ Tx Result: FAILURE (V1/V2 cannot access votes)
```

**What This Tests:** PlutusV1/V2 scripts cannot access voting procedures in the script context. The transaction body contains votes, but they are invisible to V1/V2 scripts, causing validation failure.

---

#### Test: ProposalProcedures V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Unsupported_Fields/ProposalProcedures/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Unsupported_Fields/ProposalProcedures/V2`

**Expected:** Failure

```
Event 4: Invalid Transaction (slot 3883681)
|-- Inputs: 2 UTxOs (script-locked)
|-- Witness Scripts: 1 (PlutusV1/V2)
|-- Proposal Procedures: 1 proposal
|-- Redeemers: 1
|-- Conway Features: Proposal Procedures
|__ Tx Result: FAILURE (V1/V2 cannot access proposals)
```

**What This Tests:** PlutusV1/V2 scripts cannot access governance proposals.

---

#### Test: CurrentTreasuryValue V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Unsupported_Fields/CurrentTreasuryValue/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Unsupported_Fields/CurrentTreasuryValue/V2`

**Expected:** Failure

```
Event 4: Invalid Transaction (slot 3888001)
|-- Inputs: 2 UTxOs (script-locked)
|-- Witness Scripts: 1 (PlutusV1/V2)
|-- Current Treasury Value: 467110 lovelace
|-- Redeemers: 1
|-- Conway Features: Current Treasury Value
|__ Tx Result: FAILURE (V1/V2 cannot access treasury)
```

**What This Tests:** PlutusV1/V2 scripts cannot access the current treasury value field.

---

#### Test: TreasuryDonation V1/V2

**Files:**
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Unsupported_Fields/TreasuryDonation/V1`
- `Conway_features_fail_in_Plutusdescribe_v1_and_v2/Unsupported_Fields/TreasuryDonation/V2`

**Expected:** Failure

```
Event 3: Invalid Transaction (slot 3883681)
|-- Inputs: 2 UTxOs (script-locked)
|-- Witness Scripts: 1 (PlutusV1/V2)
|-- Treasury Donation: 10000 lovelace
|-- Redeemers: 1
|-- Conway Features: Treasury Donation
|__ Tx Result: FAILURE (V1/V2 cannot access donation)
```

**What This Tests:** PlutusV1/V2 scripts cannot access the treasury donation field.

---

## Category 5: Governance Policy Scripts

These tests validate the governance policy script mechanism introduced in Conway.

### Test: AlwaysSucceeds Plutus GovPolicy Validates

**File:** `Gov_policy_scripts/alwaysSucceeds_Plutus_govPolicy_validates`
**Rule:** UTXOS
**Expected:** Success

**What This Tests:** A PlutusV3 governance policy that always succeeds allows governance actions to proceed.

---

### Test: AlwaysFails Plutus GovPolicy Does Not Validate

**File:** `Gov_policy_scripts/alwaysFails_Plutus_govPolicy_does_not_validate`
**Rule:** UTXOS
**Expected:** Failure

**What This Tests:** A PlutusV3 governance policy that always fails blocks governance actions.

---

### Test: Failing Native Script GovPolicy

**File:** `Gov_policy_scripts/failing_native_script_govPolicy`
**Rule:** UTXOS
**Expected:** Failure

**What This Tests:** Native scripts can also serve as governance policies, and failing policies block actions.

---

## Category 6: PlutusV3 Initialization

These tests validate the proper initialization sequence for PlutusV3 features.

### Test: Updating CostModels and Setting GovPolicy Afterwards Succeeds

**File:** `PlutusV3_Initialization/Updating_CostModels_and_setting_the_govPolicy_afterwards_succeeds`
**Rule:** UTXOS
**Expected:** Success (after proper setup)

```
Event Sequence:
|-- Epochs 0-2: Governance setup, proposals, voting
|-- Epoch 3 (slot 3909601):
|   |-- CostModels updated via ratified proposal
|   |-- PlutusV3 scripts now executable
|   |__ GovPolicy set successfully
|__ Final State: PlutusV3 fully operational
```

**What This Tests:** The proper sequence for enabling PlutusV3:
1. Propose CostModel update
2. Ratify through governance
3. After enactment, PlutusV3 scripts work

---

### Test: Updating CostModels with AlwaysFails GovPolicy Does Not Validate

**File:** `PlutusV3_Initialization/Updating_CostModels_with_alwaysFails_govPolicy_does_not_validate`
**Rule:** UTXOS
**Expected:** Failure at CostModel update

**What This Tests:** When a governance policy script fails, parameter updates cannot proceed.

---

### Test: CostModels with AlwaysSucceeds GovPolicy but No PlutusV3 CostModels Fails

**File:** `PlutusV3_Initialization/Updating_CostModels_with_alwaysSucceeds_govPolicy_but_no_PlutusV3_CostModels_fails`
**Rule:** UTXOS
**Expected:** Failure

**What This Tests:** Even with a passing governance policy, PlutusV3 scripts cannot execute without proper cost models.

---

## Conway UTXOS Script Compatibility Rules

### PlutusV1 Limitations
- Cannot use inline datums
- Cannot access Conway governance fields (votes, proposals, treasury)
- Cannot see Conway certificate types (DRep, Committee, combined)
- Requires datum for spending

### PlutusV2 Limitations
- Cannot access Conway governance fields
- Cannot see Conway certificate types
- Requires datum for spending

### PlutusV3 Capabilities
- Full access to Conway governance features
- Optional datum for spending scripts
- Can serve as governance policy scripts
- Requires PlutusV3 cost models to be enabled

## Test Vector Format

Each test vector is a CBOR-encoded file containing:
- `[0]` config: Network/protocol configuration
- `[1]` initial_state: NewEpochState before events
- `[2]` final_state: NewEpochState after events
- `[3]` events: Array of transaction/epoch events
- `[4]` title: Test name/path

## Related Specifications

- CIP-1694: Cardano Governance
- CIP-69: Plutus Script Context
- Conway Ledger Specification: UTXOS transition rule
- Plutus Core Specification: Version compatibility
