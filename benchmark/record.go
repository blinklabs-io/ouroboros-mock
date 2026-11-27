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

package benchmark

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/blinklabs-io/ouroboros-mock/peer"
)

// Point is a full block identity; an empty hash and slot zero denote origin.
type Point struct {
	Slot uint64 `json:"slot"`
	Hash string `json:"hash"`
}

// Record is one versioned process, action or source observation.
type Record struct {
	ManifestSHA256 string                 `json:"manifest_sha256,omitempty"`
	Build          *Identity              `json:"build,omitempty"`
	Connection     string                 `json:"connection,omitempty"`
	Version        int                    `json:"version"`
	Run            string                 `json:"run"`
	Experiment     string                 `json:"experiment"`
	Kind           string                 `json:"kind"`
	At             string                 `json:"at"`
	Role           string                 `json:"role,omitempty"`
	Endpoint       string                 `json:"endpoint,omitempty"`
	Action         string                 `json:"action,omitempty"`
	ScheduledNS    *int64                 `json:"scheduled_ns,omitempty"`
	ElapsedNS      *int64                 `json:"elapsed_ns,omitempty"`
	Sequence       uint64                 `json:"sequence,omitempty"`
	Session        uint64                 `json:"session,omitempty"`
	Request        uint64                 `json:"request,omitempty"`
	Points         []Point                `json:"points,omitempty"`
	Parent         string                 `json:"parent,omitempty"`
	Number         *uint64                `json:"number,omitempty"`
	Size           uint64                 `json:"size,omitempty"`
	Count          uint64                 `json:"count,omitempty"`
	Transactions   []string               `json:"transactions,omitempty"`
	Error          string                 `json:"error,omitempty"`
	Stats          *peer.ObservationStats `json:"stats,omitempty"`
	Complete       *bool                  `json:"recording_complete,omitempty"`
}

var eventNames = map[peer.EventKind]string{
	peer.EventTxsOffered:     "txs_offered",
	peer.EventTxsReturned:    "tx_bodies_returned",
	peer.EventSessionStarted: "session_started", peer.EventSessionClosed: "session_closed",
	peer.EventFindIntersect: "find_intersect", peer.EventIntersectFound: "intersect_found", peer.EventIntersectNotFound: "intersect_not_found",
	peer.EventRequestNext: "request_next", peer.EventAwaitReply: "await_reply", peer.EventRollForward: "roll_forward", peer.EventRollBackward: "roll_backward",
	peer.EventRequestRange: "request_range", peer.EventNoBlocks: "no_blocks", peer.EventBatchDone: "batch_done", peer.EventAwaitedReply: "awaited_reply",
	peer.EventBlockFetched: "block_fetched", peer.EventTxIdsOffered: "tx_ids_offered", peer.EventTxsReceived: "txs_received", peer.EventTxIdsRequested: "tx_ids_requested",
	peer.EventTxIdsBlocked: "tx_ids_blocked", peer.EventTxsRequested: "txs_requested", peer.EventBlockSent: "block_send_call", peer.EventFetchRequested: "fetch_api_start", peer.EventFetchCompleted: "fetch_completed",
}

func stamp(at time.Time) string { return at.UTC().Format("2006-01-02T15:04:05.000000000Z") }

func eventRecord(e peer.Event) (Record, error) {
	kind, ok := eventNames[e.Kind]
	if !ok || e.OccurredAt.IsZero() || e.Sequence == 0 || (len(e.Parent) != 0 && len(e.Parent) != 32) {
		return Record{}, fmt.Errorf("invalid source event kind/time/sequence: %d", e.Kind)
	}
	r := Record{Version: SchemaVersion, Kind: kind, At: stamp(e.OccurredAt), Sequence: e.Sequence, Session: e.Session, Request: e.Request, Connection: e.Connection, Parent: hex.EncodeToString(e.Parent), Size: e.Size, Count: e.Count}
	if e.Kind == peer.EventRollForward || e.Kind == peer.EventBlockFetched || e.Kind == peer.EventBlockSent {
		number := e.BlockNumber
		r.Number = &number
	}
	for _, p := range e.Points {
		if len(p.Hash) != 32 && (len(p.Hash) != 0 || p.Slot != 0) {
			return Record{}, fmt.Errorf("invalid source point at slot %d", p.Slot)
		}
		r.Points = append(r.Points, Point{Slot: p.Slot, Hash: hex.EncodeToString(p.Hash)})
	}
	for _, id := range e.TxIds {
		r.Transactions = append(r.Transactions, hex.EncodeToString(id[:]))
	}
	for _, tx := range e.Txs {
		r.Transactions = append(r.Transactions, hex.EncodeToString(tx.ID[:]))
	}
	if e.Err != nil {
		r.Error = e.Err.Error()
	}
	return r, nil
}
