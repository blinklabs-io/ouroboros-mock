
# Babbage Era Conformance Test Vectors

This directory contains conformance test vectors for Babbage-era features implemented in the Conway ledger rules. The Babbage era introduced several key features that are tested here:

1. **Reference Scripts** - Scripts can be stored in UTxO outputs and referenced by transactions
2. **Inline Datums** - Datum values can be stored directly in UTxO outputs
3. **Reference Inputs** - Transactions can reference UTxOs without spending them

## Test Vector Structure

Each test vector file is a CBOR-encoded 5-element array:
- `[0]` config - Network/protocol configuration
- `[1]` initial_state - NewEpochState before events
- `[2]` final_state - NewEpochState after events
- `[3]` events - Transaction/epoch events to process
- `[4]` title - Test name/path

## Rule: UTXOW

All tests in this directory validate the UTXOW (UTxO Witness) rule, which checks:
- Transaction witness validity
- Script execution and validation
- Redeemer presence and correctness
- Reference script/datum resolution

---

## Test: MalformedReferenceScripts

**File:** `UTXOW/MalformedReferenceScripts`
**Rule:** UTXOW
**Feature:** Reference Scripts - Malformed Script Detection
**Expected:** FAILURE

### Purpose

This test validates that the ledger correctly rejects transactions that attempt to create UTxO outputs with malformed (invalid) reference scripts. Reference scripts must be valid, well-formed Plutus scripts.

### State Change Diagram

```
Initial State:
+-- Epoch: 899
+-- UTxOs: (none in parsed state - genesis UTxO exists)
    +-- 03170a2e...#0: Available for spending (genesis funding)

Event 1: PassTick (Slot 1)
+-- Advances the slot counter
+-- No state changes

Event 2: Transaction at Slot 3,883,681
+-- TX Hash: 8e3b8914e9a178ec6002a1a1c78ce5ae09c1fbce6d170b5eb35c42f2ea1480bc
+-- Inputs:
|   +-- 03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314#0
|
+-- Outputs:
|   +-- Output 0: 918,030 lovelace
|   |   +-- Address: addr_test1vq7gwh8q7ermmnryku8x9q2xsraajw0cltczqwucydhdu7c4kye2h
|   |   +-- Datum Hash: 0x00000000... (zero hash)
|   |   +-- Reference Script: ca1cd6290d7526e0382e0b866b56b7b95de59da1d31f7bb18996ee41
|   |       [MALFORMED - Invalid script bytes]
|   |
|   +-- Output 1: 44,999,999,998,914,225 lovelace
|       +-- Address: addr_test1qp4ht6hmkdyszttrxhrjqrfud7a37pn496zwujzupmr8mh0lj2qzs...
|       +-- Datum Hash: 0x00000000... (zero hash)
|
+-- IsValid: true (phase-1 valid, but fails phase-2 due to malformed script)
+-- Result: FAILURE - Malformed reference script detected

Final State:
+-- Epoch: 899
+-- UTxOs: (unchanged from initial - transaction rejected)
```

### What This Tests

The test verifies that the UTXOW rule properly validates reference scripts stored in transaction outputs. When a transaction attempts to create a UTxO output containing a reference script, the ledger must:

1. Decode the reference script bytes
2. Verify the script is well-formed according to its language (PlutusV1/V2/V3)
3. Reject the transaction if the script is malformed

The script hash `ca1cd6290d7526e0382e0b866b56b7b95de59da1d31f7bb18996ee41` corresponds to invalid/corrupted script bytes labeled "invalid" in the test data.

### Validation Rule

- **MalformedScripts (MALFORMED_SCRIPT_WITNESS)**: The transaction contains a malformed script in either the witness set or as a reference script in an output.

---

## Test: MalformedScriptWitnesses

**File:** `UTXOW/MalformedScriptWitnesses`
**Rule:** UTXOW
**Feature:** Script Witnesses - Malformed Witness Validation
**Expected:** FAILURE (final transaction)

### Purpose

This test validates that the ledger correctly rejects transactions that include malformed Plutus scripts in their witness set when attempting to spend script-locked UTxOs.

### State Change Diagram

