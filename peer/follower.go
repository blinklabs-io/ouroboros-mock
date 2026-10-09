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
	"bytes"
	"errors"
	"fmt"
	"net"
	"sync"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// FollowerConfig configures a [Follower].
type FollowerConfig struct {
	// Conn is a connection to the node to follow.
	Conn net.Conn
	// NetworkMagic is the handshake network magic. Defaults to
	// [ouroboros_mock.MockNetworkMagic] when zero.
	NetworkMagic uint32
	// Intersect lists the points to intersect at, newest first. Defaults to
	// the origin.
	Intersect []pcommon.Point
	// FetchBlocks fetches each announced block with block-fetch.
	FetchBlocks bool
}

// Follower is a downstream node-to-node peer. It follows a node with
// chain-sync, optionally fetches every announced block with block-fetch, and
// records what the node served.
type Follower struct {
	conn   *ouroboros.Connection
	events *eventStream
	cfg    FollowerConfig

	mu sync.Mutex
	// base is the intersection the node chose. It is not in chain, since the
	// node never announces it, but it is a valid rollback target.
	base pcommon.Point
	// chain is the followed chain after applying rollbacks.
	chain     []pcommon.Point
	headers   []ledger.BlockHeader
	blocks    map[string]ledger.Block
	rollbacks []pcommon.Point
	err       error
}

// NewFollower completes the handshake on cfg.Conn and starts following.
// Release it with [Follower.Close].
func NewFollower(cfg FollowerConfig) (*Follower, error) {
	if cfg.NetworkMagic == 0 {
		cfg.NetworkMagic = ouroboros_mock.MockNetworkMagic
	}
	if len(cfg.Intersect) == 0 {
		cfg.Intersect = []pcommon.Point{pcommon.NewPointOrigin()}
	}
	f := &Follower{
		cfg:    cfg,
		events: newEventStream(),
		blocks: make(map[string]ledger.Block),
	}
	bfCfg, err := blockfetch.NewConfig()
	if err != nil {
		return nil, err
	}
	conn, err := ouroboros.New(
		ouroboros.WithConnection(cfg.Conn),
		ouroboros.WithNetworkMagic(cfg.NetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithChainSyncConfig(chainsync.NewConfig(
			chainsync.WithIntersectFoundFunc(f.intersectFound),
			chainsync.WithRollForwardFunc(f.rollForward),
			chainsync.WithRollBackwardFunc(f.rollBackward),
			chainsync.WithAwaitReplyFunc(f.awaitReply),
		)),
		ouroboros.WithBlockFetchConfig(bfCfg),
	)
	if err != nil {
		_ = cfg.Conn.Close()
		f.events.close()
		return nil, fmt.Errorf("handshake: %w", err)
	}
	f.conn = conn
	if err := conn.ChainSync().Client.Sync(cfg.Intersect); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("start chain-sync: %w", err)
	}
	go f.watch()
	return f, nil
}

// Events returns the channel of observable events. It is closed by
// [Follower.Close].
func (f *Follower) Events() <-chan Event {
	return f.events.out
}

// Followed returns the points of the chain the follower currently holds:
// every announced header with rolled-back headers removed.
func (f *Follower) Followed() []pcommon.Point {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]pcommon.Point(nil), f.chain...)
}

// Headers returns every header the node announced, including ones later
// rolled back, in arrival order.
func (f *Follower) Headers() []ledger.BlockHeader {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ledger.BlockHeader(nil), f.headers...)
}

// Rollbacks returns the targets of every rollback the node sent.
func (f *Follower) Rollbacks() []pcommon.Point {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]pcommon.Point(nil), f.rollbacks...)
}

// Block returns a block fetched for point.
func (f *Follower) Block(point pcommon.Point) (ledger.Block, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.blocks[string(point.Hash)]
	return b, ok
}

// Err returns the error that ended the connection, if any.
func (f *Follower) Err() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err
}

// Close ends the connection and closes the event channel.
func (f *Follower) Close() error {
	var err error
	if f.conn != nil {
		err = f.conn.Close()
	}
	f.events.close()
	return err
}

func (f *Follower) watch() {
	for err := range f.conn.ErrorChan() {
		f.mu.Lock()
		f.err = errors.Join(f.err, err)
		f.mu.Unlock()
	}
	f.events.publish(Event{Kind: EventSessionClosed, Err: f.Err()})
}

func (f *Follower) rollForward(
	_ chainsync.CallbackContext,
	_ uint,
	data any,
	_ chainsync.Tip,
) error {
	header, ok := data.(ledger.BlockHeader)
	if !ok {
		return fmt.Errorf("unexpected roll forward payload %T", data)
	}
	point := pcommon.NewPoint(header.SlotNumber(), header.Hash().Bytes())
	f.mu.Lock()
	f.headers = append(f.headers, header)
	f.chain = append(f.chain, point)
	f.mu.Unlock()
	f.events.publish(
		Event{Kind: EventRollForward, Points: []pcommon.Point{point}},
	)
	if !f.cfg.FetchBlocks {
		return nil
	}
	block, err := f.conn.BlockFetch().Client.GetBlock(point)
	if err != nil {
		return fmt.Errorf("fetch block: %w", err)
	}
	f.mu.Lock()
	f.blocks[string(point.Hash)] = block
	f.mu.Unlock()
	f.events.publish(
		Event{Kind: EventBlockFetched, Points: []pcommon.Point{point}},
	)
	return nil
}

func (f *Follower) rollBackward(
	_ chainsync.CallbackContext,
	point pcommon.Point,
	_ chainsync.Tip,
) error {
	f.mu.Lock()
	keep := 0
	if point.Slot != f.base.Slot || !bytes.Equal(point.Hash, f.base.Hash) {
		keep = -1
		for i, p := range f.chain {
			if p.Slot == point.Slot && bytes.Equal(p.Hash, point.Hash) {
				keep = i + 1
				break
			}
		}
	}
	if keep < 0 {
		f.mu.Unlock()
		return fmt.Errorf("rollback to unknown point at slot %d", point.Slot)
	}
	f.chain = f.chain[:keep]
	f.rollbacks = append(f.rollbacks, point)
	f.mu.Unlock()
	f.events.publish(
		Event{Kind: EventRollBackward, Points: []pcommon.Point{point}},
	)
	return nil
}

func (f *Follower) intersectFound(
	_ chainsync.CallbackContext,
	point pcommon.Point,
	_ chainsync.Tip,
) error {
	f.mu.Lock()
	f.base = point
	f.chain = nil
	f.mu.Unlock()
	return nil
}

func (f *Follower) awaitReply(chainsync.CallbackContext) error {
	f.events.publish(Event{Kind: EventAwaitedReply})
	return nil
}
