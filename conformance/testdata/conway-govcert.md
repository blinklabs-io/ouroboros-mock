
# Conway GOVCERT Conformance Test Vectors

This directory contains conformance test vectors for the Conway GOVCERT (governance certificate) transition rule.

## Overview

**Total Test Vectors:** 9

### GOVCERT Transition Rule

The GOVCERT rule validates governance-related certificates in Conway era transactions:

1. **DRep Registration** (type 16): Register a new DRep with deposit
2. **DRep Deregistration** (type 17): Deregister a DRep and reclaim deposit
3. **DRep Update** (type 18): Update DRep metadata anchor
4. **Committee Hot Key Authorization** (type 14): Authorize hot key for committee cold key
5. **Committee Resignation** (type 15): Resign from constitutional committee

### By Category

| Category | Count |
|----------|-------|
| Resigning_proposed_CC_key | 1 |
| fails_for | 5 |
| succeeds_for | 3 |

### By Certificate Type

| Certificate Type | Count |
|-----------------|-------|
| Committee Hot Key Authorization | 2 |
| Committee Resignation | 2 |
| DRep Deregistration (Refund) | 1 |
| DRep Lifecycle (Register/Deregister) | 2 |
| DRep Registration | 1 |
| DRep Registration (Deposit) | 1 |

### By Expected Outcome

| Outcome | Count |
|---------|-------|
| Success | 4 |
| Failure | 5 |

---

## Resigning proposed CC key

### Test: Resigning proposed CC key

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOVCERT/Resigning_proposed_CC_key`

**Rule:** GOVCERT

**Certificate Type:** Committee Resignation

**Expected:** Success

#### State Change Diagram

```
Initial State:
  ├── Epoch: 899
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0

Event: Transaction at slot 3883681 (SUCCESS)
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Resignation
  │       ├── Cold Credential: 3c875ce0f647bdcc...
  └── Validation: PASSED

Final State:
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0
```

#### What This Tests

Tests that proposed committee members can resign before enactment. Expected to succeed.

**Rules Tested:**
- GOVCERT-CC4: Proposed committee member can resign

---

## fails for

### Test: DRep already registered

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOVCERT/fails_for/DRep_already_registered`

**Rule:** GOVCERT

**Certificate Type:** DRep Registration

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  ├── Epoch: 899
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── DRep Registration
  │       ├── Credential: 3c875ce0f647bdcc... (KeyHash)
  │       ├── Deposit: 100 lovelace
  └── Validation: PASSED

Event: Transaction at slot 3883681 (FAILURE)
  ├── Certificates:
  │   └── DRep Registration
  │       ├── Credential: 3c875ce0f647bdcc... (KeyHash)
  │       ├── Deposit: 100 lovelace
  └── Validation: FAILED
      └── Reason: DRep already registered

Final State:
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0
```

#### What This Tests

Tests that the GOVCERT rule correctly rejects DRep registration when the credential is already registered. Expected to fail validation.

**Rules Tested:**
- GOVCERT-DREP1: DRep cannot be registered if already registered

---

### Test: invalid deposit provided with DRep registration cert

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOVCERT/fails_for/invalid_deposit_provided_with_DRep_registration_cert`

**Rule:** GOVCERT

**Certificate Type:** DRep Registration (Deposit)

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  ├── Epoch: 899
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0

Event: Transaction at slot 3883681 (FAILURE)
  ├── Certificates:
  │   └── DRep Registration
  │       ├── Credential: 3c875ce0f647bdcc... (KeyHash)
  │       ├── Deposit: 110 lovelace
  └── Validation: FAILED
      └── Reason: Incorrect deposit amount

Final State:
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0
```

#### What This Tests

Tests that the GOVCERT rule validates the deposit amount matches the protocol parameter drepDeposit. Expected to fail validation.

**Rules Tested:**
- GOVCERT-DREP2: DRep registration deposit must match drepDeposit protocol parameter

---

### Test: invalid refund provided with DRep deregistration cert

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOVCERT/fails_for/invalid_refund_provided_with_DRep_deregistration_cert`

**Rule:** GOVCERT

**Certificate Type:** DRep Deregistration (Refund)

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  ├── Epoch: 899
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── DRep Registration
  │       ├── Credential: 3c875ce0f647bdcc... (KeyHash)
  │       ├── Deposit: 100 lovelace
  └── Validation: PASSED

Event: Transaction at slot 3883681 (FAILURE)
  ├── Certificates:
  │   └── DRep Deregistration
  │       ├── Credential: 3c875ce0f647bdcc... (KeyHash)
  │       ├── Refund: 110 lovelace
  └── Validation: FAILED
      └── Reason: Incorrect refund amount

Final State:
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0
```

#### What This Tests

Tests that the GOVCERT rule validates the refund amount matches the original registration deposit. Expected to fail validation.

**Rules Tested:**
- GOVCERT-DREP3: DRep deregistration refund must match original deposit

---

### Test: registering a resigned CC member hotkey

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOVCERT/fails_for/registering_a_resigned_CC_member_hotkey`

**Rule:** GOVCERT

