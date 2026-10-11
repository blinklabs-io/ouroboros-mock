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
	"container/list"
	"testing"

	"github.com/stretchr/testify/require"
)

func retainedTx(id byte, size int) Tx {
	tx := Tx{Raw: make([]byte, size)}
	tx.ID[0] = id
	return tx
}

func TestRelayRetentionBoundsCountAndEvictsOldest(t *testing.T) {
	t.Parallel()
	r := &relay{}
	for i := 0; i < maxRetainedTxCount+2; i++ {
		r.retain([]Tx{retainedTx(byte(i), 1)})
	}

	got := r.Received()
	require.Len(t, got, maxRetainedTxCount)
	require.Equal(t, byte(2), got[0].ID[0])
	require.Equal(t, byte(1), got[len(got)-1].ID[0])
}

func TestRelayRetentionBoundsPayloadBytes(t *testing.T) {
	t.Parallel()
	r := &relay{}
	txSize := maxRetainedTxBytes/2 + 1
	r.retain([]Tx{
		retainedTx(1, txSize),
		retainedTx(2, txSize),
	})

	require.Equal(t, []Tx{retainedTx(2, txSize)}, r.Received())
	require.LessOrEqual(t, r.bytes, maxRetainedTxBytes)

	r.retain([]Tx{retainedTx(3, maxRetainedTxBytes+1)})
	require.Equal(t, []Tx{retainedTx(2, txSize)}, r.Received())
}

func TestRelayRetentionOwnsTransactionBodies(t *testing.T) {
	t.Parallel()
	r := &relay{}
	delivered := retainedTx(1, 1)
	delivered.Raw[0] = 1
	r.retain([]Tx{delivered})

	// Event delivery uses the same transactions passed to retain.
	event := Event{Kind: EventTxsReceived, Txs: []Tx{delivered}}
	event.Txs[0].Raw[0] = 2
	require.Equal(t, byte(1), r.Received()[0].Raw[0])

	u := &Upstream{relays: list.New()}
	u.relays.PushBack(&session{relay: r})
	relayed := u.RelayedTxs()
	relayed[0].Raw[0] = 3
	require.Equal(t, byte(1), u.RelayedTxs()[0].Raw[0])
}

func TestUpstreamRetentionKeepsRecentClosuresAndActiveSession(t *testing.T) {
	t.Parallel()
	u := &Upstream{
		sessions:      make(map[uint64]*session),
		relays:        list.New(),
		retiredRelays: list.New(),
	}
	count := maxRetiredRelayCount + 3
	for id := 1; id <= count; id++ {
		r := &relay{session: uint64(id)}
		r.retain([]Tx{retainedTx(byte(id), 1)})
		s := &session{id: uint64(id), relay: r}
		s.relayElement = u.relays.PushBack(s)
		u.sessions[uint64(id)] = s
	}

	closeOrder := []int{maxRetiredRelayCount + 1, maxRetiredRelayCount + 2}
	for id := 1; id <= maxRetiredRelayCount; id++ {
		closeOrder = append(closeOrder, id)
	}
	for _, id := range closeOrder {
		u.removeSession(uint64(id))
	}

	got := u.RelayedTxs()
	require.Len(t, got, maxRetiredRelayCount+1)
	require.Len(t, u.sessions, 1)
	require.Equal(t, maxRetiredRelayCount, u.retiredRelays.Len())
	require.Equal(t, maxRetiredRelayCount+1, u.relays.Len())
	for i := 0; i < maxRetiredRelayCount; i++ {
		require.Equal(t, byte(i+1), got[i].ID[0])
	}
	require.Equal(t, byte(count), got[maxRetiredRelayCount].ID[0])
}
