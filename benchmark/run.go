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
	fetcher    *peer.Fetcher
}

type output struct {
	mu              sync.Mutex
	enc             *json.Encoder
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
		a.fetcher, err = peer.NewFetcher(peer.FetcherConfig{Conn: conn, NetworkMagic: b.manifest.Network.Magic})
		a.peer = a.fetcher
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
	control Control
	err     error
}

func readControls(ctx context.Context, r io.Reader, out chan<- commandResult) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 65536)
	for scanner.Scan() {
		var c Control
		err := decodeJSON(scanner.Bytes(), &c)
		if err == nil && (c.Version != SchemaVersion || !oneOf(c.Kind, "start", "stop") || !name(c.Run)) {
			err = errors.New("invalid control version/kind/run")
		}
		select {
		case out <- commandResult{control: c, err: err}:
		case <-ctx.Done():
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
	select {
	case out <- commandResult{err: err}:
	case <-ctx.Done():
	}
}

// Serve runs real peer roles, accepting bounded JSON-line start/stop controls.
// The control reader is owned by Serve and is closed when the run terminates.
func Serve(ctx context.Context, b *Bundle, opts Options, controls io.ReadCloser, w io.Writer) (resultErr error) {
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
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	closeInput := context.AfterFunc(runCtx, func() { _ = controls.Close() })
	defer closeInput()
	defer controls.Close()
	out := &output{enc: json.NewEncoder(w), run: opts.Run, experiment: b.manifest.ID}
	identity, identityErr := executableIdentity()
	if identityErr != nil {
		return identityErr
	}
	if err := out.write(Record{Kind: "prepared", ManifestSHA256: b.digest, Build: &identity}); err != nil {
		return err
	}
	active := make(map[string]*activeRole, len(b.manifest.Roles))
	ordered := make([]*activeRole, 0, len(b.manifest.Roles))
	var collectors sync.WaitGroup
	var quiesceAt atomic.Int64
	failures := make(chan error, len(b.manifest.Roles)+1)
	defer func() {
		quiesceAt.Store(time.Now().UnixNano())
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cleanupCancel()
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
		for len(failures) > 0 {
			resultErr = errors.Join(resultErr, <-failures)
		}
		complete := resultErr == nil
		r := Record{Kind: "final", Complete: &complete}
		if resultErr != nil {
			r.Error = resultErr.Error()
		}
		resultErr = errors.Join(resultErr, out.write(r))
	}()
	for _, r := range b.manifest.Roles {
		a, err := startRole(runCtx, b, r, opts.Endpoints[r.ID])
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
				if reported {
					continue
				}
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
				if (e.Kind == peer.EventSessionClosed && (quiesceAt.Load() == 0 || e.OccurredAt.UnixNano() < quiesceAt.Load())) || e.Err != nil {
					err = errors.Join(err, fmt.Errorf("peer operation ended: %s: %w", a.definition.ID, errors.Join(errors.New("session or operation failed"), e.Err)))
				}
				if err != nil {
					select {
					case failures <- err:
					default:
					}
					reported = true
				}
			}
		}()
		if err = out.write(Record{Kind: "ready", Role: r.ID, Endpoint: a.endpoint}); err != nil {
			return err
		}
	}
	commands := make(chan commandResult, 1)
	readerDone := make(chan struct{})
	go func() { defer close(readerDone); readControls(runCtx, controls, commands) }()
	defer func() { cancel(); _ = controls.Close(); <-readerDone }()
	var start time.Time
	select {
	case <-runCtx.Done():
		return runCtx.Err()
	case err := <-failures:
		return err
	case command := <-commands:
		if command.err != nil {
			return command.err
		}
		c := command.control
		if c.Kind != "start" || c.Run != opts.Run || c.Start.IsZero() {
			return errors.New("explicit matching start control required")
		}
		start = c.Start
	}
	if err := out.write(Record{Kind: "started", At: stamp(start)}); err != nil {
		return err
	}
	timer := time.NewTimer(time.Until(start))
	defer timer.Stop()
	select {
	case <-runCtx.Done():
		return runCtx.Err()
	case err := <-failures:
		return err
	case command := <-commands:
		return fmt.Errorf("control before start: %v", command)
	case <-timer.C:
	}
	actionCtx, actionCancel := context.WithDeadline(runCtx, start.Add(time.Duration(b.manifest.DurationNS)))
	defer actionCancel()
	for _, action := range b.manifest.Actions {
		timer.Reset(time.Until(start.Add(time.Duration(action.AtNS))))
		select {
		case <-runCtx.Done():
			return runCtx.Err()
		case err := <-failures:
			return err
		case command := <-commands:
			return fmt.Errorf("interrupted action %s: %v", action.ID, command)
		case <-timer.C:
		}
		at := time.Now()
		role, ok := active[action.Role]
		if !ok || role == nil {
			return fmt.Errorf("action %s has no active role", action.ID)
		}
		err := perform(actionCtx, b, role, action)
		elapsed := at.Sub(start).Nanoseconds()
		scheduled := action.AtNS
		r := Record{Kind: "action", Role: action.Role, Action: action.ID, At: stamp(at), ScheduledNS: &scheduled, ElapsedNS: &elapsed, Count: uint64(len(action.Items))}
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
	timer.Reset(time.Until(start.Add(time.Duration(b.manifest.DurationNS))))
	select {
	case <-runCtx.Done():
		return runCtx.Err()
	case err := <-failures:
		return err
	case command := <-commands:
		return fmt.Errorf("interrupted measurement: %v", command)
	case <-timer.C:
		return nil
	}
}

func perform(ctx context.Context, b *Bundle, a *activeRole, action Action) error {
	switch action.Kind {
	case "offer":
		txs, err := b.Transactions(action.Items)
		if err != nil {
			return err
		}
		a.tx.Offer(txs...)
		return nil
	case "append", "fork":
		blocks, err := b.Blocks(action.Items)
		if err != nil {
			return err
		}
		if action.Kind == "append" {
			return a.upstream.Append(blocks...)
		}
		_, err = a.upstream.SwitchFork(blocks)
		return err
	case "fetch":
		var points [2]common.Point
		for i, id := range action.Items {
			blocks, err := b.Blocks([]string{id})
			if err != nil {
				return err
			}
			block := blocks[0]
			points[i] = common.NewPoint(block.SlotNumber(), block.Hash().Bytes())
		}
		_, err := a.fetcher.Request(ctx, points[0], points[1])
		return err
	default:
		return fmt.Errorf("unsupported action %q", action.Kind)
	}
}
