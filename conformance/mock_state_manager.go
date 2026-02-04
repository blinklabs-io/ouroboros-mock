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

package conformance

import (
	"encoding/hex"
	"fmt"
	"maps"
	"math/big"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
	"github.com/blinklabs-io/plutigo/data"
	utxorpc "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
)

// MockStateManager provides an in-memory StateManager implementation for testing.
// It maintains internal state and rebuilds the MockLedgerState as needed.
type MockStateManager struct {
	// protocolParams holds the current protocol parameters
	protocolParams common.ProtocolParameters

	// govState tracks governance-related state
	govState *GovernanceState

	// currentEpoch tracks the current epoch
	currentEpoch uint64

	// utxos stores UTxOs by their ID string
	utxos map[string]common.Utxo

	// stakeRegistrations tracks registered stake credentials and their balances
	stakeRegistrations map[common.Blake2b224]uint64

	// poolRegistrations tracks registered pools
	poolRegistrations map[common.Blake2b224]bool

	// drepRegistrations tracks registered DReps
	drepRegistrations map[common.Blake2b224]bool

	// committeeMembers tracks committee members (cold key -> expiry epoch)
	committeeMembers map[common.Blake2b224]uint64

	// hotKeyAuthorizations tracks hot key authorizations (cold key -> hot key)
	hotKeyAuthorizations map[common.Blake2b224]common.Blake2b224
}

// NewMockStateManager creates a new MockStateManager.
func NewMockStateManager() *MockStateManager {
	return &MockStateManager{
		govState:             NewGovernanceState(),
		utxos:                make(map[string]common.Utxo),
		stakeRegistrations:   make(map[common.Blake2b224]uint64),
		poolRegistrations:    make(map[common.Blake2b224]bool),
		drepRegistrations:    make(map[common.Blake2b224]bool),
		committeeMembers:     make(map[common.Blake2b224]uint64),
		hotKeyAuthorizations: make(map[common.Blake2b224]common.Blake2b224),
	}
}

// LoadInitialState implements StateManager.LoadInitialState.
func (m *MockStateManager) LoadInitialState(
	state *ParsedInitialState,
	pp common.ProtocolParameters,
) error {
	m.protocolParams = pp
	m.currentEpoch = state.CurrentEpoch

	// Clear existing state
	m.utxos = make(map[string]common.Utxo)
	m.stakeRegistrations = make(map[common.Blake2b224]uint64)
	m.poolRegistrations = make(map[common.Blake2b224]bool)
	m.drepRegistrations = make(map[common.Blake2b224]bool)
	m.committeeMembers = make(map[common.Blake2b224]uint64)
	m.hotKeyAuthorizations = make(map[common.Blake2b224]common.Blake2b224)

	// Load stake registrations with reward balances
	for hash, registered := range state.StakeRegistrations {
		if registered {
			balance := state.RewardAccounts[hash]
			m.stakeRegistrations[hash] = balance
		}
	}

	// Load pool registrations
	for hash, registered := range state.PoolRegistrations {
		if registered {
			m.poolRegistrations[hash] = true
		}
	}

	// Load DRep registrations
	for _, hash := range state.DRepRegistrations {
		m.drepRegistrations[hash] = true
	}

	// Load committee members
	maps.Copy(m.committeeMembers, state.CommitteeMembers)

	// Load hot key authorizations
	maps.Copy(m.hotKeyAuthorizations, state.HotKeyAuthorizations)

	// Load governance state
	m.govState = NewGovernanceState()
	m.govState.LoadFromParsedState(state)

	// Populate UTxOs from parsed state using the fully decoded Output
	for utxoId, parsedUtxo := range state.Utxos {
		// Create a mock transaction input for the UTxO ID
		var txHash common.Blake2b256
		copy(txHash[:], parsedUtxo.TxHash)

		mockInput := &mockTransactionInput{
			txId:  txHash,
			index: parsedUtxo.Index,
		}

		// Use the decoded Output directly (e.g., BabbageTransactionOutput)
		// which has all fields (datum, datumHash, scriptRef, assets, etc.)
		m.utxos[utxoId] = common.Utxo{
			Id:     mockInput,
			Output: parsedUtxo.Output,
		}
	}

	return nil
}

