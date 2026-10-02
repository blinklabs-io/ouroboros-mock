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
	"bytes"
	"errors"
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
)

// DijkstraBlockBuilder constructs a Dijkstra block in the pinned Dijkstra CDDL
// shape. It derives the header's body hash, body size, and
// block_body_contains_leios_cert flag from the encoded body.
type DijkstraBlockBuilder struct {
	blockNumber    uint64
	slot           uint64
	prevHash       common.Blake2b256
	transactions   []dijkstra.DijkstraTransaction
	leiosCert      *dijkstra.DijkstraLeiosCertificate
	perasCert      []byte
	issuerVkey     common.IssuerVkey
	vrfKey         []byte
	vrfResult      common.VrfResult
	opCert         babbage.BabbageOpCert
	protoVersion   babbage.BabbageProtoVersion
	signature      []byte
	ebAnnouncement *dijkstraEbAnnouncement
}

type dijkstraEbAnnouncement struct {
	cbor.StructAsArray
	Hash common.Blake2b256
	Size uint32
}

// dijkstraKesSignatureSize is the kes_signature width in the Dijkstra CDDL.
const dijkstraKesSignatureSize = 448

// NewDijkstraBlockBuilder creates a block builder with decodeable headers.
func NewDijkstraBlockBuilder() *DijkstraBlockBuilder {
	return &DijkstraBlockBuilder{
		vrfKey: make([]byte, 32),
		vrfResult: common.VrfResult{
			Output: make([]byte, 64),
			Proof:  make([]byte, 80),
		},
		opCert: babbage.BabbageOpCert{
			HotVkey:   make([]byte, 32),
			Signature: make([]byte, 64),
		},
		protoVersion: babbage.BabbageProtoVersion{
			Major: dijkstra.MinProtocolVersionDijkstra,
		},
		signature: make([]byte, dijkstraKesSignatureSize),
	}
}

// WithBlockNumber sets the block number.
func (b *DijkstraBlockBuilder) WithBlockNumber(
	number uint64,
) *DijkstraBlockBuilder {
	b.blockNumber = number
	return b
}

// WithSlot sets the block slot.
func (b *DijkstraBlockBuilder) WithSlot(slot uint64) *DijkstraBlockBuilder {
	b.slot = slot
	return b
}

// WithPreviousHash sets the previous block hash.
func (b *DijkstraBlockBuilder) WithPreviousHash(
	hash common.Blake2b256,
) *DijkstraBlockBuilder {
	b.prevHash = hash
	return b
}

// WithTransactions sets the block's non-segregated transactions. A
// transaction that carries block_transaction CBOR, such as one returned by
// DijkstraTransactionBuilder.Build or decoded from a Dijkstra block body, is
// encoded from those bytes. Any other transaction is encoded as
// DijkstraTransactionBuilder encodes it, from its body, witness set,
// auxiliary data or metadata, and validity flag. Build rejects a transaction
// whose encoding lacks a required body key, carries an empty sub-transaction
// set, or uses a witness set key outside the Dijkstra CDDL.
func (b *DijkstraBlockBuilder) WithTransactions(
	transactions ...dijkstra.DijkstraTransaction,
) *DijkstraBlockBuilder {
	b.transactions = append([]dijkstra.DijkstraTransaction(nil), transactions...)
	return b
}

// WithLeiosCertificate sets the optional Leios certificate body slot.
func (b *DijkstraBlockBuilder) WithLeiosCertificate(
	certificate *dijkstra.DijkstraLeiosCertificate,
) *DijkstraBlockBuilder {
	b.leiosCert = certificate
	return b
}

// WithPerasCertificate sets the optional Peras certificate body slot.
func (b *DijkstraBlockBuilder) WithPerasCertificate(
	certificate []byte,
) *DijkstraBlockBuilder {
	b.perasCert = bytes.Clone(certificate)
	return b
}

// WithIssuerVkey sets the issuer verification key.
func (b *DijkstraBlockBuilder) WithIssuerVkey(
	vkey common.IssuerVkey,
) *DijkstraBlockBuilder {
	b.issuerVkey = vkey
	return b
}

// WithVrfKey sets the VRF verification key.
func (b *DijkstraBlockBuilder) WithVrfKey(key []byte) *DijkstraBlockBuilder {
	b.vrfKey = bytes.Clone(key)
	return b
}

// WithVrfResult sets the VRF result.
func (b *DijkstraBlockBuilder) WithVrfResult(
	result common.VrfResult,
) *DijkstraBlockBuilder {
	result.Output = bytes.Clone(result.Output)
	result.Proof = bytes.Clone(result.Proof)
	b.vrfResult = result
	return b
}

// WithOpCert sets the operational certificate.
func (b *DijkstraBlockBuilder) WithOpCert(
	cert babbage.BabbageOpCert,
) *DijkstraBlockBuilder {
	cert.HotVkey = bytes.Clone(cert.HotVkey)
	cert.Signature = bytes.Clone(cert.Signature)
	b.opCert = cert
	return b
}

