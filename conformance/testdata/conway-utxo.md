
# Conway UTXO Conformance Test Vectors

This directory contains conformance test vectors for the Conway-era UTXO transition rule. These tests validate fee calculation rules, particularly for reference scripts in the Conway ledger.

## Overview

| Category | Test Count | Focus Area |
|----------|------------|------------|
| Reference Scripts | 3 | MinFee calculation with reference scripts |

## Test Summary

### Reference Scripts Tests

These tests validate the UTXO transition rule requirements around reference scripts and their impact on minimum fee calculations in Conway.

---

## Test: Required Reference Script Counts Towards MinFee Calculation

**File:** `Reference_scripts/required_reference_script_counts_towards_the_minFee_calculation`
**Rule:** UTXO
**Expected:** Success

### State Change Diagram

```
Initial State:
|-- UTxOs: Empty (genesis funding only)
|-- Reference Scripts: None
|-- Cost Models: Available for validation

Event 1: Transaction at slot 3883681
|-- Inputs: 1 UTxO (funding)
|-- Outputs: 2 UTxOs (setup outputs)
|-- Script Execution: None
|__ Tx Result: Success

Event 2: Transaction at slot 3883681
|-- Inputs: 1 UTxO
|-- Outputs: 2 UTxOs (including reference script UTxO)
|-- Script Execution: None
|__ Tx Result: Success

Event 3: Transaction at slot 3883681
|-- Inputs: 2 UTxOs (including script-locked input)
|-- Reference Inputs: 1 UTxO (containing reference script)
|-- Outputs: 1 UTxO
|-- Script Execution:
|   |-- Reference Script Used: Yes
|   |__ Script contributes to minFee calculation
|__ Tx Result: Success

Final State:
|-- UTxOs: Updated with new outputs
|__ Reference script size included in minFee
```

### What This Tests

This test validates that when a reference script is **required** to unlock an input, its size is properly accounted for in the minimum fee calculation. The UTXO rule must:
1. Identify which reference inputs contain scripts needed for validation
2. Include those script sizes in the fee calculation
3. Ensure the transaction provides sufficient fees

---

## Test: Reference Scripts Not Required for Spending Count Towards MinFee

**File:** `Reference_scripts/reference_scripts_not_required_for_spending_the_input_count_towards_the_minFee_calculation`
**Rule:** UTXO
**Expected:** Success

### State Change Diagram

```
Initial State:
|-- UTxOs: Empty (genesis funding only)
|-- Reference Scripts: None
|-- Cost Models: Available for validation

Event 1: Transaction at slot 3883681
|-- Inputs: 1 UTxO (funding)
|-- Outputs: 2 UTxOs
|-- Script Execution: None
|__ Tx Result: Success

Event 2: Transaction at slot 3883681
|-- Inputs: 1 UTxO
|-- Outputs: 7 UTxOs (including multiple reference script UTxOs)
|-- Script Execution: None
|__ Tx Result: Success

Event 3: Transaction at slot 3883681
|-- Inputs: 2 UTxOs
|-- Reference Inputs: 6 UTxOs (containing reference scripts)
|-- Outputs: 1 UTxO
|-- Script Execution:
|   |-- Reference Scripts Used: Yes (6 scripts)
|   |-- Scripts NOT required for input spending
|   |__ All script sizes still contribute to minFee
|__ Tx Result: Success

Final State:
|-- UTxOs: Updated with new outputs
|__ All 6 reference script sizes included in minFee
```

### What This Tests

This test validates that reference scripts count towards the minimum fee calculation **even when they are not required** for spending any inputs. This is important for:
1. Preventing fee manipulation by including arbitrary reference inputs
2. Ensuring consistent fee accounting regardless of script usage
3. Properly charging for blockchain data access

The Conway UTXO rule must count ALL referenced scripts in the fee calculation, not just those used for validation.

---

## Test: Script Referenced Several Times Counts for Each Reference

**File:** `Reference_scripts/a_scripts_referenced_several_times_counts_for_each_reference_towards_the_minFee_calculation`
**Rule:** UTXO
**Expected:** Success

### State Change Diagram

```
Initial State:
|-- UTxOs: Empty (genesis funding only)
|-- Reference Scripts: None
|-- Cost Models: Available for validation

Event 1: Transaction at slot 3883681
|-- Inputs: 1 UTxO (funding)
|-- Outputs: 2 UTxOs
|-- Script Execution: None
|__ Tx Result: Success

Event 2: Transaction at slot 3883681
|-- Inputs: 1 UTxO
|-- Outputs: 13 UTxOs (including many reference script UTxOs)
|-- Script Execution: None
|__ Tx Result: Success

Event 3: Transaction at slot 3883681
|-- Inputs: 2 UTxOs
|-- Reference Inputs: 12 UTxOs (many containing the SAME script)
|-- Outputs: 1 UTxO
|-- Script Execution:
|   |-- Reference Scripts: 12 references to scripts
|   |-- Same script referenced multiple times
|   |__ Each reference counts separately in minFee
|__ Tx Result: Success

Final State:
|-- UTxOs: Updated with new outputs
|__ Script size counted 12 times (once per reference) in minFee
```

### What This Tests

This test validates that when the **same script** is referenced multiple times through different reference inputs, it counts **for each reference** towards the minimum fee calculation. This is critical for:
1. Preventing fee evasion by reusing script references
2. Properly accounting for data access costs
3. Ensuring predictable fee calculation behavior

The rule ensures that fee calculation is based on total data accessed, not unique scripts.

---

## Conway UTXO Fee Calculation Rules

The Conway-era UTXO transition introduces enhanced fee calculation rules for reference scripts:

### Key Rules Tested

1. **Reference Script Inclusion**: All scripts in reference inputs contribute to the minimum fee, regardless of whether they are used for input validation.

2. **Multiple Reference Counting**: The same script referenced through multiple UTxOs counts separately for each reference.

3. **Consistent Fee Model**: The fee calculation treats reference script data access uniformly, preventing manipulation.

### Fee Formula Impact

The reference script component of the fee is calculated as:
```
refScriptFee = sum(scriptSize for each reference input with script)
             * refScriptCostPerByte
```

This ensures that blockchain data access is properly charged.

## Test Vector Format

Each test vector is a CBOR-encoded file containing:
- `[0]` config: Network/protocol configuration
- `[1]` initial_state: NewEpochState before events
- `[2]` final_state: NewEpochState after events
- `[3]` events: Array of transaction events
- `[4]` title: Test name/path

Each transaction event contains:
- Transaction CBOR bytes
- Success flag (true/false)
- Slot number

## Related Specifications

- CIP-69: Reference Scripts
- Conway Ledger Specification: UTXO transition rule
- Cardano Fee Calculation: Reference script fee component

