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

package localstatequery_test

import (
	"bytes"
	"testing"
	"time"

	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	lsq "github.com/blinklabs-io/ouroboros-mock/localstatequery"
)

// Test constants
var (
	testBlockHash = []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
	}
	testPoint     = pcommon.NewPoint(12345, testBlockHash)
	testQueryCBOR = []byte{0x82, 0x00, 0x82, 0x02, 0x81, 0x00}
	testResult    = []byte{0x06}
)

// TestConversationEntryAcquire tests the ConversationEntryAcquire type
func TestConversationEntryAcquire(t *testing.T) {
	t.Run("without point", func(t *testing.T) {
		entry := lsq.ConversationEntryAcquire{}
		result := entry.ToEntry()

		if result.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if result.MessageType != localstatequery.MessageTypeAcquire {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeAcquire,
				result.MessageType,
			)
		}
		if result.Message != nil {
			t.Error("expected Message to be nil when Point is not set")
		}
		if result.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})

	t.Run("with point", func(t *testing.T) {
		point := testPoint
		entry := lsq.ConversationEntryAcquire{Point: &point}
		result := entry.ToEntry()

		if result.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if result.MessageType != localstatequery.MessageTypeAcquire {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeAcquire,
				result.MessageType,
			)
		}
		if result.Message == nil {
			t.Error("expected Message to be set when Point is provided")
		}
	})
}

// TestConversationEntryAcquired tests the ConversationEntryAcquired type
func TestConversationEntryAcquired(t *testing.T) {
	entry := lsq.ConversationEntryAcquired{}
	result := entry.ToEntry()

	if result.ProtocolId != localstatequery.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			localstatequery.ProtocolId,
			result.ProtocolId,
		)
	}
	if !result.IsResponse {
		t.Error("expected IsResponse to be true")
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestConversationEntryAcquireFailure tests the ConversationEntryAcquireFailure type
func TestConversationEntryAcquireFailure(t *testing.T) {
	testCases := []struct {
		name    string
		failure uint8
	}{
		{"PointTooOld", localstatequery.AcquireFailurePointTooOld},
		{"PointNotOnChain", localstatequery.AcquireFailurePointNotOnChain},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := lsq.ConversationEntryAcquireFailure{Failure: tc.failure}
			result := entry.ToEntry()

			if result.ProtocolId != localstatequery.ProtocolId {
				t.Errorf(
					"expected ProtocolId %d, got %d",
					localstatequery.ProtocolId,
					result.ProtocolId,
				)
			}
			if !result.IsResponse {
				t.Error("expected IsResponse to be true")
			}
			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
			}
		})
	}
}

// TestConversationEntryReAcquire tests the ConversationEntryReAcquire type
func TestConversationEntryReAcquire(t *testing.T) {
	t.Run("without point", func(t *testing.T) {
		entry := lsq.ConversationEntryReAcquire{}
		result := entry.ToEntry()

		if result.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if result.MessageType != localstatequery.MessageTypeReacquire {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeReacquire,
				result.MessageType,
			)
		}
		if result.Message != nil {
			t.Error("expected Message to be nil when Point is not set")
		}
		if result.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})

	t.Run("with point", func(t *testing.T) {
		point := testPoint
		entry := lsq.ConversationEntryReAcquire{Point: &point}
		result := entry.ToEntry()

		if result.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if result.MessageType != localstatequery.MessageTypeReacquire {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeReacquire,
				result.MessageType,
			)
		}
		if result.Message == nil {
			t.Error("expected Message to be set when Point is provided")
		}
	})
}

// TestConversationEntryQuery tests the ConversationEntryQuery type
func TestConversationEntryQuery(t *testing.T) {
	t.Run("without query", func(t *testing.T) {
		entry := lsq.ConversationEntryQuery{}
		result := entry.ToEntry()

		if result.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if result.MessageType != localstatequery.MessageTypeQuery {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeQuery,
				result.MessageType,
			)
		}
		if result.Message != nil {
			t.Error("expected Message to be nil when Query is not set")
		}
		if result.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})

	t.Run("with query", func(t *testing.T) {
		entry := lsq.ConversationEntryQuery{Query: testQueryCBOR}
		result := entry.ToEntry()

		if result.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				result.ProtocolId,
			)
		}
		if result.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if result.MessageType != localstatequery.MessageTypeQuery {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeQuery,
				result.MessageType,
			)
		}
		if result.Message == nil {
			t.Error("expected Message to be set when Query is provided")
		}
	})
}

