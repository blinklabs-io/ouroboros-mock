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

// Fetcher fetches explicit ranges without delaying a header observer.
type Fetcher struct {
	conn      *ouroboros.Connection
	events    *eventStream
	watchDone chan struct{}
	mu        sync.Mutex
	closing   bool
	requests  sync.WaitGroup
	closeOnce sync.Once
	closeErr  error
}

// NewFetcher establishes a block-fetch client on cfg.Conn.
func NewFetcher(cfg FetcherConfig) (*Fetcher, error) {
	if cfg.Conn == nil {
		return nil, errors.New("fetch connection required")
	}
	if cfg.NetworkMagic == 0 {
		cfg.NetworkMagic = ouroboros_mock.MockNetworkMagic
	}
	f := &Fetcher{events: newEventStream(connectionName(cfg.Conn)), watchDone: make(chan struct{})}
	bf, err := blockfetch.NewConfig(
		blockfetch.WithRequestPipelining(true),
		blockfetch.WithBlockFunc(func(ctx blockfetch.CallbackContext, _ uint, b ledger.Block) error {
			at := time.Now()
			f.events.publish(Event{Kind: EventBlockFetched, Count: 1, OccurredAt: at, Session: 1, Request: ctx.RequestId, Points: []pcommon.Point{pointOf(b)}, BlockNumber: b.BlockNumber(), Parent: b.PrevHash().Bytes(), Size: uint64(len(b.Cbor()))})
			return nil
		}),
		blockfetch.WithRangeDoneFunc(func(ctx blockfetch.CallbackContext, err error) error {
			f.events.publish(Event{Kind: EventFetchCompleted, OccurredAt: time.Now(), Session: 1, Request: ctx.RequestId, Err: err})
			return nil
		}),
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
	f.mu.Unlock()
	defer f.requests.Done()
	id, err := f.conn.BlockFetch().Client.RequestRange(ctx, blockfetch.RangeRequest{Start: start, End: end})
	f.events.publish(Event{Kind: EventFetchRequested, OccurredAt: at, Session: 1, Request: id, Points: []pcommon.Point{start, end}, Err: err})
	return id, err
}

// Close stops the connection and discards unread observations.
func (f *Fetcher) Close() error {
	err := f.stop(context.Background())
	f.events.close()
	return err
}

// Finish stops the connection and drains observations while Events is consumed.
func (f *Fetcher) Finish(ctx context.Context) (ObservationStats, error) {
	err := f.stop(ctx)
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
		f.requests.Wait()
	})
	return f.closeErr
}
