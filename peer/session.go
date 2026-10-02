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
	"sync"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	gchainsync "github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
)

// session is the server side of one connection.
type session struct {
	up   *Upstream
	id   uint64
	conn *ouroboros.Connection
	// relay collects transactions the client announces.
	relay *relay

	mu     sync.Mutex
	server *gchainsync.Server
	// path holds the points the client is known to hold, oldest first. Its
	// last entry is the client's position.
	path []pcommon.Point
	// pending is set while the client has been told to wait at the tip.
	pending bool
}

func (s *session) emit(kind EventKind, points ...pcommon.Point) {
	s.up.events.publish(Event{Kind: kind, Session: s.id, Points: points})
}

// watch reports the end of the connection.
func (s *session) watch() {
	var last error
	for err := range s.conn.ErrorChan() {
		last = err
	}
	s.up.removeSession(s.id)
	s.up.events.publish(
		Event{Kind: EventSessionClosed, Session: s.id, Err: last},
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
		s.emit(EventRollForward, point)
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
	s.emit(EventRequestRange, start, end)
	blocks, ok := s.up.chain.rangeOf(start, end)
	if !ok {
		s.emit(EventNoBlocks)
		return ctx.Server.NoBlocks()
	}
	if err := ctx.Server.StartBatch(); err != nil {
		return err
	}
	for _, b := range blocks {
		if err := ctx.Server.Block(uint(b.Type()), b.Cbor()); err != nil {
			return err
		}
	}
	if err := ctx.Server.BatchDone(); err != nil {
		return err
	}
	s.emit(EventBatchDone, start, end)
	return nil
}
