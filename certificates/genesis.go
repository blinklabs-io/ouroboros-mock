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

package certificates

import (
	"fmt"

	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

// GenesisKeyDelegationBuilder builds a genesis key delegation certificate.
type GenesisKeyDelegationBuilder struct {
	genesisHash         []byte
	genesisDelegateHash []byte
	vrfKeyHash          []byte
}

// NewGenesisKeyDelegation returns an empty genesis key delegation builder.
func NewGenesisKeyDelegation() *GenesisKeyDelegationBuilder {
	return &GenesisKeyDelegationBuilder{}
}

// WithGenesisHash sets the hash of the genesis key being delegated.
func (b *GenesisKeyDelegationBuilder) WithGenesisHash(
	hash []byte,
) *GenesisKeyDelegationBuilder {
	b.genesisHash = append([]byte(nil), hash...)
	return b
}

// WithGenesisDelegateHash sets the hash of the delegate's key.
func (b *GenesisKeyDelegationBuilder) WithGenesisDelegateHash(
	hash []byte,
) *GenesisKeyDelegationBuilder {
	b.genesisDelegateHash = append([]byte(nil), hash...)
	return b
}

// WithVrfKeyHash sets the delegate's VRF key hash.
func (b *GenesisKeyDelegationBuilder) WithVrfKeyHash(
	hash []byte,
) *GenesisKeyDelegationBuilder {
	b.vrfKeyHash = append([]byte(nil), hash...)
	return b
}

// Build returns the certificate, or an error when a hash has the wrong
// length.
func (b *GenesisKeyDelegationBuilder) Build() (*lcommon.GenesisKeyDelegationCertificate, error) {
	switch {
	case len(b.genesisHash) != hashSize:
		return nil, fmt.Errorf(
			"genesis hash must be exactly %d bytes", hashSize,
		)
	case len(b.genesisDelegateHash) != hashSize:
		return nil, fmt.Errorf(
			"genesis delegate hash must be exactly %d bytes", hashSize,
		)
	case len(b.vrfKeyHash) != lcommon.Blake2b256Size:
		return nil, fmt.Errorf(
			"VRF key hash must be exactly %d bytes", lcommon.Blake2b256Size,
		)
	}
	return &lcommon.GenesisKeyDelegationCertificate{
		CertType:            uint(lcommon.CertificateTypeGenesisKeyDelegation),
		GenesisHash:         append([]byte(nil), b.genesisHash...),
		GenesisDelegateHash: append([]byte(nil), b.genesisDelegateHash...),
		VrfKeyHash:          lcommon.NewBlake2b256(b.vrfKeyHash),
	}, nil
}
