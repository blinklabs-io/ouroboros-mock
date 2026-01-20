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

package txsubmission_test

import (
	"bytes"
	"testing"

	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
	ts "github.com/blinklabs-io/ouroboros-mock/txsubmission"
)

// TestConversationEntryInitToEntry tests that ConversationEntryInit.ToEntry()
// returns the correct ConversationEntryInput with proper protocol ID,
// IsResponse flag, and message type.
func TestConversationEntryInitToEntry(t *testing.T) {
	entry := ts.ConversationEntryInit{}
	input := entry.ToEntry()

	if input.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			input.ProtocolId,
		)
	}

	if input.IsResponse {
		t.Error("expected IsResponse to be false for Init entry")
	}

	if input.MessageType != txsubmission.MessageTypeInit {
		t.Errorf(
			"expected MessageType %d, got %d",
			txsubmission.MessageTypeInit,
			input.MessageType,
		)
	}

	if input.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestConversationEntryRequestTxIdsToEntry tests that
// ConversationEntryRequestTxIds.ToEntry() returns the correct
// ConversationEntryOutput with proper protocol ID and message.
func TestConversationEntryRequestTxIdsToEntry(t *testing.T) {
	entry := ts.ConversationEntryRequestTxIds{
		Blocking: true,
		Ack:      5,
		Req:      10,
	}
	output := entry.ToEntry()

	if output.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			output.ProtocolId,
		)
	}

	if !output.IsResponse {
		t.Error("expected IsResponse to be true for RequestTxIds entry")
	}

	if len(output.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(output.Messages))
	}

	if output.Messages[0].Type() != txsubmission.MessageTypeRequestTxIds {
		t.Errorf(
			"expected message type %d, got %d",
			txsubmission.MessageTypeRequestTxIds,
			output.Messages[0].Type(),
		)
	}

	msg, ok := output.Messages[0].(*txsubmission.MsgRequestTxIds)
	if !ok {
		t.Fatalf(
			"expected message type *MsgRequestTxIds, got %T",
			output.Messages[0],
		)
	}

	if !msg.Blocking {
		t.Error("expected Blocking to be true")
	}

	if msg.Ack != 5 {
		t.Errorf("expected Ack 5, got %d", msg.Ack)
	}

	if msg.Req != 10 {
		t.Errorf("expected Req 10, got %d", msg.Req)
	}
}

// TestConversationEntryReplyTxIdsToEntry tests that
// ConversationEntryReplyTxIds.ToEntry() returns the correct
// ConversationEntryInput with proper protocol ID and message type.
func TestConversationEntryReplyTxIdsToEntry(t *testing.T) {
	t.Run("with nil TxIds", func(t *testing.T) {
		entry := ts.ConversationEntryReplyTxIds{}
		input := entry.ToEntry()

		if input.ProtocolId != txsubmission.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				txsubmission.ProtocolId,
				input.ProtocolId,
			)
		}

		if !input.IsResponse {
			t.Error("expected IsResponse to be true for ReplyTxIds entry")
		}

		if input.MessageType != txsubmission.MessageTypeReplyTxIds {
			t.Errorf(
				"expected MessageType %d, got %d",
				txsubmission.MessageTypeReplyTxIds,
				input.MessageType,
			)
		}

		if input.Message != nil {
			t.Error("expected Message to be nil when TxIds is nil")
		}
	})

	t.Run("with TxIds", func(t *testing.T) {
		txIds := []txsubmission.TxIdAndSize{
			ts.MockTxIdAndSize1,
			ts.MockTxIdAndSize2,
		}
		entry := ts.ConversationEntryReplyTxIds{
			TxIds: txIds,
		}
		input := entry.ToEntry()

		if input.Message == nil {
			t.Fatal("expected Message to be set when TxIds is provided")
		}

		msg, ok := input.Message.(*txsubmission.MsgReplyTxIds)
		if !ok {
			t.Fatalf(
				"expected message type *MsgReplyTxIds, got %T",
				input.Message,
			)
		}

		if len(msg.TxIds) != 2 {
			t.Errorf("expected 2 TxIds, got %d", len(msg.TxIds))
		}
	})
}

