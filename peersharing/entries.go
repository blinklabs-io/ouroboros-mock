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

package peersharing

import (
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/peersharing"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// ConversationEntryShareRequest expects a ShareRequest message from the client
type ConversationEntryShareRequest struct {
	Amount uint8 // Optional: expected number of peers requested (0 means accept any)
}

// ToEntry converts this PeerSharing entry to a generic ConversationEntryInput
func (e ConversationEntryShareRequest) ToEntry() ouroboros_mock.ConversationEntryInput {
	entry := ouroboros_mock.ConversationEntryInput{
		ProtocolId:      peersharing.ProtocolId,
		IsResponse:      false,
		MessageType:     peersharing.MessageTypeShareRequest,
		MsgFromCborFunc: peersharing.NewMsgFromCbor,
	}
	if e.Amount > 0 {
		entry.Message = peersharing.NewMsgShareRequest(e.Amount)
	}
	return entry
}

// ConversationEntrySharePeers sends a SharePeers response to the client
type ConversationEntrySharePeers struct {
	PeerAddresses []peersharing.PeerAddress // List of peer addresses to share
}

// ToEntry converts this PeerSharing entry to a generic ConversationEntryOutput
func (e ConversationEntrySharePeers) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: peersharing.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			peersharing.NewMsgSharePeers(e.PeerAddresses),
		},
	}
}

// ConversationEntryDone expects a Done message from the client to terminate the protocol
type ConversationEntryDone struct{}

// ToEntry converts this PeerSharing entry to a generic ConversationEntryInput
func (e ConversationEntryDone) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      peersharing.ProtocolId,
		IsResponse:      false,
		MessageType:     peersharing.MessageTypeDone,
		MsgFromCborFunc: peersharing.NewMsgFromCbor,
	}
}

// Helper functions for creating common conversation patterns

// NewShareRequestEntry creates a ConversationEntryInput for expecting a ShareRequest message
func NewShareRequestEntry(amount uint8) ouroboros_mock.ConversationEntryInput {
	return ConversationEntryShareRequest{Amount: amount}.ToEntry()
}

// NewShareRequestEntryAny creates a ConversationEntryInput for expecting any ShareRequest message
func NewShareRequestEntryAny() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryShareRequest{}.ToEntry()
}

// NewSharePeersEntry creates a ConversationEntryOutput for sending a SharePeers response
func NewSharePeersEntry(
	peerAddresses []peersharing.PeerAddress,
) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntrySharePeers{PeerAddresses: peerAddresses}.ToEntry()
}

// NewSharePeersEmptyEntry creates a ConversationEntryOutput for sending an empty SharePeers response
func NewSharePeersEmptyEntry() ouroboros_mock.ConversationEntryOutput {
	return ConversationEntrySharePeers{
		PeerAddresses: []peersharing.PeerAddress{},
	}.ToEntry()
}

// NewDoneEntry creates a ConversationEntryInput for expecting a Done message
func NewDoneEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryDone{}.ToEntry()
}
