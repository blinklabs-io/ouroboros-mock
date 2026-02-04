
# Conway RATIFY Conformance Test Vectors

This directory contains conformance test vectors for the Conway-era RATIFY transition rule. These tests validate the governance proposal ratification logic in the Cardano ledger.

## Overview

The RATIFY rule determines whether governance proposals receive enough votes to be ratified. Ratified proposals are then enacted at the next epoch boundary. The test vectors in this directory cover various aspects of vote counting, threshold calculations, and special cases.

## Test Vector Format

Each test vector is a CBOR binary file containing:
```
[0] config:        array[13]  - Network/protocol configuration
[1] initial_state: array[7]   - NewEpochState before events
[2] final_state:   array[7]   - NewEpochState after events
[3] events:        array[N]   - Transaction/epoch events
[4] title:         string     - Test name/path
```

### Event Types
- `[0, tx_cbor, success, slot]` - Transaction event
- `[1, slot]` - PassTick (slot advancement)
- `[2, epoch_delta]` - PassEpoch (epoch boundary, triggers ratification)

---

## Test Categories

### 1. Committee Members Maximum Term Length

**Directory:** `Committee_members_can_serve_full_\`CommitteeMaxTermLength\``

| File | Title | Description |
|------|-------|-------------|
| `maxTermLength_=_0` | maxTermLength = 0 | Tests behavior when committee max term length is set to 0 epochs |
| `maxTermLength_=_1` | maxTermLength = 1 | Tests behavior when committee max term length is set to 1 epoch |

```
Initial State:
+-- Committee: Members with various expiry epochs
+-- Parameter: CommitteeMaxTermLength = 0 or 1
+-- Proposals: May contain UpdateCommittee proposals

Event: PassEpoch
+-- Check: Committee member terms are validated against max length
+-- Result: Members exceeding max term are considered expired

Final State:
+-- Committee quorum adjusted for expired members
```

**What This Tests:** Validates that committee members cannot serve beyond the configured maximum term length, even if their individual expiry epoch is higher.

---

### 2. CommitteeMinSize Effects on In-Flight Proposals

**Directory:** `CommitteeMinSize_affects_in-flight_proposals`

| File | Title | Description |
|------|-------|-------------|
| `TreasuryWithdrawal_fails_to_ratify_due_to_an_increase_in_CommitteeMinSize` | TreasuryWithdrawal fails due to CommitteeMinSize increase | A proposal that would otherwise pass fails because committee is below new minimum size |
| `TreasuryWithdrawal_ratifies_due_to_a_decrease_in_CommitteeMinSize` | TreasuryWithdrawal ratifies due to CommitteeMinSize decrease | A proposal that was blocked now passes because minimum size requirement is lowered |

```
Initial State:
+-- Proposals: TreasuryWithdrawal with sufficient votes
+-- Committee: N active members
+-- Parameter: CommitteeMinSize = M

Event: ParameterChange enacted (modifies CommitteeMinSize)
Event: PassEpoch

Vote Counting:
+-- Committee Size: N members
+-- MinSize Check: N >= new_CommitteeMinSize?
+-- If N < MinSize: Committee cannot vote, ratification may fail
+-- If N >= MinSize: Normal threshold applies

Final State:
+-- Proposal ratified or remains pending based on new parameter
```

**What This Tests:** Validates that changes to CommitteeMinSize apply immediately to all pending proposals, not just new ones.

---

### 3. Counting of SPO Votes

**Directory:** `Counting_of_SPO_votes`

| File | Title | Description |
|------|-------|-------------|
| `HardForkInitiation` | Counting of SPO votes/HardForkInitiation | Tests SPO vote counting for hard fork proposals |

```
Initial State:
+-- Proposals: HardForkInitiation (requires SPO votes)
+-- Stake Pools: Multiple pools with varying stake
+-- Votes:
    +-- SPO Votes: Pool operators voting Yes/No/Abstain

Event: PassEpoch

Vote Counting:
+-- SPO Vote Weight: Based on pool stake
+-- Threshold: SPO threshold for HardForkInitiation from protocol parameters
+-- Calculation: sum(yes_stake) / sum(total_stake_of_voters) >= threshold

Final State:
+-- HardFork ratified if SPO threshold met
```

