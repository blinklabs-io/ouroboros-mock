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
	"github.com/blinklabs-io/gouroboros/ledger/allegra"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	utxorpc "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
)

// AllegraBlockBuilder implements BlockBuilder for Allegra era blocks
type AllegraBlockBuilder struct {
	slot         uint64
	blockNumber  uint64
	hash         lcommon.Blake2b256
	prevHash     lcommon.Blake2b256
	transactions []TxBuilder
	hashErr      error
	prevHashErr  error
}

// NewAllegraBlockBuilder creates a new AllegraBlockBuilder
func NewAllegraBlockBuilder() *AllegraBlockBuilder {
	return &AllegraBlockBuilder{}
}

// WithSlot sets the slot number for the block
func (b *AllegraBlockBuilder) WithSlot(slot uint64) BlockBuilder {
	b.slot = slot
	return b
}

// WithBlockNumber sets the block number
func (b *AllegraBlockBuilder) WithBlockNumber(number uint64) BlockBuilder {
	b.blockNumber = number
	return b
}

// WithHash sets the block hash
func (b *AllegraBlockBuilder) WithHash(hash []byte) BlockBuilder {
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
func (b *AllegraBlockBuilder) WithPrevHash(prevHash []byte) BlockBuilder {
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
func (b *AllegraBlockBuilder) WithTransactions(
	txs ...TxBuilder,
) BlockBuilder {
	b.transactions = txs
	return b
}

// Build constructs a mock Allegra block that satisfies the ledger.Block interface
func (b *AllegraBlockBuilder) Build() (ledger.Block, error) {
	if b.hashErr != nil {
		return nil, b.hashErr
	}
	if b.prevHashErr != nil {
		return nil, b.prevHashErr
	}
	mockBlock := &MockAllegraBlock{
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
func (b *AllegraBlockBuilder) BuildCbor() ([]byte, error) {
	// TODO: Implement CBOR encoding when needed
	return nil, nil
}

// MockAllegraBlock is a mock implementation of an Allegra block that satisfies ledger.Block
type MockAllegraBlock struct {
	slot         uint64
	blockNumber  uint64
	hash         lcommon.Blake2b256
	prevHash     lcommon.Blake2b256
	transactions []lcommon.Transaction
}

// Type returns the block type identifier for Allegra
func (b *MockAllegraBlock) Type() int {
	return allegra.BlockTypeAllegra
}

// Hash returns the block hash
func (b *MockAllegraBlock) Hash() lcommon.Blake2b256 {
	return b.hash
}

// PrevHash returns the previous block hash
func (b *MockAllegraBlock) PrevHash() lcommon.Blake2b256 {
	return b.prevHash
}

// BlockNumber returns the block number
func (b *MockAllegraBlock) BlockNumber() uint64 {
	return b.blockNumber
}

// SlotNumber returns the slot number
func (b *MockAllegraBlock) SlotNumber() uint64 {
	return b.slot
}

// IssuerVkey returns the issuer verification key
func (b *MockAllegraBlock) IssuerVkey() lcommon.IssuerVkey {
	return lcommon.IssuerVkey{}
}

// BlockBodySize returns the block body size
func (b *MockAllegraBlock) BlockBodySize() uint64 {
	return 0
}

// Era returns the Allegra era
func (b *MockAllegraBlock) Era() lcommon.Era {
	return allegra.EraAllegra
}

// Cbor returns the CBOR encoding of the block
func (b *MockAllegraBlock) Cbor() []byte {
	return nil
}

// BlockBodyHash returns the block body hash
func (b *MockAllegraBlock) BlockBodyHash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

// Header returns the block header
func (b *MockAllegraBlock) Header() ledger.BlockHeader {
	return &MockAllegraBlockHeader{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
	}
}

// Transactions returns the list of transactions in the block
func (b *MockAllegraBlock) Transactions() []lcommon.Transaction {
	return b.transactions
}

// Utxorpc returns the block in UTxO RPC format
func (b *MockAllegraBlock) Utxorpc() (*utxorpc.Block, error) {
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

// MockAllegraBlockHeader is a mock implementation of an Allegra block header
type MockAllegraBlockHeader struct {
	slot        uint64
	blockNumber uint64
	hash        lcommon.Blake2b256
	prevHash    lcommon.Blake2b256
}

// Hash returns the block header hash
func (h *MockAllegraBlockHeader) Hash() lcommon.Blake2b256 {
	return h.hash
}

// PrevHash returns the previous block hash
func (h *MockAllegraBlockHeader) PrevHash() lcommon.Blake2b256 {
	return h.prevHash
}

// BlockNumber returns the block number
func (h *MockAllegraBlockHeader) BlockNumber() uint64 {
	return h.blockNumber
}

// SlotNumber returns the slot number
func (h *MockAllegraBlockHeader) SlotNumber() uint64 {
	return h.slot
}

// IssuerVkey returns the issuer verification key
func (h *MockAllegraBlockHeader) IssuerVkey() lcommon.IssuerVkey {
	return lcommon.IssuerVkey{}
}

// BlockBodySize returns the block body size
func (h *MockAllegraBlockHeader) BlockBodySize() uint64 {
	return 0
}

// Era returns the Allegra era
func (h *MockAllegraBlockHeader) Era() lcommon.Era {
	return allegra.EraAllegra
}

// Cbor returns the CBOR encoding of the header
func (h *MockAllegraBlockHeader) Cbor() []byte {
	return nil
}

// BlockBodyHash returns the block body hash
func (h *MockAllegraBlockHeader) BlockBodyHash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

// AllegraHeaderBuilder implements HeaderBuilder for Allegra era block headers
type AllegraHeaderBuilder struct {
	slot        uint64
	blockNumber uint64
	hash        lcommon.Blake2b256
	prevHash    lcommon.Blake2b256
	hashErr     error
	prevHashErr error
}

// NewAllegraHeaderBuilder creates a new AllegraHeaderBuilder
func NewAllegraHeaderBuilder() *AllegraHeaderBuilder {
	return &AllegraHeaderBuilder{}
}

// WithSlot sets the slot number for the header
func (b *AllegraHeaderBuilder) WithSlot(slot uint64) HeaderBuilder {
	b.slot = slot
	return b
}

// WithBlockNumber sets the block number
func (b *AllegraHeaderBuilder) WithBlockNumber(number uint64) HeaderBuilder {
	b.blockNumber = number
	return b
}

// WithHash sets the block hash
func (b *AllegraHeaderBuilder) WithHash(hash []byte) HeaderBuilder {
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
func (b *AllegraHeaderBuilder) WithPrevHash(prevHash []byte) HeaderBuilder {
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

// Build constructs a mock Allegra block header that satisfies the ledger.BlockHeader interface
func (b *AllegraHeaderBuilder) Build() (ledger.BlockHeader, error) {
	if b.hashErr != nil {
		return nil, b.hashErr
	}
	if b.prevHashErr != nil {
		return nil, b.prevHashErr
	}
	return &MockAllegraBlockHeader{
		slot:        b.slot,
		blockNumber: b.blockNumber,
		hash:        b.hash,
		prevHash:    b.prevHash,
	}, nil
}

// BuildCbor returns the CBOR encoding of the header
// Currently returns nil as a placeholder
func (b *AllegraHeaderBuilder) BuildCbor() ([]byte, error) {
	// TODO: Implement CBOR encoding when needed
	return nil, nil
}
