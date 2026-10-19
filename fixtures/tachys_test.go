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
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/consensus"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
)

func TestLoadTachysLeaderVectors(t *testing.T) {
	t.Parallel()
	vectors, err := fixtures.LoadTachysLeaderVectors()
	require.NoError(t, err)
	require.Equal(t, 1, vectors.FormatVersion)
	require.Equal(t, "CIP-0177", vectors.Specification.Name)
	require.Len(t, vectors.Distribution.Pools, 3)
	require.Len(t, vectors.OrderingCases, 1)
	require.NotEmpty(t, vectors.SelectionCases)
	require.NotEmpty(t, vectors.BoundaryCases)
	require.NotEmpty(t, vectors.InvalidCases)
	require.Equal(
		t,
		"4a1931803561f431decab002e7425f0a8531d5e456a1a47fd9998a2530c0f800",
		vectors.SelectionCases[0].Sample,
	)
	require.Equal(
		t,
		"2aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab",
		vectors.BoundaryCases[2].Sample,
	)
}

func TestTachysSignedHeaderFixtures(t *testing.T) {
	t.Parallel()
	for _, eraID := range []uint8{5, 6} {
		era := ledger.GetEraById(eraID)
		set, err := fixtures.NewTachysSignedHeaderFixtures(era)
		require.NoError(t, err)
		require.Equal(t, era, set.Scheduled.Block.Era())
		require.Equal(t, era, set.WrongProducer.Block.Era())
		require.NotEqual(t, set.Scheduled.PoolID, set.WrongProducer.PoolID)
		require.False(t, set.Scheduled.PraosPassed)

		config := consensus.NetworkConfig{
			ActiveSlotCoeff:   common.GenesisRat{Rat: big.NewRat(1, 1)},
			SlotsPerKESPeriod: 129_600,
			MaxKESEvolutions:  62,
		}
		validator := consensus.NewHeaderValidator(config)
		require.True(t, validator.ValidateHeader(&set.Scheduled.Input).Valid)
		require.True(t, validator.ValidateHeader(&set.WrongProducer.Input).Valid)
		invalid := []*consensus.ValidateHeaderInput{
			&set.InvalidVRF,
			&set.InvalidKES,
			&set.InvalidOpCert,
		}
		for _, input := range invalid {
			require.False(t, validator.ValidateHeader(input).Valid)
		}
	}
}

func TestTachysChainScenario(t *testing.T) {
	t.Parallel()
	scenario, err := fixtures.NewTachysChainScenario()
	require.NoError(t, err)
	require.Len(t, scenario.Canonical, 5)
	require.Len(t, scenario.Competing, 2)
	require.Len(t, scenario.Epochs, 2)
	require.Equal(t, uint64(2), scenario.RollbackBlockNumber)
	require.Equal(t, scenario.RollbackHash, scenario.Canonical[1].Hash())
	require.Equal(t, scenario.RollbackHash, scenario.Competing[0].PrevHash())
	require.Equal(t, "Babbage", scenario.Canonical[1].Era().Name)
	require.Equal(t, "Conway", scenario.Canonical[2].Era().Name)
	require.NotEqual(
		t,
		scenario.Epochs[0].Distribution.Pools[0].Stake,
		scenario.Epochs[1].Distribution.Pools[0].Stake,
	)
}
