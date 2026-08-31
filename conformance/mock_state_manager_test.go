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
	"reflect"
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

func TestAuthorizeHotCredentialClearsCommitteeResignation(t *testing.T) {
	coldCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: common.Blake2b224{0x01},
	}
	hotCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: common.Blake2b224{0x02},
	}
	coldKey := ledger.NewRewardAccountKey(coldCredential)
	state := NewGovernanceState()
	member := &CommitteeMemberInfo{
		ColdCredential: coldCredential,
		ColdKey:        coldCredential.Credential,
		ExpiryEpoch:    42,
		Resigned:       true,
	}
	state.CommitteeMembersByCredential[coldKey] = member
	state.CommitteeResignations[coldKey] = true

	state.AuthorizeHotCredential(coldCredential, hotCredential)

	require.NotContains(t, state.CommitteeResignations, coldKey)
	require.False(t, member.Resigned)
	require.Equal(
		t,
		hotCredential,
		state.HotKeyAuthorizationsByCredential[coldKey],
	)
}

func TestSyncLegacyCommitteeMembersPreservesUnrelatedEntries(t *testing.T) {
	legacyHash := common.Blake2b224{0x01}
	typedHash := common.Blake2b224{0x02}
	ambiguousHash := common.Blake2b224{0x03}
	legacyMember := &CommitteeMemberInfo{ColdKey: legacyHash, ExpiryEpoch: 40}
	typedMember := &CommitteeMemberInfo{ColdKey: typedHash, ExpiryEpoch: 41}
	state := NewGovernanceState()
	state.CommitteeMembers[legacyHash] = legacyMember
	state.CommitteeMembers[ambiguousHash] = &CommitteeMemberInfo{
		ColdKey:     ambiguousHash,
		ExpiryEpoch: 39,
	}
	state.CommitteeMembersByCredential[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: typedHash,
	}] = typedMember
	state.CommitteeMembersByCredential[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: ambiguousHash,
	}] = &CommitteeMemberInfo{ColdKey: ambiguousHash, ExpiryEpoch: 42}
	state.CommitteeMembersByCredential[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: ambiguousHash,
	}] = &CommitteeMemberInfo{ColdKey: ambiguousHash, ExpiryEpoch: 43}

	state.syncLegacyCommitteeMembers()

	require.Same(t, legacyMember, state.CommitteeMembers[legacyHash])
	require.Same(t, typedMember, state.CommitteeMembers[typedHash])
	require.NotContains(t, state.CommitteeMembers, ambiguousHash)
}

func TestResignCommitteeCertificateMembershipCompatibility(t *testing.T) {
	coldHash := common.Blake2b224{0x01}
	keyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldHash,
	}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: coldHash,
	}
	certificate := func(
		credential common.Credential,
	) *common.ResignCommitteeColdCertificate {
		return &common.ResignCommitteeColdCertificate{
			CertType:       uint(common.CertificateTypeResignCommitteeCold),
			ColdCredential: credential,
		}
	}
	validator := NewValidator()

	t.Run("legacy current member", func(t *testing.T) {
		state := NewGovernanceState()
		state.CommitteeMembers[coldHash] = &CommitteeMemberInfo{
			ColdKey:     coldHash,
			ExpiryEpoch: 42,
		}
		require.NoError(t, validator.validateCertificate(
			certificate(keyCredential),
			state.CurrentEpoch,
			state,
			nil,
		))
	})

	t.Run("true non-member", func(t *testing.T) {
		err := validator.validateCertificate(
			certificate(keyCredential),
			0,
			NewGovernanceState(),
			nil,
		)
		require.ErrorContains(t, err, "cannot resign non-member")
	})

	t.Run("same hash script member is not key member", func(t *testing.T) {
		state := NewGovernanceState()
		state.CommitteeMembersByCredential[ledger.NewRewardAccountKey(
			scriptCredential,
		)] = &CommitteeMemberInfo{
			ColdCredential: scriptCredential,
			ColdKey:        coldHash,
			ExpiryEpoch:    42,
		}
		state.syncLegacyCommitteeMembers()
		err := validator.validateCertificate(
			certificate(keyCredential),
			state.CurrentEpoch,
			state,
			nil,
		)
		require.ErrorContains(t, err, "cannot resign non-member")
	})

	t.Run("same hash typed members remain exact", func(t *testing.T) {
		state := NewGovernanceState()
		state.CommitteeMembersByCredential[ledger.NewRewardAccountKey(
			keyCredential,
		)] = &CommitteeMemberInfo{
			ColdCredential: keyCredential,
			ColdKey:        coldHash,
			ExpiryEpoch:    42,
		}
		state.CommitteeMembersByCredential[ledger.NewRewardAccountKey(
			scriptCredential,
		)] = &CommitteeMemberInfo{
			ColdCredential: scriptCredential,
			ColdKey:        coldHash,
			ExpiryEpoch:    43,
		}
		state.syncLegacyCommitteeMembers()
		require.Nil(t, state.GetCommitteeMember(coldHash))
		require.NoError(t, validator.validateCertificate(
			certificate(keyCredential),
			state.CurrentEpoch,
			state,
			nil,
		))
		require.NoError(t, validator.validateCertificate(
			certificate(scriptCredential),
			state.CurrentEpoch,
			state,
			nil,
		))
	})
}

