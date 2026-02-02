
# Conway ENACT Conformance Test Vectors

This directory contains conformance test vectors for the Conway era ENACT (enactment) rule, which handles the execution of ratified governance proposals at epoch boundaries.

## Overview

The ENACT rule governs how ratified governance proposals are enacted when an epoch boundary is crossed. This includes:
- Constitution changes
- Committee updates (adding/removing members, re-elections)
- Protocol parameter changes
- Hard fork initiations
- Treasury withdrawals
- Motion of no confidence

## Test Vector Format

Each test vector is a CBOR-encoded array with 5 elements:
```
[config, initial_state, final_state, events, title]
```

- **config**: Network/protocol configuration (13-element array)
- **initial_state**: NewEpochState before events (7-element array)
- **final_state**: NewEpochState after events (7-element array)
- **events**: Array of events (transactions, slot ticks, epoch passes)
- **title**: String identifying the test

## Event Types

- `[0, tx_cbor, success, slot]` - Transaction event
- `[1, slot]` - PassTick event (slot advancement)
- `[2, epoch_delta]` - PassEpoch event (epoch boundary)

---

## Test: Constitution

**File:** `Constitution`
**Rule:** ENACT
**Action Type:** NewConstitution (type 5)
**Expected:** Constitution anchor and policy hash updated in governance state

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [NewConstitution proposal]
+-- Treasury: [unchanged]
+-- Committee: [unchanged]
+-- Constitution:
|   +-- URL: "https://cardano-constitution.crypto"
|   +-- Anchor Hash: [current hash]
|   +-- Policy Hash: [current or null]
+-- Protocol Params: [unchanged]
+-- Enacted Actions: [none for this purpose]

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: NewConstitution (5)
|   +-- New Anchor URL: [new URL]
|   +-- New Anchor Hash: [new hash]
|   +-- New Policy Hash: [new hash or null]
+-- Result: Constitution updated

Final State:
+-- Treasury: [unchanged]
+-- Committee: [unchanged]
+-- Constitution:
|   +-- URL: [new URL]
|   +-- Anchor Hash: [new hash]
|   +-- Policy Hash: [new policy hash]
+-- Protocol Params: [unchanged]
+-- Enacted Actions: [NewConstitution recorded as root]
```

### What This Tests
Validates that a ratified NewConstitution governance action correctly updates the constitution's anchor (URL + hash) and optional policy script hash when enacted at an epoch boundary.

---

## Test: NoConfidence

**File:** `NoConfidence`
**Rule:** ENACT
**Action Type:** MotionOfNoConfidence (type 3)
**Expected:** Constitutional Committee dissolved

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [NoConfidence proposal]
+-- Treasury: [unchanged]
+-- Committee:
|   +-- Members: {coldKey1 -> expiryEpoch1, coldKey2 -> expiryEpoch2, ...}
|   +-- Threshold: [current threshold]
+-- Constitution: [unchanged]
+-- Protocol Params: [unchanged]
+-- Hot Key Authorizations: {coldKey -> hotKey, ...}

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: MotionOfNoConfidence (3)
|   +-- Effect: Remove all committee members
+-- Result: Committee dissolved

Final State:
+-- Treasury: [unchanged]
+-- Committee:
|   +-- Members: {} (empty)
|   +-- Threshold: [preserved but ineffective]
+-- Constitution: [unchanged]
+-- Protocol Params: [unchanged]
+-- Hot Key Authorizations: [cleared]
+-- Enacted Actions: [NoConfidence recorded as CC root]
```

### What This Tests
Validates that a MotionOfNoConfidence action, when enacted, removes all constitutional committee members and clears their hot key authorizations, effectively dissolving the committee.

---

## Test: HardForkInitiation

**File:** `HardForkInitiation`
**Rule:** ENACT
**Action Type:** HardForkInitiation (type 1)
**Expected:** Protocol version updated in next epoch

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [HardForkInitiation proposal]
+-- Protocol Version: {major: 10, minor: 0}
+-- Treasury: [unchanged]
+-- Committee: [unchanged]
+-- Constitution: [unchanged]
+-- Required Votes: CC + DReps + SPOs

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: HardForkInitiation (1)
|   +-- Target Version: {major: 11, minor: 0}
|   +-- CC Votes: [yes votes]
|   +-- DRep Votes: [yes votes]
|   +-- SPO Votes: [yes votes]
+-- Result: Protocol version scheduled for update

