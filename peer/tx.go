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
	"sync"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
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

// TxsFromBlocks returns the transactions carried by blocks, in block order.
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
				EraId: conway.TxTypeConway,
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

	mu       sync.Mutex
	received []Tx
}

func (r *relay) config() txsubmission.Config {
	return txsubmission.Config{
		InitFunc: func(ctx txsubmission.CallbackContext) error {
			go r.run(ctx.Server)
			return nil
		},
		DoneFunc: func(txsubmission.CallbackContext) error { return nil },
	}
}

// run requests announcements until the protocol ends. Every request is
// blocking: each round acknowledges everything announced and fetched in the
// previous one, so the remote window is always empty.
func (r *relay) run(server *txsubmission.Server) {
	for {
		ids, err := server.RequestTxIds(true, 16)
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
		r.mu.Lock()
		r.received = append(r.received, got...)
		r.mu.Unlock()
		r.events.publish(Event{
			Kind:    EventTxsReceived,
			Session: r.session,
			Txs:     got,
		})
	}
}

// Received returns the transactions the remote node has delivered.
func (r *relay) Received() []Tx {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Tx(nil), r.received...)
}