func TestCommitteeCertificateValidationUsesSequentialState(t *testing.T) {
	coldCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: common.Blake2b224{0x01},
	}
	hotCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: common.Blake2b224{0x02},
	}
	state := NewGovernanceState()
	state.CommitteeMembersByCredential[ledger.NewRewardAccountKey(
		coldCredential,
	)] = &CommitteeMemberInfo{
		ColdCredential: coldCredential,
		ColdKey:        coldCredential.Credential,
		ExpiryEpoch:    42,
	}
	resignation := &common.ResignCommitteeColdCertificate{
		CertType:       uint(common.CertificateTypeResignCommitteeCold),
		ColdCredential: coldCredential,
	}
	authorization := &common.AuthCommitteeHotCertificate{
		CertType:       uint(common.CertificateTypeAuthCommitteeHot),
		ColdCredential: coldCredential,
		HotCredential:  hotCredential,
	}
	txWithCertificates := func(
		certificates ...common.Certificate,
	) *conway.ConwayTransaction {
		wrappers := make([]common.CertificateWrapper, len(certificates))
		for idx, certificate := range certificates {
			wrappers[idx] = common.CertificateWrapper{
				Type:        certificate.Type(),
				Certificate: certificate,
			}
		}
		return &conway.ConwayTransaction{
			Body: conway.ConwayTransactionBody{
				TxCertificates: wrappers,
			},
			TxIsValid: true,
		}
	}
	validator := NewValidator()

	require.ErrorContains(t, validator.validateCertificates(
		txWithCertificates(resignation, authorization),
		state.CurrentEpoch,
		state,
	), "cannot authorize hot key for resigned CC member")
	require.NoError(t, validator.validateCertificates(
		txWithCertificates(authorization, resignation),
		state.CurrentEpoch,
		state,
	))
	require.NoError(t, validator.validateCertificates(
		txWithCertificates(authorization, authorization),
		state.CurrentEpoch,
		state,
	))
	require.ErrorContains(t, validator.validateCertificates(
		txWithCertificates(resignation, resignation),
		state.CurrentEpoch,
		state,
	), "cannot resign already resigned CC member")
}

