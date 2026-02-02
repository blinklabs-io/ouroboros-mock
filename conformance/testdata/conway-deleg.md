
# Conway DELEG Conformance Test Vectors

This document describes the conformance test vectors for the Conway ledger DELEG (Delegation) rules.

## Overview

The DELEG rule handles stake credential registration, deregistration, and delegation in the Conway era. These test vectors validate deposit handling, duplicate registration detection, delegation target validation, and combined certificate operations.

**Total test vectors:** 24

- Expected successes: 12
- Expected failures: 12

## Directory Structure

```
DELEG/
├── Register_stake_credential/
│   ├── With_correct_deposit_or_without_any_deposit
│   ├── With_incorrect_deposit
│   ├── When_already_registered
│   └── Twice_the_same_certificate_in_the_same_transaction
├── Unregister_stake_credentials/
│   ├── When_registered
│   ├── When_not_registered
│   ├── With_incorrect_deposit
│   ├── With_non-zero_reward_balance
│   ├── Register_and_unregister_in_the_same_transaction
│   └── deregistering_returns_the_deposit
├── Delegate_stake/
│   ├── Delegate_registered_stake_credentials_to_registered_pool
│   ├── Delegate_unregistered_stake_credentials
│   ├── Delegate_to_unregistered_pool
│   ├── Delegate_and_unregister
│   ├── Register_and_delegate_in_the_same_transaction
│   └── Delegate_already_delegated_credentials
├── Delegate_vote/
│   ├── Delegate_vote_of_registered_stake_credentials_to_registered_drep
│   ├── Delegate_vote_of_unregistered_stake_credentials
│   ├── Delegate_vote_of_registered_stake_credentials_to_unregistered_drep
│   ├── Redelegate_vote
│   └── Delegate_vote_and_unregister_stake_credentials
└── Delegate_both_stake_and_vote/
    ├── Delegate_to_DRep_and_SPO_and_change_delegation_to_a_different_SPO
    ├── Delegate_and_unregister_credentials
    └── Delegate,_retire_and_re-register_pool
```

---

## Register Stake Credential Tests

### Test: With correct deposit or without any deposit

**File:** `DELEG/Register_stake_credential/With_correct_deposit_or_without_any_deposit`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: (empty or does not include target credential)
├── Reward Accounts: (no account for target credential)
└── Protocol Params: keyDeposit=2000000 lovelace

Event 1: Transaction at slot N
├── Certificates:
│   └── Registration: KeyHash(credential) [deposit=2000000]
├── Validation:
│   ├── Deposit Check: PASS (matches keyDeposit or legacy cert with no deposit)
│   ├── Already Registered: NO
│   └── Creates new stake credential entry
└── Result: Success

Final State:
├── Registered Stakes: [credential -> 2000000]
├── Reward Accounts: [credential -> 0]
└── Deposits: +2000000 lovelace
```

#### What This Tests

Tests successful stake credential registration with the correct deposit amount. In Conway, the RegistrationCertificate (type 7) requires an explicit deposit amount that must match the protocol parameter `keyDeposit`. Legacy StakeRegistrationCertificate (type 0) without deposit is also accepted for backward compatibility.

---

### Test: With incorrect deposit

**File:** `DELEG/Register_stake_credential/With_incorrect_deposit`

**Rule:** DELEG

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: (empty)
└── Protocol Params: keyDeposit=2000000 lovelace

Event 1: Transaction at slot N
├── Certificates:
│   └── Registration: KeyHash(credential) [deposit=1000000] (wrong amount)
├── Validation:
│   ├── Deposit Check: FAIL (1000000 != 2000000)
│   └── Deposit must exactly match keyDeposit protocol parameter
└── Result: Failure (IncorrectDeposit)

Final State:
├── Registered Stakes: (unchanged - empty)
└── Transaction rejected
```

#### What This Tests

Tests that stake registration fails when the deposit amount does not match the required key deposit in protocol parameters. The ledger enforces exact deposit matching to prevent under-payment or over-payment of deposits.

---

### Test: When already registered

**File:** `DELEG/Register_stake_credential/When_already_registered`

**Rule:** DELEG

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000] (already registered!)
└── Protocol Params: keyDeposit=2000000 lovelace

