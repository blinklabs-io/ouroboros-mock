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

package localtxsubmission_test

import (
	"testing"

	gouroboros_localtxsubmission "github.com/blinklabs-io/gouroboros/protocol/localtxsubmission"
	"github.com/blinklabs-io/ouroboros-mock/localtxsubmission"
)

// TestConversationEntrySubmitTx tests the ConversationEntrySubmitTx entry type
func TestConversationEntrySubmitTx(t *testing.T) {
	t.Run("ToEntry with TxCbor", func(t *testing.T) {
		entry := localtxsubmission.ConversationEntrySubmitTx{
			EraId:  localtxsubmission.MockEraIdConway,
			TxCbor: localtxsubmission.MockTxCbor,
		}
		result := entry.ToEntry()

		if result.ProtocolId != gouroboros_localtxsubmission.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				gouroboros_localtxsubmission.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse != false {
			t.Errorf("expected IsResponse to be false, got true")
		}
		if result.MessageType != gouroboros_localtxsubmission.MessageTypeSubmitTx {
			t.Errorf(
				"expected MessageType %d, got %d",
				gouroboros_localtxsubmission.MessageTypeSubmitTx,
				result.MessageType,
			)
		}
		if result.Message == nil {
			t.Error("expected Message to be set when TxCbor is provided")
		}
		if result.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})

	t.Run("ToEntry without TxCbor", func(t *testing.T) {
		entry := localtxsubmission.ConversationEntrySubmitTx{}
		result := entry.ToEntry()

		if result.ProtocolId != gouroboros_localtxsubmission.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				gouroboros_localtxsubmission.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse != false {
			t.Errorf("expected IsResponse to be false, got true")
		}
		if result.MessageType != gouroboros_localtxsubmission.MessageTypeSubmitTx {
			t.Errorf(
				"expected MessageType %d, got %d",
				gouroboros_localtxsubmission.MessageTypeSubmitTx,
				result.MessageType,
			)
		}
		if result.Message != nil {
			t.Error("expected Message to be nil when TxCbor is not provided")
		}
		if result.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})
}

// TestConversationEntryAcceptTx tests the ConversationEntryAcceptTx entry type
func TestConversationEntryAcceptTx(t *testing.T) {
	entry := localtxsubmission.ConversationEntryAcceptTx{}
	result := entry.ToEntry()

	if result.ProtocolId != gouroboros_localtxsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxsubmission.ProtocolId,
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

// TestConversationEntryRejectTx tests the ConversationEntryRejectTx entry type
func TestConversationEntryRejectTx(t *testing.T) {
	entry := localtxsubmission.ConversationEntryRejectTx{
		ReasonCbor: localtxsubmission.MockRejectReasonCbor,
	}
	result := entry.ToEntry()

	if result.ProtocolId != gouroboros_localtxsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxsubmission.ProtocolId,
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

// TestConversationEntryDone tests the ConversationEntryDone entry type
func TestConversationEntryDone(t *testing.T) {
	entry := localtxsubmission.ConversationEntryDone{}
	result := entry.ToEntry()

	if result.ProtocolId != gouroboros_localtxsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxsubmission.ProtocolId,
			result.ProtocolId,
		)
	}
	// Done is sent by the client, not as a response
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxsubmission.MessageTypeDone {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxsubmission.MessageTypeDone,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestNewSubmitTxEntry tests the NewSubmitTxEntry helper function
func TestNewSubmitTxEntry(t *testing.T) {
	result := localtxsubmission.NewSubmitTxEntry(
		localtxsubmission.MockEraIdConway,
		localtxsubmission.MockTxCbor,
	)

	if result.ProtocolId != gouroboros_localtxsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxsubmission.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxsubmission.MessageTypeSubmitTx {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxsubmission.MessageTypeSubmitTx,
			result.MessageType,
		)
	}
	if result.Message == nil {
		t.Error("expected Message to be set")
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestNewSubmitTxEntryAny tests the NewSubmitTxEntryAny helper function
func TestNewSubmitTxEntryAny(t *testing.T) {
	result := localtxsubmission.NewSubmitTxEntryAny()

	if result.ProtocolId != gouroboros_localtxsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxsubmission.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxsubmission.MessageTypeSubmitTx {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxsubmission.MessageTypeSubmitTx,
			result.MessageType,
		)
	}
	if result.Message != nil {
		t.Error("expected Message to be nil for 'any' entry")
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestNewAcceptTxEntry tests the NewAcceptTxEntry helper function
func TestNewAcceptTxEntry(t *testing.T) {
	result := localtxsubmission.NewAcceptTxEntry()

	if result.ProtocolId != gouroboros_localtxsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxsubmission.ProtocolId,
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

// TestNewRejectTxEntry tests the NewRejectTxEntry helper function
func TestNewRejectTxEntry(t *testing.T) {
	result := localtxsubmission.NewRejectTxEntry(
		localtxsubmission.MockRejectReasonCbor,
	)

	if result.ProtocolId != gouroboros_localtxsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxsubmission.ProtocolId,
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

// TestNewDoneEntry tests the NewDoneEntry helper function
func TestNewDoneEntry(t *testing.T) {
	result := localtxsubmission.NewDoneEntry()

	if result.ProtocolId != gouroboros_localtxsubmission.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			gouroboros_localtxsubmission.ProtocolId,
			result.ProtocolId,
		)
	}
	// Done is sent by client, not a response
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != gouroboros_localtxsubmission.MessageTypeDone {
		t.Errorf(
			"expected MessageType %d, got %d",
			gouroboros_localtxsubmission.MessageTypeDone,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestMockConstants tests that the mock constants are defined correctly
func TestMockConstants(t *testing.T) {
	if localtxsubmission.MockEraIdConway != 6 {
		t.Errorf(
			"expected MockEraIdConway to be 6, got %d",
			localtxsubmission.MockEraIdConway,
		)
	}
	if localtxsubmission.MockEraIdBabbage != 5 {
		t.Errorf(
			"expected MockEraIdBabbage to be 5, got %d",
			localtxsubmission.MockEraIdBabbage,
		)
	}
	if len(localtxsubmission.MockTxCbor) == 0 {
		t.Error("expected MockTxCbor to be non-empty")
	}
	if len(localtxsubmission.MockRejectReasonCbor) == 0 {
		t.Error("expected MockRejectReasonCbor to be non-empty")
	}
	if len(localtxsubmission.MockRejectReasonInvalidScript) == 0 {
		t.Error("expected MockRejectReasonInvalidScript to be non-empty")
	}
}

// TestConversationFixtures tests that the pre-defined conversation fixtures are valid
func TestConversationFixtures(t *testing.T) {
	t.Run("ConversationLocalTxSubmissionAccept", func(t *testing.T) {
		if len(localtxsubmission.ConversationLocalTxSubmissionAccept) != 4 {
			t.Errorf(
				"expected 4 entries, got %d",
				len(localtxsubmission.ConversationLocalTxSubmissionAccept),
			)
		}
	})

	t.Run("ConversationLocalTxSubmissionReject", func(t *testing.T) {
		if len(localtxsubmission.ConversationLocalTxSubmissionReject) != 4 {
			t.Errorf(
				"expected 4 entries, got %d",
				len(localtxsubmission.ConversationLocalTxSubmissionReject),
			)
		}
	})

	t.Run("ConversationLocalTxSubmissionMultiple", func(t *testing.T) {
		if len(localtxsubmission.ConversationLocalTxSubmissionMultiple) != 8 {
			t.Errorf(
				"expected 8 entries, got %d",
				len(localtxsubmission.ConversationLocalTxSubmissionMultiple),
			)
		}
	})
}
