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

package fixtures

import "testing"

func TestStrictDecodePlaceholderExecutionsAreAllOrNone(t *testing.T) {
	harness := NewHarness(HarnessConfig{})
	results, err := harness.RunAllExecutionsWithResults()
	if err != nil {
		t.Fatalf("run fixture executions: %s", err)
	}
	var rejectionCount int
	for _, result := range results {
		if result.ExpectedStrictDecodeRejection {
			rejectionCount++
			if !isStrictDecodePlaceholderFixture(result.Fixture) {
				t.Fatalf(
					"unexpected strict-decode rejection for %s",
					result.Fixture.RelPath,
				)
			}
		}
	}
	if rejectionCount != 0 &&
		rejectionCount != len(strictDecodePlaceholderFixtures) {
		t.Fatalf(
			"strict-decode rejection count = %d, want 0 or %d",
			rejectionCount,
			len(strictDecodePlaceholderFixtures),
		)
	}
}
