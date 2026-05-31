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

// TestAssertObservationPickedLongestPeerTie covers the W5.3 relaxation:
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

	// Tie resolved to peer A: accept (was "ambiguous" rejection before W5.3).
	if err := assertObservationPickedLongestPeer(peers, tipA); err != nil {
		t.Fatalf("tie, final_tip=A: want accept, got %v", err)
	}
	// Tie resolved to peer B: accept.
	if err := assertObservationPickedLongestPeer(peers, tipB); err != nil {
		t.Fatalf("tie, final_tip=B: want accept, got %v", err)
	}
	// final_tip is the shorter peer: reject (observation didn't pick longest).
	if err := assertObservationPickedLongestPeer(peers, tipC); err == nil {
		t.Fatal("final_tip=shorter peer C: want reject, got nil")
	}
	// final_tip matches no captured peer: reject.
	nonexistent := format.Tip{
		Slot: 999, Hash: make(format.HexBytes, 32), BlockNumber: 10,
	}
	if err := assertObservationPickedLongestPeer(peers, nonexistent); err == nil {
		t.Fatal("final_tip matches no peer: want reject, got nil")
	}

	// Single longest peer still must match (unchanged behavior).
	twoPeer := []format.PeerInput{c, a} // c shorter, a longest
	if err := assertObservationPickedLongestPeer(twoPeer, tipA); err != nil {
		t.Fatalf("single longest, final_tip=A: want accept, got %v", err)
	}
	if err := assertObservationPickedLongestPeer(twoPeer, tipC); err == nil {
		t.Fatal("single longest, final_tip=shorter: want reject, got nil")
	}
}
