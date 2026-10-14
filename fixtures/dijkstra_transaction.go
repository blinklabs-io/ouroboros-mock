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
	"maps"
	"slices"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
)

const (
	dijkstraTxBodyInputsKey          = 0
	dijkstraTxBodyOutputsKey         = 1
	dijkstraTxBodyFeeKey             = 2
	dijkstraTxBodySubTransactionsKey = 23

	dijkstraWitnessMaxKey             = 7
	dijkstraWitnessPlutusV4ScriptsKey = 8
)

// DijkstraTransactionBuilder constructs Dijkstra block transactions with
// guards, redeemers, metadata, subtransactions, and the validity flag. The
// Dijkstra witness set has no Plutus V4 script field (the reference decoder
// accepts keys 0-7). Dijkstra transactions can carry V4 scripts as output
// reference scripts or in auxiliary_data_map key 5, but this builder only
// exposes transaction metadata and does not construct script-bearing auxiliary
// data.
type DijkstraTransactionBuilder struct {
	tx       dijkstra.DijkstraTransaction
	metadata common.TransactionMetadatum
	err      error
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

// WithDirectDeposit sets the CIP-159 direct deposit of coin lovelace to
// account, replacing any earlier deposit to the same account. Build rejects an
// address that is not an account (reward) address.
func (b *DijkstraTransactionBuilder) WithDirectDeposit(
	account common.Address,
	coin uint64,
) *DijkstraTransactionBuilder {
	if _, err := account.RewardAccountCredential(); err != nil {
		b.err = errors.Join(b.err, fmt.Errorf("direct deposit: %w", err))
		return b
	}
	raw, err := account.Bytes()
	if err != nil {
		b.err = errors.Join(b.err, fmt.Errorf("direct deposit: %w", err))
		return b
	}
	deposits := maps.Clone(b.tx.Body.TxDirectDeposits)
	if deposits == nil {
		deposits = map[cbor.ByteString]uint64{}
	}
	deposits[cbor.NewByteString(raw)] = coin
	b.tx.Body.TxDirectDeposits = deposits
	b.tx.Body.SetCbor(nil)
	return b
}

// WithAccountBalanceInterval sets the CIP-159 balance interval asserted for
// account, replacing any earlier interval for the same account. Build rejects
// an address that is not an account (reward) address.
func (b *DijkstraTransactionBuilder) WithAccountBalanceInterval(
	account common.Address,
	interval dijkstra.DijkstraAccountBalanceInterval,
) *DijkstraTransactionBuilder {
	if _, err := account.RewardAccountCredential(); err != nil {
		b.err = errors.Join(b.err, fmt.Errorf("balance interval: %w", err))
		return b
	}
	raw, err := account.Bytes()
	if err != nil {
		b.err = errors.Join(b.err, fmt.Errorf("balance interval: %w", err))
		return b
	}
	intervals := cloneDijkstraAccountBalanceIntervals(
		b.tx.Body.TxBalanceIntervals,
	)
	if intervals == nil {
		intervals = dijkstra.DijkstraAccountBalanceIntervals{}
		b.tx.Body.TxBalanceIntervals = intervals
	}
	intervals[cbor.NewByteString(raw)] = cloneDijkstraAccountBalanceInterval(&interval)
	b.tx.Body.TxBalanceIntervals = intervals
	b.tx.Body.SetCbor(nil)
	return b
}

func cloneDijkstraAccountBalanceIntervals(
	intervals dijkstra.DijkstraAccountBalanceIntervals,
) dijkstra.DijkstraAccountBalanceIntervals {
	if intervals == nil {
		return nil
	}
	cloned := make(dijkstra.DijkstraAccountBalanceIntervals, len(intervals))
	for account, interval := range intervals {
		cloned[account] = cloneDijkstraAccountBalanceInterval(interval)
	}
	return cloned
}

func cloneDijkstraAccountBalanceInterval(
	interval *dijkstra.DijkstraAccountBalanceInterval,
) *dijkstra.DijkstraAccountBalanceInterval {
	if interval == nil {
		return nil
	}
	cloned := *interval
	cloneBound := func(value *uint64) *uint64 {
		if value == nil {
			return nil
		}
		bound := *value
		return &bound
	}
	cloned.Exact = cloneBound(interval.Exact)
	cloned.LowerBound = cloneBound(interval.LowerBound)
	cloned.UpperBound = cloneBound(interval.UpperBound)
	return &cloned
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
	if b.err != nil {
		return nil, b.err
	}
	var auxCBOR []byte
	if b.metadata != nil {
		encoded, err := encodeDijkstraMetadata(b.metadata)
		if err != nil {
			return nil, err
		}
		auxCBOR = encoded
	}
	txCBOR, err := encodeDijkstraBlockTransaction(
		b.tx.Body,
		b.tx.WitnessSet,
		auxCBOR,
		b.tx.TxIsValid,
	)
	if err != nil {
		return nil, err
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

// dijkstraBlockTransactionCBOR returns the block_transaction encoding of tx.
// A transaction that carries block_transaction CBOR keeps those bytes; any
// other transaction is encoded as DijkstraTransactionBuilder encodes it. Both
// are checked against the Dijkstra CDDL.
func dijkstraBlockTransactionCBOR(
	tx *dijkstra.DijkstraTransaction,
) ([]byte, error) {
	if raw := tx.DecodeStoreCbor.Cbor(); raw != nil {
		var elements []cbor.RawMessage
		if _, err := cbor.Decode(raw, &elements); err == nil &&
			len(elements) == 4 {
			if err := validateDijkstraBlockTransaction(raw); err != nil {
				return nil, err
			}
			return raw, nil
		}
	}
	var auxCBOR []byte
	if aux := tx.AuxiliaryData(); aux != nil && len(aux.Cbor()) > 0 {
		auxCBOR = aux.Cbor()
	} else if tx.TxMetadata != nil {
		encoded, err := encodeDijkstraMetadata(tx.TxMetadata)
		if err != nil {
			return nil, err
		}
		auxCBOR = encoded
	}
	return encodeDijkstraBlockTransaction(
		tx.Body,
		tx.WitnessSet,
		auxCBOR,
		tx.TxIsValid,
	)
}

// encodeDijkstraBlockTransaction encodes a block_transaction. It sets the
// body's auxiliary_data_hash from auxCBOR unless the body already carries one.
func encodeDijkstraBlockTransaction(
	body dijkstra.DijkstraTransactionBody,
	witnesses dijkstra.DijkstraTransactionWitnessSet,
	auxCBOR []byte,
	valid bool,
) ([]byte, error) {
	if auxCBOR != nil && body.TxAuxDataHash == nil {
		hash := common.Blake2b256Hash(auxCBOR)
		body.TxAuxDataHash = &hash
		body.SetCbor(nil)
	}
	bodyCBOR, err := encodeDijkstraTransactionBody(body)
	if err != nil {
		return nil, err
	}
	witnessCBOR, err := cbor.Encode(witnesses)
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
		valid,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"encode Dijkstra block transaction fixture: %w",
			err,
		)
	}
	if err := validateDijkstraBlockTransaction(txCBOR); err != nil {
		return nil, err
	}
	return txCBOR, nil
}

// validateDijkstraBlockTransaction checks encoded block_transaction bytes
// against the Dijkstra CDDL rules gouroboros does not enforce when it encodes
// a value or reuses its preserved CBOR: the required transaction_body and
// sub_transaction_body keys, a non-empty sub_transactions set, and
// transaction_witness_set keys 0-7. Checking the emitted bytes holds decoded
// values, whose original CBOR is reused, to the same rules as in-process ones.
func validateDijkstraBlockTransaction(encoded []byte) error {
	var elements []cbor.RawMessage
	if _, err := cbor.Decode(encoded, &elements); err != nil {
		return fmt.Errorf("decode block transaction: %w", err)
	}
	if len(elements) != 4 {
		return fmt.Errorf(
			"block transaction has %d elements, want 4",
			len(elements),
		)
	}
	body, err := decodeDijkstraBodyFields(
		elements[0],
		"transaction body",
		dijkstraTxBodyOutputsKey,
		dijkstraTxBodyFeeKey,
	)
	if err != nil {
		return err
	}
	if err := validateDijkstraWitnessSetKeys(elements[1]); err != nil {
		return err
	}
	raw, ok := body[dijkstraTxBodySubTransactionsKey]
	if !ok {
		return nil
	}
	var subtransactions cbor.SetType[cbor.RawMessage]
	if _, err := cbor.Decode(raw, &subtransactions); err != nil {
		return fmt.Errorf("decode sub-transactions: %w", err)
	}
	// sub_transactions is a nonempty_oset, and gouroboros encodes an empty
	// set as an empty tagged set instead of omitting key 23.
	if len(subtransactions.Items()) == 0 {
		return errors.New("dijkstra sub-transactions must not be empty")
	}
	for i, sub := range subtransactions.Items() {
		if err := validateDijkstraSubTransaction(sub); err != nil {
			return fmt.Errorf("subtransaction %d: %w", i, err)
		}
	}
	return nil
}

func validateDijkstraSubTransaction(encoded []byte) error {
	var elements []cbor.RawMessage
	if _, err := cbor.Decode(encoded, &elements); err != nil {
		return fmt.Errorf("decode sub-transaction: %w", err)
	}
	if len(elements) != 3 {
		return fmt.Errorf(
			"sub-transaction has %d elements, want 3",
			len(elements),
		)
	}
	if _, err := decodeDijkstraBodyFields(
		elements[0],
		"body",
		dijkstraTxBodyOutputsKey,
	); err != nil {
		return err
	}
	return validateDijkstraWitnessSetKeys(elements[1])
}

// decodeDijkstraBodyFields decodes a body map and requires key 0 (inputs) and
// each of the given keys.
func decodeDijkstraBodyFields(
	encoded []byte,
	name string,
	required ...uint64,
) (map[uint64]cbor.RawMessage, error) {
	var fields map[uint64]cbor.RawMessage
	if _, err := cbor.Decode(encoded, &fields); err != nil {
		return nil, fmt.Errorf("decode %s map: %w", name, err)
	}
	if fields == nil {
		return nil, fmt.Errorf("%s is not a map", name)
	}
	for _, key := range append([]uint64{dijkstraTxBodyInputsKey}, required...) {
		if _, ok := fields[key]; !ok {
			return nil, fmt.Errorf("%s is missing required key %d", name, key)
		}
	}
	return fields, nil
}

// validateDijkstraWitnessSetKeys rejects witness set keys past 7, which the
// Dijkstra CDDL and the reference decoder do not accept. It covers witness
// sets that carry original bytes, which gouroboros reuses without checking.
func validateDijkstraWitnessSetKeys(encoded []byte) error {
	var fields map[uint64]cbor.RawMessage
	if _, err := cbor.Decode(encoded, &fields); err != nil {
		return fmt.Errorf("decode witness set map: %w", err)
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if key == dijkstraWitnessPlutusV4ScriptsKey {
			return errors.New(
				"plutus V4 witness scripts are not part of the Dijkstra CDDL",
			)
		}
		if key > dijkstraWitnessMaxKey {
			return fmt.Errorf(
				"witness set key %d is not part of the Dijkstra CDDL",
				key,
			)
		}
	}
	return nil
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
// keeps its original bytes, so validateDijkstraBlockTransaction rejects one
// that lacks a required key.
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
	return withRequiredMapKeys(encoded, map[uint64][]byte{
		dijkstraTxBodyOutputsKey: {0x80},
		dijkstraTxBodyFeeKey:     {0x00},
	})
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
