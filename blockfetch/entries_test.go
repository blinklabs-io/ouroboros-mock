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

package blockfetch_test

import (
	"bytes"
	"testing"

	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	bf "github.com/blinklabs-io/ouroboros-mock/blockfetch"
)

// TestConversationEntryRequestRange_ToEntry tests that ConversationEntryRequestRange
// produces correct ConversationEntryInput values
func TestConversationEntryRequestRange_ToEntry(t *testing.T) {
	t.Run("with specific points", func(t *testing.T) {
		start := pcommon.NewPoint(100, bf.MockBlockHash1)
		end := pcommon.NewPoint(200, bf.MockBlockHash2)

		entry := bf.ConversationEntryRequestRange{
			Start: start,
			End:   end,
		}

		result := entry.ToEntry()

		if result.ProtocolId != blockfetch.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				blockfetch.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse != false {
			t.Errorf("expected IsResponse false, got %v", result.IsResponse)
		}
		if result.MessageType != blockfetch.MessageTypeRequestRange {
			t.Errorf(
				"expected MessageType %d, got %d",
				blockfetch.MessageTypeRequestRange,
				result.MessageType,
			)
		}
		if result.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
		if result.Message == nil {
			t.Error(
				"expected Message to be set when Start/End points are provided",
			)
		}
	})

	t.Run("without specific points", func(t *testing.T) {
		entry := bf.ConversationEntryRequestRange{}

		result := entry.ToEntry()

		if result.ProtocolId != blockfetch.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				blockfetch.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse != false {
			t.Errorf("expected IsResponse false, got %v", result.IsResponse)
		}
		if result.MessageType != blockfetch.MessageTypeRequestRange {
			t.Errorf(
				"expected MessageType %d, got %d",
				blockfetch.MessageTypeRequestRange,
				result.MessageType,
			)
		}
		if result.Message != nil {
			t.Error(
				"expected Message to be nil when no Start/End points are provided",
			)
		}
	})
}

// TestConversationEntryClientDone_ToEntry tests that ConversationEntryClientDone
// produces correct ConversationEntryInput values
func TestConversationEntryClientDone_ToEntry(t *testing.T) {
	entry := bf.ConversationEntryClientDone{}

	result := entry.ToEntry()

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse false, got %v", result.IsResponse)
	}
	if result.MessageType != blockfetch.MessageTypeClientDone {
		t.Errorf(
			"expected MessageType %d, got %d",
			blockfetch.MessageTypeClientDone,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestConversationEntryStartBatch_ToEntry tests that ConversationEntryStartBatch
// produces correct ConversationEntryOutput values
func TestConversationEntryStartBatch_ToEntry(t *testing.T) {
	entry := bf.ConversationEntryStartBatch{}

	result := entry.ToEntry()

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse true, got %v", result.IsResponse)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Messages[0] == nil {
		t.Error("expected Messages[0] to be set")
	}
}

// TestConversationEntryNoBlocks_ToEntry tests that ConversationEntryNoBlocks
// produces correct ConversationEntryOutput values
func TestConversationEntryNoBlocks_ToEntry(t *testing.T) {
	entry := bf.ConversationEntryNoBlocks{}

	result := entry.ToEntry()

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse true, got %v", result.IsResponse)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Messages[0] == nil {
		t.Error("expected Messages[0] to be set")
	}
}

// TestConversationEntryBlock_ToEntry tests that ConversationEntryBlock
// produces correct ConversationEntryOutput values
func TestConversationEntryBlock_ToEntry(t *testing.T) {
	testBlockCbor := []byte{
		0x82,
		0x06,
		0x82,
		0xa0,
		0x84,
		0x80,
		0x80,
		0x80,
		0xa0,
	}

	entry := bf.ConversationEntryBlock{
		BlockType: bf.MockBlockTypeConway,
		BlockCbor: testBlockCbor,
	}

	result := entry.ToEntry()

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse true, got %v", result.IsResponse)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Messages[0] == nil {
		t.Error("expected Messages[0] to be set")
	}
}

