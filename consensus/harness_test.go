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

package consensus_test

import (
	"testing"

	"github.com/blinklabs-io/ouroboros-mock/consensus"
	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

// firstPeerStub is a deliberately-wrong Replayer: it adopts the FIRST
// peer it sees and always reports that peer's last announced tip,
// ignoring every other peer and every rollback. It is the canonical
// "bad SUT" — it never performs chain selection.
//
// For a multi-peer vector whose final_tip is a non-first peer (e.g.
// fork_and_select_v1, where peer 0 ends at block 6 but final_tip is peer
// 1's block 15), this must produce a final_tip mismatch. It is a
// self-contained fake with no dingo dependency, so it runs in
// ouroboros-mock CI with no SUT checkout — the whole point of the test.
type firstPeerStub struct {
	have bool
	peer uint64
	tip  format.Tip
}

func (s *firstPeerStub) RollForward(
	peerID uint64, _ uint, _ []byte, tip format.Tip,
) error {
	if !s.have {
		s.have = true
		s.peer = peerID
	}
	if peerID == s.peer {
		s.tip = tip
	}
	return nil
}

func (s *firstPeerStub) RollBackward(
	_ uint64, _ format.Point, _ format.Tip,
) error {
	return nil
}

func (s *firstPeerStub) Stabilize() {}

func (s *firstPeerStub) BestTip() (format.Tip, bool) {
	return s.tip, s.have
}

func (s *firstPeerStub) DrainSwitchEvents() []format.SwitchEvent {
	return nil
}

// TestHarnessFailsBadReplayer is the load-bearing "can the harness bite?"
// test: a Replayer that ignores chain selection must FAIL at least one
// committed vector. Without this, every real adapter could pass
// trivially and the corpus would prove nothing.
func TestHarnessFailsBadReplayer(t *testing.T) {
	vectors, err := consensus.CapturedVectors()
	if err != nil {
		t.Fatalf("CapturedVectors: %v", err)
	}
	if len(vectors) == 0 {
		t.Skip("no captured vectors embedded")
	}

	failed := 0
	for _, cv := range vectors {
		// RunConsensusVector returns an error rather than calling
		// t.Fatal, so a failing vector is observable here.
		if err := consensus.RunConsensusVector(
			t, cv.Vector, &firstPeerStub{},
		); err != nil {
			failed++
			t.Logf("first-peer stub correctly failed %q: %v", cv.Name, err)
		}
	}

	if failed == 0 {
		t.Fatal(
			"first-peer stub passed every committed vector — the harness " +
				"cannot distinguish a conformant SUT from one that ignores " +
				"chain selection",
		)
	}
}
