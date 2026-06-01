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
	"errors"
	"fmt"
	"os"

	gledger "github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

// ComposeArgs is the input to Compose.
type ComposeArgs struct {
	// PeerCapturePaths is one path per upstream peer; the index is
	// assigned as peer_id in the resulting vector. Order matters.
	PeerCapturePaths []string
	// ObservationCapturePath is the capture taken from the observation
	// node after it has stabilized; its served trace becomes the
	// vector's expected_output.downstream_chainsync.
	ObservationCapturePath string
	// Title is the composed vector's top-level title. Defaults to
	// "multi-peer-<N>" when empty.
	Title string
	// SecurityParam (k) is the stability window the scenario was forged
	// with — the configurator knows it (e.g. k=6 in the fork_and_select
	// genesis). Zero leaves capture.security_param unset. When non-zero
	// and the winning peer leads the next-longest peer by more than k,
	// Compose also derives capture.local_tip so the replay SUT does not
	// reject the winner as implausible (see deriveLocalTip).
	SecurityParam uint64
}

// Compose merges N single-peer captures and one observation capture
// into a multi-peer consensus vector. Each input is expected to be a
// category=consensus vector with exactly one entry in
// capture.peers[]. The composer assigns peer_id = i for the i-th
// PeerCapturePath. expected_output.final_tip is derived from the last
// roll_forward in the observation capture's served trace.
//
// The composed vector is validated by the format encoder before
// return — Compose surfaces format-level violations (duplicate peer
// ids, wrong category in an input, etc.) as errors rather than
// emitting an invalid file.
func Compose(args ComposeArgs) (format.TestVector, error) {
	if len(args.PeerCapturePaths) == 0 {
		return format.TestVector{},
			errors.New("compose: at least one -peer is required")
	}
	if args.ObservationCapturePath == "" {
		return format.TestVector{},
			errors.New("compose: -observation is required")
	}

	peers := make([]format.PeerInput, 0, len(args.PeerCapturePaths))
	for i, path := range args.PeerCapturePaths {
		v, err := loadCaptureVector(path)
		if err != nil {
			return format.TestVector{}, fmt.Errorf(
				"peer[%d] %s: %w", i, path, err,
			)
		}
		if len(v.Capture.Peers) != 1 {
			return format.TestVector{}, fmt.Errorf(
				"peer[%d] %s: expected exactly one peer in input, got %d",
				i, path, len(v.Capture.Peers),
			)
		}
		peers = append(peers, format.PeerInput{
			PeerID: uint64(i), //nolint:gosec // small loop index
			Served: cloneServedSlice(v.Capture.Peers[0].Served),
		})
	}

	obs, err := loadCaptureVector(args.ObservationCapturePath)
	if err != nil {
		return format.TestVector{}, fmt.Errorf(
			"observation %s: %w", args.ObservationCapturePath, err,
		)
	}
	if len(obs.Capture.Peers) != 1 {
		return format.TestVector{}, fmt.Errorf(
			"observation %s: expected exactly one peer in input, got %d",
			args.ObservationCapturePath, len(obs.Capture.Peers),
		)
	}
	downstream := cloneServedSlice(obs.Capture.Peers[0].Served)
	if !servedHasRollForward(downstream) {
		return format.TestVector{}, fmt.Errorf(
			"observation %s: served trace has no roll_forward — "+
				"cannot derive expected_output.final_tip",
			args.ObservationCapturePath,
		)
	}
	finalTip := lastRollForwardTip(downstream)

	// Strict invariant: the observation must have selected the
	// longest peer. Any committed vector that violates this would
	// silently bless a wrong-selector outcome at replay time, so
	// refuse to write it. The failure surfaces a capture-pipeline
	// flake (observation didn't settle on the longer chain in time)
	// rather than letting it land in the corpus.
	if err := assertObservationPickedLongestPeer(
		peers, finalTip, args.SecurityParam,
	); err != nil {
		return format.TestVector{}, fmt.Errorf(
			"observation %s: %w",
			args.ObservationCapturePath, err,
		)
	}

	title := args.Title
	if title == "" {
		title = fmt.Sprintf("multi-peer-%d", len(peers))
	}

	expectedRollback, err := deriveExpectedRollback(peers, finalTip)
	if err != nil {
		return format.TestVector{}, fmt.Errorf("compose: %w", err)
	}

	vec := format.TestVector{
		SchemaVersion: format.CurrentSchemaVersion,
		Title:         title,
		Category:      format.CategoryConsensus,
		Capture: &format.ConsensusCapture{
			Peers: peers,
			ExpectedOutput: format.ExpectedOutput{
				DownstreamChainSync: downstream,
				FinalTip:            finalTip,
				ExpectedRollback:    expectedRollback,
			},
			SecurityParam: args.SecurityParam,
			LocalTip:      deriveLocalTip(peers, finalTip, args.SecurityParam),
		},
	}
	// Round-trip through the encoder so format-level invariants
	// (per-msg-type field-set, schema version, category exclusivity)
	// are enforced now rather than at first replay.
	if _, err := format.EncodeTestVector(vec); err != nil {
		return format.TestVector{}, fmt.Errorf("compose: %w", err)
	}
	return vec, nil
}

