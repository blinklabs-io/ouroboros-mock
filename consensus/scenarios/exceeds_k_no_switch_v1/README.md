# exceeds_k_no_switch_v1

Multi-peer capture scenario for the **k-bound no-switch**: a node settled deep
on one chain refuses to switch to a longer competing chain when doing so would
require rolling back **more than k blocks**. Two peers serve divergent chains
sharing a common prefix. Peer A — the incumbent — is forged **more than k=6
blocks past the fork**; peer B is longer still but forks from the same prefix,
so adopting it would roll the incumbent back > k. A conformant Praos node keeps
peer A; the produced vector asserts the replay SUT does too — `final_tip` is
the **shorter** peer A.

This is the complement to `fork_and_select_v1` and `within_k_fork_v1`, which
both switch to the *longer* peer because their fork is shallow (rollback ≤ k).
Here the fork is deep (rollback > k), so the longer peer is **not** adopted.

## The governing quantity is rollback depth, not tip lead

What forbids the switch is the **rollback depth** — the incumbent's block
number minus the common-ancestor block number — exceeding k, NOT how much
longer the competing chain is. In the committed vector peer A is block 9 and
peer B is block 20, but the number that matters is peer A's **7 blocks past the
fork** (ancestor block 2): rolling back 7 > k=6 to reach peer B's branch is
what a conformant node refuses. A chain twice as long that forked only a block
back would be adopted; this one is refused because its fork is deep.

## Mechanism caveat — the SUT and the oracle refuse for different reasons

cardano-node (the oracle) refuses the switch by the **rollback-depth bound**
above. dingo (in this replay) refuses by its **implausibility guard**: a peer
whose announced tip is more than k ahead of the reference, with no `local_tip`
set, is rejected as a possible spoof (`chainselection/selector.go`). These are
*different mechanisms* — one bounds rollback depth, the other bounds tip lead —
that here **agree on the outcome**: keep peer A. Because the rollback is
genuinely > k, `final_tip = peer A` is the outcome a conformant node would also
reach, so this is a legitimate conformance vector tested for outcome agreement,
not a dingo quirk. (If the rollback were ≤ k the two would *disagree* —
cardano-node would adopt peer B — and the vector would be wrong; the composer's
rollback-depth self-consistency check now rejects that shape.)

## Oracle topology — a documented stand-in

The observation node pins **only peer A** (`topology/observation.json`,
valency 1). A fully faithful both-peers capture would need a *sequenced*
bring-up — settle the oracle deep on peer A, THEN offer peer B's deep fork and
watch it refuse the rollback — which this all-at-once configurator cannot
stage: with both peers up from genesis the oracle just takes the longest
(peer B), because adopting from genesis is no rollback at all. The valency-1
pin deterministically produces the SAME `final_tip = peer A` that a node already
settled deep on A would reach (rollback > k), so it is a stand-in for the
sequenced bring-up, not a blinding that changes the answer. The vector still
records BOTH peers, so the replay SUT is offered peer B and must refuse it on
its own.

## How it stages the chains

| Phase | Pool | Starting DB | Kill slot | Becomes |
|---|---|---|---|---|
| A | pool 1 | fresh | `PREFIX_KILL_SLOT` (10) | shared prefix |
| B | pool 1 | copy of shared prefix | `+ PEER_A_EXTENSION_SLOTS` (+22) | peer A (deep incumbent) |
| C | pool 2 | copy of shared prefix | `+ PEER_B_EXTENSION_SLOTS` (+80) | peer B (longer) |

Peer A must end **> k blocks past the fork** (that is the rollback), and peer B
must end **> k blocks past peer A** (so the SUT's guard rejects it). The hard
part is the forge tolerance: peer A's window is kept small in slots so phase C
still starts inside cardano-node's CaughtUp tolerance, and a high realized
forge rate is relied on to reach > k blocks within it. The capture is
**inspect-retry**: the composer rejects any capture whose rollback is ≤ k, and
the loop re-rolls until forge variance delivers a deep-enough peer A. See
`configurator.sh`.

## Capture, inspect, commit

The committed vector must have:

- **`final_tip` = peer A** (the shorter incumbent — the observation pinned A).
- **peer A more than k blocks past the fork** (rollback > k), so the composer
  accepts the non-longest `final_tip`; and **peer B longer than peer A by > k**
  so the SUT's guard rejects it.
- `security_param: 6`, **no `local_tip`** (the composer emits `local_tip` only
  when `final_tip` is the *longer* peer; here it is the shorter, so none — and
  a `local_tip` would arm the catch-up relaxation that lets the SUT *accept*
  peer B, the opposite of this scenario), and **no `expected_rollback`** (there
  is no switch).

## What the replay asserts

dingo, fed peer A's then peer B's headers at `security_param`=6 with no
`local_tip`, must **reject peer B** (its tip is > k ahead of peer A) and keep
peer A, reaching `final_tip` = peer A. If dingo's guard were broken and it
adopted B, the replay would land on peer B ≠ `final_tip` and fail — exactly the
divergence this scenario guards against. (`fork_and_select_v1`'s
`k-guard-is-live` test covers the `local_tip` rescue from the other direction:
clearing `local_tip` there makes the SUT reject the far-ahead winner.)

## Stack contents

Same shape as `fork_and_select_v1` (subnet `172.27.0.0/24`), except the
observation topology pins only peer A.

## How to run

```bash
../../capture-scenario.sh exceeds_k_no_switch_v1
# or: ./run.sh -out /tmp/exceeds_k_no_switch_v1.json
```
