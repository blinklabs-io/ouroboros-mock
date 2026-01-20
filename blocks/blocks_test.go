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

package blocks_test

import (
	"bytes"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/allegra"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/blinklabs-io/ouroboros-mock/blocks"
)

// TestNewConwayChainBuilder tests that NewConwayChainBuilder creates a builder with defaults
func TestNewConwayChainBuilder(t *testing.T) {
	builder := blocks.NewConwayChainBuilder()
	if builder == nil {
		t.Fatal("NewConwayChainBuilder returned nil")
	}

	// Build with no blocks should return empty slice
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d blocks", len(result))
	}
}

// TestConwayChainBuilderWithStartSlot tests WithStartSlot sets correct start slot
func TestConwayChainBuilderWithStartSlot(t *testing.T) {
	startSlot := uint64(100)
	builder := blocks.NewConwayChainBuilder().
		WithStartSlot(startSlot).
		AddBlocks(3)

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(result))
	}

	// First block should have the start slot
	if result[0].SlotNumber() != startSlot {
		t.Errorf(
			"expected first block slot %d, got %d",
			startSlot,
			result[0].SlotNumber(),
		)
	}
}

// TestConwayChainBuilderWithStartBlockNumber tests WithStartBlockNumber sets correct start block number
func TestConwayChainBuilderWithStartBlockNumber(t *testing.T) {
	startBlockNum := uint64(50)
	builder := blocks.NewConwayChainBuilder().
		WithStartBlockNumber(startBlockNum).
		AddBlocks(3)

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(result))
	}

	// First block should have the start block number
	if result[0].BlockNumber() != startBlockNum {
		t.Errorf(
			"expected first block number %d, got %d",
			startBlockNum,
			result[0].BlockNumber(),
		)
	}
}

// TestConwayChainBuilderWithSlotInterval tests WithSlotInterval sets correct slot intervals
func TestConwayChainBuilderWithSlotInterval(t *testing.T) {
	startSlot := uint64(0)
	slotInterval := uint64(5)
	builder := blocks.NewConwayChainBuilder().
		WithStartSlot(startSlot).
		WithSlotInterval(slotInterval).
		AddBlocks(4)

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expectedSlots := []uint64{0, 5, 10, 15}
	for i, block := range result {
		if block.SlotNumber() != expectedSlots[i] {
			t.Errorf(
				"block %d: expected slot %d, got %d",
				i,
				expectedSlots[i],
				block.SlotNumber(),
			)
		}
	}
}

// TestConwayChainBuilderAddBlocks tests AddBlocks adds correct number of blocks
func TestConwayChainBuilderAddBlocks(t *testing.T) {
	builder := blocks.NewConwayChainBuilder().AddBlocks(5)

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(result) != 5 {
		t.Errorf("expected 5 blocks, got %d", len(result))
	}
}

// TestConwayChainBuilderBuildValidProgression tests Build returns valid blocks with proper slot/block number progression
func TestConwayChainBuilderBuildValidProgression(t *testing.T) {
	startSlot := uint64(100)
	startBlockNum := uint64(10)
	slotInterval := uint64(2)
	count := 5

	builder := blocks.NewConwayChainBuilder().
		WithStartBlockNumber(startBlockNum).
		WithStartSlot(startSlot).
		WithSlotInterval(slotInterval).
		AddBlocks(count)

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(result) != count {
		t.Fatalf("expected %d blocks, got %d", count, len(result))
	}

	for i, block := range result {
		expectedSlot := startSlot + uint64(i)*slotInterval
		expectedBlockNum := startBlockNum + uint64(i)

		if block.SlotNumber() != expectedSlot {
			t.Errorf(
				"block %d: expected slot %d, got %d",
				i,
				expectedSlot,
				block.SlotNumber(),
			)
		}
		if block.BlockNumber() != expectedBlockNum {
			t.Errorf(
				"block %d: expected block number %d, got %d",
				i,
				expectedBlockNum,
				block.BlockNumber(),
			)
		}
	}
}

// TestConwayChainBuilderBlockLinking tests that blocks are properly linked via prevHash
func TestConwayChainBuilderBlockLinking(t *testing.T) {
	genesisHash := make([]byte, 32)
	for i := range genesisHash {
		genesisHash[i] = byte(i)
	}

	builder := blocks.NewConwayChainBuilder().
		WithGenesisHash(genesisHash).
		AddBlocks(3)

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// First block's prevHash should be the genesis hash
	if !bytes.Equal(result[0].PrevHash().Bytes(), genesisHash) {
		t.Errorf("first block prevHash doesn't match genesis hash")
	}

	// Each subsequent block's prevHash should match the previous block's hash
	for i := 1; i < len(result); i++ {
		prevBlockHash := result[i-1].Hash().Bytes()
		currentPrevHash := result[i].PrevHash().Bytes()
		if !bytes.Equal(currentPrevHash, prevBlockHash) {
			t.Errorf(
				"block %d: prevHash doesn't match previous block's hash",
				i,
			)
		}
	}
}

