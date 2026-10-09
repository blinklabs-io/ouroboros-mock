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
// Every peer publishes [Event] values on a bounded, ordered channel. Tests
// should consume events as the protocol runs; a stalled consumer applies
// backpressure instead of allowing event memory to grow without limit.
package peer

import (
	"sync"

	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
)

// EventKind identifies what an [Event] reports.
type EventKind int

const (
	// EventSessionStarted is published once a connection completes its
	// handshake.
	EventSessionStarted EventKind = iota + 1
	// EventSessionClosed is published when a connection ends.
	EventSessionClosed
	// EventFindIntersect is published for each intersect request; Points holds
	// the requested points.
	EventFindIntersect
	// EventIntersectFound is published when an intersect request matched;
	// Points holds the intersection.
	EventIntersectFound
	// EventIntersectNotFound is published when no requested point matched.
	EventIntersectNotFound
	// EventRequestNext is published for each request for the next update.
	EventRequestNext
	// EventAwaitReply is published when the client is told to wait at the tip.
	EventAwaitReply
	// EventRollForward is published when a header is sent; Points holds the
	// block's point.
	EventRollForward
	// EventRollBackward is published when a rollback is sent; Points holds the
	// rollback target.
	EventRollBackward
	// EventRequestRange is published for each block range request; Points
	// holds the start and end points.
	EventRequestRange
	// EventNoBlocks is published when a requested range cannot be served.
	EventNoBlocks
	// EventBatchDone is published after the blocks of a range were sent.
	EventBatchDone
	// EventAwaitedReply is published by a [Follower] when the upstream told
	// it to wait at the tip.
	EventAwaitedReply
	// EventBlockFetched is published by a [Follower] for each block it
	// fetched; Points holds the block's point.
	EventBlockFetched
	// EventTxIdsOffered is published when the remote node announced
	// transactions; TxIds holds the announced ids.
	EventTxIdsOffered
	// EventTxsReceived is published when the remote node delivered
	// transactions; Txs holds them.
	EventTxsReceived
	// EventTxIdsRequested is published by a [TxPeer] when the node asked for
	// transaction ids; TxIds holds the ids it was offered.
	EventTxIdsRequested
	// EventTxIdsBlocked is published by a [TxPeer] when it parks a blocking
	// request for transaction ids because it has nothing to announce.
	EventTxIdsBlocked
	// EventTxsRequested is published by a [TxPeer] when the node asked for
	// transaction bodies; TxIds holds the requested ids.
	EventTxsRequested
)

// Event describes one observable step of a peer.
type Event struct {
	Kind EventKind
	// Session identifies the connection the event belongs to.
	Session uint64
	Points  []pcommon.Point
	TxIds   [][32]byte
	Txs     []Tx
	// Err is set on EventSessionClosed when the connection ended with an
	// error.
	Err error
}

const eventQueueCapacity = 256

// eventStream is a bounded, ordered event queue drained onto a channel.
type eventStream struct {
	mu      sync.Mutex
	queue   []Event
	slots   chan struct{}
	wake    chan struct{}
	out     chan Event
	done    chan struct{}
	stopped chan struct{}
	closed  bool
	once    sync.Once
}

func newEventStream() *eventStream {
	s := &eventStream{
		queue:   make([]Event, 0, eventQueueCapacity),
		slots:   make(chan struct{}, eventQueueCapacity),
		wake:    make(chan struct{}, 1),
		out:     make(chan Event),
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go s.pump()
	return s
}

func (s *eventStream) publish(e Event) {
	select {
	case s.slots <- struct{}{}:
	case <-s.done:
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		<-s.slots
		return
	}
	s.queue = append(s.queue, e)
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *eventStream) pump() {
	defer func() {
		close(s.out)
		close(s.stopped)
	}()
	for {
		s.mu.Lock()
		var e Event
		have := len(s.queue) > 0
		if have {
			e = s.queue[0]
		}
		s.mu.Unlock()
		if !have {
			select {
			case <-s.wake:
				continue
			case <-s.done:
				return
			}
		}
		select {
		case s.out <- e:
			s.mu.Lock()
			s.queue[0] = Event{}
			s.queue = s.queue[1:]
			s.mu.Unlock()
			<-s.slots
		case <-s.done:
			return
		}
	}
}

// close stops the stream. Events not yet received are discarded and the
// channel is closed.
func (s *eventStream) close() {
	s.once.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		close(s.done)
		<-s.stopped
		s.mu.Lock()
		s.queue = nil
		s.mu.Unlock()
	})
}
