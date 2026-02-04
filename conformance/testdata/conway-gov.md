
# Conway GOV Conformance Test Vectors

This directory contains conformance test vectors for the Conway GOV (governance) transition rule.

## Overview

**Total Test Vectors:** 55

### By Category

| Category | Count |
|----------|-------|
| Constitution_proposals | 5 |
| HardFork | 6 |
| Network_ID | 1 |
| PParamUpdate | 11 |
| Policy | 1 |
| Predicate_failures | 4 |
| Proposals | 8 |
| Proposing_and_voting | 7 |
| Unknown_CostModels | 1 |
| Voting | 8 |
| Withdrawals | 3 |

### By Expected Outcome

| Outcome | Count |
|---------|-------|
| Success | 23 |
| Failure | 32 |

---

## Constitution proposals

### Test: empty PrevGovId before the first constitution is enacted

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Constitution_proposals/accepted_for/empty_PrevGovId_before_the_first_constitution_is_enacted`

**Rule:** GOV

**Action Type:** NewConstitution

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://YmcuwygChpNVX5qnoZm3m2.com

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://rRm2x8V4-sge5cHvPv.3uNZ8gYbs.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests constitution proposal chain validation. Expected to succeed.

**Rules Tested:**
- GOV14: Constitution proposals require valid previous action
- GOV15: First constitution uses empty PrevGovId

---

### Test: valid GovPurposeId

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Constitution_proposals/accepted_for/valid_GovPurposeId`

**Rule:** GOV

**Action Type:** NewConstitution

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://bNJssFtMhlRfnMB.com

Event 7: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on fae8dfcae1a80999...#0

Event 8: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on fae8dfcae1a80999...#0

Event 9: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on fae8dfcae1a80999...#0

Event 10: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 11: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://TLwLZsIh5Gw6V6cLqD3SstuhhL.com

Event 12: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on 1ada86767310d0e8...#0

Event 13: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 1ada86767310d0e8...#0

Event 14: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 1ada86767310d0e8...#0

Epoch Advances: 4

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests constitution proposal chain validation. Expected to succeed.

**Rules Tested:**
- GOV14: Constitution proposals require valid previous action
- GOV15: First constitution uses empty PrevGovId

---

### Test: empty PrevGovId after the first constitution was enacted

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Constitution_proposals/rejected_for/empty_PrevGovId_after_the_first_constitution_was_enacted`

**Rule:** GOV

**Action Type:** NewConstitution

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://bNJssFtMhlRfnMB.com

Event 7: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on fae8dfcae1a80999...#0

Event 8: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on fae8dfcae1a80999...#0

Event 9: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on fae8dfcae1a80999...#0

Event 10: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 11: Transaction at slot 3892321 (FAILURE)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://TLwLZsIh5Gw6V6cLqD3SstuhhL.com

Epoch Advances: 2

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests constitution proposal chain validation. Expected to fail validation.

**Rules Tested:**
- GOV14: Constitution proposals require valid previous action
- GOV15: First constitution uses empty PrevGovId

---

### Test: invalid index in GovPurposeId

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Constitution_proposals/rejected_for/invalid_index_in_GovPurposeId`

**Rule:** GOV

**Action Type:** NewConstitution

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://YmcuwygChpNVX5qnoZm3m2.com

Event 3: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 4: Transaction at slot 3892321 (FAILURE)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://rRm2x8V4-sge5cHvPv.3uNZ8gYbs.com

Epoch Advances: 2

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests constitution proposal chain validation. Expected to fail validation.

**Rules Tested:**
- GOV14: Constitution proposals require valid previous action
- GOV15: First constitution uses empty PrevGovId

---

### Test: valid GovPurposeId but invalid purpose

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Constitution_proposals/rejected_for/valid_GovPurposeId_but_invalid_purpose`

**Rule:** GOV

**Action Type:** NewConstitution

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://YmcuwygChpNVX5qnoZm3m2.com

Event 3: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 4: Transaction at slot 3892321 (FAILURE)
  Proposal Procedures:
    - NoConfidence (deposit: 123)
      Anchor: https://-ziXabgJ6PauMvahisT.com

Epoch Advances: 2

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests constitution proposal chain validation. Expected to fail validation.

**Rules Tested:**
- GOV14: Constitution proposals require valid previous action
- GOV15: First constitution uses empty PrevGovId

---

## HardFork

### Test: Hardfork cantFollow

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/HardFork/Hardfork_is_the_first_one_(doesnt_have_a_GovPurposeId)/Hardfork_cantFollow`

**Rule:** GOV

