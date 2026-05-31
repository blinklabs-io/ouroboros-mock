# slot_battle_v1

Multi-peer capture scenario for the **Praos VRF tiebreaker**. Two
cardano-node peers serve chains that **share a prefix and each add one
block at the same height**, a few slots apart, forged by different pools
(hence different VRF). A non-forging observation node resolves the
equal-length tie by VRF and adopts one of them; the produced vector
asserts the replay SUT reaches the same `final_tip`.

This is the scenario that distinguishes a conformant selector from a
longest-tip stub on the *one genuinely hard* Praos decision: when two
chains tie on block count, the lower VRF output wins.

## How it stages the battle

The configurator (`configurator.sh`) drives three forge phases:

| Phase | Forging pool | Starting DB | Kill slot | Becomes |
|---|---|---|---|---|
| A | pool 1 | fresh | `PREFIX_KILL_SLOT` (default 10) | shared prefix |
| B | pool 1 | copy of shared prefix | `+ PEER_A_EXTENSION_SLOTS` (default +4) | peer A |
| C | pool 2 | copy of shared prefix | `+ PEER_B_EXTENSION_SLOTS` (default +4) | peer B |

Phases B and C use the **same small window** so each pool forges ~one
block extending the shared prefix, landing at the same height a few slots
apart. No key splicing or hand-synthesized blocks — just two pools forging
from the same parent.

## Capture, inspect, commit (this scenario is not push-button)

Unlike `fork_and_select_v1` (a robust ~14-block length gap), a slot battle
is **inherently non-deterministic** to forge and must be inspected before
committing. Two things must hold, and neither is guaranteed on any single
run (`activeSlotsCoeff=0.4`, two equal pools, each wins ~0.225/slot):

1. **Both peers add exactly one block** at the same height. A pool that
   wins 0 slots in the window leaves its peer with no divergent block; a
   pool that wins 2 makes the chains unequal length (a fork, not a tie).
2. **The two blocks are within 5 slots** of each other. Conway's VRF
   tiebreaker is *restricted* (`praosRestrictedTiebreakerMaxSlotDistance`
   = 5); beyond that, the SUT returns `ChainEqual` and the pick is
   order/nondeterministic — an unusable vector.

So the workflow is: run, inspect the produced vector (peer A and peer B
must have equal `block_number` with tip slots ≤5 apart), and commit only a
clean capture. Re-run, or tune `PEER_{A,B}_EXTENSION_SLOTS`, otherwise.
The committed `slot_battle_v1.json` is a hand-verified snapshot, not a
per-CI-regenerated artifact.

## What the replay asserts

- **VRF tiebreak agreement.** Replayed through the W5.1 fixture (dingo's
  real chainsync handler), which arms the Conway VRF tiebreaker via
  `GetPraosTiebreakerView` + the captured header VRF. The harness's
  longest-peer self-consistency check **accepts the tie** (W5.3) as long
  as `final_tip` matches one of the tied peers; the conformance assertion
  is that dingo independently reaches that same `final_tip`. The tip-only
  `UpdatePeerTip` path could never arm the tiebreaker — the real-handler
  fixture is mandatory here.
- **Switch decision** (W5.4). The observation adopts one block then
  switches to the VRF winner; the replay asserts dingo emits the
  corresponding switch off the loser onto `final_tip`.

## Stack contents

Same shape as `fork_and_select_v1`: `configurator` (genesis + 3-phase
forge), `cardano-peer-a` / `cardano-peer-b` (non-forging, serve the two
battle chains), `cardano-observation` (both peers as static localRoots),
`capture-sidecar` (×3) and `composer` (capture profile). Subnet
`172.25.0.0/24`.

## How to run

```bash
# Via the dispatcher.
../../capture-scenario.sh slot_battle_v1

# Regenerate the committed golden (skips the regression check).
../../capture-scenario.sh slot_battle_v1 --skip-golden

# Direct invocation.
./run.sh -out /tmp/slot_battle_v1.json
```

By default the produced vector overwrites the committed golden at
`consensus/testdata/captured/slot_battle_v1.json`. `--keep-up` leaves the
stack running for inspection.
