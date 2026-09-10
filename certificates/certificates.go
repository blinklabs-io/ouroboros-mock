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

// Package certificates provides builders for gouroboros certificate values.
package certificates

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/blinklabs-io/gouroboros/cbor"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

const hashSize = lcommon.Blake2b224Size

func credential(kind uint, hash []byte) (lcommon.Credential, error) {
	if len(hash) != hashSize {
		return lcommon.Credential{}, fmt.Errorf(
			"credential hash must be exactly %d bytes, got %d",
			hashSize, len(hash),
		)
	}
	return lcommon.Credential{
		CredType:   kind,
		Credential: lcommon.NewBlake2b224(hash),
	}, nil
}

func keyCredential(hash []byte) (lcommon.Credential, error) {
	return credential(lcommon.CredentialTypeAddrKeyHash, hash)
}

func scriptCredential(hash []byte) (lcommon.Credential, error) {
	return credential(lcommon.CredentialTypeScriptHash, hash)
}

type stakeBuilder struct {
	credential    lcommon.Credential
	credentialErr error
	credentialSet bool
}

func (b *stakeBuilder) WithCredential(hash []byte) *stakeBuilder {
	b.credential, b.credentialErr = keyCredential(hash)
	b.credentialSet = true
	return b
}

func (b *stakeBuilder) WithScriptCredential(hash []byte) *stakeBuilder {
	b.credential, b.credentialErr = scriptCredential(hash)
	b.credentialSet = true
	return b
}

func (b *stakeBuilder) validate() error {
	if !b.credentialSet {
		return errors.New("stake credential is required")
	}
	return b.credentialErr
}

// StakeRegistrationBuilder builds a Shelley stake registration certificate.
type StakeRegistrationBuilder struct{ stakeBuilder }

func NewStakeRegistration() *StakeRegistrationBuilder {
	return &StakeRegistrationBuilder{}
}

func (b *StakeRegistrationBuilder) WithCredential(hash []byte) *StakeRegistrationBuilder {
	b.stakeBuilder.WithCredential(hash)
	return b
}

func (b *StakeRegistrationBuilder) WithScriptCredential(hash []byte) *StakeRegistrationBuilder {
	b.stakeBuilder.WithScriptCredential(hash)
	return b
}

func (b *StakeRegistrationBuilder) Build() (*lcommon.StakeRegistrationCertificate, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	return &lcommon.StakeRegistrationCertificate{
		CertType:        uint(lcommon.CertificateTypeStakeRegistration),
		StakeCredential: b.credential,
	}, nil
}

// StakeDeregistrationBuilder builds a Shelley stake deregistration certificate.
type StakeDeregistrationBuilder struct{ stakeBuilder }

func NewStakeDeregistration() *StakeDeregistrationBuilder {
	return &StakeDeregistrationBuilder{}
}

func (b *StakeDeregistrationBuilder) WithCredential(hash []byte) *StakeDeregistrationBuilder {
	b.stakeBuilder.WithCredential(hash)
	return b
}

func (b *StakeDeregistrationBuilder) WithScriptCredential(hash []byte) *StakeDeregistrationBuilder {
	b.stakeBuilder.WithScriptCredential(hash)
	return b
}

func (b *StakeDeregistrationBuilder) Build() (*lcommon.StakeDeregistrationCertificate, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	return &lcommon.StakeDeregistrationCertificate{
		CertType:        uint(lcommon.CertificateTypeStakeDeregistration),
		StakeCredential: b.credential,
	}, nil
}

// StakeDelegationBuilder builds a Shelley stake delegation certificate.
type StakeDelegationBuilder struct {
	stakeBuilder
	poolKeyHash lcommon.PoolKeyHash
	poolSet     bool
}

func NewStakeDelegation() *StakeDelegationBuilder {
	return &StakeDelegationBuilder{}
}

func (b *StakeDelegationBuilder) WithCredential(hash []byte) *StakeDelegationBuilder {
	b.stakeBuilder.WithCredential(hash)
	return b
}