func TestCommitteeCertificateValidationUsesExactCredentialAcrossPhases(
	t *testing.T,
) {
	ambiguousHash := common.Blake2b224{0x01}
	unambiguousHash := common.Blake2b224{0x02}
	nonMemberHash := common.Blake2b224{0x03}
	keyMember := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: ambiguousHash,
	}
	scriptMember := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: ambiguousHash,
	}
	unambiguousMember := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: unambiguousHash,
	}
	stateManager := NewMockStateManager()
	for _, coldKey := range []ledger.RewardAccountKey{
		keyMember,
		scriptMember,
		unambiguousMember,
	} {
		stateManager.committeeMembers[coldKey] = 42
		stateManager.govState.CommitteeMembersByCredential[coldKey] =
			&CommitteeMemberInfo{
				ColdCredential: coldKey.AsCredential(),
				ColdKey:        coldKey.Credential,
				ExpiryEpoch:    42,
			}
	}
	stateManager.govState.syncLegacyCommitteeMembers()
	txWithAuthorization := func(
		coldCredential common.Credential,
	) *conway.ConwayTransaction {
		certificate := &common.AuthCommitteeHotCertificate{
			CertType:       uint(common.CertificateTypeAuthCommitteeHot),
			ColdCredential: coldCredential,
			HotCredential: common.Credential{
				Credential: common.Blake2b224{0x04},
			},
		}
		return &conway.ConwayTransaction{
			Body: conway.ConwayTransactionBody{
				TxCertificates: []common.CertificateWrapper{{
					Type:        certificate.Type(),
					Certificate: certificate,
				}},
			},
			TxIsValid: true,
		}
	}
	validator := NewValidator()
	for _, coldKey := range []ledger.RewardAccountKey{
		keyMember,
		scriptMember,
	} {
		tx := txWithAuthorization(coldKey.AsCredential())
		require.NoError(t, validator.ValidateTransaction(
			tx,
			0,
			0,
			stateManager.govState,
			nil,
		))
		require.ErrorContains(t, conway.UtxoValidateCommitteeCertificates(
			tx,
			0,
			stateManager.buildLedgerState(),
			nil,
		), "not a CC member")
	}
	require.ErrorContains(t, validator.ValidateTransaction(
		txWithAuthorization(common.Credential{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: nonMemberHash,
		}),
		0,
		0,
		stateManager.govState,
		nil,
	), "cannot authorize hot key for non-member")

	hashOnlyRule := reflect.ValueOf(
		conway.UtxoValidateCommitteeCertificates,
	).Pointer()
	for _, rule := range ConformanceValidationRules {
		if reflect.ValueOf(rule).Pointer() == hashOnlyRule {
			t.Fatal(
				"hash-only committee certificate rule must not run after exact credential validation",
			)
		}
	}
}

func TestCommitteeAuthorizationMembershipValidation(t *testing.T) {
	coldHash := common.Blake2b224{0x01}
	keyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldHash,
	}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: coldHash,
	}
	authorization := func(
		credential common.Credential,
	) *common.AuthCommitteeHotCertificate {
		return &common.AuthCommitteeHotCertificate{
			CertType:       uint(common.CertificateTypeAuthCommitteeHot),
			ColdCredential: credential,
			HotCredential: common.Credential{
				Credential: common.Blake2b224{0x02},
			},
		}
	}
	validator := NewValidator()

	t.Run("true non-member", func(t *testing.T) {
		err := validator.validateCertificate(
			authorization(keyCredential),
			0,
			NewGovernanceState(),
			nil,
		)
		require.ErrorContains(t, err, "cannot authorize hot key for non-member")
	})

	t.Run("same hash script member is not key member", func(t *testing.T) {
		state := NewGovernanceState()
		state.CommitteeMembersByCredential[ledger.NewRewardAccountKey(
			scriptCredential,
		)] = &CommitteeMemberInfo{
			ColdCredential: scriptCredential,
			ColdKey:        coldHash,
			ExpiryEpoch:    42,
		}
		err := validator.validateCertificate(
			authorization(keyCredential),
			state.CurrentEpoch,
			state,
			nil,
		)
		require.ErrorContains(t, err, "cannot authorize hot key for non-member")
		require.NoError(t, validator.validateCertificate(
			authorization(scriptCredential),
			state.CurrentEpoch,
			state,
			nil,
		))
	})

	t.Run("exact proposed member", func(t *testing.T) {
		state := NewGovernanceState()
		state.Proposals["proposal#0"] = &ProposalState{
			GovActionInfo: GovActionInfo{
				ActionType: common.GovActionTypeUpdateCommittee,
				ProposedMembersByCredential: map[ledger.RewardAccountKey]uint64{
					ledger.NewRewardAccountKey(keyCredential): 42,
				},
			},
		}
		require.NoError(t, validator.validateCertificate(
			authorization(keyCredential),
			state.CurrentEpoch,
			state,
			nil,
		))
		err := validator.validateCertificate(
			authorization(scriptCredential),
			state.CurrentEpoch,
			state,
			nil,
		)
		require.ErrorContains(t, err, "cannot authorize hot key for non-member")
	})
}

