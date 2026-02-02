
# Alonzo UTXOS Conformance Test Vectors

This directory contains conformance test vectors for the Alonzo UTXOS transition rules.
These tests validate **phase-2 validation** - the actual execution of Plutus scripts.

## Phase-2 Validation Overview

Phase-2 validation occurs after phase-1 checks pass. It involves:
- **Script execution**: Running Plutus scripts with their datums and redeemers
- **Cost model application**: Calculating execution costs using language-specific cost models
- **Collateral handling**: Consuming collateral if scripts fail
- **isValid flag verification**: Ensuring declared validity matches actual script results

## Test Summary

Total test vectors: 33 (11 per Plutus version)

| Test Category | Count per Version | Description |
|---------------|-------------------|-------------|
| Invalid script fails | 1 | Script returns False |
| Invalid tx marked valid | 1 | isValid=true but script fails |
| No cost model | 1 | Missing cost model for language |
| Scripts pass | 3 | Successful script execution |
| Spending with datum | 4 | Datum validation variants |
| Valid tx marked invalid | 1 | isValid=false with passing script |

---

## Test Categories Explained

### 1. Script Failure Tests

#### Invalid plutus script fails in phase 2

**Files:** `PlutusV{1,2,3}/Invalid_plutus_script_fails_in_phase_2`
**Expected Outcome:** Transaction applies with collateral consumed

```
Test Structure:
Event 1: PassTick (advance slot)
Event 2: Setup transaction (SUCCESS) - Creates funding UTxO
Event 3: Setup transaction (SUCCESS) - Creates script-locked UTxO + collateral
Event 4: Test transaction (SUCCESS*) - Script fails, collateral consumed

*SUCCESS means the transaction was processed, not that the script passed
```

#### State Change Diagram

```
Initial State:
└── Empty UTxO set

After Setup:
├── UTxO A: Regular funding output
├── UTxO B: Script-locked output (with always-fails script)
├── UTxO C: Collateral output
└── UTxO D: Change output

Test Transaction (with failing script):
├── Inputs: [UTxO B (script-locked)]
├── Collateral: [UTxO C]
├── Script Execution:
│   ├── Script: AlwaysFails (returns False)
│   ├── Result: SCRIPT_FAILURE
│   └── Collateral: CONSUMED
└── Tx Result: Phase-2 failure, collateral consumed

Final State:
├── UTxO A: Unchanged
├── UTxO B: Unchanged (script input NOT consumed)
├── UTxO C: REMOVED (collateral consumed)
└── UTxO D: Unchanged
└── New: Collateral return output (if any excess)
```

#### What This Tests

When a Plutus script evaluates to `False` (or throws an error), the transaction fails
in phase-2. However, because computational work was done, the collateral is consumed
to compensate block producers. The regular inputs are NOT consumed.

**Key Points:**
- Collateral is consumed on script failure
- Regular inputs remain in the UTxO set
- If `collateralReturn` output is specified, excess collateral is returned
- The transaction is "successful" from the ledger's perspective (state was updated)

---

### 2. isValid Flag Tests

#### Invalid transaction marked as valid

**Files:** `PlutusV{1,2,3}/Invalid_transaction_marked_as_valid`
**Expected Outcome:** Transaction FAILS (rejected)

```
Test Transaction:
├── isValid flag: true
├── Script would return: False
├── Node must execute scripts to verify
└── Result: REJECTED - ValidationTagMismatch
```

#### What This Tests

The `isValid` flag in the transaction must accurately reflect script execution results.
If `isValid = true` but scripts actually fail, the transaction is rejected entirely.
This prevents attackers from claiming transactions are valid to skip validation.

**Validation Rule:**
```
if tx.isValid == true:
    all scripts must pass
    collateral is NOT consumed
```

---

#### Valid transaction marked as invalid

**Files:** `PlutusV{1,2,3}/Valid_transaction_marked_as_invalid`
**Expected Outcome:** Transaction FAILS (rejected)

```
Test Transaction:
├── isValid flag: false
├── Script would return: True (if executed)
├── Scripts are NOT executed (isValid=false skips execution)
├── Collateral would be consumed
└── Result: REJECTED - ValidationTagMismatch*

*Note: Some ledger implementations may handle this differently
```

#### What This Tests

A transaction marked as invalid (`isValid = false`) claims its scripts will fail.
In Conway, if the scripts would actually pass, this is a protocol violation.
This prevents intentional collateral burning when scripts would succeed.

**Important:** This behavior changed across eras. In earlier eras, `isValid = false`
was a legitimate way to intentionally consume collateral. Conway enforces consistency.

---

### 3. Cost Model Tests

#### No cost model

**Files:** `PlutusV{1,2,3}/No_cost_model`
**Expected Outcome:** Transaction FAILS

```
Initial State:
├── Protocol parameters WITHOUT cost model for target Plutus version
└── Script-locked UTxO using that Plutus version

Test Transaction:
├── Attempts to execute PlutusVX script
├── No cost model available for PlutusVX
└── Result: REJECTED - NoCostModel / CollectErrors
```

#### What This Tests

Plutus scripts require a **cost model** to calculate execution costs. The cost model
is language-version specific (PlutusV1, V2, V3 each have different cost models).
If a transaction tries to execute a script but no cost model exists for that version,
the transaction fails.