// WithProtocolVersion sets the header protocol version.
func (b *DijkstraBlockBuilder) WithProtocolVersion(
	version babbage.BabbageProtoVersion,
) *DijkstraBlockBuilder {
	b.protoVersion = version
	return b
}

// WithSignature sets the block header signature.
func (b *DijkstraBlockBuilder) WithSignature(
	signature []byte,
) *DijkstraBlockBuilder {
	b.signature = bytes.Clone(signature)
	return b
}

// WithEbAnnouncement sets the header's eb_announcement, the endorser block
// this ranking block announces. Without it the field is encoded as nil.
func (b *DijkstraBlockBuilder) WithEbAnnouncement(
	hash common.Blake2b256,
	size uint32,
) *DijkstraBlockBuilder {
	b.ebAnnouncement = &dijkstraEbAnnouncement{Hash: hash, Size: size}
	return b
}

// Build encodes the block and decodes it with gouroboros, so the returned
// block's CBOR, hash, and header fields are what a consumer decodes.
func (b *DijkstraBlockBuilder) Build() (*dijkstra.DijkstraBlock, error) {
	transactions := make([]cbor.RawMessage, len(b.transactions))
	for i := range b.transactions {
		encoded, err := dijkstraBlockTransactionCBOR(&b.transactions[i])
		if err != nil {
			return nil, fmt.Errorf("transaction %d: %w", i, err)
		}
		transactions[i] = encoded
	}
	var leiosCert any
	if b.leiosCert != nil {
		leiosCert = b.leiosCert
	}
	var perasCert any
	if b.perasCert != nil {
		perasCert = bytes.Clone(b.perasCert)
	}
	bodyCBOR, err := cbor.Encode([]any{transactions, leiosCert, perasCert})
	if err != nil {
		return nil, fmt.Errorf("encode Dijkstra block body fixture: %w", err)
	}
	headerBody := babbage.BabbageBlockHeaderBody{
		BlockNumber:   b.blockNumber,
		Slot:          b.slot,
		PrevHash:      b.prevHash,
		IssuerVkey:    b.issuerVkey,
		VrfKey:        bytes.Clone(b.vrfKey),
		VrfResult:     b.vrfResult,
		BlockBodySize: uint64(len(bodyCBOR)),
		BlockBodyHash: common.Blake2b256Hash(bodyCBOR),
		OpCert:        b.opCert,
		ProtoVersion:  b.protoVersion,
	}
	headerBodyCBOR, err := b.encodeHeaderBody(
		headerBody,
		b.leiosCert != nil,
	)
	if err != nil {
		return nil, err
	}
	headerCBOR, err := cbor.Encode([]any{
		cbor.RawMessage(headerBodyCBOR),
		bytes.Clone(b.signature),
	})
	if err != nil {
		return nil, fmt.Errorf("encode Dijkstra block header fixture: %w", err)
	}
	blockCBOR, err := cbor.Encode([]cbor.RawMessage{headerCBOR, bodyCBOR})
	if err != nil {
		return nil, fmt.Errorf("encode Dijkstra block fixture: %w", err)
	}
	decoded, err := dijkstra.NewDijkstraBlockFromCbor(blockCBOR)
	if err != nil {
		return nil, fmt.Errorf("decode Dijkstra block fixture: %w", err)
	}
	if !bytes.Equal(decoded.Cbor(), blockCBOR) {
		return nil, errors.New("Dijkstra block fixture changed during round-trip")
	}
	return decoded, nil
}

// encodeHeaderBody appends the Dijkstra header_body fields that follow
// protocol_version. gouroboros encodes an in-process DijkstraBlockHeader as a
// 10-field Babbage header and drops LeiosHeaderExtension, so the builder
// encodes the header body itself.
func (b *DijkstraBlockBuilder) encodeHeaderBody(
	headerBody babbage.BabbageBlockHeaderBody,
	containsLeiosCert bool,
) ([]byte, error) {
	babbageCBOR, err := cbor.Encode(&headerBody)
	if err != nil {
		return nil, fmt.Errorf("encode Dijkstra header body fixture: %w", err)
	}
	var babbageFields []cbor.RawMessage
	if _, err := cbor.Decode(babbageCBOR, &babbageFields); err != nil {
		return nil, fmt.Errorf("split Dijkstra header body fixture: %w", err)
	}
	flag, err := cbor.Encode(containsLeiosCert)
	if err != nil {
		return nil, fmt.Errorf("encode Leios certificate flag: %w", err)
	}
	var announcement any
	if b.ebAnnouncement != nil {
		announcement = b.ebAnnouncement
	}
	announcementCBOR, err := cbor.Encode(announcement)
	if err != nil {
		return nil, fmt.Errorf("encode EB announcement: %w", err)
	}
	encoded, err := cbor.Encode(
		append(babbageFields, flag, announcementCBOR),
	)
	if err != nil {
		return nil, fmt.Errorf("encode Dijkstra header body fixture: %w", err)
	}
	return encoded, nil
}
