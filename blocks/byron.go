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
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	utxorpc "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
)

// ByronBlockBuilder implements BlockBuilder for Byron era blocks
type ByronBlockBuilder struct {
	slot         uint64
	blockNumber  uint64
	hash         lcommon.Blake2b256
	prevHash     lcommon.Blake2b256
	transactions []TxBuilder
	isEBB        bool
	hashErr      error
	prevHashErr  error
}

// NewByronBlockBuilder creates a new ByronBlockBuilder for regular Byron main blocks
func NewByronBlockBuilder() *ByronBlockBuilder {
	return &ByronBlockBuilder{
		isEBB: false,
	}
}

// NewByronEBBBlockBuilder creates a new ByronBlockBuilder for epoch boundary blocks
func NewByronEBBBlockBuilder() *ByronBlockBuilder {
	return &ByronBlockBuilder{
		isEBB: true,
	}
}

// WithSlot sets the slot number for the block
func (b *ByronBlockBuilder) WithSlot(slot uint64) BlockBuilder {
	b.slot = slot
	return b
}

// WithBlockNumber sets the block number
func (b *ByronBlockBuilder) WithBlockNumber(number uint64) BlockBuilder {
	b.blockNumber = number
	return b
}

// WithHash sets the block hash
func (b *ByronBlockBuilder) WithHash(hash []byte) BlockBuilder {
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
func (b *ByronBlockBuilder) WithPrevHash(prevHash []byte) BlockBuilder {
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
func (b *ByronBlockBuilder) WithTransactions(
	txs ...TxBuilder,
) BlockBuilder {
	b.transactions = txs
	return b
}

// WithEBB sets whether this is an epoch boundary block
func (b *ByronBlockBuilder) WithEBB(isEBB bool) *ByronBlockBuilder {
	b.isEBB = isEBB
	return b
}

// Build constructs a mock Byron block that satisfies the ledger.Block interface
func (b *ByronBlockBuilder) Build() (ledger.Block, error) {
	if b.hashErr != nil {
		return nil, b.hashErr
	}
	if b.prevHashErr != nil {
		return nil, b.prevHashErr
	}
	mockBlock := &MockByronBlock{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
		isEBB:       b.isEBB,
	}

	// Build transactions if any (EBB blocks don't have transactions)
	if !b.isEBB {
		for _, txBuilder := range b.transactions {
			tx, err := txBuilder.Build()
			if err != nil {
				return nil, err
			}
			mockBlock.transactions = append(mockBlock.transactions, tx)
		}
	}

	return mockBlock, nil
}

// BuildCbor returns the CBOR encoding of the block
// Currently returns nil as a placeholder
func (b *ByronBlockBuilder) BuildCbor() ([]byte, error) {
	// TODO: Implement CBOR encoding when needed
	return nil, nil
}

// MockByronBlock is a mock implementation of a Byron block that satisfies ledger.Block
type MockByronBlock struct {
	slot         uint64
	blockNumber  uint64
	hash         lcommon.Blake2b256
	prevHash     lcommon.Blake2b256
	transactions []lcommon.Transaction
	isEBB        bool
}

// Type returns the block type identifier for Byron
func (b *MockByronBlock) Type() int {
	if b.isEBB {
		return byron.BlockTypeByronEbb
	}
	return byron.BlockTypeByronMain
}

// Hash returns the block hash
func (b *MockByronBlock) Hash() lcommon.Blake2b256 {
	return b.hash
}

// PrevHash returns the previous block hash
func (b *MockByronBlock) PrevHash() lcommon.Blake2b256 {
	return b.prevHash
}

// BlockNumber returns the block number
func (b *MockByronBlock) BlockNumber() uint64 {
	return b.blockNumber
}

// SlotNumber returns the slot number
func (b *MockByronBlock) SlotNumber() uint64 {
	return b.slot
}

// IssuerVkey returns the issuer verification key
func (b *MockByronBlock) IssuerVkey() lcommon.IssuerVkey {
	// Byron blocks don't have an issuer
	return lcommon.IssuerVkey{}
}

// BlockBodySize returns the block body size
func (b *MockByronBlock) BlockBodySize() uint64 {
	return 0
}

// Era returns the Byron era
func (b *MockByronBlock) Era() lcommon.Era {
	return byron.EraByron
}

// Cbor returns the CBOR encoding of the block
func (b *MockByronBlock) Cbor() []byte {
	return nil
}

// BlockBodyHash returns the block body hash
func (b *MockByronBlock) BlockBodyHash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

// Header returns the block header
func (b *MockByronBlock) Header() ledger.BlockHeader {
	return &MockByronBlockHeader{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
		isEBB:       b.isEBB,
	}
}

// Transactions returns the list of transactions in the block
func (b *MockByronBlock) Transactions() []lcommon.Transaction {
	// EBB blocks don't have transactions
	if b.isEBB {
		return nil
	}
	return b.transactions
}

// Utxorpc returns the block in UTxO RPC format
func (b *MockByronBlock) Utxorpc() (*utxorpc.Block, error) {
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

// MockByronBlockHeader is a mock implementation of a Byron block header
type MockByronBlockHeader struct {
	slot        uint64
	blockNumber uint64
	hash        lcommon.Blake2b256
	prevHash    lcommon.Blake2b256
	isEBB       bool
}

// Hash returns the block header hash
func (h *MockByronBlockHeader) Hash() lcommon.Blake2b256 {
	return h.hash
}

// PrevHash returns the previous block hash
func (h *MockByronBlockHeader) PrevHash() lcommon.Blake2b256 {
	return h.prevHash
}

// BlockNumber returns the block number
func (h *MockByronBlockHeader) BlockNumber() uint64 {
	return h.blockNumber
}

// SlotNumber returns the slot number
func (h *MockByronBlockHeader) SlotNumber() uint64 {
	return h.slot
}

// IssuerVkey returns the issuer verification key
func (h *MockByronBlockHeader) IssuerVkey() lcommon.IssuerVkey {
	// Byron blocks don't have an issuer
	return lcommon.IssuerVkey{}
}

// BlockBodySize returns the block body size
func (h *MockByronBlockHeader) BlockBodySize() uint64 {
	return 0
}

// Era returns the Byron era
func (h *MockByronBlockHeader) Era() lcommon.Era {
	return byron.EraByron
}

// Cbor returns the CBOR encoding of the header
func (h *MockByronBlockHeader) Cbor() []byte {
	return nil
}

// BlockBodyHash returns the block body hash
func (h *MockByronBlockHeader) BlockBodyHash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

// ByronHeaderBuilder implements HeaderBuilder for Byron era block headers
type ByronHeaderBuilder struct {
	slot        uint64
	blockNumber uint64
	hash        lcommon.Blake2b256
	prevHash    lcommon.Blake2b256
	isEBB       bool
	hashErr     error
	prevHashErr error
}

// NewByronHeaderBuilder creates a new ByronHeaderBuilder
func NewByronHeaderBuilder() *ByronHeaderBuilder {
	return &ByronHeaderBuilder{}
}

// WithSlot sets the slot number for the header
func (b *ByronHeaderBuilder) WithSlot(slot uint64) HeaderBuilder {
	b.slot = slot
	return b
}

// WithBlockNumber sets the block number
func (b *ByronHeaderBuilder) WithBlockNumber(number uint64) HeaderBuilder {
	b.blockNumber = number
	return b
}

// WithHash sets the block hash
func (b *ByronHeaderBuilder) WithHash(hash []byte) HeaderBuilder {
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
func (b *ByronHeaderBuilder) WithPrevHash(prevHash []byte) HeaderBuilder {
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

// WithEBB sets whether this is an epoch boundary block header
func (b *ByronHeaderBuilder) WithEBB(isEBB bool) *ByronHeaderBuilder {
	b.isEBB = isEBB
	return b
}

// Build constructs a mock Byron block header that satisfies the ledger.BlockHeader interface
func (b *ByronHeaderBuilder) Build() (ledger.BlockHeader, error) {
	if b.hashErr != nil {
		return nil, b.hashErr
	}
	if b.prevHashErr != nil {
		return nil, b.prevHashErr
	}
	return &MockByronBlockHeader{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
		isEBB:       b.isEBB,
	}, nil
}

// BuildCbor returns the CBOR encoding of the header
// Currently returns nil as a placeholder
func (b *ByronHeaderBuilder) BuildCbor() ([]byte, error) {
	// TODO: Implement CBOR encoding when needed
	return nil, nil
}