Event 1: Transaction at slot N
├── Certificates:
│   └── Registration: KeyHash(credential) [deposit=2000000]
├── Validation:
│   ├── Already Registered: YES
│   └── Cannot re-register an already registered credential
└── Result: Failure (StakeCredentialAlreadyRegistered)

Final State:
├── Registered Stakes: [credential -> 2000000] (unchanged)
└── Transaction rejected
```

#### What This Tests

Tests that attempting to register an already-registered stake credential results in failure. Each stake credential can only be registered once. To change delegation, use delegation certificates; to withdraw deposit, deregister first.

---

### Test: Twice the same certificate in the same transaction

**File:** `DELEG/Register_stake_credential/Twice_the_same_certificate_in_the_same_transaction`

**Rule:** DELEG

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: (empty)
└── Protocol Params: keyDeposit=2000000 lovelace

Event 1: Transaction at slot N
├── Certificates:
│   ├── Registration: KeyHash(credential) [deposit=2000000]
│   └── Registration: KeyHash(credential) [deposit=2000000] (duplicate!)
├── Validation:
│   ├── First Registration: Would succeed
│   ├── Second Registration: FAIL (same credential in same tx)
│   └── Cannot register the same credential twice in one transaction
└── Result: Failure (DuplicateCertificate)

Final State:
├── Registered Stakes: (empty - transaction rejected)
└── Transaction rejected
```

#### What This Tests

Tests the behavior when the same stake credential is registered twice in a single transaction. The ledger detects duplicate certificates and rejects the transaction to prevent double-deposit collection.

---

## Unregister Stake Credentials Tests

### Test: When registered

**File:** `DELEG/Unregister_stake_credentials/When_registered`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── Reward Accounts: [credential -> 0]
└── Protocol Params: keyDeposit=2000000 lovelace

Event 1: Transaction at slot N
├── Certificates:
│   └── Deregistration: KeyHash(credential) [deposit=2000000]
├── Validation:
│   ├── Is Registered: YES
│   ├── Deposit Matches: YES
│   ├── Reward Balance: 0 (no pending rewards)
│   └── All checks pass
└── Result: Success

Final State:
├── Registered Stakes: (credential removed)
├── Reward Accounts: (credential removed)
└── Deposits: -2000000 lovelace (returned to user)
```

#### What This Tests

Tests successful stake deregistration of a registered credential. The credential must be registered, and the deposit refund amount must match the original deposit.

---

### Test: When not registered

**File:** `DELEG/Unregister_stake_credentials/When_not_registered`

**Rule:** DELEG

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: (credential NOT registered)
└── Protocol Params: keyDeposit=2000000 lovelace

Event 1: Transaction at slot N
├── Certificates:
│   └── Deregistration: KeyHash(credential) [deposit=2000000]
├── Validation:
│   ├── Is Registered: NO
│   └── Cannot deregister a non-existent credential
└── Result: Failure (StakeCredentialNotRegistered)

Final State:
├── Registered Stakes: (unchanged)
└── Transaction rejected
```

#### What This Tests

Tests that attempting to deregister a non-existent stake credential fails. You cannot deregister something that was never registered.

---

### Test: With incorrect deposit

**File:** `DELEG/Unregister_stake_credentials/With_incorrect_deposit`

**Rule:** DELEG

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
└── Protocol Params: keyDeposit=2000000 lovelace

Event 1: Transaction at slot N
├── Certificates:
│   └── Deregistration: KeyHash(credential) [deposit=1000000] (wrong!)
├── Validation:
│   ├── Is Registered: YES
│   ├── Deposit Matches: NO (1000000 != 2000000)
│   └── Refund must match original deposit
└── Result: Failure (IncorrectDeposit)

