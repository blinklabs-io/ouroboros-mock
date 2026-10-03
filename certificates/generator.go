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

package certificates

import (
	"encoding/binary"
	"math/rand/v2"

	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

// Generator produces pseudo-random certificate inputs and certificates. Two
// generators created with the same seed produce the same sequence.
type Generator struct {
	source *rand.ChaCha8
	rng    *rand.Rand
}

// NewGenerator returns a generator seeded with seed.
func NewGenerator(seed uint64) *Generator {
	var chachaSeed [32]byte
	binary.LittleEndian.PutUint64(chachaSeed[:], seed)
	source := rand.NewChaCha8(chachaSeed)
	//nolint:gosec // deterministic test fixtures do not need cryptographic randomness.
	return &Generator{source: source, rng: rand.New(source)}
}

// KeyHash returns a random 28-byte hash, the width of a stake credential,
// pool, DRep or committee key hash.
func (g *Generator) KeyHash() []byte { return g.bytes(hashSize) }

func (g *Generator) bytes(size int) []byte {
	out := make([]byte, size)
	_, _ = g.source.Read(out)
	return out
}

func (g *Generator) amount() uint64 { return g.rng.Uint64N(1 << 40) }

// certificateKinds lists one random builder per supported certificate type.
var certificateKinds = []func(g *Generator) (lcommon.Certificate, error){
	func(g *Generator) (lcommon.Certificate, error) {
		return NewStakeRegistration().WithCredential(g.KeyHash()).Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewStakeDeregistration().WithCredential(g.KeyHash()).Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewStakeDelegation().
			WithCredential(g.KeyHash()).
			WithPoolKeyHash(g.KeyHash()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewPoolRegistration(lcommon.AddressNetworkTestnet).
			WithOperator(g.KeyHash()).
			WithVrfKeyHash(g.bytes(lcommon.Blake2b256Size)).
			WithPledge(g.amount()).
			WithCost(g.amount()).
			WithMargin(g.rng.Uint64N(100), 100).
			WithRewardAccountKey(g.KeyHash()).
			WithOwners(g.KeyHash()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewPoolRetirement().
			WithPoolKeyHash(g.KeyHash()).
			WithEpoch(g.rng.Uint64N(1 << 20)).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewGenesisKeyDelegation().
			WithGenesisHash(g.KeyHash()).
			WithGenesisDelegateHash(g.KeyHash()).
			WithVrfKeyHash(g.bytes(lcommon.Blake2b256Size)).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewRegistration().
			WithCredential(g.KeyHash()).
			WithDeposit(g.amount()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewDeregistration().
			WithCredential(g.KeyHash()).
			WithDeposit(g.amount()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewVoteDelegation().
			WithCredential(g.KeyHash()).
			WithDRepKeyHash(g.KeyHash()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewStakeVoteDelegation().
			WithCredential(g.KeyHash()).
			WithPoolKeyHash(g.KeyHash()).
			WithDRepKeyHash(g.KeyHash()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewStakeRegistrationDelegation().
			WithCredential(g.KeyHash()).
			WithPoolKeyHash(g.KeyHash()).
			WithDeposit(g.amount()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewVoteRegistrationDelegation().
			WithCredential(g.KeyHash()).
			WithDRepKeyHash(g.KeyHash()).
			WithDeposit(g.amount()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewStakeVoteRegistrationDelegation().
			WithCredential(g.KeyHash()).
			WithPoolKeyHash(g.KeyHash()).
			WithDRepKeyHash(g.KeyHash()).
			WithDeposit(g.amount()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewAuthCommitteeHot().
			WithColdCredential(g.KeyHash()).
			WithHotCredential(g.KeyHash()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewResignCommitteeCold().
			WithColdCredential(g.KeyHash()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewDRepRegistration().
			WithCredential(g.KeyHash()).
			WithDeposit(g.amount()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewDRepDeregistration().
			WithCredential(g.KeyHash()).
			WithDeposit(g.amount()).
			Build()
	},
	func(g *Generator) (lcommon.Certificate, error) {
		return NewDRepUpdate().WithCredential(g.KeyHash()).Build()
	},
}

// Certificate builds a random certificate of a random supported type.
func (g *Generator) Certificate() (lcommon.Certificate, error) {
	return certificateKinds[g.rng.IntN(len(certificateKinds))](g)
}
