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
	coldCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldKey,
	}
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
			CredType:   common.CredentialTypeAddrKeyHash,
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
	require.Nil(t, firstMember.HotKey)
	require.True(t, firstMember.Resigned)
	require.NotContains(t, state.HotKeyAuthorizations, firstColdKey)
}

func TestAuthorizeHotCredentialPreservesCommitteeResignation(t *testing.T) {
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

	require.Contains(t, state.CommitteeResignations, coldKey)
	require.True(t, member.Resigned)
	require.Nil(t, member.HotCredential)
	require.NotContains(t, state.HotKeyAuthorizationsByCredential, coldKey)
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
		state := NewGovernanceState()
		state.CommitteeMembers[common.Blake2b224{0xff}] = &CommitteeMemberInfo{
			ExpiryEpoch: 42,
		}
		err := validator.validateCertificate(
			certificate(keyCredential),
			0,
			state,
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
	txWithCertificate := func(
		coldCredential common.Credential,
		resign bool,
	) *conway.ConwayTransaction {
		var certificate common.Certificate
		if resign {
			certificate = &common.ResignCommitteeColdCertificate{
				CertType:       uint(common.CertificateTypeResignCommitteeCold),
				ColdCredential: coldCredential,
			}
		} else {
			certificate = &common.AuthCommitteeHotCertificate{
				CertType:       uint(common.CertificateTypeAuthCommitteeHot),
				ColdCredential: coldCredential,
				HotCredential: common.Credential{
					CredType:   common.CredentialTypeAddrKeyHash,
					Credential: common.Blake2b224{0x04},
				},
			}
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
	ledgerState := stateManager.buildLedgerState()
	committeeRule := utxoValidateCommitteeCertificates
	for _, coldKey := range []ledger.RewardAccountKey{
		keyMember,
		scriptMember,
	} {
		member, err := ledgerState.CommitteeCredentialMember(
			coldKey.AsCredential(),
		)
		require.NoError(t, err)
		require.NotNil(t, member)
		assert.Equal(t, coldKey.Credential, member.ColdKey)
	}
	for _, resign := range []bool{false, true} {
		for _, coldKey := range []ledger.RewardAccountKey{
			keyMember,
			scriptMember,
		} {
			tx := txWithCertificate(coldKey.AsCredential(), resign)
			require.NoError(t, validator.ValidateTransaction(
				tx,
				0,
				0,
				stateManager.govState,
				nil,
			))
			require.NoError(t, committeeRule(
				tx,
				0,
				ledgerState,
				nil,
			))
		}
	}
	unknownCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: nonMemberHash,
	}
	require.ErrorContains(t, validator.ValidateTransaction(
		txWithCertificate(unknownCredential, false),
		0,
		0,
		stateManager.govState,
		nil,
	), "cannot authorize hot key for non-member")
	require.ErrorContains(t, committeeRule(
		txWithCertificate(unknownCredential, false),
		0,
		ledgerState,
		nil,
	), "not a CC member")
}

func TestCommitteeCertificateValidationUsesConfiguredLayers(t *testing.T) {
	for _, credentialType := range []uint{
		common.CredentialTypeAddrKeyHash,
		common.CredentialTypeScriptHash,
	} {
		t.Run(
			fmt.Sprintf("credential type %d", credentialType),
			func(t *testing.T) {
				coldCredential := common.Credential{
					CredType:   credentialType,
					Credential: common.Blake2b224{0x51},
				}
				coldKey := ledger.NewRewardAccountKey(coldCredential)
				stateManager := NewMockStateManager()
				stateManager.committeeMembers[coldKey] = 42
				stateManager.govState.CommitteeMembersByCredential[coldKey] =
					&CommitteeMemberInfo{
						ColdCredential: coldCredential,
						ColdKey:        coldCredential.Credential,
						ExpiryEpoch:    42,
					}
				certificate := &common.AuthCommitteeHotCertificate{
					CertType: uint(
						common.CertificateTypeAuthCommitteeHot,
					),
					ColdCredential: coldCredential,
					HotCredential: common.Credential{
						CredType:   common.CredentialTypeAddrKeyHash,
						Credential: common.Blake2b224{0x52},
					},
				}
				tx := &conway.ConwayTransaction{
					Body: conway.ConwayTransactionBody{
						TxCertificates: []common.CertificateWrapper{{
							Type:        certificate.Type(),
							Certificate: certificate,
						}},
					},
					TxIsValid: true,
				}

				require.NoError(t, NewValidator().ValidateTransaction(
					tx,
					0,
					0,
					stateManager.govState,
					nil,
				))
				require.NoError(t, conway.UtxoValidateCommitteeCertificates(
					tx,
					0,
					stateManager.buildLedgerState(),
					nil,
				))
			},
		)
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
				CredType:   common.CredentialTypeAddrKeyHash,
				Credential: common.Blake2b224{0x02},
			},
		}
	}
	validator := NewValidator()

	t.Run("true non-member", func(t *testing.T) {
		state := NewGovernanceState()
		state.CommitteeMembers[common.Blake2b224{0xff}] = &CommitteeMemberInfo{
			ExpiryEpoch: 42,
		}
		err := validator.validateCertificate(
			authorization(keyCredential),
			0,
			state,
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
				{name: "after expiry", epoch: expiresAfter + 1},
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
				state := NewGovernanceState()
				state.CommitteeMembersByCredential[ledger.RewardAccountKey{
					CredType:   common.CredentialTypeAddrKeyHash,
					Credential: common.Blake2b224{0xff},
				}] = &CommitteeMemberInfo{ExpiryEpoch: 20}
				tx := ledger.NewTransactionBuilder().WithCertificates(
					certificateCase.certificate,
				)
				err := validator.ValidateTransaction(
					tx,
					0,
					expiresAfter-1,
					state,
					nil,
				)
				require.ErrorContains(t, err, certificateCase.nonMemberError)
			})

			t.Run("empty incomplete state", func(t *testing.T) {
				tx := ledger.NewTransactionBuilder().WithCertificates(
					certificateCase.certificate,
				)
				require.NoError(t, validator.ValidateTransaction(
					tx,
					0,
					expiresAfter-1,
					NewGovernanceState(),
					nil,
				))
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

func TestCommitteeCertificateValidationIgnoresExpiredOnlyCurrentState(
	t *testing.T,
) {
	const currentEpoch = uint64(11)
	targetCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x01},
	}
	expiredCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x02},
	}
	expiredKey := ledger.NewRewardAccountKey(expiredCredential)
	stateManager := NewMockStateManager()
	stateManager.currentEpoch = currentEpoch
	stateManager.govState.CurrentEpoch = currentEpoch
	stateManager.committeeMembers[expiredKey] = currentEpoch - 1
	stateManager.govState.CommitteeMembersByCredential[expiredKey] =
		&CommitteeMemberInfo{
			ColdCredential: expiredCredential,
			ColdKey:        expiredCredential.Credential,
			ExpiryEpoch:    currentEpoch - 1,
		}
	certificate := &common.AuthCommitteeHotCertificate{
		CertType:       uint(common.CertificateTypeAuthCommitteeHot),
		ColdCredential: targetCredential,
		HotCredential: common.Credential{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: common.Blake2b224{0x03},
		},
	}
	tx := ledger.NewTransactionBuilder().WithCertificates(certificate)

	require.NoError(t, NewValidator().ValidateTransaction(
		tx,
		0,
		currentEpoch,
		stateManager.govState,
		nil,
	))
	require.Error(t, conway.UtxoValidateCommitteeCertificates(
		tx,
		0,
		stateManager.buildLedgerState(),
		nil,
	))
}

