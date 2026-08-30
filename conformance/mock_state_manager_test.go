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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLedgerStateFindsProposedCommitteeMember(t *testing.T) {
	coldKey := common.Blake2b224{0x01}
	const expiryEpoch = uint64(42)

	stateManager := NewMockStateManager()
	stateManager.govState.Proposals["proposal#0"] = &ProposalState{
		GovActionInfo: GovActionInfo{
			ActionType: common.GovActionTypeUpdateCommittee,
			ProposedMembers: map[common.Blake2b224]uint64{
				coldKey: expiryEpoch,
			},
		},
	}

	member, err := stateManager.buildLedgerState().CommitteeMember(coldKey)
	require.NoError(t, err)
	require.NotNil(t, member)
	assert.Equal(t, coldKey, member.ColdKey)
	assert.Equal(t, expiryEpoch, member.ExpiryEpoch)
}
