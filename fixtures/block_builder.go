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
	"bytes"
	"errors"
	"fmt"
	"reflect"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/allegra"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

// eraBlockShape describes how a Shelley-through-Conway era frames its blocks.
type eraBlockShape struct {
	// decode is the era's gouroboros block decoder.
	decode func([]byte, common.VerifyConfig) (ledger.Block, error)
	// protocolMajor is the header protocol version used unless overridden.
	protocolMajor uint64
	// praosHeader selects the Babbage-style header body (one VRF result and a
	// nested operational certificate) over the Shelley-style one.
	praosHeader bool
	// invalidTxs reports whether the block carries an invalid-transaction
	// list, and whether the transactions carry a validity flag.
	invalidTxs bool
	// emptyInvalid is the encoding of an empty invalid-transaction list.
	emptyInvalid []byte
}

func decodeAs[B ledger.Block](
	decode func([]byte, ...common.VerifyConfig) (B, error),
) func([]byte, common.VerifyConfig) (ledger.Block, error) {
	return func(
		data []byte,
		cfg common.VerifyConfig,
	) (ledger.Block, error) {
		return decode(data, cfg)
	}
}

// eraBlockShapes is keyed by gouroboros era ID.
var eraBlockShapes = map[uint8]eraBlockShape{
	1: {
		decode:        decodeAs(shelley.NewShelleyBlockFromCbor),
		protocolMajor: shelley.MinProtocolVersionShelley,
	},
	2: {
		decode:        decodeAs(allegra.NewAllegraBlockFromCbor),
		protocolMajor: allegra.MinProtocolVersionAllegra,
	},
	3: {
		decode:        decodeAs(mary.NewMaryBlockFromCbor),
		protocolMajor: mary.MinProtocolVersionMary,
	},
	4: {
		decode:        decodeAs(alonzo.NewAlonzoBlockFromCbor),
		protocolMajor: alonzo.MinProtocolVersionAlonzo,
		invalidTxs:    true,
		// Alonzo blocks carry an indefinite-length invalid-transaction list.
		emptyInvalid: []byte{0x9f, 0xff},
	},
	5: {
		decode:        decodeAs(babbage.NewBabbageBlockFromCbor),
		protocolMajor: babbage.MinProtocolVersionBabbage,
		praosHeader:   true,
		invalidTxs:    true,
		emptyInvalid:  []byte{0x80},
	},
	6: {
		decode:        decodeAs(conway.NewConwayBlockFromCbor),
		protocolMajor: conway.MinProtocolVersionConway,
		praosHeader:   true,
		invalidTxs:    true,
		emptyInvalid:  []byte{0x80},
	},
}

// BlockBuilder constructs a block for any era in SupportedEras. Build encodes
// the block, derives the header's body size and body hash from the encoded
// body, and returns the block as decoded by gouroboros, so its CBOR, hash and
// header fields are what a downstream consumer sees.
type BlockBuilder struct {
	era          common.Era
	blockNumber  uint64
	slot         uint64
	prevHash     common.Blake2b256
	issuerVkey   common.IssuerVkey
	protoMajor   uint64
	protoMinor   uint64
	protoSet     bool
	transactions []common.Transaction
	byronMain    bool
	bodySize     *uint64
	bodyHash     *common.Blake2b256
}

// NewBlockBuilder creates an empty-block builder for era. Byron builds an
// epoch boundary block, which requires an epoch-aligned slot, unless
// WithByronMainBlock is set.
func NewBlockBuilder(era common.Era) *BlockBuilder {
	return &BlockBuilder{era: era}
}

// WithBlockNumber sets the block number.
func (b *BlockBuilder) WithBlockNumber(number uint64) *BlockBuilder {
	b.blockNumber = number
	return b
}

// WithSlot sets the block slot.
func (b *BlockBuilder) WithSlot(slot uint64) *BlockBuilder {
	b.slot = slot
	return b
}

// WithPreviousHash sets the previous block hash.
func (b *BlockBuilder) WithPreviousHash(hash common.Blake2b256) *BlockBuilder {
	b.prevHash = hash
	return b
}