// TestConwayChainBuilderWithCustomHash tests WithCustomHash option
func TestConwayChainBuilderWithCustomHash(t *testing.T) {
	customHash := make([]byte, 32)
	for i := range customHash {
		customHash[i] = 0xAB
	}

	builder := blocks.NewConwayChainBuilder().
		AddBlock(blocks.WithCustomHash(customHash))

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 block, got %d", len(result))
	}

	if !bytes.Equal(result[0].Hash().Bytes(), customHash) {
		t.Errorf("block hash doesn't match custom hash")
	}
}

// TestConwayBlockBuilder tests NewConwayBlockBuilder
func TestConwayBlockBuilder(t *testing.T) {
	builder := blocks.NewConwayBlockBuilder()
	if builder == nil {
		t.Fatal("NewConwayBlockBuilder returned nil")
	}
}

// TestConwayBlockBuilderWithSlot tests WithSlot
func TestConwayBlockBuilderWithSlot(t *testing.T) {
	slot := uint64(12345)
	block, err := blocks.NewConwayBlockBuilder().
		WithSlot(slot).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.SlotNumber() != slot {
		t.Errorf("expected slot %d, got %d", slot, block.SlotNumber())
	}
}

// TestConwayBlockBuilderWithBlockNumber tests WithBlockNumber
func TestConwayBlockBuilderWithBlockNumber(t *testing.T) {
	blockNum := uint64(999)
	block, err := blocks.NewConwayBlockBuilder().
		WithBlockNumber(blockNum).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.BlockNumber() != blockNum {
		t.Errorf(
			"expected block number %d, got %d",
			blockNum,
			block.BlockNumber(),
		)
	}
}

// TestConwayBlockBuilderWithHash tests WithHash
func TestConwayBlockBuilderWithHash(t *testing.T) {
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i)
	}

	block, err := blocks.NewConwayBlockBuilder().
		WithHash(hash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.Hash().Bytes(), hash) {
		t.Errorf("hash doesn't match expected value")
	}
}

// TestConwayBlockBuilderWithPrevHash tests WithPrevHash
func TestConwayBlockBuilderWithPrevHash(t *testing.T) {
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = byte(255 - i)
	}

	block, err := blocks.NewConwayBlockBuilder().
		WithPrevHash(prevHash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.PrevHash().Bytes(), prevHash) {
		t.Errorf("prevHash doesn't match expected value")
	}
}

// TestConwayBlockBuilderBuildCbor tests BuildCbor returns nil (placeholder)
func TestConwayBlockBuilderBuildCbor(t *testing.T) {
	cbor, err := blocks.NewConwayBlockBuilder().BuildCbor()

	if err != nil {
		t.Fatalf("BuildCbor failed: %v", err)
	}

	// Currently returns nil as a placeholder
	if cbor != nil {
		t.Errorf("expected nil CBOR, got %v", cbor)
	}
}

// TestConwayBlockEra tests that Conway block returns correct era
func TestConwayBlockEra(t *testing.T) {
	block, err := blocks.NewConwayBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Era().Id != conway.EraIdConway {
		t.Errorf(
			"expected Conway era ID %d, got %d",
			conway.EraIdConway,
			block.Era().Id,
		)
	}
}

// TestConwayBlockType tests that Conway block returns correct type
func TestConwayBlockType(t *testing.T) {
	block, err := blocks.NewConwayBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Type() != conway.BlockTypeConway {
		t.Errorf(
			"expected Conway block type %d, got %d",
			conway.BlockTypeConway,
			block.Type(),
		)
	}
}

// TestBabbageBlockBuilder tests NewBabbageBlockBuilder
func TestBabbageBlockBuilder(t *testing.T) {
	builder := blocks.NewBabbageBlockBuilder()
	if builder == nil {
		t.Fatal("NewBabbageBlockBuilder returned nil")
	}
}

// TestBabbageBlockBuilderWithSlot tests WithSlot
func TestBabbageBlockBuilderWithSlot(t *testing.T) {
	slot := uint64(54321)
	block, err := blocks.NewBabbageBlockBuilder().
		WithSlot(slot).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.SlotNumber() != slot {
		t.Errorf("expected slot %d, got %d", slot, block.SlotNumber())
	}
}

// TestBabbageBlockBuilderWithBlockNumber tests WithBlockNumber
func TestBabbageBlockBuilderWithBlockNumber(t *testing.T) {
	blockNum := uint64(888)
	block, err := blocks.NewBabbageBlockBuilder().
		WithBlockNumber(blockNum).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.BlockNumber() != blockNum {
		t.Errorf(
			"expected block number %d, got %d",
			blockNum,
			block.BlockNumber(),
		)
	}
}