// ApplyTransaction implements StateManager.ApplyTransaction.
func (m *MockStateManager) ApplyTransaction(
	tx common.Transaction,
	slot uint64,
) error {
	// For phase-2 invalid transactions (IsValid=false), only consume collateral
	// and add collateral return output if present
	if !tx.IsValid() {
		// Consume collateral inputs
		for _, input := range tx.Collateral() {
			inputId := input.Id()
			inputIdx := input.Index()
			utxoId := fmt.Sprintf(
				"%s#%d",
				hex.EncodeToString(inputId.Bytes()),
				inputIdx,
			)
			delete(m.utxos, utxoId)
		}

		// Add collateral return output if present
		// Per Alonzo UTXOS spec, excess collateral is returned to this output
		if collateralReturn := tx.CollateralReturn(); collateralReturn != nil {
			txHash := tx.Hash()
			txHashStr := hex.EncodeToString(txHash.Bytes())
			// Collateral return uses a special index (typically total outputs count)
			// The index is determined by the transaction structure
			//nolint:gosec // output count bounded by protocol max tx size
			returnIdx := uint32(len(tx.Outputs()))
			utxoId := fmt.Sprintf("%s#%d", txHashStr, returnIdx)

			mockInput := &mockTransactionInput{
				txId:  txHash,
				index: returnIdx,
			}
			m.utxos[utxoId] = common.Utxo{
				Id:     mockInput,
				Output: collateralReturn,
			}
		}

		return nil
	}

	// Get tx hash as string
	txHash := tx.Hash()
	txHashStr := hex.EncodeToString(txHash.Bytes())

	// Process consumed UTxOs (inputs)
	inputs := tx.Inputs()
	for _, input := range inputs {
		inputId := input.Id()
		inputIdx := input.Index()
		utxoId := fmt.Sprintf(
			"%s#%d",
			hex.EncodeToString(inputId.Bytes()),
			inputIdx,
		)
		delete(m.utxos, utxoId)
	}

	// Process produced UTxOs (outputs)
	outputs := tx.Outputs()
	for idx, output := range outputs {
		utxoId := fmt.Sprintf("%s#%d", txHashStr, idx)

		// Create a mock transaction input for the UTxO ID
		mockInput := &mockTransactionInput{
			txId:  txHash,
			index: uint32(idx), //nolint:gosec // idx bounded by tx outputs
		}

		m.utxos[utxoId] = common.Utxo{
			Id:     mockInput,
			Output: output,
		}
	}

	// Process certificates
	certs := tx.Certificates()
	for _, cert := range certs {
		m.processCertificate(cert)
	}

	// Process governance proposals
	proposals := tx.ProposalProcedures()
	for idx, proposal := range proposals {
		govActionId := fmt.Sprintf("%s#%d", txHashStr, idx)
		action := proposal.GovAction()
		if action != nil {
			// Get govActionLifetime from protocol parameters
			var govActionLifetime uint64 = 6 // Default fallback
			if conwayPP, ok := m.protocolParams.(*conway.ConwayProtocolParameters); ok {
				govActionLifetime = conwayPP.GovActionValidityPeriod
			}

			info := GovActionInfo{
				ActionType:      getActionType(action),
				ExpiresAfter:    m.currentEpoch + govActionLifetime,
				SubmittedEpoch:  m.currentEpoch,
				ProposedMembers: make(map[common.Blake2b224]uint64),
			}

			// Extract action-specific data including parent action ID
			switch ga := action.(type) {
			case *common.UpdateCommitteeGovAction:
				if ga.ActionId != nil {
					key := fmt.Sprintf("%x#%d", ga.ActionId.TransactionId[:], ga.ActionId.GovActionIdx)
					info.ParentActionId = &key
				}
				for cred, epoch := range ga.CredEpochs {
					if cred != nil {
						info.ProposedMembers[cred.Credential] = uint64(epoch)
					}
				}
			case *common.NoConfidenceGovAction:
				if ga.ActionId != nil {
					key := fmt.Sprintf("%x#%d", ga.ActionId.TransactionId[:], ga.ActionId.GovActionIdx)
					info.ParentActionId = &key
				}
			case *common.HardForkInitiationGovAction:
				if ga.ActionId != nil {
					key := fmt.Sprintf("%x#%d", ga.ActionId.TransactionId[:], ga.ActionId.GovActionIdx)
					info.ParentActionId = &key
				}
				info.ProtocolVersion = &ProtocolVersionInfo{
					Major: ga.ProtocolVersion.Major,
					Minor: ga.ProtocolVersion.Minor,
				}
			case *common.NewConstitutionGovAction:
				if ga.ActionId != nil {
					key := fmt.Sprintf("%x#%d", ga.ActionId.TransactionId[:], ga.ActionId.GovActionIdx)
					info.ParentActionId = &key
				}
				// Store the proposed constitution's policy hash for enactment
				if len(ga.Constitution.ScriptHash) > 0 {
					info.PolicyHash = make([]byte, len(ga.Constitution.ScriptHash))
					copy(info.PolicyHash, ga.Constitution.ScriptHash)
				}
			case *conway.ConwayParameterChangeGovAction:
				if ga.ActionId != nil {
					key := fmt.Sprintf("%x#%d", ga.ActionId.TransactionId[:], ga.ActionId.GovActionIdx)
					info.ParentActionId = &key
				}
				// Store the parameter update for enactment
				info.ParameterUpdate = &ga.ParamUpdate
			}
			m.govState.AddProposal(govActionId, info)
		}
	}

	// Process voting procedures
	votes := tx.VotingProcedures()
	for voter, voteMap := range votes {
		for govActionId, votingProc := range voteMap {
			actionKey := fmt.Sprintf(
				"%s#%d",
				hex.EncodeToString(govActionId.TransactionId[:]),
				govActionId.GovActionIdx,
			)
			proposal := m.govState.GetProposal(actionKey)
			if proposal == nil {
				continue
			}
			if proposal.Votes == nil {
				proposal.Votes = make(map[string]uint8)
			}
			// Store vote as "voterType:credHash" -> vote value
			voterKey := fmt.Sprintf(
				"%d:%s",
				voter.Type,
				hex.EncodeToString(voter.Hash[:]),
			)
			proposal.Votes[voterKey] = votingProc.Vote
		}
	}

	return nil
}