Final State:
+-- Protocol Version: {major: 11, minor: 0} (or scheduled)
+-- Treasury: [unchanged]
+-- Committee: [unchanged]
+-- Constitution: [unchanged]
+-- Enacted Actions: [HardFork recorded as root]
```

### What This Tests
Validates the hard fork initiation process requiring votes from Constitutional Committee, DReps, and Stake Pool Operators. Tests that the protocol version is correctly updated when all voting thresholds are met.

---

## Test: HardForkInitiation_without_DRep_voting

**File:** `HardForkInitiation_without_DRep_voting`
**Rule:** ENACT
**Action Type:** HardForkInitiation (type 1)
**Expected:** Hard fork enacted without DRep participation when DReps abstain or parameter allows

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [HardForkInitiation proposal]
+-- Protocol Version: {major: 10, minor: 0}
+-- DRep State: [no active DReps or all abstaining]
+-- Required Votes: CC + SPOs (DReps not required)

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: HardForkInitiation (1)
|   +-- Target Version: {major: 11, minor: 0}
|   +-- CC Votes: [sufficient yes votes]
|   +-- DRep Votes: [none/abstain]
|   +-- SPO Votes: [sufficient yes votes]
+-- Result: Protocol version updated despite no DRep votes

Final State:
+-- Protocol Version: {major: 11, minor: 0}
+-- Enacted Actions: [HardFork recorded as root]
```

### What This Tests
Validates that hard fork initiation can proceed without DRep voting when the governance parameters allow it (e.g., when dRepVotingThresholds.hardForkInitiation is set to allow abstention or when no DReps are registered).

---

## Test: futurePParams

**File:** `futurePParams`
**Rule:** ENACT
**Action Type:** ParameterChange (type 0)
**Expected:** Protocol parameters staged for future epoch

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [ParameterChange proposal]
+-- Current PParams: [current values]
+-- Future PParams: [none or previous]
+-- Treasury: [unchanged]

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: ParameterChange (0)
|   +-- Parameter Updates: {key: newValue, ...}
|   +-- Staged For: Next epoch or N epochs
+-- Result: Parameters staged in futurePParams

Final State:
+-- Current PParams: [may be unchanged this epoch]
+-- Future PParams: [contains staged changes]
+-- Enacted Actions: [ParameterChange recorded as root]
```

### What This Tests
Validates that parameter changes are correctly staged in the futurePParams field rather than immediately applied, allowing for a grace period before parameters take effect.

---

## Committee Enactment Tests

### Test: CC_re-election

**File:** `Committee_enactment/CC_re-election`
**Rule:** ENACT
**Action Type:** UpdateCommittee (type 4)
**Expected:** Existing committee members re-elected with new terms

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [UpdateCommittee proposal]
+-- Committee:
|   +-- Members: {coldKey1 -> epoch100, coldKey2 -> epoch100}
+-- Hot Key Authorizations: {coldKey1 -> hotKey1, coldKey2 -> hotKey2}

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: UpdateCommittee (4)
|   +-- Members to Add: {coldKey1 -> epoch200, coldKey2 -> epoch200}
|   +-- Members to Remove: []
|   +-- New Threshold: [same or updated]
+-- Result: Terms extended

Final State:
+-- Committee:
|   +-- Members: {coldKey1 -> epoch200, coldKey2 -> epoch200}
+-- Hot Key Authorizations: [preserved]
+-- Enacted Actions: [UpdateCommittee recorded as CC root]
```

### What This Tests
Validates that existing committee members can be re-elected with new term limits while preserving their hot key authorizations. This is essential for continuity of the constitutional committee.

---

### Test: Enact_UpdateCommittee_with_lengthy_lifetime

**File:** `Committee_enactment/Enact_UpdateCommitee_with_lengthy_lifetime`
**Rule:** ENACT
**Action Type:** UpdateCommittee (type 4)
**Expected:** Committee members added with long-term expiry epochs

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [UpdateCommittee with long terms]
+-- Committee: {existing members}

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: UpdateCommittee (4)
|   +-- Members to Add: {coldKey -> epoch_far_future}
|   +-- Term Length: [very long, e.g., 1000+ epochs]
+-- Result: Long-term committee members added

