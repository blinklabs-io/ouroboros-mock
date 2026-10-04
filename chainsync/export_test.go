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

package chainsync

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/muxer"
	gchainsync "github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/stretchr/testify/require"
)

// VerifyConcurrentOversizedSends exercises the production fragment writer at
// the mux transport boundary, where concurrent records carry no agency state.
func VerifyConcurrentOversizedSends(t *testing.T) {
	t.Helper()
	const numPoints = 2000
	const senders = 4
	driverConn, receiverConn := net.Pipe()
	driver := muxer.New(driverConn)
	receiver := muxer.New(receiverConn)
	_, segments, _ := receiver.RegisterProtocol(
		gchainsync.ProtocolIdNtC, muxer.ProtocolRoleResponder,
	)
	driver.Start()
	receiver.Start()
	defer func() {
		driver.Stop()
		receiver.Stop()
		for range driver.ErrorChan() {
		}
		for range receiver.ErrorChan() {
		}
	}()
	harness := &Harness{muxer: driver}
	points := make([]pcommon.Point, numPoints)
	for i := range points {
		hash := make([]byte, 32)
		binary.BigEndian.PutUint64(hash, uint64(i))
		points[i] = pcommon.NewPoint(uint64(i), hash)
	}
	start := make(chan struct{})
	errors := make(chan error, senders)
	for range senders {
		go func() {
			<-start
			errors <- harness.sendSegment(
				gchainsync.ProtocolIdNtC, false,
				gchainsync.NewMsgFindIntersect(points),
			)
		}()
	}
	close(start)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var buffered []byte
	decoded := 0
	for decoded < senders {
		select {
		case segment, ok := <-segments:
			require.True(t, ok, "receiver closed before all messages")
			buffered = append(buffered, segment.Payload...)
			for len(buffered) > 0 {
				var raw cbor.RawMessage
				consumed, err := cbor.Decode(buffered, &raw)
				if err != nil {
					break
				}
				message, err := gchainsync.NewMsgFromCborNtC(
					gchainsync.MessageTypeFindIntersect, raw,
				)
				require.NoError(t, err)
				request, ok := message.(*gchainsync.MsgFindIntersect)
				require.True(t, ok)
				require.Equal(t, points, request.Points)
				buffered = buffered[consumed:]
				decoded++
			}
		case <-ctx.Done():
			t.Fatalf(
				"decoded %d of %d complete messages: %v",
				decoded, senders, ctx.Err(),
			)
		}
	}
	require.Empty(t, buffered)
	for range senders {
		require.NoError(t, <-errors)
	}
}
