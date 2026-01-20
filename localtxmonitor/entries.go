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

package localtxmonitor

import (
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/localtxmonitor"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// ConversationEntryAcquire expects an Acquire message from the client
type ConversationEntryAcquire struct{}

// ToEntry converts this LocalTxMonitor entry to a generic ConversationEntryInput
func (e ConversationEntryAcquire) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		IsResponse:      false,
		MessageType:     localtxmonitor.MessageTypeAcquire,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
}

// ConversationEntryAcquired sends an Acquired response with slot to the client
type ConversationEntryAcquired struct {
	SlotNo uint64 // The slot number at which the mempool snapshot was taken
}

// ToEntry converts this LocalTxMonitor entry to a generic ConversationEntryOutput
func (e ConversationEntryAcquired) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: localtxmonitor.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			localtxmonitor.NewMsgAcquired(e.SlotNo),
		},
	}
}

// ConversationEntryAwaitAcquire expects an Acquire message from the client
// when re-acquiring after a Release. This is semantically "await acquire" -
// waiting for mempool changes before acquiring a new snapshot. Uses the same
// MessageTypeAcquire as the initial Acquire.
type ConversationEntryAwaitAcquire struct{}

// ToEntry converts this LocalTxMonitor entry to a generic ConversationEntryInput
func (e ConversationEntryAwaitAcquire) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		IsResponse:      false,
		MessageType:     localtxmonitor.MessageTypeAcquire,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
}

// ConversationEntryHasTx expects a HasTx query from the client
type ConversationEntryHasTx struct {
	TxId []byte // Optional: expected transaction ID for validation (nil means accept any)
}

// ToEntry converts this LocalTxMonitor entry to a generic ConversationEntryInput
func (e ConversationEntryHasTx) ToEntry() ouroboros_mock.ConversationEntryInput {
	entry := ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		IsResponse:      false,
		MessageType:     localtxmonitor.MessageTypeHasTx,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
	if e.TxId != nil {
		entry.Message = localtxmonitor.NewMsgHasTx(e.TxId)
	}
	return entry
}

// ConversationEntryHasTxResult sends a HasTx result to the client
type ConversationEntryHasTxResult struct {
	Result bool // Whether the transaction exists in the mempool
}

// ToEntry converts this LocalTxMonitor entry to a generic ConversationEntryOutput
func (e ConversationEntryHasTxResult) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: localtxmonitor.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			localtxmonitor.NewMsgReplyHasTx(e.Result),
		},
	}
}

// ConversationEntryNextTx expects a NextTx message from the client
type ConversationEntryNextTx struct{}

// ToEntry converts this LocalTxMonitor entry to a generic ConversationEntryInput
func (e ConversationEntryNextTx) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		IsResponse:      false,
		MessageType:     localtxmonitor.MessageTypeNextTx,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
}

// ConversationEntryNextTxResult sends a NextTx result with optional transaction to the client
type ConversationEntryNextTxResult struct {
	EraId uint8  // Era ID of the transaction (only used if Tx is not nil)
	Tx    []byte // Transaction CBOR data (nil means no more transactions)
}

// ToEntry converts this LocalTxMonitor entry to a generic ConversationEntryOutput
func (e ConversationEntryNextTxResult) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: localtxmonitor.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			localtxmonitor.NewMsgReplyNextTx(e.EraId, e.Tx),
		},
	}
}

// ConversationEntryGetSizes expects a GetSizes message from the client
type ConversationEntryGetSizes struct{}

// ToEntry converts this LocalTxMonitor entry to a generic ConversationEntryInput
func (e ConversationEntryGetSizes) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		IsResponse:      false,
		MessageType:     localtxmonitor.MessageTypeGetSizes,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
}

// ConversationEntrySizes sends a Sizes response to the client
type ConversationEntrySizes struct {
	Capacity    uint32 // Mempool capacity in bytes
	Size        uint32 // Current mempool size in bytes
	NumberOfTxs uint32 // Number of transactions in the mempool
}

// ToEntry converts this LocalTxMonitor entry to a generic ConversationEntryOutput
func (e ConversationEntrySizes) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: localtxmonitor.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			localtxmonitor.NewMsgReplyGetSizes(
				e.Capacity,
				e.Size,
				e.NumberOfTxs,
			),
		},
	}
}

// ConversationEntryRelease expects a Release message from the client
type ConversationEntryRelease struct{}

// ToEntry converts this LocalTxMonitor entry to a generic ConversationEntryInput
func (e ConversationEntryRelease) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		IsResponse:      false,
		MessageType:     localtxmonitor.MessageTypeRelease,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
}

// ConversationEntryDone expects a Done message from the client to terminate
// the protocol.
type ConversationEntryDone struct{}

// ToEntry converts this LocalTxMonitor entry to a generic ConversationEntryInput
func (e ConversationEntryDone) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		IsResponse:      false,
		MessageType:     localtxmonitor.MessageTypeDone,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
}

// Helper functions for creating common conversation patterns

// NewAcquireEntry creates a ConversationEntryInput for expecting an Acquire message
func NewAcquireEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryAcquire{}.ToEntry()
}

// NewAcquiredEntry creates a ConversationEntryOutput for sending an Acquired response
func NewAcquiredEntry(slotNo uint64) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryAcquired{SlotNo: slotNo}.ToEntry()
}

// NewAwaitAcquireEntry creates a ConversationEntryInput for expecting an
// Acquire message after a Release (re-acquire).
func NewAwaitAcquireEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryAwaitAcquire{}.ToEntry()
}

// NewHasTxEntry creates a ConversationEntryInput for expecting a HasTx query
func NewHasTxEntry(txId []byte) ouroboros_mock.ConversationEntryInput {
	return ConversationEntryHasTx{TxId: txId}.ToEntry()
}

// NewHasTxEntryAny creates a ConversationEntryInput for expecting any HasTx query
func NewHasTxEntryAny() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryHasTx{}.ToEntry()
}

// NewHasTxResultEntry creates a ConversationEntryOutput for sending a HasTx result
func NewHasTxResultEntry(result bool) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryHasTxResult{Result: result}.ToEntry()
}

// NewNextTxEntry creates a ConversationEntryInput for expecting a NextTx message
func NewNextTxEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryNextTx{}.ToEntry()
}

// NewNextTxResultEntry creates a ConversationEntryOutput for sending a NextTx result with transaction
func NewNextTxResultEntry(
	eraId uint8,
	tx []byte,
) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryNextTxResult{EraId: eraId, Tx: tx}.ToEntry()
}

// NewNextTxResultEmptyEntry creates a ConversationEntryOutput for sending an empty NextTx result
func NewNextTxResultEmptyEntry() ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryNextTxResult{}.ToEntry()
}

// NewGetSizesEntry creates a ConversationEntryInput for expecting a GetSizes message
func NewGetSizesEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryGetSizes{}.ToEntry()
}

// NewSizesEntry creates a ConversationEntryOutput for sending a Sizes response
func NewSizesEntry(
	capacity, size, numberOfTxs uint32,
) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntrySizes{
		Capacity:    capacity,
		Size:        size,
		NumberOfTxs: numberOfTxs,
	}.ToEntry()
}

// NewReleaseEntry creates a ConversationEntryInput for expecting a Release message
func NewReleaseEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryRelease{}.ToEntry()
}

// NewDoneEntry creates a ConversationEntryInput for expecting a Done message
func NewDoneEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryDone{}.ToEntry()
}
