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

import (
	"errors"
	"fmt"
	"testing"
)

func TestStrictDecodePlaceholderMatcher(t *testing.T) {
	exactStrictDecodeErrors := []string{
		"invalid blake2b-256 hash: expected 32 bytes, got 2",
		"invalid blake2b-256 hash length: expected 32 bytes, got 2",
	}

	allowlisted := Fixture{
		RelPath: "cardano-ledger/eras/alonzo/test-suite/golden/block.cbor",
	}
	if !isStrictDecodePlaceholderFixture(allowlisted) {
		t.Fatal("expected the preserved Alonzo block to be allowlisted")
	}
	if isStrictDecodePlaceholderFixture(Fixture{RelPath: "unrelated.cbor"}) {
		t.Fatal("unexpected allowlist match for unrelated fixture")
	}

	for _, errorText := range exactStrictDecodeErrors {
		exact := errors.New(errorText)
		if !isStrictDecodePlaceholderError(exact) {
			t.Fatalf("expected exact strict decode error %q to match", errorText)
		}
		if !isStrictDecodePlaceholderError(fmt.Errorf("decode fixture: %w", exact)) {
			t.Fatalf("expected wrapped strict decode error %q to match", errorText)
		}
	}
	for _, err := range []error{
		errors.New("invalid blake2b-256 hash: expected 32 bytes, got 3"),
		errors.New("invalid blake2b-256 hash length: expected 32 bytes, got 3"),
		errors.New("unrelated error: " + exactStrictDecodeErrors[0]),
		errors.New("unrelated error: " + exactStrictDecodeErrors[1]),
	} {
		if isStrictDecodePlaceholderError(err) {
			t.Fatalf("unexpected strict decode match for %q", err)
		}
	}
}

func TestStrictDecodePlaceholderExecutionsAreAllOrNone(t *testing.T) {
	harness := NewHarness(HarnessConfig{})
	results, err := harness.RunAllExecutionsWithResults()
	if err != nil {
		t.Fatalf("run fixture executions: %s", err)
	}
	var rejectionCount int
	seen := make(map[string]struct{}, len(strictDecodePlaceholderFixtures))
	for _, result := range results {
		if _, allowlisted := strictDecodePlaceholderFixtures[result.Fixture.RelPath]; allowlisted {
			seen[result.Fixture.RelPath] = struct{}{}
			if result.Error != nil {
				t.Fatalf(
					"allowlisted fixture %s failed with an unexpected error: %v",
					result.Fixture.RelPath,
					result.Error,
				)
			}
		}
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
	if len(seen) != len(strictDecodePlaceholderFixtures) {
		t.Fatalf(
			"strict-decode allowlist covered %d fixtures, want %d",
			len(seen),
			len(strictDecodePlaceholderFixtures),
		)
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
