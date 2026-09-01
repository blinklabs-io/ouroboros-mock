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
	"github.com/blinklabs-io/ouroboros-mock/ledger"
)

// StateProvider is the read-only ledger state surface required for transaction
// validation. It is defined in the ledger package and aliased here so that
// existing code that references conformance.StateProvider continues to compile
// unchanged. Implementors should refer to ledger.StateProvider directly.
type StateProvider = ledger.StateProvider

// RewardAccountBalanceSetter is an optional StateManager extension for
// preserving the full reward-account credential identity. The harness falls
// back to StateManager.SetRewardBalances for implementations that have not yet
// adopted it.
type RewardAccountBalanceSetter interface {
	// SetRewardAccountBalances updates balances for accounts already registered
	// by the state manager. It must not create or remove registrations.
	SetRewardAccountBalances(balances map[ledger.RewardAccountKey]uint64)
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

	// CommitteeMembersByCredential preserves the full cold credential identity.
	CommitteeMembersByCredential map[ledger.RewardAccountKey]*CommitteeMemberInfo

	// DRepRegistrations tracks registered DReps.
	DRepRegistrations map[common.Blake2b224]bool

	DRepRegistrationsByCredential map[ledger.RewardAccountKey]bool
	DRepExpiries                  map[ledger.RewardAccountKey]uint64

	// DRepDelegations maps stake credentials to their delegated DRep, including
	// the special always-abstain and always-no-confidence DReps.
	// Deprecated: use DRepDelegationsByCredential when credential type matters.
	DRepDelegations map[common.Blake2b224]common.Drep

	// DRepDelegationsByCredential maps full stake credential identities to
	// their delegated DRep.
	DRepDelegationsByCredential map[ledger.RewardAccountKey]common.Drep

	// HotKeyAuthorizations maps cold keys to hot keys for committee members.
	HotKeyAuthorizations map[common.Blake2b224]common.Blake2b224

	// HotKeyAuthorizationsByCredential preserves both cold and hot credential
	// types.
	HotKeyAuthorizationsByCredential map[ledger.RewardAccountKey]common.Credential

	// CommitteeResignations tracks current and pending-proposal members that
	// resigned and therefore cannot authorize a hot key again.
	CommitteeResignations map[ledger.RewardAccountKey]bool

	// StakeRegistrations tracks which stake credentials are registered.
	// Deprecated: use StakeRegistrationsByCredential when credential type
	// matters.
	StakeRegistrations map[common.Blake2b224]bool

	// StakeRegistrationsByCredential tracks registrations by full stake
	// credential identity.
	StakeRegistrationsByCredential map[ledger.RewardAccountKey]bool

	// PoolRegistrations tracks which pools are registered.
	PoolRegistrations map[common.Blake2b224]bool

	PoolRewardAccounts          map[common.PoolKeyHash]ledger.RewardAccountKey
	PoolDelegationsByCredential map[ledger.RewardAccountKey]common.PoolKeyHash

	// PoolRetirements tracks scheduled pool retirements (pool -> retirement epoch).
	PoolRetirements map[common.Blake2b224]uint64

	// RewardAccounts maps stake credentials to their reward balances.
	// Deprecated: use RewardAccountBalances when credential type matters.
	RewardAccounts map[common.Blake2b224]uint64

	// RewardAccountBalances maps full credential identities to reward balances.
	RewardAccountBalances map[ledger.RewardAccountKey]uint64

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
	ColdCredential common.Credential
	HotCredential  *common.Credential
	ColdKey        common.Blake2b224
	HotKey         *common.Blake2b224
	ExpiryEpoch    uint64
	Resigned       bool
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
		CommitteeMembers: make(map[common.Blake2b224]*CommitteeMemberInfo),
		CommitteeMembersByCredential: make(
			map[ledger.RewardAccountKey]*CommitteeMemberInfo,
		),
		DRepRegistrations:             make(map[common.Blake2b224]bool),
		DRepRegistrationsByCredential: make(map[ledger.RewardAccountKey]bool),
		DRepExpiries:                  make(map[ledger.RewardAccountKey]uint64),
		DRepDelegations:               make(map[common.Blake2b224]common.Drep),
		DRepDelegationsByCredential: make(
			map[ledger.RewardAccountKey]common.Drep,
		),
		HotKeyAuthorizations: make(map[common.Blake2b224]common.Blake2b224),
		HotKeyAuthorizationsByCredential: make(
			map[ledger.RewardAccountKey]common.Credential,
		),
		CommitteeResignations: make(map[ledger.RewardAccountKey]bool),
		StakeRegistrations:    make(map[common.Blake2b224]bool),
		StakeRegistrationsByCredential: make(
			map[ledger.RewardAccountKey]bool,
		),
		PoolRegistrations: make(map[common.Blake2b224]bool),
		PoolRewardAccounts: make(
			map[common.PoolKeyHash]ledger.RewardAccountKey,
		),
		PoolDelegationsByCredential: make(
			map[ledger.RewardAccountKey]common.PoolKeyHash,
		),
		PoolRetirements:       make(map[common.Blake2b224]uint64),
		RewardAccounts:        make(map[common.Blake2b224]uint64),
		RewardAccountBalances: make(map[ledger.RewardAccountKey]uint64),
		Proposals:             make(map[string]*ProposalState),
		EnactedProposals:      make(map[string]bool),
	}
}

