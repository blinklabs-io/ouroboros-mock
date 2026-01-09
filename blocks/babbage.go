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
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	utxorpc "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
)

// BabbageBlockBuilder implements BlockBuilder for Babbage era blocks.
// Babbage introduced reference inputs and inline datums, enhancing smart contract capabilities.
type BabbageBlockBuilder struct {
	slot         uint64
	blockNumber  uint64
	hash         lcommon.Blake2b256
	prevHash     lcommon.Blake2b256
	transactions []TxBuilder
	hashErr      error
	prevHashErr  error
}

// NewBabbageBlockBuilder creates a new BabbageBlockBuilder
func NewBabbageBlockBuilder() *BabbageBlockBuilder {
	return &BabbageBlockBuilder{}
}

// WithSlot sets the slot number for the block
func (b *BabbageBlockBuilder) WithSlot(slot uint64) BlockBuilder {
	b.slot = slot
	return b
}

// WithBlockNumber sets the block number
func (b *BabbageBlockBuilder) WithBlockNumber(number uint64) BlockBuilder {
	b.blockNumber = number
	return b
}

// WithHash sets the block hash
func (b *BabbageBlockBuilder) WithHash(hash []byte) BlockBuilder {
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
func (b *BabbageBlockBuilder) WithPrevHash(prevHash []byte) BlockBuilder {
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
func (b *BabbageBlockBuilder) WithTransactions(
	txs ...TxBuilder,
) BlockBuilder {
	b.transactions = txs
	return b
}

// Build constructs a mock Babbage block that satisfies the ledger.Block interface
func (b *BabbageBlockBuilder) Build() (ledger.Block, error) {
	if b.hashErr != nil {
		return nil, b.hashErr
	}
	if b.prevHashErr != nil {
		return nil, b.prevHashErr
	}
	mockBlock := &MockBabbageBlock{
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
func (b *BabbageBlockBuilder) BuildCbor() ([]byte, error) {
	// TODO: Implement CBOR encoding when needed
	return nil, nil
}

// MockBabbageBlock is a mock implementation of a Babbage block that satisfies ledger.Block.
// Babbage was the era that introduced reference inputs and inline datums to Cardano.
type MockBabbageBlock struct {
	slot         uint64
	blockNumber  uint64
	hash         lcommon.Blake2b256
	prevHash     lcommon.Blake2b256
	transactions []lcommon.Transaction
}

// Type returns the block type identifier for Babbage
func (b *MockBabbageBlock) Type() int {
	return babbage.BlockTypeBabbage
}

// Hash returns the block hash
func (b *MockBabbageBlock) Hash() lcommon.Blake2b256 {
	return b.hash
}

// PrevHash returns the previous block hash
func (b *MockBabbageBlock) PrevHash() lcommon.Blake2b256 {
	return b.prevHash
}

// BlockNumber returns the block number
func (b *MockBabbageBlock) BlockNumber() uint64 {
	return b.blockNumber
}

// SlotNumber returns the slot number
func (b *MockBabbageBlock) SlotNumber() uint64 {
	return b.slot
}

// IssuerVkey returns the issuer verification key
func (b *MockBabbageBlock) IssuerVkey() lcommon.IssuerVkey {
	return lcommon.IssuerVkey{}
}

// BlockBodySize returns the block body size
func (b *MockBabbageBlock) BlockBodySize() uint64 {
	return 0
}

// Era returns the Babbage era
func (b *MockBabbageBlock) Era() lcommon.Era {
	return babbage.EraBabbage
}

// Cbor returns the CBOR encoding of the block
func (b *MockBabbageBlock) Cbor() []byte {
	return nil
}

// BlockBodyHash returns the block body hash
func (b *MockBabbageBlock) BlockBodyHash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

// Header returns the block header
func (b *MockBabbageBlock) Header() ledger.BlockHeader {
	return &MockBabbageBlockHeader{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
	}
}

// Transactions returns the list of transactions in the block
func (b *MockBabbageBlock) Transactions() []lcommon.Transaction {
	return b.transactions
}

// Utxorpc returns the block in UTxO RPC format
func (b *MockBabbageBlock) Utxorpc() (*utxorpc.Block, error) {
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

// MockBabbageBlockHeader is a mock implementation of a Babbage block header
type MockBabbageBlockHeader struct {
	slot        uint64
	blockNumber uint64
	hash        lcommon.Blake2b256
	prevHash    lcommon.Blake2b256
}

// Hash returns the block header hash
func (h *MockBabbageBlockHeader) Hash() lcommon.Blake2b256 {
	return h.hash
}

// PrevHash returns the previous block hash
func (h *MockBabbageBlockHeader) PrevHash() lcommon.Blake2b256 {
	return h.prevHash
}

// BlockNumber returns the block number
func (h *MockBabbageBlockHeader) BlockNumber() uint64 {
	return h.blockNumber
}

// SlotNumber returns the slot number
func (h *MockBabbageBlockHeader) SlotNumber() uint64 {
	return h.slot
}

// IssuerVkey returns the issuer verification key
func (h *MockBabbageBlockHeader) IssuerVkey() lcommon.IssuerVkey {
	return lcommon.IssuerVkey{}
}

// BlockBodySize returns the block body size
func (h *MockBabbageBlockHeader) BlockBodySize() uint64 {
	return 0
}

// Era returns the Babbage era
func (h *MockBabbageBlockHeader) Era() lcommon.Era {
	return babbage.EraBabbage
}

// Cbor returns the CBOR encoding of the header
func (h *MockBabbageBlockHeader) Cbor() []byte {
	return nil
}

// BlockBodyHash returns the block body hash
func (h *MockBabbageBlockHeader) BlockBodyHash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

// BabbageHeaderBuilder implements HeaderBuilder for Babbage era block headers
type BabbageHeaderBuilder struct {
	slot        uint64
	blockNumber uint64
	hash        lcommon.Blake2b256
	prevHash    lcommon.Blake2b256
	hashErr     error
	prevHashErr error
}

// NewBabbageHeaderBuilder creates a new BabbageHeaderBuilder
func NewBabbageHeaderBuilder() *BabbageHeaderBuilder {
	return &BabbageHeaderBuilder{}
}

// WithSlot sets the slot number for the header
func (b *BabbageHeaderBuilder) WithSlot(slot uint64) HeaderBuilder {
	b.slot = slot
	return b
}

// WithBlockNumber sets the block number
func (b *BabbageHeaderBuilder) WithBlockNumber(number uint64) HeaderBuilder {
	b.blockNumber = number
	return b
}

// WithHash sets the block hash
func (b *BabbageHeaderBuilder) WithHash(hash []byte) HeaderBuilder {
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
func (b *BabbageHeaderBuilder) WithPrevHash(prevHash []byte) HeaderBuilder {
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

// Build constructs a mock Babbage block header that satisfies the ledger.BlockHeader interface
func (b *BabbageHeaderBuilder) Build() (ledger.BlockHeader, error) {
	if b.hashErr != nil {
		return nil, b.hashErr
	}
	if b.prevHashErr != nil {
		return nil, b.prevHashErr
	}
	return &MockBabbageBlockHeader{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
	}, nil
}

// BuildCbor returns the CBOR encoding of the header
// Currently returns nil as a placeholder
func (b *BabbageHeaderBuilder) BuildCbor() ([]byte, error) {
	// TODO: Implement CBOR encoding when needed
	return nil, nil
}
