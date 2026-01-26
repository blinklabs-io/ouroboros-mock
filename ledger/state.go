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

package ledger

import (
	"bytes"
	"errors"
	"time"

	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	utxorpc "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
)

// ErrNotFound is returned when a requested item is not found
var ErrNotFound = errors.New("ledger: not found")

// Type aliases for convenience - use gouroboros types directly
// Note: CommitteeMember and Constitution are NOT aliased here because they
// are defined as local types in governance.go for the builder pattern.
// The interface methods use lcommon types directly.
type (
	PlutusLanguage   = lcommon.PlutusLanguage
	CostModel        = lcommon.CostModel
	DRepRegistration = lcommon.DRepRegistration
	GovActionState   = lcommon.GovActionState
)

// Plutus language version constants
const (
	PlutusV1 PlutusLanguage = 1
	PlutusV2 PlutusLanguage = 2
	PlutusV3 PlutusLanguage = 3
)

// Callback function types for customizable behavior

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
type CommitteeMemberFunc func(lcommon.Blake2b224) (*lcommon.CommitteeMember, error)

// DRepRegistrationFunc is a callback for DRep registration lookups
type DRepRegistrationFunc func(lcommon.Blake2b224) (*lcommon.DRepRegistration, error)

// ConstitutionFunc is a callback for constitution lookups
type ConstitutionFunc func() (*lcommon.Constitution, error)

// TreasuryValueFunc is a callback for treasury value lookups
type TreasuryValueFunc func() (uint64, error)

// CostModelsFunc is a callback for cost models lookups
type CostModelsFunc func() map[lcommon.PlutusLanguage]lcommon.CostModel

// GovActionByIdFunc is a callback for governance action lookups
type GovActionByIdFunc func(lcommon.GovActionId) (*lcommon.GovActionState, error)

