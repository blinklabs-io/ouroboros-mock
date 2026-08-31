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
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLedgerStateFindsProposedCommitteeMember(t *testing.T) {
	coldKey := common.Blake2b224{0x01}
	coldCredential := common.Credential{Credential: coldKey}
	coldCredentialKey := ledger.NewRewardAccountKey(coldCredential)
	hotKey := common.Blake2b224{0x02}
	const expiryEpoch = uint64(42)

	stateManager := NewMockStateManager()
	stateManager.govState.Proposals["proposal#0"] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType: common.GovActionTypeUpdateCommittee,
			ProposedMembersByCredential: map[ledger.RewardAccountKey]uint64{
				coldCredentialKey: expiryEpoch,
			},
		},
	}
	stateManager.processCertificate(&common.AuthCommitteeHotCertificate{
		CertType:       uint(common.CertificateTypeAuthCommitteeHot),
		ColdCredential: coldCredential,
		HotCredential: common.Credential{
			Credential: hotKey,
		},
	})

	member, err := stateManager.buildLedgerState().
		CommitteeCredentialMember(coldCredential)
	require.NoError(t, err)
	require.NotNil(t, member)
	assert.Equal(t, coldKey, member.ColdKey)
	assert.Equal(t, expiryEpoch, member.ExpiryEpoch)
	assert.Nil(t, member.HotKey)
	assert.False(t, member.Resigned)
	assert.Equal(
		t,
		hotKey,
		stateManager.govState.HotKeyAuthorizationsByCredential[coldCredentialKey].Credential,
	)
}

func TestGovernanceStateLegacyCommitteeMutationCompatibility(t *testing.T) {
	firstColdKey := common.Blake2b224{0x01}
	firstHotKey := common.Blake2b224{0x02}
	secondColdKey := common.Blake2b224{0x03}
	secondHotKey := common.Blake2b224{0x04}
	legacyOnlyColdKey := common.Blake2b224{0x05}
	legacyOnlyHotKey := common.Blake2b224{0x06}
	state := NewGovernanceState()
	firstMember := &CommitteeMemberInfo{
		ColdKey:     firstColdKey,
		ExpiryEpoch: 42,
		Resigned:    true,
	}
	secondMember := &CommitteeMemberInfo{
		ColdKey:     secondColdKey,
		ExpiryEpoch: 43,
	}
	state.CommitteeMembers[firstColdKey] = firstMember
	state.CommitteeMembers[secondColdKey] = secondMember
	state.HotKeyAuthorizations[legacyOnlyColdKey] = legacyOnlyHotKey

	state.AuthorizeHotKey(firstColdKey, firstHotKey)
	require.Equal(t, firstHotKey, state.HotKeyAuthorizations[firstColdKey])
	require.Equal(
		t,
		legacyOnlyHotKey,
		state.HotKeyAuthorizations[legacyOnlyColdKey],
	)
	require.NotNil(t, firstMember.HotKey)
	require.Equal(t, firstHotKey, *firstMember.HotKey)
	require.False(t, firstMember.Resigned)

	state.AuthorizeHotKey(secondColdKey, secondHotKey)
	require.Equal(t, firstHotKey, state.HotKeyAuthorizations[firstColdKey])
	require.Equal(t, secondHotKey, state.HotKeyAuthorizations[secondColdKey])
	require.NotNil(t, secondMember.HotKey)
	require.Equal(t, secondHotKey, *secondMember.HotKey)

	state.ResignCommitteeMember(firstColdKey)
	require.NotContains(t, state.HotKeyAuthorizations, firstColdKey)
	require.Equal(t, secondHotKey, state.HotKeyAuthorizations[secondColdKey])
	require.Equal(
		t,
		legacyOnlyHotKey,
		state.HotKeyAuthorizations[legacyOnlyColdKey],
	)
	require.Nil(t, firstMember.HotKey)
	require.True(t, firstMember.Resigned)

	state.AuthorizeHotKey(firstColdKey, firstHotKey)
	require.NotNil(t, firstMember.HotKey)
	require.Equal(t, firstHotKey, *firstMember.HotKey)
	require.False(t, firstMember.Resigned)
}