// processCertificate processes a single certificate and updates state.
func (m *MockStateManager) processCertificate(cert common.Certificate) {
	certType := common.CertificateType(cert.Type())

	//exhaustive:ignore
	switch certType {
	case common.CertificateTypeStakeRegistration:
		if regCert, ok := cert.(*common.StakeRegistrationCertificate); ok {
			credential := regCert.StakeCredential.Credential
			m.stakeRegistrations[credential] = 0
			m.govState.RegisterStake(credential)
		}

	case common.CertificateTypeRegistration:
		if regCert, ok := cert.(*common.RegistrationCertificate); ok {
			credential := regCert.StakeCredential.Credential
			m.stakeRegistrations[credential] = 0
			m.govState.RegisterStake(credential)
		}

	case common.CertificateTypeStakeRegistrationDelegation:
		// Combined registration + delegation (Conway)
		if regCert, ok := cert.(*common.StakeRegistrationDelegationCertificate); ok {
			credential := regCert.StakeCredential.Credential
			m.stakeRegistrations[credential] = 0
			m.govState.RegisterStake(credential)
		}

	case common.CertificateTypeVoteRegistrationDelegation:
		// Combined registration + vote delegation (Conway)
		if regCert, ok := cert.(*common.VoteRegistrationDelegationCertificate); ok {
			credential := regCert.StakeCredential.Credential
			m.stakeRegistrations[credential] = 0
			m.govState.RegisterStake(credential)
		}

	case common.CertificateTypeStakeVoteRegistrationDelegation:
		// Combined registration + stake + vote delegation (Conway)
		if regCert, ok := cert.(*common.StakeVoteRegistrationDelegationCertificate); ok {
			credential := regCert.StakeCredential.Credential
			m.stakeRegistrations[credential] = 0
			m.govState.RegisterStake(credential)
		}

	case common.CertificateTypeStakeDelegation:
		// Standalone stake delegation (without registration)
		// Used for redelegation to a different pool
		// No state change needed - delegation is tracked elsewhere in full implementation

	case common.CertificateTypeVoteDelegation:
		// Standalone vote delegation (without registration)
		// Used for changing DRep delegation
		// No state change needed - delegation is tracked elsewhere in full implementation

	case common.CertificateTypeStakeVoteDelegation:
		// Combined stake + vote delegation (without registration)
		// No state change needed - delegation is tracked elsewhere in full implementation

	case common.CertificateTypeStakeDeregistration:
		if deregCert, ok := cert.(*common.StakeDeregistrationCertificate); ok {
			credential := deregCert.StakeCredential.Credential
			delete(m.stakeRegistrations, credential)
			m.govState.DeregisterStake(credential)
		}

	case common.CertificateTypeDeregistration:
		if deregCert, ok := cert.(*common.DeregistrationCertificate); ok {
			credential := deregCert.StakeCredential.Credential
			delete(m.stakeRegistrations, credential)
			m.govState.DeregisterStake(credential)
		}

	case common.CertificateTypePoolRegistration:
		if poolCert, ok := cert.(*common.PoolRegistrationCertificate); ok {
			poolId := poolCert.Operator
			m.poolRegistrations[poolId] = true
			m.govState.RegisterPool(poolId)
		}

	case common.CertificateTypePoolRetirement:
		if retireCert, ok := cert.(*common.PoolRetirementCertificate); ok {
			poolId := retireCert.PoolKeyHash
			retireEpoch := retireCert.Epoch
			m.govState.RetirePool(poolId, retireEpoch)
		}

	case common.CertificateTypeRegistrationDrep:
		if drepCert, ok := cert.(*common.RegistrationDrepCertificate); ok {
			credential := drepCert.DrepCredential.Credential
			m.drepRegistrations[credential] = true
			m.govState.RegisterDRep(credential)
		}

	case common.CertificateTypeDeregistrationDrep:
		if drepCert, ok := cert.(*common.DeregistrationDrepCertificate); ok {
			credential := drepCert.DrepCredential.Credential
			delete(m.drepRegistrations, credential)
			m.govState.DeregisterDRep(credential)
		}

	case common.CertificateTypeAuthCommitteeHot:
		if authCert, ok := cert.(*common.AuthCommitteeHotCertificate); ok {
			coldKey := authCert.ColdCredential.Credential
			hotKey := authCert.HotCredential.Credential
			m.hotKeyAuthorizations[coldKey] = hotKey
			m.govState.AuthorizeHotKey(coldKey, hotKey)
		}

	case common.CertificateTypeResignCommitteeCold:
		if resignCert, ok := cert.(*common.ResignCommitteeColdCertificate); ok {
			coldKey := resignCert.ColdCredential.Credential
			delete(m.hotKeyAuthorizations, coldKey)
			m.govState.ResignCommitteeMember(coldKey)
		}

	default:
		// Other certificate types not relevant for state tracking
	}
}

