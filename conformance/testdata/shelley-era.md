
# ShelleyImpSpec Conformance Test Vectors

This document describes all Shelley-era conformance test vectors in the Conway implementation.

## Overview

Total test vectors: 11

### Vectors by Ledger Rule

| Rule | Count | Description |
|------|-------|-------------|
| EPOCH | 1 | Epoch transition rules - tests epoch boundary behavior |
| LEDGER | 1 | Core ledger rules - UTxO state updates |
| UTXO | 1 | UTxO validation rules - transaction value conservation |
| UTXOW | 8 | UTxO witness rules - signatures, scripts, metadata |

---

## Test: Conway/Imp/ShelleyImpSpec/EPOCH/Runs basic transaction

**File:** `ShelleyImpSpec/EPOCH/Runs_basic_transaction`
**Rule:** EPOCH
**Expected:** Success

### Initial State

```
Epoch: 899
UTxOs: 0 entries
Stake Registrations: 0
Pool Registrations: 0
DRep Registrations: 0
Committee Members: 0
Proposals: 0
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
  Outputs: 1
    - Output 0: 44999999999834587 lovelace
  Fee: 165413 lovelace
  VKey Witnesses: 1

Event 3: Pass Tick to slot 4320

Event 4: Pass 1 epoch(s)

```

### What This Tests

Tests that a basic transaction can be processed across an epoch boundary.
This validates the EPOCH rule correctly handles UTxO state transitions when epochs advance.
The transaction should successfully spend inputs and produce outputs while fees are collected.

---

## Test: Conway/Imp/ShelleyImpSpec/LEDGER/Transactions update UTxO

**File:** `ShelleyImpSpec/LEDGER/Transactions_update_UTxO`
**Rule:** LEDGER
**Expected:** Success

### Initial State

```
Epoch: 899
UTxOs: 0 entries
Stake Registrations: 0
Pool Registrations: 0
DRep Registrations: 0
Committee Members: 0
Proposals: 0
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
    - Output 0: 2000000 lovelace
    - Output 1: 44999999997831727 lovelace
  Fee: 168273 lovelace
  VKey Witnesses: 1

Event 3: Transaction at slot 3883681
  Result: SUCCESS
  Inputs: 2
    - d1a39ad4...8ccbdd17#0
    - d1a39ad4...8ccbdd17#1
  Outputs: 2
    - Output 0: 3000000 lovelace
    - Output 1: 44999999996657426 lovelace
  Fee: 174301 lovelace
  VKey Witnesses: 2

```

### What This Tests

Tests that transactions correctly update the UTxO set.
This validates the LEDGER rule's UTxO state machine transition function.
Inputs are consumed (removed from UTxO set) and outputs are produced (added to UTxO set).

---

## Test: Conway/Imp/ShelleyImpSpec/UTXO/ShelleyUtxoPredFailure/ValueNotConservedUTxO

**File:** `ShelleyImpSpec/UTXO/ShelleyUtxoPredFailure/ValueNotConservedUTxO`
**Rule:** UTXO
**Expected:** Failure

### Initial State

```
Epoch: 899
UTxOs: 0 entries
Stake Registrations: 0
Pool Registrations: 0
DRep Registrations: 0
Committee Members: 0
Proposals: 0
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
    - Output 0: 2000000 lovelace
    - Output 1: 44999999997832959 lovelace
  Fee: 167041 lovelace
  VKey Witnesses: 1

Event 3: Transaction at slot 3883681
  Result: FAILURE (expected)
  Inputs: 2
    - 0e3ba3ac...a169fc27#0
    - 0e3ba3ac...a169fc27#1
  Outputs: 2
    - Output 0: 849073 lovelace
    - Output 1: 44999999998810820 lovelace
  Fee: 173069 lovelace
  VKey Witnesses: 2

```

### What This Tests

Tests the UTxO value conservation rule (aka "balance equation").
This is a FAILURE test - the transaction intentionally violates value conservation.
The sum of input values plus mints must equal the sum of output values plus fee plus burns.
Failure indicates the ledger correctly rejects transactions that don't balance.

---

