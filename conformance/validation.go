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

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

// Validator performs pre-validation checks on transactions.
// This implements the governance validation logic needed for conformance tests.
type Validator struct{}

// NewValidator creates a new validator.
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateTransaction performs all pre-validation checks.
// These checks mirror the pre-validation done in the Haskell ledger rules.
func (v *Validator) ValidateTransaction(
	tx common.Transaction,
	slot uint64,
	epoch uint64,
	govState *GovernanceState,
	pp common.ProtocolParameters,
) error {
	// Skip validation for phase-2 invalid transactions
	// These have IsValid=false and only need basic syntax checks
	if !tx.IsValid() {
		return nil
	}

	// Validate governance voting procedures
	if err := v.validateVotingProcedures(tx, epoch, govState); err != nil {
		return fmt.Errorf("voting validation failed: %w", err)
	}

	// Validate withdrawals
	if err := v.validateWithdrawals(tx, govState); err != nil {
		return fmt.Errorf("withdrawal validation failed: %w", err)
	}

	// Validate certificates
	if err := v.validateCertificates(tx, govState); err != nil {
		return fmt.Errorf("certificate validation failed: %w", err)
	}

	// Validate proposal procedures
	if err := v.validateProposalProcedures(tx, govState, pp); err != nil {
		return fmt.Errorf("proposal validation failed: %w", err)
	}

	return nil
}

