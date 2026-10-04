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
	"cmp"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
)

// Ratification follows the Conway ratification rules. An action ratifies when
// its parent is the enacted root of its purpose, an UpdateCommittee action's
// new terms fit the committee term limit, the committee, DRep and SPO
// stake-weighted ratios meet the action's thresholds, a treasury withdrawal
// fits in the treasury, and no earlier delaying action ratified in the same
// epoch. Info actions never ratify.
//
// Unsupported edge case: actions of one priority tier are ordered by
// submission epoch and then by identifier, because the state does not record
// the order of proposals within an epoch.

// ratifyPriority orders actions as the ledger's ratification does.
func ratifyPriority(actionType common.GovActionType) int {
	//exhaustive:ignore
	switch actionType {
	case common.GovActionTypeNoConfidence:
		return 0
	case common.GovActionTypeUpdateCommittee:
		return 1
	case common.GovActionTypeNewConstitution:
		return 2
	case common.GovActionTypeHardForkInitiation:
		return 3
	case common.GovActionTypeParameterChange:
		return 4
	case common.GovActionTypeTreasuryWithdrawal:
		return 5
	default:
		return 6
	}
}

// delaysRatification reports whether ratifying the action blocks every later
// action in the same epoch.
func delaysRatification(actionType common.GovActionType) bool {
	//exhaustive:ignore
	switch actionType {
	case common.GovActionTypeNoConfidence,
		common.GovActionTypeUpdateCommittee,
		common.GovActionTypeNewConstitution,
		common.GovActionTypeHardForkInitiation:
		return true
	default:
		return false
	}
}

// compareProposals orders proposals for ratification and enactment.
func compareProposals(
	aID, bID string,
	a, b *ProposalState,
) int {
	return cmp.Or(
		cmp.Compare(ratifyPriority(a.ActionType), ratifyPriority(b.ActionType)),
		cmp.Compare(a.SubmittedEpoch, b.SubmittedEpoch),
		cmp.Compare(aID, bID),
	)
}

// proposalIDs returns the IDs of the proposals that pass keep, in no
// particular order.
func (g *GovernanceState) proposalIDs(keep func(*ProposalState) bool) []string {
	ids := make([]string, 0, len(g.Proposals))
	for id, proposal := range g.Proposals {
		if proposal != nil && keep(proposal) {
			ids = append(ids, id)
		}
	}
	return ids
}

// parentMatchesRoot reports whether the proposal chains off the given root.
// Actions without a purpose chain off nothing and always match.
func parentMatchesRoot(proposal *ProposalState, root *string) bool {
	if proposal.ParentActionId == nil || root == nil {
		return proposal.ParentActionId == nil && root == nil
	}
	return *proposal.ParentActionId == *root
}

func (m *MockStateManager) conwayParams() (*conway.ConwayProtocolParameters, error) {
	pp, ok := m.protocolParams.(*conway.ConwayProtocolParameters)
	if !ok || pp == nil {
		return nil, errors.New("conway protocol parameters unavailable")
	}
	return pp, nil
}

func (m *MockStateManager) inBootstrapPhase() bool {
	pp, ok := m.protocolParams.(*conway.ConwayProtocolParameters)
	return ok && pp != nil &&
		pp.ProtocolVersion.Major == common.ProtocolVersionConway
}

// ratifyProposals marks every proposal that meets its requirements as ratified.
func (m *MockStateManager) ratifyProposals(currentEpoch uint64) error {
	var stake map[ledger.RewardAccountKey]*big.Int
	roots := m.govState.Roots
	treasury := m.govState.Treasury
	toRatify := make([]string, 0)
	candidates := m.govState.proposalIDs(func(p *ProposalState) bool {
		// Ratification needs at least one epoch between submission and the
		// boundary that ratifies.
		return p.ActionType != common.GovActionTypeInfo &&
			currentEpoch <= p.ExpiresAfter &&
			p.RatifiedEpoch == nil &&
			currentEpoch > p.SubmittedEpoch
	})
	slices.SortFunc(candidates, func(a, b string) int {
		return compareProposals(
			a, b, m.govState.Proposals[a], m.govState.Proposals[b],
		)
	})
	for _, id := range candidates {
		proposal := m.govState.Proposals[id]
		if stake == nil {
			stake = m.credentialVotingStake(currentEpoch)
		}
		accepted, err := m.proposalAccepted(proposal, stake)
		if err != nil {
			return fmt.Errorf("ratify proposal %s: %w", id, err)
		}
		rootSlot := roots.forAction(proposal.ActionType)
		if !accepted ||
			rootSlot != nil && !parentMatchesRoot(proposal, *rootSlot) ||
			!m.withinCommitteeTermLimit(proposal, currentEpoch) ||
			proposal.withdrawalTotal() > treasury {
			continue
		}
		if rootSlot != nil {
			*rootSlot = &id
		}
		treasury -= proposal.withdrawalTotal()
		toRatify = append(toRatify, id)
		if delaysRatification(proposal.ActionType) {
			break
		}
	}

	// Commit only after every proposal has been evaluated successfully. This
	// keeps an action-specific parameter error from leaving partial ratification
	// state behind.
	for _, id := range toRatify {
		epoch := currentEpoch
		m.govState.Proposals[id].RatifiedEpoch = &epoch
	}
	return nil
}

