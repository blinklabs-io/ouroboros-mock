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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	"github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/ouroboros-mock/peer"
)

// Options identifies a run and explicitly supplies each role's network endpoint.
type Options struct {
	Run       string
	Endpoints map[string]string
}

// Control is a controller-owned start time or an explicit stop request.
type Control struct {
	Version int       `json:"version"`
	Kind    string    `json:"kind"`
	Run     string    `json:"run"`
	Start   time.Time `json:"start,omitempty"`
}

type observedPeer interface {
	Events() <-chan peer.Event
	Finish(context.Context) (peer.ObservationStats, error)
	Close() error
}

type eventCount struct{ events, items uint64 }

type activeRole struct {
	counts     map[string]eventCount
	definition Role
	peer       observedPeer
	endpoint   string
	upstream   *peer.Upstream
	tx         *peer.TxPeer
	fetcher    interface {
		Request(context.Context, common.Point, common.Point) (uint64, error)
	}
}

type output struct {
	mu              sync.Mutex
	enc             *json.Encoder
	closer          io.Closer
	closeOnce       sync.Once
	closeErr        error
	err             error
	run, experiment string
}

func (o *output) write(r Record) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.err != nil {
		return o.err
	}
	r.Version = SchemaVersion
	r.Run = o.run
	r.Experiment = o.experiment
	if r.At == "" {
		r.At = stamp(time.Now())
	}
	o.err = o.enc.Encode(r)
	return o.err
}

func (o *output) close() error {
	o.closeOnce.Do(func() { o.closeErr = o.closer.Close() })
	return o.closeErr
}

func startRole(ctx context.Context, b *Bundle, r Role, endpoint string) (*activeRole, error) {
	a := &activeRole{definition: r, endpoint: endpoint, counts: make(map[string]eventCount)}
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint required for role %q", r.ID)
	}
	if r.Kind == "upstream" {
		blocks, err := b.Blocks(r.Initial)
		if err != nil {
			return nil, err
		}
		u, err := peer.NewUpstream(peer.UpstreamConfig{Blocks: blocks, NetworkMagic: b.manifest.Network.Magic})
		if err != nil {
			return nil, err
		}
		lc := net.ListenConfig{}
		listener, err := lc.Listen(ctx, "tcp", endpoint)
		if err != nil {
			_, _ = u.Finish(ctx)
			return nil, err
		}
		if err = u.AcceptContext(ctx, listener); err != nil {
			_ = listener.Close()
			_, _ = u.Finish(ctx)
			return nil, err
		}
		a.peer = u
		a.upstream = u
		a.endpoint = listener.Addr().String()
		return a, nil
	}
	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	switch r.Kind {
	case "follower":
		a.peer, err = peer.NewFollowerContext(ctx, peer.FollowerConfig{Conn: conn, NetworkMagic: b.manifest.Network.Magic, KeepAlive: true})
	case "txpeer":
		a.tx, err = peer.NewTxPeer(peer.TxPeerConfig{Conn: conn, NetworkMagic: b.manifest.Network.Magic, KeepAlive: true})
		a.peer = a.tx
	case "fetcher":
		fetcher, fetchErr := peer.NewFetcher(peer.FetcherConfig{Conn: conn, NetworkMagic: b.manifest.Network.Magic})
		a.fetcher, a.peer, err = fetcher, fetcher, fetchErr
	default:
		err = fmt.Errorf("unknown role %q", r.Kind)
	}
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return a, nil
}

type commandResult struct {
	control  Control
	err      error
	observed time.Time
}

func readControls(ctx context.Context, r io.Reader, out chan<- commandResult) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 65536)
	for scanner.Scan() {
		observed := time.Now()
		var c Control
		err := decodeJSON(scanner.Bytes(), &c)
		if err == nil && (c.Version != SchemaVersion || !oneOf(c.Kind, "start", "stop") || !name(c.Run)) {
			err = errors.New("invalid control version/kind/run")
		}
		result := commandResult{control: c, err: err, observed: observed}
		if !sendControl(ctx, out, result) {
			return
		}
		if err != nil {
			return
		}
	}
	err := scanner.Err()
	if err == nil {
		err = io.EOF
	}
	sendControl(ctx, out, commandResult{err: err, observed: time.Now()})
}

func sendControl(ctx context.Context, out chan<- commandResult, result commandResult) bool {
	select {
	case out <- result:
		return true
	default:
	}
	select {
	case out <- result:
		return true
	case <-ctx.Done():
		return false
	}
}

func drainContext(ctx context.Context) (context.Context, context.CancelFunc) {
	deadline := time.Now().Add(10 * time.Second)
	if parent, ok := ctx.Deadline(); ok && parent.Before(deadline) {
		deadline = parent
	}
	return context.WithDeadline(context.WithoutCancel(ctx), deadline)
}

