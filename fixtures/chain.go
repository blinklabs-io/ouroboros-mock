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
	"encoding/binary"
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"golang.org/x/crypto/blake2b"
)

// GenerateConwayChain builds count Conway blocks that chain together via
// PrevHash, with CBOR that round-trips cleanly through
// ledger.NewBlockFromCbor: a consumer that re-decodes the returned block bytes
// recovers the same Hash() and PrevHash() the generator produced. This makes
// the blocks usable as stable fixtures for chain-reconcile and rollback tests
// in downstream repos.
//
// The first block's PrevHash is set to prevHash. Each block's slot is
// startSlot + i*slotIncrement; block numbers run
// startBlockNumber..startBlockNumber+count-1. All blocks have empty
// transaction, witness, auxiliary, and invalid-transaction sets, so they
// share the same block body hash. A count of zero or less returns an empty,
// non-nil slice and no error.
func GenerateConwayChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	if count <= 0 {
		return []ledger.Block{}, nil
	}
	// All generated blocks have identical empty bodies, so the four component
	// CBORs and the resulting block body hash are constant across the chain.
	emptyTxsCbor, err := cbor.Encode([]ledger.ConwayTransactionBody{})
	if err != nil {
		return nil, fmt.Errorf("encode empty tx bodies: %w", err)
	}
	emptyWitsCbor, err := cbor.Encode([]ledger.ConwayTransactionWitnessSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty witnesses: %w", err)
	}
	emptyAuxCbor, err := cbor.Encode(common.TransactionMetadataSet{})
	if err != nil {
		return nil, fmt.Errorf("encode empty metadata set: %w", err)
	}
	emptyInvalidCbor, err := cbor.Encode([]uint{})
	if err != nil {
		return nil, fmt.Errorf("encode empty invalid txs: %w", err)
	}
	bodyHash := ComputeBlockBodyHash(
		emptyTxsCbor, emptyWitsCbor, emptyAuxCbor, emptyInvalidCbor,
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
		body := babbage.BabbageBlockHeaderBody{
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
			ProtoVersion: babbage.BabbageProtoVersion{Major: 9, Minor: 0},
		}
		block := &ledger.ConwayBlock{
			BlockHeader: &ledger.ConwayBlockHeader{
				BabbageBlockHeader: ledger.BabbageBlockHeader{
					Body:      body,
					Signature: make([]byte, 64),
				},
			},
		}
		blockCbor, err := cbor.Encode(block)
		if err != nil {
			return nil, fmt.Errorf("encode block %d: %w", i, err)
		}
		// Re-decode so the returned block carries the canonical Cbor() a
		// consumer's reconcile path will observe, and so Hash() reads from the
		// post-round-trip header bytes.
		decoded, err := conway.NewConwayBlockFromCbor(blockCbor)
		if err != nil {
			return nil, fmt.Errorf("decode generated block %d: %w", i, err)
		}
		if !bytes.Equal(decoded.Cbor(), blockCbor) {
			return nil, fmt.Errorf(
				"block %d Cbor mismatch after round-trip",
				i,
			)
		}
		blocks = append(blocks, decoded)
		currentPrev = decoded.Hash()
	}
	return blocks, nil
}

