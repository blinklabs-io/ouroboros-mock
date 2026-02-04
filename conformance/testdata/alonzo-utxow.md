
# Alonzo UTXOW Conformance Test Vectors

This directory contains conformance test vectors for the Alonzo-era **UTXOW** (UTxO Witness) validation rules, tested within the Conway era context. These tests verify that witness set validation works correctly for Plutus scripts across all three versions (PlutusV1, PlutusV2, PlutusV3).

## Overview

The UTXOW rule is responsible for validating the **witness set** of a transaction, ensuring:

1. **VKey Witnesses** - Cryptographic signatures match required signers
2. **Script Witnesses** - Scripts are provided for script-locked UTxOs
3. **Datum Witnesses** - Datums are provided for Plutus script inputs
4. **Redeemer Witnesses** - Redeemers are provided for each script execution
5. **Script Data Hash** - The computed hash matches the transaction body's `script_data_hash`

## Test Vector Format

Each test vector is a binary CBOR file with the following structure:

```
[0] config:        array[13]  - Network/protocol configuration
[1] initial_state: array[7]   - NewEpochState before events
[2] final_state:   array[7]   - NewEpochState after events
[3] events:        array[N]   - Transaction events [type, tx_bytes, success, slot]
[4] title:         string     - Test name/path
```

The `success` field in transaction events indicates:
- `true` - Transaction should be accepted (phase-1 validation passes)
- `false` - Transaction should be rejected (phase-1 validation fails)

## Directory Structure

```
UTXOW/
|-- Valid_transactions/
|   |-- PlutusV1/
|   |   |-- Validating_SPEND_script
|   |   |-- Validating_MINT_script
|   |   |-- Validating_CERT_script
|   |   |-- Validating_WITHDRAWAL_script
|   |   |-- Not_validating_SPEND_script
|   |   |-- Not_validating_MINT_script
|   |   |-- Not_validating_CERT_script
|   |   +-- Not_validating_WITHDRAWAL_script
|   |-- PlutusV2/
|   |   +-- [same structure as PlutusV1]
|   +-- PlutusV3/
|       +-- [same structure as PlutusV1]
+-- Invalid_transactions/
    |-- Phase_1_script_failure
    |-- PlutusV1/
    |   |-- Extra_Redeemer/
    |   |   |-- Spending
    |   |   |-- Minting
    |   |   +-- Multiple_equal_plutus-locked_certs
    |   |-- PPViewHashesDontMatch/
    |   |   |-- Missing
    |   |   +-- Mismatched
    |   |-- Missing_phase-2_script_witness
    |   |-- MissingRedeemers
    |   |-- MissingRequiredDatums
    |   |-- Missing_witness_for_collateral_input
    |   |-- No_ExtraRedeemers_on_same_script_certificates
    |   |-- NotAllowedSupplementalDatums
    |   |-- Redeemer_with_incorrect_purpose
    |   +-- UnspendableUTxONoDatumHash
    |-- PlutusV2/
    |   +-- [same structure as PlutusV1]
    +-- PlutusV3/
        +-- [same structure as PlutusV1]
```

---

# Valid Transaction Tests

## Validating Script Tests

These tests verify that transactions with **phase-2 valid** Plutus scripts are correctly accepted.

---

## Test: Validating_SPEND_script

**Files:**
- `Valid_transactions/PlutusV1/Validating_SPEND_script`
- `Valid_transactions/PlutusV2/Validating_SPEND_script`
- `Valid_transactions/PlutusV3/Validating_SPEND_script`

