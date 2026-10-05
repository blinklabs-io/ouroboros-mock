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
		s.publish(Event{
			Kind:    EventAwaitReply,
			Session: eventQueueCapacity,
		})
		close(published)
	}()

	select {
	case <-published:
		t.Fatal("publish completed while the event queue was full")
	case <-time.After(100 * time.Millisecond):
	}

	first := <-s.out
	require.Equal(t, uint64(0), first.Session)
	require.Eventually(t, func() bool {
		select {
		case <-published:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	for i := 1; i <= eventQueueCapacity; i++ {
		e := <-s.out
		require.Equal(t, uint64(i), e.Session)
	}
}

func TestUpstreamCloseUnblocksEventPublisher(t *testing.T) {
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

func TestEventStreamCloseUnblocksPublisher(t *testing.T) {
	s := newEventStream()
	for range eventQueueCapacity {
		s.publish(Event{Kind: EventRequestNext})
	}

	published := make(chan struct{})
	go func() {
		s.publish(Event{Kind: EventAwaitReply})
		close(published)
	}()
	t.Cleanup(func() { <-published })

	select {
	case <-published:
		t.Fatal("publish completed while the event queue was full")
	case <-time.After(100 * time.Millisecond):
	}

	s.close()
	require.Eventually(t, func() bool {
		select {
		case <-published:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
}

func TestEventStreamCloseDiscardsQueuedEvents(t *testing.T) {
	s := newEventStream()
	s.publish(Event{Kind: EventRequestNext, Points: make([]pcommon.Point, 1)})

	s.close()

	s.mu.Lock()
	defer s.mu.Unlock()
	require.Empty(t, s.queue)
}
