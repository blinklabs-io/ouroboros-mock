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
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
)

// DijkstraBlockBuilder constructs a Dijkstra block and derives its body hash
// and size from the encoded body.
type DijkstraBlockBuilder struct {
	blockNumber     uint64
	slot            uint64
	prevHash        common.Blake2b256
	transactions    []dijkstra.DijkstraTransaction
	leiosCert       *dijkstra.DijkstraLeiosCertificate
	perasCert       []byte
	issuerVkey      common.IssuerVkey
	vrfKey          []byte
	vrfResult       common.VrfResult
	opCert          babbage.BabbageOpCert
	protoVersion    babbage.BabbageProtoVersion
	signature       []byte
	headerExtension []cbor.RawMessage
}

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
		signature: make([]byte, 64),
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

// WithTransactions sets the block's non-segregated transactions.
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

// WithLeiosHeaderExtension appends raw trailing header fields used by Leios.
func (b *DijkstraBlockBuilder) WithLeiosHeaderExtension(
	fields ...cbor.RawMessage,
) *DijkstraBlockBuilder {
	b.headerExtension = make([]cbor.RawMessage, len(fields))
	for i, field := range fields {
		b.headerExtension[i] = bytes.Clone(field)
	}
	return b
}

// Build encodes and decodes a block so its hash and CBOR match consumers.
func (b *DijkstraBlockBuilder) Build() (*dijkstra.DijkstraBlock, error) {
	body := dijkstra.DijkstraBlockBody{
		Transactions: append(
			[]dijkstra.DijkstraTransaction(nil), b.transactions...,
		),
		LeiosCertificate: b.leiosCert,
		PerasCertificate: bytes.Clone(b.perasCert),
	}
	bodyCBOR, err := cbor.Encode(body)
	if err != nil {
		return nil, fmt.Errorf("encode Dijkstra block body fixture: %w", err)
	}
	header := &dijkstra.DijkstraBlockHeader{
		BabbageBlockHeader: babbage.BabbageBlockHeader{
			Body: babbage.BabbageBlockHeaderBody{
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
			},
			Signature: bytes.Clone(b.signature),
		},
		LeiosHeaderExtension: append([]cbor.RawMessage(nil), b.headerExtension...),
	}
	block := &dijkstra.DijkstraBlock{BlockHeader: header, BlockBody: body}
	blockCBOR, err := cbor.Encode(block)
	if err != nil {
		return nil, fmt.Errorf("encode Dijkstra block fixture: %w", err)
	}
	decoded, err := dijkstra.NewDijkstraBlockFromCbor(blockCBOR)
	if err != nil {
		return nil, fmt.Errorf("decode Dijkstra block fixture: %w", err)
	}
	if !bytes.Equal(decoded.Cbor(), blockCBOR) {
		return nil, fmt.Errorf("Dijkstra block fixture changed during round-trip")
	}
	return decoded, nil
}
