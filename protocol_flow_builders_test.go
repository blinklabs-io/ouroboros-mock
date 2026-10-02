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
	"bytes"
	"net"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	"github.com/blinklabs-io/gouroboros/protocol/keepalive"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/gouroboros/protocol/localtxmonitor"
	"github.com/blinklabs-io/gouroboros/protocol/peersharing"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
	"github.com/stretchr/testify/require"
)

func TestProtocolFlowBuildersPreserveWireFields(t *testing.T) {
	point := NewPoint(42, bytes.Repeat([]byte{0x11}, 32))
	tip := NewTip(point, 7)
	result, err := LocalStateQueryResult([]any{uint64(9), uint64(8)})
	require.NoError(t, err)
	blockResponse, err := BlockFetchBlockResponse(7, []byte{0x80})
	require.NoError(t, err)
	tests := []struct {
		name    string
		entry   ConversationEntry
		id      uint16
		wire    []any
		decoder protocol.MessageFromCborFunc
	}{
		{"NtN await", ChainSyncAwaitReply(false), 2, []any{1}, chainsync.NewMsgFromCborNtN},
		{"NtC await", ChainSyncAwaitReply(true), 5, []any{1}, chainsync.NewMsgFromCborNtC},
		{
			"NtN intersect",
			ChainSyncIntersectFound(false, point, tip),
			2,
			[]any{5, point, tip},
			chainsync.NewMsgFromCborNtN,
		},
		{
			"NtC intersect",
			ChainSyncIntersectFound(true, point, tip),
			5,
			[]any{5, point, tip},
			chainsync.NewMsgFromCborNtC,
		},
		{
			"NtN no intersection",
			ChainSyncIntersectNotFound(false, tip),
			2,
			[]any{6, tip},
			chainsync.NewMsgFromCborNtN,
		},
		{
			"NtC no intersection",
			ChainSyncIntersectNotFound(true, tip),
			5,
			[]any{6, tip},
			chainsync.NewMsgFromCborNtC,
		},
		{"NtN done", ChainSyncDone(false), 2, []any{7}, chainsync.NewMsgFromCborNtN},
		{"NtC done", ChainSyncDone(true), 5, []any{7}, chainsync.NewMsgFromCborNtC},
		{"batch start", BlockFetchStartBatch(), 3, []any{2}, blockfetch.NewMsgFromCbor},
		{
			"single block",
			blockResponse,
			3,
			[]any{4, cbor.Tag{Number: 24, Content: []byte{0x82, 0x07, 0x80}}},
			blockfetch.NewMsgFromCbor,
		},
		{"batch done", BlockFetchBatchDone(), 3, []any{5}, blockfetch.NewMsgFromCbor},
		{"block fetch done", BlockFetchClientDone(), 3, []any{1}, blockfetch.NewMsgFromCbor},
		{"tx submission done", TxSubmissionDone(), 4, []any{4}, txsubmission.NewMsgFromCbor},
		{
			"mempool acquired",
			LocalTxMonitorAcquired(456),
			9,
			[]any{2, 456},
			localtxmonitor.NewMsgFromCbor,
		},
		{
			"mempool next",
			LocalTxMonitorReplyNextTx(6, []byte{0x80}),
			9,
			[]any{6, []any{6, cbor.Tag{Number: 24, Content: []byte{0x80}}}},
			localtxmonitor.NewMsgFromCbor,
		},
		{
			"mempool exhausted",
			LocalTxMonitorReplyNextTx(0, nil),
			9,
			[]any{6},
			localtxmonitor.NewMsgFromCbor,
		},
		{
			"mempool has",
			LocalTxMonitorReplyHasTx(true),
			9,
			[]any{8, true},
			localtxmonitor.NewMsgFromCbor,
		},
		{
			"mempool sizes",
			LocalTxMonitorReplyGetSizes(100, 20, 3),
			9,
			[]any{10, []any{100, 20, 3}},
			localtxmonitor.NewMsgFromCbor,
		},
		{"mempool done", LocalTxMonitorDone(), 9, []any{0}, localtxmonitor.NewMsgFromCbor},
		{
			"query volatile",
			LocalStateQueryAcquireVolatileTip(),
			7,
			[]any{8},
			localstatequery.NewMsgFromCbor,
		},
		{
			"query immutable",
			LocalStateQueryAcquireImmutableTip(),
			7,
			[]any{10},
			localstatequery.NewMsgFromCbor,
		},
		{
			"query reacquire volatile",
			LocalStateQueryReAcquireVolatileTip(),
			7,
			[]any{9},
			localstatequery.NewMsgFromCbor,
		},
		{
			"query reacquire immutable",
			LocalStateQueryReAcquireImmutableTip(),
			7,
			[]any{11},
			localstatequery.NewMsgFromCbor,
		},
		{"query acquired", LocalStateQueryAcquired(), 7, []any{1}, localstatequery.NewMsgFromCbor},
		{
			"query failure",
			LocalStateQueryFailure(1),
			7,
			[]any{2, 1},
			localstatequery.NewMsgFromCbor,
		},
		{"query result", result, 7, []any{4, []any{9, 8}}, localstatequery.NewMsgFromCbor},
		{"query done", LocalStateQueryDone(), 7, []any{7}, localstatequery.NewMsgFromCbor},
		{"peer request", PeerSharingRequest(5), 10, []any{0, 5}, peersharing.NewMsgFromCbor},
		{
			"IPv4 peers",
			PeerSharingPeers([]peersharing.PeerAddress{{IP: net.IP{1, 2, 3, 4}, Port: 3001}}),
			10,
			[]any{1, []any{[]any{0, uint32(0x04030201), 3001}}},
			peersharing.NewMsgFromCbor,
		},
		{
			"IPv6 peers",
			PeerSharingPeers(
				[]peersharing.PeerAddress{{IP: net.ParseIP("2001:db8::1"), Port: 3002}},
			),
			10,
			[]any{1, []any{[]any{1, uint32(0xb80d0120), 0, 0, uint32(0x01000000), 3002}}},
			peersharing.NewMsgFromCbor,
		},
		{"no peers", PeerSharingPeers(nil), 10, []any{1, []any{}}, peersharing.NewMsgFromCbor},
		{"peer done", PeerSharingDone(), 10, []any{2}, peersharing.NewMsgFromCbor},
		{"keepalive request", KeepAliveRequest(123), 8, []any{0, 123}, keepalive.NewMsgFromCbor},
		{"keepalive response", KeepAliveResponse(123), 8, []any{1, 123}, keepalive.NewMsgFromCbor},
		{"keepalive done", KeepAliveDone(), 8, []any{2}, keepalive.NewMsgFromCbor},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var message protocol.Message
			switch entry := test.entry.(type) {
			case ConversationEntryInput:
				require.Equal(t, test.id, entry.ProtocolId)
				require.False(t, entry.IsResponse)
				assertMessageRoundTrips(t, entry)
				message = entry.Message
			case ConversationEntryOutput:
				require.Equal(t, test.id, entry.ProtocolId)
				require.True(t, entry.IsResponse)
				require.Len(t, entry.Messages, 1)
				message = entry.Messages[0]
			default:
				t.Fatalf("unexpected entry %T", entry)
			}
			encoded, err := cbor.Encode(message)
			require.NoError(t, err)
			expected, err := cbor.Encode(test.wire)
			require.NoError(t, err)
			require.Equal(t, expected, encoded)
			typ, err := cbor.DecodeIdFromList(encoded)
			require.NoError(t, err)
			decoded, err := test.decoder(uint(typ), encoded)
			require.NoError(t, err)
			require.NotNil(t, decoded)
			decoded.SetCbor(nil)
			reencoded, err := cbor.Encode(decoded)
			require.NoError(t, err)
			require.Equal(t, expected, reencoded)
		})
	}
}

func TestLocalStateQueryResultRejectsUnencodableValue(t *testing.T) {
	_, err := LocalStateQueryResult(func() {})
	require.Error(t, err)
}

func TestBlockFetchBuildersRejectMalformedRawBlocks(t *testing.T) {
	for _, data := range [][]byte{nil, {}, {0xff}, {0x80, 0x80}} {
		_, err := BlockFetchBlockResponse(7, data)
		require.Error(t, err)
		entries, err := BlockFetchScenario(OriginPoint(), OriginPoint(), []uint{7}, [][]byte{data})
		require.Error(t, err)
		require.Nil(t, entries)
	}
}