// TestConversationEntryRequestTxsToEntry tests that
// ConversationEntryRequestTxs.ToEntry() returns the correct
// ConversationEntryOutput with proper protocol ID and message.
func TestConversationEntryRequestTxsToEntry(t *testing.T) {
	txIds := []txsubmission.TxId{ts.MockTxId1, ts.MockTxId2}
	entry := ts.ConversationEntryRequestTxs{
		TxIds: txIds,
	}
	output := entry.ToEntry()

	if output.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			output.ProtocolId,
		)
	}

	if !output.IsResponse {
		t.Error("expected IsResponse to be true for RequestTxs entry")
	}

	if len(output.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(output.Messages))
	}

	if output.Messages[0].Type() != txsubmission.MessageTypeRequestTxs {
		t.Errorf(
			"expected message type %d, got %d",
			txsubmission.MessageTypeRequestTxs,
			output.Messages[0].Type(),
		)
	}

	msg, ok := output.Messages[0].(*txsubmission.MsgRequestTxs)
	if !ok {
		t.Fatalf("expected message type *MsgRequestTxs, got %T", output.Messages[0])
	}

	if len(msg.TxIds) != 2 {
		t.Errorf("expected 2 TxIds, got %d", len(msg.TxIds))
	}
}

// TestConversationEntryReplyTxsToEntry tests that
// ConversationEntryReplyTxs.ToEntry() returns the correct
// ConversationEntryInput with proper protocol ID and message type.
func TestConversationEntryReplyTxsToEntry(t *testing.T) {
	t.Run("with nil Txs", func(t *testing.T) {
		entry := ts.ConversationEntryReplyTxs{}
		input := entry.ToEntry()

		if input.ProtocolId != txsubmission.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				txsubmission.ProtocolId,
				input.ProtocolId,
			)
		}

		if !input.IsResponse {
			t.Error("expected IsResponse to be true for ReplyTxs entry")
		}

		if input.MessageType != txsubmission.MessageTypeReplyTxs {
			t.Errorf(
				"expected MessageType %d, got %d",
				txsubmission.MessageTypeReplyTxs,
				input.MessageType,
			)
		}

		if input.Message != nil {
			t.Error("expected Message to be nil when Txs is nil")
		}
	})

	t.Run("with Txs", func(t *testing.T) {
		txs := []txsubmission.TxBody{ts.MockTx1, ts.MockTx2}
		entry := ts.ConversationEntryReplyTxs{
			Txs: txs,
		}
		input := entry.ToEntry()

		if input.Message == nil {
			t.Fatal("expected Message to be set when Txs is provided")
		}

		msg, ok := input.Message.(*txsubmission.MsgReplyTxs)
		if !ok {
			t.Fatalf("expected message type *MsgReplyTxs, got %T", input.Message)
		}

		if len(msg.Txs) != 2 {
			t.Errorf("expected 2 Txs, got %d", len(msg.Txs))
		}
	})
}

// TestConversationEntryDoneToEntry tests that ConversationEntryDone.ToEntry()
// returns the correct ConversationEntryInput with proper protocol ID,
// IsResponse flag, and message type.
func TestConversationEntryDoneToEntry(t *testing.T) {
	entry := ts.ConversationEntryDone{}
	input := entry.ToEntry()

	if input.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			input.ProtocolId,
		)
	}

	if input.IsResponse {
		t.Error("expected IsResponse to be false for Done entry")
	}

	if input.MessageType != txsubmission.MessageTypeDone {
		t.Errorf(
			"expected MessageType %d, got %d",
			txsubmission.MessageTypeDone,
			input.MessageType,
		)
	}

	if input.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestNewInitEntry tests the NewInitEntry helper function.
func TestNewInitEntry(t *testing.T) {
	input := ts.NewInitEntry()

	if input.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			input.ProtocolId,
		)
	}

	if input.IsResponse {
		t.Error("expected IsResponse to be false")
	}

	if input.MessageType != txsubmission.MessageTypeInit {
		t.Errorf(
			"expected MessageType %d, got %d",
			txsubmission.MessageTypeInit,
			input.MessageType,
		)
	}
}

