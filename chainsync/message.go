// Copyright 2026 Blink Labs Software
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
	"github.com/blinklabs-io/gouroboros/protocol"
	gchainsync "github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
)

// ServerMessage is a single outbound chain-sync message observed coming from
// the server under test. It wraps the decoded gouroboros protocol message and
// provides typed accessors for the values a test typically asserts on.
type ServerMessage struct {
	msg protocol.Message
}

// Type returns the chain-sync message type. Compare against the
// gouroboros chainsync.MessageType* constants (e.g.
// [github.com/blinklabs-io/gouroboros/protocol/chainsync.MessageTypeRollForward]).
func (m ServerMessage) Type() uint {
	if m.msg == nil {
		return 0
	}
	return uint(m.msg.Type())
}

// Message returns the underlying decoded gouroboros protocol message. Callers
// needing era-specific or wrapper detail can type-assert it to the concrete
// chain-sync message type (e.g. *chainsync.MsgRollForwardNtC).
func (m ServerMessage) Message() protocol.Message {
	return m.msg
}

// IsRollForward reports whether the message is a RollForward (NtN or NtC).
func (m ServerMessage) IsRollForward() bool {
	return m.Type() == gchainsync.MessageTypeRollForward
}

// IsRollBackward reports whether the message is a RollBackward.
func (m ServerMessage) IsRollBackward() bool {
	return m.Type() == gchainsync.MessageTypeRollBackward
}

// IsAwaitReply reports whether the message is an AwaitReply.
func (m ServerMessage) IsAwaitReply() bool {
	return m.Type() == gchainsync.MessageTypeAwaitReply
}

// IsIntersectFound reports whether the message is an IntersectFound.
func (m ServerMessage) IsIntersectFound() bool {
	return m.Type() == gchainsync.MessageTypeIntersectFound
}

// IsIntersectNotFound reports whether the message is an IntersectNotFound.
func (m ServerMessage) IsIntersectNotFound() bool {
	return m.Type() == gchainsync.MessageTypeIntersectNotFound
}

// Tip returns the chain tip carried by the message and true, for the message
// types that carry one (RollForward, RollBackward, IntersectFound,
// IntersectNotFound). It returns a zero tip and false otherwise (e.g.
// AwaitReply).
func (m ServerMessage) Tip() (gchainsync.Tip, bool) {
	switch msg := m.msg.(type) {
	case *gchainsync.MsgRollForwardNtN:
		return msg.Tip, true
	case *gchainsync.MsgRollForwardNtC:
		return msg.Tip, true
	case *gchainsync.MsgRollBackward:
		return msg.Tip, true
	case *gchainsync.MsgIntersectFound:
		return msg.Tip, true
	case *gchainsync.MsgIntersectNotFound:
		return msg.Tip, true
	default:
		return gchainsync.Tip{}, false
	}
}

// Point returns the point carried by the message and true, for the message
// types that carry one (RollBackward, IntersectFound). It returns a zero point
// and false otherwise.
func (m ServerMessage) Point() (pcommon.Point, bool) {
	switch msg := m.msg.(type) {
	case *gchainsync.MsgRollBackward:
		return msg.Point, true
	case *gchainsync.MsgIntersectFound:
		return msg.Point, true
	default:
		return pcommon.Point{}, false
	}
}

// RollForwardNtC returns the block type, block CBOR and tip of an NtC
// RollForward message. The final return value is false if the message is not an
// NtC RollForward.
func (m ServerMessage) RollForwardNtC() (uint, []byte, gchainsync.Tip, bool) {
	msg, ok := m.msg.(*gchainsync.MsgRollForwardNtC)
	if !ok {
		return 0, nil, gchainsync.Tip{}, false
	}
	return msg.BlockType(), msg.BlockCbor(), msg.Tip, true
}

// RollForwardNtN returns the wrapped header and tip of an NtN RollForward
// message. The final return value is false if the message is not an NtN
// RollForward.
func (m ServerMessage) RollForwardNtN() (gchainsync.WrappedHeader, gchainsync.Tip, bool) {
	msg, ok := m.msg.(*gchainsync.MsgRollForwardNtN)
	if !ok {
		return gchainsync.WrappedHeader{}, gchainsync.Tip{}, false
	}
	return msg.WrappedHeader, msg.Tip, true
}
