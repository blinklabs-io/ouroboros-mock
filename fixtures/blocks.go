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

package fixtures

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/ouroboros-mock/blocks"
)

// Pre-defined hashes for testing (32 bytes each)
var (
	// GenesisHash is the zero hash used as the genesis block's previous hash
	GenesisHash = make([]byte, 32)

	// TestHash1 through TestHash5 are pre-computed test hashes
	TestHash1 = hashFromString("test-block-hash-1")
	TestHash2 = hashFromString("test-block-hash-2")
	TestHash3 = hashFromString("test-block-hash-3")
	TestHash4 = hashFromString("test-block-hash-4")
	TestHash5 = hashFromString("test-block-hash-5")
)

// hashFromString creates a deterministic 32-byte hash from a string
func hashFromString(s string) []byte {
	hash := sha256.Sum256([]byte(s))
	return hash[:]
}

// hashFromSlotAndBlock creates a deterministic hash from slot and block number
func hashFromSlotAndBlock(slot, blockNum uint64) []byte {
	data := make([]byte, 16)
	for i := range 8 {
		// #nosec G115 - i is in range [0,7], so 56-i*8 is in range [0,56]
		data[i] = byte(slot >> uint(56-i*8))
		// #nosec G115 - i is in range [0,7], so 56-i*8 is in range [0,56]
		data[i+8] = byte(blockNum >> uint(56-i*8))
	}
	hash := sha256.Sum256(data)
	return hash[:]
}

// NewMockByronBlock creates a simple Byron era block for testing
func NewMockByronBlock(slot, blockNum uint64) (ledger.Block, error) {
	hash := hashFromSlotAndBlock(slot, blockNum)
	return blocks.NewByronBlockBuilder().
		WithSlot(slot).
		WithBlockNumber(blockNum).
		WithHash(hash).
		WithPrevHash(GenesisHash).
		Build()
}

// NewMockByronEBB creates a Byron Epoch Boundary Block for testing
func NewMockByronEBB(slot, blockNum uint64) (ledger.Block, error) {
	hash := hashFromSlotAndBlock(slot, blockNum)
	return blocks.NewByronEBBBlockBuilder().
		WithSlot(slot).
		WithBlockNumber(blockNum).
		WithHash(hash).
		WithPrevHash(GenesisHash).
		Build()
}

// NewMockShelleyBlock creates a simple Shelley era block for testing
func NewMockShelleyBlock(slot, blockNum uint64) (ledger.Block, error) {
	hash := hashFromSlotAndBlock(slot, blockNum)
	return blocks.NewShelleyBlockBuilder().
		WithSlot(slot).
		WithBlockNumber(blockNum).
		WithHash(hash).
		WithPrevHash(GenesisHash).
		Build()
}

// NewMockAllegraBlock creates a simple Allegra era block for testing
func NewMockAllegraBlock(slot, blockNum uint64) (ledger.Block, error) {
	hash := hashFromSlotAndBlock(slot, blockNum)
	return blocks.NewAllegraBlockBuilder().
		WithSlot(slot).
		WithBlockNumber(blockNum).
		WithHash(hash).
		WithPrevHash(GenesisHash).
		Build()
}

// NewMockMaryBlock creates a simple Mary era block for testing
func NewMockMaryBlock(slot, blockNum uint64) (ledger.Block, error) {
	hash := hashFromSlotAndBlock(slot, blockNum)
	return blocks.NewMaryBlockBuilder().
		WithSlot(slot).
		WithBlockNumber(blockNum).
		WithHash(hash).
		WithPrevHash(GenesisHash).
		Build()
}

// NewMockAlonzoBlock creates a simple Alonzo era block for testing
func NewMockAlonzoBlock(slot, blockNum uint64) (ledger.Block, error) {
	hash := hashFromSlotAndBlock(slot, blockNum)
	return blocks.NewAlonzoBlockBuilder().
		WithSlot(slot).
		WithBlockNumber(blockNum).
		WithHash(hash).
		WithPrevHash(GenesisHash).
		Build()
}

// NewMockBabbageBlock creates a simple Babbage era block for testing
func NewMockBabbageBlock(slot, blockNum uint64) (ledger.Block, error) {
	hash := hashFromSlotAndBlock(slot, blockNum)
	return blocks.NewBabbageBlockBuilder().
		WithSlot(slot).
		WithBlockNumber(blockNum).
		WithHash(hash).
		WithPrevHash(GenesisHash).
		Build()
}

