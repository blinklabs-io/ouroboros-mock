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
	"context"
	"sync"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/ouroboros-mock/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestIndependentFetcherCorrelatesRangesWithoutHeaderRequests(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	blocks := buildChain(t, 3, common.Blake2b256{}, 0, 1)
	up, err := peer.NewUpstream(peer.UpstreamConfig{Blocks: blocks})
	require.NoError(t, err)
	defer up.Close()
	fetch, err := peer.NewFetcher(peer.FetcherConfig{Conn: up.Pipe()})
	require.NoError(t, err)
	defer fetch.Close()
	ctx, cancel := context.WithTimeout(t.Context(), waitTimeout)
	defer cancel()
	start := pcommon.NewPoint(blocks[0].SlotNumber(), blocks[0].Hash().Bytes())
	end := pcommon.NewPoint(blocks[2].SlotNumber(), blocks[2].Hash().Bytes())
	id, err := fetch.Request(ctx, start, end)
	require.NoError(t, err)
	require.NotZero(t, id)
	var fetched []peer.Event
	for {
		e := waitEventOn(t, fetch.Events(), peer.EventBlockFetched)
		fetched = append(fetched, e)
		if len(fetched) == 3 {
			break
		}
	}
	for i, e := range fetched {
		require.Equal(t, id, e.Request)
		require.Equal(t, uint64(1), e.Session)
		require.Equal(t, blocks[i].Hash().Bytes(), e.Points[0].Hash)
		require.Equal(t, uint64(1), e.Count)
		require.False(t, e.OccurredAt.IsZero())
		require.NotEmpty(t, e.Connection)
	}
	completion := waitEventOn(t, fetch.Events(), peer.EventFetchCompleted)
	require.Equal(t, id, completion.Request)
	require.NoError(t, completion.Err)
	done := make(chan struct{})
	go func() {
		for range fetch.Events() {
		}
		close(done)
	}()
	stats, err := fetch.Finish(ctx)
	require.NoError(t, err)
	require.True(t, stats.Complete())
	<-done
	var serverEvents []peer.Event
	drained := make(chan struct{})
	go func() {
		for e := range up.Events() {
			serverEvents = append(serverEvents, e)
		}
		close(drained)
	}()
	stats, err = up.Finish(ctx)
	require.NoError(t, err)
	require.True(t, stats.Complete())
	<-drained
	sent := 0
	for _, e := range serverEvents {
		require.NotEqual(t, peer.EventRequestNext, e.Kind)
		if e.Kind == peer.EventBlockSent {
			sent++
			require.NotZero(t, e.Request)
		}
		if e.Kind == peer.EventBatchDone {
			require.Equal(t, uint64(3), e.Count)
			require.Greater(t, end.Slot-start.Slot, uint64(3))
		}
	}
	require.Equal(t, 3, sent)
}

func TestProducerAndFinalizationQuiesceWithoutLosingRecords(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	blocks := buildChain(t, 4, common.Blake2b256{}, 0, 1)
	up, err := peer.NewUpstream(peer.UpstreamConfig{Blocks: blocks[:1]})
	require.NoError(t, err)
	follower, err := peer.NewFollower(peer.FollowerConfig{Conn: up.Pipe()})
	require.NoError(t, err)
	defer follower.Close()
	waitEventOn(t, follower.Events(), peer.EventRollForward)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range up.Events() {
		}
	}()
	go func() {
		defer wg.Done()
		for range follower.Events() {
		}
	}()
	producer := make(chan error, 1)
	go func() { producer <- up.Append(blocks[1:]...) }()
	ctx, cancel := context.WithTimeout(t.Context(), waitTimeout)
	defer cancel()
	_, err = follower.Finish(ctx)
	require.NoError(t, err)
	err = <-producer
	require.NoError(t, err)
	stats, err := up.Finish(ctx)
	require.NoError(t, err)
	require.True(t, stats.Complete())
	wg.Wait()
}
