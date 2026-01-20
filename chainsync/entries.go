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
	"fmt"

	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// getProtocolId returns the appropriate ChainSync protocol ID based on mode
func getProtocolId(isNtC bool) uint16 {
	if isNtC {
		return chainsync.ProtocolIdNtC
	}
	return chainsync.ProtocolIdNtN
}

// ConversationEntryFindIntersect expects a FindIntersect message from the client
type ConversationEntryFindIntersect struct {
	Points []pcommon.Point // Optional: expected points for validation (nil means accept any)
}

// ToEntry converts this ChainSync entry to a generic ConversationEntryInput
func (e ConversationEntryFindIntersect) ToEntry(
	isNtC bool,
) ouroboros_mock.ConversationEntryInput {
	entry := ouroboros_mock.ConversationEntryInput{
		ProtocolId:      getProtocolId(isNtC),
		IsResponse:      false,
		MessageType:     chainsync.MessageTypeFindIntersect,
		MsgFromCborFunc: NewMsgFromCborFunc(isNtC),
	}
	if e.Points != nil {
		entry.Message = chainsync.NewMsgFindIntersect(e.Points)
	}
	return entry
}

// ConversationEntryIntersectFound sends an IntersectFound response to the client
type ConversationEntryIntersectFound struct {
	Point pcommon.Point // The intersection point found
	Tip   pcommon.Tip   // Current chain tip
}

// ToEntry converts this ChainSync entry to a generic ConversationEntryOutput
func (e ConversationEntryIntersectFound) ToEntry(
	isNtC bool,
) ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: getProtocolId(isNtC),
		IsResponse: true,
		Messages: []protocol.Message{
			chainsync.NewMsgIntersectFound(e.Point, e.Tip),
		},
	}
}

// ConversationEntryIntersectNotFound sends an IntersectNotFound response to the client
type ConversationEntryIntersectNotFound struct {
	Tip pcommon.Tip // Current chain tip
}

// ToEntry converts this ChainSync entry to a generic ConversationEntryOutput
func (e ConversationEntryIntersectNotFound) ToEntry(
	isNtC bool,
) ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: getProtocolId(isNtC),
		IsResponse: true,
		Messages: []protocol.Message{
			chainsync.NewMsgIntersectNotFound(e.Tip),
		},
	}
}

// ConversationEntryRequestNext expects a RequestNext message from the client
type ConversationEntryRequestNext struct{}

// ToEntry converts this ChainSync entry to a generic ConversationEntryInput
func (e ConversationEntryRequestNext) ToEntry(
	isNtC bool,
) ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      getProtocolId(isNtC),
		IsResponse:      false,
		MessageType:     chainsync.MessageTypeRequestNext,
		MsgFromCborFunc: NewMsgFromCborFunc(isNtC),
	}
}

// ConversationEntryAwaitReply sends an AwaitReply message to the client (at chain tip)
type ConversationEntryAwaitReply struct{}

// ToEntry converts this ChainSync entry to a generic ConversationEntryOutput
func (e ConversationEntryAwaitReply) ToEntry(
	isNtC bool,
) ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: getProtocolId(isNtC),
		IsResponse: true,
		Messages: []protocol.Message{
			chainsync.NewMsgAwaitReply(),
		},
	}
}

// ConversationEntryRollForward sends a RollForward message with block/header data to the client
type ConversationEntryRollForward struct {
	BlockType uint        // Block type (era)
	BlockCbor []byte      // Raw block/header CBOR data
	Tip       pcommon.Tip // Current chain tip
}

// ToEntry converts this ChainSync entry to a generic ConversationEntryOutput.
// Returns an error if the RollForward message cannot be created.
func (e ConversationEntryRollForward) ToEntry(
	isNtC bool,
) (ouroboros_mock.ConversationEntryOutput, error) {
	var msg protocol.Message
	if isNtC {
		rollForwardMsg, err := chainsync.NewMsgRollForwardNtC(
			e.BlockType,
			e.BlockCbor,
			e.Tip,
		)
		if err != nil {
			return ouroboros_mock.ConversationEntryOutput{}, fmt.Errorf(
				"failed to create RollForward NtC message: %w "+
					"(BlockType=%d, BlockCbor len=%d, Tip=%+v)",
				err, e.BlockType, len(e.BlockCbor), e.Tip,
			)
		}
		msg = rollForwardMsg
	} else {
		// For NtN, BlockType represents the era, and we pass 0 for byronType
		// The byronType is only relevant for Byron era blocks
		rollForwardMsg, err := chainsync.NewMsgRollForwardNtN(
			e.BlockType, // era
			0,           // byronType (0 for non-Byron, set appropriately for Byron)
			e.BlockCbor,
			e.Tip,
		)
		if err != nil {
			return ouroboros_mock.ConversationEntryOutput{}, fmt.Errorf(
				"failed to create RollForward NtN message: %w "+
					"(Era=%d, BlockCbor len=%d, Tip=%+v)",
				err, e.BlockType, len(e.BlockCbor), e.Tip,
			)
		}
		msg = rollForwardMsg
	}

	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: getProtocolId(isNtC),
		IsResponse: true,
		Messages:   []protocol.Message{msg},
	}, nil
}

// ConversationEntryRollForwardNtN is a specialized RollForward for NtN with explicit Byron type
type ConversationEntryRollForwardNtN struct {
	Era       uint        // Era (0=Byron, 1=Shelley, etc.)
	ByronType uint        // Byron block type (EBB=0, Main=1), only used for Byron era
	BlockCbor []byte      // Raw header CBOR data
	Tip       pcommon.Tip // Current chain tip
}

