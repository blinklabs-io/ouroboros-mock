
# Conway CERTS Conformance Test Vectors

This document describes the conformance test vectors for the Conway ledger CERTS (Certificate State) rules.

## Overview

The CERTS rule handles certificate processing within the Conway era. The test vectors in this directory validate the ledger's handling of withdrawals from reward accounts.

**Total test vectors:** 2

- Expected failures: 2
- Expected successes: 0

## Directory Structure

```
CERTS/
└── Withdrawals/
    ├── Withdrawing_from_an_unregistered_reward_account
    └── Withdrawing_the_wrong_amount
```

## Test Vectors

---

### Test: Withdrawing from an unregistered reward account

**File:** `CERTS/Withdrawals/Withdrawing_from_an_unregistered_reward_account`

**Rule:** CERTS

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: (empty or does not include the target credential)
├── Reward Accounts: (credential not registered)
└── Protocol Params: keyDeposit=[standard amount]

Event 1: Transaction at slot N
├── Withdrawals:
│   └── Attempt to withdraw from unregistered reward account
├── Validation:
│   ├── Reward Account Check: FAIL (not registered)
│   └── The stake credential must be registered to have a reward account
└── Result: Failure (NotRegisteredStakeCredential)

Final State:
├── Registered Stakes: (unchanged)
├── Reward Accounts: (unchanged)
└── UTxOs: (unchanged - transaction rejected)
```

#### What This Tests

Tests that withdrawing rewards from an unregistered reward account fails. In the Conway era, a stake credential must be registered (via a Registration or StakeRegistration certificate) before any rewards can be withdrawn. This test validates that the ledger correctly rejects withdrawal attempts from non-existent reward accounts.

**Key validation rules:**
- The withdrawal address must correspond to a registered stake credential
- Unregistered credentials cannot have associated reward accounts
- Attempting to withdraw from a non-existent account is a protocol violation

---

### Test: Withdrawing the wrong amount

**File:** `CERTS/Withdrawals/Withdrawing_the_wrong_amount`

**Rule:** CERTS

**Expected:** Failure

#### State Change Diagram

```
Initial State:
├── Registered Stakes: [credential -> deposit]
├── Reward Accounts: [credential -> actual_balance]
└── Protocol Params: keyDeposit=[standard amount]

Event 1: Transaction at slot N
├── Withdrawals:
│   └── Withdraw [claimed_amount] from registered reward account
├── Validation:
│   ├── Reward Account Exists: PASS
│   ├── Amount Check: FAIL (claimed_amount != actual_balance)
│   └── Withdrawal amount must exactly match the reward balance
└── Result: Failure (IncorrectWithdrawalAmount)

Final State:
├── Registered Stakes: (unchanged)
├── Reward Accounts: (unchanged - balance not modified)
└── UTxOs: (unchanged - transaction rejected)
```

#### What This Tests

Tests that withdrawing an incorrect amount from a reward account fails. In Cardano, when withdrawing from a reward account, the transaction must specify the exact balance available in the account. This prevents partial withdrawals and ensures atomic reward collection.

**Key validation rules:**
- The withdrawal amount must exactly match the current reward balance
- Partial withdrawals are not permitted
- Over-withdrawals (claiming more than available) are rejected
- Under-withdrawals (claiming less than available) are also rejected

**Exception:** Script-based (Plutus) withdrawals can bypass amount validation. This enables the "withdraw zero trick" where a zero-value withdrawal triggers script execution without actually withdrawing funds.

---

## CERTS Rule Overview

The CERTS rule in Conway processes certificates and validates withdrawals. Key responsibilities include:

1. **Certificate Processing:** Delegating to appropriate sub-rules (DELEG, POOL, GOVCERT) based on certificate type
2. **Withdrawal Validation:** Ensuring reward withdrawals match registered accounts and exact balances
3. **State Consistency:** Maintaining the integrity of stake registrations and reward accounts

### Certificate Types Processed

- **DELEG:** Stake registration, deregistration, and delegation certificates
- **POOL:** Pool registration and retirement certificates
- **GOVCERT:** Governance certificates (DRep registration, committee authorization)

### Withdrawal Requirements

For a withdrawal to be valid:
1. The stake credential must be registered
2. The withdrawal amount must exactly match the reward balance
3. The withdrawal address must be a reward address (stake credential hash)

---

## Related Documentation

- **DELEG Tests:** `../DELEG/README.md` - Delegation rule test vectors
- **GOVCERT Tests:** `../GOVCERT/` - Governance certificate test vectors
- **Conway Ledger Spec:** Formal specification of Conway era rules

