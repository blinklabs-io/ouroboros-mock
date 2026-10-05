// Copyright 2025 Blink Labs Software
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

package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blinklabs-io/gouroboros/protocol/handshake"
	mock "github.com/blinklabs-io/ouroboros-mock"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: ouroboros-mock <configuration.yaml>")
	}
	// The positional argument explicitly grants access to the selected configuration file.
	// #nosec G703
	file, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("open configuration: %w", err)
	}
	cfg, entries, decodeErr := loadConfiguration(file)
	if err := errors.Join(decodeErr, file.Close()); err != nil {
		return err
	}
	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(ctx, string(cfg.Listener.Network), string(cfg.Listener.Address))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()
	if _, err := fmt.Fprintf(output, "listening on %s %s\n", listener.Addr().Network(), listener.Addr()); err != nil {
		return err
	}
	return serve(ctx, listener, entries)
}

func serve(ctx context.Context, listener net.Listener, entries []mock.ConversationEntry) error {
	return serveWithInitialMessageTimeout(ctx, listener, entries, handshake.ProposeTimeout)
}

func serveWithInitialMessageTimeout(
	ctx context.Context,
	listener net.Listener,
	entries []mock.ConversationEntry,
	timeout time.Duration,
) error {
	acceptDone := make(chan struct{})
	stopAccept := context.AfterFunc(ctx, func() { _ = listener.Close(); close(acceptDone) })
	defer func() {
		if !stopAccept() {
			<-acceptDone
		}
	}()
	conn, err := listener.Accept()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("accept: %w", err)
	}
	defer conn.Close()
	conn, err = withInitialMessageTimeout(conn, timeout)
	if err != nil {
		return err
	}
	mocked := mock.NewConnection(mock.ProtocolRoleClient, entries)
	defer mocked.Close()
	conversation, ok := mocked.(interface{ ErrorChan() <-chan error })
	if !ok {
		return errors.New("mock connection does not report conversation completion")
	}
	return bridge(ctx, conn, mocked, conversation.ErrorChan())
}

type initialMessageConn struct {
	net.Conn
	header           [8]byte
	headerBytes      int
	payloadRemaining int
	complete         bool
}

func withInitialMessageTimeout(conn net.Conn, timeout time.Duration) (net.Conn, error) {
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, fmt.Errorf("set initial message deadline: %w", err)
	}
	return &initialMessageConn{Conn: conn}, nil
}

func (c *initialMessageConn) Read(data []byte) (int, error) {
	n, err := c.Conn.Read(data)
	if n == 0 || c.complete {
		return n, err
	}
	remaining := data[:n]
	if c.headerBytes < len(c.header) {
		copied := copy(c.header[c.headerBytes:], remaining)
		c.headerBytes += copied
		remaining = remaining[copied:]
		if c.headerBytes == len(c.header) {
			c.payloadRemaining = int(binary.BigEndian.Uint16(c.header[6:8]))
		}
	}
	if c.headerBytes == len(c.header) {
		consumed := min(len(remaining), c.payloadRemaining)
		c.payloadRemaining -= consumed
		if c.payloadRemaining == 0 {
			c.complete = true
			if clearErr := c.SetReadDeadline(time.Time{}); clearErr != nil && err == nil {
				err = fmt.Errorf("clear initial message deadline: %w", clearErr)
			}
		}
	}
	return n, err
}

type copyResult struct {
	err       error
	sourceErr error
}

type observedReader struct {
	io.Reader
	err error
}

func (r *observedReader) Read(data []byte) (int, error) {
	n, err := r.Reader.Read(data)
	if err != nil {
		r.err = err
	}
	return n, err
}

func copyConnection(destination io.Writer, source io.Reader, result chan<- copyResult) {
	reader := &observedReader{Reader: source}
	_, err := io.Copy(destination, reader)
	result <- copyResult{err: err, sourceErr: reader.err}
}

func bridge(ctx context.Context, conn, mocked net.Conn, conversation <-chan error) error {
	incoming := make(chan copyResult, 1)
	outgoing := make(chan copyResult, 1)
	go copyConnection(mocked, conn, incoming)
	go copyConnection(conn, mocked, outgoing)
	canceled := make(chan struct{})
	stopCancel := context.AfterFunc(ctx, func() { _ = conn.Close(); _ = mocked.Close(); close(canceled) })
	defer func() {
		if !stopCancel() {
			<-canceled
		}
	}()
	var inResult, outResult copyResult
	var result error
	var inDone, outDone bool
	select {
	case result = <-conversation:
	case inResult = <-incoming:
		inDone = true
		if inResult.sourceErr == nil && (errors.Is(inResult.err, io.ErrClosedPipe) || errors.Is(inResult.err, net.ErrClosed)) {
			select {
			case result = <-conversation:
			case <-ctx.Done():
				result = ctx.Err()
			}
		} else {
			select {
			case result = <-conversation:
			default:
				result = errors.Join(io.ErrUnexpectedEOF, inResult.err)
			}
		}
	case outResult = <-outgoing:
		outDone = true
		if outResult.err != nil && !errors.Is(outResult.sourceErr, io.ErrClosedPipe) && !errors.Is(outResult.sourceErr, net.ErrClosed) {
			result = fmt.Errorf("write response: %w", outResult.err)
			break
		}
		// Closing the mock ends its read before the conversation closes ErrorChan.
		select {
		case result = <-conversation:
		case <-ctx.Done():
			result = ctx.Err()
		}
	case <-ctx.Done():
		result = ctx.Err()
	}
	_ = mocked.Close()
	if result != nil {
		_ = conn.Close()
	}
	// Successful completion drains the last response before closing the socket.
	if !outDone {
		outResult = <-outgoing
	}
	_ = conn.Close()
	if !inDone {
		inResult = <-incoming
	}
	for conversationErr := range conversation {
		result = errors.Join(result, conversationErr)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if inResult.sourceErr != nil && !errors.Is(inResult.sourceErr, io.EOF) &&
		!errors.Is(inResult.sourceErr, io.ErrClosedPipe) && !errors.Is(inResult.sourceErr, net.ErrClosed) &&
		!errors.Is(result, inResult.sourceErr) {
		result = errors.Join(result, fmt.Errorf("read request: %w", inResult.sourceErr))
	}
	if result != nil {
		return result
	}
	if outResult.err != nil && !errors.Is(outResult.sourceErr, io.ErrClosedPipe) && !errors.Is(outResult.sourceErr, net.ErrClosed) {
		return fmt.Errorf("write response: %w", outResult.err)
	}
	return nil
}
