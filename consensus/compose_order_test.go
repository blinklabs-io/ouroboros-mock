package consensus

import (
	"testing"

	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

// TestOrderPeersForReplay verifies the composer feeds the winner last when
// final_tip is the longest chain (a switch or VRF tie) and the incumbent first
// when final_tip is a shorter chain (an exceeds-k no-switch), reassigning
// peer_id by position. This is what makes the harness's switch assertion and
// the exceeds-k guard order independent of the VRF lottery.
func TestOrderPeersForReplay(t *testing.T) {
	short := tipPeer(40, 5, 0xAA)
	long := tipPeer(90, 12, 0xBB)
	longTip := *long.Served[0].Tip
	shortTip := *short.Served[0].Tip

	// final_tip = the longer peer → fed LAST, even when passed first.
	got := orderPeersForReplay([]format.PeerInput{long, short}, longTip)
	if !tipsEqual(lastRollForwardTip(got[len(got)-1].Served), longTip) {
		t.Fatalf("winner final_tip must be the last peer fed")
	}
	if got[0].PeerID != 0 || got[1].PeerID != 1 {
		t.Fatalf("peer_id must be reassigned by position, got %d,%d",
			got[0].PeerID, got[1].PeerID)
	}

	// final_tip = the shorter peer → fed FIRST, even when passed last.
	got = orderPeersForReplay([]format.PeerInput{long, short}, shortTip)
	if !tipsEqual(lastRollForwardTip(got[0].Served), shortTip) {
		t.Fatalf("incumbent final_tip must be the first peer fed")
	}

	// Already-correct order is left as-is.
	got = orderPeersForReplay([]format.PeerInput{short, long}, longTip)
	if !tipsEqual(lastRollForwardTip(got[1].Served), longTip) {
		t.Fatalf("already winner-last order must be preserved")
	}

	// Non-two-peer arities are returned unchanged.
	one := []format.PeerInput{short}
	if got := orderPeersForReplay(one, shortTip); len(got) != 1 {
		t.Fatalf("single-peer input must be returned unchanged")
	}
}
