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
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/blinklabs-io/ouroboros-mock/certificates"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
)

func TestMIRBuildersPreserveReferenceWire(t *testing.T) {
	t.Parallel()
	key := bytes.Repeat([]byte{0xab}, 28)
	huge := new(big.Int).Lsh(big.NewInt(1), 64)
	for _, tc := range []struct {
		name    string
		builder *certificates.MoveInstantaneousRewardsBuilder
		wire    string
	}{
		{
			"empty_rewards",
			certificates.NewMoveInstantaneousRewards(0).WithRewards(nil),
			"82068200a0",
		},
		{
			"treasury_transfer",
			certificates.NewMoveInstantaneousRewards(1).WithOtherPot(77),
			"82068201184d",
		},
		{
			"zero_transfer",
			certificates.NewMoveInstantaneousRewards(0).WithOtherPot(0),
			"8206820000",
		},
		{
			"key_negative",
			certificates.NewMoveInstantaneousRewards(0).
				WithRewardKey(key, big.NewInt(-1)),
			"82068200a18200581c" + strings.Repeat("ab", 28) + "20",
		},
		{
			"script_huge",
			certificates.NewMoveInstantaneousRewards(1).WithRewardScript(key, huge),
			"82068201a18201581c" + strings.Repeat("ab", 28) + "c249010000000000000000",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cert, err := tc.builder.Build()
			require.NoError(t, err)
			expected := mustDecodeHex(t, tc.wire)
			require.Equal(t, tc.wire, wireHex(t, cert))
			var wrapper lcommon.CertificateWrapper
			_, err = cbor.Decode(expected, &wrapper)
			require.NoError(t, err)
			require.Equal(t, uint(6), wrapper.Type)
			decodedValue := wrapper.Certificate
			decoded, ok := decodedValue.(*lcommon.MoveInstantaneousRewardsCertificate)
			require.True(t, ok)
			decoded.SetCbor(nil)
			for cred := range decoded.Reward.Rewards {
				cred.SetCbor(nil)
			}
			rebuilt, err := cbor.Encode(decoded)
			require.NoError(t, err)
			require.Equal(t, expected, rebuilt)
			require.Equal(t, tc.wire, wireHex(t, cert))
		})
	}
}

func TestMIRBuilderRejectsInvalidTargets(t *testing.T) {
	t.Parallel()
	key := bytes.Repeat([]byte{0xab}, 28)
	duplicate1 := &lcommon.Credential{
		CredType:   0,
		Credential: lcommon.NewBlake2b224(key),
	}
	duplicate2 := &lcommon.Credential{
		CredType:   0,
		Credential: lcommon.NewBlake2b224(key),
	}
	nilCredential := map[*lcommon.Credential]*big.Int{nil: big.NewInt(1)}
	invalidKind := map[*lcommon.Credential]*big.Int{
		{CredType: 2}: big.NewInt(1),
	}
	duplicates := map[*lcommon.Credential]*big.Int{
		duplicate1: big.NewInt(1),
		duplicate2: big.NewInt(2),
	}
	for _, tc := range []struct {
		name    string
		builder *certificates.MoveInstantaneousRewardsBuilder
	}{
		{"missing_target", certificates.NewMoveInstantaneousRewards(0)},
		{"invalid_pot", certificates.NewMoveInstantaneousRewards(2).WithOtherPot(0)},
		{
			"short_key",
			certificates.NewMoveInstantaneousRewards(0).
				WithRewardKey(key[:27], big.NewInt(1)),
		},
		{
			"nil_delta",
			certificates.NewMoveInstantaneousRewards(0).WithRewardKey(key, nil),
		},
		{
			"nil_credential",
			certificates.NewMoveInstantaneousRewards(0).WithRewards(nilCredential),
		},
		{
			"invalid_kind",
			certificates.NewMoveInstantaneousRewards(0).WithRewards(invalidKind),
		},
		{
			"duplicate",
			certificates.NewMoveInstantaneousRewards(0).WithRewards(duplicates),
		},
		{
			"repeated_key",
			certificates.NewMoveInstantaneousRewards(0).
				WithRewardKey(key, big.NewInt(1)).
				WithRewardKey(key, big.NewInt(2)),
		},
		{
			"mixed_zero_transfer",
			certificates.NewMoveInstantaneousRewards(0).WithRewards(nil).WithOtherPot(0),
		},
		{
			"mixed_rewards",
			certificates.NewMoveInstantaneousRewards(0).
				WithOtherPot(1).
				WithRewardScript(key, big.NewInt(-1)),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cert, err := tc.builder.Build()
			require.Error(t, err)
			require.Nil(t, cert)
			if tc.name == "duplicate" || tc.name == "repeated_key" {
				var duplicate lcommon.DuplicateLogicalMapKeyError
				require.ErrorAs(t, err, &duplicate)
			}
		})
	}
}

