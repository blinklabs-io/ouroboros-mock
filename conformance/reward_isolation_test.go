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
	"testing"

	"github.com/stretchr/testify/require"
)

// Both public runners must validate and apply the corpus without importing
// expected reward balances into the state manager before execution.
func TestHarnessRewardStateIsolation(t *testing.T) {
	for _, testingRunner := range []bool{false, true} {
		name := "results runner"
		if testingRunner {
			name = "testing runner"
		}
		t.Run(name, func(t *testing.T) {
			sm := &recordingStateManager{
				MockStateManager: NewMockStateManager(),
			}
			h := NewHarness(sm, HarnessConfig{TestdataRoot: "testdata"})
			paths, err := h.collectAllVectors()
			require.NoError(t, err)
			require.NotEmpty(t, paths)
			if testingRunner {
				h.RunAllVectors(t)
			} else {
				results, err := h.RunAllVectorsWithResults()
				require.NoError(t, err)
				require.Len(t, results, len(paths))
				for _, result := range results {
					require.True(t, result.Success, "%s: %v", result.Title, result.Error)
				}
			}
			require.Zero(
				t,
				len(sm.setRewardAccountBalanceCalls),
				"execution must not import expected credential reward balances",
			)
			require.Zero(
				t,
				len(sm.setRewardBalancesCalls),
				"execution must not import expected legacy reward balances",
			)
		})
	}
}