// TestNewRequestTxIdsEntry tests the NewRequestTxIdsEntry helper function.
func TestNewRequestTxIdsEntry(t *testing.T) {
	output := ts.NewRequestTxIdsEntry(true, 5, 10)

	if output.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			output.ProtocolId,
		)
	}

	if !output.IsResponse {
		t.Error("expected IsResponse to be true")
	}

	if len(output.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(output.Messages))
	}

	msg, ok := output.Messages[0].(*txsubmission.MsgRequestTxIds)
	if !ok {
		t.Fatalf(
			"expected message type *MsgRequestTxIds, got %T",
			output.Messages[0],
		)
	}

	if !msg.Blocking {
		t.Error("expected Blocking to be true")
	}

	if msg.Ack != 5 {
		t.Errorf("expected Ack 5, got %d", msg.Ack)
	}

	if msg.Req != 10 {
		t.Errorf("expected Req 10, got %d", msg.Req)
	}
}

// TestNewReplyTxIdsEntry tests the NewReplyTxIdsEntry helper function.
func TestNewReplyTxIdsEntry(t *testing.T) {
	txIds := []txsubmission.TxIdAndSize{
		ts.MockTxIdAndSize1,
		ts.MockTxIdAndSize2,
		ts.MockTxIdAndSize3,
	}
	input := ts.NewReplyTxIdsEntry(txIds)

	if input.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			input.ProtocolId,
		)
	}

	if !input.IsResponse {
		t.Error("expected IsResponse to be true")
	}

	if input.Message == nil {
		t.Fatal("expected Message to be set")
	}

	msg, ok := input.Message.(*txsubmission.MsgReplyTxIds)
	if !ok {
		t.Fatalf("expected message type *MsgReplyTxIds, got %T", input.Message)
	}

	if len(msg.TxIds) != 3 {
		t.Errorf("expected 3 TxIds, got %d", len(msg.TxIds))
	}
}

// TestNewReplyTxIdsEntryAny tests the NewReplyTxIdsEntryAny helper function.
func TestNewReplyTxIdsEntryAny(t *testing.T) {
	input := ts.NewReplyTxIdsEntryAny()

	if input.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			input.ProtocolId,
		)
	}

	if !input.IsResponse {
		t.Error("expected IsResponse to be true")
	}

	if input.MessageType != txsubmission.MessageTypeReplyTxIds {
		t.Errorf(
			"expected MessageType %d, got %d",
			txsubmission.MessageTypeReplyTxIds,
			input.MessageType,
		)
	}

	if input.Message != nil {
		t.Error("expected Message to be nil for 'any' entry")
	}
}

// TestNewRequestTxsEntry tests the NewRequestTxsEntry helper function.
func TestNewRequestTxsEntry(t *testing.T) {
	txIds := []txsubmission.TxId{ts.MockTxId1, ts.MockTxId2}
	output := ts.NewRequestTxsEntry(txIds)

	if output.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			output.ProtocolId,
		)
	}

	if !output.IsResponse {
		t.Error("expected IsResponse to be true")
	}

	if len(output.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(output.Messages))
	}

	msg, ok := output.Messages[0].(*txsubmission.MsgRequestTxs)
	if !ok {
		t.Fatalf("expected message type *MsgRequestTxs, got %T", output.Messages[0])
	}

	if len(msg.TxIds) != 2 {
		t.Errorf("expected 2 TxIds, got %d", len(msg.TxIds))
	}
}

// TestNewReplyTxsEntry tests the NewReplyTxsEntry helper function.
func TestNewReplyTxsEntry(t *testing.T) {
	txs := []txsubmission.TxBody{ts.MockTx1, ts.MockTx2, ts.MockTx3}
	input := ts.NewReplyTxsEntry(txs)

	if input.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			input.ProtocolId,
		)
	}

	if !input.IsResponse {
		t.Error("expected IsResponse to be true")
	}

	if input.Message == nil {
		t.Fatal("expected Message to be set")
	}

	msg, ok := input.Message.(*txsubmission.MsgReplyTxs)
	if !ok {
		t.Fatalf("expected message type *MsgReplyTxs, got %T", input.Message)
	}

	if len(msg.Txs) != 3 {
		t.Errorf("expected 3 Txs, got %d", len(msg.Txs))
	}
}

