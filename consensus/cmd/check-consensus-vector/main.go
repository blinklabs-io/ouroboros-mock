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

// Command check-consensus-vector validates that a freshly composed consensus
// vector actually has the SHAPE its scenario intends, by decoding the real
// block headers and checking the rollback depth, tip lead, fork distance, VRF
// tie, peer feed order, and local_tip/expected_rollback presence against the
// declared shape. The composer's own self-consistency check is SUT-agnostic
// and cannot enforce scenario intent (a length fork passes as a "tie", an
// unreachable switch passes as a "switch"); this gate is what makes run.sh and
// capture-all.sh safe to run as regenerators — a capture that drifts out of
// shape fails here instead of overwriting the committed golden.
//
// It depends only on ouroboros-mock + gouroboros (no SUT), so the bounds it
// checks are Praos/replay-methodology properties: rollback <= k or > k, the
// Conway restricted-tiebreaker 5-slot window, and the static-capture
// reachability limit (a winning fork's tip lead must be <= 2k for a
// k-bounded implausibility guard's local_tip catch-up to reach it).
package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"

	gledger "github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

type blk struct {
	num, slot uint64
	hash      string
}

func chainOf(served []format.ServedMessage) ([]blk, error) {
	// Non-nil so the roll_backward `out[:cut]` truncation below is provably
	// safe (nilaway rejects slicing a possibly-nil slice).
	out := make([]blk, 0, len(served))
	for _, m := range served {
		switch m.MsgType {
		case format.ChainSyncMsgRollForward:
			if m.Era == nil {
				continue
			}
			h, err := gledger.NewBlockHeaderFromCbor(*m.Era, m.HeaderCbor)
			if err != nil {
				return nil, fmt.Errorf(
					"decode header (era %d): %w",
					*m.Era,
					err,
				)
			}
			out = append(out, blk{
				h.BlockNumber(), h.SlotNumber(),
				hex.EncodeToString(h.Hash().Bytes()),
			})
		case format.ChainSyncMsgRollBackward:
			// A roll_backward rewinds the chain to its point: drop every block
			// after the one whose hash matches, and truncate to nothing for an
			// origin/empty point. A static-chain capture only carries a leading
			// roll_backward to origin (a no-op against an empty chain), but
			// honouring it keeps the reconstructed tip and rollback depth
			// correct if a trace ever contains a real mid-chain reorg.
			if m.Point == nil {
				continue
			}
			ph := hex.EncodeToString([]byte(m.Point.Hash))
			cut, found := 0, ph == "" // empty point == origin
			for i, b := range out {
				if b.hash == ph {
					cut, found = i+1, true
				}
			}
			if !found {
				// A non-empty point that names no reconstructed block is a
				// malformed trace, not an origin rollback — surface it rather
				// than silently truncating to origin and mis-judging the shape.
				return nil, fmt.Errorf(
					"roll_backward to point %s is not in the reconstructed chain",
					ph,
				)
			}
			out = out[:cut]
		}
	}
	return out, nil
}