// TestConversationEntryResult tests the ConversationEntryResult type
func TestConversationEntryResult(t *testing.T) {
	entry := lsq.ConversationEntryResult{Result: testResult}
	result := entry.ToEntry()

	if result.ProtocolId != localstatequery.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			localstatequery.ProtocolId,
			result.ProtocolId,
		)
	}
	if !result.IsResponse {
		t.Error("expected IsResponse to be true")
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Messages))
	}
}

// TestConversationEntryRelease tests the ConversationEntryRelease type
func TestConversationEntryRelease(t *testing.T) {
	entry := lsq.ConversationEntryRelease{}
	result := entry.ToEntry()

	if result.ProtocolId != localstatequery.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			localstatequery.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse {
		t.Error("expected IsResponse to be false")
	}
	if result.MessageType != localstatequery.MessageTypeRelease {
		t.Errorf(
			"expected MessageType %d, got %d",
			localstatequery.MessageTypeRelease,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestConversationEntryDone tests the ConversationEntryDone type
func TestConversationEntryDone(t *testing.T) {
	entry := lsq.ConversationEntryDone{}
	result := entry.ToEntry()

	if result.ProtocolId != localstatequery.ProtocolId {
		t.Errorf(
			"expected ProtocolId %d, got %d",
			localstatequery.ProtocolId,
			result.ProtocolId,
		)
	}
	if result.IsResponse {
		t.Error("expected IsResponse to be false for Done message")
	}
	if result.MessageType != localstatequery.MessageTypeDone {
		t.Errorf(
			"expected MessageType %d, got %d",
			localstatequery.MessageTypeDone,
			result.MessageType,
		)
	}
	if result.MsgFromCborFunc == nil {
		t.Error("expected MsgFromCborFunc to be set")
	}
}

// TestHelperFunctions tests all helper functions for creating entries
func TestHelperFunctions(t *testing.T) {
	t.Run("NewAcquireEntry", func(t *testing.T) {
		point := testPoint
		entry := lsq.NewAcquireEntry(&point)

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if entry.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if entry.MessageType != localstatequery.MessageTypeAcquire {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeAcquire,
				entry.MessageType,
			)
		}
		if entry.Message == nil {
			t.Error("expected Message to be set")
		}
	})

	t.Run("NewAcquireEntryAny", func(t *testing.T) {
		entry := lsq.NewAcquireEntryAny()

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if entry.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if entry.MessageType != localstatequery.MessageTypeAcquire {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeAcquire,
				entry.MessageType,
			)
		}
		if entry.Message != nil {
			t.Error("expected Message to be nil for 'any' entry")
		}
	})

	t.Run("NewAcquiredEntry", func(t *testing.T) {
		entry := lsq.NewAcquiredEntry()

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if !entry.IsResponse {
			t.Error("expected IsResponse to be true")
		}
		if len(entry.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(entry.Messages))
		}
	})

	t.Run("NewAcquireFailureEntry", func(t *testing.T) {
		entry := lsq.NewAcquireFailureEntry(
			localstatequery.AcquireFailurePointTooOld,
		)

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if !entry.IsResponse {
			t.Error("expected IsResponse to be true")
		}
		if len(entry.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(entry.Messages))
		}
	})

	t.Run("NewAcquireFailurePointTooOldEntry", func(t *testing.T) {
		entry := lsq.NewAcquireFailurePointTooOldEntry()

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if !entry.IsResponse {
			t.Error("expected IsResponse to be true")
		}
		if len(entry.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(entry.Messages))
		}
	})

	t.Run("NewAcquireFailurePointNotOnChainEntry", func(t *testing.T) {
		entry := lsq.NewAcquireFailurePointNotOnChainEntry()

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if !entry.IsResponse {
			t.Error("expected IsResponse to be true")
		}
		if len(entry.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(entry.Messages))
		}
	})

	t.Run("NewReAcquireEntry", func(t *testing.T) {
		point := testPoint
		entry := lsq.NewReAcquireEntry(&point)

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if entry.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if entry.MessageType != localstatequery.MessageTypeReacquire {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeReacquire,
				entry.MessageType,
			)
		}
		if entry.Message == nil {
			t.Error("expected Message to be set")
		}
	})

	t.Run("NewReAcquireEntryAny", func(t *testing.T) {
		entry := lsq.NewReAcquireEntryAny()

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if entry.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if entry.MessageType != localstatequery.MessageTypeReacquire {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeReacquire,
				entry.MessageType,
			)
		}
		if entry.Message != nil {
			t.Error("expected Message to be nil for 'any' entry")
		}
	})

	t.Run("NewQueryEntry", func(t *testing.T) {
		entry := lsq.NewQueryEntry(testQueryCBOR)

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if entry.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if entry.MessageType != localstatequery.MessageTypeQuery {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeQuery,
				entry.MessageType,
			)
		}
		if entry.Message == nil {
			t.Error("expected Message to be set")
		}
	})

	t.Run("NewQueryEntryAny", func(t *testing.T) {
		entry := lsq.NewQueryEntryAny()

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if entry.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if entry.MessageType != localstatequery.MessageTypeQuery {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeQuery,
				entry.MessageType,
			)
		}
		if entry.Message != nil {
			t.Error("expected Message to be nil for 'any' entry")
		}
	})

	t.Run("NewResultEntry", func(t *testing.T) {
		entry := lsq.NewResultEntry(testResult)

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if !entry.IsResponse {
			t.Error("expected IsResponse to be true")
		}
		if len(entry.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(entry.Messages))
		}
	})

	t.Run("NewReleaseEntry", func(t *testing.T) {
		entry := lsq.NewReleaseEntry()

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if entry.IsResponse {
			t.Error("expected IsResponse to be false")
		}
		if entry.MessageType != localstatequery.MessageTypeRelease {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeRelease,
				entry.MessageType,
			)
		}
	})

	t.Run("NewDoneEntry", func(t *testing.T) {
		entry := lsq.NewDoneEntry()

		if entry.ProtocolId != localstatequery.ProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				localstatequery.ProtocolId,
				entry.ProtocolId,
			)
		}
		if entry.IsResponse {
			t.Error("expected IsResponse to be false for Done message")
		}
		if entry.MessageType != localstatequery.MessageTypeDone {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeDone,
				entry.MessageType,
			)
		}
		if entry.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})
}

