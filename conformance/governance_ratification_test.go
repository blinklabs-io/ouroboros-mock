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

// TestGovernanceRatificationFlow tests the complete ratification and enactment flow.
// This validates that:
// 1. Proposals with votes are ratified in the epoch after submission
// 2. Ratified proposals are enacted in the following epoch
// 3. Enacted proposals update governance roots
func TestGovernanceRatificationFlow(t *testing.T) {
	defer goleak.VerifyNone(t)
	stateManager := NewMockStateManager()
	govState := stateManager.GetGovernanceState()

	// Epoch 0: Add a constitution proposal with votes
	// Votes must be in format "voterType:credHash" with at least 2 voter types
	// Vote values per CIP-1694: 0=No, 1=Yes, 2=Abstain
	proposalId := "test_proposal#0"
	govState.AddProposal(proposalId, GovActionInfo{
		ActionType:     common.GovActionTypeNewConstitution,
		SubmittedEpoch: 0,
		ExpiresAfter:   10,
		Votes: map[string]uint8{
			"0:cc_voter_hash":   1, // CC Yes vote (type 0, vote=1=Yes)
			"2:drep_voter_hash": 1, // DRep Yes vote (type 2, vote=1=Yes)
		},
	})

	// Verify initial state
	if govState.Roots.Constitution != nil {
		t.Error("Constitution root should be nil initially")
	}

	// Epoch 1: Ratification should occur
	if err := stateManager.ProcessEpochBoundary(1); err != nil {
		t.Fatalf("ProcessEpochBoundary(1) failed: %v", err)
	}

	proposal := govState.GetProposal(proposalId)
	if proposal == nil {
		t.Fatal("Proposal should still exist after ratification")
	}
	if proposal.RatifiedEpoch == nil {
		t.Error("Proposal should be ratified in epoch 1")
	} else if *proposal.RatifiedEpoch != 1 {
		t.Errorf("Proposal should be ratified in epoch 1, got epoch %d", *proposal.RatifiedEpoch)
	}

	// Constitution root should still be nil (enactment happens next epoch)
	if govState.Roots.Constitution != nil {
		t.Error("Constitution root should still be nil after ratification")
	}

	// Epoch 2: Enactment should occur
	if err := stateManager.ProcessEpochBoundary(2); err != nil {
		t.Fatalf("ProcessEpochBoundary(2) failed: %v", err)
	}

	// Constitution root should now be set
	if govState.Roots.Constitution == nil {
		t.Error("Constitution root should be set after enactment")
	} else if *govState.Roots.Constitution != proposalId {
		t.Errorf("Constitution root should be %s, got %s", proposalId, *govState.Roots.Constitution)
	}

	// Proposal should be moved to enacted
	if govState.GetProposal(proposalId) != nil {
		t.Error(
			"Proposal should be removed from active proposals after enactment",
		)
	}
	if !govState.EnactedProposals[proposalId] {
		t.Error("Proposal should be marked as enacted")
	}
}

// TestGovernanceRootValidation tests that governance roots correctly validate PrevGovId.
func TestGovernanceRootValidation(t *testing.T) {
	validator := NewValidator()
	govState := NewGovernanceState()

	t.Run("EmptyPrevGovId_NoRoot_Valid", func(t *testing.T) {
		// First constitution with empty PrevGovId should be valid
		action := &common.NewConstitutionGovAction{ActionId: nil}
		err := validator.validateNewConstitution(action, govState)
		if err != nil {
			t.Errorf(
				"First constitution with empty PrevGovId should be valid: %v",
				err,
			)
		}
	})

	t.Run("EmptyPrevGovId_WithRoot_Invalid", func(t *testing.T) {
		// Set a constitution root
		rootId := "enacted_proposal#0"
		govState.Roots.Constitution = &rootId

		// Second constitution with empty PrevGovId should be INVALID
		action := &common.NewConstitutionGovAction{ActionId: nil}
		err := validator.validateNewConstitution(action, govState)
		if err == nil {
			t.Error(
				"Constitution with empty PrevGovId should be invalid when root exists",
			)
		}
	})

	t.Run("PrevGovIdMatchesRoot_Valid", func(t *testing.T) {
		// Set a specific root ID
		rootId := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef#0"
		govState.Roots.Constitution = &rootId

		// Constitution with PrevGovId matching root should be valid
		// The txHash must hex-decode to match the root's hex string
		txHash := common.Blake2b256{
			0x01,
			0x23,
			0x45,
			0x67,
			0x89,
			0xab,
			0xcd,
			0xef,
			0x01,
			0x23,
			0x45,
			0x67,
			0x89,
			0xab,
			0xcd,
			0xef,
			0x01,
			0x23,
			0x45,
			0x67,
			0x89,
			0xab,
			0xcd,
			0xef,
			0x01,
			0x23,
			0x45,
			0x67,
			0x89,
			0xab,
			0xcd,
			0xef,
		}

		action := &common.NewConstitutionGovAction{
			ActionId: &common.GovActionId{
				TransactionId: txHash,
				GovActionIdx:  0,
			},
		}

		err := validator.validateNewConstitution(action, govState)
		if err != nil {
			t.Errorf(
				"Constitution with PrevGovId matching root should be valid: %v",
				err,
			)
		}
	})
}
