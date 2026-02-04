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
	"strings"
	"testing"

	"go.uber.org/goleak"
)

func TestCollectVectorFiles(t *testing.T) {
	defer goleak.VerifyNone(t)
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}

	if len(vectors) == 0 {
		t.Fatal("expected to find test vectors, got 0")
	}

	// Verify we found a reasonable number of vectors (~320 expected)
	if len(vectors) < 100 {
		t.Errorf("expected at least 100 vectors, got %d", len(vectors))
	}

	// Verify no pparams files are included
	for _, v := range vectors {
		if strings.Contains(v, "pparams-by-hash") {
			t.Errorf("found pparams file in vectors: %s", v)
		}
	}

	// Verify vectors are sorted
	for i := 1; i < len(vectors); i++ {
		if vectors[i] < vectors[i-1] {
			t.Errorf(
				"vectors not sorted: %s comes after %s",
				vectors[i],
				vectors[i-1],
			)
		}
	}

	t.Logf("found %d test vectors", len(vectors))
}

func TestDecodeTestVector(t *testing.T) {
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}

	if len(vectors) == 0 {
		t.Fatal("no test vectors found")
	}

	// Test decoding first vector
	vector, err := DecodeTestVector(vectors[0])
	if err != nil {
		t.Fatalf("DecodeTestVector failed for %s: %v", vectors[0], err)
	}

	// Verify basic structure
	if vector.Title == "" {
		t.Error("expected non-empty title")
	}
	if len(vector.InitialState) == 0 {
		t.Error("expected non-empty initial state")
	}
	if len(vector.FinalState) == 0 {
		t.Error("expected non-empty final state")
	}
	if vector.FilePath != vectors[0] {
		t.Errorf("expected FilePath %s, got %s", vectors[0], vector.FilePath)
	}

	t.Logf("decoded vector: %s (events: %d)", vector.Title, len(vector.Events))
}

func TestDecodeAllVectors(t *testing.T) {
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}

	var (
		transactionEvents int
		passTickEvents    int
		passEpochEvents   int
		failedVectors     []string
	)

	for _, path := range vectors {
		vector, err := DecodeTestVector(path)
		if err != nil {
			failedVectors = append(failedVectors, path+": "+err.Error())
			continue
		}

		for _, event := range vector.Events {
			switch event.Type {
			case EventTypeTransaction:
				transactionEvents++
			case EventTypePassTick:
				passTickEvents++
			case EventTypePassEpoch:
				passEpochEvents++
			}
		}
	}

	if len(failedVectors) > 0 {
		t.Errorf("failed to decode %d vectors:", len(failedVectors))
		for _, msg := range failedVectors {
			t.Errorf("  %s", msg)
		}
	}

	t.Logf("successfully decoded %d vectors", len(vectors)-len(failedVectors))
	t.Logf("  transaction events: %d", transactionEvents)
	t.Logf("  pass tick events: %d", passTickEvents)
	t.Logf("  pass epoch events: %d", passEpochEvents)
}

func TestEventTypes(t *testing.T) {
	// Verify event type constants match expected values
	if EventTypeTransaction != 0 {
		t.Errorf(
			"EventTypeTransaction should be 0, got %d",
			EventTypeTransaction,
		)
	}
	if EventTypePassTick != 1 {
		t.Errorf("EventTypePassTick should be 1, got %d", EventTypePassTick)
	}
	if EventTypePassEpoch != 2 {
		t.Errorf("EventTypePassEpoch should be 2, got %d", EventTypePassEpoch)
	}
}

func TestVectorError(t *testing.T) {
	err := &VectorError{
		Path:    "test.cbor",
		Message: "test error",
	}
	expected := "test.cbor: test error"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}

	// Test with wrapped error
	wrapped := &VectorError{
		Path:    "test.cbor",
		Message: "outer",
		Err:     &VectorError{Path: "inner.cbor", Message: "inner"},
	}
	if wrapped.Unwrap() == nil {
		t.Error("expected non-nil unwrap")
	}
}
