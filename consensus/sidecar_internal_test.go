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

// TestAssertObservationKeptShorterPeerExceedsK covers the exceeds-k
// no-switch case: the oracle keeps a shorter incumbent (final_tip)
// because the longest peer leads it by more than k, so adopting it would
// exceed the stability window. Accepted only when that lead exceeds k.
func TestAssertObservationKeptShorterPeerExceedsK(t *testing.T) {
	a := tipPeer(40, 7, 0xAA)     // incumbent (kept)
	bFar := tipPeer(90, 15, 0xBB) // longest, leads A by 8
	peers := []format.PeerInput{a, bFar}
	tipA := *a.Served[0].Tip

	// final_tip=A, B leads by 8 > k=6: justified no-switch, accept.
	if err := assertObservationPickedLongestPeer(peers, tipA, 6); err != nil {
		t.Fatalf("exceeds-k (lead 8 > k 6): want accept, got %v", err)
	}
	// Same vector at k=0: nothing justifies keeping the shorter peer.
	if err := assertObservationPickedLongestPeer(peers, tipA, 0); err == nil {
		t.Fatal("k=0: keeping shorter peer not justified, want reject")
	}
	// Lead within k: the oracle should have switched, so reject.
	bNear := tipPeer(70, 12, 0xCC) // leads A by 5 <= k=6
	peersNear := []format.PeerInput{a, bNear}
	if err := assertObservationPickedLongestPeer(peersNear, tipA, 6); err == nil {
		t.Fatal("lead 5 <= k 6: keeping shorter peer not justified, reject")
	}
}
