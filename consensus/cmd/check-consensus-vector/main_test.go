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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
	"github.com/stretchr/testify/require"
)

// nonOriginIntersectCapture loads the single-peer origin golden and
// rewrites its leading roll_backward to a non-origin intersect point, the
// shape a FindIntersect against a mid-chain block produces. The point names a
// block that is not in the served trace, because the trace starts above it.
func nonOriginIntersectCapture(t *testing.T) *format.ConsensusCapture {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(
		"..", "..", "testdata", "captured",
		"intersect_origin_one_rollforward.json",
	))
	require.NoError(t, err)
	v, err := format.DecodeTestVector(raw)
	require.NoError(t, err)
	served := v.Capture.Peers[0].Served
	require.Equal(t, format.ChainSyncMsgRollBackward, served[0].MsgType)
	served[0].Point = &format.Point{
		Slot: 1,
		Hash: format.HexBytes(strings.Repeat("\x11", 32)),
	}
	return v.Capture
}

func TestCheckShapeSingleAcceptsNonOriginIntersect(t *testing.T) {
	t.Parallel()
	c := nonOriginIntersectCapture(t)
	require.NoError(t, checkShape(c, "single", 6, 0, 0))
}

func TestCheckShapeSingleNonOrigin(t *testing.T) {
	t.Parallel()

	t.Run("accepts a non-origin intersect", func(t *testing.T) {
		t.Parallel()
		c := nonOriginIntersectCapture(t)
		require.NoError(t, checkShape(c, "single-non-origin", 6, 0, 0))
	})

	t.Run("rejects an origin intersect", func(t *testing.T) {
		t.Parallel()
		c := nonOriginIntersectCapture(t)
		c.Peers[0].Served[0].Point = &format.Point{}
		err := checkShape(c, "single-non-origin", 6, 0, 0)
		require.ErrorContains(t, err, "origin")
	})

	t.Run("rejects a trace with no leading roll_backward", func(t *testing.T) {
		t.Parallel()
		c := nonOriginIntersectCapture(t)
		c.Peers[0].Served = c.Peers[0].Served[1:]
		err := checkShape(c, "single-non-origin", 6, 0, 0)
		require.ErrorContains(t, err, "roll_backward")
	})
}

func TestChainOfRejectsUnknownMidChainRollback(t *testing.T) {
	t.Parallel()
	c := nonOriginIntersectCapture(t)
	served := c.Peers[0].Served
	// A roll_backward after a roll_forward to a block the trace never
	// contained is a malformed trace, not an intersect anchor.
	bad := append(append([]format.ServedMessage{}, served...), served[0])
	_, err := chainOf(bad)
	require.ErrorContains(t, err, "not in the reconstructed chain")
}

func TestChainOfRejectsUnknownRollbackAfterOriginRollback(t *testing.T) {
	t.Parallel()
	c := nonOriginIntersectCapture(t)
	served := c.Peers[0].Served
	origin := format.ServedMessage{
		Protocol: served[0].Protocol,
		MsgType:  format.ChainSyncMsgRollBackward,
		Point:    &format.Point{},
	}
	// The origin rollback empties the chain, which must not turn the
	// unknown intersect point that follows into a valid anchor.
	bad := append(append([]format.ServedMessage{}, served...), origin, served[0])
	_, err := chainOf(bad)
	require.ErrorContains(t, err, "not in the reconstructed chain")
}
