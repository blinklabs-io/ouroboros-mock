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
	"sync"
	"testing"
	"time"

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
}

func (r *responder) requestNext(ctx chainsync.CallbackContext) error {
	r.mu.Lock()
	defer r.mu.Unlock()
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

			r := &responder{
				findIntersect: func(points []pcommon.Point) (pcommon.Point, chainsync.Tip, error) {
					require.NotEmpty(t, points)
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
