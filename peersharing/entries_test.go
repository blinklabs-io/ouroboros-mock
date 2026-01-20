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

package peersharing_test

import (
	"net"
	"testing"

	"github.com/blinklabs-io/gouroboros/protocol/peersharing"
	ps "github.com/blinklabs-io/ouroboros-mock/peersharing"
)

// Test ConversationEntryShareRequest

func TestConversationEntryShareRequest_ToEntry_WithAmount(t *testing.T) {
	entry := ps.ConversationEntryShareRequest{Amount: 5}
	result := entry.ToEntry()

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != peersharing.MessageTypeShareRequest {
		t.Errorf(
			"expected MessageType %d, got %d",
			peersharing.MessageTypeShareRequest,
			result.MessageType,
		)
	}
	if result.Message == nil {
		t.Errorf("expected Message to be non-nil when Amount > 0")
	}
	if result.MsgFromCborFunc == nil {
		t.Errorf("expected MsgFromCborFunc to be non-nil")
	}
}

func TestConversationEntryShareRequest_ToEntry_ZeroAmount(t *testing.T) {
	entry := ps.ConversationEntryShareRequest{Amount: 0}
	result := entry.ToEntry()

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != peersharing.MessageTypeShareRequest {
		t.Errorf(
			"expected MessageType %d, got %d",
			peersharing.MessageTypeShareRequest,
			result.MessageType,
		)
	}
	if result.Message != nil {
		t.Errorf("expected Message to be nil when Amount is 0, got non-nil")
	}
}

// Test ConversationEntrySharePeers