func hasCredentialHash[V any](
	values map[ledger.RewardAccountKey]V,
	hash common.Blake2b224,
) bool {
	for credential := range values {
		if credential.Credential == hash {
			return true
		}
	}
	return false
}

// LoadFromParsedState loads governance state from a parsed initial state.
func (g *GovernanceState) LoadFromParsedState(state *ParsedInitialState) {
	g.CurrentEpoch = state.CurrentEpoch

	// Reset all mutable state to prevent stale entries from previous loads
	g.CommitteeMembers = make(map[common.Blake2b224]*CommitteeMemberInfo)
	g.CommitteeMembersByCredential = make(
		map[ledger.RewardAccountKey]*CommitteeMemberInfo,
	)
	g.HotKeyAuthorizations = make(map[common.Blake2b224]common.Blake2b224)
	g.HotKeyAuthorizationsByCredential = make(
		map[ledger.RewardAccountKey]common.Credential,
	)
	g.CommitteeResignations = make(map[ledger.RewardAccountKey]bool)
	g.DRepRegistrations = make(map[common.Blake2b224]bool)
	g.DRepRegistrationsByCredential = make(map[ledger.RewardAccountKey]bool)
	g.DRepExpiries = make(map[ledger.RewardAccountKey]uint64)
	g.DRepDelegations = make(map[common.Blake2b224]common.Drep)
	g.DRepDelegationsByCredential = make(
		map[ledger.RewardAccountKey]common.Drep,
	)
	g.StakeRegistrations = make(map[common.Blake2b224]bool)
	g.StakeRegistrationsByCredential = make(
		map[ledger.RewardAccountKey]bool,
	)
	g.PoolRegistrations = make(map[common.Blake2b224]bool)
	g.PoolRewardAccounts = make(map[common.PoolKeyHash]ledger.RewardAccountKey)
	g.PoolDelegationsByCredential = make(
		map[ledger.RewardAccountKey]common.PoolKeyHash,
	)
	g.PoolRetirements = make(map[common.Blake2b224]uint64)
	g.RewardAccounts = make(map[common.Blake2b224]uint64)
	g.RewardAccountBalances = make(map[ledger.RewardAccountKey]uint64)
	g.Proposals = make(map[string]*ProposalState)
	g.EnactedProposals = make(map[string]bool)
	g.Roots = ProposalRoots{}
	g.Constitution = nil

	// Load committee members
	committeeMembers := maps.Clone(state.CommitteeMembersByCredential)
	if committeeMembers == nil {
		committeeMembers = make(map[ledger.RewardAccountKey]uint64)
	}
	for coldKey, expiry := range state.CommitteeMembers {
		if hasCredentialHash(committeeMembers, coldKey) {
			continue
		}
		committeeMembers[ledger.RewardAccountKey{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: coldKey,
		}] = expiry
	}
	for coldKey, expiry := range committeeMembers {
		member := &CommitteeMemberInfo{
			ColdCredential: coldKey.AsCredential(),
			ColdKey:        coldKey.Credential,
			ExpiryEpoch:    expiry,
		}
		g.CommitteeMembersByCredential[coldKey] = member
	}
	g.syncLegacyCommitteeMembers()

	// Load hot key authorizations and link to committee members
	hotKeyAuthorizations := maps.Clone(state.HotKeyAuthorizationsByCredential)
	if hotKeyAuthorizations == nil {
		hotKeyAuthorizations = make(
			map[ledger.RewardAccountKey]common.Credential,
		)
	}
	for coldKey, hotKey := range state.HotKeyAuthorizations {
		if hasCredentialHash(hotKeyAuthorizations, coldKey) {
			continue
		}
		hotKeyAuthorizations[ledger.RewardAccountKey{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: coldKey,
		}] = common.Credential{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: hotKey,
		}
	}
	for coldKey, hotKey := range hotKeyAuthorizations {
		g.HotKeyAuthorizationsByCredential[coldKey] = hotKey
		if member, ok := g.CommitteeMembersByCredential[coldKey]; ok {
			hotCredential := hotKey
			member.HotCredential = &hotCredential
			hotHash := hotKey.Credential
			member.HotKey = &hotHash
		}
	}
	for coldKey, resigned := range state.CommitteeResignations {
		if !resigned {
			continue
		}
		g.CommitteeResignations[coldKey] = true
		delete(g.HotKeyAuthorizationsByCredential, coldKey)
		delete(g.HotKeyAuthorizations, coldKey.Credential)
		if member, ok := g.CommitteeMembersByCredential[coldKey]; ok {
			member.HotCredential = nil
			member.HotKey = nil
			member.Resigned = true
		}
	}
	g.syncLegacyHotKeyAuthorizations()

	// Load DRep registrations
	for _, drepHash := range state.DRepRegistrations {
		g.DRepRegistrations[drepHash] = true
	}
	maps.Copy(
		g.DRepRegistrationsByCredential,
		state.DRepRegistrationsByCredential,
	)
	maps.Copy(g.DRepExpiries, state.DRepExpiries)
	if len(state.DRepDelegationsByCredential) > 0 {
		maps.Copy(
			g.DRepDelegationsByCredential,
			state.DRepDelegationsByCredential,
		)
	} else {
		for credential, delegation := range state.DRepDelegations {
			g.DRepDelegationsByCredential[ledger.RewardAccountKey{
				CredType:   common.CredentialTypeAddrKeyHash,
				Credential: credential,
			}] = delegation
		}
	}
	g.DRepDelegations = drepDelegationsByHash(
		g.DRepDelegationsByCredential,
	)

	// Load stake registrations
	maps.Copy(g.StakeRegistrations, state.StakeRegistrations)
	if len(state.StakeRegistrationsByCredential) > 0 {
		maps.Copy(
			g.StakeRegistrationsByCredential,
			state.StakeRegistrationsByCredential,
		)
	} else if len(state.RewardAccountBalances) > 0 {
		for credential := range state.RewardAccountBalances {
			g.StakeRegistrationsByCredential[credential] = true
		}
	} else {
		for hash, registered := range state.StakeRegistrations {
			if !registered {
				continue
			}
			g.StakeRegistrationsByCredential[ledger.RewardAccountKey{
				CredType:   common.CredentialTypeAddrKeyHash,
				Credential: hash,
			}] = true
		}
	}

	// Load pool registrations
	maps.Copy(g.PoolRegistrations, state.PoolRegistrations)
	maps.Copy(g.PoolRewardAccounts, state.PoolRewardAccounts)
	maps.Copy(
		g.PoolDelegationsByCredential,
		state.PoolDelegationsByCredential,
	)

	// Load reward accounts
	maps.Copy(g.RewardAccounts, state.RewardAccounts)
	maps.Copy(g.RewardAccountBalances, state.RewardAccountBalances)

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

// IsStakeCredentialRegistered checks registration by full credential identity.
func (g *GovernanceState) IsStakeCredentialRegistered(
	credential common.Credential,
) bool {
	if len(g.StakeRegistrationsByCredential) > 0 {
		return g.StakeRegistrationsByCredential[ledger.NewRewardAccountKey(credential)]
	}
	return g.IsStakeRegistered(credential.Credential)
}

// IsDRepRegistered checks if a DRep is registered.
func (g *GovernanceState) IsDRepRegistered(hash common.Blake2b224) bool {
	return g.DRepRegistrations[hash]
}

// IsPoolRegistered checks if a pool is registered.
func (g *GovernanceState) IsPoolRegistered(hash common.Blake2b224) bool {
	return g.PoolRegistrations[hash]
}

// GetCommitteeMember returns a committee member by cold key hash. When key and
// script credentials with the same hash are both present, it returns nil rather
// than guessing which credential was meant.
func (g *GovernanceState) GetCommitteeMember(
	coldKey common.Blake2b224,
) *CommitteeMemberInfo {
	keyMember := g.GetCommitteeCredentialMember(common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldKey,
	})
	scriptMember := g.GetCommitteeCredentialMember(common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: coldKey,
	})
	if keyMember != nil && scriptMember != nil {
		return nil
	}
	if keyMember != nil {
		return keyMember
	}
	return scriptMember
}

