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

// Package peer provides reactive node-to-node peers for exercising a node
// under test over real Ouroboros mini-protocols.
//
// [Upstream] accepts connections as a server, completes the handshake, and
// serves chain-sync and block-fetch from a fixture [Chain]. [Upstream.SwitchFork]
// moves the selected chain to a competing branch: every client is sent a
// rollback to the intersection and then the new branch, and a client parked on
// an await-reply is answered immediately. Blocks on an abandoned branch stay
// fetchable by point.
//
// [Follower] is the downstream counterpart: it follows a node with chain-sync
// and block-fetch and records what it was served. [TxPeer] offers fixture
// transactions through tx-submission and records what the node requests;
// [Upstream] collects what a node relays to it.
//
// Every peer publishes [Event] values on a bounded, ordered channel. When the
// queue is full, new events are dropped so protocol callbacks keep running.
// Each peer reports the number of dropped events through DroppedEvents.
package peer

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	mock "github.com/blinklabs-io/ouroboros-mock"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
)

func consumeEvents(events <-chan Event) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		for range events {
		}
		close(done)
	}()
	return done
}

func TestFollowerFinishWaitsForReceiveCallback(t *testing.T) {
	blocks, err := fixtures.GenerateConwayChain(1, common.Blake2b256{}, 0, 1, 1)
	require.NoError(t, err)
	up, err := NewUpstream(UpstreamConfig{Blocks: blocks})
	require.NoError(t, err)
	defer up.Close()
	f := &Follower{events: newEventStream(), watchDone: make(chan struct{})}
	entered, release := make(chan struct{}), make(chan struct{})
	conn, err := ouroboros.New(ouroboros.WithConnection(up.Pipe()), ouroboros.WithNetworkMagic(mock.MockNetworkMagic), ouroboros.WithNodeToNode(true), ouroboros.WithChainSyncConfig(chainsync.NewConfig(
		chainsync.WithRollForwardFunc(func(ctx chainsync.CallbackContext, kind uint, data any, tip chainsync.Tip) error {
			close(entered)
			<-release
			return f.rollForward(ctx, kind, data, tip)
		}),
		chainsync.WithRollBackwardFunc(f.rollBackward),
		chainsync.WithIntersectFoundFunc(f.intersectFound),
		chainsync.WithAwaitReplyFunc(f.awaitReply),
	)))
	require.NoError(t, err)
	f.conn = conn
	require.NoError(t, conn.ChainSync().Client.Sync([]pcommon.Point{pcommon.NewPointOrigin()}))
	go f.watch()
	drained := consumeEvents(f.Events())
	<-entered
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err = f.Finish(ctx)
	close(release)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorIs(t, err, ErrIncompleteObservations)
	<-conn.ChainSync().Client.ProtocolInstance().DoneChan()
	<-f.watchDone
	require.NoError(t, f.Close())
	<-drained
}

// blockedFetcher is a Fetcher with one accepted range whose terminal callback
// blocks on release once the connection is stopped.
type blockedFetcher struct {
	f            *Fetcher
	entered      chan struct{}
	release      chan struct{}
	callbackDone chan struct{}
	serverDone   chan error
	drained      <-chan struct{}
	releaseOnce  sync.Once
}

func (b *blockedFetcher) releaseCallback() { b.releaseOnce.Do(func() { close(b.release) }) }

func newBlockedFetcher(t *testing.T) *blockedFetcher {
	t.Helper()
	clientWire, serverWire := net.Pipe()
	serverDone := make(chan error, 1)
	serverRequested := make(chan struct{})
	go func() {
		server, err := ouroboros.New(ouroboros.WithConnection(serverWire), ouroboros.WithNetworkMagic(mock.MockNetworkMagic), ouroboros.WithNodeToNode(true), ouroboros.WithServer(true), ouroboros.WithBlockFetchConfig(blockfetch.Config{RequestRangeFunc: func(blockfetch.CallbackContext, pcommon.Point, pcommon.Point) error {
			close(serverRequested)
			return nil
		}}))
		if err != nil {
			serverDone <- err
			return
		}
		for range server.ErrorChan() {
		}
		serverDone <- server.Close(t.Context())
	}()
	b := &blockedFetcher{serverDone: serverDone, release: make(chan struct{}), entered: make(chan struct{}), callbackDone: make(chan struct{})}
	f := &Fetcher{events: newEventStream(), watchDone: make(chan struct{}), requestsDone: make(chan struct{}), ranges: map[uint64]*rangeCompletion{}}
	entered, release, callbackDone := b.entered, b.release, b.callbackDone
	bf, err := blockfetch.NewConfig(blockfetch.WithRequestPipelining(true), blockfetch.WithBlockFunc(func(blockfetch.CallbackContext, uint, ledger.Block) error { return nil }), blockfetch.WithRangeDoneFunc(func(ctx blockfetch.CallbackContext, err error) error {
		close(entered)
		<-release
		defer close(callbackDone)
		return f.rangeDone(ctx, err)
	}))
	require.NoError(t, err)
	conn, err := ouroboros.New(ouroboros.WithConnection(clientWire), ouroboros.WithNetworkMagic(mock.MockNetworkMagic), ouroboros.WithNodeToNode(true), ouroboros.WithBlockFetchConfig(bf))
	require.NoError(t, err)
	f.conn = conn
	go func() {
		defer close(f.watchDone)
		for range conn.ErrorChan() {
		}
	}()
	b.f = f
	b.drained = consumeEvents(f.Events())
	id, err := f.Request(t.Context(), pcommon.NewPointOrigin(), pcommon.NewPointOrigin())
	require.NoError(t, err)
	require.NotZero(t, id)
	<-serverRequested
	return b
}

