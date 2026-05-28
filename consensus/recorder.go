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

package consensus

import (
	"sync"
	"time"

	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/ouroboros-mock/consensus/format"
)

// Recorder collects the inbound protocol messages cardano-node served
// during a capture. It is safe for concurrent use by gouroboros's
// per-protocol goroutines.
type Recorder struct {
	mu     sync.Mutex
	cond   *sync.Cond
	peerID uint64
	served []format.ServedMessage
}

// NewRecorder builds a Recorder. peerID is stamped on the resulting
// PeerInput; single-peer captures use 0.
func NewRecorder(peerID uint64) *Recorder {
	r := &Recorder{peerID: peerID}
	r.cond = sync.NewCond(&r.mu)
	return r
}

// Snapshot returns a deep copy of the served list as it stands.
// Nested slices and pointer fields are cloned so callers can mutate
// the result without bleeding back into recorder state.
func (r *Recorder) Snapshot() []format.ServedMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]format.ServedMessage, len(r.served))
	for i, m := range r.served {
		out[i] = cloneServedMessage(m)
	}
	return out
}

// PeerInput returns a PeerInput populated with the recorder's current
// state.
func (r *Recorder) PeerInput() format.PeerInput {
	return format.PeerInput{
		PeerID: r.peerID,
		Served: r.Snapshot(),
	}
}

// PeerID returns the recorder's peer id.
func (r *Recorder) PeerID() uint64 { return r.peerID }

// Count returns the number of recorded messages so far.
func (r *Recorder) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.served)
}

// WaitForNextOrDeadline blocks until at least one more ServedMessage
// is recorded past `since`, or the deadline elapses. Returns true if
// a new message arrived. A false return without an error means the
// deadline hit first — the script handler treats that as an implicit
// AwaitReply (server has nothing more to send right now).
func (r *Recorder) WaitForNextOrDeadline(
	since int,
	timeout time.Duration,
) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Register the timer AFTER taking the lock so its broadcast
	// can't fire before this goroutine has entered cond.Wait()
	// (which atomically drops the lock). A naïve `go func(){…}()`
	// pattern races: the goroutine could grab the lock and
	// broadcast first, and that broadcast would be lost — Go's
	// sync.Cond is edge-triggered.
	deadline := time.Now().Add(timeout)
	timer := time.AfterFunc(timeout, func() {
		r.mu.Lock()
		r.cond.Broadcast()
		r.mu.Unlock()
	})
	defer timer.Stop()
	for len(r.served) <= since && time.Now().Before(deadline) {
		r.cond.Wait()
	}
	return len(r.served) > since
}

// Record appends a pre-built ServedMessage. The live capture path
// goes through the OnRollForwardRaw / OnRollBackward callbacks;
// Record is the entry point for synthetic captures (tests) and for
// merging per-upstream captures into a multi-peer vector.
func (r *Recorder) Record(msg format.ServedMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.served = append(r.served, cloneServedMessage(msg))
	r.cond.Broadcast()
}

// OnRollForwardRaw is the gouroboros RollForwardRawFunc callback. era
// is the block-header era (BlockHeaderType*). The structured
// roll_forward fields go directly into the served message — no CBOR
// rebuild needed.
func (r *Recorder) OnRollForwardRaw(
	_ chainsync.CallbackContext,
	era uint,
	headerCbor []byte,
	tip chainsync.Tip,
) error {
	e := era
	r.Record(format.ServedMessage{
		Protocol:   format.ProtocolChainSync,
		MsgType:    format.ChainSyncMsgRollForward,
		Era:        &e,
		HeaderCbor: format.HexBytes(headerCbor),
		Tip:        formatTip(tip),
	})
	return nil
}

// OnRollBackward is the gouroboros RollBackwardFunc callback.
func (r *Recorder) OnRollBackward(
	_ chainsync.CallbackContext,
	point pcommon.Point,
	tip chainsync.Tip,
) error {
	r.Record(format.ServedMessage{
		Protocol: format.ProtocolChainSync,
		MsgType:  format.ChainSyncMsgRollBackward,
		Point: &format.Point{
			Slot: point.Slot,
			Hash: format.HexBytes(point.Hash),
		},
		Tip: formatTip(tip),
	})
	return nil
}

// formatTip projects a gouroboros chainsync.Tip into the format's
// Tip type, preserving all three fields (slot, hash, block number).
// BlockNumber matters at replay time: Praos chain selection
// compares chains by block count, not slot, so dropping it would
// silently route a future intersect-from-mid-chain scenario to the
// wrong peer.
func formatTip(tip chainsync.Tip) *format.Tip {
	return &format.Tip{
		Slot:        tip.Point.Slot,
		Hash:        format.HexBytes(tip.Point.Hash),
		BlockNumber: tip.BlockNumber,
	}
}

// cloneServedMessage deep-copies the byte / slice / pointer payload
// fields so callers can reuse their buffers after Record returns.
func cloneServedMessage(m format.ServedMessage) format.ServedMessage {
	out := format.ServedMessage{
		Protocol:   m.Protocol,
		MsgType:    m.MsgType,
		HeaderCbor: append(format.HexBytes(nil), m.HeaderCbor...),
		BlockCbor:  append(format.HexBytes(nil), m.BlockCbor...),
	}
	if m.Era != nil {
		e := *m.Era
		out.Era = &e
	}
	if m.Tip != nil {
		t := *m.Tip
		t.Hash = append(format.HexBytes(nil), m.Tip.Hash...)
		out.Tip = &t
	}
	if m.Point != nil {
		p := *m.Point
		p.Hash = append(format.HexBytes(nil), m.Point.Hash...)
		out.Point = &p
	}
	if m.Start != nil {
		s := *m.Start
		s.Hash = append(format.HexBytes(nil), m.Start.Hash...)
		out.Start = &s
	}
	if m.End != nil {
		e := *m.End
		e.Hash = append(format.HexBytes(nil), m.End.Hash...)
		out.End = &e
	}
	if len(m.Points) > 0 {
		out.Points = make([]format.Point, len(m.Points))
		for i, p := range m.Points {
			out.Points[i] = format.Point{
				Slot: p.Slot,
				Hash: append(format.HexBytes(nil), p.Hash...),
			}
		}
	}
	return out
}
