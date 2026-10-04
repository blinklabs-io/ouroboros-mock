// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package fixtures

import (
	"fmt"
	"slices"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
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

// GenerateChain builds count connected empty blocks of era, linked from
// prevHash. Every returned block has been decoded by the corresponding
// gouroboros decoder, so its CBOR and hash are the values a downstream
// consumer sees. Byron blocks are epoch boundary blocks, so the start slot
// and increment must be epoch-aligned.
func GenerateChain(
	era common.Era,
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	if !slices.ContainsFunc(SupportedEras(), func(e common.Era) bool {
		return e.Id == era.Id
	}) {
		return nil, fmt.Errorf(
			"unsupported fixture era %d (%s)",
			era.Id,
			era.Name,
		)
	}
	if count <= 0 {
		return []ledger.Block{}, nil
	}
	if era.Id == byron.EraIdByron &&
		(startSlot%byron.ByronSlotsPerEpoch != 0 ||
			slotIncrement%byron.ByronSlotsPerEpoch != 0) {
		return nil, fmt.Errorf(
			"byron fixture slots must be epoch-aligned: start=%d increment=%d",
			startSlot,
			slotIncrement,
		)
	}
	if era.Id == dijkstra.EraIdDijkstra {
		return GenerateDijkstraChain(
			startBlockNumber, prevHash, startSlot, slotIncrement, count,
		)
	}
	return NewSequence(era).
		WithStart(startBlockNumber, startSlot, prevHash).
		WithSlotIncrement(slotIncrement).
		Blocks(count)
}

// GenerateBlock returns one empty block for era. Byron blocks require the
// requested slot to be aligned to a Byron epoch boundary. It is a convenience
// wrapper around GenerateChain for tests that need a single block.
func GenerateBlock(
	era common.Era,
	blockNumber, slot uint64,
) (ledger.Block, error) {
	blocks, err := GenerateChain(
		era, blockNumber, common.Blake2b256{}, slot, 0, 1,
	)
	if err != nil {
		return nil, err
	}
	return blocks[0], nil
}
