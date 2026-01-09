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
	"crypto/sha256"
	"encoding/binary"

	"github.com/blinklabs-io/gouroboros/ledger"
)

// ChainBuilder defines an interface for building linked block sequences
type ChainBuilder interface {
	WithStartSlot(slot uint64) ChainBuilder
	WithSlotInterval(interval uint64) ChainBuilder
	WithEra(era uint) ChainBuilder
	AddBlock(opts ...BlockOption) ChainBuilder
	AddBlocks(count int) ChainBuilder
	Build() ([]ledger.Block, error)
}

// BlockOption is a function that modifies a block configuration
type BlockOption func(*blockConfig)

// blockConfig holds configuration for a single block in the chain
type blockConfig struct {
	transactions []TxBuilder
	customHash   []byte
}

// WithTxBuilders adds transaction builders to the block
func WithTxBuilders(txs ...TxBuilder) BlockOption {
	return func(cfg *blockConfig) {
		cfg.transactions = append(cfg.transactions, txs...)
	}
}

// WithCustomHash sets a custom hash for the block
func WithCustomHash(hash []byte) BlockOption {
	return func(cfg *blockConfig) {
		cfg.customHash = hash
	}
}

// ConwayChainBuilder implements ChainBuilder for Conway era blocks.
// Note: This builder always creates Conway era blocks. The WithEra method
// is a no-op, preserved for interface compatibility. For other eras,
// create era-specific chain builders.
type ConwayChainBuilder struct {
	startSlot     uint64
	slotInterval  uint64
	blockConfigs  []*blockConfig
	genesisHash   []byte
	startBlockNum uint64
}

// NewConwayChainBuilder creates a new ConwayChainBuilder with default settings
func NewConwayChainBuilder() *ConwayChainBuilder {
	return &ConwayChainBuilder{
		startSlot:     0,
		slotInterval:  1,
		blockConfigs:  make([]*blockConfig, 0),
		genesisHash:   make([]byte, 32), // Zero hash as genesis
		startBlockNum: 0,
	}
}

// WithStartSlot sets the starting slot number for the chain
func (b *ConwayChainBuilder) WithStartSlot(slot uint64) ChainBuilder {
	b.startSlot = slot
	return b
}

// WithSlotInterval sets the interval between consecutive slots.
// If interval is 0, it defaults to 1 to prevent same-slot blocks.
func (b *ConwayChainBuilder) WithSlotInterval(interval uint64) ChainBuilder {
	if interval == 0 {
		interval = 1
	}
	b.slotInterval = interval
	return b
}

// WithEra is a no-op for ConwayChainBuilder (always produces Conway blocks).
// This method exists for ChainBuilder interface compatibility.
// For other eras, use era-specific chain builders.
func (b *ConwayChainBuilder) WithEra(_ uint) ChainBuilder {
	return b
}

// WithGenesisHash sets the genesis hash (prevHash of the first block)
func (b *ConwayChainBuilder) WithGenesisHash(hash []byte) *ConwayChainBuilder {
	b.genesisHash = hash
	return b
}

// WithStartBlockNumber sets the starting block number
func (b *ConwayChainBuilder) WithStartBlockNumber(
	blockNum uint64,
) *ConwayChainBuilder {
	b.startBlockNum = blockNum
	return b
}

// AddBlock adds a block with optional configuration to the chain
func (b *ConwayChainBuilder) AddBlock(opts ...BlockOption) ChainBuilder {
	cfg := &blockConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	b.blockConfigs = append(b.blockConfigs, cfg)
	return b
}

// AddBlocks adds multiple empty blocks to the chain
func (b *ConwayChainBuilder) AddBlocks(count int) ChainBuilder {
	for range count {
		b.blockConfigs = append(b.blockConfigs, &blockConfig{})
	}
	return b
}

// Build constructs the chain of linked blocks
func (b *ConwayChainBuilder) Build() ([]ledger.Block, error) {
	if len(b.blockConfigs) == 0 {
		return []ledger.Block{}, nil
	}

	blocks := make([]ledger.Block, 0, len(b.blockConfigs))
	prevHash := b.genesisHash

	for i, cfg := range b.blockConfigs {
		// #nosec G115 - i is bounded by slice length which fits in int
		slot := b.startSlot + uint64(i)*b.slotInterval
		// #nosec G115 - i is bounded by slice length which fits in int
		blockNum := b.startBlockNum + uint64(i)

		// Generate block hash (either custom or deterministic)
		var blockHash []byte
		if cfg.customHash != nil {
			blockHash = cfg.customHash
		} else {
			blockHash = generateBlockHash(slot, blockNum, prevHash)
		}

		// Build the block using ConwayBlockBuilder
		builder := NewConwayBlockBuilder().
			WithSlot(slot).
			WithBlockNumber(blockNum).
			WithHash(blockHash).
			WithPrevHash(prevHash)

		// Add transactions if any
		if len(cfg.transactions) > 0 {
			builder.WithTransactions(cfg.transactions...)
		}

		block, err := builder.Build()
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
		prevHash = blockHash
	}

	return blocks, nil
}

// generateBlockHash creates a deterministic hash based on block parameters
func generateBlockHash(slot, blockNum uint64, prevHash []byte) []byte {
	// Create a deterministic hash from slot, block number, and previous hash
	data := make([]byte, 16, 16+len(prevHash))
	binary.BigEndian.PutUint64(data[0:8], slot)
	binary.BigEndian.PutUint64(data[8:16], blockNum)
	data = append(data, prevHash...)

	hash := sha256.Sum256(data)
	return hash[:]
}
