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
	"fmt"
	"net"
	"sync"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// TxPeerConfig configures a [TxPeer].
type TxPeerConfig struct {
	// Conn is a connection to the node under test.
	Conn net.Conn
	// NetworkMagic is the handshake network magic. Defaults to
	// [ouroboros_mock.MockNetworkMagic] when zero.
	NetworkMagic uint32
	// Txs are the transactions to offer initially.
	Txs []Tx
}

// TxPeer is a node-to-node peer that offers transactions through
// tx-submission and records what the node asks for.
type TxPeer struct {
	conn   *ouroboros.Connection
	events *eventStream

	mu sync.Mutex
	// txs holds every offered transaction; txs[:announced] have been announced
	// to the node.
	txs       []Tx
	announced int
	served    []Tx
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
	p := &TxPeer{
		events: newEventStream(),
		txs:    append([]Tx(nil), cfg.Txs...),
		more:   make(chan struct{}),
	}
	txCfg := txsubmission.Config{
		RequestTxIdsFunc: p.requestTxIds,
		RequestTxsFunc:   p.requestTxs,
	}
	conn, err := ouroboros.New(
		ouroboros.WithConnection(cfg.Conn),
		ouroboros.WithNetworkMagic(cfg.NetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithTxSubmissionConfig(txCfg),
	)
	if err != nil {
		p.events.close()
		return nil, fmt.Errorf("handshake: %w", err)
	}
	p.conn = conn
	conn.TxSubmission().Client.Init()
	go p.watch()
	return p, nil
}

// Events returns the channel of observable events. It is closed by
// [TxPeer.Close].
func (p *TxPeer) Events() <-chan Event {
	return p.events.out
}

// Offer adds transactions to announce. A node waiting on a blocking request
// is answered immediately.
func (p *TxPeer) Offer(txs ...Tx) {
	p.mu.Lock()
	p.txs = append(p.txs, txs...)
	close(p.more)
	p.more = make(chan struct{})
	p.mu.Unlock()
}

// Served returns the transactions the node requested and received, in
// delivery order.
func (p *TxPeer) Served() []Tx {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Tx(nil), p.served...)
}

// Close ends the connection and closes the event channel.
func (p *TxPeer) Close() error {
	var err error
	if p.conn != nil {
		err = p.conn.Close()
	}
	p.events.close()
	return err
}

func (p *TxPeer) watch() {
	var last error
	for err := range p.conn.ErrorChan() {
		last = err
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
			p.events.publish(Event{Kind: EventTxIdsRequested, TxIds: ids})
			return out, nil
		}
		more := p.more
		p.mu.Unlock()
		p.events.publish(Event{Kind: EventTxIdsBlocked})
		// Close does not wake this request to send MsgDone: the gouroboros
		// tx-submission server this module pins races on receiving it.
		select {
		case <-more:
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
	requested := make([][32]byte, 0, len(ids))
	out := make([]txsubmission.TxBody, 0, len(ids))
	p.mu.Lock()
	for _, id := range ids {
		requested = append(requested, id.TxId)
		for _, tx := range p.txs[:p.announced] {
			if tx.ID == id.TxId {
				out = append(out, txsubmission.TxBody{
					EraId:  tx.EraId,
					TxBody: tx.Raw,
				})
				p.served = append(p.served, tx)
				break
			}
		}
	}
	p.mu.Unlock()
	p.events.publish(Event{Kind: EventTxsRequested, TxIds: requested})
	return out, nil
}