// TestConversationEntryBatchDone_ToEntry tests that ConversationEntryBatchDone
// produces correct ConversationEntryOutput values
func TestConversationEntryBatchDone_ToEntry(t *testing.T) {
	entry := bf.ConversationEntryBatchDone{}

	result := entry.ToEntry()

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse true, got %v", result.IsResponse)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Messages[0] == nil {
		t.Error("expected Messages[0] to be set")
	}
}

// TestNewRequestRangeEntry tests the helper function for creating RequestRange entries
func TestNewRequestRangeEntry(t *testing.T) {
	start := pcommon.NewPoint(100, bf.MockBlockHash1)
	end := pcommon.NewPoint(200, bf.MockBlockHash2)

	result := bf.NewRequestRangeEntry(start, end)

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse false, got %v", result.IsResponse)
	}
	if result.MessageType != blockfetch.MessageTypeRequestRange {
		t.Errorf(
			"expected MessageType %d, got %d",
			blockfetch.MessageTypeRequestRange,
			result.MessageType,
		)
	}
	if result.Message == nil {
		t.Error("expected Message to be set")
	}
}

// TestNewRequestRangeEntryAny tests the helper function for creating any RequestRange entries
func TestNewRequestRangeEntryAny(t *testing.T) {
	result := bf.NewRequestRangeEntryAny()

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse false, got %v", result.IsResponse)
	}
	if result.MessageType != blockfetch.MessageTypeRequestRange {
		t.Errorf(
			"expected MessageType %d, got %d",
			blockfetch.MessageTypeRequestRange,
			result.MessageType,
		)
	}
	if result.Message != nil {
		t.Error("expected Message to be nil for 'any' entry")
	}
}

// TestNewClientDoneEntry tests the helper function for creating ClientDone entries
func TestNewClientDoneEntry(t *testing.T) {
	result := bf.NewClientDoneEntry()

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse false, got %v", result.IsResponse)
	}
	if result.MessageType != blockfetch.MessageTypeClientDone {
		t.Errorf(
			"expected MessageType %d, got %d",
			blockfetch.MessageTypeClientDone,
			result.MessageType,
		)
	}
}

