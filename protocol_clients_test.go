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
	"testing"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestLocalTxMonitorBuildersServeGouroborosClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	hash := bytes.Repeat([]byte{0x31}, 32)
	tx := []byte{0x80}
	entries := []ConversationEntry{
		LocalTxMonitorAcquire(), LocalTxMonitorAcquired(456),
		LocalTxMonitorHasTx(hash), LocalTxMonitorReplyHasTx(true),
		LocalTxMonitorGetSizes(), LocalTxMonitorReplyGetSizes(1000, 20, 3),
		LocalTxMonitorNextTx(), LocalTxMonitorReplyNextTx(6, tx),
		LocalTxMonitorNextTx(), LocalTxMonitorReplyNextTx(0, nil),
		LocalTxMonitorRelease(), LocalTxMonitorDone(),
	}
	client, mock := newProtocolClientConnection(t, entries, false)
	monitor := client.LocalTxMonitor().Client
	require.NoError(t, monitor.Acquire())
	present, err := monitor.HasTx(hash)
	require.NoError(t, err)
	require.True(t, present)
	capacity, size, count, err := monitor.GetSizes()
	require.NoError(t, err)
	require.Equal(t, uint32(1000), capacity)
	require.Equal(t, uint32(20), size)
	require.Equal(t, uint32(3), count)
	got, err := monitor.NextTx()
	require.NoError(t, err)
	require.Equal(t, tx, got)
	got, err = monitor.NextTx()
	require.NoError(t, err)
	require.Nil(t, got)
	require.NoError(t, monitor.Release())
	require.NoError(t, monitor.Stop())
	waitProtocolConversation(t, mock)
}

func TestLocalStateQueryBuildersServeGouroborosClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	query, err := LocalStateQueryQuery([]any{localstatequery.QueryTypeChainBlockNo})
	require.NoError(t, err)
	response, err := LocalStateQueryResult([]any{uint64(1), uint64(99)})
	require.NoError(t, err)
	entries := []ConversationEntry{
		LocalStateQueryAcquireVolatileTip(), LocalStateQueryAcquired(), query, response,
		LocalStateQueryReAcquireImmutableTip(), LocalStateQueryAcquired(),
		LocalStateQueryRelease(), LocalStateQueryDone(),
	}
	client, mock := newProtocolClientConnection(t, entries, false)
	stateQuery := client.LocalStateQuery().Client
	require.NoError(t, stateQuery.AcquireVolatileTip())
	blockNo, err := stateQuery.GetChainBlockNo()
	require.NoError(t, err)
	require.Equal(t, int64(99), blockNo)
	require.NoError(t, stateQuery.AcquireImmutableTip())
	require.NoError(t, stateQuery.Release())
	require.NoError(t, stateQuery.SendMessage(localstatequery.NewMsgDone()))
	waitProtocolConversation(t, mock)
}

func newProtocolClientConnection(
	t *testing.T,
	entries []ConversationEntry,
	nodeToNode bool,
	options ...ouroboros.ConnectionOptionFunc,
) (*ouroboros.Connection, *Connection) {
	t.Helper()
	response := ConversationEntryHandshakeNtCResponse
	if nodeToNode {
		response = ConversationEntryHandshakeNtNResponse
	}
	script := append(
		[]ConversationEntry{ConversationEntryHandshakeRequestGeneric, response},
		entries...)
	mock := NewConnection(ProtocolRoleClient, script).(*Connection)
	require.NoError(t, mock.SetDeadline(time.Now().Add(5*time.Second)))
	t.Cleanup(func() { require.NoError(t, mock.Close()) })
	options = append(
		[]ouroboros.ConnectionOptionFunc{
			ouroboros.WithConnection(mock),
			ouroboros.WithNetworkMagic(MockNetworkMagic),
			ouroboros.WithNodeToNode(nodeToNode),
		},
		options...)
	client, err := ouroboros.New(options...)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()); <-client.ErrorChan() })
	return client, mock
}

func waitProtocolConversation(t *testing.T, mock *Connection) {
	t.Helper()
	select {
	case err, ok := <-mock.ErrorChan():
		require.False(t, ok, "mock failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("protocol entries did not finish")
	}
}

func TestBlockFetchScenarioServesGouroborosClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	for _, era := range fixtures.SupportedEras() {
		t.Run(era.Name, func(t *testing.T) {
			block, err := fixtures.GenerateBlock(era, 1, 0)
			require.NoError(t, err)
			point := NewPoint(block.SlotNumber(), block.Hash().Bytes())
			entries, err := BlockFetchScenario(
				point,
				point,
				[]uint{uint(block.Type())},
				[][]byte{block.Cbor()},
			)
			require.NoError(t, err)
			client, mock := newProtocolClientConnection(t, entries, true)
			got, err := client.BlockFetch().Client.GetBlock(point)
			require.NoError(t, err)
			require.Equal(t, block.Type(), got.Type())
			require.Equal(t, block.Hash(), got.Hash())
			require.Equal(t, block.Cbor(), got.Cbor())
			require.NoError(t, client.BlockFetch().Client.Stop())
			waitProtocolConversation(t, mock)
		})
	}
}

func TestTxSubmissionScenarioServesGouroborosClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	blocks, err := fixtures.GenerateConwayChainWithTransactions(1, common.Blake2b256{}, 1, 1, 1)
	require.NoError(t, err)
	transaction := blocks[0].Transactions()[0]
	data := transaction.Cbor()
	ids := []txsubmission.TxIdAndSize{
		{
			TxId: txsubmission.TxId{EraId: 6, TxId: [32]byte(transaction.Hash().Bytes())},
			Size: uint32(len(data)),
		},
	}
	txs := []txsubmission.TxBody{{EraId: 6, TxBody: data}}
	entries, err := TxSubmissionScenario(ids, txs)
	require.NoError(t, err)
	type idRequest struct {
		blocking     bool
		ack, request uint16
	}
	requests := make(chan idRequest, 2)
	requestedIDs := make(chan []txsubmission.TxId, 1)
	config := txsubmission.Config{
		RequestTxIdsFunc: func(_ txsubmission.CallbackContext, blocking bool, ack, request uint16) ([]txsubmission.TxIdAndSize, error) {
			requests <- idRequest{blocking, ack, request}
			if blocking && ack > 0 {
				return nil, txsubmission.ErrStopServerProcess
			}
			return ids, nil
		},
		RequestTxsFunc: func(_ txsubmission.CallbackContext, requested []txsubmission.TxId) ([]txsubmission.TxBody, error) {
			requestedIDs <- requested
			return txs, nil
		},
	}
	client, mock := newProtocolClientConnection(
		t,
		entries,
		true,
		ouroboros.WithTxSubmissionConfig(config),
	)
	client.TxSubmission().Client.Init()
	waitProtocolConversation(t, mock)
	require.Equal(t, idRequest{true, 0, 1}, <-requests)
	require.Equal(t, idRequest{true, 1, 1}, <-requests)
	require.Equal(t, []txsubmission.TxId{ids[0].TxId}, <-requestedIDs)
}
