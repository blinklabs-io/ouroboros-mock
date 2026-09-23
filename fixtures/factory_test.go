// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package fixtures_test

import (
	"bytes"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
)

func TestGenerateChainSupportsRegisteredEras(t *testing.T) {
	for _, era := range fixtures.SupportedEras() {
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
	era := ledger.GetEraById(6) // Conway
	block, err := fixtures.GenerateBlock(era, 11, 22)
	if err != nil {
		t.Fatal(err)
	}
	if block.BlockNumber() != 11 || block.SlotNumber() != 22 {
		t.Fatalf(
			"got block %d at slot %d",
			block.BlockNumber(),
			block.SlotNumber(),
		)
	}
}

func TestGenerateBlockByronRequiresEpochAlignedSlot(t *testing.T) {
	era := ledger.GetEraById(0)
	if _, err := fixtures.GenerateBlock(era, 1, 1); err == nil {
		t.Fatal("expected non-aligned Byron slot to fail")
	}
	if _, err := fixtures.GenerateBlock(era, 1, 0); err != nil {
		t.Fatalf("aligned Byron slot failed: %v", err)
	}
}

// TestGenerateBlockByronEBBUsesReferenceShape pins the wire shapes the
// reference boundary-block decoders require: an indefinite-length body and
// [attributes] for both the extra header and extra body data.
func TestGenerateBlockByronEBBUsesReferenceShape(t *testing.T) {
	block, err := fixtures.GenerateBlock(ledger.GetEraById(0), 1, 0)
	if err != nil {
		t.Fatalf("generate Byron EBB: %v", err)
	}
	var blockParts []cbor.RawMessage
	if _, err := cbor.Decode(block.Cbor(), &blockParts); err != nil {
		t.Fatalf("decode EBB: %v", err)
	}
	if len(blockParts) != 3 {
		t.Fatalf("EBB has %d fields, expected 3", len(blockParts))
	}
	var headerParts []cbor.RawMessage
	if _, err := cbor.Decode(blockParts[0], &headerParts); err != nil {
		t.Fatalf("decode EBB header: %v", err)
	}
	if len(headerParts) != 5 {
		t.Fatalf("EBB header has %d fields, expected 5", len(headerParts))
	}
	for name, got := range map[string][]byte{
		"body":              blockParts[1],
		"extra body data":   blockParts[2],
		"extra header data": headerParts[4],
	} {
		want := []byte{0x81, 0xa0}
		if name == "body" {
			want = []byte{0x9f, 0xff}
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s is %x, expected %x", name, got, want)
		}
	}
}
