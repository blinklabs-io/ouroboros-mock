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

package chainsync_test

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	cs "github.com/blinklabs-io/ouroboros-mock/chainsync"
)

// Test data for tests
var (
	testBlockHash = []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
	}
	testPoint = pcommon.NewPoint(100, testBlockHash)
	testTip   = pcommon.Tip{
		Point:       pcommon.NewPoint(1000, testBlockHash),
		BlockNumber: 100,
	}
	testBlockCbor = []byte{0x82, 0x00, 0xa0}
)

// TestConversationEntryFindIntersect tests the FindIntersect entry type
func TestConversationEntryFindIntersect(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		points   []pcommon.Point
		wantNil  bool
		expected uint16
	}{
		{
			name:     "NtC with nil points",
			isNtC:    true,
			points:   nil,
			wantNil:  true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN with nil points",
			isNtC:    false,
			points:   nil,
			wantNil:  true,
			expected: chainsync.ProtocolIdNtN,
		},
		{
			name:     "NtC with points",
			isNtC:    true,
			points:   []pcommon.Point{testPoint},
			wantNil:  false,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN with points",
			isNtC:    false,
			points:   []pcommon.Point{testPoint},
			wantNil:  false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := cs.ConversationEntryFindIntersect{Points: tc.points}
			result := entry.ToEntry(tc.isNtC)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != false {
				t.Errorf("expected IsResponse to be false, got true")
			}

			if result.MessageType != chainsync.MessageTypeFindIntersect {
				t.Errorf(
					"expected message type %d, got %d",
					chainsync.MessageTypeFindIntersect,
					result.MessageType,
				)
			}

			if result.MsgFromCborFunc == nil {
				t.Error("expected MsgFromCborFunc to be non-nil")
			}

			if tc.wantNil && result.Message != nil {
				t.Error("expected Message to be nil when Points is nil")
			}

			if !tc.wantNil && result.Message == nil {
				t.Error(
					"expected Message to be non-nil when Points is provided",
				)
			}
		})
	}
}

// TestConversationEntryIntersectFound tests the IntersectFound entry type
func TestConversationEntryIntersectFound(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := cs.ConversationEntryIntersectFound{
				Point: testPoint,
				Tip:   testTip,
			}
			result := entry.ToEntry(tc.isNtC)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != true {
				t.Errorf("expected IsResponse to be true, got false")
			}

			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
			}
		})
	}
}

// TestConversationEntryIntersectNotFound tests the IntersectNotFound entry type
func TestConversationEntryIntersectNotFound(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := cs.ConversationEntryIntersectNotFound{Tip: testTip}
			result := entry.ToEntry(tc.isNtC)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != true {
				t.Errorf("expected IsResponse to be true, got false")
			}

			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
			}
		})
	}
}

// TestConversationEntryRequestNext tests the RequestNext entry type
func TestConversationEntryRequestNext(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := cs.ConversationEntryRequestNext{}
			result := entry.ToEntry(tc.isNtC)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != false {
				t.Errorf("expected IsResponse to be false, got true")
			}

			if result.MessageType != chainsync.MessageTypeRequestNext {
				t.Errorf(
					"expected message type %d, got %d",
					chainsync.MessageTypeRequestNext,
					result.MessageType,
				)
			}

			if result.MsgFromCborFunc == nil {
				t.Error("expected MsgFromCborFunc to be non-nil")
			}
		})
	}
}

// TestConversationEntryAwaitReply tests the AwaitReply entry type
func TestConversationEntryAwaitReply(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := cs.ConversationEntryAwaitReply{}
			result := entry.ToEntry(tc.isNtC)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != true {
				t.Errorf("expected IsResponse to be true, got false")
			}

			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
			}
		})
	}
}

// TestConversationEntryRollForward tests the RollForward entry type
func TestConversationEntryRollForward(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := cs.ConversationEntryRollForward{
				BlockType: 0,
				BlockCbor: testBlockCbor,
				Tip:       testTip,
			}
			result, err := entry.ToEntry(tc.isNtC)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != true {
				t.Errorf("expected IsResponse to be true, got false")
			}

			// Note: RollForward may return empty messages on error,
			// but with valid CBOR it should return 1 message
			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
			}
		})
	}
}

