// Copyright 2025 Blink Labs Software
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

package blocks

import (
	"fmt"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	utxorpc "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
)

// AlonzoBlockBuilder implements BlockBuilder for Alonzo era blocks.
// Alonzo introduced Plutus scripts, enabling smart contracts on Cardano.
type AlonzoBlockBuilder struct {
	slot         uint64
	blockNumber  uint64
	hash         lcommon.Blake2b256
	prevHash     lcommon.Blake2b256
	transactions []TxBuilder
	hashErr      error
	prevHashErr  error
}

// NewAlonzoBlockBuilder creates a new AlonzoBlockBuilder
func NewAlonzoBlockBuilder() *AlonzoBlockBuilder {
	return &AlonzoBlockBuilder{}
}

// WithSlot sets the slot number for the block
func (b *AlonzoBlockBuilder) WithSlot(slot uint64) BlockBuilder {
	b.slot = slot
	return b
}

// WithBlockNumber sets the block number
func (b *AlonzoBlockBuilder) WithBlockNumber(number uint64) BlockBuilder {
	b.blockNumber = number
	return b
}

// WithHash sets the block hash
func (b *AlonzoBlockBuilder) WithHash(hash []byte) BlockBuilder {
	if len(hash) != len(b.hash) {
		b.hashErr = fmt.Errorf(
			"hash must be exactly %d bytes, got %d",
			len(b.hash),
			len(hash),
		)
		return b
	}
	copy(b.hash[:], hash)
	return b
}

// WithPrevHash sets the previous block hash
func (b *AlonzoBlockBuilder) WithPrevHash(prevHash []byte) BlockBuilder {
	if len(prevHash) != len(b.prevHash) {
		b.prevHashErr = fmt.Errorf(
			"prevHash must be exactly %d bytes, got %d",
			len(b.prevHash),
			len(prevHash),
		)
		return b
	}
	copy(b.prevHash[:], prevHash)
	return b
}

// WithTransactions sets the transactions for the block
func (b *AlonzoBlockBuilder) WithTransactions(
	txs ...TxBuilder,
) BlockBuilder {
	b.transactions = txs
	return b
}

// Build constructs a mock Alonzo block that satisfies the ledger.Block interface
func (b *AlonzoBlockBuilder) Build() (ledger.Block, error) {
	if b.hashErr != nil {
		return nil, b.hashErr
	}
	if b.prevHashErr != nil {
		return nil, b.prevHashErr
	}
	mockBlock := &MockAlonzoBlock{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
	}

	// Build transactions if any
	for _, txBuilder := range b.transactions {
		tx, err := txBuilder.Build()
		if err != nil {
			return nil, err
		}
		mockBlock.transactions = append(mockBlock.transactions, tx)
	}

	return mockBlock, nil
}

// BuildCbor returns the CBOR encoding of the block
// Currently returns nil as a placeholder
func (b *AlonzoBlockBuilder) BuildCbor() ([]byte, error) {
	// TODO: Implement CBOR encoding when needed
	return nil, nil
}

// MockAlonzoBlock is a mock implementation of an Alonzo block that satisfies ledger.Block.
// Alonzo was the era that introduced Plutus scripts to Cardano.
type MockAlonzoBlock struct {
	slot         uint64
	blockNumber  uint64
	hash         lcommon.Blake2b256
	prevHash     lcommon.Blake2b256
	transactions []lcommon.Transaction
}

// Type returns the block type identifier for Alonzo
func (b *MockAlonzoBlock) Type() int {
	return alonzo.BlockTypeAlonzo
}

// Hash returns the block hash
func (b *MockAlonzoBlock) Hash() lcommon.Blake2b256 {
	return b.hash
}

// PrevHash returns the previous block hash
func (b *MockAlonzoBlock) PrevHash() lcommon.Blake2b256 {
	return b.prevHash
}

// BlockNumber returns the block number
func (b *MockAlonzoBlock) BlockNumber() uint64 {
	return b.blockNumber
}

// SlotNumber returns the slot number
func (b *MockAlonzoBlock) SlotNumber() uint64 {
	return b.slot
}

// IssuerVkey returns the issuer verification key
func (b *MockAlonzoBlock) IssuerVkey() lcommon.IssuerVkey {
	return lcommon.IssuerVkey{}
}

// BlockBodySize returns the block body size
func (b *MockAlonzoBlock) BlockBodySize() uint64 {
	return 0
}

// Era returns the Alonzo era
func (b *MockAlonzoBlock) Era() lcommon.Era {
	return alonzo.EraAlonzo
}

// Cbor returns the CBOR encoding of the block
func (b *MockAlonzoBlock) Cbor() []byte {
	return nil
}

// BlockBodyHash returns the block body hash
func (b *MockAlonzoBlock) BlockBodyHash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