// ProcessEpochBoundary implements StateManager.ProcessEpochBoundary.
func (m *MockStateManager) ProcessEpochBoundary(newEpoch uint64) error {
	m.currentEpoch = newEpoch
	m.govState.CurrentEpoch = newEpoch

	// Snapshot pool retirements before processing, since ProcessPoolRetirements
	// will delete entries from PoolRetirements as it processes them
	retirementsSnapshot := maps.Clone(m.govState.PoolRetirements)

	// Process pool retirements in governance state
	m.govState.ProcessPoolRetirements(newEpoch)

	// Also update local pool registrations using the snapshot
	for poolId, retireEpoch := range retirementsSnapshot {
		if newEpoch >= retireEpoch {
			delete(m.poolRegistrations, poolId)
		}
	}

	// Phase 1: Enact proposals that were ratified in previous epochs
	// Collect proposals to enact (can't modify map while iterating)
	var toEnact []string
	for id, proposal := range m.govState.Proposals {
		if proposal == nil {
			continue
		}
		if proposal.RatifiedEpoch != nil && newEpoch > *proposal.RatifiedEpoch {
			toEnact = append(toEnact, id)
		}
	}

	// Enact collected proposals (update roots)
	for _, id := range toEnact {
		proposal := m.govState.Proposals[id]
		if proposal == nil {
			continue
		}
		// Info proposals cannot be enacted (per Cardano spec)
		// They just stay ratified until they expire
		if proposal.ActionType == common.GovActionTypeInfo {
			continue
		}
		m.enactProposal(id, proposal)
	}

	// Phase 2: Ratify proposals that meet threshold requirements
	m.ratifyProposals(newEpoch)

	// Phase 3: Expire old proposals
	for id, proposal := range m.govState.Proposals {
		if proposal == nil {
			continue
		}
		if newEpoch > proposal.ExpiresAfter {
			delete(m.govState.Proposals, id)
		}
	}

	return nil
}