// TestConversationEntryRollForwardNtN tests the RollForwardNtN entry type
func TestConversationEntryRollForwardNtN(t *testing.T) {
	entry := cs.ConversationEntryRollForwardNtN{
		Era:       1, // Shelley
		ByronType: 0,
		BlockCbor: testBlockCbor,
		Tip:       testTip,
	}
	result, err := entry.ToEntry()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProtocolId != chainsync.ProtocolIdNtN {
		t.Errorf(
			"expected protocol ID %d, got %d",
			chainsync.ProtocolIdNtN,
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

// TestConversationEntryRollBackward tests the RollBackward entry type
func TestConversationEntryRollBackward(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := cs.ConversationEntryRollBackward{
				Point: testPoint,
				Tip:   testTip,
			}
			result := entry.ToEntry(tc.isNtC)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != true {
				t.Errorf("expected IsResponse to be true, got false")
			}

			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
			}
		})
	}
}

// TestConversationEntryDone tests the Done entry type
func TestConversationEntryDone(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := cs.ConversationEntryDone{}
			result := entry.ToEntry(tc.isNtC)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			// Done is sent by client, not as a response
			if result.IsResponse != false {
				t.Errorf("expected IsResponse to be false, got true")
			}

			if result.MessageType != chainsync.MessageTypeDone {
				t.Errorf(
					"expected MessageType %d, got %d",
					chainsync.MessageTypeDone,
					result.MessageType,
				)
			}

			if result.MsgFromCborFunc == nil {
				t.Error("expected MsgFromCborFunc to be set")
			}
		})
	}
}

// TestNewMsgFromCborFunc tests the NewMsgFromCborFunc helper
func TestNewMsgFromCborFunc(t *testing.T) {
	t.Run("NtC", func(t *testing.T) {
		fn := cs.NewMsgFromCborFunc(true)
		if fn == nil {
			t.Error("expected non-nil function for NtC")
		}
	})

	t.Run("NtN", func(t *testing.T) {
		fn := cs.NewMsgFromCborFunc(false)
		if fn == nil {
			t.Error("expected non-nil function for NtN")
		}
	})
}

// TestNewFindIntersectEntry tests the NewFindIntersectEntry helper function
func TestNewFindIntersectEntry(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		points   []pcommon.Point
		expected uint16
	}{
		{
			name:     "NtC with nil points",
			isNtC:    true,
			points:   nil,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN with nil points",
			isNtC:    false,
			points:   nil,
			expected: chainsync.ProtocolIdNtN,
		},
		{
			name:     "NtC with points",
			isNtC:    true,
			points:   []pcommon.Point{testPoint},
			expected: chainsync.ProtocolIdNtC,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cs.NewFindIntersectEntry(tc.isNtC, tc.points)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != false {
				t.Error("expected IsResponse to be false")
			}

			if result.MessageType != chainsync.MessageTypeFindIntersect {
				t.Errorf(
					"expected message type %d, got %d",
					chainsync.MessageTypeFindIntersect,
					result.MessageType,
				)
			}
		})
	}
}

// TestNewIntersectFoundEntry tests the NewIntersectFoundEntry helper function
func TestNewIntersectFoundEntry(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cs.NewIntersectFoundEntry(tc.isNtC, testPoint, testTip)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != true {
				t.Error("expected IsResponse to be true")
			}

			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
			}
		})
	}
}

// TestNewIntersectNotFoundEntry tests the NewIntersectNotFoundEntry helper function
func TestNewIntersectNotFoundEntry(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cs.NewIntersectNotFoundEntry(tc.isNtC, testTip)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != true {
				t.Error("expected IsResponse to be true")
			}

			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
			}
		})
	}
}

// TestNewRequestNextEntry tests the NewRequestNextEntry helper function
func TestNewRequestNextEntry(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cs.NewRequestNextEntry(tc.isNtC)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != false {
				t.Error("expected IsResponse to be false")
			}

			if result.MessageType != chainsync.MessageTypeRequestNext {
				t.Errorf(
					"expected message type %d, got %d",
					chainsync.MessageTypeRequestNext,
					result.MessageType,
				)
			}
		})
	}
}

