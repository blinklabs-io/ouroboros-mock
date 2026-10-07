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

package certificates_test

import (
	"bytes"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/blinklabs-io/ouroboros-mock/certificates"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
)

func TestGeneratorIsDeterministicPerSeed(t *testing.T) {
	t.Parallel()
	draw := func(seed uint64) [][]byte {
		g := certificates.NewGenerator(seed)
		var wires [][]byte
		for range 50 {
			cert, err := g.Certificate()
			require.NoError(t, err)
			wire, err := cbor.Encode(cert)
			require.NoError(t, err)
			wires = append(wires, wire)
		}
		return wires
	}
	require.Equal(t, draw(7), draw(7))
	require.NotEqual(t, draw(7), draw(8))
}

func TestGeneratorBuildsEveryEncodableCertificateType(t *testing.T) {
	t.Parallel()
	g := certificates.NewGenerator(1)
	seen := map[uint]bool{}
	for range 2000 {
		cert, err := g.Certificate()
		require.NoError(t, err)
		wire, err := cbor.Encode(cert)
		require.NoError(t, err)
		var decoded lcommon.CertificateWrapper
		_, err = cbor.Decode(wire, &decoded)
		require.NoError(t, err, "certificate type %d", cert.Type())
		require.Equal(t, cert.Type(), decoded.Type)
		decodedWire, err := cbor.Encode(decoded.Certificate)
		require.NoError(t, err)
		require.Equal(t, wire, decodedWire, "certificate type %d", cert.Type())
		seen[cert.Type()] = true
	}
	want := map[uint]bool{}
	firstKind := lcommon.CertificateTypeStakeRegistration
	lastKind := lcommon.CertificateTypeUpdateDrep
	for kind := firstKind; kind <= lastKind; kind++ {
		want[uint(kind)] = true
	}
	require.Equal(t, want, seen)
}

func TestGeneratorKeyHashesHaveCredentialWidth(t *testing.T) {
	t.Parallel()
	g := certificates.NewGenerator(3)
	first, second := g.KeyHash(), g.KeyHash()
	require.Len(t, first, lcommon.Blake2b224Size)
	require.NotEqual(t, first, second)
}

func TestStakeLifecycleRegistersDelegatesAndRefunds(t *testing.T) {
	t.Parallel()
	certs, err := certificates.StakeLifecycle(stakeHash, poolHash, 2_000_000)
	require.NoError(t, err)
	require.Len(t, certs, 3)
	registration, ok := certs[0].(*lcommon.RegistrationCertificate)
	require.True(t, ok, "%T", certs[0])
	delegation, ok := certs[1].(*lcommon.StakeDelegationCertificate)
	require.True(t, ok, "%T", certs[1])
	deregistration, ok := certs[2].(*lcommon.DeregistrationCertificate)
	require.True(t, ok, "%T", certs[2])
	require.Equal(t, int64(2_000_000), registration.Amount)
	require.Equal(t, registration.Amount, deregistration.Amount)
	require.Equal(t, registration.StakeCredential, *delegation.StakeCredential)
	require.Equal(
		t,
		registration.StakeCredential,
		deregistration.StakeCredential,
	)
	require.Equal(t, poolHash, delegation.PoolKeyHash.Bytes())
	_, err = certificates.StakeLifecycle([]byte{1}, poolHash, 1)
	require.Error(t, err)
}

func TestDRepLifecycleRegistersUpdatesAndRefunds(t *testing.T) {
	t.Parallel()
	certs, err := certificates.DRepLifecycle(stakeHash, 500)
	require.NoError(t, err)
	require.Len(t, certs, 3)
	registration, ok := certs[0].(*lcommon.RegistrationDrepCertificate)
	require.True(t, ok, "%T", certs[0])
	update, ok := certs[1].(*lcommon.UpdateDrepCertificate)
	require.True(t, ok, "%T", certs[1])
	deregistration, ok := certs[2].(*lcommon.DeregistrationDrepCertificate)
	require.True(t, ok, "%T", certs[2])
	require.Equal(t, int64(500), registration.Amount)
	require.Equal(t, registration.Amount, deregistration.Amount)
	require.Equal(t, registration.DrepCredential, update.DrepCredential)
	require.Equal(t, registration.DrepCredential, deregistration.DrepCredential)
}

func TestPoolLifecycleRegistersAndRetires(t *testing.T) {
	t.Parallel()
	certs, err := certificates.PoolLifecycle(
		lcommon.AddressNetworkTestnet, poolHash, vrfHash, 99,
	)
	require.NoError(t, err)
	require.Len(t, certs, 2)
	registration, ok := certs[0].(*lcommon.PoolRegistrationCertificate)
	require.True(t, ok, "%T", certs[0])
	retirement, ok := certs[1].(*lcommon.PoolRetirementCertificate)
	require.True(t, ok, "%T", certs[1])
	require.Equal(t, registration.Operator, retirement.PoolKeyHash)
	require.Equal(t, uint64(99), retirement.Epoch)
}

