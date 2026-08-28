// Copyright 2026 Blink Labs Software
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

package fixtures_test

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
)

func TestGenerateAlonzoChainRoundTrip(t *testing.T) {
	assertGeneratedChain(
		t,
		fixtures.GenerateAlonzoChain,
		ledger.BlockTypeAlonzo,
		alonzo.MinProtocolVersionAlonzo,
	)
}

func TestGenerateBabbageChainRoundTrip(t *testing.T) {
	assertGeneratedChain(
		t,
		fixtures.GenerateBabbageChain,
		ledger.BlockTypeBabbage,
		babbage.MinProtocolVersionBabbage,
	)
}

func TestGenerateDijkstraChainRoundTrip(t *testing.T) {
	assertGeneratedChain(
		t,
		fixtures.GenerateDijkstraChain,
		ledger.BlockTypeDijkstra,
		dijkstra.MinProtocolVersionDijkstra,
	)
}

func TestGenerateShelleyChainRoundTrip(t *testing.T) {
	assertGeneratedChain(
		t,
		fixtures.GenerateShelleyChain,
		ledger.BlockTypeShelley,
		shelley.MinProtocolVersionShelley,
	)
}

func TestGeneratedBlocksDecodeAcrossAdjacentEras(t *testing.T) {
	for _, test := range []struct {
		name        string
		generate    chainGenerator
		decodeAsEra uint
	}{
		{
			name:        "Shelley as Mary",
			generate:    fixtures.GenerateShelleyChain,
			decodeAsEra: ledger.BlockTypeMary,
		},
		{
			name:        "Babbage as Conway",
			generate:    fixtures.GenerateBabbageChain,
			decodeAsEra: ledger.BlockTypeConway,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			blocks, err := test.generate(1, common.Blake2b256{}, 2, 1, 1)
			if err != nil {
				t.Fatalf("generate chain: %s", err)
			}
			if _, err := ledger.NewBlockFromCbor(
				test.decodeAsEra,
				blocks[0].Cbor(),
			); err != nil {
				t.Fatalf("cross-era decode: %s", err)
			}
		})
	}
}

func TestGeneratedBlockBodySizes(t *testing.T) {
	for _, test := range []struct {
		name     string
		generate chainGenerator
		wantSize uint64
	}{
		{
			name:     "Shelley",
			generate: fixtures.GenerateShelleyChain,
			wantSize: 3,
		},
		{
			name:     "Alonzo",
			generate: fixtures.GenerateAlonzoChain,
			wantSize: 5,
		},
		{
			name:     "Babbage",
			generate: fixtures.GenerateBabbageChain,
			wantSize: 4,
		},
		{
			name:     "Conway",
			generate: fixtures.GenerateConwayChain,
			wantSize: 4,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			blocks, err := test.generate(1, common.Blake2b256{}, 2, 1, 1)
			if err != nil {
				t.Fatalf("generate chain: %s", err)
			}
			if got := blocks[0].BlockBodySize(); got != test.wantSize {
				t.Fatalf("block body size = %d, want %d", got, test.wantSize)
			}
		})
	}
}

func TestGenerateBabbageChainWithUnknownProtocolVersion(t *testing.T) {
	blocks, err := fixtures.GenerateBabbageChainWithProtocolVersion(
		1,
		common.Blake2b256{},
		2,
		1,
		99,
		3,
		1,
	)
	if err != nil {
		t.Fatalf("GenerateBabbageChainWithProtocolVersion: %s", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	decoded, err := ledger.NewBlockFromCbor(
		ledger.BlockTypeBabbage,
		blocks[0].Cbor(),
	)
	if err != nil {
		t.Fatalf("decode Babbage block: %s", err)
	}
	if _, err := ledger.DetermineBlockType(decoded.Header().Cbor()); err == nil {
		t.Fatal("expected unknown protocol major to be unclassifiable")
	}
}

type chainGenerator func(
	uint64,
	common.Blake2b256,
	uint64,
	uint64,
	int,
) ([]ledger.Block, error)

func assertGeneratedChain(
	t *testing.T,
	generate chainGenerator,
	wantType uint,
	wantProtocolMajor uint64,
) {
	t.Helper()
	var origin common.Blake2b256
	blocks, err := generate(4, origin, 20, 3, 3)
	if err != nil {
		t.Fatalf("generate chain: %s", err)
	}
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	for i, block := range blocks {
		decoded, err := ledger.NewBlockFromCbor(wantType, block.Cbor())
		if err != nil {
			t.Fatalf("block %d decode failed: %s", i, err)
		}
		if decoded.Hash() != block.Hash() {
			t.Fatalf("block %d hash changed after round-trip", i)
		}
		if decoded.BlockNumber() != uint64(i+4) {
			t.Fatalf(
				"block %d number = %d, want %d",
				i,
				decoded.BlockNumber(),
				i+4,
			)
		}
		if decoded.SlotNumber() != uint64(20+i*3) {
			t.Fatalf(
				"block %d slot = %d, want %d",
				i,
				decoded.SlotNumber(),
				20+i*3,
			)
		}
		blockType, err := ledger.DetermineBlockType(decoded.Header().Cbor())
		if err != nil {
			t.Fatalf("block %d determine type: %s", i, err)
		}
		if blockType != wantType {
			t.Fatalf("block %d type = %d, want %d", i, blockType, wantType)
		}
		if protocolMajor(decoded) != wantProtocolMajor {
			t.Fatalf(
				"block %d protocol major = %d, want %d",
				i,
				protocolMajor(decoded),
				wantProtocolMajor,
			)
		}
		if i == 0 {
			if decoded.PrevHash() != origin {
				t.Fatalf("first block does not honor origin")
			}
		} else if decoded.PrevHash() != blocks[i-1].Hash() {
			t.Fatalf("block %d does not link to block %d", i, i-1)
		}
	}
	empty, err := generate(1, origin, 0, 1, 0)
	if err != nil {
		t.Fatalf("generate empty chain: %s", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("expected non-nil empty chain, got %#v", empty)
	}
}

func protocolMajor(block ledger.Block) uint64 {
	switch block := block.(type) {
	case *alonzo.AlonzoBlock:
		return block.BlockHeader.Body.ProtoMajorVersion
	case *babbage.BabbageBlock:
		return block.BlockHeader.Body.ProtoVersion.Major
	case *dijkstra.DijkstraBlock:
		return block.BlockHeader.Body.ProtoVersion.Major
	case *shelley.ShelleyBlock:
		return block.BlockHeader.Body.ProtoMajorVersion
	default:
		return 0
	}
}