// GetCommitteeCredentialMember returns a committee member by full cold
// credential.
func (g *GovernanceState) GetCommitteeCredentialMember(
	coldCredential common.Credential,
) *CommitteeMemberInfo {
	return g.getCommitteeCredentialMemberAtEpoch(
		coldCredential,
		g.CurrentEpoch,
	)
}

func (g *GovernanceState) getCommitteeCredentialMemberAtEpoch(
	coldCredential common.Credential,
	currentEpoch uint64,
) *CommitteeMemberInfo {
	coldKey := ledger.NewRewardAccountKey(coldCredential)
	if member := g.CommitteeMembersByCredential[coldKey]; isActiveCommitteeMember(
		member,
		currentEpoch,
	) {
		return member
	}
	if coldCredential.CredType == common.CredentialTypeAddrKeyHash &&
		!hasCredentialHash(
			g.CommitteeMembersByCredential,
			coldCredential.Credential,
		) {
		if member := g.CommitteeMembers[coldCredential.Credential]; isActiveCommitteeMember(
			member,
			currentEpoch,
		) {
			return member
		}
	}
	for _, proposal := range g.Proposals {
		if !isActiveUpdateCommitteeProposal(proposal, currentEpoch) {
			continue
		}
		if expiry, ok := proposal.ProposedMembersByCredential[coldKey]; ok &&
			currentEpoch <= expiry {
			return &CommitteeMemberInfo{
				ColdCredential: coldCredential,
				ColdKey:        coldCredential.Credential,
				ExpiryEpoch:    expiry,
				Resigned:       g.CommitteeResignations[coldKey],
			}
		}
		if coldCredential.CredType == common.CredentialTypeAddrKeyHash &&
			!hasCredentialHash(
				proposal.ProposedMembersByCredential,
				coldCredential.Credential,
			) {
			expiry, ok := proposal.ProposedMembers[coldCredential.Credential]
			if !ok || currentEpoch > expiry {
				continue
			}
			return &CommitteeMemberInfo{
				ColdCredential: coldCredential,
				ColdKey:        coldCredential.Credential,
				ExpiryEpoch:    expiry,
				Resigned:       g.CommitteeResignations[coldKey],
			}
		}
	}
	return nil
}

