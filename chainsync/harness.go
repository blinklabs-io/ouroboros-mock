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
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/muxer"
	"github.com/blinklabs-io/gouroboros/protocol"
	gchainsync "github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/gouroboros/protocol/handshake"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// Mode selects the chain-sync protocol variant the harness negotiates.
type Mode int

const (
	// ModeNtC drives the node-to-client chain-sync protocol (the default).
	ModeNtC Mode = iota
	// ModeNtN drives the node-to-node chain-sync protocol.
	ModeNtN
)

const (
	defaultObserveBuffer    = 32
	defaultHandshakeTimeout = 10 * time.Second
)

// ErrClosed is returned by driver and observe operations after the harness has
// been closed.
var ErrClosed = errors.New("chainsync harness closed")

// Config configures a [Harness].
type Config struct {
	// Mode selects NtN or NtC. The zero value is ModeNtC.
	Mode Mode

	// ChainSync is the chain-sync configuration for the server under test. Its
	// FindIntersectFunc and RequestNextFunc callbacks are what the harness
	// exercises. It is required.
	ChainSync gchainsync.Config

	// NetworkMagic is the handshake network magic. Defaults to
	// ouroboros_mock.MockNetworkMagic when zero.
	NetworkMagic uint32

	// ObserveBuffer is the buffer size of the observed-message channel.
	// Defaults to 32 when non-positive.
	ObserveBuffer int

	// HandshakeTimeout bounds how long New waits for the handshake to complete.
	// Defaults to 10s when non-positive.
	HandshakeTimeout time.Duration
}

func (c Config) withDefaults() Config {
	if c.NetworkMagic == 0 {
		c.NetworkMagic = ouroboros_mock.MockNetworkMagic
	}
	if c.ObserveBuffer <= 0 {
		c.ObserveBuffer = defaultObserveBuffer
	}
	if c.HandshakeTimeout <= 0 {
		c.HandshakeTimeout = defaultHandshakeTimeout
	}
	return c
}

