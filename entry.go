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
	"github.com/blinklabs-io/gouroboros/protocol/leiosfetch"
	"github.com/blinklabs-io/gouroboros/protocol/leiosnotify"
	"github.com/blinklabs-io/gouroboros/protocol/leiosvotes"
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
	ExpectedError   string
	IsRegex         bool
}

type ConversationEntryOutput struct {
	conversationEntryBase
	ProtocolId    uint16
	IsResponse    bool
	Messages      []protocol.Message
	ExpectedError string
	IsRegex       bool
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

// ConversationEntryHandshakeRequestOutput is a pre-defined conversation entry for a client handshake request
var ConversationEntryHandshakeRequestOutput = ConversationEntryOutput{
	ProtocolId: handshake.ProtocolId,
	IsResponse: false,
	Messages: []protocol.Message{
		handshake.NewMsgProposeVersions(protocol.ProtocolVersionMap{
			MockProtocolVersionNtC: protocol.VersionDataNtC9to14(
				MockNetworkMagic,
			),
			MockProtocolVersionNtN: protocol.VersionDataNtN13andUp{
				VersionDataNtN11to12: protocol.VersionDataNtN11to12{
					CborNetworkMagic:                       MockNetworkMagic,
					CborInitiatorAndResponderDiffusionMode: protocol.DiffusionModeInitiatorOnly,
					CborPeerSharing:                        protocol.PeerSharingModeNoPeerSharing,
					CborQuery:                              protocol.QueryModeDisabled,
				},
			},
		}),
	},
}

// ConversationEntryHandshakeNtCResponseInput is a pre-defined conversation entry for a client expecting a server NtC handshake response
var ConversationEntryHandshakeNtCResponseInput = ConversationEntryInput{
	ProtocolId: handshake.ProtocolId,
	IsResponse: true,
	Message: handshake.NewMsgAcceptVersion(
		MockProtocolVersionNtC,
		protocol.VersionDataNtC9to14(MockNetworkMagic),
	),
	MsgFromCborFunc: handshake.NewMsgFromCbor,
}

// ConversationEntryHandshakeNtNResponseInput is a pre-defined conversation entry for a client expecting a server NtN handshake response
var ConversationEntryHandshakeNtNResponseInput = ConversationEntryInput{
	ProtocolId: handshake.ProtocolId,
	IsResponse: true,
	Message: handshake.NewMsgAcceptVersion(
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
	MsgFromCborFunc: handshake.NewMsgFromCbor,
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

// NewConversationEntryLeiosFetchRequest builds a client request entry for the
// Leios fetch mini-protocol.
func NewConversationEntryLeiosFetchRequest(
	message protocol.Message,
) ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      leiosfetch.ProtocolId,
		Message:         message,
		MsgFromCborFunc: leiosfetch.NewMsgFromCbor,
	}
}

// NewConversationEntryLeiosFetchResponse builds a server response entry for
// the Leios fetch mini-protocol.
func NewConversationEntryLeiosFetchResponse(
	messages ...protocol.Message,
) ConversationEntryOutput {
	return ConversationEntryOutput{
		ProtocolId: leiosfetch.ProtocolId,
		IsResponse: true,
		Messages:   messages,
	}
}

// NewConversationEntryLeiosNotifyRequest builds a client request entry for the
// Leios notify mini-protocol.
func NewConversationEntryLeiosNotifyRequest(
	message protocol.Message,
) ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      leiosnotify.ProtocolId,
		Message:         message,
		MsgFromCborFunc: leiosnotify.NewMsgFromCbor,
	}
}

// NewConversationEntryLeiosNotifyResponse builds a server response entry for
// the Leios notify mini-protocol.
func NewConversationEntryLeiosNotifyResponse(
	messages ...protocol.Message,
) ConversationEntryOutput {
	return ConversationEntryOutput{
		ProtocolId: leiosnotify.ProtocolId,
		IsResponse: true,
		Messages:   messages,
	}
}

// NewConversationEntryLeiosVotesRequest builds a client request entry for the
// Leios votes mini-protocol.
func NewConversationEntryLeiosVotesRequest(
	message protocol.Message,
) ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      leiosvotes.ProtocolId,
		Message:         message,
		MsgFromCborFunc: leiosvotes.NewMsgFromCbor,
	}
}

// NewConversationEntryLeiosVotesResponse builds a server response entry for
// the Leios votes mini-protocol.
func NewConversationEntryLeiosVotesResponse(
	messages ...protocol.Message,
) ConversationEntryOutput {
	return ConversationEntryOutput{
		ProtocolId: leiosvotes.ProtocolId,
		IsResponse: true,
		Messages:   messages,
	}
}

// ConversationLeiosFetch is a minimal Leios fetch conversation that exercises
// a request, an empty response, and protocol completion.
var ConversationLeiosFetch = []ConversationEntry{
	ConversationEntryHandshakeRequestGeneric,
	ConversationEntryHandshakeNtNResponse,
	NewConversationEntryLeiosFetchRequest(leiosfetch.NewMsgDone()),
}

// ConversationLeiosNotify is a minimal Leios notify conversation that
// exercises a notification request and protocol completion.
var ConversationLeiosNotify = []ConversationEntry{
	ConversationEntryHandshakeRequestGeneric,
	ConversationEntryHandshakeNtNResponse,
	NewConversationEntryLeiosNotifyRequest(leiosnotify.NewMsgDone()),
}

// ConversationLeiosVotes is a minimal Leios votes conversation that exercises
// a request for one vote and protocol completion.
var ConversationLeiosVotes = []ConversationEntry{
	ConversationEntryHandshakeRequestGeneric,
	ConversationEntryHandshakeNtNResponse,
	NewConversationEntryLeiosVotesRequest(leiosvotes.NewMsgDone()),
}
