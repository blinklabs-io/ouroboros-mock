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
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

// ChainSelector is the surface a replayer implements so the harness
// can drive the SUT's chain-selection logic with the per-peer tips
// it derives from a consensus vector. Implementations adapt their
// node-internal chain-selection state to these three methods.
//
// Identifier choice — peerID is the vector's peer_id, an opaque
// stable handle the harness threads through. The adapter is free to
// map it to whatever its internal peer-routing type is (e.g. a
// gouroboros ConnectionId).
type ChainSelector interface {
	// UpdatePeerTip notifies the selector that peerID has advanced
	// to tip. vrfOutput is an optional Praos tiebreaker for chains
	// that tie on block_number; nil if the vector doesn't carry
	// one (none today).
	//
	// Return value: whether the update was accepted. A false return
	// is reported back as a vector failure.
	UpdatePeerTip(peerID uint64, tip chainsync.Tip, vrfOutput []byte) bool

	// EvaluateAndSwitch forces a synchronous re-evaluation. The
	// harness calls this once after feeding every per-peer tip so
	// the assertion below does not have to sleep waiting for the
	// SUT's background evaluation loop.
	EvaluateAndSwitch()

	// BestPeerTip returns the selected chain's tip. A false second
	// return means the selector did not land on any peer.
	BestPeerTip() (chainsync.Tip, bool)
}

// LoadVector reads a JSON test vector from disk and decodes it.
func LoadVector(path string) (format.TestVector, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return format.TestVector{}, fmt.Errorf(
			"read vector %s: %w", path, err,
		)
	}
	v, err := format.DecodeTestVector(raw)
	if err != nil {
		return format.TestVector{}, fmt.Errorf(
			"decode vector %s: %w", path, err,
		)
	}
	return v, nil
}

// RunConsensusVector replays one consensus-category vector against
// sel and returns nil on conformance, or an error describing the
// divergence. For each peer in the vector the harness derives the
// last roll_forward's tip from the served trace, calls
// UpdatePeerTip, then EvaluateAndSwitch, then compares BestPeerTip
// against expected_output.final_tip on slot + hash + block_number.
func RunConsensusVector(
	t *testing.T,
	v format.TestVector,
	sel ChainSelector,
) error {
	t.Helper()
	if v.Category != format.CategoryConsensus {
		return fmt.Errorf(
			"vector %q: expected category %q, got %q",
			v.Title, format.CategoryConsensus, v.Category,
		)
	}
	if v.Capture == nil {
		return fmt.Errorf(
			"consensus vector %q has no capture", v.Title,
		)
	}
	// Vector self-consistency: every committed vector must satisfy
	// the longest-peer invariant, otherwise the assertion below
	// (BestPeerTip == final_tip) would silently bless a wrong-
	// selector outcome for any vector whose final_tip points at a
	// non-longest peer. Catch that at vector-load time with a clear
	// error rather than masking it as a SUT bug.
	if err := assertObservationPickedLongestPeer(
		v.Capture.Peers, v.Capture.ExpectedOutput.FinalTip,
	); err != nil {
		return fmt.Errorf(
			"vector %q is self-inconsistent: %w", v.Title, err,
		)
	}
	return runConsensusVector(t, v.Title, v.Capture, sel)
}

func runConsensusVector(
	t *testing.T,
	title string,
	capture *format.ConsensusCapture,
	sel ChainSelector,
) error {
	t.Helper()
	for _, peer := range capture.Peers {
		tip, ok := lastServedTip(peer.Served)
		if !ok {
			return fmt.Errorf(
				"peer %d: served trace has no roll_forward — "+
					"nothing to feed the chain selector",
				peer.PeerID,
			)
		}
		if !sel.UpdatePeerTip(peer.PeerID, tip, nil) {
			return fmt.Errorf(
				"peer %d: chain selector rejected tip update",
				peer.PeerID,
			)
		}
	}
	sel.EvaluateAndSwitch()
	bestTip, ok := sel.BestPeerTip()
	if !ok {
		return fmt.Errorf(
			"%s: chain selector produced no best peer", title,
		)
	}
	if err := assertTipMatches(
		bestTip, capture.ExpectedOutput.FinalTip,
	); err != nil {
		return fmt.Errorf("%s: final_tip: %w", title, err)
	}
	return nil
}

