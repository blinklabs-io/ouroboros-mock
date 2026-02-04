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
	"path/filepath"
	"testing"

	"go.uber.org/goleak"
)

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
