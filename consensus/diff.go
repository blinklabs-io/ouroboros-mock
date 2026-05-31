// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consensus

import (
	"fmt"
	"os"

	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

// DiffResult reports the outcome of comparing a freshly composed
// vector against a committed golden.
type DiffResult struct {
	Match       bool
	Differences []string
}

// FinalTipSlotTolerance is how far the fresh capture's
// expected_output.final_tip.slot may drift from the golden's before
// the diff calls it a mismatch. Sized to absorb the per-run
// leadership-lottery variance that the testnet inherently produces
// (a few extra slots before the first winning slot at or past the
// kill slot lands in a phase).
const FinalTipSlotTolerance = 20

// DiffAgainstGolden compares fresh against the on-disk golden vector
// at goldenPath. The comparison is structural — it tolerates the
// per-run variance the multi-peer testnet's forge inherently
// produces (different VRF wins across runs ⇒ different block counts
// and tip slots). Specifically:
//
//   - peer count must match exactly;
//   - each peer's served trace must start with roll_backward and
//     contain at least one roll_forward (catches the silent-empty
//     capture regression);
//   - expected_output.downstream_chainsync must contain at least one
//     roll_forward;
//   - expected_output.final_tip.slot must be within
//     FinalTipSlotTolerance of the golden's;
//   - both vectors must individually satisfy the longest-peer
//     invariant: expected_output.final_tip must match the per-peer
//     tip with the highest block_number. A vector that violates this
//     blesses a wrong-selector outcome at replay time;
//   - opaque bytes (header_cbor / block_cbor) and per-message slot /
//     hash content are NOT compared.
//
// The diff catches structural regressions (peer count drops, peers
// fall silent, observation tip lands way outside the forge-duration
// window, observation selected the wrong peer) without tripping on
// the forge nondeterminism (different VRF wins across runs ⇒
// different block counts and tip slots) that the testnet inherently
// produces.
func DiffAgainstGolden(
	goldenPath string,
	fresh format.TestVector,
) (DiffResult, error) {
	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		return DiffResult{}, fmt.Errorf("read golden: %w", err)
	}
	golden, err := format.DecodeTestVector(raw)
	if err != nil {
		return DiffResult{}, fmt.Errorf("decode golden: %w", err)
	}
	return diffVectors(golden, fresh), nil
}

func diffVectors(golden, fresh format.TestVector) DiffResult {
	var diffs []string

	if golden.Category != fresh.Category {
		diffs = append(diffs, fmt.Sprintf(
			"category: golden=%q fresh=%q",
			golden.Category, fresh.Category,
		))
	}
	if golden.Capture == nil || fresh.Capture == nil {
		diffs = append(diffs, "capture: one side missing the capture payload")
		return DiffResult{Match: false, Differences: diffs}
	}

	gp, fp := golden.Capture.Peers, fresh.Capture.Peers
	if len(gp) != len(fp) {
		diffs = append(diffs, fmt.Sprintf(
			"peers: count golden=%d fresh=%d", len(gp), len(fp),
		))
	}
	for i, p := range gp {
		if d := servedShapeIssue(p.Served); d != "" {
			diffs = append(diffs,
				fmt.Sprintf("golden.peers[%d].served: %s", i, d),
			)
		}
	}
	for i, p := range fp {
		if d := servedShapeIssue(p.Served); d != "" {
			diffs = append(diffs,
				fmt.Sprintf("fresh.peers[%d].served: %s", i, d),
			)
		}
	}

	// Validate both sides' downstream_chainsync. A degenerate
	// golden (someone hand-edited it into a roll_forward-free shape)
	// would otherwise silently approve any fresh capture against
	// that broken baseline.
	gd := golden.Capture.ExpectedOutput.DownstreamChainSync
	switch {
	case len(gd) == 0:
		diffs = append(diffs,
			"expected_output.downstream_chainsync: golden empty",
		)
	case !servedHasRollForward(gd):
		diffs = append(diffs,
			"expected_output.downstream_chainsync: golden has no roll_forward",
		)
	}
	fd := fresh.Capture.ExpectedOutput.DownstreamChainSync
	switch {
	case len(fd) == 0:
		diffs = append(diffs,
			"expected_output.downstream_chainsync: fresh empty",
		)
	case !servedHasRollForward(fd):
		diffs = append(diffs,
			"expected_output.downstream_chainsync: fresh has no roll_forward",
		)
	}

	gs := golden.Capture.ExpectedOutput.FinalTip.Slot
	fs := fresh.Capture.ExpectedOutput.FinalTip.Slot
	if slotDelta(gs, fs) > FinalTipSlotTolerance {
		diffs = append(diffs, fmt.Sprintf(
			"expected_output.final_tip.slot: golden=%d fresh=%d (|Δ|>%d)",
			gs, fs, FinalTipSlotTolerance,
		))
	}

	// Longest-peer invariant on both sides. Compose already enforces
	// this when writing a vector, but the check is repeated here so
	// the diff also rejects a hand-edited or pre-existing broken
	// golden — without this guard, a corrupt golden expecting the
	// wrong peer would silently approve every fresh capture.
	if err := assertObservationPickedLongestPeer(
		golden.Capture.Peers,
		golden.Capture.ExpectedOutput.FinalTip,
		golden.Capture.SecurityParam,
	); err != nil {
		diffs = append(diffs, "golden: "+err.Error())
	}
	if err := assertObservationPickedLongestPeer(
		fresh.Capture.Peers,
		fresh.Capture.ExpectedOutput.FinalTip,
		fresh.Capture.SecurityParam,
	); err != nil {
		diffs = append(diffs, "fresh: "+err.Error())
	}

	return DiffResult{Match: len(diffs) == 0, Differences: diffs}
}

func slotDelta(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// servedShapeIssue returns a short human description of the first
// shape problem found in a peer's served trace, or "" if it satisfies
// the documented invariants: non-empty, starts with roll_backward,
// contains at least one roll_forward.
func servedShapeIssue(served []format.ServedMessage) string {
	if len(served) == 0 {
		return "empty"
	}
	if served[0].MsgType != format.ChainSyncMsgRollBackward {
		return fmt.Sprintf(
			"first msg is %q, want roll_backward",
			served[0].MsgType,
		)
	}
	if !servedHasRollForward(served) {
		return "no roll_forward (silent capture?)"
	}
	return ""
}
