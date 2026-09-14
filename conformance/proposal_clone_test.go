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
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

func TestParameterComparisonCoversDecodedFields(t *testing.T) {
	t.Parallel()
	rat := func(n int64) *cbor.Rat {
		return &cbor.Rat{Rat: big.NewRat(n, 3)}
	}
	decode := func(t *testing.T, key uint, value any) *conway.ConwayProtocolParameterUpdate {
		t.Helper()
		raw, err := cbor.Encode(map[uint]any{key: value})
		if err != nil {
			t.Fatal(err)
		}
		var update conway.ConwayProtocolParameterUpdate
		if _, err := cbor.Decode(raw, &update); err != nil {
			t.Fatal(err)
		}
		return &update
	}
	check := func(t *testing.T, key uint, first, second any) {
		t.Helper()
		a, same, b := decode(
			t,
			key,
			first,
		), decode(
			t,
			key,
			first,
		), decode(
			t,
			key,
			second,
		)
		if !parameterUpdatesEqual(a, same) {
			t.Fatal("independently decoded equal parameter values differ")
		}
		if parameterUpdatesEqual(a, b) || parameterUpdatesEqual(b, a) {
			t.Fatal("changed parameter value was ignored")
		}
		if parameterUpdatesEqual(a, &conway.ConwayProtocolParameterUpdate{}) {
			t.Fatal("present parameter compares equal to absent parameter")
		}
	}
	// Scalar keys from ConwayProtocolParameterUpdate's CBOR field contract.
	for _, key := range []uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 16, 17, 22, 23, 24, 27, 28, 29, 30, 31, 32} {
		t.Run(fmt.Sprintf("scalar_%d", key), func(t *testing.T) {
			t.Parallel()
			check(t, key, uint64(1), uint64(2))
		})
	}
	for _, key := range []uint{9, 10, 11, 33} {
		t.Run(fmt.Sprintf("rational_%d", key), func(t *testing.T) {
			t.Parallel()
			check(t, key, rat(1), rat(2))
		})
	}
	t.Run("cost_models_18", func(t *testing.T) {
		t.Parallel()
		check(t, 18, map[uint][]int64{0: {1}}, map[uint][]int64{0: {2}})
	})
	for key, length := range map[uint]int{14: 2, 19: 2, 20: 2, 21: 2, 25: 5, 26: 10} {
		for component := range length {
			t.Run(
				fmt.Sprintf("array_%d_component_%d", key, component),
				func(t *testing.T) {
					t.Parallel()
					first, second := make([]any, length), make([]any, length)
					for i := range length {
						if key == 19 || key == 25 || key == 26 {
							first[i], second[i] = rat(1), rat(1)
						} else {
							first[i], second[i] = uint64(1), uint64(1)
						}
					}
					if key == 19 || key == 25 || key == 26 {
						second[component] = rat(2)
					} else {
						second[component] = uint64(2)
					}
					check(t, key, first, second)
				},
			)
		}
	}
}