// TestNewStartBatchEntry tests the helper function for creating StartBatch entries
func TestNewStartBatchEntry(t *testing.T) {
	result := bf.NewStartBatchEntry()

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse true, got %v", result.IsResponse)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestNewNoBlocksEntry tests the helper function for creating NoBlocks entries
func TestNewNoBlocksEntry(t *testing.T) {
	result := bf.NewNoBlocksEntry()

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse true, got %v", result.IsResponse)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestNewBlockEntry tests the helper function for creating Block entries
func TestNewBlockEntry(t *testing.T) {
	testBlockCbor := []byte{
		0x82,
		0x06,
		0x82,
		0xa0,
		0x84,
		0x80,
		0x80,
		0x80,
		0xa0,
	}

	result := bf.NewBlockEntry(bf.MockBlockTypeConway, testBlockCbor)

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse true, got %v", result.IsResponse)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestNewBatchDoneEntry tests the helper function for creating BatchDone entries
func TestNewBatchDoneEntry(t *testing.T) {
	result := bf.NewBatchDoneEntry()

	if result.ProtocolId != blockfetch.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			blockfetch.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse true, got %v", result.IsResponse)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestConversationBlockFetchRange tests that the fixture is a valid slice
func TestConversationBlockFetchRange(t *testing.T) {
	fixture := bf.ConversationBlockFetchRange

	if fixture == nil {
		t.Fatal("ConversationBlockFetchRange should not be nil")
	}

	// Expected: handshake request, handshake response, request range,
	// start batch, 2 blocks, batch done = 7 entries
	expectedLen := 7
	if len(fixture) != expectedLen {
		t.Errorf("expected %d entries, got %d", expectedLen, len(fixture))
	}

	// Verify each entry is not nil
	for i, entry := range fixture {
		if entry == nil {
			t.Errorf("entry at index %d should not be nil", i)
		}
	}
}

// TestConversationBlockFetchNoBlocks tests that the fixture is a valid slice
func TestConversationBlockFetchNoBlocks(t *testing.T) {
	fixture := bf.ConversationBlockFetchNoBlocks

	if fixture == nil {
		t.Fatal("ConversationBlockFetchNoBlocks should not be nil")
	}

	// Expected: handshake request, handshake response, request range,
	// no blocks = 4 entries
	expectedLen := 4
	if len(fixture) != expectedLen {
		t.Errorf("expected %d entries, got %d", expectedLen, len(fixture))
	}

	// Verify each entry is not nil
	for i, entry := range fixture {
		if entry == nil {
			t.Errorf("entry at index %d should not be nil", i)
		}
	}
}

// TestConversationBlockFetchMultipleBatches tests that the fixture is a valid slice
func TestConversationBlockFetchMultipleBatches(t *testing.T) {
	fixture := bf.ConversationBlockFetchMultipleBatches

	if fixture == nil {
		t.Fatal("ConversationBlockFetchMultipleBatches should not be nil")
	}

	// Expected: handshake request, handshake response,
	// first batch: request range, start batch, block, batch done (4)
	// second batch: request range, start batch, block, block, batch done (5)
	// client done = 2 + 4 + 5 + 1 = 12 entries
	expectedLen := 12
	if len(fixture) != expectedLen {
		t.Errorf("expected %d entries, got %d", expectedLen, len(fixture))
	}

	// Verify each entry is not nil
	for i, entry := range fixture {
		if entry == nil {
			t.Errorf("entry at index %d should not be nil", i)
		}
	}
}

// TestMockConstants verifies the mock constants are set correctly
func TestMockConstants(t *testing.T) {
	t.Run("MockBlockTypeConway", func(t *testing.T) {
		if bf.MockBlockTypeConway != 6 {
			t.Errorf(
				"expected MockBlockTypeConway to be 6, got %d",
				bf.MockBlockTypeConway,
			)
		}
	})

	t.Run("MockBlockHashes", func(t *testing.T) {
		hashes := [][]byte{
			bf.MockBlockHash1,
			bf.MockBlockHash2,
			bf.MockBlockHash3,
			bf.MockBlockHash4,
		}
		for i, hash := range hashes {
			if len(hash) != 32 {
				t.Errorf(
					"MockBlockHash%d should be 32 bytes, got %d",
					i+1,
					len(hash),
				)
			}
		}

		// Verify hashes are distinct
		for i := range hashes {
			for j := i + 1; j < len(hashes); j++ {
				if bytes.Equal(hashes[i], hashes[j]) {
					t.Errorf(
						"MockBlockHash%d and MockBlockHash%d should be distinct",
						i+1,
						j+1,
					)
				}
			}
		}
	})

	t.Run("MockPoints", func(t *testing.T) {
		points := []pcommon.Point{
			bf.MockPoint1,
			bf.MockPoint2,
			bf.MockPoint3,
			bf.MockPoint4,
		}
		expectedSlots := []uint64{100, 200, 300, 400}

		for i, point := range points {
			if point.Slot != expectedSlots[i] {
				t.Errorf(
					"MockPoint%d.Slot should be %d, got %d",
					i+1,
					expectedSlots[i],
					point.Slot,
				)
			}
		}
	})

	t.Run("MockBlockCbor", func(t *testing.T) {
		cborData := [][]byte{
			bf.MockBlockCbor1,
			bf.MockBlockCbor2,
			bf.MockBlockCbor3,
		}
		for i, cbor := range cborData {
			if len(cbor) == 0 {
				t.Errorf("MockBlockCbor%d should not be empty", i+1)
			}
		}
	})
}