**What This Tests:** Validates that SPO votes are weighted by stake and correctly counted against the SPO voting threshold for hard fork initiations.

---

### 4. Delaying Actions

**Directory:** `Delaying_actions`

#### 4.1 Delaying All Actions

| File | Title | Description |
|------|-------|-------------|
| `A_delaying_action_delays_all_other_actions_even_when_all_of_them_may_be_ratified_in_the_same_epoch` | Delaying action delays all others | Tests that certain action types delay enactment of all other actions |

```
Initial State:
+-- Proposals: Multiple proposals of different types
+-- Votes: All have sufficient votes to ratify

Event: PassEpoch

Enactment Order:
+-- Delaying actions (HardFork, NoConfidence) are enacted first
+-- Other actions are delayed to next epoch
+-- Even if ratified in same epoch, non-delaying actions wait

Final State:
+-- Only delaying action enacted this epoch
+-- Other proposals remain ratified but not enacted
```

**What This Tests:** Validates the CIP-1694 rule that certain governance actions delay the enactment of all other actions.

#### 4.2 Parent-Child Delays

| File | Title | Description |
|------|-------|-------------|
| `A_delaying_action_delays_its_child_even_when_both_ere_proposed_and_ratified_in_the_same_epoch` | Parent delays child enactment | A child proposal waits for its parent even when both ratify together |

```
Initial State:
+-- Parent Proposal: HardFork with sufficient votes
+-- Child Proposal: References parent as PrevGovId, sufficient votes

Event: PassEpoch

Enactment Sequence:
+-- Epoch N: Parent ratified, Child ratified
+-- Epoch N+1: Parent enacted, Child now eligible (root updated)
+-- Epoch N+2: Child enacted

Final State:
+-- Parent enacted
+-- Child ratified but waiting for parent enactment
```

**What This Tests:** Validates that proposals in the same lineage must be enacted in order, with at least one epoch between each.

#### 4.3 Expiration After Delay

**Subdirectory:** `An_action_expires_when_delayed_enough_even_after_being_ratified`

| File | Title | Description |
|------|-------|-------------|
| `Same_lineage` | Expiration in same lineage | Proposal expires while waiting for parent in same lineage |
| `Other_lineage` | Expiration due to other lineage | Proposal expires while waiting for unrelated delaying action |
| `proposals_to_update_the_committee_get_delayed_if_the_expiration_exceeds_the_max_term` | Committee update delayed by max term | UpdateCommittee delayed when proposed expiry exceeds max term |

```
Initial State:
+-- Proposal: Ratified with ExpiresAfter = epoch N
+-- Blocking Action: Either parent or delaying action pending

Events: Multiple PassEpoch

Timeline:
+-- Epoch X: Proposal ratified
+-- Epochs X+1 to N: Blocked by delaying actions or parent chain
+-- Epoch N+1: Proposal expires (never enacted)

Final State:
+-- Proposal removed from proposals (expired)
+-- Never added to enacted list
```

**What This Tests:** Validates that ratified proposals can still expire if delayed too long, preventing stale proposals from eventually enacting.

---

### 5. Expired and Resigned Committee Members

**Directory:** `Expired_and_resigned_committee_members_are_discounted_from_quorum`

| File | Title | Description |
|------|-------|-------------|
| `Expired` | Expired members excluded from quorum | Committee members past their expiry epoch do not count toward quorum |
| `Resigned` | Resigned members excluded from quorum | Committee members who resigned do not count toward quorum |

```
Initial State:
+-- Committee: 5 members total
    +-- Member A: Expired (ExpiryEpoch < CurrentEpoch)
    +-- Member B: Resigned (via ResignCommitteeCold cert)
    +-- Members C, D, E: Active
+-- Proposals: Requiring committee approval
+-- Votes: C, D, E all vote Yes

Event: PassEpoch

Quorum Calculation:
+-- Total Members: 5
+-- Expired: 1 (Member A)
+-- Resigned: 1 (Member B)
+-- Effective Quorum Base: 5 - 1 - 1 = 3 active members
+-- Yes Votes: 3
+-- Ratio: 3/3 = 100%
+-- Result: Passes if threshold <= 100%

Final State:
+-- Proposal ratified (quorum met from active members only)
```

**What This Tests:** Validates that the quorum denominator only includes active (non-expired, non-resigned) committee members.