// TestNewReplyTxsEntryAny tests the NewReplyTxsEntryAny helper function.
func TestNewReplyTxsEntryAny(t *testing.T) {
	input := ts.NewReplyTxsEntryAny()

	if input.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			input.ProtocolId,
		)
	}

	if !input.IsResponse {
		t.Error("expected IsResponse to be true")
	}

	if input.MessageType != txsubmission.MessageTypeReplyTxs {
		t.Errorf(
			"expected MessageType %d, got %d",
			txsubmission.MessageTypeReplyTxs,
			input.MessageType,
		)
	}

	if input.Message != nil {
		t.Error("expected Message to be nil for 'any' entry")
	}
}

// TestNewDoneEntry tests the NewDoneEntry helper function.
func TestNewDoneEntry(t *testing.T) {
	input := ts.NewDoneEntry()

	if input.ProtocolId != txsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			txsubmission.ProtocolId,
			input.ProtocolId,
		)
	}

	if input.IsResponse {
		t.Error("expected IsResponse to be false")
	}

	if input.MessageType != txsubmission.MessageTypeDone {
		t.Errorf(
			"expected MessageType %d, got %d",
			txsubmission.MessageTypeDone,
			input.MessageType,
		)
	}
}

// TestMockEraConway tests the MockEraConway constant.
func TestMockEraConway(t *testing.T) {
	if ts.MockEraConway != 6 {
		t.Errorf("expected MockEraConway to be 6, got %d", ts.MockEraConway)
	}
}

// TestMockTxHashes tests the mock transaction hash fixtures.
func TestMockTxHashes(t *testing.T) {
	t.Run("MockTxHash1", func(t *testing.T) {
		if len(ts.MockTxHash1) != 32 {
			t.Errorf(
				"expected MockTxHash1 length 32, got %d",
				len(ts.MockTxHash1),
			)
		}
		if ts.MockTxHash1[0] != 0x01 {
			t.Errorf(
				"expected MockTxHash1[0] to be 0x01, got %#x",
				ts.MockTxHash1[0],
			)
		}
		if ts.MockTxHash1[31] != 0x20 {
			t.Errorf(
				"expected MockTxHash1[31] to be 0x20, got %#x",
				ts.MockTxHash1[31],
			)
		}
	})

	t.Run("MockTxHash2", func(t *testing.T) {
		if len(ts.MockTxHash2) != 32 {
			t.Errorf(
				"expected MockTxHash2 length 32, got %d",
				len(ts.MockTxHash2),
			)
		}
		if ts.MockTxHash2[0] != 0x21 {
			t.Errorf(
				"expected MockTxHash2[0] to be 0x21, got %#x",
				ts.MockTxHash2[0],
			)
		}
		if ts.MockTxHash2[31] != 0x40 {
			t.Errorf(
				"expected MockTxHash2[31] to be 0x40, got %#x",
				ts.MockTxHash2[31],
			)
		}
	})

	t.Run("MockTxHash3", func(t *testing.T) {
		if len(ts.MockTxHash3) != 32 {
			t.Errorf(
				"expected MockTxHash3 length 32, got %d",
				len(ts.MockTxHash3),
			)
		}
		if ts.MockTxHash3[0] != 0x41 {
			t.Errorf(
				"expected MockTxHash3[0] to be 0x41, got %#x",
				ts.MockTxHash3[0],
			)
		}
		if ts.MockTxHash3[31] != 0x60 {
			t.Errorf(
				"expected MockTxHash3[31] to be 0x60, got %#x",
				ts.MockTxHash3[31],
			)
		}
	})

	t.Run("hashes are unique", func(t *testing.T) {
		if ts.MockTxHash1 == ts.MockTxHash2 {
			t.Error("MockTxHash1 and MockTxHash2 should be different")
		}
		if ts.MockTxHash2 == ts.MockTxHash3 {
			t.Error("MockTxHash2 and MockTxHash3 should be different")
		}
		if ts.MockTxHash1 == ts.MockTxHash3 {
			t.Error("MockTxHash1 and MockTxHash3 should be different")
		}
	})
}