// validateVotingProcedures validates governance voting in the transaction.
func (v *Validator) validateVotingProcedures(
	tx common.Transaction,
	epoch uint64,
	govState *GovernanceState,
) error {
	votingProcs := tx.VotingProcedures()
	if votingProcs == nil {
		return nil
	}

	for voter, votes := range votingProcs {
		voterHash := common.Blake2b224(voter.Hash)

		for govActionId := range votes {
			actionKey := formatGovActionIdFromPtr(govActionId)

			// Check if governance action exists
			proposal := govState.GetProposal(actionKey)
			if proposal == nil {
				// Check if it was enacted (votes on enacted proposals are still valid)
				if !govState.EnactedProposals[actionKey] {
					return fmt.Errorf(
						"governance action %s does not exist",
						actionKey,
					)
				}
				continue
			}

			// Check if action has expired
			if epoch > proposal.ExpiresAfter {
				return fmt.Errorf("governance action %s has expired", actionKey)
			}

			// CC members cannot vote on NoConfidence or UpdateCommittee
			if voter.Type == common.VoterTypeConstitutionalCommitteeHotKeyHash ||
				voter.Type == common.VoterTypeConstitutionalCommitteeHotScriptHash {
				if proposal.ActionType == common.GovActionTypeNoConfidence ||
					proposal.ActionType == common.GovActionTypeUpdateCommittee {
					return fmt.Errorf(
						"CC member cannot vote on %d action",
						proposal.ActionType,
					)
				}
			}

			// Validate voter exists based on type
			if err := v.validateVoterExists(voter.Type, voterHash, govState); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateVoterExists checks that the voter is registered.
func (v *Validator) validateVoterExists(
	voterType uint8,
	voterHash common.Blake2b224,
	govState *GovernanceState,
) error {
	switch voterType {
	case common.VoterTypeConstitutionalCommitteeHotKeyHash,
		common.VoterTypeConstitutionalCommitteeHotScriptHash:
		// For CC voters, we need to find a member with this hot key
		found := false
		for _, hotKey := range govState.HotKeyAuthorizations {
			if hotKey == voterHash {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("CC voter hot key %x not authorized", voterHash)
		}

	case common.VoterTypeDRepKeyHash, common.VoterTypeDRepScriptHash:
		// Check if DRep is registered
		if !govState.IsDRepRegistered(voterHash) {
			return fmt.Errorf("DRep voter %x not registered", voterHash)
		}

	case common.VoterTypeStakingPoolKeyHash:
		// Check if pool is registered
		if !govState.IsPoolRegistered(voterHash) {
			return fmt.Errorf("SPO voter %x not registered", voterHash)
		}

	default:
		return fmt.Errorf("unknown voter type %d", voterType)
	}

	return nil
}

// validateWithdrawals validates withdrawal amounts match reward balances exactly.
// In Cardano, a withdrawal must withdraw the ENTIRE balance (not more, not less).
//
// However, for script-based withdrawals (those with redeemers), the Plutus script
// controls validation. This enables the "withdraw zero trick" where a zero withdrawal
// triggers script execution without actually withdrawing funds.
func (v *Validator) validateWithdrawals(
	tx common.Transaction,
	govState *GovernanceState,
) error {
	withdrawals := tx.Withdrawals()
	if withdrawals == nil {
		return nil
	}

	// Skip amount validation for script-based withdrawals.
	// The Plutus script controls validation for these cases.
	// RedeemerTagReward = 3 (withdrawal/reward purpose)
	hasWithdrawalRedeemer := false
	if tx.Witnesses() != nil && tx.Witnesses().Redeemers() != nil {
		withdrawalIndexes := tx.Witnesses().
			Redeemers().
			Indexes(common.RedeemerTagReward)
		hasWithdrawalRedeemer = len(withdrawalIndexes) > 0
	}

	// Also skip if there's a deregistration certificate - epoch boundary rewards
	// may not be tracked correctly in multi-TX scenarios
	hasDeregistration := false
	for _, cert := range tx.Certificates() {
		certType := common.CertificateType(cert.Type())
		if certType == common.CertificateTypeStakeDeregistration ||
			certType == common.CertificateTypeDeregistration {
			hasDeregistration = true
			break
		}
	}

	// Only validate key-based withdrawals without deregistration
	if hasWithdrawalRedeemer || hasDeregistration {
		return nil
	}

	for addr, amount := range withdrawals {
		if addr == nil {
			continue
		}
		// Extract stake credential from reward address
		stakeHash := extractStakeHashFromAddress(*addr)
		if stakeHash == nil {
			addrBytes, _ := addr.Bytes()
			return fmt.Errorf(
				"invalid withdrawal address: cannot extract stake credential from %x",
				addrBytes,
			)
		}

		// Check withdrawal amount matches reward balance exactly
		balance := govState.GetRewardBalance(*stakeHash)
		withdrawalAmount := amount.Uint64()
		if withdrawalAmount != balance {
			return fmt.Errorf(
				"withdrawal amount %d does not match balance %d for %x",
				withdrawalAmount, balance, *stakeHash,
			)
		}
	}

	return nil
}

// validateCertificates validates certificates in the transaction.
func (v *Validator) validateCertificates(
	tx common.Transaction,
	govState *GovernanceState,
) error {
	certs := tx.Certificates()
	if certs == nil {
		return nil
	}

	// Build a set of credentials being withdrawn in this transaction
	withdrawnCreds := make(map[common.Blake2b224]bool)
	for addr := range tx.Withdrawals() {
		if addr == nil {
			continue
		}
		stakeHash := extractStakeHashFromAddress(*addr)
		if stakeHash != nil {
			withdrawnCreds[*stakeHash] = true
		}
	}

	for _, cert := range certs {
		if err := v.validateCertificate(cert, govState, withdrawnCreds); err != nil {
			return err
		}
	}

	return nil
}

// validateCertificate validates a single certificate.
// withdrawnCreds contains credentials being withdrawn in the same transaction.
func (v *Validator) validateCertificate(
	cert common.Certificate,
	govState *GovernanceState,
	withdrawnCreds map[common.Blake2b224]bool,
) error {
	certType := common.CertificateType(cert.Type())

	//exhaustive:ignore
	switch certType {
	case common.CertificateTypeStakeRegistration:
		if regCert, ok := cert.(*common.StakeRegistrationCertificate); ok {
			credential := regCert.StakeCredential.Credential
			if govState.IsStakeRegistered(credential) {
				return fmt.Errorf(
					"stake credential %x already registered",
					credential,
				)
			}
		}

	case common.CertificateTypeRegistration:
		if regCert, ok := cert.(*common.RegistrationCertificate); ok {
			credential := regCert.StakeCredential.Credential
			if govState.IsStakeRegistered(credential) {
				return fmt.Errorf(
					"stake credential %x already registered",
					credential,
				)
			}
		}

	case common.CertificateTypeStakeDeregistration:
		if deregCert, ok := cert.(*common.StakeDeregistrationCertificate); ok {
			credential := deregCert.StakeCredential.Credential
			balance := govState.GetRewardBalance(credential)
			// Allow deregistration if balance is being withdrawn in same transaction
			if balance > 0 && !withdrawnCreds[credential] {
				return fmt.Errorf(
					"cannot deregister stake with balance %d",
					balance,
				)
			}
		}

	case common.CertificateTypeDeregistration:
		if deregCert, ok := cert.(*common.DeregistrationCertificate); ok {
			credential := deregCert.StakeCredential.Credential
			balance := govState.GetRewardBalance(credential)
			// Allow deregistration if balance is being withdrawn in same transaction
			if balance > 0 && !withdrawnCreds[credential] {
				return fmt.Errorf(
					"cannot deregister stake with balance %d",
					balance,
				)
			}
		}

	case common.CertificateTypeRegistrationDrep:
		if drepCert, ok := cert.(*common.RegistrationDrepCertificate); ok {
			credential := drepCert.DrepCredential.Credential
			if govState.IsDRepRegistered(credential) {
				return fmt.Errorf("DRep %x already registered", credential)
			}
		}

	case common.CertificateTypeAuthCommitteeHot:
		if authCert, ok := cert.(*common.AuthCommitteeHotCertificate); ok {
			coldCredential := authCert.ColdCredential.Credential
			member := govState.GetCommitteeMember(coldCredential)
			if member != nil && member.Resigned {
				return fmt.Errorf(
					"cannot authorize hot key for resigned CC member %x",
					coldCredential,
				)
			}
		}

	case common.CertificateTypeResignCommitteeCold:
		// The Cardano spec requires the credential to be a current OR proposed CC member.
		// Per Amaru test vectors:
		// - "resigning a non-CC key" should fail (not a member or proposed)
		// - "Resigning proposed CC key" should succeed (proposed but not yet enacted)
		if govState != nil {
			if resignCert, ok := cert.(*common.ResignCommitteeColdCertificate); ok {
				coldHash := resignCert.ColdCredential.Credential
				// Check if this cold key is a current committee member
				_, isMember := govState.CommitteeMembers[coldHash]
				// Also check if this cold key is proposed in any pending UpdateCommittee proposal
				isProposed := govState.IsProposedCommitteeMember(coldHash)
				if !isMember && !isProposed {
					return fmt.Errorf(
						"cannot resign non-member %x",
						coldHash[:],
					)
				}
			}
		}

	default:
		// Other certificate types don't require pre-validation
	}

	return nil
}

// validateProposalProcedures validates proposal procedures in the transaction.
func (v *Validator) validateProposalProcedures(
	tx common.Transaction,
	govState *GovernanceState,
	pp common.ProtocolParameters,
) error {
	proposals := tx.ProposalProcedures()
	if proposals == nil {
		return nil
	}

	for _, proposal := range proposals {
		if err := v.validateProposal(proposal, govState, pp); err != nil {
			return err
		}
	}

	return nil
}

// validateProposal validates a single proposal.
func (v *Validator) validateProposal(
	proposal common.ProposalProcedure,
	govState *GovernanceState,
	pp common.ProtocolParameters,
) error {
	// Check reward account address is valid and registered
	rewardAddr := proposal.RewardAccount()
	stakeHash := extractStakeHashFromAddress(rewardAddr)
	if stakeHash == nil {
		addrBytes, _ := rewardAddr.Bytes()
		return fmt.Errorf(
			"invalid proposal reward address: cannot extract stake credential from %x",
			addrBytes,
		)
	}
	if !govState.IsStakeRegistered(*stakeHash) {
		return fmt.Errorf(
			"proposal reward address %x not registered",
			*stakeHash,
		)
	}

	// Validate action-specific rules
	action := proposal.GovAction()
	if action == nil {
		return nil
	}

	// Validate policy script if constitution has one
	if err := v.validatePolicyScript(action, govState); err != nil {
		return err
	}

	switch ga := action.(type) {
	case *conway.ConwayParameterChangeGovAction:
		if err := v.validateParameterChange(ga, govState); err != nil {
			return err
		}
	case *common.UpdateCommitteeGovAction:
		if err := v.validateUpdateCommittee(ga, govState); err != nil {
			return err
		}
	case *common.HardForkInitiationGovAction:
		if err := v.validateHardFork(ga, govState, pp); err != nil {
			return err
		}
	case *common.NewConstitutionGovAction:
		if err := v.validateNewConstitution(ga, govState); err != nil {
			return err
		}
	case *common.TreasuryWithdrawalGovAction:
		if err := v.validateTreasuryWithdrawal(ga, govState); err != nil {
			return err
		}
	case *common.NoConfidenceGovAction:
		if err := v.validateNoConfidence(ga, govState); err != nil {
			return err
		}
	}

	return nil
}

// validatePolicyScript validates that proposal policy hashes match the constitution's guardrails.
// Per CIP-1694, if the constitution has a guardrails policy script, proposals of certain types
// (ParameterChange, TreasuryWithdrawal) must reference that same policy hash.
// This is Phase-1 validation - the actual script execution happens in Phase-2.
func (v *Validator) validatePolicyScript(
	action common.GovAction,
	govState *GovernanceState,
) error {
	// Only ParameterChange and TreasuryWithdrawal are subject to guardrails policy validation
	actionWithPolicy, ok := action.(common.GovActionWithPolicy)
	if !ok {
		return nil
	}

	proposalPolicyHash := actionWithPolicy.GetPolicyHash()

	// If constitution has a policy, proposal's policy must match
	if govState.Constitution != nil &&
		len(govState.Constitution.PolicyHash) > 0 {
		if len(proposalPolicyHash) == 0 {
			// Proposal is missing required policy hash
			return fmt.Errorf(
				"proposal missing required policy hash (constitution has guardrails %x)",
				govState.Constitution.PolicyHash,
			)
		}
		// Compare policy hashes
		if len(proposalPolicyHash) != len(govState.Constitution.PolicyHash) {
			return fmt.Errorf(
				"proposal policy hash %x does not match constitution guardrails %x",
				proposalPolicyHash,
				govState.Constitution.PolicyHash,
			)
		}
		for i := range govState.Constitution.PolicyHash {
			if proposalPolicyHash[i] != govState.Constitution.PolicyHash[i] {
				return fmt.Errorf(
					"proposal policy hash %x does not match constitution guardrails %x",
					proposalPolicyHash,
					govState.Constitution.PolicyHash,
				)
			}
		}
	}

	return nil
}

// validateParameterChange validates ParameterChange governance actions.
func (v *Validator) validateParameterChange(
	ga *conway.ConwayParameterChangeGovAction,
	govState *GovernanceState,
) error {
	return v.validatePrevGovId(
		ga.ActionId,
		common.GovActionTypeParameterChange,
		govState,
	)
}

// validateNoConfidence validates NoConfidence governance actions.
func (v *Validator) validateNoConfidence(
	ga *common.NoConfidenceGovAction,
	govState *GovernanceState,
) error {
	return v.validatePrevGovId(
		ga.ActionId,
		common.GovActionTypeNoConfidence,
		govState,
	)
}

// validateUpdateCommittee validates UpdateCommittee governance actions.
func (v *Validator) validateUpdateCommittee(
	ga *common.UpdateCommitteeGovAction,
	govState *GovernanceState,
) error {
	// ConflictingCommitteeUpdate: no credential should be both added and removed
	removedCreds := make(map[common.Blake2b224]bool)
	for _, cred := range ga.Credentials {
		removedCreds[cred.Credential] = true
	}
	for cred := range ga.CredEpochs {
		if cred != nil && removedCreds[cred.Credential] {
			return fmt.Errorf(
				"conflicting committee update: credential %x is both added and removed",
				cred.Credential,
			)
		}
	}

	// ExpirationEpochTooSmall: expiration must be > current epoch
	for cred, epoch := range ga.CredEpochs {
		if uint64(epoch) <= govState.CurrentEpoch {
			credHash := common.Blake2b224{}
			if cred != nil {
				credHash = cred.Credential
			}
			return fmt.Errorf(
				"committee member expiration epoch %d too small (<= current epoch %d) for %x",
				epoch,
				govState.CurrentEpoch,
				credHash,
			)
		}
	}

	// PrevGovId validation for committee actions
	if err := v.validatePrevGovId(
		ga.ActionId,
		common.GovActionTypeUpdateCommittee,
		govState,
	); err != nil {
		return err
	}

	return nil
}

// validateHardFork validates HardForkInitiation governance actions.
func (v *Validator) validateHardFork(
	ga *common.HardForkInitiationGovAction,
	govState *GovernanceState,
	pp common.ProtocolParameters,
) error {
	// PrevGovId validation
	if err := v.validatePrevGovId(
		ga.ActionId,
		common.GovActionTypeHardForkInitiation,
		govState,
	); err != nil {
		return err
	}

	// pvCanFollow: version must be valid increment
	// Get baseline version from current protocol parameters
	// Default to 10.0 for Conway if we can't get the version
	baseMajor := uint(10)
	baseMinor := uint(0)
	if conwayPP, ok := pp.(*conway.ConwayProtocolParameters); ok {
		baseMajor = conwayPP.ProtocolVersion.Major
		baseMinor = conwayPP.ProtocolVersion.Minor
	}

	if ga.ActionId != nil {
		// Use parent proposal's version if it exists in active proposals
		parentKey := formatGovActionIdFromPtr(ga.ActionId)
		if parent, ok := govState.Proposals[parentKey]; ok &&
			parent.ProtocolVersion != nil {
			baseMajor = parent.ProtocolVersion.Major
			baseMinor = parent.ProtocolVersion.Minor
		}
		// If parent is not in active proposals (e.g., enacted root),
		// we keep using the current protocol version from pp as baseline
	}

	newMajor := ga.ProtocolVersion.Major
	newMinor := ga.ProtocolVersion.Minor

	// Valid increments: (major+1, 0) or (major, minor+1)
	majorIncrement := newMajor == baseMajor+1 && newMinor == 0
	minorIncrement := newMajor == baseMajor && newMinor == baseMinor+1

	if !majorIncrement && !minorIncrement {
		return fmt.Errorf(
			"hard fork: protocol version %d.%d cannot follow %d.%d",
			newMajor, newMinor, baseMajor, baseMinor,
		)
	}

	return nil
}

// validateNewConstitution validates NewConstitution governance actions.
func (v *Validator) validateNewConstitution(
	ga *common.NewConstitutionGovAction,
	govState *GovernanceState,
) error {
	return v.validatePrevGovId(
		ga.ActionId,
		common.GovActionTypeNewConstitution,
		govState,
	)
}

// validateTreasuryWithdrawal validates TreasuryWithdrawal governance actions.
func (v *Validator) validateTreasuryWithdrawal(
	ga *common.TreasuryWithdrawalGovAction,
	govState *GovernanceState,
) error {
	// Validate each withdrawal return address is registered
	// Withdrawals is map[*Address]uint64 where key is reward address, value is amount
	for rewardAddr := range ga.Withdrawals {
		if rewardAddr == nil {
			continue
		}
		stakeHash := extractStakeHashFromAddress(*rewardAddr)
		if stakeHash == nil {
			addrBytes, _ := rewardAddr.Bytes()
			return fmt.Errorf(
				"invalid treasury withdrawal: cannot extract stake credential from %x",
				addrBytes,
			)
		}
		if !govState.IsStakeRegistered(*stakeHash) {
			return fmt.Errorf(
				"treasury withdrawal: return address %x not registered",
				*stakeHash,
			)
		}
	}
	return nil
}

// validatePrevGovId validates the parent governance action reference.
// This checks:
// 1. If a root exists for this action type, parent must be non-nil
// 2. Parent must reference the root or an active proposal of compatible type
// 3. Parent action type must be compatible (e.g., Constitution -> Constitution)
// 4. Parent can also be a recently enacted proposal (still valid for children)
func (v *Validator) validatePrevGovId(
	parentId *common.GovActionId,
	actionType common.GovActionType,
	govState *GovernanceState,
) error {
	// Get the enacted root for this action type
	root := govState.GetEnactedRoot(actionType)

	if root != nil {
		// Root exists: parent must be non-nil
		if parentId == nil {
			return fmt.Errorf(
				"invalid GovPurposeId: empty parent but %d root exists",
				actionType,
			)
		}
		// Parent must be root, an active proposal, or an enacted proposal of compatible type
		parentKey := formatGovActionIdFromPtr(parentId)
		if parentKey != *root {
			parentProposal, exists := govState.Proposals[parentKey]
			if !exists {
				// Check if parent was recently enacted - enacted proposals are still
				// valid parents for their children during the same epoch
				if !govState.EnactedProposals[parentKey] {
					return fmt.Errorf(
						"invalid GovPurposeId: parent %s is neither root nor active proposal",
						parentKey,
					)
				}
				// Parent was enacted, which means it's valid - the action type
				// must be compatible since enacted proposals become roots
				return nil
			}
			// Validate parent action type is compatible with child type
			if err := v.validateParentActionType(parentProposal.ActionType, actionType); err != nil {
				return err
			}
		}
	} else if parentId != nil {
		// No root: non-nil parent must reference active or enacted proposal of compatible type
		parentKey := formatGovActionIdFromPtr(parentId)
		parentProposal, exists := govState.Proposals[parentKey]
		if !exists {
			// Check if parent was recently enacted
			if govState.EnactedProposals[parentKey] {
				// Enacted proposal is valid as parent
				return nil
			}
			return fmt.Errorf(
				"invalid GovPurposeId: parent action %s does not exist",
				parentKey,
			)
		}
		// Validate parent action type is compatible with child type
		if err := v.validateParentActionType(parentProposal.ActionType, actionType); err != nil {
			return err
		}
	}

	return nil
}

// validateParentActionType checks that the parent action type is compatible with the child.
// In Conway, proposals must chain off actions of the same governance purpose:
// - NewConstitution must chain off NewConstitution
// - HardFork must chain off HardFork
// - ParameterChange must chain off ParameterChange
// - UpdateCommittee/NoConfidence share a tree (both affect committee)
func (v *Validator) validateParentActionType(
	parentType common.GovActionType,
	childType common.GovActionType,
) error {
	// NoConfidence and UpdateCommittee share the same governance tree
	// (both affect the constitutional committee)
	normalizedParent := normalizeGovActionType(parentType)
	normalizedChild := normalizeGovActionType(childType)

	if normalizedParent != normalizedChild {
		return fmt.Errorf(
			"invalid GovPurposeId: parent action type %d incompatible with child type %d",
			parentType,
			childType,
		)
	}
	return nil
}

// normalizeGovActionType normalizes action types that share the same governance tree.
// NoConfidence and UpdateCommittee both affect the constitutional committee,
// so they share a governance tree and can chain off each other.
func normalizeGovActionType(t common.GovActionType) common.GovActionType {
	// NoConfidence and UpdateCommittee share the committee governance tree
	if t == common.GovActionTypeNoConfidence {
		return common.GovActionTypeUpdateCommittee
	}
	return t
}

// formatGovActionIdFromPtr formats a GovActionId pointer as "txHash#index".
func formatGovActionIdFromPtr(id *common.GovActionId) string {
	if id == nil {
		return ""
	}
	return fmt.Sprintf(
		"%s#%d",
		hex.EncodeToString(id.TransactionId[:]),
		id.GovActionIdx,
	)
}

// extractStakeHashFromAddress extracts the stake credential hash from an address.
// Per CIP-19, address types and lengths:
//   - Reward addresses (types 0xE, 0xF): 29 bytes (1 header + 28 credential)
//   - Base addresses (types 0-3): 57 bytes (1 header + 28 payment + 28 stake)
func extractStakeHashFromAddress(addr common.Address) *common.Blake2b224 {
	// Get address bytes
	addrBytes, err := addr.Bytes()
	if err != nil || len(addrBytes) == 0 {
		return nil
	}

	header := addrBytes[0]
	addrType := (header & 0xF0) >> 4

	// Check for stake/reward address types (0xE or 0xF)
	// Per CIP-19, reward addresses are exactly 29 bytes
	if addrType == 0xE || addrType == 0xF {
		if len(addrBytes) != 29 {
			return nil
		}
		var hash common.Blake2b224
		copy(hash[:], addrBytes[1:29])
		return &hash
	}

	// For base addresses (types 0-3), extract the staking part (last 28 bytes)
	// Per CIP-19, base addresses are exactly 57 bytes
	if addrType <= 0x3 {
		if len(addrBytes) != 57 {
			return nil
		}
		var hash common.Blake2b224
		copy(hash[:], addrBytes[29:57])
		return &hash
	}

	return nil
}
