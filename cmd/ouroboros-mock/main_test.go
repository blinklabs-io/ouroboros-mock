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
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/handshake"
	"github.com/blinklabs-io/gouroboros/protocol/keepalive"
	mock "github.com/blinklabs-io/ouroboros-mock"
	"github.com/stretchr/testify/require"
)

const demoConfig = `listener:
  network: tcp
  address: 127.0.0.1:0
entries:
  - input:
      type: handshake.propose_versions
  - output:
      type: handshake.accept_version
      mode: node-to-node
      version: 13
      network-magic: 42
  - input:
      type: keepalive.request
      cookie: 123
  - output:
      type: keepalive.response
      cookie: 123
  - close: true
`

func TestRunRejectsEmptyConfigurationBeforeListening(t *testing.T) {
	for _, text := range []string{"", "# comment only\n", "  \n"} {
		t.Run(text, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "empty.yaml")
			require.NoError(t, os.WriteFile(path, []byte(text), 0o600))
			var output strings.Builder
			err := run(context.Background(), []string{path}, &output)
			require.EqualError(t, err, "configuration is empty")
			require.Empty(t, output.String())
		})
	}
}

type closeReleasedReaderConn struct {
	net.Conn
	closed chan struct{}
	once   sync.Once
	err    error
}

func (c *closeReleasedReaderConn) Read([]byte) (int, error) {
	<-c.closed
	return 0, c.err
}

func (c *closeReleasedReaderConn) Close() error {
	c.once.Do(func() {
		_ = c.Conn.Close()
		close(c.closed)
	})
	return nil
}

func TestBridgePreservesReadFailureAfterConversationCompletion(t *testing.T) {
	failure := errors.New("client read failed after close")
	for _, readErr := range []error{failure, io.EOF, io.ErrClosedPipe, net.ErrClosed} {
		t.Run(readErr.Error(), func(t *testing.T) {
			server, client := net.Pipe()
			conn := &closeReleasedReaderConn{Conn: server, closed: make(chan struct{}), err: readErr}
			mocked := &closeReleasedReaderConn{Conn: client, closed: make(chan struct{}), err: net.ErrClosed}
			defer conn.Close()
			defer mocked.Close()
			conversation := make(chan error)
			close(conversation)
			result := make(chan error, 1)
			go func() {
				result <- bridge(
					context.Background(),
					conn,
					mocked,
					conversation,
					handshake.ConfirmTimeout,
				)
			}()
			select {
			case err := <-result:
				if errors.Is(readErr, failure) {
					require.ErrorIs(t, err, failure)
				} else {
					require.NoError(t, err)
				}
			case <-time.After(time.Second):
				_ = conn.Close()
				_ = mocked.Close()
				<-result
				t.Fatal("bridge did not join completed copies")
			}
		})
	}
}

func TestConfigurationRejectsUnsupportedAndAmbiguousEntries(t *testing.T) {
	cases := []string{
		strings.ReplaceAll(demoConfig, "network: tcp", "network: udp"),
		strings.ReplaceAll(demoConfig, "cookie: 123", "cookie: 65536"),
		strings.ReplaceAll(demoConfig, "cookie: 123", "cookie: 123.5"),
		strings.ReplaceAll(demoConfig, "cookie: 123", "cookie: -1"),
		strings.ReplaceAll(demoConfig, "version: 13", "version: 13.5"),
		strings.ReplaceAll(demoConfig, "network-magic: 42", "network-magic: 42.5"),
		strings.ReplaceAll(demoConfig, "network-magic: 42", "network-magic: 4294967296"),
		strings.ReplaceAll(demoConfig, "close: true", "close: yes"),
		strings.ReplaceAll(demoConfig, "address: 127.0.0.1:0", "address: true"),
		strings.ReplaceAll(demoConfig, "address: 127.0.0.1:0", "address: 123"),
		strings.ReplaceAll(demoConfig, "handshake.propose_versions", "chainsync.request_next"),
		strings.ReplaceAll(demoConfig, "version: 13", "version: 12"),
		strings.ReplaceAll(demoConfig, "close: true", "close: false"),
		strings.ReplaceAll(demoConfig, "close: true", "sleep: -1s"),
		strings.ReplaceAll(demoConfig, "close: true", "sleep: nonsense"),
		strings.ReplaceAll(demoConfig, "close: true", "close: true\n    sleep: 1s"),
		strings.ReplaceAll(demoConfig, "cookie: 123", "cookie: 123\n      unrelated: true"),
		strings.ReplaceAll(
			demoConfig,
			"type: handshake.propose_versions",
			"type: handshake.propose_versions\n      cookie: 1",
		),
		strings.ReplaceAll(demoConfig, "close: true", "close: true\n  - sleep: 1s"),
		demoConfig + "---\nentries: []\n",
	}
	for _, text := range cases {
		t.Run(
			text,
			func(t *testing.T) { _, _, err := loadConfiguration(strings.NewReader(text)); require.Error(t, err) },
		)
	}
}

