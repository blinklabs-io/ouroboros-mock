package consensus

import (
	"fmt"
	"testing"

	gledger "github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

// blkInfo is one decoded block in a peer's served chain.
type blkInfo struct {
	num  uint64
	hash string
}

// decodeServedChain decodes the real block headers in a peer's served
// roll_forward trace into an ordered list of (block_number, hash). It fails
// the test on any undecodable header — the captures are supposed to carry
// valid Conway headers.
func decodeServedChain(t *testing.T, served []format.ServedMessage) []blkInfo {
	t.Helper()
	var out []blkInfo
	for _, m := range served {
		if m.MsgType != format.ChainSyncMsgRollForward || m.Era == nil {
			continue
		}
		h, err := gledger.NewBlockHeaderFromCbor(*m.Era, m.HeaderCbor)
		if err != nil {
			t.Fatalf("decode header (era %d): %v", *m.Era, err)
		}
		out = append(out, blkInfo{
			num:  h.BlockNumber(),
			hash: fmt.Sprintf("%x", h.Hash().Bytes()),
		})
	}
	return out
}

// commonAncestorBlock returns the block number of the deepest block shared by
// two origin-anchored, index-aligned chains (the fork point), and true; or
// 0,false when they share no block.
func commonAncestorBlock(a, b []blkInfo) (uint64, bool) {
	last := -1
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i].hash != b[i].hash {
			break
		}
		last = i
	}
	if last < 0 {
		return 0, false
	}
	return a[last].num, true
}

// TestCorpusFinalTipIsRollbackConformant decodes the real block headers in
// every committed multi-peer vector and checks final_tip against Praos chain
// selection with the k-bound expressed as ROLLBACK DEPTH, not tip-length lead.
//
// A conformant Praos node, having received and validated every competing
// chain, selects the longest chain reachable within a k-deep rollback from its
// current tip. So a vector may end on a SHORTER final_tip only when every
// longer competing peer would require rolling back more than k blocks —
// finalBlock - commonAncestorBlock > k — to adopt. A shorter final_tip that a
// longer peer could be reached from within k encodes an outcome a conformant
// node would not produce.
//
// This is the check that was missing: the composer's self-consistency guard
// gated on tip lead (peerBlock - finalBlock), which admits a shallow fork with
// a long competing chain. This test decodes the headers and uses the real
// rollback depth, so it fails on exactly those drifted captures.
func TestCorpusFinalTipIsRollbackConformant(t *testing.T) {
	vectors, err := CapturedVectors()
	if err != nil {
		t.Fatalf("CapturedVectors: %v", err)
	}
	for _, cv := range vectors {
		capt := cv.Vector.Capture
		if capt == nil || len(capt.Peers) < 2 {
			continue
		}
		t.Run(cv.Name, func(t *testing.T) {
			k := capt.SecurityParam
			ft := capt.ExpectedOutput.FinalTip
			chains := map[uint64][]blkInfo{}
			maxBlock := uint64(0)
			for _, p := range capt.Peers {
				ch := decodeServedChain(t, p.Served)
				if len(ch) == 0 {
					t.Fatalf("peer %d: no decoded blocks", p.PeerID)
				}
				chains[p.PeerID] = ch
				if tip := ch[len(ch)-1]; tip.num > maxBlock {
					maxBlock = tip.num
				}
			}
			// final_tip on a longest chain is always conformant.
			if ft.BlockNumber == maxBlock {
				return
			}
			// final_tip is a shorter chain: locate it and require every
			// longer peer to be beyond a k-deep rollback.
			var ftChain []blkInfo
			for _, p := range capt.Peers {
				ch := chains[p.PeerID]
				if ch[len(ch)-1].num == ft.BlockNumber {
					ftChain = ch
				}
			}
			if ftChain == nil {
				t.Fatalf("final_tip block %d matches no peer chain", ft.BlockNumber)
			}
			for _, p := range capt.Peers {
				ch := chains[p.PeerID]
				tip := ch[len(ch)-1]
				if tip.num <= ft.BlockNumber {
					continue // not longer than the incumbent
				}
				anc, ok := commonAncestorBlock(ftChain, ch)
				if !ok {
					continue // disjoint from origin — full rollback, > k
				}
				rollback := ft.BlockNumber - anc
				if rollback <= k {
					t.Errorf(
						"NON-CONFORMANT: final_tip ends on the shorter chain "+
							"(block %d), but a longer peer reaches block %d "+
							"sharing ancestor block %d — switching is only a "+
							"%d-block rollback, within k=%d. A conformant Praos "+
							"node adopts the longer chain; this vector demands "+
							"it stay on the shorter one.",
						ft.BlockNumber, tip.num, anc, rollback, k,
					)
				}
			}
		})
	}
}