// TestNewAwaitReplyEntry tests the NewAwaitReplyEntry helper function
func TestNewAwaitReplyEntry(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cs.NewAwaitReplyEntry(tc.isNtC)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != true {
				t.Error("expected IsResponse to be true")
			}

			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
			}
		})
	}
}

// TestNewRollForwardEntry tests the NewRollForwardEntry helper function
func TestNewRollForwardEntry(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := cs.NewRollForwardEntry(
				tc.isNtC,
				0,
				testBlockCbor,
				testTip,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != true {
				t.Error("expected IsResponse to be true")
			}
		})
	}
}

// TestNewRollBackwardEntry tests the NewRollBackwardEntry helper function
func TestNewRollBackwardEntry(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cs.NewRollBackwardEntry(tc.isNtC, testPoint, testTip)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != true {
				t.Error("expected IsResponse to be true")
			}

			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
			}
		})
	}
}

// TestNewDoneEntry tests the NewDoneEntry helper function
func TestNewDoneEntry(t *testing.T) {
	testCases := []struct {
		name     string
		isNtC    bool
		expected uint16
	}{
		{
			name:     "NtC",
			isNtC:    true,
			expected: chainsync.ProtocolIdNtC,
		},
		{
			name:     "NtN",
			isNtC:    false,
			expected: chainsync.ProtocolIdNtN,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cs.NewDoneEntry(tc.isNtC)

			if result.ProtocolId != tc.expected {
				t.Errorf(
					"expected protocol ID %d, got %d",
					tc.expected,
					result.ProtocolId,
				)
			}

			if result.IsResponse != false {
				t.Error("expected IsResponse to be false")
			}

			if result.MessageType != chainsync.MessageTypeDone {
				t.Errorf(
					"expected MessageType %d, got %d",
					chainsync.MessageTypeDone,
					result.MessageType,
				)
			}

			if result.MsgFromCborFunc == nil {
				t.Error("expected MsgFromCborFunc to be set")
			}
		})
	}
}

// TestFixtures tests that the pre-defined conversation fixtures are valid slices
func TestFixtures(t *testing.T) {
	t.Run("MockOriginPoint", func(t *testing.T) {
		if cs.MockOriginPoint.Slot != 0 {
			t.Errorf(
				"expected origin point slot 0, got %d",
				cs.MockOriginPoint.Slot,
			)
		}
	})

	t.Run("MockBlockHash", func(t *testing.T) {
		if len(cs.MockBlockHash) != 32 {
			t.Errorf(
				"expected block hash length 32, got %d",
				len(cs.MockBlockHash),
			)
		}
	})

	t.Run("MockBlockHash2", func(t *testing.T) {
		if len(cs.MockBlockHash2) != 32 {
			t.Errorf(
				"expected block hash length 32, got %d",
				len(cs.MockBlockHash2),
			)
		}
	})

	t.Run("MockBlockHash3", func(t *testing.T) {
		if len(cs.MockBlockHash3) != 32 {
			t.Errorf(
				"expected block hash length 32, got %d",
				len(cs.MockBlockHash3),
			)
		}
	})

	t.Run("MockTip", func(t *testing.T) {
		if cs.MockTip.BlockNumber != 100 {
			t.Errorf(
				"expected block number 100, got %d",
				cs.MockTip.BlockNumber,
			)
		}
		if cs.MockTip.Point.Slot != 1000 {
			t.Errorf("expected slot 1000, got %d", cs.MockTip.Point.Slot)
		}
	})

	t.Run("MockTip2", func(t *testing.T) {
		if cs.MockTip2.BlockNumber != 50 {
			t.Errorf(
				"expected block number 50, got %d",
				cs.MockTip2.BlockNumber,
			)
		}
		if cs.MockTip2.Point.Slot != 500 {
			t.Errorf("expected slot 500, got %d", cs.MockTip2.Point.Slot)
		}
	})

	t.Run("MockPoint1", func(t *testing.T) {
		if cs.MockPoint1.Slot != 100 {
			t.Errorf("expected slot 100, got %d", cs.MockPoint1.Slot)
		}
	})

	t.Run("MockPoint2", func(t *testing.T) {
		if cs.MockPoint2.Slot != 200 {
			t.Errorf("expected slot 200, got %d", cs.MockPoint2.Slot)
		}
	})

	t.Run("MockPoint3", func(t *testing.T) {
		if cs.MockPoint3.Slot != 300 {
			t.Errorf("expected slot 300, got %d", cs.MockPoint3.Slot)
		}
	})

	t.Run("MockBlockCbor", func(t *testing.T) {
		if len(cs.MockBlockCbor) == 0 {
			t.Error("expected non-empty block CBOR")
		}
	})
}