// WithIssuerVkey sets the header's issuer verification key. Byron headers
// carry no issuer key, so Byron builds ignore it.
func (b *BlockBuilder) WithIssuerVkey(vkey common.IssuerVkey) *BlockBuilder {
	b.issuerVkey = vkey
	return b
}

// WithProtocolVersion overrides the header protocol version, which defaults to
// the era's first version. Byron headers carry no protocol version, so Build
// rejects it for Byron.
func (b *BlockBuilder) WithProtocolVersion(major, minor uint64) *BlockBuilder {
	b.protoMajor, b.protoMinor, b.protoSet = major, minor, true
	return b
}

// WithTransactions sets the block's transactions. Each must carry the CBOR of
// its own era's transaction encoding, as the gouroboros transaction types do.
// Dijkstra blocks accept only *dijkstra.DijkstraTransaction values. Byron
// blocks accept none.
func (b *BlockBuilder) WithTransactions(
	transactions ...common.Transaction,
) *BlockBuilder {
	b.transactions = append([]common.Transaction(nil), transactions...)
	return b
}

// WithBodySize overrides the header's block body size, which is otherwise the
// size of the encoded body. A block whose header disagrees with its body is
// useful for testing consumers that validate it. Byron and Dijkstra builds
// reject it.
func (b *BlockBuilder) WithBodySize(size uint64) *BlockBuilder {
	b.bodySize = &size
	return b
}

// WithBodyHash overrides the header's block body hash, which is otherwise the
// hash of the encoded body. See WithBodySize.
func (b *BlockBuilder) WithBodyHash(hash common.Blake2b256) *BlockBuilder {
	b.bodyHash = &hash
	return b
}

// WithByronMainBlock makes a Byron builder produce a regular (main) block
// instead of an epoch boundary block. The body is empty and the header's body
// proof matches it.
func (b *BlockBuilder) WithByronMainBlock() *BlockBuilder {
	b.byronMain = true
	return b
}

// Build returns the block. A Dijkstra block has the 12-field header body of
// the pinned Dijkstra CDDL, as NewDijkstraBlockBuilder emits.
func (b *BlockBuilder) Build() (ledger.Block, error) {
	switch b.era.Id {
	case byron.EraIdByron:
		return b.buildByron()
	case dijkstra.EraIdDijkstra:
		return b.buildDijkstra()
	}
	shape, ok := eraBlockShapes[b.era.Id]
	if !ok {
		return nil, fmt.Errorf(
			"unsupported fixture era %d (%s)", b.era.Id, b.era.Name,
		)
	}
	return b.buildShelleyFamily(shape)
}

// BuildHeader returns the header of the block Build would return.
func (b *BlockBuilder) BuildHeader() (ledger.BlockHeader, error) {
	block, err := b.Build()
	if err != nil {
		return nil, err
	}
	return block.Header(), nil
}

func (b *BlockBuilder) buildDijkstra() (ledger.Block, error) {
	if b.bodySize != nil || b.bodyHash != nil {
		return nil, errors.New(
			"dijkstra block builder derives the body size and hash",
		)
	}
	txs := make([]dijkstra.DijkstraTransaction, len(b.transactions))
	for i, tx := range b.transactions {
		if transactionIsNil(tx) {
			return nil, fmt.Errorf("transaction %d is nil", i)
		}
		v, ok := tx.(*dijkstra.DijkstraTransaction)
		if !ok {
			return nil, fmt.Errorf(
				"transaction %d is %T, expected *dijkstra.DijkstraTransaction",
				i,
				tx,
			)
		}
		txs[i] = *v
	}
	builder := NewDijkstraBlockBuilder().
		WithBlockNumber(b.blockNumber).
		WithSlot(b.slot).
		WithPreviousHash(b.prevHash).
		WithIssuerVkey(b.issuerVkey).
		WithTransactions(txs...)
	if b.protoSet {
		builder.WithProtocolVersion(babbage.BabbageProtoVersion{
			Major: b.protoMajor,
			Minor: b.protoMinor,
		})
	}
	return builder.Build()
}