func (b *StakeDelegationBuilder) WithScriptCredential(hash []byte) *StakeDelegationBuilder {
	b.stakeBuilder.WithScriptCredential(hash)
	return b
}

func (b *StakeDelegationBuilder) WithPoolKeyHash(hash []byte) *StakeDelegationBuilder {
	b.poolKeyHash = lcommon.NewBlake2b224(hash)
	b.poolSet = len(hash) == hashSize
	return b
}

func (b *StakeDelegationBuilder) Build() (*lcommon.StakeDelegationCertificate, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	if !b.poolSet {
		return nil, fmt.Errorf("pool key hash must be exactly %d bytes", hashSize)
	}
	credential := b.credential
	return &lcommon.StakeDelegationCertificate{
		CertType:        uint(lcommon.CertificateTypeStakeDelegation),
		StakeCredential: &credential,
		PoolKeyHash:     b.poolKeyHash,
	}, nil
}

// PoolRegistrationBuilder builds a stake pool registration certificate.
type PoolRegistrationBuilder struct {
	operator      lcommon.PoolKeyHash
	operatorSet   bool
	vrfKeyHash    lcommon.VrfKeyHash
	vrfSet        bool
	pledge        uint64
	cost          uint64
	margin        cbor.Rat
	owners        []lcommon.AddrKeyHash
	rewardAccount lcommon.AddrKeyHash
	rewardSet     bool
	relays        []lcommon.PoolRelay
	metadata      *lcommon.PoolMetadata
	metadataErr   error
}

func NewPoolRegistration() *PoolRegistrationBuilder {
	return &PoolRegistrationBuilder{margin: cbor.Rat{Rat: big.NewRat(0, 1)}}
}

func (b *PoolRegistrationBuilder) WithOperator(hash []byte) *PoolRegistrationBuilder {
	b.operator = lcommon.NewBlake2b224(hash)
	b.operatorSet = len(hash) == hashSize
	return b
}

func (b *PoolRegistrationBuilder) WithVrfKeyHash(hash []byte) *PoolRegistrationBuilder {
	b.vrfKeyHash = lcommon.NewBlake2b256(hash)
	b.vrfSet = len(hash) == lcommon.Blake2b256Size
	return b
}

func (b *PoolRegistrationBuilder) WithPledge(amount uint64) *PoolRegistrationBuilder {
	b.pledge = amount
	return b
}

func (b *PoolRegistrationBuilder) WithCost(amount uint64) *PoolRegistrationBuilder {
	b.cost = amount
	return b
}

func (b *PoolRegistrationBuilder) WithMargin(numerator, denominator uint64) *PoolRegistrationBuilder {
	if denominator == 0 {
		b.margin = cbor.Rat{}
		return b
	}
	b.margin = cbor.Rat{Rat: new(big.Rat).SetFrac(
		new(big.Int).SetUint64(numerator), new(big.Int).SetUint64(denominator),
	)}
	return b
}

func (b *PoolRegistrationBuilder) WithRewardAccountKey(hash []byte) *PoolRegistrationBuilder {
	b.rewardAccount = lcommon.NewBlake2b224(hash)
	b.rewardSet = len(hash) == hashSize
	return b
}

func (b *PoolRegistrationBuilder) WithOwners(hashes ...[]byte) *PoolRegistrationBuilder {
	b.owners = make([]lcommon.AddrKeyHash, len(hashes))
	for i, hash := range hashes {
		b.owners[i] = lcommon.NewBlake2b224(hash)
	}
	return b
}

func (b *PoolRegistrationBuilder) WithRelays(relays ...lcommon.PoolRelay) *PoolRegistrationBuilder {
	b.relays = append([]lcommon.PoolRelay(nil), relays...)
	return b
}

func (b *PoolRegistrationBuilder) WithMetadata(url string, hash []byte) *PoolRegistrationBuilder {
	if len(hash) != lcommon.Blake2b256Size {
		b.metadataErr = fmt.Errorf("pool metadata hash must be exactly %d bytes", lcommon.Blake2b256Size)
		return b
	}
	metadataHash := lcommon.NewBlake2b256(hash)
	b.metadata = &lcommon.PoolMetadata{Url: url, Hash: metadataHash}
	b.metadataErr = nil
	return b
}