// deriveLocalTip returns the chain tip the observation node would have
// been following before it switched to the winner, or nil when no
// local_tip is needed. It is the highest-block tip among the non-winning
// peers, returned only when the winner (final_tip) leads it by more than
// k — exactly the case where the replay SUT's implausibility guard would
// otherwise reject the winner as a spoof. Returns nil when k is zero,
// there is no second peer, or the lead is within k.
func deriveLocalTip(
	peers []format.PeerInput, finalTip format.Tip, k uint64,
) *format.Tip {
	if k == 0 {
		return nil
	}
	var incumbent *format.Tip
	for i := range peers {
		tip := lastRollForwardTip(peers[i].Served)
		if tipsEqual(tip, finalTip) {
			continue // the winner
		}
		if incumbent == nil || tip.BlockNumber > incumbent.BlockNumber {
			t := tip
			incumbent = &t
		}
	}
	if incumbent == nil {
		return nil
	}
	if finalTip.BlockNumber <= incumbent.BlockNumber+k {
		return nil
	}
	return incumbent
}

func tipsEqual(a, b format.Tip) bool {
	return a.Slot == b.Slot &&
		a.BlockNumber == b.BlockNumber &&
		bytes.Equal(a.Hash, b.Hash)
}

// deriveExpectedRollback computes the fork switch the SUT should perform:
// roll back to the shared-prefix common ancestor of the winning chain
// (final_tip) and the incumbent (the highest-block non-winning peer), then
// adopt final_tip. The intersect point is the last header byte-identical
// between the two peers' roll_forward sequences. Returns nil when there is
// no incumbent (single peer) or no shared prefix (the rollback target
// would be origin, which the fork scenarios are designed to avoid).
func deriveExpectedRollback(
	peers []format.PeerInput, finalTip format.Tip,
) (*format.ExpectedRollback, error) {
	winnerIdx, incumbentIdx := -1, -1
	var incumbentBlock uint64
	for i := range peers {
		tip := lastRollForwardTip(peers[i].Served)
		if tipsEqual(tip, finalTip) {
			winnerIdx = i
			continue
		}
		if incumbentIdx == -1 || tip.BlockNumber > incumbentBlock {
			incumbentIdx = i
			incumbentBlock = tip.BlockNumber
		}
	}
	if winnerIdx < 0 || incumbentIdx < 0 {
		return nil, nil
	}
	// No-switch case (exceeds-k): final_tip is a SHORTER peer than the
	// incumbent — the oracle kept the shorter chain because adopting the
	// longer one would exceed k. There is no switch to record.
	if finalTip.BlockNumber < incumbentBlock {
		return nil, nil
	}
	w := rollForwardHeaders(peers[winnerIdx].Served)
	in := rollForwardHeaders(peers[incumbentIdx].Served)
	last := -1
	for k := 0; k < len(w) && k < len(in); k++ {
		if !bytes.Equal(w[k].cbor, in[k].cbor) {
			break
		}
		last = k
	}
	if last < 0 {
		return nil, nil
	}
	pt, err := headerPoint(w[last].era, w[last].cbor)
	if err != nil {
		return nil, fmt.Errorf("expected_rollback intersect: %w", err)
	}
	return &format.ExpectedRollback{Point: pt, Tip: finalTip}, nil
}

