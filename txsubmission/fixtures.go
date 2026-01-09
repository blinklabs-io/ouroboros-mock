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

package txsubmission

import (
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// Mock constants for TxSubmission testing

// MockEraConway is the era ID for Conway era transactions
const MockEraConway uint16 = 6

// MockTxHash1 is a sample transaction hash for testing
var MockTxHash1 = [32]byte{
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
}

// MockTxHash2 is a second sample transaction hash for testing
var MockTxHash2 = [32]byte{
	0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
	0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30,
	0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38,
	0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40,
}

// MockTxHash3 is a third sample transaction hash for testing
var MockTxHash3 = [32]byte{
	0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48,
	0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f, 0x50,
	0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58,
	0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f, 0x60,
}

// MockTxId1 is a sample transaction ID for testing
var MockTxId1 = txsubmission.TxId{
	EraId: MockEraConway,
	TxId:  MockTxHash1,
}

// MockTxId2 is a second sample transaction ID for testing
var MockTxId2 = txsubmission.TxId{
	EraId: MockEraConway,
	TxId:  MockTxHash2,
}

// MockTxId3 is a third sample transaction ID for testing
var MockTxId3 = txsubmission.TxId{
	EraId: MockEraConway,
	TxId:  MockTxHash3,
}

// MockTxSize1 is a sample transaction size for testing (256 bytes)
const MockTxSize1 uint32 = 256

// MockTxSize2 is a second sample transaction size for testing (512 bytes)
const MockTxSize2 uint32 = 512

// MockTxSize3 is a third sample transaction size for testing (1024 bytes)
const MockTxSize3 uint32 = 1024

// MockTxIdAndSize1 is a sample TxIdAndSize for testing
var MockTxIdAndSize1 = txsubmission.TxIdAndSize{
	TxId: MockTxId1,
	Size: MockTxSize1,
}

// MockTxIdAndSize2 is a second sample TxIdAndSize for testing
var MockTxIdAndSize2 = txsubmission.TxIdAndSize{
	TxId: MockTxId2,
	Size: MockTxSize2,
}

// MockTxIdAndSize3 is a third sample TxIdAndSize for testing
var MockTxIdAndSize3 = txsubmission.TxIdAndSize{
	TxId: MockTxId3,
	Size: MockTxSize3,
}

// MockTxBody1 is sample transaction CBOR body for testing
var MockTxBody1 = []byte{
	0x84, // Array of 4 (transaction structure)
	0xa0, // Empty map (inputs)
	0x80, // Empty array (outputs)
	0xa0, // Empty map (fee, etc)
	0xa0, // Empty map (witnesses)
}

// MockTxBody2 is a second sample transaction CBOR body for testing
var MockTxBody2 = []byte{
	0x84,             // Array of 4
	0xa1, 0x00, 0x80, // Map with inputs
	0x81, 0xa0, // Array with one output
	0xa1, 0x02, 0x00, // Map with fee
	0xa0, // Empty witnesses
}

// MockTxBody3 is a third sample transaction CBOR body for testing
var MockTxBody3 = []byte{
	0x84,                               // Array of 4
	0xa1, 0x00, 0x81, 0x82, 0x58, 0x20, // Map with inputs
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
	0x00,       // Index 0
	0x80,       // Empty outputs
	0xa1, 0x02, // Fee
	0x19, 0x01, 0x00, // 256
	0xa0, // Empty witnesses
}

// MockTx1 is a sample TxBody for testing
var MockTx1 = txsubmission.TxBody{
	EraId:  MockEraConway,
	TxBody: MockTxBody1,
}

// MockTx2 is a second sample TxBody for testing
var MockTx2 = txsubmission.TxBody{
	EraId:  MockEraConway,
	TxBody: MockTxBody2,
}

// MockTx3 is a third sample TxBody for testing
var MockTx3 = txsubmission.TxBody{
	EraId:  MockEraConway,
	TxBody: MockTxBody3,
}

// Pre-defined conversations for common TxSubmission scenarios

// ConversationTxSubmissionBasic is a pre-defined conversation for basic tx submission flow:
// - Handshake request (generic)
// - Handshake NtN response
// - Init (client starts protocol)
// - RequestTxIds (server requests tx IDs, non-blocking)
// - ReplyTxIds (client sends one tx ID)
// - RequestTxs (server requests the transaction)
// - ReplyTxs (client sends the transaction body)
// - RequestTxIds (server requests more, blocking)
// - Done (client terminates, no more transactions)
var ConversationTxSubmissionBasic = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtNResponse,
	// Init (expect from client)
	NewInitEntry(),
	// Server sends request for tx IDs (non-blocking)
	NewRequestTxIdsEntry(false, 0, 10),
	// Expect client to reply with tx IDs
	NewReplyTxIdsEntryAny(),
	// Server sends request for the transactions
	NewRequestTxsEntry([]txsubmission.TxId{MockTxId1}),
	// Expect client to send the transaction body
	NewReplyTxsEntryAny(),
	// Server sends request for more tx IDs (blocking)
	NewRequestTxIdsEntry(true, 1, 10),
	// Expect client to terminate (no more transactions)
	NewDoneEntry(),
}