// TestBabbageBlockBuilderWithHash tests WithHash
func TestBabbageBlockBuilderWithHash(t *testing.T) {
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i * 2)
	}

	block, err := blocks.NewBabbageBlockBuilder().
		WithHash(hash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.Hash().Bytes(), hash) {
		t.Errorf("hash doesn't match expected value")
	}
}

// TestBabbageBlockBuilderWithPrevHash tests WithPrevHash
func TestBabbageBlockBuilderWithPrevHash(t *testing.T) {
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = byte(i * 3)
	}

	block, err := blocks.NewBabbageBlockBuilder().
		WithPrevHash(prevHash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.PrevHash().Bytes(), prevHash) {
		t.Errorf("prevHash doesn't match expected value")
	}
}

// TestBabbageBlockBuilderBuildCbor tests BuildCbor returns nil (placeholder)
func TestBabbageBlockBuilderBuildCbor(t *testing.T) {
	cbor, err := blocks.NewBabbageBlockBuilder().BuildCbor()

	if err != nil {
		t.Fatalf("BuildCbor failed: %v", err)
	}

	if cbor != nil {
		t.Errorf("expected nil CBOR, got %v", cbor)
	}
}

// TestBabbageBlockEra tests that Babbage block returns correct era
func TestBabbageBlockEra(t *testing.T) {
	block, err := blocks.NewBabbageBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Era().Id != babbage.EraIdBabbage {
		t.Errorf(
			"expected Babbage era ID %d, got %d",
			babbage.EraIdBabbage,
			block.Era().Id,
		)
	}
}

// TestBabbageBlockType tests that Babbage block returns correct type
func TestBabbageBlockType(t *testing.T) {
	block, err := blocks.NewBabbageBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Type() != babbage.BlockTypeBabbage {
		t.Errorf(
			"expected Babbage block type %d, got %d",
			babbage.BlockTypeBabbage,
			block.Type(),
		)
	}
}

// TestAlonzoBlockBuilder tests NewAlonzoBlockBuilder
func TestAlonzoBlockBuilder(t *testing.T) {
	builder := blocks.NewAlonzoBlockBuilder()
	if builder == nil {
		t.Fatal("NewAlonzoBlockBuilder returned nil")
	}
}

// TestAlonzoBlockBuilderWithSlot tests WithSlot
func TestAlonzoBlockBuilderWithSlot(t *testing.T) {
	slot := uint64(11111)
	block, err := blocks.NewAlonzoBlockBuilder().
		WithSlot(slot).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.SlotNumber() != slot {
		t.Errorf("expected slot %d, got %d", slot, block.SlotNumber())
	}
}

// TestAlonzoBlockBuilderWithBlockNumber tests WithBlockNumber
func TestAlonzoBlockBuilderWithBlockNumber(t *testing.T) {
	blockNum := uint64(777)
	block, err := blocks.NewAlonzoBlockBuilder().
		WithBlockNumber(blockNum).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.BlockNumber() != blockNum {
		t.Errorf(
			"expected block number %d, got %d",
			blockNum,
			block.BlockNumber(),
		)
	}
}

// TestAlonzoBlockBuilderWithHash tests WithHash
func TestAlonzoBlockBuilderWithHash(t *testing.T) {
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i + 10)
	}

	block, err := blocks.NewAlonzoBlockBuilder().
		WithHash(hash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.Hash().Bytes(), hash) {
		t.Errorf("hash doesn't match expected value")
	}
}

// TestAlonzoBlockBuilderWithPrevHash tests WithPrevHash
func TestAlonzoBlockBuilderWithPrevHash(t *testing.T) {
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = byte(i + 20)
	}

	block, err := blocks.NewAlonzoBlockBuilder().
		WithPrevHash(prevHash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.PrevHash().Bytes(), prevHash) {
		t.Errorf("prevHash doesn't match expected value")
	}
}

// TestAlonzoBlockBuilderBuildCbor tests BuildCbor returns nil (placeholder)
func TestAlonzoBlockBuilderBuildCbor(t *testing.T) {
	cbor, err := blocks.NewAlonzoBlockBuilder().BuildCbor()

	if err != nil {
		t.Fatalf("BuildCbor failed: %v", err)
	}

	if cbor != nil {
		t.Errorf("expected nil CBOR, got %v", cbor)
	}
}

// TestAlonzoBlockEra tests that Alonzo block returns correct era
func TestAlonzoBlockEra(t *testing.T) {
	block, err := blocks.NewAlonzoBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Era().Id != alonzo.EraIdAlonzo {
		t.Errorf(
			"expected Alonzo era ID %d, got %d",
			alonzo.EraIdAlonzo,
			block.Era().Id,
		)
	}
}