Final State:
├── Registered Stakes: [credential -> 2000000] (unchanged)
└── Transaction rejected
```

#### What This Tests

Tests that stake deregistration fails when the refund amount does not match the original deposit. This prevents users from claiming incorrect refund amounts.

---

### Test: With non-zero reward balance

**File:** `DELEG/Unregister_stake_credentials/With_non-zero_reward_balance`

**Rule:** DELEG

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── Reward Accounts: [credential -> 5000000] (has rewards!)
└── Protocol Params: keyDeposit=2000000 lovelace

Event 1: Transaction at slot N
├── Certificates:
│   └── Deregistration: KeyHash(credential) [deposit=2000000]
├── Validation:
│   ├── Is Registered: YES
│   ├── Reward Balance: 5000000 (non-zero!)
│   └── Must withdraw rewards before deregistering
└── Result: Failure (NonZeroRewardBalance)

Final State:
├── Registered Stakes: [credential -> 2000000] (unchanged)
├── Reward Accounts: [credential -> 5000000] (unchanged)
└── Transaction rejected
```

#### What This Tests

Tests the behavior of deregistration when the stake credential has a non-zero reward balance. Users must withdraw their rewards before deregistering to prevent loss of funds.

---

### Test: Register and unregister in the same transaction

**File:** `DELEG/Unregister_stake_credentials/Register_and_unregister_in_the_same_transaction`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: (empty)
└── Protocol Params: keyDeposit=2000000 lovelace

Event 1: Transaction at slot N
├── Certificates:
│   ├── Registration: KeyHash(credential) [deposit=2000000]
│   └── Deregistration: KeyHash(credential) [deposit=2000000]
├── Validation:
│   ├── Registration: Creates credential
│   ├── Deregistration: Removes just-created credential
│   └── Net effect: no change in state
└── Result: Success

Final State:
├── Registered Stakes: (empty - net zero effect)
└── Deposits: net 0 (paid then refunded)
```

#### What This Tests

Tests registering and unregistering a stake credential within the same transaction. This is a valid operation that results in no net state change but validates both certificate types work correctly in sequence.

---

### Test: deregistering returns the deposit

**File:** `DELEG/Unregister_stake_credentials/deregistering_returns_the_deposit`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── Reward Accounts: [credential -> 0]
└── Protocol Params: keyDeposit=2000000 lovelace

Event 1: Transaction at slot N
├── Certificates:
│   └── Deregistration: KeyHash(credential) [deposit=2000000]
├── Validation:
│   ├── Deposit returned to transaction outputs
│   └── User receives original deposit back
└── Result: Success

Final State:
├── Registered Stakes: (credential removed)
├── Reward Accounts: (credential removed)
└── User Balance: +2000000 lovelace (deposit returned)
```

#### What This Tests

Tests that successful deregistration returns the original stake deposit to the user. The deposit amount must be available in the transaction outputs.

---

## Delegate Stake Tests

### Test: Delegate registered stake credentials to registered pool

**File:** `DELEG/Delegate_stake/Delegate_registered_stake_credentials_to_registered_pool`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── Pool Registrations: [pool_key_hash -> registered]
└── Delegations: (no delegation for credential)

Event 1: Transaction at slot N
├── Certificates:
│   └── StakeDelegation: credential -> pool_key_hash
├── Validation:
│   ├── Credential Registered: YES
│   ├── Pool Registered: YES
│   └── All checks pass
└── Result: Success

Final State:
├── Registered Stakes: [credential -> 2000000]
├── Delegations: [credential -> pool_key_hash]
└── Stake now earns rewards from pool
```

#### What This Tests

Tests successful stake delegation from a registered credential to a registered pool. Both the stake credential and the target pool must be registered for delegation to succeed.

---

### Test: Delegate unregistered stake credentials

**File:** `DELEG/Delegate_stake/Delegate_unregistered_stake_credentials`

**Rule:** DELEG

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: (credential NOT registered)
├── Pool Registrations: [pool_key_hash -> registered]
└── Delegations: (empty)

Event 1: Transaction at slot N
├── Certificates:
│   └── StakeDelegation: credential -> pool_key_hash
├── Validation:
│   ├── Credential Registered: NO
│   └── Cannot delegate unregistered credentials
└── Result: Failure (StakeCredentialNotRegistered)

Final State:
├── Delegations: (unchanged - empty)
└── Transaction rejected
```

#### What This Tests

Tests that stake delegation fails when the stake credential is not registered. Users must register their stake credentials before delegating.

---

### Test: Delegate to unregistered pool

**File:** `DELEG/Delegate_stake/Delegate_to_unregistered_pool`

**Rule:** DELEG

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── Pool Registrations: (pool NOT registered)
└── Delegations: (empty)

