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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/muxer"
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/gouroboros/protocol/keepalive"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/gouroboros/protocol/localtxmonitor"
	"github.com/blinklabs-io/gouroboros/protocol/peersharing"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestProtocolScenariosRunThroughMockConnection(t *testing.T) {
	defer goleak.VerifyNone(t)
	blocks, err := fixtures.GenerateConwayChain(1, common.Blake2b256{}, 1, 1, 2)
	require.NoError(t, err)
	point := NewPoint(blocks[0].SlotNumber(), blocks[0].Hash().Bytes())
	tip := NewTip(
		NewPoint(blocks[1].SlotNumber(), blocks[1].Hash().Bytes()),
		blocks[1].BlockNumber(),
	)
	headers := [][]byte{blocks[0].Header().Cbor(), blocks[1].Header().Cbor()}
	rawBlocks := [][]byte{blocks[0].Cbor(), blocks[1].Cbor()}
	forwardNtN, err := ChainSyncForwardScenarioNtN(conway.EraIdConway, 0, headers, tip)
	require.NoError(t, err)
	forwardNtC, err := ChainSyncForwardScenarioNtC([]uint{7, 7}, rawBlocks, tip)
	require.NoError(t, err)
	intersectionNtN, err := ChainSyncIntersectionScenario(
		false,
		[]pcommon.Point{OriginPoint(), point},
		&point,
		tip,
	)
	require.NoError(t, err)
	intersectionNtC, err := ChainSyncIntersectionScenario(true, []pcommon.Point{point}, &point, tip)
	require.NoError(t, err)
	noIntersection, err := ChainSyncIntersectionScenario(
		true,
		[]pcommon.Point{OriginPoint()},
		nil,
		tip,
	)
	require.NoError(t, err)
	transactionBlocks, err := fixtures.GenerateConwayChainWithTransactions(
		1,
		common.Blake2b256{},
		1,
		1,
		1,
	)
	require.NoError(t, err)
	transaction := transactionBlocks[0].Transactions()[0]
	transactionCBOR := transaction.Cbor()
	ids := []txsubmission.TxIdAndSize{
		{
			TxId: txsubmission.TxId{EraId: 6, TxId: [32]byte(transaction.Hash().Bytes())},
			Size: uint32(len(transactionCBOR)),
		},
	}
	txs := []txsubmission.TxBody{{EraId: 6, TxBody: transactionCBOR}}
	submission, err := TxSubmissionScenario(ids, txs)
	require.NoError(t, err)
	emptySubmission, err := TxSubmissionScenario(nil, nil)
	require.NoError(t, err)
	query, err := LocalStateQueryQuery([]any{localstatequery.QueryTypeChainBlockNo})
	require.NoError(t, err)
	result, err := LocalStateQueryResult([]any{uint64(1), uint64(99)})
	require.NoError(t, err)
	emptyForwardNtN, err := ChainSyncForwardScenarioNtN(conway.EraIdConway, 0, nil, tip)
	require.NoError(t, err)
	emptyForwardNtC, err := ChainSyncForwardScenarioNtC(nil, nil, tip)
	require.NoError(t, err)
	awaitNtN := []ConversationEntry{
		ChainSyncRequestNext(false),
		ChainSyncAwaitReply(false),
		forwardNtN[1],
		ChainSyncDone(false),
	}
	awaitNtC := []ConversationEntry{
		ChainSyncRequestNext(true),
		ChainSyncAwaitReply(true),
		forwardNtC[1],
		ChainSyncDone(true),
	}
	batch, err := BlockFetchScenario(point, tip.Point, []uint{7, 7}, rawBlocks)
	require.NoError(t, err)
	noBlocks, err := BlockFetchScenario(point, tip.Point, nil, nil)
	require.NoError(t, err)
	tests := []struct {
		name    string
		entries []ConversationEntry
		id      uint16
		states  protocol.StateMap
		initial string
		decoder protocol.MessageFromCborFunc
	}{
		{
			"empty forward NtN",
			emptyForwardNtN,
			2,
			chainsync.StateMapNtN,
			"Idle",
			chainsync.NewMsgFromCborNtN,
		},
		{
			"empty forward NtC",
			emptyForwardNtC,
			5,
			chainsync.StateMapNtC,
			"Idle",
			chainsync.NewMsgFromCborNtC,
		},
		{
			"await forward NtN",
			awaitNtN,
			2,
			chainsync.StateMapNtN,
			"Idle",
			chainsync.NewMsgFromCborNtN,
		},
		{
			"await forward NtC",
			awaitNtC,
			5,
			chainsync.StateMapNtC,
			"Idle",
			chainsync.NewMsgFromCborNtC,
		},
		{"forward NtN", forwardNtN, 2, chainsync.StateMapNtN, "Idle", chainsync.NewMsgFromCborNtN},
		{"forward NtC", forwardNtC, 5, chainsync.StateMapNtC, "Idle", chainsync.NewMsgFromCborNtC},
		{
			"rollback NtN",
			ChainSyncRollbackScenario(false, point, tip),
			2,
			chainsync.StateMapNtN,
			"Idle",
			chainsync.NewMsgFromCborNtN,
		},
		{
			"rollback NtC",
			ChainSyncRollbackScenario(true, point, tip),
			5,
			chainsync.StateMapNtC,
			"Idle",
			chainsync.NewMsgFromCborNtC,
		},
		{
			"intersection NtN",
			intersectionNtN,
			2,
			chainsync.StateMapNtN,
			"Idle",
			chainsync.NewMsgFromCborNtN,
		},
		{
			"intersection NtC",
			intersectionNtC,
			5,
			chainsync.StateMapNtC,
			"Idle",
			chainsync.NewMsgFromCborNtC,
		},
		{
			"no intersection",
			noIntersection,
			5,
			chainsync.StateMapNtC,
			"Idle",
			chainsync.NewMsgFromCborNtC,
		},
		{
			"batch",
			batch,
			3,
			blockfetch.StateMap,
			"Idle",
			blockfetch.NewMsgFromCbor,
		},
		{
			"no blocks",
			noBlocks,
			3,
			blockfetch.StateMap,
			"Idle",
			blockfetch.NewMsgFromCbor,
		},
		{"submission", submission, 4, txsubmission.StateMap, "Init", txsubmission.NewMsgFromCbor},
		{
			"no transactions",
			emptySubmission,
			4,
			txsubmission.StateMap,
			"Init",
			txsubmission.NewMsgFromCbor,
		},
		{
			"mempool",
			[]ConversationEntry{
				LocalTxMonitorAcquire(),
				LocalTxMonitorAcquired(42),
				LocalTxMonitorHasTx(ids[0].TxId.TxId[:]),
				LocalTxMonitorReplyHasTx(true),
				LocalTxMonitorGetSizes(),
				LocalTxMonitorReplyGetSizes(100, 1, 1),
				LocalTxMonitorNextTx(),
				LocalTxMonitorReplyNextTx(6, txs[0].TxBody),
				LocalTxMonitorNextTx(),
				LocalTxMonitorReplyNextTx(0, nil),
				LocalTxMonitorRelease(),
				LocalTxMonitorDone(),
			},
			9,
			localtxmonitor.StateMap,
			"Idle",
			localtxmonitor.NewMsgFromCbor,
		},
		{
			"query",
			[]ConversationEntry{
				LocalStateQueryAcquireVolatileTip(),
				LocalStateQueryAcquired(),
				query,
				result,
				LocalStateQueryReAcquireImmutableTip(),
				LocalStateQueryAcquired(),
				LocalStateQueryRelease(),
				LocalStateQueryDone(),
			},
			7,
			localstatequery.StateMap,
			"Idle",
			localstatequery.NewMsgFromCbor,
		},
		{
			"query failure",
			[]ConversationEntry{
				LocalStateQueryAcquire(point),
				LocalStateQueryFailure(localstatequery.AcquireFailurePointNotOnChain),
				LocalStateQueryDone(),
			},
			7,
			localstatequery.StateMap,
			"Idle",
			localstatequery.NewMsgFromCbor,
		},
		{
			"peer sharing",
			[]ConversationEntry{PeerSharingRequest(1), PeerSharingPeers(nil), PeerSharingDone()},
			10,
			peersharing.StateMap,
			"Idle",
			peersharing.NewMsgFromCbor,
		},
		{
			"keepalive",
			[]ConversationEntry{KeepAliveRequest(27), KeepAliveResponse(27), KeepAliveDone()},
			8,
			keepalive.StateMap,
			"Client",
			keepalive.NewMsgFromCbor,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runProtocolScenario(t, test.entries, test.id, test.states, test.initial, test.decoder)
		})
	}
}

