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
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func parseSyntheticInitialState(
	t *testing.T,
	votingStateHex string,
	poolStateHex string,
	delegationStateHex string,
	proposalsHex string,
) *ParsedInitialState {
	t.Helper()
	if votingStateHex == "" {
		votingStateHex = "82a0a0"
	}
	if poolStateHex == "" {
		poolStateHex = "81a0"
	}
	if delegationStateHex == "" {
		delegationStateHex = "81a0"
	}
	if proposalsHex == "" {
		proposalsHex = "85a0f6f6f6f6"
	}
	// InitialState[3].BeginEpochState[1].LedgerState contains CertState and
	// UTxOState. The latter carries GovState at index 3. Empty surrounding
	// fields are valid sentinels and keep every target fragment on its real
	// exported ParseInitialState path.
	rawHex := "8400f6f682f68283" +
		votingStateHex + poolStateHex + delegationStateHex +
		"84a0a0f683" + proposalsHex + "8182a0f681f6"
	raw, err := hex.DecodeString(rawHex)
	require.NoError(t, err)
	state, err := ParseInitialState(cbor.RawMessage(raw))
	require.NoError(t, err)
	return state
}

func filledBlake2b224(fill byte) common.Blake2b224 {
	var hash common.Blake2b224
	for i := range hash {
		hash[i] = fill
	}
	return hash
}

func TestParseVotingStateCardanoWireShapes(t *testing.T) {
	// cardano-ledger encodes VState as [dreps, committee-authorizations].
	// DRepState stores expiry at index 0. CommitteeAuthorization is the sum
	// [0, hot-credential] | [1, anchor]. Distinct adjacent values ensure the
	// parser cannot satisfy these assertions from a neighboring offset.
	raw := "82" +
		"a2" +
		"8200581c" + strings.Repeat("11", common.Blake2b224Size) +
		"82184d1863" +
		"8201581c" + strings.Repeat("22", common.Blake2b224Size) +
		"82184e1862" +
		"a3" +
		"8200581c" + strings.Repeat("33", common.Blake2b224Size) +
		"82008201581c" + strings.Repeat("44", common.Blake2b224Size) +
		"8201581c" + strings.Repeat("55", common.Blake2b224Size) +
		"82008200581c" + strings.Repeat("66", common.Blake2b224Size) +
		"8200581c" + strings.Repeat("77", common.Blake2b224Size) +
		"8201f6"
	state := parseSyntheticInitialState(t, raw, "", "", "")
	drepKey := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: filledBlake2b224(0x11),
	}
	drepScript := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: filledBlake2b224(0x22),
	}
	require.True(t, state.DRepRegistrationsByCredential[drepKey])
	require.True(t, state.DRepRegistrationsByCredential[drepScript])
	require.Equal(t, uint64(77), state.DRepExpiries[drepKey])
	require.Equal(t, uint64(78), state.DRepExpiries[drepScript])

	keyCold := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: filledBlake2b224(0x33),
	}
	scriptCold := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: filledBlake2b224(0x55),
	}
	resignedCold := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: filledBlake2b224(0x77),
	}
	require.Equal(t, common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: filledBlake2b224(0x44),
	}, state.HotKeyAuthorizationsByCredential[keyCold])
	require.Equal(t, common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: filledBlake2b224(0x66),
	}, state.HotKeyAuthorizationsByCredential[scriptCold])
	require.True(t, state.CommitteeResignations[resignedCold])
	require.NotContains(t, state.HotKeyAuthorizationsByCredential, resignedCold)
}

func TestParsePoolStateRewardAccountWireOffset(t *testing.T) {
	// PoolParams is [operator, vrf, pledge, cost, margin, reward-account,
	// owners, ...]. The distinct margin and owners sentinels protect index 5.
	raw := "81a1581c" + strings.Repeat("88", common.Blake2b224Size) +
		"87" +
		"581c" + strings.Repeat("89", common.Blake2b224Size) +
		"5820" + strings.Repeat("8a", common.Blake2b256Size) +
		"19012c190190820102" +
		"581df0" + strings.Repeat("8b", common.Blake2b224Size) +
		"81581c" + strings.Repeat("8c", common.Blake2b224Size)
	poolID := filledBlake2b224(0x88)
	state := parseSyntheticInitialState(t, "", raw, "", "")
	require.True(t, state.PoolRegistrations[poolID])
	require.Equal(t, ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: filledBlake2b224(0x8b),
	}, state.PoolRewardAccounts[poolID])
}

