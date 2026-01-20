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

package ledger

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"slices"

	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

// AdaPotsBuilder defines an interface for building mock AdaPots
type AdaPotsBuilder interface {
	WithReserves(lovelace uint64) AdaPotsBuilder
	WithTreasury(lovelace uint64) AdaPotsBuilder
	WithRewards(lovelace uint64) AdaPotsBuilder
	Build() (*lcommon.AdaPots, error)
}

// MockAdaPots holds the state for building AdaPots
type MockAdaPots struct {
	reserves uint64
	treasury uint64
	rewards  uint64
}

// NewAdaPotsBuilder creates a new MockAdaPots builder
func NewAdaPotsBuilder() *MockAdaPots {
	return &MockAdaPots{}
}

// WithReserves sets the reserves pot
func (m *MockAdaPots) WithReserves(lovelace uint64) AdaPotsBuilder {
	m.reserves = lovelace
	return m
}

// WithTreasury sets the treasury pot
func (m *MockAdaPots) WithTreasury(lovelace uint64) AdaPotsBuilder {
	m.treasury = lovelace
	return m
}

// WithRewards sets the rewards pot
func (m *MockAdaPots) WithRewards(lovelace uint64) AdaPotsBuilder {
	m.rewards = lovelace
	return m
}

// Build constructs an AdaPots from the builder state
func (m *MockAdaPots) Build() (*lcommon.AdaPots, error) {
	pots := &lcommon.AdaPots{
		Reserves: m.reserves,
		Treasury: m.treasury,
		Rewards:  m.rewards,
	}
	return pots, nil
}

// RewardSnapshotBuilder defines an interface for building mock RewardSnapshots
type RewardSnapshotBuilder interface {
	WithTotalActiveStake(stake uint64) RewardSnapshotBuilder
	WithPoolStake(pool lcommon.PoolKeyHash, stake uint64) RewardSnapshotBuilder
	WithDelegatorStake(delegator []byte, stake uint64) RewardSnapshotBuilder
	WithDelegatorStakeForPool(
		pool lcommon.PoolKeyHash,
		delegator []byte,
		stake uint64,
	) RewardSnapshotBuilder
	WithPoolParams(
		pool lcommon.PoolKeyHash,
		cert *lcommon.PoolRegistrationCertificate,
	) RewardSnapshotBuilder
	WithPoolBlocks(pool lcommon.PoolKeyHash, blocks uint64) RewardSnapshotBuilder
	Build() (*lcommon.RewardSnapshot, error)
}

// MockRewardSnapshot holds the state for building a RewardSnapshot
type MockRewardSnapshot struct {
	totalActiveStake uint64
	poolStake        map[lcommon.PoolKeyHash]uint64
	delegatorStake   map[lcommon.PoolKeyHash]map[lcommon.AddrKeyHash]uint64
	poolParams       map[lcommon.PoolKeyHash]*lcommon.PoolRegistrationCertificate
	poolBlocks       map[lcommon.PoolKeyHash]uint32
	// Track which pool a delegator belongs to for delegator stake
	delegatorPools    map[lcommon.AddrKeyHash]lcommon.PoolKeyHash
	poolBlocksErr     error // tracks if pool blocks exceeded uint32
	delegatorStakeErr error // tracks if delegator stake was called with no pools
}

// NewRewardSnapshotBuilder creates a new MockRewardSnapshot builder
func NewRewardSnapshotBuilder() *MockRewardSnapshot {
	return &MockRewardSnapshot{
		poolStake: make(map[lcommon.PoolKeyHash]uint64),
		delegatorStake: make(
			map[lcommon.PoolKeyHash]map[lcommon.AddrKeyHash]uint64,
		),
		poolParams: make(
			map[lcommon.PoolKeyHash]*lcommon.PoolRegistrationCertificate,
		),
		poolBlocks:     make(map[lcommon.PoolKeyHash]uint32),
		delegatorPools: make(map[lcommon.AddrKeyHash]lcommon.PoolKeyHash),
	}
}

// WithTotalActiveStake sets the total active stake in the system
func (m *MockRewardSnapshot) WithTotalActiveStake(
	stake uint64,
) RewardSnapshotBuilder {
	m.totalActiveStake = stake
	return m
}

// WithPoolStake sets the stake for a specific pool
func (m *MockRewardSnapshot) WithPoolStake(
	pool lcommon.PoolKeyHash,
	stake uint64,
) RewardSnapshotBuilder {
	m.poolStake[pool] = stake
	// Clear any previous delegatorStakeErr since pools now exist
	m.delegatorStakeErr = nil
	return m
}

