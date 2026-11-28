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
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
	"github.com/blinklabs-io/ouroboros-mock/internal/lifecycle"
)

// FetcherConfig configures an independent block-fetch connection.
type FetcherConfig struct {
	Conn         net.Conn
	NetworkMagic uint32
}

type rangeCompletion struct {
	accepted  bool
	completed bool
	running   bool
	done      chan struct{}
}

// Fetcher fetches explicit ranges without delaying a header observer.
type Fetcher struct {
	conn         *ouroboros.Connection
	events       *eventStream
	watchDone    chan struct{}
	mu           sync.Mutex
	closing      bool
	requests     sync.WaitGroup
	requestsDone chan struct{}
	active       int
	ranges       map[uint64]*rangeCompletion
	closeOnce    sync.Once
	closeErr     error
}

// NewFetcher establishes a block-fetch client on cfg.Conn.
func NewFetcher(cfg FetcherConfig) (*Fetcher, error) {
	if cfg.Conn == nil {
		return nil, errors.New("fetch connection required")
	}
	if cfg.NetworkMagic == 0 {
		cfg.NetworkMagic = ouroboros_mock.MockNetworkMagic
	}
	f := &Fetcher{events: newEventStream(connectionName(cfg.Conn)), watchDone: make(chan struct{}), requestsDone: make(chan struct{}), ranges: make(map[uint64]*rangeCompletion)}
	bf, err := blockfetch.NewConfig(
		blockfetch.WithRequestPipelining(true),
		blockfetch.WithBlockFunc(func(ctx blockfetch.CallbackContext, _ uint, b ledger.Block) error {
			at := time.Now()
			f.events.publish(Event{Kind: EventBlockFetched, Count: 1, OccurredAt: at, Session: 1, Request: ctx.RequestId, Points: []pcommon.Point{pointOf(b)}, BlockNumber: b.BlockNumber(), Parent: b.PrevHash().Bytes(), Size: uint64(len(b.Cbor()))})
			return nil
		}),
		blockfetch.WithRangeDoneFunc(f.rangeDone),
	)
	if err != nil {
		f.events.close()
		_ = cfg.Conn.Close()
		return nil, err
	}
	conn, err := ouroboros.New(ouroboros.WithConnection(cfg.Conn), ouroboros.WithNetworkMagic(cfg.NetworkMagic), ouroboros.WithNodeToNode(true), ouroboros.WithKeepAlive(true), ouroboros.WithBlockFetchConfig(bf))
	if err != nil {
		f.events.close()
		_ = cfg.Conn.Close()
		return nil, fmt.Errorf("fetch handshake: %w", err)
	}
	f.conn = conn
	go func() {
		defer close(f.watchDone)
		var last error
		for err := range conn.ErrorChan() {
			last = errors.Join(last, err)
		}
		f.events.publish(Event{Kind: EventSessionClosed, Session: 1, Err: last})
	}()
	return f, nil
}

// Events returns fetch-operation observations until Close or Finish completes.
func (f *Fetcher) Events() <-chan Event { return f.events.out }

// DroppedEvents returns the number of observations lost to queue overflow.
func (f *Fetcher) DroppedEvents() uint64 { return f.events.droppedEvents() }

// Request starts a range operation. Its event timestamps API entry, not wire send.
func (f *Fetcher) Request(ctx context.Context, start, end pcommon.Point) (uint64, error) {
	at := time.Now()
	f.mu.Lock()
	if f.closing {
		f.mu.Unlock()
		return 0, ErrClosed
	}
	f.requests.Add(1)
	f.active++
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.active--
		if f.active == 0 {
			for id, r := range f.ranges {
				if !r.accepted && !r.running {
					delete(f.ranges, id)
				}
			}
		}
		f.mu.Unlock()
		f.requests.Done()
	}()
	id, err := f.conn.BlockFetch().Client.RequestRange(ctx, blockfetch.RangeRequest{Start: start, End: end})
	f.mu.Lock()
	if id != 0 {
		r := f.ranges[id]
		if r != nil && r.completed {
			delete(f.ranges, id)
		} else {
			if r == nil {
				r = &rangeCompletion{done: make(chan struct{})}
				f.ranges[id] = r
			}
			r.accepted = true
		}
	}
	f.mu.Unlock()
	f.events.publish(Event{Kind: EventFetchRequested, OccurredAt: at, Session: 1, Request: id, Points: []pcommon.Point{start, end}, Err: err})
	return id, err
}

// Close stops the connection and discards unread observations.
func (f *Fetcher) Close() error {
	err := f.stop(context.Background())
	<-f.requestsDone
	f.events.close()
	return err
}

// Finish stops the connection and drains observations while Events is consumed.
func (f *Fetcher) Finish(ctx context.Context) (ObservationStats, error) {
	proto := f.conn.BlockFetch().Client.ProtocolInstance()
	err := f.stop(ctx)
	if waitErr := waitProtocols(ctx, proto); waitErr != nil {
		f.events.close()
		return f.events.stats(), errors.Join(err, waitErr, ErrIncompleteObservations)
	}
	select {
	case <-f.requestsDone:
	case <-ctx.Done():
		f.events.close()
		return f.events.stats(), errors.Join(err, ctx.Err(), ErrIncompleteObservations)
	}
	f.mu.Lock()
	pending := make([]<-chan struct{}, 0, len(f.ranges))
	for _, r := range f.ranges {
		pending = append(pending, r.done)
	}
	f.mu.Unlock()
	for _, done := range pending {
		select {
		case <-done:
		case <-ctx.Done():
			f.events.close()
			return f.events.stats(), errors.Join(err, ctx.Err(), ErrIncompleteObservations)
		}
	}
	select {
	case <-f.watchDone:
	case <-ctx.Done():
		f.events.close()
		return f.events.stats(), errors.Join(err, ctx.Err(), ErrIncompleteObservations)
	}
	stats, drainErr := f.events.finish(ctx)
	return stats, errors.Join(err, drainErr)
}

func (f *Fetcher) stop(ctx context.Context) error {
	f.closeOnce.Do(func() {
		f.mu.Lock()
		f.closing = true
		f.mu.Unlock()
		f.closeErr = lifecycle.Close(ctx, f.conn)
		go func() { f.requests.Wait(); close(f.requestsDone) }()
	})
	return f.closeErr
}

func (f *Fetcher) rangeDone(ctx blockfetch.CallbackContext, err error) error {
	at := time.Now()
	f.mu.Lock()
	r := f.ranges[ctx.RequestId]
	if f.closing && f.active == 0 && r == nil {
		f.mu.Unlock()
		return nil
	}
	if r == nil {
		r = &rangeCompletion{done: make(chan struct{})}
		f.ranges[ctx.RequestId] = r
	}
	if r.running || r.completed {
		f.mu.Unlock()
		return errors.New("duplicate terminal range callback")
	}
	r.running = true
	f.mu.Unlock()
	f.events.publish(Event{Kind: EventFetchCompleted, OccurredAt: at, Session: 1, Request: ctx.RequestId, Err: err})
	f.mu.Lock()
	r.running = false
	r.completed = true
	close(r.done)
	if r.accepted || f.active == 0 {
		delete(f.ranges, ctx.RequestId)
	}
	f.mu.Unlock()
	return nil
}
