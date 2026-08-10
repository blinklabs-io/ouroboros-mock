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

package chainsync_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"testing"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	csmock "github.com/blinklabs-io/ouroboros-mock/chainsync"
)

// serverAction performs a single outbound send from within a RequestNext
// callback.
type serverAction func(*chainsync.Server) error

// responder is a scriptable chain-sync server implementation used to drive the
// harness deterministically. FindIntersect and each RequestNext consult
// caller-provided scripts, so tests control exactly which message the server
// emits in response to each request.
type responder struct {
	mu sync.Mutex

	findIntersect func([]pcommon.Point) (pcommon.Point, chainsync.Tip, error)
	actions       []serverAction

	// requestNextOverride, when set, handles RequestNext instead of the
	// action script. It receives the full callback context, so a test can
	// exercise behaviour that depends on CallbackContext.ConnectionId.
	requestNextOverride func(chainsync.CallbackContext) error
}

func (r *responder) requestNext(ctx chainsync.CallbackContext) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.requestNextOverride != nil {
		return r.requestNextOverride(ctx)
	}
	if len(r.actions) == 0 {
		return ctx.Server.AwaitReply()
	}
	action := r.actions[0]
	r.actions = r.actions[1:]
	return action(ctx.Server)
}

func (r *responder) config() chainsync.Config {
	return chainsync.NewConfig(
		chainsync.WithFindIntersectFunc(
			func(_ chainsync.CallbackContext, points []pcommon.Point) (pcommon.Point, chainsync.Tip, error) {
				return r.findIntersect(points)
			},
		),
		chainsync.WithRequestNextFunc(r.requestNext),
	)
}

// newHarness starts a harness for the given mode wired to the responder. The
// caller is responsible for closing it (typically `defer h.Close()` placed
// after `defer goleak.VerifyNone(t)` so teardown runs before the leak check).
func newHarness(
	t *testing.T,
	mode csmock.Mode,
	r *responder,
) *csmock.Harness {
	t.Helper()
	h, err := csmock.New(csmock.Config{
		Mode:      mode,
		ChainSync: r.config(),
	})
	require.NoError(t, err)
	return h
}

// observe returns the next server message, failing the test on timeout.
func observe(t *testing.T, h *csmock.Harness) csmock.ServerMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	msg, err := h.Observe(ctx)
	require.NoError(t, err)
	return msg
}

func allModes() []struct {
	name string
	mode csmock.Mode
} {
	return []struct {
		name string
		mode csmock.Mode
	}{
		{"NtC", csmock.ModeNtC},
		{"NtN", csmock.ModeNtN},
	}
}

// Drive FindIntersect and verify the matched point and tip round-trip back to
// the driver via IntersectFound.
func TestFindIntersectFound(t *testing.T) {
	for _, tc := range allModes() {
		t.Run(tc.name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			chain, err := csmock.BuildChain(1, common.Blake2b256{}, 0, 20, 3)
			require.NoError(t, err)
			wantPoint := chain.Points[1]
			wantTip := chain.Tip()

			// The callback runs on the server goroutine, where require's
			// FailNow (runtime.Goexit) would only unwind the callback and
			// hang the test. Capture the received points and assert them on
			// the test goroutine instead.
			gotPoints := make(chan []pcommon.Point, 1)
			r := &responder{
				findIntersect: func(points []pcommon.Point) (pcommon.Point, chainsync.Tip, error) {
					gotPoints <- points
					return wantPoint, wantTip, nil
				},
			}
			h := newHarness(t, tc.mode, r)
			defer h.Close()

			require.NoError(t, h.FindIntersect(chain.Points))

			msg := observe(t, h)
			require.True(t, msg.IsIntersectFound(), "expected IntersectFound")

			gotPoint, ok := msg.Point()
			require.True(t, ok)
			require.Equal(t, wantPoint, gotPoint)

			gotTip, ok := msg.Tip()
			require.True(t, ok)
			require.Equal(t, wantTip, gotTip)

			// The server received exactly the points we drove it with.
			require.Equal(t, chain.Points, <-gotPoints)
		})
	}
}

