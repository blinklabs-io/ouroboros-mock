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

package fixtures_test

import (
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/ouroboros-mock/address"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/blinklabs-io/plutigo/data"
	"github.com/stretchr/testify/require"
)

func testDijkstraGuard() *dijkstra.DijkstraGuards {
	var keyHash common.Blake2b224
	keyHash[0] = 1
	return &dijkstra.DijkstraGuards{KeyHashes: []common.Blake2b224{keyHash}}
}

func testDijkstraRedeemers(value int64) dijkstra.DijkstraRedeemers {
	return dijkstra.DijkstraRedeemers{
		Redeemers: map[common.RedeemerKey]common.RedeemerValue{
			{Tag: common.RedeemerTagGuarding, Index: 0}: {
				Data: common.Datum{Data: data.NewInteger(big.NewInt(value))},
			},
		},
	}
}

// blockTransactionFields splits a block_transaction into its four elements
// and decodes the body into its keyed fields.
func blockTransactionFields(
	t *testing.T,
	tx *dijkstra.DijkstraTransaction,
) ([]cbor.RawMessage, map[uint64]cbor.RawMessage) {
	t.Helper()
	var elements []cbor.RawMessage
	_, err := cbor.Decode(tx.Cbor(), &elements)
	require.NoError(t, err)
	require.Len(t, elements, 4)
	var body map[uint64]cbor.RawMessage
	_, err = cbor.Decode(elements[0], &body)
	require.NoError(t, err)
	return elements, body
}

func TestDijkstraTransactionBuilder(t *testing.T) {
	guard := testDijkstraGuard()
	redeemers := testDijkstraRedeemers(1)
	subtransaction := dijkstra.DijkstraSubTransaction{
		Body: dijkstra.DijkstraSubTransactionBody{TxGuards: guard},
	}

	tx, err := fixtures.NewDijkstraTransactionBuilder().
		WithTxGuards(guard).
		WithSubTransactions(subtransaction).
		WithRedeemers(redeemers).
		WithTxIsValid(false).
		Build()
	require.NoError(t, err)
	require.False(t, tx.IsValid())
	require.Len(t, tx.Body.TxGuards.KeyHashes, 1)
	require.Equal(t, redeemers.Len(), tx.WitnessSet.WsRedeemers.Len())
	require.Len(t, tx.Body.TxSubTransactions.Items(), 1)
	require.NotEmpty(t, tx.Cbor())
}

func TestDijkstraTransactionBuilderRejectsInvalidGuards(t *testing.T) {
	_, err := fixtures.NewDijkstraTransactionBuilder().
		WithTxGuards(&dijkstra.DijkstraGuards{}).
		Build()
	require.Error(t, err)
}

func TestDijkstraTransactionBuilderPreservesValidTransactions(t *testing.T) {
	tx, err := fixtures.NewDijkstraTransactionBuilder().
		WithBody(dijkstra.DijkstraTransactionBody{
			TxSubTransactions: cbor.NewSetType(
				[]dijkstra.DijkstraSubTransaction{{
					Body: dijkstra.DijkstraSubTransactionBody{
						TxGuards: testDijkstraGuard(),
					},
				}}, true,
			),
		}).
		WithTxIsValid(true).
		Build()
	require.NoError(t, err)
	require.True(t, tx.TxIsValid)
}

// sub_transactions is a nonempty_oset, so an empty set has no valid encoding.
func TestDijkstraTransactionBuilderRejectsEmptySubTransactions(t *testing.T) {
	_, err := fixtures.NewDijkstraTransactionBuilder().
		WithBody(dijkstra.DijkstraTransactionBody{
			TxSubTransactions: cbor.NewSetType(
				[]dijkstra.DijkstraSubTransaction{}, true,
			),
		}).
		Build()
	require.ErrorContains(t, err, "sub-transactions must not be empty")
}

func TestDijkstraTransactionBuilderRejectsDecodedEmptySubTransactions(t *testing.T) {
	bodyCBOR, err := cbor.Encode(map[uint64]any{
		0:  []any{},
		1:  []any{},
		2:  uint64(0),
		23: cbor.NewSetType([]cbor.RawMessage{}, true),
	})
	require.NoError(t, err)
	var body dijkstra.DijkstraTransactionBody
	body.SetCbor(bodyCBOR)
	require.NotEmpty(t, body.Cbor(), "test requires the raw-CBOR path")

	_, err = fixtures.NewDijkstraTransactionBuilder().WithBody(body).Build()
	require.ErrorContains(t, err, "sub-transactions must not be empty")
}

