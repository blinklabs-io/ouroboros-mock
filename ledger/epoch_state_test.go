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

func poolRetirementTx(operator lcommon.PoolKeyHash, epoch uint64) *shelley.ShelleyTransaction {
	return &shelley.ShelleyTransaction{
		Body: shelley.ShelleyTransactionBody{
			TxCertificates: []lcommon.CertificateWrapper{{
				Type: uint(lcommon.CertificateTypePoolRetirement),
				Certificate: &lcommon.PoolRetirementCertificate{
					CertType:    uint(lcommon.CertificateTypePoolRetirement),
					PoolKeyHash: operator,
					Epoch:       epoch,
				},
			}},
		},
	}
}

// The Shelley POOL retirement bound needs the current epoch, so a state
// without an epoch mapping must fail closed rather than skip the bound.
func TestDefaultStateRejectsPoolRetirementWithoutEpochState(t *testing.T) {
	operator := lcommon.PoolKeyHash{0x01}
	state := ledger.NewLedgerStateBuilder().
		WithPools([]*lcommon.PoolRegistrationCertificate{
			{Operator: operator},
		}).
		Build()
	pparams := &shelley.ShelleyProtocolParameters{
		ProtocolMajor: 2,
		MaxEpoch:      18,
	}
	err := shelley.UtxoValidatePoolCertificates(
		poolRetirementTx(operator, 5),
		100,
		state,
		pparams,
	)
	require.ErrorIs(t, err, lcommon.EpochStateUnavailableError{})
}

func TestConfiguredEpochStateEnforcesPoolRetirementBound(t *testing.T) {
	operator := lcommon.PoolKeyHash{0x01}
	state := ledger.NewLedgerStateBuilder().
		WithPools([]*lcommon.PoolRegistrationCertificate{
			{Operator: operator},
		}).
		WithEpochForSlot(func(slot uint64) (uint64, error) {
			return slot / 100, nil
		}).
		Build()
	pparams := &shelley.ShelleyProtocolParameters{
		ProtocolMajor: 2,
		MaxEpoch:      18,
	}
	// Slot 100 is in epoch 1, so the valid retirement epochs are 2 to 19.
	for _, epoch := range []uint64{2, 19} {
		require.NoError(
			t,
			shelley.UtxoValidatePoolCertificates(
				poolRetirementTx(operator, epoch), 100, state, pparams,
			),
			"epoch %d",
			epoch,
		)
	}
	for _, epoch := range []uint64{1, 20} {
		err := shelley.UtxoValidatePoolCertificates(
			poolRetirementTx(operator, epoch), 100, state, pparams,
		)
		var wrong shelley.StakePoolRetirementWrongEpochError
		require.ErrorAs(t, err, &wrong, "epoch %d", epoch)
	}
}