// NewMockConwayBlock creates a simple Conway era block for testing
func NewMockConwayBlock(slot, blockNum uint64) (ledger.Block, error) {
	hash := hashFromSlotAndBlock(slot, blockNum)
	return blocks.NewConwayBlockBuilder().
		WithSlot(slot).
		WithBlockNumber(blockNum).
		WithHash(hash).
		WithPrevHash(GenesisHash).
		Build()
}

// NewMockConwayChain creates a linked chain of Conway blocks for testing
func NewMockConwayChain(
	startSlot, startBlockNum uint64,
	count int,
) ([]ledger.Block, error) {
	return blocks.NewConwayChainBuilder().
		WithGenesisHash(GenesisHash).
		WithStartBlockNumber(startBlockNum).
		WithStartSlot(startSlot).
		AddBlocks(count).
		Build()
}

// ForkScenario represents a chain fork scenario for testing rollback handling
type ForkScenario struct {
	// MainChain is the original chain before the fork
	MainChain []ledger.Block
	// ForkPoint is the block number where the fork diverges
	ForkPoint uint64
	// ForkChain is the alternative chain that replaces part of MainChain
	ForkChain []ledger.Block
}

// NewMockForkScenario creates a fork scenario for testing rollback behavior.
// mainLength is the number of blocks in the main chain (must be > 0).
// forkPoint is the block number where the fork occurs.
// forkLength is the number of blocks in the forked chain after the fork point.
func NewMockForkScenario(
	mainLength, forkPoint, forkLength int,
) (*ForkScenario, error) {
	if mainLength <= 0 {
		return nil, errors.New("mainLength must be positive")
	}
	if forkPoint < 0 || forkPoint >= mainLength {
		forkPoint = mainLength / 2
	}

	// Build main chain
	mainChain, err := blocks.NewConwayChainBuilder().
		WithGenesisHash(GenesisHash).
		WithStartBlockNumber(0).
		WithStartSlot(0).
		AddBlocks(mainLength).
		Build()
	if err != nil {
		return nil, err
	}

	// Get the common ancestor's hash (block at forkPoint - 1)
	var forkPrevHash []byte
	if forkPoint == 0 {
		forkPrevHash = GenesisHash
	} else {
		forkPrevHash = mainChain[forkPoint-1].Hash().Bytes()
	}

	// Build fork chain starting from the fork point
	// Use different hashes to create an alternative chain
	forkChain := make([]ledger.Block, 0, forkLength)
	prevHash := forkPrevHash
	// #nosec G115 - forkPoint is validated above
	forkSlot := uint64(forkPoint)

	for i := range forkLength {
		// Create a different hash for the fork by adding a suffix
		// #nosec G115 - i is bounded by forkLength which fits in int
		slot := forkSlot + uint64(i)
		// #nosec G115 - forkPoint and i are bounded by slice lengths
		blockNum := uint64(forkPoint + i)
		hash := hashFromString(fmt.Sprintf("fork-%d", i))

		block, buildErr := blocks.NewConwayBlockBuilder().
			WithSlot(slot).
			WithBlockNumber(blockNum).
			WithHash(hash).
			WithPrevHash(prevHash).
			Build()
		if buildErr != nil {
			return nil, buildErr
		}

		forkChain = append(forkChain, block)
		prevHash = hash
	}

	return &ForkScenario{
		MainChain: mainChain,
		// #nosec G115 - forkPoint is validated above
		ForkPoint: uint64(forkPoint),
		ForkChain: forkChain,
	}, nil
}

// EraTransitionScenario represents an era transition for testing
type EraTransitionScenario struct {
	// PreTransitionBlocks are blocks before the era transition
	PreTransitionBlocks []ledger.Block
	// PostTransitionBlocks are blocks after the era transition
	PostTransitionBlocks []ledger.Block
	// TransitionSlot is the slot where the era changes
	TransitionSlot uint64
}