// A body that carries original bytes is encoded from them, so the builder
// cannot add the transaction_body keys 1 (outputs) and 2 (fee) that the bytes
// omit. The decoder rejects such bytes, so they are set directly.
func TestDijkstraTransactionBuilderRejectsDecodedBodyMissingRequiredKeys(
	t *testing.T,
) {
	var body dijkstra.DijkstraTransactionBody
	body.SetCbor([]byte{0xa1, 0x00, 0x80})
	require.NotEmpty(t, body.Cbor(), "test requires the raw-CBOR path")

	_, err := fixtures.NewDijkstraTransactionBuilder().WithBody(body).Build()
	require.ErrorContains(t, err, "transaction body is missing required key 1")
}

func TestDijkstraTransactionBuilderRejectsDecodedSubTransactionBodyMissingOutputs(
	t *testing.T,
) {
	var body dijkstra.DijkstraSubTransactionBody
	body.SetCbor([]byte{0xa1, 0x00, 0x80})
	require.NotEmpty(t, body.Cbor(), "test requires the raw-CBOR path")

	_, err := fixtures.NewDijkstraTransactionBuilder().
		WithSubTransactions(dijkstra.DijkstraSubTransaction{Body: body}).
		Build()
	require.ErrorContains(
		t, err, "subtransaction 0: body is missing required key 1",
	)
}

// A witness set that carries original bytes with key 8 reaches the builder's
// key rule. The decoder rejects such bytes, so they are set directly.
func TestDijkstraTransactionBuilderRejectsDecodedPlutusV4WitnessScripts(
	t *testing.T,
) {
	var witnesses dijkstra.DijkstraTransactionWitnessSet
	witnesses.SetCbor(
		[]byte{0xa1, 0x08, 0xd9, 0x01, 0x02, 0x81, 0x41, 0x01},
	)

	_, err := fixtures.NewDijkstraTransactionBuilder().
		WithWitnessSet(witnesses).
		Build()
	require.ErrorContains(t, err, "plutus V4 witness scripts are not part of the Dijkstra CDDL")
}

func TestDijkstraTransactionBuilderRejectsPlutusV4WitnessScripts(t *testing.T) {
	_, err := fixtures.NewDijkstraTransactionBuilder().
		WithWitnessSet(dijkstra.DijkstraTransactionWitnessSet{
			WsPlutusV4Scripts: cbor.NewSetType(
				[]common.PlutusV4Script{{0x01}},
				true,
			),
		}).
		Build()
	require.ErrorContains(t, err, "does not support field 8")
}

func TestDijkstraTransactionBuilderRejectsSubTransactionPlutusV4WitnessScripts(
	t *testing.T,
) {
	_, err := fixtures.NewDijkstraTransactionBuilder().
		WithSubTransactions(dijkstra.DijkstraSubTransaction{
			WitnessSet: dijkstra.DijkstraTransactionWitnessSet{
				WsPlutusV4Scripts: cbor.NewSetType(
					[]common.PlutusV4Script{{0x01}},
					true,
				),
			},
		}).
		Build()
	require.ErrorContains(t, err, "does not support field 8")
}

// The Dijkstra CDDL requires transaction_body keys 0 (inputs), 1 (outputs),
// and 2 (fee), and sub_transaction_body keys 0 and 1, whatever their values.
func TestDijkstraTransactionBuilderEncodesRequiredBodyKeys(t *testing.T) {
	tx, err := fixtures.NewDijkstraTransactionBuilder().
		WithSubTransactions(dijkstra.DijkstraSubTransaction{
			Body: dijkstra.DijkstraSubTransactionBody{TxGuards: testDijkstraGuard()},
		}).
		Build()
	require.NoError(t, err)
	_, body := blockTransactionFields(t, tx)
	for _, key := range []uint64{0, 1, 2} {
		require.Contains(t, body, key, "transaction body key %d", key)
	}
	require.Equal(t, cbor.RawMessage{0x80}, body[1], "empty outputs")
	require.Equal(t, cbor.RawMessage{0x00}, body[2], "zero fee")

	var subtransactions cbor.SetType[cbor.RawMessage]
	_, err = cbor.Decode(body[23], &subtransactions)
	require.NoError(t, err)
	require.Len(t, subtransactions.Items(), 1)
	var sub []cbor.RawMessage
	_, err = cbor.Decode(subtransactions.Items()[0], &sub)
	require.NoError(t, err)
	require.Len(t, sub, 3)
	var subBody map[uint64]cbor.RawMessage
	_, err = cbor.Decode(sub[0], &subBody)
	require.NoError(t, err)
	for _, key := range []uint64{0, 1} {
		require.Contains(t, subBody, key, "subtransaction body key %d", key)
	}
	require.NotContains(t, subBody, uint64(2), "subtransaction bodies carry no fee")
}