func TestValidateVotingPreservesLegacyCommitteeAuthorization(t *testing.T) {
	hotKey := common.Blake2b224{0x11}
	state := NewGovernanceState()
	state.HotKeyAuthorizations[common.Blake2b224{0x12}] = hotKey
	actionID := &common.GovActionId{TransactionId: [32]byte{0x13}}
	state.Proposals[formatGovActionIdFromPtr(actionID)] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType:   common.GovActionTypeNewConstitution,
			ExpiresAfter: 10,
		},
	}
	vote := func(voterType uint8) *ledger.MockTransaction {
		return ledger.NewTransactionBuilder().WithVotingProcedures(
			common.VotingProcedures{
				&common.Voter{Type: voterType, Hash: hotKey}: {
					actionID: {Vote: 1},
				},
			},
		)
	}

	require.NoError(t, NewValidator().ValidateTransaction(
		vote(common.VoterTypeConstitutionalCommitteeHotKeyHash),
		0,
		0,
		state,
		nil,
	))
	require.ErrorContains(t, NewValidator().ValidateTransaction(
		vote(common.VoterTypeConstitutionalCommitteeHotScriptHash),
		0,
		0,
		state,
		nil,
	), "not authorized")
}

func TestCommitteeHotCredentialMemberAllowsSharedAuthorization(t *testing.T) {
	hotCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x21},
	}
	firstCold := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x22},
	}
	secondCold := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: common.Blake2b224{0x23},
	}
	stateManager := NewMockStateManager()
	stateManager.currentEpoch = 3
	stateManager.govState.CurrentEpoch = 3
	stateManager.committeeMembers[firstCold] = 10
	stateManager.committeeMembers[secondCold] = 11
	stateManager.hotKeyAuthorizations[firstCold] = hotCredential
	stateManager.hotKeyAuthorizations[secondCold] = hotCredential

	ledgerState := stateManager.buildLedgerState()
	member, err := ledgerState.CommitteeHotCredentialMember(hotCredential)
	require.NoError(t, err)
	require.NotNil(t, member)
	assert.Equal(t, firstCold.Credential, member.ColdKey)

	tx := ledger.NewTransactionBuilder().WithVotingProcedures(
		common.VotingProcedures{
			&common.Voter{
				Type: common.VoterTypeConstitutionalCommitteeHotKeyHash,
				Hash: hotCredential.Credential,
			}: {},
		},
	)
	require.NoError(t, conway.UtxoValidateUnknownVoters(
		tx,
		0,
		ledgerState,
		nil,
	))
}

func TestDRepCredentialActivityUsesExactRegistration(t *testing.T) {
	sharedHash := common.Blake2b224{0x21}
	keyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: sharedHash,
	}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: sharedHash,
	}
	state := NewGovernanceState()
	state.DRepRegistrations[sharedHash] = true
	state.DRepRegistrationsByCredential[ledger.NewRewardAccountKey(
		scriptCredential,
	)] = true
	state.DRepExpiries[ledger.NewRewardAccountKey(scriptCredential)] = 10

	require.False(t, state.IsDRepCredentialActive(keyCredential, 5))
	require.True(t, state.IsDRepCredentialActive(scriptCredential, 5))

	state.DRepRegistrationsByCredential[ledger.NewRewardAccountKey(
		keyCredential,
	)] = true
	state.DRepExpiries[ledger.NewRewardAccountKey(keyCredential)] = 4
	require.False(t, state.IsDRepCredentialActive(keyCredential, 5))
	require.True(t, state.IsDRepCredentialActive(scriptCredential, 5))
}