```
Initial State:
+-- Epoch: 899
+-- UTxOs: (genesis funding available)
    +-- 03170a2e...#0: Available for spending

Event 1: PassTick (Slot 1)
+-- Advances the slot counter

Event 2: Transaction 1 at Slot 3,883,681 [SUCCESS]
+-- TX Hash: 9a655a93b0c241faf310db2f722ac69ee27d60228574a99bf456f3a40a8027c1
+-- Creates script-locked UTxO with datum hash
+-- Inputs:
|   +-- 03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314#0
|
+-- Outputs:
|   +-- Output 0: 995,610 lovelace (SCRIPT-LOCKED)
|   |   +-- Address: addr_test1wr9pe43fp46jdcpc9c9cv66kk7u4meva58f377a33xtwusgeugch4
|   |   +-- Datum Hash: 873e4fe9e41e924911bba3ec53ff4782efc8c0f244fb75c879f8a4328d0142ca
|   |
|   +-- Output 1: 44,999,999,998,835,853 lovelace (Change)
|
+-- Result: SUCCESS

Intermediate State After Event 2:
+-- UTxOs:
    +-- 9a655a93...#0: 995,610 lovelace (script-locked with datum hash)
    +-- 9a655a93...#1: 44,999,999,998,835,853 lovelace (change)

Event 3: Transaction 2 at Slot 3,883,681 [SUCCESS]
+-- TX Hash: a038988867beba9f4c360d3c464a7d97e6cdfe8c42c3334b26886208cd6daa60
+-- Spends change output to create additional funding
+-- Inputs:
|   +-- 9a655a93b0c241faf310db2f722ac69ee27d60228574a99bf456f3a40a8027c1#1
|
+-- Outputs:
|   +-- Output 0: 10,000,000 lovelace
|   +-- Output 1: 44,999,999,988,668,812 lovelace (Change)
|
+-- Result: SUCCESS

Intermediate State After Event 3:
+-- UTxOs:
    +-- 9a655a93...#0: 995,610 lovelace (script-locked, still unspent)
    +-- a0389888...#0: 10,000,000 lovelace
    +-- a0389888...#1: 44,999,999,988,668,812 lovelace

Event 4: Transaction 3 at Slot 3,883,681 [FAILURE]
+-- TX Hash: 550e949178c0bed318923ee01b2f91f766428ff14270139ea2d9784db1f50fd9
+-- Attempts to spend script-locked UTxO with MALFORMED script witness
+-- Inputs:
|   +-- 9a655a93b0c241faf310db2f722ac69ee27d60228574a99bf456f3a40a8027c1#0 (script-locked)
|   +-- a038988867beba9f4c360d3c464a7d97e6cdfe8c42c3334b26886208cd6daa60#1
|
+-- Witnesses:
|   +-- PlutusV2 Script: ca1cd6290d7526e0382e0b866b56b7b95de59da1d31f7bb18996ee41
|   |   [MALFORMED - Invalid script bytes]
|   +-- Datum: 873e4fe9e41e924911bba3ec53ff4782efc8c0f244fb75c879f8a4328d0142ca
|   +-- Redeemer: Tag=0 (Spend), Index=0
|
+-- Outputs:
|   +-- Output 0: 44,999,999,988,189,833 lovelace
|
+-- Result: FAILURE - Malformed script in witness set

Final State:
+-- UTxOs: (same as after Event 3 - final transaction rejected)
    +-- 9a655a93...#0: 995,610 lovelace (script-locked, still unspent)
    +-- a0389888...#0: 10,000,000 lovelace
    +-- a0389888...#1: 44,999,999,988,668,812 lovelace
```

### What This Tests

This test validates the multi-step process of:

1. **Creating a script-locked UTxO** (TX 1): A valid transaction creates an output at a script address with a datum hash, preparing it for later script-based spending.

2. **Preparing additional funds** (TX 2): Another transaction moves funds to provide collateral and fees.

3. **Attempting script validation with malformed witness** (TX 3): The final transaction tries to spend the script-locked UTxO but provides a malformed PlutusV2 script in the witness set.

The key validation being tested is that the ledger:
- Decodes script witnesses before execution
- Rejects transactions with malformed scripts (even if the datum and redeemer are correct)
- Preserves the UTxO state when validation fails

### Witnesses Analysis

