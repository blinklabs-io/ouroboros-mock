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

func TestSummarizeCoverage(t *testing.T) {
	results := []VectorResult{
		{
			Path:    "testdata/eras/conway/ConwayImpSpec_-_Version_10.GOV.0",
			Success: true,
		},
		{
			Path:    "testdata/eras/conway/ConwayImpSpec_-_Version_10.GOVCERT.0",
			Success: false,
		},
		{Path: "testdata/eras/conway/AlonzoImpSpec.UTXOS.0", Success: true},
		{Path: "synthetic/rollback/CurrentTreasuryValue_V1", Success: true},
	}

	summary := SummarizeCoverage(results)
	require.Equal(t, CoverageSummary{Total: 1, Passed: 1}, summary[CoverageKey{
		Era: "Conway", RuleFamily: "GOV",
	}])
	require.Equal(t, CoverageSummary{Total: 1, Failed: 1}, summary[CoverageKey{
		Era: "Conway", RuleFamily: "GOVCERT",
	}])
	require.Equal(t, CoverageSummary{Total: 1, Passed: 1}, summary[CoverageKey{
		Era: "Alonzo", RuleFamily: "UTXOS",
	}])
	require.Equal(t, CoverageSummary{Total: 1, Passed: 1}, summary[CoverageKey{
		Era: "unknown", RuleFamily: "unknown",
	}])
}

func TestSortedCoverageKeys(t *testing.T) {
	keys := SortedCoverageKeys(map[CoverageKey]CoverageSummary{
		{Era: "Conway", RuleFamily: "GOV"}:   {},
		{Era: "Alonzo", RuleFamily: "UTXOS"}: {},
		{Era: "Alonzo", RuleFamily: "GOV"}:   {},
	})
	require.Equal(t, []CoverageKey{
		{Era: "Alonzo", RuleFamily: "GOV"},
		{Era: "Alonzo", RuleFamily: "UTXOS"},
		{Era: "Conway", RuleFamily: "GOV"},
	}, keys)
}