func TestTypedConfiguration(t *testing.T) {
	_, entries, err := loadConfiguration(strings.NewReader(demoConfig))
	require.NoError(t, err)
	require.Len(t, entries, 5)
	output, ok := entries[1].(mock.ConversationEntryOutput)
	require.True(t, ok)
	require.Equal(t, uint16(handshake.ProtocolId), output.ProtocolId)
	require.True(t, output.IsResponse)
	require.Len(t, output.Messages, 1)
	expected := handshake.NewMsgAcceptVersion(
		13,
		protocol.VersionDataNtN13andUp{
			VersionDataNtN11to12: protocol.VersionDataNtN11to12{
				CborNetworkMagic:                       42,
				CborInitiatorAndResponderDiffusionMode: protocol.DiffusionModeInitiatorOnly,
				CborPeerSharing:                        protocol.PeerSharingModeNoPeerSharing,
				CborQuery:                              protocol.QueryModeDisabled,
			},
		},
	)
	expectedBytes, err := cbor.Encode(expected)
	require.NoError(t, err)
	actualBytes, err := cbor.Encode(output.Messages[0])
	require.NoError(t, err)
	require.Equal(t, expectedBytes, actualBytes)
	input, ok := entries[2].(mock.ConversationEntryInput)
	require.True(t, ok)
	require.Equal(t, keepalive.NewMsgKeepAlive(123), input.Message)
	ntc := strings.ReplaceAll(
		strings.ReplaceAll(demoConfig, "mode: node-to-node", "mode: node-to-client"),
		"version: 13",
		"version: 14",
	)
	_, entries, err = loadConfiguration(strings.NewReader(ntc))
	require.NoError(t, err)
	output, ok = entries[1].(mock.ConversationEntryOutput)
	require.True(t, ok)
	expectedBytes, err = cbor.Encode(
		handshake.NewMsgAcceptVersion(
			14+protocol.ProtocolVersionNtCOffset,
			protocol.VersionDataNtC9to14(42),
		),
	)
	require.NoError(t, err)
	actualBytes, err = cbor.Encode(output.Messages[0])
	require.NoError(t, err)
	require.Equal(t, expectedBytes, actualBytes)
}

func TestTCPConversationDrainsFinalResponse(t *testing.T) {
	_, entries, err := loadConfiguration(strings.NewReader(demoConfig))
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- serve(ctx, listener, entries) }()
	client, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer client.Close()
	require.NoError(t, client.SetDeadline(time.Now().Add(5*time.Second)))
	sendMessage(
		t,
		client,
		handshake.ProtocolId,
		handshake.NewMsgProposeVersions(
			protocol.ProtocolVersionMap{
				13: protocol.VersionDataNtN13andUp{
					VersionDataNtN11to12: protocol.VersionDataNtN11to12{CborNetworkMagic: 42},
				},
			},
		),
	)
	payload := readMessage(t, client, handshake.ProtocolId)
	decoded, err := handshake.NewMsgFromCbor(handshake.MessageTypeAcceptVersion, payload)
	require.NoError(t, err)
	accepted, ok := decoded.(*handshake.MsgAcceptVersion)
	require.True(t, ok)
	require.Equal(t, uint16(13), accepted.Version)
	sendMessage(t, client, keepalive.ProtocolId, keepalive.NewMsgKeepAlive(123))
	payload = readMessage(t, client, keepalive.ProtocolId)
	decoded, err = keepalive.NewMsgFromCbor(keepalive.MessageTypeKeepAliveResponse, payload)
	require.NoError(t, err)
	require.Equal(
		t,
		keepalive.NewMsgKeepAliveResponse(123).Cookie,
		decoded.(*keepalive.MsgKeepAliveResponse).Cookie,
	)
	var one [1]byte
	_, err = client.Read(one[:])
	require.ErrorIs(t, err, io.EOF)
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("listener did not finish")
	}
}