// withinCommitteeTermLimit reports whether every member an UpdateCommittee
// action elects expires no later than the term limit allows from currentEpoch.
func (m *MockStateManager) withinCommitteeTermLimit(
	proposal *ProposalState,
	currentEpoch uint64,
) bool {
	pp, ok := m.protocolParams.(*conway.ConwayProtocolParameters)
	if !ok || pp == nil ||
		proposal.ActionType != common.GovActionTypeUpdateCommittee {
		return true
	}
	for _, expiry := range proposal.ProposedMembersByCredential {
		if expiry > currentEpoch+pp.CommitteeTermLimit {
			return false
		}
	}
	for _, expiry := range proposal.ProposedMembers {
		if expiry > currentEpoch+pp.CommitteeTermLimit {
			return false
		}
	}
	return true
}

func (p *ProposalState) withdrawalTotal() uint64 {
	var total uint64
	for _, amount := range p.Withdrawals {
		total += amount
	}
	return total
}

// proposalAccepted applies the committee, DRep and SPO acceptance rules.
func (m *MockStateManager) proposalAccepted(
	proposal *ProposalState,
	stake map[ledger.RewardAccountKey]*big.Int,
) (bool, error) {
	pp, err := m.conwayParams()
	if err != nil {
		return false, err
	}
	drepThreshold, poolThreshold, err := m.votingThresholds(proposal, pp)
	if err != nil {
		return false, err
	}
	if !m.committeeAccepted(proposal) {
		return false, nil
	}
	if !m.drepAccepted(proposal, stake, drepThreshold) {
		return false, nil
	}
	return poolThreshold == nil ||
		m.spoAccepted(proposal, stake, poolThreshold), nil
}

// votingThresholds returns the DRep and SPO thresholds for an action. A nil
// SPO threshold means stake pools do not vote on the action. Every DRep
// threshold is zero during the bootstrap phase.
func (m *MockStateManager) votingThresholds(
	proposal *ProposalState,
	pp *conway.ConwayProtocolParameters,
) (*big.Rat, *big.Rat, error) {
	drepRats, poolRat := m.thresholdParameters(proposal, pp)
	drep := new(big.Rat)
	if !m.inBootstrapPhase() {
		for _, threshold := range drepRats {
			if threshold.Rat == nil {
				return nil, nil, errors.New("DRep voting threshold unavailable")
			}
			if threshold.Cmp(drep) > 0 {
				drep = threshold.Rat
			}
		}
	}
	if poolRat == nil {
		return drep, nil, nil
	}
	if poolRat.Rat == nil {
		return nil, nil, errors.New("SPO voting threshold unavailable")
	}
	return drep, poolRat.Rat, nil
}

// thresholdParameters selects the protocol-parameter thresholds that govern an
// action. The DRep result holds every threshold the action must clear; the
// SPO result is nil when stake pools do not vote on the action.
func (m *MockStateManager) thresholdParameters(
	proposal *ProposalState,
	pp *conway.ConwayProtocolParameters,
) ([]cbor.Rat, *cbor.Rat) {
	drep, pool := pp.DRepVotingThresholds, pp.PoolVotingThresholds
	//exhaustive:ignore
	switch proposal.ActionType {
	case common.GovActionTypeNoConfidence:
		return []cbor.Rat{drep.MotionNoConfidence}, &pool.MotionNoConfidence
	case common.GovActionTypeUpdateCommittee:
		if m.govState.hasCommittee() {
			return []cbor.Rat{drep.CommitteeNormal}, &pool.CommitteeNormal
		}
		return []cbor.Rat{drep.CommitteeNoConfidence},
			&pool.CommitteeNoConfidence
	case common.GovActionTypeNewConstitution:
		return []cbor.Rat{drep.UpdateToConstitution}, nil
	case common.GovActionTypeHardForkInitiation:
		return []cbor.Rat{drep.HardForkInitiation}, &pool.HardForkInitiation
	case common.GovActionTypeParameterChange:
		update := proposal.ParameterUpdate
		if update == nil {
			return nil, nil
		}
		var poolRat *cbor.Rat
		if len(update.SecurityGroupFields()) > 0 {
			poolRat = &pool.PpSecurityGroup
		}
		return parameterGroupThresholds(update, drep), poolRat
	case common.GovActionTypeTreasuryWithdrawal:
		return []cbor.Rat{drep.TreasuryWithdrawal}, nil
	}
	return nil, nil
}