// TestConversationChainSyncFromOrigin tests that the fixture is a valid slice
func TestConversationChainSyncFromOrigin(t *testing.T) {
	if cs.ConversationChainSyncFromOrigin == nil {
		t.Fatal("ConversationChainSyncFromOrigin should not be nil")
	}

	if len(cs.ConversationChainSyncFromOrigin) == 0 {
		t.Error("ConversationChainSyncFromOrigin should not be empty")
	}

	// Expected structure:
	// - Handshake request
	// - Handshake response
	// - FindIntersect
	// - IntersectFound
	// - RequestNext
	// - RollForward
	// - RequestNext
	// - AwaitReply
	expectedLen := 8
	if len(cs.ConversationChainSyncFromOrigin) != expectedLen {
		t.Errorf(
			"expected %d entries, got %d",
			expectedLen,
			len(cs.ConversationChainSyncFromOrigin),
		)
	}

	// Verify each entry is not nil
	for i, entry := range cs.ConversationChainSyncFromOrigin {
		if entry == nil {
			t.Errorf("entry at index %d should not be nil", i)
		}
	}
}

// TestConversationChainSyncRollback tests that the rollback fixture is valid
func TestConversationChainSyncRollback(t *testing.T) {
	if cs.ConversationChainSyncRollback == nil {
		t.Fatal("ConversationChainSyncRollback should not be nil")
	}

	if len(cs.ConversationChainSyncRollback) == 0 {
		t.Error("ConversationChainSyncRollback should not be empty")
	}

	// Expected structure:
	// - Handshake request
	// - Handshake response
	// - FindIntersect
	// - IntersectFound
	// - RequestNext
	// - RollForward (block 1)
	// - RequestNext
	// - RollForward (block 2)
	// - RequestNext
	// - RollBackward
	// - RequestNext
	// - RollForward (new chain)
	expectedLen := 12
	if len(cs.ConversationChainSyncRollback) != expectedLen {
		t.Errorf(
			"expected %d entries, got %d",
			expectedLen,
			len(cs.ConversationChainSyncRollback),
		)
	}

	// Verify each entry is not nil
	for i, entry := range cs.ConversationChainSyncRollback {
		if entry == nil {
			t.Errorf("entry at index %d should not be nil", i)
		}
	}
}

// TestConversationChainSyncIntersectNotFound tests that the intersect not found fixture is valid
func TestConversationChainSyncIntersectNotFound(t *testing.T) {
	if cs.ConversationChainSyncIntersectNotFound == nil {
		t.Fatal("ConversationChainSyncIntersectNotFound should not be nil")
	}

	if len(cs.ConversationChainSyncIntersectNotFound) == 0 {
		t.Error("ConversationChainSyncIntersectNotFound should not be empty")
	}

	// Expected structure:
	// - Handshake request
	// - Handshake response
	// - FindIntersect
	// - IntersectNotFound
	expectedLen := 4
	if len(cs.ConversationChainSyncIntersectNotFound) != expectedLen {
		t.Errorf(
			"expected %d entries, got %d",
			expectedLen,
			len(cs.ConversationChainSyncIntersectNotFound),
		)
	}

	// Verify each entry is not nil
	for i, entry := range cs.ConversationChainSyncIntersectNotFound {
		if entry == nil {
			t.Errorf("entry at index %d should not be nil", i)
		}
	}
}

// TestProtocolIDConsistency tests that NtC and NtN protocol IDs are different
func TestProtocolIDConsistency(t *testing.T) {
	if chainsync.ProtocolIdNtC == chainsync.ProtocolIdNtN {
		t.Error("NtC and NtN protocol IDs should be different")
	}
}

