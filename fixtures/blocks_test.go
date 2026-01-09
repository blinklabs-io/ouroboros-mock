// Copyright 2025 Blink Labs Software
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
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/allegra"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

func TestHashFromString(t *testing.T) {
	hash1 := hashFromString("test")
	hash2 := hashFromString("test")
	hash3 := hashFromString("different")

	if len(hash1) != 32 {
		t.Errorf("expected hash length 32, got %d", len(hash1))
	}

	// Same input should produce same hash
	if !bytes.Equal(hash1, hash2) {
		t.Error("same input should produce same hash")
	}

	// Different input should produce different hash
	if bytes.Equal(hash1, hash3) {
		t.Error("different input should produce different hash")
	}
}

func TestHashFromSlotAndBlock(t *testing.T) {
	hash1 := hashFromSlotAndBlock(100, 50)
	hash2 := hashFromSlotAndBlock(100, 50)
	hash3 := hashFromSlotAndBlock(101, 50)

	if len(hash1) != 32 {
		t.Errorf("expected hash length 32, got %d", len(hash1))
	}

	// Same input should produce same hash
	if !bytes.Equal(hash1, hash2) {
		t.Error("same input should produce same hash")
	}

	// Different input should produce different hash
	if bytes.Equal(hash1, hash3) {
		t.Error("different input should produce different hash")
	}
}

func TestPreDefinedHashes(t *testing.T) {
	if len(GenesisHash) != 32 {
		t.Errorf("GenesisHash should be 32 bytes, got %d", len(GenesisHash))
	}

	// Genesis hash should be all zeros
	for i, b := range GenesisHash {
		if b != 0 {
			t.Errorf("GenesisHash[%d] should be 0, got %d", i, b)
		}
	}

	hashes := [][]byte{TestHash1, TestHash2, TestHash3, TestHash4, TestHash5}
	for i, h := range hashes {
		if len(h) != 32 {
			t.Errorf("TestHash%d should be 32 bytes, got %d", i+1, len(h))
		}
	}

	// All test hashes should be different
	for i := range len(hashes) - 1 {
		for j := i + 1; j < len(hashes); j++ {
			if bytes.Equal(hashes[i], hashes[j]) {
				t.Errorf(
					"TestHash%d and TestHash%d should be different",
					i+1,
					j+1,
				)
			}
		}
	}
}

