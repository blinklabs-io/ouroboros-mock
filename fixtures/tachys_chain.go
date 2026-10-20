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
	"fmt"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// TachysEpochFixture associates one epoch nonce with its active distribution.
type TachysEpochFixture struct {
	Epoch        uint64
	FirstSlot    uint64
	EpochNonce   common.Blake2b256
	Distribution TachysDistributionVector
}

// TachysChainScenario contains a Babbage-to-Conway chain, a competing Conway
// branch, the common rollback point, and two epoch stake distributions.
type TachysChainScenario struct {
	Canonical           []ledger.Block
	Competing           []ledger.Block
	RollbackBlockNumber uint64
	RollbackHash        common.Blake2b256
	Epochs              []TachysEpochFixture
}

// NewTachysChainScenario builds a deterministic transition and rollback
// scenario. The canonical chain has two Babbage and three Conway blocks. The
// competing branch forks after the Babbage tip and carries two Conway blocks.
func NewTachysChainScenario() (TachysChainScenario, error) {
	genesis := common.Blake2b256Hash([]byte("Tachys chain genesis"))
	babbageBlocks, err := GenerateBabbageChain(1, genesis, 90, 5, 2)
	if err != nil {
		return TachysChainScenario{}, fmt.Errorf("build Babbage prefix: %w", err)
	}
	rollback := babbageBlocks[len(babbageBlocks)-1]
	conwayBlocks, err := GenerateConwayChain(
		3,
		rollback.Hash(),
		100,
		5,
		3,
	)
	if err != nil {
		return TachysChainScenario{}, fmt.Errorf("build Conway suffix: %w", err)
	}
	competing, err := GenerateConwayChain(
		3,
		rollback.Hash(),
		101,
		5,
		2,
	)
	if err != nil {
		return TachysChainScenario{}, fmt.Errorf("build competing branch: %w", err)
	}
	vectors, err := LoadTachysLeaderVectors()
	if err != nil {
		return TachysChainScenario{}, fmt.Errorf("load leader vectors: %w", err)
	}
	second := TachysDistributionVector{
		Pools: []TachysPoolVector{
			{PoolID: vectors.Distribution.Pools[0].PoolID, Stake: 3},
			{PoolID: vectors.Distribution.Pools[1].PoolID, Stake: 1},
			{PoolID: vectors.Distribution.Pools[2].PoolID, Stake: 2},
		},
		TotalStake: 6,
	}
	canonical := append([]ledger.Block{}, babbageBlocks...)
	canonical = append(canonical, conwayBlocks...)
	return TachysChainScenario{
		Canonical:           canonical,
		Competing:           competing,
		RollbackBlockNumber: rollback.BlockNumber(),
		RollbackHash:        rollback.Hash(),
		Epochs: []TachysEpochFixture{
			{
				Epoch:        0,
				FirstSlot:    0,
				EpochNonce:   common.Blake2b256Hash([]byte("Tachys epoch zero")),
				Distribution: vectors.Distribution,
			},
			{
				Epoch:        1,
				FirstSlot:    100,
				EpochNonce:   common.Blake2b256Hash([]byte("Tachys epoch one")),
				Distribution: second,
			},
		},
	}, nil
}
