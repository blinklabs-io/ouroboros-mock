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

package fixtures

import (
	"errors"
	"fmt"
	"slices"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
)

const (
	dijkstraTxBodyOutputsKey         = 1
	dijkstraTxBodyFeeKey             = 2
	dijkstraTxBodySubTransactionsKey = 23
)

// DijkstraTransactionBuilder constructs Dijkstra block transactions with
// guards, redeemers, metadata, subtransactions, and the validity flag. The
// Dijkstra witness set has no Plutus V4 script field (the reference decoder
// accepts keys 0-7), so a Plutus V4 script can only be supplied as a reference
// script.
type DijkstraTransactionBuilder struct {
	tx       dijkstra.DijkstraTransaction
	metadata common.TransactionMetadatum
}

// NewDijkstraTransactionBuilder creates an empty Dijkstra transaction builder.
func NewDijkstraTransactionBuilder() *DijkstraTransactionBuilder {
	return &DijkstraTransactionBuilder{
		tx: dijkstra.DijkstraTransaction{TxIsValid: true},
	}
}

// WithBody replaces the transaction body. A decoded body keeps its original
// CBOR until a later builder call changes one of its fields.
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
	b.tx.Body.SetCbor(nil)
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
	b.tx.Body.SetCbor(nil)
	return b
}

// WithWitnessSet replaces the transaction witness set.
func (b *DijkstraTransactionBuilder) WithWitnessSet(
	witnesses dijkstra.DijkstraTransactionWitnessSet,
) *DijkstraTransactionBuilder {
	b.tx.WitnessSet = witnesses
	return b
}

// WithRedeemers sets the Dijkstra redeemer map.
func (b *DijkstraTransactionBuilder) WithRedeemers(
	redeemers dijkstra.DijkstraRedeemers,
) *DijkstraTransactionBuilder {
	b.tx.WitnessSet.WsRedeemers = redeemers
	b.tx.WitnessSet.SetCbor(nil)
	return b
}

// WithMetadata sets the transaction metadata, encoded as metadata-only
// auxiliary data. The metadata must be a map keyed by unsigned integer labels.
// Build sets the body's auxiliary_data_hash from the encoded metadata unless
// the body already carries one.
func (b *DijkstraTransactionBuilder) WithMetadata(
	metadata common.TransactionMetadatum,
) *DijkstraTransactionBuilder {
	b.metadata = metadata
	return b
}

// WithTxIsValid sets the transaction validity flag.
func (b *DijkstraTransactionBuilder) WithTxIsValid(
	valid bool,
) *DijkstraTransactionBuilder {
	b.tx.TxIsValid = valid
	return b
}

