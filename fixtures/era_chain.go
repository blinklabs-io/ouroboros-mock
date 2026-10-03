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

package fixtures

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
)

// GenerateConwayToDijkstraChain builds connected Conway and Dijkstra blocks
// for tests that exercise the PV12 era boundary. The Dijkstra blocks come from
// NewDijkstraBlockBuilder.
func GenerateConwayToDijkstraChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	conwayCount, dijkstraCount int,
) ([]ledger.Block, error) {
	if conwayCount < 0 || dijkstraCount < 0 {
		return nil, errors.New("block counts must not be negative")
	}
	if conwayCount+dijkstraCount == 0 {
		return []ledger.Block{}, nil
	}
	if err := checkChainRange(
		startBlockNumber,
		startSlot,
		slotIncrement,
		uint64(conwayCount)+uint64(dijkstraCount),
	); err != nil {
		return nil, err
	}
	blocks, err := GenerateConwayChain(
		startBlockNumber,
		prevHash,
		startSlot,
		slotIncrement,
		conwayCount,
	)
	if err != nil {
		return nil, fmt.Errorf("generate Conway transition blocks: %w", err)
	}
	currentPrev := prevHash
	if len(blocks) > 0 {
		currentPrev = blocks[len(blocks)-1].Hash()
	}
	for i := range dijkstraCount {
		offset := uint64(conwayCount) + uint64(i)
		block, err := NewDijkstraBlockBuilder().
			WithBlockNumber(startBlockNumber + offset).
			WithSlot(startSlot + offset*slotIncrement).
			WithPreviousHash(currentPrev).
			Build()
		if err != nil {
			return nil, fmt.Errorf(
				"build Dijkstra transition block %d: %w",
				i,
				err,
			)
		}
		blocks = append(blocks, block)
		currentPrev = block.Hash()
	}
	return blocks, nil
}

// checkChainRange rejects a chain whose last block number or slot would
// overflow uint64.
func checkChainRange(
	startBlockNumber, startSlot, slotIncrement, count uint64,
) error {
	lastOffset := count - 1
	if lastOffset > math.MaxUint64-startBlockNumber {
		return errors.New("block number range overflows uint64")
	}
	if slotIncrement != 0 &&
		lastOffset > (math.MaxUint64-startSlot)/slotIncrement {
		return errors.New("slot range overflows uint64")
	}
	return nil
}

// generateEraChain builds count connected empty blocks of the era with the
// given gouroboros ID.
func generateEraChain(
	eraID uint8,
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	return NewSequence(ledger.GetEraById(eraID)).
		WithStart(startBlockNumber, startSlot, prevHash).
		WithSlotIncrement(slotIncrement).
		Blocks(count)
}

// GenerateShelleyChain builds a connected chain of empty Shelley blocks using
// the first Shelley protocol version.
func GenerateShelleyChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	return generateEraChain(
		1, startBlockNumber, prevHash, startSlot, slotIncrement, count,
	)
}

// GenerateAllegraChain builds a connected chain of empty Allegra blocks.
func GenerateAllegraChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	return generateEraChain(
		2, startBlockNumber, prevHash, startSlot, slotIncrement, count,
	)
}

// GenerateMaryChain builds a connected chain of empty Mary blocks.
func GenerateMaryChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	return generateEraChain(
		3, startBlockNumber, prevHash, startSlot, slotIncrement, count,
	)
}

// GenerateAlonzoChain builds a connected chain of empty Alonzo blocks using
// the first Alonzo protocol version.
func GenerateAlonzoChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	return generateEraChain(
		4, startBlockNumber, prevHash, startSlot, slotIncrement, count,
	)
}

// GenerateBabbageChain builds a connected chain of empty Babbage blocks using
// the first Babbage protocol version.
func GenerateBabbageChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	return generateEraChain(
		5, startBlockNumber, prevHash, startSlot, slotIncrement, count,
	)
}

// GenerateBabbageChainWithProtocolVersion builds a connected chain of empty
// Babbage blocks with the requested header protocol version. The explicit
// version is useful for exercising header classification at hard-fork and
// unknown-version boundaries while retaining structurally valid block bytes.
func GenerateBabbageChainWithProtocolVersion(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	protocolMajor, protocolMinor uint64,
	count int,
) ([]ledger.Block, error) {
	return NewSequence(ledger.GetEraById(5)).
		WithStart(startBlockNumber, startSlot, prevHash).
		WithSlotIncrement(slotIncrement).
		WithProtocolVersion(protocolMajor, protocolMinor).
		Blocks(count)
}

// GenerateDijkstraChain builds a connected chain of empty Dijkstra blocks.
//
// Its headers carry the 10-field Babbage header body, which
// ledger.DetermineBlockType classifies. NewDijkstraBlockBuilder and
// GenerateConwayToDijkstraChain emit the 12-field header body of the pinned
// Dijkstra CDDL instead, which DetermineBlockType rejects as an unknown header
// body length.
func GenerateDijkstraChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	if count <= 0 {
		return []ledger.Block{}, nil
	}
	if err := checkChainRange(
		startBlockNumber, startSlot, slotIncrement, uint64(count),
	); err != nil {
		return nil, err
	}
	body := dijkstra.DijkstraBlockBody{
		InvalidTransactions: []uint{},
		Transactions:        []dijkstra.DijkstraTransaction{},
	}
	bodyCbor, err := cbor.Encode(body)
	if err != nil {
		return nil, fmt.Errorf("encode empty Dijkstra block body: %w", err)
	}
	bodySize := uint64(len(bodyCbor))
	bodyHash := body.Hash()
	blocks := make([]ledger.Block, 0, count)
	currentPrev := prevHash
	for i := range count {
		block := &dijkstra.DijkstraBlock{
			BlockHeader: &dijkstra.DijkstraBlockHeader{
				BabbageBlockHeader: babbage.BabbageBlockHeader{
					Body: babbage.BabbageBlockHeaderBody{
						BlockNumber: startBlockNumber + uint64(i),
						Slot:        startSlot + uint64(i)*slotIncrement,
						PrevHash:    currentPrev,
						IssuerVkey:  common.IssuerVkey{},
						VrfKey:      make([]byte, 32),
						VrfResult: common.VrfResult{
							Output: make([]byte, 64),
							Proof:  make([]byte, 80),
						},
						BlockBodySize: bodySize,
						BlockBodyHash: bodyHash,
						OpCert: babbage.BabbageOpCert{
							HotVkey:   make([]byte, 32),
							Signature: make([]byte, 64),
						},
						ProtoVersion: babbage.BabbageProtoVersion{
							Major: dijkstra.MinProtocolVersionDijkstra,
						},
					},
					Signature: make([]byte, 448),
				},
			},
			BlockBody: body,
		}
		blockCbor, err := cbor.Encode(block)
		if err != nil {
			return nil, fmt.Errorf("encode Dijkstra block %d: %w", i, err)
		}
		decoded, err := dijkstra.NewDijkstraBlockFromCbor(blockCbor)
		if err != nil {
			return nil, fmt.Errorf(
				"decode generated Dijkstra block %d: %w",
				i,
				err,
			)
		}
		if !bytes.Equal(decoded.Cbor(), blockCbor) {
			return nil, fmt.Errorf(
				"dijkstra block %d Cbor mismatch after round-trip",
				i,
			)
		}
		blocks = append(blocks, decoded)
		currentPrev = decoded.Hash()
	}
	return blocks, nil
}
