
# Alonzo UTXO Conformance Test Vectors

This directory contains conformance test vectors for the Alonzo UTXO transition rules.
These tests validate **phase-1 validation** rules that are checked before any script execution occurs.

## Phase-1 Validation Overview

Phase-1 validation performs deterministic checks that do not require script execution:
- **Collateral adequacy**: Sufficient collateral to cover potential script failure costs
- **Execution unit limits**: Total declared execution units within protocol limits
- **Network ID validation**: Transaction targets the correct network (mainnet/testnet)
- **Input/output balance**: Transaction is well-formed structurally

## Test Summary

Total test vectors: 7

| Test | Plutus Version | Expected Outcome | Rule Tested |
|------|----------------|------------------|-------------|
| Insufficient collateral | PlutusV1 | FAIL | Collateral adequacy |
| Too many execution units | PlutusV1 | FAIL | ExUnits limit |
| Insufficient collateral | PlutusV2 | FAIL | Collateral adequacy |
| Too many execution units | PlutusV2 | FAIL | ExUnits limit |
| Insufficient collateral | PlutusV3 | FAIL | Collateral adequacy |
| Too many execution units | PlutusV3 | FAIL | ExUnits limit |
| Wrong network ID | N/A | FAIL | Network ID |

---

## PlutusV1 Tests

### Test: Insufficient collateral (PlutusV1)

**File:** `PlutusV1/Insufficient_collateral`
**Rule:** UTXO collateral adequacy (phase-1)
**Expected Outcome:** Transaction FAILS

#### Test Structure

```
Event 1: PassTick (advance slot)

Event 2: Setup transaction 1 (SUCCESS)
├── Creates funding UTxOs for the test

Event 3: Setup transaction 2 (SUCCESS)
├── Creates script-locked UTxO
├── Creates collateral UTxO (insufficient amount)

Event 4: Test transaction (FAILS)
├── Attempts to spend script-locked UTxO
├── Provides insufficient collateral
└── Result: REJECTED in phase-1 validation
```

#### State Change Diagram

```
Initial State:
└── Empty UTxO set

After Setup:
├── UTxO A: Funding output
├── UTxO B: Script-locked output (requires Plutus execution)
└── UTxO C: Collateral output (< required collateral amount)

Test Transaction Attempts:
├── Input: UTxO B (script-locked)
├── Collateral: UTxO C (insufficient!)
├── Phase-1 Check: collateral_value < (fee * collateralPercentage / 100)
└── Result: REJECTED - CollateralNotEnough

Final State:
└── Unchanged (transaction rejected)
```

#### What This Tests

The **collateral adequacy rule** requires that transactions with Plutus scripts must provide
collateral worth at least `collateralPercentage` (typically 150%) of the transaction fee.
This protects the network from denial-of-service attacks where malicious actors submit
transactions with scripts that fail after consuming computation resources.

**Protocol Parameter:** `collateralPercentage` (default: 150)

**Validation Formula:**
```
totalCollateral >= (txFee * collateralPercentage) / 100
```

---

### Test: Too many execution units for tx (PlutusV1)

**File:** `PlutusV1/Too_many_execution_units_for_tx`
**Rule:** UTXO execution unit limits (phase-1)
**Expected Outcome:** Transaction FAILS

#### Test Structure

```
Event 1: PassTick (advance slot)

Event 2: Setup transaction 1 (SUCCESS)
├── Creates funding UTxOs

Event 3: Setup transaction 2 (SUCCESS)
├── Creates script-locked UTxO

Event 4: Test transaction (FAILS)
├── Declares excessive execution units in redeemer
└── Result: REJECTED in phase-1 validation
```

#### What This Tests

The **execution unit limits rule** ensures transactions cannot declare more computational
resources than the protocol allows. This is checked in phase-1 before any scripts execute.

**Protocol Parameters:**
- `maxTxExUnits.mem`: Maximum memory units per transaction
- `maxTxExUnits.steps`: Maximum CPU steps per transaction

**Validation:**
```
sum(redeemer.exUnits.mem) <= maxTxExUnits.mem
sum(redeemer.exUnits.steps) <= maxTxExUnits.steps
```

---

## PlutusV2 Tests

### Test: Insufficient collateral (PlutusV2)

**File:** `PlutusV2/Insufficient_collateral`
**Rule:** UTXO collateral adequacy (phase-1)
**Expected Outcome:** Transaction FAILS

Same rule as PlutusV1 test, but uses PlutusV2 scripts. The collateral requirements
are identical across Plutus versions.

---

### Test: Too many execution units for tx (PlutusV2)

**File:** `PlutusV2/Too_many_execution_units_for_tx`
**Rule:** UTXO execution unit limits (phase-1)
**Expected Outcome:** Transaction FAILS

Same rule as PlutusV1 test, but uses PlutusV2 scripts.

---

## PlutusV3 Tests

### Test: Insufficient collateral (PlutusV3)

**File:** `PlutusV3/Insufficient_collateral`
**Rule:** UTXO collateral adequacy (phase-1)
**Expected Outcome:** Transaction FAILS

Same rule as PlutusV1/V2 tests, but uses PlutusV3 scripts.

---

### Test: Too many execution units for tx (PlutusV3)

**File:** `PlutusV3/Too_many_execution_units_for_tx`
**Rule:** UTXO execution unit limits (phase-1)
**Expected Outcome:** Transaction FAILS

Same rule as PlutusV1/V2 tests, but uses PlutusV3 scripts.

---

## Non-Plutus Tests

### Test: Wrong network ID

**File:** `Wrong_network_ID`
**Rule:** UTXO network ID validation (phase-1)
**Expected Outcome:** Transaction FAILS

#### Test Structure

```
Event 1: PassTick (advance slot)

Event 2: Test transaction (FAILS)
├── Transaction body contains wrong network ID
└── Result: REJECTED in phase-1 validation
```

#### What This Tests

The **network ID rule** prevents transactions from being accidentally submitted to the
wrong network. Transactions can optionally specify a network ID, and if present, it must
match the ledger's network configuration.

**Validation:**
```
if tx.networkId is present:
    tx.networkId == ledger.networkId
```

**Network IDs:**
- Mainnet: 1
- Testnet: 0

---

## Implementation Notes

### Collateral Calculation

The collateral percentage is applied to the **transaction fee**, not the total value
being transacted. The formula used by the ledger is:

```haskell
requiredCollateral = (txFee * collateralPercentage + 99) `div` 100
```

Note: The `+ 99` ensures rounding up, so the collateral requirement is always sufficient.

### ExUnits Calculation

Execution units are declared per-redeemer in the transaction witness set. The ledger
sums all declared units and compares against protocol limits:

```haskell
totalMem = sum [r.exUnits.mem | r <- redeemers]
totalSteps = sum [r.exUnits.steps | r <- redeemers]
```

Both `totalMem` and `totalSteps` must be within their respective limits.

### Phase-1 vs Phase-2

These UTXO tests specifically exercise **phase-1** validation, which occurs before
Plutus scripts are evaluated. Phase-1 checks are:

1. Deterministic (same inputs and ledger state produce the same result)
2. Computationally cheap (no script execution)
3. Protective (prevent resource exhaustion before expensive computation)

See the UTXOS directory for phase-2 (script execution) tests.

