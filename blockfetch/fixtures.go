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

package blockfetch

import (
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// Mock constants for BlockFetch testing

// MockBlockTypeConway is the block type for Conway era blocks
const MockBlockTypeConway uint = 6

// MockBlockHash1 is a sample block hash for testing
var MockBlockHash1 = []byte{
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
}

// MockBlockHash2 is a second sample block hash for testing
var MockBlockHash2 = []byte{
	0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
	0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30,
	0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38,
	0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40,
}

// MockBlockHash3 is a third sample block hash for testing
var MockBlockHash3 = []byte{
	0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48,
	0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f, 0x50,
	0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58,
	0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f, 0x60,
}

// MockBlockHash4 is a fourth sample block hash for testing
var MockBlockHash4 = []byte{
	0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68,
	0x69, 0x6a, 0x6b, 0x6c, 0x6d, 0x6e, 0x6f, 0x70,
	0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78,
	0x79, 0x7a, 0x7b, 0x7c, 0x7d, 0x7e, 0x7f, 0x80,
}

// MockPoint1 represents a point on the chain for testing (slot 100)
var MockPoint1 = pcommon.NewPoint(100, MockBlockHash1)

// MockPoint2 represents a second point on the chain for testing (slot 200)
var MockPoint2 = pcommon.NewPoint(200, MockBlockHash2)

// MockPoint3 represents a third point on the chain for testing (slot 300)
var MockPoint3 = pcommon.NewPoint(300, MockBlockHash3)

// MockPoint4 represents a fourth point on the chain for testing (slot 400)
var MockPoint4 = pcommon.NewPoint(400, MockBlockHash4)

// MockBlockCbor1 is sample block CBOR data for testing
var MockBlockCbor1 = []byte{
	0x82, 0x06, 0x82, // Conway era block wrapper
	0xa0,       // Empty header map
	0x84,       // Array of 4
	0x80, 0x80, // Empty arrays for body parts
	0x80, 0xa0, // Empty arrays/maps
}

// MockBlockCbor2 is a second sample block CBOR data for testing
var MockBlockCbor2 = []byte{
	0x82, 0x06, 0x82, // Conway era block wrapper
	0xa1, 0x00, 0x01, // Header with slot
	0x84,       // Array of 4
	0x80, 0x80, // Empty arrays for body parts
	0x80, 0xa0, // Empty arrays/maps
}

// MockBlockCbor3 is a third sample block CBOR data for testing
var MockBlockCbor3 = []byte{
	0x82, 0x06, 0x82, // Conway era block wrapper
	0xa1, 0x00, 0x02, // Header with slot
	0x84,       // Array of 4
	0x80, 0x80, // Empty arrays for body parts
	0x80, 0xa0, // Empty arrays/maps
}

// Pre-defined conversations for common BlockFetch scenarios

// ConversationBlockFetchRange is a pre-defined conversation for fetching a range of blocks:
// - Handshake request (generic)
// - Handshake NtN response
// - RequestRange (from MockPoint1 to MockPoint2)
// - StartBatch
// - Block (MockBlockCbor1)
// - Block (MockBlockCbor2)
// - BatchDone
var ConversationBlockFetchRange = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtNResponse,
	// RequestRange
	NewRequestRangeEntryAny(),
	// StartBatch
	NewStartBatchEntry(),
	// Blocks
	NewBlockEntry(MockBlockTypeConway, MockBlockCbor1),
	NewBlockEntry(MockBlockTypeConway, MockBlockCbor2),
	// BatchDone
	NewBatchDoneEntry(),
}

// ConversationBlockFetchNoBlocks is a pre-defined conversation for when requested blocks
// are not available:
// - Handshake request (generic)
// - Handshake NtN response
// - RequestRange
// - NoBlocks
var ConversationBlockFetchNoBlocks = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtNResponse,
	// RequestRange
	NewRequestRangeEntryAny(),
	// NoBlocks response
	NewNoBlocksEntry(),
}

// ConversationBlockFetchMultipleBatches is a pre-defined conversation for multiple
// range requests:
// - Handshake request (generic)
// - Handshake NtN response
// - First batch: RequestRange -> StartBatch -> Block -> BatchDone
// - Second batch: RequestRange -> StartBatch -> Block -> Block -> BatchDone
// - ClientDone
var ConversationBlockFetchMultipleBatches = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtNResponse,
	// First batch
	NewRequestRangeEntryAny(),
	NewStartBatchEntry(),
	NewBlockEntry(MockBlockTypeConway, MockBlockCbor1),
	NewBatchDoneEntry(),
	// Second batch
	NewRequestRangeEntryAny(),
	NewStartBatchEntry(),
	NewBlockEntry(MockBlockTypeConway, MockBlockCbor2),
	NewBlockEntry(MockBlockTypeConway, MockBlockCbor3),
	NewBatchDoneEntry(),
	// Client done
	NewClientDoneEntry(),
}
