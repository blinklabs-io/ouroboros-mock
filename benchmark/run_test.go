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
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/ouroboros-mock/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

type recordWriter struct {
	records  chan Record
	failKind string
	failure  error
}

func (w *recordWriter) Write(data []byte) (int, error) {
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return 0, err
	}
	if r.Kind == w.failKind {
		return 0, w.failure
	}
	w.records <- r
	return len(data), nil
}

type runHandle struct {
	input   *io.PipeWriter
	records chan Record
	done    chan error
	cancel  context.CancelFunc
}

func startTestRun(t *testing.T, requirements []Requirement, failKind string) *runHandle {
	t.Helper()
	path, m := mutableCorpus(t)
	m.DurationNS = int64(250 * time.Millisecond)
	m.Actions = nil
	m.Roles[0].Required = requirements
	m.Roles[0].Initial = m.Roles[0].Initial[:3]
	saveManifest(t, path, m)
	b, err := Load(path)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	input, control := io.Pipe()
	r := &runHandle{input: control, records: make(chan Record, 1024), done: make(chan error, 1), cancel: cancel}
	out := &recordWriter{records: r.records, failKind: failKind, failure: errors.New("record writer failed")}
	go func() {
		r.done <- Serve(ctx, b, Options{Run: "trial", Endpoints: map[string]string{"upstream": "127.0.0.1:0"}}, input, out)
		close(r.records)
	}()
	t.Cleanup(func() { cancel(); _ = control.Close() })
	return r
}

func nextRecord(t *testing.T, r *runHandle, kind string) Record {
	t.Helper()
	for {
		select {
		case row, ok := <-r.records:
			require.True(t, ok, "missing %s", kind)
			if row.Kind == kind {
				return row
			}
		case <-time.After(5 * time.Second):
			t.Fatal("record deadline")
		}
	}
}

func startTrial(t *testing.T, r *runHandle) {
	t.Helper()
	require.NoError(t, json.NewEncoder(r.input).Encode(Control{Version: 1, Kind: "start", Run: "trial", Start: time.Now()}))
}

func TestServeCompletesOnlyAfterDurableSourceTail(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	r := startTestRun(t, []Requirement{{Event: "block_send_call", Items: 3}}, "")
	ready := nextRecord(t, r, "ready")
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", ready.Endpoint)
	require.NoError(t, err)
	f, err := peer.NewFollower(peer.FollowerConfig{Conn: conn, NetworkMagic: 42, FetchBlocks: true})
	require.NoError(t, err)
	defer f.Close()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range f.Events() {
		}
	}()
	startTrial(t, r)
	require.NoError(t, <-r.done)
	var lastSource uint64
	var finalized bool
	var delivered uint64
	var complete bool
	for record := range r.records {
		if record.Sequence > 0 {
			require.False(t, finalized)
			lastSource = record.Sequence
		}
		if record.Kind == "source_final" {
			finalized = true
			require.NotNil(t, record.Stats)
			require.True(t, record.Stats.Complete())
			delivered = record.Stats.Delivered
		}
		if record.Kind == "final" {
			require.NotNil(t, record.Complete)
			complete = *record.Complete
		}
	}
	require.NotZero(t, lastSource)
	require.Equal(t, lastSource, delivered)
	require.True(t, complete)
	require.NoError(t, f.Close())
	wg.Wait()
}

func TestServeRejectsMissingActivityEOFStopCancellationAndWriterFailure(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	for _, mode := range []string{"missing population", "eof", "stop", "cancel", "writer"} {
		t.Run(mode, func(t *testing.T) {
			requirements := []Requirement{}
			fail := ""
			if mode == "missing population" {
				requirements = []Requirement{{Event: "block_send_call", Items: 1}}
			}
			if mode == "writer" {
				fail = "started"
			}
			r := startTestRun(t, requirements, fail)
			nextRecord(t, r, "ready")
			startTrial(t, r)
			switch mode {
			case "eof":
				require.NoError(t, r.input.Close())
			case "stop":
				require.NoError(t, json.NewEncoder(r.input).Encode(Control{Version: 1, Kind: "stop", Run: "trial"}))
			case "cancel":
				r.cancel()
			}
			err := <-r.done
			require.Error(t, err)
			var complete *bool
			for record := range r.records {
				if record.Kind == "final" {
					complete = record.Complete
				}
			}
			if mode != "writer" {
				require.NotNil(t, complete)
				require.False(t, *complete)
			} else {
				require.ErrorContains(t, err, "record writer failed")
			}
		})
	}
}

