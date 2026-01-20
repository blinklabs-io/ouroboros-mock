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
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// ConversationEntryRequestRange expects a RequestRange message from the client
type ConversationEntryRequestRange struct {
	Start pcommon.Point // Optional: expected start point for validation (empty hash means accept any)
	End   pcommon.Point // Optional: expected end point for validation (empty hash means accept any)
}

// ToEntry converts this BlockFetch entry to a generic ConversationEntryInput
func (e ConversationEntryRequestRange) ToEntry() ouroboros_mock.ConversationEntryInput {
	entry := ouroboros_mock.ConversationEntryInput{
		ProtocolId:      blockfetch.ProtocolId,
		IsResponse:      false,
		MessageType:     blockfetch.MessageTypeRequestRange,
		MsgFromCborFunc: blockfetch.NewMsgFromCbor,
	}
	// If Start and End are provided, set the expected message for validation
	// Check hash instead of slot since slot 0 is valid (genesis block)
	if len(e.Start.Hash) > 0 || len(e.End.Hash) > 0 {
		entry.Message = blockfetch.NewMsgRequestRange(e.Start, e.End)
	}
	return entry
}

// ConversationEntryStartBatch sends a StartBatch message to the client
type ConversationEntryStartBatch struct{}

// ToEntry converts this BlockFetch entry to a generic ConversationEntryOutput
func (e ConversationEntryStartBatch) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: blockfetch.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			blockfetch.NewMsgStartBatch(),
		},
	}
}

// ConversationEntryBlock sends a Block message to the client
type ConversationEntryBlock struct {
	BlockType uint   // Block type (era) - for documentation/metadata only, not used in wire protocol
	BlockCbor []byte // Raw block CBOR data (wrapped block format)
}

// ToEntry converts this BlockFetch entry to a generic ConversationEntryOutput
func (e ConversationEntryBlock) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: blockfetch.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			blockfetch.NewMsgBlock(e.BlockCbor),
		},
	}
}

// ConversationEntryBatchDone sends a BatchDone message to the client
type ConversationEntryBatchDone struct{}

// ToEntry converts this BlockFetch entry to a generic ConversationEntryOutput
func (e ConversationEntryBatchDone) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: blockfetch.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			blockfetch.NewMsgBatchDone(),
		},
	}
}

// ConversationEntryNoBlocks sends a NoBlocks message when blocks are unavailable
type ConversationEntryNoBlocks struct{}

// ToEntry converts this BlockFetch entry to a generic ConversationEntryOutput
func (e ConversationEntryNoBlocks) ToEntry() ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: blockfetch.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			blockfetch.NewMsgNoBlocks(),
		},
	}
}

// ConversationEntryClientDone expects a ClientDone message from the client
type ConversationEntryClientDone struct{}

// ToEntry converts this BlockFetch entry to a generic ConversationEntryInput
func (e ConversationEntryClientDone) ToEntry() ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      blockfetch.ProtocolId,
		IsResponse:      false,
		MessageType:     blockfetch.MessageTypeClientDone,
		MsgFromCborFunc: blockfetch.NewMsgFromCbor,
	}
}

// Helper functions for creating common conversation patterns

// NewRequestRangeEntry creates a ConversationEntryInput for expecting a RequestRange message
func NewRequestRangeEntry(
	start, end pcommon.Point,
) ouroboros_mock.ConversationEntryInput {
	return ConversationEntryRequestRange{
		Start: start,
		End:   end,
	}.ToEntry()
}

// NewRequestRangeEntryAny creates a ConversationEntryInput for expecting any RequestRange message
func NewRequestRangeEntryAny() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryRequestRange{}.ToEntry()
}

// NewStartBatchEntry creates a ConversationEntryOutput for sending a StartBatch message
func NewStartBatchEntry() ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryStartBatch{}.ToEntry()
}

// NewBlockEntry creates a ConversationEntryOutput for sending a Block message
func NewBlockEntry(
	blockType uint,
	blockCbor []byte,
) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryBlock{
		BlockType: blockType,
		BlockCbor: blockCbor,
	}.ToEntry()
}

// NewBatchDoneEntry creates a ConversationEntryOutput for sending a BatchDone message
func NewBatchDoneEntry() ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryBatchDone{}.ToEntry()
}

// NewNoBlocksEntry creates a ConversationEntryOutput for sending a NoBlocks message
func NewNoBlocksEntry() ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryNoBlocks{}.ToEntry()
}

// NewClientDoneEntry creates a ConversationEntryInput for expecting a ClientDone message
func NewClientDoneEntry() ouroboros_mock.ConversationEntryInput {
	return ConversationEntryClientDone{}.ToEntry()
}