func (b *PoolRegistrationBuilder) Build() (*lcommon.PoolRegistrationCertificate, error) {
	switch {
	case !b.operatorSet:
		return nil, fmt.Errorf("pool operator hash must be exactly %d bytes", hashSize)
	case !b.vrfSet:
		return nil, fmt.Errorf("VRF key hash must be exactly %d bytes", lcommon.Blake2b256Size)
	case !b.rewardSet:
		return nil, fmt.Errorf("reward account key hash must be exactly %d bytes", hashSize)
	case b.metadataErr != nil:
		return nil, b.metadataErr
	case b.margin.Rat == nil:
		return nil, errors.New("pool margin denominator cannot be zero")
	}
	if b.margin.Sign() < 0 || b.margin.Cmp(big.NewRat(1, 1)) > 0 {
		return nil, errors.New("pool margin must be in the unit interval [0,1]")
	}
	return &lcommon.PoolRegistrationCertificate{
		CertType:      uint(lcommon.CertificateTypePoolRegistration),
		Operator:      b.operator,
		VrfKeyHash:    b.vrfKeyHash,
		Pledge:        b.pledge,
		Cost:          b.cost,
		Margin:        b.margin,
		RewardAccount: b.rewardAccount,
		PoolOwners:    b.owners,
		Relays:        b.relays,
		PoolMetadata:  b.metadata,
	}, nil
}

// PoolRetirementBuilder builds a stake pool retirement certificate.
type PoolRetirementBuilder struct {
	poolKeyHash lcommon.PoolKeyHash
	poolSet     bool
	epoch       uint64
}

func NewPoolRetirement() *PoolRetirementBuilder { return &PoolRetirementBuilder{} }

func (b *PoolRetirementBuilder) WithPoolKeyHash(hash []byte) *PoolRetirementBuilder {
	b.poolKeyHash = lcommon.NewBlake2b224(hash)
	b.poolSet = len(hash) == hashSize
	return b
}

func (b *PoolRetirementBuilder) WithEpoch(epoch uint64) *PoolRetirementBuilder {
	b.epoch = epoch
	return b
}

func (b *PoolRetirementBuilder) Build() (*lcommon.PoolRetirementCertificate, error) {
	if !b.poolSet {
		return nil, fmt.Errorf("pool key hash must be exactly %d bytes", hashSize)
	}
	return &lcommon.PoolRetirementCertificate{
		CertType:    uint(lcommon.CertificateTypePoolRetirement),
		PoolKeyHash: b.poolKeyHash,
		Epoch:       b.epoch,
	}, nil
}

type conwayBuilder struct {
	credential    lcommon.Credential
	credentialErr error
	credentialSet bool
	deposit       uint64
	anchor        *lcommon.GovAnchor
	anchorErr     error
}

func checkedAmount(amount uint64) (int64, error) {
	if amount > math.MaxInt64 {
		return 0, fmt.Errorf("deposit %d exceeds maximum int64 value", amount)
	}
	return int64(amount), nil
}

func (b *conwayBuilder) WithCredential(hash []byte) *conwayBuilder {
	b.credential, b.credentialErr = keyCredential(hash)
	b.credentialSet = true
	return b
}

func (b *conwayBuilder) WithScriptCredential(hash []byte) *conwayBuilder {
	b.credential, b.credentialErr = scriptCredential(hash)
	b.credentialSet = true
	return b
}

func (b *conwayBuilder) WithDeposit(amount uint64) *conwayBuilder {
	b.deposit = amount
	return b
}

func (b *conwayBuilder) WithAnchor(url string, dataHash []byte) *conwayBuilder {
	if len(dataHash) != 0 && len(dataHash) != lcommon.Blake2b256Size {
		b.anchorErr = fmt.Errorf("anchor data hash must be exactly %d bytes", lcommon.Blake2b256Size)
		return b
	}
	b.anchorErr = nil
	b.anchor = &lcommon.GovAnchor{Url: url}
	if len(dataHash) != 0 {
		copy(b.anchor.DataHash[:], dataHash)
	}
	return b
}

