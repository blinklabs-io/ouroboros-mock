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

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/blinklabs-io/ouroboros-mock/consensus"
	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

func TestComposeMergesPeersAndObservation(t *testing.T) {
	dir := t.TempDir()
	peerA := writeSinglePeerCapture(t, dir, "peer-a", []format.ServedMessage{
		rollBackwardToOrigin(t),
		rollForward(t, 6, "aa", 10, "11aa"),
	})
	peerB := writeSinglePeerCapture(t, dir, "peer-b", []format.ServedMessage{
		rollBackwardToOrigin(t),
		rollForward(t, 6, "bb", 20, "22bb"),
		rollForward(t, 6, "bbcc", 30, "33bb"),
	})
	observation := writeSinglePeerCapture(t, dir, "obs",
		[]format.ServedMessage{
			rollForward(t, 6, "bb", 20, "22bb"),
			rollForward(t, 6, "bbcc", 30, "33bb"),
		},
	)

	vec, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{peerA, peerB},
		ObservationCapturePath: observation,
		Title:                  "fork_and_select_v1",
	})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if vec.Title != "fork_and_select_v1" {
		t.Fatalf("title = %q", vec.Title)
	}
	if vec.Capture == nil || len(vec.Capture.Peers) != 2 {
		t.Fatalf("peers = %+v", vec.Capture)
	}
	if vec.Capture.Peers[0].PeerID != 0 ||
		vec.Capture.Peers[1].PeerID != 1 {
		t.Fatalf("peer ids not assigned in order: %+v",
			vec.Capture.Peers,
		)
	}
	if len(vec.Capture.Peers[0].Served) != 2 ||
		len(vec.Capture.Peers[1].Served) != 3 {
		t.Fatalf("served counts: a=%d b=%d",
			len(vec.Capture.Peers[0].Served),
			len(vec.Capture.Peers[1].Served),
		)
	}
	if len(vec.Capture.ExpectedOutput.DownstreamChainSync) != 2 {
		t.Fatalf("downstream length = %d, want 2",
			len(vec.Capture.ExpectedOutput.DownstreamChainSync),
		)
	}
	if vec.Capture.ExpectedOutput.FinalTip.Slot != 30 {
		t.Fatalf("final_tip.slot = %d, want 30 (peer B's tip)",
			vec.Capture.ExpectedOutput.FinalTip.Slot,
		)
	}
}

func TestComposeRejectsMultiPeerInput(t *testing.T) {
	dir := t.TempDir()
	multi := filepath.Join(dir, "multi.json")
	if err := consensus.WriteVector(multi, format.TestVector{
		SchemaVersion: format.CurrentSchemaVersion,
		Title:         "already-merged",
		Category:      format.CategoryConsensus,
		Capture: &format.ConsensusCapture{
			Peers: []format.PeerInput{
				{PeerID: 0, Served: []format.ServedMessage{
					rollBackwardToOrigin(t),
				}},
				{PeerID: 1, Served: []format.ServedMessage{
					rollBackwardToOrigin(t),
				}},
			},
		},
	}); err != nil {
		t.Fatalf("WriteVector: %v", err)
	}
	if _, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{multi},
		ObservationCapturePath: multi,
	}); err == nil {
		t.Fatal("expected error for multi-peer input")
	}
}

func TestDiffAgainstGoldenStructuralMatch(t *testing.T) {
	dir := t.TempDir()
	peer := writeSinglePeerCapture(t, dir, "peer", []format.ServedMessage{
		rollBackwardToOrigin(t),
		rollForward(t, 6, "aa", 10, "1111"),
	})
	obs := writeSinglePeerCapture(t, dir, "obs", []format.ServedMessage{
		rollForward(t, 6, "aa", 10, "1111"),
	})
	golden, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{peer},
		ObservationCapturePath: obs,
		Title:                  "golden",
	})
	if err != nil {
		t.Fatalf("Compose golden: %v", err)
	}
	goldenPath := filepath.Join(dir, "golden.json")
	if err := consensus.WriteVector(goldenPath, golden); err != nil {
		t.Fatalf("WriteVector: %v", err)
	}

	// Fresh capture with different header bytes + tip hash, but same
	// structural shape and same tip slot. Should match.
	peerFresh := writeSinglePeerCapture(t, dir, "peer-fresh",
		[]format.ServedMessage{
			rollBackwardToOrigin(t),
			rollForward(t, 6, "bb", 10, "2222"),
		},
	)
	obsFresh := writeSinglePeerCapture(t, dir, "obs-fresh",
		[]format.ServedMessage{
			rollForward(t, 6, "bb", 10, "2222"),
		},
	)
	fresh, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{peerFresh},
		ObservationCapturePath: obsFresh,
		Title:                  "fresh",
	})
	if err != nil {
		t.Fatalf("Compose fresh: %v", err)
	}
	res, err := consensus.DiffAgainstGolden(goldenPath, fresh)
	if err != nil {
		t.Fatalf("DiffAgainstGolden: %v", err)
	}
	if !res.Match {
		t.Fatalf("expected structural match, got differences: %v",
			res.Differences,
		)
	}
}

