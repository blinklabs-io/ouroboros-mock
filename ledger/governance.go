// Copyright 2025 Blink Labs Software
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
	"math"

	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

// CommitteeMemberBuilder defines an interface for building mock committee member state
type CommitteeMemberBuilder interface {
	WithColdKey(key []byte) CommitteeMemberBuilder
	WithHotKey(key []byte) CommitteeMemberBuilder
	WithExpiryEpoch(epoch uint64) CommitteeMemberBuilder
	WithResigned(resigned bool) CommitteeMemberBuilder
	WithResignAnchor(url string, dataHash []byte) CommitteeMemberBuilder
	Build() (*CommitteeMember, error)
}

// CommitteeMember represents a committee member state for mock purposes
type CommitteeMember struct {
	ColdCredential lcommon.Credential
	HotCredential  lcommon.Credential
	ExpiryEpoch    uint64
	Resigned       bool
	ResignAnchor   *lcommon.GovAnchor
}

// committeeMemberBuilder implements CommitteeMemberBuilder
type committeeMemberBuilder struct {
	coldKey           []byte
	hotKey            []byte
	expiryEpoch       uint64
	resigned          bool
	resignAnchor      *lcommon.GovAnchor
	resignDataHashErr error // Stores dataHash validation error for deferred reporting
}

// NewCommitteeMemberBuilder creates a new CommitteeMemberBuilder
func NewCommitteeMemberBuilder() CommitteeMemberBuilder {
	return &committeeMemberBuilder{}
}

// WithColdKey sets the cold key credential for the committee member
func (b *committeeMemberBuilder) WithColdKey(
	key []byte,
) CommitteeMemberBuilder {
	b.coldKey = key
	return b
}

// WithHotKey sets the hot key credential for the committee member
func (b *committeeMemberBuilder) WithHotKey(key []byte) CommitteeMemberBuilder {
	b.hotKey = key
	return b
}

// WithExpiryEpoch sets the expiry epoch for the committee member
func (b *committeeMemberBuilder) WithExpiryEpoch(
	epoch uint64,
) CommitteeMemberBuilder {
	b.expiryEpoch = epoch
	return b
}

// WithResigned sets whether the committee member has resigned
func (b *committeeMemberBuilder) WithResigned(
	resigned bool,
) CommitteeMemberBuilder {
	b.resigned = resigned
	return b
}

// WithResignAnchor sets the resignation anchor for the committee member
// If dataHash is provided but not exactly 32 bytes, Build() will return an error
func (b *committeeMemberBuilder) WithResignAnchor(
	url string,
	dataHash []byte,
) CommitteeMemberBuilder {
	if url != "" {
		anchor := &lcommon.GovAnchor{
			Url: url,
		}
		if len(dataHash) > 0 {
			if len(dataHash) != 32 {
				b.resignDataHashErr = fmt.Errorf(
					"resign anchor dataHash must be exactly 32 bytes, got %d",
					len(dataHash),
				)
			} else {
				// Clear any previous error when valid hash is provided
				b.resignDataHashErr = nil
				copy(anchor.DataHash[:], dataHash)
			}
		}
		b.resignAnchor = anchor
	}
	return b
}

// Build constructs a CommitteeMember from the builder state
func (b *committeeMemberBuilder) Build() (*CommitteeMember, error) {
	if len(b.coldKey) == 0 {
		return nil, errors.New("cold key is required")
	}
	// Return any stored validation errors
	if b.resignDataHashErr != nil {
		return nil, b.resignDataHashErr
	}

	member := &CommitteeMember{
		ColdCredential: lcommon.Credential{
			CredType:   lcommon.CredentialTypeAddrKeyHash,
			Credential: lcommon.NewBlake2b224(b.coldKey),
		},
		ExpiryEpoch:  b.expiryEpoch,
		Resigned:     b.resigned,
		ResignAnchor: b.resignAnchor,
	}

	if len(b.hotKey) > 0 {
		member.HotCredential = lcommon.Credential{
			CredType:   lcommon.CredentialTypeAddrKeyHash,
			Credential: lcommon.NewBlake2b224(b.hotKey),
		}
	}

	return member, nil
}

// DRepRegistrationBuilder defines an interface for building mock DRep registrations
type DRepRegistrationBuilder interface {
	WithCredential(cred []byte) DRepRegistrationBuilder
	WithAnchor(url string, dataHash []byte) DRepRegistrationBuilder
	WithDeposit(lovelace uint64) DRepRegistrationBuilder
	Build() (*lcommon.RegistrationDrepCertificate, error)
}

// drepRegistrationBuilder implements DRepRegistrationBuilder
type drepRegistrationBuilder struct {
	credential []byte
	anchorURL  string
	dataHash   []byte
	deposit    uint64
}

// NewDRepRegistrationBuilder creates a new DRepRegistrationBuilder
func NewDRepRegistrationBuilder() DRepRegistrationBuilder {
	return &drepRegistrationBuilder{}
}

// WithCredential sets the DRep credential
func (b *drepRegistrationBuilder) WithCredential(
	cred []byte,
) DRepRegistrationBuilder {
	b.credential = cred
	return b
}