// ToEntry converts this ChainSync entry to a generic ConversationEntryOutput.
// Returns an error if the RollForward message cannot be created.
func (e ConversationEntryRollForwardNtN) ToEntry() (ouroboros_mock.ConversationEntryOutput, error) {
	rollForwardMsg, err := chainsync.NewMsgRollForwardNtN(
		e.Era,
		e.ByronType,
		e.BlockCbor,
		e.Tip,
	)
	if err != nil {
		return ouroboros_mock.ConversationEntryOutput{}, fmt.Errorf(
			"failed to create RollForward NtN message: %w "+
				"(Era=%d, ByronType=%d, BlockCbor len=%d, Tip=%+v)",
			err, e.Era, e.ByronType, len(e.BlockCbor), e.Tip,
		)
	}
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: chainsync.ProtocolIdNtN,
		IsResponse: true,
		Messages:   []protocol.Message{rollForwardMsg},
	}, nil
}

// ConversationEntryRollBackward sends a RollBackward message to the client
type ConversationEntryRollBackward struct {
	Point pcommon.Point // The point to roll back to
	Tip   pcommon.Tip   // Current chain tip
}

// ToEntry converts this ChainSync entry to a generic ConversationEntryOutput
func (e ConversationEntryRollBackward) ToEntry(
	isNtC bool,
) ouroboros_mock.ConversationEntryOutput {
	return ouroboros_mock.ConversationEntryOutput{
		ProtocolId: getProtocolId(isNtC),
		IsResponse: true,
		Messages: []protocol.Message{
			chainsync.NewMsgRollBackward(e.Point, e.Tip),
		},
	}
}

// ConversationEntryDone expects a Done message from the client to terminate the protocol
type ConversationEntryDone struct{}

// ToEntry converts this ChainSync entry to a generic ConversationEntryInput
func (e ConversationEntryDone) ToEntry(
	isNtC bool,
) ouroboros_mock.ConversationEntryInput {
	return ouroboros_mock.ConversationEntryInput{
		ProtocolId:      getProtocolId(isNtC),
		IsResponse:      false,
		MessageType:     chainsync.MessageTypeDone,
		MsgFromCborFunc: NewMsgFromCborFunc(isNtC),
	}
}

// NewMsgFromCborFunc returns the appropriate message parser function based on protocol mode
func NewMsgFromCborFunc(isNtC bool) protocol.MessageFromCborFunc {
	if isNtC {
		return chainsync.NewMsgFromCborNtC
	}
	return chainsync.NewMsgFromCborNtN
}

// Helper functions for creating common conversation patterns

// NewFindIntersectEntry creates a ConversationEntryInput for expecting a FindIntersect message
func NewFindIntersectEntry(
	isNtC bool,
	points []pcommon.Point,
) ouroboros_mock.ConversationEntryInput {
	return ConversationEntryFindIntersect{Points: points}.ToEntry(isNtC)
}

// NewIntersectFoundEntry creates a ConversationEntryOutput for sending an IntersectFound response
func NewIntersectFoundEntry(
	isNtC bool,
	point pcommon.Point,
	tip pcommon.Tip,
) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryIntersectFound{
		Point: point,
		Tip:   tip,
	}.ToEntry(
		isNtC,
	)
}

// NewIntersectNotFoundEntry creates a ConversationEntryOutput for sending an IntersectNotFound response
func NewIntersectNotFoundEntry(
	isNtC bool,
	tip pcommon.Tip,
) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryIntersectNotFound{Tip: tip}.ToEntry(isNtC)
}

// NewRequestNextEntry creates a ConversationEntryInput for expecting a RequestNext message
func NewRequestNextEntry(isNtC bool) ouroboros_mock.ConversationEntryInput {
	return ConversationEntryRequestNext{}.ToEntry(isNtC)
}

// NewAwaitReplyEntry creates a ConversationEntryOutput for sending an AwaitReply message
func NewAwaitReplyEntry(isNtC bool) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryAwaitReply{}.ToEntry(isNtC)
}

// NewRollForwardEntry creates a ConversationEntryOutput for sending a RollForward message.
// Returns an error if the RollForward message cannot be created.
func NewRollForwardEntry(
	isNtC bool,
	blockType uint,
	blockCbor []byte,
	tip pcommon.Tip,
) (ouroboros_mock.ConversationEntryOutput, error) {
	return ConversationEntryRollForward{
		BlockType: blockType,
		BlockCbor: blockCbor,
		Tip:       tip,
	}.ToEntry(isNtC)
}

// NewRollBackwardEntry creates a ConversationEntryOutput for sending a RollBackward message
func NewRollBackwardEntry(
	isNtC bool,
	point pcommon.Point,
	tip pcommon.Tip,
) ouroboros_mock.ConversationEntryOutput {
	return ConversationEntryRollBackward{Point: point, Tip: tip}.ToEntry(isNtC)
}

// MustRollForwardEntry creates a ConversationEntryOutput for sending a RollForward message.
// Panics if the RollForward message cannot be created. Use this for fixtures and test setup
// where errors indicate programmer mistakes. For production code, use NewRollForwardEntry.
func MustRollForwardEntry(
	isNtC bool,
	blockType uint,
	blockCbor []byte,
	tip pcommon.Tip,
) ouroboros_mock.ConversationEntryOutput {
	entry, err := NewRollForwardEntry(isNtC, blockType, blockCbor, tip)
	if err != nil {
		panic(err)
	}
	return entry
}

// NewDoneEntry creates a ConversationEntryInput for expecting a Done message
func NewDoneEntry(isNtC bool) ouroboros_mock.ConversationEntryInput {
	return ConversationEntryDone{}.ToEntry(isNtC)
}