---

### 6. Hard Fork with Minimal Committee

**File:** `Hard_Fork_can_still_be_initiated_with_less_than_minimal_committee_size`

```
Initial State:
+-- Committee: 2 members (below MinCommitteeSize of 3)
+-- Proposal: HardForkInitiation
+-- Votes: All active members vote Yes
+-- SPO Votes: Sufficient stake voting Yes

Event: PassEpoch

Special Rule:
+-- HardForkInitiation is exempt from MinCommitteeSize requirement
+-- Rationale: Must be able to hard fork even in emergency situations
+-- Committee threshold still applies if any members active

Final State:
+-- HardFork ratified despite undersized committee
```

**What This Tests:** Validates that hard fork initiation can proceed even when the committee is smaller than the minimum size, ensuring the chain can always upgrade.

---

### 7. Multiple Hot Credentials

**File:** `Many_CC_Cold_Credentials_map_to_the_same_Hot_Credential_act_as_many_votes`

```
Initial State:
+-- Committee: 5 cold credentials
+-- Hot Key Authorizations:
    +-- Cold A -> Hot X
    +-- Cold B -> Hot X
    +-- Cold C -> Hot X
    +-- Cold D -> Hot Y
    +-- Cold E -> Hot Y
+-- Proposal: Requiring committee approval
+-- Votes: Hot X votes Yes (counting as 3 votes: A, B, C)

Event: PassEpoch

Vote Counting:
+-- Hot X vote: Counts for Cold A, B, C (3 votes)
+-- Hot Y vote: Would count for Cold D, E (2 votes)
+-- Each cold credential is a separate vote
+-- Hot credential is just the signing key

Threshold Check:
+-- If only Hot X votes Yes: 3/5 = 60%
+-- Threshold comparison determines ratification

Final State:
+-- Proposal ratified if threshold met by vote weight
```

**What This Tests:** Validates that when multiple cold credentials authorize the same hot credential, a vote from that hot credential counts as multiple committee votes.

---

### 8. ParameterChange Effects on Existing Proposals

**Directory:** `ParameterChange_affects_existing_proposals`

#### 8.1 DRep Threshold Changes

**Subdirectory:** `DRep`

| File | Title | Description |
|------|-------|-------------|
| `Decreasing_the_threshold_ratifies_a_hitherto-unratifiable_proposal` | Threshold decrease enables ratification | Proposal blocked at old threshold passes at new lower threshold |
| `Increasing_the_threshold_prevents_a_hitherto-ratifiable_proposal_from_being_ratified` | Threshold increase blocks ratification | Proposal that would pass at old threshold fails at new higher threshold |

```
Initial State:
+-- Proposals:
    +-- ParameterChange A (to modify threshold)
    +-- Proposal B (affected by threshold)
+-- Votes: B has X% support

Sequence:
1. Epoch N: ParameterChange A enacted (threshold changes)
2. Epoch N: Proposal B evaluated with NEW threshold

Example (Threshold Decrease):
+-- Old DRepThreshold: 67%
+-- Proposal B Support: 55%
+-- Status: Would not ratify
+-- New DRepThreshold: 51% (from ParameterChange A)
+-- Status: Now ratifies (55% >= 51%)

Final State:
+-- Proposal B ratified at new threshold
```

**What This Tests:** Validates that threshold changes from enacted ParameterChange proposals apply immediately to all pending proposals.

#### 8.2 SPO Threshold Changes

**Subdirectory:** `SPO`

| File | Title | Description |
|------|-------|-------------|
| `Decreasing_the_threshold_ratifies_a_hitherto-unratifiable_proposal` | SPO threshold decrease enables ratification | SPO-voted proposal passes at new lower threshold |
| `Increasing_the_threshold_prevents_a_hitherto-ratifiable_proposal_from_being_ratified` | SPO threshold increase blocks ratification | SPO-voted proposal fails at new higher threshold |

Same pattern as DRep threshold changes but for SPO voting thresholds.

#### 8.3 Parent Chain Blocking

| File | Title | Description |
|------|-------|-------------|
| `A_parent_ParameterChange_proposal_can_prevent_its_child_from_being_enacted` | Parent blocks child enactment | Child proposal cannot be enacted until parent is enacted first |