The failing transaction includes:
- **Script**: Hash `ca1cd6290d7526e0382e0b866b56b7b95de59da1d31f7bb18996ee41` - This is a malformed PlutusV2 script
- **Datum**: Hash `873e4fe9e41e924911bba3ec53ff4782efc8c0f244fb75c879f8a4328d0142ca` - Provided correctly
- **Redeemer**: Tag 0 (Spend), Index 0 - Points to the first input correctly

### Validation Rule

- **MalformedScripts (MALFORMED_SCRIPT_WITNESS)**: The script witness cannot be decoded as valid Plutus bytecode.

---

## Test: ExtraRedeemers-RedeemerPointerPointsToNothing

**File:** `UTXOW/ExtraRedeemers-RedeemerPointerPointsToNothing`
**Rule:** UTXOW
**Feature:** Redeemers - Extra Redeemer Validation
**Expected:** FAILURE (transactions 3 and 6)

### Purpose

This test validates that the ledger correctly rejects transactions that include redeemers pointing to non-existent script purposes. Each redeemer must correspond to a valid script execution context (spend, mint, certificate, or reward withdrawal).

### State Change Diagram

```
Initial State:
+-- Epoch: 899
+-- UTxOs: (genesis funding available)

Event 1: PassTick (Slot 1)
+-- Advances the slot counter

Event 2: Transaction 1 at Slot 3,883,681 [SUCCESS]
+-- TX Hash: 8046318a54dd556bfa6192d65a192e14345130a455b82fe9672df50c3f0f3a03
+-- Creates first script-locked UTxO
+-- Inputs:
|   +-- 03170a2e...#0 (genesis)
|
+-- Outputs:
|   +-- Output 0: 995,610 lovelace (SCRIPT-LOCKED)
|   |   +-- Address: addr_test1wzm6ar5z0eljf54wzgwfrhl0ndfttkqy5278qutplyul9vc0kps6e
|   |   +-- Datum Hash: e88bd757ad5b9bedf372d8d3f0cf6c962a469db61a265f6418e1ffed86da29ec
|   |
|   +-- Output 1: Change output
|
+-- Result: SUCCESS

Event 3: Transaction 2 at Slot 3,883,681 [SUCCESS]
+-- TX Hash: 9eb53137aea5224cead23e0486c6db298610c7125c231d252e607ae6261a361f
+-- Moves funds for later use
+-- Result: SUCCESS

Intermediate State:
+-- UTxOs:
    +-- 8046318a...#0: Script-locked (PlutusV2)
    +-- 9eb53137...#0: 10,000,000 lovelace
    +-- 9eb53137...#1: 44,999,999,988,668,812 lovelace

Event 4: Transaction 3 at Slot 3,883,681 [FAILURE]
+-- TX Hash: 7015ec6965e48dbebcc95c288fead91ad0cb0555b8aa78b2723e23511fd9ee82
+-- Attempts to spend script UTxO with EXTRA REDEEMER
+-- Inputs:
|   +-- 8046318a54dd556bfa6192d65a192e14345130a455b82fe9672df50c3f0f3a03#0 (script-locked)
|   +-- 9eb53137aea5224cead23e0486c6db298610c7125c231d252e607ae6261a361f#1
|
+-- Witnesses:
|   +-- PlutusV2 Script: b7ae8e827e7f24d2ae121c91dfef9b52b5d804a2bc707161f939f2b3
|   +-- Datum: e88bd757ad5b9bedf372d8d3f0cf6c962a469db61a265f6418e1ffed86da29ec
|   +-- Redeemers:
|       +-- Tag: 0 (Spend), Index: 0   [VALID - for script input at index 0]
|       +-- Tag: 1 (Mint), Index: 2    [INVALID - no minting policy at index 2]
|
+-- Result: FAILURE - Extra redeemer points to nothing

Event 5: Transaction 4 at Slot 3,883,681 [SUCCESS]
+-- TX Hash: 40b485aaabb4d979d2f3e77fd7cc904142495af5e4bff26c76884aae8ceeba8a
+-- Creates second script-locked UTxO (PlutusV3)
+-- Result: SUCCESS

Event 6: Transaction 5 at Slot 3,883,681 [SUCCESS]
+-- TX Hash: df28b03418edc612a0043710428db272c2f53eb300ae167d460a46e58fafb1ac
+-- Moves funds
+-- Result: SUCCESS

Event 7: Transaction 6 at Slot 3,883,681 [FAILURE]
+-- TX Hash: a1cac00c6853c8d1c432d1529775027db27a88a7c68ab778406c3782d5bb379c
+-- Attempts to spend script UTxO with EXTRA REDEEMER (PlutusV3)
+-- Inputs:
|   +-- 40b485aaabb4d979d2f3e77fd7cc904142495af5e4bff26c76884aae8ceeba8a#0 (script-locked)
|   +-- df28b03418edc612a0043710428db272c2f53eb300ae167d460a46e58fafb1ac#1
|
+-- Witnesses:
|   +-- PlutusV3 Script: 7c65f4f9310aef567d052ab87d2f8d030598d285dd51ceaafafb0f37
|   +-- Datum: e88bd757ad5b9bedf372d8d3f0cf6c962a469db61a265f6418e1ffed86da29ec
|   +-- Redeemers:
|       +-- Tag: 0 (Spend), Index: 0   [VALID - for script input at index 0]
|       +-- Tag: 1 (Mint), Index: 2    [INVALID - no minting policy at index 2]
|
+-- Result: FAILURE - Extra redeemer points to nothing

Final State:
+-- UTxOs: Script-locked UTxOs remain unspent due to validation failures
```