// IsProposedCommitteeMember checks whether an unambiguous cold key hash is
// proposed in a pending UpdateCommittee proposal.
func (g *GovernanceState) IsProposedCommitteeMember(
	coldKey common.Blake2b224,
) bool {
	keyProposed := g.IsProposedCommitteeCredentialMember(common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldKey,
	})
	scriptProposed := g.IsProposedCommitteeCredentialMember(common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: coldKey,
	})
	return keyProposed || scriptProposed
}

// IsProposedCommitteeCredentialMember checks if a full cold credential is
// proposed in any pending UpdateCommittee proposal.
func (g *GovernanceState) IsProposedCommitteeCredentialMember(
	coldCredential common.Credential,
) bool {
	coldKey := ledger.NewRewardAccountKey(coldCredential)
	for _, proposal := range g.Proposals {
		if isActiveUpdateCommitteeProposal(proposal, g.CurrentEpoch) {
			if _, ok := proposal.ProposedMembersByCredential[coldKey]; ok {
				return true
			}
			if coldCredential.CredType == common.CredentialTypeAddrKeyHash &&
				!hasCredentialHash(
					proposal.ProposedMembersByCredential,
					coldCredential.Credential,
				) {
				_, ok := proposal.ProposedMembers[coldCredential.Credential]
				if ok {
					return true
				}
			}
		}
	}
	return false
}