func TestCommitteeCertificateValidationHonorsProposalExpiry(t *testing.T) {
	const expiresAfter = uint64(10)
	coldCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x01},
	}
	hotCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x02},
	}
	committeeKey := ledger.NewRewardAccountKey(coldCredential)
	certificateCases := []struct {
		name           string
		certificate    common.Certificate
		nonMemberError string
	}{
		{
			name: "authorization",
			certificate: &common.AuthCommitteeHotCertificate{
				CertType:       uint(common.CertificateTypeAuthCommitteeHot),
				ColdCredential: coldCredential,
				HotCredential:  hotCredential,
			},
			nonMemberError: "cannot authorize hot key for non-member",
		},
		{
			name: "resignation",
			certificate: &common.ResignCommitteeColdCertificate{
				CertType:       uint(common.CertificateTypeResignCommitteeCold),
				ColdCredential: coldCredential,
			},
			nonMemberError: "cannot resign non-member",
		},
	}
	validator := NewValidator()

	newProposedMemberState := func() *GovernanceState {
		state := NewGovernanceState()
		state.AddProposal("proposal#0", GovActionInfo{
			ActionType:   common.GovActionTypeUpdateCommittee,
			ExpiresAfter: expiresAfter,
			ProposedMembersByCredential: map[ledger.RewardAccountKey]uint64{
				committeeKey: 20,
			},
		})
		return state
	}

	for _, certificateCase := range certificateCases {
		t.Run(certificateCase.name, func(t *testing.T) {
			for _, epochCase := range []struct {
				name        string
				epoch       uint64
				shouldError bool
			}{
				{name: "before expiry", epoch: expiresAfter - 1},
				{name: "at expiry", epoch: expiresAfter},
				{name: "after expiry", epoch: expiresAfter + 1, shouldError: true},
			} {
				t.Run(epochCase.name, func(t *testing.T) {
					state := newProposedMemberState()
					// Keep the stored epoch stale to prove transaction validation
					// uses the epoch supplied by the harness.
					state.CurrentEpoch = 0
					tx := ledger.NewTransactionBuilder().WithCertificates(
						certificateCase.certificate,
					)
					err := validator.ValidateTransaction(
						tx,
						0,
						epochCase.epoch,
						state,
						nil,
					)
					if epochCase.shouldError {
						require.ErrorContains(
							t,
							err,
							certificateCase.nonMemberError,
						)
						return
					}
					require.NoError(t, err)
				})
			}

			t.Run("current member after proposal expiry", func(t *testing.T) {
				state := newProposedMemberState()
				state.CommitteeMembersByCredential[committeeKey] = &CommitteeMemberInfo{
					ColdCredential: coldCredential,
					ColdKey:        coldCredential.Credential,
					ExpiryEpoch:    20,
				}
				tx := ledger.NewTransactionBuilder().WithCertificates(
					certificateCase.certificate,
				)
				require.NoError(t, validator.ValidateTransaction(
					tx,
					0,
					expiresAfter+1,
					state,
					nil,
				))
			})

			t.Run("non-member", func(t *testing.T) {
				tx := ledger.NewTransactionBuilder().WithCertificates(
					certificateCase.certificate,
				)
				err := validator.ValidateTransaction(
					tx,
					0,
					expiresAfter-1,
					NewGovernanceState(),
					nil,
				)
				require.ErrorContains(t, err, certificateCase.nonMemberError)
			})
		})
	}

	state := newProposedMemberState()
	state.CurrentEpoch = expiresAfter
	require.NotNil(t, state.GetCommitteeCredentialMember(coldCredential))
	require.True(t, state.IsProposedCommitteeCredentialMember(coldCredential))
	state.CurrentEpoch++
	require.Nil(t, state.GetCommitteeCredentialMember(coldCredential))
	require.False(t, state.IsProposedCommitteeCredentialMember(coldCredential))
}

