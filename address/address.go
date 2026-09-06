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

// Package address provides small Cardano address builders for test data.
package address

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	mathrand "math/rand"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

const (
	Mainnet = common.AddressNetworkMainnet
	Testnet = common.AddressNetworkTestnet
)

// AddressBuilder constructs Shelley-era addresses from credential hashes.
type AddressBuilder struct {
	networkID     uint8
	payment       []byte
	paymentScript bool
	stake         []byte
	stakeScript   bool
	noStake       bool
	pointer       *common.AddressPayloadPointer
}

// NewAddress returns a builder using the testnet network ID.
func NewAddress() *AddressBuilder { return &AddressBuilder{networkID: Testnet} }

func (b *AddressBuilder) WithMainnet() *AddressBuilder           { return b.WithNetworkId(Mainnet) }
func (b *AddressBuilder) WithTestnet() *AddressBuilder           { return b.WithNetworkId(Testnet) }
func (b *AddressBuilder) WithNetworkId(id uint8) *AddressBuilder { b.networkID = id; return b }

func (b *AddressBuilder) WithPaymentKeyHash(hash []byte) *AddressBuilder {
	b.payment, b.paymentScript = clone(hash), false
	return b
}
func (b *AddressBuilder) WithPaymentScript(hash []byte) *AddressBuilder {
	b.payment, b.paymentScript = clone(hash), true
	return b
}
func (b *AddressBuilder) WithStakingKeyHash(hash []byte) *AddressBuilder {
	b.stake, b.stakeScript, b.noStake, b.pointer = clone(hash), false, false, nil
	return b
}
func (b *AddressBuilder) WithStakingScript(hash []byte) *AddressBuilder {
	b.stake, b.stakeScript, b.noStake, b.pointer = clone(hash), true, false, nil
	return b
}
func (b *AddressBuilder) WithStakePointer(slot, txIndex, certIndex uint64) *AddressBuilder {
	b.stake, b.noStake = nil, false
	b.pointer = &common.AddressPayloadPointer{Slot: slot, TxIndex: txIndex, CertIndex: certIndex}
	return b
}
func (b *AddressBuilder) WithNoStaking() *AddressBuilder {
	b.stake, b.noStake, b.pointer = nil, true, nil
	return b
}

// Build constructs and validates the address using gouroboros' address parser.
func (b *AddressBuilder) Build() (common.Address, error) {
	if len(b.payment) != common.AddressHashSize {
		return common.Address{}, fmt.Errorf("payment hash must be %d bytes", common.AddressHashSize)
	}
	if b.networkID != Testnet && b.networkID != Mainnet {
		return common.Address{}, fmt.Errorf("unsupported network ID: %d", b.networkID)
	}
	var addrType uint8
	switch {
	case b.pointer != nil:
		addrType = common.AddressTypeKeyPointer
		if b.paymentScript {
			addrType = common.AddressTypeScriptPointer
		}
		payload := append(clone(b.payment), encodePointer(*b.pointer)...)
		return common.NewAddressFromBytes(append([]byte{addrType<<4 | b.networkID}, payload...))
	case b.noStake:
		addrType = common.AddressTypeKeyNone
		if b.paymentScript {
			addrType = common.AddressTypeScriptNone
		}
		return common.NewAddressFromParts(addrType, b.networkID, b.payment, nil)
	case len(b.stake) == common.AddressHashSize:
		switch {
		case b.paymentScript && b.stakeScript:
			addrType = common.AddressTypeScriptScript
		case b.paymentScript:
			addrType = common.AddressTypeScriptKey
		case b.stakeScript:
			addrType = common.AddressTypeKeyScript
		default:
			addrType = common.AddressTypeKeyKey
		}
		return common.NewAddressFromParts(addrType, b.networkID, b.payment, b.stake)
	case b.stake == nil:
		return common.Address{}, errors.New("staking credential is required")
	default:
		return common.Address{}, fmt.Errorf("staking hash must be %d bytes", common.AddressHashSize)
	}
}

func (b *AddressBuilder) BuildBech32() (string, error) {
	addr, err := b.Build()
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}
func (b *AddressBuilder) BuildBytes() ([]byte, error) {
	addr, err := b.Build()
	if err != nil {
		return nil, err
	}
	return addr.Bytes()
}

// ByronAddressBuilder constructs legacy Byron addresses.
type ByronAddressBuilder struct {
	network  *uint32
	hash     []byte
	addrType uint64
}

