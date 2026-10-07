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

package ledger_test

import (
	"testing"

	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
	"github.com/stretchr/testify/require"
)

// lcommon.EpochState is a degrading capability: a state without it keeps
// every POOL predicate except the retirement-epoch bound. The default mock
// must not advertise the capability and then fail every lookup, which would
// reject valid retirements.
func TestDefaultStateAcceptsPoolRetirementWithoutEpochState(t *testing.T) {
	operator := lcommon.PoolKeyHash{0x01}
	state := ledger.NewLedgerStateBuilder().
		WithPools([]*lcommon.PoolRegistrationCertificate{
			{Operator: operator},
		}).
		Build()
	tx := &shelley.ShelleyTransaction{
		Body: shelley.ShelleyTransactionBody{
			TxCertificates: []lcommon.CertificateWrapper{{
				Type: uint(lcommon.CertificateTypePoolRetirement),
				Certificate: &lcommon.PoolRetirementCertificate{
					CertType:    uint(lcommon.CertificateTypePoolRetirement),
					PoolKeyHash: operator,
					Epoch:       5,
				},
			}},
		},
	}
	pparams := &shelley.ShelleyProtocolParameters{
		ProtocolMajor: 2,
		MaxEpoch:      18,
	}
	require.NoError(
		t,
		shelley.UtxoValidatePoolCertificates(tx, 100, state, pparams),
	)
}