type drainBudget struct {
	once            sync.Once
	ctx             context.Context
	cancel          context.CancelFunc
	stopOutputAbort func() bool
}

func (b *drainBudget) begin(ctx context.Context, out *output) context.Context {
	b.once.Do(func() {
		cleanupCtx, cancel := drainContext(ctx)
		b.ctx, b.cancel = cleanupCtx, cancel
		b.stopOutputAbort = context.AfterFunc(cleanupCtx, func() { _ = out.close() })
	})
	return b.ctx
}

// Serve runs real peer roles, accepting bounded JSON-line start/stop controls.
// Serve owns both streams. Their Close methods must interrupt pending I/O.
// Required populations cover the complete capture, including startup and drain;
// measurement-window populations are checked by the external controller.
func Serve(ctx context.Context, b *Bundle, opts Options, controls io.ReadCloser, w io.WriteCloser) (resultErr error) {
	return serve(ctx, b, opts, controls, w, startRole)
}

func serve(ctx context.Context, b *Bundle, opts Options, controls io.ReadCloser, w io.WriteCloser, starter func(context.Context, *Bundle, Role, string) (*activeRole, error)) (resultErr error) {
	if !name(opts.Run) {
		return errors.New("run identity required")
	}
	roles := map[string]bool{}
	for _, r := range b.manifest.Roles {
		roles[r.ID] = true
	}
	for id := range opts.Endpoints {
		if !roles[id] {
			return fmt.Errorf("unknown role endpoint %s", id)
		}
	}
	runCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	out := &output{enc: json.NewEncoder(w), closer: w, run: opts.Run, experiment: b.manifest.ID}
	defer func() { resultErr = errors.Join(resultErr, out.close()) }()
	budget := &drainBudget{}
	beginCleanup := func() { budget.begin(ctx, out) }
	stopParent := context.AfterFunc(ctx, beginCleanup)
	defer stopParent()
	commands := make(chan commandResult, 1)
	starts := make(chan commandResult, 1)
	readerCtx, stopReader := context.WithCancel(context.WithoutCancel(ctx))
	readerDone, monitorDone := make(chan struct{}), make(chan struct{})
	var inputClosing atomic.Int64
	go func() { defer close(readerDone); defer close(commands); readControls(readerCtx, controls, commands) }()
	go func() {
		defer close(monitorDone)
		started := false
		for command := range commands {
			if closing := inputClosing.Load(); closing != 0 && command.observed.UnixNano() > closing {
				continue
			}
			if !started && command.err == nil && command.control.Kind == "start" && command.control.Run == opts.Run && !command.control.Start.IsZero() {
				started = true
				starts <- command
				continue
			}
			err := command.err
			if err == nil {
				err = fmt.Errorf("unexpected control: %s for %s", command.control.Kind, command.control.Run)
			}
			cancel(fmt.Errorf("control input ended before finalization: %w", err))
			beginCleanup()
		}
	}()
	active := make(map[string]*activeRole, len(b.manifest.Roles))
	ordered := make([]*activeRole, 0, len(b.manifest.Roles))
	var collectors sync.WaitGroup
	var quiesceAt atomic.Int64
	defer func() {
		cleanupCtx := budget.begin(ctx, out)
		defer func() { budget.stopOutputAbort(); budget.cancel() }()
		quiesceAt.Store(time.Now().UnixNano())
		statsByRole := make(map[string]peer.ObservationStats, len(ordered))
		for _, a := range ordered {
			stats, err := a.peer.Finish(cleanupCtx)
			statsByRole[a.definition.ID] = stats
			resultErr = errors.Join(resultErr, err)
		}
		collectors.Wait()
		for _, a := range ordered {
			stats := statsByRole[a.definition.ID]
			resultErr = errors.Join(resultErr, out.write(Record{Kind: "source_final", Role: a.definition.ID, Stats: &stats}))
			for _, req := range a.definition.Required {
				count := a.counts[req.Event]
				if count.events < req.Events || count.items < req.Items {
					resultErr = errors.Join(resultErr, fmt.Errorf("role %s incomplete %s: events=%d items=%d", a.definition.ID, req.Event, count.events, count.items))
				}
			}
		}
		inputClosing.Store(time.Now().UnixNano())
		resultErr = errors.Join(resultErr, controls.Close())
		stopReader()
		<-readerDone
		<-monitorDone
		resultErr = errors.Join(resultErr, context.Cause(runCtx), ctx.Err(), cleanupCtx.Err())
		complete := resultErr == nil
		r := Record{Kind: "final", Complete: &complete}
		if resultErr != nil {
			r.Error = resultErr.Error()
		}
		resultErr = errors.Join(resultErr, out.write(r), ctx.Err(), cleanupCtx.Err())
	}()
	identity, err := executableIdentity()
	if err != nil {
		return err
	}
	if err = out.write(Record{Kind: "prepared", ManifestSHA256: b.digest, Build: &identity}); err != nil {
		return err
	}
	for _, r := range b.manifest.Roles {
		a, err := starter(runCtx, b, r, opts.Endpoints[r.ID])
		if err != nil {
			return err
		}
		ordered = append(ordered, a)
		active[r.ID] = a
		collectors.Add(1)
		go func() {
			defer collectors.Done()
			var previous uint64
			reported := false
			for e := range a.peer.Events() {
				record, err := eventRecord(e)
				if err == nil {
					record.Role = a.definition.ID
					count := a.counts[record.Kind]
					count.events++
					count.items += e.Count
					a.counts[record.Kind] = count
					err = out.write(record)
				}
				if e.Sequence != previous+1 {
					err = errors.Join(err, fmt.Errorf("observation gap in %s: %d after %d", a.definition.ID, e.Sequence, previous))
				}
				previous = e.Sequence
				if (e.Kind == peer.EventSessionClosed && (quiesceAt.Load() == 0 || e.OccurredAt.UnixNano() < quiesceAt.Load())) || (e.Err != nil && (e.Kind != peer.EventFetchCompleted || !errors.Is(e.Err, blockfetch.ErrNoBlocks))) {
					err = errors.Join(err, fmt.Errorf("peer operation ended: %s: %w", a.definition.ID, errors.Join(errors.New("session or operation failed"), e.Err)))
				}
				if err != nil && !reported {
					cancel(err)
					beginCleanup()
					reported = true
				}
			}
		}()
		if err = out.write(Record{Kind: "ready", Role: r.ID, Endpoint: a.endpoint}); err != nil {
			return err
		}
	}
	var start time.Time
	select {
	case <-runCtx.Done():
		return context.Cause(runCtx)
	case command := <-starts:
		start = command.control.Start
	}
	end := start.Add(time.Duration(b.manifest.DurationNS))
	endTimer := time.AfterFunc(time.Until(end), beginCleanup)
	defer endTimer.Stop()
	if err := out.write(Record{Kind: "started", At: stamp(start)}); err != nil {
		return err
	}
	timer := time.NewTimer(time.Until(start))
	defer timer.Stop()
	select {
	case <-runCtx.Done():
		return context.Cause(runCtx)
	case <-timer.C:
	}
	actionCtx, actionCancel := context.WithDeadline(runCtx, end)
	defer actionCancel()
	for _, action := range b.manifest.Actions {
		timer.Reset(time.Until(start.Add(time.Duration(action.AtNS))))
		select {
		case <-runCtx.Done():
			return context.Cause(runCtx)
		case <-timer.C:
		}
		at := time.Now()
		if err := actionCtx.Err(); err != nil {
			return err
		}
		if !at.Before(end) {
			return fmt.Errorf("action %s dispatched after trial end", action.ID)
		}
		role := active[action.Role]
		if role == nil {
			return fmt.Errorf("action %s has no active role", action.ID)
		}
		request, err := perform(actionCtx, b, role, action)
		elapsed, scheduled := at.Sub(start).Nanoseconds(), action.AtNS
		r := Record{Kind: "action", Role: action.Role, Action: action.ID, Request: request, At: stamp(at), ScheduledNS: &scheduled, ElapsedNS: &elapsed, Count: uint64(len(action.Items))}
		if err != nil {
			r.Error = err.Error()
		}
		if writeErr := out.write(r); writeErr != nil {
			return errors.Join(err, writeErr)
		}
		if err != nil {
			return err
		}
	}
	timer.Reset(time.Until(end))
	select {
	case <-runCtx.Done():
		return context.Cause(runCtx)
	case <-timer.C:
		return nil
	}
}

func perform(ctx context.Context, b *Bundle, a *activeRole, action Action) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	switch action.Kind {
	case "offer":
		txs, err := b.Transactions(action.Items)
		if err != nil {
			return 0, err
		}
		a.tx.Offer(txs...)
		return 0, nil
	case "append", "fork":
		blocks, err := b.Blocks(action.Items)
		if err != nil {
			return 0, err
		}
		if action.Kind == "append" {
			return 0, a.upstream.Append(blocks...)
		}
		_, err = a.upstream.SwitchFork(blocks)
		return 0, err
	case "fetch":
		var points [2]common.Point
		for i, id := range action.Items {
			blocks, err := b.Blocks([]string{id})
			if err != nil {
				return 0, err
			}
			block := blocks[0]
			points[i] = common.NewPoint(block.SlotNumber(), block.Hash().Bytes())
		}
		return a.fetcher.Request(ctx, points[0], points[1])
	default:
		return 0, fmt.Errorf("unsupported action %q", action.Kind)
	}
}