func TestDRepVotingUsesCredentialTypeWithoutRejectingExpiredRegistration(
	t *testing.T,
) {
	sharedHash := common.Blake2b224{0x31}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: sharedHash,
	}
	state := NewGovernanceState()
	state.RegisterDRepCredentialUntil(scriptCredential, 4)
	actionID := &common.GovActionId{TransactionId: [32]byte{0x32}}
	state.Proposals[formatGovActionIdFromPtr(actionID)] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType:   common.GovActionTypeNewConstitution,
			ExpiresAfter: 10,
		},
	}
	vote := func(voterType uint8) *ledger.MockTransaction {
		return ledger.NewTransactionBuilder().WithVotingProcedures(
			common.VotingProcedures{
				&common.Voter{Type: voterType, Hash: sharedHash}: {
					actionID: {Vote: 1},
				},
			},
		)
	}

	require.ErrorContains(t, NewValidator().ValidateTransaction(
		vote(common.VoterTypeDRepKeyHash),
		0,
		5,
		state,
		nil,
	), "not registered")
	require.NoError(t, NewValidator().ValidateTransaction(
		vote(common.VoterTypeDRepScriptHash),
		0,
		5,
		state,
		nil,
	))
}

func TestDRepRegistrationValidationUsesExactCredential(t *testing.T) {
	sharedHash := common.Blake2b224{0x41}
	keyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: sharedHash,
	}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: sharedHash,
	}
	state := NewGovernanceState()
	state.RegisterDRepCredentialUntil(scriptCredential, 10)
	certificate := func(credential common.Credential) common.Certificate {
		return &common.RegistrationDrepCertificate{
			CertType:       uint(common.CertificateTypeRegistrationDrep),
			DrepCredential: credential,
		}
	}
	validator := NewValidator()

	require.NoError(t, validator.ValidateTransaction(
		ledger.NewTransactionBuilder().
			WithCertificates(certificate(keyCredential)),
		0,
		0,
		state,
		nil,
	))
	require.ErrorContains(t, validator.ValidateTransaction(
		ledger.NewTransactionBuilder().
			WithCertificates(certificate(scriptCredential)),
		0,
		0,
		state,
		nil,
	), "already registered")
}

func TestDRepDelegationValidationUsesExactCredential(t *testing.T) {
	sharedHash := common.Blake2b224{0x45}
	keyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: sharedHash,
	}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: sharedHash,
	}
	stakeCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x46},
	}
	drep := func(credential common.Credential) common.Drep {
		drepType := common.DrepTypeAddrKeyHash
		if credential.CredType == common.CredentialTypeScriptHash {
			drepType = common.DrepTypeScriptHash
		}
		return common.Drep{
			Type:       drepType,
			Credential: append([]byte(nil), credential.Credential[:]...),
		}
	}
	certificateBuilders := map[string]func(common.Drep) common.Certificate{
		"vote delegation": func(target common.Drep) common.Certificate {
			return &common.VoteDelegationCertificate{
				CertType:        uint(common.CertificateTypeVoteDelegation),
				StakeCredential: stakeCredential,
				Drep:            target,
			}
		},
		"stake vote delegation": func(target common.Drep) common.Certificate {
			return &common.StakeVoteDelegationCertificate{
				CertType: uint(
					common.CertificateTypeStakeVoteDelegation,
				),
				StakeCredential: stakeCredential,
				Drep:            target,
			}
		},
		"vote registration delegation": func(target common.Drep) common.Certificate {
			return &common.VoteRegistrationDelegationCertificate{
				CertType: uint(
					common.CertificateTypeVoteRegistrationDelegation,
				),
				StakeCredential: stakeCredential,
				Drep:            target,
			}
		},
		"stake vote registration delegation": func(target common.Drep) common.Certificate {
			return &common.StakeVoteRegistrationDelegationCertificate{
				CertType: uint(
					common.CertificateTypeStakeVoteRegistrationDelegation,
				),
				StakeCredential: stakeCredential,
				Drep:            target,
			}
		},
	}
	tests := []struct {
		name       string
		registered *common.Credential
		target     common.Credential
		legacy     bool
		valid      bool
	}{
		{
			name:       "key exact",
			registered: &keyCredential,
			target:     keyCredential,
			valid:      true,
		},
		{
			name:       "script exact",
			registered: &scriptCredential,
			target:     scriptCredential,
			valid:      true,
		},
		{
			name:       "key does not alias script",
			registered: &keyCredential,
			target:     scriptCredential,
		},
		{
			name:       "script does not alias key",
			registered: &scriptCredential,
			target:     keyCredential,
		},
		{name: "legacy key", target: keyCredential, legacy: true, valid: true},
		{
			name:   "legacy script",
			target: scriptCredential,
			legacy: true,
			valid:  true,
		},
	}

	for certificateName, buildCertificate := range certificateBuilders {
		for _, test := range tests {
			t.Run(certificateName+"/"+test.name, func(t *testing.T) {
				stateManager := NewMockStateManager()
				if test.registered != nil {
					stateManager.govState.RegisterDRepCredentialUntil(
						*test.registered,
						10,
					)
				}
				if test.legacy {
					stateManager.govState.DRepRegistrations[sharedHash] = true
				}
				tx := ledger.NewTransactionBuilder().WithCertificates(
					buildCertificate(drep(test.target)),
				)

				err := NewValidator().ValidateTransaction(
					tx,
					0,
					0,
					stateManager.govState,
					nil,
				)
				if !test.valid {
					require.ErrorContains(t, err, "DRep delegation target")
					require.Empty(
						t,
						stateManager.govState.DRepDelegationsByCredential,
					)
					return
				}
				require.NoError(t, err)
				require.NoError(t, stateManager.ApplyTransaction(tx, 0))
				assert.Equal(
					t,
					drep(test.target),
					stateManager.govState.DRepDelegationsByCredential[ledger.NewRewardAccountKey(stakeCredential)],
				)
			})
		}
	}
}

