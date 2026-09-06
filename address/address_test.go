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

package address

import (
	"bytes"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func TestAddressBuilderTypes(t *testing.T) {
	hash := bytes.Repeat([]byte{0x11}, common.AddressHashSize)
	cases := []struct {
		name  string
		build func() (common.Address, error)
		want  uint8
	}{
		{"base key-key", func() (common.Address, error) {
			return NewAddress().WithMainnet().WithPaymentKeyHash(hash).WithStakingKeyHash(hash).Build()
		}, common.AddressTypeKeyKey},
		{"base script-key", func() (common.Address, error) {
			return NewAddress().WithPaymentScript(hash).WithStakingKeyHash(hash).Build()
		}, common.AddressTypeScriptKey},
		{"base key-script", func() (common.Address, error) {
			return NewAddress().WithPaymentKeyHash(hash).WithStakingScript(hash).Build()
		}, common.AddressTypeKeyScript},
		{"base script-script", func() (common.Address, error) {
			return NewAddress().WithPaymentScript(hash).WithStakingScript(hash).Build()
		}, common.AddressTypeScriptScript},
		{"enterprise key", func() (common.Address, error) { return NewAddress().WithPaymentKeyHash(hash).WithNoStaking().Build() }, common.AddressTypeKeyNone},
		{"enterprise script", func() (common.Address, error) { return NewAddress().WithPaymentScript(hash).WithNoStaking().Build() }, common.AddressTypeScriptNone},
		{"reward key", func() (common.Address, error) {
			return NewAddress().WithMainnet().WithStakingKeyHash(hash).BuildReward()
		}, common.AddressTypeNoneKey},
		{"reward script", func() (common.Address, error) { return NewAddress().WithStakingScript(hash).BuildReward() }, common.AddressTypeNoneScript},
		{"pointer key", func() (common.Address, error) {
			return NewAddress().WithPaymentKeyHash(hash).WithStakePointer(300, 2, 1).Build()
		}, common.AddressTypeKeyPointer},
		{"pointer script", func() (common.Address, error) {
			return NewAddress().WithPaymentScript(hash).WithStakePointer(300, 2, 1).Build()
		}, common.AddressTypeScriptPointer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			addr, err := tc.build()
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			if addr.Type() != tc.want {
				t.Fatalf("type = %d, want %d", addr.Type(), tc.want)
			}
			parsed, err := ParseAddress(addr.String())
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got, err := parsed.Bytes()
			if err != nil {
				t.Fatalf("bytes: %v", err)
			}
			want, err := addr.Bytes()
			if err != nil {
				t.Fatalf("expected bytes: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("round trip changed bytes: %x != %x", got, want)
			}
		})
	}
}

func TestAddressHashHelpers(t *testing.T) {
	data := []byte("test key")
	if got, want := PaymentKeyHash(data), common.Blake2b224Hash(data); got != want {
		t.Fatal("payment hash mismatch")
	}
	if got, want := StakingKeyHash(data), common.Blake2b224Hash(data); got != want {
		t.Fatal("staking hash mismatch")
	}
	if got, want := ScriptHash(data), common.Blake2b224Hash(data); got != want {
		t.Fatal("script hash mismatch")
	}
}

func TestDeterministicAddresses(t *testing.T) {
	one, err := RandomBaseWithRand(NewRand(42), Mainnet)
	if err != nil {
		t.Fatalf("first address: %v", err)
	}
	two, err := RandomBaseWithRand(NewRand(42), Mainnet)
	if err != nil {
		t.Fatalf("second address: %v", err)
	}
	if one.String() != two.String() {
		t.Fatalf("seeded addresses differ: %s != %s", one, two)
	}
}

func TestByronAddressBuilder(t *testing.T) {
	hash := bytes.Repeat([]byte{0x22}, common.AddressHashSize)
	builder := NewByronAddress().WithPaymentKeyHash(hash).WithNetworkId(42)
	built, err := builder.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	value, err := builder.BuildBase58()
	if err != nil {
		t.Fatalf("build base58: %v", err)
	}
	parsed, err := ParseAddress(value)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Type() != common.AddressTypeByron {
		t.Fatalf("type = %d, want Byron", parsed.Type())
	}
	builtBytes, err := built.Bytes()
	if err != nil {
		t.Fatalf("built bytes: %v", err)
	}
	parsedBytes, err := parsed.Bytes()
	if err != nil {
		t.Fatalf("parsed bytes: %v", err)
	}
	if !bytes.Equal(parsedBytes, builtBytes) {
		t.Fatalf("round trip changed bytes: %x != %x", parsedBytes, builtBytes)
	}
	if got := parsed.PaymentKeyHash().Bytes(); !bytes.Equal(got, hash) {
		t.Fatalf("payment hash = %x, want %x", got, hash)
	}
	attr := parsed.ByronAttr()
	if attr.Network == nil || *attr.Network != 42 {
		t.Fatalf("network = %v, want 42", attr.Network)
	}
}

func TestAddressBuilderValidation(t *testing.T) {
	if _, err := NewAddress().WithPaymentKeyHash([]byte{1}).WithNoStaking().Build(); err == nil {
		t.Fatal("expected short payment hash error")
	}
	if _, err := NewAddress().WithNetworkId(2).WithPaymentKeyHash(make([]byte, common.AddressHashSize)).WithNoStaking().Build(); err == nil {
		t.Fatal("expected unsupported network error")
	}
	if _, err := NewAddress().WithPaymentKeyHash(make([]byte, common.AddressHashSize)).Build(); err == nil {
		t.Fatal("expected missing staking credential error")
	} else if err.Error() != "staking credential is required" {
		t.Fatalf("missing staking credential error = %q", err)
	}
	if _, err := NewAddress().WithPaymentKeyHash(make([]byte, common.AddressHashSize)).WithStakingKeyHash(make([]byte, common.AddressHashSize-1)).Build(); err == nil {
		t.Fatal("expected wrong-length staking hash error")
	} else if err.Error() != "staking hash must be 28 bytes" {
		t.Fatalf("wrong-length staking hash error = %q", err)
	}
}
