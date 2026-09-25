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

package ledger_test

import (
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
	"github.com/blinklabs-io/plutigo/data"
	"github.com/stretchr/testify/require"
)

func TestDijkstraTransactionBuilder(t *testing.T) {
	var keyHash common.Blake2b224
	keyHash[0] = 1
	guard := &dijkstra.DijkstraGuards{KeyHashes: []common.Blake2b224{keyHash}}
	script := common.PlutusV4Script{0x01, 0x02}
	redeemers := dijkstra.DijkstraRedeemers{
		Redeemers: map[common.RedeemerKey]common.RedeemerValue{
			{Tag: common.RedeemerTagGuarding, Index: 0}: {
				Data: common.Datum{Data: data.NewInteger(big.NewInt(1))},
			},
		},
	}
	subtransaction := dijkstra.DijkstraSubTransaction{
		Body: dijkstra.DijkstraSubTransactionBody{TxGuards: guard},
	}

	tx, err := ledger.NewDijkstraTransactionBuilder().
		WithTxGuards(guard).
		WithSubTransactions(subtransaction).
		WithPlutusV4Scripts(script).
		WithRedeemers(redeemers).
		WithTxIsValid(false).
		Build()
	require.NoError(t, err)
	require.False(t, tx.IsValid())
	require.Len(t, tx.Body.TxGuards.KeyHashes, 1)
	require.Len(t, tx.WitnessSet.PlutusV4Scripts(), 1)
	require.Equal(t, script, tx.WitnessSet.PlutusV4Scripts()[0])
	require.Equal(t, redeemers.Len(), tx.WitnessSet.WsRedeemers.Len())
	require.Len(t, tx.Body.TxSubTransactions.Items(), 1)
	require.NotEmpty(t, tx.Cbor())
}

func TestDijkstraTransactionBuilderRejectsInvalidGuards(t *testing.T) {
	_, err := ledger.NewDijkstraTransactionBuilder().
		WithTxGuards(&dijkstra.DijkstraGuards{}).
		Build()
	require.Error(t, err)
}

func TestDijkstraTransactionBuilderPreservesValidTransactions(t *testing.T) {
	tx, err := ledger.NewDijkstraTransactionBuilder().
		WithBody(dijkstra.DijkstraTransactionBody{
			TxSubTransactions: cbor.NewSetType(
				[]dijkstra.DijkstraSubTransaction{}, true,
			),
		}).
		WithTxIsValid(true).
		Build()
	require.NoError(t, err)
	require.True(t, tx.TxIsValid)
}