Final State:
+-- Committee:
|   +-- Members: {..., newColdKey -> epoch_far_future}
+-- Enacted Actions: [UpdateCommittee recorded]
```

### What This Tests
Validates that the system correctly handles committee member additions with very long term limits, ensuring no overflow issues with epoch calculations and that lengthy terms are properly stored.

---

### Test: Removing_CC_with_UpdateCommittee/Non_registered

**File:** `Committee_enactment/Removing_CC_with_UpdateCommittee/Non_registered`
**Rule:** ENACT
**Action Type:** UpdateCommittee (type 4)
**Expected:** Non-registered cold keys can be removed from committee

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [UpdateCommittee to remove member]
+-- Committee:
|   +-- Members: {coldKey1 -> expiry, coldKey2 -> expiry}
+-- Cold Key Registration: coldKey1 not registered as credential

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: UpdateCommittee (4)
|   +-- Members to Remove: [coldKey1]
|   +-- Note: coldKey1 was never registered
+-- Result: Member removed regardless of registration

Final State:
+-- Committee:
|   +-- Members: {coldKey2 -> expiry}
+-- Hot Key Authorizations: [coldKey1's authorization removed if any]
```

### What This Tests
Validates that committee members can be removed via UpdateCommittee even if their cold key was never formally registered as a credential. The removal should succeed based on committee membership, not credential registration status.

---

### Test: Removing_CC_with_UpdateCommittee/Registered

**File:** `Committee_enactment/Removing_CC_with_UpdateCommittee/Registered`
**Rule:** ENACT
**Action Type:** UpdateCommittee (type 4)
**Expected:** Registered cold keys properly removed from committee

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [UpdateCommittee to remove member]
+-- Committee:
|   +-- Members: {coldKey1 -> expiry, coldKey2 -> expiry}
+-- Cold Key Registration: coldKey1 is registered credential
+-- Hot Key Authorization: coldKey1 -> hotKey1

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: UpdateCommittee (4)
|   +-- Members to Remove: [coldKey1]
+-- Result: Registered member removed

Final State:
+-- Committee:
|   +-- Members: {coldKey2 -> expiry}
+-- Hot Key Authorizations: {coldKey1 -> hotKey1 removed}
+-- Credential Registration: [may be preserved]
```

### What This Tests
Validates that removing a registered committee member properly cleans up the committee membership and associated hot key authorizations, while potentially preserving the underlying credential registration.

---

## Treasury Withdrawal Tests

### Test: Modify_EnactState_as_expected

**File:** `Treasury_withdrawals/Modify_EnactState_as_expected`
**Rule:** ENACT
**Action Type:** TreasuryWithdrawals (type 2)
**Expected:** Treasury balance reduced, reward accounts credited

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [TreasuryWithdrawals proposal]
+-- Treasury Balance: 1,000,000 ADA
+-- Reward Accounts:
|   +-- stakeKey1: 0 ADA
|   +-- stakeKey2: 0 ADA
+-- Withdrawal Requests:
|   +-- stakeKey1: 100,000 ADA
|   +-- stakeKey2: 50,000 ADA

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: TreasuryWithdrawals (2)
|   +-- Total Withdrawal: 150,000 ADA
+-- Verification: Total <= Treasury Balance
+-- Result: Withdrawals processed

Final State:
+-- Treasury Balance: 850,000 ADA
+-- Reward Accounts:
|   +-- stakeKey1: 100,000 ADA
|   +-- stakeKey2: 50,000 ADA
+-- Enacted Actions: [TreasuryWithdrawals recorded]
```

### What This Tests
Validates the basic treasury withdrawal flow where the enact state is properly modified to reflect the treasury reduction and corresponding reward account credits.

---

### Test: Withdrawals_exceeding_maxBound_Word64_submitted_in_a_single_proposal

**File:** `Treasury_withdrawals/Withdrawals_exceeding_maxBound_Word64_submitted_in_a_single_proposal`
**Rule:** ENACT
**Action Type:** TreasuryWithdrawals (type 2)
**Expected:** Proposal with sum exceeding Word64 max handled correctly

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [TreasuryWithdrawals with overflow]
+-- Treasury Balance: [large amount]
+-- Withdrawal Requests:
|   +-- Sum of all requests > 2^64 - 1 (Word64 max)

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: TreasuryWithdrawals (2)
|   +-- Total Sum: > maxBound::Word64
+-- Overflow Check: Required
+-- Result: Proposal rejected or truncated

