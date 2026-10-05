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

package peer_test

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
	csmock "github.com/blinklabs-io/ouroboros-mock/chainsync"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/blinklabs-io/ouroboros-mock/peer"
	"github.com/stretchr/testify/require"
)

const waitTimeout = 10 * time.Second

// closeTimeout is well below the 10s handshake propose timeout, so a Close
// that only returns once a stalled handshake times out is caught.
const closeTimeout = 3 * time.Second

// update is one chain-sync message received by a test client.
type update struct {
	rollback bool
	point    pcommon.Point
}

// testClient is a real gouroboros node-to-node client connected to a peer.
type testClient struct {
	conn    *ouroboros.Connection
	updates chan update
}

func newTestClient(t *testing.T, up *peer.Upstream) *testClient {
	t.Helper()
	tc := &testClient{updates: make(chan update, 64)}
	bfCfg, err := blockfetch.NewConfig()
	require.NoError(t, err)
	conn, err := ouroboros.New(
		ouroboros.WithConnection(up.Pipe()),
		ouroboros.WithNetworkMagic(ouroboros_mock.MockNetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithChainSyncConfig(chainsync.NewConfig(
			chainsync.WithPipelineLimit(0),
			chainsync.WithRollForwardFunc(
				func(
					_ chainsync.CallbackContext,
					_ uint,
					data any,
					_ chainsync.Tip,
				) error {
					header, ok := data.(ledger.BlockHeader)
					if !ok {
						return fmt.Errorf("unexpected header type %T", data)
					}
					tc.updates <- update{
						point: pcommon.NewPoint(
							header.SlotNumber(),
							header.Hash().Bytes(),
						),
					}
					return nil
				},
			),
			chainsync.WithRollBackwardFunc(
				func(
					_ chainsync.CallbackContext,
					point pcommon.Point,
					_ chainsync.Tip,
				) error {
					tc.updates <- update{rollback: true, point: point}
					return nil
				},
			),
		)),
		ouroboros.WithBlockFetchConfig(bfCfg),
	)
	require.NoError(t, err)
	tc.conn = conn
	t.Cleanup(func() { _ = conn.Close() })
	return tc
}

func (tc *testClient) next(t *testing.T) update {
	t.Helper()
	select {
	case u := <-tc.updates:
		return u
	case <-time.After(waitTimeout):
		require.FailNow(t, "timed out waiting for a chain-sync update")
	}
	return update{}
}

func (tc *testClient) expectForward(t *testing.T, blocks ...ledger.Block) {
	t.Helper()
	for _, b := range blocks {
		u := tc.next(t)
		require.False(t, u.rollback, "expected roll forward, got rollback")
		require.Equal(t, csmock.PointOf(b), u.point)
	}
}

// waitEvent returns the next event of kind from up, skipping others.
func waitEvent(
	t *testing.T,
	up *peer.Upstream,
	kind peer.EventKind,
) peer.Event {
	t.Helper()
	return waitEventOn(t, up.Events(), kind)
}

// waitEventOn returns the next event of kind from events, skipping others.
func waitEventOn(
	t *testing.T,
	events <-chan peer.Event,
	kind peer.EventKind,
) peer.Event {
	t.Helper()
	timeout := time.After(waitTimeout)
	for {
		select {
		case e, ok := <-events:
			require.True(t, ok, "event stream closed before %d", kind)
			if e.Kind == kind {
				return e
			}
		case <-timeout:
			require.FailNow(t, "timed out waiting for event", "kind %d", kind)
		}
	}
}

func buildChain(
	t *testing.T,
	count int,
	prev common.Blake2b256,
	startBlock, startSlot uint64,
) []ledger.Block {
	t.Helper()
	blocks, err := fixtures.GenerateConwayChain(
		startBlock, prev, startSlot, 10, count,
	)
	require.NoError(t, err)
	return blocks
}

// forkOf builds a competing branch that attaches after base[keep] and whose
// slots differ from the base so the hashes differ.
func forkOf(
	t *testing.T,
	base []ledger.Block,
	keep, count int,
) []ledger.Block {
	t.Helper()
	parent := base[keep]
	return buildChain(
		t,
		count,
		parent.Hash(),
		parent.BlockNumber()+1,
		parent.SlotNumber()+3,
	)
}

func newUpstream(t *testing.T, blocks []ledger.Block) *peer.Upstream {
	t.Helper()
	up, err := peer.NewUpstream(peer.UpstreamConfig{Blocks: blocks})
	require.NoError(t, err)
	t.Cleanup(func() { _ = up.Close() })
	return up
}

func TestUpstreamServesChainSyncFromOrigin(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 3, common.Blake2b256{}, 1, 100)
	up := newUpstream(t, chain)
	tc := newTestClient(t, up)

	require.NoError(
		t,
		tc.conn.ChainSync().Client.Sync([]pcommon.Point{csmock.OriginPoint()}),
	)
	tc.expectForward(t, chain...)
	waitEvent(t, up, peer.EventAwaitReply)

	// A block appended while the client waits at the tip is pushed without a
	// further request.
	more := buildChain(t, 1, chain[2].Hash(), 4, 130)
	require.NoError(t, up.Append(more...))
	tc.expectForward(t, more...)
}