func isActiveCommitteeMember(
	member *CommitteeMemberInfo,
	currentEpoch uint64,
) bool {
	return member != nil && currentEpoch <= member.ExpiryEpoch
}

func (g *GovernanceState) hasActiveCommitteeMember(currentEpoch uint64) bool {
	for _, member := range g.CommitteeMembersByCredential {
		if isActiveCommitteeMember(member, currentEpoch) {
			return true
		}
	}
	for hash, member := range g.CommitteeMembers {
		if hasCredentialHash(g.CommitteeMembersByCredential, hash) {
			continue
		}
		if isActiveCommitteeMember(member, currentEpoch) {
			return true
		}
	}
	return false
}

func (g *GovernanceState) hasCommitteeState(currentEpoch uint64) bool {
	if g.hasActiveCommitteeMember(currentEpoch) {
		return true
	}
	for _, proposal := range g.Proposals {
		if !isActiveUpdateCommitteeProposal(proposal, currentEpoch) {
			continue
		}
		for _, expiry := range proposal.ProposedMembersByCredential {
			if currentEpoch <= expiry {
				return true
			}
		}
		for hash, expiry := range proposal.ProposedMembers {
			if hasCredentialHash(proposal.ProposedMembersByCredential, hash) {
				continue
			}
			if currentEpoch <= expiry {
				return true
			}
		}
	}
	return false
}

func isActiveUpdateCommitteeProposal(
	proposal *ProposalState,
	currentEpoch uint64,
) bool {
	return proposal != nil &&
		proposal.ActionType == common.GovActionTypeUpdateCommittee &&
		currentEpoch <= proposal.ExpiresAfter
}

// GetProposal returns a proposal by its GovActionId.
func (g *GovernanceState) GetProposal(govActionId string) *ProposalState {
	return g.Proposals[govActionId]
}

// GetRewardBalance returns the reward balance for a stake credential.
func (g *GovernanceState) GetRewardBalance(hash common.Blake2b224) uint64 {
	return g.RewardAccounts[hash]
}

// GetRewardAccountBalance returns the reward balance for a full credential
// identity. It falls back to the legacy key-hash map for state constructed by
// older consumers.
func (g *GovernanceState) GetRewardAccountBalance(
	credential common.Credential,
) uint64 {
	if len(g.RewardAccountBalances) > 0 {
		return g.RewardAccountBalances[ledger.NewRewardAccountKey(credential)]
	}
	return g.GetRewardBalance(credential.Credential)
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
	g.StakeRegistrationsByCredential[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: hash,
	}] = true
}

// RegisterStakeCredential registers a full stake credential and initializes
// its reward account without aliasing another credential type.
func (g *GovernanceState) RegisterStakeCredential(
	credential common.Credential,
) {
	g.StakeRegistrations[credential.Credential] = true
	g.StakeRegistrationsByCredential[ledger.NewRewardAccountKey(credential)] = true
	g.RewardAccountBalances[ledger.NewRewardAccountKey(credential)] = 0
	if _, exists := g.RewardAccounts[credential.Credential]; !exists ||
		credential.CredType == common.CredentialTypeAddrKeyHash {
		g.RewardAccounts[credential.Credential] = 0
	}
}

