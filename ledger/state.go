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
	"time"

	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

// ErrNotFound is returned when a requested item is not found
var ErrNotFound = errors.New("ledger: not found")

// PlutusLanguage represents a Plutus language version
type PlutusLanguage uint

const (
	PlutusV1 PlutusLanguage = 1
	PlutusV2 PlutusLanguage = 2
	PlutusV3 PlutusLanguage = 3
)

// CostModel represents a Plutus cost model
type CostModel []int64

// UtxoByIdFunc is a callback for UTxO lookups by transaction input
type UtxoByIdFunc func(lcommon.TransactionInput) (lcommon.Utxo, error)

// StakeRegistrationFunc is a callback for stake registration lookups
type StakeRegistrationFunc func([]byte) ([]lcommon.StakeRegistrationCertificate, error)

// SlotToTimeFunc is a callback for converting slots to time
type SlotToTimeFunc func(uint64) (time.Time, error)

// TimeToSlotFunc is a callback for converting time to slots
type TimeToSlotFunc func(time.Time) (uint64, error)

// PoolCurrentStateFunc is a callback for pool state lookups
type PoolCurrentStateFunc func(lcommon.PoolKeyHash) (*lcommon.PoolRegistrationCertificate, *uint64, error)

// CalculateRewardsFunc is a callback for reward calculation
type CalculateRewardsFunc func(lcommon.AdaPots, lcommon.RewardSnapshot, lcommon.RewardParameters) (*lcommon.RewardCalculationResult, error)

// GetRewardSnapshotFunc is a callback for reward snapshot lookups
type GetRewardSnapshotFunc func(uint64) (lcommon.RewardSnapshot, error)

// CommitteeMemberFunc is a callback for committee member lookups
type CommitteeMemberFunc func(lcommon.Blake2b224) (*CommitteeMember, error)

// DRepRegistrationFunc is a callback for DRep registration lookups
type DRepRegistrationFunc func(lcommon.Blake2b224) (*lcommon.RegistrationDrepCertificate, error)

// ConstitutionFunc is a callback for constitution lookups
type ConstitutionFunc func() (*Constitution, error)

// TreasuryValueFunc is a callback for treasury value lookups
type TreasuryValueFunc func() (uint64, error)

// CostModelsFunc is a callback for cost models lookups
type CostModelsFunc func() map[PlutusLanguage]CostModel

// MockLedgerState implements the ledger.LedgerState interface from gouroboros
// using callback functions for customizable behavior
type MockLedgerState struct {
	// UtxoState callbacks
	UtxoByIdCallback UtxoByIdFunc

	// CertState callbacks
	StakeRegistrationCallback StakeRegistrationFunc

	// SlotState callbacks
	SlotToTimeCallback SlotToTimeFunc
	TimeToSlotCallback TimeToSlotFunc

	// PoolState callbacks
	PoolCurrentStateCallback PoolCurrentStateFunc

	// RewardState callbacks
	CalculateRewardsCallback  CalculateRewardsFunc
	GetRewardSnapshotCallback GetRewardSnapshotFunc

	// GovState callbacks (for future governance queries)
	CommitteeMemberCallback  CommitteeMemberFunc
	DRepRegistrationCallback DRepRegistrationFunc
	ConstitutionCallback     ConstitutionFunc
	TreasuryValueCallback    TreasuryValueFunc
	CostModelsCallback       CostModelsFunc

	// Static fields
	networkId uint
	adaPots   lcommon.AdaPots
}

// NetworkId returns the network identifier
func (ls *MockLedgerState) NetworkId() uint {
	return ls.networkId
}

// UtxoById looks up a UTxO by transaction input
func (ls *MockLedgerState) UtxoById(
	id lcommon.TransactionInput,
) (lcommon.Utxo, error) {
	if ls.UtxoByIdCallback != nil {
		return ls.UtxoByIdCallback(id)
	}
	return lcommon.Utxo{}, ErrNotFound
}

// StakeRegistration looks up stake registrations by staking key
func (ls *MockLedgerState) StakeRegistration(
	stakingKey []byte,
) ([]lcommon.StakeRegistrationCertificate, error) {
	if ls.StakeRegistrationCallback != nil {
		return ls.StakeRegistrationCallback(stakingKey)
	}
	return []lcommon.StakeRegistrationCertificate{}, nil
}

// SlotToTime converts a slot number to a time
func (ls *MockLedgerState) SlotToTime(slot uint64) (time.Time, error) {
	if ls.SlotToTimeCallback != nil {
		return ls.SlotToTimeCallback(slot)
	}
	return time.Time{}, nil
}

// TimeToSlot converts a time to a slot number
func (ls *MockLedgerState) TimeToSlot(t time.Time) (uint64, error) {
	if ls.TimeToSlotCallback != nil {
		return ls.TimeToSlotCallback(t)
	}
	return 0, nil
}

