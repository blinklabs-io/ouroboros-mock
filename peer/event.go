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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"sync"
	"time"

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
	// EventBlockSent records a body-send call, with its start time and result.
	EventBlockSent
	// EventFetchRequested records the start of a client range operation, not a wire write.
	EventFetchRequested
	// EventFetchCompleted records terminal range completion.
	EventFetchCompleted
	// EventTxsOffered records actual local insertion into a transaction offer queue.
	EventTxsOffered
	// EventTxsReturned records the prepared reply to a transaction-body request.
	EventTxsReturned
	// EventTxReceiveFailed reports a failed body request with its requested identities.
	EventTxReceiveFailed
)

// Event describes one observable step of a peer.
type Event struct {
	// OccurredAt is captured at the observed operation, before queueing.
	OccurredAt time.Time
	// Sequence orders publication attempts, including dropped observations.
	Sequence uint64
	// Request identifies a range or transaction exchange within Session.
	Request uint64
	// Connection is the actual local endpoint followed by the remote endpoint.
	Connection string
	// Count is the actual operation population, never a slot difference.
	Count uint64
	// BlockNumber and Parent identify an observed header independently of the tip.
	BlockNumber uint64
	Parent      []byte
	// Size is the serialized payload size when the operation exposes it.
	Size uint64
	Kind EventKind
	// Session identifies the connection the event belongs to.
	Session uint64
	Points  []pcommon.Point
	TxIds   [][32]byte
	Txs     []Tx
	// Err reports an operation failure or the error that ended a session.
	Err error
}

// ObservationStats accounts for every attempted observation and its disposition.
type ObservationStats struct {
	Produced  uint64 `json:"produced"`
	Delivered uint64 `json:"delivered"`
	Dropped   uint64 `json:"dropped"`
	Discarded uint64 `json:"discarded"`
	Finalized bool   `json:"finalized"`
}

// Complete reports whether finalization delivered every observation.
func (s ObservationStats) Complete() bool {
	return s.Finalized && s.Produced == s.Delivered && s.Dropped == 0 && s.Discarded == 0
}

// ErrIncompleteObservations indicates loss or interrupted observation finalization.
var ErrIncompleteObservations = errors.New("incomplete peer observations")

const eventQueueCapacity = 256

// eventStream is a bounded, ordered event queue drained onto a channel.
type eventStream struct {
	connection string
	mu         sync.Mutex
	queue      []Event
	dropped    uint64
	produced   uint64
	delivered  uint64
	discarded  uint64
	draining   bool
	finalized  bool
	wake       chan struct{}
	out        chan Event
	done       chan struct{}
	stopped    chan struct{}
	closed     bool
	once       sync.Once
}

func newEventStream(connection ...string) *eventStream {
	s := &eventStream{
		queue:   make([]Event, 0, eventQueueCapacity),
		wake:    make(chan struct{}, 1),
		out:     make(chan Event),
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	if len(connection) > 0 {
		s.connection = connection[0]
	}
	go s.pump()
	return s
}

func (s *eventStream) publish(e Event) {
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now()
	}
	e.Parent = bytes.Clone(e.Parent)
	e.Points = slices.Clone(e.Points)
	for i := range e.Points {
		e.Points[i].Hash = bytes.Clone(e.Points[i].Hash)
	}
	e.TxIds = slices.Clone(e.TxIds)
	e.Txs = cloneTxs(e.Txs)
	s.mu.Lock()
	s.produced++
	e.Sequence = s.produced
	if e.Connection == "" {
		e.Connection = s.connection
		if s.connection != "" && e.Session == 0 {
			e.Session = 1
		}
	}
	if s.closed {
		s.discarded++
		s.mu.Unlock()
		return
	}
	if len(s.queue) == eventQueueCapacity {
		s.dropped++
		s.mu.Unlock()
		return
	}
	s.queue = append(s.queue, e)
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *eventStream) droppedEvents() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
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
			s.mu.Lock()
			finished := s.draining && len(s.queue) == 0
			if finished {
				s.finalized = true
			}
			s.mu.Unlock()
			if finished {
				return
			}
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
			s.delivered++
			s.queue[0] = Event{}
			s.queue = s.queue[1:]
			s.mu.Unlock()
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
		s.discarded += uint64(len(s.queue))
		s.queue = nil
		s.mu.Unlock()
	})
}

func (s *eventStream) stats() ObservationStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ObservationStats{Produced: s.produced, Delivered: s.delivered, Dropped: s.dropped, Discarded: s.discarded, Finalized: s.finalized}
}

func (s *eventStream) finish(ctx context.Context) (ObservationStats, error) {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		s.draining = true
	}
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
	select {
	case <-s.stopped:
		stats := s.stats()
		if !stats.Complete() {
			return stats, ErrIncompleteObservations
		}
		return stats, nil
	case <-ctx.Done():
		s.close()
		return s.stats(), errors.Join(ctx.Err(), ErrIncompleteObservations)
	}
}

func connectionName(conn net.Conn) string {
	if conn == nil {
		return ""
	}
	return fmt.Sprintf("%s %s", conn.LocalAddr(), conn.RemoteAddr())
}

func cloneTxs(txs []Tx) []Tx {
	out := slices.Clone(txs)
	for i := range out {
		out[i].Raw = bytes.Clone(out[i].Raw)
	}
	return out
}