func TestDRepTransitionValidationUsesSequentialCredentialState(t *testing.T) {
	const (
		currentEpoch = uint64(10)
		activity     = uint64(4)
	)
	sharedHash := common.Blake2b224{0x49}
	keyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: sharedHash,
	}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: sharedHash,
	}
	registration := func(credential common.Credential) common.Certificate {
		return &common.RegistrationDrepCertificate{
			CertType:       uint(common.CertificateTypeRegistrationDrep),
			DrepCredential: credential,
		}
	}
	tests := []struct {
		name        string
		certificate func(common.Credential) common.Certificate
		assertState func(*testing.T, *MockStateManager)
	}{
		{
			name: "update",
			certificate: func(credential common.Credential) common.Certificate {
				return &common.UpdateDrepCertificate{
					CertType:       uint(common.CertificateTypeUpdateDrep),
					DrepCredential: credential,
				}
			},
			assertState: func(t *testing.T, stateManager *MockStateManager) {
				t.Helper()
				require.True(
					t,
					stateManager.govState.IsDRepCredentialRegistered(
						keyCredential,
					),
				)
				require.Equal(
					t,
					currentEpoch+activity,
					stateManager.govState.DRepExpiries[ledger.NewRewardAccountKey(
						keyCredential,
					)],
				)
			},
		},
		{
			name: "deregistration",
			certificate: func(credential common.Credential) common.Certificate {
				return &common.DeregistrationDrepCertificate{
					CertType: uint(
						common.CertificateTypeDeregistrationDrep,
					),
					DrepCredential: credential,
				}
			},
			assertState: func(t *testing.T, stateManager *MockStateManager) {
				t.Helper()
				require.False(
					t,
					stateManager.govState.IsDRepCredentialRegistered(
						keyCredential,
					),
				)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Run("missing credential", func(t *testing.T) {
				stateManager := NewMockStateManager()
				tx := ledger.NewTransactionBuilder().WithCertificates(
					test.certificate(keyCredential),
				)

				err := NewValidator().ValidateTransaction(
					tx,
					0,
					currentEpoch,
					stateManager.govState,
					stateManager.protocolParams,
				)
				require.ErrorContains(t, err, "not registered")
				require.Empty(t, stateManager.govState.DRepRegistrations)
			})

			t.Run(
				"other credential type registered earlier",
				func(t *testing.T) {
					stateManager := NewMockStateManager()
					tx := ledger.NewTransactionBuilder().WithCertificates(
						registration(scriptCredential),
						test.certificate(keyCredential),
					)

					err := NewValidator().ValidateTransaction(
						tx,
						0,
						currentEpoch,
						stateManager.govState,
						stateManager.protocolParams,
					)
					require.ErrorContains(t, err, "not registered")
					require.Empty(t, stateManager.govState.DRepRegistrations)
				},
			)

			t.Run("other credential type in current state", func(t *testing.T) {
				stateManager := NewMockStateManager()
				stateManager.govState.RegisterDRepCredentialUntil(
					scriptCredential,
					currentEpoch+activity,
				)
				tx := ledger.NewTransactionBuilder().WithCertificates(
					test.certificate(keyCredential),
				)

				err := NewValidator().ValidateTransaction(
					tx,
					0,
					currentEpoch,
					stateManager.govState,
					stateManager.protocolParams,
				)
				require.ErrorContains(t, err, "not registered")
				require.True(
					t,
					stateManager.govState.IsDRepCredentialRegistered(
						scriptCredential,
					),
				)
			})

			t.Run("exact credential registered earlier", func(t *testing.T) {
				stateManager := NewMockStateManager()
				stateManager.currentEpoch = currentEpoch
				stateManager.govState.CurrentEpoch = currentEpoch
				stateManager.protocolParams = &conway.ConwayProtocolParameters{
					DRepInactivityPeriod: activity,
				}
				tx := ledger.NewTransactionBuilder().WithCertificates(
					registration(keyCredential),
					test.certificate(keyCredential),
				)

				require.NoError(t, NewValidator().ValidateTransaction(
					tx,
					0,
					currentEpoch,
					stateManager.govState,
					stateManager.protocolParams,
				))
				require.NoError(t, stateManager.ApplyTransaction(tx, 0))
				test.assertState(t, stateManager)
			})

			t.Run("legacy hash-only registration", func(t *testing.T) {
				stateManager := NewMockStateManager()
				stateManager.currentEpoch = currentEpoch
				stateManager.govState.CurrentEpoch = currentEpoch
				stateManager.protocolParams = &conway.ConwayProtocolParameters{
					DRepInactivityPeriod: activity,
				}
				stateManager.drepRegistrations[sharedHash] = true
				stateManager.govState.DRepRegistrations[sharedHash] = true
				tx := ledger.NewTransactionBuilder().WithCertificates(
					test.certificate(keyCredential),
				)

				require.NoError(t, NewValidator().ValidateTransaction(
					tx,
					0,
					currentEpoch,
					stateManager.govState,
					stateManager.protocolParams,
				))
				require.NoError(t, stateManager.ApplyTransaction(tx, 0))
				test.assertState(t, stateManager)
			})
		})
	}
}