## Test: Conway/Imp/ShelleyImpSpec/UTXOW/Bootstrap Witness/InvalidWitnessesUTXOW

**File:** `ShelleyImpSpec/UTXOW/Bootstrap_Witness/InvalidWitnessesUTXOW`
**Rule:** UTXOW
**Expected:** Failure

### Initial State

```
Epoch: 899
UTxOs: 0 entries
Stake Registrations: 0
Pool Registrations: 0
DRep Registrations: 0
Committee Members: 0
Proposals: 0
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
    - Output 0: 1051640 lovelace
    - Output 1: 44999999998779251 lovelace
  Fee: 169109 lovelace
  VKey Witnesses: 1

Event 3: Transaction at slot 3883681
  Result: FAILURE (expected)
  Inputs: 2
    - 4fe9f00f...6fc43366#0
    - 4fe9f00f...6fc43366#1
  Outputs: 1
    - Output 0: 44999999999648626 lovelace
  Fee: 182265 lovelace
  VKey Witnesses: 1
  Bootstrap Witnesses: 1

```

### What This Tests

Tests that invalid bootstrap (Byron-era) witnesses are rejected.
This is a FAILURE test - the transaction has malformed or incorrect bootstrap witnesses.
The UTXOW rule requires all witnesses to be valid cryptographic signatures.

---

## Test: Conway/Imp/ShelleyImpSpec/UTXOW/Bootstrap Witness/Valid Witnesses

**File:** `ShelleyImpSpec/UTXOW/Bootstrap_Witness/Valid_Witnesses`
**Rule:** UTXOW
**Expected:** Success

### Initial State

```
Epoch: 899
UTxOs: 0 entries
Stake Registrations: 0
Pool Registrations: 0
DRep Registrations: 0
Committee Members: 0
Proposals: 0
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
    - Output 0: 1051640 lovelace
    - Output 1: 44999999998779251 lovelace
  Fee: 169109 lovelace
  VKey Witnesses: 1

Event 3: Transaction at slot 3883681
  Result: SUCCESS
  Inputs: 2
    - 4fe9f00f...6fc43366#0
    - 4fe9f00f...6fc43366#1
  Outputs: 1
    - Output 0: 44999999999656150 lovelace
  Fee: 174741 lovelace
  VKey Witnesses: 1
  Bootstrap Witnesses: 1

```

### What This Tests

Tests that valid bootstrap (Byron-era) witnesses are accepted.
This validates backwards compatibility with Byron-era addresses and witnesses.
The transaction should succeed because all bootstrap witnesses are correctly formed.

---

## Test: Conway/Imp/ShelleyImpSpec/UTXOW/ConflictingMetadataHash

**File:** `ShelleyImpSpec/UTXOW/ConflictingMetadataHash`
**Rule:** UTXOW
**Expected:** Failure

### Initial State

```
Epoch: 899
UTxOs: 0 entries
Stake Registrations: 0
Pool Registrations: 0
DRep Registrations: 0
Committee Members: 0
Proposals: 0
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
    - Output 0: 44999999999832915 lovelace
  Fee: 167085 lovelace
  VKey Witnesses: 1

```

### What This Tests

Tests that conflicting metadata hashes are rejected.
This is a FAILURE test - the transaction body's metadata hash doesn't match the actual metadata.
The UTXOW rule requires AuxiliaryDataHash in the body to match the hash of provided metadata.

---

## Test: Conway/Imp/ShelleyImpSpec/UTXOW/ExtraneousScriptWitnessesUTXOW

**File:** `ShelleyImpSpec/UTXOW/ExtraneousScriptWitnessesUTXOW`
**Rule:** UTXOW
**Expected:** Failure

### Initial State

```
Epoch: 899
UTxOs: 0 entries
Stake Registrations: 0
Pool Registrations: 0
DRep Registrations: 0
Committee Members: 0
Proposals: 0
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
    - Output 0: 44999999999832959 lovelace
  Fee: 167041 lovelace
  VKey Witnesses: 1
  Native Scripts: 1

```

### What This Tests