// Harness wires a real gouroboros chain-sync server (the system under test) to
// a mock client driver over an in-memory pipe. Tests drive the server with
// [Harness.FindIntersect] and [Harness.RequestNext] and observe every message
// the server emits with [Harness.Observe].
//
// A Harness must be created with [New] and released with [Harness.Close].
type Harness struct {
	cfg Config

	sut    *ouroboros.Connection
	server *gchainsync.Server

	driverConn   net.Conn
	muxer        *muxer.Muxer
	recvChan     chan *muxer.Segment
	csProtocolId uint16
	msgFromCbor  protocol.MessageFromCborFunc

	observed  chan ServerMessage
	doneChan  chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// New creates and starts a Harness. It stands up the chain-sync server under
// test, completes the handshake with the mock driver, and starts observing
// server traffic. The returned Harness is ready to be driven.
func New(cfg Config) (*Harness, error) {
	cfg = cfg.withDefaults()

	driverConn, sutConn := net.Pipe()

	m := muxer.New(driverConn)
	// A single ProtocolUnknown catch-all receiver collects inbound segments for
	// every protocol, exactly as the scripted mock Connection does. The driver
	// can still send on any protocol ID via muxer.Send.
	_, recvChan, _ := m.RegisterProtocol(
		muxer.ProtocolUnknown,
		muxer.ProtocolRoleInitiator,
	)
	m.Start()

	h := &Harness{
		cfg:          cfg,
		driverConn:   driverConn,
		muxer:        m,
		recvChan:     recvChan,
		csProtocolId: cfg.Mode.chainSyncProtocolId(),
		msgFromCbor:  cfg.Mode.msgFromCborFunc(),
		observed:     make(chan ServerMessage, cfg.ObserveBuffer),
		doneChan:     make(chan struct{}),
	}

	// ouroboros.New blocks until the handshake completes, so run it concurrently
	// while the driver performs the client half of the handshake.
	sutCh := make(chan sutResult, 1)
	go func() {
		conn, err := ouroboros.New(h.sutOptions(sutConn)...)
		sutCh <- sutResult{conn: conn, err: err}
	}()

	if err := h.clientHandshake(); err != nil {
		h.abort(sutCh, sutConn)
		return nil, err
	}

	res := <-sutCh
	if res.err != nil {
		h.teardownDriver()
		return nil, fmt.Errorf("server setup failed: %w", res.err)
	}
	h.sut = res.conn

	// Start ONLY the chain-sync server. In NtN mode the block-fetch and
	// tx-submission servers would otherwise emit unsolicited traffic onto the
	// driver's catch-all channel; delaying and selectively starting keeps the
	// observed stream limited to chain-sync.
	cs := h.sut.ChainSync()
	if cs == nil || cs.Server == nil {
		_ = h.Close()
		return nil, errors.New("chain-sync server was not initialized")
	}
	h.server = cs.Server
	h.server.Start()

	// The handshake response has been consumed; everything from here on is
	// chain-sync traffic.
	h.wg.Add(1)
	go h.readLoop()

	return h, nil
}

// sutResult carries the outcome of the concurrent ouroboros.New call.
type sutResult struct {
	conn *ouroboros.Connection
	err  error
}

// abort cleans up after a failed handshake, ensuring the concurrently-running
// ouroboros.New goroutine unblocks and its connection (if any) is closed.
func (h *Harness) abort(sutCh <-chan sutResult, sutConn net.Conn) {
	// Closing our side unblocks the server's handshake so its New returns.
	h.teardownDriver()
	_ = sutConn.Close()
	res := <-sutCh
	if res.conn != nil {
		_ = res.conn.Close()
	}
}

func (h *Harness) sutOptions(
	sutConn net.Conn,
) []ouroboros.ConnectionOptionFunc {
	opts := []ouroboros.ConnectionOptionFunc{
		ouroboros.WithConnection(sutConn),
		ouroboros.WithNetworkMagic(h.cfg.NetworkMagic),
		ouroboros.WithServer(true),
		// Delay protocol start so we can start only chain-sync ourselves.
		ouroboros.WithDelayProtocolStart(true),
		ouroboros.WithChainSyncConfig(h.cfg.ChainSync),
	}
	if h.cfg.Mode == ModeNtN {
		opts = append(opts, ouroboros.WithNodeToNode(true))
	}
	return opts
}

// clientHandshake sends ProposeVersions and consumes the server's
// AcceptVersion so the server transitions into running the mini-protocols.
func (h *Harness) clientHandshake() error {
	proposeMsg := h.cfg.Mode.proposeVersionsMsg(h.cfg.NetworkMagic)
	if err := h.sendSegment(handshake.ProtocolId, false, proposeMsg); err != nil {
		return fmt.Errorf("send handshake propose: %w", err)
	}
	timer := time.NewTimer(h.cfg.HandshakeTimeout)
	defer timer.Stop()
	select {
	case seg, ok := <-h.recvChan:
		if !ok {
			return errors.New(
				"handshake failed: connection closed before response",
			)
		}
		if seg.GetProtocolId() != handshake.ProtocolId {
			return fmt.Errorf(
				"handshake: unexpected protocol id %d",
				seg.GetProtocolId(),
			)
		}
		// Reaching here means the server accepted our proposal; the negotiated
		// version detail is not needed by the harness.
		return nil
	case <-timer.C:
		return errors.New("handshake timed out")
	}
}

// FindIntersect drives the server with a FindIntersect request for the provided
// points, invoking the server's FindIntersectFunc callback.
func (h *Harness) FindIntersect(points []pcommon.Point) error {
	return h.sendChainSync(gchainsync.NewMsgFindIntersect(points))
}

// RequestNext drives the server with a RequestNext request, invoking the
// server's RequestNextFunc callback.
func (h *Harness) RequestNext() error {
	return h.sendChainSync(gchainsync.NewMsgRequestNext())
}

// SendDone sends a chain-sync Done message, asking the server to stop the
// protocol. It is the deterministic "protocol stop" control: a graceful stop
// leaves the server-under-test with no protocol error on [Harness.ServerErrors].
//
// Done is terminal for the current drive. The gouroboros server handles it by
// stopping and asynchronously re-initializing a fresh protocol instance, so a
// request sent immediately after SendDone races that restart and may be
// dropped. To continue driving after a Done, start a new [Harness] rather than
// reusing this one.
func (h *Harness) SendDone() error {
	return h.sendChainSync(gchainsync.NewMsgDone())
}

func (h *Harness) sendChainSync(msg protocol.Message) error {
	select {
	case <-h.doneChan:
		return ErrClosed
	default:
	}
	return h.sendSegment(h.csProtocolId, false, msg)
}

func (h *Harness) sendSegment(
	protocolId uint16,
	isResponse bool,
	msg protocol.Message,
) error {
	data := msg.Cbor()
	if data == nil {
		var err error
		data, err = cbor.Encode(msg)
		if err != nil {
			return fmt.Errorf("encode message: %w", err)
		}
	}
	// A muxer segment payload cannot exceed SegmentMaxPayloadLength, so split
	// oversized messages into sequential fragments. muxer.Send is serialized,
	// and the peer reassembles segments per protocol before decoding, so
	// ordering is preserved. This mirrors the reassembly done in readLoop.
	for {
		chunk := data
		if len(chunk) > muxer.SegmentMaxPayloadLength {
			chunk = chunk[:muxer.SegmentMaxPayloadLength]
		}
		seg := muxer.NewSegment(protocolId, chunk, isResponse)
		if seg == nil {
			return fmt.Errorf(
				"failed to build muxer segment for %d-byte fragment",
				len(chunk),
			)
		}
		if err := h.muxer.Send(seg); err != nil {
			return fmt.Errorf("send segment: %w", err)
		}
		data = data[len(chunk):]
		if len(data) == 0 {
			return nil
		}
	}
}

// Observe returns the next server message, blocking until one arrives, the
// context is cancelled, or the harness is closed. It returns the context error
// on cancellation and [ErrClosed] once the harness is closed and drained.
func (h *Harness) Observe(ctx context.Context) (ServerMessage, error) {
	select {
	case <-ctx.Done():
		return ServerMessage{}, ctx.Err()
	case msg, ok := <-h.observed:
		if !ok {
			return ServerMessage{}, ErrClosed
		}
		return msg, nil
	}
}

// Observed returns the channel of decoded server messages. It is closed once
// the harness shuts down. Most callers should prefer [Harness.Observe].
func (h *Harness) Observed() <-chan ServerMessage {
	return h.observed
}

// ServerErrors returns the error channel of the server-under-test connection.
// A protocol error raised by a server callback, or a send failure after
// [Harness.Disconnect], is surfaced here.
func (h *Harness) ServerErrors() <-chan error {
	return h.sut.ErrorChan()
}

// Server returns the chain-sync server under test. Callbacks receive the same
// server via their CallbackContext; this accessor is provided for tests that
// push messages (e.g. a late RollForward after an AwaitReply) out of band.
func (h *Harness) Server() *gchainsync.Server {
	return h.server
}

// Disconnect abruptly closes the driver side of the connection without a
// protocol Done. An in-flight or subsequent server send fails, which the server
// surfaces on [Harness.ServerErrors]. Use it to exercise send-failure paths
// deterministically. Call [Harness.Close] afterwards for full teardown.
func (h *Harness) Disconnect() error {
	return h.driverConn.Close()
}

// Close shuts down the harness: the server under test, the driver muxer, and
// all goroutines. It is safe to call multiple times.
func (h *Harness) Close() error {
	h.closeOnce.Do(func() {
		close(h.doneChan)
		if h.sut != nil {
			_ = h.sut.Close()
		}
		h.teardownDriver()
		h.wg.Wait()
	})
	return nil
}

// teardownDriver stops the driver muxer and waits for its goroutines to fully
// drain (signalled by the muxer error channel closing), so a subsequent
// goleak check sees no lingering goroutines.
func (h *Harness) teardownDriver() {
	h.muxer.Stop()
	// The muxer closes its error channel only after its internal goroutines
	// have exited and the connection is closed; draining to closure guarantees
	// a clean teardown.
	for range h.muxer.ErrorChan() { //nolint:revive
	}
}

func (h *Harness) readLoop() {
	defer h.wg.Done()
	defer close(h.observed)
	var leftover []byte
	for {
		select {
		case <-h.doneChan:
			return
		case seg, ok := <-h.recvChan:
			if !ok {
				return
			}
			if seg.GetProtocolId() != h.csProtocolId {
				// Only chain-sync traffic is expected; ignore anything else.
				continue
			}
			data := seg.Payload
			if len(leftover) > 0 {
				data = append(leftover, data...)
				leftover = nil
			}
			for len(data) > 0 {
				var raw cbor.RawMessage
				n, err := cbor.Decode(data, &raw)
				if err != nil {
					// Message is split across segments; keep a private copy of
					// the remainder and wait for more data.
					leftover = append([]byte(nil), data...)
					break
				}
				msgType, err := cbor.DecodeIdFromList(raw)
				if err != nil || msgType < 0 {
					// Undecodable framing; drop the rest of this segment.
					break
				}
				msg, err := h.msgFromCbor(uint(msgType), raw)
				if err != nil || msg == nil {
					break
				}
				data = data[n:]
				select {
				case h.observed <- ServerMessage{msg: msg}:
				case <-h.doneChan:
					return
				}
			}
		}
	}
}

// chainSyncProtocolId returns the muxer protocol ID for the mode.
func (m Mode) chainSyncProtocolId() uint16 {
	if m == ModeNtN {
		return gchainsync.ProtocolIdNtN
	}
	return gchainsync.ProtocolIdNtC
}

// msgFromCborFunc returns the chain-sync message decoder for the mode.
func (m Mode) msgFromCborFunc() protocol.MessageFromCborFunc {
	if m == ModeNtN {
		return gchainsync.NewMsgFromCborNtN
	}
	return gchainsync.NewMsgFromCborNtC
}

// proposeVersionsMsg builds the handshake ProposeVersions message the driver
// sends, mirroring the mock connection's negotiated versions.
func (m Mode) proposeVersionsMsg(networkMagic uint32) protocol.Message {
	if m == ModeNtN {
		return handshake.NewMsgProposeVersions(protocol.ProtocolVersionMap{
			ouroboros_mock.MockProtocolVersionNtN: protocol.VersionDataNtN13andUp{
				VersionDataNtN11to12: protocol.VersionDataNtN11to12{
					CborNetworkMagic:                       networkMagic,
					CborInitiatorAndResponderDiffusionMode: protocol.DiffusionModeInitiatorOnly,
					CborPeerSharing:                        protocol.PeerSharingModeNoPeerSharing,
					CborQuery:                              protocol.QueryModeDisabled,
				},
			},
		})
	}
	return handshake.NewMsgProposeVersions(protocol.ProtocolVersionMap{
		ouroboros_mock.MockProtocolVersionNtC: protocol.VersionDataNtC9to14(
			networkMagic,
		),
	})
}