**Action Type:** HardForkInitiation

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - HardForkInitiation (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests hard fork initiation protocol version rules. Expected to fail validation.

**Rules Tested:**
- GOV16: HardFork must increment protocol version
- GOV17: HardFork version must be valid successor

---

### Test: Hardfork majorFollow

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/HardFork/Hardfork_is_the_first_one_(doesnt_have_a_GovPurposeId)/Hardfork_majorFollow`

**Rule:** GOV

**Action Type:** HardForkInitiation

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - HardForkInitiation (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests hard fork initiation protocol version rules. Expected to succeed.

**Rules Tested:**
- GOV16: HardFork must increment protocol version
- GOV17: HardFork version must be valid successor

---

### Test: Hardfork minorFollow

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/HardFork/Hardfork_is_the_first_one_(doesnt_have_a_GovPurposeId)/Hardfork_minorFollow`

**Rule:** GOV

**Action Type:** HardForkInitiation

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - HardForkInitiation (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests hard fork initiation protocol version rules. Expected to succeed.

**Rules Tested:**
- GOV16: HardFork must increment protocol version
- GOV17: HardFork version must be valid successor

---

### Test: Hardfork cantFollow

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/HardFork/Hardfork_is_the_second_one_(has_a_GovPurposeId)/Hardfork_cantFollow`

**Rule:** GOV

**Action Type:** HardForkInitiation

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - HardForkInitiation (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 4: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - HardForkInitiation (deposit: 123)
      Anchor: https://f2jt6pji8Cn98l5Pke-T10M76xfZGRC4CdBt3lR...

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests hard fork initiation protocol version rules. Expected to fail validation.

**Rules Tested:**
- GOV16: HardFork must increment protocol version
- GOV17: HardFork version must be valid successor

---

### Test: Hardfork majorFollow

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/HardFork/Hardfork_is_the_second_one_(has_a_GovPurposeId)/Hardfork_majorFollow`

**Rule:** GOV

**Action Type:** HardForkInitiation

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - HardForkInitiation (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - HardForkInitiation (deposit: 123)
      Anchor: https://f2jt6pji8Cn98l5Pke-T10M76xfZGRC4CdBt3lR...

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests hard fork initiation protocol version rules. Expected to succeed.

**Rules Tested:**
- GOV16: HardFork must increment protocol version
- GOV17: HardFork version must be valid successor

---

### Test: Hardfork minorFollow

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/HardFork/Hardfork_is_the_second_one_(has_a_GovPurposeId)/Hardfork_minorFollow`

**Rule:** GOV

**Action Type:** HardForkInitiation

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - HardForkInitiation (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - HardForkInitiation (deposit: 123)
      Anchor: https://f2jt6pji8Cn98l5Pke-T10M76xfZGRC4CdBt3lR...

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests hard fork initiation protocol version rules. Expected to succeed.

**Rules Tested:**
- GOV16: HardFork must increment protocol version
- GOV17: HardFork version must be valid successor

---

## Network ID

### Test: Fails with invalid network ID in proposal return address

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Network_ID/Fails_with_invalid_network_ID_in_proposal_return_address`

**Rule:** GOV

**Action Type:** NetworkID

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - InfoAction (deposit: 123)
      Anchor: https://R7Uh0RdzD4NomVqifS7jCCymlaWBeyK39OyDG0p...

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests network ID validation in governance actions. Expected to fail validation.

**Rules Tested:**
- GOV23: Network ID must match in proposal return address

---

## PParamUpdate

### Test: PPU cannot be empty

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/PParamUpdate/PPU_needs_to_be_wellformed/PPU_cannot_be_empty`

**Rule:** GOV

**Action Type:** ParameterChange

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests parameter update proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV18: Parameter updates must be well-formed
- GOV19: Certain parameters cannot be zero

---

### Test: ppuCollateralPercentageL cannot be 0

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/PParamUpdate/PPU_needs_to_be_wellformed/ppuCollateralPercentageL_cannot_be_0`

**Rule:** GOV

**Action Type:** ParameterChange

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests parameter update proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV18: Parameter updates must be well-formed
- GOV19: Certain parameters cannot be zero

---

### Test: ppuCommitteeMaxTermLengthL cannot be 0

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/PParamUpdate/PPU_needs_to_be_wellformed/ppuCommitteeMaxTermLengthL_cannot_be_0`

**Rule:** GOV

**Action Type:** ParameterChange

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests parameter update proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV18: Parameter updates must be well-formed
- GOV19: Certain parameters cannot be zero

---

### Test: ppuDRepDepositL cannot be 0

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/PParamUpdate/PPU_needs_to_be_wellformed/ppuDRepDepositL_cannot_be_0`

**Rule:** GOV

**Action Type:** ParameterChange

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests parameter update proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV18: Parameter updates must be well-formed
- GOV19: Certain parameters cannot be zero

---

### Test: ppuGovActionDepositL cannot be 0

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/PParamUpdate/PPU_needs_to_be_wellformed/ppuGovActionDepositL_cannot_be_0`

**Rule:** GOV

**Action Type:** ParameterChange

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests parameter update proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV18: Parameter updates must be well-formed
- GOV19: Certain parameters cannot be zero

---

### Test: ppuGovActionLifetimeL cannot be 0

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/PParamUpdate/PPU_needs_to_be_wellformed/ppuGovActionLifetimeL_cannot_be_0`

**Rule:** GOV

**Action Type:** ParameterChange

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests parameter update proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV18: Parameter updates must be well-formed
- GOV19: Certain parameters cannot be zero

---

### Test: ppuMaxBBSizeL cannot be 0

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/PParamUpdate/PPU_needs_to_be_wellformed/ppuMaxBBSizeL_cannot_be_0`

**Rule:** GOV

**Action Type:** ParameterChange

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests parameter update proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV18: Parameter updates must be well-formed
- GOV19: Certain parameters cannot be zero

---

### Test: ppuMaxBHSizeL cannot be 0

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/PParamUpdate/PPU_needs_to_be_wellformed/ppuMaxBHSizeL_cannot_be_0`

**Rule:** GOV

**Action Type:** ParameterChange

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests parameter update proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV18: Parameter updates must be well-formed
- GOV19: Certain parameters cannot be zero

---

### Test: ppuMaxTxSizeL cannot be 0

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/PParamUpdate/PPU_needs_to_be_wellformed/ppuMaxTxSizeL_cannot_be_0`

**Rule:** GOV

**Action Type:** ParameterChange

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests parameter update proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV18: Parameter updates must be well-formed
- GOV19: Certain parameters cannot be zero

---

### Test: ppuMaxValSizeL cannot be 0

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/PParamUpdate/PPU_needs_to_be_wellformed/ppuMaxValSizeL_cannot_be_0`

**Rule:** GOV

**Action Type:** ParameterChange

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests parameter update proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV18: Parameter updates must be well-formed
- GOV19: Certain parameters cannot be zero

---

### Test: ppuPoolDepositL cannot be 0

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/PParamUpdate/PPU_needs_to_be_wellformed/ppuPoolDepositL_cannot_be_0`

**Rule:** GOV

**Action Type:** ParameterChange

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests parameter update proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV18: Parameter updates must be well-formed
- GOV19: Certain parameters cannot be zero

---

## Policy

### Test: policy is respected by proposals

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Policy/policy_is_respected_by_proposals`

**Rule:** GOV

**Action Type:** Policy

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://vDXhIdY8akvbUC4Ob.com

Event 7: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on 7c395149073c9515...#0

Event 8: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 7c395149073c9515...#0

Event 9: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 7c395149073c9515...#0

Event 10: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 11: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://hqUsq4Z.com

Event 12: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 13: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 14: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - TreasuryWithdrawal (deposit: 123)
      Anchor: https://STVo7-MThy4Nt.com

Event 15: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 16: Transaction at slot 3892321 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://EXuURhG9PaE.com

Event 17: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 18: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 19: Transaction at slot 3892321 (FAILURE)
  Proposal Procedures:
    - TreasuryWithdrawal (deposit: 123)
      Anchor: https://.3tYc8TY8Mzsl4dVY4V.com

Epoch Advances: 2

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests constitution policy enforcement. Expected to fail validation.

**Rules Tested:**
- GOV24: Proposals must respect constitution policy

---

## Predicate failures

### Test: ConflictingCommitteeUpdate

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Predicate_failures/ConflictingCommitteeUpdate`

**Rule:** GOV

**Action Type:** PredicateFailure

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - UpdateCommittee (deposit: 123)
      Anchor: https://790eMzUN2fR2Ytiq.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests that the GOV rule correctly rejects invalid governance actions. Expected to fail validation.

**Rules Tested:**
- GOV4: Committee update cannot add and remove same member

---

### Test: ExpirationEpochTooSmall

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Predicate_failures/ExpirationEpochTooSmall`

**Rule:** GOV

**Action Type:** PredicateFailure

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 900
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3888001 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3888001 (FAILURE)
  Proposal Procedures:
    - UpdateCommittee (deposit: 123)
      Anchor: https://790eMzUN2fR2Ytiq.com

Epoch Advances: 1

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests that the GOV rule correctly rejects invalid governance actions. Expected to fail validation.

**Rules Tested:**
- GOV3: Committee expiration epoch must be valid

---

### Test: ProposalDepositIncorrect

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Predicate_failures/ProposalDepositIncorrect`

**Rule:** GOV

**Action Type:** PredicateFailure

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - InfoAction (deposit: 122)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests that the GOV rule correctly rejects invalid governance actions. Expected to fail validation.

**Rules Tested:**
- GOV1: Proposal deposit must match govActionDeposit

---

### Test: ProposalReturnAccountDoesNotExist

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Predicate_failures/ProposalReturnAccountDoesNotExist`

**Rule:** GOV

**Action Type:** PredicateFailure

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - InfoAction (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - InfoAction (deposit: 123)
      Anchor: https://rml7vah4NG1NajIMhn3.V7dBMmScWRp2EEIXlWU...

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests that the GOV rule correctly rejects invalid governance actions. Expected to fail validation.

**Rules Tested:**
- GOV2: Proposal return address must be registered

---

## Proposals

### Test: Proposals are stored in the expected order

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposals/Consistency/Proposals_are_stored_in_the_expected_order`

**Rule:** GOV

**Action Type:** ProposalConsistency

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - InfoAction (deposit: 123)
      Anchor: https://rml7vah4NG1NajIMhn3.V7dBMmScWRp2EEIXlWU...

Event 7: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NoConfidence (deposit: 123)
      Anchor: https://34XwtXDuKGEUlqthCJvmLbGbjO-ZE6B6siofYK0...

Event 8: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - InfoAction (deposit: 123)
      Anchor: https://IuPXpx9sdoujjxZ0rtbgtX-BjWvEX0lt5xzcEZe...

Event 9: Transaction at slot 3883681 (SUCCESS)

Event 10: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - TreasuryWithdrawal (deposit: 123)
      Anchor: https://nRvR1ZiOKYLt59U4Om8vjUAYMKOPAc98NDfOIXI...

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests proposal consistency and lifecycle management. Expected to succeed.

**Rules Tested:**
- GOV11: Proposals stored in submission order
- GOV12: Proposals require valid parent action ID
- GOV13: Proposal subtrees pruned on expiry/enactment

---

### Test: Proposals submitted without proper parent fail

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposals/Consistency/Proposals_submitted_without_proper_parent_fail`

**Rule:** GOV

**Action Type:** ProposalConsistency

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 5: Transaction at slot 3883681 (SUCCESS)

Event 6: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://0Y-C91JZBQ8P59rPxTK.com

Event 7: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 8: Transaction at slot 3883681 (SUCCESS)

Event 9: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://snsCKcmsOGD1k5298dUbjrr-Hh-CWP7HFfcp1Kw...

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests proposal consistency and lifecycle management. Expected to fail validation.

**Rules Tested:**
- GOV11: Proposals stored in submission order
- GOV12: Proposals require valid parent action ID
- GOV13: Proposal subtrees pruned on expiry/enactment

---

### Test: Subtrees are pruned for both enactment and expiry over multiple rounds

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposals/Consistency/Subtrees_are_pruned_for_both_enactment_and_expiry_over_multiple_rounds`

**Rule:** GOV

**Action Type:** ProposalConsistency

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://bNJssFtMhlRfnMB.com

Event 7: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 8: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://H-SqaqL6I4.com

Event 9: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 10: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://eTpWQRCEetGzrJLaeO9JK7WhfYLSTO32LW2dUi7...

Event 11: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 12: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://KDuK.dJ925t8O2nMx.LaNlUnSpxXyRJcbq4jYdN...

Event 13: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 14: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://gjB7qz-svONCn.com

Event 15: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 16: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://3PknZA-e.QeAZtpUWea74cOOErDdaGSV00plE4u...

Event 17: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 18: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://9BP8KaJ-2jzcLSbUCioKEzL2d3MwuWAI4Uw7VLj...

Event 19: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 20: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://5Ov9u.jQTyUQgSZJrbRLCwylK4ettaJO3XyvIoD...

Event 21: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on fae8dfcae1a80999...#0

Event 22: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on fae8dfcae1a80999...#0

Event 23: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on fae8dfcae1a80999...#0

Event 24: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on 5cadf6fade36d94a...#0

Event 25: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 5cadf6fade36d94a...#0

Event 26: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 5cadf6fade36d94a...#0

Event 27: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on e859f8d2cc6fc1f0...#0

Event 28: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on e859f8d2cc6fc1f0...#0

Event 29: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on e859f8d2cc6fc1f0...#0

Event 30: Transaction at slot 3913921 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 31: Transaction at slot 3913921 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://cHzrL5hWvVongi-bskv81Td1-VL.com

Event 32: Transaction at slot 3913921 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 33: Transaction at slot 3913921 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://a5-Oc9MkjXDW7k.3M05XHA.com

Event 34: Transaction at slot 3931201 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 35: Transaction at slot 3931201 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://i36sqwQo7tN2EWrOjPxgZ4EcJ3ENCG5pjBhLRql...

Event 36: Transaction at slot 3931201 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 37: Transaction at slot 3931201 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://RFvBGk.Y59CCfJEVDAVZ1Ycl5vmBRv2Ge7rcJ0U...

Event 38: Transaction at slot 3939841 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on 134dbd9954128f93...#0

Event 39: Transaction at slot 3939841 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 134dbd9954128f93...#0

Event 40: Transaction at slot 3939841 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 134dbd9954128f93...#0

Epoch Advances: 16

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests proposal consistency and lifecycle management. Expected to succeed.

**Rules Tested:**
- GOV11: Proposals stored in submission order
- GOV12: Proposals require valid parent action ID
- GOV13: Proposal subtrees pruned on expiry/enactment

---

### Test: Subtrees are pruned when competing proposals are enacted

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposals/Consistency/Subtrees_are_pruned_when_competing_proposals_are_enacted`

**Rule:** GOV

**Action Type:** ProposalConsistency

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - PoolRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)

Event 7: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistrationDelegation

Event 8: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 9: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 10: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - UpdateCommittee (deposit: 123)
      Anchor: https://NI3ukxOr0AJmr-3DY.6fG4Tn3GO0.2Mjyu6.up9...
    - UpdateCommittee (deposit: 123)
      Anchor: https://l2J8VT5u39PF.com

Event 11: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on 35bae853a2e01641...#0

Event 12: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - StakePool votes No on 35bae853a2e01641...#0

Event 13: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - AuthCommitteeHot

Event 14: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 15: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://bn2OvANS.com

Event 16: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 17: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://3B1.com

Event 18: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 19: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://QmuExaaIHflvyEUF4kzaXIDARX9rg9mwzk6goF.com

Event 20: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 21: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://JWIW-QYqP8oRZqqS.com

Event 22: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 23: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://dFAl9qE.com

Event 24: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 25: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://fSHRXFE9LsIDxZwZ3G9ss43lwGgoFB4Ig5KQyhS...

Event 26: Transaction at slot 3896641 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on 10c5c90cb727e6f7...#0

Event 27: Transaction at slot 3896641 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 10c5c90cb727e6f7...#0

Epoch Advances: 5

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests proposal consistency and lifecycle management. Expected to succeed.

**Rules Tested:**
- GOV11: Proposals stored in submission order
- GOV12: Proposals require valid parent action ID
- GOV13: Proposal subtrees pruned on expiry/enactment

---

### Test: Subtrees are pruned when competing proposals are enacted over multiple rounds

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposals/Consistency/Subtrees_are_pruned_when_competing_proposals_are_enacted_over_multiple_rounds`

**Rule:** GOV

**Action Type:** ProposalConsistency

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://bNJssFtMhlRfnMB.com

Event 7: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 8: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://H-SqaqL6I4.com

Event 9: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 10: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://eTpWQRCEetGzrJLaeO9JK7WhfYLSTO32LW2dUi7...

Event 11: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 12: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://KDuK.dJ925t8O2nMx.LaNlUnSpxXyRJcbq4jYdN...

Event 13: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 14: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://gjB7qz-svONCn.com

Event 15: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 16: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://3PknZA-e.QeAZtpUWea74cOOErDdaGSV00plE4u...

Event 17: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 18: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://9BP8KaJ-2jzcLSbUCioKEzL2d3MwuWAI4Uw7VLj...

Event 19: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 20: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://5Ov9u.jQTyUQgSZJrbRLCwylK4ettaJO3XyvIoD...

Event 21: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on cf0e4a9a6ffce953...#0

Event 22: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on cf0e4a9a6ffce953...#0

Event 23: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on cf0e4a9a6ffce953...#0

Event 24: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on b514a7d4ea162e4d...#0

Event 25: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on b514a7d4ea162e4d...#0

Event 26: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on b514a7d4ea162e4d...#0

Event 27: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on e859f8d2cc6fc1f0...#0

Event 28: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on e859f8d2cc6fc1f0...#0

Event 29: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on e859f8d2cc6fc1f0...#0

Event 30: Transaction at slot 3888001 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 31: Transaction at slot 3888001 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://cHzrL5hWvVongi-bskv81Td1-VL.com

Event 32: Transaction at slot 3888001 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 33: Transaction at slot 3888001 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://a5-Oc9MkjXDW7k.3M05XHA.com

Event 34: Transaction at slot 3888001 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 35: Transaction at slot 3888001 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://i36sqwQo7tN2EWrOjPxgZ4EcJ3ENCG5pjBhLRql...

Event 36: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 37: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://RFvBGk.Y59CCfJEVDAVZ1Ycl5vmBRv2Ge7rcJ0U...

Event 38: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 39: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://Ow.com

Event 40: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 41: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://-3jEmSKi6meAFCrZnVOkjQUuzJRGYUMP.com

Event 42: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 43: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://Q2S5FVLaZIIL-SYZ5xlxUkc4nLdZN8puK5w8vr5...

Event 44: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 45: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://vRAM4YyKqS0gbq0Jh8IqoiIG32pWEGaOUo.com

Event 46: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on 62318b5d2af734a8...#0

Event 47: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 62318b5d2af734a8...#0

Event 48: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 62318b5d2af734a8...#0

Epoch Advances: 4

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests proposal consistency and lifecycle management. Expected to succeed.

**Rules Tested:**
- GOV11: Proposals stored in submission order
- GOV12: Proposals require valid parent action ID
- GOV13: Proposal subtrees pruned on expiry/enactment

---

### Test: Subtrees are pruned when proposals expire

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposals/Consistency/Subtrees_are_pruned_when_proposals_expire`

**Rule:** GOV

**Action Type:** ProposalConsistency

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://wnCFPAhvPKHuILapvTVDaBZAJ7f3.com

Event 4: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 5: Transaction at slot 3896641 (SUCCESS)

Event 6: Transaction at slot 3896641 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://aHKQKeNRE-pN3Xf1NpFZtHwepb.com

Event 7: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 8: Transaction at slot 3896641 (SUCCESS)

Event 9: Transaction at slot 3896641 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://jIY.j.com

Event 10: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 11: Transaction at slot 3896641 (SUCCESS)

Event 12: Transaction at slot 3896641 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://uCpxDOWKVBqOlKWVUY.com

Event 13: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 14: Transaction at slot 3896641 (SUCCESS)

Event 15: Transaction at slot 3896641 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://WUWOF3bVoN03Ge.com

Event 16: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 17: Transaction at slot 3896641 (SUCCESS)

Event 18: Transaction at slot 3896641 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://PbB2.com

Epoch Advances: 6

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests proposal consistency and lifecycle management. Expected to succeed.

**Rules Tested:**
- GOV11: Proposals stored in submission order
- GOV12: Proposals require valid parent action ID
- GOV13: Proposal subtrees pruned on expiry/enactment

---

### Test: Subtrees are pruned when proposals expire over multiple rounds

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposals/Consistency/Subtrees_are_pruned_when_proposals_expire_over_multiple_rounds`

**Rule:** GOV

**Action Type:** ProposalConsistency

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Event 4: Transaction at slot 3888001 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 5: Transaction at slot 3888001 (SUCCESS)

Event 6: Transaction at slot 3888001 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://0Y-C91JZBQ8P59rPxTK.com

Event 7: Transaction at slot 3888001 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 8: Transaction at slot 3888001 (SUCCESS)

Event 9: Transaction at slot 3888001 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://uJ5siUdfyidD937VtKh0waMxEPqyI2inRZ.5y.R...

Event 10: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 11: Transaction at slot 3892321 (SUCCESS)

Event 12: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://8JsvTD2xMk41H72YOog6jP-.com

Event 13: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 14: Transaction at slot 3892321 (SUCCESS)

Event 15: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://EtTS08sAKuj6zpK7fzl8OomBR6e5TKFL9BLZ.com

Event 16: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 17: Transaction at slot 3892321 (SUCCESS)

Event 18: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Xr2GgUWdLfeUQpXG9thtGXT8KGIdk0aBh3eyZ7.com

Event 19: Transaction at slot 3892321 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 20: Transaction at slot 3892321 (SUCCESS)

Event 21: Transaction at slot 3892321 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://Z.com

Event 22: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 23: Transaction at slot 3896641 (SUCCESS)

Event 24: Transaction at slot 3896641 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://y9uR.com

Event 25: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 26: Transaction at slot 3896641 (SUCCESS)

Event 27: Transaction at slot 3896641 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://QAdcl0AYFk0o46tJf51rR.com

Event 28: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 29: Transaction at slot 3896641 (SUCCESS)

Event 30: Transaction at slot 3896641 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://kJh7N3cc8NfE9vdL9.E7WaqdDkd..com

Event 31: Transaction at slot 3909601 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 32: Transaction at slot 3909601 (SUCCESS)

Event 33: Transaction at slot 3909601 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://VdLP5-Sscsl.com

Event 34: Transaction at slot 3909601 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 35: Transaction at slot 3909601 (SUCCESS)

Event 36: Transaction at slot 3909601 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://dnTh-rqp0Vojmr17RCEZ76dy5sWpQm5QdAwAaDD...

Event 37: Transaction at slot 3909601 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 38: Transaction at slot 3909601 (SUCCESS)

Event 39: Transaction at slot 3909601 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://raXZBq7bL-BCZ-q4rB3UqpI0fqGe9aI-VNVQ.com

Event 40: Transaction at slot 3909601 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 41: Transaction at slot 3909601 (SUCCESS)

Event 42: Transaction at slot 3909601 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://XE2z7-29zgf28EPlTH3YLePU1Vcws.com

Event 43: Transaction at slot 3913921 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 44: Transaction at slot 3913921 (SUCCESS)

Event 45: Transaction at slot 3913921 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://5HTCmi7PTLyp9.sd93du3.WG-CzM2ewNPjEUISf...

Event 46: Transaction at slot 3913921 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 47: Transaction at slot 3913921 (SUCCESS)

Event 48: Transaction at slot 3913921 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://bQwD.com

Event 49: Transaction at slot 3913921 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 50: Transaction at slot 3913921 (SUCCESS)

Event 51: Transaction at slot 3913921 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://tQ2.S44ISmEy9tKDrE39n0eytyLI.com

Event 52: Transaction at slot 3913921 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 53: Transaction at slot 3913921 (SUCCESS)

Event 54: Transaction at slot 3913921 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://rK1UlSLdpFRise36kSpwQDauhzkp.com

Epoch Advances: 13

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests proposal consistency and lifecycle management. Expected to succeed.

**Rules Tested:**
- GOV11: Proposals stored in submission order
- GOV12: Proposals require valid parent action ID
- GOV13: Proposal subtrees pruned on expiry/enactment

---

### Test: Votes from subsequent epochs are considered for ratification

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposals/Consistency/Votes_from_subsequent_epochs_are_considered_for_ratification`

**Rule:** GOV

**Action Type:** ProposalConsistency

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://bNJssFtMhlRfnMB.com

Event 7: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on fae8dfcae1a80999...#0

Event 8: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on fae8dfcae1a80999...#0

Event 9: Transaction at slot 3892321 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on fae8dfcae1a80999...#0

Epoch Advances: 4

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests proposal consistency and lifecycle management. Expected to succeed.

**Rules Tested:**
- GOV11: Proposals stored in submission order
- GOV12: Proposals require valid parent action ID
- GOV13: Proposal subtrees pruned on expiry/enactment

---

## Proposing and voting

### Test: Hardfork initiation

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposing_and_voting/Hardfork_initiation`

**Rule:** GOV

**Action Type:** Multiple/Unknown

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - HardForkInitiation (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 6: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 7: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 8: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - PoolRegistration

Event 9: Transaction at slot 3883681 (SUCCESS)

Event 10: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistrationDelegation

Event 11: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes Yes on 6964c2144b04f07d...#0

Event 12: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - StakePool votes No on 6964c2144b04f07d...#0

Event 13: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 6964c2144b04f07d...#0

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests complete governance workflow for Multiple/Unknown actions. Expected to succeed.

**Rules Tested:**
- GOV26: End-to-end proposal and voting lifecycle

---

### Test: Info action

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposing_and_voting/Info_action`

**Rule:** GOV

**Action Type:** Multiple/Unknown

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - InfoAction (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 6: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 7: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 8: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - PoolRegistration

Event 9: Transaction at slot 3883681 (SUCCESS)

Event 10: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistrationDelegation

Event 11: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on 0653cc66eb658ccf...#0

Event 12: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - StakePool votes No on 0653cc66eb658ccf...#0

Event 13: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 0653cc66eb658ccf...#0

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests complete governance workflow for Multiple/Unknown actions. Expected to succeed.

**Rules Tested:**
- GOV26: End-to-end proposal and voting lifecycle

---

### Test: NewConstitution

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposing_and_voting/NewConstitution`

**Rule:** GOV

**Action Type:** Multiple/Unknown

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://YmcuwygChpNVX5qnoZm3m2.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests complete governance workflow for Multiple/Unknown actions. Expected to succeed.

**Rules Tested:**
- GOV26: End-to-end proposal and voting lifecycle

---

### Test: NoConfidence

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposing_and_voting/NoConfidence`

**Rule:** GOV

**Action Type:** Multiple/Unknown

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NoConfidence (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests complete governance workflow for Multiple/Unknown actions. Expected to succeed.

**Rules Tested:**
- GOV26: End-to-end proposal and voting lifecycle

---

### Test: Parameter change

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposing_and_voting/Parameter_change`

**Rule:** GOV

**Action Type:** Multiple/Unknown

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://wnCFPAhvPKHuILapvTVDaBZAJ7f3.com

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 7: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 8: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 9: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - PoolRegistration

Event 10: Transaction at slot 3883681 (SUCCESS)

Event 11: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistrationDelegation

Event 12: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on a47fc84c73b9e7d9...#0

Event 13: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - StakePool votes No on a47fc84c73b9e7d9...#0

Event 14: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on a47fc84c73b9e7d9...#0

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests complete governance workflow for Multiple/Unknown actions. Expected to succeed.

**Rules Tested:**
- GOV26: End-to-end proposal and voting lifecycle

---

### Test: Treasury withdrawal

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposing_and_voting/Treasury_withdrawal`

**Rule:** GOV

**Action Type:** Multiple/Unknown

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - TreasuryWithdrawal (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests complete governance workflow for Multiple/Unknown actions. Expected to succeed.

**Rules Tested:**
- GOV26: End-to-end proposal and voting lifecycle

---

### Test: UpdateCommittee

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Proposing_and_voting/UpdateCommittee`

**Rule:** GOV

**Action Type:** Multiple/Unknown

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - UpdateCommittee (deposit: 123)
      Anchor: https://790eMzUN2fR2Ytiq.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests complete governance workflow for Multiple/Unknown actions. Expected to succeed.

**Rules Tested:**
- GOV26: End-to-end proposal and voting lifecycle

---

## Unknown CostModels

### Test: Are accepted

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Unknown_CostModels/Are_accepted`

**Rule:** GOV

**Action Type:** CostModels

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)

Event 7: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - ParameterChange (deposit: 123)
      Anchor: https://.sw8SUT7neqMvv2yoWamnvM1XX-Tkh02dEE.com

Event 8: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on a14df1c0066d6af3...#0

Event 9: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on a14df1c0066d6af3...#0

Event 10: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on a14df1c0066d6af3...#0

Epoch Advances: 2

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests handling of unknown cost model versions. Expected to succeed.

**Rules Tested:**
- GOV25: Unknown cost model versions are accepted

---

## Voting

### Test: CC cannot ratify if below threshold

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Voting/CC_cannot_ratify_if_below_threshold`

**Rule:** GOV

**Action Type:** Voting

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - PoolRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)

Event 7: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistrationDelegation

Event 8: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 9: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - UpdateCommittee (deposit: 123)
      Anchor: https://yrifsUab..com

Event 10: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on 15dfee657f6ef1d8...#0

Event 11: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - StakePool votes No on 15dfee657f6ef1d8...#0

Event 12: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - AuthCommitteeHot

Event 13: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - AuthCommitteeHot

Event 14: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 15: Transaction at slot 3896641 (SUCCESS)
  Proposal Procedures:
    - NewConstitution (deposit: 123)
      Anchor: https://1BKWYK9S1B8wuOsJk2lX6wuqH9l25bd0h0XcZfp...

Event 16: Transaction at slot 3896641 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes No on c3ab88dfc3e16799...#0

Event 17: Transaction at slot 3896641 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on c3ab88dfc3e16799...#0

Event 18: Transaction at slot 3896641 (SUCCESS)
  Certificates:
    - ResignCommitteeCold

Event 19: Transaction at slot 3896641 (SUCCESS)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on c3ab88dfc3e16799...#0

Epoch Advances: 8

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests voting mechanics and validation rules. Expected to succeed.

**Rules Tested:**
- GOV8: CC members cannot vote on NoConfidence/UpdateCommittee

---

### Test: DRep votes are removed

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Voting/DRep_votes_are_removed`

**Rule:** GOV

**Action Type:** Voting

**Expected:** Success

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - InfoAction (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Voting Procedures:
    - DRep(KeyHash) votes Yes on 0653cc66eb658ccf...#0

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepDeregistration

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests voting mechanics and validation rules. Expected to succeed.

**Rules Tested:**
- GOV10: DRep votes removed when DRep deregisters

---

### Test: VotersDoNotExist

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Voting/VotersDoNotExist`

**Rule:** GOV

**Action Type:** Voting

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - HardForkInitiation (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Event 3: Transaction at slot 3883681 (FAILURE)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on bf181398270f2d94...#0

Event 4: Transaction at slot 3883681 (FAILURE)
  Voting Procedures:
    - StakePool votes No on bf181398270f2d94...#0

Event 5: Transaction at slot 3883681 (FAILURE)
  Voting Procedures:
    - DRep(KeyHash) votes No on bf181398270f2d94...#0

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests voting mechanics and validation rules. Expected to fail validation.

**Rules Tested:**
- GOV7: Voter must be registered (DRep/SPO/CC)

---

### Test: committee member can not vote on NoConfidence action

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Voting/committee_member_can_not_vote_on_NoConfidence_action`

**Rule:** GOV

**Action Type:** Voting

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - NoConfidence (deposit: 123)
      Anchor: https://2xdDAEnhiZ9M3hdoQltQ4mHGV2tWku7X6STXHU0...

Event 4: Transaction at slot 3883681 (FAILURE)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes Yes on fa95e7f5d76e3139...#0

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests voting mechanics and validation rules. Expected to fail validation.

---

### Test: committee member can not vote on UpdateCommittee action

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Voting/committee_member_can_not_vote_on_UpdateCommittee_action`

**Rule:** GOV

**Action Type:** Voting

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - UpdateCommittee (deposit: 123)
      Anchor: https://TY.com

Event 4: Transaction at slot 3883681 (FAILURE)
  Voting Procedures:
    - ConstitutionalCommittee(KeyHash) votes No on 7cbc3446d472ed87...#0

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests voting mechanics and validation rules. Expected to fail validation.

---

### Test: committee member mixed with other voters can not vote on UpdateCommittee action

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Voting/committee_member_mixed_with_other_voters_can_not_vote_on_UpdateCommittee_action`

**Rule:** GOV

**Action Type:** Voting

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - AuthCommitteeHot
    - AuthCommitteeHot

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 7: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - UpdateCommittee (deposit: 123)
      Anchor: https://1BmZgzBv6FZSPa.fHGy39zWPnL6jZ1qWrUA4Ntu...

Event 8: Transaction at slot 3883681 (FAILURE)
  Voting Procedures:
    - DRep(KeyHash) votes No on 851f0db3afaf37df...#0
    - DRep(KeyHash) votes No on 851f0db3afaf37df...#0
    - ConstitutionalCommittee(KeyHash) votes Yes on 851f0db3afaf37df...#0

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests voting mechanics and validation rules. Expected to fail validation.

---

### Test: expired gov-actions

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Voting/expired_gov-actions`

**Rule:** GOV

**Action Type:** Voting

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 5: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - InfoAction (deposit: 123)
      Anchor: https://RQOlpzSSbq8X6fh6G2d0oR.com

Event 6: Transaction at slot 3896641 (FAILURE)
  Voting Procedures:
    - DRep(KeyHash) votes No on 6ba5ce06c4ab06a2...#0

Epoch Advances: 3

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests voting mechanics and validation rules. Expected to fail validation.

**Rules Tested:**
- GOV6: Cannot vote on expired governance action

---

### Test: non-existent gov-actions

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Voting/non-existent_gov-actions`

**Rule:** GOV

**Action Type:** Voting

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - DRepRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - Registration

Event 3: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - VoteDelegation

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 5: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - InfoAction (deposit: 123)
      Anchor: https://RQOlpzSSbq8X6fh6G2d0oR.com

Event 6: Transaction at slot 3883681 (FAILURE)
  Voting Procedures:
    - DRep(KeyHash) votes No on 6ba5ce06c4ab06a2...#99

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests voting mechanics and validation rules. Expected to fail validation.

**Rules Tested:**
- GOV5: Cannot vote on non-existent governance action

---

## Withdrawals

### Test: Fails for empty withdrawals

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Withdrawals/Fails_for_empty_withdrawals`

**Rule:** GOV

**Action Type:** TreasuryWithdrawal

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - TreasuryWithdrawal (deposit: 123)
      Anchor: https://Gqew7xrua-evVzfAnJsHuGmZex91.com

Event 4: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 5: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 6: Transaction at slot 3883681 (SUCCESS)

Event 7: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - TreasuryWithdrawal (deposit: 123)
      Anchor: https://EN1cgK1DF.TWWzt6y6EfS6ZLk2Pbf7fSyuCpDpI...

Event 8: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 9: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 10: Transaction at slot 3883681 (SUCCESS)

Event 11: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - TreasuryWithdrawal (deposit: 123)
      Anchor: https://B8AH8YXVW-kDSo28U6iCBVjMfg-a4YgpeYlgkGx...

Event 12: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 13: Transaction at slot 3883681 (SUCCESS)

Event 14: Transaction at slot 3883681 (SUCCESS)
  Proposal Procedures:
    - TreasuryWithdrawal (deposit: 123)
      Anchor: https://kZzFCJQCdyHU-e3ju.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests treasury withdrawal proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV20: Treasury withdrawal addresses must exist
- GOV21: Treasury withdrawals cannot be empty
- GOV22: Network ID must match in withdrawal addresses

---

### Test: Fails predicate when treasury withdrawal has nonexistent return address

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Withdrawals/Fails_predicate_when_treasury_withdrawal_has_nonexistent_return_address`

**Rule:** GOV

**Action Type:** TreasuryWithdrawal

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 3: Transaction at slot 3883681 (SUCCESS)

Event 4: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - TreasuryWithdrawal (deposit: 123)
      Anchor: https://h6pHXCdlD-tG7Nx3Ak3h2.zn3ti.6OdK3Q3vnmF...

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests treasury withdrawal proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV20: Treasury withdrawal addresses must exist
- GOV21: Treasury withdrawals cannot be empty
- GOV22: Network ID must match in withdrawal addresses

---

### Test: Fails with invalid network ID in withdrawal addresses

**File:** `conformance/testdata/eras/conway/impl/dump/Conway/Imp/ConwayImpSpec_-_Version_10/GOV/Withdrawals/Fails_with_invalid_network_ID_in_withdrawal_addresses`

**Rule:** GOV

**Action Type:** TreasuryWithdrawal

**Expected:** Failure

#### State Change Diagram

```
Initial State:
  Epoch: 899
  Proposals: 0
  Committee Members: 0

Event 1: Transaction at slot 3883681 (SUCCESS)
  Certificates:
    - StakeRegistration

Event 2: Transaction at slot 3883681 (SUCCESS)

Event 3: Transaction at slot 3883681 (FAILURE)
  Proposal Procedures:
    - TreasuryWithdrawal (deposit: 123)
      Anchor: https://790eMzUN2fR2Ytiq.com

Final State:
  Proposals: 0
  Committee Members: 0
```

#### What This Tests

Tests treasury withdrawal proposal validation. Expected to fail validation.

**Rules Tested:**
- GOV20: Treasury withdrawal addresses must exist
- GOV21: Treasury withdrawals cannot be empty
- GOV22: Network ID must match in withdrawal addresses

---