// Build encodes the transaction in the Dijkstra block_transaction form and
// decodes it through the Dijkstra block body decoder, so the block-only
// validity flag is preserved and validated.
func (b *DijkstraTransactionBuilder) Build() (
	*dijkstra.DijkstraTransaction,
	error,
) {
	if len(b.tx.WitnessSet.WsPlutusV4Scripts.Items()) > 0 {
		return nil, errors.New(
			"Plutus V4 witness scripts are not part of the Dijkstra CDDL",
		)
	}
	body := b.tx.Body
	var auxCBOR []byte
	if b.metadata != nil {
		encoded, err := encodeDijkstraMetadata(b.metadata)
		if err != nil {
			return nil, err
		}
		auxCBOR = encoded
		if body.TxAuxDataHash == nil {
			hash := common.Blake2b256Hash(encoded)
			body.TxAuxDataHash = &hash
			body.SetCbor(nil)
		}
	}
	bodyCBOR, err := encodeDijkstraTransactionBody(body)
	if err != nil {
		return nil, err
	}
	witnessCBOR, err := cbor.Encode(b.tx.WitnessSet)
	if err != nil {
		return nil, fmt.Errorf("encode Dijkstra witness set fixture: %w", err)
	}
	var aux any
	if auxCBOR != nil {
		aux = cbor.RawMessage(auxCBOR)
	}
	txCBOR, err := cbor.Encode([]any{
		cbor.RawMessage(bodyCBOR),
		cbor.RawMessage(witnessCBOR),
		aux,
		b.tx.TxIsValid,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"encode Dijkstra block transaction fixture: %w",
			err,
		)
	}
	blockBodyCBOR, err := cbor.Encode([]any{
		[]cbor.RawMessage{txCBOR},
		nil,
		nil,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Dijkstra block body fixture: %w", err)
	}
	var decoded dijkstra.DijkstraBlockBody
	if _, err := cbor.Decode(blockBodyCBOR, &decoded); err != nil {
		return nil, fmt.Errorf(
			"decode Dijkstra block transaction fixture: %w",
			err,
		)
	}
	if len(decoded.Transactions) != 1 {
		return nil, fmt.Errorf(
			"decoded Dijkstra transaction count is %d, want 1",
			len(decoded.Transactions),
		)
	}
	tx := decoded.Transactions[0]
	if tx.TxIsValid != b.tx.TxIsValid {
		return nil, errors.New(
			"dijkstra transaction validity changed during round-trip",
		)
	}
	if auxCBOR != nil && tx.Metadata() == nil {
		return nil, errors.New(
			"dijkstra transaction metadata was lost during round-trip",
		)
	}
	return &tx, nil
}

func encodeDijkstraMetadata(
	metadata common.TransactionMetadatum,
) ([]byte, error) {
	var metaMap common.MetaMap
	switch value := metadata.(type) {
	case common.MetaMap:
		metaMap = value
	case *common.MetaMap:
		if value == nil {
			return nil, errors.New("dijkstra transaction metadata map is nil")
		}
		metaMap = *value
	default:
		return nil, fmt.Errorf(
			"dijkstra transaction metadata must be a map, got %s",
			metadata.TypeName(),
		)
	}
	for _, pair := range metaMap.Pairs {
		label, ok := pair.Key.(common.MetaInt)
		if !ok || label.Value == nil || !label.Value.IsUint64() {
			return nil, errors.New(
				"dijkstra transaction metadata labels must be unsigned 64-bit integers",
			)
		}
	}
	encoded, err := cbor.Encode(metaMap)
	if err != nil {
		return nil, fmt.Errorf("encode Dijkstra transaction metadata: %w", err)
	}
	return encoded, nil
}

// encodeDijkstraTransactionBody encodes a body with the keys the Dijkstra CDDL
// requires. gouroboros encodes outputs (1) and the fee (2) with omitempty, so
// an in-process body with no outputs or a zero fee drops required keys;
// subtransaction bodies drop outputs the same way. A decoded, unmodified body
// keeps its original bytes.
func encodeDijkstraTransactionBody(
	body dijkstra.DijkstraTransactionBody,
) ([]byte, error) {
	if raw := body.Cbor(); raw != nil {
		return raw, nil
	}
	if subtransactions := body.TxSubTransactions.Items(); len(subtransactions) > 0 {
		normalized := make(
			[]dijkstra.DijkstraSubTransaction,
			len(subtransactions),
		)
		for i, sub := range subtransactions {
			fixed, err := normalizeDijkstraSubTransaction(sub)
			if err != nil {
				return nil, fmt.Errorf("subtransaction %d: %w", i, err)
			}
			normalized[i] = fixed
		}
		body.TxSubTransactions = cbor.NewSetType(normalized, true)
	}
	encoded, err := cbor.Encode(body)
	if err != nil {
		return nil, fmt.Errorf(
			"encode Dijkstra transaction body fixture: %w",
			err,
		)
	}
	if err := rejectEmptySubTransactions(encoded); err != nil {
		return nil, err
	}
	return withRequiredMapKeys(encoded, map[uint64][]byte{
		dijkstraTxBodyOutputsKey: {0x80},
		dijkstraTxBodyFeeKey:     {0x00},
	})
}

// rejectEmptySubTransactions refuses key 23 encoded as an empty set.
// sub_transactions is a nonempty_oset, and gouroboros encodes an empty tagged
// set instead of omitting the key.
func rejectEmptySubTransactions(encoded []byte) error {
	var fields map[uint64]cbor.RawMessage
	if _, err := cbor.Decode(encoded, &fields); err != nil {
		return fmt.Errorf("decode transaction body map: %w", err)
	}
	raw, ok := fields[dijkstraTxBodySubTransactionsKey]
	if !ok {
		return nil
	}
	var subtransactions cbor.SetType[cbor.RawMessage]
	if _, err := cbor.Decode(raw, &subtransactions); err != nil {
		return fmt.Errorf("decode sub-transactions: %w", err)
	}
	if len(subtransactions.Items()) == 0 {
		return errors.New("dijkstra sub-transactions must not be empty")
	}
	return nil
}

func normalizeDijkstraSubTransaction(
	sub dijkstra.DijkstraSubTransaction,
) (dijkstra.DijkstraSubTransaction, error) {
	if sub.Cbor() != nil || sub.Body.Cbor() != nil {
		return sub, nil
	}
	encoded, err := cbor.Encode(sub.Body)
	if err != nil {
		return sub, fmt.Errorf("encode body: %w", err)
	}
	fixed, err := withRequiredMapKeys(encoded, map[uint64][]byte{
		dijkstraTxBodyOutputsKey: {0x80},
	})
	if err != nil {
		return sub, err
	}
	var body dijkstra.DijkstraSubTransactionBody
	if _, err := cbor.Decode(fixed, &body); err != nil {
		return sub, fmt.Errorf("decode body: %w", err)
	}
	sub.Body = body
	return sub, nil
}

// withRequiredMapKeys adds each missing key with its default encoded value.
// gouroboros encodes maps in core deterministic order, which for these small
// unsigned keys is the ascending order cardano-ledger writes.
func withRequiredMapKeys(
	encoded []byte,
	required map[uint64][]byte,
) ([]byte, error) {
	var fields map[uint64]cbor.RawMessage
	if _, err := cbor.Decode(encoded, &fields); err != nil {
		return nil, fmt.Errorf("decode transaction body map: %w", err)
	}
	if fields == nil {
		return nil, errors.New("transaction body is not a map")
	}
	missing := false
	for key, value := range required {
		if _, ok := fields[key]; !ok {
			fields[key] = slices.Clone(value)
			missing = true
		}
	}
	if !missing {
		return encoded, nil
	}
	ret, err := cbor.Encode(fields)
	if err != nil {
		return nil, fmt.Errorf("encode transaction body map: %w", err)
	}
	return ret, nil
}