// TestAlonzoBlockType tests that Alonzo block returns correct type
func TestAlonzoBlockType(t *testing.T) {
	block, err := blocks.NewAlonzoBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Type() != alonzo.BlockTypeAlonzo {
		t.Errorf(
			"expected Alonzo block type %d, got %d",
			alonzo.BlockTypeAlonzo,
			block.Type(),
		)
	}
}

// TestMaryBlockBuilder tests NewMaryBlockBuilder
func TestMaryBlockBuilder(t *testing.T) {
	builder := blocks.NewMaryBlockBuilder()
	if builder == nil {
		t.Fatal("NewMaryBlockBuilder returned nil")
	}
}

// TestMaryBlockBuilderWithSlot tests WithSlot
func TestMaryBlockBuilderWithSlot(t *testing.T) {
	slot := uint64(22222)
	block, err := blocks.NewMaryBlockBuilder().
		WithSlot(slot).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.SlotNumber() != slot {
		t.Errorf("expected slot %d, got %d", slot, block.SlotNumber())
	}
}

// TestMaryBlockBuilderWithBlockNumber tests WithBlockNumber
func TestMaryBlockBuilderWithBlockNumber(t *testing.T) {
	blockNum := uint64(666)
	block, err := blocks.NewMaryBlockBuilder().
		WithBlockNumber(blockNum).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.BlockNumber() != blockNum {
		t.Errorf(
			"expected block number %d, got %d",
			blockNum,
			block.BlockNumber(),
		)
	}
}

// TestMaryBlockBuilderWithHash tests WithHash
func TestMaryBlockBuilderWithHash(t *testing.T) {
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i + 30)
	}

	block, err := blocks.NewMaryBlockBuilder().
		WithHash(hash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.Hash().Bytes(), hash) {
		t.Errorf("hash doesn't match expected value")
	}
}

// TestMaryBlockBuilderWithPrevHash tests WithPrevHash
func TestMaryBlockBuilderWithPrevHash(t *testing.T) {
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = byte(i + 40)
	}

	block, err := blocks.NewMaryBlockBuilder().
		WithPrevHash(prevHash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.PrevHash().Bytes(), prevHash) {
		t.Errorf("prevHash doesn't match expected value")
	}
}

// TestMaryBlockBuilderBuildCbor tests BuildCbor returns nil (placeholder)
func TestMaryBlockBuilderBuildCbor(t *testing.T) {
	cbor, err := blocks.NewMaryBlockBuilder().BuildCbor()

	if err != nil {
		t.Fatalf("BuildCbor failed: %v", err)
	}

	if cbor != nil {
		t.Errorf("expected nil CBOR, got %v", cbor)
	}
}

// TestMaryBlockEra tests that Mary block returns correct era
func TestMaryBlockEra(t *testing.T) {
	block, err := blocks.NewMaryBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Era().Id != mary.EraIdMary {
		t.Errorf(
			"expected Mary era ID %d, got %d",
			mary.EraIdMary,
			block.Era().Id,
		)
	}
}

// TestMaryBlockType tests that Mary block returns correct type
func TestMaryBlockType(t *testing.T) {
	block, err := blocks.NewMaryBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Type() != mary.BlockTypeMary {
		t.Errorf(
			"expected Mary block type %d, got %d",
			mary.BlockTypeMary,
			block.Type(),
		)
	}
}

// TestAllegraBlockBuilder tests NewAllegraBlockBuilder
func TestAllegraBlockBuilder(t *testing.T) {
	builder := blocks.NewAllegraBlockBuilder()
	if builder == nil {
		t.Fatal("NewAllegraBlockBuilder returned nil")
	}
}

// TestAllegraBlockBuilderWithSlot tests WithSlot
func TestAllegraBlockBuilderWithSlot(t *testing.T) {
	slot := uint64(33333)
	block, err := blocks.NewAllegraBlockBuilder().
		WithSlot(slot).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.SlotNumber() != slot {
		t.Errorf("expected slot %d, got %d", slot, block.SlotNumber())
	}
}

// TestAllegraBlockBuilderWithBlockNumber tests WithBlockNumber
func TestAllegraBlockBuilderWithBlockNumber(t *testing.T) {
	blockNum := uint64(555)
	block, err := blocks.NewAllegraBlockBuilder().
		WithBlockNumber(blockNum).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.BlockNumber() != blockNum {
		t.Errorf(
			"expected block number %d, got %d",
			blockNum,
			block.BlockNumber(),
		)
	}
}

// TestAllegraBlockBuilderWithHash tests WithHash
func TestAllegraBlockBuilderWithHash(t *testing.T) {
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i + 50)
	}

	block, err := blocks.NewAllegraBlockBuilder().
		WithHash(hash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.Hash().Bytes(), hash) {
		t.Errorf("hash doesn't match expected value")
	}
}

