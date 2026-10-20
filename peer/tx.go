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
	"fmt"
	"sync"

	"github.com/blinklabs-io/gouroboros/connection"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
)

const (
	maxRetainedTxCount = 256
	maxRetainedTxBytes = 4 << 20
)

// Tx is a transaction a peer offers or has received.
type Tx struct {
	// EraId is the tx-submission era index of the transaction.
	EraId uint16
	// ID is the transaction body hash.
	ID [32]byte
	// Raw is the full transaction CBOR.
	Raw []byte
}

// TxsFromBlocks returns the transactions carried by blocks, in block order,
// each tagged with its own era.
// Use it with [github.com/blinklabs-io/ouroboros-mock/fixtures.GenerateConwayChainWithTransactions]
// to obtain wire-valid transactions.
func TxsFromBlocks(blocks []ledger.Block) ([]Tx, error) {
	var txs []Tx
	for _, b := range blocks {
		for _, tx := range b.Transactions() {
			raw := tx.Cbor()
			if len(raw) == 0 {
				return nil, fmt.Errorf(
					"transaction %s has no CBOR",
					tx.Hash().String(),
				)
			}
			txs = append(txs, Tx{
				//nolint:gosec // era ids are small non-negative constants
				EraId: uint16(tx.Type()),
				ID:    tx.Hash(),
				Raw:   raw,
			})
		}
	}
	return txs, nil
}

// relay is the server side of tx-submission on one connection. It asks the
// remote node for the transactions it announces and records them.
type relay struct {
	events  *eventStream
	session uint64

	// ready is closed by attach once the connection, and so its server, is
	// known. OnInit carries only a connection id, and a client's Init can
	// arrive before the handshake has returned the connection.
	ready      chan struct{}
	attachOnce sync.Once
	server     *txsubmission.Server

	mu       sync.Mutex
	received []Tx
	bytes    int
}

func newRelay(events *eventStream, session uint64) *relay {
	return &relay{
		events:  events,
		session: session,
		ready:   make(chan struct{}),
	}
}

// attach supplies the server the relay will drive. A nil server means the
// connection never came up, which releases a waiting run without starting it.
func (r *relay) attach(server *txsubmission.Server) {
	r.attachOnce.Do(func() {
		r.server = server
		close(r.ready)
	})
}

func (r *relay) config() txsubmission.Config {
	return txsubmission.Config{
		OnInit: func(connection.ConnectionId) {
			go func() {
				<-r.ready
				if r.server != nil {
					r.run(r.server)
				}
			}()
		},
	}
}

// run requests announcements until the protocol ends. Every request is
// blocking: each round acknowledges everything announced and fetched in the
// previous one, so the remote window is always empty.
func (r *relay) run(server *txsubmission.Server) {
	for {
		ids, err := server.RequestTxIds(true, txsubmission.MaxUnackedTxIds)
		if err != nil {
			return
		}
		if len(ids) == 0 {
			continue
		}
		offered := make([][32]byte, 0, len(ids))
		want := make([]txsubmission.TxId, 0, len(ids))
		for _, id := range ids {
			offered = append(offered, id.TxId.TxId)
			want = append(want, id.TxId)
		}
		r.events.publish(Event{
			Kind:    EventTxIdsOffered,
			Session: r.session,
			TxIds:   offered,
		})
		bodies, err := server.RequestTxs(want)
		if err != nil {
			return
		}
		got := make([]Tx, 0, len(bodies))
		for _, b := range bodies {
			tx := Tx{EraId: b.EraId, Raw: b.TxBody}
			if parsed, perr := ledger.NewTransactionFromCbor(
				uint(b.EraId),
				b.TxBody,
			); perr == nil {
				tx.ID = parsed.Hash()
			}
			got = append(got, tx)
		}
		r.retain(got)
		r.events.publish(Event{
			Kind:    EventTxsReceived,
			Session: r.session,
			Txs:     got,
		})
	}
}

func (r *relay) retain(txs []Tx) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, tx := range txs {
		if len(tx.Raw) > maxRetainedTxBytes {
			continue
		}
		tx.Raw = bytes.Clone(tx.Raw)
		r.received = append(r.received, tx)
		r.bytes += len(tx.Raw)
		for len(r.received) > maxRetainedTxCount ||
			r.bytes > maxRetainedTxBytes {
			r.bytes -= len(r.received[0].Raw)
			r.received[0] = Tx{}
			r.received = r.received[1:]
		}
	}
}

// Received returns the most recently retained transactions delivered by the
// remote node.
func (r *relay) Received() []Tx {
	r.mu.Lock()
	defer r.mu.Unlock()
	ret := make([]Tx, len(r.received))
	for i, tx := range r.received {
		tx.Raw = bytes.Clone(tx.Raw)
		ret[i] = tx
	}
	return ret
}
