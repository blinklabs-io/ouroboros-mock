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
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
)

// Generated blocks must round-trip through ledger.NewBlockFromCbor with stable
// Hash()/PrevHash()/BlockNumber()/SlotNumber(), and link together via PrevHash.
func TestGenerateConwayChainRoundTrip(t *testing.T) {
	var origin common.Blake2b256
	gen, err := fixtures.GenerateConwayChain(1, origin, 0, 20, 5)
	if err != nil {
		t.Fatalf("GenerateConwayChain: %s", err)
	}
	if len(gen) != 5 {
		t.Fatalf("expected 5 blocks, got %d", len(gen))
	}
	for i, b := range gen {
		decoded, err := ledger.NewBlockFromCbor(uint(b.Type()), b.Cbor())
		if err != nil {
			t.Fatalf("block %d decode failed: %s", i, err)
		}
		if decoded.Hash() != b.Hash() {
			t.Fatalf(
				"block %d hash changed after round-trip: %s -> %s",
				i, b.Hash(), decoded.Hash(),
			)
		}
		if decoded.PrevHash() != b.PrevHash() {
			t.Fatalf(
				"block %d prev hash changed after round-trip: %s -> %s",
				i, b.PrevHash(), decoded.PrevHash(),
			)
		}
		if decoded.BlockNumber() != uint64(i+1) {
			t.Fatalf(
				"block %d unexpected block number %d",
				i, decoded.BlockNumber(),
			)
		}
		if decoded.SlotNumber() != uint64(i)*20 {
			t.Fatalf(
				"block %d unexpected slot %d",
				i, decoded.SlotNumber(),
			)
		}
	}
	for i := 1; i < len(gen); i++ {
		if gen[i].PrevHash() != gen[i-1].Hash() {
			t.Fatalf(
				"chain link mismatch at index %d: prev=%s, want=%s",
				i, gen[i].PrevHash(), gen[i-1].Hash(),
			)
		}
	}
}

// The first block's PrevHash must be the caller-supplied prevHash, so chains
// can be grafted onto an existing block (e.g. to build a fork).
func TestGenerateConwayChainHonorsPrevHash(t *testing.T) {
	root, err := fixtures.GenerateConwayChain(1, common.Blake2b256{}, 0, 20, 1)
	if err != nil {
		t.Fatalf("GenerateConwayChain root: %s", err)
	}
	fork, err := fixtures.GenerateConwayChain(2, root[0].Hash(), 20, 20, 2)
	if err != nil {
		t.Fatalf("GenerateConwayChain fork: %s", err)
	}
	if fork[0].PrevHash() != root[0].Hash() {
		t.Fatalf(
			"fork does not link to root: prev=%s, want=%s",
			fork[0].PrevHash(), root[0].Hash(),
		)
	}
	if fork[0].BlockNumber() != 2 {
		t.Fatalf("fork start block number = %d, want 2", fork[0].BlockNumber())
	}
}

// A non-positive count returns an empty, non-nil slice and no error.
func TestGenerateConwayChainEmpty(t *testing.T) {
	for _, count := range []int{0, -1} {
		gen, err := fixtures.GenerateConwayChain(
			1,
			common.Blake2b256{},
			0,
			1,
			count,
		)
		if err != nil {
			t.Fatalf("count %d: unexpected error %s", count, err)
		}
		if gen == nil {
			t.Fatalf("count %d: expected non-nil slice", count)
		}
		if len(gen) != 0 {
			t.Fatalf("count %d: expected empty slice, got %d", count, len(gen))
		}
	}
}

