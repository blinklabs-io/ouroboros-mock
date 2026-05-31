# exceeds_k_no_switch_v1

Multi-peer capture scenario for the **k-bound refusal**: when a competing
chain is too far ahead to adopt within the stability window k, a node
keeps its current (shorter) chain. Two peers serve divergent chains
sharing a common prefix; peer B leads peer A by **more than k blocks**.
The observation node settles on peer A; the produced vector asserts the
replay SUT also stays on peer A — `final_tip` is the **shorter** peer.

This is the complement to `fork_and_select_v1` and `within_k_fork_v1`,
which both end on the *longer* peer. Here the longer peer is **not**
adopted.

## Mechanism caveat — read this

cardano-node refuses a switch by bounding **rollback depth** from its
current chain. dingo (in this replay) refuses by the **tip-implausibility
guard**: a peer whose announced tip is more than k ahead of the reference,
with no `local_tip` set, is rejected as a spoof. These are *different
mechanisms* that happen to agree on the outcome (don't adopt peer B). This
scenario therefore tests **outcome-agreement, not mechanism-agreement** —
it verifies dingo ends on peer A, the same chain the oracle is on.

To keep the oracle on the shorter chain deterministically, the observation
node's topology pins **only peer A** (`topology/observation.json`), so it
settles on A. The vector still records BOTH peers, so the replay SUT is
offered both and must reject the > k-ahead peer B on its own. A fully
faithful capture (observation established deep on A, then offered B's deep
fork, refusing the rollback) would need a sequenced bring-up; this
topology pin is the pragmatic stand-in.

## How it stages the chains

| Phase | Pool | Starting DB | Kill slot | Becomes |
|---|---|---|---|---|
| A | pool 1 | fresh | `PREFIX_KILL_SLOT` (10) | shared prefix |
| B | pool 1 | copy of shared prefix | `+ PEER_A_EXTENSION_SLOTS` (+8) | peer A (shorter) |
| C | pool 2 | copy of shared prefix | `+ PEER_B_EXTENSION_SLOTS` (+60) | peer B (> k longer) |

Peer A expects ~1.8 extension blocks, peer B ~13.5 — a lead of ~12,
robustly > k=6. Peer A is deliberately kept short: isolated single-pool
forging is high-variance on few slots, and an over-performing peer A is
what erodes the lead. Both extensions stay within epoch 0 (total slot
< epochLength=75). Re-run if the lead drifts to ≤ k.

## Capture, inspect, commit

The committed vector must have:

- **`final_tip` = peer A** (the shorter peer — the observation only saw A).
- **peer B leading peer A by > k** (else the composer rejects the
  non-longest `final_tip` as an unjustified wrong-peer selection).
- `security_param: 6`, **no `local_tip`** (setting `local_tip` would make
  the SUT *accept* peer B — the opposite of this scenario), and **no
  `expected_rollback`** (there is no switch).

Re-run if the lead drifts to ≤ k.

## What the replay asserts

dingo, fed peer A's then peer B's headers at `security_param`=6 with no
`local_tip`, must **reject peer B** (tip > k ahead of peer A) and keep peer
A, reaching `final_tip` = peer A. If dingo's implausibility guard were
broken (it adopted B), the replay would land on peer B ≠ `final_tip` and
fail — which is exactly the divergence this scenario guards against. (The
`fork_and_select_v1` `k-guard-is-live` test covers the same guard from the
other direction.)

## Stack contents

Same shape as `fork_and_select_v1` (subnet `172.27.0.0/24`), except the
observation topology pins only peer A.

## How to run

```bash
../../capture-scenario.sh exceeds_k_no_switch_v1
# or: ./run.sh -out /tmp/exceeds_k_no_switch_v1.json
```