func NewByronAddress() *ByronAddressBuilder { return &ByronAddressBuilder{} }
func (b *ByronAddressBuilder) WithPaymentKeyHash(hash []byte) *ByronAddressBuilder {
	b.hash = clone(hash)
	return b
}
func (b *ByronAddressBuilder) WithNetworkId(id uint32) *ByronAddressBuilder {
	b.network = &id
	return b
}
func (b *ByronAddressBuilder) WithAddressType(addrType uint64) *ByronAddressBuilder {
	b.addrType = addrType
	return b
}
func (b *ByronAddressBuilder) Build() (common.Address, error) {
	if len(b.hash) != common.AddressHashSize {
		return common.Address{}, fmt.Errorf("payment hash must be %d bytes", common.AddressHashSize)
	}
	return common.NewByronAddressFromParts(b.addrType, b.hash, common.ByronAddressAttributes{Network: b.network})
}
func (b *ByronAddressBuilder) BuildBase58() (string, error) {
	addr, err := b.Build()
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}

func PaymentKeyHash(publicKey []byte) common.Blake2b224 { return common.Blake2b224Hash(publicKey) }
func StakingKeyHash(publicKey []byte) common.Blake2b224 { return common.Blake2b224Hash(publicKey) }
func ScriptHash(script []byte) common.Blake2b224        { return common.Blake2b224Hash(script) }

func ParseAddress(value string) (common.Address, error) { return common.NewAddress(value) }
func MustParseAddress(value string) common.Address {
	addr, err := ParseAddress(value)
	if err != nil {
		panic(err)
	}
	return addr
}

func RandomMainnet() (common.Address, error) { return randomAddress(Mainnet, cryptorand.Reader) }
func RandomTestnet() (common.Address, error) { return randomAddress(Testnet, cryptorand.Reader) }
func RandomBase(networkID uint8) (common.Address, error) {
	return randomAddress(networkID, cryptorand.Reader)
}
func RandomEnterprise(networkID uint8) (common.Address, error) {
	hash, err := randomHash(cryptorand.Reader)
	if err != nil {
		return common.Address{}, err
	}
	return NewAddress().WithNetworkId(networkID).WithPaymentKeyHash(hash).WithNoStaking().Build()
}
func RandomReward(networkID uint8) (common.Address, error) {
	hash, err := randomHash(cryptorand.Reader)
	if err != nil {
		return common.Address{}, err
	}
	return NewAddress().WithNetworkId(networkID).WithStakingKeyHash(hash).BuildReward()
}

// NewRand returns a deterministic source for the Random*WithRand helpers.
func NewRand(seed int64) *mathrand.Rand { return mathrand.New(mathrand.NewSource(seed)) }
func RandomBaseWithRand(r *mathrand.Rand, networkID uint8) (common.Address, error) {
	return randomAddress(networkID, r)
}
func RandomEnterpriseWithRand(r *mathrand.Rand, networkID uint8) (common.Address, error) {
	hash, err := randomHash(r)
	if err != nil {
		return common.Address{}, err
	}
	return NewAddress().WithNetworkId(networkID).WithPaymentKeyHash(hash).WithNoStaking().Build()
}
func RandomRewardWithRand(r *mathrand.Rand, networkID uint8) (common.Address, error) {
	hash, err := randomHash(r)
	if err != nil {
		return common.Address{}, err
	}
	return NewAddress().WithNetworkId(networkID).WithStakingKeyHash(hash).BuildReward()
}

func (b *AddressBuilder) BuildReward() (common.Address, error) {
	if len(b.stake) != common.AddressHashSize {
		return common.Address{}, fmt.Errorf("staking hash must be %d bytes", common.AddressHashSize)
	}
	var typ uint8 = common.AddressTypeNoneKey
	if b.stakeScript {
		typ = common.AddressTypeNoneScript
	}
	return common.NewAddressFromParts(typ, b.networkID, nil, b.stake)
}

func randomAddress(networkID uint8, r interface{ Read([]byte) (int, error) }) (common.Address, error) {
	payment, err := randomHash(r)
	if err != nil {
		return common.Address{}, err
	}
	stake, err := randomHash(r)
	if err != nil {
		return common.Address{}, err
	}
	return NewAddress().WithNetworkId(networkID).WithPaymentKeyHash(payment).WithStakingKeyHash(stake).Build()
}
func randomHash(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	hash := make([]byte, common.AddressHashSize)
	_, err := r.Read(hash)
	return hash, err
}
func clone(value []byte) []byte { return append([]byte(nil), value...) }
func encodePointer(pointer common.AddressPayloadPointer) []byte {
	var ret []byte
	for _, value := range []uint64{pointer.Slot, pointer.TxIndex, pointer.CertIndex} {
		var buf [10]byte
		n := binary.PutUvarint(buf[:], value)
		ret = append(ret, buf[:n]...)
	}
	return ret
}
