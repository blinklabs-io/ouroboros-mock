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

package ledger

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/blinklabs-io/gouroboros/cbor"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

// PoolBuilder defines an interface for building mock pool registration certificates
type PoolBuilder interface {
	WithOperator(keyHash []byte) PoolBuilder
	WithVrfKeyHash(hash []byte) PoolBuilder
	WithPledge(lovelace uint64) PoolBuilder
	WithCost(lovelace uint64) PoolBuilder
	WithMargin(numerator, denominator uint64) PoolBuilder
	WithRewardAccountKey(keyHash []byte) PoolBuilder
	WithOwners(owners ...[]byte) PoolBuilder
	WithRelays(relays ...lcommon.PoolRelay) PoolBuilder
	WithMetadata(url string, hash []byte) PoolBuilder
	Build() (*lcommon.PoolRegistrationCertificate, error)
}

// MockPool holds the state for building a pool registration certificate
type MockPool struct {
	operator        lcommon.PoolKeyHash
	vrfKeyHash      lcommon.VrfKeyHash
	vrfKeyHashErr   error // tracks VRF key hash validation error
	pledge          uint64
	cost            uint64
	margin          cbor.Rat
	marginDenomZero bool // tracks if denominator was set to zero
	rewardAccount   lcommon.AddrKeyHash
	poolOwners      []lcommon.AddrKeyHash
	relays          []lcommon.PoolRelay
	poolMetadata    *lcommon.PoolMetadata
	metadataHashErr error // tracks metadata hash validation error
}

// NewPoolBuilder creates a new MockPool builder
func NewPoolBuilder() *MockPool {
	return &MockPool{
		margin: cbor.Rat{Rat: big.NewRat(0, 1)},
	}
}

// WithOperator sets the pool operator key hash
func (p *MockPool) WithOperator(keyHash []byte) PoolBuilder {
	p.operator = lcommon.NewBlake2b224(keyHash)
	return p
}

// WithVrfKeyHash sets the VRF key hash
// If hash length is not exactly 32 bytes, Build() will return an error
func (p *MockPool) WithVrfKeyHash(hash []byte) PoolBuilder {
	if len(hash) != len(p.vrfKeyHash) {
		p.vrfKeyHashErr = fmt.Errorf(
			"VRF key hash must be exactly %d bytes, got %d",
			len(p.vrfKeyHash),
			len(hash),
		)
		return p
	}
	// Clear any previous error when valid hash is provided
	p.vrfKeyHashErr = nil
	copy(p.vrfKeyHash[:], hash)
	return p
}

// WithPledge sets the pool pledge amount in lovelace
func (p *MockPool) WithPledge(lovelace uint64) PoolBuilder {
	p.pledge = lovelace
	return p
}

// WithCost sets the pool fixed cost in lovelace
func (p *MockPool) WithCost(lovelace uint64) PoolBuilder {
	p.cost = lovelace
	return p
}

// WithMargin sets the pool margin as a ratio (numerator/denominator)
// If denominator is zero, Build() will return an error
func (p *MockPool) WithMargin(numerator, denominator uint64) PoolBuilder {
	if denominator == 0 {
		p.marginDenomZero = true
		return p
	}
	// Clear any previous error flag when valid denominator is provided
	p.marginDenomZero = false
	// Use big.Int to avoid int64 overflow for large uint64 values
	num := new(big.Int).SetUint64(numerator)
	denom := new(big.Int).SetUint64(denominator)
	p.margin = cbor.Rat{Rat: new(big.Rat).SetFrac(num, denom)}
	return p
}

// WithRewardAccountKey sets the pool reward account key hash
// The keyHash should be a Blake2b224 hash (28 bytes)
func (p *MockPool) WithRewardAccountKey(keyHash []byte) PoolBuilder {
	p.rewardAccount = lcommon.NewBlake2b224(keyHash)
	return p
}

// WithOwners sets the pool owners
func (p *MockPool) WithOwners(owners ...[]byte) PoolBuilder {
	p.poolOwners = make([]lcommon.AddrKeyHash, len(owners))
	for i, owner := range owners {
		p.poolOwners[i] = lcommon.NewBlake2b224(owner)
	}
	return p
}

// WithRelays sets the pool relays
func (p *MockPool) WithRelays(relays ...lcommon.PoolRelay) PoolBuilder {
	p.relays = relays
	return p
}

// WithMetadata sets the pool metadata URL and hash
// If hash length is not exactly 32 bytes, Build() will return an error
func (p *MockPool) WithMetadata(url string, hash []byte) PoolBuilder {
	var metadataHash lcommon.PoolMetadataHash
	if len(hash) != len(metadataHash) {
		p.metadataHashErr = fmt.Errorf(
			"pool metadata hash must be exactly %d bytes, got %d",
			len(metadataHash),
			len(hash),
		)
		return p
	}
	// Clear any previous error when valid hash is provided
	p.metadataHashErr = nil
	copy(metadataHash[:], hash)
	p.poolMetadata = &lcommon.PoolMetadata{
		Url:  url,
		Hash: metadataHash,
	}
	return p
}

// Build constructs a PoolRegistrationCertificate from the builder state
func (p *MockPool) Build() (*lcommon.PoolRegistrationCertificate, error) {
	// Return any stored validation errors
	if p.vrfKeyHashErr != nil {
		return nil, p.vrfKeyHashErr
	}
	if p.metadataHashErr != nil {
		return nil, p.metadataHashErr
	}
	// Validate margin denominator was not zero
	if p.marginDenomZero {
		return nil, errors.New("pool margin denominator cannot be zero")
	}

	cert := &lcommon.PoolRegistrationCertificate{
		CertType:      3, // Pool registration certificate type
		Operator:      p.operator,
		VrfKeyHash:    p.vrfKeyHash,
		Pledge:        p.pledge,
		Cost:          p.cost,
		Margin:        p.margin,
		RewardAccount: p.rewardAccount,
		PoolOwners:    p.poolOwners,
		Relays:        p.relays,
		PoolMetadata:  p.poolMetadata,
	}
	return cert, nil
}