func TestParseDelegationStateCardanoAccountWireOffsets(t *testing.T) {
	// Conway AccountState is [reward, deposit, pool-delegation,
	// drep-delegation]. Key and script stake credentials intentionally share a
	// hash so assertions require the credential type to survive parsing.
	raw := "81a2" +
		"8200581c" + strings.Repeat("99", common.Blake2b224Size) +
		"84186402581c" + strings.Repeat("aa", common.Blake2b224Size) +
		"8201581c" + strings.Repeat("ab", common.Blake2b224Size) +
		"8201581c" + strings.Repeat("99", common.Blake2b224Size) +
		"8418c803581c" + strings.Repeat("ac", common.Blake2b224Size) +
		"8200581c" + strings.Repeat("ad", common.Blake2b224Size)
	sharedHash := filledBlake2b224(0x99)
	keyAccount := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: sharedHash,
	}
	scriptAccount := ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: sharedHash,
	}
	state := parseSyntheticInitialState(t, "", "", raw, "")
	require.Len(t, state.RewardAccountBalances, 2)
	require.Equal(t, uint64(100), state.RewardAccountBalances[keyAccount])
	require.Equal(t, uint64(200), state.RewardAccountBalances[scriptAccount])
	require.True(t, state.StakeRegistrationsByCredential[keyAccount])
	require.True(t, state.StakeRegistrationsByCredential[scriptAccount])
	require.Equal(t, uint64(100), state.RewardAccounts[sharedHash])
	require.Equal(t, filledBlake2b224(0xaa), state.PoolDelegationsByCredential[keyAccount])
	require.Equal(t, filledBlake2b224(0xac), state.PoolDelegationsByCredential[scriptAccount])
	require.Equal(t, common.Drep{
		Type:       int(common.CredentialTypeScriptHash),
		Credential: bytes.Repeat([]byte{0xab}, common.Blake2b224Size),
	}, state.DRepDelegationsByCredential[keyAccount])
	require.Equal(t, common.Drep{
		Type:       int(common.CredentialTypeAddrKeyHash),
		Credential: bytes.Repeat([]byte{0xad}, common.Blake2b224Size),
	}, state.DRepDelegationsByCredential[scriptAccount])
}

func TestParseProposalCardanoWireOffsets(t *testing.T) {
	// GovActionState is [id, cc-votes, drep-votes, pool-votes, procedure,
	// proposed-in, expires-after]. ProposalProcedure is [deposit,
	// return-account, action, anchor], and UpdateCommittee is
	// [4, previous, removed, added, threshold]. Raw CBOR and distinct adjacent
	// sentinels keep the test independent from the production extractors.
	txID := strings.Repeat("ee", common.Blake2b256Size)
	raw := "85a1" +
		"825820" + txID + "03" +
		"87825820" + txID + "03a0a0a0" +
		"841903e8581de0" + strings.Repeat("f1", common.Blake2b224Size) +
		"8504f6d90102818200581c" + strings.Repeat("a1", common.Blake2b224Size) +
		"a28200581c" + strings.Repeat("b1", common.Blake2b224Size) +
		"1901f48201581c" + strings.Repeat("c1", common.Blake2b224Size) +
		"1902588219012c190190" +
		"8218771878" +
		"18411863" +
		"f6f6f6f6"
	state := parseSyntheticInitialState(t, "", "", "", raw)
	info, ok := state.Proposals[txID+"#3"]
	require.True(t, ok)
	require.Equal(t, common.GovActionTypeUpdateCommittee, info.ActionType)
	require.Equal(t, uint64(1000), info.Deposit)
	require.Equal(t, &ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: filledBlake2b224(0xf1),
	}, info.ReturnAccount)
	require.Equal(t, uint64(65), info.SubmittedEpoch)
	require.Equal(t, uint64(99), info.ExpiresAfter)
	require.True(t, info.RemovedMembers[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: filledBlake2b224(0xa1),
	}])
	require.Equal(t, uint64(500), info.ProposedMembersByCredential[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: filledBlake2b224(0xb1),
	}])
	require.Equal(t, uint64(600), info.ProposedMembersByCredential[ledger.RewardAccountKey{
		CredType:   common.CredentialTypeScriptHash,
		Credential: filledBlake2b224(0xc1),
	}])
}

