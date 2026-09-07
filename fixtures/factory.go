// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package fixtures

import (
	"bytes"
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// SupportedEras returns the era IDs with block fixtures provided by this
// package. The list follows gouroboros' ledger era registration order.
func SupportedEras() []common.Era {
	return []common.Era{
		ledger.GetEraById(byron.EraIdByron),
		ledger.GetEraById(1),
		ledger.GetEraById(2),
		ledger.GetEraById(3),
		ledger.GetEraById(4),
		ledger.GetEraById(5),
		ledger.GetEraById(6),
		ledger.GetEraById(7),
	}
}

// GenerateChain dispatches to the era-specific empty-chain generators.
// Every returned block has been decoded by the corresponding gouroboros
// decoder, so its CBOR and hash are the values a downstream consumer sees.
func GenerateChain(
	era common.Era,
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	switch era.Id {
	case byron.EraIdByron:
		return generateByronChain(
			startBlockNumber, prevHash, startSlot, slotIncrement, count,
		)
	case 1:
		return GenerateShelleyChain(startBlockNumber, prevHash, startSlot, slotIncrement, count)
	case 2:
		return GenerateAllegraChain(startBlockNumber, prevHash, startSlot, slotIncrement, count)
	case 3:
		return GenerateMaryChain(startBlockNumber, prevHash, startSlot, slotIncrement, count)
	case 4:
		return GenerateAlonzoChain(startBlockNumber, prevHash, startSlot, slotIncrement, count)
	case 5:
		return GenerateBabbageChain(startBlockNumber, prevHash, startSlot, slotIncrement, count)
	case 6:
		return GenerateConwayChain(startBlockNumber, prevHash, startSlot, slotIncrement, count)
	case 7:
		return GenerateDijkstraChain(startBlockNumber, prevHash, startSlot, slotIncrement, count)
	default:
		return nil, fmt.Errorf("unsupported fixture era %d (%s)", era.Id, era.Name)
	}
}

// GenerateBlock returns one empty block for era. Byron blocks require the
// requested slot to be aligned to a Byron epoch boundary. It is a convenience
// wrapper around GenerateChain for tests that need a single block.
func GenerateBlock(era common.Era, blockNumber, slot uint64) (ledger.Block, error) {
	blocks, err := GenerateChain(
		era, blockNumber, common.Blake2b256{}, slot, 0, 1,
	)
	if err != nil {
		return nil, err
	}
	return blocks[0], nil
}

func generateByronChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	if count <= 0 {
		return []ledger.Block{}, nil
	}
	if startSlot%byron.ByronSlotsPerEpoch != 0 ||
		slotIncrement%byron.ByronSlotsPerEpoch != 0 {
		return nil, fmt.Errorf(
			"Byron fixture slots must be epoch-aligned: start=%d increment=%d",
			startSlot,
			slotIncrement,
		)
	}
	blocks := make([]ledger.Block, 0, count)
	currentPrev := prevHash
	for i := range count {
		body := []common.Blake2b224{}
		bodyCbor, err := cbor.Encode(body)
		if err != nil {
			return nil, fmt.Errorf("encode Byron EBB body %d: %w", i, err)
		}
		header := &byron.ByronEpochBoundaryBlockHeader{
			ProtocolMagic: byron.TestnetProtocolMagic,
			PrevBlock:     currentPrev,
			BodyProof:     common.Blake2b256Hash(bodyCbor).Bytes(),
		}
		header.ConsensusData.Epoch = (startSlot + uint64(i)*slotIncrement) / byron.ByronSlotsPerEpoch
		header.ConsensusData.Difficulty.Value = startBlockNumber + uint64(i)
		block := &byron.ByronEpochBoundaryBlock{
			BlockHeader: header,
			Body:        body,
			Extra:       []any{},
		}
		blockCbor, err := cbor.Encode(block)
		if err != nil {
			return nil, fmt.Errorf("encode Byron EBB %d: %w", i, err)
		}
		decoded, err := byron.NewByronEpochBoundaryBlockFromCbor(blockCbor)
		if err != nil {
			return nil, fmt.Errorf("decode Byron EBB %d: %w", i, err)
		}
		if !bytes.Equal(decoded.Cbor(), blockCbor) {
			return nil, fmt.Errorf("Byron EBB %d CBOR mismatch after round-trip", i)
		}
		blocks = append(blocks, decoded)
		currentPrev = decoded.Hash()
	}
	return blocks, nil
}
