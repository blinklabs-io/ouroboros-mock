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
	"github.com/blinklabs-io/ouroboros-mock/ledger"
	"github.com/stretchr/testify/require"
)

type committeeCapabilityProvider struct {
	available bool
	cold      *common.CommitteeMember
	hot       *common.CommitteeMember
}

func (p committeeCapabilityProvider) CommitteeStateAvailable() (bool, error) {
	return p.available, nil
}

func (p committeeCapabilityProvider) CommitteeCredentialMember(
	common.Credential,
) (*common.CommitteeMember, error) {
	return p.cold, nil
}

func (p committeeCapabilityProvider) CommitteeHotCredentialMember(
	common.Credential,
) (*common.CommitteeMember, error) {
	return p.hot, nil
}

func TestCommitteeCertificateLedgerStateDelegatesCapabilities(t *testing.T) {
	cold := &common.CommitteeMember{ColdKey: common.Blake2b224{0x31}}
	hot := &common.CommitteeMember{ColdKey: common.Blake2b224{0x32}}
	state := committeeCertificateLedgerState{
		LedgerState: ledger.NewLedgerStateBuilder().Build(),
		provider: committeeCapabilityProvider{
			available: true,
			cold:      cold,
			hot:       hot,
		},
	}

	available, err := state.CommitteeStateAvailable()
	require.NoError(t, err)
	require.True(t, available)

	gotCold, err := state.CommitteeCredentialMember(common.Credential{})
	require.NoError(t, err)
	require.Same(t, cold, gotCold)

	gotHot, err := state.CommitteeHotCredentialMember(common.Credential{})
	require.NoError(t, err)
	require.Same(t, hot, gotHot)
}