Event 1: Transaction at slot N
├── Certificates:
│   └── StakeDelegation: credential -> pool_key_hash
├── Validation:
│   ├── Credential Registered: YES
│   ├── Pool Registered: NO
│   └── Cannot delegate to non-existent pool
└── Result: Failure (PoolNotRegistered)

Final State:
├── Delegations: (unchanged - empty)
└── Transaction rejected
```

#### What This Tests

Tests that stake delegation fails when the target pool is not registered. Users can only delegate to pools that have completed registration.

---

### Test: Delegate and unregister

**File:** `DELEG/Delegate_stake/Delegate_and_unregister`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── Pool Registrations: [pool_key_hash -> registered]
├── Reward Accounts: [credential -> 0]
└── Delegations: (empty)

Event 1: Transaction at slot N
├── Certificates:
│   ├── StakeDelegation: credential -> pool_key_hash
│   └── Deregistration: credential [deposit=2000000]
├── Validation:
│   ├── Delegation: Creates delegation
│   ├── Deregistration: Removes credential and delegation
│   └── Both operations succeed
└── Result: Success

Final State:
├── Registered Stakes: (credential removed)
├── Delegations: (removed)
└── Deposits: returned to user
```

#### What This Tests

Tests stake delegation followed by credential deregistration in the same transaction. This demonstrates that delegation and deregistration can be combined, with the final state being unregistered.

---

### Test: Register and delegate in the same transaction

**File:** `DELEG/Delegate_stake/Register_and_delegate_in_the_same_transaction`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: (empty)
├── Pool Registrations: [pool_key_hash -> registered]
└── Protocol Params: keyDeposit=2000000 lovelace

Event 1: Transaction at slot N
├── Certificates:
│   ├── Registration: credential [deposit=2000000]
│   └── StakeDelegation: credential -> pool_key_hash
├── Validation:
│   ├── Registration: Creates credential
│   ├── Delegation: Uses newly created credential
│   └── Both operations succeed in sequence
└── Result: Success

Final State:
├── Registered Stakes: [credential -> 2000000]
├── Delegations: [credential -> pool_key_hash]
└── User delegating immediately after registration
```

#### What This Tests

Tests combining stake registration and delegation in the same transaction. This is a common pattern that allows users to register and delegate in a single atomic operation.

---

### Test: Delegate already delegated credentials

**File:** `DELEG/Delegate_stake/Delegate_already_delegated_credentials`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── Pool Registrations: [pool1 -> registered, pool2 -> registered]
└── Delegations: [credential -> pool1] (already delegated!)

Event 1: Transaction at slot N
├── Certificates:
│   └── StakeDelegation: credential -> pool2 (different pool)
├── Validation:
│   ├── Credential Registered: YES
│   ├── New Pool Registered: YES
│   └── Redelegation is allowed
└── Result: Success

Final State:
├── Delegations: [credential -> pool2] (updated to new pool)
└── Stake now earns rewards from pool2
```

#### What This Tests

Tests redelegation of an already-delegated stake credential to a different pool. Users can change their delegation target at any time by submitting a new delegation certificate.

---

## Delegate Vote Tests

### Test: Delegate vote of registered stake credentials to registered drep

**File:** `DELEG/Delegate_vote/Delegate_vote_of_registered_stake_credentials_to_registered_drep`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── DRep Registrations: [drep_key_hash -> registered]
└── Vote Delegations: (empty)

Event 1: Transaction at slot N
├── Certificates:
│   └── VoteDelegation: credential -> drep(KeyHash(drep_key_hash))
├── Validation:
│   ├── Credential Registered: YES
│   ├── DRep Registered: YES
│   └── All checks pass
└── Result: Success

Final State:
├── Vote Delegations: [credential -> drep_key_hash]
└── Stake's voting power delegated to DRep
```

#### What This Tests

Tests successful vote delegation from a registered stake credential to a registered DRep. This is the Conway era's mechanism for delegating governance voting power.

---

### Test: Delegate vote of unregistered stake credentials

**File:** `DELEG/Delegate_vote/Delegate_vote_of_unregistered_stake_credentials`

**Rule:** DELEG

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: (credential NOT registered)
├── DRep Registrations: [drep_key_hash -> registered]
└── Vote Delegations: (empty)

Event 1: Transaction at slot N
├── Certificates:
│   └── VoteDelegation: credential -> drep(KeyHash(drep_key_hash))
├── Validation:
│   ├── Credential Registered: NO
│   └── Cannot delegate vote for unregistered credential
└── Result: Failure (StakeCredentialNotRegistered)

Final State:
├── Vote Delegations: (unchanged)
└── Transaction rejected
```

