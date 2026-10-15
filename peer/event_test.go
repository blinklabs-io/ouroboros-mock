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
	"sync"
	"testing"
	"time"

	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/stretchr/testify/require"
)

func TestEventStreamBoundsPendingEvents(t *testing.T) {
	s := newEventStream()
	t.Cleanup(s.close)

	for i := range eventQueueCapacity {
		s.publish(Event{Kind: EventRequestNext, Session: uint64(i)})
	}

	published := make(chan struct{})
	go func() {
		s.publish(Event{Kind: EventAwaitReply, Session: eventQueueCapacity})
		close(published)
	}()
	select {
	case <-published:
	case <-time.After(time.Second):
		t.Fatal("full event queue blocked publication")
	}
	require.Equal(t, uint64(1), s.droppedEvents())
	for i := range eventQueueCapacity {
		select {
		case e := <-s.out:
			require.Equal(t, uint64(i), e.Session)
		case <-time.After(time.Second):
			t.Fatal("queued event was not delivered")
		}
	}
	require.Eventually(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return len(s.queue) == 0
	}, time.Second, time.Millisecond)
	s.publish(Event{Kind: EventAwaitReply, Session: eventQueueCapacity + 1})
	select {
	case e := <-s.out:
		require.Equal(t, uint64(eventQueueCapacity+1), e.Session)
	case <-time.After(time.Second):
		t.Fatal("publication did not resume after draining the queue")
	}
	require.Equal(t, uint64(1), s.droppedEvents())
}

func TestUpstreamClosesWithFullEventQueue(t *testing.T) {
	u, err := NewUpstream(UpstreamConfig{})
	require.NoError(t, err)
	for range eventQueueCapacity {
		u.events.publish(Event{Kind: EventRequestNext})
	}

	u.wg.Add(1)
	go func() {
		defer u.wg.Done()
		u.events.publish(Event{Kind: EventSessionClosed})
	}()

	require.NoError(t, u.Close())
}

func TestEventStreamConcurrentPublishAndClose(t *testing.T) {
	s := newEventStream()
	start := make(chan struct{})
	var publishers sync.WaitGroup
	for range 32 {
		publishers.Add(1)
		go func() {
			defer publishers.Done()
			<-start
			for range 1024 {
				s.publish(Event{Kind: EventRequestNext})
			}
		}()
	}
	close(start)
	s.close()

	finished := make(chan struct{})
	go func() {
		publishers.Wait()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("publishers remained blocked after close")
	}
}

func TestEventStreamPublishAfterClose(t *testing.T) {
	s := newEventStream()
	s.close()
	s.publish(Event{Kind: EventAwaitReply})
	_, ok := <-s.out
	require.False(t, ok)
	require.Zero(t, s.droppedEvents())
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Empty(t, s.queue)
}

func TestEventStreamCloseDiscardsQueuedEvents(t *testing.T) {
	s := newEventStream()
	s.publish(Event{Kind: EventRequestNext, Points: make([]pcommon.Point, 1)})

	s.close()

	s.mu.Lock()
	defer s.mu.Unlock()
	require.Empty(t, s.queue)
}
