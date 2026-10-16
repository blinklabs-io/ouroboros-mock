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

package ouroboros_mock

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/muxer"
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/keepalive"
	"github.com/stretchr/testify/require"
)

// closeRecorder wraps a net.Conn to record that Close was reached and, when
// err is set, to fail the call without touching the wrapped connection.
type closeRecorder struct {
	net.Conn
	err    error
	closed atomic.Bool
}

type unencodableMessage struct {
	protocol.MessageBase
	Value chan int
}

func (c *closeRecorder) Close() error {
	c.closed.Store(true)
	if c.err != nil {
		return c.err
	}
	return c.Conn.Close()
}

func TestCloseClosesBothHalvesWhenTheClientHalfFails(t *testing.T) {
	conn := NewConnection(ProtocolRoleClient, nil).(*Connection)

	// A nil conversation means asyncLoop returns without reading either
	// half, so swapping them here races nothing.
	wantErr := errors.New("client half refused to close")
	client := &closeRecorder{Conn: conn.conn, err: wantErr}
	mock := &closeRecorder{Conn: conn.mockConn}
	conn.conn, conn.mockConn = client, mock

	err := conn.Close()

	require.ErrorIs(t, err, wantErr)
	require.True(t, mock.closed.Load(),
		"the mock half must still be closed when the client half fails: "+
			"onceClose never runs the body again, so an early return "+
			"strands this half of the pipe permanently")
}

func TestCloseReportsNoErrorOnAHealthyConnection(t *testing.T) {
	conn := NewConnection(ProtocolRoleClient, nil).(*Connection)
	require.NoError(t, conn.Close())
}

func TestPendingInputClassifiesClosedReceive(t *testing.T) {
	entry := ConversationEntryInput{ProtocolId: keepalive.ProtocolId}
	done := make(chan any)
	received := make(chan *muxer.Segment)
	close(done)
	close(received)
	conn := &Connection{
		doneChan:      done,
		muxerRecvChan: received,
		inputBuffers:  make(map[uint16]*bytes.Buffer),
	}
	require.ErrorIs(t, conn.processInputEntry(entry), errConversationClosed)

	remote := &Connection{
		doneChan:      make(chan any),
		muxerRecvChan: make(chan *muxer.Segment),
		inputBuffers:  make(map[uint16]*bytes.Buffer),
	}
	close(remote.muxerRecvChan)
	require.ErrorIs(t, remote.processInputEntry(entry), io.ErrUnexpectedEOF)
}

func TestPendingInputPreservesQueuedSegmentError(t *testing.T) {
	done := make(chan any)
	received := make(chan *muxer.Segment, 1)
	received <- muxer.NewSegment(keepalive.ProtocolId+1, nil, false)
	close(done)
	close(received)
	conn := &Connection{
		doneChan:      done,
		muxerRecvChan: received,
		inputBuffers:  make(map[uint16]*bytes.Buffer),
	}

	require.EqualError(
		t,
		conn.processInputEntry(
			ConversationEntryInput{ProtocolId: keepalive.ProtocolId},
		),
		"input muxer segment protocol ID did not match expected value: expected 8, got 9",
	)
}

