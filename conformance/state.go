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

// Package conformance provides state management interfaces for conformance testing.
package conformance

import (
	"maps"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// StateProvider combines all gouroboros state interfaces needed for validation.
// Implementations provide read-only access to ledger state.
type StateProvider interface {
	common.LedgerState
	common.UtxoState
	common.CertState
	common.SlotState
	common.PoolState
	common.RewardState
	common.GovState
}

// StateManager handles state mutations during test execution.
// Implementations manage the full lifecycle of ledger state for a test vector.
type StateManager interface {
	// LoadInitialState loads the initial state from a parsed test vector.
	// This should initialize UTxOs, governance state, registrations, etc.
	LoadInitialState(
		state *ParsedInitialState,
		pp common.ProtocolParameters,
	) error

	// ApplyTransaction updates state after a successful transaction.
	// This should handle UTxO consumption/production and certificate processing.
	ApplyTransaction(tx common.Transaction, slot uint64) error

	// ProcessEpochBoundary handles epoch transitions.
	// This should process pool retirements and proposal ratification.
	ProcessEpochBoundary(newEpoch uint64) error

	// GetStateProvider returns the current state for validation.
	// The returned provider should reflect all applied state changes.
	GetStateProvider() StateProvider

	// GetGovernanceState returns the current governance state for validation.
	GetGovernanceState() *GovernanceState

	// SetRewardBalances sets the reward account balances.
	// Used by the harness to provide adjusted balances for withdrawal validation.
	SetRewardBalances(balances map[common.Blake2b224]uint64)

	// GetProtocolParameters returns the current protocol parameters.
	// These may be updated when ParameterChange proposals are enacted.
	GetProtocolParameters() common.ProtocolParameters

	// Reset clears all state for the next test vector.
	Reset() error
}

// GovernanceState tracks governance-related state during test execution.
type GovernanceState struct {
	// CurrentEpoch is the current epoch number.
	CurrentEpoch uint64

	// CommitteeMembers maps cold key hash to committee member info.
	CommitteeMembers map[common.Blake2b224]*CommitteeMemberInfo

	// DRepRegistrations tracks registered DReps.
	DRepRegistrations map[common.Blake2b224]bool

	// HotKeyAuthorizations maps cold keys to hot keys for committee members.
	HotKeyAuthorizations map[common.Blake2b224]common.Blake2b224

	// StakeRegistrations tracks which stake credentials are registered.
	StakeRegistrations map[common.Blake2b224]bool

	// PoolRegistrations tracks which pools are registered.
	PoolRegistrations map[common.Blake2b224]bool

	// PoolRetirements tracks scheduled pool retirements (pool -> retirement epoch).
	PoolRetirements map[common.Blake2b224]uint64

	// RewardAccounts maps stake credentials to their reward balances.
	RewardAccounts map[common.Blake2b224]uint64

	// Proposals maps GovActionId (as "txHash#index") to proposal info.
	Proposals map[string]*ProposalState

	// EnactedProposals tracks which proposals have been enacted.
	EnactedProposals map[string]bool

	// Roots tracks the last enacted proposal for each governance purpose.
	Roots ProposalRoots

	// Constitution contains the current constitution.
	Constitution *ConstitutionInfo
}

// CommitteeMemberInfo contains committee member details.
type CommitteeMemberInfo struct {
	ColdKey     common.Blake2b224
	HotKey      *common.Blake2b224
	ExpiryEpoch uint64
	Resigned    bool
}

// ProposalState contains the full state of a governance proposal.
type ProposalState struct {
	GovActionInfo

	// RatifiedEpoch is set when the proposal is ratified.
	RatifiedEpoch *uint64
}

// NewGovernanceState creates a new empty governance state.
func NewGovernanceState() *GovernanceState {
	return &GovernanceState{
		CommitteeMembers:     make(map[common.Blake2b224]*CommitteeMemberInfo),
		DRepRegistrations:    make(map[common.Blake2b224]bool),
		HotKeyAuthorizations: make(map[common.Blake2b224]common.Blake2b224),
		StakeRegistrations:   make(map[common.Blake2b224]bool),
		PoolRegistrations:    make(map[common.Blake2b224]bool),
		PoolRetirements:      make(map[common.Blake2b224]uint64),
		RewardAccounts:       make(map[common.Blake2b224]uint64),
		Proposals:            make(map[string]*ProposalState),
		EnactedProposals:     make(map[string]bool),
	}
}

// LoadFromParsedState loads governance state from a parsed initial state.
func (g *GovernanceState) LoadFromParsedState(state *ParsedInitialState) {
	g.CurrentEpoch = state.CurrentEpoch

	// Reset all mutable state to prevent stale entries from previous loads
	g.CommitteeMembers = make(map[common.Blake2b224]*CommitteeMemberInfo)
	g.HotKeyAuthorizations = make(map[common.Blake2b224]common.Blake2b224)
	g.DRepRegistrations = make(map[common.Blake2b224]bool)
	g.StakeRegistrations = make(map[common.Blake2b224]bool)
	g.PoolRegistrations = make(map[common.Blake2b224]bool)
	g.PoolRetirements = make(map[common.Blake2b224]uint64)
	g.RewardAccounts = make(map[common.Blake2b224]uint64)
	g.Proposals = make(map[string]*ProposalState)
	g.EnactedProposals = make(map[string]bool)
	g.Roots = ProposalRoots{}
	g.Constitution = nil

	// Load committee members
	for coldKey, expiry := range state.CommitteeMembers {
		g.CommitteeMembers[coldKey] = &CommitteeMemberInfo{
			ColdKey:     coldKey,
			ExpiryEpoch: expiry,
		}
	}

	// Load hot key authorizations and link to committee members
	for coldKey, hotKey := range state.HotKeyAuthorizations {
		g.HotKeyAuthorizations[coldKey] = hotKey
		if member, ok := g.CommitteeMembers[coldKey]; ok {
			hk := hotKey
			member.HotKey = &hk
		}
	}

	// Load DRep registrations
	for _, drepHash := range state.DRepRegistrations {
		g.DRepRegistrations[drepHash] = true
	}

	// Load stake registrations
	maps.Copy(g.StakeRegistrations, state.StakeRegistrations)

	// Load pool registrations
	maps.Copy(g.PoolRegistrations, state.PoolRegistrations)

	// Load reward accounts
	maps.Copy(g.RewardAccounts, state.RewardAccounts)

	// Load proposals (preserve RatifiedEpoch from parsed state)
	for id, info := range state.Proposals {
		g.Proposals[id] = &ProposalState{
			GovActionInfo: info,
			RatifiedEpoch: info.RatifiedEpoch,
		}
	}

	// Load proposal roots
	g.Roots = state.ProposalRoots

	// Load constitution
	g.Constitution = state.Constitution
}

// IsStakeRegistered checks if a stake credential is registered.
func (g *GovernanceState) IsStakeRegistered(hash common.Blake2b224) bool {
	return g.StakeRegistrations[hash]
}

// IsDRepRegistered checks if a DRep is registered.
func (g *GovernanceState) IsDRepRegistered(hash common.Blake2b224) bool {
	return g.DRepRegistrations[hash]
}

// IsPoolRegistered checks if a pool is registered.
func (g *GovernanceState) IsPoolRegistered(hash common.Blake2b224) bool {
	return g.PoolRegistrations[hash]
}

// GetCommitteeMember returns a committee member by cold key.
func (g *GovernanceState) GetCommitteeMember(
	coldKey common.Blake2b224,
) *CommitteeMemberInfo {
	return g.CommitteeMembers[coldKey]
}

// IsProposedCommitteeMember checks if a cold key is proposed in any pending UpdateCommittee proposal.
func (g *GovernanceState) IsProposedCommitteeMember(
	coldKey common.Blake2b224,
) bool {
	for _, proposal := range g.Proposals {
		if proposal.ActionType == common.GovActionTypeUpdateCommittee {
			if _, ok := proposal.ProposedMembers[coldKey]; ok {
				return true
			}
		}
	}
	return false
}

// GetProposal returns a proposal by its GovActionId.
func (g *GovernanceState) GetProposal(govActionId string) *ProposalState {
	return g.Proposals[govActionId]
}

// GetRewardBalance returns the reward balance for a stake credential.
func (g *GovernanceState) GetRewardBalance(hash common.Blake2b224) uint64 {
	return g.RewardAccounts[hash]
}

// GetEnactedRoot returns the enacted root for a governance action type.
func (g *GovernanceState) GetEnactedRoot(
	actionType common.GovActionType,
) *string {
	//exhaustive:ignore
	switch actionType {
	case common.GovActionTypeParameterChange:
		return g.Roots.ProtocolParameters
	case common.GovActionTypeHardForkInitiation:
		return g.Roots.HardFork
	case common.GovActionTypeNoConfidence, common.GovActionTypeUpdateCommittee:
		return g.Roots.ConstitutionalCommittee
	case common.GovActionTypeNewConstitution:
		return g.Roots.Constitution
	default:
		return nil
	}
}

// RegisterStake registers a stake credential.
func (g *GovernanceState) RegisterStake(hash common.Blake2b224) {
	g.StakeRegistrations[hash] = true
}

// DeregisterStake deregisters a stake credential.
func (g *GovernanceState) DeregisterStake(hash common.Blake2b224) {
	delete(g.StakeRegistrations, hash)
	delete(g.RewardAccounts, hash)
}

// RegisterDRep registers a DRep.
func (g *GovernanceState) RegisterDRep(hash common.Blake2b224) {
	g.DRepRegistrations[hash] = true
}

// DeregisterDRep deregisters a DRep.
func (g *GovernanceState) DeregisterDRep(hash common.Blake2b224) {
	delete(g.DRepRegistrations, hash)
}

// RegisterPool registers a pool.
// If the pool has a pending retirement, the retirement is cancelled.
// This matches Cardano ledger behavior where re-registration cancels scheduled retirements.
func (g *GovernanceState) RegisterPool(hash common.Blake2b224) {
	g.PoolRegistrations[hash] = true
	// Cancel any pending retirement for this pool
	delete(g.PoolRetirements, hash)
}

// RetirePool schedules a pool retirement.
func (g *GovernanceState) RetirePool(
	hash common.Blake2b224,
	retireEpoch uint64,
) {
	g.PoolRetirements[hash] = retireEpoch
}

// ProcessPoolRetirements processes pool retirements for the given epoch.
func (g *GovernanceState) ProcessPoolRetirements(epoch uint64) {
	for poolKey, retireEpoch := range g.PoolRetirements {
		if epoch >= retireEpoch {
			delete(g.PoolRegistrations, poolKey)
			delete(g.PoolRetirements, poolKey)
		}
	}
}

// AuthorizeHotKey authorizes a hot key for a committee member.
func (g *GovernanceState) AuthorizeHotKey(coldKey, hotKey common.Blake2b224) {
	g.HotKeyAuthorizations[coldKey] = hotKey
	if member, ok := g.CommitteeMembers[coldKey]; ok {
		hk := hotKey
		member.HotKey = &hk
		member.Resigned = false
	}
}

// ResignCommitteeMember marks a committee member as resigned.
func (g *GovernanceState) ResignCommitteeMember(coldKey common.Blake2b224) {
	delete(g.HotKeyAuthorizations, coldKey)
	if member, ok := g.CommitteeMembers[coldKey]; ok {
		member.HotKey = nil
		member.Resigned = true
	}
}

// AddProposal adds a new governance proposal.
func (g *GovernanceState) AddProposal(govActionId string, info GovActionInfo) {
	g.Proposals[govActionId] = &ProposalState{
		GovActionInfo: info,
	}
}
