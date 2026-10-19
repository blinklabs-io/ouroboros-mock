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

package fixtures

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"math/big"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/consensus"
	"github.com/blinklabs-io/gouroboros/kes"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

type tachysKESSigner struct {
	secret *kes.SecretKey
	public []byte
}

func (s *tachysKESSigner) Sign(message []byte) ([]byte, error) {
	return kes.Sign(s.secret, 0, message)
}

func (s *tachysKESSigner) PublicKey() []byte {
	return append([]byte(nil), s.public...)
}

func (s *tachysKESSigner) Period() uint64 {
	return 0
}

// TachysSignedHeaderFixture contains a decoded Babbage or Conway block and
// the full consensus validation input derived from its signed header.
type TachysSignedHeaderFixture struct {
	Block       ledger.Block
	Input       consensus.ValidateHeaderInput
	PoolID      common.PoolKeyHash
	EpochNonce  common.Blake2b256
	PraosPassed bool
}

// TachysSignedHeaderFixtures groups a scheduled producer, a different validly
// signed producer, and single-fault cryptographic validation inputs.
type TachysSignedHeaderFixtures struct {
	Scheduled     TachysSignedHeaderFixture
	WrongProducer TachysSignedHeaderFixture
	InvalidVRF    consensus.ValidateHeaderInput
	InvalidKES    consensus.ValidateHeaderInput
	InvalidOpCert consensus.ValidateHeaderInput
}

// NewTachysSignedHeaderFixtures builds deterministic signed fixtures for the
// Babbage or Conway era. Scheduled carries the pool ID for a one-pool Tachys
// distribution. WrongProducer has valid VRF, KES, and operational certificate
// cryptography under a different issuer.
func NewTachysSignedHeaderFixtures(
	era common.Era,
) (TachysSignedHeaderFixtures, error) {
	if era.Id != 5 && era.Id != 6 {
		return TachysSignedHeaderFixtures{}, errors.New(
			"tachys signed fixtures require Babbage or Conway",
		)
	}
	scheduled, err := buildTachysSignedHeaderFixture(era, "scheduled")
	if err != nil {
		return TachysSignedHeaderFixtures{}, err
	}
	wrong, err := buildTachysSignedHeaderFixture(era, "wrong-producer")
	if err != nil {
		return TachysSignedHeaderFixtures{}, err
	}
	invalidVRF := cloneTachysValidateInput(scheduled.Input)
	invalidVRF.VrfProof[0] ^= 0xff
	invalidKES := cloneTachysValidateInput(scheduled.Input)
	invalidKES.KesSignature[0] ^= 0xff
	invalidOpCert := cloneTachysValidateInput(scheduled.Input)
	invalidOpCert.OpCertSignature[0] ^= 0xff
	return TachysSignedHeaderFixtures{
		Scheduled:     scheduled,
		WrongProducer: wrong,
		InvalidVRF:    invalidVRF,
		InvalidKES:    invalidKES,
		InvalidOpCert: invalidOpCert,
	}, nil
}