// lastServedTip walks served in reverse and returns the tip of the
// most recent roll_forward. Returns the captured Tip's slot + hash +
// block_number verbatim — the recorder copies all three off the
// gouroboros chainsync.Tip callback argument at capture time.
func lastServedTip(
	served []format.ServedMessage,
) (chainsync.Tip, bool) {
	for _, m := range slices.Backward(served) {
		if m.MsgType != format.ChainSyncMsgRollForward || m.Tip == nil {
			continue
		}
		return chainsync.Tip{
			Point: ocommon.Point{
				Slot: m.Tip.Slot,
				Hash: append([]byte(nil), m.Tip.Hash...),
			},
			BlockNumber: m.Tip.BlockNumber,
		}, true
	}
	return chainsync.Tip{}, false
}

// assertTipMatches compares the selected chain's tip against the
// vector's recorded final_tip on slot + hash + block number.
func assertTipMatches(got chainsync.Tip, want format.Tip) error {
	if got.Point.Slot != want.Slot {
		return fmt.Errorf(
			"tip slot mismatch: got %d, want %d",
			got.Point.Slot, want.Slot,
		)
	}
	if !bytes.Equal(got.Point.Hash, want.Hash) {
		return fmt.Errorf(
			"tip hash mismatch: got %x, want %x",
			got.Point.Hash, []byte(want.Hash),
		)
	}
	if got.BlockNumber != want.BlockNumber {
		return fmt.Errorf(
			"tip block_number mismatch: got %d, want %d",
			got.BlockNumber, want.BlockNumber,
		)
	}
	return nil
}

//go:embed testdata/captured/*.json
var capturedFS embed.FS

// CapturedVectors returns every committed consensus vector as a
// (name, vector) pair, suitable for driving t.Run subtests. The
// vectors are embedded into the binary at build time, so callers
// don't need to resolve the upstream package's on-disk layout.
//
// name is the basename of the JSON file (without extension).
func CapturedVectors() ([]CapturedVector, error) {
	const root = "testdata/captured"
	entries, err := fs.ReadDir(capturedFS, root)
	if err != nil {
		return nil, fmt.Errorf("list embedded captured vectors: %w", err)
	}
	var out []CapturedVector
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := capturedFS.ReadFile(path.Join(root, e.Name()))
		if err != nil {
			return nil, fmt.Errorf(
				"read embedded vector %s: %w", e.Name(), err,
			)
		}
		v, err := format.DecodeTestVector(raw)
		if err != nil {
			return nil, fmt.Errorf(
				"decode embedded vector %s: %w", e.Name(), err,
			)
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		out = append(out, CapturedVector{Name: name, Vector: v})
	}
	return out, nil
}

// CapturedVector pairs a vector with the base name of the file it
// came from, for surfacing in subtest names.
type CapturedVector struct {
	Name   string
	Vector format.TestVector
}

// RunAllCapturedVectors iterates the embedded captured corpus and
// replays each vector as its own subtest. newSelector is called once
// per subtest to produce a fresh ChainSelector — vectors must run in
// isolation; sharing one selector across subtests would leak the
// previous vector's per-peer tips into the next replay. Empty corpus
// is a soft skip; a vacuous pass would silently hide a regression
// that removes the corpus.
func RunAllCapturedVectors(t *testing.T, newSelector func() ChainSelector) {
	t.Helper()
	vectors, err := CapturedVectors()
	if err != nil {
		t.Fatalf("CapturedVectors: %v", err)
	}
	if len(vectors) == 0 {
		t.Skip("no captured vectors embedded")
	}
	for _, cv := range vectors {
		t.Run(cv.Name, func(t *testing.T) {
			sel := newSelector()
			if err := RunConsensusVector(t, cv.Vector, sel); err != nil {
				t.Fatalf("%s: %v", cv.Vector.Title, err)
			}
		})
	}
}
