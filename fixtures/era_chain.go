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
	"github.com/blinklabs-io/gouroboros/ledger/allegra"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

// GenerateDijkstraChain builds a connected chain of empty Dijkstra blocks.
func GenerateDijkstraChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	if count <= 0 {
		return []ledger.Block{}, nil
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
					Signature: make([]byte, 64),
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
			return nil, fmt.Errorf("decode generated Dijkstra block %d: %w", i, err)
		}
		if !bytes.Equal(decoded.Cbor(), blockCbor) {
			return nil, fmt.Errorf("dijkstra block %d Cbor mismatch after round-trip", i)
		}
		blocks = append(blocks, decoded)
		currentPrev = decoded.Hash()
	}
	return blocks, nil
}

// GenerateAllegraChain builds a connected chain of empty Allegra blocks.
func GenerateAllegraChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	if count <= 0 {
		return []ledger.Block{}, nil
	}
	emptyTxsCbor, err := cbor.Encode([]allegra.AllegraTransactionBody{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Allegra tx bodies: %w", err)
	}
	emptyWitsCbor, err := cbor.Encode([]shelley.ShelleyTransactionWitnessSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Allegra witnesses: %w", err)
	}
	emptyAuxCbor, err := cbor.Encode(common.TransactionMetadataSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Allegra metadata set: %w", err)
	}
	bodyHash := ComputeBlockBodyHash(emptyTxsCbor, emptyWitsCbor, emptyAuxCbor)
	bodySize := computeBlockBodySize(emptyTxsCbor, emptyWitsCbor, emptyAuxCbor)
	blocks := make([]ledger.Block, 0, count)
	currentPrev := prevHash
	for i := range count {
		block := &allegra.AllegraBlock{
			BlockHeader: &allegra.AllegraBlockHeader{
				ShelleyBlockHeader: shelley.ShelleyBlockHeader{
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
						ProtoMajorVersion: allegra.MinProtocolVersionAllegra,
						ProtoMinorVersion: 0,
					},
					Signature: make([]byte, 64),
				},
			},
			TransactionBodies:      []allegra.AllegraTransactionBody{},
			TransactionWitnessSets: []shelley.ShelleyTransactionWitnessSet{},
			TransactionMetadataSet: common.TransactionMetadataSet{},
		}
		blockCbor, err := cbor.Encode(block)
		if err != nil {
			return nil, fmt.Errorf("encode Allegra block %d: %w", i, err)
		}
		decoded, err := allegra.NewAllegraBlockFromCbor(blockCbor)
		if err != nil {
			return nil, fmt.Errorf("decode generated Allegra block %d: %w", i, err)
		}
		if !bytes.Equal(decoded.Cbor(), blockCbor) {
			return nil, fmt.Errorf("allegra block %d Cbor mismatch after round-trip", i)
		}
		blocks = append(blocks, decoded)
		currentPrev = decoded.Hash()
	}
	return blocks, nil
}

// GenerateMaryChain builds a connected chain of empty Mary blocks.
func GenerateMaryChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	if count <= 0 {
		return []ledger.Block{}, nil
	}
	emptyTxsCbor, err := cbor.Encode([]mary.MaryTransactionBody{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Mary tx bodies: %w", err)
	}
	emptyWitsCbor, err := cbor.Encode([]shelley.ShelleyTransactionWitnessSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Mary witnesses: %w", err)
	}
	emptyAuxCbor, err := cbor.Encode(common.TransactionMetadataSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Mary metadata set: %w", err)
	}
	bodyHash := ComputeBlockBodyHash(emptyTxsCbor, emptyWitsCbor, emptyAuxCbor)
	bodySize := computeBlockBodySize(emptyTxsCbor, emptyWitsCbor, emptyAuxCbor)
	blocks := make([]ledger.Block, 0, count)
	currentPrev := prevHash
	for i := range count {
		block := &mary.MaryBlock{
			BlockHeader: &mary.MaryBlockHeader{
				ShelleyBlockHeader: shelley.ShelleyBlockHeader{
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
						ProtoMajorVersion: mary.MinProtocolVersionMary,
						ProtoMinorVersion: 0,
					},
					Signature: make([]byte, 64),
				},
			},
			TransactionBodies:      []mary.MaryTransactionBody{},
			TransactionWitnessSets: []shelley.ShelleyTransactionWitnessSet{},
			TransactionMetadataSet: common.TransactionMetadataSet{},
		}
		blockCbor, err := cbor.Encode(block)
		if err != nil {
			return nil, fmt.Errorf("encode Mary block %d: %w", i, err)
		}
		decoded, err := mary.NewMaryBlockFromCbor(blockCbor)
		if err != nil {
			return nil, fmt.Errorf("decode generated Mary block %d: %w", i, err)
		}
		if !bytes.Equal(decoded.Cbor(), blockCbor) {
			return nil, fmt.Errorf("mary block %d Cbor mismatch after round-trip", i)
		}
		blocks = append(blocks, decoded)
		currentPrev = decoded.Hash()
	}
	return blocks, nil
}

// GenerateAlonzoChain builds a connected chain of empty Alonzo blocks using
// the first Alonzo protocol version.
func GenerateAlonzoChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	if count <= 0 {
		return []ledger.Block{}, nil
	}
	emptyTxsCbor, err := cbor.Encode([]alonzo.AlonzoTransactionBody{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Alonzo tx bodies: %w", err)
	}
	emptyWitsCbor, err := cbor.Encode([]alonzo.AlonzoTransactionWitnessSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Alonzo witnesses: %w", err)
	}
	emptyAuxCbor, err := cbor.Encode(common.TransactionMetadataSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Alonzo metadata set: %w", err)
	}
	emptyInvalidCbor, err := cbor.Encode(cbor.IndefLengthList{})
	if err != nil {
		return nil, fmt.Errorf("encode empty Alonzo invalid txs: %w", err)
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
		block := &alonzo.AlonzoBlock{
			BlockHeader: &alonzo.AlonzoBlockHeader{
				ShelleyBlockHeader: shelley.ShelleyBlockHeader{
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
						ProtoMajorVersion: alonzo.MinProtocolVersionAlonzo,
						ProtoMinorVersion: 0,
					},
					Signature: make([]byte, 64),
				},
			},
		}
		blockCbor, err := cbor.Encode(block)
		if err != nil {
			return nil, fmt.Errorf("encode Alonzo block %d: %w", i, err)
		}
		decoded, err := alonzo.NewAlonzoBlockFromCbor(blockCbor)
		if err != nil {
			return nil, fmt.Errorf(
				"decode generated Alonzo block %d: %w",
				i,
				err,
			)
		}
		if !bytes.Equal(decoded.Cbor(), blockCbor) {
			return nil, fmt.Errorf(
				"Alonzo block %d Cbor mismatch after round-trip",
				i,
			)
		}
		blocks = append(blocks, decoded)
		currentPrev = decoded.Hash()
	}
	return blocks, nil
}

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
			return nil, fmt.Errorf(
				"decode generated Babbage block %d: %w",
				i,
				err,
			)
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
			return nil, fmt.Errorf(
				"decode generated Shelley block %d: %w",
				i,
				err,
			)
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
