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

package localtxmonitor_test

import (
	"testing"

	gouroboros_localtxmonitor "github.com/blinklabs-io/gouroboros/protocol/localtxmonitor"
	"github.com/blinklabs-io/ouroboros-mock/localtxmonitor"
)

// TestConversationEntryAcquire tests the ConversationEntryAcquire entry type
func TestConversationEntryAcquire(t *testing.T) {
	entry := localtxmonitor.ConversationEntryAcquire{}
	result := entry.ToEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeAcquire {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeAcquire,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestConversationEntryAcquired tests the ConversationEntryAcquired entry type
func TestConversationEntryAcquired(t *testing.T) {
	entry := localtxmonitor.ConversationEntryAcquired{
		SlotNo: localtxmonitor.MockSlotNo,
	}
	result := entry.ToEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse to be true, got false")
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestConversationEntryAwaitAcquire tests the ConversationEntryAwaitAcquire entry type
func TestConversationEntryAwaitAcquire(t *testing.T) {
	entry := localtxmonitor.ConversationEntryAwaitAcquire{}
	result := entry.ToEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	// AwaitAcquire expects client to send Acquire message
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	// AwaitAcquire uses same MessageTypeAcquire as regular Acquire
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeAcquire {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeAcquire,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestConversationEntryHasTx tests the ConversationEntryHasTx entry type
func TestConversationEntryHasTx(t *testing.T) {
	t.Run("ToEntry with TxId", func(t *testing.T) {
		entry := localtxmonitor.ConversationEntryHasTx{
			TxId: localtxmonitor.MockTxId,
		}
		result := entry.ToEntry()

		if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				gouroboros_localtxmonitor.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse != false {
			t.Errorf("expected IsResponse to be false, got true")
		}
		if result.MessageType != gouroboros_localtxmonitor.MessageTypeHasTx {
			t.Errorf(
				"expected MessageType %d, got %d",
				gouroboros_localtxmonitor.MessageTypeHasTx,
				result.MessageType,
			)
		}
		if result.Message == nil {
			t.Error("expected Message to be set when TxId is provided")
		}
		if result.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})

	t.Run("ToEntry without TxId", func(t *testing.T) {
		entry := localtxmonitor.ConversationEntryHasTx{}
		result := entry.ToEntry()

		if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				gouroboros_localtxmonitor.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.Message != nil {
			t.Error("expected Message to be nil when TxId is not provided")
		}
	})
}

// TestConversationEntryHasTxResult tests the ConversationEntryHasTxResult entry type
func TestConversationEntryHasTxResult(t *testing.T) {
	t.Run("Result true", func(t *testing.T) {
		entry := localtxmonitor.ConversationEntryHasTxResult{Result: true}
		result := entry.ToEntry()

		if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				gouroboros_localtxmonitor.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse != true {
			t.Errorf("expected IsResponse to be true, got false")
		}
		if len(result.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(result.Messages))
		}
	})

	t.Run("Result false", func(t *testing.T) {
		entry := localtxmonitor.ConversationEntryHasTxResult{Result: false}
		result := entry.ToEntry()

		if result.IsResponse != true {
			t.Errorf("expected IsResponse to be true, got false")
		}
		if len(result.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(result.Messages))
		}
	})
}

// TestConversationEntryNextTx tests the ConversationEntryNextTx entry type
func TestConversationEntryNextTx(t *testing.T) {
	entry := localtxmonitor.ConversationEntryNextTx{}
	result := entry.ToEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeNextTx {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeNextTx,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestConversationEntryNextTxResult tests the ConversationEntryNextTxResult entry type
func TestConversationEntryNextTxResult(t *testing.T) {
	t.Run("with transaction", func(t *testing.T) {
		entry := localtxmonitor.ConversationEntryNextTxResult{
			EraId: localtxmonitor.MockEraIdConway,
			Tx:    localtxmonitor.MockTxCbor,
		}
		result := entry.ToEntry()

		if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				gouroboros_localtxmonitor.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse != true {
			t.Errorf("expected IsResponse to be true, got false")
		}
		if len(result.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(result.Messages))
		}
	})

	t.Run("empty result", func(t *testing.T) {
		entry := localtxmonitor.ConversationEntryNextTxResult{}
		result := entry.ToEntry()

		if result.IsResponse != true {
			t.Errorf("expected IsResponse to be true, got false")
		}
		if len(result.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(result.Messages))
		}
	})
}

// TestConversationEntryGetSizes tests the ConversationEntryGetSizes entry type
func TestConversationEntryGetSizes(t *testing.T) {
	entry := localtxmonitor.ConversationEntryGetSizes{}
	result := entry.ToEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeGetSizes {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeGetSizes,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestConversationEntrySizes tests the ConversationEntrySizes entry type
func TestConversationEntrySizes(t *testing.T) {
	entry := localtxmonitor.ConversationEntrySizes{
		Capacity:    localtxmonitor.MockMempoolCapacity,
		Size:        localtxmonitor.MockMempoolSize,
		NumberOfTxs: localtxmonitor.MockMempoolTxCount,
	}
	result := entry.ToEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse to be true, got false")
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestConversationEntryRelease tests the ConversationEntryRelease entry type
func TestConversationEntryRelease(t *testing.T) {
	entry := localtxmonitor.ConversationEntryRelease{}
	result := entry.ToEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeRelease {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeRelease,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestConversationEntryDone tests the ConversationEntryDone entry type
func TestConversationEntryDone(t *testing.T) {
	entry := localtxmonitor.ConversationEntryDone{}
	result := entry.ToEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	// Done is sent by the client, not as a response
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeDone {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeDone,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestNewAcquireEntry tests the NewAcquireEntry helper function
func TestNewAcquireEntry(t *testing.T) {
	result := localtxmonitor.NewAcquireEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeAcquire {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeAcquire,
			result.MessageType,
		)
	}
}

// TestNewAcquiredEntry tests the NewAcquiredEntry helper function
func TestNewAcquiredEntry(t *testing.T) {
	result := localtxmonitor.NewAcquiredEntry(localtxmonitor.MockSlotNo)

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse to be true, got false")
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestNewAwaitAcquireEntry tests the NewAwaitAcquireEntry helper function
func TestNewAwaitAcquireEntry(t *testing.T) {
	result := localtxmonitor.NewAwaitAcquireEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	// AwaitAcquire uses same MessageTypeAcquire as regular Acquire
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeAcquire {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeAcquire,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestNewHasTxEntry tests the NewHasTxEntry helper function
func TestNewHasTxEntry(t *testing.T) {
	result := localtxmonitor.NewHasTxEntry(localtxmonitor.MockTxId)

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeHasTx {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeHasTx,
			result.MessageType,
		)
	}
	if result.Message == nil {
		t.Error("expected Message to be set")
	}
}

// TestNewHasTxEntryAny tests the NewHasTxEntryAny helper function
func TestNewHasTxEntryAny(t *testing.T) {
	result := localtxmonitor.NewHasTxEntryAny()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.Message != nil {
		t.Error("expected Message to be nil for 'any' entry")
	}
}

// TestNewHasTxResultEntry tests the NewHasTxResultEntry helper function
func TestNewHasTxResultEntry(t *testing.T) {
	t.Run("result true", func(t *testing.T) {
		result := localtxmonitor.NewHasTxResultEntry(true)

		if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				gouroboros_localtxmonitor.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse != true {
			t.Errorf("expected IsResponse to be true, got false")
		}
		if len(result.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(result.Messages))
		}
	})

	t.Run("result false", func(t *testing.T) {
		result := localtxmonitor.NewHasTxResultEntry(false)

		if result.IsResponse != true {
			t.Errorf("expected IsResponse to be true, got false")
		}
	})
}

// TestNewNextTxEntry tests the NewNextTxEntry helper function
func TestNewNextTxEntry(t *testing.T) {
	result := localtxmonitor.NewNextTxEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeNextTx {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeNextTx,
			result.MessageType,
		)
	}
}

// TestNewNextTxResultEntry tests the NewNextTxResultEntry helper function
func TestNewNextTxResultEntry(t *testing.T) {
	result := localtxmonitor.NewNextTxResultEntry(
		localtxmonitor.MockEraIdConway,
		localtxmonitor.MockTxCbor,
	)

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse to be true, got false")
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestNewNextTxResultEmptyEntry tests the NewNextTxResultEmptyEntry helper function
func TestNewNextTxResultEmptyEntry(t *testing.T) {
	result := localtxmonitor.NewNextTxResultEmptyEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse to be true, got false")
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestNewGetSizesEntry tests the NewGetSizesEntry helper function
func TestNewGetSizesEntry(t *testing.T) {
	result := localtxmonitor.NewGetSizesEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeGetSizes {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeGetSizes,
			result.MessageType,
		)
	}
}

// TestNewSizesEntry tests the NewSizesEntry helper function
func TestNewSizesEntry(t *testing.T) {
	result := localtxmonitor.NewSizesEntry(
		localtxmonitor.MockMempoolCapacity,
		localtxmonitor.MockMempoolSize,
		localtxmonitor.MockMempoolTxCount,
	)

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != true {
		t.Errorf("expected IsResponse to be true, got false")
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestNewReleaseEntry tests the NewReleaseEntry helper function
func TestNewReleaseEntry(t *testing.T) {
	result := localtxmonitor.NewReleaseEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeRelease {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeRelease,
			result.MessageType,
		)
	}
}

// TestNewDoneEntry tests the NewDoneEntry helper function
func TestNewDoneEntry(t *testing.T) {
	result := localtxmonitor.NewDoneEntry()

	if result.ProtocolId != gouroboros_localtxmonitor.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxmonitor.ProtocolId,
			result.ProtocolId,
		)
	}
	// Done is sent by client, not a response
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxmonitor.MessageTypeDone {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxmonitor.MessageTypeDone,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestMockConstants tests that the mock constants are defined correctly
func TestMockConstants(t *testing.T) {
	if localtxmonitor.MockSlotNo != 100000 {
		t.Errorf(
			"expected MockSlotNo to be 100000, got %d",
			localtxmonitor.MockSlotNo,
		)
	}
	if localtxmonitor.MockEraIdConway != 6 {
		t.Errorf(
			"expected MockEraIdConway to be 6, got %d",
			localtxmonitor.MockEraIdConway,
		)
	}
	if localtxmonitor.MockMempoolCapacity != 178176 {
		t.Errorf(
			"expected MockMempoolCapacity to be 178176, got %d",
			localtxmonitor.MockMempoolCapacity,
		)
	}
	if localtxmonitor.MockMempoolSize != 4096 {
		t.Errorf(
			"expected MockMempoolSize to be 4096, got %d",
			localtxmonitor.MockMempoolSize,
		)
	}
	if localtxmonitor.MockMempoolTxCount != 1 {
		t.Errorf(
			"expected MockMempoolTxCount to be 1, got %d",
			localtxmonitor.MockMempoolTxCount,
		)
	}
	if len(localtxmonitor.MockTxId) != 32 {
		t.Errorf(
			"expected MockTxId to have 32 bytes, got %d",
			len(localtxmonitor.MockTxId),
		)
	}
	if len(localtxmonitor.MockTxId2) != 32 {
		t.Errorf(
			"expected MockTxId2 to have 32 bytes, got %d",
			len(localtxmonitor.MockTxId2),
		)
	}
	if len(localtxmonitor.MockTxCbor) == 0 {
		t.Error("expected MockTxCbor to be non-empty")
	}
	if len(localtxmonitor.MockTxCbor2) == 0 {
		t.Error("expected MockTxCbor2 to be non-empty")
	}
}

// TestConversationFixtures tests that the pre-defined conversation fixtures are valid
func TestConversationFixtures(t *testing.T) {
	t.Run("ConversationLocalTxMonitorBasic", func(t *testing.T) {
		if len(localtxmonitor.ConversationLocalTxMonitorBasic) != 13 {
			t.Errorf(
				"expected 13 entries, got %d",
				len(localtxmonitor.ConversationLocalTxMonitorBasic),
			)
		}
	})

	t.Run("ConversationLocalTxMonitorEmpty", func(t *testing.T) {
		if len(localtxmonitor.ConversationLocalTxMonitorEmpty) != 9 {
			t.Errorf(
				"expected 9 entries, got %d",
				len(localtxmonitor.ConversationLocalTxMonitorEmpty),
			)
		}
	})
}