// Header returns the block header
func (b *MockAlonzoBlock) Header() ledger.BlockHeader {
	return &MockAlonzoBlockHeader{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
	}
}

// Transactions returns the list of transactions in the block
func (b *MockAlonzoBlock) Transactions() []lcommon.Transaction {
	return b.transactions
}

// Utxorpc returns the block in UTxO RPC format
func (b *MockAlonzoBlock) Utxorpc() (*utxorpc.Block, error) {
	txs := make([]*utxorpc.Tx, 0, len(b.transactions))
	for _, t := range b.transactions {
		tx, err := t.Utxorpc()
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	body := &utxorpc.BlockBody{
		Tx: txs,
	}
	header := &utxorpc.BlockHeader{
		Hash:   b.hash[:],
		Height: b.blockNumber,
		Slot:   b.slot,
	}
	block := &utxorpc.Block{
		Body:   body,
		Header: header,
	}
	return block, nil
}

// MockAlonzoBlockHeader is a mock implementation of an Alonzo block header
type MockAlonzoBlockHeader struct {
	slot        uint64
	blockNumber uint64
	hash        lcommon.Blake2b256
	prevHash    lcommon.Blake2b256
}

// Hash returns the block header hash
func (h *MockAlonzoBlockHeader) Hash() lcommon.Blake2b256 {
	return h.hash
}

// PrevHash returns the previous block hash
func (h *MockAlonzoBlockHeader) PrevHash() lcommon.Blake2b256 {
	return h.prevHash
}

// BlockNumber returns the block number
func (h *MockAlonzoBlockHeader) BlockNumber() uint64 {
	return h.blockNumber
}

// SlotNumber returns the slot number
func (h *MockAlonzoBlockHeader) SlotNumber() uint64 {
	return h.slot
}

// IssuerVkey returns the issuer verification key
func (h *MockAlonzoBlockHeader) IssuerVkey() lcommon.IssuerVkey {
	return lcommon.IssuerVkey{}
}

// BlockBodySize returns the block body size
func (h *MockAlonzoBlockHeader) BlockBodySize() uint64 {
	return 0
}

// Era returns the Alonzo era
func (h *MockAlonzoBlockHeader) Era() lcommon.Era {
	return alonzo.EraAlonzo
}

// Cbor returns the CBOR encoding of the header
func (h *MockAlonzoBlockHeader) Cbor() []byte {
	return nil
}

// BlockBodyHash returns the block body hash
func (h *MockAlonzoBlockHeader) BlockBodyHash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

// AlonzoHeaderBuilder implements HeaderBuilder for Alonzo era block headers
type AlonzoHeaderBuilder struct {
	slot        uint64
	blockNumber uint64
	hash        lcommon.Blake2b256
	prevHash    lcommon.Blake2b256
	hashErr     error
	prevHashErr error
}

// NewAlonzoHeaderBuilder creates a new AlonzoHeaderBuilder
func NewAlonzoHeaderBuilder() *AlonzoHeaderBuilder {
	return &AlonzoHeaderBuilder{}
}

// WithSlot sets the slot number for the header
func (b *AlonzoHeaderBuilder) WithSlot(slot uint64) HeaderBuilder {
	b.slot = slot
	return b
}

// WithBlockNumber sets the block number
func (b *AlonzoHeaderBuilder) WithBlockNumber(number uint64) HeaderBuilder {
	b.blockNumber = number
	return b
}

// WithHash sets the block hash
func (b *AlonzoHeaderBuilder) WithHash(hash []byte) HeaderBuilder {
	if len(hash) != len(b.hash) {
		b.hashErr = fmt.Errorf(
			"hash must be exactly %d bytes, got %d",
			len(b.hash),
			len(hash),
		)
		return b
	}
	copy(b.hash[:], hash)
	return b
}

// WithPrevHash sets the previous block hash
func (b *AlonzoHeaderBuilder) WithPrevHash(prevHash []byte) HeaderBuilder {
	if len(prevHash) != len(b.prevHash) {
		b.prevHashErr = fmt.Errorf(
			"prevHash must be exactly %d bytes, got %d",
			len(b.prevHash),
			len(prevHash),
		)
		return b
	}
	copy(b.prevHash[:], prevHash)
	return b
}

// Build constructs a mock Alonzo block header that satisfies the ledger.BlockHeader interface
func (b *AlonzoHeaderBuilder) Build() (ledger.BlockHeader, error) {
	if b.hashErr != nil {
		return nil, b.hashErr
	}
	if b.prevHashErr != nil {
		return nil, b.prevHashErr
	}
	return &MockAlonzoBlockHeader{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
	}, nil
}

// BuildCbor returns the CBOR encoding of the header
// Currently returns nil as a placeholder
func (b *AlonzoHeaderBuilder) BuildCbor() ([]byte, error) {
	// TODO: Implement CBOR encoding when needed
	return nil, nil
}
