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
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/ouroboros-mock/address"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// govFixture builds a Conway governance state in which every voter class has
// explicit stake, so each test can state the exact vote weights it exercises.
type govFixture struct {
	t    *testing.T
	sm   *MockStateManager
	seed byte
	// lastCold and lastPool identify the most recently added member and pool.
	lastCold common.Blake2b224
	lastPool common.PoolKeyHash
}

func newGovFixture(t *testing.T) *govFixture {
	t.Helper()
	half := func() cbor.Rat { return cbor.Rat{Rat: big.NewRat(1, 2)} }
	sm := NewMockStateManager()
	sm.protocolParams = &conway.ConwayProtocolParameters{
		ProtocolVersion: common.ProtocolParametersProtocolVersion{Major: 10},
		DRepVotingThresholds: conway.DRepVotingThresholds{
			MotionNoConfidence:    half(),
			CommitteeNormal:       half(),
			CommitteeNoConfidence: half(),
			UpdateToConstitution:  half(),
			HardForkInitiation:    half(),
			PpNetworkGroup:        half(),
			PpEconomicGroup:       half(),
			PpTechnicalGroup:      half(),
			PpGovGroup:            half(),
			TreasuryWithdrawal:    half(),
		},
		PoolVotingThresholds: conway.PoolVotingThresholds{
			MotionNoConfidence:    half(),
			CommitteeNormal:       half(),
			CommitteeNoConfidence: half(),
			HardForkInitiation:    half(),
			PpSecurityGroup:       half(),
		},
	}
	return &govFixture{t: t, sm: sm}
}

func requireRatEqual(t *testing.T, expected, actual *big.Rat) {
	t.Helper()
	require.NotNil(t, actual)
	require.Zero(t, actual.Cmp(expected))
}

func (f *govFixture) params() *conway.ConwayProtocolParameters {
	return f.sm.protocolParams.(*conway.ConwayProtocolParameters)
}

func (f *govFixture) next() common.Blake2b224 {
	f.seed++
	return common.Blake2b224{f.seed}
}

func keyCredential(hash common.Blake2b224) ledger.RewardAccountKey {
	return ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: hash,
	}
}

// delegator registers a stake credential holding stake lovelace and returns it.
func (f *govFixture) delegator(stake uint64) ledger.RewardAccountKey {
	credential := keyCredential(f.next())
	f.sm.rewardAccounts[credential] = stake
	f.sm.stakeRegistrations[credential] = stake
	f.sm.govState.StakeRegistrationsByCredential[credential] = true
	return credential
}

// drep registers a DRep with the given delegated stake and returns its vote key.
func (f *govFixture) drep(stake uint64) string {
	hash := f.next()
	f.sm.govState.DRepRegistrationsByCredential[keyCredential(hash)] = true
	f.sm.govState.DRepDelegationsByCredential[f.delegator(stake)] = common.Drep{
		Type:       common.DrepTypeAddrKeyHash,
		Credential: hash[:],
	}
	return voteKey(common.VoterTypeDRepKeyHash, hash)
}

// predefinedDRep delegates stake to an always-abstain or no-confidence DRep.
func (f *govFixture) predefinedDRep(drepType int, stake uint64) {
	f.sm.govState.DRepDelegationsByCredential[f.delegator(stake)] = common.Drep{
		Type: drepType,
	}
}

// pool registers a pool with the given delegated stake and returns its vote key.
func (f *govFixture) pool(stake uint64) string {
	hash := f.next()
	f.lastPool = hash
	f.sm.govState.PoolRegistrations[hash] = true
	f.sm.govState.PoolDelegationsByCredential[f.delegator(stake)] = hash
	return voteKey(common.VoterTypeStakingPoolKeyHash, hash)
}

// ccMember adds a committee member and returns its hot-key vote key. An
// unauthorized member has no hot credential on record.
func (f *govFixture) ccMember(expiry uint64, authorized bool) string {
	cold, hot := f.next(), f.next()
	f.lastCold = cold
	f.sm.govState.CommitteeMembersByCredential[keyCredential(cold)] = &CommitteeMemberInfo{
		ColdCredential: common.Credential{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: cold,
		},
		ColdKey:     cold,
		ExpiryEpoch: expiry,
	}
	if authorized {
		f.sm.govState.AuthorizeHotKey(cold, hot)
	}
	return voteKey(common.VoterTypeConstitutionalCommitteeHotKeyHash, hot)
}

func (f *govFixture) committeeThreshold(num, den int64) {
	f.sm.govState.CommitteeThreshold = big.NewRat(num, den)
}

func (f *govFixture) propose(id string, info GovActionInfo) {
	info.ExpiresAfter = 10
	f.sm.govState.AddProposal(id, info)
}

func (f *govFixture) ratified(id string) bool {
	f.t.Helper()
	proposal := f.sm.govState.Proposals[id]
	require.NotNil(f.t, proposal, "proposal %s", id)
	return proposal.RatifiedEpoch != nil
}

func yes(keys ...string) map[string]uint8 {
	votes := map[string]uint8{}
	for _, key := range keys {
		votes[key] = 1
	}
	return votes
}