// TestAllegraBlockBuilderWithPrevHash tests WithPrevHash
func TestAllegraBlockBuilderWithPrevHash(t *testing.T) {
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = byte(i + 60)
	}

	block, err := blocks.NewAllegraBlockBuilder().
		WithPrevHash(prevHash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.PrevHash().Bytes(), prevHash) {
		t.Errorf("prevHash doesn't match expected value")
	}
}

// TestAllegraBlockBuilderBuildCbor tests BuildCbor returns nil (placeholder)
func TestAllegraBlockBuilderBuildCbor(t *testing.T) {
	cbor, err := blocks.NewAllegraBlockBuilder().BuildCbor()

	if err != nil {
		t.Fatalf("BuildCbor failed: %v", err)
	}

	if cbor != nil {
		t.Errorf("expected nil CBOR, got %v", cbor)
	}
}

// TestAllegraBlockEra tests that Allegra block returns correct era
func TestAllegraBlockEra(t *testing.T) {
	block, err := blocks.NewAllegraBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Era().Id != allegra.EraIdAllegra {
		t.Errorf(
			"expected Allegra era ID %d, got %d",
			allegra.EraIdAllegra,
			block.Era().Id,
		)
	}
}

// TestAllegraBlockType tests that Allegra block returns correct type
func TestAllegraBlockType(t *testing.T) {
	block, err := blocks.NewAllegraBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Type() != allegra.BlockTypeAllegra {
		t.Errorf(
			"expected Allegra block type %d, got %d",
			allegra.BlockTypeAllegra,
			block.Type(),
		)
	}
}

// TestShelleyBlockBuilder tests NewShelleyBlockBuilder
func TestShelleyBlockBuilder(t *testing.T) {
	builder := blocks.NewShelleyBlockBuilder()
	if builder == nil {
		t.Fatal("NewShelleyBlockBuilder returned nil")
	}
}

// TestShelleyBlockBuilderWithSlot tests WithSlot
func TestShelleyBlockBuilderWithSlot(t *testing.T) {
	slot := uint64(44444)
	block, err := blocks.NewShelleyBlockBuilder().
		WithSlot(slot).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.SlotNumber() != slot {
		t.Errorf("expected slot %d, got %d", slot, block.SlotNumber())
	}
}

// TestShelleyBlockBuilderWithBlockNumber tests WithBlockNumber
func TestShelleyBlockBuilderWithBlockNumber(t *testing.T) {
	blockNum := uint64(444)
	block, err := blocks.NewShelleyBlockBuilder().
		WithBlockNumber(blockNum).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.BlockNumber() != blockNum {
		t.Errorf(
			"expected block number %d, got %d",
			blockNum,
			block.BlockNumber(),
		)
	}
}

// TestShelleyBlockBuilderWithHash tests WithHash
func TestShelleyBlockBuilderWithHash(t *testing.T) {
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i + 70)
	}

	block, err := blocks.NewShelleyBlockBuilder().
		WithHash(hash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.Hash().Bytes(), hash) {
		t.Errorf("hash doesn't match expected value")
	}
}

// TestShelleyBlockBuilderWithPrevHash tests WithPrevHash
func TestShelleyBlockBuilderWithPrevHash(t *testing.T) {
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = byte(i + 80)
	}

	block, err := blocks.NewShelleyBlockBuilder().
		WithPrevHash(prevHash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.PrevHash().Bytes(), prevHash) {
		t.Errorf("prevHash doesn't match expected value")
	}
}

// TestShelleyBlockBuilderBuildCbor tests BuildCbor returns nil (placeholder)
func TestShelleyBlockBuilderBuildCbor(t *testing.T) {
	cbor, err := blocks.NewShelleyBlockBuilder().BuildCbor()

	if err != nil {
		t.Fatalf("BuildCbor failed: %v", err)
	}

	if cbor != nil {
		t.Errorf("expected nil CBOR, got %v", cbor)
	}
}

// TestShelleyBlockEra tests that Shelley block returns correct era
func TestShelleyBlockEra(t *testing.T) {
	block, err := blocks.NewShelleyBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Era().Id != shelley.EraIdShelley {
		t.Errorf(
			"expected Shelley era ID %d, got %d",
			shelley.EraIdShelley,
			block.Era().Id,
		)
	}
}

// TestShelleyBlockType tests that Shelley block returns correct type
func TestShelleyBlockType(t *testing.T) {
	block, err := blocks.NewShelleyBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Type() != shelley.BlockTypeShelley {
		t.Errorf(
			"expected Shelley block type %d, got %d",
			shelley.BlockTypeShelley,
			block.Type(),
		)
	}
}

// TestByronBlockBuilder tests NewByronBlockBuilder
func TestByronBlockBuilder(t *testing.T) {
	builder := blocks.NewByronBlockBuilder()
	if builder == nil {
		t.Fatal("NewByronBlockBuilder returned nil")
	}
}