// ratifyProposals performs simplified proposal ratification.
// For conformance testing, we use a simplified model based on CIP-1694 requirements.
//
// Per CIP-1694, different action types require votes from different stakeholders:
// - UpdateCommittee: CC + DRep (no SPO)
// - NoConfidence: CC + DRep + SPO
// - HardFork: CC + DRep + SPO
// - NewConstitution: CC + DRep (no SPO)
// - ParameterChange: CC + DRep (no SPO)
// - TreasuryWithdrawal: CC + DRep (no SPO)
// - Info: No votes required (auto-ratified)
//
// Voter types: 0=CC, 2=DRep, 4=SPO
// Vote values per CIP-1694: 0=No, 1=Yes, 2=Abstain
func (m *MockStateManager) ratifyProposals(currentEpoch uint64) {
	for id, proposal := range m.govState.Proposals {
		// Skip already-ratified proposals
		if proposal.RatifiedEpoch != nil {
			continue
		}

		// Require at least 1 epoch between submission and ratification
		if currentEpoch <= proposal.SubmittedEpoch {
			continue
		}

		// Info proposals are auto-ratified (no votes required)
		if proposal.ActionType == common.GovActionTypeInfo {
			epoch := currentEpoch
			proposal.RatifiedEpoch = &epoch
			m.govState.Proposals[id] = proposal
			continue
		}

		// Skip proposals that haven't been voted on
		if len(proposal.Votes) == 0 {
			continue
		}

		// Count YES votes by voter type
		// Vote values per CIP-1694: 0=No, 1=Yes, 2=Abstain
		voterTypesWithYes := make(map[uint8]bool)
		for voterKey, voteValue := range proposal.Votes {
			// Only count YES votes (value = 1)
			if voteValue != 1 {
				continue
			}
			// Voter key format is "voterType:credHash"
			if len(voterKey) > 0 {
				voterType := voterKey[0] - '0' // Simple parse of first char
				voterTypesWithYes[voterType] = true
			}
		}

		// Check if required voter types have voted YES based on action type
		hasCC := voterTypesWithYes[0] ||
			voterTypesWithYes[1] // Type 0 or 1 (hot key hash or script)
		hasDRep := voterTypesWithYes[2] || voterTypesWithYes[3] // Type 2 or 3
		hasSPO := voterTypesWithYes[4] || voterTypesWithYes[5]  // Type 4 or 5

		var meetsRequirements bool
		//exhaustive:ignore
		switch proposal.ActionType {
		case common.GovActionTypeNoConfidence,
			common.GovActionTypeHardForkInitiation:
			// Requires CC + DRep + SPO
			meetsRequirements = hasCC && hasDRep && hasSPO
		case common.GovActionTypeUpdateCommittee,
			common.GovActionTypeNewConstitution,
			common.GovActionTypeParameterChange,
			common.GovActionTypeTreasuryWithdrawal:
			// Requires CC + DRep (no SPO)
			meetsRequirements = hasCC && hasDRep
		default:
			// Unknown action type - require any 2 voter types as fallback
			meetsRequirements = len(voterTypesWithYes) >= 2
		}

		if !meetsRequirements {
			continue
		}

		// Ratify: mark as ratified in current epoch
		// Enactment will happen in the next epoch (handled by ProcessEpochBoundary)
		epoch := currentEpoch
		proposal.RatifiedEpoch = &epoch
		m.govState.Proposals[id] = proposal
	}
}