```
Initial State:
+-- Parent: ParameterChange A (not yet ratified)
+-- Child: ParameterChange B (references A as PrevGovId)
+-- Votes: Child has sufficient votes

Event: PassEpoch

Parent Chain Check:
+-- Current Root: None (or different from Parent A)
+-- Child B PrevGovId: References A
+-- A not enacted: Root != A's GovActionId
+-- Result: Child B cannot ratify (parent mismatch)

Final State:
+-- Child B remains pending
+-- Must wait for Parent A to be ratified and enacted
```

**What This Tests:** Validates that proposals with PrevGovId references must wait for their parent to be enacted before they can ratify.

---

### 9. Voting Categories

**Directory:** `Voting`

#### 9.1 Active Voting Stake

**Subdirectory:** `Active_voting_stake`

##### DRep Stake Sources

| File | Title | Description |
|------|-------|-------------|
| `Proposal_deposits_contribute_to_active_voting_stake/Directly` | Direct deposit contribution | Proposal deposits count toward submitter's voting stake |
| `Proposal_deposits_contribute_to_active_voting_stake/After_switching_delegations` | Deposit after delegation switch | Deposits follow delegation changes |
| `Rewards_contribute_to_active_voting_stake` | Rewards in voting stake | Unclaimed rewards count toward voting stake |
| `Rewards_contribute_to_active_voting_stake_even_in_the_absence_of_StakeDistr` | Rewards without StakeDistr | Rewards count even when not in stake distribution |
| `UTxOs_contribute_to_active_voting_stake` | UTxOs in voting stake | UTxO value counts toward voting stake |

```
Initial State:
+-- DRep D: Registered
+-- Delegators to D:
    +-- Address A: 1000 ADA in UTxOs
    +-- Address A: 500 ADA in rewards
    +-- Address A: 100 ADA deposit on proposal

Event: Vote by DRep D

Voting Stake Calculation:
+-- UTxO Stake: 1000 ADA
+-- Reward Stake: 500 ADA
+-- Deposit Stake: 100 ADA
+-- Total: 1600 ADA attributed to DRep D's vote

Final State:
+-- DRep D's vote weighted by 1600 ADA
```

**What This Tests:** Validates all sources of stake that contribute to a DRep's voting power.

##### Predefined DReps

| File | Title | Description |
|------|-------|-------------|
| `acceptedRatio_with_default_DReps` | Accepted ratio calculation | How predefined DReps affect vote calculations |
| `AlwaysAbstain` | AlwaysAbstain behavior | Stake delegated to AlwaysAbstain reduces voting population |
| `AlwaysNoConfidence` | AlwaysNoConfidence behavior | Stake delegated to AlwaysNoConfidence counts as automatic No |
| `DRepAlwaysNoConfidence_is_sufficient_to_pass_NoConfidence` | NoConfidence via predefined DRep | Sufficient delegation to AlwaysNoConfidence can pass NoConfidence |

```
Predefined DReps:
+-- AlwaysAbstain: Stake is excluded from denominator
+-- AlwaysNoConfidence: Stake counts as "No" on all except NoConfidence proposals

Example (AlwaysNoConfidence):
+-- Total Active Stake: 1000 ADA
+-- Delegated to AlwaysNoConfidence: 600 ADA
+-- Delegated to regular DReps: 400 ADA
+-- Regular DReps vote Yes: 400 ADA
+-- NoConfidence Proposal:
    +-- Yes: 600 (NoConf DRep) + 0 = 600 ADA
    +-- No: 400 ADA (regular DReps opposing)
    +-- Result: 60% Yes -> may ratify

Example (AlwaysAbstain):
+-- Total Active Stake: 1000 ADA
+-- Delegated to AlwaysAbstain: 200 ADA
+-- Effective Voting Population: 800 ADA
+-- DReps vote Yes on 400 ADA
+-- Ratio: 400/800 = 50% (not 400/1000 = 40%)
```

**What This Tests:** Validates the special behavior of predefined DReps in vote calculations.

##### StakePool Stake Sources

