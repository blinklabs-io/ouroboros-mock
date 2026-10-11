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

func TestUpstreamRetentionKeepsBoundedHistoryAndActiveSession(t *testing.T) {
	t.Parallel()
	u := &Upstream{sessions: make(map[uint64]*session)}
	count := maxRetiredRelayCount + 3
	for id := 1; id <= count; id++ {
		r := &relay{session: uint64(id)}
		r.retain([]Tx{retainedTx(byte(id), 1)})
		u.relays = append(u.relays, r)
		u.sessions[uint64(id)] = &session{id: uint64(id), relay: r}
	}

	for id := 1; id < count; id++ {
		u.removeSession(uint64(id))
	}

	got := u.RelayedTxs()
	require.Len(t, got, maxRetiredRelayCount+1)
	require.Equal(t, byte(3), got[0].ID[0])
	require.Equal(t, byte(count), got[len(got)-1].ID[0])
}
