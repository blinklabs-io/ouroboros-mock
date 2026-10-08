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
	"encoding/binary"
	"math/rand/v2"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// Sequence builds connected blocks: each block's previous hash is the hash of
// the block before it, and its number and slot advance by one block and the
// slot increment.
type Sequence struct {
	template      BlockBuilder
	slotIncrement uint64
	advanceErr    error
}

// NewSequence creates a sequence of empty blocks for era that starts at block
// number 0, slot 0, with a zero previous hash and a slot increment of 1 (one
// Byron epoch for Byron).
func NewSequence(era common.Era) *Sequence {
	s := &Sequence{template: BlockBuilder{era: era}, slotIncrement: 1}
	if era.Id == byron.EraIdByron {
		s.slotIncrement = byron.ByronSlotsPerEpoch
	}
	return s
}

// WithStart sets the first block's number, slot and previous hash.
func (s *Sequence) WithStart(
	blockNumber, slot uint64,
	prevHash common.Blake2b256,
) *Sequence {
	s.template.blockNumber = blockNumber
	s.template.slot = slot
	s.template.prevHash = prevHash
	s.advanceErr = nil
	return s
}

// WithSlotIncrement sets the slot distance between consecutive blocks.
func (s *Sequence) WithSlotIncrement(increment uint64) *Sequence {
	s.slotIncrement = increment
	return s
}

// WithProtocolVersion sets the protocol version of every block header.
func (s *Sequence) WithProtocolVersion(major, minor uint64) *Sequence {
	s.template.WithProtocolVersion(major, minor)
	return s
}

// Next builds the next block, rejecting number or slot overflow.
func (s *Sequence) Next() (ledger.Block, error) {
	if s.advanceErr != nil {
		return nil, s.advanceErr
	}
	block, err := s.template.Build()
	if err != nil {
		return nil, err
	}
	s.advanceErr = checkChainRange(
		s.template.blockNumber, s.template.slot, s.slotIncrement, 2,
	)
	if s.advanceErr == nil {
		s.template.blockNumber++
		s.template.slot += s.slotIncrement
		s.template.prevHash = block.Hash()
	}
	return block, nil
}

// Blocks builds the next count blocks. A count of zero or less returns an
// empty, non-nil slice.
func (s *Sequence) Blocks(count int) ([]ledger.Block, error) {
	if count <= 0 {
		return []ledger.Block{}, nil
	}
	if s.advanceErr != nil {
		return nil, s.advanceErr
	}
	if err := checkChainRange(
		s.template.blockNumber, s.template.slot, s.slotIncrement, uint64(count),
	); err != nil {
		return nil, err
	}
	blocks := make([]ledger.Block, 0, count)
	for range count {
		block, err := s.Next()
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// GenesisBlock returns the empty first block of era: block number 0, slot 0
// and a zero previous hash.
func GenesisBlock(era common.Era) (ledger.Block, error) {
	return GenerateBlock(era, 0, 0)
}

// RandomBlock returns an empty block of era whose block number, slot,
// previous hash and issuer key are drawn from seed. The same era and seed
// produce the same block within a Go release. Byron blocks are epoch boundary
// blocks, so their slot is a multiple of the Byron epoch length.
func RandomBlock(era common.Era, seed uint64) (ledger.Block, error) {
	var chachaSeed [32]byte
	binary.LittleEndian.PutUint64(chachaSeed[:], seed)
	chachaSeed[8] = era.Id
	source := rand.NewChaCha8(chachaSeed)
	//nolint:gosec // deterministic fixture randomness.
	rng := rand.New(source)
	var prevHash common.Blake2b256
	var issuer common.IssuerVkey
	_, _ = source.Read(prevHash[:])
	_, _ = source.Read(issuer[:])
	slot := rng.Uint64N(1 << 40)
	if era.Id == byron.EraIdByron {
		slot -= slot % byron.ByronSlotsPerEpoch
	}
	return NewBlockBuilder(era).
		WithBlockNumber(rng.Uint64N(1 << 32)).
		WithSlot(slot).
		WithPreviousHash(prevHash).
		WithIssuerVkey(issuer).
		Build()
}
