# intersect_non_origin_v1

Single-peer capture of a `FindIntersect` against a block other than origin.

1. The sidecar syncs a probe connection from origin until the peer has
   served its second block and derives that block's slot and hash. A forged
   chain's hashes are unknown in advance, so the conversation names the point
   as `chain:2` rather than `<slot>:<hex>`.
2. `find_intersect [chain:2]` on the capture connection: cardano-node replies
   `intersect_found` for that block.
3. First `request_next` returns `roll_backward` to the intersect point, which
   is not origin and is not part of the served trace.
4. The remaining `request_next` calls return `roll_forward` for the blocks
   after the intersect.

The vector has one peer (`peer_id: 0`). `expected_output.downstream_chainsync`
mirrors the served trace and `final_tip` is the last roll_forward tip.
`run.sh` validates the shape with `check-consensus-vector -shape
single-non-origin`, which requires the trace to open with a roll_backward to a
non-origin point, so a capture that intersected at origin fails without
overwriting the committed golden.

## How to run

```bash
../../capture-scenario.sh intersect_non_origin_v1 -out /tmp/vector.json
```

Subnet `172.28.0.0/24`; host port `CONSENSUS_CAPTURE_CARDANO_PORT` (default
3031).