func TestTCPConversationPreservesResponseAfterCloseWrite(t *testing.T) {
	_, entries, err := loadConfiguration(strings.NewReader(demoConfig))
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	result := make(chan error, 1)
	go func() { result <- serve(context.Background(), listener, entries) }()
	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	client := conn.(*net.TCPConn)
	defer client.Close()
	require.NoError(t, client.SetDeadline(time.Now().Add(5*time.Second)))
	sendMessage(
		t,
		client,
		handshake.ProtocolId,
		handshake.NewMsgProposeVersions(
			protocol.ProtocolVersionMap{
				13: protocol.VersionDataNtN13andUp{
					VersionDataNtN11to12: protocol.VersionDataNtN11to12{CborNetworkMagic: 42},
				},
			},
		),
	)
	_ = readMessage(t, client, handshake.ProtocolId)
	sendMessage(t, client, keepalive.ProtocolId, keepalive.NewMsgKeepAlive(123))
	require.NoError(t, client.CloseWrite())
	payload := readMessage(t, client, keepalive.ProtocolId)
	decoded, err := keepalive.NewMsgFromCbor(keepalive.MessageTypeKeepAliveResponse, payload)
	require.NoError(t, err)
	require.Equal(t, uint16(123), decoded.(*keepalive.MsgKeepAliveResponse).Cookie)
	var one [1]byte
	_, err = client.Read(one[:])
	require.ErrorIs(t, err, io.EOF)
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("listener did not finish after draining the half-closed client response")
	}
}

func TestIdleClientTimesOutBeforeInitialMessageCompletes(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	result := make(chan error, 1)
	go func() {
		result <- serveWithInitialMessageTimeout(
			context.Background(),
			listener,
			[]mock.ConversationEntry{
				mock.ConversationEntryInput{
					ProtocolId:      handshake.ProtocolId,
					MessageType:     handshake.MessageTypeProposeVersions,
					MsgFromCborFunc: handshake.NewMsgFromCbor,
				},
			},
			50*time.Millisecond,
		)
	}()
	client, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer client.Close()
	_, err = client.Write([]byte{0})
	require.NoError(t, err)
	select {
	case err := <-result:
		require.Error(t, err)
		var netErr net.Error
		if errors.As(err, &netErr) {
			require.True(t, netErr.Timeout())
		} else {
			t.Fatalf("initial message failure is not a network timeout: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("partial initial message kept the listener occupied")
	}
}

func TestInitialMessageTimeoutRejectsUndecodedSegments(t *testing.T) {
	for name, payload := range map[string][]byte{
		"zero length":     nil,
		"incomplete CBOR": {0x82},
	} {
		t.Run(name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer listener.Close()
			result := make(chan error, 1)
			go func() {
				result <- serveWithInitialMessageTimeout(
					context.Background(),
					listener,
					[]mock.ConversationEntry{
						mock.ConversationEntryInput{
							ProtocolId:      handshake.ProtocolId,
							MessageType:     handshake.MessageTypeProposeVersions,
							MsgFromCborFunc: handshake.NewMsgFromCbor,
						},
					},
					50*time.Millisecond,
				)
			}()
			client, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer client.Close()
			header := make([]byte, 8)
			binary.BigEndian.PutUint16(header[4:6], handshake.ProtocolId)
			binary.BigEndian.PutUint16(header[6:8], uint16(len(payload)))
			_, err = client.Write(append(header, payload...))
			require.NoError(t, err)
			select {
			case err := <-result:
				require.Error(t, err)
			case <-time.After(time.Second):
				t.Fatal("undecoded initial segment removed the read deadline")
			}
		})
	}
}

type readDeadlineRecorder struct {
	net.Conn
	set chan time.Time
}

func (c *readDeadlineRecorder) SetReadDeadline(deadline time.Time) error {
	c.set <- deadline
	return c.Conn.SetReadDeadline(deadline)
}

func TestInitialMessageDeadlineFollowsAcceptance(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	recorder := &readDeadlineRecorder{Conn: server, set: make(chan time.Time, 2)}
	conn, err := withInitialMessageTimeout(recorder, time.Second)
	require.NoError(t, err)
	require.False(t, (<-recorder.set).IsZero())
	accepted := make(chan struct{})
	conversation := make(chan error)
	close(accepted)
	result := waitForInitialMessage(conn, accepted, conversation)
	select {
	case deadline := <-recorder.set:
		require.True(t, deadline.IsZero())
	case <-time.After(time.Second):
		t.Fatal("accepted initial message did not clear the read deadline")
	}
	close(conversation)
	require.NoError(t, <-result)

	for name, conversationErr := range map[string]error{
		"zero length":     nil,
		"incomplete CBOR": errors.New("decode initial message: unexpected EOF"),
	} {
		t.Run(name, func(t *testing.T) {
			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()
			recorder := &readDeadlineRecorder{Conn: server, set: make(chan time.Time, 2)}
			conn, err := withInitialMessageTimeout(recorder, time.Second)
			require.NoError(t, err)
			require.False(t, (<-recorder.set).IsZero())
			accepted := make(chan struct{})
			conversation := make(chan error, 1)
			if conversationErr != nil {
				conversation <- conversationErr
			}
			close(conversation)
			result := waitForInitialMessage(conn, accepted, conversation)
			require.Error(t, <-result)
			select {
			case <-recorder.set:
				t.Fatal("rejected initial message cleared the read deadline")
			default:
			}
		})
	}
}

func TestInitialMessageTimeoutClearsAfterCompleteSegment(t *testing.T) {
	_, entries, err := loadConfiguration(strings.NewReader(strings.Replace(
		demoConfig,
		"  - output:\n      type: handshake.accept_version",
		"  - sleep: 100ms\n  - output:\n      type: handshake.accept_version",
		1,
	)))
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	result := make(chan error, 1)
	go func() {
		result <- serveWithInitialMessageTimeout(
			context.Background(),
			listener,
			entries,
			25*time.Millisecond,
		)
	}()
	client, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer client.Close()
	require.NoError(t, client.SetDeadline(time.Now().Add(time.Second)))
	sendMessage(
		t,
		client,
		handshake.ProtocolId,
		handshake.NewMsgProposeVersions(
			protocol.ProtocolVersionMap{
				13: protocol.VersionDataNtN13andUp{
					VersionDataNtN11to12: protocol.VersionDataNtN11to12{CborNetworkMagic: 42},
				},
			},
		),
	)
	header := make([]byte, 8)
	_, err = io.ReadFull(client, header)
	require.NoError(t, err, "initial message deadline remained active after a complete segment")
	require.Equal(t, uint16(handshake.ProtocolId|0x8000), binary.BigEndian.Uint16(header[4:6]))
	payload := make([]byte, binary.BigEndian.Uint16(header[6:8]))
	_, err = io.ReadFull(client, payload)
	require.NoError(t, err, "initial message deadline remained active after a complete segment")
	decoded, err := handshake.NewMsgFromCbor(handshake.MessageTypeAcceptVersion, payload)
	require.NoError(t, err)
	accepted, ok := decoded.(*handshake.MsgAcceptVersion)
	require.True(t, ok)
	require.Equal(t, uint16(13), accepted.Version)
	sendMessage(t, client, keepalive.ProtocolId, keepalive.NewMsgKeepAlive(123))
	_ = readMessage(t, client, keepalive.ProtocolId)
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("listener did not finish")
	}
}