func (b *conwayBuilder) validate() error {
	if !b.credentialSet {
		return errors.New("credential is required")
	}
	if b.credentialErr != nil {
		return b.credentialErr
	}
	if b.anchorErr != nil {
		return b.anchorErr
	}
	if b.deposit > math.MaxInt64 {
		return fmt.Errorf("deposit %d exceeds maximum int64 value", b.deposit)
	}
	return nil
}

// DRepRegistrationBuilder builds a Conway DRep registration certificate.
type DRepRegistrationBuilder struct{ conwayBuilder }

func NewDRepRegistration() *DRepRegistrationBuilder { return &DRepRegistrationBuilder{} }

func (b *DRepRegistrationBuilder) WithCredential(hash []byte) *DRepRegistrationBuilder {
	b.conwayBuilder.WithCredential(hash)
	return b
}

func (b *DRepRegistrationBuilder) WithScriptCredential(hash []byte) *DRepRegistrationBuilder {
	b.conwayBuilder.WithScriptCredential(hash)
	return b
}

func (b *DRepRegistrationBuilder) WithDeposit(amount uint64) *DRepRegistrationBuilder {
	b.conwayBuilder.WithDeposit(amount)
	return b
}

func (b *DRepRegistrationBuilder) WithAnchor(url string, hash []byte) *DRepRegistrationBuilder {
	b.conwayBuilder.WithAnchor(url, hash)
	return b
}

func (b *DRepRegistrationBuilder) Build() (*lcommon.RegistrationDrepCertificate, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	amount, err := checkedAmount(b.deposit)
	if err != nil {
		return nil, err
	}
	return &lcommon.RegistrationDrepCertificate{
		CertType:       uint(lcommon.CertificateTypeRegistrationDrep),
		DrepCredential: b.credential,
		Amount:         amount,
		Anchor:         b.anchor,
	}, nil
}

// DRepDeregistrationBuilder builds a Conway DRep deregistration certificate.
type DRepDeregistrationBuilder struct{ conwayBuilder }

func NewDRepDeregistration() *DRepDeregistrationBuilder { return &DRepDeregistrationBuilder{} }

func (b *DRepDeregistrationBuilder) WithCredential(hash []byte) *DRepDeregistrationBuilder {
	b.conwayBuilder.WithCredential(hash)
	return b
}

func (b *DRepDeregistrationBuilder) WithScriptCredential(hash []byte) *DRepDeregistrationBuilder {
	b.conwayBuilder.WithScriptCredential(hash)
	return b
}

func (b *DRepDeregistrationBuilder) WithDeposit(amount uint64) *DRepDeregistrationBuilder {
	b.conwayBuilder.WithDeposit(amount)
	return b
}

func (b *DRepDeregistrationBuilder) Build() (*lcommon.DeregistrationDrepCertificate, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	amount, err := checkedAmount(b.deposit)
	if err != nil {
		return nil, err
	}
	return &lcommon.DeregistrationDrepCertificate{
		CertType:       uint(lcommon.CertificateTypeDeregistrationDrep),
		DrepCredential: b.credential,
		Amount:         amount,
	}, nil
}

// DRepUpdateBuilder builds a Conway DRep update certificate.
type DRepUpdateBuilder struct{ conwayBuilder }

func NewDRepUpdate() *DRepUpdateBuilder { return &DRepUpdateBuilder{} }

func (b *DRepUpdateBuilder) WithCredential(hash []byte) *DRepUpdateBuilder {
	b.conwayBuilder.WithCredential(hash)
	return b
}

func (b *DRepUpdateBuilder) WithScriptCredential(hash []byte) *DRepUpdateBuilder {
	b.conwayBuilder.WithScriptCredential(hash)
	return b
}

