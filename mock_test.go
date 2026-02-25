// Copyright 2024 Blink Labs Software
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

package ouroboros_mock_test

import (
	"fmt"
	"testing"
	"time"

	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"go.uber.org/goleak"
)

// Basic test of conversation mock functionality
func TestBasic(t *testing.T) {
	defer goleak.VerifyNone(t)
	mockConn := ouroboros_mock.NewConnection(
		ouroboros_mock.ProtocolRoleClient,
		[]ouroboros_mock.ConversationEntry{
			ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
			ouroboros_mock.ConversationEntryHandshakeNtCResponse,
		},
	)
	// Async mock connection error handler
	go func() {
		err, ok := <-mockConn.(*ouroboros_mock.Connection).ErrorChan()
		if ok {
			panic(err)
		}
	}()
	oConn, err := ouroboros.New(
		ouroboros.WithConnection(mockConn),
		ouroboros.WithNetworkMagic(ouroboros_mock.MockNetworkMagic),
	)
	if err != nil {
		t.Fatalf("unexpected error when creating Ouroboros object: %s", err)
	}
	// Close Ouroboros connection
	if err := oConn.Close(); err != nil {
		t.Fatalf("unexpected error when closing Ouroboros object: %s", err)
	}
	// Wait for connection shutdown
	select {
	case <-oConn.ErrorChan():
	case <-time.After(10 * time.Second):
		t.Errorf("did not shutdown within timeout")
	}
}

func TestError(t *testing.T) {
	defer goleak.VerifyNone(t)
	mockConn := ouroboros_mock.NewConnection(
		ouroboros_mock.ProtocolRoleClient,
		[]ouroboros_mock.ConversationEntry{
			ouroboros_mock.ConversationEntryInput{
				ProtocolId:    999,
				ExpectedError: "input muxer segment protocol ID did not match expected value: expected 999, got 0",
			},
		},
	)
	// Async mock connection error handler
	asyncErrChan := make(chan error, 1)
	go func() {
		select {
		case err, ok := <-mockConn.(*ouroboros_mock.Connection).ErrorChan():
			if !ok {
				// channel closed, no error
				close(asyncErrChan)
				return
			}
			asyncErrChan <- fmt.Errorf("received unexpected error: %v", err)
		case <-time.After(1 * time.Second):
			// no error received within timeout
			close(asyncErrChan)
		}
	}()
	_, err := ouroboros.New(
		ouroboros.WithConnection(mockConn),
		ouroboros.WithNetworkMagic(ouroboros_mock.MockNetworkMagic),
	)
	if err == nil {
		t.Fatalf("did not receive expected error")
	}
	// Wait for mock connection shutdown
	select {
	case err, ok := <-asyncErrChan:
		if ok {
			t.Fatal(err.Error())
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("did not complete within timeout")
	}
}