// WithDelegatorStake sets the stake for a delegator
// This selects the lexicographically smallest pool key from the existing pool
// stake map for deterministic behavior. For explicit pool selection, use
// WithDelegatorStakeForPool instead. If no pools exist, Build() will return
// an error.
func (m *MockRewardSnapshot) WithDelegatorStake(
	delegator []byte,
	stake uint64,
) RewardSnapshotBuilder {
	// Fail-fast if no pools exist
	if len(m.poolStake) == 0 {
		m.delegatorStakeErr = errors.New(
			"WithDelegatorStake called but no pools exist; call WithPoolStake first",
		)
		return m
	}

	delegatorKey := lcommon.NewBlake2b224(delegator)

	// Collect pool keys and sort for deterministic selection
	pools := make([]lcommon.PoolKeyHash, 0, len(m.poolStake))
	for p := range m.poolStake {
		pools = append(pools, p)
	}
	slices.SortFunc(pools, func(a, b lcommon.PoolKeyHash) int {
		return bytes.Compare(a.Bytes(), b.Bytes())
	})
	// Select the lexicographically smallest pool
	pool := pools[0]

	if m.delegatorStake[pool] == nil {
		m.delegatorStake[pool] = make(map[lcommon.AddrKeyHash]uint64)
	}
	m.delegatorStake[pool][delegatorKey] = stake
	m.delegatorPools[delegatorKey] = pool
	return m
}

// WithDelegatorStakeForPool sets the stake for a delegator associated with a specific pool
func (m *MockRewardSnapshot) WithDelegatorStakeForPool(
	pool lcommon.PoolKeyHash,
	delegator []byte,
	stake uint64,
) RewardSnapshotBuilder {
	delegatorKey := lcommon.NewBlake2b224(delegator)

	if m.delegatorStake[pool] == nil {
		m.delegatorStake[pool] = make(map[lcommon.AddrKeyHash]uint64)
	}
	m.delegatorStake[pool][delegatorKey] = stake
	m.delegatorPools[delegatorKey] = pool
	return m
}

// WithPoolParams sets the pool parameters for a specific pool
func (m *MockRewardSnapshot) WithPoolParams(
	pool lcommon.PoolKeyHash,
	cert *lcommon.PoolRegistrationCertificate,
) RewardSnapshotBuilder {
	m.poolParams[pool] = cert
	return m
}

// WithPoolBlocks sets the number of blocks produced by a pool
// If blocks exceeds uint32 max value, Build() will return an error
func (m *MockRewardSnapshot) WithPoolBlocks(
	pool lcommon.PoolKeyHash,
	blocks uint64,
) RewardSnapshotBuilder {
	if blocks > math.MaxUint32 {
		m.poolBlocksErr = fmt.Errorf(
			"blocks %d exceeds maximum uint32 value",
			blocks,
		)
		return m
	}
	m.poolBlocks[pool] = uint32(blocks)
	return m
}

// Build constructs a RewardSnapshot from the builder state
func (m *MockRewardSnapshot) Build() (*lcommon.RewardSnapshot, error) {
	// Check for validation errors
	if m.poolBlocksErr != nil {
		return nil, m.poolBlocksErr
	}
	if m.delegatorStakeErr != nil {
		return nil, m.delegatorStakeErr
	}

	// Calculate total blocks in uint64 to detect overflow
	var totalBlocksSum uint64
	for _, blocks := range m.poolBlocks {
		totalBlocksSum += uint64(blocks)
	}
	if totalBlocksSum > math.MaxUint32 {
		return nil, fmt.Errorf(
			"total blocks %d exceeds maximum uint32 value",
			totalBlocksSum,
		)
	}
	totalBlocks := uint32(totalBlocksSum)

	snapshot := &lcommon.RewardSnapshot{
		TotalActiveStake:     m.totalActiveStake,
		PoolStake:            m.poolStake,
		DelegatorStake:       m.delegatorStake,
		PoolParams:           m.poolParams,
		StakeRegistrations:   make(map[lcommon.AddrKeyHash]bool),
		PoolBlocks:           m.poolBlocks,
		TotalBlocksInEpoch:   totalBlocks,
		EarlyDeregistrations: make(map[lcommon.AddrKeyHash]uint64),
		LateDeregistrations:  make(map[lcommon.AddrKeyHash]uint64),
		RetiredPools: make(
			map[lcommon.PoolKeyHash]lcommon.PoolRetirementInfo,
		),
		StakeKeyPoolAssociations: make(
			map[lcommon.AddrKeyHash][]lcommon.PoolKeyHash,
		),
	}

	// Mark all delegators as registered
	for _, delegators := range m.delegatorStake {
		for delegator := range delegators {
			snapshot.StakeRegistrations[delegator] = true
		}
	}

	// Populate stake key pool associations from delegator pools
	for delegator, pool := range m.delegatorPools {
		snapshot.StakeKeyPoolAssociations[delegator] = []lcommon.PoolKeyHash{
			pool,
		}
	}

	return snapshot, nil
}