#### What This Tests

Tests that vote delegation fails when the stake credential is not registered. Only registered stake credentials can delegate their voting power.

---

### Test: Delegate vote of registered stake credentials to unregistered drep

**File:** `DELEG/Delegate_vote/Delegate_vote_of_registered_stake_credentials_to_unregistered_drep`

**Rule:** DELEG

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── DRep Registrations: (drep NOT registered)
└── Vote Delegations: (empty)

Event 1: Transaction at slot N
├── Certificates:
│   └── VoteDelegation: credential -> drep(KeyHash(unregistered_drep))
├── Validation:
│   ├── Credential Registered: YES
│   ├── DRep Registered: NO (and not Abstain/NoConfidence)
│   └── Cannot delegate to non-existent DRep
└── Result: Failure (DRepNotRegistered)

Final State:
├── Vote Delegations: (unchanged)
└── Transaction rejected
```

#### What This Tests

Tests that vote delegation fails when the target DRep is not registered. Note: Delegation to special DReps (Abstain, NoConfidence) does not require registration as these are protocol-level options.

---

### Test: Redelegate vote

**File:** `DELEG/Delegate_vote/Redelegate_vote`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── DRep Registrations: [drep1 -> registered, drep2 -> registered]
└── Vote Delegations: [credential -> drep1] (already delegated!)

Event 1: Transaction at slot N
├── Certificates:
│   └── VoteDelegation: credential -> drep(KeyHash(drep2))
├── Validation:
│   ├── Credential Registered: YES
│   ├── New DRep Registered: YES
│   └── Redelegation is allowed
└── Result: Success

Final State:
├── Vote Delegations: [credential -> drep2] (updated)
└── Voting power now with new DRep
```

#### What This Tests

Tests changing an existing vote delegation to a new DRep. Users can change their voting delegation at any time by submitting a new VoteDelegation certificate.

---

### Test: Delegate vote and unregister stake credentials

**File:** `DELEG/Delegate_vote/Delegate_vote_and_unregister_stake_credentials`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── DRep Registrations: [drep_key_hash -> registered]
├── Reward Accounts: [credential -> 0]
└── Vote Delegations: (empty)

Event 1: Transaction at slot N
├── Certificates:
│   ├── VoteDelegation: credential -> drep(KeyHash(drep_key_hash))
│   └── Deregistration: credential [deposit=2000000]
├── Validation:
│   ├── Vote Delegation: Creates delegation
│   ├── Deregistration: Removes credential and delegation
│   └── Both operations succeed
└── Result: Success

Final State:
├── Registered Stakes: (credential removed)
├── Vote Delegations: (removed)
└── Deposits: returned to user
```

#### What This Tests

Tests vote delegation followed by stake credential deregistration in the same transaction. The final state has the credential unregistered.

---

## Delegate Both Stake and Vote Tests

### Test: Delegate to DRep and SPO and change delegation to a different SPO

**File:** `DELEG/Delegate_both_stake_and_vote/Delegate_to_DRep_and_SPO_and_change_delegation_to_a_different_SPO`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── Pool Registrations: [pool1 -> registered, pool2 -> registered]
├── DRep Registrations: [drep -> registered]
└── Delegations: (empty for credential)

Event 1: Transaction at slot N
├── Certificates:
│   └── StakeVoteDelegation: credential -> pool1, drep
├── Validation:
│   └── Combined stake and vote delegation succeeds
└── Result: Success

Event 2: Transaction at slot M
├── Certificates:
│   └── StakeDelegation: credential -> pool2
├── Validation:
│   └── Changes stake delegation independently of vote
└── Result: Success

Final State:
├── Stake Delegations: [credential -> pool2]
├── Vote Delegations: [credential -> drep] (unchanged)
└── Independent control of stake vs vote delegation
```