// TestMockTxIds tests the mock transaction ID fixtures.
func TestMockTxIds(t *testing.T) {
	t.Run("MockTxId1", func(t *testing.T) {
		if ts.MockTxId1.EraId != ts.MockEraConway {
			t.Errorf(
				"expected MockTxId1.EraId to be %d, got %d",
				ts.MockEraConway,
				ts.MockTxId1.EraId,
			)
		}
		if ts.MockTxId1.TxId != ts.MockTxHash1 {
			t.Error("expected MockTxId1.TxId to equal MockTxHash1")
		}
	})

	t.Run("MockTxId2", func(t *testing.T) {
		if ts.MockTxId2.EraId != ts.MockEraConway {
			t.Errorf(
				"expected MockTxId2.EraId to be %d, got %d",
				ts.MockEraConway,
				ts.MockTxId2.EraId,
			)
		}
		if ts.MockTxId2.TxId != ts.MockTxHash2 {
			t.Error("expected MockTxId2.TxId to equal MockTxHash2")
		}
	})

	t.Run("MockTxId3", func(t *testing.T) {
		if ts.MockTxId3.EraId != ts.MockEraConway {
			t.Errorf(
				"expected MockTxId3.EraId to be %d, got %d",
				ts.MockEraConway,
				ts.MockTxId3.EraId,
			)
		}
		if ts.MockTxId3.TxId != ts.MockTxHash3 {
			t.Error("expected MockTxId3.TxId to equal MockTxHash3")
		}
	})
}

// TestMockTxSizes tests the mock transaction size constants.
func TestMockTxSizes(t *testing.T) {
	if ts.MockTxSize1 != 256 {
		t.Errorf("expected MockTxSize1 to be 256, got %d", ts.MockTxSize1)
	}

	if ts.MockTxSize2 != 512 {
		t.Errorf("expected MockTxSize2 to be 512, got %d", ts.MockTxSize2)
	}

	if ts.MockTxSize3 != 1024 {
		t.Errorf("expected MockTxSize3 to be 1024, got %d", ts.MockTxSize3)
	}
}

// TestMockTxIdAndSizes tests the mock TxIdAndSize fixtures.
func TestMockTxIdAndSizes(t *testing.T) {
	t.Run("MockTxIdAndSize1", func(t *testing.T) {
		if ts.MockTxIdAndSize1.TxId != ts.MockTxId1 {
			t.Error("expected MockTxIdAndSize1.TxId to equal MockTxId1")
		}
		if ts.MockTxIdAndSize1.Size != ts.MockTxSize1 {
			t.Errorf(
				"expected MockTxIdAndSize1.Size to be %d, got %d",
				ts.MockTxSize1,
				ts.MockTxIdAndSize1.Size,
			)
		}
	})

	t.Run("MockTxIdAndSize2", func(t *testing.T) {
		if ts.MockTxIdAndSize2.TxId != ts.MockTxId2 {
			t.Error("expected MockTxIdAndSize2.TxId to equal MockTxId2")
		}
		if ts.MockTxIdAndSize2.Size != ts.MockTxSize2 {
			t.Errorf(
				"expected MockTxIdAndSize2.Size to be %d, got %d",
				ts.MockTxSize2,
				ts.MockTxIdAndSize2.Size,
			)
		}
	})

	t.Run("MockTxIdAndSize3", func(t *testing.T) {
		if ts.MockTxIdAndSize3.TxId != ts.MockTxId3 {
			t.Error("expected MockTxIdAndSize3.TxId to equal MockTxId3")
		}
		if ts.MockTxIdAndSize3.Size != ts.MockTxSize3 {
			t.Errorf(
				"expected MockTxIdAndSize3.Size to be %d, got %d",
				ts.MockTxSize3,
				ts.MockTxIdAndSize3.Size,
			)
		}
	})
}