func (b *DRepUpdateBuilder) WithAnchor(url string, hash []byte) *DRepUpdateBuilder {
	b.conwayBuilder.WithAnchor(url, hash)
	return b
}

func (b *DRepUpdateBuilder) Build() (*lcommon.UpdateDrepCertificate, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	return &lcommon.UpdateDrepCertificate{
		CertType:       uint(lcommon.CertificateTypeUpdateDrep),
		DrepCredential: b.credential,
		Anchor:         b.anchor,
	}, nil
}

type drepBuilder struct {
	drepSet bool
	drep    lcommon.Drep
}

func (b *drepBuilder) WithDRep(drep lcommon.Drep) *drepBuilder {
	b.drep = drep
	b.drepSet = true
	return b
}

func (b *drepBuilder) WithDRepKeyHash(hash []byte) *drepBuilder {
	b.drep = lcommon.Drep{Type: lcommon.DrepTypeAddrKeyHash, Credential: append([]byte(nil), hash...)}
	b.drepSet = true
	return b
}

func (b *drepBuilder) WithDRepScriptHash(hash []byte) *drepBuilder {
	b.drep = lcommon.Drep{Type: lcommon.DrepTypeScriptHash, Credential: append([]byte(nil), hash...)}
	b.drepSet = true
	return b
}

func (b *drepBuilder) validateDRep() error {
	if !b.drepSet {
		return errors.New("DRep is required")
	}
	if (b.drep.Type == lcommon.DrepTypeAddrKeyHash || b.drep.Type == lcommon.DrepTypeScriptHash) && len(b.drep.Credential) != hashSize {
		return fmt.Errorf("DRep hash must be exactly %d bytes", hashSize)
	}
	return nil
}

// VoteDelegationBuilder builds a Conway vote delegation certificate.
type VoteDelegationBuilder struct {
	stakeBuilder
	drepBuilder
}

func NewVoteDelegation() *VoteDelegationBuilder { return &VoteDelegationBuilder{} }

func (b *VoteDelegationBuilder) WithCredential(hash []byte) *VoteDelegationBuilder {
	b.stakeBuilder.WithCredential(hash)
	return b
}

func (b *VoteDelegationBuilder) WithScriptCredential(hash []byte) *VoteDelegationBuilder {
	b.stakeBuilder.WithScriptCredential(hash)
	return b
}

func (b *VoteDelegationBuilder) WithDRep(drep lcommon.Drep) *VoteDelegationBuilder {
	b.drepBuilder.WithDRep(drep)
	return b
}

func (b *VoteDelegationBuilder) WithDRepKeyHash(hash []byte) *VoteDelegationBuilder {
	b.drepBuilder.WithDRepKeyHash(hash)
	return b
}

func (b *VoteDelegationBuilder) WithDRepScriptHash(hash []byte) *VoteDelegationBuilder {
	b.drepBuilder.WithDRepScriptHash(hash)
	return b
}

func (b *VoteDelegationBuilder) Build() (*lcommon.VoteDelegationCertificate, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	if err := b.validateDRep(); err != nil {
		return nil, err
	}
	return &lcommon.VoteDelegationCertificate{
		CertType:        uint(lcommon.CertificateTypeVoteDelegation),
		StakeCredential: b.credential,
		Drep:            b.drep,
	}, nil
}

// StakeVoteDelegationBuilder builds a Conway stake-and-vote delegation certificate.
type StakeVoteDelegationBuilder struct {
	stakeBuilder
	drepBuilder
	poolKeyHash lcommon.PoolKeyHash
	poolSet     bool
}

func NewStakeVoteDelegation() *StakeVoteDelegationBuilder { return &StakeVoteDelegationBuilder{} }

func (b *StakeVoteDelegationBuilder) WithCredential(hash []byte) *StakeVoteDelegationBuilder {
	b.stakeBuilder.WithCredential(hash)
	return b
}

func (b *StakeVoteDelegationBuilder) WithScriptCredential(hash []byte) *StakeVoteDelegationBuilder {
	b.stakeBuilder.WithScriptCredential(hash)
	return b
}

