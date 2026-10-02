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
	"errors"
	"fmt"
	"net"
	"sync"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	gchainsync "github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// ErrClosed is returned by operations on a closed peer.
var ErrClosed = errors.New("peer closed")

// UpstreamConfig configures an [Upstream].
type UpstreamConfig struct {
	// Blocks is the initial selected chain. It must be linked.
	Blocks []ledger.Block
	// NetworkMagic is the handshake network magic. Defaults to
	// [ouroboros_mock.MockNetworkMagic] when zero.
	NetworkMagic uint32
}

// Upstream is a reactive node-to-node server peer. It answers a connecting
// node's handshake, keep-alive, chain-sync and block-fetch requests from a
// [Chain], and follows the chain when it is extended or switched to a
// competing fork.
type Upstream struct {
	cfg    UpstreamConfig
	chain  *Chain
	events *eventStream

	mu       sync.Mutex
	closed   bool
	nextID   uint64
	sessions map[uint64]*session
	// handshaking holds connections still negotiating, so Close can unblock
	// a handshake whose client never arrives.
	handshaking map[net.Conn]struct{}
	listeners   []net.Listener
	// relays keeps every connection's relay so received transactions remain
	// readable after the connection ends.
	relays []*relay
	wg     sync.WaitGroup
}

// NewUpstream returns an Upstream serving cfg.Blocks. Release it with
// [Upstream.Close].
func NewUpstream(cfg UpstreamConfig) (*Upstream, error) {
	if cfg.NetworkMagic == 0 {
		cfg.NetworkMagic = ouroboros_mock.MockNetworkMagic
	}
	chain, err := NewChain(cfg.Blocks)
	if err != nil {
		return nil, err
	}
	return &Upstream{
		cfg:      cfg,
		chain:    chain,
		events:   newEventStream(),
		sessions: make(map[uint64]*session),

		handshaking: make(map[net.Conn]struct{}),
	}, nil
}

// Events returns the channel of observable events. It is closed by
// [Upstream.Close].
func (u *Upstream) Events() <-chan Event {
	return u.events.out
}

// Chain returns the chain the peer serves.
func (u *Upstream) Chain() *Chain {
	return u.chain
}

// Append extends the selected chain and answers clients waiting at the tip.
func (u *Upstream) Append(blocks ...ledger.Block) error {
	if err := u.chain.Append(blocks...); err != nil {
		return err
	}
	u.wakeSessions()
	return nil
}

// SwitchFork moves the selected chain to a competing branch and returns the
// intersection point. Clients already past the intersection receive a rollback
// to it and then the new branch; clients waiting at the old tip are answered
// without a further request.
func (u *Upstream) SwitchFork(fork []ledger.Block) (pcommon.Point, error) {
	point, err := u.chain.SwitchFork(fork)
	if err != nil {
		return pcommon.Point{}, err
	}
	u.wakeSessions()
	return point, nil
}

// Serve runs the server side of a node-to-node connection on conn. It returns
// once the handshake has completed or failed; the mini-protocols then run in
// the background until the connection ends or the peer is closed.
func (u *Upstream) Serve(conn net.Conn) error {
	u.mu.Lock()
	if u.closed {
		u.mu.Unlock()
		_ = conn.Close()
		return ErrClosed
	}
	u.nextID++
	s := &session{up: u, id: u.nextID}
	s.relay = &relay{events: u.events, session: s.id}
	u.wg.Add(1)
	u.handshaking[conn] = struct{}{}
	u.mu.Unlock()
	defer func() {
		if s.conn == nil {
			u.wg.Done()
		}
	}()

	oConn, err := ouroboros.New(
		ouroboros.WithConnection(conn),
		ouroboros.WithNetworkMagic(u.cfg.NetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithServer(true),
		ouroboros.WithChainSyncConfig(gchainsync.Config{
			FindIntersectFunc: s.findIntersect,
			RequestNextFunc:   s.requestNext,
		}),
		ouroboros.WithBlockFetchConfig(blockfetch.Config{
			RequestRangeFunc: s.requestRange,
		}),
		ouroboros.WithTxSubmissionConfig(s.relay.config()),
	)
	u.mu.Lock()
	delete(u.handshaking, conn)
	u.mu.Unlock()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("handshake: %w", err)
	}

	u.mu.Lock()
	if u.closed {
		u.mu.Unlock()
		_ = oConn.Close()
		return ErrClosed
	}
	s.conn = oConn
	u.sessions[s.id] = s
	u.relays = append(u.relays, s.relay)
	u.mu.Unlock()

	u.events.publish(Event{Kind: EventSessionStarted, Session: s.id})
	go s.watch()
	return nil
}

