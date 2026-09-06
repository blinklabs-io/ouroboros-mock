// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package fixtures_test

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
)

func TestGenerateChainSupportsRegisteredEras(t *testing.T) {
	for _, era := range fixtures.SupportedEras() {
		era := era
		t.Run(era.Name, func(t *testing.T) {
			slotIncrement := uint64(1)
			if era.Id == 0 {
				slotIncrement = 21600
			}
			blocks, err := fixtures.GenerateChain(
				era, 1, common.Blake2b256{}, 0, slotIncrement, 2,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(blocks) != 2 {
				t.Fatalf("got %d blocks, want 2", len(blocks))
			}
			if blocks[1].PrevHash() != blocks[0].Hash() {
				t.Fatal("generated blocks are not linked")
			}
			decoded, err := ledger.NewBlockFromCbor(
				uint(blocks[0].Type()), blocks[0].Cbor(),
			)
			if err != nil {
				t.Fatalf("decode generated block: %v", err)
			}
			if decoded.Hash() != blocks[0].Hash() {
				t.Fatal("decoded block hash changed")
			}
		})
	}
}

func TestGenerateBlockUsesRequestedValues(t *testing.T) {
	era := fixtures.SupportedEras()[6] // Conway
	block, err := fixtures.GenerateBlock(era, 11, 22)
	if err != nil {
		t.Fatal(err)
	}
	if block.BlockNumber() != 11 || block.SlotNumber() != 22 {
		t.Fatalf("got block %d at slot %d", block.BlockNumber(), block.SlotNumber())
	}
}