// ancestorBlock returns the block number of the deepest block two
// origin-anchored, index-aligned chains share.
func ancestorBlock(a, b []blk) (uint64, bool) {
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

func absDiff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

func main() {
	vectorPath := flag.String("vector", "", "path to the composed vector JSON")
	shape := flag.String("shape", "",
		"expected shape: single | switch | noswitch | tie")
	k := flag.Uint64("security-param", 6, "k the vector was forged for")
	minLead := flag.Uint64("min-lead", 0,
		"minimum winner-over-incumbent block lead (0 = unset)")
	maxLead := flag.Uint64("max-lead", 0,
		"maximum lead (0 = default 2k for switch)")
	flag.Parse()

	if *vectorPath == "" || *shape == "" {
		fail("usage: check-consensus-vector -vector <path> -shape " +
			"<single|switch|noswitch|tie> [-security-param k] " +
			"[-min-lead n] [-max-lead n]")
	}
	raw, err := os.ReadFile(*vectorPath)
	if err != nil {
		fail("read vector: %v", err)
	}
	v, err := format.DecodeTestVector(raw)
	if err != nil {
		fail("decode vector: %v", err)
	}
	if v.Capture == nil {
		fail("vector has no capture")
	}
	if err := checkShape(v.Capture, *shape, *k, *minLead, *maxLead); err != nil {
		fail("shape=%s: %v", *shape, err)
	}
	fmt.Printf("OK: %q is a valid %s vector (k=%d)\n", v.Title, *shape, *k)
}

func fail(f string, args ...any) {
	fmt.Fprintf(os.Stderr, "check-consensus-vector: "+f+"\n", args...)
	os.Exit(1)
}

func checkShape(
	c *format.ConsensusCapture, shape string, k, minLead, maxLead uint64,
) error {
	if shape == "single" {
		if len(c.Peers) != 1 {
			return fmt.Errorf("expected 1 peer, got %d", len(c.Peers))
		}
		ch, err := chainOf(c.Peers[0].Served)
		if err != nil {
			return err
		}
		if len(ch) == 0 {
			return errors.New("peer has no roll_forwards")
		}
		return nil
	}

	if len(c.Peers) != 2 {
		return fmt.Errorf("expected 2 peers, got %d", len(c.Peers))
	}
	c0, err := chainOf(c.Peers[0].Served)
	if err != nil {
		return fmt.Errorf("peer0: %w", err)
	}
	c1, err := chainOf(c.Peers[1].Served)
	if err != nil {
		return fmt.Errorf("peer1: %w", err)
	}
	if len(c0) == 0 || len(c1) == 0 {
		return errors.New("a peer has no roll_forwards")
	}
	t0, t1 := c0[len(c0)-1], c1[len(c1)-1]
	anc, hasAnc := ancestorBlock(c0, c1)
	ft := c.ExpectedOutput.FinalTip
	ftHash := hex.EncodeToString([]byte(ft.Hash))
	ftPeer := -1
	if ft.BlockNumber == t0.num && ftHash == t0.hash {
		ftPeer = 0
	}
	if ft.BlockNumber == t1.num && ftHash == t1.hash {
		ftPeer = 1
	}
	if ftPeer < 0 {
		return fmt.Errorf(
			"final_tip (block %d) matches no peer tip",
			ft.BlockNumber,
		)
	}
	hasRB := c.ExpectedOutput.ExpectedRollback != nil
	hasLocal := c.LocalTip != nil
	// peer0 is always the chain fed first (incumbent / loser); the rollback to
	// switch from it to peer1 is its depth past the shared fork. When the two
	// chains share no decoded block they fork at origin (a valid case, not a
	// corrupt vector), so the rollback is the whole incumbent chain (N+1).
	rollback := t0.num + 1
	if hasAnc {
		rollback = t0.num - anc
	}
	lead := absDiff(t1.num, t0.num)

	switch shape {
	case "switch":
		if ftPeer != 1 {
			return fmt.Errorf(
				"final_tip must be peer_id 1 (winner fed last); it is peer %d",
				ftPeer)
		}
		if t1.num <= t0.num {
			return fmt.Errorf(
				"winner (peer1 block %d) must be strictly longer than the "+
					"incumbent (peer0 block %d)", t1.num, t0.num)
		}
		if rollback > k {
			return fmt.Errorf(
				"rollback to switch is %d > k=%d — that is a no-switch, not a "+
					"switch (peer0 forked %d blocks back)", rollback, k, rollback)
		}
		ml := maxLead
		if ml == 0 {
			ml = 2 * k
		}
		if lead > ml {
			return fmt.Errorf(
				"winner leads by %d > %d — beyond the SUT's local_tip catch-up "+
					"(2k); not replayable as a switch",
				lead,
				ml,
			)
		}
		if minLead > 0 && lead < minLead {
			return fmt.Errorf("winner leads by %d, below required min %d",
				lead, minLead)
		}
		if !hasRB {
			return errors.New("a switch vector must carry expected_rollback")
		}
		if (lead > k) != hasLocal {
			if lead > k {
				return fmt.Errorf(
					"lead %d > k=%d needs local_tip to arm the catch-up "+
						"relaxation, but none is set", lead, k)
			}
			return fmt.Errorf(
				"lead %d <= k=%d must not carry local_tip (none needed)",
				lead, k)
		}
		return nil

	case "noswitch":
		if ftPeer != 0 {
			return fmt.Errorf(
				"final_tip must be peer_id 0 (incumbent fed first); it is peer %d",
				ftPeer,
			)
		}
		if t1.num <= t0.num {
			return fmt.Errorf(
				"the competing peer (peer1 block %d) must be strictly longer "+
					"than the incumbent (peer0 block %d)", t1.num, t0.num)
		}
		// rollback > k is the Praos no-switch invariant: a conformant node
		// refuses to roll back more than k to adopt the longer peer.
		if rollback <= k {
			return fmt.Errorf(
				"rollback to the longer peer is %d <= k=%d — a conformant node "+
					"would switch; this is not an exceeds-k no-switch",
				rollback,
				k,
			)
		}
		// lead > k is a SEPARATE, SUT-reachability requirement, not a Praos
		// rule: the SUT (dingo) refuses the longer peer via its tip-
		// implausibility guard, which only fires when the peer's tip is more
		// than k AHEAD. With rollback > k but lead <= k the vector is still
		// Praos-conformant, yet the SUT would accept the peer's tip and switch
		// — failing replay. Because run.sh cannot run the SUT, this offline
		// gate encodes that reachability condition so a committed no-switch
		// vector is one the SUT actually reproduces.
		if lead <= k {
			return fmt.Errorf(
				"longer peer leads by %d <= k=%d — the SUT's implausibility "+
					"guard would not reject its tip, so it would switch", lead, k)
		}
		if hasRB {
			return errors.New(
				"a no-switch vector must not carry expected_rollback",
			)
		}
		if hasLocal {
			return errors.New(
				"a no-switch vector must not carry local_tip (it would arm the " +
					"catch-up and let the SUT accept the longer peer)",
			)
		}
		return nil

	case "tie":
		if ftPeer != 1 {
			return fmt.Errorf(
				"final_tip must be peer_id 1 (VRF winner fed last); it is peer %d",
				ftPeer,
			)
		}
		if t0.num != t1.num {
			return fmt.Errorf(
				"a tie requires equal block_number, got peer0=%d peer1=%d",
				t0.num, t1.num)
		}
		if t0.hash == t1.hash {
			return errors.New("tie peers have identical tips — not divergent")
		}
		if gap := absDiff(t0.slot, t1.slot); gap > 5 {
			return fmt.Errorf(
				"tie tips are %d slots apart (> 5) — the Conway restricted "+
					"tiebreaker does not arm beyond 5 slots", gap)
		}
		if rollback > k {
			return fmt.Errorf("tie rollback %d > k=%d", rollback, k)
		}
		if !hasRB {
			return errors.New(
				"a tie (VRF switch) vector must carry expected_rollback")
		}
		return nil

	default:
		return fmt.Errorf("unknown shape %q", shape)
	}
}