// A callback that returns ErrIntersectNotFound must produce an IntersectNotFound
// carrying the tip.
func TestFindIntersectNotFound(t *testing.T) {
	for _, tc := range allModes() {
		t.Run(tc.name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			chain, err := csmock.BuildChain(1, common.Blake2b256{}, 0, 20, 2)
			require.NoError(t, err)
			wantTip := chain.Tip()

			r := &responder{
				findIntersect: func([]pcommon.Point) (pcommon.Point, chainsync.Tip, error) {
					return pcommon.Point{}, wantTip, chainsync.ErrIntersectNotFound
				},
			}
			h := newHarness(t, tc.mode, r)
			defer h.Close()

			require.NoError(
				t,
				h.FindIntersect([]pcommon.Point{csmock.OriginPoint()}),
			)

			msg := observe(t, h)
			require.True(
				t,
				msg.IsIntersectNotFound(),
				"expected IntersectNotFound",
			)

			gotTip, ok := msg.Tip()
			require.True(t, ok)
			require.Equal(t, wantTip, gotTip)
		})
	}
}

// A FindIntersect whose encoded form exceeds a single muxer segment
// (SegmentMaxPayloadLength) must be split across segments by the driver and
// reassembled by the server, delivering every point to the callback.
func TestFindIntersectOversizedPayload(t *testing.T) {
	defer goleak.VerifyNone(t)

	// ~40 bytes/point encoded; 3000 points is well over the 65535-byte limit.
	const numPoints = 3000
	points := make([]pcommon.Point, numPoints)
	for i := range points {
		hash := make([]byte, 32)
		binary.BigEndian.PutUint64(hash, uint64(i))
		points[i] = pcommon.NewPoint(uint64(i), hash)
	}

	gotCount := make(chan int, 1)
	r := &responder{
		findIntersect: func(p []pcommon.Point) (pcommon.Point, chainsync.Tip, error) {
			gotCount <- len(p)
			return csmock.OriginPoint(), chainsync.Tip{}, nil
		},
	}
	h := newHarness(t, csmock.ModeNtC, r)
	defer h.Close()

	require.NoError(t, h.FindIntersect(points))

	msg := observe(t, h)
	require.True(t, msg.IsIntersectFound(), "expected IntersectFound")
	require.Equal(t, numPoints, <-gotCount)
}

// Concurrent driver calls, each sending a multi-segment message, must not
// interleave their fragments on the wire: every message the server decodes
// must carry the full point set, and no CBOR-decode error may surface.
func TestConcurrentOversizedSends(t *testing.T) {
	defer goleak.VerifyNone(t)

	const numPoints = 2000
	const senders = 4
	points := make([]pcommon.Point, numPoints)
	for i := range points {
		hash := make([]byte, 32)
		binary.BigEndian.PutUint64(hash, uint64(i))
		points[i] = pcommon.NewPoint(uint64(i), hash)
	}

	gotCounts := make(chan int, senders)
	r := &responder{
		findIntersect: func(p []pcommon.Point) (pcommon.Point, chainsync.Tip, error) {
			gotCounts <- len(p)
			return csmock.OriginPoint(), chainsync.Tip{}, nil
		},
	}
	h := newHarness(t, csmock.ModeNtC, r)
	defer h.Close()

	sendErrs := make(chan error, senders)
	for range senders {
		go func() { sendErrs <- h.FindIntersect(points) }()
	}
	for range senders {
		require.NoError(t, <-sendErrs)
	}

	// Every response must be a well-formed IntersectFound whose request
	// carried all the points; interleaved fragments would corrupt decoding.
	for range senders {
		msg := observe(t, h)
		require.True(t, msg.IsIntersectFound(), "expected IntersectFound")
		require.Equal(t, numPoints, <-gotCounts)
	}

	select {
	case err := <-h.ServerErrors():
		require.NoError(t, err)
	default:
	}
}

