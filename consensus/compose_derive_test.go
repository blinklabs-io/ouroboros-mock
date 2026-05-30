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
	"bytes"
	"testing"

	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

func tipPeer(slot, block uint64, hb byte) format.PeerInput {
	h := make([]byte, 32)
	for i := range h {
		h[i] = hb
	}
	return format.PeerInput{Served: []format.ServedMessage{{
		Protocol: format.ProtocolChainSync,
		MsgType:  format.ChainSyncMsgRollForward,
		Tip:      &format.Tip{Slot: slot, Hash: h, BlockNumber: block},
	}}}
}

func TestDeriveLocalTip(t *testing.T) {
	a := tipPeer(26, 6, 0xAA)     // incumbent (shorter chain)
	bWin := tipPeer(90, 15, 0xBB) // winner (final_tip)
	finalB := format.Tip{
		Slot: 90, Hash: bWin.Served[0].Tip.Hash, BlockNumber: 15,
	}

	// k=0: never derives a local_tip.
	if got := deriveLocalTip([]format.PeerInput{a, bWin}, finalB, 0); got != nil {
		t.Fatalf("k=0: want nil, got %+v", got)
	}

	// Winner leads incumbent by 9 > k=6: derive the incumbent (peer A).
	got := deriveLocalTip([]format.PeerInput{a, bWin}, finalB, 6)
	if got == nil ||
		got.Slot != 26 ||
		got.BlockNumber != 6 ||
		!bytes.Equal(got.Hash, a.Served[0].Tip.Hash) {
		t.Fatalf("gap>k: want peer A tip (slot 26, block 6), got %+v", got)
	}

	// Lead within k (10-6=4 <= 6): the guard accepts the winner, no rescue.
	bNear := tipPeer(40, 10, 0xCC)
	finalNear := format.Tip{
		Slot: 40, Hash: bNear.Served[0].Tip.Hash, BlockNumber: 10,
	}
	if got := deriveLocalTip([]format.PeerInput{a, bNear}, finalNear, 6); got != nil {
		t.Fatalf("gap<=k: want nil, got %+v", got)
	}

	// Single peer: no incumbent to follow.
	if got := deriveLocalTip([]format.PeerInput{bWin}, finalB, 6); got != nil {
		t.Fatalf("single peer: want nil, got %+v", got)
	}
}