func TestMixedLegacyAndTypedCommitteeState(t *testing.T) {
	legacyHash := common.Blake2b224{0x01}
	typedHash := common.Blake2b224{0x02}
	legacyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: legacyHash,
	}
	typedCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: typedHash,
	}
	legacyMember := &CommitteeMemberInfo{
		ColdKey:     legacyHash,
		ExpiryEpoch: 41,
	}
	state := NewGovernanceState()
	state.CommitteeMembers[legacyHash] = legacyMember
	state.CommitteeMembersByCredential[ledger.NewRewardAccountKey(
		typedCredential,
	)] = &CommitteeMemberInfo{
		ColdCredential: typedCredential,
		ColdKey:        typedHash,
		ExpiryEpoch:    42,
	}

	require.Same(
		t,
		legacyMember,
		state.GetCommitteeCredentialMember(legacyCredential),
	)
	require.NoError(t, NewValidator().validateCertificate(
		&common.ResignCommitteeColdCertificate{
			CertType:       uint(common.CertificateTypeResignCommitteeCold),
			ColdCredential: legacyCredential,
		},
		state.CurrentEpoch,
		state,
		nil,
	))
}

func TestAuthorizeHotCredentialPreservesLegacyHashCollision(t *testing.T) {
	coldHash := common.Blake2b224{0x01}
	legacyHotHash := common.Blake2b224{0x02}
	scriptHotHash := common.Blake2b224{0x03}
	keyCold := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldHash,
	}
	scriptCold := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: coldHash,
	}
	state := NewGovernanceState()
	state.HotKeyAuthorizations[coldHash] = legacyHotHash

	state.AuthorizeHotCredential(scriptCold.AsCredential(), common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: scriptHotHash,
	})

	require.Equal(t, common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: legacyHotHash,
	}, state.HotKeyAuthorizationsByCredential[keyCold])
	require.Equal(t, common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: scriptHotHash,
	}, state.HotKeyAuthorizationsByCredential[scriptCold])
	require.NotContains(t, state.HotKeyAuthorizations, coldHash)
}

func TestLoadInitialStateMergesLegacyAndTypedCommitteeState(t *testing.T) {
	legacyHash := common.Blake2b224{0x01}
	legacyHotHash := common.Blake2b224{0x02}
	typedHash := common.Blake2b224{0x03}
	typedHotHash := common.Blake2b224{0x04}
	legacyKey := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: legacyHash,
	}
	typedScript := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: typedHash,
	}
	initialState := &ParsedInitialState{
		CommitteeMembers: map[common.Blake2b224]uint64{
			legacyHash: 41,
		},
		CommitteeMembersByCredential: map[ledger.RewardAccountKey]uint64{
			typedScript: 42,
		},
		HotKeyAuthorizations: map[common.Blake2b224]common.Blake2b224{
			legacyHash: legacyHotHash,
		},
		HotKeyAuthorizationsByCredential: map[ledger.RewardAccountKey]common.Credential{
			typedScript: {
				CredType:   common.CredentialTypeScriptHash,
				Credential: typedHotHash,
			},
		},
	}
	stateManager := NewMockStateManager()
	require.NoError(t, stateManager.LoadInitialState(initialState, nil))

	require.Equal(t, uint64(41), stateManager.committeeMembers[legacyKey])
	require.Equal(t, uint64(42), stateManager.committeeMembers[typedScript])
	require.Equal(t, common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: legacyHotHash,
	}, stateManager.hotKeyAuthorizations[legacyKey])
	require.Equal(t, common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: typedHotHash,
	}, stateManager.hotKeyAuthorizations[typedScript])
	require.NotNil(
		t,
		stateManager.govState.CommitteeMembersByCredential[legacyKey],
	)
	require.NotNil(
		t,
		stateManager.govState.CommitteeMembersByCredential[typedScript],
	)
	require.Contains(
		t,
		stateManager.govState.HotKeyAuthorizationsByCredential,
		legacyKey,
	)
	require.Contains(
		t,
		stateManager.govState.HotKeyAuthorizationsByCredential,
		typedScript,
	)
}