func (b *BlockBuilder) buildShelleyFamily(
	shape eraBlockShape,
) (ledger.Block, error) {
	bodies := make([]cbor.RawMessage, 0, len(b.transactions))
	witnesses := make([]cbor.RawMessage, 0, len(b.transactions))
	aux := make(map[uint]cbor.RawMessage)
	invalid := []uint{}
	for i, tx := range b.transactions {
		parts, err := splitTransaction(tx, shape.invalidTxs)
		if err != nil {
			return nil, fmt.Errorf("transaction %d: %w", i, err)
		}
		bodies = append(bodies, parts.body)
		witnesses = append(witnesses, parts.witnesses)
		if parts.aux != nil {
			aux[uint(i)] = parts.aux
		}
		if !parts.valid {
			invalid = append(invalid, uint(i))
		}
	}
	bodiesCbor, err := cbor.Encode(bodies)
	if err != nil {
		return nil, fmt.Errorf("encode transaction bodies: %w", err)
	}
	witnessesCbor, err := cbor.Encode(witnesses)
	if err != nil {
		return nil, fmt.Errorf("encode witness sets: %w", err)
	}
	auxCbor, err := cbor.Encode(aux)
	if err != nil {
		return nil, fmt.Errorf("encode auxiliary data set: %w", err)
	}
	bodyParts := [][]byte{bodiesCbor, witnessesCbor, auxCbor}
	if shape.invalidTxs {
		invalidCbor := shape.emptyInvalid
		if len(invalid) > 0 {
			if invalidCbor, err = cbor.Encode(invalid); err != nil {
				return nil, fmt.Errorf("encode invalid transactions: %w", err)
			}
		}
		bodyParts = append(bodyParts, invalidCbor)
	}
	major, minor := shape.protocolMajor, uint64(0)
	if b.protoSet {
		major, minor = b.protoMajor, b.protoMinor
	}
	bodySize := computeBlockBodySize(bodyParts...)
	bodyHash := ComputeBlockBodyHash(bodyParts...)
	if b.bodySize != nil {
		bodySize = *b.bodySize
	}
	if b.bodyHash != nil {
		bodyHash = *b.bodyHash
	}
	headerCbor, err := b.encodeHeader(shape, major, minor, bodySize, bodyHash)
	if err != nil {
		return nil, err
	}
	fields := []cbor.RawMessage{headerCbor}
	for _, part := range bodyParts {
		fields = append(fields, part)
	}
	// A header that disagrees with its body is rejected by the decoder's
	// body hash check, so the check is skipped when the caller asked for one.
	cfg := common.VerifyConfig{
		SkipBodyHashValidation: b.bodySize != nil || b.bodyHash != nil,
	}
	return b.decodeEncoded(fields, func(data []byte) (ledger.Block, error) {
		return shape.decode(data, cfg)
	})
}