func TestApplyTransactionDeregistersExactDRepCredential(t *testing.T) {
	sharedHash := common.Blake2b224{0x51}
	keyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: sharedHash,
	}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: sharedHash,
	}
	stateManager := NewMockStateManager()
	stateManager.drepRegistrations[sharedHash] = true
	stateManager.govState.RegisterDRepCredentialUntil(keyCredential, 10)
	stateManager.govState.RegisterDRepCredentialUntil(scriptCredential, 20)
	tx := ledger.NewTransactionBuilder().WithCertificates(
		&common.DeregistrationDrepCertificate{
			CertType:       uint(common.CertificateTypeDeregistrationDrep),
			DrepCredential: scriptCredential,
		},
	)

	require.NoError(t, stateManager.ApplyTransaction(tx, 0))
	require.True(
		t,
		stateManager.govState.IsDRepCredentialActive(keyCredential, 10),
	)
	require.False(
		t,
		stateManager.govState.IsDRepCredentialActive(scriptCredential, 10),
	)
	require.True(t, stateManager.drepRegistrations[sharedHash])
}

func TestDRepVoteRefreshesExactCredentialForRatification(t *testing.T) {
	const (
		currentEpoch = uint64(5)
		activity     = uint64(3)
	)
	sharedHash := common.Blake2b224{0x61}
	keyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: sharedHash,
	}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: sharedHash,
	}
	stakeCredential := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x62},
	}
	actionID := &common.GovActionId{
		TransactionId: common.Blake2b256{0x63},
	}
	proposalID := formatGovActionIdFromPtr(actionID)
	stateManager := NewMockStateManager()
	stateManager.currentEpoch = currentEpoch
	stateManager.govState.CurrentEpoch = currentEpoch
	stateManager.protocolParams = &conway.ConwayProtocolParameters{
		DRepInactivityPeriod: activity,
		DRepVotingThresholds: conway.DRepVotingThresholds{
			CommitteeNoConfidence: cbor.Rat{Rat: big.NewRat(1, 1)},
		},
		PoolVotingThresholds: conway.PoolVotingThresholds{
			CommitteeNoConfidence: cbor.Rat{Rat: new(big.Rat)},
		},
	}
	stateManager.govState.RegisterDRepCredentialUntil(keyCredential, 4)
	stateManager.govState.RegisterDRepCredentialUntil(scriptCredential, 4)
	stateManager.rewardAccounts[stakeCredential] = 100
	stateManager.govState.DRepDelegationsByCredential[stakeCredential] = common.Drep{
		Type:       common.DrepTypeScriptHash,
		Credential: scriptCredential.Credential[:],
	}
	stateManager.govState.Proposals[proposalID] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType:     common.GovActionTypeUpdateCommittee,
			SubmittedEpoch: currentEpoch - 1,
			ExpiresAfter:   currentEpoch + 5,
		},
	}
	tx := ledger.NewTransactionBuilder().WithVotingProcedures(
		common.VotingProcedures{
			&common.Voter{
				Type: common.VoterTypeDRepScriptHash,
				Hash: sharedHash,
			}: {
				actionID: {Vote: common.GovVoteYes},
			},
		},
	)

	require.NoError(t, NewValidator().ValidateTransaction(
		tx,
		0,
		currentEpoch,
		stateManager.govState,
		stateManager.protocolParams,
	))
	require.NoError(t, stateManager.ApplyTransaction(tx, 0))
	require.Equal(
		t,
		currentEpoch+activity,
		stateManager.govState.DRepExpiries[ledger.NewRewardAccountKey(
			scriptCredential,
		)],
	)
	require.Equal(
		t,
		uint64(4),
		stateManager.govState.DRepExpiries[ledger.NewRewardAccountKey(
			keyCredential,
		)],
	)

	require.NoError(t, stateManager.ProcessEpochBoundary(currentEpoch+1))
	proposal := stateManager.govState.Proposals[proposalID]
	require.NotNil(t, proposal)
	require.NotNil(t, proposal.RatifiedEpoch)
	require.Equal(t, currentEpoch+1, *proposal.RatifiedEpoch)
}

func TestUpdateDRepCertificateReactivatesExactCredential(t *testing.T) {
	const (
		currentEpoch = uint64(10)
		activity     = uint64(4)
	)
	sharedHash := common.Blake2b224{0x71}
	keyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: sharedHash,
	}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: sharedHash,
	}
	stateManager := NewMockStateManager()
	stateManager.currentEpoch = currentEpoch
	stateManager.govState.CurrentEpoch = currentEpoch
	stateManager.protocolParams = &conway.ConwayProtocolParameters{
		DRepInactivityPeriod: activity,
	}
	stateManager.govState.RegisterDRepCredentialUntil(keyCredential, 9)
	stateManager.govState.RegisterDRepCredentialUntil(scriptCredential, 20)
	tx := ledger.NewTransactionBuilder().WithCertificates(
		&common.UpdateDrepCertificate{
			CertType:       uint(common.CertificateTypeUpdateDrep),
			DrepCredential: keyCredential,
		},
	)

	require.NoError(t, stateManager.ApplyTransaction(tx, 0))
	require.True(
		t,
		stateManager.govState.IsDRepCredentialActive(
			keyCredential,
			currentEpoch,
		),
	)
	require.Equal(
		t,
		currentEpoch+activity,
		stateManager.govState.DRepExpiries[ledger.NewRewardAccountKey(
			keyCredential,
		)],
	)
	require.Equal(
		t,
		uint64(20),
		stateManager.govState.DRepExpiries[ledger.NewRewardAccountKey(
			scriptCredential,
		)],
	)
}