func TestSourceRecordsPreserveFullIdentitiesAndOccurrenceOrdering(t *testing.T) {
	at := time.Date(2026, 1, 1, 0, 0, 0, 1, time.UTC)
	hash := common.Blake2b256{1}
	e := peer.Event{Kind: peer.EventBlockFetched, OccurredAt: at, Sequence: 12, Session: 2, Request: 7, Connection: "local remote", Points: []pcommon.Point{pcommon.NewPoint(99, hash.Bytes())}, BlockNumber: 4, Parent: hash.Bytes(), Count: 1, Size: 123}
	r, err := eventRecord(e)
	require.NoError(t, err)
	require.Equal(t, "2026-01-01T00:00:00.000000001Z", r.At)
	require.Len(t, r.Points[0].Hash, 64)
	require.Len(t, r.Parent, 64)
	require.Equal(t, uint64(7), r.Request)
	require.NotNil(t, r.Number)
	require.Equal(t, uint64(4), *r.Number)
	e.Points[0].Hash = []byte{1}
	_, err = eventRecord(e)
	require.Error(t, err)
}

func TestClientRolesRunOverIndependentRealConnections(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	for _, kind := range []string{"follower", "fetcher", "txpeer"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.CopyFS(root, os.DirFS("testdata/reference")))
			path := filepath.Join(root, kind+".json")
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			var m Manifest
			require.NoError(t, json.Unmarshal(raw, &m))
			m.DurationNS = int64(time.Second)
			saveManifest(t, path, m)
			b, err := Load(path)
			require.NoError(t, err)
			blockIDs := []string{"b1-000", "b1-001", "b1-002"}
			if kind == "fetcher" {
				blockIDs = []string{"b1-021", "b1-022", "b1-023"}
			}
			blocks, err := b.Blocks(blockIDs)
			require.NoError(t, err)
			up, err := peer.NewUpstream(peer.UpstreamConfig{Blocks: blocks, NetworkMagic: 42})
			require.NoError(t, err)
			defer up.Close()
			l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
			require.NoError(t, err)
			require.NoError(t, up.AcceptContext(t.Context(), l))
			var remote []peer.Event
			remoteDone := make(chan struct{})
			go func() {
				for e := range up.Events() {
					remote = append(remote, e)
				}
				close(remoteDone)
			}()
			input, control := io.Pipe()
			defer control.Close()
			out := &recordWriter{records: make(chan Record, 1024)}
			done := make(chan error, 1)
			go func() {
				done <- Serve(t.Context(), b, Options{Run: "client-trial", Endpoints: map[string]string{kind: l.Addr().String()}}, input, out)
			}()
			for {
				r := <-out.records
				if r.Kind == "ready" {
					break
				}
			}
			require.NoError(t, json.NewEncoder(control).Encode(Control{Version: 1, Kind: "start", Run: "client-trial", Start: time.Now()}))
			require.NoError(t, <-done)
			stats, err := up.Finish(t.Context())
			require.NoError(t, err)
			require.True(t, stats.Complete())
			<-remoteDone
			var headerRequests, bodyRequests, transactions int
			for _, e := range remote {
				switch e.Kind {
				case peer.EventRequestNext:
					headerRequests++
				case peer.EventRequestRange:
					bodyRequests++
				case peer.EventTxsReceived:
					transactions += len(e.Txs)
				}
			}
			switch kind {
			case "follower":
				require.Positive(t, headerRequests)
				require.Zero(t, bodyRequests)
			case "fetcher":
				require.Zero(t, headerRequests)
				require.Equal(t, 1, bodyRequests)
			case "txpeer":
				require.Equal(t, 1, transactions)
			}
		})
	}
}

func TestRawPeerNormalizationMatchesIndependentExpectedRecords(t *testing.T) {
	b, err := Load("testdata/reference/upstream.json")
	require.NoError(t, err)
	raw, err := b.Read("raw-events")
	require.NoError(t, err)
	var input struct {
		Version int          `json:"version"`
		Events  []peer.Event `json:"events"`
	}
	require.NoError(t, decodeJSON(raw, &input))
	raw, err = b.Read("normalized-expected")
	require.NoError(t, err)
	var expected struct {
		Version int `json:"version"`
		Cases   map[string]struct {
			Records       []Record `json:"records"`
			Serialization []uint64 `json:"serialization_order"`
			Occurrence    []uint64 `json:"occurrence_order"`
			Unique        int      `json:"unique_blocks"`
			Peers         int      `json:"eligible_peers"`
		} `json:"cases"`
	}
	require.NoError(t, decodeJSON(raw, &expected))
	want := expected.Cases["peer-normalization"]
	require.Len(t, input.Events, len(want.Records))
	for i, e := range input.Events {
		r, err := eventRecord(e)
		require.NoError(t, err)
		require.Equal(t, want.Records[i], r)
	}
	require.Greater(t, input.Events[0].OccurredAt.UnixNano(), input.Events[1].OccurredAt.UnixNano())
	require.Equal(t, []uint64{2, 1}, want.Occurrence)
}