func TestDiffAgainstGoldenToleratesLengthDifference(t *testing.T) {
	// Per CONSENSUS_W2.md §5 "Golden tolerance", forge nondeterminism
	// means per-run captures can have different message counts. The
	// diff should treat that as a match as long as the structural
	// skeleton holds (peer count, presence of roll_forwards, tip
	// slot within tolerance).
	dir := t.TempDir()
	peer := writeSinglePeerCapture(t, dir, "peer",
		[]format.ServedMessage{
			rollBackwardToOrigin(t),
			rollForward(t, 6, "aa", 10, "1111"),
		},
	)
	obs := writeSinglePeerCapture(t, dir, "obs",
		[]format.ServedMessage{
			rollForward(t, 6, "aa", 10, "1111"),
		},
	)
	golden, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{peer},
		ObservationCapturePath: obs,
	})
	if err != nil {
		t.Fatalf("Compose golden: %v", err)
	}
	goldenPath := filepath.Join(dir, "golden.json")
	if err := consensus.WriteVector(goldenPath, golden); err != nil {
		t.Fatalf("WriteVector: %v", err)
	}

	// Fresh has an extra roll_forward in peer 0 — should still match
	// because the structural skeleton is intact and the tip stays
	// inside FinalTipSlotTolerance.
	peerFresh := writeSinglePeerCapture(t, dir, "peer-fresh",
		[]format.ServedMessage{
			rollBackwardToOrigin(t),
			rollForward(t, 6, "aa", 10, "1111"),
			rollForward(t, 6, "cc", 11, "1112"),
		},
	)
	obsFresh := writeSinglePeerCapture(t, dir, "obs-fresh",
		[]format.ServedMessage{
			rollForward(t, 6, "aa", 11, "1112"),
		},
	)
	fresh, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{peerFresh},
		ObservationCapturePath: obsFresh,
	})
	if err != nil {
		t.Fatalf("Compose fresh: %v", err)
	}
	res, err := consensus.DiffAgainstGolden(goldenPath, fresh)
	if err != nil {
		t.Fatalf("DiffAgainstGolden: %v", err)
	}
	if !res.Match {
		t.Fatalf(
			"expected match despite length difference; got: %v",
			res.Differences,
		)
	}
}

func TestDiffAgainstGoldenFlagsPeerCountMismatch(t *testing.T) {
	dir := t.TempDir()
	// Each peer trace must start with roll_backward so the diff's
	// served-shape check passes and we isolate the failure to the
	// peer-count assertion this test exercises.
	peerA := writeSinglePeerCapture(t, dir, "a",
		[]format.ServedMessage{
			rollBackwardToOrigin(t),
			rollForward(t, 6, "aa", 10, "1111"),
		},
	)
	peerB := writeSinglePeerCapture(t, dir, "b",
		[]format.ServedMessage{
			rollBackwardToOrigin(t),
			rollForward(t, 6, "bb", 20, "2222"),
		},
	)
	obs := writeSinglePeerCapture(t, dir, "obs",
		[]format.ServedMessage{
			rollForward(t, 6, "bb", 20, "2222"),
		},
	)
	golden, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{peerA, peerB},
		ObservationCapturePath: obs,
	})
	if err != nil {
		t.Fatalf("Compose golden: %v", err)
	}
	goldenPath := filepath.Join(dir, "golden.json")
	if err := consensus.WriteVector(goldenPath, golden); err != nil {
		t.Fatalf("WriteVector: %v", err)
	}

	// Fresh dropped a peer — should flag. Use a fresh observation
	// matching peer A's tip (since peer A is now the only peer in
	// the fresh vector), so the compose-time longest-peer invariant
	// is satisfied and the diff is exercised on the peer-count delta.
	obsFresh := writeSinglePeerCapture(t, dir, "obs-fresh",
		[]format.ServedMessage{
			rollForward(t, 6, "aa", 10, "1111"),
		},
	)
	fresh, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{peerA},
		ObservationCapturePath: obsFresh,
	})
	if err != nil {
		t.Fatalf("Compose fresh: %v", err)
	}
	res, err := consensus.DiffAgainstGolden(goldenPath, fresh)
	if err != nil {
		t.Fatalf("DiffAgainstGolden: %v", err)
	}
	if res.Match {
		t.Fatal("expected mismatch; fresh has only 1 peer vs golden's 2")
	}
}

