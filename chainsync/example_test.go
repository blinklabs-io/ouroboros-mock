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

package chainsync_test

import (
	"bytes"
	"sync"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	csmock "github.com/blinklabs-io/ouroboros-mock/chainsync"
)

// exampleChainServer is the kind of chain-sync server a downstream consumer
// (e.g. dingo) writes: it serves a fixed chain and tracks a per-connection read
// cursor. It implements the two chain-sync server callbacks — FindIntersect and
// RequestNext — using nothing but standard gouroboros types, so the harness can
// drive it without any knowledge of consumer-specific internals.
//
// The RequestNext behaviour follows the usual node convention: the first reply
// after an intersect is a RollBackward to the intersection point, subsequent
// replies roll the chain forward block by block, and once the chain is
// exhausted the server parks the client with AwaitReply.
type exampleChainServer struct {
	mu         sync.Mutex
	chain      csmock.Chain
	cursor     int  // index of the next block to roll forward
	rolledBack bool // whether the post-intersect rollback has been sent
}

func (s *exampleChainServer) findIntersect(
	_ chainsync.CallbackContext,
	points []pcommon.Point,
) (pcommon.Point, chainsync.Tip, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rolledBack = false
	// Match the first requested point that exists on our chain.
	for _, p := range points {
		for i, cp := range s.chain.Points {
			if pointsEqual(p, cp) {
				s.cursor = i + 1
				return cp, s.chain.Tip(), nil
			}
		}
	}
	// Otherwise intersect at origin and serve from the start of the chain.
	s.cursor = 0
	return csmock.OriginPoint(), s.chain.Tip(), nil
}

func (s *exampleChainServer) requestNext(ctx chainsync.CallbackContext) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// First reply after an intersect is a rollback to the intersection point.
	if !s.rolledBack {
		s.rolledBack = true
		point := csmock.OriginPoint()
		if s.cursor > 0 {
			point = s.chain.Points[s.cursor-1]
		}
		return ctx.Server.RollBackward(point, s.chain.Tip())
	}
	// Chain exhausted: park the client until more blocks arrive.
	if s.cursor >= s.chain.Len() {
		return ctx.Server.AwaitReply()
	}
	// Roll the next block forward.
	block := s.chain.Blocks[s.cursor]
	tip := s.chain.Tips[s.cursor]
	s.cursor++
	return ctx.Server.RollForward(uint(block.Type()), block.Cbor(), tip)
}

func pointsEqual(a, b pcommon.Point) bool {
	return a.Slot == b.Slot && bytes.Equal(a.Hash, b.Hash)
}

// TestExampleDriveChainServer is an end-to-end walkthrough of using the harness
// as an external consumer would: stand up a chain-sync server, drive it through
// a full intersect → rollback → roll-forward → await-reply sequence, and assert
// on the messages the server emits at each step.
func TestExampleDriveChainServer(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Build a small Conway chain to serve. In a real test the block builders
	// (issue #199) could substitute richer, multi-era blocks here.
	chain, err := csmock.BuildChain(1, common.Blake2b256{}, 0, 20, 3)
	require.NoError(t, err)

	srv := &exampleChainServer{chain: chain}

	h, err := csmock.New(csmock.Config{
		Mode: csmock.ModeNtC,
		ChainSync: chainsync.NewConfig(
			chainsync.WithFindIntersectFunc(srv.findIntersect),
			chainsync.WithRequestNextFunc(srv.requestNext),
		),
	})
	require.NoError(t, err)
	defer h.Close()

	// 1. Find an intersection at origin.
	require.NoError(t, h.FindIntersect([]pcommon.Point{csmock.OriginPoint()}))
	found := observe(t, h)
	require.True(t, found.IsIntersectFound())
	gotPoint, ok := found.Point()
	require.True(t, ok)
	require.Equal(t, csmock.OriginPoint(), gotPoint)

	// 2. The first RequestNext yields a rollback to the intersection point.
	require.NoError(t, h.RequestNext())
	rollback := observe(t, h)
	require.True(t, rollback.IsRollBackward())
	rbPoint, ok := rollback.Point()
	require.True(t, ok)
	require.Equal(t, csmock.OriginPoint(), rbPoint)

	// 3. Subsequent RequestNext calls roll each block forward in order.
	for i := range chain.Len() {
		require.NoError(t, h.RequestNext())
		forward := observe(t, h)
		require.True(t, forward.IsRollForward())
		_, blockCbor, tip, ok := forward.RollForwardNtC()
		require.True(t, ok)
		require.Equal(t, chain.Blocks[i].Cbor(), blockCbor)
		require.Equal(t, chain.Tips[i], tip)
	}

	// 4. With the chain exhausted, the server parks us with AwaitReply.
	require.NoError(t, h.RequestNext())
	awaitReply := observe(t, h)
	require.True(t, awaitReply.IsAwaitReply())
}
