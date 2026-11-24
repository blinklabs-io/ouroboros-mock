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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/stretchr/testify/require"
)

// voteOnExpiry runs gouroboros' vote-expiry rule against the ledger state the
// manager builds, for a vote cast at slot on an action that expires after
// epoch expiresAfter. The reference accepts the vote while the current epoch
// is at most expiresAfter (cardano-ledger Conway Rules/Gov.hs,
// checkVotesAreNotForExpiredActions), so the action's ExpirySlot must be the
// last slot of that epoch.
func voteOnExpiry(
	t *testing.T,
	sm *MockStateManager,
	expiresAfter uint64,
	slot uint64,
) error {
	t.Helper()
	actionId := common.GovActionId{TransactionId: common.Blake2b256{0xe1}}
	sm.govState.Proposals[fmt.Sprintf(
		"%x#%d", actionId.TransactionId[:], actionId.GovActionIdx,
	)] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType:   common.GovActionTypeInfo,
			ExpiresAfter: expiresAfter,
		},
	}
	voter := common.Voter{
		Type: common.VoterTypeDRepKeyHash,
		Hash: common.Blake2b224{0xe2},
	}
	tx := &conway.ConwayTransaction{TxIsValid: true}
	tx.Body.TxVotingProcedures = common.VotingProcedures{
		&voter: {&actionId: common.VotingProcedure{Vote: common.GovVoteYes}},
	}
	return conway.UtxoValidateVotingOnExpiredGovAction(
		tx, slot, sm.buildLedgerState(), nil,
	)
}

func TestGovActionExpirySlotIsLastSlotOfExpiryEpoch(t *testing.T) {
	const expiresAfter = 5
	cases := []struct {
		name     string
		schedule func(*MockStateManager)
		// first and last slot of epoch expiresAfter
		first, last uint64
	}{
		{
			name:     "without a schedule, 432000-slot epochs from slot 0",
			schedule: func(*MockStateManager) {},
			first:    5 * 432000,
			last:     6*432000 - 1,
		},
		{
			name: "vector schedule",
			schedule: func(sm *MockStateManager) {
				// Epoch 3 starts at slot 100; epoch 5 spans 120-129.
				sm.SetEpochSchedule(100, 3, 10)
			},
			first: 120,
			last:  129,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vote := func(slot uint64) error {
				sm := NewMockStateManager()
				tc.schedule(sm)
				return voteOnExpiry(t, sm, expiresAfter, slot)
			}
			require.NoError(t, vote(tc.first), "first slot of expiry epoch")
			require.NoError(t, vote(tc.first+1), "inside expiry epoch")
			require.NoError(t, vote(tc.last), "last slot of expiry epoch")
			var expired conway.VotingOnExpiredGovActionError
			require.ErrorAs(t, vote(tc.last+1), &expired)
			require.Equal(t, tc.last, expired.ExpirySlot)
		})
	}
}

type scheduleRecordingStateManager struct {
	*MockStateManager
	schedules [][3]uint64
}

func (r *scheduleRecordingStateManager) SetEpochSchedule(
	startSlot, initialEpoch, epochLength uint64,
) {
	r.schedules = append(
		r.schedules,
		[3]uint64{startSlot, initialEpoch, epochLength},
	)
	r.MockStateManager.SetEpochSchedule(startSlot, initialEpoch, epochLength)
}

// The harness must pass each vector's schedule to the state manager after
// loading its initial state, and again after the reload a rollback performs,
// which follows a Reset that clears it.
func TestHarnessSendsEpochSchedule(t *testing.T) {
	root, err := ExtractEmbeddedTestdata(t.TempDir())
	require.NoError(t, err)
	vectorPath := filepath.Join(
		root, "synthetic", "rollback", "CurrentTreasuryValue_V1",
	)
	sm := &scheduleRecordingStateManager{MockStateManager: NewMockStateManager()}
	h := NewHarness(sm, HarnessConfig{TestdataRoot: root})

	h.RunVector(t, vectorPath)
	require.NotZero(t, h.epochLength)
	want := [3]uint64{h.startSlot, h.initialEpoch, h.epochLength}
	require.NotEmpty(t, sm.schedules)
	require.Equal(t, want, sm.schedules[0])

	sm.schedules = nil
	result := h.runVectorWithResult(vectorPath)
	require.NoError(t, result.Error)
	require.NotEmpty(t, sm.schedules)
	require.Equal(t, want, sm.schedules[0])

	sm.schedules = nil
	require.NoError(t, h.rollback(h.startSlot))
	require.Equal(t, [][3]uint64{want}, sm.schedules)
	require.Equal(t, h.epochLength, sm.scheduleEpochLength)
}