// NewMockByronToShelleyTransition creates a Byron to Shelley era transition.
// Both byronBlocks and shelleyBlocks must be positive.
func NewMockByronToShelleyTransition(
	byronBlocks, shelleyBlocks int,
) (*EraTransitionScenario, error) {
	if byronBlocks <= 0 {
		return nil, errors.New("byronBlocks must be positive")
	}
	if shelleyBlocks <= 0 {
		return nil, errors.New("shelleyBlocks must be positive")
	}

	// Build Byron chain
	preBlocks := make([]ledger.Block, 0, byronBlocks)
	prevHash := GenesisHash

	for i := range byronBlocks {
		// #nosec G115 - i is bounded by byronBlocks
		slot := uint64(i) * 20 // Byron uses 20 second slots
		// #nosec G115 - i is bounded by byronBlocks
		blockNum := uint64(i)
		hash := hashFromSlotAndBlock(slot, blockNum)

		block, err := blocks.NewByronBlockBuilder().
			WithSlot(slot).
			WithBlockNumber(blockNum).
			WithHash(hash).
			WithPrevHash(prevHash).
			Build()
		if err != nil {
			return nil, err
		}

		preBlocks = append(preBlocks, block)
		prevHash = hash
	}

	// Calculate transition slot
	// #nosec G115 - byronBlocks is a positive int
	transitionSlot := uint64(byronBlocks) * 20

	// Build Shelley chain continuing from Byron
	postBlocks := make([]ledger.Block, 0, shelleyBlocks)
	for i := range shelleyBlocks {
		// #nosec G115 - i is bounded by shelleyBlocks
		slot := transitionSlot + uint64(i) // Shelley uses 1 second slots
		// #nosec G115 - both values fit in uint64
		blockNum := uint64(byronBlocks + i)
		hash := hashFromSlotAndBlock(slot, blockNum)

		block, err := blocks.NewShelleyBlockBuilder().
			WithSlot(slot).
			WithBlockNumber(blockNum).
			WithHash(hash).
			WithPrevHash(prevHash).
			Build()
		if err != nil {
			return nil, err
		}

		postBlocks = append(postBlocks, block)
		prevHash = hash
	}

	return &EraTransitionScenario{
		PreTransitionBlocks:  preBlocks,
		PostTransitionBlocks: postBlocks,
		TransitionSlot:       transitionSlot,
	}, nil
}

// NewMockBabbageToConwayTransition creates a Babbage to Conway era transition.
// Both babbageBlocks and conwayBlocks must be positive.
func NewMockBabbageToConwayTransition(
	babbageBlocks, conwayBlocks int,
) (*EraTransitionScenario, error) {
	if babbageBlocks <= 0 {
		return nil, errors.New("babbageBlocks must be positive")
	}
	if conwayBlocks <= 0 {
		return nil, errors.New("conwayBlocks must be positive")
	}

	// Build Babbage chain
	preBlocks := make([]ledger.Block, 0, babbageBlocks)
	prevHash := GenesisHash

	for i := range babbageBlocks {
		// #nosec G115 - i is bounded by babbageBlocks
		slot := uint64(i)
		// #nosec G115 - i is bounded by babbageBlocks
		blockNum := uint64(i)
		hash := hashFromSlotAndBlock(slot, blockNum)

		block, err := blocks.NewBabbageBlockBuilder().
			WithSlot(slot).
			WithBlockNumber(blockNum).
			WithHash(hash).
			WithPrevHash(prevHash).
			Build()
		if err != nil {
			return nil, err
		}

		preBlocks = append(preBlocks, block)
		prevHash = hash
	}

	// Calculate transition slot
	// #nosec G115 - babbageBlocks is a positive int
	transitionSlot := uint64(babbageBlocks)

	// Build Conway chain continuing from Babbage
	postBlocks := make([]ledger.Block, 0, conwayBlocks)
	for i := range conwayBlocks {
		// #nosec G115 - i is bounded by conwayBlocks
		slot := transitionSlot + uint64(i)
		// #nosec G115 - both values fit in uint64
		blockNum := uint64(babbageBlocks + i)
		hash := hashFromSlotAndBlock(slot, blockNum)

		block, err := blocks.NewConwayBlockBuilder().
			WithSlot(slot).
			WithBlockNumber(blockNum).
			WithHash(hash).
			WithPrevHash(prevHash).
			Build()
		if err != nil {
			return nil, err
		}

		postBlocks = append(postBlocks, block)
		prevHash = hash
	}

	return &EraTransitionScenario{
		PreTransitionBlocks:  preBlocks,
		PostTransitionBlocks: postBlocks,
		TransitionSlot:       transitionSlot,
	}, nil
}
