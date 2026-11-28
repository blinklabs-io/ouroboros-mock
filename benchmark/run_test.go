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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/ouroboros-mock/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

type recordWriter struct {
	records             chan Record
	closed              chan struct{}
	closeOnce           sync.Once
	failKind            string
	failure             error
	closeFailure        error
	cancelAtSourceFinal context.CancelFunc
}

func (w *recordWriter) Write(data []byte) (int, error) {
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return 0, err
	}
	if r.Kind == w.failKind {
		return 0, w.failure
	}
	if r.Kind == "source_final" && w.cancelAtSourceFinal != nil {
		w.cancelAtSourceFinal()
	}
	select {
	case w.records <- r:
	case <-w.closed:
		return 0, io.ErrClosedPipe
	}
	return len(data), nil
}

func (w *recordWriter) Close() error {
	w.closeOnce.Do(func() { close(w.closed) })
	return w.closeFailure
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
	out := &recordWriter{closed: make(chan struct{}), records: r.records, failKind: failKind, failure: errors.New("record writer failed")}
	if failKind == "close" {
		out.closeFailure = errors.New("output close failed")
	}
	if failKind == "final cancellation" {
		out.failKind = ""
		out.cancelAtSourceFinal = cancel
	}
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
	for _, mode := range []string{"missing population", "eof", "stop", "cancel", "writer", "final cancellation"} {
		t.Run(mode, func(t *testing.T) {
			requirements := []Requirement{}
			fail := ""
			if mode == "missing population" {
				requirements = []Requirement{{Event: "block_send_call", Items: 1}}
			}
			if mode == "writer" {
				fail = "started"
			}
			if mode == "final cancellation" {
				fail = mode
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
	for _, scenario := range []string{"follower", "fetcher", "txpeer", "fetcher-no-blocks"} {
		t.Run(scenario, func(t *testing.T) {
			kind := strings.TrimSuffix(scenario, "-no-blocks")
			root := t.TempDir()
			require.NoError(t, os.CopyFS(root, os.DirFS("testdata/reference")))
			path := filepath.Join(root, kind+".json")
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			var m Manifest
			require.NoError(t, json.Unmarshal(raw, &m))
			m.DurationNS = int64(time.Second)
			if scenario == "fetcher-no-blocks" {
				m.Roles[0].Required = []Requirement{{Event: "fetch_completed", Events: 1}}
			}
			saveManifest(t, path, m)
			b, err := Load(path)
			require.NoError(t, err)
			blockIDs := []string{"b1-000", "b1-001", "b1-002"}
			if kind == "fetcher" {
				blockIDs = []string{"b1-021", "b1-022", "b1-023"}
			}
			blocks, err := b.Blocks(blockIDs)
			require.NoError(t, err)
			if scenario == "fetcher-no-blocks" {
				blocks = nil
			}
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
			out := &recordWriter{closed: make(chan struct{}), records: make(chan Record, 1024)}
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
			if kind == "fetcher" {
				var action, completion Record
				for len(out.records) > 0 {
					row := <-out.records
					if row.Kind == "action" {
						action = row
					}
					if row.Kind == "fetch_completed" {
						completion = row
					}
				}
				require.NotZero(t, action.Request)
				require.Equal(t, action.Request, completion.Request)
				if scenario == "fetcher-no-blocks" {
					require.Equal(t, blockfetch.ErrNoBlocks.Error(), completion.Error)
				} else {
					require.Empty(t, completion.Error)
				}
			}
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

func TestReadControlsPreservesObservedEOFAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	commands := make(chan commandResult, 1)
	readControls(ctx, bytes.NewReader(nil), commands)
	require.ErrorIs(t, (<-commands).err, io.EOF)
}

type blockedOutput struct {
	entered   chan struct{}
	closed    chan struct{}
	once      sync.Once
	closeOnce sync.Once
}

func (w *blockedOutput) Write([]byte) (int, error) {
	w.once.Do(func() { close(w.entered) })
	<-w.closed
	return 0, io.ErrClosedPipe
}
func (w *blockedOutput) Close() error { w.closeOnce.Do(func() { close(w.closed) }); return nil }

func TestServeInterruptsBlockedOutputAtCallerDeadline(t *testing.T) {
	b, err := Load("testdata/reference/upstream.json")
	require.NoError(t, err)
	w := &blockedOutput{entered: make(chan struct{}), closed: make(chan struct{})}
	input, control := io.Pipe()
	defer control.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, b, Options{Run: "blocked", Endpoints: map[string]string{"upstream": "127.0.0.1:0"}}, input, w)
	}()
	<-w.entered
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(5 * time.Second):
		_ = w.Close()
		<-done
		t.Fatal("output write survived drain deadline")
	}
}

type controlledPeer struct {
	events   chan peer.Event
	draining chan struct{}
	release  chan struct{}
}

func (p *controlledPeer) Events() <-chan peer.Event { return p.events }
func (p *controlledPeer) Close() error              { return nil }
func (p *controlledPeer) Finish(ctx context.Context) (peer.ObservationStats, error) {
	close(p.draining)
	var err error
	select {
	case <-p.release:
	case <-ctx.Done():
		err = ctx.Err()
	}
	close(p.events)
	return peer.ObservationStats{Finalized: true}, err
}

func controlledRun(t *testing.T, p *controlledPeer, fetch interface {
	Request(context.Context, pcommon.Point, pcommon.Point) (uint64, error)
},
) *runHandle {
	t.Helper()
	b, err := Load("testdata/reference/upstream.json")
	require.NoError(t, err)
	b.manifest.DurationNS = int64(20 * time.Millisecond)
	b.manifest.Actions = nil
	b.manifest.Roles[0].Required = []Requirement{}
	if fetch != nil {
		b.manifest.Actions = []Action{{ID: "fetch", Kind: "fetch", Role: b.manifest.Roles[0].ID, Items: []string{"b1-000", "b1-000"}}}
		b.manifest.DurationNS = int64(time.Hour)
	}
	input, control := io.Pipe()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	r := &runHandle{input: control, records: make(chan Record, 100), done: make(chan error, 1), cancel: cancel}
	w := &recordWriter{records: r.records, closed: make(chan struct{})}
	go func() {
		r.done <- serve(ctx, b, Options{Run: "trial"}, input, w, func(_ context.Context, _ *Bundle, role Role, _ string) (*activeRole, error) {
			return &activeRole{definition: role, peer: p, fetcher: fetch, counts: map[string]eventCount{}}, nil
		})
		close(r.records)
	}()
	t.Cleanup(func() { cancel(); _ = control.Close() })
	nextRecord(t, r, "ready")
	startTrial(t, r)
	return r
}

func TestControlFailuresDuringDrainInvalidateFinal(t *testing.T) {
	for _, mode := range []string{"stop", "eof"} {
		t.Run(mode, func(t *testing.T) {
			p := &controlledPeer{events: make(chan peer.Event), draining: make(chan struct{}), release: make(chan struct{})}
			r := controlledRun(t, p, nil)
			<-p.draining
			if mode == "stop" {
				require.NoError(t, json.NewEncoder(r.input).Encode(Control{Version: 1, Kind: "stop", Run: "trial"}))
			} else {
				require.NoError(t, r.input.Close())
			}
			close(p.release)
			require.Error(t, <-r.done)
			final := nextRecord(t, r, "final")
			require.NotNil(t, final.Complete)
			require.False(t, *final.Complete)
		})
	}
}

func TestPeerFailureStillExportsRemainingTail(t *testing.T) {
	p := &controlledPeer{events: make(chan peer.Event, 2), draining: make(chan struct{}), release: make(chan struct{})}
	// Delay the failure until startup returns; retain a valid record after it.
	r := controlledRun(t, p, nil)
	p.events <- peer.Event{Kind: peer.EventFetchCompleted, Sequence: 1, OccurredAt: time.Now(), Err: errors.New("range failed")}
	p.events <- peer.Event{Kind: peer.EventRequestNext, Sequence: 2, OccurredAt: time.Now()}
	close(p.release)
	require.ErrorContains(t, <-r.done, "range failed")
	tail := nextRecord(t, r, "request_next")
	require.Equal(t, uint64(2), tail.Sequence)
	final := nextRecord(t, r, "final")
	require.False(t, *final.Complete)
}

type blockedFetch struct {
	entered chan struct{}
	release chan struct{}
}

func (f *blockedFetch) Request(ctx context.Context, _, _ pcommon.Point) (uint64, error) {
	close(f.entered)
	select {
	case <-ctx.Done():
		return 0, context.Cause(ctx)
	case <-f.release:
		return 0, context.Canceled
	}
}

func TestStopCancelsBlockedFetchAdmission(t *testing.T) {
	f := &blockedFetch{entered: make(chan struct{}), release: make(chan struct{})}
	p := &controlledPeer{events: make(chan peer.Event), draining: make(chan struct{}), release: make(chan struct{})}
	close(p.release)
	r := controlledRun(t, p, f)
	<-f.entered
	require.NoError(t, json.NewEncoder(r.input).Encode(Control{Version: 1, Kind: "stop", Run: "trial"}))
	select {
	case err := <-r.done:
		require.ErrorContains(t, err, "unexpected control: stop")
	case <-time.After(time.Second):
		r.cancel()
		close(f.release)
		<-r.done
		t.Fatal("stop did not cancel fetch admission")
	}
}

func TestTransactionRecordsDistinguishEraForTheSameHash(t *testing.T) {
	var id [32]byte
	id[0] = 1
	e := peer.Event{Kind: peer.EventTxsRequested, Sequence: 1, OccurredAt: time.Now(), TxRefs: []peer.TxRef{{EraId: 5, ID: id}, {EraId: 6, ID: id}}}
	r, err := eventRecord(e)
	require.NoError(t, err)
	require.Len(t, r.TransactionRefs, 2)
	require.Equal(t, r.TransactionRefs[0].Hash, r.TransactionRefs[1].Hash)
	require.Equal(t, uint16(5), r.TransactionRefs[0].Era)
	require.Equal(t, uint16(6), r.TransactionRefs[1].Era)
	e.TxRefs = nil
	e.Txs = []peer.Tx{{EraId: 6, ID: id}}
	r, err = eventRecord(e)
	require.NoError(t, err)
	require.Equal(t, uint16(6), r.TransactionRefs[0].Era)
}

func TestFailedTransactionReceiveRetainsItemPopulation(t *testing.T) {
	r, err := eventRecord(peer.Event{Kind: peer.EventTxReceiveFailed, OccurredAt: time.Now(), Sequence: 1, Count: 2, TxRefs: []peer.TxRef{{EraId: 5}, {EraId: 6}}, Err: errors.New("receive failed")})
	require.NoError(t, err)
	require.Equal(t, uint64(2), r.Count)
	require.Len(t, r.TransactionRefs, 2)
	require.Equal(t, "receive failed", r.Error)
}

func TestServeReportsOutputCloseFailure(t *testing.T) {
	r := startTestRun(t, []Requirement{}, "close")
	nextRecord(t, r, "ready")
	startTrial(t, r)
	require.ErrorContains(t, <-r.done, "output close failed")
}
