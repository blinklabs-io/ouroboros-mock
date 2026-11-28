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
	"sync/atomic"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
	"github.com/blinklabs-io/ouroboros-mock/internal/lifecycle"
)

// TxPeerConfig configures a [TxPeer].
type TxPeerConfig struct {
	// Conn is a connection to the node under test.
	Conn net.Conn
	// NetworkMagic is the handshake network magic. Defaults to
	// [ouroboros_mock.MockNetworkMagic] when zero.
	NetworkMagic uint32
	// KeepAlive enables client keep-alive for long-lived observation sessions.
	KeepAlive bool
	// Txs are the transactions to offer initially.
	Txs []Tx
}

// TxPeer is a node-to-node peer that offers transactions through
// tx-submission and records what the node asks for.
type TxPeer struct {
	conn      *ouroboros.Connection
	events    *eventStream
	done      chan struct{}
	watchDone chan struct{}

	requestSequence atomic.Uint64
	closeOnce       sync.Once
	closeErr        error

	mu sync.Mutex
	// txs holds every offered transaction; txs[:announced] have been announced
	// to the node.
	offersClosed bool
	txs          []Tx
	announced    int
	served       []Tx
	// more is closed and replaced whenever a transaction is offered, waking a
	// blocking id request.
	more chan struct{}
}

// NewTxPeer completes the handshake on cfg.Conn and starts the tx-submission
// client. Release it with [TxPeer.Close].
func NewTxPeer(cfg TxPeerConfig) (*TxPeer, error) {
	if cfg.NetworkMagic == 0 {
		cfg.NetworkMagic = ouroboros_mock.MockNetworkMagic
	}
	offeredAt := time.Now()
	p := &TxPeer{
		events:    newEventStream(connectionName(cfg.Conn)),
		txs:       cloneTxs(cfg.Txs),
		more:      make(chan struct{}),
		done:      make(chan struct{}),
		watchDone: make(chan struct{}),
	}
	txCfg := txsubmission.Config{
		RequestTxIdsFunc: p.requestTxIds,
		RequestTxsFunc:   p.requestTxs,
	}
	conn, err := ouroboros.New(
		ouroboros.WithConnection(cfg.Conn),
		ouroboros.WithNetworkMagic(cfg.NetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithKeepAlive(cfg.KeepAlive),
		ouroboros.WithTxSubmissionConfig(txCfg),
	)
	if err != nil {
		p.events.close()
		_ = cfg.Conn.Close()
		return nil, fmt.Errorf("handshake: %w", err)
	}
	p.conn = conn
	if len(p.txs) > 0 {
		p.observeOffer(offeredAt, p.txs)
	}
	conn.TxSubmission().Client.Init()
	go p.watch()
	return p, nil
}

// Events returns the channel of observable events. It is closed by
// [TxPeer.Close].
func (p *TxPeer) Events() <-chan Event {
	return p.events.out
}

// DroppedEvents returns the number of events discarded because the event
// queue was full. Protocol progress does not depend on consuming Events.
func (p *TxPeer) DroppedEvents() uint64 {
	return p.events.droppedEvents()
}

// Offer adds transactions to announce. A node waiting on a blocking request
// is answered immediately.
func (p *TxPeer) Offer(txs ...Tx) {
	occurred := time.Now()
	txs = cloneTxs(txs)
	p.mu.Lock()
	if p.offersClosed {
		p.mu.Unlock()
		return
	}
	p.txs = append(p.txs, txs...)
	close(p.more)
	p.more = make(chan struct{})
	p.observeOffer(occurred, txs)
	p.mu.Unlock()
}

func (p *TxPeer) observeOffer(at time.Time, txs []Tx) {
	ids := make([][32]byte, len(txs))
	refs := make([]TxRef, len(txs))
	for i, tx := range txs {
		ids[i] = tx.ID
		refs[i] = TxRef{EraId: tx.EraId, ID: tx.ID}
	}
	p.events.publish(Event{Kind: EventTxsOffered, OccurredAt: at, Session: 1, Count: uint64(len(txs)), TxIds: ids, TxRefs: refs})
}

// Served returns the transactions the node requested and received, in
// delivery order.
func (p *TxPeer) Served() []Tx {
	p.mu.Lock()
	defer p.mu.Unlock()
	return cloneTxs(p.served)
}

func (p *TxPeer) stop(ctx context.Context) error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.offersClosed = true
		close(p.done)
		p.mu.Unlock()
		if p.conn != nil {
			p.closeErr = lifecycle.Close(ctx, p.conn)
		}
	})
	return p.closeErr
}

func (p *TxPeer) watch() {
	defer close(p.watchDone)
	var last error
	for err := range p.conn.ErrorChan() {
		last = errors.Join(last, err)
	}
	p.events.publish(Event{Kind: EventSessionClosed, Err: last})
}