// enactProposal processes a ratified proposal by updating the appropriate root.
func (m *MockStateManager) enactProposal(id string, proposal *ProposalState) {
	// Update the appropriate root based on action type
	//exhaustive:ignore
	switch proposal.ActionType {
	case common.GovActionTypeNewConstitution:
		m.govState.Roots.Constitution = &id
		// Update the constitution's policy hash from the enacted proposal
		// A NewConstitution with empty PolicyHash removes the guardrails policy
		if m.govState.Constitution == nil {
			m.govState.Constitution = &ConstitutionInfo{}
		}
		if len(proposal.PolicyHash) > 0 {
			m.govState.Constitution.PolicyHash = make(
				[]byte,
				len(proposal.PolicyHash),
			)
			copy(m.govState.Constitution.PolicyHash, proposal.PolicyHash)
		} else {
			// Clear the policy hash if the new constitution has no guardrails
			m.govState.Constitution.PolicyHash = nil
		}
	case common.GovActionTypeParameterChange:
		m.govState.Roots.ProtocolParameters = &id
		// Apply parameter updates to protocol parameters
		if proposal.ParameterUpdate != nil {
			if conwayPP, ok := m.protocolParams.(*conway.ConwayProtocolParameters); ok {
				applyParameterUpdate(conwayPP, proposal.ParameterUpdate)
			}
		}
	case common.GovActionTypeHardForkInitiation:
		m.govState.Roots.HardFork = &id
	case common.GovActionTypeNoConfidence, common.GovActionTypeUpdateCommittee:
		m.govState.Roots.ConstitutionalCommittee = &id
		// For UpdateCommittee, apply the committee changes
		if proposal.ActionType == common.GovActionTypeUpdateCommittee {
			for coldKey, expiry := range proposal.ProposedMembers {
				m.govState.CommitteeMembers[coldKey] = &CommitteeMemberInfo{
					ColdKey:     coldKey,
					ExpiryEpoch: expiry,
				}
				m.committeeMembers[coldKey] = expiry
			}
		}
	}

	// Mark as enacted and remove from active proposals
	m.govState.EnactedProposals[id] = true
	delete(m.govState.Proposals, id)
}

// applyParameterUpdate applies a parameter update to protocol parameters.
func applyParameterUpdate(
	pp *conway.ConwayProtocolParameters,
	update *conway.ConwayProtocolParameterUpdate,
) {
	// Use the existing Update method which properly handles all protocol parameter fields
	pp.Update(update)
}

// GetStateProvider implements StateManager.GetStateProvider.
func (m *MockStateManager) GetStateProvider() StateProvider {
	return m.buildLedgerState()
}

// GetGovernanceState implements StateManager.GetGovernanceState.
func (m *MockStateManager) GetGovernanceState() *GovernanceState {
	return m.govState
}

// SetRewardBalances implements StateManager.SetRewardBalances.
func (m *MockStateManager) SetRewardBalances(
	balances map[common.Blake2b224]uint64,
) {
	// Update both the state manager's internal tracking and governance state
	for cred, balance := range balances {
		if _, exists := m.stakeRegistrations[cred]; exists {
			m.stakeRegistrations[cred] = balance
		}
	}
	if m.govState != nil {
		for cred, balance := range balances {
			if m.govState.StakeRegistrations[cred] {
				m.govState.RewardAccounts[cred] = balance
			}
		}
	}
}

// GetProtocolParameters implements StateManager.GetProtocolParameters.
func (m *MockStateManager) GetProtocolParameters() common.ProtocolParameters {
	return m.protocolParams
}

// Reset implements StateManager.Reset.
func (m *MockStateManager) Reset() error {
	m.protocolParams = nil
	m.currentEpoch = 0
	m.utxos = make(map[string]common.Utxo)
	m.stakeRegistrations = make(map[common.Blake2b224]uint64)
	m.poolRegistrations = make(map[common.Blake2b224]bool)
	m.drepRegistrations = make(map[common.Blake2b224]bool)
	m.committeeMembers = make(map[common.Blake2b224]uint64)
	m.hotKeyAuthorizations = make(map[common.Blake2b224]common.Blake2b224)
	m.govState = NewGovernanceState()
	return nil
}