func TestDijkstraTransactionBuilderEncodesMetadata(t *testing.T) {
	metadata := common.MetaMap{Pairs: []common.MetaPair{{
		Key:   common.MetaInt{Value: big.NewInt(674)},
		Value: common.MetaText{Value: "fixture"},
	}}}
	tx, err := fixtures.NewDijkstraTransactionBuilder().
		WithMetadata(metadata).
		Build()
	require.NoError(t, err)
	require.NotNil(t, tx.Metadata())
	require.NotNil(t, tx.AuxiliaryData())

	elements, _ := blockTransactionFields(t, tx)
	var labels map[uint64]string
	_, err = cbor.Decode(elements[2], &labels)
	require.NoError(t, err)
	require.Equal(t, map[uint64]string{674: "fixture"}, labels)
	require.NotNil(t, tx.Body.TxAuxDataHash)
	require.Equal(t, common.Blake2b256Hash(elements[2]), *tx.Body.TxAuxDataHash)
}

func TestDijkstraTransactionBuilderKeepsCallerAuxiliaryDataHash(t *testing.T) {
	callerHash := common.Blake2b256Hash([]byte("caller"))
	tx, err := fixtures.NewDijkstraTransactionBuilder().
		WithBody(dijkstra.DijkstraTransactionBody{TxAuxDataHash: &callerHash}).
		WithMetadata(common.MetaMap{Pairs: []common.MetaPair{{
			Key:   common.MetaInt{Value: big.NewInt(1)},
			Value: common.MetaInt{Value: big.NewInt(2)},
		}}}).
		Build()
	require.NoError(t, err)
	require.Equal(t, callerHash, *tx.Body.TxAuxDataHash)
}

func TestDijkstraTransactionBuilderRejectsNonMapMetadata(t *testing.T) {
	_, err := fixtures.NewDijkstraTransactionBuilder().
		WithMetadata(common.MetaText{Value: "not auxiliary data"}).
		Build()
	require.ErrorContains(t, err, "must be a map")

	_, err = fixtures.NewDijkstraTransactionBuilder().
		WithMetadata(common.MetaMap{Pairs: []common.MetaPair{{
			Key:   common.MetaText{Value: "label"},
			Value: common.MetaInt{Value: big.NewInt(1)},
		}}}).
		Build()
	require.ErrorContains(t, err, "labels must be unsigned")
}

func TestDijkstraTransactionBuilderRebuildsDecodedBody(t *testing.T) {
	base, err := fixtures.NewDijkstraTransactionBuilder().Build()
	require.NoError(t, err)
	require.NotNil(t, base.Body.Cbor())

	tx, err := fixtures.NewDijkstraTransactionBuilder().
		WithBody(base.Body).
		WithTxGuards(testDijkstraGuard()).
		Build()
	require.NoError(t, err)
	require.NotNil(t, tx.Body.TxGuards)

	tx, err = fixtures.NewDijkstraTransactionBuilder().
		WithBody(base.Body).
		WithSubTransactions(dijkstra.DijkstraSubTransaction{
			Body: dijkstra.DijkstraSubTransactionBody{TxGuards: testDijkstraGuard()},
		}).
		Build()
	require.NoError(t, err)
	require.Len(t, tx.Body.TxSubTransactions.Items(), 1)
}

