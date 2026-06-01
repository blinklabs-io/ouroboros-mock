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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

// Replayer is the surface a SUT-side adapter implements so the harness
// can drive its chain-selection logic with the captured per-peer wire
// trace. The harness owns all iteration and assertion; the adapter
// translates each call into a SUT call and projects the SUT's outputs
// back into format-package types, so the harness stays free of any
// SUT/gouroboros types.
//
// peerID is the vector's peer_id — an opaque stable handle the harness
// threads through. The adapter maps it to whatever its internal
// peer-routing type is (e.g. a gouroboros ConnectionId).
type Replayer interface {
	// RollForward delivers one captured chainsync roll_forward for
	// peerID, in trace order: era is the captured block era (== the
	// SUT handler's blockType), headerCbor the raw header bytes, tip
	// the peer's announced chain tip. A non-nil error fails the vector.
	RollForward(peerID uint64, era uint, headerCbor []byte, tip format.Tip) error

	// RollBackward delivers one captured chainsync roll_backward for
	// peerID: point is the rollback target, tip the peer's announced
	// tip after it.
	RollBackward(peerID uint64, point format.Point, tip format.Tip) error

	// Stabilize is called once after every peer's served trace has been
	// replayed, before the assertions below. The adapter drives the SUT
	// to a quiescent decision (e.g. draining an async event bus and
	// forcing a re-evaluation) so BestTip/DrainSwitchEvents do not race
	// the SUT's background work.
	Stabilize()

	// BestTip returns the SUT's selected chain tip. A false second
	// return means the SUT did not land on any peer.
	BestTip() (format.Tip, bool)

	// DrainSwitchEvents returns the SUT's fork-choice decisions emitted
	// during the replay, oldest first. The method surfaces the SUT's switch
	// endpoints; the switch-decision assertion that consumes them lives in
	// the harness (assertSwitchedToWinner).
	DrainSwitchEvents() []format.SwitchEvent
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

// RunConsensusVector replays one consensus-category vector against r and
// returns nil on conformance, or an error describing the divergence. The
// harness feeds every peer's served trace to the Replayer in order
// (roll_forward / roll_backward), calls Stabilize, then compares BestTip
// against expected_output.final_tip on slot + hash + block_number.
func RunConsensusVector(
	t *testing.T,
	v format.TestVector,
	r Replayer,
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
	// Vector self-consistency: every committed vector must satisfy the
	// longest-peer invariant, otherwise the assertion below (BestTip ==
	// final_tip) would silently bless a wrong-selector outcome for any
	// vector whose final_tip points at a non-longest peer. Catch that at
	// vector-load time with a clear error rather than masking it as a
	// SUT bug.
	if err := assertObservationPickedLongestPeer(
		v.Capture.Peers, v.Capture.ExpectedOutput.FinalTip,
		v.Capture.SecurityParam,
	); err != nil {
		return fmt.Errorf(
			"vector %q is self-inconsistent: %w", v.Title, err,
		)
	}
	return runConsensusVector(t, v.Title, v.Capture, r)
}

func runConsensusVector(
	t *testing.T,
	title string,
	capture *format.ConsensusCapture,
	r Replayer,
) error {
	t.Helper()
	fed := 0
	for _, peer := range capture.Peers {
		for _, m := range peer.Served {
			switch m.MsgType {
			case format.ChainSyncMsgRollForward:
				if m.Era == nil || m.Tip == nil {
					return fmt.Errorf(
						"%s: peer %d roll_forward missing era or tip",
						title, peer.PeerID,
					)
				}
				if err := r.RollForward(
					peer.PeerID, *m.Era, m.HeaderCbor, *m.Tip,
				); err != nil {
					return fmt.Errorf(
						"%s: peer %d roll_forward: %w",
						title, peer.PeerID, err,
					)
				}
				fed++
			case format.ChainSyncMsgRollBackward:
				if m.Point == nil || m.Tip == nil {
					return fmt.Errorf(
						"%s: peer %d roll_backward missing point or tip",
						title, peer.PeerID,
					)
				}
				if err := r.RollBackward(
					peer.PeerID, *m.Point, *m.Tip,
				); err != nil {
					return fmt.Errorf(
						"%s: peer %d roll_backward: %w",
						title, peer.PeerID, err,
					)
				}
				fed++
			default:
				// Other chainsync/blockfetch message types are not fed
				// to the selector; captured traces contain only
				// roll_forward / roll_backward today.
			}
		}
	}
	if fed == 0 {
		return fmt.Errorf(
			"%s: served traces contained no roll_forward/roll_backward",
			title,
		)
	}
	r.Stabilize()
	bestTip, ok := r.BestTip()
	if !ok {
		return fmt.Errorf(
			"%s: replayer produced no best tip", title,
		)
	}
	if err := assertTipMatches(
		bestTip, capture.ExpectedOutput.FinalTip,
	); err != nil {
		return fmt.Errorf("%s: final_tip: %w", title, err)
	}
	// Switch-decision assertion. When the vector expects a fork
	// switch, the SUT must not merely *end* on the winning chain — it must
	// have *switched* onto it off a shorter chain. This catches a SUT that
	// adopts the longest tip from the start without ever considering the
	// competing chain. The rollback *point* is not checked here: the
	// switch event carries only endpoints, and verifying the canonical
	// rollback target needs block bodies the header-only trace omits.
	if capture.ExpectedOutput.ExpectedRollback != nil {
		if err := assertSwitchedToWinner(
			r.DrainSwitchEvents(),
			capture.Peers,
			capture.ExpectedOutput.FinalTip,
		); err != nil {
			return fmt.Errorf("%s: switch decision: %w", title, err)
		}
	}
	return nil
}

// assertSwitchedToWinner verifies the SUT emitted a fork switch onto the
// winning chain (final_tip) from a different, shorter-or-equal-length peer —
// i.e. it adopted a competing chain first and then switched up to (a strictly
// longer fork) or across to (an equal-length VRF-tie winner) final_tip.
// Returns an error when no such switch is present among the drained events.
//
// The PreviousTip must belong to a non-winning peer whose block_number is <=
// final_tip's: a "switch" from a longer chain down to a shorter winner is
// nonsensical under longest-chain selection and is not accepted as evidence.
//
// Feed-order contract: the harness replays peers in slice order, so the winner
// must be fed AFTER the chain it displaces for a switch to be observable — a
// SUT fed the winner first adopts it outright and never switches. The capture
// pipeline assigns the winner the last peer slot for exactly this reason; a
// vector that expects a switch but lists the winner first would fail here
// despite a correct selector.
func assertSwitchedToWinner(
	switches []format.SwitchEvent,
	peers []format.PeerInput,
	finalTip format.Tip,
) error {
	for _, sw := range switches {
		if !tipsEqual(sw.NewTip, finalTip) {
			continue
		}
		// The chain we switched away from must be a real, non-winning peer
		// (the incumbent), and no longer than the winner.
		for _, p := range peers {
			pt := lastRollForwardTip(p.Served)
			if tipsEqual(pt, sw.PreviousTip) && !tipsEqual(pt, finalTip) &&
				pt.BlockNumber <= finalTip.BlockNumber {
				return nil
			}
		}
	}
	return errors.New(
		"SUT never switched onto the winning chain off a shorter or " +
			"equal-length peer (no ChainSwitchEvent with new_tip==final_tip " +
			"from a non-winning peer of block_number <= final_tip)",
	)
}

// assertTipMatches compares the SUT's selected tip against the vector's
// recorded final_tip on slot + hash + block number.
func assertTipMatches(got, want format.Tip) error {
	if got.Slot != want.Slot {
		return fmt.Errorf(
			"tip slot mismatch: got %d, want %d",
			got.Slot, want.Slot,
		)
	}
	if !bytes.Equal(got.Hash, want.Hash) {
		return fmt.Errorf(
			"tip hash mismatch: got %x, want %x",
			[]byte(got.Hash), []byte(want.Hash),
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

// RunAllCapturedVectors iterates the embedded captured corpus and replays
// each vector as its own subtest. newReplayer is called once per subtest with
// that vector's capture, so the SUT-side adapter can configure itself from the
// per-vector security_param and local_tip before replay — different scenarios
// are forged under different k and pre-seeded local chains, and a single fixed
// Replayer could not honour them. A fresh Replayer per subtest also keeps
// vectors isolated; sharing one would leak the previous vector's SUT state
// into the next replay. Empty corpus is a soft skip; a vacuous pass would
// silently hide a regression that removes the corpus.
func RunAllCapturedVectors(
	t *testing.T,
	newReplayer func(capture *format.ConsensusCapture) Replayer,
) {
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
			r := newReplayer(cv.Vector.Capture)
			if err := RunConsensusVector(t, cv.Vector, r); err != nil {
				t.Fatalf("%s: %v", cv.Vector.Title, err)
			}
		})
	}
}