// TestEntryFieldsDirectAccess tests direct field access on entry types
func TestEntryFieldsDirectAccess(t *testing.T) {
	t.Run("ConversationEntryFindIntersect", func(t *testing.T) {
		points := []pcommon.Point{testPoint}
		entry := cs.ConversationEntryFindIntersect{Points: points}
		if len(entry.Points) != 1 {
			t.Errorf("expected 1 point, got %d", len(entry.Points))
		}
		if entry.Points[0].Slot != testPoint.Slot {
			t.Errorf(
				"expected slot %d, got %d",
				testPoint.Slot,
				entry.Points[0].Slot,
			)
		}
	})

	t.Run("ConversationEntryIntersectFound", func(t *testing.T) {
		entry := cs.ConversationEntryIntersectFound{
			Point: testPoint,
			Tip:   testTip,
		}
		if entry.Point.Slot != testPoint.Slot {
			t.Errorf(
				"expected slot %d, got %d",
				testPoint.Slot,
				entry.Point.Slot,
			)
		}
		if entry.Tip.BlockNumber != testTip.BlockNumber {
			t.Errorf(
				"expected block number %d, got %d",
				testTip.BlockNumber,
				entry.Tip.BlockNumber,
			)
		}
	})

	t.Run("ConversationEntryIntersectNotFound", func(t *testing.T) {
		entry := cs.ConversationEntryIntersectNotFound{Tip: testTip}
		if entry.Tip.BlockNumber != testTip.BlockNumber {
			t.Errorf(
				"expected block number %d, got %d",
				testTip.BlockNumber,
				entry.Tip.BlockNumber,
			)
		}
	})

	t.Run("ConversationEntryRollForward", func(t *testing.T) {
		entry := cs.ConversationEntryRollForward{
			BlockType: 1,
			BlockCbor: testBlockCbor,
			Tip:       testTip,
		}
		if entry.BlockType != 1 {
			t.Errorf("expected block type 1, got %d", entry.BlockType)
		}
		if len(entry.BlockCbor) != len(testBlockCbor) {
			t.Errorf(
				"expected block cbor length %d, got %d",
				len(testBlockCbor),
				len(entry.BlockCbor),
			)
		}
		if entry.Tip.BlockNumber != testTip.BlockNumber {
			t.Errorf(
				"expected block number %d, got %d",
				testTip.BlockNumber,
				entry.Tip.BlockNumber,
			)
		}
	})

	t.Run("ConversationEntryRollForwardNtN", func(t *testing.T) {
		entry := cs.ConversationEntryRollForwardNtN{
			Era:       0, // Byron
			ByronType: 1, // Main block
			BlockCbor: testBlockCbor,
			Tip:       testTip,
		}
		if entry.Era != 0 {
			t.Errorf("expected era 0, got %d", entry.Era)
		}
		if entry.ByronType != 1 {
			t.Errorf("expected byron type 1, got %d", entry.ByronType)
		}
	})

	t.Run("ConversationEntryRollBackward", func(t *testing.T) {
		entry := cs.ConversationEntryRollBackward{
			Point: testPoint,
			Tip:   testTip,
		}
		if entry.Point.Slot != testPoint.Slot {
			t.Errorf(
				"expected slot %d, got %d",
				testPoint.Slot,
				entry.Point.Slot,
			)
		}
		if entry.Tip.BlockNumber != testTip.BlockNumber {
			t.Errorf(
				"expected block number %d, got %d",
				testTip.BlockNumber,
				entry.Tip.BlockNumber,
			)
		}
	})
}

// TestEmptyEntryTypes tests that empty struct entry types can be created
func TestEmptyEntryTypes(t *testing.T) {
	t.Run("ConversationEntryRequestNext", func(t *testing.T) {
		entry := cs.ConversationEntryRequestNext{}
		// Verify it can be converted to an entry
		result := entry.ToEntry(true)
		if result.ProtocolId == 0 {
			t.Error("expected non-zero protocol ID")
		}
	})

	t.Run("ConversationEntryAwaitReply", func(t *testing.T) {
		entry := cs.ConversationEntryAwaitReply{}
		result := entry.ToEntry(true)
		if result.ProtocolId == 0 {
			t.Error("expected non-zero protocol ID")
		}
	})

	t.Run("ConversationEntryDone", func(t *testing.T) {
		entry := cs.ConversationEntryDone{}
		result := entry.ToEntry(true)
		if result.ProtocolId == 0 {
			t.Error("expected non-zero protocol ID")
		}
	})
}