// GenerateConwayChainWithTransactions builds count connected Conway blocks,
// each containing one wire-decodable transaction with one simple output. The
// generated transactions have distinct output addresses and therefore
// distinct transaction-body hashes. The transaction data is protocol/wire
// valid test data, but is not intended to satisfy full ledger balance rules:
// the transactions have no inputs and are not spendable ledger transactions.
func GenerateConwayChainWithTransactions(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) ([]ledger.Block, error) {
	if count <= 0 {
		return []ledger.Block{}, nil
	}
	blocks := make([]ledger.Block, 0, count)
	currentPrev := prevHash
	for i := range count {
		transactionBody, err := newConwayFixtureTransactionBody(
			uint64(i),
		)
		if err != nil {
			return nil, fmt.Errorf("build transaction %d: %w", i, err)
		}
		transactionBodies := []ledger.ConwayTransactionBody{transactionBody}
		transactionWitnessSets := []ledger.ConwayTransactionWitnessSet{{}}
		txsCbor, err := cbor.Encode(transactionBodies)
		if err != nil {
			return nil, fmt.Errorf("encode transaction bodies %d: %w", i, err)
		}
		witsCbor, err := cbor.Encode(transactionWitnessSets)
		if err != nil {
			return nil, fmt.Errorf("encode witnesses %d: %w", i, err)
		}
		auxCbor, err := cbor.Encode(common.TransactionMetadataSet{})
		if err != nil {
			return nil, fmt.Errorf("encode metadata set %d: %w", i, err)
		}
		invalidCbor, err := cbor.Encode([]uint{})
		if err != nil {
			return nil, fmt.Errorf("encode invalid transactions %d: %w", i, err)
		}
		bodyHash := ComputeBlockBodyHash(
			txsCbor, witsCbor, auxCbor, invalidCbor,
		)
		bodySize := computeBlockBodySize(
			txsCbor, witsCbor, auxCbor, invalidCbor,
		)
		body := babbage.BabbageBlockHeaderBody{
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
			ProtoVersion: babbage.BabbageProtoVersion{Major: 9, Minor: 0},
		}
		block := &ledger.ConwayBlock{
			BlockHeader: &ledger.ConwayBlockHeader{
				BabbageBlockHeader: ledger.BabbageBlockHeader{
					Body:      body,
					Signature: make([]byte, 64),
				},
			},
			TransactionBodies:      transactionBodies,
			TransactionWitnessSets: transactionWitnessSets,
			TransactionMetadataSet: common.TransactionMetadataSet{},
			InvalidTransactions:    []uint{},
		}
		blockCbor, err := cbor.Encode(block)
		if err != nil {
			return nil, fmt.Errorf("encode block %d: %w", i, err)
		}
		// Re-decode so the returned block carries the canonical Cbor() a
		// consumer's reconcile path will observe, and so Hash() reads from the
		// post-round-trip header bytes.
		decoded, err := conway.NewConwayBlockFromCbor(blockCbor)
		if err != nil {
			return nil, fmt.Errorf("decode generated block %d: %w", i, err)
		}
		if !bytes.Equal(decoded.Cbor(), blockCbor) {
			return nil, fmt.Errorf(
				"block %d Cbor mismatch after round-trip",
				i,
			)
		}
		blocks = append(blocks, decoded)
		currentPrev = decoded.Hash()
	}
	return blocks, nil
}

func newConwayFixtureTransactionBody(
	index uint64,
) (ledger.ConwayTransactionBody, error) {
	inputHash := common.Blake2b256{}
	binary.BigEndian.PutUint64(inputHash[:], index)
	addressBytes := make([]byte, 1+common.AddressHashSize)
	addressBytes[0] = common.AddressTypeKeyNone << 4
	// The address remains a fixed-width testnet enterprise address while its
	// payment payload identifies the block's transaction without sharing
	// mutable storage.
	binary.BigEndian.PutUint64(addressBytes[1:], index)
	address, err := common.NewAddressFromBytes(addressBytes)
	if err != nil {
		return ledger.ConwayTransactionBody{}, fmt.Errorf(
			"create output address: %w",
			err,
		)
	}
	return ledger.ConwayTransactionBody{
		TxInputs: conway.NewConwayTransactionInputSet(
			[]shelley.ShelleyTransactionInput{{
				TxId: inputHash,
			}},
		),
		TxOutputs: []babbage.BabbageTransactionOutput{{
			OutputAddress: address,
			OutputAmount: mary.MaryTransactionOutputValue{
				Amount: 1_000_000 + index,
			},
		}},
		TxFee: 1,
	}, nil
}

// ComputeBlockBodyHash returns
// blake2b256(blake2b256(parts[0]) || blake2b256(parts[1]) || ...), which
// matches the derivation expected by common.ValidateBlockBodyHash.
func ComputeBlockBodyHash(parts ...[]byte) common.Blake2b256 {
	var combined []byte
	for _, p := range parts {
		h := blake2b.Sum256(p)
		combined = append(combined, h[:]...)
	}
	h := blake2b.Sum256(combined)
	return common.NewBlake2b256(h[:])
}

func computeBlockBodySize(parts ...[]byte) uint64 {
	var size uint64
	for _, part := range parts {
		size += uint64(len(part))
	}
	return size
}