// Drive RequestNext and distinguish the roll-forward path. In NtC the full
// block CBOR round-trips; in NtN the wrapped header and tip are observed.
func TestRequestNextRollForward(t *testing.T) {
	for _, tc := range allModes() {
		t.Run(tc.name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			chain, err := csmock.BuildChain(1, common.Blake2b256{}, 0, 20, 1)
			require.NoError(t, err)
			block := chain.Blocks[0]
			tip := chain.Tips[0]

			r := &responder{
				actions: []serverAction{
					func(s *chainsync.Server) error {
						return s.RollForward(
							uint(block.Type()),
							block.Cbor(),
							tip,
						)
					},
				},
			}
			h := newHarness(t, tc.mode, r)
			defer h.Close()

			require.NoError(t, h.RequestNext())

			msg := observe(t, h)
			require.True(t, msg.IsRollForward(), "expected RollForward")

			gotTip, ok := msg.Tip()
			require.True(t, ok)
			require.Equal(t, tip, gotTip)

			switch tc.mode {
			case csmock.ModeNtC:
				blockType, blockCbor, rfTip, ok := msg.RollForwardNtC()
				require.True(t, ok)
				require.Equal(t, uint(block.Type()), blockType)
				require.Equal(t, block.Cbor(), blockCbor)
				require.Equal(t, tip, rfTip)
			case csmock.ModeNtN:
				header, rfTip, ok := msg.RollForwardNtN()
				require.True(t, ok)
				require.Equal(t, tip, rfTip)
				require.NotEmpty(t, header.HeaderCbor())
			}
		})
	}
}

// Drive RequestNext and distinguish the rollback path, verifying the point and
// tip.
func TestRequestNextRollBackward(t *testing.T) {
	for _, tc := range allModes() {
		t.Run(tc.name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			chain, err := csmock.BuildChain(1, common.Blake2b256{}, 0, 20, 2)
			require.NoError(t, err)
			rollbackPoint := chain.Points[0]
			tip := chain.Tip()

			r := &responder{
				actions: []serverAction{
					func(s *chainsync.Server) error {
						return s.RollBackward(rollbackPoint, tip)
					},
				},
			}
			h := newHarness(t, tc.mode, r)
			defer h.Close()

			require.NoError(t, h.RequestNext())

			msg := observe(t, h)
			require.True(t, msg.IsRollBackward(), "expected RollBackward")

			gotPoint, ok := msg.Point()
			require.True(t, ok)
			require.Equal(t, rollbackPoint, gotPoint)

			gotTip, ok := msg.Tip()
			require.True(t, ok)
			require.Equal(t, tip, gotTip)
		})
	}
}

// A server may answer RequestNext with AwaitReply and deliver the block later
// out of band. The harness observes both, exercising the Server accessor.
func TestRequestNextAwaitReplyThenRollForward(t *testing.T) {
	defer goleak.VerifyNone(t)

	chain, err := csmock.BuildChain(1, common.Blake2b256{}, 0, 20, 1)
	require.NoError(t, err)
	block := chain.Blocks[0]
	tip := chain.Tips[0]

	r := &responder{
		actions: []serverAction{
			func(s *chainsync.Server) error { return s.AwaitReply() },
		},
	}
	h := newHarness(t, csmock.ModeNtC, r)
	defer h.Close()

	require.NoError(t, h.RequestNext())

	awaitMsg := observe(t, h)
	require.True(t, awaitMsg.IsAwaitReply(), "expected AwaitReply")

	// Deliver the block out of band via the server accessor.
	require.NoError(
		t,
		h.Server().RollForward(uint(block.Type()), block.Cbor(), tip),
	)

	fwdMsg := observe(t, h)
	require.True(t, fwdMsg.IsRollForward(), "expected RollForward")
	gotTip, ok := fwdMsg.Tip()
	require.True(t, ok)
	require.Equal(t, tip, gotTip)
}