#### What This Tests

Tests combined stake and vote delegation, with the ability to change delegations independently. The StakeVoteDelegation certificate delegates both at once, but subsequent certificates can modify either independently.

---

### Test: Delegate and unregister credentials

**File:** `DELEG/Delegate_both_stake_and_vote/Delegate_and_unregister_credentials`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── Pool Registrations: [pool -> registered]
├── DRep Registrations: [drep -> registered]
├── Reward Accounts: [credential -> 0]
└── Delegations: (empty)

Event 1: Transaction at slot N
├── Certificates:
│   ├── StakeVoteDelegation: credential -> pool, drep
│   └── Deregistration: credential [deposit=2000000]
├── Validation:
│   ├── Combined delegation creates both delegations
│   ├── Deregistration removes everything
│   └── Both operations succeed
└── Result: Success

Final State:
├── Registered Stakes: (empty)
├── Delegations: (empty)
└── Deposits: returned
```

#### What This Tests

Tests combined delegation followed by credential deregistration. This demonstrates that even combined delegations can be immediately followed by deregistration.

---

### Test: Delegate, retire and re-register pool

**File:** `DELEG/Delegate_both_stake_and_vote/Delegate,_retire_and_re-register_pool`

**Rule:** DELEG

**Expected:** Success

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> 2000000]
├── Pool Registrations: [pool -> registered]
├── DRep Registrations: [drep -> registered]
└── Delegations: (empty)

Event 1: Combined delegation and pool retirement
├── Certificates:
│   ├── StakeVoteDelegation: credential -> pool, drep
│   └── PoolRetirement: pool, retire_epoch
├── Validation:
│   └── Delegation to pool about to retire is valid
└── Result: Success

Event 2: After retire_epoch, pool re-registers
├── Certificates:
│   └── PoolRegistration: pool (new registration)
├── Validation:
│   └── Re-registration after retirement is valid
└── Result: Success

Final State:
├── Pool Registrations: [pool -> registered]
├── Delegations: [credential -> pool] (still valid after re-registration)
└── Delegation persists through retire/re-register cycle
```

#### What This Tests

Tests behavior when a delegated pool retires and is re-registered. This validates that delegations can persist or need to be re-established depending on the timing of pool lifecycle events.

---

## DELEG Rule Overview

The DELEG rule in Conway handles all stake and vote delegation operations:

### Certificate Types

| Type | ID | Description |
|------|-----|-------------|
| StakeRegistration | 0 | Legacy stake registration (no explicit deposit) |
| StakeDeregistration | 1 | Legacy stake deregistration |
| StakeDelegation | 2 | Delegate stake to a pool |
| Registration | 7 | Conway stake registration (with deposit) |
| Deregistration | 8 | Conway stake deregistration (with refund) |
| VoteDelegation | 9 | Delegate voting power to DRep |
| StakeVoteDelegation | 10 | Combined stake and vote delegation |
| StakeRegistrationDelegation | 11 | Register and delegate stake in one cert |
| VoteRegistrationDelegation | 12 | Register and delegate vote in one cert |
| StakeVoteRegistrationDelegation | 13 | Register and delegate both in one cert |

### Key Validation Rules

1. **Registration:** Deposit must match protocol parameter `keyDeposit`
2. **Deregistration:** Refund must match original deposit, reward balance must be zero
3. **Stake Delegation:** Both credential and target pool must be registered
4. **Vote Delegation:** Credential must be registered; DRep must be registered (except Abstain/NoConfidence)
5. **Duplicate Detection:** Same credential cannot appear twice in same transaction's certificates

### State Transitions

- Registration: Creates entry in `stakeRegistrations`, `rewardAccounts`
- Deregistration: Removes from `stakeRegistrations`, `delegations`, `voteDelegations`, `rewardAccounts`
- Delegation: Updates `delegations` map
- Vote Delegation: Updates `voteDelegations` map

---

## Related Documentation

- **CERTS Tests:** `../CERTS/README.md` - Certificate state rule tests
- **GOVCERT Tests:** `../GOVCERT/` - Governance certificate tests (DRep, Committee)
- **POOL Tests:** Pool registration/retirement tests
- **Conway Ledger Spec:** Formal specification of Conway era delegation rules