func TestNewMockByronBlock(t *testing.T) {
	block, err := NewMockByronBlock(100, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.SlotNumber() != 100 {
		t.Errorf("expected slot 100, got %d", block.SlotNumber())
	}
	if block.BlockNumber() != 50 {
		t.Errorf("expected block number 50, got %d", block.BlockNumber())
	}
	if block.Era().Id != byron.EraIdByron {
		t.Errorf("expected Byron era, got %v", block.Era())
	}
}

func TestNewMockByronEBB(t *testing.T) {
	block, err := NewMockByronEBB(0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.Era().Id != byron.EraIdByron {
		t.Errorf("expected Byron era, got %v", block.Era())
	}
}

func TestNewMockShelleyBlock(t *testing.T) {
	block, err := NewMockShelleyBlock(100, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.SlotNumber() != 100 {
		t.Errorf("expected slot 100, got %d", block.SlotNumber())
	}
	if block.BlockNumber() != 50 {
		t.Errorf("expected block number 50, got %d", block.BlockNumber())
	}
	if block.Era().Id != shelley.EraIdShelley {
		t.Errorf("expected Shelley era, got %v", block.Era())
	}
}

func TestNewMockAllegraBlock(t *testing.T) {
	block, err := NewMockAllegraBlock(100, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.Era().Id != allegra.EraIdAllegra {
		t.Errorf("expected Allegra era, got %v", block.Era())
	}
}

func TestNewMockMaryBlock(t *testing.T) {
	block, err := NewMockMaryBlock(100, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.Era().Id != mary.EraIdMary {
		t.Errorf("expected Mary era, got %v", block.Era())
	}
}

func TestNewMockAlonzoBlock(t *testing.T) {
	block, err := NewMockAlonzoBlock(100, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.Era().Id != alonzo.EraIdAlonzo {
		t.Errorf("expected Alonzo era, got %v", block.Era())
	}
}

func TestNewMockBabbageBlock(t *testing.T) {
	block, err := NewMockBabbageBlock(100, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.Era().Id != babbage.EraIdBabbage {
		t.Errorf("expected Babbage era, got %v", block.Era())
	}
}

func TestNewMockConwayBlock(t *testing.T) {
	block, err := NewMockConwayBlock(100, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.SlotNumber() != 100 {
		t.Errorf("expected slot 100, got %d", block.SlotNumber())
	}
	if block.BlockNumber() != 50 {
		t.Errorf("expected block number 50, got %d", block.BlockNumber())
	}
	if block.Era().Id != conway.EraIdConway {
		t.Errorf("expected Conway era, got %v", block.Era())
	}
}

func TestNewMockConwayChain(t *testing.T) {
	chain, err := NewMockConwayChain(100, 50, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain) != 5 {
		t.Fatalf("expected 5 blocks, got %d", len(chain))
	}

	// Verify chain properties
	for i, block := range chain {
		expectedSlot := uint64(100 + i)
		expectedBlockNum := uint64(50 + i)

		if block.SlotNumber() != expectedSlot {
			t.Errorf(
				"block %d: expected slot %d, got %d",
				i,
				expectedSlot,
				block.SlotNumber(),
			)
		}
		if block.BlockNumber() != expectedBlockNum {
			t.Errorf(
				"block %d: expected block number %d, got %d",
				i,
				expectedBlockNum,
				block.BlockNumber(),
			)
		}
	}

	// Verify blocks are linked
	for i := 1; i < len(chain); i++ {
		prevHash := chain[i].PrevHash()
		expectedPrev := chain[i-1].Hash()
		if prevHash != expectedPrev {
			t.Errorf(
				"block %d: prevHash does not match previous block's hash",
				i,
			)
		}
	}
}

func TestNewMockForkScenario(t *testing.T) {
	scenario, err := NewMockForkScenario(10, 5, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scenario.MainChain) != 10 {
		t.Errorf("expected 10 main blocks, got %d", len(scenario.MainChain))
	}
	if scenario.ForkPoint != 5 {
		t.Errorf("expected fork point 5, got %d", scenario.ForkPoint)
	}
	if len(scenario.ForkChain) != 3 {
		t.Errorf("expected 3 fork blocks, got %d", len(scenario.ForkChain))
	}

	// Verify fork chain starts from correct point
	if len(scenario.ForkChain) > 0 {
		forkStart := scenario.ForkChain[0]
		expectedPrev := scenario.MainChain[scenario.ForkPoint-1].Hash()
		if forkStart.PrevHash() != expectedPrev {
			t.Error(
				"fork chain should reference main chain block before fork point",
			)
		}
	}

	// Verify fork blocks are linked
	for i := 1; i < len(scenario.ForkChain); i++ {
		prevHash := scenario.ForkChain[i].PrevHash()
		expectedPrev := scenario.ForkChain[i-1].Hash()
		if prevHash != expectedPrev {
			t.Errorf(
				"fork block %d: prevHash does not match previous fork block's hash",
				i,
			)
		}
	}
}

func TestNewMockForkScenario_InvalidForkPoint(t *testing.T) {
	// Fork point beyond main chain length should be adjusted
	scenario, err := NewMockForkScenario(10, 15, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be adjusted to mainLength / 2
	if scenario.ForkPoint != 5 {
		t.Errorf("expected adjusted fork point 5, got %d", scenario.ForkPoint)
	}
}

func TestNewMockByronToShelleyTransition(t *testing.T) {
	scenario, err := NewMockByronToShelleyTransition(5, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scenario.PreTransitionBlocks) != 5 {
		t.Errorf(
			"expected 5 Byron blocks, got %d",
			len(scenario.PreTransitionBlocks),
		)
	}
	if len(scenario.PostTransitionBlocks) != 3 {
		t.Errorf(
			"expected 3 Shelley blocks, got %d",
			len(scenario.PostTransitionBlocks),
		)
	}

	// Verify all pre-transition blocks are Byron
	for i, block := range scenario.PreTransitionBlocks {
		if block.Era().Id != byron.EraIdByron {
			t.Errorf(
				"pre-transition block %d should be Byron, got %v",
				i,
				block.Era(),
			)
		}
	}

	// Verify all post-transition blocks are Shelley
	for i, block := range scenario.PostTransitionBlocks {
		if block.Era().Id != shelley.EraIdShelley {
			t.Errorf(
				"post-transition block %d should be Shelley, got %v",
				i,
				block.Era(),
			)
		}
	}

	// Verify transition slot
	expectedTransitionSlot := uint64(5 * 20) // Byron uses 20 second slots
	if scenario.TransitionSlot != expectedTransitionSlot {
		t.Errorf(
			"expected transition slot %d, got %d",
			expectedTransitionSlot,
			scenario.TransitionSlot,
		)
	}

	// Verify Shelley blocks continue from Byron
	if len(scenario.PreTransitionBlocks) > 0 &&
		len(scenario.PostTransitionBlocks) > 0 {
		lastByron := scenario.PreTransitionBlocks[len(scenario.PreTransitionBlocks)-1]
		firstShelley := scenario.PostTransitionBlocks[0]
		if firstShelley.PrevHash() != lastByron.Hash() {
			t.Error("Shelley chain should continue from last Byron block")
		}
	}
}

func TestNewMockBabbageToConwayTransition(t *testing.T) {
	scenario, err := NewMockBabbageToConwayTransition(5, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scenario.PreTransitionBlocks) != 5 {
		t.Errorf(
			"expected 5 Babbage blocks, got %d",
			len(scenario.PreTransitionBlocks),
		)
	}
	if len(scenario.PostTransitionBlocks) != 3 {
		t.Errorf(
			"expected 3 Conway blocks, got %d",
			len(scenario.PostTransitionBlocks),
		)
	}

	// Verify all pre-transition blocks are Babbage
	for i, block := range scenario.PreTransitionBlocks {
		if block.Era().Id != babbage.EraIdBabbage {
			t.Errorf(
				"pre-transition block %d should be Babbage, got %v",
				i,
				block.Era(),
			)
		}
	}

	// Verify all post-transition blocks are Conway
	for i, block := range scenario.PostTransitionBlocks {
		if block.Era().Id != conway.EraIdConway {
			t.Errorf(
				"post-transition block %d should be Conway, got %v",
				i,
				block.Era(),
			)
		}
	}

	// Verify Conway blocks continue from Babbage
	if len(scenario.PreTransitionBlocks) > 0 &&
		len(scenario.PostTransitionBlocks) > 0 {
		lastBabbage := scenario.PreTransitionBlocks[len(scenario.PreTransitionBlocks)-1]
		firstConway := scenario.PostTransitionBlocks[0]
		if firstConway.PrevHash() != lastBabbage.Hash() {
			t.Error("Conway chain should continue from last Babbage block")
		}
	}
}