func TestMIRBuilderOwnsRewardInputsAndResults(t *testing.T) {
	t.Parallel()
	key := bytes.Repeat([]byte{0xab}, 28)
	delta := big.NewInt(-1)
	credential := &lcommon.Credential{
		CredType:   0,
		Credential: lcommon.NewBlake2b224(key),
	}
	rewards := map[*lcommon.Credential]*big.Int{credential: delta}
	builder := certificates.NewMoveInstantaneousRewards(0).WithRewards(rewards)
	credential.CredType = 1
	credential.Credential[0] = 0
	delta.SetInt64(77)
	clear(rewards)
	first, err := builder.Build()
	require.NoError(t, err)
	want := "82068200a18200581c" + strings.Repeat("ab", 28) + "20"
	require.Equal(t, want, wireHex(t, first))
	for c, d := range first.Reward.Rewards {
		c.CredType = 1
		c.Credential[0] = 0
		d.SetInt64(88)
	}
	second, err := builder.Build()
	require.NoError(t, err)
	require.Equal(t, want, wireHex(t, second))
	builder = certificates.NewMoveInstantaneousRewards(0).
		WithRewardKey(key, big.NewInt(-1))
	key[0] = 0
	third, err := builder.Build()
	require.NoError(t, err)
	require.Equal(t, want, wireHex(t, third))
}

func TestMIRCertificatesServeShelleyTransactionAndBlock(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		builder *certificates.MoveInstantaneousRewardsBuilder
		wire    string
	}{
		{
			"rewards",
			certificates.NewMoveInstantaneousRewards(0).
				WithRewardScript(bytes.Repeat([]byte{0xab}, 28), big.NewInt(-1)),
			"82068200a18201581c" + strings.Repeat("ab", 28) + "20",
		},
		{
			"transfer",
			certificates.NewMoveInstantaneousRewards(1).WithOtherPot(77),
			"82068201184d",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cert, err := tc.builder.Build()
			require.NoError(t, err)
			tx := &shelley.ShelleyTransaction{
				Body: shelley.ShelleyTransactionBody{
					TxInputs: shelley.NewShelleyTransactionInputSet(
						[]shelley.ShelleyTransactionInput{{}},
					),
					TxFee: 1,
					TxCertificates: []lcommon.CertificateWrapper{
						{Type: 6, Certificate: cert},
					},
				},
			}
			encoded, err := cbor.Encode(tx)
			require.NoError(t, err)
			decoded, err := shelley.NewShelleyTransactionFromCbor(encoded)
			require.NoError(t, err)
			require.Len(t, decoded.Certificates(), 1)
			require.Equal(
				t,
				mustDecodeHex(t, tc.wire),
				decoded.Certificates()[0].Cbor(),
			)
			block, err := fixtures.NewBlockBuilder(ledger.GetEraById(1)).
				WithTransactions(tx).
				Build()
			require.NoError(t, err)
			reloaded, err := ledger.NewBlockFromCbor(
				uint(block.Type()),
				block.Cbor(),
			)
			require.NoError(t, err)
			require.Len(t, reloaded.Transactions(), 1)
			got := reloaded.Transactions()[0].Certificates()
			require.Len(t, got, 1)
			require.Equal(t, mustDecodeHex(t, tc.wire), got[0].Cbor())
		})
	}
}