func TestDijkstraTransactionBuilderRebuildsDecodedWitnessSet(t *testing.T) {
	base, err := fixtures.NewDijkstraTransactionBuilder().
		WithRedeemers(testDijkstraRedeemers(1)).
		Build()
	require.NoError(t, err)

	replacement := testDijkstraRedeemers(2)
	tx, err := fixtures.NewDijkstraTransactionBuilder().
		WithWitnessSet(base.WitnessSet).
		WithRedeemers(replacement).
		Build()
	require.NoError(t, err)
	got := tx.WitnessSet.WsRedeemers.Value(0, common.RedeemerTagGuarding)
	encoded, err := cbor.Encode(got.Data)
	require.NoError(t, err)
	require.Equal(t, []byte{0x02}, encoded, "replacement redeemer datum")
}

func testAccountAddress(t *testing.T, seed byte) common.Address {
	t.Helper()
	hash := make([]byte, common.AddressHashSize)
	hash[0] = seed
	account, err := address.NewAddress().WithStakingKeyHash(hash).BuildAccount()
	require.NoError(t, err)
	return account
}

func TestDijkstraTransactionBuilderEncodesDirectDeposits(t *testing.T) {
	t.Parallel()
	first, second := testAccountAddress(t, 1), testAccountAddress(t, 2)
	firstBytes, err := first.Bytes()
	require.NoError(t, err)
	secondBytes, err := second.Bytes()
	require.NoError(t, err)

	tx, err := fixtures.NewDijkstraTransactionBuilder().
		WithDirectDeposit(first, 5).
		WithDirectDeposit(second, 7).
		WithDirectDeposit(first, 9).
		Build()
	require.NoError(t, err)
	require.Equal(t, dijkstra.DijkstraDirectDeposits{
		cbor.NewByteString(firstBytes):  9,
		cbor.NewByteString(secondBytes): 7,
	}, tx.Body.TxDirectDeposits)
	_, body := blockTransactionFields(t, tx)
	require.Contains(t, body, uint64(25))
}

func TestDijkstraTransactionBuilderEncodesAccountBalanceIntervals(t *testing.T) {
	t.Parallel()
	first, second := testAccountAddress(t, 1), testAccountAddress(t, 2)
	firstBytes, err := first.Bytes()
	require.NoError(t, err)
	secondBytes, err := second.Bytes()
	require.NoError(t, err)
	exact, lower, upper := uint64(10), uint64(20), uint64(30)

	tx, err := fixtures.NewDijkstraTransactionBuilder().
		WithAccountBalanceInterval(
			first,
			dijkstra.DijkstraAccountBalanceInterval{Exact: &exact},
		).
		WithAccountBalanceInterval(
			second,
			dijkstra.DijkstraAccountBalanceInterval{LowerBound: &lower},
		).
		WithAccountBalanceInterval(
			second,
			dijkstra.DijkstraAccountBalanceInterval{
				LowerBound: &lower,
				UpperBound: &upper,
			},
		).
		Build()
	require.NoError(t, err)
	require.Len(t, tx.Body.TxBalanceIntervals, 2)
	firstInterval, ok := tx.Body.TxBalanceIntervals[cbor.NewByteString(firstBytes)]
	require.True(t, ok)
	require.NotNil(t, firstInterval)
	require.Equal(
		t,
		dijkstra.DijkstraAccountBalanceInterval{Exact: &exact},
		*firstInterval,
	)
	secondInterval, ok := tx.Body.TxBalanceIntervals[cbor.NewByteString(secondBytes)]
	require.True(t, ok)
	require.NotNil(t, secondInterval)
	require.Equal(
		t,
		dijkstra.DijkstraAccountBalanceInterval{LowerBound: &lower, UpperBound: &upper},
		*secondInterval,
		"a later interval for the same account replaces the earlier one",
	)
	_, body := blockTransactionFields(t, tx)
	require.Contains(t, body, uint64(26))
}

