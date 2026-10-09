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
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"golang.org/x/crypto/blake2b"
)

// decodeBlock decodes a built block the way a downstream consumer does.
func decodeBlock(t *testing.T, block ledger.Block) ledger.Block {
	t.Helper()
	var (
		decoded ledger.Block
		err     error
	)
	if block.Type() == ledger.BlockTypeDijkstra {
		// The generic decoder cannot classify the 12-field Dijkstra header.
		decoded, err = dijkstra.NewDijkstraBlockFromCbor(block.Cbor())
	} else {
		decoded, err = ledger.NewBlockFromCbor(
			uint(block.Type()), block.Cbor(),
		)
	}
	if err != nil {
		t.Fatalf("decode built block: %v", err)
	}
	if decoded.Hash() != block.Hash() {
		t.Fatal("decoded block hash differs from the built block")
	}
	return decoded
}

func TestBlockBuilderSetsHeaderFields(t *testing.T) {
	t.Parallel()
	prev := common.NewBlake2b256(bytes.Repeat([]byte{0xab}, 32))
	var issuer common.IssuerVkey
	copy(issuer[:], bytes.Repeat([]byte{0xcd}, len(issuer)))
	for _, era := range fixtures.SupportedEras() {
		t.Run(era.Name, func(t *testing.T) {
			t.Parallel()
			slot := uint64(43200)
			block, err := fixtures.NewBlockBuilder(era).
				WithBlockNumber(77).
				WithSlot(slot).
				WithPreviousHash(prev).
				WithIssuerVkey(issuer).
				Build()
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			decoded := decodeBlock(t, block)
			if decoded.BlockNumber() != 77 || decoded.SlotNumber() != slot {
				t.Fatalf(
					"block %d at slot %d, want 77 at %d",
					decoded.BlockNumber(), decoded.SlotNumber(), slot,
				)
			}
			if decoded.PrevHash() != prev {
				t.Fatalf("prev hash %s, want %s", decoded.PrevHash(), prev)
			}
			wantIssuer := issuer
			if era.Id == byron.EraIdByron {
				wantIssuer = common.IssuerVkey{}
			}
			if decoded.IssuerVkey() != wantIssuer {
				t.Fatalf("issuer %x, want %x", decoded.IssuerVkey(), wantIssuer)
			}
			header, err := fixtures.NewBlockBuilder(era).
				WithBlockNumber(77).
				WithSlot(slot).
				WithPreviousHash(prev).
				BuildHeader()
			if err != nil {
				t.Fatalf("build header: %v", err)
			}
			if header.BlockNumber() != 77 || header.PrevHash() != prev {
				t.Fatal("header does not carry the requested fields")
			}
		})
	}
}

func TestBlockBuilderBodyHashAndSizeMatchEncodedBody(t *testing.T) {
	t.Parallel()
	for _, era := range fixtures.SupportedEras() {
		if era.Id == byron.EraIdByron || era.Id == dijkstra.EraIdDijkstra {
			continue
		}
		t.Run(era.Name, func(t *testing.T) {
			t.Parallel()
			block, err := fixtures.NewBlockBuilder(era).WithSlot(5).Build()
			if err != nil {
				t.Fatal(err)
			}
			var fields []cbor.RawMessage
			if _, err := cbor.Decode(block.Cbor(), &fields); err != nil {
				t.Fatal(err)
			}
			var hashes []byte
			var size uint64
			for _, part := range fields[1:] {
				sum := blake2b.Sum256(part)
				hashes = append(hashes, sum[:]...)
				size += uint64(len(part))
			}
			want := blake2b.Sum256(hashes)
			header := block.Header()
			if got := header.BlockBodyHash(); got != common.NewBlake2b256(
				want[:],
			) {
				t.Fatalf("body hash %s, want %x", got, want)
			}
			if got := header.BlockBodySize(); got != size {
				t.Fatalf("body size %d, want %d", got, size)
			}
		})
	}
}

