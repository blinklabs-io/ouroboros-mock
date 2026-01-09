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

package chainsync

import (
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// Mock constants for ChainSync testing

// MockOriginPoint represents the origin point (slot 0, empty hash)
var MockOriginPoint = pcommon.NewPointOrigin()

// MockBlockHash is a sample block hash for testing
var MockBlockHash = []byte{
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
}

// MockBlockHash2 is a second sample block hash for testing rollback scenarios
var MockBlockHash2 = []byte{
	0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
	0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30,
	0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38,
	0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40,
}

// MockBlockHash3 is a third sample block hash for testing new chain after
// rollback
var MockBlockHash3 = []byte{
	0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48,
	0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f, 0x50,
	0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58,
	0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f, 0x60,
}

// MockTip is a sample chain tip for testing
var MockTip = pcommon.Tip{
	Point:       pcommon.NewPoint(1000, MockBlockHash),
	BlockNumber: 100,
}

// MockTip2 is a sample chain tip after rollback for testing
var MockTip2 = pcommon.Tip{
	Point:       pcommon.NewPoint(500, MockBlockHash2),
	BlockNumber: 50,
}

// MockPoint1 represents a point on the chain for testing
var MockPoint1 = pcommon.NewPoint(100, MockBlockHash)

// MockPoint2 represents a second point on the chain for testing
var MockPoint2 = pcommon.NewPoint(200, MockBlockHash2)

// MockPoint3 represents a third point on the chain for testing
var MockPoint3 = pcommon.NewPoint(300, MockBlockHash3)

// MockBlockCbor is sample block CBOR data for testing
var MockBlockCbor = []byte{
	0x82, 0x00, 0xa0, // Minimal CBOR array with header
}

// Pre-defined conversations for common ChainSync scenarios

// ConversationChainSyncFromOrigin is a pre-defined conversation for basic
// sync from origin:
// - Handshake request (generic)
// - Handshake NtC response
// - FindIntersect request
// - IntersectFound at origin
// - RequestNext
// - RollForward with a mock block
// - RequestNext
// - AwaitReply (at tip)
var ConversationChainSyncFromOrigin = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// FindIntersect at origin
	NewFindIntersectEntry(true, nil),
	NewIntersectFoundEntry(true, MockOriginPoint, MockTip),
	// First RequestNext - returns a block
	NewRequestNextEntry(true),
	MustRollForwardEntry(true, 0, MockBlockCbor, MockTip),
	// Second RequestNext - at tip, await reply
	NewRequestNextEntry(true),
	NewAwaitReplyEntry(true),
}

// ConversationChainSyncRollback is a pre-defined conversation for sync with
// rollback scenario:
// - Handshake
// - FindIntersect
// - IntersectFound
// - RollForward (a few blocks)
// - RollBackward
// - RollForward (new chain)
var ConversationChainSyncRollback = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// FindIntersect
	NewFindIntersectEntry(true, nil),
	NewIntersectFoundEntry(true, MockPoint1, MockTip),
	// First RequestNext - RollForward block 1
	NewRequestNextEntry(true),
	MustRollForwardEntry(true, 0, MockBlockCbor, MockTip),
	// Second RequestNext - RollForward block 2
	NewRequestNextEntry(true),
	MustRollForwardEntry(true, 0, MockBlockCbor, MockTip),
	// Third RequestNext - RollBackward (chain reorganization)
	NewRequestNextEntry(true),
	NewRollBackwardEntry(true, MockPoint1, MockTip2),
	// Fourth RequestNext - RollForward on new chain
	NewRequestNextEntry(true),
	MustRollForwardEntry(true, 0, MockBlockCbor, MockTip2),
}

// ConversationChainSyncIntersectNotFound is a pre-defined conversation for
// when intersection point is not found:
// - Handshake
// - FindIntersect
// - IntersectNotFound
var ConversationChainSyncIntersectNotFound = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// FindIntersect
	NewFindIntersectEntry(true, nil),
	// IntersectNotFound
	NewIntersectNotFoundEntry(true, MockTip),
}