func (b *StakeVoteDelegationBuilder) WithDRep(drep lcommon.Drep) *StakeVoteDelegationBuilder {
	b.drepBuilder.WithDRep(drep)
	return b
}

func (b *StakeVoteDelegationBuilder) WithDRepKeyHash(hash []byte) *StakeVoteDelegationBuilder {
	b.drepBuilder.WithDRepKeyHash(hash)
	return b
}

func (b *StakeVoteDelegationBuilder) WithDRepScriptHash(hash []byte) *StakeVoteDelegationBuilder {
	b.drepBuilder.WithDRepScriptHash(hash)
	return b
}

func (b *StakeVoteDelegationBuilder) WithPoolKeyHash(hash []byte) *StakeVoteDelegationBuilder {
	b.poolKeyHash = lcommon.NewBlake2b224(hash)
	b.poolSet = len(hash) == hashSize
	return b
}

func (b *StakeVoteDelegationBuilder) Build() (*lcommon.StakeVoteDelegationCertificate, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	if err := b.validateDRep(); err != nil {
		return nil, err
	}
	if !b.poolSet {
		return nil, fmt.Errorf("pool key hash must be exactly %d bytes", hashSize)
	}
	return &lcommon.StakeVoteDelegationCertificate{
		CertType:        uint(lcommon.CertificateTypeStakeVoteDelegation),
		StakeCredential: b.credential,
		PoolKeyHash:     b.poolKeyHash,
		Drep:            b.drep,
	}, nil
}

type combinedBuilder struct {
	stakeBuilder
	drepBuilder
	poolKeyHash lcommon.PoolKeyHash
	poolSet     bool
	deposit     uint64
}

func (b *combinedBuilder) WithCredential(hash []byte) {
	b.stakeBuilder.WithCredential(hash)
}

func (b *combinedBuilder) WithDRepKeyHash(hash []byte) {
	b.drepBuilder.WithDRepKeyHash(hash)
}

func (b *combinedBuilder) WithDRep(drep lcommon.Drep) {
	b.drepBuilder.WithDRep(drep)
}

func (b *combinedBuilder) WithDRepScriptHash(hash []byte) {
	b.drepBuilder.WithDRepScriptHash(hash)
}

func (b *combinedBuilder) WithPoolKeyHash(hash []byte) {
	b.poolKeyHash = lcommon.NewBlake2b224(hash)
	b.poolSet = len(hash) == hashSize
}

func (b *combinedBuilder) WithDeposit(amount uint64) {
	b.deposit = amount
}

func (b *combinedBuilder) validateCombined(requireDRep, requirePool bool) error {
	if err := b.validate(); err != nil {
		return err
	}
	if requireDRep {
		if err := b.validateDRep(); err != nil {
			return err
		}
	}
	if requirePool && !b.poolSet {
		return fmt.Errorf("pool key hash must be exactly %d bytes", hashSize)
	}
	if b.deposit > math.MaxInt64 {
		return fmt.Errorf("deposit %d exceeds maximum int64 value", b.deposit)
	}
	return nil
}

// StakeRegistrationDelegationBuilder builds a Conway stake registration and delegation certificate.
type StakeRegistrationDelegationBuilder struct{ combinedBuilder }

func NewStakeRegistrationDelegation() *StakeRegistrationDelegationBuilder {
	return &StakeRegistrationDelegationBuilder{}
}

func (b *StakeRegistrationDelegationBuilder) WithCredential(hash []byte) *StakeRegistrationDelegationBuilder {
	b.combinedBuilder.WithCredential(hash)
	return b
}

func (b *StakeRegistrationDelegationBuilder) WithPoolKeyHash(hash []byte) *StakeRegistrationDelegationBuilder {
	b.combinedBuilder.WithPoolKeyHash(hash)
	return b
}

func (b *StakeRegistrationDelegationBuilder) WithScriptCredential(hash []byte) *StakeRegistrationDelegationBuilder {
	b.stakeBuilder.WithScriptCredential(hash)
	return b
}