// TestByronEBBBlockBuilder tests NewByronEBBBlockBuilder
func TestByronEBBBlockBuilder(t *testing.T) {
	builder := blocks.NewByronEBBBlockBuilder()
	if builder == nil {
		t.Fatal("NewByronEBBBlockBuilder returned nil")
	}
}

// TestByronBlockBuilderWithSlot tests WithSlot
func TestByronBlockBuilderWithSlot(t *testing.T) {
	slot := uint64(55555)
	block, err := blocks.NewByronBlockBuilder().
		WithSlot(slot).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.SlotNumber() != slot {
		t.Errorf("expected slot %d, got %d", slot, block.SlotNumber())
	}
}

// TestByronBlockBuilderWithBlockNumber tests WithBlockNumber
func TestByronBlockBuilderWithBlockNumber(t *testing.T) {
	blockNum := uint64(333)
	block, err := blocks.NewByronBlockBuilder().
		WithBlockNumber(blockNum).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.BlockNumber() != blockNum {
		t.Errorf(
			"expected block number %d, got %d",
			blockNum,
			block.BlockNumber(),
		)
	}
}

// TestByronBlockBuilderWithHash tests WithHash
func TestByronBlockBuilderWithHash(t *testing.T) {
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i + 90)
	}

	block, err := blocks.NewByronBlockBuilder().
		WithHash(hash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.Hash().Bytes(), hash) {
		t.Errorf("hash doesn't match expected value")
	}
}

// TestByronBlockBuilderWithPrevHash tests WithPrevHash
func TestByronBlockBuilderWithPrevHash(t *testing.T) {
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = byte(i + 100)
	}

	block, err := blocks.NewByronBlockBuilder().
		WithPrevHash(prevHash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !bytes.Equal(block.PrevHash().Bytes(), prevHash) {
		t.Errorf("prevHash doesn't match expected value")
	}
}

// TestByronBlockBuilderBuildCbor tests BuildCbor returns nil (placeholder)
func TestByronBlockBuilderBuildCbor(t *testing.T) {
	cbor, err := blocks.NewByronBlockBuilder().BuildCbor()

	if err != nil {
		t.Fatalf("BuildCbor failed: %v", err)
	}

	if cbor != nil {
		t.Errorf("expected nil CBOR, got %v", cbor)
	}
}

// TestByronBlockEra tests that Byron block returns correct era
func TestByronBlockEra(t *testing.T) {
	block, err := blocks.NewByronBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Era().Id != byron.EraIdByron {
		t.Errorf(
			"expected Byron era ID %d, got %d",
			byron.EraIdByron,
			block.Era().Id,
		)
	}
}

// TestByronBlockType tests that Byron main block returns correct type
func TestByronBlockType(t *testing.T) {
	block, err := blocks.NewByronBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Type() != byron.BlockTypeByronMain {
		t.Errorf(
			"expected Byron main block type %d, got %d",
			byron.BlockTypeByronMain,
			block.Type(),
		)
	}
}

// TestByronEBBBlockType tests that Byron EBB block returns correct type
func TestByronEBBBlockType(t *testing.T) {
	block, err := blocks.NewByronEBBBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Type() != byron.BlockTypeByronEbb {
		t.Errorf(
			"expected Byron EBB block type %d, got %d",
			byron.BlockTypeByronEbb,
			block.Type(),
		)
	}
}

// TestByronBlockBuilderWithEBB tests WithEBB method
func TestByronBlockBuilderWithEBB(t *testing.T) {
	// Create a regular block and convert to EBB
	block, err := blocks.NewByronBlockBuilder().
		WithEBB(true).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.Type() != byron.BlockTypeByronEbb {
		t.Errorf(
			"expected Byron EBB block type %d, got %d",
			byron.BlockTypeByronEbb,
			block.Type(),
		)
	}
}

// TestBlockHeader tests that block headers are correctly returned
func TestBlockHeader(t *testing.T) {
	slot := uint64(1000)
	blockNum := uint64(100)
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i)
	}
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = byte(255 - i)
	}

	block, err := blocks.NewConwayBlockBuilder().
		WithSlot(slot).
		WithBlockNumber(blockNum).
		WithHash(hash).
		WithPrevHash(prevHash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	header := block.Header()
	if header == nil {
		t.Fatal("Header returned nil")
	}

	if header.SlotNumber() != slot {
		t.Errorf("header slot: expected %d, got %d", slot, header.SlotNumber())
	}
	if header.BlockNumber() != blockNum {
		t.Errorf(
			"header block number: expected %d, got %d",
			blockNum,
			header.BlockNumber(),
		)
	}
	if !bytes.Equal(header.Hash().Bytes(), hash) {
		t.Errorf("header hash doesn't match")
	}
	if !bytes.Equal(header.PrevHash().Bytes(), prevHash) {
		t.Errorf("header prevHash doesn't match")
	}
}

