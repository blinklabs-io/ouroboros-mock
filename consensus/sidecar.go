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

// Package consensus contains the capture-sidecar runtime, scenario
// orchestration helpers, and JSON test-vector format for the
// consensus-conformance harness. See README.md for the shared-base +
// per-scenario directory layout and how to add a scenario.
package consensus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

// Config drives a single capture run. Mirrors the cmd/capture-sidecar
// CLI flags 1:1.
type Config struct {
	// Address is the cardano-node NtN TCP endpoint to dial
	// (host:port).
	Address string
	// NetworkMagic identifies the testnet (per the scenario's
	// testnet.yaml).
	NetworkMagic uint32
	// ConversationPath is the path to the scenario's
	// capture-conversation.json.
	ConversationPath string
	// OutputPath is the path where the resulting JSON vector is
	// written.
	OutputPath string
	// PeerID is the peer id stamped on the recorder's PeerInput.
	// Defaults to 0 for the single-peer capture scenarios; the
	// multi-peer composer assigns one per peer.
	PeerID uint64
	// DialTimeout bounds how long Dial waits to establish the TCP
	// connection. Defaults to 10s when zero.
	DialTimeout time.Duration
	// Title is the vector's top-level title. Defaults to the
	// conversation's Name when zero.
	Title string
}

// Sidecar holds the live capture state. One Sidecar per cardano-node
// connection. Not safe for concurrent use beyond the gouroboros
// callbacks, which run on their own goroutines and serialise through
// the recorder's mutex.
type Sidecar struct {
	cfg          Config
	conversation Conversation

	conn     *ouroboros.Connection
	recorder *Recorder
}

// NewSidecar wires up a Sidecar from a Config and a parsed
// conversation script. The TCP connection is not opened here — call
// Connect for that.
func NewSidecar(cfg Config, conv Conversation) *Sidecar {
	return &Sidecar{
		cfg:          cfg,
		conversation: conv,
		recorder:     NewRecorder(cfg.PeerID),
	}
}

// Connect dials cardano-node over TCP, runs the gouroboros handshake,
// and registers chainsync callbacks on the resulting connection.
func (s *Sidecar) Connect() error {
	timeout := s.cfg.DialTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	raw, err := dialer.DialContext(ctx, "tcp", s.cfg.Address)
	if err != nil {
		return fmt.Errorf("dial %s: %w", s.cfg.Address, err)
	}

	chainSyncCfg := chainsync.NewConfig(
		// PipelineLimit=1 keeps the chainsync syncLoop's
		// "request, wait, request" cadence to a single in-flight
		// message at a time. That makes the per-step capture
		// counting deterministic — each RequestNextStep maps to
		// exactly one incoming RollForward/RollBackward. Without
		// this, gouroboros's default pipeline depth (75) can
		// burst-receive enough messages that the per-step
		// "wait for next" semantic stops corresponding to one
		// served block.
		chainsync.WithPipelineLimit(1),
		chainsync.WithRollForwardRawFunc(s.recorder.OnRollForwardRaw),
		chainsync.WithRollBackwardFunc(s.recorder.OnRollBackward),
	)

	oConn, err := ouroboros.NewConnection(
		ouroboros.WithConnection(raw),
		ouroboros.WithNetworkMagic(s.cfg.NetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithKeepAlive(true),
		ouroboros.WithChainSyncConfig(chainSyncCfg),
	)
	if err != nil {
		_ = raw.Close()
		return fmt.Errorf("ouroboros handshake: %w", err)
	}
	s.conn = oConn
	return nil
}

// Run executes every conversation step in order and returns. It
// does NOT close the underlying NtN connection — callers must call
// Sidecar.Close() (typically in a defer) to release it. Returning
// early on a step error or ctx cancellation also leaves the
// connection open for the caller's deferred Close.
func (s *Sidecar) Run(ctx context.Context) error {
	if s.conn == nil {
		return errors.New("sidecar: Connect must be called before Run")
	}
	for i, step := range s.conversation.Steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := step.Run(ctx, s); err != nil {
			return fmt.Errorf(
				"step[%d] (%s): %w", i, step.Type(), err,
			)
		}
	}
	return nil
}