**Cost Model Contents:**
- ~150-200 cost parameters for PlutusV1/V2
- ~250+ cost parameters for PlutusV3
- Parameters for each Plutus Core builtin function
- Memory and CPU cost coefficients

**When Cost Models Are Missing:**
- New Plutus versions before cost model governance proposal passes
- Network not yet upgraded to support a Plutus version
- Test environments without full protocol parameter sets

---

### 4. Successful Script Execution Tests

#### Scripts pass in phase 2

**Files:** `PlutusV{1,2,3}/Scripts_pass_in_phase_2/{variant}`

Three variants per Plutus version:
- `datumIsWellformed` - Tests datum CBOR validity
- `inputsOutputsAreNotEmptyWithDatum` - Tests ScriptContext construction
- `purposeIsWellformedWithDatum` - Tests ScriptPurpose encoding

```
Test Transaction (success case):
├── Inputs: Script-locked UTxO
├── Datum: Well-formed, matches script expectations
├── Redeemer: Valid data for script
├── Script Execution:
│   ├── Script: Validator that checks specific property
│   ├── Arguments: (datum, redeemer, scriptContext)
│   └── Result: PASS
└── Tx Result: SUCCESS - inputs consumed, outputs created
```

#### What This Tests

These tests verify that valid transactions with passing scripts are processed correctly:
- Script-locked UTxOs can be spent when validation passes
- Datums, redeemers, and script context are correctly assembled
- Execution costs are calculated and charged
- New UTxOs are created from transaction outputs

---

### 5. Spending Scripts with Datum Tests

#### Spending scripts with a Datum

**Files:** `PlutusV{1,2,3}/Spending_scripts_with_a_Datum/{variant}`

Four variants per Plutus version:
- `datumIsWellformed` - Datum CBOR decoding works
- `inputsOutputsAreNotEmptyWithDatum` - Transaction has inputs/outputs
- `purposeIsWellformedWithDatum` - Spending purpose correctly identifies UTxO
- `redeemerSameAsDatum` - Common pattern: redeemer must match stored datum

#### State Change Diagram (redeemerSameAsDatum variant)

```
Initial State:
├── UTxO at script address
│   ├── Value: 10 ADA
│   ├── Datum: ConstrData(0, [IntegerData(42)])
│   └── Script requires: redeemer == datum

Test Transaction:
├── Input: Script-locked UTxO
├── Redeemer: ConstrData(0, [IntegerData(42)])  // Matches datum!
├── Script Logic:
│   │   validateSpend datum redeemer ctx =
│   │       datum == redeemer  // Must match
│   ├── Result: True (datum matches redeemer)
└── Tx Result: SUCCESS

Final State:
├── Original UTxO: CONSUMED
└── New outputs as specified in transaction
```

#### What This Tests

Spending validators receive three arguments:
1. **Datum**: Data stored with the UTxO (set when UTxO was created)
2. **Redeemer**: Data provided by spender (in transaction witness)
3. **ScriptContext**: Transaction information (inputs, outputs, signatories, etc.)

The `redeemerSameAsDatum` test uses a validator that requires the redeemer to exactly
match the datum. This is a common pattern for "unlock with password" scripts.

---

## Implementation Notes

### Script Context Construction

For each Plutus script execution, the ledger constructs a `ScriptContext`:

```haskell
data ScriptContext = ScriptContext
    { scriptContextTxInfo :: TxInfo
    , scriptContextPurpose :: ScriptPurpose
    }

data TxInfo = TxInfo
    { txInfoInputs :: [TxInInfo]
    , txInfoOutputs :: [TxOut]
    , txInfoFee :: Value
    , txInfoMint :: Value
    , txInfoDCert :: [DCert]
    , txInfoWdrl :: Map StakingCredential Integer
    , txInfoValidRange :: POSIXTimeRange
    , txInfoSignatories :: [PubKeyHash]
    , txInfoData :: Map DatumHash Datum
    , txInfoId :: TxId
    }
```

### Datum Resolution

Datums can be provided in multiple ways:
1. **Inline datum**: Stored directly in the UTxO (Babbage+)
2. **Datum hash**: Hash stored in UTxO, datum in transaction witness
3. **Reference script**: Datum from a reference input (Babbage+)

The ledger must resolve all required datums before script execution.

### Execution Cost Calculation

For each script execution:
```
executionCost = evaluateScript(script, arguments, costModel)
totalCost = sum(executionCosts)
fee >= minFee + scriptCost(totalCost, prices)
```

Where `prices` contains the cost per memory unit and cost per CPU step.

### Collateral Mechanics

On phase-2 failure:
1. All collateral inputs are consumed
2. If `collateralReturn` output exists, excess value is returned
3. Total collateral consumed = sum(collateralInputs) - collateralReturn
4. Consumed amount must be >= required collateral

---

## Plutus Version Differences

| Feature | PlutusV1 | PlutusV2 | PlutusV3 |
|---------|----------|----------|----------|
| Inline datums | No | Yes | Yes |
| Reference inputs | No | Yes | Yes |
| Reference scripts | No | Yes | Yes |
| Cost model size | ~166 params | ~175 params | ~251 params |
| New builtins | - | serialiseData, etc. | BLS, bitwise, etc. |

These tests validate that each Plutus version behaves correctly according to its
specification, with the appropriate cost models and features.

