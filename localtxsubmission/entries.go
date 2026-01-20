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

package localtxsubmission

import (
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/localtxsubmission"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// ConversationEntrySubmitTx expects a SubmitTx message from the client
type ConversationEntrySubmitTx struct {
	// EraId is the era ID for transaction validation. Only used when TxCbor
	// is also provided. When TxCbor is nil, all messages are accepted
	// regardless of EraId.
	EraId uint16
	// TxCbor is the expected transaction CBOR for validation. If nil, any
	// SubmitTx message is accepted. If provided, the received message must
	// match both EraId and TxCbor exactly.
	TxCbor []byte
}

// ToEntry converts this LocalTxSubmission entry to a generic ConversationEntryInput
func (e ConversationEntrySubmitTx) ToEntry() ouroboros_mock.ConversationEntryInput {
	entry := ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localtxsubmission.ProtocolId,
		IsResponse:      false,
		MessageType:     localtxsubmission.MessageTypeSubmitTx,
		MsgFromCborFunc: localtxsubmission.NewMsgFromCbor,
	}
	if e.TxCbor != nil {
		entry.Message = localtxsubmission.NewMsgSubmitTx(e.EraId, e.TxCbor)
	}
	return entry
}

// ConversationEntryAcceptTx sends an AcceptTx response to the client
type ConversationEntryAcceptTx struct{}

// ToEntry converts this LocalTxSubmission entry to a generic ConversationEntryOutput
func (e ConversationEntryAcceptTx) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: localtxsubmission.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			localtxsubmission.NewMsgAcceptTx(),
		},
	}
}

// ConversationEntryRejectTx sends a RejectTx response with a rejection reason to the client
type ConversationEntryRejectTx struct {
	ReasonCbor []byte // CBOR-encoded rejection reason
}

// ToEntry converts this LocalTxSubmission entry to a generic ConversationEntryOutput
func (e ConversationEntryRejectTx) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: localtxsubmission.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			localtxsubmission.NewMsgRejectTx(e.ReasonCbor),
		},
	}
}

// ConversationEntryDone expects a Done message from the client to terminate the protocol
type ConversationEntryDone struct{}

// ToEntry converts this LocalTxSubmission entry to a generic ConversationEntryInput
func (e ConversationEntryDone) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localtxsubmission.ProtocolId,
		IsResponse:      false,
		MessageType:     localtxsubmission.MessageTypeDone,
		MsgFromCborFunc: localtxsubmission.NewMsgFromCbor,
	}
}

// Helper functions for creating common conversation patterns

// NewSubmitTxEntry creates a ConversationEntryInput for expecting a SubmitTx message
// with specific era ID and transaction CBOR validation
func NewSubmitTxEntry(
	eraId uint16,
	txCbor []byte,
) ouroboros_mock.ConversationEntryInput {
	return ConversationEntrySubmitTx{EraId: eraId, TxCbor: txCbor}.ToEntry()
}

// NewSubmitTxEntryAny creates a ConversationEntryInput for expecting any SubmitTx message
func NewSubmitTxEntryAny() ouroboros_mock.ConversationEntryInput {
	return ConversationEntrySubmitTx{}.ToEntry()
}

// NewAcceptTxEntry creates a ConversationEntryOutput for sending an AcceptTx response
func NewAcceptTxEntry() ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryAcceptTx{}.ToEntry()
}

// NewRejectTxEntry creates a ConversationEntryOutput for sending a RejectTx response
func NewRejectTxEntry(
	reasonCbor []byte,
) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryRejectTx{ReasonCbor: reasonCbor}.ToEntry()
}

// NewDoneEntry creates a ConversationEntryInput for expecting a Done message
func NewDoneEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryDone{}.ToEntry()
}
