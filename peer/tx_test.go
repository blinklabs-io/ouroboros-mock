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

package peer_test

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/blinklabs-io/ouroboros-mock/peer"
	"github.com/stretchr/testify/require"
)

type noCborTransaction struct {
	ledger.Transaction
}

func (noCborTransaction) Cbor() []byte { return nil }

type transactionOverrideBlock struct {
	ledger.Block
	txs []ledger.Transaction
}

func (b transactionOverrideBlock) Transactions() []ledger.Transaction {
	return b.txs
}

func fixtureTxs(t *testing.T, count int) []peer.Tx {
	t.Helper()
	blocks, err := fixtures.GenerateConwayChainWithTransactions(
		1, common.Blake2b256{}, 100, 10, count,
	)
	require.NoError(t, err)
	txs, err := peer.TxsFromBlocks(blocks)
	require.NoError(t, err)
	require.Len(t, txs, count)
	if txs == nil {
		t.Fatal("fixture transactions are missing")
	}
	return txs
}

func newTxPeer(t *testing.T, up *peer.Upstream, txs []peer.Tx) *peer.TxPeer {
	t.Helper()
	p, err := peer.NewTxPeer(peer.TxPeerConfig{Conn: up.Pipe(), Txs: txs})
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })
	return p
}

func TestTxsFromBlocks(t *testing.T) {
	t.Parallel()
	blocks, err := fixtures.GenerateConwayChainWithTransactions(
		1, common.Blake2b256{}, 100, 10, 3,
	)
	require.NoError(t, err)
	txs, err := peer.TxsFromBlocks(blocks)
	require.NoError(t, err)
	require.Len(t, txs, 3)
	for i, b := range blocks {
		require.Equal(t, [32]byte(b.Transactions()[0].Hash()), txs[i].ID)
		require.Equal(t, b.Transactions()[0].Cbor(), txs[i].Raw)
		require.Equal(t, uint16(ledger.TxTypeConway), txs[i].EraId)
	}

	empty := buildChain(t, 2, common.Blake2b256{}, 1, 100)
	none, err := peer.TxsFromBlocks(empty)
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestTxPeerOffersTransactionsTheNodeRequests(t *testing.T) {
	t.Parallel()
	up := newUpstream(t, nil)
	txs := fixtureTxs(t, 3)
	p := newTxPeer(t, up, txs)

	var got []peer.Tx
	for len(got) < len(txs) {
		e := waitEvent(t, up, peer.EventTxsReceived)
		got = append(got, e.Txs...)
	}
	require.Equal(t, txs, got)
	require.Equal(t, txs, up.RelayedTxs())
	require.Equal(t, txs, p.Served())
}

func TestTxPeerRecordsWhatTheNodeAsked(t *testing.T) {
	t.Parallel()
	up := newUpstream(t, nil)
	txs := fixtureTxs(t, 2)
	p := newTxPeer(t, up, txs)

	ids := waitEventOn(t, p.Events(), peer.EventTxIdsRequested)
	require.Equal(t, [][32]byte{txs[0].ID, txs[1].ID}, ids.TxIds)
	req := waitEventOn(t, p.Events(), peer.EventTxsRequested)
	require.Equal(t, ids.TxIds, req.TxIds)
}

func TestTxPeerOfferWakesBlockingRequest(t *testing.T) {
	t.Parallel()
	up := newUpstream(t, nil)
	p := newTxPeer(t, up, nil)

	// The node's first request is blocking and stays parked until a
	// transaction is offered.
	waitEventOn(t, p.Events(), peer.EventTxIdsBlocked)
	txs := fixtureTxs(t, 2)
	p.Offer(txs[0])
	e := waitEvent(t, up, peer.EventTxsReceived)
	require.Equal(t, txs[:1], e.Txs)

	waitEventOn(t, p.Events(), peer.EventTxIdsBlocked)
	p.Offer(txs[1])
	e = waitEvent(t, up, peer.EventTxsReceived)
	require.Equal(t, txs[1:], e.Txs)
	require.Equal(t, txs, up.RelayedTxs())
}

func TestTxsFromBlocksReportsTheTransactionEra(t *testing.T) {
	t.Parallel()
	blocks, err := fixtures.GenerateConwayChainWithTransactions(
		1, common.Blake2b256{}, 100, 10, 1,
	)
	require.NoError(t, err)
	// The Babbage decoder accepts these bodies, giving Babbage-era
	// transactions with the same bytes.
	asBabbage, err := babbage.NewBabbageBlockFromCbor(blocks[0].Cbor())
	require.NoError(t, err)
	txs, err := peer.TxsFromBlocks([]ledger.Block{asBabbage})
	require.NoError(t, err)
	require.Len(t, txs, 1)
	require.Equal(t, uint16(ledger.TxTypeBabbage), txs[0].EraId)
}

func TestTxsFromBlocksRejectsTransactionWithoutCbor(t *testing.T) {
	t.Parallel()
	blocks, err := fixtures.GenerateConwayChainWithTransactions(
		1, common.Blake2b256{}, 100, 10, 1,
	)
	require.NoError(t, err)
	require.Len(t, blocks[0].Transactions(), 1)
	block := transactionOverrideBlock{
		Block: blocks[0],
		txs: []ledger.Transaction{
			noCborTransaction{Transaction: blocks[0].Transactions()[0]},
		},
	}

	txs, err := peer.TxsFromBlocks([]ledger.Block{block})

	require.ErrorContains(t, err, "has no CBOR")
	require.Nil(t, txs)
}