func TestDiffAgainstGoldenFlagsFinalTipSlotMismatch(t *testing.T) {
	dir := t.TempDir()
	// Each peer trace must start with roll_backward so the diff's
	// served-shape check passes and we isolate the failure to the
	// final-tip-slot assertion this test exercises.
	peer := writeSinglePeerCapture(t, dir, "peer",
		[]format.ServedMessage{
			rollBackwardToOrigin(t),
			rollForward(t, 6, "aa", 10, "1111"),
		},
	)
	obs := writeSinglePeerCapture(t, dir, "obs",
		[]format.ServedMessage{
			rollForward(t, 6, "aa", 10, "1111"),
		},
	)
	golden, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{peer},
		ObservationCapturePath: obs,
	})
	if err != nil {
		t.Fatalf("Compose golden: %v", err)
	}
	goldenPath := filepath.Join(dir, "golden.json")
	if err := consensus.WriteVector(goldenPath, golden); err != nil {
		t.Fatalf("WriteVector: %v", err)
	}

	// Fresh observation tip is at a different slot — should flag.
	peerFresh := writeSinglePeerCapture(t, dir, "peer-fresh",
		[]format.ServedMessage{
			rollBackwardToOrigin(t),
			rollForward(t, 6, "aa", 99, "1111"),
		},
	)
	obsFresh := writeSinglePeerCapture(t, dir, "obs-fresh",
		[]format.ServedMessage{
			rollForward(t, 6, "aa", 99, "1111"),
		},
	)
	fresh, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{peerFresh},
		ObservationCapturePath: obsFresh,
	})
	if err != nil {
		t.Fatalf("Compose fresh: %v", err)
	}
	res, err := consensus.DiffAgainstGolden(goldenPath, fresh)
	if err != nil {
		t.Fatalf("DiffAgainstGolden: %v", err)
	}
	if res.Match {
		t.Fatal("expected mismatch; fresh has a different final_tip slot")
	}
}

