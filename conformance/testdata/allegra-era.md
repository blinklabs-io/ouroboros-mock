
# AllegraImpSpec Conformance Test Vectors

This document describes the Allegra-era conformance test vectors.
Allegra introduced time-locking (validity intervals) and metadata improvements.

## Overview

Total test vectors: 1

### Vectors by Ledger Rule

| Rule | Count | Description |
|------|-------|-------------|
| UTXOW | 1 | UTxO witness rules - signatures, scripts, metadata with time-locking |

---

## Test: Conway/Imp/AllegraImpSpec/UTXOW/InvalidMetadata

**File:** `AllegraImpSpec/UTXOW/InvalidMetadata`
**Rule:** UTXOW
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
  Result: FAILURE (expected)
  Inputs: 1
    - 03170a2e...4c111314#0
  Outputs: 1
    - Output 0: 44999999999803215 lovelace
  Fee: 196785 lovelace
  Has Metadata: yes (INVALID)
  VKey Witnesses: 1

```

### What This Tests

Tests the UTXOW rule's metadata validation:

1. **Metadata Hash Verification**: Allegra (and later eras) require that if
   auxiliary data (metadata) is included in a transaction, the hash of that
   metadata must match the auxiliary_data_hash field in the transaction body

2. **Invalid Metadata Rejection**: The transaction is rejected because:
   - The metadata is malformed or doesn't match its declared hash
   - This could be due to:
     - Hash mismatch between body and actual metadata
     - Malformed CBOR in the metadata
     - Metadata present but hash missing from body (or vice versa)

3. **State Preservation**: Since the transaction fails validation,
   the UTxO set remains unchanged

The UTXOW rule ensures transaction integrity by validating:
- All required signatures are present and valid
- All scripts are satisfied
- Metadata hash matches actual metadata
- No extraneous witnesses are included

---