| File | Title | Description |
|------|-------|-------------|
| `Proposal_deposits_contribute_to_active_voting_stake/Directly` | Direct pool stake | Pool operator's stake includes their deposits |
| `Proposal_deposits_contribute_to_active_voting_stake/After_switching_delegations` | Stake after delegation | Pool stake follows delegation changes |
| `Rewards_contribute_to_active_voting_stake` | Rewards in pool stake | Pool rewards count toward voting stake |
| `Rewards_contribute_to_active_voting_stake_even_in_the_absence_of_StakeDistr` | Rewards without distribution | Edge case for reward accounting |
| `UTxOs_contribute_to_active_voting_stake` | UTxO stake for pools | UTxO holdings count toward pool voting weight |

Same patterns as DRep stake sources but for stake pool operators.

#### 9.2 Interaction Between Governing Bodies

**Subdirectory:** `Interaction_between_governing_bodies`

| File | Title | Description |
|------|-------|-------------|
| `A_governance_action_is_automatically_ratified_if_threshold_is_set_to_0_for_all_related_governance_bodies` | Zero threshold auto-ratification | Proposals pass automatically when all relevant thresholds are 0 |
| `Hard-fork_initiation` | HardFork body interaction | Tests DRep, SPO, and CC voting for hard forks |
| `Motion_of_no-confidence` | NoConfidence body interaction | Tests DRep and SPO voting for no-confidence |
| `Update_committee_-_normal_state` | UpdateCommittee body interaction | Tests DRep, SPO, and CC voting for committee updates |

```
Governance Action Types and Required Bodies:

| Action Type          | DRep | SPO | Committee |
|---------------------|------|-----|-----------|
| ParameterChange     | Yes  | Sec | Yes       |
| HardForkInitiation  | Yes  | Yes | Yes       |
| TreasuryWithdrawal  | Yes  | No  | Yes       |
| NoConfidence        | Yes  | Yes | No        |
| UpdateCommittee     | Yes  | Yes | No*       |
| NewConstitution     | Yes  | No  | Yes       |
| Info                | Auto | No  | No        |

*Committee cannot vote on changes to itself

Security-relevant parameters (Sec) require SPO approval.

Final State:
+-- All required bodies must approve for ratification
+-- Missing approval from any required body blocks ratification
```

**What This Tests:** Validates that each governance action type requires the correct combination of governing bodies to approve.

#### 9.3 SPO Default Votes

**Subdirectory:** `SPO_default_votes/After_bootstrap_phase`

| File | Title | Description |
|------|-------|-------------|
| `Default_vote_is_No_in_general` | SPO default is No | Pools that don't vote are counted as No |
| `HardForkInitiation_-_default_vote_is_No` | HardFork default vote | SPOs default to No on hard forks |
| `Reward_account_delegated_to_AlwaysAbstain` | Pool with Abstain delegation | Pool reward account delegated to AlwaysAbstain affects default |
| `Reward_account_delegated_to_AlwaysNoConfidence` | Pool with NoConf delegation | Pool reward account delegation affects pool's default vote |

```
SPO Default Vote Rules:
+-- If pool explicitly votes: Use that vote
+-- If pool does not vote:
    +-- Check pool reward account delegation
    +-- If delegated to AlwaysAbstain: Vote is Abstain
    +-- If delegated to AlwaysNoConfidence: Vote depends on proposal type
    +-- Otherwise: Default to No

Example:
+-- Pool P has 100 ADA stake
+-- Pool P does not explicitly vote
+-- P's reward account delegated to AlwaysAbstain
+-- P's stake is excluded from SPO voting population

Final State:
+-- Non-voting pools counted per their delegation or as No
```

**What This Tests:** Validates the default voting behavior for stake pools that do not explicitly vote.

#### 9.4 Security-Relevant Parameter Changes

| File | Title | Description |
|------|-------|-------------|
| `SPO_needs_to_vote_on_security-relevant_parameter_changes` | SPO required for security params | Security group parameters require SPO approval |

```
Security-Relevant Parameters (require SPO vote):
+-- MaxBlockBodySize
+-- MaxTxSize
+-- MaxBlockHeaderSize
+-- MaxValSize
+-- MaxBlockExUnits
+-- MaxTxExUnits
+-- MaxCollateralInputs
+-- (and others in security group)

Initial State:
+-- Proposal: ParameterChange affecting security parameters
+-- Votes: DReps approve, Committee approves

Event: PassEpoch

Threshold Check:
+-- DRep Threshold: Met
+-- Committee Threshold: Met
+-- SPO Threshold: Required but not met
+-- Result: NOT RATIFIED (missing SPO approval)

Final State:
+-- Proposal remains pending until SPO threshold met
```