func (b *StakeRegistrationDelegationBuilder) WithDeposit(amount uint64) *StakeRegistrationDelegationBuilder {
	b.combinedBuilder.WithDeposit(amount)
	return b
}

func (b *StakeRegistrationDelegationBuilder) Build() (*lcommon.StakeRegistrationDelegationCertificate, error) {
	if err := b.validateCombined(false, true); err != nil {
		return nil, err
	}
	amount, err := checkedAmount(b.deposit)
	if err != nil {
		return nil, err
	}
	return &lcommon.StakeRegistrationDelegationCertificate{CertType: uint(lcommon.CertificateTypeStakeRegistrationDelegation), StakeCredential: b.credential, PoolKeyHash: b.poolKeyHash, Amount: amount}, nil
}

// VoteRegistrationDelegationBuilder builds a Conway vote registration and delegation certificate.
type VoteRegistrationDelegationBuilder struct{ combinedBuilder }

func NewVoteRegistrationDelegation() *VoteRegistrationDelegationBuilder {
	return &VoteRegistrationDelegationBuilder{}
}

func (b *VoteRegistrationDelegationBuilder) WithCredential(hash []byte) *VoteRegistrationDelegationBuilder {
	b.combinedBuilder.WithCredential(hash)
	return b
}

func (b *VoteRegistrationDelegationBuilder) WithDRepKeyHash(hash []byte) *VoteRegistrationDelegationBuilder {
	b.combinedBuilder.WithDRepKeyHash(hash)
	return b
}

func (b *VoteRegistrationDelegationBuilder) WithDRep(drep lcommon.Drep) *VoteRegistrationDelegationBuilder {
	b.combinedBuilder.WithDRep(drep)
	return b
}

func (b *VoteRegistrationDelegationBuilder) WithDRepScriptHash(hash []byte) *VoteRegistrationDelegationBuilder {
	b.combinedBuilder.WithDRepScriptHash(hash)
	return b
}

func (b *VoteRegistrationDelegationBuilder) WithDeposit(amount uint64) *VoteRegistrationDelegationBuilder {
	b.combinedBuilder.WithDeposit(amount)
	return b
}

func (b *VoteRegistrationDelegationBuilder) Build() (*lcommon.VoteRegistrationDelegationCertificate, error) {
	if err := b.validateCombined(true, false); err != nil {
		return nil, err
	}
	amount, err := checkedAmount(b.deposit)
	if err != nil {
		return nil, err
	}
	return &lcommon.VoteRegistrationDelegationCertificate{CertType: uint(lcommon.CertificateTypeVoteRegistrationDelegation), StakeCredential: b.credential, Drep: b.drep, Amount: amount}, nil
}

// StakeVoteRegistrationDelegationBuilder builds a Conway stake and vote registration and delegation certificate.
type StakeVoteRegistrationDelegationBuilder struct{ combinedBuilder }

func NewStakeVoteRegistrationDelegation() *StakeVoteRegistrationDelegationBuilder {
	return &StakeVoteRegistrationDelegationBuilder{}
}

func (b *StakeVoteRegistrationDelegationBuilder) WithCredential(hash []byte) *StakeVoteRegistrationDelegationBuilder {
	b.combinedBuilder.WithCredential(hash)
	return b
}

func (b *StakeVoteRegistrationDelegationBuilder) WithDRepKeyHash(hash []byte) *StakeVoteRegistrationDelegationBuilder {
	b.combinedBuilder.WithDRepKeyHash(hash)
	return b
}

func (b *StakeVoteRegistrationDelegationBuilder) WithDRep(drep lcommon.Drep) *StakeVoteRegistrationDelegationBuilder {
	b.combinedBuilder.WithDRep(drep)
	return b
}

func (b *StakeVoteRegistrationDelegationBuilder) WithDRepScriptHash(hash []byte) *StakeVoteRegistrationDelegationBuilder {
	b.combinedBuilder.WithDRepScriptHash(hash)
	return b
}