### What This Tests

This test validates the **ExtraRedeemers** validation rule, which requires that every redeemer in a transaction must correspond to a valid script execution purpose. The test demonstrates this with both PlutusV2 (Event 4) and PlutusV3 (Event 7) scripts.

The redeemer tag types are:
- **Tag 0 (Spend)**: For spending script-locked UTxOs
- **Tag 1 (Mint)**: For executing minting policies
- **Tag 2 (Cert)**: For certificate validation
- **Tag 3 (Reward)**: For reward withdrawal

In both failing transactions:
- The first redeemer (Tag 0, Index 0) correctly points to the script input
- The second redeemer (Tag 1, Index 2) claims to be for a minting policy at index 2, but:
  - The transaction has no minting operations
  - There is no minting policy at any index
  - This redeemer "points to nothing"

### Redeemer Pointer Analysis

```
Transaction 3 Redeemers:
+-- Redeemer 1: Tag=0 (Spend), Index=0
|   +-- Purpose: Spend input at sorted index 0
|   +-- Input at index 0: 8046318a...#0 (script-locked)
|   +-- Status: VALID
|
+-- Redeemer 2: Tag=1 (Mint), Index=2
    +-- Purpose: Execute minting policy at index 2
    +-- Minting policies in tx: NONE
    +-- Status: INVALID (points to nothing)

Transaction 6 Redeemers:
+-- Same pattern with PlutusV3 script
```

### Validation Rule

- **ExtraRedeemers**: Redeemers are present for which there is no corresponding script purpose (no script-locked input to spend, no minting policy to execute, no certificate to validate, no reward withdrawal requiring script authorization).

---

## Summary

| Test | Feature | Scripts | Expected | Validation Rule |
|------|---------|---------|----------|-----------------|
| MalformedReferenceScripts | Reference Scripts | N/A (output contains malformed ref script) | FAILURE | MalformedScripts |
| MalformedScriptWitnesses | Script Witnesses | PlutusV2 (malformed) | FAILURE | MalformedScripts |
| ExtraRedeemers | Redeemer Validation | PlutusV2, PlutusV3 | FAILURE | ExtraRedeemers |

## Babbage Features Demonstrated

1. **Reference Scripts**: The MalformedReferenceScripts test shows scripts being embedded in UTxO outputs
2. **Datum Hashes**: All tests use datum hashes to lock UTxOs to scripts
3. **Redeemer Tags**: The ExtraRedeemers test demonstrates all redeemer tag types
4. **Script Witnesses**: MalformedScriptWitnesses shows script provision in witness sets

## Related UTXOW Rules

The UTXOW rule (UTxO with witnesses) validates:
- `ScriptsNotPaidUTxO` - All scripts executed are paid for
- `ExtraneousScriptWitnesses` - No unused scripts in witness set
- `MissingScriptWitnesses` - All required scripts present
- `MalformedScripts` - All scripts are well-formed
- `MissingRedeemers` - All script executions have redeemers
- `ExtraRedeemers` - No redeemers without corresponding scripts
- `MissingDatum` - All script inputs have datums available
- `UnspendableUTxO` - Scripts can actually be executed