Final State:
+-- Treasury Balance: [unchanged or partially processed]
+-- Reward Accounts: [unchanged or partially credited]
+-- Error Handling: Overflow prevented
```

### What This Tests
Validates that the system correctly handles edge cases where the sum of withdrawal amounts in a single proposal would overflow a 64-bit unsigned integer. The implementation must either reject such proposals or handle the arithmetic safely.

---

### Test: Withdrawals_exceeding_treasury_submitted_in_a_single_proposal

**File:** `Treasury_withdrawals/Withdrawals_exceeding_treasury_submitted_in_a_single_proposal`
**Rule:** ENACT
**Action Type:** TreasuryWithdrawals (type 2)
**Expected:** Withdrawal request exceeding treasury balance handled

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [TreasuryWithdrawals proposal]
+-- Treasury Balance: 100,000 ADA
+-- Withdrawal Requests:
|   +-- stakeKey1: 150,000 ADA (exceeds treasury)

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Action Type: TreasuryWithdrawals (2)
|   +-- Requested Amount: 150,000 ADA
|   +-- Available: 100,000 ADA
+-- Validation: Request > Treasury
+-- Result: Withdrawal skipped/rejected

Final State:
+-- Treasury Balance: 100,000 ADA (unchanged)
+-- Reward Accounts: [unchanged]
+-- Proposal Status: Not enacted (insufficient funds)
```

### What This Tests
Validates that withdrawal proposals requesting more than the available treasury balance are properly rejected during enactment, preventing the treasury from going negative.

---

### Test: Withdrawals_exceeding_treasury_submitted_in_several_proposals_within_the_same_epoch

**File:** `Treasury_withdrawals/Withdrawals_exceeding_treasury_submitted_in_several_proposals_within_the_same_epoch`
**Rule:** ENACT
**Action Type:** TreasuryWithdrawals (type 2)
**Expected:** Multiple proposals competing for limited treasury handled

### State Change Diagram

```
Initial State:
+-- Ratified Actions: [Multiple TreasuryWithdrawals proposals]
|   +-- Proposal A: 60,000 ADA
|   +-- Proposal B: 60,000 ADA
|   +-- Proposal C: 60,000 ADA
+-- Treasury Balance: 100,000 ADA
+-- Total Requested: 180,000 ADA

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- Process in submission order
|   +-- Proposal A: 60,000 ADA - ENACTED (40,000 remaining)
|   +-- Proposal B: 60,000 ADA - REJECTED (insufficient: 40,000 < 60,000)
|   +-- Proposal C: 60,000 ADA - REJECTED (insufficient)
+-- Result: Only first proposal enacted

Final State:
+-- Treasury Balance: 40,000 ADA
+-- Enacted: Proposal A only
+-- Not Enacted: Proposals B, C (insufficient funds)
```

### What This Tests
Validates the ordering and resource competition behavior when multiple treasury withdrawal proposals are ratified in the same epoch but their combined total exceeds available funds. Earlier proposals (by submission order) should be processed first.

---

## Competing Proposals Tests

### Test: higher_action_priority_wins

**File:** `Competing_proposals/higher_action_priority_wins`
**Rule:** ENACT
**Action Type:** Multiple types with different priorities
**Expected:** Higher priority actions enacted before lower priority

### State Change Diagram

```
Initial State:
+-- Ratified Actions:
|   +-- UpdateCommittee (priority: 4)
|   +-- ParameterChange (priority: 0)
|   +-- TreasuryWithdrawals (priority: 2)
+-- All actions target same governance path

Event: Epoch Boundary (PassEpoch)
+-- Enactment Order by Priority:
|   1. ParameterChange (0) - lowest number = highest priority
|   2. TreasuryWithdrawals (2)
|   3. UpdateCommittee (4)
+-- Conflicting Actions: Later ones invalidated
+-- Result: Highest priority action wins

Final State:
+-- Enacted: ParameterChange
+-- Discarded: Lower priority conflicting actions
+-- Reason: Action priority determines enactment order
```

### What This Tests
Validates that when multiple governance actions are ratified that would conflict with each other, the action with the higher priority (lower type number) takes precedence and is enacted first. Actions are prioritized as:
1. ParameterChange (0)
2. HardForkInitiation (1)
3. TreasuryWithdrawals (2)
4. NoConfidence (3)
5. UpdateCommittee (4)
6. NewConstitution (5)
7. InfoAction (6)