**What This Tests:** Validates that security-relevant parameter changes require SPO approval in addition to DRep and Committee approval.

---

### 10. When CC Expired

**Directory:** `When_CC_expired`

| File | Title | Description |
|------|-------|-------------|
| `SPOs_alone_cant_enact_hard-fork` | Expired CC blocks hard fork | Hard fork requires CC even if SPOs approve |
| `SPOs_alone_cant_enact_security_group_parameter_change` | Expired CC blocks security params | Security params require CC even if SPOs approve |

```
Initial State:
+-- Committee: All members expired
+-- Proposal: HardForkInitiation or Security ParameterChange
+-- Votes: SPOs approve, DReps approve

Event: PassEpoch

Committee Status Check:
+-- All members expired: No committee quorum possible
+-- Hard fork requires committee: BLOCKED
+-- Security params require committee: BLOCKED

Final State:
+-- Proposal NOT ratified
+-- Must wait for new committee to be installed
```

**What This Tests:** Validates that expired committees block proposals that require committee approval, even with full SPO and DRep support.

---

### 11. When CC Threshold is 0

**Directory:** `When_CC_threshold_is_0`

| File | Title | Description |
|------|-------|-------------|
| `SPOs_alone_can_enact_hard-fork_during_bootstrap` | Bootstrap hard fork without CC | During bootstrap, zero CC threshold allows SPO-only hard forks |
| `SPOs_alone_can_enact_security_group_parameter_change_during_bootstrap` | Bootstrap security params without CC | During bootstrap, zero CC threshold allows SPO-only security changes |

```
Bootstrap Phase (Protocol Version <= 9):
+-- DRep Thresholds: 0 (effectively auto-approve)
+-- CC Threshold: 0 (effectively auto-approve)
+-- SPO Thresholds: Normal values

Initial State:
+-- Protocol Version: 9 (bootstrap)
+-- Proposal: HardForkInitiation
+-- Votes: SPOs approve

Event: PassEpoch

Threshold Check:
+-- DRep: 0 threshold -> Auto-pass
+-- CC: 0 threshold -> Auto-pass
+-- SPO: Threshold met by votes
+-- Result: RATIFIED

Final State:
+-- Hard fork ratified with only SPO votes
```

**What This Tests:** Validates that during the bootstrap phase, zero thresholds for DReps and CC allow proposals to pass with only SPO votes.

---

## Ratification Process Summary

### Vote Counting
1. **DRep Votes**: Weighted by delegated stake (UTxOs + rewards + deposits)
2. **SPO Votes**: Weighted by pool stake
3. **Committee Votes**: One vote per non-expired, non-resigned member

### Threshold Calculation
```
accepted_ratio = sum(yes_stake) / (sum(total_stake) - sum(abstain_stake))
ratified = accepted_ratio >= threshold
```

### Special Cases
1. **AlwaysAbstain**: Stake excluded from denominator
2. **AlwaysNoConfidence**: Automatic No (except for NoConfidence proposals)
3. **Expired/Resigned CC**: Excluded from quorum
4. **Bootstrap Phase**: DRep and CC thresholds are 0
5. **MinCommitteeSize**: Committee cannot vote if below minimum

### Enactment Order
1. Delaying actions (HardFork, NoConfidence) enacted first
2. Other actions delayed to next epoch
3. Parent proposals must be enacted before children

---

## Protocol Parameters Affecting Ratification

| Parameter | Effect |
|-----------|--------|
| `dRepVotingThresholds` | Thresholds for each action type |
| `poolVotingThresholds` | SPO thresholds for each action type |
| `committeeMinSize` | Minimum active committee members for quorum |
| `committeeMaxTermLength` | Maximum epochs a member can serve |
| `govActionLifetime` | Epochs until proposal expires |

---

## References

- [CIP-1694: A First Step Towards On-Chain Decentralized Governance](https://github.com/cardano-foundation/CIPs/tree/master/CIP-1694)
- [Cardano Ledger Specification (Conway Era)](https://github.com/IntersectMBO/cardano-ledger)
- [Amaru Test Vector Generation](https://github.com/pragma-org/amaru)