func TestLegacyCommitteeMutationDoesNotGuessScriptCredential(t *testing.T) {
	firstColdKey := common.Blake2b224{0x01}
	firstHotKey := common.Blake2b224{0x02}
	scriptState := NewGovernanceState()
	scriptMember := &CommitteeMemberInfo{
		ColdCredential: common.Credential{
			CredType:   common.CredentialTypeScriptHash,
			Credential: firstColdKey,
		},
		ColdKey:     firstColdKey,
		ExpiryEpoch: 42,
	}
	scriptState.CommitteeMembersByCredential[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: firstColdKey,
	}] = scriptMember
	scriptState.syncLegacyCommitteeMembers()
	scriptState.AuthorizeHotKey(firstColdKey, firstHotKey)
	require.Nil(t, scriptMember.HotKey)
	require.False(t, scriptMember.Resigned)
	scriptState.ResignCommitteeMember(firstColdKey)
	require.Nil(t, scriptMember.HotKey)
	require.False(t, scriptMember.Resigned)
}

func TestCurrentCommitteeResignationIsVisible(t *testing.T) {
	coldKey := common.Blake2b224{0x01}
	coldCredential := common.Credential{Credential: coldKey}
	coldCredentialKey := ledger.NewRewardAccountKey(coldCredential)
	hotKey := common.Blake2b224{0x02}
	stateManager := NewMockStateManager()
	stateManager.committeeMembers[coldCredentialKey] = 42
	stateManager.govState.CommitteeMembersByCredential[coldCredentialKey] = &CommitteeMemberInfo{
		ColdCredential: coldCredential,
		ColdKey:        coldKey,
		ExpiryEpoch:    42,
	}
	authorization := &common.AuthCommitteeHotCertificate{
		CertType:       uint(common.CertificateTypeAuthCommitteeHot),
		ColdCredential: coldCredential,
		HotCredential: common.Credential{
			Credential: hotKey,
		},
	}
	stateManager.processCertificate(authorization)
	stateManager.processCertificate(&common.ResignCommitteeColdCertificate{
		CertType:       uint(common.CertificateTypeResignCommitteeCold),
		ColdCredential: coldCredential,
	})

	member, err := stateManager.buildLedgerState().
		CommitteeCredentialMember(coldCredential)
	require.NoError(t, err)
	require.NotNil(t, member)
	assert.Nil(t, member.HotKey)
	assert.True(t, member.Resigned)
	require.ErrorContains(t, NewValidator().validateCertificate(
		authorization,
		stateManager.govState,
		nil,
	), "cannot authorize hot key for resigned CC member")
	require.ErrorContains(t, NewValidator().validateCertificate(
		&common.ResignCommitteeColdCertificate{
			CertType:       uint(common.CertificateTypeResignCommitteeCold),
			ColdCredential: coldCredential,
		},
		stateManager.govState,
		nil,
	), "cannot resign already resigned CC member")
}

func TestProposedCommitteeResignationRejectsReauthorization(t *testing.T) {
	coldKey := common.Blake2b224{0x01}
	coldCredential := common.Credential{Credential: coldKey}
	coldCredentialKey := ledger.NewRewardAccountKey(coldCredential)
	hotKey := common.Blake2b224{0x02}
	stateManager := NewMockStateManager()
	stateManager.govState.Proposals["proposal#0"] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType: common.GovActionTypeUpdateCommittee,
			ProposedMembersByCredential: map[ledger.RewardAccountKey]uint64{
				coldCredentialKey: 42,
			},
		},
	}
	stateManager.processCertificate(&common.ResignCommitteeColdCertificate{
		CertType:       uint(common.CertificateTypeResignCommitteeCold),
		ColdCredential: coldCredential,
	})
	authorization := &common.AuthCommitteeHotCertificate{
		CertType:       uint(common.CertificateTypeAuthCommitteeHot),
		ColdCredential: coldCredential,
		HotCredential: common.Credential{
			Credential: hotKey,
		},
	}

	member, err := stateManager.buildLedgerState().
		CommitteeCredentialMember(coldCredential)
	require.NoError(t, err)
	require.NotNil(t, member)
	assert.Nil(t, member.HotKey)
	assert.True(t, member.Resigned)
	require.ErrorContains(t, NewValidator().validateCertificate(
		authorization,
		stateManager.govState,
		nil,
	), "cannot authorize hot key for resigned CC member")
}

