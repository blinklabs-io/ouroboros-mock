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
	for i, response := range responses {
		if response.ProtocolId == 0 || !response.IsResponse || len(response.Messages) != 1 {
			t.Errorf("response %d has unexpected shape: %#v", i, response)
		}
	}

	for name, conversation := range map[string][]mock.ConversationEntry{
		"fetch":  mock.ConversationLeiosFetch,
		"notify": mock.ConversationLeiosNotify,
		"votes":  mock.ConversationLeiosVotes,
	} {
		if len(conversation) != 3 {
			t.Errorf("%s conversation length = %d, want 3", name, len(conversation))
		}
	}
}