// TestDiffFlagsWrongPeerSelected guards the diff's longest-peer
// invariant directly: a multi-peer vector whose final_tip points at
// the shorter peer is the wrong-selector regression wolf31o2 flagged
// in PR #207. The diff must catch it on both golden and fresh paths
// so a broken committed corpus doesn't silently approve every
// future capture against itself.
func TestDiffFlagsWrongPeerSelected(t *testing.T) {
	dir := t.TempDir()
	// Hand-craft a vector that bypasses Compose's same invariant:
	// peer B is longer (slot/block 20 vs peer A's 10), but final_tip
	// points at peer A. Compose would refuse to emit this; we write
	// the JSON directly to simulate a hand-edited or pre-existing
	// broken golden.
	era := uint(6)
	broken := format.TestVector{
		SchemaVersion: format.CurrentSchemaVersion,
		Title:         "wrong-peer-golden",
		Category:      format.CategoryConsensus,
		Capture: &format.ConsensusCapture{
			Peers: []format.PeerInput{
				{
					PeerID: 0,
					Served: []format.ServedMessage{
						rollBackwardToOrigin(t),
						rollForward(t, era, "aa", 10, "11aa"),
					},
				},
				{
					PeerID: 1,
					Served: []format.ServedMessage{
						rollBackwardToOrigin(t),
						rollForward(t, era, "bb", 20, "22bb"),
					},
				},
			},
			ExpectedOutput: format.ExpectedOutput{
				DownstreamChainSync: []format.ServedMessage{
					rollForward(t, era, "aa", 10, "11aa"),
				},
				FinalTip: format.Tip{
					Slot:        10,
					Hash:        mustHexBytes(t, "11aa"),
					BlockNumber: 10,
				},
			},
		},
	}
	// Bypass EncodeTestVector (which we don't have a "skip validate"
	// hook for) by writing the JSON via the unvalidated stdlib path.
	raw, err := json.MarshalIndent(broken, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	goldenPath := filepath.Join(dir, "broken-golden.json")
	if err := os.WriteFile(goldenPath, raw, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	// A clean fresh vector built through Compose (which now refuses
	// to emit one with the wrong-peer pattern). Use distinct slots
	// so the longest peer is unambiguous.
	peerA := writeSinglePeerCapture(t, dir, "fresh-a",
		[]format.ServedMessage{
			rollBackwardToOrigin(t),
			rollForward(t, era, "aa", 10, "11aa"),
		},
	)
	peerB := writeSinglePeerCapture(t, dir, "fresh-b",
		[]format.ServedMessage{
			rollBackwardToOrigin(t),
			rollForward(t, era, "bb", 20, "22bb"),
		},
	)
	obs := writeSinglePeerCapture(t, dir, "fresh-obs",
		[]format.ServedMessage{
			rollForward(t, era, "bb", 20, "22bb"),
		},
	)
	fresh, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{peerA, peerB},
		ObservationCapturePath: obs,
	})
	if err != nil {
		t.Fatalf("Compose fresh: %v", err)
	}

	res, err := consensus.DiffAgainstGolden(goldenPath, fresh)
	if err != nil {
		t.Fatalf("DiffAgainstGolden: %v", err)
	}
	if res.Match {
		t.Fatal("expected mismatch on broken-golden longest-peer invariant")
	}
	// The diff should specifically attribute the failure to the
	// golden, not the fresh — fresh was composed through the strict
	// invariant and is internally consistent.
	var sawGoldenFailure bool
	for _, d := range res.Differences {
		if strings.HasPrefix(d, "golden:") &&
			strings.Contains(d, "observation selected peer_id=0") {
			sawGoldenFailure = true
			break
		}
	}
	if !sawGoldenFailure {
		t.Fatalf(
			"expected diff to flag the golden's wrong-peer selection, got: %v",
			res.Differences,
		)
	}
}