// Pipe returns the client end of an in-memory connection served by the peer.
// The handshake completes once the caller's client runs over the returned
// connection; a failed handshake is reported as [EventSessionClosed].
func (u *Upstream) Pipe() net.Conn {
	client, server := net.Pipe()
	u.mu.Lock()
	if u.closed {
		u.mu.Unlock()
		_ = client.Close()
		_ = server.Close()
		return client
	}
	u.wg.Add(1)
	u.mu.Unlock()
	go func() {
		defer u.wg.Done()
		if err := u.Serve(server); err != nil && !errors.Is(err, ErrClosed) {
			u.events.publish(Event{Kind: EventSessionClosed, Err: err})
		}
	}()
	return client
}

// Accept serves connections from l until the peer is closed. Close closes l.
func (u *Upstream) Accept(l net.Listener) error {
	u.mu.Lock()
	if u.closed {
		u.mu.Unlock()
		_ = l.Close()
		return ErrClosed
	}
	u.listeners = append(u.listeners, l)
	u.wg.Add(1)
	u.mu.Unlock()
	go func() {
		defer u.wg.Done()
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			u.mu.Lock()
			if u.closed {
				u.mu.Unlock()
				_ = conn.Close()
				return
			}
			u.wg.Add(1)
			u.mu.Unlock()
			go func() {
				defer u.wg.Done()
				if err := u.Serve(conn); err != nil &&
					!errors.Is(err, ErrClosed) {
					u.events.publish(
						Event{Kind: EventSessionClosed, Err: err},
					)
				}
			}()
		}
	}()
	return nil
}

// RelayedTxs returns the transactions connected nodes have delivered through
// tx-submission, in delivery order per connection.
func (u *Upstream) RelayedTxs() []Tx {
	u.mu.Lock()
	relays := make([]*relay, 0, len(u.relays))
	relays = append(relays, u.relays...)
	u.mu.Unlock()
	var out []Tx
	for _, r := range relays {
		out = append(out, r.Received()...)
	}
	return out
}

// Close ends every connection, closes accepted listeners and waits for the
// peer's goroutines. The event channel is closed; events not yet received are
// discarded.
func (u *Upstream) Close() error {
	u.mu.Lock()
	if u.closed {
		u.mu.Unlock()
		return nil
	}
	u.closed = true
	listeners := u.listeners
	sessions := make([]*session, 0, len(u.sessions))
	for _, s := range u.sessions {
		sessions = append(sessions, s)
	}
	negotiating := make([]net.Conn, 0, len(u.handshaking))
	for c := range u.handshaking {
		negotiating = append(negotiating, c)
	}
	u.mu.Unlock()
	for _, l := range listeners {
		_ = l.Close()
	}
	for _, c := range negotiating {
		_ = c.Close()
	}
	for _, s := range sessions {
		_ = s.conn.Close()
	}
	u.wg.Wait()
	u.events.close()
	return nil
}

func (u *Upstream) wakeSessions() {
	u.mu.Lock()
	sessions := make([]*session, 0, len(u.sessions))
	for _, s := range u.sessions {
		sessions = append(sessions, s)
	}
	u.mu.Unlock()
	for _, s := range sessions {
		s.wake()
	}
}

func (u *Upstream) removeSession(id uint64) {
	u.mu.Lock()
	delete(u.sessions, id)
	u.mu.Unlock()
}