func TestCommitteeRemovalClearsStateBeforeReelection(t *testing.T) {
	coldKey := common.Blake2b224{0x01}
	coldCredential := common.Credential{Credential: coldKey}
	coldCredentialKey := ledger.NewRewardAccountKey(coldCredential)
	hotKey := common.Blake2b224{0x02}
	stateManager := NewMockStateManager()
	stateManager.protocolParams = &conway.ConwayProtocolParameters{
		DRepVotingThresholds: conway.DRepVotingThresholds{
			CommitteeNormal: cbor.Rat{Rat: new(big.Rat)},
		},
		PoolVotingThresholds: conway.PoolVotingThresholds{
			CommitteeNormal: cbor.Rat{Rat: new(big.Rat)},
		},
	}
	stateManager.committeeMembers[coldCredentialKey] = 10
	stateManager.govState.CommitteeMembersByCredential[coldCredentialKey] = &CommitteeMemberInfo{
		ColdCredential: coldCredential,
		ColdKey:        coldKey,
		ExpiryEpoch:    10,
	}
	stateManager.govState.syncLegacyCommitteeMembers()
	stateManager.processCertificate(&common.ResignCommitteeColdCertificate{
		CertType:       uint(common.CertificateTypeResignCommitteeCold),
		ColdCredential: coldCredential,
	})
	stateManager.govState.Proposals["remove#0"] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType:     common.GovActionTypeUpdateCommittee,
			SubmittedEpoch: 0,
			Votes: map[string]uint8{
				"2:drep": 1,
				"4:pool": 1,
			},
			RemovedMembers: map[ledger.RewardAccountKey]bool{
				coldCredentialKey: true,
			},
			ExpiresAfter: 10,
		},
	}

	require.NoError(t, stateManager.ProcessEpochBoundary(1))
	proposal := stateManager.govState.Proposals["remove#0"]
	require.NotNil(t, proposal)
	require.NotNil(t, proposal.RatifiedEpoch)
	assert.Equal(t, uint64(1), *proposal.RatifiedEpoch)
	assert.Contains(t, stateManager.committeeMembers, coldCredentialKey)

	require.NoError(t, stateManager.ProcessEpochBoundary(2))
	assert.NotContains(t, stateManager.committeeMembers, coldCredentialKey)
	assert.NotContains(
		t,
		stateManager.govState.CommitteeMembersByCredential,
		coldCredentialKey,
	)
	assert.NotContains(t, stateManager.govState.CommitteeMembers, coldKey)
	assert.NotContains(t, stateManager.govState.HotKeyAuthorizations, coldKey)
	assert.NotContains(t, stateManager.committeeResignations, coldCredentialKey)
	assert.NotContains(
		t,
		stateManager.govState.CommitteeResignations,
		coldCredentialKey,
	)

	reelectionEpoch := uint64(2)
	stateManager.govState.Proposals["reelect#0"] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType: common.GovActionTypeUpdateCommittee,
			ProposedMembersByCredential: map[ledger.RewardAccountKey]uint64{
				coldCredentialKey: 20,
			},
			ExpiresAfter: 10,
		},
		RatifiedEpoch: &reelectionEpoch,
	}
	authorization := &common.AuthCommitteeHotCertificate{
		CertType:       uint(common.CertificateTypeAuthCommitteeHot),
		ColdCredential: coldCredential,
		HotCredential: common.Credential{
			Credential: hotKey,
		},
	}
	require.NoError(t, NewValidator().validateCertificate(
		authorization,
		stateManager.govState,
		nil,
	))
	stateManager.processCertificate(authorization)
	proposedMember, err := stateManager.buildLedgerState().
		CommitteeCredentialMember(coldCredential)
	require.NoError(t, err)
	require.NotNil(t, proposedMember)
	assert.Nil(t, proposedMember.HotKey)
	assert.False(t, proposedMember.Resigned)

	require.NoError(t, stateManager.ProcessEpochBoundary(3))
	currentMember, err := stateManager.buildLedgerState().
		CommitteeCredentialMember(coldCredential)
	require.NoError(t, err)
	require.NotNil(t, currentMember)
	require.NotNil(t, currentMember.HotKey)
	assert.Equal(t, hotKey, *currentMember.HotKey)
	assert.False(t, currentMember.Resigned)
	legacyMember := stateManager.govState.CommitteeMembers[coldKey]
	require.NotNil(t, legacyMember)
	require.NotNil(t, legacyMember.HotKey)
	assert.Equal(t, hotKey, *legacyMember.HotKey)
	assert.False(t, legacyMember.Resigned)
}