Tests that extraneous script witnesses are rejected.
This is a FAILURE test - the transaction includes script witnesses not required by any input.
The UTXOW rule requires all provided witnesses to be necessary for validation.

---

## Test: Conway/Imp/ShelleyImpSpec/UTXOW/MissingScriptWitnessesUTXOW

**File:** `ShelleyImpSpec/UTXOW/MissingScriptWitnessesUTXOW`
**Rule:** UTXOW
**Expected:** Failure

### Initial State

```
Epoch: 899
UTxOs: 0 entries
Stake Registrations: 0
Pool Registrations: 0
DRep Registrations: 0
Committee Members: 0
Proposals: 0
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
    - Output 0: 849070 lovelace
    - Output 1: 44999999998983889 lovelace
  Fee: 167041 lovelace
  VKey Witnesses: 1

Event 3: Transaction at slot 3883681
  Result: FAILURE (expected)
  Inputs: 2
    - 06eec292...3cde2783#0
    - 06eec292...3cde2783#1
  Outputs: 1
    - Output 0: 44999999999665962 lovelace
  Fee: 166997 lovelace
  VKey Witnesses: 1

```

### What This Tests

Tests that missing required script witnesses are detected.
This is a FAILURE test - the transaction is missing a script required to validate a script-locked input.
The UTXOW rule requires all scripts referenced by inputs to be present in witnesses.

---

## Test: Conway/Imp/ShelleyImpSpec/UTXOW/MissingTxBodyMetadataHash

**File:** `ShelleyImpSpec/UTXOW/MissingTxBodyMetadataHash`
**Rule:** UTXOW
**Expected:** Failure

### Initial State

```
Epoch: 899
UTxOs: 0 entries
Stake Registrations: 0
Pool Registrations: 0
DRep Registrations: 0
Committee Members: 0
Proposals: 0
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
    - Output 0: 44999999999833751 lovelace
  Fee: 166249 lovelace
  VKey Witnesses: 1

```

### What This Tests

Tests that transactions with metadata must have a metadata hash.
This is a FAILURE test - metadata is provided but no hash is in the transaction body.
The UTXOW rule requires AuxiliaryDataHash when metadata is present.

---

## Test: Conway/Imp/ShelleyImpSpec/UTXOW/MissingTxMetadata

**File:** `ShelleyImpSpec/UTXOW/MissingTxMetadata`
**Rule:** UTXOW
**Expected:** Failure

### Initial State

```
Epoch: 899
UTxOs: 0 entries
Stake Registrations: 0
Pool Registrations: 0
DRep Registrations: 0
Committee Members: 0
Proposals: 0
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
    - Output 0: 44999999999833047 lovelace
  Fee: 166953 lovelace
  VKey Witnesses: 1

```

### What This Tests

Tests that transactions with a metadata hash must provide metadata.
This is a FAILURE test - a metadata hash is in the body but no metadata is provided.
The UTXOW rule requires actual metadata when AuxiliaryDataHash is present.

---

## Test: Conway/Imp/ShelleyImpSpec/UTXOW/MissingVKeyWitnessesUTXOW

**File:** `ShelleyImpSpec/UTXOW/MissingVKeyWitnessesUTXOW`
**Rule:** UTXOW
**Expected:** Failure

### Initial State

```
Epoch: 899
UTxOs: 0 entries
Stake Registrations: 0
Pool Registrations: 0
DRep Registrations: 0
Committee Members: 0
Proposals: 0
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
    - Output 0: 849070 lovelace
    - Output 1: 44999999998983889 lovelace
  Fee: 167041 lovelace
  VKey Witnesses: 1

Event 3: Transaction at slot 3883681
  Result: FAILURE (expected)
  Inputs: 2
    - ca291387...bab3e1ba#0
    - ca291387...bab3e1ba#1
  Outputs: 1
    - Output 0: 44999999999661518 lovelace
  Fee: 171441 lovelace
  VKey Witnesses: 1

```

### What This Tests

Tests that missing required VKey witnesses are detected.
This is a FAILURE test - the transaction is missing a signature required by an input address.
The UTXOW rule requires all pub-key-hash inputs to have corresponding signatures.

---