// parameterGroupThresholds returns the DRep threshold of every parameter group
// the update changes.
func parameterGroupThresholds(
	u *conway.ConwayProtocolParameterUpdate,
	t conway.DRepVotingThresholds,
) []cbor.Rat {
	var rats []cbor.Rat
	if u.MaxBlockBodySize != nil || u.MaxTxSize != nil ||
		u.MaxBlockHeaderSize != nil || u.MaxValueSize != nil ||
		u.MaxTxExUnits != nil || u.MaxBlockExUnits != nil ||
		u.MaxCollateralInputs != nil {
		rats = append(rats, t.PpNetworkGroup)
	}
	if u.MinFeeA != nil || u.MinFeeB != nil || u.KeyDeposit != nil ||
		u.PoolDeposit != nil || u.Rho != nil || u.Tau != nil ||
		u.MinPoolCost != nil || u.AdaPerUtxoByte != nil ||
		u.ExecutionCosts != nil || u.MinFeeRefScriptCostPerByte != nil {
		rats = append(rats, t.PpEconomicGroup)
	}
	if u.MaxEpoch != nil || u.NOpt != nil || u.A0 != nil ||
		u.CostModels != nil || u.CollateralPercentage != nil {
		rats = append(rats, t.PpTechnicalGroup)
	}
	if u.PoolVotingThresholds != nil || u.DRepVotingThresholds != nil ||
		u.MinCommitteeSize != nil || u.CommitteeTermLimit != nil ||
		u.GovActionValidityPeriod != nil || u.GovActionDeposit != nil ||
		u.DRepDeposit != nil || u.DRepInactivityPeriod != nil {
		rats = append(rats, t.PpGovGroup)
	}
	return rats
}

// committeeAccepted applies the committee vote, which NoConfidence and
// UpdateCommittee do not need. Members that are expired or without an
// authorized hot credential (a resignation removes it) abstain; members that
// did not vote count as No.
func (m *MockStateManager) committeeAccepted(proposal *ProposalState) bool {
	//exhaustive:ignore
	switch proposal.ActionType {
	case common.GovActionTypeNoConfidence,
		common.GovActionTypeUpdateCommittee:
		return true
	}
	g := m.govState
	threshold := g.CommitteeThreshold
	if threshold == nil {
		return false
	}
	yesVotes, total := new(big.Int), new(big.Int)
	var active uint64
	for cold, member := range g.CommitteeMembersByCredential {
		hot, authorized := g.HotKeyAuthorizationsByCredential[cold]
		if member == nil || m.currentEpoch > member.ExpiryEpoch || !authorized {
			continue
		}
		active++
		voterType := common.VoterTypeConstitutionalCommitteeHotKeyHash
		if hot.CredType == common.CredentialTypeScriptHash {
			voterType = common.VoterTypeConstitutionalCommitteeHotScriptHash
		}
		switch proposal.Votes[voteKey(voterType, hot.Credential)] {
		case 2:
		case 1:
			yesVotes.Add(yesVotes, big.NewInt(1))
			total.Add(total, big.NewInt(1))
		default:
			total.Add(total, big.NewInt(1))
		}
	}
	if pp, ok := m.protocolParams.(*conway.ConwayProtocolParameters); ok &&
		pp != nil && !m.inBootstrapPhase() &&
		active < uint64(pp.MinCommitteeSize) {
		return false
	}
	return votingStakeAccepted(yesVotes, total, threshold)
}

// voteKey formats the key under which a voter's vote is stored.
func voteKey(voterType uint8, hash common.Blake2b224) string {
	return fmt.Sprintf("%d:%s", voterType, hex.EncodeToString(hash[:]))
}

func (m *MockStateManager) credentialVotingStake(
	currentEpoch uint64,
) map[ledger.RewardAccountKey]*big.Int {
	credentialStake := make(map[ledger.RewardAccountKey]*big.Int)
	addStake := func(credential ledger.RewardAccountKey, amount *big.Int) {
		if amount == nil || amount.Sign() <= 0 {
			return
		}
		if current := credentialStake[credential]; current != nil {
			current.Add(current, amount)
		} else {
			credentialStake[credential] = new(big.Int).Set(amount)
		}
	}
	for _, utxo := range m.utxos {
		if utxo.Output == nil {
			continue
		}
		address := utxo.Output.Address()
		credential, ok := address.StakeCredential()
		if !ok {
			continue
		}
		addStake(ledger.NewRewardAccountKey(credential), utxo.Output.Amount())
	}
	for credential, balance := range m.rewardAccounts {
		addStake(credential, new(big.Int).SetUint64(balance))
	}
	for _, activeProposal := range m.govState.Proposals {
		if activeProposal == nil || activeProposal.ReturnAccount == nil ||
			activeProposal.Deposit == 0 ||
			currentEpoch > activeProposal.ExpiresAfter {
			continue
		}
		addStake(
			*activeProposal.ReturnAccount,
			new(big.Int).SetUint64(activeProposal.Deposit),
		)
	}
	return credentialStake
}

