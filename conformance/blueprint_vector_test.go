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

package conformance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func blueprintVectorJSON(t *testing.T, source blueprintVector) []byte {
	t.Helper()
	data, err := json.Marshal(source)
	require.NoError(t, err)
	return data
}

// TestDecodeBlueprintVectorRejectsSuccessWithoutFinalState covers the only
// end-to-end check a Blueprint vector carries. compareFinalState treats a
// final state shorter than two bytes as "nothing to compare", so a vector
// whose transaction is expected to succeed but that carries no newLedgerState
// would run its transaction and assert nothing about the resulting state.
func TestDecodeBlueprintVectorRejectsSuccessWithoutFinalState(t *testing.T) {
	t.Parallel()
	data := blueprintVectorJSON(t, blueprintVector{
		CBOR:           "00",
		OldLedgerState: "a0",
		NewLedgerState: "",
		Success:        true,
		TestState:      "successful vector without a final state",
	})
	_, err := decodeBlueprintVector("vectors/success.json", data)
	require.ErrorContains(
		t,
		err,
		"successful Blueprint vector has no newLedgerState",
	)
}

// TestDecodeBlueprintVectorAllowsFailureWithoutFinalState pins the other side:
// a vector whose transaction is expected to be rejected has no resulting state
// to compare, so an absent newLedgerState is correct there.
func TestDecodeBlueprintVectorAllowsFailureWithoutFinalState(t *testing.T) {
	t.Parallel()
	data := blueprintVectorJSON(t, blueprintVector{
		CBOR:           "00",
		OldLedgerState: "a0",
		NewLedgerState: "",
		Success:        false,
		TestState:      "rejected vector without a final state",
	})
	vector, err := decodeBlueprintVector("vectors/failure.json", data)
	require.NoError(t, err)
	require.Empty(t, vector.FinalState)
	require.Len(t, vector.Events, 1)
	require.False(t, vector.Events[0].Success)
}
