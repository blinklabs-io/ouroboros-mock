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

package peer

import (
	"container/list"
	"errors"
	"sync"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	gchainsync "github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
)

// session is the server side of one connection.
type session struct {
	up         *Upstream
	id         uint64
	connection string
	conn       *ouroboros.Connection
	// relay collects transactions the client announces.
	relay        *relay
	relayElement *list.Element

	mu     sync.Mutex
	server *gchainsync.Server
	// path holds the points the client is known to hold, oldest first. Its
	// last entry is the client's position.
	path []pcommon.Point
	// pending is set while the client has been told to wait at the tip.
	pending         bool
	requestSequence uint64
	// intersectRollback sends the ChainDB follower's initial rollback before
	// advancing from a newly negotiated intersection.
	intersectRollback bool
}

func (s *session) emit(kind EventKind, points ...pcommon.Point) {
	s.up.events.publish(Event{Kind: kind, Session: s.id, Connection: s.connection, OccurredAt: time.Now(), Points: points})
}

// watch reports the end of the connection.
func (s *session) watch() {
	var last error
	for err := range s.conn.ErrorChan() {
		last = errors.Join(last, err)
	}
	s.relay.wg.Wait()
	s.up.removeSession(s.id)
	s.up.events.publish(
		Event{Kind: EventSessionClosed, Session: s.id, Connection: s.connection, Err: last},
	)
	s.up.wg.Done()
}

func (s *session) findIntersect(
	ctx gchainsync.CallbackContext,
	points []pcommon.Point,
) (pcommon.Point, gchainsync.Tip, error) {
	s.emit(EventFindIntersect, points...)
	point, tip, ok := s.up.chain.find(points)
	if !ok {
		s.emit(EventIntersectNotFound)
		return pcommon.Point{}, tip, gchainsync.ErrIntersectNotFound
	}
	s.mu.Lock()
	s.server = ctx.Server
	s.path = []pcommon.Point{point}
	s.pending = false
	s.intersectRollback = true
	s.mu.Unlock()
	s.emit(EventIntersectFound, point)
	return point, tip, nil
}

func (s *session) requestNext(ctx gchainsync.CallbackContext) error {
	s.emit(EventRequestNext)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.server = ctx.Server
	if len(s.path) == 0 {
		s.path = []pcommon.Point{pcommon.NewPointOrigin()}
	}
	return s.serveLocked(true)
}

// wake answers a client waiting at the tip after the chain changed.
func (s *session) wake() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.pending {
		return
	}
	// A send failure means the connection is ending, which watch reports.
	_ = s.serveLocked(false)
}

// serveLocked sends the client its next update. fromRequest is false when
// answering an earlier await-reply, which must not be repeated.
func (s *session) serveLocked(fromRequest bool) error {
	if s.intersectRollback {
		s.intersectRollback = false
		point := s.path[0]
		s.emit(EventRollBackward, point)
		return s.server.RollBackward(point, s.up.chain.Tip())
	}
	n, path := s.up.chain.follow(s.path)
	s.path = path
	switch {
	case n.rollback:
		s.pending = false
		s.emit(EventRollBackward, n.point)
		return s.server.RollBackward(n.point, n.tip)
	case n.block != nil:
		s.pending = false
		point := pointOf(n.block)
		s.path = append(s.path, point)
		s.up.events.publish(Event{Kind: EventRollForward, Count: 1, Session: s.id, Connection: s.connection, OccurredAt: time.Now(), Points: []pcommon.Point{point}, BlockNumber: n.block.BlockNumber(), Parent: n.block.PrevHash().Bytes(), Size: n.block.BlockBodySize()})
		return s.rollForward(n.block, n.tip)
	case fromRequest:
		s.pending = true
		s.emit(EventAwaitReply)
		return s.server.AwaitReply()
	}
	return nil
}

func (s *session) rollForward(block ledger.Block, tip gchainsync.Tip) error {
	// In node-to-node mode the server extracts the header from the block CBOR
	// itself.
	return s.server.RollForward(uint(block.Type()), block.Cbor(), tip)
}

func (s *session) requestRange(
	ctx blockfetch.CallbackContext,
	start, end pcommon.Point,
) error {
	occurred := time.Now()
	s.mu.Lock()
	s.requestSequence++
	request := s.requestSequence
	s.mu.Unlock()
	s.up.events.publish(Event{Kind: EventRequestRange, OccurredAt: occurred, Session: s.id, Connection: s.connection, Request: request, Points: []pcommon.Point{start, end}})
	blocks, ok := s.up.chain.rangeOf(start, end)
	if !ok {
		s.up.events.publish(Event{Kind: EventNoBlocks, Session: s.id, Connection: s.connection, Request: request})
		return ctx.Server.NoBlocks()
	}
	if err := ctx.Server.StartBatch(); err != nil {
		return err
	}
	for _, b := range blocks {
		sent := time.Now()
		err := ctx.Server.Block(uint(b.Type()), b.Cbor())
		s.up.events.publish(Event{Kind: EventBlockSent, Count: 1, OccurredAt: sent, Session: s.id, Connection: s.connection, Request: request, Points: []pcommon.Point{pointOf(b)}, BlockNumber: b.BlockNumber(), Parent: b.PrevHash().Bytes(), Size: uint64(len(b.Cbor())), Err: err})
		if err != nil {
			return err
		}
	}
	if err := ctx.Server.BatchDone(); err != nil {
		return err
	}
	s.up.events.publish(Event{Kind: EventBatchDone, OccurredAt: time.Now(), Session: s.id, Connection: s.connection, Request: request, Points: []pcommon.Point{start, end}, Count: uint64(len(blocks))})
	return nil
}
