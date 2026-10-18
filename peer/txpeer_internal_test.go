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

package peer

import (
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
	"github.com/stretchr/testify/require"
)

func TestTxPeerCloseUnblocksBlockingTxIdRequest(t *testing.T) {
	t.Parallel()
	p := &TxPeer{
		events: newEventStream(),
		more:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	errCh := make(chan error, 1)
	go func() {
		_, err := p.requestTxIds(
			txsubmission.CallbackContext{},
			true,
			0,
			1,
		)
		errCh <- err
	}()

	select {
	case event := <-p.Events():
		require.Equal(t, EventTxIdsBlocked, event.Kind)
	case <-time.After(time.Second):
		t.Fatal("blocking transaction ID request did not start")
	}
	require.NoError(t, p.Close())
	select {
	case err := <-errCh:
		require.ErrorIs(t, err, txsubmission.ErrStopServerProcess)
	case <-time.After(time.Second):
		t.Fatal("Close did not unblock transaction ID request")
	}
	require.NoError(t, p.Close())
}

func TestTxPeerRequestTxsMatchesEraAndHash(t *testing.T) {
	t.Parallel()
	var id [32]byte
	id[0] = 1
	p := &TxPeer{
		events: newEventStream(),
		txs: []Tx{
			{EraId: 5, ID: id, Raw: []byte{5}},
			{EraId: 6, ID: id, Raw: []byte{6}},
		},
		announced: 2,
	}
	t.Cleanup(p.events.close)

	got, err := p.requestTxs(
		txsubmission.CallbackContext{},
		[]txsubmission.TxId{{EraId: 6, TxId: id}},
	)

	require.NoError(t, err)
	require.Equal(t, []txsubmission.TxBody{{EraId: 6, TxBody: []byte{6}}}, got)
	require.Equal(t, p.txs[1:], p.served)
}