---

### Test: only_the_first_action_of_a_transaction_gets_enacted

**File:** `Competing_proposals/only_the_first_action_of_a_transaction_gets_enacted`
**Rule:** ENACT
**Action Type:** Multiple proposals from same transaction
**Expected:** Only first proposal index from transaction enacted

### State Change Diagram

```
Initial State:
+-- Transaction TX1 contains:
|   +-- Proposal Index 0: UpdateCommittee (action A)
|   +-- Proposal Index 1: UpdateCommittee (action B)
|   +-- Proposal Index 2: UpdateCommittee (action C)
+-- All ratified in same epoch
+-- All are same type (same governance purpose)

Event: Epoch Boundary (PassEpoch)
+-- Actions to Enact:
|   +-- TX1#0 (action A) - First index
|   +-- TX1#1 (action B) - Higher index
|   +-- TX1#2 (action C) - Higher index
+-- Rule: Only first index per transaction per purpose
+-- Result: Only TX1#0 enacted

Final State:
+-- Enacted: Action A (TX1#0)
+-- Discarded: Actions B, C (TX1#1, TX1#2)
+-- Enacted Actions Root: Points to TX1#0
```

### What This Tests
Validates that when a single transaction submits multiple governance proposals of the same type, only the first proposal (index 0) is considered for enactment. This prevents a single transaction from monopolizing governance action slots.

---

### Test: proposals_of_same_priority_are_enacted_in_order_of_submission

**File:** `Competing_proposals/proposals_of_same_priority_are_enacted_in_order_of_submission`
**Rule:** ENACT
**Action Type:** Multiple same-type proposals
**Expected:** Proposals enacted in submission order (FIFO)

### State Change Diagram

```
Initial State:
+-- Ratified Actions (same type, e.g., TreasuryWithdrawals):
|   +-- Proposal A: submitted at slot 100
|   +-- Proposal B: submitted at slot 200
|   +-- Proposal C: submitted at slot 300
+-- All have same priority (type 2)

Event: Epoch Boundary (PassEpoch)
+-- Enactment Order:
|   1. Proposal A (slot 100) - submitted first
|   2. Proposal B (slot 200) - submitted second
|   3. Proposal C (slot 300) - submitted third
+-- Processing: Sequential, respecting submission order
+-- Result: FIFO order maintained

Final State:
+-- Processing Order: A -> B -> C
+-- Each proposal's effect applied in submission order
+-- Root Action: Updates to most recently enacted
```

### What This Tests
Validates that when multiple proposals of the same type (and thus same priority) are ratified, they are enacted in the order they were originally submitted to the chain (first-in-first-out). This ensures fairness and predictability in governance.

---

## Enactment Priority Reference

The ENACT rule processes actions by type priority (lower number = higher priority):

| Priority | Type Value | Action Type |
|----------|-----------|-------------|
| 1 | 0 | ParameterChange |
| 2 | 1 | HardForkInitiation |
| 3 | 2 | TreasuryWithdrawals |
| 4 | 3 | MotionOfNoConfidence |
| 5 | 4 | UpdateCommittee |
| 6 | 5 | NewConstitution |
| 7 | 6 | InfoAction |

Within the same priority, actions are processed in submission order.

## Key Enactment Rules

1. **Epoch Boundary Processing**: Actions are only enacted at epoch boundaries (PassEpoch events)
2. **Priority Ordering**: Lower action type numbers have higher priority
3. **Submission Order**: Same-priority actions processed in submission order
4. **Single Action Per Purpose**: Only one action per governance purpose enacted per epoch
5. **Resource Constraints**: Treasury withdrawals limited by available balance
6. **Parent Validation**: Actions must reference valid parent action IDs
7. **Root Tracking**: Each purpose maintains a root pointing to last enacted action

## State Components Modified by ENACT

- **Committee State**: Members map, hot key authorizations
- **Constitution**: Anchor (URL + hash), policy script hash
- **Protocol Parameters**: Current, future, and proposed pparams
- **Treasury**: Balance modifications from withdrawals
- **Reward Accounts**: Credits from treasury withdrawals
- **Enacted Action Roots**: Tracking last enacted action per purpose