func TestUpstreamForkSwitchRollsBackWaitingClient(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 5, common.Blake2b256{}, 1, 100)
	fork := forkOf(t, chain, 1, 3)
	up := newUpstream(t, chain)
	tc := newTestClient(t, up)

	require.NoError(
		t,
		tc.conn.ChainSync().Client.Sync([]pcommon.Point{csmock.OriginPoint()}),
	)
	tc.expectForward(t, chain...)
	waitEvent(t, up, peer.EventAwaitReply)

	intersection, err := up.SwitchFork(fork)
	require.NoError(t, err)
	require.Equal(t, csmock.PointOf(chain[1]), intersection)

	u := tc.next(t)
	require.True(t, u.rollback, "expected a rollback to the intersection")
	require.Equal(t, csmock.PointOf(chain[1]), u.point)
	tc.expectForward(t, fork...)
	require.Equal(t, csmock.TipOf(fork[2]), up.Chain().Tip())
}

func TestUpstreamForkAtOriginRollsBackToOrigin(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 2, common.Blake2b256{}, 1, 100)
	fork := buildChain(t, 2, common.Blake2b256{}, 1, 103)
	up := newUpstream(t, chain)
	tc := newTestClient(t, up)

	require.NoError(
		t,
		tc.conn.ChainSync().Client.Sync([]pcommon.Point{csmock.OriginPoint()}),
	)
	tc.expectForward(t, chain...)
	waitEvent(t, up, peer.EventAwaitReply)

	intersection, err := up.SwitchFork(fork)
	require.NoError(t, err)
	require.Equal(t, csmock.OriginPoint(), intersection)
	u := tc.next(t)
	require.True(t, u.rollback)
	require.Equal(t, csmock.OriginPoint(), u.point)
	tc.expectForward(t, fork...)
}

func TestUpstreamIntersectsMostRecentKnownPoint(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 4, common.Blake2b256{}, 1, 100)
	up := newUpstream(t, chain)
	tc := newTestClient(t, up)

	unknown := pcommon.NewPoint(9999, make([]byte, 32))
	require.NoError(
		t,
		tc.conn.ChainSync().Client.Sync(
			[]pcommon.Point{unknown, csmock.PointOf(chain[1])},
		),
	)
	tc.expectForward(t, chain[2:]...)
}

func TestUpstreamIntersectNotFound(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 2, common.Blake2b256{}, 1, 100)
	up := newUpstream(t, chain)
	tc := newTestClient(t, up)

	unknown := pcommon.NewPoint(9999, make([]byte, 32))
	err := tc.conn.ChainSync().Client.Sync([]pcommon.Point{unknown})
	require.ErrorIs(t, err, chainsync.ErrIntersectNotFound)
	waitEvent(t, up, peer.EventIntersectNotFound)
}

func TestUpstreamServesBlockFetchIncludingAbandonedBranch(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 4, common.Blake2b256{}, 1, 100)
	fork := forkOf(t, chain, 1, 2)
	up := newUpstream(t, chain)
	tc := newTestClient(t, up)
	bf := tc.conn.BlockFetch().Client

	got, err := bf.GetBlock(csmock.PointOf(chain[3]))
	require.NoError(t, err)
	require.Equal(t, chain[3].Hash(), got.Hash())
	require.Equal(t, chain[3].Cbor(), got.Cbor())

	_, err = up.SwitchFork(fork)
	require.NoError(t, err)

	// The fork is fetchable, and so is the branch it replaced.
	got, err = bf.GetBlock(csmock.PointOf(fork[1]))
	require.NoError(t, err)
	require.Equal(t, fork[1].Hash(), got.Hash())
	got, err = bf.GetBlock(csmock.PointOf(chain[3]))
	require.NoError(t, err)
	require.Equal(t, chain[3].Hash(), got.Hash())

	// A point no branch holds is refused.
	_, err = bf.GetBlock(pcommon.NewPoint(9999, make([]byte, 32)))
	require.Error(t, err)
	waitEvent(t, up, peer.EventNoBlocks)
}

func TestChainSwitchFork(t *testing.T) {
	t.Parallel()
	blocks := buildChain(t, 4, common.Blake2b256{}, 1, 100)
	c, err := peer.NewChain(blocks)
	require.NoError(t, err)

	t.Run("rejects a fork that does not attach", func(t *testing.T) {
		t.Parallel()
		unknown := common.NewBlake2b256(bytes.Repeat([]byte{7}, 32))
		stray := buildChain(t, 2, unknown, 9, 500)
		_, err := c.SwitchFork(stray)
		require.ErrorIs(t, err, peer.ErrNoIntersection)
	})
	t.Run("rejects an unlinked fork", func(t *testing.T) {
		t.Parallel()
		a := buildChain(t, 1, blocks[0].Hash(), 2, 103)
		b := buildChain(t, 1, blocks[0].Hash(), 3, 113)
		_, err := c.SwitchFork([]ledger.Block{a[0], b[0]})
		require.ErrorIs(t, err, peer.ErrNotLinked)
	})
	t.Run("rejects an empty fork", func(t *testing.T) {
		t.Parallel()
		_, err := c.SwitchFork(nil)
		require.ErrorIs(t, err, peer.ErrNotLinked)
	})
	t.Run("rejects an append that skips the tip", func(t *testing.T) {
		t.Parallel()
		skip := buildChain(t, 1, blocks[1].Hash(), 3, 500)
		require.ErrorIs(t, c.Append(skip...), peer.ErrNotLinked)
	})
}