func TestDijkstraTransactionBuilderAccountSettersOwnMapChanges(t *testing.T) {
	t.Parallel()
	first, second := testAccountAddress(t, 1), testAccountAddress(t, 2)
	firstBytes, err := first.Bytes()
	require.NoError(t, err)
	exact := uint64(10)
	body := dijkstra.DijkstraTransactionBody{
		TxDirectDeposits: map[cbor.ByteString]uint64{
			cbor.NewByteString(firstBytes): 5,
		},
		TxBalanceIntervals: dijkstra.DijkstraAccountBalanceIntervals{
			cbor.NewByteString(firstBytes): {Exact: &exact},
		},
	}
	builder := fixtures.NewDijkstraTransactionBuilder().
		WithBody(body).
		WithDirectDeposit(second, 7).
		WithAccountBalanceInterval(
			second,
			dijkstra.DijkstraAccountBalanceInterval{Exact: new(uint64)},
		)

	require.Len(t, body.TxDirectDeposits, 1)
	require.Len(t, body.TxBalanceIntervals, 1)
	body.TxDirectDeposits[cbor.NewByteString(firstBytes)] = 99
	exact = 99
	tx, err := builder.Build()
	require.NoError(t, err)
	require.Equal(t, uint64(5), tx.Body.TxDirectDeposits[cbor.NewByteString(firstBytes)])
	firstInterval, ok := tx.Body.TxBalanceIntervals[cbor.NewByteString(firstBytes)]
	require.True(t, ok)
	require.NotNil(t, firstInterval)
	require.NotNil(t, firstInterval.Exact)
	require.Equal(t, uint64(10), *firstInterval.Exact)
}

func TestDijkstraTransactionBuilderRejectsInvalidAccountFields(t *testing.T) {
	t.Parallel()
	enterprise, err := address.RandomEnterprise(address.Testnet)
	require.NoError(t, err)
	account := testAccountAddress(t, 1)

	_, err = fixtures.NewDijkstraTransactionBuilder().
		WithDirectDeposit(enterprise, 1).
		Build()
	require.ErrorContains(t, err, "not a reward account")

	_, err = fixtures.NewDijkstraTransactionBuilder().
		WithAccountBalanceInterval(
			enterprise,
			dijkstra.DijkstraAccountBalanceInterval{Exact: new(uint64)},
		).
		Build()
	require.ErrorContains(t, err, "not a reward account")

	_, err = fixtures.NewDijkstraTransactionBuilder().
		WithAccountBalanceInterval(
			account,
			dijkstra.DijkstraAccountBalanceInterval{},
		).
		Build()
	require.ErrorContains(t, err, "requires a lower or upper bound")
}

func TestDijkstraTransactionBuilderRejectsEmptyBalanceIntervalAccount(
	t *testing.T,
) {
	t.Parallel()
	exact := uint64(10)
	body := dijkstra.DijkstraTransactionBody{
		TxBalanceIntervals: dijkstra.DijkstraAccountBalanceIntervals{
			cbor.NewByteString(nil): {Exact: &exact},
		},
	}

	_, err := fixtures.NewDijkstraTransactionBuilder().
		WithBody(body).
		WithAccountBalanceInterval(
			testAccountAddress(t, 1),
			dijkstra.DijkstraAccountBalanceInterval{Exact: &exact},
		).
		Build()
	require.Error(t, err)
}

// A decoded body keeps its original bytes, so each account setter must drop
// them or Build would encode the body without the new field.
func TestDijkstraTransactionBuilderAccountFieldsReplaceDecodedBodyCBOR(
	t *testing.T,
) {
	t.Parallel()
	account := testAccountAddress(t, 1)
	exact := uint64(10)
	for _, test := range []struct {
		name string
		key  uint64
		set  func(*fixtures.DijkstraTransactionBuilder) *fixtures.DijkstraTransactionBuilder
	}{
		{"direct deposit", 25, func(b *fixtures.DijkstraTransactionBuilder) *fixtures.DijkstraTransactionBuilder {
			return b.WithDirectDeposit(account, 5)
		}},
		{"balance interval", 26, func(b *fixtures.DijkstraTransactionBuilder) *fixtures.DijkstraTransactionBuilder {
			return b.WithAccountBalanceInterval(
				account,
				dijkstra.DijkstraAccountBalanceInterval{Exact: &exact},
			)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			bodyCBOR, err := cbor.Encode(map[uint64]any{
				0: []any{},
				1: []any{},
				2: uint64(0),
			})
			require.NoError(t, err)
			var body dijkstra.DijkstraTransactionBody
			_, err = cbor.Decode(bodyCBOR, &body)
			require.NoError(t, err)
			require.NotEmpty(t, body.Cbor(), "test requires the decoded raw-CBOR path")

			tx, err := test.set(
				fixtures.NewDijkstraTransactionBuilder().WithBody(body),
			).Build()
			require.NoError(t, err)
			_, fields := blockTransactionFields(t, tx)
			require.Contains(t, fields, test.key)
		})
	}
}