func buildTachysSignedHeaderFixture(
	era common.Era,
	identity string,
) (TachysSignedHeaderFixture, error) {
	shape := eraBlockShapes[era.Id]
	bodies, err := cbor.Encode([]cbor.RawMessage{})
	if err != nil {
		return TachysSignedHeaderFixture{}, fmt.Errorf("encode bodies: %w", err)
	}
	witnesses, err := cbor.Encode([]cbor.RawMessage{})
	if err != nil {
		return TachysSignedHeaderFixture{}, fmt.Errorf("encode witnesses: %w", err)
	}
	metadata, err := cbor.Encode(map[uint]cbor.RawMessage{})
	if err != nil {
		return TachysSignedHeaderFixture{}, fmt.Errorf("encode metadata: %w", err)
	}
	invalid := []byte{0x80}
	bodyParts := [][]byte{bodies, witnesses, metadata, invalid}
	bodySize := computeBlockBodySize(bodyParts...)
	bodyHash := ComputeBlockBodyHash(bodyParts...)

	vrfSeed := common.Blake2b256Hash([]byte("Tachys VRF " + identity))
	vrfSigner, err := consensus.NewSimpleVRFSigner(vrfSeed[:])
	if err != nil {
		return TachysSignedHeaderFixture{}, err
	}
	kesSeed := common.Blake2b256Hash([]byte("Tachys KES " + identity))
	kesSecret, kesPublic, err := kes.KeyGen(kes.CardanoKesDepth, kesSeed[:])
	if err != nil {
		return TachysSignedHeaderFixture{}, err
	}
	kesSigner := &tachysKESSigner{secret: kesSecret, public: kesPublic}
	coldSeed := common.Blake2b256Hash([]byte("Tachys cold " + identity))
	coldPrivate := ed25519.NewKeyFromSeed(coldSeed[:])
	coldPublic := coldPrivate.Public().(ed25519.PublicKey)
	poolID := common.PoolKeyHash(common.Blake2b224Hash(coldPublic))
	opCertBody := common.OpCertSignableBytes(kesPublic, 0, 0)
	opCertSignature := ed25519.Sign(coldPrivate, opCertBody)
	opCert := &consensus.OperationalCert{
		HotVkey:   kesPublic,
		Signature: opCertSignature,
	}
	builder := consensus.NewBlockBuilder(
		vrfSigner,
		kesSigner,
		opCert,
		poolID[:],
		coldPublic,
		big.NewRat(1, 1),
	)
	epochNonce := common.Blake2b256Hash([]byte("Tachys fixture epoch nonce"))
	prevHash := common.Blake2b256Hash([]byte("Tachys fixture predecessor"))
	slot := uint64(100 + era.Id)
	header, _, err := builder.BuildHeader(consensus.BuildHeaderInput{
		Slot:          slot,
		BlockNumber:   1,
		PrevHash:      prevHash[:],
		EpochNonce:    epochNonce[:],
		PoolStake:     1,
		TotalStake:    1,
		BlockBodyHash: bodyHash[:],
		BlockBodySize: bodySize,
		ProtoMajor:    shape.protocolMajor,
	})
	if err != nil {
		return TachysSignedHeaderFixture{}, err
	}
	issuer := common.IssuerVkey{}
	copy(issuer[:], header.Body.IssuerVkey)
	typedBody := babbage.BabbageBlockHeaderBody{
		BlockNumber: header.Body.BlockNumber,
		Slot:        header.Body.Slot,
		PrevHash:    common.NewBlake2b256(header.Body.PrevHash),
		IssuerVkey:  issuer,
		VrfKey:      append([]byte(nil), header.Body.VrfKey...),
		VrfResult: common.VrfResult{
			Output: append([]byte(nil), header.Body.VrfOutput...),
			Proof:  append([]byte(nil), header.Body.VrfProof...),
		},
		BlockBodySize: header.Body.BlockBodySize,
		BlockBodyHash: common.NewBlake2b256(header.Body.BlockBodyHash),
		OpCert: babbage.BabbageOpCert{
			HotVkey:        append([]byte(nil), header.Body.OpCertHotVkey...),
			SequenceNumber: header.Body.OpCertSequenceNumber,
			KesPeriod:      header.Body.OpCertKesPeriod,
			Signature:      append([]byte(nil), header.Body.OpCertSignature...),
		},
		ProtoVersion: babbage.BabbageProtoVersion{
			Major: header.Body.ProtoMajor,
			Minor: header.Body.ProtoMinor,
		},
	}
	typedHeader := babbage.BabbageBlockHeader{
		Body:      typedBody,
		Signature: append([]byte(nil), header.Signature...),
	}
	headerCbor, err := cbor.Encode(typedHeader)
	if err != nil {
		return TachysSignedHeaderFixture{}, fmt.Errorf("encode signed header: %w", err)
	}
	fields := []cbor.RawMessage{headerCbor, bodies, witnesses, metadata, invalid}
	fixtureBuilder := NewBlockBuilder(era)
	block, err := fixtureBuilder.decodeEncoded(fields, func(data []byte) (ledger.Block, error) {
		return shape.decode(data, common.VerifyConfig{})
	})
	if err != nil {
		return TachysSignedHeaderFixture{}, err
	}
	bodyCbor, err := cbor.Encode(typedBody)
	if err != nil {
		return TachysSignedHeaderFixture{}, err
	}
	input := consensus.ValidateHeaderInput{
		Slot:                 slot,
		BlockNumber:          1,
		PrevHash:             prevHash[:],
		IssuerVkey:           append([]byte(nil), coldPublic...),
		VrfKey:               append([]byte(nil), header.Body.VrfKey...),
		VrfProof:             append([]byte(nil), header.Body.VrfProof...),
		VrfOutput:            append([]byte(nil), header.Body.VrfOutput...),
		KesSignature:         append([]byte(nil), header.Signature...),
		HeaderBodyCbor:       bodyCbor,
		OpCertHotVkey:        append([]byte(nil), kesPublic...),
		OpCertSequenceNumber: 0,
		OpCertKesPeriod:      0,
		OpCertSignature:      append([]byte(nil), opCertSignature...),
		PrevSlot:             slot - 1,
		PrevBlockNumber:      0,
		PrevHeaderHash:       prevHash[:],
		EpochNonce:           epochNonce[:],
		PoolStake:            1,
		TotalStake:           1,
		RegisteredVrfKeyHash: common.Blake2b256Hash(vrfSigner.PublicKey()).Bytes(),
	}
	praosPassed, err := consensus.IsSlotLeaderFromComponentsWithMode(
		input.VrfOutput,
		1,
		1,
		new(big.Rat).SetFrac(
			big.NewInt(1),
			new(big.Int).SetUint64(10_000_000_000_000_000_000),
		),
		consensus.ConsensusModeCPraos,
	)
	if err != nil {
		return TachysSignedHeaderFixture{}, err
	}
	return TachysSignedHeaderFixture{
		Block:       block,
		Input:       input,
		PoolID:      poolID,
		EpochNonce:  epochNonce,
		PraosPassed: praosPassed,
	}, nil
}

func cloneTachysValidateInput(
	input consensus.ValidateHeaderInput,
) consensus.ValidateHeaderInput {
	result := input
	result.PrevHash = append([]byte(nil), input.PrevHash...)
	result.IssuerVkey = append([]byte(nil), input.IssuerVkey...)
	result.VrfKey = append([]byte(nil), input.VrfKey...)
	result.VrfProof = append([]byte(nil), input.VrfProof...)
	result.VrfOutput = append([]byte(nil), input.VrfOutput...)
	result.KesSignature = append([]byte(nil), input.KesSignature...)
	result.HeaderBodyCbor = append([]byte(nil), input.HeaderBodyCbor...)
	result.OpCertHotVkey = append([]byte(nil), input.OpCertHotVkey...)
	result.OpCertSignature = append([]byte(nil), input.OpCertSignature...)
	result.PrevHeaderHash = append([]byte(nil), input.PrevHeaderHash...)
	result.EpochNonce = append([]byte(nil), input.EpochNonce...)
	result.RegisteredVrfKeyHash = append(
		[]byte(nil), input.RegisteredVrfKeyHash...,
	)
	return result
}
