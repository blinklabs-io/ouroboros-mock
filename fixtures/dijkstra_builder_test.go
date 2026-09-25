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

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	mockledger "github.com/blinklabs-io/ouroboros-mock/ledger"
)

func TestDijkstraBlockBuilderEncodesTransactionsAndCertificates(t *testing.T) {
	tx, err := mockledger.NewDijkstraTransactionBuilder().
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
		WithLeiosCertificate(&dijkstra.DijkstraLeiosCertificate{
			Signers:             []byte{0x80},
			AggregatedSignature: make([]byte, common.LeiosBlsSignatureSize),
		}).
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
	if block.BlockBody.LeiosCertificate == nil {
		t.Fatal("Leios certificate slot was not populated")
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