// SetDRepDelegation records a vote delegation for one full stake credential.
func (g *GovernanceState) SetDRepDelegation(
	credential common.Credential,
	delegation common.Drep,
) {
	if g.DRepDelegationsByCredential == nil {
		g.DRepDelegationsByCredential = make(
			map[ledger.RewardAccountKey]common.Drep,
		)
	}
	credentialKey := ledger.NewRewardAccountKey(credential)
	g.DRepDelegationsByCredential[credentialKey] = delegation
	g.DRepDelegations = drepDelegationsByHash(
		g.DRepDelegationsByCredential,
	)
}

// DeregisterStake deregisters a stake credential.
func (g *GovernanceState) DeregisterStake(hash common.Blake2b224) {
	delete(g.StakeRegistrations, hash)
	for credential := range g.StakeRegistrationsByCredential {
		if credential.Credential == hash {
			delete(g.StakeRegistrationsByCredential, credential)
		}
	}
	delete(g.RewardAccounts, hash)
	for credential := range g.RewardAccountBalances {
		if credential.Credential == hash {
			delete(g.RewardAccountBalances, credential)
		}
	}
	for credential := range g.DRepDelegationsByCredential {
		if credential.Credential == hash {
			delete(g.DRepDelegationsByCredential, credential)
		}
	}
	g.DRepDelegations = drepDelegationsByHash(
		g.DRepDelegationsByCredential,
	)
}

// DeregisterStakeCredential deregisters one full stake credential without
// deleting a different credential type that carries the same hash.
func (g *GovernanceState) DeregisterStakeCredential(
	credential common.Credential,
) {
	delete(
		g.StakeRegistrationsByCredential,
		ledger.NewRewardAccountKey(credential),
	)
	delete(g.RewardAccountBalances, ledger.NewRewardAccountKey(credential))
	delete(
		g.DRepDelegationsByCredential,
		ledger.NewRewardAccountKey(credential),
	)
	delete(
		g.PoolDelegationsByCredential,
		ledger.NewRewardAccountKey(credential),
	)
	g.DRepDelegations = drepDelegationsByHash(
		g.DRepDelegationsByCredential,
	)
	for registration, registered := range g.StakeRegistrationsByCredential {
		if registered && registration.Credential == credential.Credential {
			g.StakeRegistrations[credential.Credential] = true
			remainingBalances := rewardBalancesByHash(g.RewardAccountBalances)
			if balance, exists := remainingBalances[credential.Credential]; exists {
				g.RewardAccounts[credential.Credential] = balance
			} else {
				g.RewardAccounts[credential.Credential] = 0
			}
			return
		}
	}
	delete(g.StakeRegistrations, credential.Credential)
	delete(g.RewardAccounts, credential.Credential)
}

func drepDelegationsByHash(
	delegations map[ledger.RewardAccountKey]common.Drep,
) map[common.Blake2b224]common.Drep {
	result := make(map[common.Blake2b224]common.Drep, len(delegations))
	for credential, delegation := range delegations {
		_, exists := result[credential.Credential]
		if !exists || credential.CredType == common.CredentialTypeAddrKeyHash {
			result[credential.Credential] = delegation
		}
	}
	return result
}

// RegisterDRep registers a DRep.
func (g *GovernanceState) RegisterDRep(hash common.Blake2b224) {
	g.DRepRegistrations[hash] = true
	g.DRepRegistrationsByCredential[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: hash,
	}] = true
}

func (g *GovernanceState) RegisterDRepCredentialUntil(
	credential common.Credential,
	expiry uint64,
) {
	key := ledger.NewRewardAccountKey(credential)
	g.DRepRegistrationsByCredential[key] = true
	g.DRepExpiries[key] = expiry
	g.DRepRegistrations[credential.Credential] = true
}