// TestQueryBuilders tests the query builder functions
func TestQueryBuilders(t *testing.T) {
	t.Run("NewCurrentEraQuery", func(t *testing.T) {
		query, err := lsq.NewCurrentEraQuery()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(query) == 0 {
			t.Error("expected non-empty query")
		}
	})

	t.Run("NewEpochNoQuery", func(t *testing.T) {
		query, err := lsq.NewEpochNoQuery(6) // Conway era
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(query) == 0 {
			t.Error("expected non-empty query")
		}
	})

	t.Run("NewSystemStartQuery", func(t *testing.T) {
		query, err := lsq.NewSystemStartQuery()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(query) == 0 {
			t.Error("expected non-empty query")
		}
	})

	t.Run("NewChainPointQuery", func(t *testing.T) {
		query, err := lsq.NewChainPointQuery()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(query) == 0 {
			t.Error("expected non-empty query")
		}
	})

	t.Run("NewProtocolParamsQuery", func(t *testing.T) {
		query, err := lsq.NewProtocolParamsQuery(6) // Conway era
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(query) == 0 {
			t.Error("expected non-empty query")
		}
	})

	t.Run("NewStakeDistributionQuery", func(t *testing.T) {
		query, err := lsq.NewStakeDistributionQuery(6) // Conway era
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(query) == 0 {
			t.Error("expected non-empty query")
		}
	})
}