func TestParseSyntheticCardanoWireOffsetsRejectShiftedFields(t *testing.T) {
	tests := []struct {
		name       string
		voting     string
		pool       string
		proposals  string
		assertions func(*testing.T, *ParsedInitialState)
	}{
		{
			name: "DRep expiry moved from index zero",
			voting: "82a18200581c" +
				strings.Repeat("d1", common.Blake2b224Size) +
				"82f6184da0",
			assertions: func(t *testing.T, state *ParsedInitialState) {
				key := ledger.RewardAccountKey{
					CredType:   common.CredentialTypeAddrKeyHash,
					Credential: filledBlake2b224(0xd1),
				}
				require.True(t, state.DRepRegistrationsByCredential[key])
				require.NotContains(t, state.DRepExpiries, key)
			},
		},
		{
			name: "pool reward account moved from index five",
			pool: "81a1581c" +
				strings.Repeat("d2", common.Blake2b224Size) +
				"87581c" + strings.Repeat("d3", common.Blake2b224Size) +
				"5820" + strings.Repeat("d4", common.Blake2b256Size) +
				"0102581de0" + strings.Repeat("d5", common.Blake2b224Size) +
				"f68180",
			assertions: func(t *testing.T, state *ParsedInitialState) {
				poolID := filledBlake2b224(0xd2)
				require.True(t, state.PoolRegistrations[poolID])
				require.NotContains(t, state.PoolRewardAccounts, poolID)
			},
		},
		{
			name: "proposal deposit and return account shifted right",
			proposals: "85a1825820" +
				strings.Repeat("d6", common.Blake2b256Size) + "00" +
				"87825820" + strings.Repeat("d6", common.Blake2b256Size) +
				"00a0a0a084f61903e88106f60102f6f6f6f6",
			assertions: func(t *testing.T, state *ParsedInitialState) {
				info := state.Proposals[strings.Repeat("d6", common.Blake2b256Size)+"#0"]
				require.Zero(t, info.Deposit)
				require.Nil(t, info.ReturnAccount)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := parseSyntheticInitialState(
				t,
				test.voting,
				test.pool,
				"",
				test.proposals,
			)
			test.assertions(t, state)
		})
	}
}

func TestParseStakeCredentialMapRewardAccountLayouts(t *testing.T) {
	hash := common.NewBlake2b224(make([]byte, 28))
	tests := []struct {
		name    string
		account []any
	}{
		{
			name: "vendored legacy UMap",
			account: []any{
				[]any{[]any{uint64(11), uint64(2)}},
				[]any{},
				[]any{},
				[]any{},
			},
		},
		{
			name: "modern Conway account state",
			account: []any{
				uint64(11),
				uint64(2),
				[]any{},
				[]any{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encodedAccount, err := cbor.Encode(test.account)
			require.NoError(t, err)
			encodedMap := []byte{0xa1, 0x82, 0x00, 0x58, 0x1c}
			encodedMap = append(encodedMap, hash.Bytes()...)
			encodedMap = append(encodedMap, encodedAccount...)

			entries := parseStakeCredentialMap(encodedMap)

			require.Len(t, entries, 1)
			require.Equal(t, uint64(11), entries[0].Balance)
			require.Equal(t, uint64(0), entries[0].CredType)
			require.Equal(t, hash.Bytes(), entries[0].Hash)
		})
	}
}

func TestExtractDRepDelegationPreservesType(t *testing.T) {
	tests := []struct {
		name     string
		raw      any
		expected int
	}{
		{
			name:     "always abstain",
			raw:      []any{uint64(common.DrepTypeAbstain)},
			expected: common.DrepTypeAbstain,
		},
		{
			name:     "always no confidence",
			raw:      []any{uint64(common.DrepTypeNoConfidence)},
			expected: common.DrepTypeNoConfidence,
		},
		{
			name: "wrapped always abstain",
			raw: []any{
				[]any{uint64(common.DrepTypeAbstain)},
			},
			expected: common.DrepTypeAbstain,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			delegation := extractDRepDelegation(test.raw)
			require.NotNil(t, delegation)
			require.Equal(t, test.expected, delegation.Type)
			require.Empty(t, delegation.Credential)
		})
	}
}