func TestRatificationDRepThresholdIsStakeWeighted(t *testing.T) {
	t.Parallel()
	// Two DReps hold 60 and 40 lovelace; the constitution threshold is 2/3.
	for _, test := range []struct {
		name      string
		threshold *big.Rat
		ratified  bool
	}{
		{"below threshold", big.NewRat(2, 3), false},
		{"at threshold", big.NewRat(3, 5), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			f := newGovFixture(t)
			f.params().DRepVotingThresholds.UpdateToConstitution = cbor.Rat{
				Rat: test.threshold,
			}
			large := f.drep(60)
			f.drep(40)
			cc := f.ccMember(10, true)
			f.committeeThreshold(1, 2)
			f.propose("constitution#0", GovActionInfo{
				ActionType: common.GovActionTypeNewConstitution,
				Votes:      yes(large, cc),
			})
			require.NoError(t, f.sm.ProcessEpochBoundary(1))
			assert.Equal(t, test.ratified, f.ratified("constitution#0"))
		})
	}
}

func TestRatificationDRepAbstainAndInactiveStakeIsExcluded(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.params().DRepVotingThresholds.UpdateToConstitution = cbor.Rat{
		Rat: big.NewRat(2, 3),
	}
	voter := f.drep(50)
	f.predefinedDRep(common.DrepTypeAbstain, 1000)
	// 50 yes of 50 counted stake; the abstaining 1000 is not in the base.
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	f.propose("constitution#0", GovActionInfo{
		ActionType: common.GovActionTypeNewConstitution,
		Votes:      yes(voter, cc),
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.True(t, f.ratified("constitution#0"))
}

func TestRatificationNoConfidenceDRepStakeCountsAsYesOnlyForNoConfidence(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.params().DRepVotingThresholds.MotionNoConfidence = cbor.Rat{
		Rat: big.NewRat(3, 4),
	}
	f.params().PoolVotingThresholds.MotionNoConfidence = cbor.Rat{Rat: new(big.Rat)}
	f.predefinedDRep(common.DrepTypeNoConfidence, 80)
	f.drep(20)
	f.committeeThreshold(1, 2)
	f.propose("noconfidence#0", GovActionInfo{
		ActionType: common.GovActionTypeNoConfidence,
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.True(t, f.ratified("noconfidence#0"),
		"always-no-confidence stake is a yes vote on a no-confidence motion")

	g := newGovFixture(t)
	g.params().DRepVotingThresholds.UpdateToConstitution = cbor.Rat{
		Rat: big.NewRat(1, 4),
	}
	g.predefinedDRep(common.DrepTypeNoConfidence, 80)
	voter := g.drep(20)
	cc := g.ccMember(10, true)
	g.committeeThreshold(1, 2)
	g.propose("constitution#0", GovActionInfo{
		ActionType: common.GovActionTypeNewConstitution,
		Votes:      yes(voter, cc),
	})
	require.NoError(t, g.sm.ProcessEpochBoundary(1))
	assert.False(t, g.ratified("constitution#0"),
		"always-no-confidence stake is a no vote on any other action")
}

func TestRatificationBootstrapPhaseIgnoresDRepThresholds(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.params().ProtocolVersion.Major = common.ProtocolVersionConway
	f.params().DRepVotingThresholds.HardForkInitiation = cbor.Rat{
		Rat: big.NewRat(1, 1),
	}
	f.params().PoolVotingThresholds.HardForkInitiation = cbor.Rat{
		Rat: big.NewRat(1, 2),
	}
	f.drep(100)
	pool := f.pool(100)
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	f.propose("hardfork#0", GovActionInfo{
		ActionType: common.GovActionTypeHardForkInitiation,
		Votes:      yes(pool, cc),
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.True(t, f.ratified("hardfork#0"))
}

func TestRatificationHardForkSPONonVotersCountAsNo(t *testing.T) {
	t.Parallel()
	// During bootstrap a non-voting pool abstains on every other action, so
	// only the hard fork rule counts its stake against the action.
	f := newGovFixture(t)
	f.params().ProtocolVersion.Major = common.ProtocolVersionConway
	f.params().PoolVotingThresholds.HardForkInitiation = cbor.Rat{
		Rat: big.NewRat(2, 3),
	}
	voter := f.drep(10)
	yesPool := f.pool(50)
	f.pool(50)
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	f.propose("hardfork#0", GovActionInfo{
		ActionType: common.GovActionTypeHardForkInitiation,
		Votes:      yes(voter, yesPool, cc),
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.False(t, f.ratified("hardfork#0"), "50 of 100 pool stake is below 2/3")
}

func TestRatificationParameterChangeUsesTouchedGroupThresholds(t *testing.T) {
	t.Parallel()
	economicOnly := &conway.ConwayProtocolParameterUpdate{
		PoolDeposit: new(uint),
	}
	securityAndEconomic := &conway.ConwayProtocolParameterUpdate{
		PoolDeposit: new(uint),
		MinFeeA:     new(uint),
	}
	for _, test := range []struct {
		name     string
		update   *conway.ConwayProtocolParameterUpdate
		ratified bool
	}{
		// DRep stake is 40 of 100 yes. The economic threshold is 1/3 and the
		// technical threshold is 1/2, so only an economic-only change passes.
		{"economic group only", economicOnly, true},
		{"economic and technical group", &conway.ConwayProtocolParameterUpdate{
			PoolDeposit: new(uint),
			MaxEpoch:    new(uint),
		}, false},
		// The network threshold is zero here, so a network-group change passes.
		{"max collateral inputs is network", &conway.ConwayProtocolParameterUpdate{
			MaxCollateralInputs: new(uint),
		}, true},
		// The security group adds the pool vote, and no pool voted yes.
		{"security group needs pools", securityAndEconomic, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			f := newGovFixture(t)
			f.params().DRepVotingThresholds.PpEconomicGroup = cbor.Rat{
				Rat: big.NewRat(1, 3),
			}
			f.params().DRepVotingThresholds.PpTechnicalGroup = cbor.Rat{
				Rat: big.NewRat(1, 2),
			}
			f.params().DRepVotingThresholds.PpNetworkGroup = cbor.Rat{Rat: new(big.Rat)}
			f.params().DRepVotingThresholds.PpGovGroup = cbor.Rat{Rat: new(big.Rat)}
			yesDRep := f.drep(40)
			f.drep(60)
			f.pool(100)
			cc := f.ccMember(10, true)
			f.committeeThreshold(1, 2)
			f.propose("params#0", GovActionInfo{
				ActionType:      common.GovActionTypeParameterChange,
				ParameterUpdate: test.update,
				Votes:           yes(yesDRep, cc),
			})
			require.NoError(t, f.sm.ProcessEpochBoundary(1))
			assert.Equal(t, test.ratified, f.ratified("params#0"))
		})
	}
}

func TestRatificationCommitteeThresholdCountsEligibleMembers(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		setup    func(f *govFixture) []string
		ratified bool
	}{
		{
			name: "no votes among three members",
			setup: func(f *govFixture) []string {
				f.ccMember(10, true)
				f.ccMember(10, true)
				f.ccMember(10, true)
				return nil
			},
		},
		{
			name: "one of three yes is below two thirds",
			setup: func(f *govFixture) []string {
				first := f.ccMember(10, true)
				f.ccMember(10, true)
				f.ccMember(10, true)
				return []string{first}
			},
		},
		{
			name: "expired member leaves the denominator",
			setup: func(f *govFixture) []string {
				first := f.ccMember(10, true)
				f.ccMember(0, true)
				return []string{first}
			},
			ratified: true,
		},
		{
			name: "resigned member leaves the denominator",
			setup: func(f *govFixture) []string {
				first := f.ccMember(10, true)
				f.ccMember(10, true)
				f.sm.govState.ResignCommitteeMember(f.lastCold)
				return []string{first}
			},
			ratified: true,
		},
		{
			name: "member without hot key leaves the denominator",
			setup: func(f *govFixture) []string {
				first := f.ccMember(10, true)
				f.ccMember(10, false)
				return []string{first}
			},
			ratified: true,
		},
		{
			name: "two of three yes",
			setup: func(f *govFixture) []string {
				first, second := f.ccMember(10, true), f.ccMember(10, true)
				f.ccMember(10, true)
				return []string{first, second}
			},
			ratified: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			f := newGovFixture(t)
			f.sm.currentEpoch = 1
			f.sm.govState.CurrentEpoch = 1
			f.params().MinCommitteeSize = 0
			drep := f.drep(100)
			f.committeeThreshold(2, 3)
			votes := yes(append(test.setup(f), drep)...)
			f.propose("constitution#0", GovActionInfo{
				ActionType:     common.GovActionTypeNewConstitution,
				SubmittedEpoch: 1,
				Votes:          votes,
			})
			require.NoError(t, f.sm.ProcessEpochBoundary(2))
			assert.Equal(t, test.ratified, f.ratified("constitution#0"))
		})
	}
}

func TestRatificationRequiresCommitteeForCommitteeVotedActions(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	drep := f.drep(100)
	f.propose("constitution#0", GovActionInfo{
		ActionType: common.GovActionTypeNewConstitution,
		Votes:      yes(drep),
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.False(t, f.ratified("constitution#0"),
		"no committee means a committee-voted action cannot ratify")
}

func TestRatificationEnforcesMinCommitteeSize(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.params().MinCommitteeSize = 2
	drep := f.drep(100)
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	f.propose("constitution#0", GovActionInfo{
		ActionType: common.GovActionTypeNewConstitution,
		Votes:      yes(drep, cc),
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.False(t, f.ratified("constitution#0"),
		"one active member is below the minimum committee size")
}

func TestRatificationRequiresParentToBeEnactedRoot(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	drep := f.drep(100)
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	root := "root#0"
	f.sm.govState.Roots.ProtocolParameters = &root
	// Parameter changes do not delay later actions, so each proposal is
	// judged on its own parent.
	for id, parent := range map[string]*string{
		"orphan#0":   ptr("stale#0"),
		"child#0":    &root,
		"rootless#0": nil,
	} {
		f.propose(id, GovActionInfo{
			ActionType:     common.GovActionTypeParameterChange,
			ParentActionId: parent,
			Votes:          yes(drep, cc),
		})
	}
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.True(t, f.ratified("child#0"))
	assert.False(t, f.ratified("orphan#0"), "parent is not the enacted root")
	assert.False(t, f.ratified("rootless#0"), "a root exists, so a parent is required")
}

func TestRatificationDelaysLaterActionsAfterDelayingAction(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.params().PoolVotingThresholds.MotionNoConfidence = cbor.Rat{Rat: new(big.Rat)}
	drep := f.drep(100)
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	f.propose("noconfidence#0", GovActionInfo{
		ActionType: common.GovActionTypeNoConfidence,
		Votes:      yes(drep),
	})
	f.propose("constitution#0", GovActionInfo{
		ActionType: common.GovActionTypeNewConstitution,
		Votes:      yes(drep, cc),
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.True(t, f.ratified("noconfidence#0"))
	assert.False(t, f.ratified("constitution#0"),
		"a ratified no-confidence motion delays every later action in the epoch")
}

func TestRatificationUpdatesCommitteeThresholdOnEnactment(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.params().PoolVotingThresholds.CommitteeNormal = cbor.Rat{Rat: new(big.Rat)}
	drep := f.drep(100)
	f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	proposedThreshold := big.NewRat(3, 4)
	f.propose("update#0", GovActionInfo{
		ActionType:        common.GovActionTypeUpdateCommittee,
		ProposedThreshold: proposedThreshold,
		Votes:             yes(drep),
	})
	proposedThreshold.SetInt64(0)
	storedProposal := f.sm.govState.Proposals["update#0"]
	require.NotNil(t, storedProposal)
	requireRatEqual(
		t, big.NewRat(3, 4), storedProposal.ProposedThreshold,
	)
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	require.True(t, f.ratified("update#0"))
	require.NoError(t, f.sm.ProcessEpochBoundary(2))
	requireRatEqual(t, big.NewRat(3, 4), f.sm.govState.CommitteeThreshold)

	f.propose("noconfidence#0", GovActionInfo{
		ActionType:     common.GovActionTypeNoConfidence,
		ParentActionId: ptr("update#0"),
		Votes:          yes(drep),
	})
	f.params().PoolVotingThresholds.MotionNoConfidence = cbor.Rat{Rat: new(big.Rat)}
	require.NoError(t, f.sm.ProcessEpochBoundary(3))
	require.NoError(t, f.sm.ProcessEpochBoundary(4))
	assert.Nil(t, f.sm.govState.CommitteeThreshold,
		"no-confidence removes the committee and its threshold")
}

func TestGovernanceRationalStateOwnsCopies(t *testing.T) {
	t.Parallel()
	initialThreshold := big.NewRat(2, 3)
	proposalThreshold := big.NewRat(3, 4)
	state := NewGovernanceState()
	state.LoadFromParsedState(&ParsedInitialState{
		CommitteeThreshold: initialThreshold,
		Proposals: map[string]GovActionInfo{
			"update#0": {
				ActionType:        common.GovActionTypeUpdateCommittee,
				ProposedThreshold: proposalThreshold,
			},
		},
	})
	initialThreshold.SetInt64(0)
	proposalThreshold.SetInt64(0)
	requireRatEqual(t, big.NewRat(2, 3), state.CommitteeThreshold)
	storedProposal := state.Proposals["update#0"]
	require.NotNil(t, storedProposal)
	requireRatEqual(t, big.NewRat(3, 4), storedProposal.ProposedThreshold)

	cloned := cloneGovernanceState(state)
	require.NotNil(t, cloned)
	require.NotNil(t, cloned.CommitteeThreshold)
	cloned.CommitteeThreshold.SetInt64(0)
	clonedProposal := cloned.Proposals["update#0"]
	require.NotNil(t, clonedProposal)
	require.NotNil(t, clonedProposal.ProposedThreshold)
	clonedProposal.ProposedThreshold.SetInt64(0)
	requireRatEqual(t, big.NewRat(2, 3), state.CommitteeThreshold)
	requireRatEqual(t, big.NewRat(3, 4), storedProposal.ProposedThreshold)

	manager := NewMockStateManager()
	enactedThreshold := big.NewRat(4, 5)
	require.NoError(t, manager.enactProposal("update#1", &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType:        common.GovActionTypeUpdateCommittee,
			ProposedThreshold: enactedThreshold,
		},
	}))
	enactedThreshold.SetInt64(0)
	requireRatEqual(t, big.NewRat(4, 5), manager.govState.CommitteeThreshold)
}

func TestRatificationTreasuryWithdrawalRespectsTreasury(t *testing.T) {
	t.Parallel()
	recipient := keyCredential(common.Blake2b224{0x77})
	for _, test := range []struct {
		name     string
		treasury uint64
		ratified bool
	}{
		{"treasury covers withdrawal", 100, true},
		{"treasury short of withdrawal", 99, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			f := newGovFixture(t)
			f.sm.govState.Treasury = test.treasury
			drep := f.drep(100)
			cc := f.ccMember(10, true)
			f.committeeThreshold(1, 2)
			f.propose("withdraw#0", GovActionInfo{
				ActionType: common.GovActionTypeTreasuryWithdrawal,
				Withdrawals: map[ledger.RewardAccountKey]uint64{
					recipient:                              60,
					keyCredential(common.Blake2b224{0x78}): 40,
				},
				Votes: yes(drep, cc),
			})
			require.NoError(t, f.sm.ProcessEpochBoundary(1))
			assert.Equal(t, test.ratified, f.ratified("withdraw#0"))
		})
	}
}

func TestRatificationRejectsOverflowingTreasuryWithdrawal(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.sm.govState.Treasury = 1
	drep := f.drep(100)
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	f.propose("withdraw#0", GovActionInfo{
		ActionType: common.GovActionTypeTreasuryWithdrawal,
		Withdrawals: map[ledger.RewardAccountKey]uint64{
			keyCredential(common.Blake2b224{0x77}): ^uint64(0) - 5,
			keyCredential(common.Blake2b224{0x78}): 7,
		},
		Votes: yes(drep, cc),
	})

	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.False(t, f.ratified("withdraw#0"))
}

func TestRatificationAcceptsMaximumTreasuryWithdrawal(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.sm.govState.Treasury = ^uint64(0)
	drep := f.drep(100)
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	f.propose("withdraw#0", GovActionInfo{
		ActionType: common.GovActionTypeTreasuryWithdrawal,
		Withdrawals: map[ledger.RewardAccountKey]uint64{
			keyCredential(common.Blake2b224{0x77}): ^uint64(0) - 1,
			keyCredential(common.Blake2b224{0x78}): 1,
		},
		Votes: yes(drep, cc),
	})

	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.True(t, f.ratified("withdraw#0"))
}

func TestEnactmentRejectsOverflowingTreasuryWithdrawalAtomically(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.sm.govState.Treasury = 10
	first := f.delegator(1)
	second := f.delegator(2)
	ratifiedEpoch := uint64(1)
	f.propose("withdraw#0", GovActionInfo{
		ActionType: common.GovActionTypeTreasuryWithdrawal,
		Withdrawals: map[ledger.RewardAccountKey]uint64{
			first:  ^uint64(0) - 5,
			second: 7,
		},
	})
	proposal := f.sm.govState.Proposals["withdraw#0"]
	require.NotNil(t, proposal)
	proposal.RatifiedEpoch = &ratifiedEpoch

	err := f.sm.ProcessEpochBoundary(2)
	require.ErrorContains(t, err, "treasury withdrawal exceeds available treasury")
	assert.Equal(t, uint64(10), f.sm.govState.Treasury)
	assert.Equal(t, uint64(1), f.sm.rewardAccounts[first])
	assert.Equal(t, uint64(2), f.sm.rewardAccounts[second])
	assert.Contains(t, f.sm.govState.Proposals, "withdraw#0")
	assert.NotContains(t, f.sm.govState.EnactedProposals, "withdraw#0")
}

func TestEnactmentRejectsRewardAccountOverflowAtomically(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.sm.govState.Treasury = 1
	recipient := f.delegator(^uint64(0))
	ratifiedEpoch := uint64(1)
	f.propose("withdraw#0", GovActionInfo{
		ActionType: common.GovActionTypeTreasuryWithdrawal,
		Withdrawals: map[ledger.RewardAccountKey]uint64{
			recipient: 1,
		},
	})
	proposal := f.sm.govState.Proposals["withdraw#0"]
	require.NotNil(t, proposal)
	proposal.RatifiedEpoch = &ratifiedEpoch

	err := f.sm.ProcessEpochBoundary(2)
	require.ErrorContains(t, err, "treasury withdrawal overflows reward account")
	assert.Equal(t, ^uint64(0), f.sm.rewardAccounts[recipient])
	assert.Equal(t, uint64(1), f.sm.govState.Treasury)
	assert.Contains(t, f.sm.govState.Proposals, "withdraw#0")
	assert.NotContains(t, f.sm.govState.EnactedProposals, "withdraw#0")
}

func TestEnactmentMovesTreasuryToRegisteredAccounts(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.sm.govState.Treasury = 100
	registered := f.delegator(5)
	unregistered := keyCredential(common.Blake2b224{0x79})
	drep := f.drep(100)
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	f.propose("withdraw#0", GovActionInfo{
		ActionType: common.GovActionTypeTreasuryWithdrawal,
		Withdrawals: map[ledger.RewardAccountKey]uint64{
			registered:   60,
			unregistered: 30,
		},
		Votes: yes(drep, cc),
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	require.NoError(t, f.sm.ProcessEpochBoundary(2))
	assert.Equal(t, uint64(65), f.sm.rewardAccounts[registered])
	assert.Equal(t, uint64(65), f.sm.govState.RewardAccountBalances[registered])
	assert.NotContains(t, f.sm.rewardAccounts, unregistered)
	assert.Equal(t, uint64(40), f.sm.govState.Treasury,
		"a withdrawal to an unregistered account stays in the treasury")
}

func TestProposalDepositRefundedOnExpiry(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	registered := f.delegator(5)
	unregistered := keyCredential(common.Blake2b224{0x7a})
	f.sm.govState.Treasury = 10
	f.sm.govState.AddProposal("kept#0", GovActionInfo{
		ActionType:    common.GovActionTypeInfo,
		Deposit:       50,
		ReturnAccount: &registered,
		ExpiresAfter:  3,
	})
	f.sm.govState.AddProposal("lost#0", GovActionInfo{
		ActionType:    common.GovActionTypeInfo,
		Deposit:       7,
		ReturnAccount: &unregistered,
		ExpiresAfter:  3,
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(3))
	assert.Contains(t, f.sm.govState.Proposals, "kept#0")
	require.NoError(t, f.sm.ProcessEpochBoundary(4))
	assert.NotContains(t, f.sm.govState.Proposals, "kept#0")
	assert.Equal(t, uint64(55), f.sm.rewardAccounts[registered])
	assert.Equal(t, uint64(17), f.sm.govState.Treasury,
		"a deposit with no registered return account moves to the treasury")
}

func TestProposalDepositRefundOverflowIsAtomic(t *testing.T) {
	t.Parallel()

	t.Run("registered reward account", func(t *testing.T) {
		t.Parallel()
		f := newGovFixture(t)
		account := f.delegator(^uint64(0))
		f.sm.govState.AddProposal("expired#0", GovActionInfo{
			ActionType:    common.GovActionTypeInfo,
			Deposit:       1,
			ReturnAccount: &account,
			ExpiresAfter:  1,
		})

		err := f.sm.ProcessEpochBoundary(2)
		require.ErrorContains(t, err, "proposal deposit refund overflows reward account")
		assert.Equal(t, ^uint64(0), f.sm.rewardAccounts[account])
		assert.Contains(t, f.sm.govState.Proposals, "expired#0")
	})

	t.Run("treasury", func(t *testing.T) {
		t.Parallel()
		f := newGovFixture(t)
		f.sm.govState.Treasury = ^uint64(0)
		account := keyCredential(common.Blake2b224{0x7b})
		f.sm.govState.AddProposal("expired#0", GovActionInfo{
			ActionType:    common.GovActionTypeInfo,
			Deposit:       1,
			ReturnAccount: &account,
			ExpiresAfter:  1,
		})

		err := f.sm.ProcessEpochBoundary(2)
		require.ErrorContains(t, err, "proposal deposit refund overflows treasury")
		assert.Equal(t, ^uint64(0), f.sm.govState.Treasury)
		assert.Contains(t, f.sm.govState.Proposals, "expired#0")
	})
}

func TestEnactmentRemovesOrphanedSiblingsAndRefundsDeposits(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	refund := f.delegator(0)
	drep := f.drep(100)
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	f.propose("winner#0", GovActionInfo{
		ActionType: common.GovActionTypeNewConstitution,
		Votes:      yes(drep, cc),
	})
	f.propose("sibling#0", GovActionInfo{
		ActionType:    common.GovActionTypeNewConstitution,
		Deposit:       9,
		ReturnAccount: &refund,
	})
	winner := "winner#0"
	f.propose("grandchild#0", GovActionInfo{
		ActionType:     common.GovActionTypeNewConstitution,
		ParentActionId: &winner,
	})
	f.propose("descendant#0", GovActionInfo{
		ActionType:     common.GovActionTypeNewConstitution,
		ParentActionId: ptr("sibling#0"),
		Deposit:        4,
		ReturnAccount:  &refund,
	})
	// Equal votes would make the sibling ratify too; only the winner has them.
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	require.NoError(t, f.sm.ProcessEpochBoundary(2))
	assert.Equal(t, &winner, f.sm.govState.Roots.Constitution)
	assert.NotContains(t, f.sm.govState.Proposals, "sibling#0")
	assert.NotContains(t, f.sm.govState.Proposals, "descendant#0")
	assert.Contains(t, f.sm.govState.Proposals, "grandchild#0",
		"a child of the enacted action stays valid")
	assert.Equal(t, uint64(13), f.sm.rewardAccounts[refund])
}

func ptr[T any](value T) *T { return &value }

func TestParseInitialStateCarriesTreasuryAndCommitteeThreshold(t *testing.T) {
	t.Parallel()
	// AccountState is [treasury, reserves] and the committee is
	// [[members, threshold]] with the threshold as tag 30.
	account := "821903e81907d0"
	committee := "8182a18200581c" + strings.Repeat("11", common.Blake2b224Size) +
		"1864d81e820304"
	state := parseSyntheticInitialStateWithAccountAndCommittee(
		t, "", "", "", "", account, committee,
	)
	assert.Equal(t, uint64(1000), state.Treasury)
	require.NotNil(t, state.CommitteeThreshold)
	assert.Equal(t, 0, state.CommitteeThreshold.Cmp(big.NewRat(3, 4)))

	manager := NewMockStateManager()
	require.NoError(t, manager.LoadInitialState(
		state,
		&conway.ConwayProtocolParameters{},
	))
	assert.Equal(t, uint64(1000), manager.govState.Treasury)
	assert.Equal(t, 0, manager.govState.CommitteeThreshold.Cmp(big.NewRat(3, 4)))
}

func TestParseProposalCarriesCommitteeQuorumAndWithdrawals(t *testing.T) {
	t.Parallel()
	txID := strings.Repeat("ee", common.Blake2b256Size)
	updateTagged := "8504f6d9010280a0d81e8219012c190190"
	withdraw := "8302a1581de0" + strings.Repeat("aa", common.Blake2b224Size) +
		"1832f6"
	raw := "85a2" +
		"825820" + txID + "03" + proposalRecordBody(txID, "03", updateTagged) +
		"825820" + txID + "04" + proposalRecordBody(txID, "04", withdraw) +
		"f6f6f6f6"
	state := parseSyntheticInitialState(t, "", "", "", raw)

	updateInfo, ok := state.Proposals[txID+"#3"]
	require.True(t, ok)
	require.NotNil(t, updateInfo.ProposedThreshold)
	assert.Equal(t, 0, updateInfo.ProposedThreshold.Cmp(big.NewRat(3, 4)))

	withdrawInfo, ok := state.Proposals[txID+"#4"]
	require.True(t, ok)
	assert.Equal(t, common.GovActionTypeTreasuryWithdrawal, withdrawInfo.ActionType)
	assert.Equal(t, map[ledger.RewardAccountKey]uint64{
		keyCredential(filledBlake2b224(0xaa)): 50,
	}, withdrawInfo.Withdrawals)
}

func proposalRecordBody(txID, index, action string) string {
	return "87825820" + txID + index + "a0a0a0" +
		"841864581de0" + strings.Repeat("f1", common.Blake2b224Size) +
		action + "8218771878" + "18411863"
}

func TestApplyTransactionRecordsQuorumAndWithdrawals(t *testing.T) {
	t.Parallel()
	recipient, err := address.NewAddress().
		WithStakingKeyHash(bytes.Repeat([]byte{0x31}, common.Blake2b224Size)).
		BuildReward()
	require.NoError(t, err)
	update := &common.UpdateCommitteeGovAction{
		Type:   uint(common.GovActionTypeUpdateCommittee),
		Quorum: cbor.Rat{Rat: big.NewRat(2, 3)},
	}
	withdrawal := &common.TreasuryWithdrawalGovAction{
		Type:        uint(common.GovActionTypeTreasuryWithdrawal),
		Withdrawals: map[*common.Address]uint64{&recipient: 77},
	}
	proposal := func(actionType common.GovActionType, action common.GovAction) conway.ConwayProposalProcedure {
		return conway.ConwayProposalProcedure{
			PPGovAction: conway.ConwayGovAction{
				Type:   uint(actionType),
				Action: action,
			},
		}
	}
	tx := ledger.NewTransactionBuilder().WithProposalProcedures(
		proposal(common.GovActionTypeUpdateCommittee, update),
		proposal(common.GovActionTypeTreasuryWithdrawal, withdrawal),
	)
	manager := NewMockStateManager()
	require.NoError(t, manager.ApplyTransaction(tx, 0))

	var gotQuorum *big.Rat
	var gotWithdrawals map[ledger.RewardAccountKey]uint64
	for _, proposal := range manager.govState.Proposals {
		switch proposal.ActionType {
		case common.GovActionTypeUpdateCommittee:
			gotQuorum = proposal.ProposedThreshold
		case common.GovActionTypeTreasuryWithdrawal:
			gotWithdrawals = proposal.Withdrawals
		}
	}
	require.NotNil(t, gotQuorum)
	update.Quorum.Rat.SetInt64(0)
	assert.Equal(t, 0, gotQuorum.Cmp(big.NewRat(2, 3)))
	assert.Equal(t, map[ledger.RewardAccountKey]uint64{
		keyCredential(filledBlake2b224(0x31)): 77,
	}, gotWithdrawals)
}

func TestRatificationSPODefaultVotesFollowRewardAccountDRep(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		drepType int
		ratified bool
	}{
		{"always no confidence is yes on no-confidence", common.DrepTypeNoConfidence, true},
		{"always abstain leaves the base", common.DrepTypeAbstain, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			f := newGovFixture(t)
			f.params().DRepVotingThresholds.MotionNoConfidence = cbor.Rat{Rat: new(big.Rat)}
			f.params().PoolVotingThresholds.MotionNoConfidence = cbor.Rat{
				Rat: big.NewRat(1, 2),
			}
			f.pool(60)
			rewardAccount := f.delegator(0)
			f.sm.govState.PoolRewardAccounts[f.lastPool] = rewardAccount
			f.sm.govState.DRepDelegationsByCredential[rewardAccount] = common.Drep{
				Type: test.drepType,
			}
			f.pool(40)
			f.propose("noconfidence#0", GovActionInfo{
				ActionType: common.GovActionTypeNoConfidence,
			})
			require.NoError(t, f.sm.ProcessEpochBoundary(1))
			assert.Equal(t, test.ratified, f.ratified("noconfidence#0"))
		})
	}
}

func TestEnactmentHardForkSetsProtocolVersion(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.params().PoolVotingThresholds.HardForkInitiation = cbor.Rat{Rat: new(big.Rat)}
	drep := f.drep(100)
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	f.propose("hardfork#0", GovActionInfo{
		ActionType:      common.GovActionTypeHardForkInitiation,
		ProtocolVersion: &ProtocolVersionInfo{Major: 11, Minor: 2},
		Votes:           yes(drep, cc),
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	require.NoError(t, f.sm.ProcessEpochBoundary(2))
	assert.Equal(t, uint(11), f.params().ProtocolVersion.Major)
	assert.Equal(t, uint(2), f.params().ProtocolVersion.Minor)
}

func TestExpiredProposalRemovesDescendants(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	refund := f.delegator(0)
	parent := "parent#0"
	f.sm.govState.AddProposal(parent, GovActionInfo{
		ActionType:   common.GovActionTypeParameterChange,
		ExpiresAfter: 3,
	})
	f.sm.govState.AddProposal("child#0", GovActionInfo{
		ActionType:     common.GovActionTypeParameterChange,
		ParentActionId: &parent,
		Deposit:        6,
		ReturnAccount:  &refund,
		ExpiresAfter:   10,
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(4))
	assert.Empty(t, f.sm.govState.Proposals,
		"a child cannot outlive its expired parent")
	assert.Equal(t, uint64(6), f.sm.rewardAccounts[refund])
}

func TestRatificationRejectsUpdateCommitteeBeyondTermLimit(t *testing.T) {
	t.Parallel()
	// Ratifying at epoch 1 with a term limit of 5 admits expiries up to 6.
	for _, test := range []struct {
		name     string
		expiry   uint64
		ratified bool
	}{
		{"within term limit", 6, true},
		{"beyond term limit", 7, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			f := newGovFixture(t)
			f.params().CommitteeTermLimit = 5
			f.params().PoolVotingThresholds.CommitteeNoConfidence = cbor.Rat{Rat: new(big.Rat)}
			drep := f.drep(100)
			f.propose("update#0", GovActionInfo{
				ActionType: common.GovActionTypeUpdateCommittee,
				ProposedMembersByCredential: map[ledger.RewardAccountKey]uint64{
					keyCredential(common.Blake2b224{0x55}): test.expiry,
				},
				Votes: yes(drep),
			})
			require.NoError(t, f.sm.ProcessEpochBoundary(1))
			assert.Equal(t, test.ratified, f.ratified("update#0"))
		})
	}
}

func TestCommitteeTermLimitComparisonDoesNotOverflow(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.params().CommitteeTermLimit = 5
	proposal := &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType: common.GovActionTypeUpdateCommittee,
			ProposedMembersByCredential: map[ledger.RewardAccountKey]uint64{
				keyCredential(common.Blake2b224{0x55}): ^uint64(0),
			},
		},
	}

	assert.True(
		t,
		f.sm.withinCommitteeTermLimit(proposal, ^uint64(0)-2),
		"an expiry two epochs away is within a five-epoch term limit",
	)
}

func TestRatificationNeverRatifiesInfoAction(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	drep := f.drep(100)
	pool := f.pool(100)
	cc := f.ccMember(10, true)
	f.committeeThreshold(1, 2)
	f.propose("info#0", GovActionInfo{
		ActionType: common.GovActionTypeInfo,
		Votes:      yes(drep, pool, cc),
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.False(t, f.ratified("info#0"),
		"an info action has no threshold and stays until it expires")
}

func TestRatificationCommitteeWithoutMembersUsesItsThreshold(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	drep := f.drep(100)
	f.sm.govState.CommitteeThreshold = new(big.Rat)
	f.propose("constitution#0", GovActionInfo{
		ActionType: common.GovActionTypeNewConstitution,
		Votes:      yes(drep),
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.True(t, f.ratified("constitution#0"),
		"a zero committee threshold accepts with no eligible members when the minimum size is zero")
}

func TestRatificationUpdateCommitteeUsesNormalThresholdWhileCommitteeExists(t *testing.T) {
	t.Parallel()
	f := newGovFixture(t)
	f.params().DRepVotingThresholds.CommitteeNoConfidence = cbor.Rat{Rat: big.NewRat(1, 1)}
	f.params().PoolVotingThresholds.CommitteeNormal = cbor.Rat{Rat: new(big.Rat)}
	f.params().PoolVotingThresholds.CommitteeNoConfidence = cbor.Rat{Rat: new(big.Rat)}
	drep := f.drep(60)
	f.drep(40)
	// Every member expired before the ratifying epoch, yet the committee
	// itself is still in place.
	f.ccMember(0, true)
	f.committeeThreshold(1, 2)
	f.propose("update#0", GovActionInfo{
		ActionType: common.GovActionTypeUpdateCommittee,
		Votes:      yes(drep),
	})
	require.NoError(t, f.sm.ProcessEpochBoundary(1))
	assert.True(t, f.ratified("update#0"),
		"an elected committee selects the normal threshold even when no member is active")
}