func TestGenerateConwayChainWithTransactionsRoundTrip(t *testing.T) {
	blocks, err := fixtures.GenerateConwayChainWithTransactions(
		10,
		common.Blake2b256{},
		100,
		20,
		3,
	)
	if err != nil {
		t.Fatalf("GenerateConwayChainWithTransactions: %s", err)
	}

	var transactionHashes []common.Blake2b256
	for i, block := range blocks {
		decoded, err := ledger.NewBlockFromCbor(uint(block.Type()), block.Cbor())
		if err != nil {
			t.Fatalf("block %d decode failed: %s", i, err)
		}
		transactions := decoded.Transactions()
		if len(transactions) != 1 {
			t.Fatalf("block %d has %d transactions, want 1", i, len(transactions))
		}
		outputs := transactions[0].Outputs()
		if len(outputs) != 1 {
			t.Fatalf("block %d transaction has %d outputs, want 1", i, len(outputs))
		}
		if got := outputs[0].Amount().Uint64(); got != 1_000_000+uint64(i) {
			t.Fatalf("block %d output amount = %d, want %d", i, got, 1_000_000+uint64(i))
		}
		addressBytes, err := outputs[0].Address().Bytes()
		if err != nil {
			t.Fatalf("block %d output address encode failed: %s", i, err)
		}
		if got := len(addressBytes); got != 1+common.AddressHashSize {
			t.Fatalf("block %d output address width = %d, want %d", i, got, 1+common.AddressHashSize)
		}

		wireTransaction, err := conway.NewConwayTransactionFromCbor(
			transactions[0].Cbor(),
		)
		if err != nil {
			t.Fatalf("block %d transaction decode failed: %s", i, err)
		}
		if len(wireTransaction.Outputs()) != 1 {
			t.Fatalf("block %d wire transaction has no output", i)
		}
		transactionHashes = append(transactionHashes, transactions[0].Hash())

		var fields []cbor.RawMessage
		if _, err := cbor.Decode(decoded.Cbor(), &fields); err != nil {
			t.Fatalf("block %d body decode failed: %s", i, err)
		}
		if len(fields) != 5 {
			t.Fatalf("block %d has %d body fields, want 5", i, len(fields))
		}
		if got := fixtures.ComputeBlockBodyHash(
			fields[1], fields[2], fields[3], fields[4],
		); got != decoded.BlockBodyHash() {
			t.Fatalf("block %d body hash mismatch", i)
		}
		var bodySize uint64
		for _, field := range fields[1:] {
			bodySize += uint64(len(field))
		}
		if got := decoded.BlockBodySize(); got != bodySize {
			t.Fatalf("block %d body size = %d, want %d", i, got, bodySize)
		}

		conwayBlock, ok := decoded.(*conway.ConwayBlock)
		if !ok {
			t.Fatalf("block %d decoded as %T, want Conway block", i, decoded)
		}
		if conwayBlock.BlockHeader == nil {
			t.Fatalf("block %d has no Conway header", i)
		}
	}
	if len(transactionHashes) != 3 {
		t.Fatalf("got %d transaction hashes, want 3", len(transactionHashes))
	}
	for i := 1; i < len(transactionHashes); i++ {
		if transactionHashes[i] == transactionHashes[i-1] {
			t.Fatalf("transaction hash %d equals previous transaction hash", i)
		}
	}
}

func TestGenerateConwayChainWithTransactionsDoesNotAliasBodies(t *testing.T) {
	blocks, err := fixtures.GenerateConwayChainWithTransactions(
		1,
		common.Blake2b256{},
		1,
		1,
		2,
	)
	if err != nil {
		t.Fatalf("GenerateConwayChainWithTransactions: %s", err)
	}
	first := blocks[0].(*conway.ConwayBlock)
	second := blocks[1].(*conway.ConwayBlock)
	secondAmount := second.TransactionBodies[0].TxOutputs[0].OutputAmount.Amount
	first.TransactionBodies[0].TxOutputs[0].OutputAmount.Amount++
	if got := second.TransactionBodies[0].TxOutputs[0].OutputAmount.Amount; got != secondAmount {
		t.Fatalf("mutating one block changed another block's transaction output")
	}
}

// ComputeBlockBodyHash is deterministic and order-sensitive.
func TestComputeBlockBodyHash(t *testing.T) {
	a := fixtures.ComputeBlockBodyHash([]byte("one"), []byte("two"))
	b := fixtures.ComputeBlockBodyHash([]byte("one"), []byte("two"))
	if a != b {
		t.Fatalf("hash not deterministic: %s != %s", a, b)
	}
	swapped := fixtures.ComputeBlockBodyHash([]byte("two"), []byte("one"))
	if a == swapped {
		t.Fatalf("hash should depend on part order")
	}
	if bytes.Equal(a.Bytes(), make([]byte, 32)) {
		t.Fatalf("hash should not be the zero value")
	}
}