func TestClosePendingInputClosesErrorChannelWithoutValue(t *testing.T) {
	conn := NewConnection(ProtocolRoleClient, []ConversationEntry{
		ConversationEntryOutput{
			ProtocolId: keepalive.ProtocolId,
			IsResponse: true,
			Messages: []protocol.Message{
				keepalive.NewMsgKeepAliveResponse(7),
			},
		},
		ConversationEntryInput{ProtocolId: keepalive.ProtocolId},
	}).(*Connection)
	waiting := make(chan struct{})
	conn.hooks.onInputWait = func() { close(waiting) }

	data := make([]byte, 64)
	_, err := conn.Read(data)
	require.NoError(t, err)
	select {
	case <-waiting:
	case <-time.After(time.Second):
		t.Fatal("conversation did not reach the pending input")
	}
	require.NoError(t, conn.Close())
	select {
	case err, ok := <-conn.ErrorChan():
		require.False(t, ok, "shutdown published an input error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("error channel did not close after connection shutdown")
	}
}

func TestClosePendingOutputClosesErrorChannelWithoutValue(t *testing.T) {
	waiting := make(chan struct{})
	conn := newConnection(ProtocolRoleClient, []ConversationEntry{
		ConversationEntryOutput{
			ProtocolId: keepalive.ProtocolId,
			IsResponse: true,
			Messages: []protocol.Message{
				keepalive.NewMsgKeepAliveResponse(7),
			},
		},
	}, connectionHooks{onOutputWait: func() { close(waiting) }})

	select {
	case <-waiting:
	case <-time.After(time.Second):
		t.Fatal("conversation did not reach the pending output")
	}
	require.NoError(t, conn.Close())
	select {
	case err, ok := <-conn.ErrorChan():
		require.False(t, ok, "shutdown published an output error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("error channel did not close after connection shutdown")
	}
}

func TestCloseAfterInputResultPreservesError(t *testing.T) {
	resultReady := make(chan struct{})
	releaseResult := make(chan struct{})
	conn := newConnection(
		ProtocolRoleClient,
		[]ConversationEntry{
			ConversationEntryInput{ProtocolId: keepalive.ProtocolId},
		},
		connectionHooks{onEntryResult: func() {
			close(resultReady)
			<-releaseResult
		}},
	)
	go func() {
		conn.muxerRecvChan <- muxer.NewSegment(keepalive.ProtocolId+1, nil, false)
	}()

	select {
	case <-resultReady:
	case <-time.After(time.Second):
		t.Fatal("conversation did not produce the input result")
	}
	require.NoError(t, conn.Close())
	close(releaseResult)
	require.EqualError(
		t,
		<-conn.ErrorChan(),
		"input error: input muxer segment protocol ID did not match expected value: expected 8, got 9",
	)
}

func TestCloseAfterOutputResultPreservesExpectedError(t *testing.T) {
	resultReady := make(chan struct{})
	releaseResult := make(chan struct{})
	conn := newConnection(
		ProtocolRoleClient,
		[]ConversationEntry{ConversationEntryOutput{
			ProtocolId: keepalive.ProtocolId,
			IsResponse: true,
			Messages: []protocol.Message{
				keepalive.NewMsgKeepAliveResponse(7),
			},
			ExpectedError: "send failed",
		}},
		connectionHooks{onEntryResult: func() {
			close(resultReady)
			<-releaseResult
		}},
	)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		_, _ = conn.Read(make([]byte, 64))
	}()

	select {
	case <-resultReady:
	case <-time.After(time.Second):
		t.Fatal("conversation did not produce the output result")
	}
	require.NoError(t, conn.Close())
	close(releaseResult)
	require.EqualError(
		t,
		<-conn.ErrorChan(),
		`expected error "send failed" but none occurred`,
	)
	<-readDone
}

func TestCloseAfterOutputResultPreservesEncodeError(t *testing.T) {
	resultReady := make(chan struct{})
	releaseResult := make(chan struct{})
	conn := newConnection(
		ProtocolRoleClient,
		[]ConversationEntry{ConversationEntryOutput{
			ProtocolId: keepalive.ProtocolId,
			Messages: []protocol.Message{
				&unencodableMessage{Value: make(chan int)},
			},
		}},
		connectionHooks{onEntryResult: func() {
			close(resultReady)
			<-releaseResult
		}},
	)

	select {
	case <-resultReady:
	case <-time.After(time.Second):
		t.Fatal("conversation did not produce the output result")
	}
	require.NoError(t, conn.Close())
	close(releaseResult)
	err := <-conn.ErrorChan()
	require.Error(t, err)
	require.ErrorContains(t, err, "output error")
	require.ErrorContains(t, err, "unencodableMessage")
}

func TestConnectionErrorDeliveryAndClosure(t *testing.T) {
	conn := &Connection{errorChan: make(chan error, 1)}
	conn.deliverError(errors.New("test error"))
	conn.closeErrorChan()
	select {
	case err := <-conn.ErrorChan():
		require.EqualError(t, err, "test error")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for connection error")
	}
	select {
	case _, ok := <-conn.ErrorChan():
		require.False(t, ok, "error channel should be closed")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error channel closure")
	}
}

func TestConnectionErrorDeliverySurvivesConcurrentClose(t *testing.T) {
	for range 100 {
		conn := NewConnection(ProtocolRoleClient, nil).(*Connection)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			conn.sendError(errors.New("test error"))
		}()
		go func() {
			defer wg.Done()
			_ = conn.Close()
		}()
		wg.Wait()
		conn.closeErrorChan()
	}
}

func TestProcessOutputRejectsOversizedPayload(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	testMuxer := muxer.New(server)
	defer func() {
		testMuxer.Stop()
		for range testMuxer.ErrorChan() {
		}
	}()

	msg := &protocol.MessageBase{}
	msg.SetCbor(make([]byte, muxer.SegmentMaxPayloadLength+1))

	err := (&Connection{muxer: testMuxer}).processOutputEntry(ConversationEntryOutput{
		Messages: []protocol.Message{msg},
	})

	require.EqualError(
		t,
		err,
		"output payload length 65536 exceeds muxer segment limit 65535",
	)
}

func TestCloseInterruptsConversationSleep(t *testing.T) {
	conn := NewConnection(ProtocolRoleClient, []ConversationEntry{
		ConversationEntryOutput{ProtocolId: keepalive.ProtocolId, IsResponse: true, Messages: []protocol.Message{keepalive.NewMsgKeepAliveResponse(7)}},
		ConversationEntrySleep{Duration: time.Hour},
	}).(*Connection)
	// Reading the output lets the conversation proceed to its sleep.
	data := make([]byte, 64)
	_, err := conn.Read(data)
	require.NoError(t, err)
	require.NoError(t, conn.Close())
	select {
	case _, ok := <-conn.ErrorChan():
		require.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("closed conversation still sleeping")
	}
}

func TestConversationSleepHonorsDuration(t *testing.T) {
	started := time.Now()
	conn := NewConnection(ProtocolRoleClient, []ConversationEntry{ConversationEntrySleep{Duration: 50 * time.Millisecond}}).(*Connection)
	defer conn.Close()
	select {
	case _, ok := <-conn.ErrorChan():
		require.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("sleep did not finish")
	}
	require.GreaterOrEqual(t, time.Since(started), 50*time.Millisecond)
}

func TestCanceledConversationSleepReturnsImmediately(t *testing.T) {
	conn := &Connection{doneChan: make(chan any)}
	close(conn.doneChan)
	finished := make(chan struct{})
	go func() { conn.processSleepEntry(ConversationEntrySleep{Duration: 2 * time.Second}); close(finished) }()
	select {
	case <-finished:
	case <-time.After(500 * time.Millisecond):
		<-finished
		t.Fatal("canceled conversation slept for its original duration")
	}
}