**Rule:** UTXOW
**Expected:** Success

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked output (alwaysSucceeds script)
|   |-- UTxO B: Regular ADA for fees
|   +-- UTxO C: Collateral input (key-witnessed)
|-- Script: AlwaysSucceeds Plutus[V1|V2|V3] script

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A (spending script-locked output)
|-- Collateral:
|   +-- UTxO C
|-- Witnesses Provided:
|   |-- VKey Witnesses: [signature for UTxO B, signature for collateral]
|   |-- Script Witnesses: [alwaysSucceeds script]
|   |-- Datum Witnesses: [datum matching UTxO A's datum hash]
|   +-- Redeemer Witnesses: [{tag: SPEND, index: 0, data: unit, ex_units: {...}}]
|-- Script Data Hash: Valid (matches computed hash)
|-- IsValid: true
|-- Script Execution: alwaysSucceeds returns True
+-- Result: SUCCESS - All witnesses valid, script succeeds

Final State:
|-- UTxOs: Updated (UTxO A consumed, new outputs created)
+-- Collateral: NOT consumed (IsValid=true)
```

### What This Tests

Validates the complete happy path for spending a Plutus script-locked UTxO:

1. **Script witness present** - The Plutus script is included in the witness set
2. **Datum witness present** - The datum matching the UTxO's datum hash is provided
3. **Redeemer present** - A redeemer with tag=SPEND and correct index is provided
4. **Script data hash valid** - Blake2b256(redeemers || datums || language_views) matches tx body
5. **Script executes successfully** - The script returns True

---

## Test: Validating_MINT_script

**Files:**
- `Valid_transactions/PlutusV1/Validating_MINT_script`
- `Valid_transactions/PlutusV2/Validating_MINT_script`
- `Valid_transactions/PlutusV3/Validating_MINT_script`

**Rule:** UTXOW
**Expected:** Success

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Regular ADA for fees
|   +-- UTxO B: Collateral input
|-- Script: AlwaysSucceeds minting policy

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A
|-- Collateral:
|   +-- UTxO B
|-- Mint:
|   +-- PolicyId (alwaysSucceeds hash): +100 TokenName
|-- Witnesses Provided:
|   |-- VKey Witnesses: [signatures]
|   |-- Script Witnesses: [alwaysSucceeds minting policy]
|   |-- Datum Witnesses: []
|   +-- Redeemer Witnesses: [{tag: MINT, index: 0, data: unit, ex_units: {...}}]
|-- Script Data Hash: Valid
|-- IsValid: true
+-- Result: SUCCESS - Minting policy validates

Final State:
|-- UTxOs: Updated with minted tokens in output
+-- Asset Supply: +100 TokenName under PolicyId
```

### What This Tests

Validates minting tokens with a Plutus minting policy:

1. **Minting policy witness** - Script for the policy ID is provided
2. **MINT redeemer** - Redeemer with tag=MINT for the policy index
3. **No datum required** - Minting doesn't require datums
4. **Script data hash** - Includes the minting redeemer

---

## Test: Validating_CERT_script

**Files:**
- `Valid_transactions/PlutusV1/Validating_CERT_script`
- `Valid_transactions/PlutusV2/Validating_CERT_script`
- `Valid_transactions/PlutusV3/Validating_CERT_script`

**Rule:** UTXOW
**Expected:** Success

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Regular ADA for fees + deposit
|   +-- UTxO B: Collateral input
|-- Script: AlwaysSucceeds staking script
|-- Stake Registrations: []

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A
|-- Collateral:
|   +-- UTxO B
|-- Certificates:
|   +-- StakeRegistration(script_credential)
|-- Witnesses Provided:
|   |-- VKey Witnesses: [signatures]
|   |-- Script Witnesses: [alwaysSucceeds staking script]
|   |-- Datum Witnesses: []
|   +-- Redeemer Witnesses: [{tag: CERT, index: 0, data: unit, ex_units: {...}}]
|-- Script Data Hash: Valid
|-- IsValid: true
+-- Result: SUCCESS - Certificate script validates

Final State:
|-- Stake Registrations: [script_credential]
+-- Deposits: +keyDeposit
```

### What This Tests

Validates certificates controlled by Plutus scripts:

1. **Certificate script witness** - Script for the stake credential is provided
2. **CERT redeemer** - Redeemer with tag=CERT for the certificate index
3. **Certificate processing** - Stake registration succeeds

---

## Test: Validating_WITHDRAWAL_script

**Files:**
- `Valid_transactions/PlutusV1/Validating_WITHDRAWAL_script`
- `Valid_transactions/PlutusV2/Validating_WITHDRAWAL_script`
- `Valid_transactions/PlutusV3/Validating_WITHDRAWAL_script`

**Rule:** UTXOW
**Expected:** Success

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Regular ADA
|   +-- UTxO B: Collateral input
|-- Script: AlwaysSucceeds staking script
|-- Reward Accounts:
|   +-- script_reward_address: 1000000 lovelace

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A
|-- Collateral:
|   +-- UTxO B
|-- Withdrawals:
|   +-- script_reward_address: 1000000 lovelace
|-- Witnesses Provided:
|   |-- VKey Witnesses: [signatures]
|   |-- Script Witnesses: [alwaysSucceeds staking script]
|   |-- Datum Witnesses: []
|   +-- Redeemer Witnesses: [{tag: REWARD, index: 0, data: unit, ex_units: {...}}]
|-- Script Data Hash: Valid
|-- IsValid: true
+-- Result: SUCCESS - Withdrawal script validates

Final State:
|-- Reward Accounts:
|   +-- script_reward_address: 0 lovelace
+-- Transaction outputs include withdrawn amount
```

### What This Tests

Validates withdrawals from script-controlled reward addresses:

1. **Withdrawal script witness** - Script for the reward credential is provided
2. **REWARD redeemer** - Redeemer with tag=REWARD for the withdrawal index
3. **Full balance withdrawal** - Must withdraw entire balance

---

## Not Validating Script Tests

These tests verify that transactions with **phase-2 invalid** Plutus scripts (IsValid=false) are correctly handled. The transaction is accepted into the block, but collateral is consumed.

---

## Test: Not_validating_SPEND_script

**Files:**
- `Valid_transactions/PlutusV1/Not_validating_SPEND_script`
- `Valid_transactions/PlutusV2/Not_validating_SPEND_script`
- `Valid_transactions/PlutusV3/Not_validating_SPEND_script`

**Rule:** UTXOW
**Expected:** Success (with IsValid=false)

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked output (alwaysFails script)
|   |-- UTxO B: Regular ADA for fees
|   +-- UTxO C: Collateral input (sufficient for collateral percentage)

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A (spending script-locked output)
|-- Collateral:
|   +-- UTxO C (will be consumed due to script failure)
|-- Collateral Return: [output returning excess collateral]
|-- Witnesses Provided:
|   |-- VKey Witnesses: [signature for collateral]
|   |-- Script Witnesses: [alwaysFails script]
|   |-- Datum Witnesses: [datum for UTxO A]
|   +-- Redeemer Witnesses: [{tag: SPEND, index: 0, data: unit, ex_units: {...}}]
|-- Script Data Hash: Valid
|-- IsValid: false (script will fail phase-2)
+-- Result: SUCCESS - Transaction accepted, collateral consumed

Final State:
|-- UTxOs:
|   |-- UTxO A: STILL EXISTS (not consumed)
|   |-- UTxO C: CONSUMED (collateral taken)
|   +-- Collateral Return Output: Created if specified
```

### What This Tests

Validates phase-2 failure handling:

1. **IsValid=false accepted** - Transaction with failing script is accepted
2. **Collateral consumed** - Collateral inputs are spent when IsValid=false
3. **Regular inputs preserved** - Script-locked inputs are NOT consumed
4. **Witness validation still applies** - All phase-1 checks must pass

---

## Test: Not_validating_MINT_script

**Files:**
- `Valid_transactions/PlutusV1/Not_validating_MINT_script`
- `Valid_transactions/PlutusV2/Not_validating_MINT_script`
- `Valid_transactions/PlutusV3/Not_validating_MINT_script`

**Rule:** UTXOW
**Expected:** Success (with IsValid=false)

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Regular ADA
|   +-- UTxO B: Collateral input

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A
|-- Collateral:
|   +-- UTxO B
|-- Mint:
|   +-- PolicyId (alwaysFails hash): +100 TokenName
|-- Witnesses Provided:
|   |-- Script Witnesses: [alwaysFails minting policy]
|   +-- Redeemer Witnesses: [{tag: MINT, index: 0, ...}]
|-- IsValid: false
+-- Result: SUCCESS - Transaction accepted, no tokens minted

Final State:
|-- UTxOs: Collateral consumed
|-- Asset Supply: UNCHANGED (mint not applied)
```

### What This Tests

Validates failed minting:

1. **Mint not applied** - When IsValid=false, the mint field is ignored
2. **Collateral consumed** - Fee compensation via collateral

---

## Test: Not_validating_CERT_script

**Files:**
- `Valid_transactions/PlutusV1/Not_validating_CERT_script`
- `Valid_transactions/PlutusV2/Not_validating_CERT_script`
- `Valid_transactions/PlutusV3/Not_validating_CERT_script`

**Rule:** UTXOW
**Expected:** Success (with IsValid=false)

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Regular ADA
|   +-- UTxO B: Collateral input
|-- Stake Registrations: []

Event 1: Transaction at slot X
|-- Certificates:
|   +-- StakeRegistration(script_credential) with alwaysFails script
|-- IsValid: false
+-- Result: SUCCESS - Transaction accepted, certificate not processed

Final State:
|-- Stake Registrations: [] (UNCHANGED)
|-- Collateral: Consumed
```

### What This Tests

Validates failed certificate processing:

1. **Certificate not applied** - Registration doesn't happen with IsValid=false
2. **No deposit taken** - Deposit only taken on successful registration

---

## Test: Not_validating_WITHDRAWAL_script

**Files:**
- `Valid_transactions/PlutusV1/Not_validating_WITHDRAWAL_script`
- `Valid_transactions/PlutusV2/Not_validating_WITHDRAWAL_script`
- `Valid_transactions/PlutusV3/Not_validating_WITHDRAWAL_script`

**Rule:** UTXOW
**Expected:** Success (with IsValid=false)

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Regular ADA
|   +-- UTxO B: Collateral input
|-- Reward Accounts:
|   +-- script_reward_address: 1000000 lovelace

Event 1: Transaction at slot X
|-- Withdrawals:
|   +-- script_reward_address: 1000000 lovelace (alwaysFails script)
|-- IsValid: false
+-- Result: SUCCESS - Transaction accepted, withdrawal not processed

Final State:
|-- Reward Accounts:
|   +-- script_reward_address: 1000000 lovelace (UNCHANGED)
|-- Collateral: Consumed
```

### What This Tests

Validates failed withdrawal:

1. **Withdrawal not applied** - Rewards remain in account
2. **Value conservation** - Collateral consumed but rewards not withdrawn

---

# Invalid Transaction Tests

These tests verify that malformed transactions are correctly **rejected** during phase-1 validation.

---

## Test: Phase_1_script_failure

**File:** `Invalid_transactions/Phase_1_script_failure`

**Rule:** UTXOW
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked (native timelock script)
|   +-- UTxO B: Collateral

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A (timelock not yet satisfied)
|-- Witnesses Provided:
|   +-- Native Script: [timelock script requiring slot > X]
|-- Script Validation:
|   +-- Timelock: FAILS (current slot doesn't satisfy time condition)
+-- Result: FAILURE - Native script failure is phase-1

Final State: UNCHANGED
```

### What This Tests

Validates that native script failures are phase-1 rejections:

1. **Native scripts evaluated in phase-1** - Unlike Plutus scripts
2. **No collateral consumed** - Phase-1 failures don't consume collateral
3. **Transaction rejected** - Not included in block

---

## Test: Extra_Redeemer/Spending

**Files:**
- `Invalid_transactions/PlutusV1/Extra_Redeemer/Spending`
- `Invalid_transactions/PlutusV2/Extra_Redeemer/Spending`
- `Invalid_transactions/PlutusV3/Extra_Redeemer/Spending`

**Rule:** UTXOW - ExtraRedeemers
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked output
|   +-- UTxO B: Collateral

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A (index 0)
|-- Witnesses Provided:
|   |-- Script Witnesses: [script]
|   |-- Datum Witnesses: [datum]
|   +-- Redeemer Witnesses:
|       |-- {tag: SPEND, index: 0, ...} (valid)
|       +-- {tag: SPEND, index: 1, ...} (EXTRA - no input at index 1)
|-- Redeemer Validation:
|   +-- Index 1 >= input count (1) - FAILS
+-- Result: FAILURE - ExtraRedeemers error

Final State: UNCHANGED
```

### What This Tests

Validates that redeemers must have valid purposes:

1. **Redeemer index bounds checking** - Index must be < count of that purpose type
2. **Extra redeemers rejected** - Cannot include unused redeemers

---

## Test: Extra_Redeemer/Minting

**Files:**
- `Invalid_transactions/PlutusV1/Extra_Redeemer/Minting`
- `Invalid_transactions/PlutusV2/Extra_Redeemer/Minting`
- `Invalid_transactions/PlutusV3/Extra_Redeemer/Minting`

**Rule:** UTXOW - ExtraRedeemers
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   +-- UTxO A: Regular ADA with collateral

Event 1: Transaction at slot X
|-- Mint:
|   +-- PolicyId_A: +100 tokens (1 policy, index 0)
|-- Witnesses Provided:
|   |-- Script Witnesses: [PolicyId_A script]
|   +-- Redeemer Witnesses:
|       |-- {tag: MINT, index: 0, ...} (valid)
|       +-- {tag: MINT, index: 1, ...} (EXTRA - only 1 policy)
|-- Redeemer Validation:
|   +-- Index 1 >= policy count (1) - FAILS
+-- Result: FAILURE - ExtraRedeemers error

Final State: UNCHANGED
```

### What This Tests

Validates mint redeemer bounds:

1. **Policy count validation** - Mint redeemer index < distinct policy count
2. **Sorted policy ordering** - Policies are sorted, redeemer index maps to sorted position

---

## Test: Extra_Redeemer/Multiple_equal_plutus-locked_certs

**Files:**
- `Invalid_transactions/PlutusV1/Extra_Redeemer/Multiple_equal_plutus-locked_certs`
- `Invalid_transactions/PlutusV2/Extra_Redeemer/Multiple_equal_plutus-locked_certs`
- `Invalid_transactions/PlutusV3/Extra_Redeemer/Multiple_equal_plutus-locked_certs`

**Rule:** UTXOW - ExtraRedeemers
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs with collateral

Event 1: Transaction at slot X
|-- Certificates:
|   |-- Cert 0: StakeRegistration(script_cred)
|   +-- Cert 1: StakeDelegation(script_cred, pool)
|-- Witnesses Provided:
|   +-- Redeemer Witnesses:
|       |-- {tag: CERT, index: 0, ...}
|       |-- {tag: CERT, index: 1, ...}
|       +-- {tag: CERT, index: 2, ...} (EXTRA - only 2 certs)
|-- Redeemer Validation:
|   +-- Index 2 >= cert count (2) - FAILS
+-- Result: FAILURE - ExtraRedeemers error

Final State: UNCHANGED
```

### What This Tests

Validates certificate redeemer bounds with multiple same-script certificates:

1. **Certificate count** - Each certificate needs at most one redeemer
2. **Same script, different redeemers** - Each cert index maps to its redeemer

---

## Test: PPViewHashesDontMatch/Missing

**Files:**
- `Invalid_transactions/PlutusV1/PPViewHashesDontMatch/Missing`
- `Invalid_transactions/PlutusV2/PPViewHashesDontMatch/Missing`
- `Invalid_transactions/PlutusV3/PPViewHashesDontMatch/Missing`

**Rule:** UTXOW - PPViewHashesDontMatch
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked
|   +-- UTxO B: Collateral
|-- Protocol Parameters: [with cost models]

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A
|-- Transaction Body:
|   +-- script_data_hash: NULL/MISSING
|-- Witnesses Provided:
|   |-- Script Witnesses: [plutus script]
|   |-- Datum Witnesses: [datum]
|   +-- Redeemer Witnesses: [redeemer]
|-- Script Data Hash Validation:
|   |-- Expected: Blake2b256(redeemers || datums || language_views)
|   +-- Actual: MISSING
+-- Result: FAILURE - PPViewHashesDontMatch (missing hash)

Final State: UNCHANGED
```

### What This Tests

Validates that script_data_hash is required when Plutus scripts are used:

1. **Script data hash required** - Cannot be null/missing with redeemers
2. **Deterministic validation** - Hash ensures same validation across nodes

---

## Test: PPViewHashesDontMatch/Mismatched

**Files:**
- `Invalid_transactions/PlutusV1/PPViewHashesDontMatch/Mismatched`
- `Invalid_transactions/PlutusV2/PPViewHashesDontMatch/Mismatched`
- `Invalid_transactions/PlutusV3/PPViewHashesDontMatch/Mismatched`

**Rule:** UTXOW - PPViewHashesDontMatch
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked
|   +-- UTxO B: Collateral
|-- Protocol Parameters: [with cost models]

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A
|-- Transaction Body:
|   +-- script_data_hash: 0xDEADBEEF... (incorrect hash)
|-- Witnesses Provided:
|   |-- Script Witnesses: [plutus script]
|   |-- Datum Witnesses: [datum]
|   +-- Redeemer Witnesses: [redeemer]
|-- Script Data Hash Validation:
|   |-- Expected: 0xABCD1234... (computed from witnesses)
|   +-- Actual: 0xDEADBEEF... (doesn't match)
+-- Result: FAILURE - PPViewHashesDontMatch (hash mismatch)

Final State: UNCHANGED
```

### What This Tests

Validates script_data_hash correctness:

1. **Hash computation** - `Blake2b256(redeemers || datums || language_views)`
2. **Language views encoding** - Version-specific cost model encoding
3. **Exact match required** - Any mismatch causes rejection

---

## Test: Missing_phase-2_script_witness

**Files:**
- `Invalid_transactions/PlutusV1/Missing_phase-2_script_witness`
- `Invalid_transactions/PlutusV2/Missing_phase-2_script_witness`
- `Invalid_transactions/PlutusV3/Missing_phase-2_script_witness`

**Rule:** UTXOW - MissingScriptWitnesses
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked (Plutus script address)
|   +-- UTxO B: Collateral

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A (script-locked)
|-- Witnesses Provided:
|   |-- VKey Witnesses: [collateral signature]
|   |-- Script Witnesses: [] (MISSING - no Plutus script)
|   |-- Datum Witnesses: [datum]
|   +-- Redeemer Witnesses: [redeemer for index 0]
|-- Script Witness Validation:
|   |-- Required: Script hash from UTxO A address
|   +-- Provided: NONE
+-- Result: FAILURE - MissingScriptWitnesses

Final State: UNCHANGED
```

### What This Tests

Validates that Plutus scripts must be provided:

1. **Script hash resolution** - Extract required script from payment credential
2. **Witness set check** - Script must be in witness set or reference input
3. **Phase-1 rejection** - Missing script is phase-1 failure

---

## Test: MissingRedeemers

**Files:**
- `Invalid_transactions/PlutusV1/MissingRedeemers`
- `Invalid_transactions/PlutusV2/MissingRedeemers`
- `Invalid_transactions/PlutusV3/MissingRedeemers`

**Rule:** UTXOW - MissingRedeemers
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked
|   +-- UTxO B: Collateral

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A (script-locked, index 0)
|-- Witnesses Provided:
|   |-- Script Witnesses: [plutus script]
|   |-- Datum Witnesses: [datum]
|   +-- Redeemer Witnesses: [] (MISSING)
|-- Redeemer Validation:
|   |-- Expected: {tag: SPEND, index: 0, ...}
|   +-- Provided: NONE
+-- Result: FAILURE - MissingRedeemers

Final State: UNCHANGED
```

### What This Tests

Validates that redeemers are required for Plutus scripts:

1. **Redeemer presence** - Each script execution needs a redeemer
2. **Purpose mapping** - Redeemer tag+index must match script purpose

---

## Test: MissingRequiredDatums

**Files:**
- `Invalid_transactions/PlutusV1/MissingRequiredDatums`
- `Invalid_transactions/PlutusV2/MissingRequiredDatums`
- `Invalid_transactions/PlutusV3/MissingRequiredDatums`

**Rule:** UTXOW - MissingRequiredDatums
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked with datum_hash=0xABC...
|   +-- UTxO B: Collateral

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A (requires datum)
|-- Witnesses Provided:
|   |-- Script Witnesses: [plutus script]
|   |-- Datum Witnesses: [] (MISSING)
|   +-- Redeemer Witnesses: [redeemer]
|-- Datum Validation:
|   |-- Required: Datum with hash 0xABC...
|   +-- Provided: NONE (no datum, no inline datum)
+-- Result: FAILURE - MissingRequiredDatums

Final State: UNCHANGED
```

### What This Tests

Validates datum requirements for PlutusV1:

1. **Datum hash lookup** - UTxO has datum_hash field
2. **Witness set check** - Datum must be in witness set
3. **PlutusV2+ inline datums** - Can alternatively have inline datum in UTxO

---

## Test: Missing_witness_for_collateral_input

**Files:**
- `Invalid_transactions/PlutusV1/Missing_witness_for_collateral_input`
- `Invalid_transactions/PlutusV2/Missing_witness_for_collateral_input`
- `Invalid_transactions/PlutusV3/Missing_witness_for_collateral_input`

**Rule:** UTXOW - MissingVKeyWitnesses
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked
|   +-- UTxO B: Key-locked (pubkey address, for collateral)

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A
|-- Collateral:
|   +-- UTxO B (requires signature)
|-- Witnesses Provided:
|   |-- VKey Witnesses: [] (MISSING collateral signature)
|   |-- Script Witnesses: [script]
|   |-- Datum Witnesses: [datum]
|   +-- Redeemer Witnesses: [redeemer]
|-- VKey Witness Validation:
|   |-- Required for collateral: signature for UTxO B's pubkey hash
|   +-- Provided: NONE
+-- Result: FAILURE - MissingVKeyWitnesses (for collateral)

Final State: UNCHANGED
```

### What This Tests

Validates collateral witness requirements:

1. **Collateral must be key-witnessed** - Cannot be script-locked
2. **Signature required** - VKey witness for collateral pubkey
3. **Phase-1 check** - Missing signature is phase-1 failure

---

## Test: No_ExtraRedeemers_on_same_script_certificates

**Files:**
- `Invalid_transactions/PlutusV1/No_ExtraRedeemers_on_same_script_certificates`
- `Invalid_transactions/PlutusV2/No_ExtraRedeemers_on_same_script_certificates`
- `Invalid_transactions/PlutusV3/No_ExtraRedeemers_on_same_script_certificates`

**Rule:** UTXOW - ExtraRedeemers (negative test)
**Expected:** Failure (if extra redeemers) or Success (if exactly right number)

### State Change Diagram

```
Initial State:
|-- UTxOs with collateral

Event 1: Transaction at slot X
|-- Certificates:
|   |-- Cert 0: StakeRegistration(script_cred_A)
|   +-- Cert 1: StakeDelegation(script_cred_A, pool) (same script)
|-- Witnesses Provided:
|   +-- Redeemer Witnesses:
|       |-- {tag: CERT, index: 0, ...}
|       +-- {tag: CERT, index: 1, ...}
|       (No extra redeemers - exactly 2 for 2 certs)
|-- Redeemer Validation:
|   +-- Each certificate index has exactly one redeemer
+-- Result: Depends on test - validates no false positives

Final State: Depends on test outcome
```

### What This Tests

Validates that same-script certificates don't cause false "extra redeemer" errors:

1. **Certificate-redeemer mapping** - 1:1 mapping by index, not by script
2. **Same script, multiple certs** - Each cert needs its own redeemer
3. **No false positives** - Don't reject valid multi-cert transactions

---

## Test: NotAllowedSupplementalDatums

**Files:**
- `Invalid_transactions/PlutusV1/NotAllowedSupplementalDatums`
- `Invalid_transactions/PlutusV2/NotAllowedSupplementalDatums`
- `Invalid_transactions/PlutusV3/NotAllowedSupplementalDatums`

**Rule:** UTXOW - NotAllowedSupplementalDatums
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked with datum_hash=0xABC...
|   +-- UTxO B: Collateral

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A
|-- Witnesses Provided:
|   |-- Script Witnesses: [script]
|   |-- Datum Witnesses:
|   |   |-- Datum with hash 0xABC... (required)
|   |   +-- Datum with hash 0xDEF... (EXTRA - not referenced)
|   +-- Redeemer Witnesses: [redeemer]
|-- Datum Validation:
|   |-- Required datums: [0xABC...]
|   |-- Allowed supplemental: [] (none for V1)
|   +-- Provided: [0xABC..., 0xDEF...]
+-- Result: FAILURE - NotAllowedSupplementalDatums

Final State: UNCHANGED
```

### What This Tests

Validates datum witness restrictions:

1. **PlutusV1 strict datums** - Only required datums allowed in witness set
2. **PlutusV2+ relaxed** - Supplemental datums allowed (for reference input observation)
3. **Extra datum detection** - Unreferenced datums cause failure for V1

---

## Test: Redeemer_with_incorrect_purpose

**Files:**
- `Invalid_transactions/PlutusV1/Redeemer_with_incorrect_purpose`
- `Invalid_transactions/PlutusV2/Redeemer_with_incorrect_purpose`
- `Invalid_transactions/PlutusV3/Redeemer_with_incorrect_purpose`

**Rule:** UTXOW - ExtraRedeemers / MissingRedeemers
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked (index 0 in sorted inputs)
|   +-- UTxO B: Collateral

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A (script-locked)
|-- Witnesses Provided:
|   |-- Script Witnesses: [script]
|   |-- Datum Witnesses: [datum]
|   +-- Redeemer Witnesses:
|       +-- {tag: MINT, index: 0, ...} (WRONG - should be SPEND)
|-- Redeemer Validation:
|   |-- Expected: {tag: SPEND, index: 0}
|   |-- Provided: {tag: MINT, index: 0}
|   +-- Result: SPEND redeemer missing, MINT redeemer extra
+-- Result: FAILURE - Wrong redeemer purpose

Final State: UNCHANGED
```

### What This Tests

Validates redeemer purpose correctness:

1. **Purpose tag matching** - SPEND/MINT/CERT/REWARD must match usage
2. **Missing vs Extra** - Wrong tag creates both missing and extra conditions

---

## Test: UnspendableUTxONoDatumHash

**Files:**
- `Invalid_transactions/PlutusV1/UnspendableUTxONoDatumHash`
- `Invalid_transactions/PlutusV2/UnspendableUTxONoDatumHash`
- `Invalid_transactions/PlutusV3/UnspendableUTxONoDatumHash`

**Rule:** UTXOW - UnspendableUTxONoDatumHash
**Expected:** Failure

### State Change Diagram

```
Initial State:
|-- UTxOs:
|   |-- UTxO A: Script-locked BUT no datum_hash (invalid UTxO state)
|   +-- UTxO B: Collateral

Event 1: Transaction at slot X
|-- Inputs:
|   +-- UTxO A (script address, no datum hash)
|-- Witnesses Provided:
|   |-- Script Witnesses: [script]
|   |-- Datum Witnesses: [] (cannot provide - no hash to match)
|   +-- Redeemer Witnesses: [redeemer]
|-- Datum Validation:
|   |-- UTxO is Plutus script address
|   |-- UTxO has NO datum_hash field
|   +-- Cannot provide datum without hash to match
+-- Result: FAILURE - UnspendableUTxONoDatumHash

Final State: UNCHANGED
```

### What This Tests

Validates that PlutusV1 script outputs require datum hashes:

1. **Datum hash required for V1** - Script outputs must have datum_hash
2. **Unspendable detection** - UTxO exists but cannot be validly spent
3. **PlutusV2+ inline datums** - Can have inline datum instead of hash

---

# Summary: Witness Set CBOR Keys

Conway-era witness set structure (CBOR map keys):

| Key | Field | Description |
|-----|-------|-------------|
| 0 | vkeywitness | VKey witnesses (signatures) |
| 1 | native_script | Native scripts |
| 2 | bootstrap_witness | Byron bootstrap witnesses |
| 3 | plutus_v1_script | PlutusV1 scripts |
| 4 | plutus_data | Datums |
| 5 | redeemer | Redeemers |
| 6 | plutus_v2_script | PlutusV2 scripts |
| 7 | plutus_v3_script | PlutusV3 scripts |

---

# Script Data Hash Computation

The `script_data_hash` in the transaction body is computed as:

```
script_data_hash = Blake2b256(redeemers_cbor || datums_cbor || language_views)
```

Where:
- `redeemers_cbor` - Original CBOR encoding of redeemers
- `datums_cbor` - Original CBOR encoding of datums (only if non-empty)
- `language_views` - Cost model encoding per language

### Language Views Encoding

**PlutusV1** (double-bagged for historical compatibility):
```
{ 0x4100: 0x5F[...cost_model_values...]0xFF }
```
- Key: `serialize(serialize(0))` = bytestring containing 0x00
- Value: Indefinite-length list of cost values, wrapped in bytestring

**PlutusV2/V3**:
```
{ 0x01: [...cost_model_values...] }  // V2
{ 0x02: [...cost_model_values...] }  // V3
```
- Key: `serialize(version)` = 0x01 or 0x02
- Value: Definite-length list of cost values (no wrapper)

---

# Test Count Summary

| Category | PlutusV1 | PlutusV2 | PlutusV3 | Total |
|----------|----------|----------|----------|-------|
| Valid (Validating) | 4 | 4 | 4 | 12 |
| Valid (Not validating) | 4 | 4 | 4 | 12 |
| Invalid | 13 | 12 | 12 | 37 |
| Phase_1_script_failure | - | - | - | 1 |
| **Total** | **21** | **20** | **20** | **62** |

---

# References

- [CIP-55: Script Data Hash](https://cips.cardano.org/cip/CIP-0055)
- [Alonzo Ledger Specification](https://github.com/intersectmbo/cardano-ledger/blob/master/eras/alonzo/impl/src/Cardano/Ledger/Alonzo/Rules/Utxow.hs)
- [Conway Era Rules](https://github.com/intersectmbo/cardano-ledger/tree/master/eras/conway/impl/src/Cardano/Ledger/Conway/Rules)
- [Amaru Test Vectors Source](https://github.com/pragma-org/amaru)

