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
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	csmock "github.com/blinklabs-io/ouroboros-mock/chainsync"
	"github.com/blinklabs-io/ouroboros-mock/peer"
	"github.com/stretchr/testify/require"
)

func newFollower(
	t *testing.T,
	up *peer.Upstream,
	fetch bool,
) *peer.Follower {
	t.Helper()
	f, err := peer.NewFollower(peer.FollowerConfig{
		Conn:        up.Pipe(),
		FetchBlocks: fetch,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestFollowerRecordsHeadersAndBlocks(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 3, common.Blake2b256{}, 1, 100)
	up := newUpstream(t, chain)
	f := newFollower(t, up, true)

	for range chain {
		waitEventOn(t, f.Events(), peer.EventBlockFetched)
	}
	waitEventOn(t, f.Events(), peer.EventAwaitedReply)

	var want []pcommon.Point
	for _, b := range chain {
		want = append(want, csmock.PointOf(b))
	}
	require.Equal(t, want, f.Followed())
	require.Len(t, f.Headers(), len(chain))
	for _, b := range chain {
		got, ok := f.Block(csmock.PointOf(b))
		if !ok || got == nil {
			t.Fatal("fetched block is missing")
		}
		require.Equal(t, b.Cbor(), got.Cbor())
	}
	require.Empty(t, f.Rollbacks())
}

func TestFollowerWithoutFetchLeavesBlocksUnfetched(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 2, common.Blake2b256{}, 1, 100)
	up := newUpstream(t, chain)
	f := newFollower(t, up, false)

	waitEventOn(t, f.Events(), peer.EventAwaitedReply)
	require.Len(t, f.Followed(), 2)
	_, ok := f.Block(csmock.PointOf(chain[0]))
	require.False(t, ok)
}

func TestFollowerSeesForkSwitch(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 4, common.Blake2b256{}, 1, 100)
	fork := forkOf(t, chain, 1, 3)
	up := newUpstream(t, chain)
	f := newFollower(t, up, true)

	waitEventOn(t, f.Events(), peer.EventAwaitedReply)
	intersection, err := up.SwitchFork(fork)
	require.NoError(t, err)

	rb := waitEventOn(t, f.Events(), peer.EventRollBackward)
	require.Equal(t, []pcommon.Point{intersection}, rb.Points)
	for range fork {
		waitEventOn(t, f.Events(), peer.EventBlockFetched)
	}

	want := []pcommon.Point{
		csmock.PointOf(chain[0]),
		csmock.PointOf(chain[1]),
	}
	for _, b := range fork {
		want = append(want, csmock.PointOf(b))
	}
	require.Equal(t, want, f.Followed())
	require.Equal(t, []pcommon.Point{intersection}, f.Rollbacks())
	// Headers keeps the abandoned branch.
	require.Len(t, f.Headers(), len(chain)+len(fork))
}

func TestFollowerRollsBackToItsIntersection(t *testing.T) {
	t.Parallel()
	chain := buildChain(t, 4, common.Blake2b256{}, 1, 100)
	fork := forkOf(t, chain, 1, 2)
	up := newUpstream(t, chain)
	f, err := peer.NewFollower(peer.FollowerConfig{
		Conn:      up.Pipe(),
		Intersect: []pcommon.Point{csmock.PointOf(chain[1])},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	waitEventOn(t, f.Events(), peer.EventAwaitedReply)
	require.Equal(
		t,
		[]pcommon.Point{csmock.PointOf(chain[2]), csmock.PointOf(chain[3])},
		f.Followed(),
	)

	// The fork attaches at the intersection, which the follower never
	// received as a header.
	intersection, err := up.SwitchFork(fork)
	require.NoError(t, err)
	require.Equal(t, csmock.PointOf(chain[1]), intersection)
	waitEventOn(t, f.Events(), peer.EventRollBackward)
	for range fork {
		waitEventOn(t, f.Events(), peer.EventRollForward)
	}
	require.NoError(t, f.Err())
	require.Equal(
		t,
		[]pcommon.Point{csmock.PointOf(fork[0]), csmock.PointOf(fork[1])},
		f.Followed(),
	)
}