// WithAnchor sets the anchor URL and data hash for the DRep registration
func (b *drepRegistrationBuilder) WithAnchor(
	url string,
	dataHash []byte,
) DRepRegistrationBuilder {
	b.anchorURL = url
	b.dataHash = dataHash
	return b
}

// WithDeposit sets the deposit amount in lovelace
func (b *drepRegistrationBuilder) WithDeposit(
	lovelace uint64,
) DRepRegistrationBuilder {
	b.deposit = lovelace
	return b
}

// Build constructs a RegistrationDrepCertificate from the builder state
func (b *drepRegistrationBuilder) Build() (*lcommon.RegistrationDrepCertificate, error) {
	if len(b.credential) == 0 {
		return nil, errors.New("credential is required")
	}

	// Validate deposit doesn't overflow int64
	if b.deposit > uint64(math.MaxInt64) {
		return nil, fmt.Errorf(
			"deposit %d exceeds maximum int64 value",
			b.deposit,
		)
	}

	// Validate dataHash length if provided (Blake2b256 is 32 bytes)
	if len(b.dataHash) > 0 && len(b.dataHash) != 32 {
		return nil, fmt.Errorf(
			"dataHash must be exactly 32 bytes, got %d",
			len(b.dataHash),
		)
	}

	cert := &lcommon.RegistrationDrepCertificate{
		CertType: uint(lcommon.CertificateTypeRegistrationDrep),
		DrepCredential: lcommon.Credential{
			CredType:   lcommon.CredentialTypeAddrKeyHash,
			Credential: lcommon.NewBlake2b224(b.credential),
		},
		Amount: int64(b.deposit),
	}

	if b.anchorURL != "" {
		anchor := &lcommon.GovAnchor{
			Url: b.anchorURL,
		}
		if len(b.dataHash) > 0 {
			copy(anchor.DataHash[:], b.dataHash)
		}
		cert.Anchor = anchor
	}

	return cert, nil
}

// ConstitutionBuilder defines an interface for building mock constitutions
type ConstitutionBuilder interface {
	WithAnchor(url string, dataHash []byte) ConstitutionBuilder
	WithScriptHash(hash []byte) ConstitutionBuilder
	Build() (*Constitution, error)
}

// Constitution represents a constitution for mock purposes
type Constitution struct {
	Anchor     lcommon.GovAnchor
	ScriptHash []byte
}

// constitutionBuilder implements ConstitutionBuilder
type constitutionBuilder struct {
	anchorURL  string
	dataHash   []byte
	scriptHash []byte
}

// NewConstitutionBuilder creates a new ConstitutionBuilder
func NewConstitutionBuilder() ConstitutionBuilder {
	return &constitutionBuilder{}
}

// WithAnchor sets the anchor URL and data hash for the constitution
func (b *constitutionBuilder) WithAnchor(
	url string,
	dataHash []byte,
) ConstitutionBuilder {
	b.anchorURL = url
	b.dataHash = dataHash
	return b
}

// WithScriptHash sets the optional script hash for the constitution
func (b *constitutionBuilder) WithScriptHash(hash []byte) ConstitutionBuilder {
	b.scriptHash = hash
	return b
}

// Build constructs a Constitution from the builder state
func (b *constitutionBuilder) Build() (*Constitution, error) {
	if b.anchorURL == "" {
		return nil, errors.New("anchor URL is required")
	}

	// Validate dataHash length if provided (Blake2b256 is 32 bytes)
	if len(b.dataHash) > 0 && len(b.dataHash) != 32 {
		return nil, fmt.Errorf(
			"dataHash must be exactly 32 bytes, got %d",
			len(b.dataHash),
		)
	}

	// Validate scriptHash length if provided (Blake2b224 is 28 bytes)
	if len(b.scriptHash) > 0 && len(b.scriptHash) != 28 {
		return nil, fmt.Errorf(
			"scriptHash must be exactly 28 bytes, got %d",
			len(b.scriptHash),
		)
	}

	constitution := &Constitution{
		Anchor: lcommon.GovAnchor{
			Url: b.anchorURL,
		},
		ScriptHash: b.scriptHash,
	}

	if len(b.dataHash) > 0 {
		copy(constitution.Anchor.DataHash[:], b.dataHash)
	}

	return constitution, nil
}

// GovAnchorBuilder defines an interface for building governance anchors
type GovAnchorBuilder interface {
	WithURL(url string) GovAnchorBuilder
	WithDataHash(hash []byte) GovAnchorBuilder
	Build() (*lcommon.GovAnchor, error)
}

// govAnchorBuilder implements GovAnchorBuilder
type govAnchorBuilder struct {
	url      string
	dataHash []byte
}

// NewGovAnchorBuilder creates a new GovAnchorBuilder
func NewGovAnchorBuilder() GovAnchorBuilder {
	return &govAnchorBuilder{}
}

// WithURL sets the URL for the governance anchor
func (b *govAnchorBuilder) WithURL(url string) GovAnchorBuilder {
	b.url = url
	return b
}