// IsDRepCredentialRegistered checks registration by full credential identity.
// The hash-only map is used only for legacy state without a typed entry sharing
// the hash.
func (g *GovernanceState) IsDRepCredentialRegistered(
	credential common.Credential,
) bool {
	key := ledger.NewRewardAccountKey(credential)
	if registered, exists := g.DRepRegistrationsByCredential[key]; exists {
		return registered
	}
	if hasCredentialHash(
		g.DRepRegistrationsByCredential,
		credential.Credential,
	) {
		return false
	}
	return g.DRepRegistrations[credential.Credential]
}

func (g *GovernanceState) IsDRepCredentialActive(
	credential common.Credential,
	currentEpoch uint64,
) bool {
	key := ledger.NewRewardAccountKey(credential)
	if !g.IsDRepCredentialRegistered(credential) {
		return false
	}
	expiry, known := g.DRepExpiries[key]
	return !known || currentEpoch <= expiry
}

// DeregisterDRepCredential deregisters one full DRep credential without
// deleting a different credential type that carries the same hash.
func (g *GovernanceState) DeregisterDRepCredential(
	credential common.Credential,
) {
	key := ledger.NewRewardAccountKey(credential)
	delete(g.DRepRegistrationsByCredential, key)
	delete(g.DRepExpiries, key)
	for other, registered := range g.DRepRegistrationsByCredential {
		if registered && other.Credential == credential.Credential {
			g.DRepRegistrations[credential.Credential] = true
			return
		}
	}
	delete(g.DRepRegistrations, credential.Credential)
}

// DeregisterDRep deregisters a DRep.
func (g *GovernanceState) DeregisterDRep(hash common.Blake2b224) {
	delete(g.DRepRegistrations, hash)
	for credential := range g.DRepRegistrationsByCredential {
		if credential.Credential == hash {
			delete(g.DRepRegistrationsByCredential, credential)
			delete(g.DRepExpiries, credential)
		}
	}
}

func (g *GovernanceState) SetPoolDelegation(
	credential common.Credential,
	pool common.PoolKeyHash,
) {
	g.PoolDelegationsByCredential[ledger.NewRewardAccountKey(credential)] = pool
}

func (g *GovernanceState) SetPoolRewardAccount(
	pool common.PoolKeyHash,
	account ledger.RewardAccountKey,
) {
	g.PoolRewardAccounts[pool] = account
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

// AuthorizeHotCredential authorizes a hot credential for a committee member.
func (g *GovernanceState) AuthorizeHotCredential(
	coldCredential common.Credential,
	hotCredential common.Credential,
) {
	coldKey := ledger.NewRewardAccountKey(coldCredential)
	if g.CommitteeResignations[coldKey] {
		return
	}
	if member := g.CommitteeMembersByCredential[coldKey]; member != nil &&
		member.Resigned {
		return
	}
	if coldCredential.CredType == common.CredentialTypeAddrKeyHash {
		if member := g.CommitteeMembers[coldCredential.Credential]; member != nil &&
			member.Resigned {
			return
		}
	}
	if !hasCredentialHash(
		g.HotKeyAuthorizationsByCredential,
		coldCredential.Credential,
	) {
		if legacyHotKey, ok := g.HotKeyAuthorizations[coldCredential.Credential]; ok {
			g.HotKeyAuthorizationsByCredential[ledger.RewardAccountKey{
				CredType:   common.CredentialTypeAddrKeyHash,
				Credential: coldCredential.Credential,
			}] = common.Credential{
				CredType:   common.CredentialTypeAddrKeyHash,
				Credential: legacyHotKey,
			}
		}
	}
	g.HotKeyAuthorizationsByCredential[coldKey] = hotCredential
	g.syncLegacyHotKeyAuthorizations()
	if member, ok := g.CommitteeMembersByCredential[coldKey]; ok {
		hotCredentialCopy := hotCredential
		member.HotCredential = &hotCredentialCopy
		hotHash := hotCredential.Credential
		member.HotKey = &hotHash
	}
}

// AuthorizeHotKey authorizes a legacy key-hash hot credential for a legacy
// key-hash cold credential.
func (g *GovernanceState) AuthorizeHotKey(
	coldKey,
	hotKey common.Blake2b224,
) {
	g.AuthorizeHotCredential(
		common.Credential{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: coldKey,
		},
		common.Credential{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: hotKey,
		},
	)
	if g.CommitteeResignations[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldKey,
	}] {
		return
	}
	if member := g.legacyKeyCommitteeMember(coldKey); member != nil {
		if member.Resigned {
			return
		}
		hotKeyCopy := hotKey
		member.HotKey = &hotKeyCopy
	}
}