func runProtocolScenario(
	t *testing.T,
	entries []ConversationEntry,
	id uint16,
	states protocol.StateMap,
	initialName string,
	decoder protocol.MessageFromCborFunc,
) {
	t.Helper()
	var initial protocol.State
	found := false
	for state := range states {
		if state.Name == initialName {
			initial = state
			found = true
			break
		}
	}
	require.True(t, found, "initial state %s", initialName)
	state := initial
	for _, entry := range entries {
		var messages []protocol.Message
		agency := protocol.AgencyClient
		switch entry := entry.(type) {
		case ConversationEntryInput:
			require.Equal(t, id, entry.ProtocolId)
			require.False(t, entry.IsResponse)
			messages = []protocol.Message{entry.Message}
		case ConversationEntryOutput:
			require.Equal(t, id, entry.ProtocolId)
			require.True(t, entry.IsResponse)
			messages = entry.Messages
			agency = protocol.AgencyServer
		default:
			t.Fatalf("unexpected entry %T", entry)
		}
		for _, message := range messages {
			require.Equal(t, agency, states[state].Agency, "agency before %T", message)
			nextFound := false
			for _, transition := range states[state].Transitions {
				if transition.MsgType == message.Type() &&
					(transition.MatchFunc == nil || transition.MatchFunc(nil, message)) {
					state = transition.NewState
					nextFound = true
					break
				}
			}
			require.True(t, nextFound, "no transition for %T from %s", message, state)
		}
	}
	require.Equal(t, protocol.AgencyNone, states[state].Agency, "scenario must finish the protocol")
	mock := NewConnection(ProtocolRoleClient, entries).(*Connection)
	clientMuxer := muxer.New(mock)
	errors := make(chan error, 8)
	received := make(chan protocol.Message, 8)
	mode := protocol.ProtocolModeNodeToNode
	if id == 5 || id == 7 || id == 9 {
		mode = protocol.ProtocolModeNodeToClient
	}
	client := protocol.New(
		protocol.ProtocolConfig{
			Name:                "scenario",
			ProtocolId:          id,
			Role:                protocol.ProtocolRoleClient,
			Mode:                mode,
			Muxer:               clientMuxer,
			ErrorChan:           errors,
			StateMap:            states,
			InitialState:        initial,
			MessageFromCborFunc: decoder,
			MessageHandlerFunc:  func(message protocol.Message) error { received <- message; return nil },
		},
	)
	t.Cleanup(
		func() { client.Stop(); clientMuxer.Stop(); require.NoError(t, mock.Close()); <-client.DoneChan() },
	)
	client.Start()
	clientMuxer.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, entry := range entries {
		switch entry := entry.(type) {
		case ConversationEntryInput:
			require.NoError(t, client.SendMessageContextAndWait(ctx, entry.Message))
		case ConversationEntryOutput:
			for _, expected := range entry.Messages {
				select {
				case message := <-received:
					require.Equal(t, expected.Type(), message.Type())
					want, err := cbor.Encode(expected)
					require.NoError(t, err)
					message.SetCbor(nil)
					got, err := cbor.Encode(message)
					require.NoError(t, err)
					require.Equal(t, want, got)
				case err := <-errors:
					t.Fatalf("client protocol failed: %v", err)
				case err := <-mock.ErrorChan():
					t.Fatalf("mock protocol failed before response: %v", err)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
		}
	}
	select {
	case err, ok := <-mock.ErrorChan():
		require.False(t, ok, "mock failed: %v", err)
	case err := <-errors:
		t.Fatalf("client protocol failed: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestProtocolScenariosRejectInconsistentInputs(t *testing.T) {
	_, err := BlockFetchScenario(OriginPoint(), OriginPoint(), []uint{7}, nil)
	require.Error(t, err)
	_, err = ChainSyncForwardScenarioNtC([]uint{7}, nil, chainsync.Tip{})
	require.Error(t, err)
	point := NewPoint(2, bytes.Repeat([]byte{1}, 32))
	_, err = ChainSyncIntersectionScenario(
		false,
		[]pcommon.Point{OriginPoint()},
		&point,
		chainsync.Tip{},
	)
	require.Error(t, err)
	_, err = TxSubmissionScenario([]txsubmission.TxIdAndSize{{}}, nil)
	require.Error(t, err)
	_, err = TxSubmissionScenario(
		make([]txsubmission.TxIdAndSize, 65536),
		make([]txsubmission.TxBody, 65536),
	)
	require.Error(t, err)
}

func TestTxSubmissionScenarioRejectsPairedEraMismatch(t *testing.T) {
	ids := []txsubmission.TxIdAndSize{
		{TxId: txsubmission.TxId{EraId: 3}, Size: 1},
		{TxId: txsubmission.TxId{EraId: 6}, Size: 1},
	}
	for _, mismatch := range []int{0, 1} {
		t.Run(fmt.Sprintf("transaction_%d", mismatch), func(t *testing.T) {
			bodies := []txsubmission.TxBody{
				{EraId: 3, TxBody: []byte{0x80}},
				{EraId: 6, TxBody: []byte{0x80}},
			}
			bodies[mismatch].EraId++
			entries, err := TxSubmissionScenario(ids, bodies)
			require.ErrorContains(t, err, "equal era IDs")
			require.Nil(t, entries)
		})
	}
}

func TestTxSubmissionScenarioRejectsPairedSizeMismatch(t *testing.T) {
	ids := []txsubmission.TxIdAndSize{{
		TxId: txsubmission.TxId{EraId: 6},
		Size: 2,
	}}
	bodies := []txsubmission.TxBody{{EraId: 6, TxBody: []byte{0x80}}}

	entries, err := TxSubmissionScenario(ids, bodies)
	require.ErrorContains(t, err, "advertised size")
	require.Nil(t, entries)
}

// TestTxSubmissionScenarioFollowsOutboundRequestRules replays scenarios through
// the outbound side's RequestTxIds checks: a blocking request needs no
// unacknowledged txids after its ack, and a non-blocking one needs some.
func TestTxSubmissionScenarioFollowsOutboundRequestRules(t *testing.T) {
	for _, count := range []int{0, 1, 3} {
		t.Run(fmt.Sprintf("transactions_%d", count), func(t *testing.T) {
			ids := make([]txsubmission.TxIdAndSize, count)
			txs := make([]txsubmission.TxBody, count)
			for i := range count {
				ids[i] = txsubmission.TxIdAndSize{
					TxId: txsubmission.TxId{EraId: 6, TxId: [32]byte{byte(i + 1)}},
				}
				txs[i] = txsubmission.TxBody{EraId: 6}
			}
			entries, err := TxSubmissionScenario(ids, txs)
			require.NoError(t, err)
			unacked := 0
			for i, entry := range entries {
				switch entry := entry.(type) {
				case ConversationEntryOutput:
					for _, message := range entry.Messages {
						request, ok := message.(*txsubmission.MsgRequestTxIds)
						if !ok {
							continue
						}
						require.LessOrEqual(t, int(request.Ack), unacked, "entry %d acks too many", i)
						unacked -= int(request.Ack)
						if request.Blocking {
							require.NotZero(t, request.Req, "entry %d requests nothing", i)
							require.Zero(t, unacked, "entry %d blocks with outstanding txids", i)
						} else {
							require.False(t, request.Req == 0 && request.Ack == 0, "entry %d requests nothing", i)
							require.NotZero(t, unacked, "entry %d is non-blocking with no outstanding txids", i)
						}
					}
				case ConversationEntryInput:
					if reply, ok := entry.Message.(*txsubmission.MsgReplyTxIds); ok {
						unacked += len(reply.TxIds)
					}
				}
			}
		})
	}
}