func TestLoadInitialStateDoesNotPromoteTypedProjection(t *testing.T) {
	coldHash := common.Blake2b224{0x01}
	hotHash := common.Blake2b224{0x02}
	keyCredential := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldHash,
	}
	scriptCredential := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: coldHash,
	}
	initialState := &ParsedInitialState{
		CommitteeMembers: map[common.Blake2b224]uint64{
			coldHash: 42,
		},
		CommitteeMembersByCredential: map[ledger.RewardAccountKey]uint64{
			scriptCredential: 42,
		},
		HotKeyAuthorizations: map[common.Blake2b224]common.Blake2b224{
			coldHash: hotHash,
		},
		HotKeyAuthorizationsByCredential: map[ledger.RewardAccountKey]common.Credential{
			scriptCredential: {
				CredType:   common.CredentialTypeScriptHash,
				Credential: hotHash,
			},
		},
	}
	stateManager := NewMockStateManager()
	require.NoError(t, stateManager.LoadInitialState(initialState, nil))

	require.NotContains(t, stateManager.committeeMembers, keyCredential)
	require.Contains(t, stateManager.committeeMembers, scriptCredential)
	require.NotContains(t, stateManager.hotKeyAuthorizations, keyCredential)
	require.Contains(t, stateManager.hotKeyAuthorizations, scriptCredential)
	require.NotContains(
		t,
		stateManager.govState.CommitteeMembersByCredential,
		keyCredential,
	)
	require.Contains(
		t,
		stateManager.govState.CommitteeMembersByCredential,
		scriptCredential,
	)
}

func TestBuildLedgerStatePreservesLegacyCommitteeMembers(t *testing.T) {
	firstHash := common.Blake2b224{0x01}
	secondHash := common.Blake2b224{0x02}
	firstHotHash := common.Blake2b224{0x03}
	firstKey := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: firstHash,
	}
	secondKey := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: secondHash,
	}
	secondScript := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: secondHash,
	}
	stateManager := NewMockStateManager()
	stateManager.committeeMembers[firstKey] = 41
	stateManager.committeeMembers[secondKey] = 42
	stateManager.committeeMembers[secondScript] = 43
	stateManager.hotKeyAuthorizations[firstKey] = common.Credential{
		Credential: firstHotHash,
	}

	members, err := stateManager.buildLedgerState().CommitteeMembers()
	require.NoError(t, err)
	require.Equal(t, []common.CommitteeMember{{
		ColdKey:     firstHash,
		HotKey:      &firstHotHash,
		ExpiryEpoch: 41,
	}}, members)
}

func TestProcessAuthorizationClearsAllResignationState(t *testing.T) {
	coldCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: common.Blake2b224{0x01},
	}
	hotCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: common.Blake2b224{0x02},
	}
	coldKey := ledger.NewRewardAccountKey(coldCredential)
	stateManager := NewMockStateManager()
	stateManager.committeeMembers[coldKey] = 42
	stateManager.govState.CommitteeMembersByCredential[coldKey] = &CommitteeMemberInfo{
		ColdCredential: coldCredential,
		ColdKey:        coldCredential.Credential,
		ExpiryEpoch:    42,
	}
	stateManager.processCertificate(&common.ResignCommitteeColdCertificate{
		CertType:       uint(common.CertificateTypeResignCommitteeCold),
		ColdCredential: coldCredential,
	})
	stateManager.processCertificate(&common.AuthCommitteeHotCertificate{
		CertType:       uint(common.CertificateTypeAuthCommitteeHot),
		ColdCredential: coldCredential,
		HotCredential:  hotCredential,
	})

	require.NotContains(t, stateManager.committeeResignations, coldKey)
	require.NotContains(
		t,
		stateManager.govState.CommitteeResignations,
		coldKey,
	)
	member, err := stateManager.buildLedgerState().
		CommitteeCredentialMember(coldCredential)
	require.NoError(t, err)
	require.NotNil(t, member)
	require.False(t, member.Resigned)
	require.NotNil(t, member.HotKey)
	require.Equal(t, hotCredential.Credential, *member.HotKey)
}