func TestUpstreamAcceptServesListenerUntilClose(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 2, common.Blake2b256{}, 1, 100)
	up, err := peer.NewUpstream(peer.UpstreamConfig{Blocks: chain})
	require.NoError(t, err)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	require.NoError(t, up.Accept(l))

	conn, err := net.Dial("tcp", l.Addr().String())
	require.NoError(t, err)
	f, err := peer.NewFollower(peer.FollowerConfig{Conn: conn})
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	waitEventOn(t, f.Events(), peer.EventAwaitedReply)
	require.Len(t, f.Followed(), 2)

	closed := make(chan error, 1)
	go func() { closed <- up.Close() }()
	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(closeTimeout):
		require.FailNow(t, "Close did not return")
	}
	_, err = net.Dial("tcp", l.Addr().String())
	require.Error(t, err, "listener still accepting after Close")
	waitEventOn(t, f.Events(), peer.EventSessionClosed)
}

func TestUpstreamCloseUnblocksPendingHandshake(t *testing.T) {
	t.Parallel()
	up, err := peer.NewUpstream(peer.UpstreamConfig{})
	require.NoError(t, err)
	client := up.Pipe()
	defer func() { _ = client.Close() }()
	// A pipe write returns once the server has read it, so the server is
	// inside its handshake, waiting for the rest of the message.
	_, err = client.Write([]byte{0})
	require.NoError(t, err)

	closed := make(chan error, 1)
	go func() { closed <- up.Close() }()
	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(closeTimeout):
		require.FailNow(t, "Close did not return")
	}
}

func TestUpstreamRejectsChainChangesAfterClose(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 2, common.Blake2b256{}, 1, 100)
	up, err := peer.NewUpstream(peer.UpstreamConfig{Blocks: chain})
	require.NoError(t, err)
	require.NoError(t, up.Close())

	more := buildChain(t, 1, chain[1].Hash(), 3, 120)
	require.ErrorIs(t, up.Append(more...), peer.ErrClosed)
	require.Len(t, up.Chain().Blocks(), len(chain))

	fork := forkOf(t, chain, 0, 1)
	_, err = up.SwitchFork(fork)
	require.ErrorIs(t, err, peer.ErrClosed)
	require.Equal(t, chain, up.Chain().Blocks())
}

func TestUpstreamCloseSerializesChainChanges(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 2, common.Blake2b256{}, 1, 100)
	fork := forkOf(t, chain, 0, 1)
	appendBlocks := buildChain(t, 1, chain[1].Hash(), 3, 120)

	for _, change := range []struct {
		name string
		fn   func(*peer.Upstream) error
		want []ledger.Block
	}{
		{
			name: "append",
			fn: func(up *peer.Upstream) error {
				return up.Append(appendBlocks...)
			},
			want: append(append([]ledger.Block(nil), chain...), appendBlocks...),
		},
		{
			name: "switch fork",
			fn: func(up *peer.Upstream) error {
				_, err := up.SwitchFork(fork)
				return err
			},
			want: append(append([]ledger.Block(nil), chain[:1]...), fork...),
		},
	} {
		t.Run(change.name, func(t *testing.T) {
			t.Parallel()
			prior, err := peer.NewUpstream(peer.UpstreamConfig{Blocks: chain})
			require.NoError(t, err)
			require.NoError(t, change.fn(prior))
			priorClosed := make(chan error, 1)
			go func() { priorClosed <- prior.Close() }()
			select {
			case err := <-priorClosed:
				require.NoError(t, err)
			case <-time.After(closeTimeout):
				require.FailNow(t, "Close did not return after chain change")
			}
			require.Equal(t, change.want, prior.Chain().Blocks())

			up, err := peer.NewUpstream(peer.UpstreamConfig{Blocks: chain})
			require.NoError(t, err)

			start := make(chan struct{})
			closed := make(chan error, 1)
			changed := make(chan error, 1)
			go func() {
				<-start
				closed <- up.Close()
			}()
			go func() {
				<-start
				changed <- change.fn(up)
			}()
			close(start)

			select {
			case err := <-closed:
				require.NoError(t, err)
			case <-time.After(closeTimeout):
				require.FailNow(t, "Close did not return")
			}
			select {
			case err := <-changed:
				require.True(t, err == nil || errors.Is(err, peer.ErrClosed))
				expected := chain
				if err == nil {
					expected = change.want
				}
				require.Equal(t, expected, up.Chain().Blocks())
			case <-time.After(closeTimeout):
				require.FailNow(t, "chain change did not return")
			}
			require.ErrorIs(t, change.fn(up), peer.ErrClosed)
		})
	}
}