// WithDataHash sets the data hash for the governance anchor
func (b *govAnchorBuilder) WithDataHash(hash []byte) GovAnchorBuilder {
	b.dataHash = hash
	return b
}

// Build constructs a GovAnchor from the builder state
func (b *govAnchorBuilder) Build() (*lcommon.GovAnchor, error) {
	if b.url == "" {
		return nil, errors.New("URL is required")
	}

	// Validate dataHash length if provided (Blake2b256 is 32 bytes)
	if len(b.dataHash) > 0 && len(b.dataHash) != 32 {
		return nil, fmt.Errorf(
			"dataHash must be exactly 32 bytes, got %d",
			len(b.dataHash),
		)
	}

	anchor := &lcommon.GovAnchor{
		Url: b.url,
	}

	if len(b.dataHash) > 0 {
		copy(anchor.DataHash[:], b.dataHash)
	}

	return anchor, nil
}

// VoterBuilder defines an interface for building mock voters
type VoterBuilder interface {
	WithType(voterType uint8) VoterBuilder
	WithHash(hash []byte) VoterBuilder
	Build() (*lcommon.Voter, error)
}

// voterBuilder implements VoterBuilder
type voterBuilder struct {
	voterType uint8
	hash      []byte
}

// NewVoterBuilder creates a new VoterBuilder
func NewVoterBuilder() VoterBuilder {
	return &voterBuilder{}
}

// WithType sets the voter type (constitutional committee, drep, or pool)
func (b *voterBuilder) WithType(voterType uint8) VoterBuilder {
	b.voterType = voterType
	return b
}

// WithHash sets the credential hash for the voter
func (b *voterBuilder) WithHash(hash []byte) VoterBuilder {
	b.hash = hash
	return b
}

// Build constructs a Voter from the builder state
func (b *voterBuilder) Build() (*lcommon.Voter, error) {
	if len(b.hash) == 0 {
		return nil, errors.New("hash is required")
	}
	// Validate hash length (Blake2b224 is 28 bytes)
	if len(b.hash) != 28 {
		return nil, fmt.Errorf(
			"hash must be exactly 28 bytes, got %d",
			len(b.hash),
		)
	}
	// Validate voter type (0-4 per CIP-1694):
	// 0=CC hot key hash, 1=CC hot script hash, 2=DRep key hash,
	// 3=DRep script hash, 4=staking pool key hash
	if b.voterType > 4 {
		return nil, fmt.Errorf(
			"invalid voter type %d, must be 0-4",
			b.voterType,
		)
	}

	voter := &lcommon.Voter{
		Type: b.voterType,
	}
	copy(voter.Hash[:], b.hash)

	return voter, nil
}

// VotingProcedureBuilder defines an interface for building mock voting procedures
type VotingProcedureBuilder interface {
	WithVote(vote uint8) VotingProcedureBuilder
	WithAnchor(url string, dataHash []byte) VotingProcedureBuilder
	Build() (*lcommon.VotingProcedure, error)
}

// votingProcedureBuilder implements VotingProcedureBuilder
type votingProcedureBuilder struct {
	vote      uint8
	voteSet   bool // Tracks whether WithVote was called
	anchorURL string
	dataHash  []byte
}

// NewVotingProcedureBuilder creates a new VotingProcedureBuilder
func NewVotingProcedureBuilder() VotingProcedureBuilder {
	return &votingProcedureBuilder{}
}

// WithVote sets the vote value (yes, no, or abstain)
func (b *votingProcedureBuilder) WithVote(vote uint8) VotingProcedureBuilder {
	b.vote = vote
	b.voteSet = true
	return b
}

// WithAnchor sets the optional anchor for the voting procedure
func (b *votingProcedureBuilder) WithAnchor(
	url string,
	dataHash []byte,
) VotingProcedureBuilder {
	b.anchorURL = url
	b.dataHash = dataHash
	return b
}

// Build constructs a VotingProcedure from the builder state
func (b *votingProcedureBuilder) Build() (*lcommon.VotingProcedure, error) {
	// Require explicit vote setting to avoid unintentional defaults
	if !b.voteSet {
		return nil, errors.New(
			"vote is required; call WithVote(0), WithVote(1), or WithVote(2)",
		)
	}
	// Validate vote value (0=no, 1=yes, 2=abstain)
	if b.vote > 2 {
		return nil, fmt.Errorf(
			"invalid vote value %d, must be 0 (no), 1 (yes), or 2 (abstain)",
			b.vote,
		)
	}
	// Validate dataHash length if provided (Blake2b256 is 32 bytes)
	if len(b.dataHash) > 0 && len(b.dataHash) != 32 {
		return nil, fmt.Errorf(
			"dataHash must be exactly 32 bytes, got %d",
			len(b.dataHash),
		)
	}

	procedure := &lcommon.VotingProcedure{
		Vote: b.vote,
	}

	if b.anchorURL != "" {
		anchor := &lcommon.GovAnchor{
			Url: b.anchorURL,
		}
		if len(b.dataHash) > 0 {
			copy(anchor.DataHash[:], b.dataHash)
		}
		procedure.Anchor = anchor
	}

	return procedure, nil
}