func TestUpdateCommitteeConflictUsesCredentialIdentity(t *testing.T) {
	coldHash := common.Blake2b224{0x01}
	keyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldHash,
	}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: coldHash,
	}
	state := NewGovernanceState()
	validator := NewValidator()
	update := func(
		removed common.Credential,
		added common.Credential,
	) *common.UpdateCommitteeGovAction {
		return &common.UpdateCommitteeGovAction{
			Credentials: []common.Credential{removed},
			CredEpochs:  map[*common.Credential]uint{&added: 1},
		}
	}

	require.NoError(t, validator.validateUpdateCommittee(
		update(keyCredential, scriptCredential),
		state,
	))
	require.NoError(t, validator.validateUpdateCommittee(
		update(scriptCredential, keyCredential),
		state,
	))
	require.ErrorContains(t, validator.validateUpdateCommittee(
		update(keyCredential, keyCredential),
		state,
	), "conflicting committee update")
	require.ErrorContains(t, validator.validateUpdateCommittee(
		update(scriptCredential, scriptCredential),
		state,
	), "conflicting committee update")
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
		stateManager.govState.CurrentEpoch,
		stateManager.govState,
		nil,
	), "cannot authorize hot key for resigned CC member")
	require.ErrorContains(t, NewValidator().validateCertificate(
		&common.ResignCommitteeColdCertificate{
			CertType:       uint(common.CertificateTypeResignCommitteeCold),
			ColdCredential: coldCredential,
		},
		stateManager.govState.CurrentEpoch,
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
		stateManager.govState.CurrentEpoch,
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
		stateManager.govState.CurrentEpoch,
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

func TestUpdateCommitteeEnactmentMergesCredentialViews(t *testing.T) {
	legacyHash := common.Blake2b224{0x01}
	typedHash := common.Blake2b224{0x02}
	ambiguousHash := common.Blake2b224{0x03}
	replacementHash := common.Blake2b224{0x04}
	legacyKey := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: legacyHash,
	}
	typedScript := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: typedHash,
	}
	typedKeyProjection := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: typedHash,
	}
	ambiguousKey := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: ambiguousHash,
	}
	ambiguousScript := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: ambiguousHash,
	}
	replacementKey := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: replacementHash,
	}
	replacementScript := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: replacementHash,
	}
	stateManager := NewMockStateManager()
	stateManager.committeeMembers[replacementScript] = 40
	stateManager.govState.CommitteeMembersByCredential[replacementScript] =
		&CommitteeMemberInfo{
			ColdCredential: replacementScript.AsCredential(),
			ColdKey:        replacementHash,
			ExpiryEpoch:    40,
		}
	stateManager.govState.syncLegacyCommitteeMembers()
	ratifiedEpoch := uint64(1)
	stateManager.govState.Proposals["update#0"] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType: common.GovActionTypeUpdateCommittee,
			RemovedMembers: map[ledger.RewardAccountKey]bool{
				replacementScript: true,
			},
			ProposedMembers: map[common.Blake2b224]uint64{
				legacyHash:      41,
				typedHash:       42,
				ambiguousHash:   43,
				replacementHash: 45,
			},
			ProposedMembersByCredential: map[ledger.RewardAccountKey]uint64{
				typedScript:     42,
				ambiguousKey:    43,
				ambiguousScript: 44,
				replacementKey:  45,
			},
			ExpiresAfter: 10,
		},
		RatifiedEpoch: &ratifiedEpoch,
	}

	require.NoError(t, stateManager.ProcessEpochBoundary(2))
	require.NotContains(t, stateManager.govState.Proposals, "update#0")
	require.Equal(t, uint64(41), stateManager.committeeMembers[legacyKey])
	require.Equal(t, uint64(42), stateManager.committeeMembers[typedScript])
	require.NotContains(t, stateManager.committeeMembers, typedKeyProjection)
	require.Equal(t, uint64(43), stateManager.committeeMembers[ambiguousKey])
	require.Equal(t, uint64(44), stateManager.committeeMembers[ambiguousScript])
	require.NotContains(t, stateManager.committeeMembers, replacementScript)
	require.Equal(t, uint64(45), stateManager.committeeMembers[replacementKey])
	require.NotNil(
		t,
		stateManager.govState.GetCommitteeCredentialMember(
			legacyKey.AsCredential(),
		),
	)
	require.Nil(
		t,
		stateManager.govState.GetCommitteeCredentialMember(
			typedKeyProjection.AsCredential(),
		),
	)
	require.NotNil(
		t,
		stateManager.govState.GetCommitteeCredentialMember(
			typedScript.AsCredential(),
		),
	)
	require.Nil(t, stateManager.govState.GetCommitteeMember(ambiguousHash))
	require.NotNil(
		t,
		stateManager.govState.GetCommitteeCredentialMember(
			ambiguousKey.AsCredential(),
		),
	)
	require.NotNil(
		t,
		stateManager.govState.GetCommitteeCredentialMember(
			ambiguousScript.AsCredential(),
		),
	)
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