// TestLifecycleCertificatesSurviveAConwayBlock puts certificates in a
// transaction of a generated block and reads them back from the decoded
// block.
func TestLifecycleCertificatesSurviveAConwayBlock(t *testing.T) {
	t.Parallel()
	certs, err := certificates.StakeLifecycle(stakeHash, poolHash, 2_000_000)
	require.NoError(t, err)
	wrapped := make([]lcommon.CertificateWrapper, len(certs))
	for i, cert := range certs {
		wrapped[i] = lcommon.CertificateWrapper{
			Type:        cert.Type(),
			Certificate: cert,
		}
	}
	tx := &conway.ConwayTransaction{
		Body: conway.ConwayTransactionBody{
			TxInputs: conway.NewConwayTransactionInputSet(
				[]shelley.ShelleyTransactionInput{{}},
			),
			TxFee:          1,
			TxCertificates: wrapped,
		},
		TxIsValid: true,
	}
	block, err := fixtures.NewBlockBuilder(ledger.GetEraById(6)).
		WithTransactions(tx).Build()
	require.NoError(t, err)
	decoded := block.Transactions()
	require.Len(t, decoded, 1)
	got := decoded[0].Certificates()
	require.Len(t, got, len(certs))
	for i, cert := range got {
		require.Equal(t, certs[i].Type(), cert.Type(), "certificate %d", i)
		require.True(t, bytes.Equal(mustEncode(t, certs[i]), cert.Cbor()))
	}
}

func mustEncode(t *testing.T, v any) []byte {
	t.Helper()
	wire, err := cbor.Encode(v)
	require.NoError(t, err)
	return wire
}

func TestGeneratorSelectsPoolRewardNetwork(t *testing.T) {
	t.Parallel()
	networks := []uint{
		lcommon.AddressNetworkTestnet,
		lcommon.AddressNetworkMainnet,
	}
	for _, network := range networks {
		generator := certificates.NewGenerator(7).WithNetwork(network)
		seen := false
		for range 2000 {
			cert, err := generator.Certificate()
			require.NoError(t, err)
			if pool, ok := cert.(*lcommon.PoolRegistrationCertificate); ok {
				wire, err := cbor.Encode(pool)
				require.NoError(t, err)
				var decoded lcommon.PoolRegistrationCertificate
				_, err = cbor.Decode(wire, &decoded)
				require.NoError(t, err)
				gotNetwork, ok := decoded.RewardAccountNetworkId()
				require.True(t, ok)
				require.Equal(t, network, gotNetwork)
				seen = true
			}
		}
		require.True(t, seen)
	}
}

func TestGeneratorIncludesBothMIRTargets(t *testing.T) {
	t.Parallel()
	generator := certificates.NewGenerator(7)
	seenMap, seenTransfer := false, false
	for range 2000 {
		cert, err := generator.Certificate()
		require.NoError(t, err)
		if mir, ok := cert.(*lcommon.MoveInstantaneousRewardsCertificate); ok {
			require.LessOrEqual(t, mir.Reward.Source, uint(1))
			if mir.Reward.Rewards == nil {
				seenTransfer = true
			} else {
				seenMap = true
			}
		}
	}
	require.True(t, seenMap)
	require.True(t, seenTransfer)
}

func TestPoolLifecycleEncodesEmptyCollectionsAsArrays(t *testing.T) {
	certs, err := certificates.PoolLifecycle(0, poolHash, vrfHash, 99)
	require.NoError(t, err)
	assertEmptyPoolCollections(t, certs[0])
}

func TestPoolBuilderPreservesExplicitEmptyCollections(t *testing.T) {
	cert, err := certificates.NewPoolRegistration(0).
		WithOperator(poolHash).
		WithVrfKeyHash(vrfHash).
		WithRewardAccountKey(stakeHash).
		WithOwners(stakeHash).
		WithRelays(lcommon.PoolRelay{Type: 2}).
		WithOwners().
		WithRelays().
		Build()
	require.NoError(t, err)
	assertEmptyPoolCollections(t, cert)
}

func assertEmptyPoolCollections(t *testing.T, cert lcommon.Certificate) {
	t.Helper()
	wire, err := cbor.Encode(cert)
	require.NoError(t, err)
	var fields []cbor.RawMessage
	_, err = cbor.Decode(wire, &fields)
	require.NoError(t, err)
	require.Len(t, fields, 10)
	require.Equal(t, cbor.RawMessage{0x80}, fields[7], "empty owner set")
	require.Equal(t, cbor.RawMessage{0x80}, fields[8], "empty relay list")
}