func TestBlockBuilderProtocolVersion(t *testing.T) {
	t.Parallel()
	for _, id := range []uint8{1, 4, 5, 6} {
		era := ledger.GetEraById(id)
		t.Run(era.Name, func(t *testing.T) {
			t.Parallel()
			major := uint64(21)
			if id == 6 {
				major = 10
			}
			block, err := fixtures.NewBlockBuilder(era).
				WithProtocolVersion(major, 3).Build()
			if err != nil {
				t.Fatal(err)
			}
			var header [2]cbor.RawMessage
			if _, err := cbor.Decode(block.Header().Cbor(), &header); err != nil {
				t.Fatal(err)
			}
			var body []cbor.RawMessage
			if _, err := cbor.Decode(header[0], &body); err != nil {
				t.Fatal(err)
			}
			// Shelley-style headers end in major/minor fields after the
			// operational certificate; Praos headers end in their pair.
			var version []uint64
			if _, err := cbor.Decode(body[len(body)-1], &version); err != nil {
				var major, minor uint64
				if _, err := cbor.Decode(body[len(body)-2], &major); err != nil {
					t.Fatalf("locate protocol version: %v", err)
				}
				if _, err := cbor.Decode(body[len(body)-1], &minor); err != nil {
					t.Fatalf("locate protocol version: %v", err)
				}
				version = []uint64{major, minor}
			}
			if len(version) != 2 || version[0] != major || version[1] != 3 {
				t.Fatalf("protocol version %v, want [%d 3]", version, major)
			}
		})
	}
}

// testInputs returns a transaction input set that encodes as an untagged
// array, as the Shelley through Alonzo decoders require.
func testInputs() shelley.ShelleyTransactionInputSet {
	return shelley.NewShelleyTransactionInputSet(
		[]shelley.ShelleyTransactionInput{{}},
	)
}

func shelleyTransaction() *shelley.ShelleyTransaction {
	return &shelley.ShelleyTransaction{
		Body: shelley.ShelleyTransactionBody{
			TxInputs: testInputs(),
			TxFee:    170000,
		},
	}
}

func alonzoTransaction(valid bool, fee uint64) *alonzo.AlonzoTransaction {
	return &alonzo.AlonzoTransaction{
		Body: alonzo.AlonzoTransactionBody{
			TxInputs: testInputs(),
			TxFee:    fee,
		},
		TxIsValid: valid,
	}
}

func conwayTransaction(valid bool, fee uint64) *conway.ConwayTransaction {
	return &conway.ConwayTransaction{
		Body:      conway.ConwayTransactionBody{TxFee: fee},
		TxIsValid: valid,
	}
}

func TestBlockBuilderCarriesTransactions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		era  uint8
		txs  []common.Transaction
		fees []uint64
	}{
		{
			"Shelley", 1,
			[]common.Transaction{shelleyTransaction(), shelleyTransaction()},
			[]uint64{170000, 170000},
		},
		{
			"Alonzo", 4,
			[]common.Transaction{
				alonzoTransaction(true, 11),
				alonzoTransaction(true, 12),
			},
			[]uint64{11, 12},
		},
		{
			"Conway", 6,
			[]common.Transaction{
				conwayTransaction(true, 21),
				conwayTransaction(true, 22),
			},
			[]uint64{21, 22},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			block, err := fixtures.NewBlockBuilder(ledger.GetEraById(tt.era)).
				WithBlockNumber(1).WithSlot(9).
				WithTransactions(tt.txs...).Build()
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			decoded := decodeBlock(t, block)
			got := decoded.Transactions()
			if len(got) != len(tt.fees) {
				t.Fatalf("%d transactions, want %d", len(got), len(tt.fees))
			}
			for i, tx := range got {
				if tx.Fee().Uint64() != tt.fees[i] {
					t.Errorf(
						"transaction %d fee %s, want %d",
						i,
						tx.Fee(),
						tt.fees[i],
					)
				}
			}
			empty, err := fixtures.NewBlockBuilder(ledger.GetEraById(tt.era)).
				WithBlockNumber(1).WithSlot(9).Build()
			if err != nil {
				t.Fatal(err)
			}
			if block.Header().
				BlockBodyHash() ==
				empty.Header().
					BlockBodyHash() ||
				block.Header().
					BlockBodySize() <=
					empty.Header().
						BlockBodySize() {
				t.Fatal("header body hash and size ignore the transactions")
			}
		})
	}
}

