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
	"path/filepath"
	"sort"
	"strings"
)

// CoverageKey identifies a vector's ledger era and Blueprint rule family.
type CoverageKey struct {
	Era        string
	RuleFamily string
}

// CoverageSummary contains the outcome counts for one era/rule-family pair.
type CoverageSummary struct {
	Total  int
	Passed int
	Failed int
}

// SummarizeCoverage groups vector results by ledger era and rule family.
// Classification uses the Blueprint path because the JSON title is not
// required to preserve the corpus directory structure.
func SummarizeCoverage(results []VectorResult) map[CoverageKey]CoverageSummary {
	summary := make(map[CoverageKey]CoverageSummary)
	for _, result := range results {
		key := coverageKey(result.Path, result.Title)
		counts := summary[key]
		counts.Total++
		if result.Success {
			counts.Passed++
		} else {
			counts.Failed++
		}
		summary[key] = counts
	}
	return summary
}

// SortedCoverageKeys returns report keys in stable display order.
func SortedCoverageKeys(summary map[CoverageKey]CoverageSummary) []CoverageKey {
	keys := make([]CoverageKey, 0, len(summary))
	for key := range summary {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Era != keys[j].Era {
			return keys[i].Era < keys[j].Era
		}
		return keys[i].RuleFamily < keys[j].RuleFamily
	})
	return keys
}

func coverageKey(path, title string) CoverageKey {
	corpusPath := filepath.ToSlash(path)
	if title != "" {
		corpusPath += " " + title
	}
	if strings.Contains(corpusPath, "/synthetic/") {
		return CoverageKey{Era: "synthetic", RuleFamily: "rollback"}
	}

	const unknown = "unknown"
	era := unknown
	for _, candidate := range []string{
		"Allegra",
		"Alonzo",
		"Babbage",
		"Conway",
		"Mary",
		"Shelley",
	} {
		if strings.Contains(corpusPath, candidate+"ImpSpec") {
			era = candidate
			break
		}
	}

	// Longer names must precede their prefixes (for example GOVCERT/GOV).
	for _, family := range []string{
		"GOVCERT",
		"RATIFY",
		"ENACT",
		"DELEG",
		"CERTS",
		"EPOCH",
		"UTXOS",
		"UTXOW",
		"UTXO",
		"LEDGER",
	} {
		if strings.Contains(corpusPath, "."+family+".") {
			return CoverageKey{Era: era, RuleFamily: family}
		}
	}
	if strings.Contains(corpusPath, ".GOV.") {
		return CoverageKey{Era: era, RuleFamily: "GOV"}
	}
	return CoverageKey{Era: era, RuleFamily: unknown}
}