func (b *BlockBuilder) encodeHeader(
	shape eraBlockShape,
	protoMajor, protoMinor, bodySize uint64,
	bodyHash common.Blake2b256,
) ([]byte, error) {
	vrfResult := common.VrfResult{
		Output: make([]byte, 64),
		Proof:  make([]byte, 80),
	}
	var header any
	if shape.praosHeader {
		header = babbage.BabbageBlockHeader{
			Body: babbage.BabbageBlockHeaderBody{
				BlockNumber:   b.blockNumber,
				Slot:          b.slot,
				PrevHash:      b.prevHash,
				IssuerVkey:    b.issuerVkey,
				VrfKey:        make([]byte, 32),
				VrfResult:     vrfResult,
				BlockBodySize: bodySize,
				BlockBodyHash: bodyHash,
				OpCert: babbage.BabbageOpCert{
					HotVkey:   make([]byte, 32),
					Signature: make([]byte, 64),
				},
				ProtoVersion: babbage.BabbageProtoVersion{
					Major: protoMajor,
					Minor: protoMinor,
				},
			},
			Signature: make([]byte, 448),
		}
	} else {
		header = shelley.ShelleyBlockHeader{
			Body: shelley.ShelleyBlockHeaderBody{
				BlockNumber:       b.blockNumber,
				Slot:              b.slot,
				PrevHash:          b.prevHash,
				IssuerVkey:        b.issuerVkey,
				VrfKey:            make([]byte, 32),
				NonceVrf:          vrfResult,
				LeaderVrf:         vrfResult,
				BlockBodySize:     bodySize,
				BlockBodyHash:     bodyHash,
				OpCertHotVkey:     make([]byte, 32),
				OpCertSignature:   make([]byte, 64),
				ProtoMajorVersion: protoMajor,
				ProtoMinorVersion: protoMinor,
			},
			Signature: make([]byte, 448),
		}
	}
	encoded, err := cbor.Encode(header)
	if err != nil {
		return nil, fmt.Errorf("encode %s block header: %w", b.era.Name, err)
	}
	return encoded, nil
}

// decodeEncoded frames fields as a CBOR array, decodes it, and rejects a
// decode that does not reproduce the encoded bytes.
func (b *BlockBuilder) decodeEncoded(
	fields []cbor.RawMessage,
	decode func([]byte) (ledger.Block, error),
) (ledger.Block, error) {
	blockCbor, err := cbor.Encode(fields)
	if err != nil {
		return nil, fmt.Errorf("encode %s block: %w", b.era.Name, err)
	}
	decoded, err := decode(blockCbor)
	if err != nil {
		return nil, fmt.Errorf("decode generated %s block: %w", b.era.Name, err)
	}
	if !bytes.Equal(decoded.Cbor(), blockCbor) {
		return nil, fmt.Errorf(
			"%s block CBOR mismatch after round-trip", b.era.Name,
		)
	}
	return decoded, nil
}

type transactionParts struct {
	body, witnesses, aux cbor.RawMessage
	valid                bool
}

// splitTransaction separates a transaction's CBOR into the pieces a block
// stores separately. Alonzo and later transactions are
// [body, witnesses, valid, aux]; earlier ones are [body, witnesses, aux].
func splitTransaction(
	tx common.Transaction,
	hasValidFlag bool,
) (transactionParts, error) {
	if transactionIsNil(tx) {
		return transactionParts{}, errors.New("transaction is nil")
	}
	raw := tx.Cbor()
	if len(raw) == 0 {
		var err error
		if raw, err = cbor.Encode(tx); err != nil {
			return transactionParts{}, fmt.Errorf("encode transaction: %w", err)
		}
	}
	var fields []cbor.RawMessage
	if _, err := cbor.Decode(raw, &fields); err != nil {
		return transactionParts{}, fmt.Errorf(
			"decode transaction CBOR: %w",
			err,
		)
	}
	want := 3
	if hasValidFlag {
		want = 4
	}
	if len(fields) != want {
		return transactionParts{}, fmt.Errorf(
			"transaction has %d fields, expected %d for this era",
			len(fields),
			want,
		)
	}
	parts := transactionParts{
		body:      fields[0],
		witnesses: fields[1],
		valid:     true,
	}
	auxField := fields[want-1]
	if hasValidFlag {
		if _, err := cbor.Decode(fields[2], &parts.valid); err != nil {
			return transactionParts{}, fmt.Errorf(
				"decode transaction validity flag: %w", err,
			)
		}
	}
	if !bytes.Equal(auxField, []byte{0xf6}) {
		parts.aux = auxField
	}
	return parts, nil
}

func transactionIsNil(tx common.Transaction) bool {
	if tx == nil {
		return true
	}
	value := reflect.ValueOf(tx)
	kind := value.Kind()
	canBeNil := kind == reflect.Chan || kind == reflect.Func ||
		kind == reflect.Interface || kind == reflect.Map ||
		kind == reflect.Pointer || kind == reflect.Slice
	return canBeNil && value.IsNil()
}
