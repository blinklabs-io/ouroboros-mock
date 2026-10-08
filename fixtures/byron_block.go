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
	"errors"
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// The reference decoders require an indefinite-length transaction list and
// body for an empty block, and [attributes] for the extra body data, so these
// are written as raw CBOR: re-encoding the structs would emit definite
// lengths and a null ExtraData.
var (
	byronEmptyTxPayload  = []byte{0x9f, 0xff}
	byronExtraBodyData   = []byte{0x81, 0xa0}
	byronEmptyDlgPayload = []byte{0x9f, 0xff}
	byronEmptyUpdPayload = []byte{0x82, 0x80, 0x9f, 0xff}
	// byronEmptySscPayload is a certificates payload holding an empty
	// tag-258 set; its proof hashes the empty canonical map.
	byronEmptySscPayload = []byte{0x82, 0x03, 0xd9, 0x01, 0x02, 0x80}
	byronEmptyCertsMap   = []byte{0xa0}
)

func (b *BlockBuilder) buildByron() (ledger.Block, error) {
	if len(b.transactions) > 0 {
		return nil, errors.New(
			"byron block builder does not accept transactions",
		)
	}
	if b.protoSet {
		return nil, errors.New("byron headers carry no protocol version")
	}
	if b.bodySize != nil || b.bodyHash != nil {
		return nil, errors.New("byron headers carry no body size or hash")
	}
	if b.byronMain {
		return b.buildByronMain()
	}
	return b.buildByronEBB()
}

func (b *BlockBuilder) buildByronEBB() (ledger.Block, error) {
	if b.slot%byron.ByronSlotsPerEpoch != 0 {
		return nil, fmt.Errorf(
			"byron epoch boundary block slot %d is not epoch-aligned",
			b.slot,
		)
	}
	header := &byron.ByronEpochBoundaryBlockHeader{
		ProtocolMagic: byron.TestnetProtocolMagic,
		PrevBlock:     b.prevHash,
		BodyProof:     common.Blake2b256Hash(byronEmptyTxPayload).Bytes(),
		ExtraData:     []any{map[uint8][]byte{}},
	}
	header.ConsensusData.Epoch = b.slot / byron.ByronSlotsPerEpoch
	header.ConsensusData.Difficulty.Value = b.blockNumber
	headerCbor, err := cbor.Encode(header)
	if err != nil {
		return nil, fmt.Errorf("encode byron EBB header: %w", err)
	}
	return b.decodeEncoded(
		[]cbor.RawMessage{headerCbor, byronEmptyTxPayload, byronExtraBodyData},
		func(data []byte) (ledger.Block, error) {
			return byron.NewByronEpochBoundaryBlockFromCbor(data)
		},
	)
}

func (b *BlockBuilder) buildByronMain() (ledger.Block, error) {
	header := &byron.ByronMainBlockHeader{
		ProtocolMagic: byron.TestnetProtocolMagic,
		PrevBlock:     b.prevHash,
		BodyProof: []any{
			[]any{
				uint64(0),
				byron.MerkleRoot(nil).Bytes(),
				common.Blake2b256Hash(byronEmptyTxPayload).Bytes(),
			},
			[]any{
				uint64(byron.SscTypeCertificates),
				common.Blake2b256Hash(byronEmptyCertsMap).Bytes(),
			},
			common.Blake2b256Hash(byronEmptyDlgPayload).Bytes(),
			common.Blake2b256Hash(byronEmptyUpdPayload).Bytes(),
		},
	}
	header.ConsensusData.SlotId.Epoch = b.slot / byron.ByronSlotsPerEpoch
	header.ConsensusData.SlotId.Slot = b.slot % byron.ByronSlotsPerEpoch
	header.ConsensusData.PubKey = make([]byte, 64)
	header.ConsensusData.Difficulty.Value = b.blockNumber
	header.ConsensusData.BlockSig = []any{uint64(0), make([]byte, 64)}
	header.ExtraData.BlockVersion = byron.ByronBlockVersion{}
	header.ExtraData.SoftwareVersion = byron.ByronSoftwareVersion{
		Name: "cardano-sl",
	}
	header.ExtraData.Attributes = map[uint8][]byte{}
	header.ExtraData.ExtraProof = make([]byte, common.Blake2b256Size)
	headerCbor, err := cbor.Encode(header)
	if err != nil {
		return nil, fmt.Errorf("encode byron main block header: %w", err)
	}
	body := []byte{0x84}
	for _, part := range [][]byte{
		byronEmptyTxPayload,
		byronEmptySscPayload,
		byronEmptyDlgPayload,
		byronEmptyUpdPayload,
	} {
		body = append(body, part...)
	}
	return b.decodeEncoded(
		[]cbor.RawMessage{headerCbor, body, byronExtraBodyData},
		func(data []byte) (ledger.Block, error) {
			return byron.NewByronMainBlockFromCbor(data)
		},
	)
}
