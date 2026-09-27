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

package fixtures_test

import (
	"bytes"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
)

func TestDijkstraBlockBuilderEncodesTransactionsAndCertificates(t *testing.T) {
	tx, err := fixtures.NewDijkstraTransactionBuilder().
		WithTxIsValid(false).
		Build()
	if err != nil {
		t.Fatalf("build Dijkstra transaction: %s", err)
	}
	prevHash := common.Blake2b256Hash([]byte("previous"))
	perasCertificate := []byte{0x01, 0x02, 0x03}
	block, err := fixtures.NewDijkstraBlockBuilder().
		WithBlockNumber(17).
		WithSlot(600).
		WithPreviousHash(prevHash).
		WithTransactions(*tx).
		WithPerasCertificate(perasCertificate).
		Build()
	if err != nil {
		t.Fatalf("build Dijkstra block: %s", err)
	}
	if block.BlockHeader.Body.BlockNumber != 17 ||
		block.BlockHeader.Body.Slot != 600 {
		t.Fatalf(
			"unexpected block point: number=%d slot=%d",
			block.BlockHeader.Body.BlockNumber,
			block.BlockHeader.Body.Slot,
		)
	}
	if block.BlockHeader.Body.PrevHash != prevHash {
		t.Fatalf("unexpected previous hash: %x", block.BlockHeader.Body.PrevHash)
	}
	if block.BlockHeader.Body.BlockBodyHash != block.BlockBody.Hash() {
		t.Fatal("header body hash does not match encoded body")
	}
	if block.BlockHeader.Body.BlockBodySize != uint64(
		len(block.BlockBody.Cbor()),
	) {
		t.Fatal("header body size does not match encoded body")
	}
	if len(block.BlockBody.Transactions) != 1 ||
		block.BlockBody.Transactions[0].TxIsValid {
		t.Fatal("Dijkstra block transaction validity flag was not preserved")
	}
	if !bytes.Equal(block.BlockBody.PerasCertificate, perasCertificate) {
		t.Fatalf("unexpected Peras certificate: %x", block.BlockBody.PerasCertificate)
	}
	decoded, err := ledger.NewBlockFromCbor(ledger.BlockTypeDijkstra, block.Cbor())
	if err != nil {
		t.Fatalf("decode built block through generic decoder: %s", err)
	}
	if decoded.Hash() != block.Hash() {
		t.Fatal("generic decoder changed the Dijkstra block hash")
	}

	// CIP-0164: a ranking block carries a Leios certificate or transactions,
	// not both, and gouroboros rejects a body with both when decoding.
	certified, err := fixtures.NewDijkstraBlockBuilder().
		WithPreviousHash(block.Hash()).
		WithLeiosCertificate(&dijkstra.DijkstraLeiosCertificate{
			Signers:             []byte{0x80},
			AggregatedSignature: make([]byte, common.LeiosBlsSignatureSize),
		}).
		Build()
	if err != nil {
		t.Fatalf("build certified Dijkstra block: %s", err)
	}
	if certified.BlockBody.LeiosCertificate == nil {
		t.Fatal("Leios certificate slot was not populated")
	}
	if len(certified.BlockBody.Transactions) != 0 {
		t.Fatal("certified Dijkstra block carries transactions")
	}
}

func TestGenerateConwayToDijkstraChain(t *testing.T) {
	blocks, err := fixtures.GenerateConwayToDijkstraChain(
		10, common.Blake2b256{}, 100, 3, 2, 2,
	)
	if err != nil {
		t.Fatalf("generate Conway to Dijkstra chain: %s", err)
	}
	if len(blocks) != 4 {
		t.Fatalf("unexpected block count: got %d, want 4", len(blocks))
	}
	if blocks[2].Type() != ledger.BlockTypeDijkstra {
		t.Fatalf("transition block type is %d, want Dijkstra", blocks[2].Type())
	}
	for i := 1; i < len(blocks); i++ {
		if blocks[i].PrevHash() != blocks[i-1].Hash() {
			t.Fatalf("block %d does not link to block %d", i, i-1)
		}
	}
	if blocks[2].BlockNumber() != blocks[1].BlockNumber()+1 {
		t.Fatal("Dijkstra block number did not continue from Conway")
	}
	if blocks[2].SlotNumber() != blocks[1].SlotNumber()+3 {
		t.Fatal("Dijkstra slot did not continue from Conway")
	}
}

