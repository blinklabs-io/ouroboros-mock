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
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
	"github.com/stretchr/testify/require"
)

func TestValidityIntervalFixtures(t *testing.T) {
	testCases := ledger.ValidityIntervalFixtures()
	require.Len(t, testCases, 7)
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			for _, decoder := range []struct {
				name string
				run  func() (lcommon.Transaction, error)
			}{
				{
					name: "Alonzo",
					run: func() (lcommon.Transaction, error) {
						return testCase.AlonzoTransaction()
					},
				},
				{
					name: "Babbage",
					run: func() (lcommon.Transaction, error) {
						return testCase.BabbageTransaction()
					},
				},
			} {
				t.Run(decoder.name, func(t *testing.T) {
					tx, err := decoder.run()
					require.NoError(t, err)
					requireBound(
						t,
						tx,
						8,
						testCase.StartSlot,
					)
					requireBound(
						t,
						tx,
						3,
						testCase.EndSlot,
					)
				})
			}
		})
	}
}

func requireBound(
	t *testing.T,
	tx lcommon.Transaction,
	key uint,
	expected *uint64,
) {
	t.Helper()
	var txFields []cbor.RawMessage
	_, err := cbor.Decode(tx.Cbor(), &txFields)
	require.NoError(t, err)
	require.NotEmpty(t, txFields)
	var bodyFields map[uint]cbor.RawMessage
	_, err = cbor.Decode(txFields[0], &bodyFields)
	require.NoError(t, err)
	raw, present := bodyFields[key]
	require.Equal(t, expected != nil, present)
	if expected == nil {
		return
	}
	var actual uint64
	_, err = cbor.Decode(raw, &actual)
	require.NoError(t, err)
	require.Equal(t, *expected, actual)
	if key == 8 {
		require.Equal(t, *expected, tx.ValidityIntervalStart())
	} else {
		require.Equal(t, *expected, tx.TTL())
	}
}