func TestParseInitialState(t *testing.T) {
	defer goleak.VerifyNone(t)
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}

	if len(vectors) == 0 {
		t.Fatal("no test vectors found")
	}

	// Test parsing first vector's initial state
	vector, err := DecodeTestVector(vectors[0])
	if err != nil {
		t.Fatalf("DecodeTestVector failed: %v", err)
	}

	state, err := ParseInitialState(vector.InitialState)
	if err != nil {
		t.Fatalf("ParseInitialState failed for %s: %v", vector.Title, err)
	}

	t.Logf("Parsed state for: %s", vector.Title)
	t.Logf("  Current epoch: %d", state.CurrentEpoch)
	t.Logf("  UTxOs: %d", len(state.Utxos))
	t.Logf("  Stake registrations: %d", len(state.StakeRegistrations))
	t.Logf("  Reward accounts: %d", len(state.RewardAccounts))
	t.Logf("  Pool registrations: %d", len(state.PoolRegistrations))
	t.Logf("  Committee members: %d", len(state.CommitteeMembers))
	t.Logf("  DRep registrations: %d", len(state.DRepRegistrations))
	t.Logf("  Hot key authorizations: %d", len(state.HotKeyAuthorizations))
	t.Logf("  Proposals: %d", len(state.Proposals))
	t.Logf("  Cost models: %d", len(state.CostModels))
	if state.Constitution != nil {
		t.Logf("  Constitution: URL=%s", state.Constitution.AnchorURL)
	}
	if len(state.PParamsHash) > 0 {
		t.Logf("  PParams hash: %x", state.PParamsHash)
	}
}

func TestParseAllInitialStates(t *testing.T) {
	defer goleak.VerifyNone(t)
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}
	if len(vectors) == 0 {
		t.Fatal("no test vectors found - testdata may be missing")
	}

	var (
		successCount   int
		failedVectors  []string
		totalUtxos     int
		totalPools     int
		totalStakes    int
		totalCommittee int
		totalDReps     int
		totalProposals int
	)

	for _, path := range vectors {
		vector, err := DecodeTestVector(path)
		if err != nil {
			failedVectors = append(failedVectors, path+": decode: "+err.Error())
			continue
		}

		state, err := ParseInitialState(vector.InitialState)
		if err != nil {
			failedVectors = append(failedVectors, path+": parse: "+err.Error())
			continue
		}

		successCount++
		totalUtxos += len(state.Utxos)
		totalPools += len(state.PoolRegistrations)
		totalStakes += len(state.StakeRegistrations)
		totalCommittee += len(state.CommitteeMembers)
		totalDReps += len(state.DRepRegistrations)
		totalProposals += len(state.Proposals)
	}

	if len(failedVectors) > 0 {
		t.Errorf("failed to parse %d vectors:", len(failedVectors))
		// Only show first 10 failures
		for i, msg := range failedVectors {
			if i >= 10 {
				t.Errorf("  ... and %d more", len(failedVectors)-10)
				break
			}
			t.Errorf("  %s", msg)
		}
	}

	t.Logf("Successfully parsed %d/%d vectors", successCount, len(vectors))
	t.Logf("Totals across all vectors:")
	t.Logf("  UTxOs: %d", totalUtxos)
	t.Logf("  Pool registrations: %d", totalPools)
	t.Logf("  Stake registrations: %d", totalStakes)
	t.Logf("  Committee members: %d", totalCommittee)
	t.Logf("  DRep registrations: %d", totalDReps)
	t.Logf("  Proposals: %d", totalProposals)
}

func TestParseInitialStateUtxos(t *testing.T) {
	// Find a vector with UTxOs
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}

	var foundUtxos bool
	for _, path := range vectors {
		vector, err := DecodeTestVector(path)
		if err != nil {
			continue
		}

		state, err := ParseInitialState(vector.InitialState)
		if err != nil {
			continue
		}

		if len(state.Utxos) > 0 {
			foundUtxos = true
			t.Logf("Vector %s has %d UTxOs", vector.Title, len(state.Utxos))

			// Verify UTxO structure
			for id, utxo := range state.Utxos {
				if len(utxo.TxHash) == 0 {
					t.Errorf("UTxO %s has empty TxHash", id)
				}
				if utxo.Output == nil {
					t.Errorf("UTxO %s has nil Output", id)
				} else {
					t.Logf(
						"  UTxO %s: amount=%s, address=%s",
						id,
						utxo.Output.Amount().String(),
						utxo.Output.Address().String(),
					)
				}
				break
			}
			break
		}
	}

	if !foundUtxos {
		t.Log("No vectors with UTxOs found (this may be expected)")
	}
}