// PoolCurrentState returns the current state of a pool
func (ls *MockLedgerState) PoolCurrentState(
	poolKeyHash lcommon.PoolKeyHash,
) (*lcommon.PoolRegistrationCertificate, *uint64, error) {
	if ls.PoolCurrentStateCallback != nil {
		return ls.PoolCurrentStateCallback(poolKeyHash)
	}
	return nil, nil, nil
}

// CalculateRewards calculates rewards for the given epoch
func (ls *MockLedgerState) CalculateRewards(
	pots lcommon.AdaPots,
	snapshot lcommon.RewardSnapshot,
	params lcommon.RewardParameters,
) (*lcommon.RewardCalculationResult, error) {
	if ls.CalculateRewardsCallback != nil {
		return ls.CalculateRewardsCallback(pots, snapshot, params)
	}
	return lcommon.CalculateRewards(pots, snapshot, params)
}

// GetAdaPots returns the current ADA pots
func (ls *MockLedgerState) GetAdaPots() lcommon.AdaPots {
	return ls.adaPots
}

// UpdateAdaPots updates the ADA pots
func (ls *MockLedgerState) UpdateAdaPots(pots lcommon.AdaPots) error {
	ls.adaPots = pots
	return nil
}

// GetRewardSnapshot returns the stake snapshot for reward calculation
func (ls *MockLedgerState) GetRewardSnapshot(
	epoch uint64,
) (lcommon.RewardSnapshot, error) {
	if ls.GetRewardSnapshotCallback != nil {
		return ls.GetRewardSnapshotCallback(epoch)
	}
	return lcommon.RewardSnapshot{}, nil
}

// CommitteeMember looks up a constitutional committee member by credential hash
func (ls *MockLedgerState) CommitteeMember(
	credHash lcommon.Blake2b224,
) (*CommitteeMember, error) {
	if ls.CommitteeMemberCallback != nil {
		return ls.CommitteeMemberCallback(credHash)
	}
	return nil, ErrNotFound
}

// DRepRegistration looks up a DRep registration by credential hash
func (ls *MockLedgerState) DRepRegistration(
	credHash lcommon.Blake2b224,
) (*lcommon.RegistrationDrepCertificate, error) {
	if ls.DRepRegistrationCallback != nil {
		return ls.DRepRegistrationCallback(credHash)
	}
	return nil, ErrNotFound
}

// Constitution returns the current constitution
func (ls *MockLedgerState) Constitution() (*Constitution, error) {
	if ls.ConstitutionCallback != nil {
		return ls.ConstitutionCallback()
	}
	return nil, ErrNotFound
}

// TreasuryValue returns the current treasury value
func (ls *MockLedgerState) TreasuryValue() (uint64, error) {
	if ls.TreasuryValueCallback != nil {
		return ls.TreasuryValueCallback()
	}
	return ls.adaPots.Treasury, nil
}

// CostModels returns the current cost models for Plutus scripts
func (ls *MockLedgerState) CostModels() map[PlutusLanguage]CostModel {
	if ls.CostModelsCallback != nil {
		return ls.CostModelsCallback()
	}
	return make(map[PlutusLanguage]CostModel)
}

// LedgerStateBuilder provides a fluent API for setting up MockLedgerState
type LedgerStateBuilder struct {
	state *MockLedgerState
}

// NewLedgerStateBuilder creates a new LedgerStateBuilder
func NewLedgerStateBuilder() *LedgerStateBuilder {
	return &LedgerStateBuilder{
		state: &MockLedgerState{},
	}
}

// WithNetworkId sets the network ID
func (b *LedgerStateBuilder) WithNetworkId(networkId uint) *LedgerStateBuilder {
	b.state.networkId = networkId
	return b
}

// WithAdaPots sets the ADA pots
func (b *LedgerStateBuilder) WithAdaPots(
	pots lcommon.AdaPots,
) *LedgerStateBuilder {
	b.state.adaPots = pots
	return b
}

// WithUtxoById sets the UTxO lookup callback
func (b *LedgerStateBuilder) WithUtxoById(fn UtxoByIdFunc) *LedgerStateBuilder {
	b.state.UtxoByIdCallback = fn
	return b
}

// WithStakeRegistration sets the stake registration lookup callback
func (b *LedgerStateBuilder) WithStakeRegistration(
	fn StakeRegistrationFunc,
) *LedgerStateBuilder {
	b.state.StakeRegistrationCallback = fn
	return b
}

// WithSlotToTime sets the slot to time conversion callback
func (b *LedgerStateBuilder) WithSlotToTime(
	fn SlotToTimeFunc,
) *LedgerStateBuilder {
	b.state.SlotToTimeCallback = fn
	return b
}

// WithTimeToSlot sets the time to slot conversion callback
func (b *LedgerStateBuilder) WithTimeToSlot(
	fn TimeToSlotFunc,
) *LedgerStateBuilder {
	b.state.TimeToSlotCallback = fn
	return b
}