// ConversationTxSubmissionMultipleTxs is a pre-defined conversation for multiple transactions:
// - Handshake request (generic)
// - Handshake NtN response
// - Init (client starts protocol)
// - RequestTxIds (server requests tx IDs)
// - ReplyTxIds (client sends multiple tx IDs)
// - RequestTxs (server requests transactions)
// - ReplyTxs (client sends transaction bodies)
// - RequestTxIds (server requests more)
// - ReplyTxIds (client sends one more)
// - RequestTxs (server requests that transaction)
// - ReplyTxs (client sends the transaction)
// - RequestTxIds (blocking)
// - Done (no more transactions)
var ConversationTxSubmissionMultipleTxs = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtNResponse,
	// Init (expect from client)
	NewInitEntry(),
	// First batch: server sends request for tx IDs
	NewRequestTxIdsEntry(false, 0, 10),
	// Expect client to reply with tx IDs
	NewReplyTxIdsEntryAny(),
	// Server sends request for both transactions
	NewRequestTxsEntry([]txsubmission.TxId{MockTxId1, MockTxId2}),
	// Expect client to send transaction bodies
	NewReplyTxsEntryAny(),
	// Second batch: server sends request for more tx IDs
	NewRequestTxIdsEntry(false, 2, 10),
	// Expect client to reply with more tx IDs
	NewReplyTxIdsEntryAny(),
	// Server sends request for that transaction
	NewRequestTxsEntry([]txsubmission.TxId{MockTxId3}),
	// Expect client to send the transaction body
	NewReplyTxsEntryAny(),
	// Server sends request for more tx IDs (blocking)
	NewRequestTxIdsEntry(true, 1, 10),
	// Expect client to terminate (no more transactions)
	NewDoneEntry(),
}

// ConversationTxSubmissionEmpty is a pre-defined conversation when no transactions are available:
// - Handshake request (generic)
// - Handshake NtN response
// - Init (client starts protocol)
// - RequestTxIds (server requests tx IDs, non-blocking)
// - ReplyTxIds (client sends empty list)
// - RequestTxIds (server requests again, blocking)
// - Done (client terminates, no transactions)
var ConversationTxSubmissionEmpty = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtNResponse,
	// Init (expect from client)
	NewInitEntry(),
	// Server sends request for tx IDs (non-blocking)
	NewRequestTxIdsEntry(false, 0, 10),
	// Expect client to reply with empty list
	NewReplyTxIdsEntryAny(),
	// Server sends request again (blocking, waiting for new transactions)
	NewRequestTxIdsEntry(true, 0, 10),
	// Expect client to terminate (no transactions available)
	NewDoneEntry(),
}