// buildLedgerState builds a MockLedgerState from current state.
func (m *MockStateManager) buildLedgerState() *ledger.MockLedgerState {
	builder := ledger.NewLedgerStateBuilder()

	// Set up UTxO lookup callback
	utxos := m.utxos // capture for closure
	builder.WithUtxoById(func(id common.TransactionInput) (common.Utxo, error) {
		if id == nil {
			return common.Utxo{}, ledger.ErrNotFound
		}
		inputId := id.Id()
		inputIdx := id.Index()
		utxoId := fmt.Sprintf("%x#%d", inputId.Bytes(), inputIdx)
		if utxo, ok := utxos[utxoId]; ok {
			return utxo, nil
		}
		return common.Utxo{}, ledger.ErrNotFound
	})

	// Set up stake registrations
	stakeRegs := m.stakeRegistrations // capture for closure
	builder.WithStakeCredentials(func() map[common.Blake2b224]bool {
		result := make(map[common.Blake2b224]bool)
		for cred := range stakeRegs {
			result[cred] = true
		}
		return result
	}())

	// Set up reward account balances
	builder.WithRewardAccounts(stakeRegs)

	// Set up pool lookup callback
	// Pool is considered registered if:
	// 1. It's in poolRegistrations, OR
	// 2. It's scheduled for retirement (still registered until retirement epoch)
	poolRegs := m.poolRegistrations               // capture for closure
	poolRetirements := m.govState.PoolRetirements // capture for closure
	builder.WithPoolCurrentState(
		func(poolKeyHash common.PoolKeyHash) (*common.PoolRegistrationCertificate, *uint64, error) {
			if poolRegs[poolKeyHash] {
				// Pool is registered
				// Check if it has a pending retirement
				if retireEpoch, retiring := poolRetirements[poolKeyHash]; retiring {
					return &common.PoolRegistrationCertificate{
						Operator: poolKeyHash,
					}, &retireEpoch, nil
				}
				return &common.PoolRegistrationCertificate{
					Operator: poolKeyHash,
				}, nil, nil
			}
			// Also check if pool is pending retirement (registered but marked for retirement)
			if retireEpoch, retiring := poolRetirements[poolKeyHash]; retiring {
				return &common.PoolRegistrationCertificate{
					Operator: poolKeyHash,
				}, &retireEpoch, nil
			}
			return nil, nil, nil
		},
	)

	// Set up DRep lookup callback
	drepRegs := m.drepRegistrations // capture for closure
	builder.WithDRepRegistration(
		func(cred common.Blake2b224) (*common.DRepRegistration, error) {
			if drepRegs[cred] {
				return &common.DRepRegistration{
					Credential: cred,
				}, nil
			}
			return nil, nil
		},
	)

	// Set up committee member lookup
	committeeMembers := m.committeeMembers         // capture for closure
	hotKeyAuth := m.hotKeyAuthorizations           // capture for closure
	proposedMembers := m.govState.CommitteeMembers // get proposed from govState
	builder.WithCommitteeMember(
		func(coldKey common.Blake2b224) (*common.CommitteeMember, error) {
			// Check current members first
			if expiry, ok := committeeMembers[coldKey]; ok {
				member := &common.CommitteeMember{
					ColdKey:     coldKey,
					ExpiryEpoch: expiry,
				}
				// Add hot key if authorized
				if hotKey, hasHot := hotKeyAuth[coldKey]; hasHot {
					member.HotKey = &hotKey
				}
				return member, nil
			}
			// Check proposed members
			if memberInfo, ok := proposedMembers[coldKey]; ok {
				member := &common.CommitteeMember{
					ColdKey:     coldKey,
					ExpiryEpoch: memberInfo.ExpiryEpoch,
				}
				if hotKey, hasHot := hotKeyAuth[coldKey]; hasHot {
					member.HotKey = &hotKey
				}
				return member, nil
			}
			return nil, nil
		},
	)

	// Set up governance actions
	if len(m.govState.Proposals) > 0 {
		actions := make(map[string]*common.GovActionState)
		for id, proposal := range m.govState.Proposals {
			actions[id] = &common.GovActionState{
				ActionType: proposal.ActionType,
				ExpirySlot: proposal.ExpiresAfter * 432000, // Approximate: epoch * slots per epoch
			}
		}
		builder.WithGovActions(actions)
	}

	// Set up cost models from protocol parameters
	// This is essential for Plutus script validation
	if m.protocolParams != nil {
		costModels := extractCostModels(m.protocolParams)
		if len(costModels) > 0 {
			builder.WithCostModelsMap(costModels)
		}
	}

	return builder.Build()
}

