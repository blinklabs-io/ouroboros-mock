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
	"encoding/hex"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/certificates"
	"github.com/stretchr/testify/require"
)

func wireHex(t *testing.T, cert lcommon.Certificate) string {
	t.Helper()
	wire, err := cbor.Encode(cert)
	require.NoError(t, err)
	return hex.EncodeToString(wire)
}

func TestGenesisKeyDelegationBuildsReferenceWire(t *testing.T) {
	t.Parallel()
	genesis := bytes.Repeat([]byte{0x01}, 28)
	delegate := bytes.Repeat([]byte{0x02}, 28)
	vrf := bytes.Repeat([]byte{0x03}, 32)
	cert, err := certificates.NewGenesisKeyDelegation().
		WithGenesisHash(genesis).
		WithGenesisDelegateHash(delegate).
		WithVrfKeyHash(vrf).
		Build()
	require.NoError(t, err)
	// genesis_key_delegation = (5, genesishash, genesis_delegate_hash,
	// vrf_keyhash)
	want := "8405581c" + hex.EncodeToString(genesis) +
		"581c" + hex.EncodeToString(delegate) +
		"5820" + hex.EncodeToString(vrf)
	require.Equal(t, want, wireHex(t, cert))

	var decoded lcommon.CertificateWrapper
	_, err = cbor.Decode(mustDecodeHex(t, want), &decoded)
	require.NoError(t, err)
	got, ok := decoded.Certificate.(*lcommon.GenesisKeyDelegationCertificate)
	require.True(t, ok, "decoded %T", decoded.Certificate)
	require.Equal(t, genesis, got.GenesisHash)
	require.Equal(t, delegate, got.GenesisDelegateHash)
	require.Equal(t, vrf, got.VrfKeyHash.Bytes())
}

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

func TestGenesisKeyDelegationRejectsBadHashes(t *testing.T) {
	t.Parallel()
	good := func() *certificates.GenesisKeyDelegationBuilder {
		return certificates.NewGenesisKeyDelegation().
			WithGenesisHash(make([]byte, 28)).
			WithGenesisDelegateHash(make([]byte, 28)).
			WithVrfKeyHash(make([]byte, 32))
	}
	_, err := good().Build()
	require.NoError(t, err)
	_, err = good().WithGenesisHash(make([]byte, 27)).Build()
	require.Error(t, err)
	_, err = good().WithGenesisDelegateHash(nil).Build()
	require.Error(t, err)
	_, err = good().WithVrfKeyHash(make([]byte, 28)).Build()
	require.Error(t, err)
}
