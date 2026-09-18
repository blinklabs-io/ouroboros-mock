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
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// closeRecorder wraps a net.Conn to record that Close was reached and, when
// err is set, to fail the call without touching the wrapped connection.
type closeRecorder struct {
	net.Conn
	err    error
	closed atomic.Bool
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