// TestMockTxBodies tests the mock transaction CBOR body fixtures.
func TestMockTxBodies(t *testing.T) {
	t.Run("MockTxBody1", func(t *testing.T) {
		if len(ts.MockTxBody1) == 0 {
			t.Error("expected MockTxBody1 to have non-zero length")
		}
		// Check CBOR array tag (0x84 = array of 4 elements)
		if ts.MockTxBody1[0] != 0x84 {
			t.Errorf(
				"expected MockTxBody1[0] to be 0x84, got %#x",
				ts.MockTxBody1[0],
			)
		}
	})

	t.Run("MockTxBody2", func(t *testing.T) {
		if len(ts.MockTxBody2) == 0 {
			t.Error("expected MockTxBody2 to have non-zero length")
		}
		// Check CBOR array tag (0x84 = array of 4 elements)
		if ts.MockTxBody2[0] != 0x84 {
			t.Errorf(
				"expected MockTxBody2[0] to be 0x84, got %#x",
				ts.MockTxBody2[0],
			)
		}
	})

	t.Run("MockTxBody3", func(t *testing.T) {
		if len(ts.MockTxBody3) == 0 {
			t.Error("expected MockTxBody3 to have non-zero length")
		}
		// Check CBOR array tag (0x84 = array of 4 elements)
		if ts.MockTxBody3[0] != 0x84 {
			t.Errorf(
				"expected MockTxBody3[0] to be 0x84, got %#x",
				ts.MockTxBody3[0],
			)
		}
	})

	t.Run("bodies are unique", func(t *testing.T) {
		// Check each pair separately to ensure all are unique
		if bytes.Equal(ts.MockTxBody1, ts.MockTxBody2) {
			t.Error("MockTxBody1 and MockTxBody2 should be different")
		}
		if bytes.Equal(ts.MockTxBody2, ts.MockTxBody3) {
			t.Error("MockTxBody2 and MockTxBody3 should be different")
		}
		if bytes.Equal(ts.MockTxBody1, ts.MockTxBody3) {
			t.Error("MockTxBody1 and MockTxBody3 should be different")
		}
	})
}

// TestMockTxs tests the mock TxBody fixtures.
func TestMockTxs(t *testing.T) {
	t.Run("MockTx1", func(t *testing.T) {
		if ts.MockTx1.EraId != ts.MockEraConway {
			t.Errorf(
				"expected MockTx1.EraId to be %d, got %d",
				ts.MockEraConway,
				ts.MockTx1.EraId,
			)
		}
		if len(ts.MockTx1.TxBody) == 0 {
			t.Error("expected MockTx1.TxBody to have non-zero length")
		}
	})

	t.Run("MockTx2", func(t *testing.T) {
		if ts.MockTx2.EraId != ts.MockEraConway {
			t.Errorf(
				"expected MockTx2.EraId to be %d, got %d",
				ts.MockEraConway,
				ts.MockTx2.EraId,
			)
		}
		if len(ts.MockTx2.TxBody) == 0 {
			t.Error("expected MockTx2.TxBody to have non-zero length")
		}
	})

	t.Run("MockTx3", func(t *testing.T) {
		if ts.MockTx3.EraId != ts.MockEraConway {
			t.Errorf(
				"expected MockTx3.EraId to be %d, got %d",
				ts.MockEraConway,
				ts.MockTx3.EraId,
			)
		}
		if len(ts.MockTx3.TxBody) == 0 {
			t.Error("expected MockTx3.TxBody to have non-zero length")
		}
	})
}

// TestConversationTxSubmissionBasic tests that the basic conversation fixture
// is a valid slice with expected entries.
func TestConversationTxSubmissionBasic(t *testing.T) {
	if ts.ConversationTxSubmissionBasic == nil {
		t.Fatal("ConversationTxSubmissionBasic should not be nil")
	}

	if len(ts.ConversationTxSubmissionBasic) == 0 {
		t.Fatal("ConversationTxSubmissionBasic should not be empty")
	}

	// Expected entry count: 2 handshake + 7 tx submission = 9
	expectedLen := 9
	if len(ts.ConversationTxSubmissionBasic) != expectedLen {
		t.Errorf(
			"expected %d entries, got %d",
			expectedLen,
			len(ts.ConversationTxSubmissionBasic),
		)
	}

	// Verify all entries are not nil
	for i, entry := range ts.ConversationTxSubmissionBasic {
		if entry == nil {
			t.Errorf("entry %d should not be nil", i)
		}
	}
}