// Disconnecting the driver while the server is mid-callback deterministically
// fails the server's send path and surfaces a non-nil error on ServerErrors.
// The callback is held in CanAwait (a non-idle state) across the disconnect, so
// gouroboros never treats the close as graceful and never suppresses the error.
func TestSendFailureOnDisconnect(t *testing.T) {
	defer goleak.VerifyNone(t)

	chain, err := csmock.BuildChain(1, common.Blake2b256{}, 0, 20, 1)
	require.NoError(t, err)
	block := chain.Blocks[0]
	tip := chain.Tips[0]

	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	doRelease := func() { releaseOnce.Do(func() { close(release) }) }
	// Always release the blocked callback so its goroutine can exit before the
	// leak check, even if an assertion fails first.
	defer doRelease()

	r := &responder{
		actions: []serverAction{
			func(s *chainsync.Server) error {
				close(entered)
				<-release
				// The connection is gone by now; this send fails.
				return s.RollForward(uint(block.Type()), block.Cbor(), tip)
			},
		},
	}
	h := newHarness(t, csmock.ModeNtC, r)
	defer h.Close()

	require.NoError(t, h.RequestNext())

	// Wait until the callback is executing (chain-sync now in CanAwait), then
	// disconnect the driver while the protocol is non-idle.
	<-entered
	require.NoError(t, h.Disconnect())

	select {
	case err, ok := <-h.ServerErrors():
		require.True(t, ok, "expected an error, not a closed channel")
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for server send failure")
	}

	// Let the callback attempt its (doomed) send and unwind.
	doRelease()
}

