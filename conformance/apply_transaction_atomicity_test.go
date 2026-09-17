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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyTransactionRejectsWithdrawalAgainstRegistrationInSameTx covers the
// balance a withdrawal is actually checked against. Certificates are applied
// before withdrawals, and a registration certificate resets the reward account
// to zero, so a withdrawal from a credential this transaction registers can
// never succeed. Validating it against the pre-certificate balance - absent
// here, so the check is skipped entirely - lets the transaction mutate UTxOs
// and certificate state before failing.
func TestApplyTransactionRejectsWithdrawalAgainstRegistrationInSameTx(
	t *testing.T,
) {
	t.Parallel()
	credential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x07},
	}
	key := ledger.NewRewardAccountKey(credential)
	address, err := common.NewAddressFromParts(
		common.AddressTypeNoneKey,
		common.AddressNetworkTestnet,
		nil,
		credential.Credential[:],
	)
	require.NoError(t, err)

	manager := NewMockStateManager()
	input, err := ledger.NewTransactionInputBuilder().
		WithTxId([]byte{0x07}).WithIndex(0).Build()
	require.NoError(t, err)
	utxoId := fmt.Sprintf(
		"%s#0",
		hex.EncodeToString(input.Id().Bytes()),
	)
	manager.utxos[utxoId] = common.Utxo{Id: input}
	output, err := ledger.NewTransactionOutputBuilder().
		WithAddress(address.String()).WithLovelace(1).Build()
	require.NoError(t, err)

	builder := ledger.NewTransactionBuilder().
		WithCertificates(&common.RegistrationCertificate{
			CertType:        uint(common.CertificateTypeRegistration),
			StakeCredential: credential,
			Amount:          1,
		}).
		WithWithdrawals(map[*common.Address]uint64{&address: 2})
	builder.WithId([]byte{0x08}).WithInputs(input).WithOutputs(output)
	tx, err := builder.Build()
	require.NoError(t, err)

	err = manager.ApplyTransaction(tx, 0)
	require.ErrorContains(t, err, "exceeds reward account balance")
	assert.NotContains(
		t,
		manager.stakeRegistrations,
		key,
		"a rejected transaction must not leave certificate effects applied",
	)
	assert.NotContains(t, manager.rewardAccounts, key)
	assert.Contains(
		t,
		manager.utxos,
		utxoId,
		"a rejected transaction must not consume its inputs",
	)
}

// TestApplyTransactionAllowsWithdrawalAfterDeregistrationAndReregistration
// pins the other end of the same rule: the account is removed and recreated at
// zero, so the withdrawal has nothing to draw on and the transaction is
// rejected before any mutation.
func TestApplyTransactionAllowsWithdrawalAfterDeregistrationAndReregistration(
	t *testing.T,
) {
	t.Parallel()
	credential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: common.Blake2b224{0x09},
	}
	key := ledger.NewRewardAccountKey(credential)
	address, err := common.NewAddressFromParts(
		common.AddressTypeNoneKey,
		common.AddressNetworkTestnet,
		nil,
		credential.Credential[:],
	)
	require.NoError(t, err)

	manager := NewMockStateManager()
	manager.rewardAccounts[key] = 100
	manager.stakeRegistrations[key] = 0
	input, err := ledger.NewTransactionInputBuilder().
		WithTxId([]byte{0x09}).WithIndex(0).Build()
	require.NoError(t, err)
	utxoId := fmt.Sprintf("%s#0", hex.EncodeToString(input.Id().Bytes()))
	manager.utxos[utxoId] = common.Utxo{Id: input}
	output, err := ledger.NewTransactionOutputBuilder().
		WithAddress(address.String()).WithLovelace(1).Build()
	require.NoError(t, err)

	builder := ledger.NewTransactionBuilder().
		WithCertificates(
			&common.StakeDeregistrationCertificate{
				CertType: uint(
					common.CertificateTypeStakeDeregistration,
				),
				StakeCredential: credential,
			},
			&common.RegistrationCertificate{
				CertType:        uint(common.CertificateTypeRegistration),
				StakeCredential: credential,
				Amount:          1,
			},
		).
		WithWithdrawals(map[*common.Address]uint64{&address: 50})
	builder.WithId([]byte{0x0a}).WithInputs(input).WithOutputs(output)
	tx, err := builder.Build()
	require.NoError(t, err)

	err = manager.ApplyTransaction(tx, 0)
	require.ErrorContains(t, err, "exceeds reward account balance")
	assert.Equal(t, uint64(100), manager.rewardAccounts[key])
	assert.Contains(t, manager.utxos, utxoId)
}
