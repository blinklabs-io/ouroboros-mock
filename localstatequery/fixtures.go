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

package localstatequery

import (
	"time"

	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// Mock constants for LocalStateQuery testing

// MockCurrentEra is the default mock era (Conway = 6)
const MockCurrentEra uint = 6

// MockEpochNo is a sample epoch number for testing
const MockEpochNo uint64 = 500

// MockSystemStartTime is the blockchain origin time (Byron mainnet start: September 23, 2017)
// Note: This is the chain's SystemStart, not when any particular era started
var MockSystemStartTime = time.Date(2017, 9, 23, 21, 44, 51, 0, time.UTC)

// MockBlockHash is a sample block hash for testing
var MockBlockHash = []byte{
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
}

// MockPoint is a sample chain point for testing
var MockPoint = pcommon.NewPoint(100000, MockBlockHash)

// MockTxHash is a sample transaction hash for UTxO testing
var MockTxHash = []byte{
	0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11,
	0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99,
	0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11,
	0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99,
}

// MockAddress is a sample Shelley address bytes for testing
var MockAddress = []byte{
	0x01, // Shelley payment address type
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	0x19, 0x1a, 0x1b, 0x1c, 0x01, 0x02, 0x03, 0x04,
	0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c,
	0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14,
	0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c,
}

// MockPoolId is a sample pool ID for testing
var MockPoolId = []byte{
	0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90,
	0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90,
	0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90,
	0xab, 0xcd, 0xef, 0x12,
}

// MockVrfKeyHash is a sample VRF key hash for testing
var MockVrfKeyHash = []byte{
	0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0,
	0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0,
	0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0,
	0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0,
}

// Helper function to build mock current era result CBOR
func mockCurrentEraResult() []byte {
	result, err := NewCurrentEraResult(MockCurrentEra)
	if err != nil {
		return []byte{0x06} // fallback: CBOR encoded 6
	}
	return result
}

// Helper function to build mock epoch number result CBOR
func mockEpochNoResult() []byte {
	result, err := NewEpochNoResult(MockEpochNo)
	if err != nil {
		return []byte{0x19, 0x01, 0xf4} // fallback: CBOR encoded 500
	}
	return result
}

// Helper function to build mock system start result CBOR
func mockSystemStartResult() []byte {
	result, err := NewSystemStartResult(MockSystemStartTime)
	if err != nil {
		// Fallback: minimal CBOR array
		return []byte{0x83, 0x19, 0x07, 0xe1, 0x19, 0x01, 0x0a, 0x00}
	}
	return result
}

// Helper function to build mock chain point result CBOR
func mockChainPointResult() []byte {
	result, err := NewChainPointResult(MockPoint)
	if err != nil {
		return []byte{0x82, 0x00, 0x40} // fallback: minimal point
	}
	return result
}

// Helper function to build mock protocol params result CBOR
func mockProtocolParamsResult() []byte {
	params := ProtocolParamsResult{
		MinFeeA:            44,
		MinFeeB:            155381,
		MaxBlockBodySize:   90112,
		MaxTxSize:          16384,
		MaxBlockHeaderSize: 1100,
		KeyDeposit:         2000000,
		PoolDeposit:        500000000,
		EMax:               18,
		NOpt:               500,
		ProtocolMajorVer:   9,
		ProtocolMinorVer:   0,
		MinPoolCost:        170000000,
		CoinsPerUTxOByte:   4310,
	}
	result, err := NewProtocolParamsResult(params)
	if err != nil {
		return []byte{0xa0} // fallback: empty map
	}
	return result
}

// Helper function to build mock UTxO result CBOR
func mockUtxoResult() []byte {
	utxos := []UTxOResult{
		{
			TxHash:      MockTxHash,
			OutputIndex: 0,
			Address:     MockAddress,
			Amount:      10000000, // 10 ADA
			Assets:      nil,
		},
		{
			TxHash:      MockTxHash,
			OutputIndex: 1,
			Address:     MockAddress,
			Amount:      5000000, // 5 ADA
			Assets:      nil,
		},
	}
	result, err := NewUTxOByAddressResult(utxos)
	if err != nil {
		return []byte{0xa0} // fallback: empty map
	}
	return result
}

// Pre-defined conversations for common LocalStateQuery scenarios

// ConversationQueryCurrentEra is a pre-defined conversation for querying the current era:
// - Handshake request (generic)
// - Handshake NtC response
// - Acquire (volatile tip)
// - Acquired
// - Query (current era)
// - Result (era number)
// - Release
var ConversationQueryCurrentEra = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// Acquire at volatile tip
	NewAcquireEntryAny(),
	NewAcquiredEntry(),
	// Query current era
	NewQueryEntryAny(),
	NewResultEntry(mockCurrentEraResult()),
	// Release
	NewReleaseEntry(),
}