// TestConwayHeaderBuilder tests NewConwayHeaderBuilder
func TestConwayHeaderBuilder(t *testing.T) {
	builder := blocks.NewConwayHeaderBuilder()
	if builder == nil {
		t.Fatal("NewConwayHeaderBuilder returned nil")
	}
}

// TestConwayHeaderBuilderBuild tests building a Conway header
func TestConwayHeaderBuilderBuild(t *testing.T) {
	slot := uint64(5000)
	blockNum := uint64(500)
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = 0xAA
	}
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = 0xBB
	}

	header, err := blocks.NewConwayHeaderBuilder().
		WithSlot(slot).
		WithBlockNumber(blockNum).
		WithHash(hash).
		WithPrevHash(prevHash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if header.SlotNumber() != slot {
		t.Errorf("expected slot %d, got %d", slot, header.SlotNumber())
	}
	if header.BlockNumber() != blockNum {
		t.Errorf(
			"expected block number %d, got %d",
			blockNum,
			header.BlockNumber(),
		)
	}
	if !bytes.Equal(header.Hash().Bytes(), hash) {
		t.Errorf("hash doesn't match")
	}
	if !bytes.Equal(header.PrevHash().Bytes(), prevHash) {
		t.Errorf("prevHash doesn't match")
	}
}

// TestConwayHeaderBuilderBuildCbor tests BuildCbor returns nil (placeholder)
func TestConwayHeaderBuilderBuildCbor(t *testing.T) {
	cbor, err := blocks.NewConwayHeaderBuilder().BuildCbor()

	if err != nil {
		t.Fatalf("BuildCbor failed: %v", err)
	}

	if cbor != nil {
		t.Errorf("expected nil CBOR, got %v", cbor)
	}
}

// TestBabbageHeaderBuilder tests NewBabbageHeaderBuilder
func TestBabbageHeaderBuilder(t *testing.T) {
	builder := blocks.NewBabbageHeaderBuilder()
	if builder == nil {
		t.Fatal("NewBabbageHeaderBuilder returned nil")
	}

	header, err := builder.
		WithSlot(1000).
		WithBlockNumber(100).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if header.SlotNumber() != 1000 {
		t.Errorf("expected slot 1000, got %d", header.SlotNumber())
	}
	if header.BlockNumber() != 100 {
		t.Errorf("expected block number 100, got %d", header.BlockNumber())
	}
}

// TestAlonzoHeaderBuilder tests NewAlonzoHeaderBuilder
func TestAlonzoHeaderBuilder(t *testing.T) {
	builder := blocks.NewAlonzoHeaderBuilder()
	if builder == nil {
		t.Fatal("NewAlonzoHeaderBuilder returned nil")
	}

	header, err := builder.
		WithSlot(2000).
		WithBlockNumber(200).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if header.SlotNumber() != 2000 {
		t.Errorf("expected slot 2000, got %d", header.SlotNumber())
	}
	if header.BlockNumber() != 200 {
		t.Errorf("expected block number 200, got %d", header.BlockNumber())
	}
}

// TestMaryHeaderBuilder tests NewMaryHeaderBuilder
func TestMaryHeaderBuilder(t *testing.T) {
	builder := blocks.NewMaryHeaderBuilder()
	if builder == nil {
		t.Fatal("NewMaryHeaderBuilder returned nil")
	}

	header, err := builder.
		WithSlot(3000).
		WithBlockNumber(300).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if header.SlotNumber() != 3000 {
		t.Errorf("expected slot 3000, got %d", header.SlotNumber())
	}
	if header.BlockNumber() != 300 {
		t.Errorf("expected block number 300, got %d", header.BlockNumber())
	}
}

// TestAllegraHeaderBuilder tests NewAllegraHeaderBuilder
func TestAllegraHeaderBuilder(t *testing.T) {
	builder := blocks.NewAllegraHeaderBuilder()
	if builder == nil {
		t.Fatal("NewAllegraHeaderBuilder returned nil")
	}

	header, err := builder.
		WithSlot(4000).
		WithBlockNumber(400).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if header.SlotNumber() != 4000 {
		t.Errorf("expected slot 4000, got %d", header.SlotNumber())
	}
	if header.BlockNumber() != 400 {
		t.Errorf("expected block number 400, got %d", header.BlockNumber())
	}
}

// TestShelleyHeaderBuilder tests NewShelleyHeaderBuilder
func TestShelleyHeaderBuilder(t *testing.T) {
	builder := blocks.NewShelleyHeaderBuilder()
	if builder == nil {
		t.Fatal("NewShelleyHeaderBuilder returned nil")
	}

	header, err := builder.
		WithSlot(5000).
		WithBlockNumber(500).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if header.SlotNumber() != 5000 {
		t.Errorf("expected slot 5000, got %d", header.SlotNumber())
	}
	if header.BlockNumber() != 500 {
		t.Errorf("expected block number 500, got %d", header.BlockNumber())
	}
}