func TestParseInitialStateGovernance(t *testing.T) {
	// Find a vector with governance state
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}

	var (
		foundCommittee    bool
		foundDReps        bool
		foundProposals    bool
		foundConstitution bool
	)

	for _, path := range vectors {
		vector, err := DecodeTestVector(path)
		if err != nil {
			continue
		}

		state, err := ParseInitialState(vector.InitialState)
		if err != nil {
			continue
		}

		if len(state.CommitteeMembers) > 0 && !foundCommittee {
			foundCommittee = true
			t.Logf(
				"Vector %s has %d committee members",
				vector.Title,
				len(state.CommitteeMembers),
			)
		}

		if len(state.DRepRegistrations) > 0 && !foundDReps {
			foundDReps = true
			t.Logf(
				"Vector %s has %d DReps",
				vector.Title,
				len(state.DRepRegistrations),
			)
		}

		if len(state.Proposals) > 0 && !foundProposals {
			foundProposals = true
			t.Logf(
				"Vector %s has %d proposals",
				vector.Title,
				len(state.Proposals),
			)
			for id, info := range state.Proposals {
				t.Logf("  Proposal %s: type=%d, expires=%d, votes=%d",
					id, info.ActionType, info.ExpiresAfter, len(info.Votes))
				break
			}
		}

		if state.Constitution != nil && !foundConstitution {
			foundConstitution = true
			t.Logf(
				"Vector %s has constitution: URL=%s",
				vector.Title,
				state.Constitution.AnchorURL,
			)
		}

		if foundCommittee && foundDReps && foundProposals && foundConstitution {
			break
		}
	}

	t.Logf(
		"Found governance state: committee=%v, dreps=%v, proposals=%v, constitution=%v",
		foundCommittee,
		foundDReps,
		foundProposals,
		foundConstitution,
	)
}

func TestExtractUtxoId(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string format",
			input:    "abc123#0",
			expected: "abc123#0",
		},
		{
			name:     "array format",
			input:    []any{[]byte{0x01, 0x02, 0x03}, uint64(5)},
			expected: "010203#5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUtxoId(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractGovActionId(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "array format",
			input:    []any{[]byte{0xab, 0xcd}, uint64(3)},
			expected: "abcd#3",
		},
		{
			name:     "string passthrough",
			input:    "txhash#1",
			expected: "txhash#1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractGovActionId(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDecodeCompactBlueprintTransactionOutput(t *testing.T) {
	raw, err := hex.DecodeString(
		"001d6088028438394946279f9ed8d66d718679b82f26b75391cb6df8107c8f00cff7e8afb7928000",
	)
	require.NoError(t, err)
	output, ok := decodeCompactTransactionOutput(raw)
	require.True(t, ok)
	shelleyOutput, ok := output.(*shelley.ShelleyTransactionOutput)
	require.True(t, ok)
	require.Equal(t, uint64(45000000000000000), shelleyOutput.OutputAmount)
	require.Equal(t, "addr_test1vzyq9ppc89y5vfulnmvdvmt3seumstexkaferjmdlqg8ercx8lee2", shelleyOutput.OutputAddress.String())

	_, ok = decodeCompactTransactionOutput(append([]byte{6}, raw[1:]...))
	require.False(t, ok)

	// The datum-hash and optimized Ada-only constructors are also emitted by
	// the Blueprint corpus and must remain visible to the state provider.
	for _, encoded := range []string{
		"011d706ba8f502e9f994ed5518d3007f705452969a10b01901259a1141405600bce21ae88bd757ad5b9bedf372d8d3f0cf6c962a469db61a265f6418e1ffed86da29ec",
		"020100adf84fd242cd66edf5d9c60026a50a521408eaf8a7fcfe833180779fb4faeb54dda0db6c564d59d7d9c4c909d5b698a681cbf0010000006a92d36a00cff7e8afa897db1e",
	} {
		compact, err := hex.DecodeString(encoded)
		require.NoError(t, err)
		_, ok := decodeCompactTransactionOutput(compact)
		require.True(t, ok)
	}
}