// Vector builds a category=consensus TestVector from the recorded
// state. One Sidecar emits exactly one peer's worth of recordings,
// so the returned vector always has len(Capture.Peers) == 1. For
// multi-peer scenarios, run one Sidecar per upstream peer and merge
// the resulting single-peer vectors with cmd/compose-consensus-vector
// (which assigns peer_id by argument order and lifts the observation
// node's served trace into ExpectedOutput.DownstreamChainSync).
//
// For a single-peer capture, ExpectedOutput.DownstreamChainSync
// mirrors the peer's served trace verbatim (no chain-selection
// ambiguity with one upstream); the composer replaces this with the
// observation-node capture.
func (s *Sidecar) Vector() format.TestVector {
	served := s.recorder.Snapshot()
	final := lastRollForwardTip(served)
	title := s.cfg.Title
	if title == "" {
		title = s.conversation.Name
	}
	return format.TestVector{
		SchemaVersion: format.CurrentSchemaVersion,
		Title:         title,
		Category:      format.CategoryConsensus,
		Capture: &format.ConsensusCapture{
			Peers: []format.PeerInput{
				{PeerID: s.recorder.PeerID(), Served: served},
			},
			ExpectedOutput: format.ExpectedOutput{
				DownstreamChainSync: served,
				FinalTip:            final,
			},
		},
	}
}

// Close stops the gouroboros protocol clients and releases the TCP
// connection. Safe to call multiple times.
func (s *Sidecar) Close() error {
	var firstErr error
	if s.conn != nil {
		// Best-effort: Stop the chainsync client cleanly so the
		// MsgDone is sent. Errors are recorded but do not prevent
		// further cleanup.
		if err := s.conn.ChainSync().Client.Stop(); err != nil {
			firstErr = fmt.Errorf("chainsync stop: %w", err)
		}
		if err := s.conn.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("conn close: %w", err)
		}
		s.conn = nil
	}
	return firstErr
}

// lastRollForwardTip walks the served list to find the most recent
// RollForward and returns its Tip. Returns the zero Tip when no
// RollForward has been observed. Used as the structured chain-state
// sanity check baked into ExpectedOutput.FinalTip.
func lastRollForwardTip(served []format.ServedMessage) format.Tip {
	for _, m := range slices.Backward(served) {
		if m.Protocol != format.ProtocolChainSync ||
			m.MsgType != format.ChainSyncMsgRollForward ||
			m.Tip == nil {
			continue
		}
		return format.Tip{
			Slot:        m.Tip.Slot,
			Hash:        append(format.HexBytes(nil), m.Tip.Hash...),
			BlockNumber: m.Tip.BlockNumber,
		}
	}
	return format.Tip{}
}

// assertObservationPickedLongestPeer confirms that finalTip (the
// observation's last roll_forward tip) matches one of the per-peer tips
// with the highest block_number — i.e. the observation node really did
// settle on a longest chain.
//
// A multi-way tie at the top block_number is accepted, not rejected:
// Praos breaks such ties by VRF, and while this format does not encode
// the VRF, the oracle (cardano-node) already resolved the tie when it
// produced finalTip. We trust that resolution and only require that
// finalTip matches one of the tied longest peers. The replay then checks
// that the SUT independently reaches the same finalTip (BestTip ==
// final_tip), which is the conformance assertion that actually bites.
//
// A finalTip that is a SHORTER, non-longest peer is also accepted, but
// only as an exceeds-k no-switch: securityParam must be > 0 and switching
// to every longer competing peer must require rolling back MORE than
// securityParam blocks from finalTip — i.e. finalTip.BlockNumber minus the
// block number of the common ancestor with that peer exceeds k. The bound
// is rollback DEPTH, not tip-length lead: a chain that is much longer but
// forks only a block or two back is reachable within k and a conformant
// node adopts it, while a chain only slightly longer that forks beyond k is
// refused. A non-longest finalTip that a longer peer could be reached from
// within a k-deep rollback is rejected as a wrong-selector vector.
//
// Single-peer vectors trivially satisfy the invariant (the lone
// peer's tip is both the max and what observation served).
func assertObservationPickedLongestPeer(
	peers []format.PeerInput, finalTip format.Tip, securityParam uint64,
) error {
	if len(peers) == 0 {
		return errors.New("no peers in vector")
	}
	maxBlock := uint64(0)
	winners := make([]int, 0, 1)
	winnerTips := make([]format.Tip, 0, 1)
	for i, p := range peers {
		tip := lastRollForwardTip(p.Served)
		if !servedHasRollForward(p.Served) {
			return fmt.Errorf(
				"peers[%d] (peer_id=%d): served trace has no roll_forward",
				i, p.PeerID,
			)
		}
		switch {
		case tip.BlockNumber > maxBlock:
			maxBlock = tip.BlockNumber
			winners = winners[:0]
			winnerTips = winnerTips[:0]
			winners = append(winners, i)
			winnerTips = append(winnerTips, tip)
		case tip.BlockNumber == maxBlock:
			winners = append(winners, i)
			winnerTips = append(winnerTips, tip)
		}
	}
	// Accept iff final_tip matches one of the longest peers. A single
	// winner is the common case; multiple winners means a VRF tie the
	// oracle already resolved (see the doc comment).
	for _, want := range winnerTips {
		if finalTip.Slot == want.Slot &&
			bytes.Equal(finalTip.Hash, want.Hash) &&
			finalTip.BlockNumber == want.BlockNumber {
			return nil
		}
	}
	var ids []uint64
	for _, i := range winners {
		ids = append(ids, peers[i].PeerID)
	}
	// final_tip is not a longest peer. The one consistent reason is an
	// exceeds-k no-switch: the oracle kept a shorter incumbent because
	// adopting a longer peer would have required rolling back more than the
	// stability window k. That holds only when final_tip matches a captured
	// peer AND *every* longer peer leads it by more than k — so the replay
	// SUT's implausibility guard (with no local_tip) rejects all of them and
	// stays on final_tip. It is not enough for the single longest peer to be
	// out of reach: if any peer were longer than final_tip but within k, the
	// SUT would switch to it and final_tip would be the wrong selection, so
	// reject that vector here rather than bless it.
	selected := "<unknown>"
	if id, ok := peerIDFor(peers, finalTip); ok {
		selected = strconv.FormatUint(id, 10)
		if securityParam > 0 &&
			everyLongerPeerNeedsDeepRollback(peers, finalTip, securityParam) {
			return nil
		}
	}
	return fmt.Errorf(
		"observation selected peer_id=%s (slot=%d block=%d), but the "+
			"longest peer(s) %v reach block_number=%d",
		selected,
		finalTip.Slot, finalTip.BlockNumber,
		ids, maxBlock,
	)
}