// WithPoolCurrentState sets the pool current state callback
func (b *LedgerStateBuilder) WithPoolCurrentState(
	fn PoolCurrentStateFunc,
) *LedgerStateBuilder {
	b.state.PoolCurrentStateCallback = fn
	return b
}

// WithCalculateRewards sets the calculate rewards callback
func (b *LedgerStateBuilder) WithCalculateRewards(
	fn CalculateRewardsFunc,
) *LedgerStateBuilder {
	b.state.CalculateRewardsCallback = fn
	return b
}

// WithGetRewardSnapshot sets the get reward snapshot callback
func (b *LedgerStateBuilder) WithGetRewardSnapshot(
	fn GetRewardSnapshotFunc,
) *LedgerStateBuilder {
	b.state.GetRewardSnapshotCallback = fn
	return b
}

// WithCommitteeMember sets the committee member lookup callback
func (b *LedgerStateBuilder) WithCommitteeMember(
	fn CommitteeMemberFunc,
) *LedgerStateBuilder {
	b.state.CommitteeMemberCallback = fn
	return b
}

// WithDRepRegistration sets the DRep registration lookup callback
func (b *LedgerStateBuilder) WithDRepRegistration(
	fn DRepRegistrationFunc,
) *LedgerStateBuilder {
	b.state.DRepRegistrationCallback = fn
	return b
}

// WithConstitution sets the constitution lookup callback
func (b *LedgerStateBuilder) WithConstitution(
	fn ConstitutionFunc,
) *LedgerStateBuilder {
	b.state.ConstitutionCallback = fn
	return b
}

// WithTreasuryValue sets the treasury value lookup callback
func (b *LedgerStateBuilder) WithTreasuryValue(
	fn TreasuryValueFunc,
) *LedgerStateBuilder {
	b.state.TreasuryValueCallback = fn
	return b
}

// WithCostModels sets the cost models lookup callback
func (b *LedgerStateBuilder) WithCostModels(
	fn CostModelsFunc,
) *LedgerStateBuilder {
	b.state.CostModelsCallback = fn
	return b
}

// WithUtxos configures the mock with a static set of UTxOs for lookup
func (b *LedgerStateBuilder) WithUtxos(
	utxos []lcommon.Utxo,
) *LedgerStateBuilder {
	b.state.UtxoByIdCallback = func(id lcommon.TransactionInput) (lcommon.Utxo, error) {
		// Guard against nil input
		if id == nil {
			return lcommon.Utxo{}, ErrNotFound
		}
		inputId := id.Id()
		if inputId == (lcommon.Blake2b256{}) {
			return lcommon.Utxo{}, ErrNotFound
		}

		for _, utxo := range utxos {
			// Guard against nil utxo.Id
			if utxo.Id == nil {
				continue
			}
			utxoId := utxo.Id.Id()

			// Compare index and ID using bytes.Equal
			if id.Index() != utxo.Id.Index() {
				continue
			}
			if !bytes.Equal(inputId.Bytes(), utxoId.Bytes()) {
				continue
			}
			return utxo, nil
		}
		return lcommon.Utxo{}, ErrNotFound
	}
	return b
}

// WithPools configures the mock with a static set of pool registrations
func (b *LedgerStateBuilder) WithPools(
	pools []*lcommon.PoolRegistrationCertificate,
) *LedgerStateBuilder {
	b.state.PoolCurrentStateCallback = func(poolKeyHash lcommon.PoolKeyHash) (*lcommon.PoolRegistrationCertificate, *uint64, error) {
		for _, pool := range pools {
			// Skip nil pool entries
			if pool == nil {
				continue
			}
			// Compare pool operator hash directly with the lookup key using bytes.Equal
			// (both are already Blake2b224 hashes, no need to hash again)
			if bytes.Equal(pool.Operator.Bytes(), poolKeyHash.Bytes()) {
				return pool, nil, nil
			}
		}
		return nil, nil, nil
	}
	return b
}

// WithStakeRegistrations configures the mock with a static set of stake registrations
func (b *LedgerStateBuilder) WithStakeRegistrations(
	certs []lcommon.StakeRegistrationCertificate,
) *LedgerStateBuilder {
	b.state.StakeRegistrationCallback = func(stakingKey []byte) ([]lcommon.StakeRegistrationCertificate, error) {
		var result []lcommon.StakeRegistrationCertificate
		for _, cert := range certs {
			// Compare credential directly using bytes.Equal
			// (it's already a Blake2b224 hash, no need to hash again)
			if bytes.Equal(
				cert.StakeCredential.Credential.Bytes(),
				stakingKey,
			) {
				result = append(result, cert)
			}
		}
		return result, nil
	}
	return b
}

// Build constructs the MockLedgerState
func (b *LedgerStateBuilder) Build() *MockLedgerState {
	return b.state
}
