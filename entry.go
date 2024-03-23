// Copyright 2024 Blink Labs Software
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

package ouroboros_mock

import (
	"time"

	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/handshake"
	"github.com/blinklabs-io/gouroboros/protocol/keepalive"
)

const (
	MockNetworkMagic       uint32 = 999999
	MockProtocolVersionNtC uint16 = (14 + protocol.ProtocolVersionNtCOffset)
	MockProtocolVersionNtN uint16 = 13
	MockKeepAliveCookie    uint16 = 999
)

type ConversationEntry interface {
	isConversationEntry()
}

type conversationEntryBase struct{}

func (c conversationEntryBase) isConversationEntry() {}

type ConversationEntryInput struct {
	conversationEntryBase
	ProtocolId      uint16
	IsResponse      bool
	Message         protocol.Message
	MessageType     uint
	MsgFromCborFunc protocol.MessageFromCborFunc
}

type ConversationEntryOutput struct {
	conversationEntryBase
	ProtocolId uint16
	IsResponse bool
	Messages   []protocol.Message
}

type ConversationEntryClose struct {
	conversationEntryBase
}

type ConversationEntrySleep struct {
	conversationEntryBase
	Duration time.Duration
}

// ConversationEntryHandshakeRequestGeneric is a pre-defined conversation event that matches a generic
// handshake request from a client
var ConversationEntryHandshakeRequestGeneric = ConversationEntryInput{
	ProtocolId:  handshake.ProtocolId,
	MessageType: handshake.MessageTypeProposeVersions,
}

// ConversationEntryHandshakeNtCResponse is a pre-defined conversation entry for a server NtC handshake response
var ConversationEntryHandshakeNtCResponse = ConversationEntryOutput{
	ProtocolId: handshake.ProtocolId,
	IsResponse: true,
	Messages: []protocol.Message{
		handshake.NewMsgAcceptVersion(
			MockProtocolVersionNtC,
			protocol.VersionDataNtC9to14(MockNetworkMagic),
		),
	},
}

// ConversationEntryHandshakeNtNResponse is a pre-defined conversation entry for a server NtN handshake response
var ConversationEntryHandshakeNtNResponse = ConversationEntryOutput{
	ProtocolId: handshake.ProtocolId,
	IsResponse: true,
	Messages: []protocol.Message{
		handshake.NewMsgAcceptVersion(
			MockProtocolVersionNtN,
			protocol.VersionDataNtN13andUp{
				VersionDataNtN11to12: protocol.VersionDataNtN11to12{
					CborNetworkMagic:                       MockNetworkMagic,
					CborInitiatorAndResponderDiffusionMode: protocol.DiffusionModeInitiatorOnly,
					CborPeerSharing:                        protocol.PeerSharingModeNoPeerSharing,
					CborQuery:                              protocol.QueryModeDisabled,
				},
			},
		),
	},
}

// ConversationEntryKeepAliveRequest is a pre-defined conversation entry for a keep-alive request
var ConversationEntryKeepAliveRequest = ConversationEntryInput{
	ProtocolId:      keepalive.ProtocolId,
	Message:         keepalive.NewMsgKeepAlive(MockKeepAliveCookie),
	MsgFromCborFunc: keepalive.NewMsgFromCbor,
}

// ConversationEntryKeepAliveResponse is a pre-defined conversation entry for a keep-alive response
var ConversationEntryKeepAliveResponse = ConversationEntryOutput{
	ProtocolId: keepalive.ProtocolId,
	IsResponse: true,
	Messages: []protocol.Message{
		keepalive.NewMsgKeepAliveResponse(MockKeepAliveCookie),
	},
}

// ConversationKeepAlive is a pre-defined conversation with a NtN handshake and repeated keep-alive requests
// and responses
var ConversationKeepAlive = []ConversationEntry{
	ConversationEntryHandshakeRequestGeneric,
	ConversationEntryHandshakeNtNResponse,
	ConversationEntryKeepAliveRequest,
	ConversationEntryKeepAliveResponse,
	ConversationEntryKeepAliveRequest,
	ConversationEntryKeepAliveResponse,
	ConversationEntryKeepAliveRequest,
	ConversationEntryKeepAliveResponse,
	ConversationEntryKeepAliveRequest,
	ConversationEntryKeepAliveResponse,
}

// ConversationKeepAliveClose is a pre-defined conversation with a NtN handshake that will close the connection
// after receiving a keep-alive request
var ConversationKeepAliveClose = []ConversationEntry{
	ConversationEntryHandshakeRequestGeneric,
	ConversationEntryHandshakeNtNResponse,
	ConversationEntryKeepAliveRequest,
	ConversationEntryClose{},
}
