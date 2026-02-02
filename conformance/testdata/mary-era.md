
# MaryImpSpec Conformance Test Vectors

This document describes the Mary-era conformance test vectors.
Mary introduced multi-asset (native token) support to Cardano.

## Overview

Total test vectors: 2

### Vectors by Ledger Rule

| Rule | Count | Description |
|------|-------|-------------|
| UTXO | 2 | UTxO validation rules - multi-asset value conservation and minting |

---

## Test: Conway/Imp/MaryImpSpec/UTXO/Mint a Token

**File:** `MaryImpSpec/UTXO/Mint_a_Token`
**Rule:** UTXO
**Expected:** Success

### Initial State

```
Epoch: 899
UTxOs: 0 entries
```

### Final State

```
Epoch: 899
UTxOs: 0 entries
```

### State Change Diagram

```
Event 1: Pass Tick to slot 1

Event 2: Transaction at slot 3883681
  Result: SUCCESS
  Inputs: 1
    - 03170a2e...4c111314#0
  Outputs: 2
    - Output 0: 1038710 lovelace
      + ec93b0de2e26bb59.testAsset: 1
    - Output 1: 44999999998784305 lovelace
  Minting:
    ec93b0de2e26bb59.testAsset: 1
  Fee: 176985 lovelace
  VKey Witnesses: 2
  Native Scripts: 1

```

### What This Tests

Tests the Mary-era multi-asset minting functionality:

1. **Token Minting**: A new native token called "testAsset" is minted with quantity 1
2. **Policy Script**: The minting is authorized by a native script (minting policy)
3. **Value Conservation**: Input value + minted tokens = Output value + fee
4. **Multi-Asset Output**: The output contains both ADA and the newly minted token

The Mary UTXO rule extends Shelley's value conservation to include:
- sum(inputs) + mint = sum(outputs) + fee
- Where mint can be positive (minting) or negative (burning)

---

## Test: Conway/Imp/MaryImpSpec/UTXO/ShelleyUtxoPredFailure/ValueNotConservedUTxO

**File:** `MaryImpSpec/UTXO/ShelleyUtxoPredFailure/ValueNotConservedUTxO`
**Rule:** UTXO
**Expected:** Failure

### Initial State

```
Epoch: 899
UTxOs: 0 entries
```

### Final State

```
Epoch: 899
UTxOs: 0 entries
```

### State Change Diagram

```
Event 1: Pass Tick to slot 1

Event 2: Transaction at slot 3883681
  Result: SUCCESS
  Inputs: 1
    - 03170a2e...4c111314#0
  Outputs: 2
    - Output 0: 1038710 lovelace
      + 82b474fbbd69ac8f.testAsset: 1
    - Output 1: 44999999998784305 lovelace
  Minting:
    82b474fbbd69ac8f.testAsset: 1
  Fee: 176985 lovelace
  VKey Witnesses: 2
  Native Scripts: 1

Event 3: Transaction at slot 3883681
  Result: FAILURE (expected)
  Inputs: 2
    - 6c1a1296...4ff790be#0
    - 6c1a1296...4ff790be#1
  Outputs: 1
    - Output 0: 44999999999643566 lovelace
  Minting:
    82b474fbbd69ac8f.testAsset: -2
  Fee: 179449 lovelace
  VKey Witnesses: 3
  Native Scripts: 1

```

### What This Tests

Tests that the multi-asset value conservation rule is enforced:

1. **Setup Transaction (SUCCESS)**: First, a token called "testAsset" is minted correctly
   - Input: Genesis UTxO with ~45,000,000 ADA
   - Output 1: UTxO with 1,038,710 lovelace + 1 testAsset
   - Output 2: UTxO with remaining ADA
   - Mint: +1 testAsset (properly authorized by native script)

2. **Failing Transaction (EXPECTED FAILURE)**: Attempts to burn more tokens than exist
   - Inputs: Consumes both outputs from the setup transaction
   - Input tokens available: 1 testAsset
   - Burn requested: -2 testAsset (tries to burn 2 tokens when only 1 exists!)
   - This violates: sum(inputs) + mint >= 0 for each asset

3. **Error**: The ShelleyUtxoPredFailure/ValueNotConservedUTxO error is raised because
   the balance equation cannot be satisfied - you cannot burn tokens that don't exist.

This validates that the ledger correctly tracks multi-asset balances and rejects
transactions that would result in negative token quantities.

---


