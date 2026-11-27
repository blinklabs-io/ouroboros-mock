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
	"context"
	"errors"
	"testing"
	"time"

	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
	"github.com/stretchr/testify/require"
)

func TestObservationFinalizationPreservesQueuedSourceTimes(t *testing.T) {
	s := newEventStream()
	t.Cleanup(s.close)
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for range 32 {
		s.publish(Event{Kind: EventRequestNext, OccurredAt: at})
	}
	result := make(chan error, 1)
	go func() {
		stats, err := s.finish(t.Context())
		if err == nil && !stats.Complete() {
			err = errors.New("incomplete final stats")
		}
		result <- err
	}()
	var count uint64
	for e := range s.out {
		count++
		require.Equal(t, count, e.Sequence)
		require.Equal(t, at, e.OccurredAt)
	}
	require.NoError(t, <-result)
	require.Equal(t, uint64(32), count)
	require.Equal(t, ObservationStats{Produced: 32, Delivered: 32, Finalized: true}, s.stats())
}

func TestObservationFinalizationAccountsForOverflowAndTail(t *testing.T) {
	s := newEventStream()
	for range eventQueueCapacity + 1 {
		s.publish(Event{Kind: EventRequestNext})
	}
	drained := make(chan struct{})
	go func() {
		for range s.out {
		}
		close(drained)
	}()
	stats, err := s.finish(t.Context())
	<-drained
	require.ErrorIs(t, err, ErrIncompleteObservations)
	require.Equal(t, uint64(eventQueueCapacity+1), stats.Produced)
	require.Equal(t, uint64(eventQueueCapacity), stats.Delivered)
	require.Equal(t, uint64(1), stats.Dropped)
	require.True(t, stats.Finalized)
}

func TestObservationFinalizationCancellationDoesNotCertifyLostTail(t *testing.T) {
	s := newEventStream()
	s.publish(Event{Kind: EventRequestNext})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	stats, err := s.finish(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, err, ErrIncompleteObservations)
	require.False(t, stats.Complete())
	require.Equal(t, uint64(1), stats.Discarded)
	_, ok := <-s.out
	require.False(t, ok)
}

func TestOrdinaryCloseStillDiscardsPendingObservations(t *testing.T) {
	s := newEventStream()
	s.publish(Event{Kind: EventRequestNext})
	s.close()
	require.Equal(t, uint64(1), s.stats().Discarded)
	stats, err := s.finish(t.Context())
	require.ErrorIs(t, err, ErrIncompleteObservations)
	require.False(t, stats.Complete())
}

func TestPublishedObservationsOwnMutablePayloads(t *testing.T) {
	s := newEventStream()
	defer s.close()
	hash := make([]byte, 32)
	hash[0] = 1
	tx := Tx{Raw: []byte{1, 2, 3}}
	e := Event{Kind: EventBlockFetched, Parent: hash, Points: []pcommon.Point{{Slot: 1, Hash: hash}}, TxIds: [][32]byte{{1}}, Txs: []Tx{tx}}
	s.publish(e)
	hash[0] = 9
	e.TxIds[0][0] = 9
	tx.Raw[0] = 9
	e.Points[0].Slot = 9
	got := <-s.out
	require.Equal(t, byte(1), got.Parent[0])
	require.Equal(t, byte(1), got.Points[0].Hash[0])
	require.Equal(t, uint64(1), got.Points[0].Slot)
	require.Equal(t, byte(1), got.TxIds[0][0])
	require.Equal(t, byte(1), got.Txs[0].Raw[0])
}

func TestTransactionPullAndPreparedReplyRemainDistinct(t *testing.T) {
	var offered, unknown [32]byte
	offered[0] = 1
	unknown[0] = 2
	p := &TxPeer{events: newEventStream(), txs: []Tx{{EraId: 6, ID: offered, Raw: []byte{1}}}, announced: 1}
	defer p.events.close()
	bodies, err := p.requestTxs(txsubmission.CallbackContext{}, []txsubmission.TxId{{EraId: 6, TxId: offered}, {EraId: 6, TxId: unknown}})
	require.NoError(t, err)
	require.Len(t, bodies, 1)
	pull := <-p.Events()
	reply := <-p.Events()
	require.Equal(t, EventTxsRequested, pull.Kind)
	require.Equal(t, uint64(2), pull.Count)
	require.Len(t, pull.TxIds, 2)
	require.Equal(t, EventTxsReturned, reply.Kind)
	require.Equal(t, uint64(1), reply.Count)
	require.Equal(t, [][32]byte{offered}, reply.TxIds)
	require.NotZero(t, pull.Request)
	require.Equal(t, pull.Request, reply.Request)
	require.False(t, reply.OccurredAt.Before(pull.OccurredAt))
}
