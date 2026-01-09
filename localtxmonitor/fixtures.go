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

package localtxmonitor

import (
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// Mock constants for LocalTxMonitor testing

// MockSlotNo is a sample slot number for testing
const MockSlotNo uint64 = 100000

// MockEraIdConway is the era ID for Conway era transactions
const MockEraIdConway uint8 = 6

// MockMempoolCapacity is a sample mempool capacity in bytes
const MockMempoolCapacity uint32 = 178176

// MockMempoolSize is a sample mempool size in bytes
const MockMempoolSize uint32 = 4096

// MockMempoolTxCount is a sample number of transactions in mempool
// Note: ConversationLocalTxMonitorBasic returns 1 transaction before returning empty
const MockMempoolTxCount uint32 = 1

// MockTxId is a sample transaction ID for testing
var MockTxId = []byte{
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
}

// MockTxId2 is a second sample transaction ID for testing
var MockTxId2 = []byte{
	0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
	0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30,
	0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38,
	0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40,
}

// MockTxCbor is sample transaction CBOR data for testing
var MockTxCbor = []byte{
	0x84, // Array of 4 (transaction structure)
	0xa0, // Empty map (transaction body)
	0xa0, // Empty map (transaction witness set)
	0xf5, // True (is valid)
	0xf6, // Null (auxiliary data)
}

// MockTxCbor2 is a second sample transaction CBOR data for testing
var MockTxCbor2 = []byte{
	0x84,             // Array of 4 (transaction structure)
	0xa1, 0x00, 0x80, // Map with inputs (empty array)
	0xa0, // Empty map (transaction witness set)
	0xf5, // True (is valid)
	0xf6, // Null (auxiliary data)
}

// Pre-defined conversations for common LocalTxMonitor scenarios

// ConversationLocalTxMonitorBasic is a pre-defined conversation for basic mempool query:
// - Handshake request (generic)
// - Handshake NtC response
// - Acquire
// - Acquired (with slot)
// - GetSizes
// - Sizes (mempool stats)
// - HasTx (check for transaction)
// - HasTx result (true)
// - NextTx
// - NextTx result (transaction)
// - NextTx
// - NextTx result (empty - no more transactions)
// - Release
var ConversationLocalTxMonitorBasic = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// Acquire mempool snapshot
	NewAcquireEntry(),
	NewAcquiredEntry(MockSlotNo),
	// Query mempool sizes
	NewGetSizesEntry(),
	NewSizesEntry(MockMempoolCapacity, MockMempoolSize, MockMempoolTxCount),
	// Check if specific transaction exists
	NewHasTxEntryAny(),
	NewHasTxResultEntry(true),
	// Get first transaction
	NewNextTxEntry(),
	NewNextTxResultEntry(MockEraIdConway, MockTxCbor),
	// Get second transaction (none remaining)
	NewNextTxEntry(),
	NewNextTxResultEmptyEntry(),
	// Release snapshot
	NewReleaseEntry(),
}

// ConversationLocalTxMonitorEmpty is a pre-defined conversation for empty mempool:
// - Handshake request (generic)
// - Handshake NtC response
// - Acquire
// - Acquired (with slot)
// - GetSizes
// - Sizes (empty mempool)
// - NextTx
// - NextTx result (empty)
// - Release
var ConversationLocalTxMonitorEmpty = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// Acquire mempool snapshot
	NewAcquireEntry(),
	NewAcquiredEntry(MockSlotNo),
	// Query mempool sizes (empty)
	NewGetSizesEntry(),
	NewSizesEntry(MockMempoolCapacity, 0, 0),
	// Try to get next transaction (none available)
	NewNextTxEntry(),
	NewNextTxResultEmptyEntry(),
	// Release snapshot
	NewReleaseEntry(),
}