func TestProposalComparisonUsesParameterFields(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		gotRaw, wantRaw []byte
		mutate          func(*conway.ConwayProtocolParameterUpdate)
		equal           bool
	}{
		"equivalent encodings": {
			gotRaw:  []byte{0xa1, 0x00, 0x01},
			wantRaw: []byte{0xbf, 0x00, 0x01, 0xff},
			equal:   true,
		},
		"equal large rational": {
			// A0 = 2^65 / 1: accepted by the parameter decoder, but its
			// rational encoder cannot encode the numerator as uint64.
			gotRaw: []byte{
				0xa1, 0x09, 0xd8, 0x1e, 0x82, 0xc2, 0x49,
				0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0x01,
			},
			wantRaw: []byte{
				0xa1, 0x09, 0xd8, 0x1e, 0x82, 0xc2, 0x49,
				0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0x01,
			},
			equal: true,
		},
		"changed field with unchanged cache": {
			gotRaw:  []byte{0xa1, 0x00, 0x01},
			wantRaw: []byte{0xa1, 0x00, 0x01},
			mutate: func(p *conway.ConwayProtocolParameterUpdate) {
				value := uint(2)
				p.MinFeeA = &value
			},
		},
		"different cost model values": {
			gotRaw:  []byte{0xa1, 0x12, 0xa1, 0x00, 0x81, 0x01},
			wantRaw: []byte{0xa1, 0x12, 0xa1, 0x00, 0x81, 0x02},
		},
		"null versus empty cost model": {
			gotRaw:  []byte{0xa1, 0x12, 0xa1, 0x00, 0xf6},
			wantRaw: []byte{0xa1, 0x12, 0xa1, 0x00, 0x80},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var got, want conway.ConwayProtocolParameterUpdate
			if _, err := cbor.Decode(tc.gotRaw, &got); err != nil {
				t.Fatal(err)
			}
			if _, err := cbor.Decode(tc.wantRaw, &want); err != nil {
				t.Fatal(err)
			}
			if tc.mutate != nil {
				tc.mutate(&got)
			}
			a := map[string]*ProposalState{
				"p": {GovActionInfo: GovActionInfo{ParameterUpdate: &got}},
			}
			b := map[string]*ProposalState{
				"p": {GovActionInfo: GovActionInfo{ParameterUpdate: &want}},
			}
			if equal := proposalStatesEqual(a, b, 0); equal != tc.equal {
				t.Errorf("parameter comparison=%v, want %v", equal, tc.equal)
			}
			if !bytes.Equal(got.Cbor(), tc.gotRaw) ||
				!bytes.Equal(want.Cbor(), tc.wantRaw) {
				t.Error("comparison changed preserved CBOR")
			}
		})
	}
}

func TestGovernanceClonesPreserveHashSlices(t *testing.T) {
	t.Parallel()
	for field, clone := range map[string]func([]byte) []byte{
		"proposal policy": func(hash []byte) []byte {
			return cloneProposalState(&ProposalState{
				GovActionInfo: GovActionInfo{PolicyHash: hash},
			}).PolicyHash
		},
		"constitution anchor": func(hash []byte) []byte {
			return cloneConstitutionInfo(&ConstitutionInfo{
				AnchorHash: hash,
			}).AnchorHash
		},
		"constitution policy": func(hash []byte) []byte {
			return cloneConstitutionInfo(&ConstitutionInfo{
				PolicyHash: hash,
			}).PolicyHash
		},
	} {
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			for name, original := range map[string][]byte{
				"nil": nil, "empty": {}, "populated": {7, 3},
			} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					copied := clone(original)
					if (copied == nil) != (original == nil) ||
						!bytes.Equal(copied, original) {
						t.Fatal("clone changed hash nilness or bytes")
					}
					if len(copied) > 0 {
						copied[0]++
						if original[0] != 7 {
							t.Fatal("clone aliases the original hash")
						}
					}
				})
			}
		})
	}
}

func TestCloneProposalStatePreservesCostModelSlices(t *testing.T) {
	t.Parallel()
	for name, model := range map[string][]int64{
		"nil":       nil,
		"empty":     {},
		"populated": {7, -3},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			original := &ProposalState{GovActionInfo: GovActionInfo{
				ParameterUpdate: &conway.ConwayProtocolParameterUpdate{
					CostModels: map[uint][]int64{0: model},
				},
			}}
			cloned := cloneProposalState(original)
			copied, ok := cloned.ParameterUpdate.CostModels[0]
			if !ok || (copied == nil) != (model == nil) ||
				!slices.Equal(copied, model) {
				t.Fatal("clone changed cost model presence, nilness, or values")
			}
			if len(copied) > 0 {
				copied[0]++
				if model[0] != 7 {
					t.Fatal("clone aliases the original cost model")
				}
			}
			delete(cloned.ParameterUpdate.CostModels, 0)
			if _, ok := original.ParameterUpdate.CostModels[0]; !ok {
				t.Fatal("clone aliases the original cost model map")
			}
		})
	}
}