func TestNoConfidenceEnactmentClearsCommitteeState(t *testing.T) {
	coldKey := common.Blake2b224{0x01}
	coldCredential := common.Credential{Credential: coldKey}
	coldCredentialKey := ledger.NewRewardAccountKey(coldCredential)
	stateManager := NewMockStateManager()
	stateManager.committeeMembers[coldCredentialKey] = 10
	stateManager.govState.CommitteeMembersByCredential[coldCredentialKey] = &CommitteeMemberInfo{
		ColdCredential: coldCredential,
		ColdKey:        coldKey,
		ExpiryEpoch:    10,
	}
	stateManager.processCertificate(&common.ResignCommitteeColdCertificate{
		CertType:       uint(common.CertificateTypeResignCommitteeCold),
		ColdCredential: coldCredential,
	})
	ratifiedEpoch := uint64(1)
	stateManager.govState.Proposals["no-confidence#0"] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType:   common.GovActionTypeNoConfidence,
			ExpiresAfter: 10,
		},
		RatifiedEpoch: &ratifiedEpoch,
	}

	require.NoError(t, stateManager.ProcessEpochBoundary(2))
	assert.Empty(t, stateManager.committeeMembers)
	assert.Empty(t, stateManager.govState.CommitteeMembersByCredential)
	assert.Empty(t, stateManager.committeeResignations)
	assert.Empty(t, stateManager.govState.CommitteeResignations)
}

func TestUpdateCommitteeCountsProposalDepositInDRepStake(t *testing.T) {
	stateManager := NewMockStateManager()
	stateManager.protocolParams = &conway.ConwayProtocolParameters{
		ProtocolVersion: common.ProtocolParametersProtocolVersion{Major: 10},
		DRepVotingThresholds: conway.DRepVotingThresholds{
			CommitteeNormal: cbor.Rat{Rat: big.NewRat(1, 2)},
		},
		PoolVotingThresholds: conway.PoolVotingThresholds{
			CommitteeNormal: cbor.Rat{Rat: new(big.Rat)},
		},
	}
	stateManager.govState.CommitteeMembersByCredential[ledger.RewardAccountKey{
		Credential: common.Blake2b224{0x01},
	}] = &CommitteeMemberInfo{}
	yesStake := ledger.RewardAccountKey{Credential: common.Blake2b224{0x11}}
	noStake := ledger.RewardAccountKey{Credential: common.Blake2b224{0x12}}
	yesDRep := common.Blake2b224{0x21}
	noDRep := common.Blake2b224{0x22}
	stateManager.rewardAccounts[yesStake] = 40
	stateManager.rewardAccounts[noStake] = 60
	stateManager.govState.DRepDelegationsByCredential[yesStake] = common.Drep{
		Type:       common.DrepTypeAddrKeyHash,
		Credential: yesDRep[:],
	}
	stateManager.govState.DRepDelegationsByCredential[noStake] = common.Drep{
		Type:       common.DrepTypeAddrKeyHash,
		Credential: noDRep[:],
	}
	stateManager.govState.DRepRegistrationsByCredential[ledger.RewardAccountKey{
		Credential: yesDRep,
	}] = true
	stateManager.govState.DRepRegistrationsByCredential[ledger.RewardAccountKey{
		Credential: noDRep,
	}] = true
	proposal := &ProposalState{GovActionInfo: GovActionInfo{
		ActionType:    common.GovActionTypeUpdateCommittee,
		Deposit:       20,
		ReturnAccount: &yesStake,
		Votes: map[string]uint8{
			fmt.Sprintf(
				"%d:%s",
				common.VoterTypeDRepKeyHash,
				hex.EncodeToString(yesDRep[:]),
			): 1,
		},
	}}
	stateManager.govState.Proposals["update#0"] = proposal

	stake := stateManager.credentialVotingStake()
	assert.True(t, stateManager.drepAcceptedForUpdateCommittee(
		proposal,
		stake,
		big.NewRat(1, 2),
	))
	assert.True(t, stateManager.spoAcceptedForUpdateCommittee(
		proposal,
		stake,
		new(big.Rat),
	))
	assert.True(t, stateManager.updateCommitteeAccepted(proposal))
	proposal.Deposit = 0
	assert.False(t, stateManager.updateCommitteeAccepted(proposal))
}

