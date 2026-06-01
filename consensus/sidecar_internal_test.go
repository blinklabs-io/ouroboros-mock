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
	"path/filepath"
	"testing"

	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

// TestAssertObservationPickedLongestPeerTie covers the VRF-tie relaxation:
// a multi-way block_number tie is accepted when final_tip matches one of
// the tied longest peers (the oracle resolved the tie by VRF), and
// rejected when final_tip matches a shorter peer or no peer at all.
func TestAssertObservationPickedLongestPeerTie(t *testing.T) {
	a := tipPeer(100, 10, 0xAA) // tied longest
	b := tipPeer(102, 10, 0xBB) // tied longest (different VRF in reality)
	c := tipPeer(50, 7, 0xCC)   // shorter
	peers := []format.PeerInput{a, b, c}
	tipA := *a.Served[0].Tip
	tipB := *b.Served[0].Tip
	tipC := *c.Served[0].Tip

	// Tie resolved to peer A: accept (the oracle resolved it by VRF).
	if err := assertObservationPickedLongestPeer(peers, tipA, 0); err != nil {
		t.Fatalf("tie, final_tip=A: want accept, got %v", err)
	}
	// Tie resolved to peer B: accept.
	if err := assertObservationPickedLongestPeer(peers, tipB, 0); err != nil {
		t.Fatalf("tie, final_tip=B: want accept, got %v", err)
	}
	// final_tip is the shorter peer: reject (observation didn't pick longest).
	if err := assertObservationPickedLongestPeer(peers, tipC, 0); err == nil {
		t.Fatal("final_tip=shorter peer C: want reject, got nil")
	}
	// final_tip matches no captured peer: reject.
	nonexistent := format.Tip{
		Slot: 999, Hash: make(format.HexBytes, 32), BlockNumber: 10,
	}
	if err := assertObservationPickedLongestPeer(peers, nonexistent, 0); err == nil {
		t.Fatal("final_tip matches no peer: want reject, got nil")
	}

	// Single longest peer still must match (unchanged behavior).
	twoPeer := []format.PeerInput{c, a} // c shorter, a longest
	if err := assertObservationPickedLongestPeer(twoPeer, tipA, 0); err != nil {
		t.Fatalf("single longest, final_tip=A: want accept, got %v", err)
	}
	if err := assertObservationPickedLongestPeer(twoPeer, tipC, 0); err == nil {
		t.Fatal("single longest, final_tip=shorter: want reject, got nil")
	}
}

// loadForkFixture loads a frozen real-header fork fixture and returns its
// peers plus a helper yielding a given peer's last roll_forward tip (usable as
// final_tip). These fixtures carry genuine Conway headers with a real shared
// prefix, so the rollback-depth guard has an ancestor to compute — unlike the
// tip-only fixtures, which can only exercise the discredited tip-lead model.
func loadForkFixture(
	t *testing.T, name string,
) ([]format.PeerInput, func(peerID uint64) format.Tip) {
	t.Helper()
	v, err := LoadVector(filepath.Join("testdata", "fixtures", name))
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	peers := v.Capture.Peers
	tipOf := func(peerID uint64) format.Tip {
		for _, p := range peers {
			if p.PeerID == peerID {
				return lastRollForwardTip(p.Served)
			}
		}
		t.Fatalf("fixture %s has no peer_id=%d", name, peerID)
		return format.Tip{}
	}
	return peers, tipOf
}

// TestAssertObservationKeptShorterPeerExceedsK exercises exceeds-k no-switch
// acceptance with REAL multi-block shared-prefix fixtures, so the guard reasons
// about rollback DEPTH (finalBlock - commonAncestorBlock), not tip-length lead.
// The fixtures were frozen from real captures:
//
//	shallow_fork_big_lead: peer0 block 7, peer1 block 15, ancestor block 5
//	                       -> rollback 2, tip lead 8
//	deep_fork:             peer0 block 12, peer1 block 24, ancestor block 5
//	                       -> rollback 7, tip lead 12
//
// The shallow-fork case is the one the previous tip-lead guard got wrong: an
// 8-block lead "exceeds k=6", but the fork is only 2 blocks back, so a
// conformant node switches — keeping peer0 must be REJECTED. The deep-fork
// case flips from accept to reject as k grows past the rollback depth, which a
// tip-lead guard could never model.
func TestAssertObservationKeptShorterPeerExceedsK(t *testing.T) {
	shallow, shallowTip := loadForkFixture(t, "shallow_fork_big_lead.json")
	deep, deepTip := loadForkFixture(t, "deep_fork.json")

	cases := []struct {
		name     string
		peers    []format.PeerInput
		finalTip format.Tip
		k        uint64
		wantErr  bool
	}{
		{
			// rollback 2 <= k=6: peer1 adoptable, keeping peer0 is wrong.
			// The old tip-lead guard wrongly accepted this (lead 8 > 6).
			"shallow fork, rollback within k -> reject",
			shallow, shallowTip(0), 6, true,
		},
		{
			// rollback 7 > k=6: no longer chain reachable within k -> accept.
			"deep fork, rollback beyond k -> accept",
			deep, deepTip(0), 6, false,
		},
		{
			// same deep fork, k=10: rollback 7 <= 10, peer1 adoptable -> reject.
			// The old tip-lead guard wrongly accepted this (lead 12 > 10).
			"deep fork, larger k makes rollback adoptable -> reject",
			deep, deepTip(0), 10, true,
		},
		{
			// final_tip on the longest peer is always accepted.
			"final_tip on the longest peer -> accept",
			deep, deepTip(1), 6, false,
		},
		{
			// k=0 disables the exceeds-k path: a shorter final_tip is never
			// justified.
			"k=0, shorter final_tip -> reject",
			deep, deepTip(0), 0, true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := assertObservationPickedLongestPeer(tc.peers, tc.finalTip, tc.k)
			if tc.wantErr && err == nil {
				t.Fatalf("want reject, got accept")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("want accept, got %v", err)
			}
		})
	}
}