// extractCostModels extracts cost models from protocol parameters.
// Supports Conway, Babbage, and Alonzo protocol parameters.
func extractCostModels(
	pp common.ProtocolParameters,
) map[common.PlutusLanguage]common.CostModel {
	if pp == nil {
		return nil
	}

	// Try Conway first (most common for conformance tests)
	if conwayPP, ok := pp.(*conway.ConwayProtocolParameters); ok {
		return convertCostModels(conwayPP.CostModels)
	}

	// Try Babbage
	type babbageParams interface {
		common.ProtocolParameters
		GetCostModels() map[uint][]int64
	}
	if babbagePP, ok := pp.(babbageParams); ok {
		return convertCostModels(babbagePP.GetCostModels())
	}

	// Try Alonzo
	type alonzoParams interface {
		common.ProtocolParameters
		GetCostModels() map[uint][]int64
	}
	if alonzoPP, ok := pp.(alonzoParams); ok {
		return convertCostModels(alonzoPP.GetCostModels())
	}

	return nil
}

// convertCostModels converts from map[uint][]int64 to map[PlutusLanguage]CostModel.
// Note: CostModel is a placeholder struct in the common package.
func convertCostModels(
	models map[uint][]int64,
) map[common.PlutusLanguage]common.CostModel {
	if models == nil {
		return nil
	}

	result := make(map[common.PlutusLanguage]common.CostModel)
	for version := range models {
		// Only allow valid Plutus versions (0=V1, 1=V2, 2=V3)
		if version > 2 {
			continue
		}
		// Convert uint version to PlutusLanguage (safe: version bounded 0-2)
		//nolint:gosec // G115: version is bounds checked above (0-2)
		plutusLang := common.PlutusLanguage(version + 1)
		result[plutusLang] = common.CostModel{}
	}
	return result
}

// getActionType extracts the action type from a GovAction.
func getActionType(action common.GovAction) common.GovActionType {
	switch action.(type) {
	case *common.HardForkInitiationGovAction:
		return common.GovActionTypeHardForkInitiation
	case *common.TreasuryWithdrawalGovAction:
		return common.GovActionTypeTreasuryWithdrawal
	case *common.NoConfidenceGovAction:
		return common.GovActionTypeNoConfidence
	case *common.UpdateCommitteeGovAction:
		return common.GovActionTypeUpdateCommittee
	case *common.NewConstitutionGovAction:
		return common.GovActionTypeNewConstitution
	case *common.InfoGovAction:
		return common.GovActionTypeInfo
	default:
		return common.GovActionTypeParameterChange
	}
}

// mockTransactionInput implements common.TransactionInput for mock UTxOs.
type mockTransactionInput struct {
	txId  common.Blake2b256
	index uint32
}

func (m *mockTransactionInput) Id() common.Blake2b256 {
	return m.txId
}

func (m *mockTransactionInput) Index() uint32 {
	return m.index
}

func (m *mockTransactionInput) String() string {
	return fmt.Sprintf("%x#%d", m.txId[:], m.index)
}

func (m *mockTransactionInput) Utxorpc() (*utxorpc.TxInput, error) {
	return &utxorpc.TxInput{
		TxHash:      m.txId[:],
		OutputIndex: m.index,
	}, nil
}

func (m *mockTransactionInput) ToPlutusData() data.PlutusData {
	return data.NewConstr(0,
		data.NewByteString(m.txId[:]),
		data.NewInteger(big.NewInt(int64(m.index))),
	)
}

// Compile-time interface check
var _ StateManager = (*MockStateManager)(nil)