func TestBlockBuilderRecordsInvalidTransactions(t *testing.T) {
	t.Parallel()
	for _, era := range []uint8{4, 6} {
		var txs []common.Transaction
		if era == 4 {
			txs = []common.Transaction{
				alonzoTransaction(true, 1), alonzoTransaction(false, 2),
			}
		} else {
			txs = []common.Transaction{
				conwayTransaction(true, 1), conwayTransaction(false, 2),
			}
		}
		block, err := fixtures.NewBlockBuilder(ledger.GetEraById(era)).
			WithTransactions(txs...).Build()
		if err != nil {
			t.Fatalf("era %d: build: %v", era, err)
		}
		got := block.Transactions()
		if len(got) != 2 || !got[0].IsValid() || got[1].IsValid() {
			t.Fatalf("era %d: validity flags not preserved", era)
		}
	}
}

func TestBlockBuilderRejectsMismatchedTransactions(t *testing.T) {
	t.Parallel()
	if _, err := fixtures.NewBlockBuilder(ledger.GetEraById(1)).
		WithTransactions(conwayTransaction(true, 1)).Build(); err == nil {
		t.Fatal("a Conway transaction was accepted in a Shelley block")
	}
	if _, err := fixtures.NewBlockBuilder(ledger.GetEraById(6)).
		WithTransactions(shelleyTransaction()).Build(); err == nil {
		t.Fatal("a Shelley transaction was accepted in a Conway block")
	}
	if _, err := fixtures.NewBlockBuilder(ledger.GetEraById(byron.EraIdByron)).
		WithTransactions(shelleyTransaction()).Build(); err == nil {
		t.Fatal("Byron accepted transactions")
	}
	if _, err := fixtures.NewBlockBuilder(ledger.GetEraById(dijkstra.EraIdDijkstra)).
		WithTransactions(conwayTransaction(true, 1)).Build(); err == nil {
		t.Fatal("Dijkstra accepted a Conway transaction")
	}
}

func TestBlockBuilderRejectsNilTransactions(t *testing.T) {
	t.Parallel()
	var conwayTx *conway.ConwayTransaction
	_, err := fixtures.NewBlockBuilder(ledger.GetEraById(conway.EraIdConway)).
		WithTransactions(conwayTx).
		Build()
	if err == nil || err.Error() != "transaction 0: transaction is nil" {
		t.Fatalf("Conway nil transaction error = %v", err)
	}

	var dijkstraTx *dijkstra.DijkstraTransaction
	_, err = fixtures.NewBlockBuilder(ledger.GetEraById(dijkstra.EraIdDijkstra)).
		WithTransactions(dijkstraTx).
		Build()
	if err == nil || err.Error() != "transaction 0 is nil" {
		t.Fatalf("Dijkstra nil transaction error = %v", err)
	}
}

func TestBlockBuilderByronMainBlock(t *testing.T) {
	t.Parallel()
	era := ledger.GetEraById(byron.EraIdByron)
	slot := uint64(3*byron.ByronSlotsPerEpoch + 17)
	prev := common.NewBlake2b256(bytes.Repeat([]byte{0x11}, 32))
	block, err := fixtures.NewBlockBuilder(era).
		WithByronMainBlock().
		WithBlockNumber(9).WithSlot(slot).WithPreviousHash(prev).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if block.Type() != ledger.BlockTypeByronMain {
		t.Fatalf("block type %d, want Byron main", block.Type())
	}
	decoded := decodeBlock(t, block)
	if decoded.SlotNumber() != slot || decoded.BlockNumber() != 9 ||
		decoded.PrevHash() != prev {
		t.Fatalf(
			"slot %d number %d prev %s", decoded.SlotNumber(),
			decoded.BlockNumber(), decoded.PrevHash(),
		)
	}
	main, ok := decoded.(*byron.ByronMainBlock)
	if !ok {
		t.Fatalf("decoded %T, want *byron.ByronMainBlock", decoded)
	}
	if err := main.ValidateBodyProof(common.VerifyConfig{
		EnableByronSscProofHashValidation: true,
	}); err != nil {
		t.Fatalf("body proof: %v", err)
	}
}

func TestBlockBuilderByronRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	era := ledger.GetEraById(byron.EraIdByron)
	if _, err := fixtures.NewBlockBuilder(era).WithSlot(1).Build(); err == nil {
		t.Fatal("misaligned Byron epoch boundary slot was accepted")
	}
	if _, err := fixtures.NewBlockBuilder(era).
		WithProtocolVersion(1, 0).Build(); err == nil {
		t.Fatal("Byron accepted a protocol version")
	}
	if _, err := fixtures.NewBlockBuilder(common.Era{Id: 99, Name: "Nope"}).
		Build(); err == nil {
		t.Fatal("unsupported era was accepted")
	}
}

func TestSequenceBuildsConnectedBlocks(t *testing.T) {
	t.Parallel()
	for _, era := range fixtures.SupportedEras() {
		t.Run(era.Name, func(t *testing.T) {
			t.Parallel()
			seq := fixtures.NewSequence(era).
				WithStart(10, 2*byron.ByronSlotsPerEpoch, common.Blake2b256{})
			if era.Id != byron.EraIdByron {
				seq.WithSlotIncrement(3)
			}
			blocks, err := seq.Blocks(4)
			if err != nil {
				t.Fatal(err)
			}
			next, err := seq.Next()
			if err != nil {
				t.Fatal(err)
			}
			blocks = append(blocks, next)
			step := uint64(3)
			if era.Id == byron.EraIdByron {
				step = byron.ByronSlotsPerEpoch
			}
			for i, block := range blocks {
				decoded := decodeBlock(t, block)
				if decoded.BlockNumber() != 10+uint64(i) ||
					decoded.SlotNumber() != 2*byron.ByronSlotsPerEpoch+uint64(
						i,
					)*step {
					t.Fatalf(
						"block %d is number %d slot %d",
						i, decoded.BlockNumber(), decoded.SlotNumber(),
					)
				}
				if i > 0 && decoded.PrevHash() != blocks[i-1].Hash() {
					t.Fatalf("block %d does not link to block %d", i, i-1)
				}
			}
		})
	}
	empty, err := fixtures.NewSequence(ledger.GetEraById(6)).Blocks(0)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("Blocks(0) = %v, %v; want empty non-nil", empty, err)
	}
}

func TestRandomBlockIsDeterministicPerSeed(t *testing.T) {
	t.Parallel()
	for _, era := range fixtures.SupportedEras() {
		t.Run(era.Name, func(t *testing.T) {
			t.Parallel()
			a, err := fixtures.RandomBlock(era, 42)
			if err != nil {
				t.Fatal(err)
			}
			b, err := fixtures.RandomBlock(era, 42)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(a.Cbor(), b.Cbor()) {
				t.Fatal("the same seed produced different blocks")
			}
			c, err := fixtures.RandomBlock(era, 43)
			if err != nil {
				t.Fatal(err)
			}
			if a.Hash() == c.Hash() {
				t.Fatal("different seeds produced the same block")
			}
			decodeBlock(t, a)
		})
	}
}

func TestGenesisBlockStartsAtOrigin(t *testing.T) {
	t.Parallel()
	for _, era := range fixtures.SupportedEras() {
		block, err := fixtures.GenesisBlock(era)
		if err != nil {
			t.Fatalf("%s: %v", era.Name, err)
		}
		if block.BlockNumber() != 0 || block.SlotNumber() != 0 ||
			block.PrevHash() != (common.Blake2b256{}) {
			t.Fatalf("%s genesis block is not at the origin", era.Name)
		}
	}
}