func (b *StakeVoteRegistrationDelegationBuilder) WithPoolKeyHash(hash []byte) *StakeVoteRegistrationDelegationBuilder {
	b.combinedBuilder.WithPoolKeyHash(hash)
	return b
}

func (b *StakeVoteRegistrationDelegationBuilder) WithDeposit(amount uint64) *StakeVoteRegistrationDelegationBuilder {
	b.combinedBuilder.WithDeposit(amount)
	return b
}

func (b *StakeVoteRegistrationDelegationBuilder) Build() (*lcommon.StakeVoteRegistrationDelegationCertificate, error) {
	if err := b.validateCombined(true, true); err != nil {
		return nil, err
	}
	amount, err := checkedAmount(b.deposit)
	if err != nil {
		return nil, err
	}
	return &lcommon.StakeVoteRegistrationDelegationCertificate{CertType: uint(lcommon.CertificateTypeStakeVoteRegistrationDelegation), StakeCredential: b.credential, PoolKeyHash: b.poolKeyHash, Drep: b.drep, Amount: amount}, nil
}

// AuthCommitteeHotBuilder builds a Conway committee hot-key authorization certificate.
type AuthCommitteeHotBuilder struct {
	cold            lcommon.Credential
	hot             lcommon.Credential
	coldSet, hotSet bool
}

func NewAuthCommitteeHot() *AuthCommitteeHotBuilder { return &AuthCommitteeHotBuilder{} }

func (b *AuthCommitteeHotBuilder) WithColdCredential(hash []byte) *AuthCommitteeHotBuilder {
	b.cold, _ = keyCredential(hash)
	b.coldSet = len(hash) == hashSize
	return b
}

func (b *AuthCommitteeHotBuilder) WithHotCredential(hash []byte) *AuthCommitteeHotBuilder {
	b.hot, _ = keyCredential(hash)
	b.hotSet = len(hash) == hashSize
	return b
}

func (b *AuthCommitteeHotBuilder) Build() (*lcommon.AuthCommitteeHotCertificate, error) {
	if !b.coldSet || !b.hotSet {
		return nil, fmt.Errorf("committee credentials must be exactly %d bytes", hashSize)
	}
	return &lcommon.AuthCommitteeHotCertificate{CertType: uint(lcommon.CertificateTypeAuthCommitteeHot), ColdCredential: b.cold, HotCredential: b.hot}, nil
}

// ResignCommitteeColdBuilder builds a Conway committee cold-key resignation certificate.
type ResignCommitteeColdBuilder struct {
	cold      lcommon.Credential
	coldSet   bool
	anchor    *lcommon.GovAnchor
	anchorErr error
}

func NewResignCommitteeCold() *ResignCommitteeColdBuilder { return &ResignCommitteeColdBuilder{} }

func (b *ResignCommitteeColdBuilder) WithColdCredential(hash []byte) *ResignCommitteeColdBuilder {
	b.cold, _ = keyCredential(hash)
	b.coldSet = len(hash) == hashSize
	return b
}

func (b *ResignCommitteeColdBuilder) WithAnchor(url string, dataHash []byte) *ResignCommitteeColdBuilder {
	if len(dataHash) != 0 && len(dataHash) != lcommon.Blake2b256Size {
		b.anchorErr = fmt.Errorf("anchor data hash must be exactly %d bytes", lcommon.Blake2b256Size)
		return b
	}
	b.anchorErr = nil
	b.anchor = &lcommon.GovAnchor{Url: url}
	if len(dataHash) != 0 {
		copy(b.anchor.DataHash[:], dataHash)
	}
	return b
}

func (b *ResignCommitteeColdBuilder) Build() (*lcommon.ResignCommitteeColdCertificate, error) {
	if !b.coldSet {
		return nil, fmt.Errorf("committee cold credential must be exactly %d bytes", hashSize)
	}
	if b.anchorErr != nil {
		return nil, b.anchorErr
	}
	return &lcommon.ResignCommitteeColdCertificate{CertType: uint(lcommon.CertificateTypeResignCommitteeCold), ColdCredential: b.cold, Anchor: b.anchor}, nil
}
