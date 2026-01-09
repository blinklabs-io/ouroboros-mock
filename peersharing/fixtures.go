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
	"net"

	"github.com/blinklabs-io/gouroboros/protocol/peersharing"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// Mock constants for PeerSharing testing

// MockPeerAddress1 is a sample IPv4 peer address for testing
var MockPeerAddress1 = peersharing.PeerAddress{
	IP:   net.ParseIP("192.168.1.100"),
	Port: 3001,
}

// MockPeerAddress2 is a second sample IPv4 peer address for testing
var MockPeerAddress2 = peersharing.PeerAddress{
	IP:   net.ParseIP("10.0.0.50"),
	Port: 3001,
}

// MockPeerAddress3 is a sample IPv6 peer address for testing
var MockPeerAddress3 = peersharing.PeerAddress{
	IP:   net.ParseIP("2001:db8::1"),
	Port: 3001,
}

// MockPeerAddresses is a sample list of peer addresses for testing
var MockPeerAddresses = []peersharing.PeerAddress{
	MockPeerAddress1,
	MockPeerAddress2,
}

// MockPeerAddressesWithIPv6 is a sample list of peer addresses including IPv6 for testing
var MockPeerAddressesWithIPv6 = []peersharing.PeerAddress{
	MockPeerAddress1,
	MockPeerAddress2,
	MockPeerAddress3,
}

// Pre-defined conversations for common PeerSharing scenarios

// ConversationPeerSharingBasic is a pre-defined conversation for basic peer sharing request/response:
// - Handshake request (generic)
// - Handshake NtN response
// - ShareRequest (request peers)
// - SharePeers (return peer addresses)
var ConversationPeerSharingBasic = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtNResponse,
	// Request peers
	NewShareRequestEntryAny(),
	// Return peer addresses
	NewSharePeersEntry(MockPeerAddresses),
}

// ConversationPeerSharingEmpty is a pre-defined conversation for peer sharing with no peers available:
// - Handshake request (generic)
// - Handshake NtN response
// - ShareRequest (request peers)
// - SharePeers (empty response)
var ConversationPeerSharingEmpty = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtNResponse,
	// Request peers
	NewShareRequestEntryAny(),
	// Return empty peer list
	NewSharePeersEmptyEntry(),
}

// ConversationPeerSharingMultiple is a pre-defined conversation for multiple peer sharing requests:
// - Handshake request (generic)
// - Handshake NtN response
// - ShareRequest (first request)
// - SharePeers (return peers)
// - ShareRequest (second request)
// - SharePeers (return peers with IPv6)
// - Done
var ConversationPeerSharingMultiple = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtNResponse,
	// First peer request
	NewShareRequestEntryAny(),
	NewSharePeersEntry(MockPeerAddresses),
	// Second peer request
	NewShareRequestEntryAny(),
	NewSharePeersEntry(MockPeerAddressesWithIPv6),
	// Done
	NewDoneEntry(),
}