// TestByronHeaderBuilder tests NewByronHeaderBuilder
func TestByronHeaderBuilder(t *testing.T) {
	builder := blocks.NewByronHeaderBuilder()
	if builder == nil {
		t.Fatal("NewByronHeaderBuilder returned nil")
	}

	header, err := builder.
		WithSlot(6000).
		WithBlockNumber(600).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if header.SlotNumber() != 6000 {
		t.Errorf("expected slot 6000, got %d", header.SlotNumber())
	}
	if header.BlockNumber() != 600 {
		t.Errorf("expected block number 600, got %d", header.BlockNumber())
	}
}

// TestByronHeaderBuilderWithEBB tests WithEBB on header builder
func TestByronHeaderBuilderWithEBB(t *testing.T) {
	builder := blocks.NewByronHeaderBuilder().WithEBB(true)
	if builder == nil {
		t.Fatal("NewByronHeaderBuilder returned nil")
	}

	header, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Header should be created successfully
	if header == nil {
		t.Fatal("Header is nil")
	}
}

// TestBlockTransactionsEmpty tests that blocks with no transactions return nil or empty slice
func TestBlockTransactionsEmpty(t *testing.T) {
	block, err := blocks.NewConwayBlockBuilder().Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	txs := block.Transactions()
	if len(txs) != 0 {
		t.Errorf("expected 0 transactions, got %d", len(txs))
	}
}

// TestChainBuilderInterface tests that ConwayChainBuilder implements ChainBuilder interface
func TestChainBuilderInterface(t *testing.T) {
	var _ blocks.ChainBuilder = blocks.NewConwayChainBuilder()
}

// TestBlockBuilderInterface tests that era-specific builders implement BlockBuilder interface
func TestBlockBuilderInterface(t *testing.T) {
	var _ blocks.BlockBuilder = blocks.NewConwayBlockBuilder()
	var _ blocks.BlockBuilder = blocks.NewBabbageBlockBuilder()
	var _ blocks.BlockBuilder = blocks.NewAlonzoBlockBuilder()
	var _ blocks.BlockBuilder = blocks.NewMaryBlockBuilder()
	var _ blocks.BlockBuilder = blocks.NewAllegraBlockBuilder()
	var _ blocks.BlockBuilder = blocks.NewShelleyBlockBuilder()
	var _ blocks.BlockBuilder = blocks.NewByronBlockBuilder()
}

// TestHeaderBuilderInterface tests that era-specific header builders implement HeaderBuilder interface
func TestHeaderBuilderInterface(t *testing.T) {
	var _ blocks.HeaderBuilder = blocks.NewConwayHeaderBuilder()
	var _ blocks.HeaderBuilder = blocks.NewBabbageHeaderBuilder()
	var _ blocks.HeaderBuilder = blocks.NewAlonzoHeaderBuilder()
	var _ blocks.HeaderBuilder = blocks.NewMaryHeaderBuilder()
	var _ blocks.HeaderBuilder = blocks.NewAllegraHeaderBuilder()
	var _ blocks.HeaderBuilder = blocks.NewShelleyHeaderBuilder()
	var _ blocks.HeaderBuilder = blocks.NewByronHeaderBuilder()
}

// TestChainBuilderMethodChaining tests that method chaining works correctly
func TestChainBuilderMethodChaining(t *testing.T) {
	result, err := blocks.NewConwayChainBuilder().
		WithStartSlot(100).
		WithSlotInterval(5).
		WithEra(6).
		AddBlocks(2).
		AddBlock().
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(result))
	}

	// Check slot progression
	expectedSlots := []uint64{100, 105, 110}
	for i, block := range result {
		if block.SlotNumber() != expectedSlots[i] {
			t.Errorf(
				"block %d: expected slot %d, got %d",
				i,
				expectedSlots[i],
				block.SlotNumber(),
			)
		}
	}
}

// TestBlockBuilderMethodChaining tests that block builder method chaining works correctly
func TestBlockBuilderMethodChaining(t *testing.T) {
	hash := make([]byte, 32)
	prevHash := make([]byte, 32)

	block, err := blocks.NewConwayBlockBuilder().
		WithSlot(1000).
		WithBlockNumber(100).
		WithHash(hash).
		WithPrevHash(prevHash).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if block.SlotNumber() != 1000 {
		t.Errorf("expected slot 1000, got %d", block.SlotNumber())
	}
	if block.BlockNumber() != 100 {
		t.Errorf("expected block number 100, got %d", block.BlockNumber())
	}
}