// TestDiffFlagsUnknownSelectedPeer guards the unknown-peer
// diagnostic path: when the observation's final_tip matches NONE
// of the captured peers (e.g. a forge nondeterminism flake where
// observation served a chain none of the recorded peers did),
// the diff must say `peer_id=<unknown>` rather than misattributing
// the tip to peer_id=0.
func TestDiffFlagsUnknownSelectedPeer(t *testing.T) {
	dir := t.TempDir()
	era := uint(6)
	broken := format.TestVector{
		SchemaVersion: format.CurrentSchemaVersion,
		Title:         "unknown-peer-golden",
		Category:      format.CategoryConsensus,
		Capture: &format.ConsensusCapture{
			Peers: []format.PeerInput{
				{
					PeerID: 0,
					Served: []format.ServedMessage{
						rollBackwardToOrigin(t),
						rollForward(t, era, "aa", 10, "11aa"),
					},
				},
				{
					PeerID: 1,
					Served: []format.ServedMessage{
						rollBackwardToOrigin(t),
						rollForward(t, era, "bb", 20, "22bb"),
					},
				},
			},
			ExpectedOutput: format.ExpectedOutput{
				DownstreamChainSync: []format.ServedMessage{
					rollForward(t, era, "cc", 99, "deadbeef"),
				},
				// final_tip matches no captured peer.
				FinalTip: format.Tip{
					Slot:        99,
					Hash:        mustHexBytes(t, "deadbeef"),
					BlockNumber: 99,
				},
			},
		},
	}
	raw, err := json.MarshalIndent(broken, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	goldenPath := filepath.Join(dir, "broken-golden.json")
	if err := os.WriteFile(goldenPath, raw, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	peerA := writeSinglePeerCapture(t, dir, "fresh-a",
		[]format.ServedMessage{
			rollBackwardToOrigin(t),
			rollForward(t, era, "aa", 10, "11aa"),
		},
	)
	peerB := writeSinglePeerCapture(t, dir, "fresh-b",
		[]format.ServedMessage{
			rollBackwardToOrigin(t),
			rollForward(t, era, "bb", 20, "22bb"),
		},
	)
	obs := writeSinglePeerCapture(t, dir, "fresh-obs",
		[]format.ServedMessage{
			rollForward(t, era, "bb", 20, "22bb"),
		},
	)
	fresh, err := consensus.Compose(consensus.ComposeArgs{
		PeerCapturePaths:       []string{peerA, peerB},
		ObservationCapturePath: obs,
	})
	if err != nil {
		t.Fatalf("Compose fresh: %v", err)
	}

	res, err := consensus.DiffAgainstGolden(goldenPath, fresh)
	if err != nil {
		t.Fatalf("DiffAgainstGolden: %v", err)
	}
	if res.Match {
		t.Fatal("expected mismatch on unknown-peer golden")
	}
	var sawUnknown bool
	for _, d := range res.Differences {
		if strings.HasPrefix(d, "golden:") &&
			strings.Contains(d, "peer_id=<unknown>") {
			sawUnknown = true
			break
		}
	}
	if !sawUnknown {
		t.Fatalf(
			"expected diff to report peer_id=<unknown>, got: %v",
			res.Differences,
		)
	}
}

// --- helpers ----------------------------------------------------------

func writeSinglePeerCapture(
	t *testing.T,
	dir, name string,
	served []format.ServedMessage,
) string {
	t.Helper()
	v := format.TestVector{
		SchemaVersion: format.CurrentSchemaVersion,
		Title:         name,
		Category:      format.CategoryConsensus,
		Capture: &format.ConsensusCapture{
			Peers: []format.PeerInput{
				{PeerID: 0, Served: served},
			},
			ExpectedOutput: format.ExpectedOutput{
				DownstreamChainSync: served,
				FinalTip:            extractTip(served),
			},
		},
	}
	path := filepath.Join(dir, name+".json")
	if err := consensus.WriteVector(path, v); err != nil {
		t.Fatalf("WriteVector %s: %v", path, err)
	}
	return path
}

func extractTip(served []format.ServedMessage) format.Tip {
	for _, m := range slices.Backward(served) {
		if m.Tip != nil &&
			m.MsgType == format.ChainSyncMsgRollForward {
			return format.Tip{
				Slot: m.Tip.Slot,
				Hash: append(format.HexBytes(nil), m.Tip.Hash...),
			}
		}
	}
	return format.Tip{}
}

func rollBackwardToOrigin(t *testing.T) format.ServedMessage {
	t.Helper()
	return format.ServedMessage{
		Protocol: format.ProtocolChainSync,
		MsgType:  format.ChainSyncMsgRollBackward,
		Point: &format.Point{
			Slot: 0,
			Hash: format.HexBytes{},
		},
		Tip: &format.Tip{
			Slot: 0,
			Hash: format.HexBytes{},
		},
	}
}

// rollForward synthesizes a roll_forward ServedMessage. tipSlot is
// used as both the tip's slot AND its block_number; real captures
// have block_number != slot (slots can be skipped under Praos), but
// for synthetic two-peer test vectors this preserves the
// longest-chain ordering by construction so the compose-time
// invariant (final_tip = longest peer's tip) holds without each
// test having to pick block numbers by hand.
func rollForward(
	t *testing.T,
	era uint,
	headerHex string,
	tipSlot uint64,
	tipHashHex string,
) format.ServedMessage {
	t.Helper()
	e := era
	return format.ServedMessage{
		Protocol:   format.ProtocolChainSync,
		MsgType:    format.ChainSyncMsgRollForward,
		Era:        &e,
		HeaderCbor: mustHexBytes(t, headerHex),
		Tip: &format.Tip{
			Slot:        tipSlot,
			Hash:        mustHexBytes(t, tipHashHex),
			BlockNumber: tipSlot,
		},
	}
}

func mustHexBytes(t *testing.T, s string) format.HexBytes {
	t.Helper()
	var hb format.HexBytes
	if err := hb.UnmarshalJSON([]byte(`"` + s + `"`)); err != nil {
		t.Fatalf("hex %q: %v", s, err)
	}
	return hb
}
