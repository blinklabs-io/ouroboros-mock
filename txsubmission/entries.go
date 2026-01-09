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

package txsubmission

import (
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// ConversationEntryInit expects an Init message from the client (client starts the protocol)
type ConversationEntryInit struct{}

// ToEntry converts this TxSubmission entry to a generic ConversationEntryInput
func (e ConversationEntryInit) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      txsubmission.ProtocolId,
		IsResponse:      false,
		MessageType:     txsubmission.MessageTypeInit,
		MsgFromCborFunc: txsubmission.NewMsgFromCbor,
	}
}

// ConversationEntryRequestTxIds sends a RequestTxIds message to the client
type ConversationEntryRequestTxIds struct {
	Blocking bool   // Whether to block until txs are available
	Ack      uint16 // Number of txs to acknowledge
	Req      uint16 // Number of txs to request
}

// ToEntry converts this TxSubmission entry to a generic ConversationEntryOutput
func (e ConversationEntryRequestTxIds) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: txsubmission.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			txsubmission.NewMsgRequestTxIds(e.Blocking, e.Ack, e.Req),
		},
	}
}

// ConversationEntryReplyTxIds expects a ReplyTxIds message from the client
type ConversationEntryReplyTxIds struct {
	TxIds []txsubmission.TxIdAndSize // Optional: expected transaction IDs for validation (nil means accept any)
}

// ToEntry converts this TxSubmission entry to a generic ConversationEntryInput
func (e ConversationEntryReplyTxIds) ToEntry() ouroboros_mock.ConversationEntryInput {
	entry := ouroboros_mock.ConversationEntryInput{
		ProtocolId:      txsubmission.ProtocolId,
		IsResponse:      true,
		MessageType:     txsubmission.MessageTypeReplyTxIds,
		MsgFromCborFunc: txsubmission.NewMsgFromCbor,
	}
	if e.TxIds != nil {
		entry.Message = txsubmission.NewMsgReplyTxIds(e.TxIds)
	}
	return entry
}

// ConversationEntryRequestTxs sends a RequestTxs message to the client
type ConversationEntryRequestTxs struct {
	TxIds []txsubmission.TxId // Transaction IDs to request
}

// ToEntry converts this TxSubmission entry to a generic ConversationEntryOutput
func (e ConversationEntryRequestTxs) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: txsubmission.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			txsubmission.NewMsgRequestTxs(e.TxIds),
		},
	}
}

// ConversationEntryReplyTxs expects a ReplyTxs message from the client
type ConversationEntryReplyTxs struct {
	Txs []txsubmission.TxBody // Optional: expected transaction bodies for validation (nil means accept any)
}

// ToEntry converts this TxSubmission entry to a generic ConversationEntryInput
func (e ConversationEntryReplyTxs) ToEntry() ouroboros_mock.ConversationEntryInput {
	entry := ouroboros_mock.ConversationEntryInput{
		ProtocolId:      txsubmission.ProtocolId,
		IsResponse:      true,
		MessageType:     txsubmission.MessageTypeReplyTxs,
		MsgFromCborFunc: txsubmission.NewMsgFromCbor,
	}
	if e.Txs != nil {
		entry.Message = txsubmission.NewMsgReplyTxs(e.Txs)
	}
	return entry
}

// ConversationEntryDone expects a Done message from the client to terminate the protocol
type ConversationEntryDone struct{}

// ToEntry converts this TxSubmission entry to a generic ConversationEntryInput
func (e ConversationEntryDone) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      txsubmission.ProtocolId,
		IsResponse:      false,
		MessageType:     txsubmission.MessageTypeDone,
		MsgFromCborFunc: txsubmission.NewMsgFromCbor,
	}
}

// Helper functions for creating common conversation patterns

// NewInitEntry creates a ConversationEntryInput for expecting an Init message
func NewInitEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryInit{}.ToEntry()
}

// NewRequestTxIdsEntry creates a ConversationEntryOutput for sending a RequestTxIds message
func NewRequestTxIdsEntry(
	blocking bool,
	ack, req uint16,
) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryRequestTxIds{
		Blocking: blocking,
		Ack:      ack,
		Req:      req,
	}.ToEntry()
}

// NewReplyTxIdsEntry creates a ConversationEntryInput for expecting a ReplyTxIds message
func NewReplyTxIdsEntry(
	txIds []txsubmission.TxIdAndSize,
) ouroboros_mock.ConversationEntryInput {
	return ConversationEntryReplyTxIds{
		TxIds: txIds,
	}.ToEntry()
}

// NewReplyTxIdsEntryAny creates a ConversationEntryInput for expecting any ReplyTxIds message
func NewReplyTxIdsEntryAny() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryReplyTxIds{}.ToEntry()
}

// NewRequestTxsEntry creates a ConversationEntryOutput for sending a RequestTxs message
func NewRequestTxsEntry(
	txIds []txsubmission.TxId,
) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryRequestTxs{
		TxIds: txIds,
	}.ToEntry()
}

// NewReplyTxsEntry creates a ConversationEntryInput for expecting a ReplyTxs message
func NewReplyTxsEntry(
	txs []txsubmission.TxBody,
) ouroboros_mock.ConversationEntryInput {
	return ConversationEntryReplyTxs{
		Txs: txs,
	}.ToEntry()
}

// NewReplyTxsEntryAny creates a ConversationEntryInput for expecting any ReplyTxs message
func NewReplyTxsEntryAny() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryReplyTxs{}.ToEntry()
}

// NewDoneEntry creates a ConversationEntryInput for expecting a Done message
func NewDoneEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryDone{}.ToEntry()
}