func TestConversationEntrySharePeers_ToEntry_WithAddresses(t *testing.T) {
	peerAddresses := []peersharing.PeerAddress{
		{IP: net.ParseIP("192.168.1.1"), Port: 3001},
		{IP: net.ParseIP("10.0.0.1"), Port: 3002},
	}
	entry := ps.ConversationEntrySharePeers{PeerAddresses: peerAddresses}
	result := entry.ToEntry()

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
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

func TestConversationEntrySharePeers_ToEntry_EmptyAddresses(t *testing.T) {
	entry := ps.ConversationEntrySharePeers{
		PeerAddresses: []peersharing.PeerAddress{},
	}
	result := entry.ToEntry()

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
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

func TestConversationEntrySharePeers_ToEntry_NilAddresses(t *testing.T) {
	entry := ps.ConversationEntrySharePeers{PeerAddresses: nil}
	result := entry.ToEntry()

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
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

// Test ConversationEntryDone

func TestConversationEntryDone_ToEntry(t *testing.T) {
	entry := ps.ConversationEntryDone{}
	result := entry.ToEntry()

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf(
			"expected IsResponse to be false (Done is sent by client), got true",
		)
	}
	if result.MessageType != peersharing.MessageTypeDone {
		t.Errorf(
			"expected MessageType %d, got %d",
			peersharing.MessageTypeDone,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// Test helper functions

func TestNewShareRequestEntry(t *testing.T) {
	testCases := []struct {
		name   string
		amount uint8
	}{
		{"amount 1", 1},
		{"amount 5", 5},
		{"amount 10", 10},
		{"max amount", 255},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ps.NewShareRequestEntry(tc.amount)

			if result.ProtocolId != peersharing.ProtocolId {
				t.Errorf(
					"expected ProtocolId %d, got %d",
					peersharing.ProtocolId,
					result.ProtocolId,
				)
			}
			if result.IsResponse != false {
				t.Errorf("expected IsResponse to be false, got true")
			}
			if result.MessageType != peersharing.MessageTypeShareRequest {
				t.Errorf(
					"expected MessageType %d, got %d",
					peersharing.MessageTypeShareRequest,
					result.MessageType,
				)
			}
			if result.Message == nil {
				t.Errorf("expected Message to be non-nil")
			}
		})
	}
}

func TestNewShareRequestEntryAny(t *testing.T) {
	result := ps.NewShareRequestEntryAny()

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != peersharing.MessageTypeShareRequest {
		t.Errorf(
			"expected MessageType %d, got %d",
			peersharing.MessageTypeShareRequest,
			result.MessageType,
		)
	}
	if result.Message != nil {
		t.Errorf("expected Message to be nil for 'any' request, got non-nil")
	}
}

func TestNewSharePeersEntry(t *testing.T) {
	peerAddresses := []peersharing.PeerAddress{
		{IP: net.ParseIP("192.168.1.100"), Port: 3001},
		{IP: net.ParseIP("10.0.0.50"), Port: 3002},
	}

	result := ps.NewSharePeersEntry(peerAddresses)

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
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

func TestNewSharePeersEntry_SingleAddress(t *testing.T) {
	peerAddresses := []peersharing.PeerAddress{
		{IP: net.ParseIP("192.168.1.100"), Port: 3001},
	}

	result := ps.NewSharePeersEntry(peerAddresses)

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
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

func TestNewSharePeersEmptyEntry(t *testing.T) {
	result := ps.NewSharePeersEmptyEntry()

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
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

func TestNewDoneEntry(t *testing.T) {
	result := ps.NewDoneEntry()

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse != false {
		t.Errorf("expected IsResponse to be false, got true")
	}
	if result.MessageType != peersharing.MessageTypeDone {
		t.Errorf(
			"expected MessageType %d, got %d",
			peersharing.MessageTypeDone,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// Test mock fixtures

func TestMockPeerAddress1(t *testing.T) {
	addr := ps.MockPeerAddress1

	expectedIP := net.ParseIP("192.168.1.100")
	if !addr.IP.Equal(expectedIP) {
		t.Errorf("expected IP %v, got %v", expectedIP, addr.IP)
	}
	if addr.Port != 3001 {
		t.Errorf("expected Port 3001, got %d", addr.Port)
	}
}

func TestMockPeerAddress2(t *testing.T) {
	addr := ps.MockPeerAddress2

	expectedIP := net.ParseIP("10.0.0.50")
	if !addr.IP.Equal(expectedIP) {
		t.Errorf("expected IP %v, got %v", expectedIP, addr.IP)
	}
	if addr.Port != 3001 {
		t.Errorf("expected Port 3001, got %d", addr.Port)
	}
}

func TestMockPeerAddress3_IPv6(t *testing.T) {
	addr := ps.MockPeerAddress3

	expectedIP := net.ParseIP("2001:db8::1")
	if !addr.IP.Equal(expectedIP) {
		t.Errorf("expected IP %v, got %v", expectedIP, addr.IP)
	}
	if addr.Port != 3001 {
		t.Errorf("expected Port 3001, got %d", addr.Port)
	}
}

func TestMockPeerAddresses(t *testing.T) {
	addrs := ps.MockPeerAddresses

	if len(addrs) != 2 {
		t.Errorf("expected 2 addresses, got %d", len(addrs))
	}

	// Verify it contains MockPeerAddress1 and MockPeerAddress2
	if !addrs[0].IP.Equal(ps.MockPeerAddress1.IP) ||
		addrs[0].Port != ps.MockPeerAddress1.Port {
		t.Errorf("first address does not match MockPeerAddress1")
	}
	if !addrs[1].IP.Equal(ps.MockPeerAddress2.IP) ||
		addrs[1].Port != ps.MockPeerAddress2.Port {
		t.Errorf("second address does not match MockPeerAddress2")
	}
}

func TestMockPeerAddressesWithIPv6(t *testing.T) {
	addrs := ps.MockPeerAddressesWithIPv6

	if len(addrs) != 3 {
		t.Errorf("expected 3 addresses, got %d", len(addrs))
	}

	// Verify it contains all three mock addresses
	if !addrs[0].IP.Equal(ps.MockPeerAddress1.IP) ||
		addrs[0].Port != ps.MockPeerAddress1.Port {
		t.Errorf("first address does not match MockPeerAddress1")
	}
	if !addrs[1].IP.Equal(ps.MockPeerAddress2.IP) ||
		addrs[1].Port != ps.MockPeerAddress2.Port {
		t.Errorf("second address does not match MockPeerAddress2")
	}
	if !addrs[2].IP.Equal(ps.MockPeerAddress3.IP) ||
		addrs[2].Port != ps.MockPeerAddress3.Port {
		t.Errorf("third address does not match MockPeerAddress3")
	}
}

// Test conversation fixtures

func TestConversationPeerSharingBasic(t *testing.T) {
	conv := ps.ConversationPeerSharingBasic

	if conv == nil {
		t.Fatalf("ConversationPeerSharingBasic should not be nil")
	}
	if len(conv) != 4 {
		t.Errorf("expected 4 entries, got %d", len(conv))
	}
}

func TestConversationPeerSharingEmpty(t *testing.T) {
	conv := ps.ConversationPeerSharingEmpty

	if conv == nil {
		t.Fatalf("ConversationPeerSharingEmpty should not be nil")
	}
	if len(conv) != 4 {
		t.Errorf("expected 4 entries, got %d", len(conv))
	}
}

func TestConversationPeerSharingMultiple(t *testing.T) {
	conv := ps.ConversationPeerSharingMultiple

	if conv == nil {
		t.Fatalf("ConversationPeerSharingMultiple should not be nil")
	}
	if len(conv) != 7 {
		t.Errorf("expected 7 entries, got %d", len(conv))
	}
}

// Test that entries implement ConversationEntry interface (compile-time check)

func TestConversationEntryInterface(t *testing.T) {
	// This test verifies that the ToEntry() methods return types that can be
	// used in conversation slices. The actual interface compliance is checked
	// at compile time through the fixtures.

	// ShareRequest entry
	shareReqEntry := ps.NewShareRequestEntry(5)
	if shareReqEntry.ProtocolId == 0 {
		t.Error("ShareRequest entry should have non-zero ProtocolId")
	}

	// SharePeers entry
	sharePeersEntry := ps.NewSharePeersEntry(ps.MockPeerAddresses)
	if sharePeersEntry.ProtocolId == 0 {
		t.Error("SharePeers entry should have non-zero ProtocolId")
	}

	// Done entry
	doneEntry := ps.NewDoneEntry()
	if doneEntry.ProtocolId == 0 {
		t.Error("Done entry should have non-zero ProtocolId")
	}
}

// Test edge cases

func TestShareRequestEntry_ZeroAmount_ViaHelper(t *testing.T) {
	result := ps.NewShareRequestEntry(0)

	// According to the implementation, Amount > 0 sets Message, so 0 results in nil Message
	// This matches the behavior of NewShareRequestEntryAny
	if result.Message != nil {
		t.Errorf("expected Message to be nil for amount 0, got non-nil")
	}
}

func TestSharePeersEntry_ManyAddresses(t *testing.T) {
	// Test with many peer addresses
	var addresses []peersharing.PeerAddress
	for i := range 100 {
		addresses = append(addresses, peersharing.PeerAddress{
			IP:   net.ParseIP("192.168.1.1"),
			Port: uint16(3000 + i),
		})
	}

	result := ps.NewSharePeersEntry(addresses)

	if result.ProtocolId != peersharing.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			peersharing.ProtocolId,
			result.ProtocolId,
		)
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
}