// TestConversationTxSubmissionMultipleTxs tests that the multiple transactions
// conversation fixture is a valid slice with expected entries.
func TestConversationTxSubmissionMultipleTxs(t *testing.T) {
	if ts.ConversationTxSubmissionMultipleTxs == nil {
		t.Fatal("ConversationTxSubmissionMultipleTxs should not be nil")
	}

	if len(ts.ConversationTxSubmissionMultipleTxs) == 0 {
		t.Fatal("ConversationTxSubmissionMultipleTxs should not be empty")
	}

	// Expected entry count: 2 handshake + 11 tx submission = 13
	expectedLen := 13
	if len(ts.ConversationTxSubmissionMultipleTxs) != expectedLen {
		t.Errorf(
			"expected %d entries, got %d",
			expectedLen,
			len(ts.ConversationTxSubmissionMultipleTxs),
		)
	}

	// Verify all entries are not nil
	for i, entry := range ts.ConversationTxSubmissionMultipleTxs {
		if entry == nil {
			t.Errorf("entry %d should not be nil", i)
		}
	}
}

// TestConversationTxSubmissionEmpty tests that the empty conversation fixture
// is a valid slice with expected entries.
func TestConversationTxSubmissionEmpty(t *testing.T) {
	if ts.ConversationTxSubmissionEmpty == nil {
		t.Fatal("ConversationTxSubmissionEmpty should not be nil")
	}

	if len(ts.ConversationTxSubmissionEmpty) == 0 {
		t.Fatal("ConversationTxSubmissionEmpty should not be empty")
	}

	// Expected entry count: 2 handshake + 5 tx submission = 7
	expectedLen := 7
	if len(ts.ConversationTxSubmissionEmpty) != expectedLen {
		t.Errorf(
			"expected %d entries, got %d",
			expectedLen,
			len(ts.ConversationTxSubmissionEmpty),
		)
	}

	// Verify all entries are not nil
	for i, entry := range ts.ConversationTxSubmissionEmpty {
		if entry == nil {
			t.Errorf("entry %d should not be nil", i)
		}
	}
}

// TestProtocolId verifies the protocol ID constant is as expected.
func TestProtocolId(t *testing.T) {
	// The TxSubmission protocol ID should be 4
	expectedProtocolId := uint16(4)
	if txsubmission.ProtocolId != expectedProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			expectedProtocolId,
			txsubmission.ProtocolId,
		)
	}
}

// TestMessageTypes verifies the message type constants are as expected.
func TestMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		actual   uint
		expected uint
	}{
		{"MessageTypeRequestTxIds", txsubmission.MessageTypeRequestTxIds, 0},
		{"MessageTypeReplyTxIds", txsubmission.MessageTypeReplyTxIds, 1},
		{"MessageTypeRequestTxs", txsubmission.MessageTypeRequestTxs, 2},
		{"MessageTypeReplyTxs", txsubmission.MessageTypeReplyTxs, 3},
		{"MessageTypeDone", txsubmission.MessageTypeDone, 4},
		{"MessageTypeInit", txsubmission.MessageTypeInit, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf(
					"expected %s to be %d, got %d",
					tt.name,
					tt.expected,
					tt.actual,
				)
			}
		})
	}
}

// TestMsgFromCborFunc verifies that the MsgFromCborFunc is set correctly
// in ConversationEntryInput entries.
func TestMsgFromCborFunc(t *testing.T) {
	t.Run("Init entry has MsgFromCborFunc", func(t *testing.T) {
		input := ts.NewInitEntry()
		if input.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})

	t.Run("ReplyTxIds entry has MsgFromCborFunc", func(t *testing.T) {
		input := ts.NewReplyTxIdsEntryAny()
		if input.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})

	t.Run("ReplyTxs entry has MsgFromCborFunc", func(t *testing.T) {
		input := ts.NewReplyTxsEntryAny()
		if input.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})

	t.Run("Done entry has MsgFromCborFunc", func(t *testing.T) {
		input := ts.NewDoneEntry()
		if input.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})
}