type servedHeader struct {
	era  uint
	cbor []byte
}

func rollForwardHeaders(served []format.ServedMessage) []servedHeader {
	// Non-nil even when served has no roll_forwards: callers index into the
	// result under a separately-tracked bound (e.g. deriveExpectedRollback's
	// last >= 0 guard), and a nil return defeats nilaway's ability to prove
	// those accesses safe.
	out := make([]servedHeader, 0, len(served))
	for _, m := range served {
		if m.MsgType != format.ChainSyncMsgRollForward || m.Era == nil {
			continue
		}
		out = append(out, servedHeader{era: *m.Era, cbor: m.HeaderCbor})
	}
	return out
}

func headerPoint(era uint, cbor []byte) (format.Point, error) {
	h, err := gledger.NewBlockHeaderFromCbor(era, cbor)
	if err != nil {
		return format.Point{}, err
	}
	return format.Point{
		Slot: h.SlotNumber(),
		Hash: format.HexBytes(h.Hash().Bytes()),
	}, nil
}

// commonAncestorBlockNumber decodes the headers of two served traces and
// returns the block number of the deepest block they share (the fork point),
// and true; or 0,false when they share no block or a header cannot be decoded.
//
// The traces produced by the capture pipeline are origin-anchored and
// index-aligned (block i sits at index i in every peer that has it), so an
// index-wise CBOR comparison locates the fork. This is the same prefix scan
// deriveExpectedRollback uses; it is factored out here so the chain-selection
// guard can reason about rollback DEPTH (finalTip.BlockNumber - ancestor)
// rather than tip-length lead.
func commonAncestorBlockNumber(a, b []format.ServedMessage) (uint64, bool) {
	ha := rollForwardHeaders(a)
	hb := rollForwardHeaders(b)
	last := -1
	for i := 0; i < len(ha) && i < len(hb); i++ {
		if !bytes.Equal(ha[i].cbor, hb[i].cbor) {
			break
		}
		last = i
	}
	if last < 0 {
		return 0, false
	}
	h, err := gledger.NewBlockHeaderFromCbor(ha[last].era, ha[last].cbor)
	if err != nil {
		return 0, false
	}
	return h.BlockNumber(), true
}

// loadCaptureVector reads a JSON vector file, decodes it via
// format.DecodeTestVector, and confirms it's a consensus-category
// vector with a non-nil Capture.
func loadCaptureVector(path string) (format.TestVector, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return format.TestVector{}, err
	}
	v, err := format.DecodeTestVector(raw)
	if err != nil {
		return format.TestVector{}, err
	}
	if v.Category != format.CategoryConsensus || v.Capture == nil {
		return format.TestVector{}, fmt.Errorf(
			"expected consensus-category capture, got category=%q",
			v.Category,
		)
	}
	return v, nil
}

// cloneServedSlice deep-copies a served slice so the composed vector
// does not alias buffers owned by the loaded inputs.
func cloneServedSlice(in []format.ServedMessage) []format.ServedMessage {
	out := make([]format.ServedMessage, len(in))
	for i, m := range in {
		out[i] = cloneServedMessage(m)
	}
	return out
}
