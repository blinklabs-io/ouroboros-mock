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

package localtxsubmission

import (
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// Mock constants for LocalTxSubmission testing

// MockEraIdConway is the Conway era ID
const MockEraIdConway uint16 = 6

// MockEraIdBabbage is the Babbage era ID
const MockEraIdBabbage uint16 = 5

// MockTxCbor is a minimal sample transaction CBOR for testing
var MockTxCbor = []byte{
	0x84, // array(4) - transaction structure
	0xa0, // map(0) - empty transaction body
	0xa0, // map(0) - empty witness set
	0xf6, // null - no auxiliary data
	0xf6, // null - no script validity
}

// MockRejectReasonCbor is a sample CBOR-encoded rejection reason for testing
// This represents a generic transaction validation error
var MockRejectReasonCbor = []byte{
	0x82,       // array(2)
	0x00,       // error code 0
	0x78, 0x1a, // text string (26 bytes)
	0x49, 0x6e, 0x73, 0x75, 0x66, 0x66, 0x69, 0x63, // "Insuffic"
	0x69, 0x65, 0x6e, 0x74, 0x20, 0x66, 0x75, 0x6e, // "ient fun"
	0x64, 0x73, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x66, // "ds for f"
	0x65, 0x65, // "ee"
}

// MockRejectReasonInvalidScript is a sample CBOR-encoded rejection reason for script validation failure
var MockRejectReasonInvalidScript = []byte{
	0x82,       // array(2)
	0x01,       // error code 1
	0x78, 0x18, // text string (24 bytes)
	0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x20, 0x76, // "Script v"
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x69, 0x6f, // "alidatio"
	0x6e, 0x20, 0x66, 0x61, 0x69, 0x6c, 0x65, 0x64, // "n failed"
}

// Pre-defined conversations for common LocalTxSubmission scenarios

// ConversationLocalTxSubmissionAccept is a pre-defined conversation for successful tx submission:
// - Handshake request (generic)
// - Handshake NtC response
// - SubmitTx (any transaction)
// - AcceptTx
var ConversationLocalTxSubmissionAccept = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// Submit transaction
	NewSubmitTxEntryAny(),
	// Accept transaction
	NewAcceptTxEntry(),
}

// ConversationLocalTxSubmissionReject is a pre-defined conversation for rejected tx submission:
// - Handshake request (generic)
// - Handshake NtC response
// - SubmitTx (any transaction)
// - RejectTx with rejection reason
var ConversationLocalTxSubmissionReject = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// Submit transaction
	NewSubmitTxEntryAny(),
	// Reject transaction with reason
	NewRejectTxEntry(MockRejectReasonCbor),
}

// ConversationLocalTxSubmissionMultiple is a pre-defined conversation for multiple tx submissions:
// - Handshake request (generic)
// - Handshake NtC response
// - SubmitTx -> AcceptTx (first tx accepted)
// - SubmitTx -> RejectTx (second tx rejected)
// - SubmitTx -> AcceptTx (third tx accepted)
var ConversationLocalTxSubmissionMultiple = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// First transaction - accepted
	NewSubmitTxEntryAny(),
	NewAcceptTxEntry(),
	// Second transaction - rejected
	NewSubmitTxEntryAny(),
	NewRejectTxEntry(MockRejectReasonCbor),
	// Third transaction - accepted
	NewSubmitTxEntryAny(),
	NewAcceptTxEntry(),
}
