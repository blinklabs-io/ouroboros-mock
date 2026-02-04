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
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"go.uber.org/goleak"
)

// TestConstitutionPrevGovIdValidation tests the PrevGovId validation logic directly.
func TestConstitutionPrevGovIdValidation(t *testing.T) {
	defer goleak.VerifyNone(t)
	validator := NewValidator()
	govState := NewGovernanceState()

	// Case 1: No root exists, empty PrevGovId is valid (first constitution)
	t.Run("FirstConstitution_EmptyPrevGovId_Valid", func(t *testing.T) {
		action := &common.NewConstitutionGovAction{
			ActionId: nil, // Empty PrevGovId
		}

		err := validator.validateNewConstitution(action, govState)
		if err != nil {
			t.Errorf(
				"First constitution with empty PrevGovId should be valid, got error: %v",
				err,
			)
		}
	})

	// Case 2: Root exists, empty PrevGovId is INVALID
	t.Run("SecondConstitution_EmptyPrevGovId_Invalid", func(t *testing.T) {
		// Set a constitution root (simulate enacted proposal)
		rootId := "fae8dfcae1a80999d5a9f3dcd696ddaa7edecb2e86c1bd4dc09d59e54dcda7d4#0"
		govState.Roots.Constitution = &rootId

		action := &common.NewConstitutionGovAction{
			ActionId: nil, // Empty PrevGovId
		}

		err := validator.validateNewConstitution(action, govState)
		if err == nil {
			t.Error(
				"Constitution with empty PrevGovId after root exists should be INVALID",
			)
		} else {
			t.Logf("Correctly rejected: %v", err)
		}
	})

	// Case 3: Root exists, PrevGovId matches root, VALID
	t.Run("SecondConstitution_PrevGovIdMatchesRoot_Valid", func(t *testing.T) {
		// Root already set from previous test case
		rootId := "fae8dfcae1a80999d5a9f3dcd696ddaa7edecb2e86c1bd4dc09d59e54dcda7d4#0"
		govState.Roots.Constitution = &rootId

		// Create action with PrevGovId matching root
		var txHash common.Blake2b256
		copy(
			txHash[:],
			[]byte{
				0xfa,
				0xe8,
				0xdf,
				0xca,
				0xe1,
				0xa8,
				0x09,
				0x99,
				0xd5,
				0xa9,
				0xf3,
				0xdc,
				0xd6,
				0x96,
				0xdd,
				0xaa,
				0x7e,
				0xde,
				0xcb,
				0x2e,
				0x86,
				0xc1,
				0xbd,
				0x4d,
				0xc0,
				0x9d,
				0x59,
				0xe5,
				0x4d,
				0xcd,
				0xa7,
				0xd4,
			},
		)

		action := &common.NewConstitutionGovAction{
			ActionId: &common.GovActionId{
				TransactionId: txHash,
				GovActionIdx:  0,
			},
		}

		err := validator.validateNewConstitution(action, govState)
		if err != nil {
			t.Errorf(
				"Constitution with PrevGovId matching root should be valid, got error: %v",
				err,
			)
		}
	})
}
