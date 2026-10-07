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
	"errors"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"sort"

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
	stakeRegistrations map[ledger.RewardAccountKey]uint64

	// stakeCredentialDeposits tracks the original registration deposit. It is
	// intentionally separate from reward balances because protocol parameters
	// may change after registration.
	stakeCredentialDeposits map[ledger.RewardAccountKey]uint64

	// rewardAccounts tracks balances by full credential identity.
	rewardAccounts map[ledger.RewardAccountKey]uint64

	// poolRegistrations tracks registered pools
	poolRegistrations map[common.Blake2b224]bool

	// drepRegistrations tracks registered DReps by full credential
	// identity, holding the deposit recorded against each registration. A
	// nil value is a registration the fixture supplied no deposit for,
	// which is distinct from a recorded deposit of zero.
	drepRegistrations map[ledger.RewardAccountKey]*uint64

	// committeeMembers tracks committee members (cold key -> expiry epoch)
	committeeMembers map[ledger.RewardAccountKey]uint64

	// hotKeyAuthorizations tracks hot key authorizations (cold key -> hot key)
	hotKeyAuthorizations map[ledger.RewardAccountKey]common.Credential

	// committeeResignations tracks current and pending committee credentials
	// that have resigned.
	committeeResignations map[ledger.RewardAccountKey]bool
}

// drepSeedDeposit returns the deposit the initial state records for a DRep
// credential, or nil when the state supplies none. It is a pointer rather
// than a zero default so an unsupplied deposit stays distinguishable from a
// recorded deposit of zero.
func drepSeedDeposit(
	state *ParsedInitialState,
	key ledger.RewardAccountKey,
) *uint64 {
	deposit, found := state.DRepDeposits[key]
	if !found {
		return nil
	}
	return &deposit
}