// everyLongerPeerNeedsDeepRollback reports whether switching from finalTip's
// chain to any longer competing peer would require rolling back more than
// securityParam blocks — so no longer chain is reachable within the k-deep
// rollback bound and keeping finalTip is the correct no-switch outcome.
//
// The rollback depth to adopt a competing peer is finalTip.BlockNumber minus
// the block number of the deepest block the two chains share (their fork
// point), NOT the tip-length lead peerBlock - finalBlock. The two diverge
// exactly when the incumbent chain is short and the fork is shallow: a peer
// that is many blocks longer but branches only a block or two back is
// adoptable within k, so finalTip would be the wrong selection and this
// returns false.
func everyLongerPeerNeedsDeepRollback(
	peers []format.PeerInput, finalTip format.Tip, securityParam uint64,
) bool {
	finalServed, ok := servedForTip(peers, finalTip)
	if !ok {
		return false
	}
	for _, p := range peers {
		tip := lastRollForwardTip(p.Served)
		if tip.BlockNumber <= finalTip.BlockNumber {
			continue // not longer than the incumbent
		}
		anc, ok := commonAncestorBlockNumber(finalServed, p.Served)
		if !ok {
			// No shared block at all: adopting the peer would roll back the
			// entire incumbent chain (depth finalTip.BlockNumber + 1), which
			// is within k only for a near-origin incumbent.
			if finalTip.BlockNumber <= securityParam {
				return false
			}
			continue
		}
		if finalTip.BlockNumber-anc <= securityParam {
			return false // longer peer reachable within k — should switch
		}
	}
	return true
}

// servedForTip returns the served trace of the peer whose last roll_forward
// tip matches t, and true; or nil,false when no peer matches.
func servedForTip(
	peers []format.PeerInput, t format.Tip,
) ([]format.ServedMessage, bool) {
	for _, p := range peers {
		pt := lastRollForwardTip(p.Served)
		if pt.Slot == t.Slot &&
			bytes.Equal(pt.Hash, t.Hash) &&
			pt.BlockNumber == t.BlockNumber {
			return p.Served, true
		}
	}
	return nil, false
}

// peerIDFor returns the PeerID of the peer whose last roll_forward
// tip matches t. The second return is false when no captured peer
// matches — observation may have served a chain none of the
// recorded peers did (e.g. a forge nondeterminism flake), in which
// case callers should format an explicit unknown-peer sentinel
// rather than misattributing the tip to peer_id=0.
func peerIDFor(peers []format.PeerInput, t format.Tip) (uint64, bool) {
	for _, p := range peers {
		pt := lastRollForwardTip(p.Served)
		if pt.Slot == t.Slot &&
			bytes.Equal(pt.Hash, t.Hash) &&
			pt.BlockNumber == t.BlockNumber {
			return p.PeerID, true
		}
	}
	return 0, false
}