func TestDRepVoteRefreshPreservesLegacyRegistration(t *testing.T) {
	const (
		currentEpoch = uint64(7)
		activity     = uint64(2)
	)
	credential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x81},
	}
	actionID := &common.GovActionId{TransactionId: common.Blake2b256{0x82}}
	stateManager := NewMockStateManager()
	stateManager.currentEpoch = currentEpoch
	stateManager.govState.CurrentEpoch = currentEpoch
	stateManager.protocolParams = &conway.ConwayProtocolParameters{
		DRepInactivityPeriod: activity,
	}
	stateManager.govState.DRepRegistrations[credential.Credential] = true
	stateManager.govState.Proposals[formatGovActionIdFromPtr(actionID)] =
		&ProposalState{GovActionInfo: GovActionInfo{ExpiresAfter: 20}}
	tx := ledger.NewTransactionBuilder().WithVotingProcedures(
		common.VotingProcedures{
			&common.Voter{
				Type: common.VoterTypeDRepKeyHash,
				Hash: credential.Credential,
			}: {
				actionID: {Vote: common.GovVoteYes},
			},
		},
	)

	require.NoError(t, NewValidator().ValidateTransaction(
		tx,
		0,
		currentEpoch,
		stateManager.govState,
		stateManager.protocolParams,
	))
	require.NoError(t, stateManager.ApplyTransaction(tx, 0))
	require.Equal(
		t,
		currentEpoch+activity,
		stateManager.govState.DRepExpiries[ledger.NewRewardAccountKey(
			credential,
		)],
	)
	require.Empty(t, stateManager.govState.DRepRegistrationsByCredential)
	require.True(
		t,
		stateManager.govState.IsDRepCredentialActive(
			credential,
			currentEpoch+activity,
		),
	)
}

func TestIsProposedCommitteeMemberCompatibility(t *testing.T) {
	hash := common.Blake2b224{0x42}
	key := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: hash,
	}
	script := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: hash,
	}
	tests := []struct {
		name     string
		members  map[ledger.RewardAccountKey]uint64
		epoch    uint64
		expected bool
	}{
		{
			name:     "key only",
			members:  map[ledger.RewardAccountKey]uint64{key: 10},
			expected: true,
		},
		{
			name:     "script only",
			members:  map[ledger.RewardAccountKey]uint64{script: 10},
			expected: true,
		},
		{
			name:     "both credential types",
			members:  map[ledger.RewardAccountKey]uint64{key: 10, script: 10},
			expected: true,
		},
		{name: "none", expected: false},
		{
			name:     "at expiry",
			members:  map[ledger.RewardAccountKey]uint64{key: 10},
			epoch:    10,
			expected: true,
		},
		{
			name:     "after expiry",
			members:  map[ledger.RewardAccountKey]uint64{key: 10},
			epoch:    11,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := NewGovernanceState()
			state.CurrentEpoch = test.epoch
			state.Proposals["proposal#0"] = &ProposalState{
				GovActionInfo: GovActionInfo{
					ActionType:                  common.GovActionTypeUpdateCommittee,
					ExpiresAfter:                10,
					ProposedMembersByCredential: test.members,
				},
			}
			require.Equal(
				t,
				test.expected,
				state.IsProposedCommitteeMember(hash),
			)
		})
	}
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
		CredType:   common.CredentialTypeAddrKeyHash,
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

func TestProcessAuthorizationPreservesAllResignationState(t *testing.T) {
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

	require.Contains(t, stateManager.committeeResignations, coldKey)
	require.Contains(
		t,
		stateManager.govState.CommitteeResignations,
		coldKey,
	)
	member, err := stateManager.buildLedgerState().
		CommitteeCredentialMember(coldCredential)
	require.NoError(t, err)
	require.NotNil(t, member)
	require.True(t, member.Resigned)
	require.Nil(t, member.HotKey)
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

func TestApplyTransactionDropsAmbiguousProposedMemberHash(t *testing.T) {
	sharedHash := common.Blake2b224{0x61}
	keyCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: sharedHash,
	}
	scriptCredential := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: sharedHash,
	}
	action := &common.UpdateCommitteeGovAction{
		Type: uint(common.GovActionTypeUpdateCommittee),
		CredEpochs: map[*common.Credential]uint{
			&keyCredential:    41,
			&scriptCredential: 42,
		},
	}
	tx := ledger.NewTransactionBuilder().WithProposalProcedures(
		conway.ConwayProposalProcedure{
			PPGovAction: conway.ConwayGovAction{
				Type:   uint(common.GovActionTypeUpdateCommittee),
				Action: action,
			},
		},
	)
	stateManager := NewMockStateManager()

	require.NoError(t, stateManager.ApplyTransaction(tx, 0))
	require.Len(t, stateManager.govState.Proposals, 1)
	for _, proposal := range stateManager.govState.Proposals {
		require.Len(t, proposal.ProposedMembersByCredential, 2)
		require.NotContains(t, proposal.ProposedMembers, sharedHash)
	}
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
	coldCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldKey,
	}
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
			CredType:   common.CredentialTypeAddrKeyHash,
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
	coldCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldKey,
	}
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
			CredType:   common.CredentialTypeAddrKeyHash,
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
	coldCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldKey,
	}
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
			CredType:   common.CredentialTypeAddrKeyHash,
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
	coldCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: coldKey,
	}
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

