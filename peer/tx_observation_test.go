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

package peer_test

import (
	"errors"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/ouroboros-mock/peer"
	"github.com/stretchr/testify/require"
)

func TestRelayCorrelatesFailedBodyRequests(t *testing.T) {
	up, err := peer.NewUpstream(peer.UpstreamConfig{})
	require.NoError(t, err)
	defer up.Close()
	tx := peer.Tx{EraId: 6, ID: [32]byte{1}, Raw: []byte{0x80}}
	p, err := peer.NewTxPeer(peer.TxPeerConfig{Conn: up.Pipe(), Txs: []peer.Tx{tx}})
	require.NoError(t, err)
	defer p.Close()
	var e peer.Event
	for {
		select {
		case e = <-up.Events():
			require.NotEqual(t, peer.EventSessionClosed, e.Kind, "body failure must retain request correlation before session closure")
			if e.Kind != peer.EventTxReceiveFailed {
				continue
			}
		case <-time.After(waitTimeout):
			t.Fatal("body failure observation missing")
		}
		break
	}
	require.True(t, errors.Is(e.Err, protocol.ErrProtocolShuttingDown) || errors.Is(e.Err, protocol.ErrProtocolViolationInvalidMessage), "unexpected body-request failure: %v", e.Err)
	require.Equal(t, [][32]byte{tx.ID}, e.TxIds)
	require.Equal(t, uint64(1), e.Count)
	require.Equal(t, []peer.TxRef{{EraId: tx.EraId, ID: tx.ID}}, e.TxRefs)
	require.NotZero(t, e.Request)
	require.NotZero(t, e.Session)
	require.NotEmpty(t, e.Connection)
}