// NewMockStateManager creates a new MockStateManager.
func NewMockStateManager() *MockStateManager {
	return &MockStateManager{
		govState:                NewGovernanceState(),
		utxos:                   make(map[string]common.Utxo),
		stakeRegistrations:      make(map[ledger.RewardAccountKey]uint64),
		stakeCredentialDeposits: make(map[ledger.RewardAccountKey]uint64),
		rewardAccounts:          make(map[ledger.RewardAccountKey]uint64),
		poolRegistrations:       make(map[common.Blake2b224]bool),
		drepRegistrations:       make(map[ledger.RewardAccountKey]*uint64),
		committeeMembers:        make(map[ledger.RewardAccountKey]uint64),
		hotKeyAuthorizations: make(
			map[ledger.RewardAccountKey]common.Credential,
		),
		committeeResignations: make(map[ledger.RewardAccountKey]bool),
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
	m.stakeRegistrations = make(map[ledger.RewardAccountKey]uint64)
	m.stakeCredentialDeposits = make(map[ledger.RewardAccountKey]uint64)
	m.rewardAccounts = make(map[ledger.RewardAccountKey]uint64)
	m.poolRegistrations = make(map[common.Blake2b224]bool)
	m.drepRegistrations = make(map[ledger.RewardAccountKey]*uint64)
	m.committeeMembers = make(map[ledger.RewardAccountKey]uint64)
	m.hotKeyAuthorizations = make(map[ledger.RewardAccountKey]common.Credential)
	m.committeeResignations = make(map[ledger.RewardAccountKey]bool)

	// Load stake registrations with reward balances. Prefer full credential
	// identity. Legacy hash-only registrations represent key credentials.
	if len(state.StakeRegistrationsByCredential) > 0 {
		for credential, registered := range state.StakeRegistrationsByCredential {
			if !registered {
				continue
			}
			balance, exists := state.RewardAccountBalances[credential]
			if !exists {
				balance = state.RewardAccounts[credential.Credential]
			}
			m.stakeRegistrations[credential] = balance
		}
	} else if len(state.RewardAccountBalances) > 0 {
		maps.Copy(m.stakeRegistrations, state.RewardAccountBalances)
	} else {
		for hash, registered := range state.StakeRegistrations {
			if !registered {
				continue
			}
			m.stakeRegistrations[ledger.RewardAccountKey{
				CredType:   common.CredentialTypeAddrKeyHash,
				Credential: hash,
			}] = state.RewardAccounts[hash]
		}
	}
	// Preserve the parsed registration deposit when the initial state exposes
	// it. Do not fabricate a current-parameter value for older state formats;
	// the original deposit is not recoverable from registration alone.
	for credential := range m.stakeRegistrations {
		if deposit, ok := state.StakeCredentialDeposits[credential]; ok {
			m.stakeCredentialDeposits[credential] = deposit
		}
	}
	if len(state.RewardAccountBalances) > 0 {
		maps.Copy(m.rewardAccounts, state.RewardAccountBalances)
	} else {
		for hash, balance := range state.RewardAccounts {
			m.rewardAccounts[ledger.RewardAccountKey{
				CredType:   common.CredentialTypeAddrKeyHash,
				Credential: hash,
			}] = balance
		}
	}
	// Preserve compatibility for callers that provide only a registration
	// view. Every registered account has a balance entry, including zero.
	for credential, balance := range m.stakeRegistrations {
		if _, exists := m.rewardAccounts[credential]; !exists {
			m.rewardAccounts[credential] = balance
		}
	}

	// Load pool registrations
	for hash, registered := range state.PoolRegistrations {
		if registered {
			m.poolRegistrations[hash] = true
		}
	}

	// Load DRep registrations. The typed set is authoritative; the legacy
	// hash list is its key-hash compatibility projection, matching
	// GovernanceState.RegisterDRep.
	for key, registered := range state.DRepRegistrationsByCredential {
		if !registered {
			continue
		}
		m.drepRegistrations[key] = drepSeedDeposit(state, key)
	}
	for _, hash := range state.DRepRegistrations {
		if hasCredentialHash(state.DRepRegistrationsByCredential, hash) {
			continue
		}
		key := ledger.RewardAccountKey{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: hash,
		}
		m.drepRegistrations[key] = drepSeedDeposit(state, key)
	}
	// Load committee members. Legacy entries represent key credentials, but a
	// hash already present in the typed state may be its compatibility projection.
	maps.Copy(m.committeeMembers, state.CommitteeMembersByCredential)
	for coldKey, expiry := range state.CommitteeMembers {
		if hasCredentialHash(m.committeeMembers, coldKey) {
			continue
		}
		m.committeeMembers[ledger.RewardAccountKey{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: coldKey,
		}] = expiry
	}

	// Load hot key authorizations using the same compatibility rule.
	maps.Copy(
		m.hotKeyAuthorizations,
		state.HotKeyAuthorizationsByCredential,
	)
	for coldKey, hotKey := range state.HotKeyAuthorizations {
		if hasCredentialHash(m.hotKeyAuthorizations, coldKey) {
			continue
		}
		m.hotKeyAuthorizations[ledger.RewardAccountKey{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: coldKey,
		}] = common.Credential{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: hotKey,
		}
	}
	maps.Copy(m.committeeResignations, state.CommitteeResignations)

	// Load governance state
	m.govState = NewGovernanceState()
	m.govState.LoadFromParsedState(state)
	m.syncRewardBalanceMirrors()

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

	// Validate withdrawals before mutating any UTxO or certificate state.
	//
	// Withdrawals are applied after this transaction's certificates, so the
	// balance a withdrawal is checked against is the one left by the
	// certificate sequence: a deregistration removes the account and a
	// registration resets it to zero. Project those effects here so an
	// insufficient withdrawal is rejected before anything is mutated, rather
	// than part-way through.
	projected := m.projectedRewardAccounts(tx.Certificates())
	withdrawals := make(map[ledger.RewardAccountKey]uint64)
	for rewardAccount, amount := range tx.Withdrawals() {
		if rewardAccount == nil || amount == nil {
			continue
		}
		credential, ok := rewardAccount.StakeCredential()
		if !ok {
			continue
		}
		key := ledger.NewRewardAccountKey(credential)
		withdrawals[key] += amount.Uint64()
	}
	for key, withdrawal := range withdrawals {
		balance, exists := projected(key)
		if exists && withdrawal > balance {
			return fmt.Errorf(
				"withdrawal amount %d exceeds reward account balance %d",
				withdrawal, balance,
			)
		}
	}

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

	// Process withdrawals against balances derived from initial state and
	// previously applied events, after pre-validation above.
	for rewardAccount, amount := range tx.Withdrawals() {
		if rewardAccount == nil || amount == nil {
			continue
		}
		credential, ok := rewardAccount.StakeCredential()
		if !ok {
			continue
		}
		key := ledger.NewRewardAccountKey(credential)
		balance, exists := m.rewardAccounts[key]
		if !exists {
			continue
		}
		withdrawal := amount.Uint64()
		if withdrawal > balance {
			return fmt.Errorf(
				"withdrawal amount %d exceeds reward account balance %d",
				withdrawal, balance,
			)
		}
		m.rewardAccounts[key] = balance - withdrawal
	}
	m.syncRewardBalanceMirrors()
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
				Deposit:         proposal.Deposit(),
				RemovedMembers:  make(map[ledger.RewardAccountKey]bool),
				ProposedMembers: make(map[common.Blake2b224]uint64),
				ProposedMembersByCredential: make(
					map[ledger.RewardAccountKey]uint64,
				),
			}
			rewardAccount := proposal.RewardAccount()
			if credential, ok := rewardAccount.StakeCredential(); ok {
				returnAccount := ledger.NewRewardAccountKey(credential)
				info.ReturnAccount = &returnAccount
			}

			// Extract action-specific data including parent action ID
			switch ga := action.(type) {
			case *common.UpdateCommitteeGovAction:
				if ga.ActionId != nil {
					key := fmt.Sprintf("%x#%d", ga.ActionId.TransactionId[:], ga.ActionId.GovActionIdx)
					info.ParentActionId = &key
				}
				for _, cred := range ga.Credentials {
					info.RemovedMembers[ledger.NewRewardAccountKey(cred)] = true
				}
				for cred, epoch := range ga.CredEpochs {
					if cred != nil {
						credentialKey := ledger.NewRewardAccountKey(*cred)
						info.ProposedMembersByCredential[credentialKey] = uint64(epoch)
					}
				}
				info.ProposedMembers = committeeMembersByHash(
					info.ProposedMembersByCredential,
				)
				info.ProposedThreshold = cloneBigRat(ga.Quorum.Rat)
			case *common.TreasuryWithdrawalGovAction:
				info.Withdrawals = make(
					map[ledger.RewardAccountKey]uint64,
					len(ga.Withdrawals),
				)
				for recipient, amount := range ga.Withdrawals {
					if recipient == nil {
						continue
					}
					if credential, ok := recipient.StakeCredential(); ok {
						info.Withdrawals[ledger.NewRewardAccountKey(credential)] += amount
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
		m.refreshDRepVoter(voter)
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
			credential := regCert.StakeCredential
			key := ledger.NewRewardAccountKey(credential)
			m.stakeRegistrations[key] = 0
			m.stakeCredentialDeposits[key] = keyDepositAmount(m.protocolParams)
			m.rewardAccounts[ledger.NewRewardAccountKey(credential)] = 0
			m.govState.RegisterStakeCredential(credential)
		}

	case common.CertificateTypeRegistration:
		if regCert, ok := cert.(*common.RegistrationCertificate); ok {
			credential := regCert.StakeCredential
			key := ledger.NewRewardAccountKey(credential)
			m.stakeRegistrations[key] = 0
			m.stakeCredentialDeposits[key] = nonNegativeDeposit(regCert.Amount)
			m.rewardAccounts[ledger.NewRewardAccountKey(credential)] = 0
			m.govState.RegisterStakeCredential(credential)
		}

	case common.CertificateTypeStakeRegistrationDelegation:
		// Combined registration + delegation (Conway)
		if regCert, ok := cert.(*common.StakeRegistrationDelegationCertificate); ok {
			credential := regCert.StakeCredential
			key := ledger.NewRewardAccountKey(credential)
			m.stakeRegistrations[key] = 0
			m.stakeCredentialDeposits[key] = nonNegativeDeposit(regCert.Amount)
			m.rewardAccounts[ledger.NewRewardAccountKey(credential)] = 0
			m.govState.RegisterStakeCredential(credential)
			m.govState.SetPoolDelegation(credential, regCert.PoolKeyHash)
		}

	case common.CertificateTypeVoteRegistrationDelegation:
		// Combined registration + vote delegation (Conway)
		if regCert, ok := cert.(*common.VoteRegistrationDelegationCertificate); ok {
			credential := regCert.StakeCredential
			key := ledger.NewRewardAccountKey(credential)
			m.stakeRegistrations[key] = 0
			m.stakeCredentialDeposits[key] = nonNegativeDeposit(regCert.Amount)
			m.rewardAccounts[ledger.NewRewardAccountKey(credential)] = 0
			m.govState.RegisterStakeCredential(credential)
			m.govState.SetDRepDelegation(
				credential,
				drepDelegation(regCert.Drep),
			)
		}

	case common.CertificateTypeStakeVoteRegistrationDelegation:
		// Combined registration + stake + vote delegation (Conway)
		if regCert, ok := cert.(*common.StakeVoteRegistrationDelegationCertificate); ok {
			credential := regCert.StakeCredential
			key := ledger.NewRewardAccountKey(credential)
			m.stakeRegistrations[key] = 0
			m.stakeCredentialDeposits[key] = nonNegativeDeposit(regCert.Amount)
			m.rewardAccounts[key] = 0
			m.govState.RegisterStakeCredential(credential)
			m.govState.SetDRepDelegation(
				credential,
				drepDelegation(regCert.Drep),
			)
			m.govState.SetPoolDelegation(credential, regCert.PoolKeyHash)
		}

	case common.CertificateTypeStakeDelegation:
		if delegationCert, ok := cert.(*common.StakeDelegationCertificate); ok &&
			delegationCert.StakeCredential != nil {
			m.govState.SetPoolDelegation(
				*delegationCert.StakeCredential,
				delegationCert.PoolKeyHash,
			)
		}

	case common.CertificateTypeVoteDelegation:
		if voteCert, ok := cert.(*common.VoteDelegationCertificate); ok {
			m.govState.SetDRepDelegation(
				voteCert.StakeCredential,
				drepDelegation(voteCert.Drep),
			)
		}

	case common.CertificateTypeStakeVoteDelegation:
		if voteCert, ok := cert.(*common.StakeVoteDelegationCertificate); ok {
			m.govState.SetDRepDelegation(
				voteCert.StakeCredential,
				drepDelegation(voteCert.Drep),
			)
			m.govState.SetPoolDelegation(
				voteCert.StakeCredential,
				voteCert.PoolKeyHash,
			)
		}

	case common.CertificateTypeStakeDeregistration:
		if deregCert, ok := cert.(*common.StakeDeregistrationCertificate); ok {
			m.deregisterStakeCredential(deregCert.StakeCredential)
		}

	case common.CertificateTypeDeregistration:
		if deregCert, ok := cert.(*common.DeregistrationCertificate); ok {
			m.deregisterStakeCredential(deregCert.StakeCredential)
		}

	case common.CertificateTypePoolRegistration:
		if poolCert, ok := cert.(*common.PoolRegistrationCertificate); ok {
			poolId := poolCert.Operator
			m.poolRegistrations[poolId] = true
			m.govState.RegisterPool(poolId)
			m.govState.SetPoolRewardAccount(
				poolId,
				ledger.RewardAccountKey{
					CredType:   common.CredentialTypeAddrKeyHash,
					Credential: poolCert.RewardAccount,
				},
			)
		}

	case common.CertificateTypePoolRetirement:
		if retireCert, ok := cert.(*common.PoolRetirementCertificate); ok {
			poolId := retireCert.PoolKeyHash
			retireEpoch := retireCert.Epoch
			m.govState.RetirePool(poolId, retireEpoch)
		}

	case common.CertificateTypeRegistrationDrep:
		if drepCert, ok := cert.(*common.RegistrationDrepCertificate); ok {
			credential := drepCert.DrepCredential
			// The certificate carries the deposit the DRep paid, and
			// that is the amount its deregistration must refund.
			deposit := uint64(0)
			if drepCert.Amount >= 0 {
				deposit = uint64(drepCert.Amount)
			}
			m.drepRegistrations[ledger.NewRewardAccountKey(credential)] = &deposit
			m.govState.RegisterDRepCredentialUntil(
				credential,
				m.drepActivityExpiry(),
			)
		}

	case common.CertificateTypeDeregistrationDrep:
		if drepCert, ok := cert.(*common.DeregistrationDrepCertificate); ok {
			credential := drepCert.DrepCredential
			key := ledger.NewRewardAccountKey(credential)
			m.govState.DeregisterDRepCredential(credential)
			if !m.govState.IsDRepCredentialRegistered(credential) {
				delete(m.drepRegistrations, key)
				if !hasRegisteredCredentialHash(
					m.govState.DRepRegistrationsByCredential,
					credential.Credential,
				) {
					delete(m.drepRegistrations, ledger.RewardAccountKey{
						CredType:   common.CredentialTypeAddrKeyHash,
						Credential: credential.Credential,
					})
				}
			}
		}

	case common.CertificateTypeUpdateDrep:
		if drepCert, ok := cert.(*common.UpdateDrepCertificate); ok {
			m.govState.refreshDRepCredentialUntil(
				drepCert.DrepCredential,
				m.drepActivityExpiry(),
			)
		}

	case common.CertificateTypeAuthCommitteeHot:
		if authCert, ok := cert.(*common.AuthCommitteeHotCertificate); ok {
			coldKey := ledger.NewRewardAccountKey(authCert.ColdCredential)
			if m.committeeResignations[coldKey] ||
				m.govState.CommitteeResignations[coldKey] {
				break
			}
			m.hotKeyAuthorizations[coldKey] = authCert.HotCredential
			m.govState.AuthorizeHotCredential(
				authCert.ColdCredential,
				authCert.HotCredential,
			)
		}

	case common.CertificateTypeResignCommitteeCold:
		if resignCert, ok := cert.(*common.ResignCommitteeColdCertificate); ok {
			coldKey := ledger.NewRewardAccountKey(resignCert.ColdCredential)
			delete(m.hotKeyAuthorizations, coldKey)
			m.committeeResignations[coldKey] = true
			m.govState.ResignCommitteeCredential(resignCert.ColdCredential)
		}

	default:
		// Other certificate types not relevant for state tracking
	}
}

func (m *MockStateManager) drepActivityExpiry() uint64 {
	period := uint64(0)
	if pp, ok := m.protocolParams.(*conway.ConwayProtocolParameters); ok {
		period = pp.DRepInactivityPeriod
	}
	expiry := m.currentEpoch + period
	if expiry < m.currentEpoch {
		return ^uint64(0)
	}
	return expiry
}

func (m *MockStateManager) refreshDRepVoter(voter *common.Voter) {
	if voter == nil {
		return
	}
	credential := common.Credential{
		Credential: common.Blake2b224(voter.Hash),
	}
	switch voter.Type {
	case common.VoterTypeDRepKeyHash:
		credential.CredType = common.CredentialTypeAddrKeyHash
	case common.VoterTypeDRepScriptHash:
		credential.CredType = common.CredentialTypeScriptHash
	default:
		return
	}
	m.govState.refreshDRepCredentialUntil(
		credential,
		m.drepActivityExpiry(),
	)
}

// projectedRewardAccounts returns a lookup for the reward-account balance each
// credential will hold once certs have been applied by processCertificate.
// It mirrors the reward-account effects of that function: every registration
// form resets the balance to zero and every deregistration form removes the
// account. Credentials the certificates do not touch keep their current
// balance.
func (m *MockStateManager) projectedRewardAccounts(
	certs []common.Certificate,
) func(ledger.RewardAccountKey) (uint64, bool) {
	type projectedBalance struct {
		balance uint64
		exists  bool
	}
	projected := make(map[ledger.RewardAccountKey]projectedBalance)
	for _, cert := range certs {
		var credential common.Credential
		registered := false
		switch cert := cert.(type) {
		case *common.StakeRegistrationCertificate:
			credential, registered = cert.StakeCredential, true
		case *common.RegistrationCertificate:
			credential, registered = cert.StakeCredential, true
		case *common.StakeRegistrationDelegationCertificate:
			credential, registered = cert.StakeCredential, true
		case *common.VoteRegistrationDelegationCertificate:
			credential, registered = cert.StakeCredential, true
		case *common.StakeVoteRegistrationDelegationCertificate:
			credential, registered = cert.StakeCredential, true
		case *common.StakeDeregistrationCertificate:
			credential = cert.StakeCredential
		case *common.DeregistrationCertificate:
			credential = cert.StakeCredential
		default:
			continue
		}
		projected[ledger.NewRewardAccountKey(credential)] = projectedBalance{
			exists: registered,
		}
	}
	return func(key ledger.RewardAccountKey) (uint64, bool) {
		if entry, ok := projected[key]; ok {
			return entry.balance, entry.exists
		}
		balance, exists := m.rewardAccounts[key]
		return balance, exists
	}
}

func (m *MockStateManager) deregisterStakeCredential(
	credential common.Credential,
) {
	credentialKey := ledger.NewRewardAccountKey(credential)
	delete(m.rewardAccounts, credentialKey)
	delete(m.stakeRegistrations, credentialKey)
	delete(m.stakeCredentialDeposits, credentialKey)
	m.govState.DeregisterStakeCredential(credential)
}

func keyDepositAmount(pp common.ProtocolParameters) uint64 {
	if conwayPP, ok := pp.(*conway.ConwayProtocolParameters); ok &&
		conwayPP != nil {
		return uint64(conwayPP.KeyDeposit)
	}
	provider, ok := pp.(interface{ KeyDepositAmount() *big.Int })
	if !ok || provider.KeyDepositAmount() == nil ||
		!provider.KeyDepositAmount().IsUint64() {
		return 0
	}
	return provider.KeyDepositAmount().Uint64()
}

func nonNegativeDeposit(amount int64) uint64 {
	if amount < 0 {
		return 0
	}
	return uint64(amount)
}

func checkedAddCoin(balance, amount uint64) (uint64, bool) {
	if amount > ^uint64(0)-balance {
		return 0, false
	}
	return balance + amount, true
}

func drepDelegation(drep common.Drep) common.Drep {
	return drep
}

// ProcessEpochBoundary implements StateManager.ProcessEpochBoundary.
func (m *MockStateManager) ProcessEpochBoundary(newEpoch uint64) error {
	staged, err := m.cloneForEpochBoundary()
	if err != nil {
		return err
	}
	if err := staged.processEpochBoundary(newEpoch); err != nil {
		return err
	}
	m.commitEpochBoundary(staged)
	return nil
}

func (m *MockStateManager) processEpochBoundary(newEpoch uint64) error {
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

	// Phase 1: Enact proposals that were ratified in previous epochs, in
	// identifier order. A proposal waits for the pass that enacts its parent;
	// proposals whose parent never becomes the root are enacted last.
	pending := m.govState.proposalIDs(func(p *ProposalState) bool {
		return p.RatifiedEpoch != nil && newEpoch > *p.RatifiedEpoch &&
			p.ActionType != common.GovActionTypeInfo
	})
	slices.Sort(pending)
	enact := func(id string) error {
		proposal := m.govState.Proposals[id]
		if proposal == nil {
			return errors.New("proposal is missing")
		}
		slot := m.govState.Roots.forAction(proposal.ActionType)
		chained := slot != nil && parentMatchesRoot(proposal, *slot)
		if err := m.enactProposal(id, proposal); err != nil {
			return err
		}
		if !chained {
			return nil
		}
		// Siblings chained off the same parent can no longer be enacted.
		var siblings []string
		for siblingID, sibling := range m.govState.Proposals {
			if sibling != nil && sibling.RatifiedEpoch == nil &&
				m.govState.Roots.forAction(sibling.ActionType) == slot &&
				parentMatchesRoot(sibling, proposal.ParentActionId) {
				siblings = append(siblings, siblingID)
			}
		}
		if err := m.removeWithDescendants(siblings); err != nil {
			return err
		}
		return nil
	}
	for len(pending) > 0 {
		var waiting []string
		for _, id := range pending {
			proposal := m.govState.Proposals[id]
			if proposal == nil {
				return fmt.Errorf("enact proposal %s: proposal is missing", id)
			}
			slot := m.govState.Roots.forAction(proposal.ActionType)
			if slot == nil || parentMatchesRoot(proposal, *slot) {
				if err := enact(id); err != nil {
					return fmt.Errorf("enact proposal %s: %w", id, err)
				}
			} else {
				waiting = append(waiting, id)
			}
		}
		if len(waiting) == len(pending) {
			for _, id := range waiting {
				if err := enact(id); err != nil {
					return fmt.Errorf("enact proposal %s: %w", id, err)
				}
			}
			break
		}
		pending = waiting
	}

	// Phase 2: Ratify proposals that meet threshold requirements
	if err := m.ratifyProposals(newEpoch); err != nil {
		return err
	}

	// Phase 3: Expire old proposals and return their deposits.
	if err := m.removeWithDescendants(m.govState.proposalIDs(
		func(p *ProposalState) bool { return newEpoch > p.ExpiresAfter },
	)); err != nil {
		return err
	}
	m.syncRewardBalanceMirrors()

	return nil
}

// removeProposal drops a proposal and returns its deposit.
func (m *MockStateManager) removeProposal(id string) error {
	proposal := m.govState.Proposals[id]
	if proposal == nil || proposal.Deposit == 0 {
		delete(m.govState.Proposals, id)
		return nil
	}
	if proposal.ReturnAccount != nil {
		if _, registered := m.stakeRegistrations[*proposal.ReturnAccount]; registered {
			balance := m.rewardAccounts[*proposal.ReturnAccount]
			refunded, ok := checkedAddCoin(balance, proposal.Deposit)
			if !ok {
				return errors.New("proposal deposit refund overflows reward account")
			}
			delete(m.govState.Proposals, id)
			m.rewardAccounts[*proposal.ReturnAccount] = refunded
			return nil
		}
	}
	refunded, ok := checkedAddCoin(m.govState.Treasury, proposal.Deposit)
	if !ok {
		return errors.New("proposal deposit refund overflows treasury")
	}
	delete(m.govState.Proposals, id)
	m.govState.Treasury = refunded
	return nil
}

// removeWithDescendants removes the proposals and every proposal chained off
// them, returning each deposit.
func (m *MockStateManager) removeWithDescendants(ids []string) error {
	removed := make(map[string]bool, len(ids))
	for _, id := range ids {
		removed[id] = true
	}
	for grew := true; grew; {
		grew = false
		for id, p := range m.govState.Proposals {
			if p != nil && !removed[id] && p.ParentActionId != nil &&
				removed[*p.ParentActionId] {
				removed[id] = true
				grew = true
			}
		}
	}
	for id := range removed {
		if err := m.removeProposal(id); err != nil {
			return fmt.Errorf("remove proposal %s: %w", id, err)
		}
	}
	return nil
}

func (m *MockStateManager) cloneForEpochBoundary() (*MockStateManager, error) {
	if conwayParams, ok := m.protocolParams.(*conway.ConwayProtocolParameters); ok &&
		conwayParams == nil {
		return nil, errors.New("conway protocol parameters unavailable")
	}
	staged := *m
	staged.protocolParams = deepCopyPParams(m.protocolParams)
	staged.govState = cloneGovernanceState(m.govState)
	staged.poolRegistrations = maps.Clone(m.poolRegistrations)
	staged.stakeCredentialDeposits = maps.Clone(m.stakeCredentialDeposits)
	staged.stakeRegistrations = maps.Clone(m.stakeRegistrations)
	staged.rewardAccounts = maps.Clone(m.rewardAccounts)
	staged.committeeMembers = maps.Clone(m.committeeMembers)
	staged.hotKeyAuthorizations = maps.Clone(m.hotKeyAuthorizations)
	staged.committeeResignations = maps.Clone(m.committeeResignations)
	return &staged, nil
}

func (m *MockStateManager) commitEpochBoundary(staged *MockStateManager) {
	m.currentEpoch = staged.currentEpoch
	m.poolRegistrations = staged.poolRegistrations
	m.stakeCredentialDeposits = staged.stakeCredentialDeposits
	m.stakeRegistrations = staged.stakeRegistrations
	m.rewardAccounts = staged.rewardAccounts
	m.committeeMembers = staged.committeeMembers
	m.hotKeyAuthorizations = staged.hotKeyAuthorizations
	m.committeeResignations = staged.committeeResignations
	m.protocolParams = commitProtocolParameters(
		m.protocolParams,
		staged.protocolParams,
	)
	if m.govState == nil {
		m.govState = staged.govState
	} else {
		*m.govState = *staged.govState
	}
}

func commitProtocolParameters(
	current common.ProtocolParameters,
	staged common.ProtocolParameters,
) common.ProtocolParameters {
	currentConway, currentOK := current.(*conway.ConwayProtocolParameters)
	stagedConway, stagedOK := staged.(*conway.ConwayProtocolParameters)
	if currentOK && stagedOK {
		*currentConway = *stagedConway
		return currentConway
	}
	return staged
}

func cloneGovernanceState(state *GovernanceState) *GovernanceState {
	if state == nil {
		return nil
	}
	cloned := *state
	cloned.CommitteeThreshold = cloneBigRat(state.CommitteeThreshold)

	memberCopies := make(map[*CommitteeMemberInfo]*CommitteeMemberInfo)
	cloneMember := func(member *CommitteeMemberInfo) *CommitteeMemberInfo {
		if member == nil {
			return nil
		}
		if clonedMember, ok := memberCopies[member]; ok {
			return clonedMember
		}
		clonedMember := *member
		if member.HotCredential != nil {
			hotCredential := *member.HotCredential
			clonedMember.HotCredential = &hotCredential
		}
		if member.HotKey != nil {
			hotKey := *member.HotKey
			clonedMember.HotKey = &hotKey
		}
		memberCopies[member] = &clonedMember
		return &clonedMember
	}
	cloned.CommitteeMembers = make(
		map[common.Blake2b224]*CommitteeMemberInfo,
		len(state.CommitteeMembers),
	)
	for credential, member := range state.CommitteeMembers {
		cloned.CommitteeMembers[credential] = cloneMember(member)
	}
	cloned.CommitteeMembersByCredential = make(
		map[ledger.RewardAccountKey]*CommitteeMemberInfo,
		len(state.CommitteeMembersByCredential),
	)
	for credential, member := range state.CommitteeMembersByCredential {
		cloned.CommitteeMembersByCredential[credential] = cloneMember(member)
	}

	cloned.DRepRegistrations = maps.Clone(state.DRepRegistrations)
	cloned.DRepRegistrationsByCredential = maps.Clone(
		state.DRepRegistrationsByCredential,
	)
	cloned.DRepExpiries = maps.Clone(state.DRepExpiries)
	cloned.DRepDelegations = maps.Clone(state.DRepDelegations)
	cloned.DRepDelegationsByCredential = maps.Clone(
		state.DRepDelegationsByCredential,
	)
	cloned.HotKeyAuthorizations = maps.Clone(state.HotKeyAuthorizations)
	cloned.HotKeyAuthorizationsByCredential = maps.Clone(
		state.HotKeyAuthorizationsByCredential,
	)
	cloned.CommitteeResignations = maps.Clone(state.CommitteeResignations)
	cloned.StakeRegistrations = maps.Clone(state.StakeRegistrations)
	cloned.StakeRegistrationsByCredential = maps.Clone(
		state.StakeRegistrationsByCredential,
	)
	cloned.PoolRegistrations = maps.Clone(state.PoolRegistrations)
	cloned.PoolRewardAccounts = maps.Clone(state.PoolRewardAccounts)
	cloned.PoolDelegationsByCredential = maps.Clone(
		state.PoolDelegationsByCredential,
	)
	cloned.PoolRetirements = maps.Clone(state.PoolRetirements)
	cloned.RewardAccounts = maps.Clone(state.RewardAccounts)
	cloned.RewardAccountBalances = maps.Clone(state.RewardAccountBalances)
	cloned.Proposals = make(map[string]*ProposalState, len(state.Proposals))
	for id, proposal := range state.Proposals {
		cloned.Proposals[id] = cloneProposalState(proposal)
	}
	cloned.EnactedProposals = maps.Clone(state.EnactedProposals)
	cloned.Roots = cloneProposalRoots(state.Roots)
	cloned.Constitution = cloneConstitutionInfo(state.Constitution)
	return &cloned
}

func cloneProposalState(proposal *ProposalState) *ProposalState {
	if proposal == nil {
		return nil
	}
	cloned := *proposal
	cloned.ProposedThreshold = cloneBigRat(proposal.ProposedThreshold)
	cloned.Votes = maps.Clone(proposal.Votes)
	cloned.RemovedMembers = maps.Clone(proposal.RemovedMembers)
	cloned.ProposedMembers = maps.Clone(proposal.ProposedMembers)
	cloned.Withdrawals = maps.Clone(proposal.Withdrawals)
	cloned.ProposedMembersByCredential = maps.Clone(
		proposal.ProposedMembersByCredential,
	)
	cloned.PolicyHash = slices.Clone(proposal.PolicyHash)
	if proposal.ParentActionId != nil {
		parentActionID := *proposal.ParentActionId
		cloned.ParentActionId = &parentActionID
	}
	if proposal.ReturnAccount != nil {
		returnAccount := *proposal.ReturnAccount
		cloned.ReturnAccount = &returnAccount
	}
	if proposal.ProtocolVersion != nil {
		protocolVersion := *proposal.ProtocolVersion
		cloned.ProtocolVersion = &protocolVersion
	}
	if proposal.ParameterUpdate != nil {
		parameterUpdate := *proposal.ParameterUpdate
		if proposal.ParameterUpdate.CostModels != nil {
			parameterUpdate.CostModels = make(
				map[uint][]int64,
				len(proposal.ParameterUpdate.CostModels),
			)
			for version, costModel := range proposal.ParameterUpdate.CostModels {
				parameterUpdate.CostModels[version] = slices.Clone(costModel)
			}
		}
		cloned.ParameterUpdate = &parameterUpdate
	}
	if proposal.RatifiedEpoch != nil {
		ratifiedEpoch := *proposal.RatifiedEpoch
		cloned.RatifiedEpoch = &ratifiedEpoch
	}
	return &cloned
}

func cloneProposalRoots(roots ProposalRoots) ProposalRoots {
	cloned := roots
	if roots.ProtocolParameters != nil {
		root := *roots.ProtocolParameters
		cloned.ProtocolParameters = &root
	}
	if roots.HardFork != nil {
		root := *roots.HardFork
		cloned.HardFork = &root
	}
	if roots.ConstitutionalCommittee != nil {
		root := *roots.ConstitutionalCommittee
		cloned.ConstitutionalCommittee = &root
	}
	if roots.Constitution != nil {
		root := *roots.Constitution
		cloned.Constitution = &root
	}
	return cloned
}

func cloneConstitutionInfo(constitution *ConstitutionInfo) *ConstitutionInfo {
	if constitution == nil {
		return nil
	}
	cloned := *constitution
	cloned.AnchorHash = slices.Clone(constitution.AnchorHash)
	cloned.PolicyHash = slices.Clone(constitution.PolicyHash)
	return &cloned
}

// enactProposal processes a ratified proposal by updating the appropriate root.
func (m *MockStateManager) enactProposal(
	id string,
	proposal *ProposalState,
) error {
	var creditedBalances map[ledger.RewardAccountKey]uint64
	if proposal.ActionType == common.GovActionTypeTreasuryWithdrawal {
		creditedBalances = make(map[ledger.RewardAccountKey]uint64)
		if _, ok := proposal.withdrawalTotalWithin(m.govState.Treasury); !ok {
			return errors.New("treasury withdrawal exceeds available treasury")
		}
		for account, amount := range proposal.Withdrawals {
			if _, registered := m.stakeRegistrations[account]; !registered {
				continue
			}
			credited, ok := checkedAddCoin(m.rewardAccounts[account], amount)
			if !ok {
				return errors.New("treasury withdrawal overflows reward account")
			}
			creditedBalances[account] = credited
		}
	}
	if slot := m.govState.Roots.forAction(proposal.ActionType); slot != nil {
		*slot = &id
	}
	//exhaustive:ignore
	switch proposal.ActionType {
	case common.GovActionTypeNewConstitution:
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
		// Apply parameter updates to protocol parameters
		if proposal.ParameterUpdate != nil {
			if conwayPP, ok := m.protocolParams.(*conway.ConwayProtocolParameters); ok {
				applyParameterUpdate(conwayPP, proposal.ParameterUpdate)
			}
		}
	case common.GovActionTypeHardForkInitiation:
		if conwayPP, ok := m.protocolParams.(*conway.ConwayProtocolParameters); ok &&
			conwayPP != nil && proposal.ProtocolVersion != nil {
			conwayPP.ProtocolVersion = common.ProtocolParametersProtocolVersion{
				Major: proposal.ProtocolVersion.Major,
				Minor: proposal.ProtocolVersion.Minor,
			}
		}
	case common.GovActionTypeTreasuryWithdrawal:
		for account, amount := range proposal.Withdrawals {
			if credited, registered := creditedBalances[account]; registered &&
				amount <= m.govState.Treasury {
				m.rewardAccounts[account] = credited
				m.govState.Treasury -= amount
			}
		}
		m.syncRewardBalanceMirrors()
	case common.GovActionTypeNoConfidence:
		m.govState.CommitteeThreshold = nil
		clear(m.govState.CommitteeMembers)
		clear(m.govState.CommitteeMembersByCredential)
		clear(m.committeeMembers)
		clear(m.govState.HotKeyAuthorizations)
		clear(m.govState.HotKeyAuthorizationsByCredential)
		clear(m.hotKeyAuthorizations)
		clear(m.govState.CommitteeResignations)
		clear(m.committeeResignations)
	case common.GovActionTypeUpdateCommittee:
		if proposal.ProposedThreshold != nil {
			m.govState.CommitteeThreshold = cloneBigRat(
				proposal.ProposedThreshold,
			)
		}
		for coldKey := range proposal.RemovedMembers {
			delete(m.govState.CommitteeMembersByCredential, coldKey)
			delete(m.govState.CommitteeMembers, coldKey.Credential)
			delete(m.committeeMembers, coldKey)
			delete(m.govState.HotKeyAuthorizationsByCredential, coldKey)
			delete(
				m.govState.HotKeyAuthorizations,
				coldKey.Credential,
			)
			delete(m.hotKeyAuthorizations, coldKey)
			delete(m.govState.CommitteeResignations, coldKey)
			delete(m.committeeResignations, coldKey)
		}
		proposedMembers := maps.Clone(proposal.ProposedMembersByCredential)
		if proposedMembers == nil {
			proposedMembers = make(map[ledger.RewardAccountKey]uint64)
		}
		for coldKey, expiry := range proposal.ProposedMembers {
			if hasCredentialHash(proposedMembers, coldKey) {
				continue
			}
			proposedMembers[ledger.RewardAccountKey{
				CredType:   common.CredentialTypeAddrKeyHash,
				Credential: coldKey,
			}] = expiry
		}
		for coldKey, expiry := range proposedMembers {
			member := &CommitteeMemberInfo{
				ColdCredential: coldKey.AsCredential(),
				ColdKey:        coldKey.Credential,
				ExpiryEpoch:    expiry,
				Resigned:       m.govState.CommitteeResignations[coldKey],
			}
			if hotKey, ok := m.govState.HotKeyAuthorizationsByCredential[coldKey]; ok &&
				!member.Resigned {
				hotCredential := hotKey
				member.HotCredential = &hotCredential
				hotHash := hotKey.Credential
				member.HotKey = &hotHash
			}
			m.govState.CommitteeMembersByCredential[coldKey] = member
			m.committeeMembers[coldKey] = expiry
		}
		m.govState.syncLegacyCommitteeMembers()
		m.govState.syncLegacyHotKeyAuthorizations()
	}

	// Mark as enacted and remove from active proposals
	m.govState.EnactedProposals[id] = true
	return m.removeProposal(id)
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

// GetStateSnapshot implements StateSnapshotProvider.
func (m *MockStateManager) GetStateSnapshot() *StateSnapshot {
	utxoIDs := make([]string, 0, len(m.utxos))
	for id := range m.utxos {
		utxoIDs = append(utxoIDs, id)
	}
	sort.Strings(utxoIDs)
	registrations := make(
		map[ledger.RewardAccountKey]bool,
		len(m.stakeRegistrations),
	)
	for credential := range m.stakeRegistrations {
		registrations[credential] = true
	}
	return &StateSnapshot{
		CurrentEpoch:                   m.currentEpoch,
		UtxoIDs:                        utxoIDs,
		StakeRegistrationsByCredential: registrations,
		RewardAccountBalances:          maps.Clone(m.rewardAccounts),
		StakeCredentialDeposits:        maps.Clone(m.stakeCredentialDeposits),
		PoolRegistrations:              maps.Clone(m.poolRegistrations),
		Governance:                     cloneGovernanceState(m.govState),
	}
}

// SetRewardBalances implements StateManager.SetRewardBalances.
func (m *MockStateManager) SetRewardBalances(
	balances map[common.Blake2b224]uint64,
) {
	for credential := range m.rewardAccounts {
		if balance, exists := balances[credential.Credential]; exists {
			m.rewardAccounts[credential] = balance
		}
	}
	m.syncRewardBalanceMirrors()
}

// SetRewardAccountBalances updates currently registered reward balances by
// full credential identity without changing registration state.
func (m *MockStateManager) SetRewardAccountBalances(
	balances map[ledger.RewardAccountKey]uint64,
) {
	for credential := range m.rewardAccounts {
		if balance, exists := balances[credential]; exists {
			m.rewardAccounts[credential] = balance
		}
	}
	m.syncRewardBalanceMirrors()
}

func (m *MockStateManager) syncRewardBalanceMirrors() {
	legacyBalances := rewardBalancesByHash(m.rewardAccounts)
	for cred := range m.stakeRegistrations {
		if balance, exists := m.rewardAccounts[cred]; exists {
			m.stakeRegistrations[cred] = balance
		}
	}
	if m.govState != nil {
		m.govState.RewardAccountBalances = maps.Clone(m.rewardAccounts)
		m.govState.RewardAccounts = legacyBalances
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
	m.stakeRegistrations = make(map[ledger.RewardAccountKey]uint64)
	m.stakeCredentialDeposits = make(map[ledger.RewardAccountKey]uint64)
	m.rewardAccounts = make(map[ledger.RewardAccountKey]uint64)
	m.poolRegistrations = make(map[common.Blake2b224]bool)
	m.drepRegistrations = make(map[ledger.RewardAccountKey]*uint64)
	m.committeeMembers = make(map[ledger.RewardAccountKey]uint64)
	m.hotKeyAuthorizations = make(map[ledger.RewardAccountKey]common.Credential)
	m.committeeResignations = make(map[ledger.RewardAccountKey]bool)
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

	// Reward-account entries carry both registration and balance. Every
	// registration path above creates an entry, including registered-zero
	// accounts, so using the credential-aware builder avoids fabricating a
	// key credential for a registered script account.
	builder.WithRewardAccountCredentialBalances(m.rewardAccounts)
	builder.WithStakeCredentialDeposits(m.stakeCredentialDeposits)

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
	drepRegs := maps.Clone(m.drepRegistrations) // capture for closure
	builder.WithDRepRegistration(
		func(cred common.Credential) (*common.DRepRegistration, error) {
			deposit, found := drepRegs[ledger.NewRewardAccountKey(cred)]
			if !found {
				return nil, nil
			}
			return &common.DRepRegistration{
				Credential: cred,
				Deposit:    deposit,
			}, nil
		},
	)
	drepDelegations := m.govState.DRepDelegationsByCredential
	legacyDRepDelegations := m.govState.DRepDelegations
	builder.WithDRepDelegation(
		func(cred common.Credential) (*common.Drep, error) {
			delegation, ok := drepDelegations[ledger.NewRewardAccountKey(cred)]
			if !ok && len(drepDelegations) == 0 {
				delegation, ok = legacyDRepDelegations[cred.Credential]
			}
			if !ok {
				return nil, nil
			}
			delegation.Credential = append(
				[]byte(nil),
				delegation.Credential...)
			return &delegation, nil
		},
	)

	// Set up committee member lookup
	committeeMembers := m.committeeMembers  // capture for closure
	hotKeyAuth := m.hotKeyAuthorizations    // capture for closure
	resignations := m.committeeResignations // capture for closure
	proposals := m.govState.Proposals       // capture for closure
	currentEpoch := m.govState.CurrentEpoch
	legacyMembersByHash := make(map[common.Blake2b224]common.CommitteeMember)
	ambiguousMemberHashes := make(map[common.Blake2b224]bool)
	for coldKey, expiry := range committeeMembers {
		if currentEpoch > expiry {
			continue
		}
		hash := coldKey.Credential
		if ambiguousMemberHashes[hash] {
			continue
		}
		if _, exists := legacyMembersByHash[hash]; exists {
			delete(legacyMembersByHash, hash)
			ambiguousMemberHashes[hash] = true
			continue
		}
		member := common.CommitteeMember{
			ColdKey:     hash,
			ExpiryEpoch: expiry,
			Resigned:    resignations[coldKey],
		}
		if hotKey, ok := hotKeyAuth[coldKey]; ok && !member.Resigned {
			hotHash := hotKey.Credential
			member.HotKey = &hotHash
		}
		legacyMembersByHash[hash] = member
	}
	legacyMembers := make([]common.CommitteeMember, 0, len(legacyMembersByHash))
	for _, member := range legacyMembersByHash {
		legacyMembers = append(legacyMembers, member)
	}
	builder.WithCommitteeMembers(legacyMembers)
	//nolint:unparam // The ledger-state callback contract requires an error.
	credentialMember := func(
		coldCredential common.Credential,
	) (*common.CommitteeMember, error) {
		coldKey := ledger.NewRewardAccountKey(coldCredential)
		// Check current members first
		if expiry, ok := committeeMembers[coldKey]; ok &&
			currentEpoch <= expiry {
			member := &common.CommitteeMember{
				ColdKey:     coldCredential.Credential,
				ExpiryEpoch: expiry,
				Resigned:    resignations[coldKey],
			}
			// Add hot key if authorized
			if hotKey, hasHot := hotKeyAuth[coldKey]; hasHot &&
				!member.Resigned {
				hotHash := hotKey.Credential
				member.HotKey = &hotHash
			}
			return member, nil
		}
		// Check members proposed by pending UpdateCommittee actions.
		for _, proposal := range proposals {
			if !isActiveUpdateCommitteeProposal(proposal, currentEpoch) {
				continue
			}
			if expiry, ok := proposal.ProposedMembersByCredential[coldKey]; ok &&
				currentEpoch <= expiry {
				return &common.CommitteeMember{
					ColdKey:     coldCredential.Credential,
					ExpiryEpoch: expiry,
					Resigned:    resignations[coldKey],
				}, nil
			}
			if coldCredential.CredType == common.CredentialTypeAddrKeyHash &&
				!hasCredentialHash(
					proposal.ProposedMembersByCredential,
					coldCredential.Credential,
				) {
				if expiry, ok := proposal.ProposedMembers[coldCredential.Credential]; ok &&
					currentEpoch <= expiry {
					return &common.CommitteeMember{
						ColdKey:     coldCredential.Credential,
						ExpiryEpoch: expiry,
						Resigned:    resignations[coldKey],
					}, nil
				}
			}
		}
		return nil, nil
	}
	builder.WithCommitteeCredentialMember(credentialMember)
	builder.WithCommitteeHotCredentialMember(
		func(hotCredential common.Credential) (*common.CommitteeMember, error) {
			coldKeys := make([]ledger.RewardAccountKey, 0, len(hotKeyAuth))
			for coldKey := range hotKeyAuth {
				coldKeys = append(coldKeys, coldKey)
			}
			sort.Slice(coldKeys, func(i, j int) bool {
				if coldKeys[i].CredType != coldKeys[j].CredType {
					return coldKeys[i].CredType < coldKeys[j].CredType
				}
				return string(coldKeys[i].Credential[:]) <
					string(coldKeys[j].Credential[:])
			})
			for _, coldKey := range coldKeys {
				authorizedHotCredential := hotKeyAuth[coldKey]
				if authorizedHotCredential.CredType != hotCredential.CredType ||
					authorizedHotCredential.Credential != hotCredential.Credential ||
					resignations[coldKey] {
					continue
				}
				member, err := credentialMember(coldKey.AsCredential())
				if err != nil {
					return nil, err
				}
				if member == nil {
					continue
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
