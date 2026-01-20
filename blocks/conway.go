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
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	utxorpc "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
)

// ConwayBlockBuilder implements BlockBuilder for Conway era blocks
type ConwayBlockBuilder struct {
	slot         uint64
	blockNumber  uint64
	hash         lcommon.Blake2b256
	prevHash     lcommon.Blake2b256
	transactions []TxBuilder
	hashErr      error
	prevHashErr  error
}

// NewConwayBlockBuilder creates a new ConwayBlockBuilder
func NewConwayBlockBuilder() *ConwayBlockBuilder {
	return &ConwayBlockBuilder{}
}

// WithSlot sets the slot number for the block
func (b *ConwayBlockBuilder) WithSlot(slot uint64) BlockBuilder {
	b.slot = slot
	return b
}

// WithBlockNumber sets the block number
func (b *ConwayBlockBuilder) WithBlockNumber(number uint64) BlockBuilder {
	b.blockNumber = number
	return b
}

// WithHash sets the block hash
func (b *ConwayBlockBuilder) WithHash(hash []byte) BlockBuilder {
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
func (b *ConwayBlockBuilder) WithPrevHash(prevHash []byte) BlockBuilder {
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
func (b *ConwayBlockBuilder) WithTransactions(
	txs ...TxBuilder,
) BlockBuilder {
	b.transactions = txs
	return b
}

// Build constructs a mock Conway block that satisfies the ledger.Block interface
func (b *ConwayBlockBuilder) Build() (ledger.Block, error) {
	if b.hashErr != nil {
		return nil, b.hashErr
	}
	if b.prevHashErr != nil {
		return nil, b.prevHashErr
	}
	mockBlock := &MockConwayBlock{
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
func (b *ConwayBlockBuilder) BuildCbor() ([]byte, error) {
	// TODO: Implement CBOR encoding when needed
	return nil, nil
}

// MockConwayBlock is a mock implementation of a Conway block that satisfies ledger.Block
type MockConwayBlock struct {
	slot         uint64
	blockNumber  uint64
	hash         lcommon.Blake2b256
	prevHash     lcommon.Blake2b256
	transactions []lcommon.Transaction
}

// Type returns the block type identifier for Conway
func (b *MockConwayBlock) Type() int {
	return conway.BlockTypeConway
}

// Hash returns the block hash
func (b *MockConwayBlock) Hash() lcommon.Blake2b256 {
	return b.hash
}

// PrevHash returns the previous block hash
func (b *MockConwayBlock) PrevHash() lcommon.Blake2b256 {
	return b.prevHash
}

// BlockNumber returns the block number
func (b *MockConwayBlock) BlockNumber() uint64 {
	return b.blockNumber
}

// SlotNumber returns the slot number
func (b *MockConwayBlock) SlotNumber() uint64 {
	return b.slot
}

// IssuerVkey returns the issuer verification key
func (b *MockConwayBlock) IssuerVkey() lcommon.IssuerVkey {
	return lcommon.IssuerVkey{}
}

// BlockBodySize returns the block body size
func (b *MockConwayBlock) BlockBodySize() uint64 {
	return 0
}

// Era returns the Conway era
func (b *MockConwayBlock) Era() lcommon.Era {
	return conway.EraConway
}

// Cbor returns the CBOR encoding of the block
func (b *MockConwayBlock) Cbor() []byte {
	return nil
}

// BlockBodyHash returns the block body hash
func (b *MockConwayBlock) BlockBodyHash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

// Header returns the block header
func (b *MockConwayBlock) Header() ledger.BlockHeader {
	return &MockConwayBlockHeader{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
	}
}

// Transactions returns the list of transactions in the block
func (b *MockConwayBlock) Transactions() []lcommon.Transaction {
	return b.transactions
}

// Utxorpc returns the block in UTxO RPC format
func (b *MockConwayBlock) Utxorpc() (*utxorpc.Block, error) {
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

// MockConwayBlockHeader is a mock implementation of a Conway block header
type MockConwayBlockHeader struct {
	slot        uint64
	blockNumber uint64
	hash        lcommon.Blake2b256
	prevHash    lcommon.Blake2b256
}

// Hash returns the block header hash
func (h *MockConwayBlockHeader) Hash() lcommon.Blake2b256 {
	return h.hash
}

// PrevHash returns the previous block hash
func (h *MockConwayBlockHeader) PrevHash() lcommon.Blake2b256 {
	return h.prevHash
}

// BlockNumber returns the block number
func (h *MockConwayBlockHeader) BlockNumber() uint64 {
	return h.blockNumber
}

// SlotNumber returns the slot number
func (h *MockConwayBlockHeader) SlotNumber() uint64 {
	return h.slot
}

// IssuerVkey returns the issuer verification key
func (h *MockConwayBlockHeader) IssuerVkey() lcommon.IssuerVkey {
	return lcommon.IssuerVkey{}
}

// BlockBodySize returns the block body size
func (h *MockConwayBlockHeader) BlockBodySize() uint64 {
	return 0
}

// Era returns the Conway era
func (h *MockConwayBlockHeader) Era() lcommon.Era {
	return conway.EraConway
}

// Cbor returns the CBOR encoding of the header
func (h *MockConwayBlockHeader) Cbor() []byte {
	return nil
}

// BlockBodyHash returns the block body hash
func (h *MockConwayBlockHeader) BlockBodyHash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

// ConwayHeaderBuilder implements HeaderBuilder for Conway era block headers
type ConwayHeaderBuilder struct {
	slot        uint64
	blockNumber uint64
	hash        lcommon.Blake2b256
	prevHash    lcommon.Blake2b256
	hashErr     error
	prevHashErr error
}

// NewConwayHeaderBuilder creates a new ConwayHeaderBuilder
func NewConwayHeaderBuilder() *ConwayHeaderBuilder {
	return &ConwayHeaderBuilder{}
}

// WithSlot sets the slot number for the header
func (b *ConwayHeaderBuilder) WithSlot(slot uint64) HeaderBuilder {
	b.slot = slot
	return b
}

// WithBlockNumber sets the block number
func (b *ConwayHeaderBuilder) WithBlockNumber(number uint64) HeaderBuilder {
	b.blockNumber = number
	return b
}

// WithHash sets the block hash
func (b *ConwayHeaderBuilder) WithHash(hash []byte) HeaderBuilder {
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
func (b *ConwayHeaderBuilder) WithPrevHash(prevHash []byte) HeaderBuilder {
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

// Build constructs a mock Conway block header that satisfies the ledger.BlockHeader interface
func (b *ConwayHeaderBuilder) Build() (ledger.BlockHeader, error) {
	if b.hashErr != nil {
		return nil, b.hashErr
	}
	if b.prevHashErr != nil {
		return nil, b.prevHashErr
	}
	return &MockConwayBlockHeader{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
	}, nil
}

// BuildCbor returns the CBOR encoding of the header
// Currently returns nil as a placeholder
func (b *ConwayHeaderBuilder) BuildCbor() ([]byte, error) {
	// TODO: Implement CBOR encoding when needed
	return nil, nil
}