// TestResultBuilders tests the result builder functions
func TestResultBuilders(t *testing.T) {
	t.Run("NewCurrentEraResult", func(t *testing.T) {
		result, err := lsq.NewCurrentEraResult(6)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("NewEpochNoResult", func(t *testing.T) {
		result, err := lsq.NewEpochNoResult(500)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("NewSystemStartResult", func(t *testing.T) {
		startTime := time.Date(2017, 9, 23, 21, 44, 51, 0, time.UTC)
		result, err := lsq.NewSystemStartResult(startTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("NewChainPointResult with origin", func(t *testing.T) {
		originPoint := pcommon.NewPoint(0, nil)
		result, err := lsq.NewChainPointResult(originPoint)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("NewChainPointResult with point", func(t *testing.T) {
		result, err := lsq.NewChainPointResult(testPoint)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("NewProtocolParamsResult", func(t *testing.T) {
		params := lsq.ProtocolParamsResult{
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
		result, err := lsq.NewProtocolParamsResult(params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("NewUTxOByAddressResult empty", func(t *testing.T) {
		result, err := lsq.NewUTxOByAddressResult(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("NewUTxOByAddressResult with utxos", func(t *testing.T) {
		utxos := []lsq.UTxOResult{
			{
				TxHash:      testBlockHash,
				OutputIndex: 0,
				Address:     lsq.MockAddress,
				Amount:      10000000,
				Assets:      nil,
			},
		}
		result, err := lsq.NewUTxOByAddressResult(utxos)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("NewUTxOByAddressResult with assets", func(t *testing.T) {
		utxos := []lsq.UTxOResult{
			{
				TxHash:      testBlockHash,
				OutputIndex: 0,
				Address:     lsq.MockAddress,
				Amount:      10000000,
				Assets: map[string]map[string]uint64{
					"policy1": {
						"asset1": 100,
					},
				},
			},
		}
		result, err := lsq.NewUTxOByAddressResult(utxos)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("NewStakeDistributionResult empty", func(t *testing.T) {
		result, err := lsq.NewStakeDistributionResult(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("NewStakeDistributionResult with entries", func(t *testing.T) {
		distribution := []lsq.StakeDistributionEntry{
			{
				PoolId:        lsq.MockPoolId,
				StakeFraction: [2]uint64{1, 100},
				VrfKeyHash:    lsq.MockVrfKeyHash,
			},
		}
		result, err := lsq.NewStakeDistributionResult(distribution)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("NewEmptyResult", func(t *testing.T) {
		result, err := lsq.NewEmptyResult()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})
}

// TestQueryTypeConstants tests that query type constants are exported correctly
func TestQueryTypeConstants(t *testing.T) {
	// Verify constants match the underlying gouroboros values
	if lsq.QueryTypeBlock != localstatequery.QueryTypeBlock {
		t.Errorf(
			"QueryTypeBlock mismatch: expected %d, got %d",
			localstatequery.QueryTypeBlock,
			lsq.QueryTypeBlock,
		)
	}
	if lsq.QueryTypeSystemStart != localstatequery.QueryTypeSystemStart {
		t.Errorf(
			"QueryTypeSystemStart mismatch: expected %d, got %d",
			localstatequery.QueryTypeSystemStart,
			lsq.QueryTypeSystemStart,
		)
	}
	if lsq.QueryTypeChainPoint != localstatequery.QueryTypeChainPoint {
		t.Errorf(
			"QueryTypeChainPoint mismatch: expected %d, got %d",
			localstatequery.QueryTypeChainPoint,
			lsq.QueryTypeChainPoint,
		)
	}
	if lsq.QueryTypeShelley != localstatequery.QueryTypeShelley {
		t.Errorf(
			"QueryTypeShelley mismatch: expected %d, got %d",
			localstatequery.QueryTypeShelley,
			lsq.QueryTypeShelley,
		)
	}
	if lsq.QueryTypeHardFork != localstatequery.QueryTypeHardFork {
		t.Errorf(
			"QueryTypeHardFork mismatch: expected %d, got %d",
			localstatequery.QueryTypeHardFork,
			lsq.QueryTypeHardFork,
		)
	}
	if lsq.QueryTypeHardForkCurrentEra != localstatequery.QueryTypeHardForkCurrentEra {
		t.Errorf(
			"QueryTypeHardForkCurrentEra mismatch: expected %d, got %d",
			localstatequery.QueryTypeHardForkCurrentEra,
			lsq.QueryTypeHardForkCurrentEra,
		)
	}
	if lsq.QueryTypeShelleyEpochNo != localstatequery.QueryTypeShelleyEpochNo {
		t.Errorf(
			"QueryTypeShelleyEpochNo mismatch: expected %d, got %d",
			localstatequery.QueryTypeShelleyEpochNo,
			lsq.QueryTypeShelleyEpochNo,
		)
	}
	if lsq.QueryTypeShelleyCurrentProtocolParams != localstatequery.QueryTypeShelleyCurrentProtocolParams {
		t.Errorf(
			"QueryTypeShelleyCurrentProtocolParams mismatch: expected %d, got %d",
			localstatequery.QueryTypeShelleyCurrentProtocolParams,
			lsq.QueryTypeShelleyCurrentProtocolParams,
		)
	}
	if lsq.QueryTypeShelleyStakeDistribution != localstatequery.QueryTypeShelleyStakeDistribution {
		t.Errorf(
			"QueryTypeShelleyStakeDistribution mismatch: expected %d, got %d",
			localstatequery.QueryTypeShelleyStakeDistribution,
			lsq.QueryTypeShelleyStakeDistribution,
		)
	}
	if lsq.QueryTypeShelleyUtxoByAddress != localstatequery.QueryTypeShelleyUtxoByAddress {
		t.Errorf(
			"QueryTypeShelleyUtxoByAddress mismatch: expected %d, got %d",
			localstatequery.QueryTypeShelleyUtxoByAddress,
			lsq.QueryTypeShelleyUtxoByAddress,
		)
	}
}

// TestMockConstants tests the mock constants and fixtures
func TestMockConstants(t *testing.T) {
	t.Run("MockCurrentEra", func(t *testing.T) {
		if lsq.MockCurrentEra != 6 {
			t.Errorf(
				"expected MockCurrentEra to be 6 (Conway), got %d",
				lsq.MockCurrentEra,
			)
		}
	})

	t.Run("MockEpochNo", func(t *testing.T) {
		if lsq.MockEpochNo != 500 {
			t.Errorf("expected MockEpochNo to be 500, got %d", lsq.MockEpochNo)
		}
	})

	t.Run("MockSystemStartTime", func(t *testing.T) {
		expected := time.Date(2017, 9, 23, 21, 44, 51, 0, time.UTC)
		if !lsq.MockSystemStartTime.Equal(expected) {
			t.Errorf(
				"expected MockSystemStartTime to be %v, got %v",
				expected,
				lsq.MockSystemStartTime,
			)
		}
	})

	t.Run("MockBlockHash", func(t *testing.T) {
		if len(lsq.MockBlockHash) != 32 {
			t.Errorf(
				"expected MockBlockHash to be 32 bytes, got %d",
				len(lsq.MockBlockHash),
			)
		}
	})

	t.Run("MockPoint", func(t *testing.T) {
		if lsq.MockPoint.Slot != 100000 {
			t.Errorf(
				"expected MockPoint.Slot to be 100000, got %d",
				lsq.MockPoint.Slot,
			)
		}
		if !bytes.Equal(lsq.MockPoint.Hash, lsq.MockBlockHash) {
			t.Error("expected MockPoint.Hash to equal MockBlockHash")
		}
	})

	t.Run("MockTxHash", func(t *testing.T) {
		if len(lsq.MockTxHash) != 32 {
			t.Errorf(
				"expected MockTxHash to be 32 bytes, got %d",
				len(lsq.MockTxHash),
			)
		}
	})

	t.Run("MockAddress", func(t *testing.T) {
		if len(lsq.MockAddress) == 0 {
			t.Error("expected MockAddress to be non-empty")
		}
	})

	t.Run("MockPoolId", func(t *testing.T) {
		if len(lsq.MockPoolId) != 28 {
			t.Errorf(
				"expected MockPoolId to be 28 bytes, got %d",
				len(lsq.MockPoolId),
			)
		}
	})

	t.Run("MockVrfKeyHash", func(t *testing.T) {
		if len(lsq.MockVrfKeyHash) != 32 {
			t.Errorf(
				"expected MockVrfKeyHash to be 32 bytes, got %d",
				len(lsq.MockVrfKeyHash),
			)
		}
	})
}

// TestFixtureConversations tests that fixture conversations are valid slices
func TestFixtureConversations(t *testing.T) {
	t.Run("ConversationQueryCurrentEra", func(t *testing.T) {
		if len(lsq.ConversationQueryCurrentEra) == 0 {
			t.Error("expected ConversationQueryCurrentEra to be non-empty")
		}
		// Verify it has expected structure: handshake + acquire + acquired + query + result + release
		if len(lsq.ConversationQueryCurrentEra) < 7 {
			t.Errorf(
				"expected ConversationQueryCurrentEra to have at least 7 entries, got %d",
				len(lsq.ConversationQueryCurrentEra),
			)
		}
	})

	t.Run("ConversationQueryProtocolParams", func(t *testing.T) {
		if len(lsq.ConversationQueryProtocolParams) == 0 {
			t.Error("expected ConversationQueryProtocolParams to be non-empty")
		}
		if len(lsq.ConversationQueryProtocolParams) < 7 {
			t.Errorf(
				"expected ConversationQueryProtocolParams to have at least 7 entries, got %d",
				len(lsq.ConversationQueryProtocolParams),
			)
		}
	})

	t.Run("ConversationQueryUTxOByAddress", func(t *testing.T) {
		if len(lsq.ConversationQueryUTxOByAddress) == 0 {
			t.Error("expected ConversationQueryUTxOByAddress to be non-empty")
		}
		if len(lsq.ConversationQueryUTxOByAddress) < 7 {
			t.Errorf(
				"expected ConversationQueryUTxOByAddress to have at least 7 entries, got %d",
				len(lsq.ConversationQueryUTxOByAddress),
			)
		}
	})

	t.Run("ConversationQueryMultiple", func(t *testing.T) {
		if len(lsq.ConversationQueryMultiple) == 0 {
			t.Error("expected ConversationQueryMultiple to be non-empty")
		}
		// Should have: handshake(2) + acquire(1) + acquired(1) + 4*(query+result) + release(1) = 13
		if len(lsq.ConversationQueryMultiple) < 13 {
			t.Errorf(
				"expected ConversationQueryMultiple to have at least 13 entries, got %d",
				len(lsq.ConversationQueryMultiple),
			)
		}
	})

	t.Run("ConversationQuerySystemStart", func(t *testing.T) {
		if len(lsq.ConversationQuerySystemStart) == 0 {
			t.Error("expected ConversationQuerySystemStart to be non-empty")
		}
		if len(lsq.ConversationQuerySystemStart) < 7 {
			t.Errorf(
				"expected ConversationQuerySystemStart to have at least 7 entries, got %d",
				len(lsq.ConversationQuerySystemStart),
			)
		}
	})

	t.Run("ConversationQueryAcquireFailure", func(t *testing.T) {
		if len(lsq.ConversationQueryAcquireFailure) == 0 {
			t.Error("expected ConversationQueryAcquireFailure to be non-empty")
		}
		// Should have: handshake(2) + acquire(1) + failure(1) = 4
		if len(lsq.ConversationQueryAcquireFailure) < 4 {
			t.Errorf(
				"expected ConversationQueryAcquireFailure to have at least 4 entries, got %d",
				len(lsq.ConversationQueryAcquireFailure),
			)
		}
	})

	t.Run("ConversationQueryWithReacquire", func(t *testing.T) {
		if len(lsq.ConversationQueryWithReacquire) == 0 {
			t.Error("expected ConversationQueryWithReacquire to be non-empty")
		}
		// Should have: handshake(2) + acquire(1) + acquired(1) + query+result(2) + reacquire(1) + acquired(1) + query+result(2) + release(1) = 11
		if len(lsq.ConversationQueryWithReacquire) < 11 {
			t.Errorf(
				"expected ConversationQueryWithReacquire to have at least 11 entries, got %d",
				len(lsq.ConversationQueryWithReacquire),
			)
		}
	})
}

// TestInputEntriesHaveMsgFromCborFunc verifies that all input entries have MsgFromCborFunc set
func TestInputEntriesHaveMsgFromCborFunc(t *testing.T) {
	t.Run("ConversationEntryAcquire", func(t *testing.T) {
		result := lsq.ConversationEntryAcquire{}.ToEntry()
		if result.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})

	t.Run("ConversationEntryReAcquire", func(t *testing.T) {
		result := lsq.ConversationEntryReAcquire{}.ToEntry()
		if result.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})

	t.Run("ConversationEntryQuery", func(t *testing.T) {
		result := lsq.ConversationEntryQuery{}.ToEntry()
		if result.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})

	t.Run("ConversationEntryRelease", func(t *testing.T) {
		result := lsq.ConversationEntryRelease{}.ToEntry()
		if result.MsgFromCborFunc == nil {
			t.Error("expected MsgFromCborFunc to be set")
		}
	})
}

// TestOutputEntriesHaveMessages verifies that all output entries have at least one message
func TestOutputEntriesHaveMessages(t *testing.T) {
	t.Run("ConversationEntryAcquired", func(t *testing.T) {
		result := lsq.ConversationEntryAcquired{}.ToEntry()
		if len(result.Messages) == 0 {
			t.Error("expected at least one message")
		}
	})

	t.Run("ConversationEntryAcquireFailure", func(t *testing.T) {
		result := lsq.ConversationEntryAcquireFailure{Failure: 0}.ToEntry()
		if len(result.Messages) == 0 {
			t.Error("expected at least one message")
		}
	})

	t.Run("ConversationEntryResult", func(t *testing.T) {
		result := lsq.ConversationEntryResult{Result: []byte{0x00}}.ToEntry()
		if len(result.Messages) == 0 {
			t.Error("expected at least one message")
		}
	})

	t.Run("ConversationEntryDone", func(t *testing.T) {
		result := lsq.ConversationEntryDone{}.ToEntry()
		if result.MessageType != localstatequery.MessageTypeDone {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeDone,
				result.MessageType,
			)
		}
	})
}

// TestProtocolIdConsistency verifies all entries use the correct protocol ID
func TestProtocolIdConsistency(t *testing.T) {
	expectedProtocolId := localstatequery.ProtocolId

	t.Run("Acquire", func(t *testing.T) {
		entry := lsq.ConversationEntryAcquire{}
		result := entry.ToEntry()
		if result.ProtocolId != expectedProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				expectedProtocolId,
				result.ProtocolId,
			)
		}
	})

	t.Run("ReAcquire", func(t *testing.T) {
		entry := lsq.ConversationEntryReAcquire{}
		result := entry.ToEntry()
		if result.ProtocolId != expectedProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				expectedProtocolId,
				result.ProtocolId,
			)
		}
	})

	t.Run("Query", func(t *testing.T) {
		entry := lsq.ConversationEntryQuery{}
		result := entry.ToEntry()
		if result.ProtocolId != expectedProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				expectedProtocolId,
				result.ProtocolId,
			)
		}
	})

	t.Run("Release", func(t *testing.T) {
		entry := lsq.ConversationEntryRelease{}
		result := entry.ToEntry()
		if result.ProtocolId != expectedProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				expectedProtocolId,
				result.ProtocolId,
			)
		}
	})

	t.Run("Acquired", func(t *testing.T) {
		entry := lsq.ConversationEntryAcquired{}
		result := entry.ToEntry()
		if result.ProtocolId != expectedProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				expectedProtocolId,
				result.ProtocolId,
			)
		}
	})

	t.Run("AcquireFailure", func(t *testing.T) {
		entry := lsq.ConversationEntryAcquireFailure{}
		result := entry.ToEntry()
		if result.ProtocolId != expectedProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				expectedProtocolId,
				result.ProtocolId,
			)
		}
	})

	t.Run("Result", func(t *testing.T) {
		entry := lsq.ConversationEntryResult{}
		result := entry.ToEntry()
		if result.ProtocolId != expectedProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				expectedProtocolId,
				result.ProtocolId,
			)
		}
	})

	t.Run("Done", func(t *testing.T) {
		entry := lsq.ConversationEntryDone{}
		result := entry.ToEntry()
		if result.ProtocolId != expectedProtocolId {
			t.Errorf(
				"expected ProtocolId %d, got %d",
				expectedProtocolId,
				result.ProtocolId,
			)
		}
	})
}

// TestIsResponseValues verifies correct IsResponse values for all entry types
func TestIsResponseValues(t *testing.T) {
	t.Run("Acquire should have IsResponse=false", func(t *testing.T) {
		entry := lsq.ConversationEntryAcquire{}
		result := entry.ToEntry()
		if result.IsResponse {
			t.Error("ConversationEntryAcquire should have IsResponse=false")
		}
	})

	t.Run("ReAcquire should have IsResponse=false", func(t *testing.T) {
		entry := lsq.ConversationEntryReAcquire{}
		result := entry.ToEntry()
		if result.IsResponse {
			t.Error("ConversationEntryReAcquire should have IsResponse=false")
		}
	})

	t.Run("Query should have IsResponse=false", func(t *testing.T) {
		entry := lsq.ConversationEntryQuery{}
		result := entry.ToEntry()
		if result.IsResponse {
			t.Error("ConversationEntryQuery should have IsResponse=false")
		}
	})

	t.Run("Release should have IsResponse=false", func(t *testing.T) {
		entry := lsq.ConversationEntryRelease{}
		result := entry.ToEntry()
		if result.IsResponse {
			t.Error("ConversationEntryRelease should have IsResponse=false")
		}
	})

	t.Run("Acquired should have IsResponse=true", func(t *testing.T) {
		entry := lsq.ConversationEntryAcquired{}
		result := entry.ToEntry()
		if !result.IsResponse {
			t.Error("ConversationEntryAcquired should have IsResponse=true")
		}
	})

	t.Run("AcquireFailure should have IsResponse=true", func(t *testing.T) {
		entry := lsq.ConversationEntryAcquireFailure{}
		result := entry.ToEntry()
		if !result.IsResponse {
			t.Error(
				"ConversationEntryAcquireFailure should have IsResponse=true",
			)
		}
	})

	t.Run("Result should have IsResponse=true", func(t *testing.T) {
		entry := lsq.ConversationEntryResult{}
		result := entry.ToEntry()
		if !result.IsResponse {
			t.Error("ConversationEntryResult should have IsResponse=true")
		}
	})

	t.Run("Done should have IsResponse=false", func(t *testing.T) {
		entry := lsq.ConversationEntryDone{}
		result := entry.ToEntry()
		if result.IsResponse {
			t.Error(
				"ConversationEntryDone should have IsResponse=false (client-initiated)",
			)
		}
	})
}

// TestMessageTypeValues verifies correct message types for input entries
func TestMessageTypeValues(t *testing.T) {
	t.Run("Acquire", func(t *testing.T) {
		entry := lsq.ConversationEntryAcquire{}
		result := entry.ToEntry()
		if result.MessageType != localstatequery.MessageTypeAcquire {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeAcquire,
				result.MessageType,
			)
		}
	})

	t.Run("ReAcquire", func(t *testing.T) {
		entry := lsq.ConversationEntryReAcquire{}
		result := entry.ToEntry()
		if result.MessageType != localstatequery.MessageTypeReacquire {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeReacquire,
				result.MessageType,
			)
		}
	})

	t.Run("Query", func(t *testing.T) {
		entry := lsq.ConversationEntryQuery{}
		result := entry.ToEntry()
		if result.MessageType != localstatequery.MessageTypeQuery {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeQuery,
				result.MessageType,
			)
		}
	})

	t.Run("Release", func(t *testing.T) {
		entry := lsq.ConversationEntryRelease{}
		result := entry.ToEntry()
		if result.MessageType != localstatequery.MessageTypeRelease {
			t.Errorf(
				"expected MessageType %d, got %d",
				localstatequery.MessageTypeRelease,
				result.MessageType,
			)
		}
	})
}
