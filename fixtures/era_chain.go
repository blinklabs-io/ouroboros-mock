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
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

// GenerateBabbageChain builds a connected chain of empty Babbage blocks using
// the first Babbage protocol version.
func GenerateBabbageChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	return GenerateBabbageChainWithProtocolVersion(
		startBlockNumber,
		prevHash,
		startSlot,
		slotIncrement,
		babbage.MinProtocolVersionBabbage,
		0,
		count,
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
	if count <= 0 {
		return []ledger.Block{}, nil
	}
	emptyTxsCbor, err := cbor.Encode([]babbage.BabbageTransactionBody{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Babbage tx bodies: %w", err)
	}
	emptyWitsCbor, err := cbor.Encode([]babbage.BabbageTransactionWitnessSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Babbage witnesses: %w", err)
	}
	emptyAuxCbor, err := cbor.Encode(common.TransactionMetadataSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Babbage metadata set: %w", err)
	}
	emptyInvalidCbor, err := cbor.Encode([]uint{})
	if err != nil {
		return nil, fmt.Errorf("encode empty invalid txs: %w", err)
	}
	bodyHash := ComputeBlockBodyHash(
		emptyTxsCbor,
		emptyWitsCbor,
		emptyAuxCbor,
		emptyInvalidCbor,
	)
	bodySize := computeBlockBodySize(
		emptyTxsCbor,
		emptyWitsCbor,
		emptyAuxCbor,
		emptyInvalidCbor,
	)
	blocks := make([]ledger.Block, 0, count)
	currentPrev := prevHash
	for i := range count {
		block := &babbage.BabbageBlock{
			BlockHeader: &babbage.BabbageBlockHeader{
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
						Major: protocolMajor,
						Minor: protocolMinor,
					},
				},
				Signature: make([]byte, 64),
			},
		}
		blockCbor, err := cbor.Encode(block)
		if err != nil {
			return nil, fmt.Errorf("encode Babbage block %d: %w", i, err)
		}
		decoded, err := babbage.NewBabbageBlockFromCbor(blockCbor)
		if err != nil {
			return nil, fmt.Errorf("decode generated Babbage block %d: %w", i, err)
		}
		if !bytes.Equal(decoded.Cbor(), blockCbor) {
			return nil, fmt.Errorf(
				"Babbage block %d Cbor mismatch after round-trip",
				i,
			)
		}
		blocks = append(blocks, decoded)
		currentPrev = decoded.Hash()
	}
	return blocks, nil
}

// GenerateShelleyChain builds a connected chain of empty Shelley blocks using
// the first Shelley protocol version.
func GenerateShelleyChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	if count <= 0 {
		return []ledger.Block{}, nil
	}
	emptyTxsCbor, err := cbor.Encode([]shelley.ShelleyTransactionBody{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Shelley tx bodies: %w", err)
	}
	emptyWitsCbor, err := cbor.Encode([]shelley.ShelleyTransactionWitnessSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Shelley witnesses: %w", err)
	}
	emptyAuxCbor, err := cbor.Encode(common.TransactionMetadataSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Shelley metadata set: %w", err)
	}
	bodyHash := ComputeBlockBodyHash(
		emptyTxsCbor,
		emptyWitsCbor,
		emptyAuxCbor,
	)
	bodySize := computeBlockBodySize(
		emptyTxsCbor,
		emptyWitsCbor,
		emptyAuxCbor,
	)
	blocks := make([]ledger.Block, 0, count)
	currentPrev := prevHash
	for i := range count {
		block := &shelley.ShelleyBlock{
			BlockHeader: &shelley.ShelleyBlockHeader{
				Body: shelley.ShelleyBlockHeaderBody{
					BlockNumber: startBlockNumber + uint64(i),
					Slot:        startSlot + uint64(i)*slotIncrement,
					PrevHash:    currentPrev,
					IssuerVkey:  common.IssuerVkey{},
					VrfKey:      make([]byte, 32),
					NonceVrf: common.VrfResult{
						Output: make([]byte, 64),
						Proof:  make([]byte, 80),
					},
					LeaderVrf: common.VrfResult{
						Output: make([]byte, 64),
						Proof:  make([]byte, 80),
					},
					BlockBodySize:     bodySize,
					BlockBodyHash:     bodyHash,
					OpCertHotVkey:     make([]byte, 32),
					OpCertSignature:   make([]byte, 64),
					ProtoMajorVersion: shelley.MinProtocolVersionShelley,
					ProtoMinorVersion: 0,
				},
				Signature: make([]byte, 64),
			},
		}
		blockCbor, err := cbor.Encode(block)
		if err != nil {
			return nil, fmt.Errorf("encode Shelley block %d: %w", i, err)
		}
		decoded, err := shelley.NewShelleyBlockFromCbor(blockCbor)
		if err != nil {
			return nil, fmt.Errorf("decode generated Shelley block %d: %w", i, err)
		}
		if !bytes.Equal(decoded.Cbor(), blockCbor) {
			return nil, fmt.Errorf(
				"Shelley block %d Cbor mismatch after round-trip",
				i,
			)
		}
		blocks = append(blocks, decoded)
		currentPrev = decoded.Hash()
	}
	return blocks, nil
}