// ConversationQueryProtocolParams is a pre-defined conversation for querying protocol parameters:
// - Handshake request (generic)
// - Handshake NtC response
// - Acquire (volatile tip)
// - Acquired
// - Query (protocol parameters)
// - Result (protocol parameters)
// - Release
var ConversationQueryProtocolParams = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// Acquire at volatile tip
	NewAcquireEntryAny(),
	NewAcquiredEntry(),
	// Query protocol parameters
	NewQueryEntryAny(),
	NewResultEntry(mockProtocolParamsResult()),
	// Release
	NewReleaseEntry(),
}

// ConversationQueryUTxOByAddress is a pre-defined conversation for querying UTxOs by address:
// - Handshake request (generic)
// - Handshake NtC response
// - Acquire (volatile tip)
// - Acquired
// - Query (UTxO by address)
// - Result (UTxO set)
// - Release
var ConversationQueryUTxOByAddress = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// Acquire at volatile tip
	NewAcquireEntryAny(),
	NewAcquiredEntry(),
	// Query UTxO by address
	NewQueryEntryAny(),
	NewResultEntry(mockUtxoResult()),
	// Release
	NewReleaseEntry(),
}

// ConversationQueryMultiple is a pre-defined conversation for multiple queries in one session:
// - Handshake request (generic)
// - Handshake NtC response
// - Acquire (volatile tip)
// - Acquired
// - Query (current era) -> Result
// - Query (epoch number) -> Result
// - Query (protocol parameters) -> Result
// - Query (chain point) -> Result
// - Release
var ConversationQueryMultiple = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// Acquire at volatile tip
	NewAcquireEntryAny(),
	NewAcquiredEntry(),
	// Query 1: current era
	NewQueryEntryAny(),
	NewResultEntry(mockCurrentEraResult()),
	// Query 2: epoch number
	NewQueryEntryAny(),
	NewResultEntry(mockEpochNoResult()),
	// Query 3: protocol parameters
	NewQueryEntryAny(),
	NewResultEntry(mockProtocolParamsResult()),
	// Query 4: chain point
	NewQueryEntryAny(),
	NewResultEntry(mockChainPointResult()),
	// Release
	NewReleaseEntry(),
}

// ConversationQuerySystemStart is a pre-defined conversation for querying system start:
// - Handshake request (generic)
// - Handshake NtC response
// - Acquire (volatile tip)
// - Acquired
// - Query (system start)
// - Result (system start time)
// - Release
var ConversationQuerySystemStart = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// Acquire at volatile tip
	NewAcquireEntryAny(),
	NewAcquiredEntry(),
	// Query system start
	NewQueryEntryAny(),
	NewResultEntry(mockSystemStartResult()),
	// Release
	NewReleaseEntry(),
}

// ConversationQueryAcquireFailure is a pre-defined conversation for acquire failure scenario:
// - Handshake request (generic)
// - Handshake NtC response
// - Acquire (specific point)
// - Failure (point not on chain)
var ConversationQueryAcquireFailure = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// Acquire at specific point
	NewAcquireEntryAny(),
	// Failure - point not on chain
	NewAcquireFailurePointNotOnChainEntry(),
}

// ConversationQueryWithReacquire is a pre-defined conversation with reacquire:
// - Handshake request (generic)
// - Handshake NtC response
// - Acquire (volatile tip)
// - Acquired
// - Query (current era) -> Result
// - ReAcquire (volatile tip)
// - Acquired
// - Query (epoch number) -> Result
// - Release
var ConversationQueryWithReacquire = []ouroboros_mock.ConversationEntry{
	// Handshake
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
	// First acquire
	NewAcquireEntryAny(),
	NewAcquiredEntry(),
	// Query current era
	NewQueryEntryAny(),
	NewResultEntry(mockCurrentEraResult()),
	// ReAcquire at volatile tip
	NewReAcquireEntryAny(),
	NewAcquiredEntry(),
	// Query epoch number
	NewQueryEntryAny(),
	NewResultEntry(mockEpochNoResult()),
	// Release
	NewReleaseEntry(),
}