// Observe honours context cancellation without any sleeps.
func TestObserveCancellation(t *testing.T) {
	defer goleak.VerifyNone(t)

	r := &responder{
		findIntersect: func([]pcommon.Point) (pcommon.Point, chainsync.Tip, error) {
			return pcommon.Point{}, chainsync.Tip{}, nil
		},
	}
	h := newHarness(t, csmock.ModeNtC, r)
	defer h.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := h.Observe(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

// Observe returns ErrClosed once the harness is closed.
func TestObserveAfterClose(t *testing.T) {
	defer goleak.VerifyNone(t)

	r := &responder{
		findIntersect: func([]pcommon.Point) (pcommon.Point, chainsync.Tip, error) {
			return pcommon.Point{}, chainsync.Tip{}, nil
		},
	}
	h, err := csmock.New(
		csmock.Config{Mode: csmock.ModeNtC, ChainSync: r.config()},
	)
	require.NoError(t, err)
	require.NoError(t, h.Close())

	_, err = h.Observe(context.Background())
	require.ErrorIs(t, err, csmock.ErrClosed)
}

// peerRegistry is a caller-owned connection registry keyed by connection ID,
// standing in for a consumer's connection manager (e.g. Dingo's connmanager).
type peerRegistry struct {
	mu    sync.Mutex
	conns map[ouroboros.ConnectionId]*ouroboros.Connection
}

func newPeerRegistry() *peerRegistry {
	return &peerRegistry{
		conns: make(map[ouroboros.ConnectionId]*ouroboros.Connection),
	}
}

func (p *peerRegistry) add(conn *ouroboros.Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.conns[conn.Id()] = conn
}

func (p *peerRegistry) get(
	id ouroboros.ConnectionId,
) *ouroboros.Connection {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.conns[id]
}

// A callback that parks the client with AwaitReply and then resolves its peer
// through a caller-owned registry must be drivable by the harness: the
// connection the harness exposes has to be the same one the callback is told
// about in CallbackContext.ConnectionId, so the lookup succeeds and the
// asynchronous RollForward that follows can be observed.
func TestRequestNextAsyncRollForwardViaCallerRegistry(t *testing.T) {
	for _, tc := range allModes() {
		t.Run(tc.name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			chain, err := csmock.BuildChain(1, common.Blake2b256{}, 0, 20, 1)
			require.NoError(t, err)
			block := chain.Blocks[0]
			tip := chain.Tips[0]

			registry := newPeerRegistry()
			blockReady := make(chan struct{})
			resolved := make(chan ouroboros.ConnectionId, 1)
			var wg sync.WaitGroup
			defer wg.Wait()

			// Mirror the downstream shape: AwaitReply, resolve the peer from
			// the registry, then serve asynchronously.
			r := &responder{}
			r.findIntersect = func(
				points []pcommon.Point,
			) (pcommon.Point, chainsync.Tip, error) {
				return points[0], tip, nil
			}
			r.requestNextOverride = func(
				ctx chainsync.CallbackContext,
			) error {
				if err := ctx.Server.AwaitReply(); err != nil {
					return err
				}
				conn := registry.get(ctx.ConnectionId)
				if conn == nil {
					return fmt.Errorf(
						"connection %s not found in registry",
						ctx.ConnectionId.String(),
					)
				}
				resolved <- ctx.ConnectionId
				wg.Add(1)
				go func() {
					defer wg.Done()
					select {
					case <-blockReady:
						_ = ctx.Server.RollForward(
							uint(block.Type()),
							block.Cbor(),
							tip,
						)
					case <-conn.ErrorChan():
					}
				}()
				return nil
			}

			h := newHarness(t, tc.mode, r)
			defer h.Close()

			// Register the server-under-test connection as a consumer would.
			conn := h.ServerConnection()
			require.NotNil(
				t,
				conn,
				"harness must expose its server connection",
			)
			registry.add(conn)

			require.NoError(t, h.RequestNext())
			require.True(t, observe(t, h).IsAwaitReply(), "expected AwaitReply")

			select {
			case gotId := <-resolved:
				require.Equal(
					t,
					conn.Id(),
					gotId,
					"callback ConnectionId must match the exposed connection",
				)
			case <-time.After(5 * time.Second):
				t.Fatal("callback never resolved its peer from the registry")
			}

			close(blockReady)

			fwdMsg := observe(t, h)
			require.True(t, fwdMsg.IsRollForward(), "expected async RollForward")
			gotTip, ok := fwdMsg.Tip()
			require.True(t, ok)
			require.Equal(t, tip, gotTip)
		})
	}
}

// A callback that watches its resolved peer's error channel while waiting to
// serve — the shape a consumer uses to abandon a blocked read when the peer
// goes away — must be woken by [Harness.Disconnect]. The callback takes that
// branch instead of sending, so no send is attempted and none is asserted
// here; send failure after a disconnect is covered by
// TestSendFailureOnDisconnect.
func TestRegistryResolvedPeerErrorChanWakesOnDisconnect(t *testing.T) {
	defer goleak.VerifyNone(t)

	chain, err := csmock.BuildChain(1, common.Blake2b256{}, 0, 20, 1)
	require.NoError(t, err)
	block := chain.Blocks[0]
	tip := chain.Tips[0]

	registry := newPeerRegistry()
	// Never signalled: the only way out of the callback's select is the peer
	// error channel.
	blockReady := make(chan struct{})
	abandoned := make(chan struct{})
	var wg sync.WaitGroup
	defer wg.Wait()

	r := &responder{}
	r.requestNextOverride = func(ctx chainsync.CallbackContext) error {
		if err := ctx.Server.AwaitReply(); err != nil {
			return err
		}
		conn := registry.get(ctx.ConnectionId)
		if conn == nil {
			return fmt.Errorf(
				"connection %s not found in registry",
				ctx.ConnectionId.String(),
			)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-blockReady:
				_ = ctx.Server.RollForward(
					uint(block.Type()),
					block.Cbor(),
					tip,
				)
			case <-conn.ErrorChan():
				close(abandoned)
			}
		}()
		return nil
	}

	h := newHarness(t, csmock.ModeNtC, r)
	defer h.Close()
	registry.add(h.ServerConnection())

	require.NoError(t, h.RequestNext())
	require.True(t, observe(t, h).IsAwaitReply(), "expected AwaitReply")

	require.NoError(t, h.Disconnect())

	select {
	case <-abandoned:
	case <-time.After(5 * time.Second):
		t.Fatal("resolved peer's error channel never woke the callback")
	}
}

// A consumer's connection manager typically closes the connections it owns on
// shutdown. Because the harness also closes the server connection, both can
// run; neither ordering may panic, double-close, or leak.
func TestServerConnectionCloseIsSafeFromCallerAndHarness(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("caller closes first", func(t *testing.T) {
		r := &responder{}
		h := newHarness(t, csmock.ModeNtC, r)
		defer h.Close()

		conn := h.ServerConnection()
		require.NotNil(t, conn)
		require.NoError(t, conn.Close())
		require.NoError(t, h.Close())
	})

	t.Run("harness closes first", func(t *testing.T) {
		r := &responder{}
		h := newHarness(t, csmock.ModeNtC, r)
		defer h.Close()

		conn := h.ServerConnection()
		require.NotNil(t, conn)
		require.NoError(t, h.Close())
		require.NoError(t, conn.Close())
	})
}