// requestTxIds answers a node's request for ids. A non-blocking request with
// nothing to announce is answered empty; a blocking one waits for [TxPeer.Offer]
// and ends the exchange when the connection closes.
func (p *TxPeer) requestTxIds(
	ctx txsubmission.CallbackContext,
	blocking bool,
	_, req uint16,
) ([]txsubmission.TxIdAndSize, error) {
	occurred := time.Now()
	request := p.requestSequence.Add(1)
	for {
		p.mu.Lock()
		avail := p.txs[p.announced:]
		n := min(len(avail), int(req))
		if n > 0 || !blocking {
			out := make([]txsubmission.TxIdAndSize, 0, n)
			ids := make([][32]byte, 0, n)
			for _, tx := range avail[:n] {
				out = append(out, txsubmission.TxIdAndSize{
					TxId: txsubmission.TxId{EraId: tx.EraId, TxId: tx.ID},
					//nolint:gosec // fixture transactions are far below 4 GiB
					Size: uint32(len(tx.Raw)),
				})
				ids = append(ids, tx.ID)
			}
			p.announced += n
			p.mu.Unlock()
			refs := make([]TxRef, len(out))
			for i, item := range out {
				refs[i] = TxRef{EraId: item.TxId.EraId, ID: item.TxId.TxId}
			}
			p.events.publish(Event{Kind: EventTxIdsRequested, Session: 1, Request: request, OccurredAt: occurred, Count: uint64(len(ids)), TxIds: ids, TxRefs: refs})
			return out, nil
		}
		more := p.more
		p.mu.Unlock()
		p.events.publish(Event{Kind: EventTxIdsBlocked, Session: 1, Request: request})
		select {
		case <-more:
		case <-p.done:
			return nil, txsubmission.ErrStopServerProcess
		case <-ctx.DoneChan:
			return nil, txsubmission.ErrStopServerProcess
		}
	}
}

// requestTxs answers a node's request for bodies. Ids the peer never
// announced are omitted, as the protocol allows.
func (p *TxPeer) requestTxs(
	_ txsubmission.CallbackContext,
	ids []txsubmission.TxId,
) ([]txsubmission.TxBody, error) {
	occurred := time.Now()
	request := p.requestSequence.Add(1)
	requested := make([][32]byte, 0, len(ids))
	requestedRefs := make([]TxRef, 0, len(ids))
	for _, id := range ids {
		requested = append(requested, id.TxId)
		requestedRefs = append(requestedRefs, TxRef{EraId: id.EraId, ID: id.TxId})
	}
	p.events.publish(Event{Kind: EventTxsRequested, Session: 1, Request: request, OccurredAt: occurred, Count: uint64(len(ids)), TxIds: requested, TxRefs: requestedRefs})
	returned := make([][32]byte, 0, len(ids))
	returnedRefs := make([]TxRef, 0, len(ids))
	out := make([]txsubmission.TxBody, 0, len(ids))
	p.mu.Lock()
	for _, id := range ids {
		for _, tx := range p.txs[:p.announced] {
			if tx.EraId == id.EraId && tx.ID == id.TxId {
				out = append(out, txsubmission.TxBody{
					EraId:  tx.EraId,
					TxBody: tx.Raw,
				})
				p.served = append(p.served, tx)
				returned = append(returned, tx.ID)
				returnedRefs = append(returnedRefs, TxRef{EraId: tx.EraId, ID: tx.ID})
				break
			}
		}
	}
	p.mu.Unlock()
	p.events.publish(Event{Kind: EventTxsReturned, Session: 1, Request: request, OccurredAt: time.Now(), Count: uint64(len(out)), TxIds: returned, TxRefs: returnedRefs})
	return out, nil
}

// Close ends the connection and discards unread observations.
func (p *TxPeer) Close() error {
	err := p.stop(context.Background())
	p.events.close()
	return err
}

// Finish stops transaction offers and drains observations while Events is consumed.
func (p *TxPeer) Finish(ctx context.Context) (ObservationStats, error) {
	txProtocol := p.conn.TxSubmission().Client.ProtocolInstance()
	err := p.stop(ctx)
	if waitErr := waitProtocols(ctx, txProtocol); waitErr != nil {
		p.events.close()
		return p.events.stats(), errors.Join(err, waitErr, ErrIncompleteObservations)
	}
	select {
	case <-p.watchDone:
	case <-ctx.Done():
		p.events.close()
		return p.events.stats(), errors.Join(err, ctx.Err(), ErrIncompleteObservations)
	}
	stats, drainErr := p.events.finish(ctx)
	return stats, errors.Join(err, drainErr)
}