func TestBlockBuilderOverridesBodySizeAndHash(t *testing.T) {
	t.Parallel()
	wantHash := common.NewBlake2b256(bytes.Repeat([]byte{0x5a}, 32))
	for _, id := range []uint8{1, 2, 3, 4, 5, 6} {
		era := ledger.GetEraById(id)
		t.Run(era.Name, func(t *testing.T) {
			t.Parallel()
			block, err := fixtures.NewBlockBuilder(era).
				WithBodySize(12345).WithBodyHash(wantHash).Build()
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			header := block.Header()
			if header.BlockBodySize() != 12345 ||
				header.BlockBodyHash() != wantHash {
				t.Fatalf(
					"body size %d hash %s, want 12345 %s",
					header.BlockBodySize(), header.BlockBodyHash(), wantHash,
				)
			}
			if _, err := ledger.NewBlockFromCbor(
				uint(block.Type()), block.Cbor(),
			); err == nil {
				t.Fatal("a header that disagrees with its body was accepted")
			}
		})
	}
	for _, id := range []uint8{byron.EraIdByron, dijkstra.EraIdDijkstra} {
		era := ledger.GetEraById(id)
		if _, err := fixtures.NewBlockBuilder(era).
			WithBodySize(1).Build(); err == nil {
			t.Errorf("%s accepted a body size override", era.Name)
		}
		if _, err := fixtures.NewBlockBuilder(era).
			WithBodyHash(wantHash).Build(); err == nil {
			t.Errorf("%s accepted a body hash override", era.Name)
		}
	}
}

func TestGenerateByronChainPreservesEmptyAndAlignmentContracts(t *testing.T) {
	t.Parallel()
	era := ledger.GetEraById(byron.EraIdByron)
	for _, count := range []int{-1, 0} {
		blocks, err := fixtures.GenerateChain(
			era,
			0,
			common.Blake2b256{},
			1,
			1,
			count,
		)
		if err != nil || blocks == nil || len(blocks) != 0 {
			t.Fatalf(
				"count%d returned %v, %v; want empty nonnil chain",
				count,
				blocks,
				err,
			)
		}
	}
	for _, slots := range [][2]uint64{{1, 0}, {0, 1}, {1, 1}} {
		_, err := fixtures.GenerateChain(
			era,
			0,
			common.Blake2b256{},
			slots[0],
			slots[1],
			1,
		)
		want := fmt.Sprintf(
			"byron fixture slots must be epoch-aligned: start=%d increment=%d",
			slots[0],
			slots[1],
		)
		if err == nil || err.Error() != want {
			t.Fatalf("alignment error %v, want%s", err, want)
		}
	}
	blocks, err := fixtures.GenerateChain(
		era,
		0,
		common.Blake2b256{},
		byron.ByronSlotsPerEpoch,
		byron.ByronSlotsPerEpoch,
		2,
	)
	if err != nil || len(blocks) != 2 {
		t.Fatalf("aligned chain: %v, %v", blocks, err)
	}
}

func TestSequenceRejectsNumberAndSlotOverflow(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		number, slot, increment uint64
		message                 string
	}{
		{"block_number", math.MaxUint64, 0, 1, "block number range overflows uint64"},
		{"slot", 0, math.MaxUint64, 1, "slot range overflows uint64"},
		{"slot_increment", 0, math.MaxUint64 - 1, 2, "slot range overflows uint64"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sequence := fixtures.NewSequence(ledger.GetEraById(1)).
				WithStart(tc.number, tc.slot, common.Blake2b256{}).
				WithSlotIncrement(tc.increment)
			first, err := sequence.Next()
			if err != nil {
				t.Fatal(err)
			}
			if first.BlockNumber() != tc.number || first.SlotNumber() != tc.slot {
				t.Fatal("first representable block changed")
			}
			block, err := sequence.Next()
			if err == nil || !strings.Contains(err.Error(), tc.message) || block != nil {
				t.Fatalf("overflow returned block=%T error=%v", block, err)
			}
			sequence.WithStart(0, 0, common.Blake2b256{})
			reset, err := sequence.Next()
			if err != nil {
				t.Fatal(err)
			}
			if reset.BlockNumber() != 0 || reset.SlotNumber() != 0 {
				t.Fatal("explicit start did not reset sequence")
			}
			blocks, err := fixtures.GenerateChain(ledger.GetEraById(1),
				tc.number, common.Blake2b256{}, tc.slot, tc.increment, 2)
			if err == nil || !strings.Contains(err.Error(), tc.message) || blocks != nil {
				t.Fatalf("overflow bulk returned %d blocks error=%v", len(blocks), err)
			}
		})
	}
}