// ResignCommitteeCredential marks a committee member as resigned.
func (g *GovernanceState) ResignCommitteeCredential(
	coldCredential common.Credential,
) {
	coldKey := ledger.NewRewardAccountKey(coldCredential)
	delete(g.HotKeyAuthorizationsByCredential, coldKey)
	delete(g.HotKeyAuthorizations, coldCredential.Credential)
	g.syncLegacyHotKeyAuthorizations()
	g.CommitteeResignations[coldKey] = true
	if member, ok := g.CommitteeMembersByCredential[coldKey]; ok {
		member.HotCredential = nil
		member.HotKey = nil
		member.Resigned = true
	}
}

// ResignCommitteeMember resigns a legacy key-hash cold credential.
func (g *GovernanceState) ResignCommitteeMember(coldKey common.Blake2b224) {
	g.ResignCommitteeCredential(common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldKey,
	})
	if member := g.legacyKeyCommitteeMember(coldKey); member != nil {
		member.HotKey = nil
		member.Resigned = true
	}
}

// legacyKeyCommitteeMember returns a member stored through the legacy
// hash-only view unless that entry is known to represent a script credential.
func (g *GovernanceState) legacyKeyCommitteeMember(
	coldKey common.Blake2b224,
) *CommitteeMemberInfo {
	member := g.CommitteeMembers[coldKey]
	if member == nil {
		return nil
	}
	scriptMember := g.CommitteeMembersByCredential[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: coldKey,
	}]
	if member == scriptMember {
		return nil
	}
	return member
}

func (g *GovernanceState) syncLegacyCommitteeMembers() {
	legacyMembers := maps.Clone(g.CommitteeMembers)
	clear(g.CommitteeMembers)
	ambiguous := make(map[common.Blake2b224]bool)
	typedHashes := make(map[common.Blake2b224]bool)
	for credential, member := range g.CommitteeMembersByCredential {
		hash := credential.Credential
		typedHashes[hash] = true
		if ambiguous[hash] {
			continue
		}
		if _, exists := g.CommitteeMembers[hash]; exists {
			delete(g.CommitteeMembers, hash)
			ambiguous[hash] = true
			continue
		}
		g.CommitteeMembers[hash] = member
	}
	for coldKey, member := range legacyMembers {
		if typedHashes[coldKey] {
			continue
		}
		g.CommitteeMembers[coldKey] = member
	}
}

func (g *GovernanceState) syncLegacyHotKeyAuthorizations() {
	legacyAuthorizations := maps.Clone(g.HotKeyAuthorizations)
	clear(g.HotKeyAuthorizations)
	ambiguous := make(map[common.Blake2b224]bool)
	typedHashes := make(map[common.Blake2b224]bool)
	for credential, hotCredential := range g.HotKeyAuthorizationsByCredential {
		hash := credential.Credential
		typedHashes[hash] = true
		if ambiguous[hash] {
			continue
		}
		if _, exists := g.HotKeyAuthorizations[hash]; exists {
			delete(g.HotKeyAuthorizations, hash)
			ambiguous[hash] = true
			continue
		}
		g.HotKeyAuthorizations[hash] = hotCredential.Credential
	}
	for coldKey, hotKey := range legacyAuthorizations {
		if typedHashes[coldKey] {
			continue
		}
		g.HotKeyAuthorizations[coldKey] = hotKey
	}
}

// AddProposal adds a new governance proposal.
func (g *GovernanceState) AddProposal(govActionId string, info GovActionInfo) {
	g.Proposals[govActionId] = &ProposalState{
		GovActionInfo: info,
	}
}
