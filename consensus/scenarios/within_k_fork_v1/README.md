# within_k_fork_v1

Multi-peer capture scenario for the **comfortable fork within the
stability window**. Two cardano-node peers serve divergent chains sharing
a common prefix; peer B leads peer A by a **few blocks (≤ k)**. A
non-forging observation node adopts peer A's tail, then switches to peer
B; the produced vector asserts the replay SUT reaches the same `final_tip`.

This is the complement to `fork_and_select_v1`. There, peer B leads by ~14
blocks (**> k**), so the SUT's implausibility guard would reject B's
single announced tip as a spoof unless `local_tip` is set. Here the
lead is **≤ k**, so the guard accepts B directly and **no `local_tip` is
needed** — the normal, no-rescue switch path. Together the two scenarios
bracket the k boundary from both sides.

## How it stages the fork

Three forge phases (`configurator.sh`), identical in shape to
`fork_and_select_v1` but with a **modest** peer-B extension:

| Phase | Forging pool | Starting DB | Kill slot | Becomes |
|---|---|---|---|---|
| A | pool 1 | fresh | `PREFIX_KILL_SLOT` (default 10) | shared prefix |
| B | pool 1 | copy of shared prefix | `+ PEER_A_EXTENSION_SLOTS` (default +15) | peer A |
| C | pool 2 | copy of shared prefix | `+ PEER_B_EXTENSION_SLOTS` (default +30) | peer B (longer) |

At `activeSlotsCoeff=0.4`, peer A expects ~3.4 extension blocks and peer B
~6.8, so peer B leads by ~3 blocks — comfortably in `[1, k]` with k=6.

## Capture, inspect, commit

Inspect each capture before committing:

- **Peer B must lead peer A by 1..k blocks.** A lead of 0 is a tie
  (that's `slot_battle_v1`, not this); a lead **> k** turns it into
  `fork_and_select_v1` — the composer would then derive a `local_tip`,
  and the vector would test the rescue path rather than the comfortable
  one. Re-run, or narrow `PEER_B_EXTENSION_SLOTS`, if the lead drifts out
  of range.
- The committed vector should therefore have `security_param: 6` and
  **no `local_tip`** (the composer omits it when the lead is ≤ k).

## What the replay asserts

- **Longest-chain selection + switch:** dingo adopts peer A,
  then switches to the longer peer B, reaching `final_tip` = peer B's tip
  — asserted via the emitted switch decision.
- **k accepts the within-window peer:** replayed at `security_param`
  = 6 with no `local_tip`; the implausibility guard must accept peer B
  because its lead is within k. (The `fork_and_select_v1` k-guard-is-live
  test covers the > k rejection.)

## Stack contents

Same shape as `fork_and_select_v1`: `configurator`, `cardano-peer-a` /
`cardano-peer-b` (non-forging), `cardano-observation`, `capture-sidecar`
(×3) and `composer` (capture profile). Subnet `172.26.0.0/24`.

## How to run

```bash
# Via the dispatcher.
../../capture-scenario.sh within_k_fork_v1

# Direct invocation.
./run.sh -out /tmp/within_k_fork_v1.json
```

By default the produced vector overwrites the committed golden at
`consensus/testdata/captured/within_k_fork_v1.json`.
