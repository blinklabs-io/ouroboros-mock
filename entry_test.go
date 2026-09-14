// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package ouroboros_mock_test

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/protocol/leiosfetch"
	"github.com/blinklabs-io/gouroboros/protocol/leiosnotify"
	"github.com/blinklabs-io/gouroboros/protocol/leiosvotes"
	mock "github.com/blinklabs-io/ouroboros-mock"
)

func TestLeiosConversationBuilders(t *testing.T) {
	fetch := mock.NewConversationEntryLeiosFetchRequest(leiosfetch.NewMsgDone())
	if fetch.ProtocolId != leiosfetch.ProtocolId || fetch.MsgFromCborFunc == nil {
		t.Fatalf("unexpected Leios fetch entry: %#v", fetch)
	}

	notify := mock.NewConversationEntryLeiosNotifyRequest(leiosnotify.NewMsgDone())
	if notify.ProtocolId != leiosnotify.ProtocolId || notify.MsgFromCborFunc == nil {
		t.Fatalf("unexpected Leios notify entry: %#v", notify)
	}

	votes := mock.NewConversationEntryLeiosVotesRequest(leiosvotes.NewMsgDone())
	if votes.ProtocolId != leiosvotes.ProtocolId || votes.MsgFromCborFunc == nil {
		t.Fatalf("unexpected Leios votes entry: %#v", votes)
	}

	responses := []mock.ConversationEntryOutput{
		mock.NewConversationEntryLeiosFetchResponse(leiosfetch.NewMsgDone()),
		mock.NewConversationEntryLeiosNotifyResponse(leiosnotify.NewMsgDone()),
		mock.NewConversationEntryLeiosVotesResponse(leiosvotes.NewMsgDone()),
	}
	expectedResponses := []struct {
		protocolID uint16
		messageID  uint8
	}{
		{leiosfetch.ProtocolId, leiosfetch.MessageTypeDone},
		{leiosnotify.ProtocolId, leiosnotify.MessageTypeDone},
		{leiosvotes.ProtocolId, leiosvotes.MessageTypeDone},
	}
	for i, response := range responses {
		if response.ProtocolId != expectedResponses[i].protocolID || !response.IsResponse {
			t.Errorf("response %d has unexpected shape: %#v", i, response)
			continue
		}
		if len(response.Messages) != 1 {
			t.Fatalf("response %d contains %d messages, want 1", i, len(response.Messages))
		}
		if response.Messages[0].Type() != expectedResponses[i].messageID {
			t.Errorf(
				"response %d message type = %d, want %d",
				i,
				response.Messages[0].Type(),
				expectedResponses[i].messageID,
			)
		}
	}

	conversations := map[string]struct {
		entries      []mock.ConversationEntry
		protocolID   uint16
		completionID uint8
	}{
		"fetch": {
			entries:      mock.ConversationLeiosFetch,
			protocolID:   leiosfetch.ProtocolId,
			completionID: leiosfetch.MessageTypeDone,
		},
		"notify": {
			entries:      mock.ConversationLeiosNotify,
			protocolID:   leiosnotify.ProtocolId,
			completionID: leiosnotify.MessageTypeDone,
		},
		"votes": {
			entries:      mock.ConversationLeiosVotes,
			protocolID:   leiosvotes.ProtocolId,
			completionID: leiosvotes.MessageTypeDone,
		},
	}
	for name, conversation := range conversations {
		entries := conversation.entries
		if len(entries) != 3 {
			t.Fatalf("%s conversation length = %d, want 3", name, len(entries))
		}
		handshakeRequest, ok := entries[0].(mock.ConversationEntryInput)
		if !ok || handshakeRequest.ProtocolId != mock.ConversationEntryHandshakeRequestGeneric.ProtocolId || handshakeRequest.MessageType != mock.ConversationEntryHandshakeRequestGeneric.MessageType {
			t.Fatalf("%s handshake request has unexpected shape: %#v", name, entries[0])
		}
		handshakeResponse, ok := entries[1].(mock.ConversationEntryOutput)
		if !ok || handshakeResponse.ProtocolId != mock.ConversationEntryHandshakeNtNResponse.ProtocolId || !handshakeResponse.IsResponse {
			t.Fatalf("%s handshake response has unexpected shape: %#v", name, entries[1])
		}
		completion, ok := entries[2].(mock.ConversationEntryInput)
		if !ok || completion.ProtocolId != conversation.protocolID || completion.IsResponse || completion.Message == nil {
			t.Fatalf("%s completion entry has unexpected shape: %#v", name, entries[2])
		}
		if completion.Message.Type() != conversation.completionID {
			t.Errorf("%s completion message type = %d, want %d", name, completion.Message.Type(), conversation.completionID)
		}
	}
}