func TestGenerateConwayToDijkstraChainRejectsOverflow(t *testing.T) {
	_, err := fixtures.GenerateConwayToDijkstraChain(
		^uint64(0), common.Blake2b256{}, 0, 1, 1, 1,
	)
	if err == nil {
		t.Fatal("expected block number overflow error")
	}
	_, err = fixtures.GenerateDijkstraChain(
		1, common.Blake2b256{}, ^uint64(0), 1, 2,
	)
	if err == nil {
		t.Fatal("expected Dijkstra slot overflow error")
	}
}

// dijkstraHeaderBodyFields decodes the header_body array of a Dijkstra block.
func dijkstraHeaderBodyFields(
	t *testing.T,
	block *dijkstra.DijkstraBlock,
) ([]cbor.RawMessage, []byte) {
	t.Helper()
	var blockFields []cbor.RawMessage
	if _, err := cbor.Decode(block.Cbor(), &blockFields); err != nil {
		t.Fatalf("decode block: %s", err)
	}
	if len(blockFields) != 2 {
		t.Fatalf("block has %d elements, want 2", len(blockFields))
	}
	var header []cbor.RawMessage
	if _, err := cbor.Decode(blockFields[0], &header); err != nil {
		t.Fatalf("decode header: %s", err)
	}
	if len(header) != 2 {
		t.Fatalf("header has %d elements, want 2", len(header))
	}
	var headerBody []cbor.RawMessage
	if _, err := cbor.Decode(header[0], &headerBody); err != nil {
		t.Fatalf("decode header body: %s", err)
	}
	var signature []byte
	if _, err := cbor.Decode(header[1], &signature); err != nil {
		t.Fatalf("decode header signature: %s", err)
	}
	return headerBody, signature
}

// The pinned Dijkstra CDDL header_body ends with
// block_body_contains_leios_cert : bool and eb_announcement / nil, and the
// header signature is a 448-byte kes_signature.
func TestDijkstraBlockBuilderEncodesPinnedHeaderShape(t *testing.T) {
	plain, err := fixtures.NewDijkstraBlockBuilder().Build()
	if err != nil {
		t.Fatalf("build Dijkstra block: %s", err)
	}
	headerBody, signature := dijkstraHeaderBodyFields(t, plain)
	if len(headerBody) != 12 {
		t.Fatalf("header body has %d fields, want 12", len(headerBody))
	}
	if !bytes.Equal(headerBody[10], []byte{0xf4}) {
		t.Fatalf("block_body_contains_leios_cert = %x, want false", headerBody[10])
	}
	if !bytes.Equal(headerBody[11], []byte{0xf6}) {
		t.Fatalf("eb_announcement = %x, want nil", headerBody[11])
	}
	if len(signature) != 448 {
		t.Fatalf("KES signature width = %d, want 448", len(signature))
	}
	if certified, present := plain.BlockHeader.LeiosCertified(); certified || !present {
		t.Fatalf("LeiosCertified() = %t, %t, want false, true", certified, present)
	}

	ebHash := common.Blake2b256Hash([]byte("endorser block"))
	certified, err := fixtures.NewDijkstraBlockBuilder().
		WithLeiosCertificate(&dijkstra.DijkstraLeiosCertificate{
			Signers:             []byte{0x80},
			AggregatedSignature: make([]byte, common.LeiosBlsSignatureSize),
		}).
		WithEbAnnouncement(ebHash, 4096).
		Build()
	if err != nil {
		t.Fatalf("build certified Dijkstra block: %s", err)
	}
	headerBody, _ = dijkstraHeaderBodyFields(t, certified)
	if len(headerBody) != 12 {
		t.Fatalf("header body has %d fields, want 12", len(headerBody))
	}
	if !bytes.Equal(headerBody[10], []byte{0xf5}) {
		t.Fatalf("block_body_contains_leios_cert = %x, want true", headerBody[10])
	}
	if flag, present := certified.BlockHeader.LeiosCertified(); !flag || !present {
		t.Fatalf("LeiosCertified() = %t, %t, want true, true", flag, present)
	}
	gotHash, gotSize, ok := certified.BlockHeader.LeiosAnnouncement()
	if !ok || gotHash != ebHash || gotSize != 4096 {
		t.Fatalf(
			"LeiosAnnouncement() = %s, %d, %t, want %s, 4096, true",
			gotHash,
			gotSize,
			ok,
			ebHash,
		)
	}
}