func TestUpdateCommitteeUsesStakeWeightedSPOApproval(t *testing.T) {
	stateManager := NewMockStateManager()
	stateManager.protocolParams = &conway.ConwayProtocolParameters{
		ProtocolVersion: common.ProtocolParametersProtocolVersion{Major: 10},
		DRepVotingThresholds: conway.DRepVotingThresholds{
			CommitteeNormal: cbor.Rat{Rat: new(big.Rat)},
		},
		PoolVotingThresholds: conway.PoolVotingThresholds{
			CommitteeNormal: cbor.Rat{Rat: big.NewRat(3, 5)},
		},
	}
	stateManager.govState.CommitteeMembersByCredential[ledger.RewardAccountKey{
		Credential: common.Blake2b224{0x01},
	}] = &CommitteeMemberInfo{}
	yesStake := ledger.RewardAccountKey{Credential: common.Blake2b224{0x11}}
	noStake := ledger.RewardAccountKey{Credential: common.Blake2b224{0x12}}
	yesPool := common.PoolKeyHash{0x21}
	noPool := common.PoolKeyHash{0x22}
	stateManager.rewardAccounts[yesStake] = 40
	stateManager.rewardAccounts[noStake] = 60
	stateManager.govState.PoolRegistrations[yesPool] = true
	stateManager.govState.PoolRegistrations[noPool] = true
	stateManager.govState.PoolDelegationsByCredential[yesStake] = yesPool
	stateManager.govState.PoolDelegationsByCredential[noStake] = noPool
	proposal := &ProposalState{GovActionInfo: GovActionInfo{
		ActionType: common.GovActionTypeUpdateCommittee,
		Votes: map[string]uint8{
			fmt.Sprintf(
				"%d:%s",
				common.VoterTypeStakingPoolKeyHash,
				hex.EncodeToString(yesPool[:]),
			): 1,
		},
	}}

	stake := stateManager.credentialVotingStake()
	assert.True(t, stateManager.drepAcceptedForUpdateCommittee(
		proposal,
		stake,
		new(big.Rat),
	))
	assert.False(t, stateManager.spoAcceptedForUpdateCommittee(
		proposal,
		stake,
		big.NewRat(3, 5),
	))
	assert.False(t, stateManager.updateCommitteeAccepted(proposal))
	proposal.Votes[fmt.Sprintf(
		"%d:%s",
		common.VoterTypeStakingPoolKeyHash,
		hex.EncodeToString(noPool[:]),
	)] = 1
	assert.True(t, stateManager.spoAcceptedForUpdateCommittee(
		proposal,
		stake,
		big.NewRat(3, 5),
	))
	assert.True(t, stateManager.updateCommitteeAccepted(proposal))
}

func TestUpdateCommitteeZeroThresholdRatifiesWithoutVotes(t *testing.T) {
	stateManager := NewMockStateManager()
	stateManager.protocolParams = &conway.ConwayProtocolParameters{
		DRepVotingThresholds: conway.DRepVotingThresholds{
			CommitteeNoConfidence: cbor.Rat{Rat: new(big.Rat)},
		},
		PoolVotingThresholds: conway.PoolVotingThresholds{
			CommitteeNoConfidence: cbor.Rat{Rat: new(big.Rat)},
		},
	}
	stateManager.govState.Proposals["update#0"] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType:     common.GovActionTypeUpdateCommittee,
			SubmittedEpoch: 0,
			ExpiresAfter:   10,
		},
	}

	require.NoError(t, stateManager.ProcessEpochBoundary(1))
	proposal := stateManager.govState.Proposals["update#0"]
	require.NotNil(t, proposal)
	require.NotNil(t, proposal.RatifiedEpoch)
	assert.Equal(t, uint64(1), *proposal.RatifiedEpoch)
}