func TestProcessEpochBoundaryEnactsProposalIDsDeterministically(t *testing.T) {
	addedCredential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x44},
	}
	addedKey := ledger.NewRewardAccountKey(addedCredential)
	for range 64 {
		stateManager := NewMockStateManager()
		ratifiedEpoch := uint64(1)
		stateManager.govState.Proposals["a-update#0"] = &ProposalState{
			GovActionInfo: GovActionInfo{
				ActionType:   common.GovActionTypeUpdateCommittee,
				ExpiresAfter: 10,
				ProposedMembersByCredential: map[ledger.RewardAccountKey]uint64{
					addedKey: 20,
				},
			},
			RatifiedEpoch: &ratifiedEpoch,
		}
		stateManager.govState.Proposals["b-no-confidence#0"] = &ProposalState{
			GovActionInfo: GovActionInfo{
				ActionType:   common.GovActionTypeNoConfidence,
				ExpiresAfter: 10,
			},
			RatifiedEpoch: &ratifiedEpoch,
		}

		require.NoError(t, stateManager.ProcessEpochBoundary(2))
		require.Empty(t, stateManager.govState.CommitteeMembersByCredential)
		require.Equal(
			t,
			"b-no-confidence#0",
			*stateManager.govState.Roots.ConstitutionalCommittee,
		)
	}
}

func TestUpdateCommitteeCountsProposalDepositInDRepStake(t *testing.T) {
	stateManager := NewMockStateManager()
	stateManager.currentEpoch = 5
	stateManager.govState.CurrentEpoch = 5
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
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x01},
	}] = &CommitteeMemberInfo{ExpiryEpoch: 10}
	yesStake := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x11},
	}
	noStake := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x12},
	}
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
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: yesDRep,
	}] = true
	stateManager.govState.DRepRegistrationsByCredential[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: noDRep,
	}] = true
	proposal := &ProposalState{GovActionInfo: GovActionInfo{
		ActionType:    common.GovActionTypeUpdateCommittee,
		Deposit:       20,
		ReturnAccount: &yesStake,
		ExpiresAfter:  10,
		Votes: map[string]uint8{
			fmt.Sprintf(
				"%d:%s",
				common.VoterTypeDRepKeyHash,
				hex.EncodeToString(yesDRep[:]),
			): 1,
		},
	}}
	stateManager.govState.Proposals["update#0"] = proposal
	stateManager.govState.Proposals["expired#0"] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType:    common.GovActionTypeInfo,
			Deposit:       40,
			ReturnAccount: &noStake,
			ExpiresAfter:  4,
		},
	}

	stake := stateManager.credentialVotingStake(5)
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
	accepted, err := stateManager.updateCommitteeAcceptedWithStake(
		proposal,
		stake,
	)
	require.NoError(t, err)
	assert.True(t, accepted)
	proposal.Deposit = 0
	accepted, err = stateManager.updateCommitteeAcceptedWithStake(
		proposal,
		stateManager.credentialVotingStake(5),
	)
	require.NoError(t, err)
	assert.False(t, accepted)
}

func TestUpdateCommitteeRatificationRequiresThresholdConfiguration(
	t *testing.T,
) {
	tests := []struct {
		name      string
		params    common.ProtocolParameters
		errorText string
	}{
		{
			name:      "Conway parameters unavailable",
			errorText: "conway protocol parameters unavailable",
		},
		{
			name: "DRep threshold unavailable",
			params: &conway.ConwayProtocolParameters{
				PoolVotingThresholds: conway.PoolVotingThresholds{
					CommitteeNoConfidence: cbor.Rat{Rat: new(big.Rat)},
				},
			},
			errorText: "DRep voting threshold unavailable",
		},
		{
			name: "SPO threshold unavailable",
			params: &conway.ConwayProtocolParameters{
				DRepVotingThresholds: conway.DRepVotingThresholds{
					CommitteeNoConfidence: cbor.Rat{Rat: new(big.Rat)},
				},
			},
			errorText: "SPO voting threshold unavailable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stateManager := NewMockStateManager()
			stateManager.protocolParams = test.params
			stateManager.govState.Proposals["update#0"] = &ProposalState{
				GovActionInfo: GovActionInfo{
					ActionType:     common.GovActionTypeUpdateCommittee,
					SubmittedEpoch: 0,
					ExpiresAfter:   10,
				},
			}

			err := stateManager.ProcessEpochBoundary(1)
			require.ErrorContains(t, err, test.errorText)
			require.Nil(
				t,
				stateManager.govState.Proposals["update#0"].RatifiedEpoch,
			)
		})
	}
}

func TestProcessEpochBoundaryRejectsTypedNilProtocolParameters(t *testing.T) {
	stateManager := NewMockStateManager()
	stateManager.currentEpoch = 7
	stateManager.govState.CurrentEpoch = 7
	pool := common.PoolKeyHash{0x70}
	stateManager.poolRegistrations[pool] = true
	stateManager.govState.PoolRegistrations[pool] = true
	stateManager.govState.PoolRetirements[pool] = 8
	var typedNil *conway.ConwayProtocolParameters
	stateManager.protocolParams = typedNil

	var err error
	require.NotPanics(t, func() {
		err = stateManager.ProcessEpochBoundary(8)
	})
	require.ErrorContains(t, err, "conway protocol parameters unavailable")
	assert.Equal(t, uint64(7), stateManager.currentEpoch)
	assert.Equal(t, uint64(7), stateManager.govState.CurrentEpoch)
	assert.True(t, stateManager.poolRegistrations[pool])
	assert.True(t, stateManager.govState.PoolRegistrations[pool])
	assert.Equal(t, uint64(8), stateManager.govState.PoolRetirements[pool])
}