**Certificate Type:** Committee Hot Key Authorization

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  ├── Epoch: 899
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   ├── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: 3c875ce0f647bdcc...
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: 6b75eafbb349012d...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: f1e760111aba8c48...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Resignation
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (FAILURE)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: f1e760111aba8c48...
  └── Validation: FAILED
      └── Reason: Committee member has resigned

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: 3bd89d35297cb0d6...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Resignation
  │       ├── Cold Credential: 65d36c561e076d2a...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (FAILURE)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: 3bd89d35297cb0d6...
  └── Validation: FAILED
      └── Reason: Committee member has resigned

Final State:
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0
```

#### What This Tests

Tests that resigned committee members cannot authorize new hot keys. Expected to fail validation.

**Rules Tested:**
- GOVCERT-CC1: Cannot authorize hot key for resigned committee member

---

### Test: unregistering a nonexistent DRep

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOVCERT/fails_for/unregistering_a_nonexistent_DRep`

**Rule:** GOVCERT

**Certificate Type:** DRep Lifecycle (Register/Deregister)

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  ├── Epoch: 899
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0

Event: Transaction at slot 3883681 (FAILURE)
  ├── Certificates:
  │   └── DRep Deregistration
  │       ├── Credential: 3c875ce0f647bdcc... (KeyHash)
  │       ├── Refund: 100 lovelace
  └── Validation: FAILED
      └── Reason: DRep not registered

Final State:
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0
```

#### What This Tests

Tests that the GOVCERT rule rejects deregistration of a non-existent DRep. Expected to fail validation.

**Rules Tested:**
- GOVCERT-DREP4: Cannot deregister a non-existent DRep

---

## succeeds for

### Test: re-registering a CC hot key

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOVCERT/succeeds_for/re-registering_a_CC_hot_key`

**Rule:** GOVCERT

**Certificate Type:** Committee Hot Key Authorization

**Expected:** Success

#### State Change Diagram

```
Initial State:
  ├── Epoch: 899
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   ├── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: 3c875ce0f647bdcc...
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: 6b75eafbb349012d...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: f1e760111aba8c48...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: 7ca77ed06e594628...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: 1a88058808c2a761...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: c73ef03951c90de9...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: daec416a6217ca94...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: a12b4d44520a6fc5...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: a7b37bbaec93d167...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: 88f2d1f5b3021745...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: 5f7ce813e7f280d1...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 204c5f1bafe8ee28...
  │       ├── Hot Credential: c9d8be537e3cdad5...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: 401836f8ed2393a8...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: 714d411c5c5c690f...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: d9e2be633605a37a...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: b0cdd051e7ce8f8d...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: 708483b924148db8...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: 084e9ad60b5f5291...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: af74ab3b52abbeb0...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: f1cf5d35acabb037...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: ba0e70e3a771996e...
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── Committee Hot Key Authorization
  │       ├── Cold Credential: 65d36c561e076d2a...
  │       ├── Hot Credential: 8ac6c35a8c4e8d71...
  └── Validation: PASSED

Final State:
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0
```

#### What This Tests

Tests that committee members can update their hot key authorization. Expected to succeed.

**Rules Tested:**
- GOVCERT-CC2: Committee member can re-register hot key authorization

---

### Test: registering and unregistering a DRep

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOVCERT/succeeds_for/registering_and_unregistering_a_DRep`

**Rule:** GOVCERT

**Certificate Type:** DRep Lifecycle (Register/Deregister)

**Expected:** Success

#### State Change Diagram

```
Initial State:
  ├── Epoch: 899
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── DRep Registration
  │       ├── Credential: 3c875ce0f647bdcc... (KeyHash)
  │       ├── Deposit: 100 lovelace
  └── Validation: PASSED

Event: Transaction at slot 3883681 (SUCCESS)
  ├── Certificates:
  │   └── DRep Deregistration
  │       ├── Credential: 3c875ce0f647bdcc... (KeyHash)
  │       ├── Refund: 100 lovelace
  └── Validation: PASSED

Final State:
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0
```

#### What This Tests

Tests the complete DRep lifecycle from registration to deregistration. Expected to succeed.

**Rules Tested:**
- GOVCERT-DREP5: DRep lifecycle - registration followed by deregistration

---

### Test: resigning a non-CC key

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOVCERT/succeeds_for/resigning_a_non-CC_key`

**Rule:** GOVCERT

**Certificate Type:** Committee Resignation

**Expected:** Transaction fails validation (test passes by correctly rejecting)

#### State Change Diagram

```
Initial State:
  ├── Epoch: 899
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0

Event: Transaction at slot 3883681 (FAILURE)
  ├── Certificates:
  │   └── Committee Resignation
  │       ├── Cold Credential: 3c875ce0f647bdcc...
  └── Validation: FAILED
      └── Reason: Not a committee member

Final State:
  ├── DReps: 0 registered
  ├── Committee Members: 0
  └── Hot Key Authorizations: 0
```

#### What This Tests

Tests that only current or proposed committee members can submit resignation certificates. A credential that is neither a current member nor proposed in a pending UpdateCommittee action cannot resign. The test vector is in `succeeds_for/` because the harness correctly handles this rejection.

**Rules Tested:**
- GOVCERT-CC3: Resignation requires credential to be current or proposed CC member

---


