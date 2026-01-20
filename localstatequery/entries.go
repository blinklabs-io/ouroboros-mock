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

package localstatequery

import (
	"github.com/blinklabs-io/gouroboros/protocol"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// ConversationEntryAcquire expects an Acquire message from the client
type ConversationEntryAcquire struct {
	Point *pcommon.Point // Optional: expected point for validation (nil means accept any)
}

// ToEntry converts this LocalStateQuery entry to a generic ConversationEntryInput
func (e ConversationEntryAcquire) ToEntry() ouroboros_mock.ConversationEntryInput {
	entry := ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localstatequery.ProtocolId,
		IsResponse:      false,
		MessageType:     localstatequery.MessageTypeAcquire,
		MsgFromCborFunc: localstatequery.NewMsgFromCbor,
	}
	if e.Point != nil {
		entry.Message = localstatequery.NewMsgAcquire(*e.Point)
	}
	return entry
}

// ConversationEntryAcquired sends an Acquired response to the client
type ConversationEntryAcquired struct{}

// ToEntry converts this LocalStateQuery entry to a generic ConversationEntryOutput
func (e ConversationEntryAcquired) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: localstatequery.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			localstatequery.NewMsgAcquired(),
		},
	}
}

// ConversationEntryAcquireFailure sends an AcquireFailure response to the client
type ConversationEntryAcquireFailure struct {
	Failure uint8 // Failure reason: AcquireFailurePointTooOld or AcquireFailurePointNotOnChain
}

// ToEntry converts this LocalStateQuery entry to a generic ConversationEntryOutput
func (e ConversationEntryAcquireFailure) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: localstatequery.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			localstatequery.NewMsgFailure(e.Failure),
		},
	}
}

// ConversationEntryReAcquire expects a ReAcquire message from the client
type ConversationEntryReAcquire struct {
	Point *pcommon.Point // Optional: expected point for validation (nil means accept any)
}

// ToEntry converts this LocalStateQuery entry to a generic ConversationEntryInput
func (e ConversationEntryReAcquire) ToEntry() ouroboros_mock.ConversationEntryInput {
	entry := ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localstatequery.ProtocolId,
		IsResponse:      false,
		MessageType:     localstatequery.MessageTypeReacquire,
		MsgFromCborFunc: localstatequery.NewMsgFromCbor,
	}
	if e.Point != nil {
		entry.Message = localstatequery.NewMsgReAcquire(*e.Point)
	}
	return entry
}

// ConversationEntryQuery expects a Query message from the client
type ConversationEntryQuery struct {
	Query []byte // Optional: expected query CBOR for validation (nil means accept any)
}

// ToEntry converts this LocalStateQuery entry to a generic ConversationEntryInput
func (e ConversationEntryQuery) ToEntry() ouroboros_mock.ConversationEntryInput {
	entry := ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localstatequery.ProtocolId,
		IsResponse:      false,
		MessageType:     localstatequery.MessageTypeQuery,
		MsgFromCborFunc: localstatequery.NewMsgFromCbor,
	}
	if e.Query != nil {
		entry.Message = localstatequery.NewMsgQuery(e.Query)
	}
	return entry
}

// ConversationEntryResult sends a query Result to the client
type ConversationEntryResult struct {
	Result []byte // Result CBOR data
}

// ToEntry converts this LocalStateQuery entry to a generic ConversationEntryOutput
func (e ConversationEntryResult) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: localstatequery.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			localstatequery.NewMsgResult(e.Result),
		},
	}
}

// ConversationEntryRelease expects a Release message from the client
type ConversationEntryRelease struct{}

// ToEntry converts this LocalStateQuery entry to a generic ConversationEntryInput
func (e ConversationEntryRelease) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localstatequery.ProtocolId,
		IsResponse:      false,
		MessageType:     localstatequery.MessageTypeRelease,
		MsgFromCborFunc: localstatequery.NewMsgFromCbor,
	}
}

// ConversationEntryDone expects a Done message from the client to terminate the protocol
type ConversationEntryDone struct{}

// ToEntry converts this LocalStateQuery entry to a generic ConversationEntryInput
func (e ConversationEntryDone) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      localstatequery.ProtocolId,
		IsResponse:      false,
		MessageType:     localstatequery.MessageTypeDone,
		MsgFromCborFunc: localstatequery.NewMsgFromCbor,
	}
}

// Helper functions for creating common conversation patterns

// NewAcquireEntry creates a ConversationEntryInput for expecting an Acquire message
func NewAcquireEntry(
	point *pcommon.Point,
) ouroboros_mock.ConversationEntryInput {
	return ConversationEntryAcquire{Point: point}.ToEntry()
}

// NewAcquireEntryAny creates a ConversationEntryInput for expecting any Acquire message
func NewAcquireEntryAny() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryAcquire{}.ToEntry()
}

// NewAcquiredEntry creates a ConversationEntryOutput for sending an Acquired response
func NewAcquiredEntry() ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryAcquired{}.ToEntry()
}

// NewAcquireFailureEntry creates a ConversationEntryOutput for sending an AcquireFailure response
func NewAcquireFailureEntry(
	failure uint8,
) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryAcquireFailure{Failure: failure}.ToEntry()
}

// NewAcquireFailurePointTooOldEntry creates a ConversationEntryOutput for PointTooOld failure
func NewAcquireFailurePointTooOldEntry() ouroboros_mock.ConversationEntryOutput {
	return NewAcquireFailureEntry(localstatequery.AcquireFailurePointTooOld)
}

// NewAcquireFailurePointNotOnChainEntry creates a ConversationEntryOutput for PointNotOnChain failure
func NewAcquireFailurePointNotOnChainEntry() ouroboros_mock.ConversationEntryOutput {
	return NewAcquireFailureEntry(localstatequery.AcquireFailurePointNotOnChain)
}

// NewReAcquireEntry creates a ConversationEntryInput for expecting a ReAcquire message
func NewReAcquireEntry(
	point *pcommon.Point,
) ouroboros_mock.ConversationEntryInput {
	return ConversationEntryReAcquire{Point: point}.ToEntry()
}

// NewReAcquireEntryAny creates a ConversationEntryInput for expecting any ReAcquire message
func NewReAcquireEntryAny() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryReAcquire{}.ToEntry()
}

// NewQueryEntry creates a ConversationEntryInput for expecting a Query message
func NewQueryEntry(query []byte) ouroboros_mock.ConversationEntryInput {
	return ConversationEntryQuery{Query: query}.ToEntry()
}

// NewQueryEntryAny creates a ConversationEntryInput for expecting any Query message
func NewQueryEntryAny() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryQuery{}.ToEntry()
}

// NewResultEntry creates a ConversationEntryOutput for sending a Result response
func NewResultEntry(result []byte) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryResult{Result: result}.ToEntry()
}

// NewReleaseEntry creates a ConversationEntryInput for expecting a Release message
func NewReleaseEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryRelease{}.ToEntry()
}

// NewDoneEntry creates a ConversationEntryInput for expecting a Done message
func NewDoneEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryDone{}.ToEntry()
}