// MockLedgerState implements the ledger.LedgerState interface from gouroboros
// using callback functions for customizable behavior
type MockLedgerState struct {
	// UtxoState callbacks
	UtxoByIdCallback UtxoByIdFunc

	// CertState callbacks and state
	StakeRegistrationCallback StakeRegistrationFunc
	stakeRegistrations        map[lcommon.Blake2b224]bool // credential -> registered

	// SlotState callbacks
	SlotToTimeCallback SlotToTimeFunc
	TimeToSlotCallback TimeToSlotFunc

	// PoolState callbacks and state
	PoolCurrentStateCallback PoolCurrentStateFunc
	poolRegistrations        []lcommon.PoolRegistrationCertificate

	// RewardState callbacks and state
	CalculateRewardsCallback  CalculateRewardsFunc
	GetRewardSnapshotCallback GetRewardSnapshotFunc
	rewardAccounts            map[lcommon.Blake2b224]uint64 // credential -> balance

	// GovState callbacks and state
	CommitteeMemberCallback  CommitteeMemberFunc
	DRepRegistrationCallback DRepRegistrationFunc
	ConstitutionCallback     ConstitutionFunc
	TreasuryValueCallback    TreasuryValueFunc
	GovActionByIdCallback    GovActionByIdFunc
	committeeMembers         []lcommon.CommitteeMember
	drepRegistrations        []lcommon.DRepRegistration
	govActions               map[string]*lcommon.GovActionState // "txhash#index" -> state
	// ProposedCommitteeMembers tracks committee members proposed in pending
	// UpdateCommittee governance actions. Per Cardano ledger spec, AUTH_CC
	// should succeed if the member is either a current member OR proposed
	// in a pending UpdateCommittee action. Maps coldKey -> expiryEpoch.
	proposedCommitteeMembers map[lcommon.Blake2b224]uint64

	// LedgerState fields
	CostModelsCallback CostModelsFunc
	networkId          uint
	adaPots            lcommon.AdaPots
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

// IsStakeCredentialRegistered checks if a stake credential is currently registered
func (ls *MockLedgerState) IsStakeCredentialRegistered(
	cred lcommon.Credential,
) bool {
	if ls.stakeRegistrations == nil {
		return false
	}
	return ls.stakeRegistrations[cred.Credential]
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
	// Search in stored pool registrations
	for i := range ls.poolRegistrations {
		if ls.poolRegistrations[i].Operator == poolKeyHash {
			return &ls.poolRegistrations[i], nil, nil
		}
	}
	return nil, nil, nil
}

// IsPoolRegistered checks if a pool is currently registered
func (ls *MockLedgerState) IsPoolRegistered(
	poolKeyHash lcommon.PoolKeyHash,
) bool {
	// Check callback first
	if ls.PoolCurrentStateCallback != nil {
		cert, _, _ := ls.PoolCurrentStateCallback(poolKeyHash)
		return cert != nil
	}
	// Search in stored pool registrations
	for i := range ls.poolRegistrations {
		if ls.poolRegistrations[i].Operator == poolKeyHash {
			return true
		}
	}
	return false
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

// IsRewardAccountRegistered checks if a reward account is registered
func (ls *MockLedgerState) IsRewardAccountRegistered(
	cred lcommon.Credential,
) bool {
	// Reward account registration is tied to stake credential registration
	return ls.IsStakeCredentialRegistered(cred)
}

// RewardAccountBalance returns the current reward balance for a stake credential
func (ls *MockLedgerState) RewardAccountBalance(
	cred lcommon.Credential,
) (*uint64, error) {
	if ls.rewardAccounts == nil {
		return nil, nil
	}
	balance, exists := ls.rewardAccounts[cred.Credential]
	if !exists {
		return nil, nil
	}
	return &balance, nil
}

// CommitteeMember looks up a constitutional committee member by credential hash.
// Per Cardano ledger spec, AUTH_CC should succeed if the member is either a
// current committee member OR proposed in a pending UpdateCommittee action.
func (ls *MockLedgerState) CommitteeMember(
	coldKey lcommon.Blake2b224,
) (*lcommon.CommitteeMember, error) {
	if ls.CommitteeMemberCallback != nil {
		return ls.CommitteeMemberCallback(coldKey)
	}
	// Search in stored committee members
	for i := range ls.committeeMembers {
		if ls.committeeMembers[i].ColdKey == coldKey {
			return &ls.committeeMembers[i], nil
		}
	}
	// Also check proposed members from pending UpdateCommittee proposals
	if ls.proposedCommitteeMembers != nil {
		if expiryEpoch, ok := ls.proposedCommitteeMembers[coldKey]; ok {
			return &lcommon.CommitteeMember{
				ColdKey:     coldKey,
				HotKey:      nil,
				ExpiryEpoch: expiryEpoch,
				Resigned:    false,
			}, nil
		}
	}
	return nil, nil
}

// CommitteeMembers returns all committee members
func (ls *MockLedgerState) CommitteeMembers() ([]lcommon.CommitteeMember, error) {
	return ls.committeeMembers, nil
}

// DRepRegistration looks up a DRep registration by credential hash
func (ls *MockLedgerState) DRepRegistration(
	credential lcommon.Blake2b224,
) (*lcommon.DRepRegistration, error) {
	if ls.DRepRegistrationCallback != nil {
		return ls.DRepRegistrationCallback(credential)
	}
	// Search in stored DRep registrations
	for i := range ls.drepRegistrations {
		if ls.drepRegistrations[i].Credential == credential {
			return &ls.drepRegistrations[i], nil
		}
	}
	return nil, nil
}

// DRepRegistrations returns all DRep registrations
func (ls *MockLedgerState) DRepRegistrations() ([]lcommon.DRepRegistration, error) {
	return ls.drepRegistrations, nil
}

// Constitution returns the current constitution
func (ls *MockLedgerState) Constitution() (*lcommon.Constitution, error) {
	if ls.ConstitutionCallback != nil {
		return ls.ConstitutionCallback()
	}
	return nil, nil
}

// TreasuryValue returns the current treasury value
func (ls *MockLedgerState) TreasuryValue() (uint64, error) {
	if ls.TreasuryValueCallback != nil {
		return ls.TreasuryValueCallback()
	}
	return ls.adaPots.Treasury, nil
}

// GovActionById looks up a governance action by its ID
func (ls *MockLedgerState) GovActionById(
	id lcommon.GovActionId,
) (*lcommon.GovActionState, error) {
	if ls.GovActionByIdCallback != nil {
		return ls.GovActionByIdCallback(id)
	}
	if ls.govActions == nil {
		return nil, nil
	}
	// Key format matches gouroboros internal mock: "txhash#index"
	key := id.String()
	state, exists := ls.govActions[key]
	if !exists {
		return nil, nil
	}
	return state, nil
}

// GovActionExists checks if a governance action exists
func (ls *MockLedgerState) GovActionExists(id lcommon.GovActionId) bool {
	state, _ := ls.GovActionById(id)
	return state != nil
}

// CostModels returns the current cost models for Plutus scripts
func (ls *MockLedgerState) CostModels() map[lcommon.PlutusLanguage]lcommon.CostModel {
	if ls.CostModelsCallback != nil {
		return ls.CostModelsCallback()
	}
	return make(map[lcommon.PlutusLanguage]lcommon.CostModel)
}

// MockProtocolParamsRules is a simple protocol params provider used in tests.
// Utxorpc() returns a zero-value struct to prevent nil pointer dereferences.
type MockProtocolParamsRules struct {
	PParams *utxorpc.PParams
}

// Utxorpc returns the protocol parameters in UTxO RPC format
func (m *MockProtocolParamsRules) Utxorpc() (*utxorpc.PParams, error) {
	if m.PParams != nil {
		return m.PParams, nil
	}
	return &utxorpc.PParams{}, nil
}

// LedgerStateBuilder provides a fluent API for setting up MockLedgerState
type LedgerStateBuilder struct {
	state *MockLedgerState
}

// NewLedgerStateBuilder creates a new LedgerStateBuilder
func NewLedgerStateBuilder() *LedgerStateBuilder {
	return &LedgerStateBuilder{
		state: &MockLedgerState{
			stakeRegistrations:       make(map[lcommon.Blake2b224]bool),
			rewardAccounts:           make(map[lcommon.Blake2b224]uint64),
			govActions:               make(map[string]*lcommon.GovActionState),
			proposedCommitteeMembers: make(map[lcommon.Blake2b224]uint64),
		},
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

// WithStakeCredentialRegistered sets whether a stake credential is registered
func (b *LedgerStateBuilder) WithStakeCredentialRegistered(
	cred lcommon.Blake2b224,
	registered bool,
) *LedgerStateBuilder {
	b.state.stakeRegistrations[cred] = registered
	return b
}

// WithStakeCredentials sets multiple stake credential registrations
func (b *LedgerStateBuilder) WithStakeCredentials(
	creds map[lcommon.Blake2b224]bool,
) *LedgerStateBuilder {
	for cred, registered := range creds {
		b.state.stakeRegistrations[cred] = registered
	}
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

// WithPoolRegistrations sets the pool registrations
func (b *LedgerStateBuilder) WithPoolRegistrations(
	pools []lcommon.PoolRegistrationCertificate,
) *LedgerStateBuilder {
	b.state.poolRegistrations = pools
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

// WithRewardAccountBalance sets the balance for a reward account
func (b *LedgerStateBuilder) WithRewardAccountBalance(
	cred lcommon.Blake2b224,
	balance uint64,
) *LedgerStateBuilder {
	b.state.rewardAccounts[cred] = balance
	// Also mark the stake credential as registered
	b.state.stakeRegistrations[cred] = true
	return b
}

// WithRewardAccounts sets multiple reward account balances
func (b *LedgerStateBuilder) WithRewardAccounts(
	accounts map[lcommon.Blake2b224]uint64,
) *LedgerStateBuilder {
	for cred, balance := range accounts {
		b.state.rewardAccounts[cred] = balance
		b.state.stakeRegistrations[cred] = true
	}
	return b
}

// WithCommitteeMember sets the committee member lookup callback
func (b *LedgerStateBuilder) WithCommitteeMember(
	fn CommitteeMemberFunc,
) *LedgerStateBuilder {
	b.state.CommitteeMemberCallback = fn
	return b
}

// WithCommitteeMembers sets the committee members
func (b *LedgerStateBuilder) WithCommitteeMembers(
	members []lcommon.CommitteeMember,
) *LedgerStateBuilder {
	b.state.committeeMembers = members
	return b
}

// WithProposedCommitteeMembers sets the proposed committee members from pending
// UpdateCommittee governance actions. Per Cardano ledger spec, AUTH_CC should
// succeed if the member is either a current member OR proposed in a pending
// UpdateCommittee action. The map keys are cold key hashes and values are
// expiry epochs.
func (b *LedgerStateBuilder) WithProposedCommitteeMembers(
	members map[lcommon.Blake2b224]uint64,
) *LedgerStateBuilder {
	b.state.proposedCommitteeMembers = members
	return b
}

// WithDRepRegistration sets the DRep registration lookup callback
func (b *LedgerStateBuilder) WithDRepRegistration(
	fn DRepRegistrationFunc,
) *LedgerStateBuilder {
	b.state.DRepRegistrationCallback = fn
	return b
}

// WithDRepRegistrations sets the DRep registrations
func (b *LedgerStateBuilder) WithDRepRegistrations(
	dreps []lcommon.DRepRegistration,
) *LedgerStateBuilder {
	b.state.drepRegistrations = dreps
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

// WithGovActionById sets the governance action lookup callback
func (b *LedgerStateBuilder) WithGovActionById(
	fn GovActionByIdFunc,
) *LedgerStateBuilder {
	b.state.GovActionByIdCallback = fn
	return b
}

// WithGovActions sets the governance actions
func (b *LedgerStateBuilder) WithGovActions(
	actions map[string]*lcommon.GovActionState,
) *LedgerStateBuilder {
	b.state.govActions = actions
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

// WithPools configures the mock with a static set of pool registrations (pointer version)
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
			if bytes.Equal(pool.Operator.Bytes(), poolKeyHash.Bytes()) {
				return pool, nil, nil
			}
		}
		return nil, nil, nil
	}
	// Also store for IsPoolRegistered
	for _, pool := range pools {
		if pool != nil {
			b.state.poolRegistrations = append(b.state.poolRegistrations, *pool)
		}
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
			if bytes.Equal(
				cert.StakeCredential.Credential.Bytes(),
				stakingKey,
			) {
				result = append(result, cert)
			}
		}
		return result, nil
	}
	// Also mark credentials as registered for IsStakeCredentialRegistered
	for _, cert := range certs {
		b.state.stakeRegistrations[cert.StakeCredential.Credential] = true
	}
	return b
}

// Build constructs the MockLedgerState
func (b *LedgerStateBuilder) Build() *MockLedgerState {
	return b.state
}

// NewMockLedgerStateWithUtxos creates a MockLedgerState with lookup behavior for provided UTxOs.
// This helper matches the gouroboros internal mock API.
func NewMockLedgerStateWithUtxos(utxos []lcommon.Utxo) *MockLedgerState {
	return NewLedgerStateBuilder().WithUtxos(utxos).Build()
}