func TestRatificationErrorIsDeterministicAndTransactional(t *testing.T) {
	const runs = 64
	wantError := "ratify proposal update-a#0: DRep voting threshold unavailable"
	observedErrors := make(map[string]int)
	newStateManager := func() *MockStateManager {
		stateManager := NewMockStateManager()
		stateManager.currentEpoch = 7
		stateManager.govState.CurrentEpoch = 7
		stateManager.protocolParams = &conway.ConwayProtocolParameters{
			MinFeeA: 1,
			PoolVotingThresholds: conway.PoolVotingThresholds{
				CommitteeNoConfidence: cbor.Rat{Rat: new(big.Rat)},
			},
		}

		pool := common.Blake2b224{0x71}
		stateManager.poolRegistrations[pool] = true
		stateManager.govState.PoolRegistrations[pool] = true
		stateManager.govState.PoolRetirements[pool] = 8

		coldCredential := common.Credential{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: common.Blake2b224{0x72},
		}
		hotCredential := common.Credential{
			CredType:   common.CredentialTypeScriptHash,
			Credential: common.Blake2b224{0x73},
		}
		coldKey := ledger.NewRewardAccountKey(coldCredential)
		member := &CommitteeMemberInfo{
			ColdCredential: coldCredential,
			HotCredential:  &hotCredential,
			ColdKey:        coldCredential.Credential,
			HotKey:         &hotCredential.Credential,
			ExpiryEpoch:    20,
		}
		stateManager.committeeMembers[coldKey] = member.ExpiryEpoch
		stateManager.hotKeyAuthorizations[coldKey] = hotCredential
		stateManager.committeeResignations[coldKey] = true
		stateManager.govState.CommitteeMembers[coldCredential.Credential] = member
		stateManager.govState.CommitteeMembersByCredential[coldKey] = member
		stateManager.govState.HotKeyAuthorizations[coldCredential.Credential] =
			hotCredential.Credential
		stateManager.govState.HotKeyAuthorizationsByCredential[coldKey] =
			hotCredential
		stateManager.govState.CommitteeResignations[coldKey] = true

		protocolRoot := "old-parameters#0"
		hardForkRoot := "old-hard-fork#0"
		committeeRoot := "old-committee#0"
		constitutionRoot := "old-constitution#0"
		stateManager.govState.Roots = ProposalRoots{
			ProtocolParameters:      &protocolRoot,
			HardFork:                &hardForkRoot,
			ConstitutionalCommittee: &committeeRoot,
			Constitution:            &constitutionRoot,
		}
		stateManager.govState.Constitution = &ConstitutionInfo{
			AnchorURL:  "https://example.com/old",
			AnchorHash: []byte{0x74},
			PolicyHash: []byte{0x75},
		}
		stateManager.govState.EnactedProposals["old-enacted#0"] = true

		ratifiedEpoch := uint64(7)
		minFeeA := uint(99)
		stateManager.govState.Proposals["enact-committee#0"] = &ProposalState{
			GovActionInfo: GovActionInfo{
				ActionType:   common.GovActionTypeNoConfidence,
				ExpiresAfter: 20,
			},
			RatifiedEpoch: &ratifiedEpoch,
		}
		stateManager.govState.Proposals["enact-constitution#0"] = &ProposalState{
			GovActionInfo: GovActionInfo{
				ActionType:   common.GovActionTypeNewConstitution,
				ExpiresAfter: 20,
				PolicyHash:   []byte{0x76},
			},
			RatifiedEpoch: &ratifiedEpoch,
		}
		stateManager.govState.Proposals["enact-hard-fork#0"] = &ProposalState{
			GovActionInfo: GovActionInfo{
				ActionType:   common.GovActionTypeHardForkInitiation,
				ExpiresAfter: 20,
			},
			RatifiedEpoch: &ratifiedEpoch,
		}
		stateManager.govState.Proposals["enact-parameters#0"] = &ProposalState{
			GovActionInfo: GovActionInfo{
				ActionType:   common.GovActionTypeParameterChange,
				ExpiresAfter: 20,
				ParameterUpdate: &conway.ConwayProtocolParameterUpdate{
					MinFeeA: &minFeeA,
				},
			},
			RatifiedEpoch: &ratifiedEpoch,
		}

		for idx := 0; idx < 32; idx++ {
			id := fmt.Sprintf("info-%02d#0", idx)
			stateManager.govState.Proposals[id] = &ProposalState{
				GovActionInfo: GovActionInfo{
					ActionType:     common.GovActionTypeInfo,
					SubmittedEpoch: 0,
					ExpiresAfter:   10,
				},
			}
		}
		for _, id := range []string{"update-z#0", "update-a#0"} {
			stateManager.govState.Proposals[id] = &ProposalState{
				GovActionInfo: GovActionInfo{
					ActionType:     common.GovActionTypeUpdateCommittee,
					SubmittedEpoch: 0,
					ExpiresAfter:   10,
				},
			}
		}
		return stateManager
	}

	for run := 0; run < runs; run++ {
		stateManager := newStateManager()
		expected := newStateManager()
		for range 2 {
			err := stateManager.ProcessEpochBoundary(8)
			require.Error(t, err)
			require.Equal(t, wantError, err.Error())
			observedErrors[err.Error()]++
			require.True(
				t,
				assert.ObjectsAreEqual(expected, stateManager),
				"ratification error mutated observable boundary state",
			)
		}
	}

	require.Equal(t, map[string]int{wantError: runs * 2}, observedErrors)
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
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x01},
	}] = &CommitteeMemberInfo{}
	yesStake := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x11},
	}
	noStake := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x12},
	}
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

	stake := stateManager.credentialVotingStake(0)
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
	accepted, err := stateManager.updateCommitteeAcceptedWithStake(
		proposal,
		stake,
	)
	require.NoError(t, err)
	assert.False(t, accepted)
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
	accepted, err = stateManager.updateCommitteeAcceptedWithStake(
		proposal,
		stake,
	)
	require.NoError(t, err)
	assert.True(t, accepted)
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