func (m *MockStateManager) drepAccepted(
	proposal *ProposalState,
	credentialStake map[ledger.RewardAccountKey]*big.Int,
	threshold *big.Rat,
) bool {
	if threshold.Sign() == 0 {
		return true
	}
	yesStake := new(big.Int)
	totalStake := new(big.Int)
	for stakeCredential, stake := range credentialStake {
		delegation, ok := m.govState.DRepDelegationsByCredential[stakeCredential]
		if !ok {
			continue
		}
		switch delegation.Type {
		case common.DrepTypeAbstain:
			continue
		case common.DrepTypeNoConfidence:
			totalStake.Add(totalStake, stake)
			if proposal.ActionType == common.GovActionTypeNoConfidence {
				yesStake.Add(yesStake, stake)
			}
		case common.DrepTypeAddrKeyHash, common.DrepTypeScriptHash:
			if len(delegation.Credential) != common.Blake2b224Size {
				continue
			}
			drepCredential := common.Credential{
				CredType:   common.CredentialTypeAddrKeyHash,
				Credential: common.NewBlake2b224(delegation.Credential),
			}
			voterType := common.VoterTypeDRepKeyHash
			if delegation.Type == common.DrepTypeScriptHash {
				drepCredential.CredType = common.CredentialTypeScriptHash
				voterType = common.VoterTypeDRepScriptHash
			}
			if !m.govState.IsDRepCredentialActive(
				drepCredential,
				m.currentEpoch,
			) {
				continue
			}
			vote, voted := proposal.Votes[voteKey(
				voterType,
				drepCredential.Credential,
			)]
			if voted && vote == 2 {
				continue
			}
			totalStake.Add(totalStake, stake)
			if voted && vote == 1 {
				yesStake.Add(yesStake, stake)
			}
		}
	}
	return votingStakeAccepted(yesStake, totalStake, threshold)
}

func (m *MockStateManager) spoAccepted(
	proposal *ProposalState,
	credentialStake map[ledger.RewardAccountKey]*big.Int,
	threshold *big.Rat,
) bool {
	if threshold.Sign() == 0 {
		return true
	}
	poolStake := make(map[common.PoolKeyHash]*big.Int)
	for stakeCredential, stake := range credentialStake {
		pool, ok := m.govState.PoolDelegationsByCredential[stakeCredential]
		if !ok || !m.govState.IsPoolRegistered(pool) {
			continue
		}
		if current := poolStake[pool]; current != nil {
			current.Add(current, stake)
		} else {
			poolStake[pool] = new(big.Int).Set(stake)
		}
	}
	yesStake := new(big.Int)
	totalStake := new(big.Int)
	for pool, stake := range poolStake {
		vote, voted := proposal.Votes[voteKey(
			common.VoterTypeStakingPoolKeyHash,
			pool,
		)]
		if voted {
			switch vote {
			case 1:
				yesStake.Add(yesStake, stake)
				totalStake.Add(totalStake, stake)
			case 0:
				totalStake.Add(totalStake, stake)
			case 2:
			}
			continue
		}
		// A pool that did not vote counts as No on a hard fork. On other
		// actions it abstains during bootstrap and otherwise follows its
		// reward account's DRep delegation.
		if proposal.ActionType != common.GovActionTypeHardForkInitiation {
			if m.inBootstrapPhase() {
				continue
			}
			if rewardAccount, ok := m.govState.PoolRewardAccounts[pool]; ok {
				delegation, ok := m.govState.DRepDelegationsByCredential[rewardAccount]
				if ok && delegation.Type == common.DrepTypeAbstain {
					continue
				}
				if ok && delegation.Type == common.DrepTypeNoConfidence &&
					proposal.ActionType == common.GovActionTypeNoConfidence {
					yesStake.Add(yesStake, stake)
				}
			}
		}
		totalStake.Add(totalStake, stake)
	}
	return votingStakeAccepted(yesStake, totalStake, threshold)
}

func votingStakeAccepted(
	yesStake *big.Int,
	totalStake *big.Int,
	threshold *big.Rat,
) bool {
	if threshold.Sign() == 0 {
		return true
	}
	if totalStake.Sign() == 0 {
		return false
	}
	return new(big.Int).Mul(
		yesStake,
		threshold.Denom(),
	).Cmp(new(big.Int).Mul(totalStake, threshold.Num())) >= 0
}
