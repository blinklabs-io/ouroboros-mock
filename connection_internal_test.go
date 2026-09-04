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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