func TestFetcherFinishReportsIncompleteWhileTerminalCallbackRuns(t *testing.T) {
	t.Parallel()
	b := newBlockedFetcher(t)
	defer b.releaseCallback() // before any teardown that joins the callback
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	// Stopping the connection runs the pending range's terminal callback,
	// and the stop joins it, so Finish cannot report completion while it blocks.
	_, err := b.f.Finish(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorIs(t, err, ErrIncompleteObservations)
	select {
	case <-b.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("terminal callback never ran")
	}
	select {
	case <-b.callbackDone:
		t.Fatal("terminal callback finished while it was held")
	default:
	}
	b.releaseCallback()
	<-b.callbackDone
	require.ErrorIs(t, b.f.Close(), context.DeadlineExceeded)
	<-b.drained
	require.NoError(t, <-b.serverDone)
}

func TestFetcherFinishWaitsForAsynchronousTerminalCallback(t *testing.T) {
	t.Parallel()
	b := newBlockedFetcher(t)
	defer b.releaseCallback()
	finished := make(chan error, 1)
	go func() {
		_, err := b.f.Finish(t.Context())
		finished <- err
	}()
	<-b.entered
	select {
	case err := <-finished:
		t.Fatalf("Finish returned %v while the terminal callback was held", err)
	case <-time.After(200 * time.Millisecond):
	}
	b.releaseCallback()
	require.NoError(t, <-finished)
	select {
	case <-b.callbackDone:
	default:
		t.Fatal("Finish returned before the terminal callback completed")
	}
	require.NoError(t, b.f.Close())
	<-b.drained
	require.NoError(t, <-b.serverDone)
}

func TestUpstreamFinishWaitsForAdmittedChainWake(t *testing.T) {
	blocks, err := fixtures.GenerateConwayChain(1, common.Blake2b256{}, 0, 1, 2)
	require.NoError(t, err)
	prior, err := NewUpstream(UpstreamConfig{Blocks: blocks[:1]})
	require.NoError(t, err)
	f, err := NewFollower(FollowerConfig{Conn: prior.Pipe()})
	require.NoError(t, err)
	drained := consumeEvents(f.Events())
	var closed *session
	for event := range prior.Events() {
		if event.Kind == EventAwaitReply {
			prior.mu.Lock()
			closed = prior.sessions[event.Session]
			prior.mu.Unlock()
			break
		}
	}
	require.NotNil(t, closed)
	require.NoError(t, prior.Close())
	require.NoError(t, f.Close())
	<-drained
	// Reproduce an admitted wake whose captured connection has already shut down.
	up, err := NewUpstream(UpstreamConfig{Blocks: blocks[:1]})
	require.NoError(t, err)
	defer up.Close()
	s := &session{up: up, id: 1, conn: closed.conn, server: closed.server, pending: true, path: []pcommon.Point{pointOf(blocks[0])}}
	up.sessions[1] = s
	s.mu.Lock()
	appendDone := make(chan error, 1)
	go func() { appendDone <- up.Append(blocks[1]) }()
	require.Eventually(t, func() bool { return up.chain.Tip().Point.Slot == blocks[1].SlotNumber() }, time.Second, time.Millisecond)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err = up.Finish(ctx)
	s.mu.Unlock()
	require.NoError(t, <-appendDone)
	require.NoError(t, up.Close())
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorIs(t, err, ErrIncompleteObservations)
}
