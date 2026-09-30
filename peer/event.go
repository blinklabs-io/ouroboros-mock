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
// Every peer publishes [Event] values on a channel that never drops and never
// blocks the protocol, so tests wait on the event they expect instead of
// sleeping.
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
)

// Event describes one observable step of a peer.
type Event struct {
	Kind EventKind
	// Session identifies the connection the event belongs to.
	Session uint64
	Points  []pcommon.Point
	// Err is set on EventSessionClosed when the connection ended with an
	// error.
	Err error
}

// eventStream is an unbounded, ordered event queue drained onto a channel.
// Publishing never blocks, so a protocol callback can report an event without
// waiting on the test.
type eventStream struct {
	mu     sync.Mutex
	queue  []Event
	wake   chan struct{}
	out    chan Event
	done   chan struct{}
	closed bool
	once   sync.Once
}

func newEventStream() *eventStream {
	s := &eventStream{
		wake: make(chan struct{}, 1),
		out:  make(chan Event),
		done: make(chan struct{}),
	}
	go s.pump()
	return s
}

func (s *eventStream) publish(e Event) {
	s.mu.Lock()
	if s.closed {
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

func (s *eventStream) pump() {
	defer close(s.out)
	for {
		s.mu.Lock()
		var e Event
		have := len(s.queue) > 0
		if have {
			e = s.queue[0]
			s.queue = s.queue[1:]
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
	})
}
