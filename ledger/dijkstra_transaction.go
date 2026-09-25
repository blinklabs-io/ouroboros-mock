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

package ledger

import (
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
)

// DijkstraTransactionBuilder constructs Dijkstra block transactions with
// guards, Plutus V4 witnesses, redeemers, and subtransactions.
type DijkstraTransactionBuilder struct {
	tx dijkstra.DijkstraTransaction
}

// NewDijkstraTransactionBuilder creates an empty Dijkstra transaction builder.
func NewDijkstraTransactionBuilder() *DijkstraTransactionBuilder {
	return &DijkstraTransactionBuilder{
		tx: dijkstra.DijkstraTransaction{TxIsValid: true},
	}
}

// WithBody replaces the transaction body.
func (b *DijkstraTransactionBuilder) WithBody(
	body dijkstra.DijkstraTransactionBody,
) *DijkstraTransactionBuilder {
	b.tx.Body = body
	return b
}

// WithTxGuards sets transaction-level Dijkstra guards.
func (b *DijkstraTransactionBuilder) WithTxGuards(
	guards *dijkstra.DijkstraGuards,
) *DijkstraTransactionBuilder {
	b.tx.Body.TxGuards = guards
	return b
}

// WithSubTransactions appends transaction subtransactions.
func (b *DijkstraTransactionBuilder) WithSubTransactions(
	subtransactions ...dijkstra.DijkstraSubTransaction,
) *DijkstraTransactionBuilder {
	items := append(
		[]dijkstra.DijkstraSubTransaction(nil),
		b.tx.Body.TxSubTransactions.Items()...,
	)
	items = append(items, subtransactions...)
	b.tx.Body.TxSubTransactions = cbor.NewSetType(items, true)
	return b
}

// WithWitnessSet replaces the transaction witness set.
func (b *DijkstraTransactionBuilder) WithWitnessSet(
	witnesses dijkstra.DijkstraTransactionWitnessSet,
) *DijkstraTransactionBuilder {
	b.tx.WitnessSet = witnesses
	return b
}

// WithPlutusV4Scripts appends Plutus V4 scripts to the witness set.
func (b *DijkstraTransactionBuilder) WithPlutusV4Scripts(
	scripts ...common.PlutusV4Script,
) *DijkstraTransactionBuilder {
	items := append(
		[]common.PlutusV4Script(nil),
		b.tx.WitnessSet.WsPlutusV4Scripts.Items()...,
	)
	items = append(items, scripts...)
	b.tx.WitnessSet.WsPlutusV4Scripts = cbor.NewSetType(items, true)
	return b
}

// WithRedeemers sets the Dijkstra redeemer map.
func (b *DijkstraTransactionBuilder) WithRedeemers(
	redeemers dijkstra.DijkstraRedeemers,
) *DijkstraTransactionBuilder {
	b.tx.WitnessSet.WsRedeemers = redeemers
	return b
}

// WithMetadata sets transaction metadata.
func (b *DijkstraTransactionBuilder) WithMetadata(
	metadata common.TransactionMetadatum,
) *DijkstraTransactionBuilder {
	b.tx.TxMetadata = metadata
	return b
}

// WithTxIsValid sets the transaction validity flag.
func (b *DijkstraTransactionBuilder) WithTxIsValid(
	valid bool,
) *DijkstraTransactionBuilder {
	b.tx.TxIsValid = valid
	return b
}

// Build encodes and decodes the transaction through the Dijkstra block body
// decoder so the block-only validity flag is preserved and validated.
func (b *DijkstraTransactionBuilder) Build() (
	*dijkstra.DijkstraTransaction,
	error,
) {
	body := dijkstra.DijkstraBlockBody{
		Transactions: []dijkstra.DijkstraTransaction{b.tx},
	}
	encoded, err := cbor.Encode(body)
	if err != nil {
		return nil, fmt.Errorf("encode Dijkstra block transaction fixture: %w", err)
	}
	var decoded dijkstra.DijkstraBlockBody
	if _, err := cbor.Decode(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("decode Dijkstra block transaction fixture: %w", err)
	}
	if len(decoded.Transactions) != 1 {
		return nil, fmt.Errorf(
			"decoded Dijkstra transaction count is %d, want 1",
			len(decoded.Transactions),
		)
	}
	tx := decoded.Transactions[0]
	if tx.TxIsValid != b.tx.TxIsValid {
		return nil, fmt.Errorf(
			"Dijkstra transaction validity changed during round-trip",
		)
	}
	return &tx, nil
}