func sendMessage(t *testing.T, conn net.Conn, protocolID uint16, message protocol.Message) {
	t.Helper()
	payload, err := cbor.Encode(message)
	require.NoError(t, err)
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[4:6], protocolID)
	require.Less(t, len(payload), 65536)
	binary.BigEndian.PutUint16(header[6:8], uint16(len(payload)))
	_, err = conn.Write(append(header, payload...))
	require.NoError(t, err)
}

func readMessage(t *testing.T, conn net.Conn, protocolID uint16) []byte {
	t.Helper()
	header := make([]byte, 8)
	_, err := io.ReadFull(conn, header)
	require.NoError(t, err)
	require.Equal(t, protocolID|0x8000, binary.BigEndian.Uint16(header[4:6]))
	payload := make([]byte, binary.BigEndian.Uint16(header[6:8]))
	_, err = io.ReadFull(conn, payload)
	require.NoError(t, err)
	return payload
}

func TestCancelListenerAndSleepingConversation(t *testing.T) {
	for _, connect := range []bool{false, true} {
		t.Run(map[bool]string{false: "accept", true: "sleep"}[connect], func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer listener.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				result <- serve(ctx, listener, []mock.ConversationEntry{mock.ConversationEntrySleep{Duration: time.Hour}})
			}()
			if connect {
				conn, err := net.Dial("tcp", listener.Addr().String())
				require.NoError(t, err)
				defer conn.Close()
			}
			cancel()
			select {
			case err := <-result:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(time.Second):
				t.Fatal("cancellation did not join listener and copies")
			}
		})
	}
}

func TestRunPreservesExistingUnixPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "listener.sock")
	require.NoError(t, os.WriteFile(path, []byte("owned"), 0o600))
	config := filepath.Join(t.TempDir(), "config.yaml")
	text := strings.ReplaceAll(
		strings.ReplaceAll(demoConfig, "network: tcp", "network: unix"),
		"127.0.0.1:0",
		path,
	)
	require.NoError(t, os.WriteFile(config, []byte(text), 0o600))
	err := run(context.Background(), []string{config}, io.Discard)
	require.Error(t, err)
	contents, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.Equal(t, "owned", string(contents))
	require.False(t, errors.Is(err, context.Canceled))
}

type failingWriterConn struct {
	net.Conn
	err error
}

func (c failingWriterConn) Write([]byte) (int, error) { return 0, c.err }

func TestBridgeWriteFailureUnwindsConversation(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	failure := errors.New("response writer failed")
	conn := mock.NewConnection(mock.ProtocolRoleClient, []mock.ConversationEntry{
		mock.ConversationEntryOutput{ProtocolId: keepalive.ProtocolId, IsResponse: true, Messages: []protocol.Message{keepalive.NewMsgKeepAliveResponse(123)}},
		mock.ConversationEntryInput{ProtocolId: keepalive.ProtocolId, Message: keepalive.NewMsgKeepAlive(123), MsgFromCborFunc: keepalive.NewMsgFromCbor},
	}).(*mock.Connection)
	defer conn.Close()
	result := make(chan error, 1)
	go func() {
		result <- bridge(
			context.Background(),
			failingWriterConn{Conn: server, err: failure},
			conn,
			conn.ErrorChan(),
			handshake.ConfirmTimeout,
		)
	}()
	select {
	case err := <-result:
		require.ErrorIs(t, err, failure)
	case <-time.After(time.Second):
		_ = conn.Close()
		_ = server.Close()
		<-result
		t.Fatal("failed response write stranded conversation")
	}
	select {
	case <-conn.ErrorChan():
	case <-time.After(time.Second):
		t.Fatal("conversation did not unwind")
	}
}

func TestBridgeBoundsSuccessfulResponseDrain(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	mocked, producer := net.Pipe()
	defer producer.Close()
	conversation := make(chan error)
	result := make(chan error, 1)
	go func() {
		result <- bridge(
			context.Background(),
			server,
			mocked,
			conversation,
			25*time.Millisecond,
		)
	}()
	produced := make(chan error, 1)
	go func() {
		_, err := producer.Write([]byte("response"))
		produced <- err
		close(conversation)
	}()
	select {
	case err := <-produced:
		require.NoError(t, err)
	case <-time.After(time.Second):
		_ = client.Close()
		<-result
		t.Fatal("mock response was not forwarded to the socket writer")
	}
	select {
	case err := <-result:
		require.Error(t, err)
		var netErr net.Error
		if errors.As(err, &netErr) {
			require.True(t, netErr.Timeout())
		} else {
			t.Fatalf("blocked response drain did not return a network timeout: %v", err)
		}
	case <-time.After(time.Second):
		_ = client.Close()
		err := <-result
		t.Fatalf("successful response drain remained blocked: %v", err)
	}
}

type listeningWriter chan string

func (w listeningWriter) Write(
	data []byte,
) (int, error) {
	w <- string(data)
	return len(data), nil
}

func TestRunUnixListenerCleansOwnedSocket(t *testing.T) {
	socketDir, err := os.MkdirTemp("", "om")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(socketDir)) })
	socket := filepath.Join(socketDir, "demo.sock")
	config := filepath.Join(t.TempDir(), "demo.yaml")
	text := "listener:\n  network: unix\n  address: " + socket + "\nentries:\n  - sleep: 1h\n"
	require.NoError(t, os.WriteFile(config, []byte(text), 0o600))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	listening := make(listeningWriter, 1)
	result := make(chan error, 1)
	go func() { result <- run(ctx, []string{config}, listening) }()
	select {
	case line := <-listening:
		require.Contains(t, line, "listening on unix "+socket)
	case err := <-result:
		t.Fatalf("listener failed: %v", err)
	case <-time.After(time.Second):
		t.Fatal("listener did not start")
	}
	client, err := net.Dial("unix", socket)
	require.NoError(t, err)
	defer client.Close()
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("listener did not stop")
	}
	_, err = os.Stat(socket)
	require.ErrorIs(t, err, os.ErrNotExist)
}
