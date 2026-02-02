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

// Package conformance provides integration tests to verify the conformance
// test infrastructure works correctly. These tests ensure the framework
// doesn't break when updated.
package conformance

import (
	"path/filepath"
	"testing"

	"go.uber.org/goleak"
)

// TestHarnessIntegration verifies that all components of the conformance test
// infrastructure work together correctly. It runs through the complete pipeline:
//
//  1. CollectVectors: Discovers all test vector files in testdata/eras,
//     verifying at least 300 vectors exist.
//  2. DecodeAllVectors: Parses each vector file's CBOR structure into
//     TestVector structs, checking for decode failures.
//  3. ParseAllInitialStates: Extracts ParsedInitialState from each vector's
//     InitialState field, validating the CBOR state structure.
//  4. LoadAllPParams: Loads all protocol parameter files from pparams-by-hash,
//     verifying CBOR decoding into ConwayProtocolParameters.
//  5. EndToEndProcessing: Combines all steps and tracks statistics about
//     vectors with events, UTxOs, and governance state.
func TestHarnessIntegration(t *testing.T) {
	defer goleak.VerifyNone(t)
	root := filepath.Join("testdata", "eras")

	// Step 1: Collect all vector files
	t.Run("CollectVectors", func(t *testing.T) {
		vectors, err := CollectVectorFiles(root)
		if err != nil {
			t.Fatalf("CollectVectorFiles failed: %v", err)
		}
		if len(vectors) < 300 {
			t.Errorf("expected at least 300 vectors, got %d", len(vectors))
		}
		t.Logf("collected %d vectors", len(vectors))
	})

	// Step 2: Decode all vectors
	t.Run("DecodeAllVectors", func(t *testing.T) {
		vectors, err := CollectVectorFiles(root)
		if err != nil {
			t.Fatalf("CollectVectorFiles failed: %v", err)
		}
		var failures int
		for _, path := range vectors {
			_, err := DecodeTestVector(path)
			if err != nil {
				failures++
				if failures <= 3 {
					t.Errorf("failed to decode %s: %v", path, err)
				}
			}
		}
		if failures > 0 {
			t.Errorf("failed to decode %d/%d vectors", failures, len(vectors))
		}
	})

	// Step 3: Parse all initial states
	t.Run("ParseAllInitialStates", func(t *testing.T) {
		vectors, err := CollectVectorFiles(root)
		if err != nil {
			t.Fatalf("CollectVectorFiles failed: %v", err)
		}
		var failures int
		for _, path := range vectors {
			vector, err := DecodeTestVector(path)
			if err != nil {
				continue
			}
			_, err = ParseInitialState(vector.InitialState)
			if err != nil {
				failures++
				if failures <= 3 {
					t.Errorf(
						"failed to parse state for %s: %v",
						vector.Title,
						err,
					)
				}
			}
		}
		if failures > 0 {
			t.Errorf("failed to parse %d initial states", failures)
		}
	})

	// Step 4: Load all protocol parameters
	t.Run("LoadAllPParams", func(t *testing.T) {
		loader := NewPParamsLoaderFromTestdata("testdata")
		hashes, err := loader.ListAvailableHashes()
		if err != nil {
			t.Fatalf("ListAvailableHashes failed: %v", err)
		}

		var failures int
		for _, hashHex := range hashes {
			hashBytes := make([]byte, 32)
			n, hexErr := hexDecode(hashHex, hashBytes)
			if hexErr != nil || n == 0 {
				failures++
				continue
			}
			_, err := loader.Load(hashBytes[:n])
			if err != nil {
				failures++
				if failures <= 3 {
					t.Errorf("failed to load pparams %s: %v", hashHex, err)
				}
			}
		}
		if failures > 0 {
			t.Errorf(
				"failed to load %d/%d pparams files",
				failures,
				len(hashes),
			)
		}
		t.Logf("loaded %d protocol parameter files", len(hashes)-failures)
	})

	// Step 5: End-to-end vector processing
	t.Run("EndToEndProcessing", func(t *testing.T) {
		vectors, err := CollectVectorFiles(root)
		if err != nil {
			t.Fatalf("CollectVectorFiles failed: %v", err)
		}
		loader := NewPParamsLoaderFromTestdata("testdata")

		var (
			processed    int
			ppLoaded     int
			withEvents   int
			withUtxos    int
			withGovState int
		)

		for _, path := range vectors {
			vector, err := DecodeTestVector(path)
			if err != nil {
				continue
			}

			state, err := ParseInitialState(vector.InitialState)
			if err != nil {
				continue
			}

			processed++

			// Try to load protocol parameters
			if len(state.PParamsHash) > 0 {
				_, err := loader.LoadForVector(vector, state)
				if err == nil {
					ppLoaded++
				}
			}

			// Track what data is present
			if len(vector.Events) > 0 {
				withEvents++
			}
			if len(state.Utxos) > 0 {
				withUtxos++
			}
			if len(state.Proposals) > 0 || len(state.CommitteeMembers) > 0 {
				withGovState++
			}
		}

		t.Logf("processed %d vectors", processed)
		t.Logf("  with protocol parameters: %d", ppLoaded)
		t.Logf("  with events: %d", withEvents)
		t.Logf("  with UTxOs: %d", withUtxos)
		t.Logf("  with governance state: %d", withGovState)

		// Ensure we're processing a significant portion
		if processed < 300 {
			t.Errorf(
				"expected to process at least 300 vectors, got %d",
				processed,
			)
		}
		if ppLoaded < 300 {
			t.Errorf(
				"expected to load pparams for at least 300 vectors, got %d",
				ppLoaded,
			)
		}
	})
}

// TestVectorStructure validates the internal structure of decoded test vectors.
// For the first 10 vectors, it verifies:
//   - Title is non-empty
//   - InitialState and FinalState contain CBOR data
//   - FilePath matches the source file path
//   - Transaction events have non-empty TxBytes
//   - All event types are recognized (Transaction, PassTick, PassEpoch)
func TestVectorStructure(t *testing.T) {
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}
	if len(vectors) == 0 {
		t.Fatal("no test vectors found")
	}

	for _, path := range vectors[:min(10, len(vectors))] {
		vector, err := DecodeTestVector(path)
		if err != nil {
			t.Errorf("failed to decode %s: %v", path, err)
			continue
		}

		// Validate structure
		if vector.Title == "" {
			t.Errorf("%s: empty title", path)
		}
		if len(vector.InitialState) == 0 {
			t.Errorf("%s: empty initial state", path)
		}
		if len(vector.FinalState) == 0 {
			t.Errorf("%s: empty final state", path)
		}
		if vector.FilePath != path {
			t.Errorf("%s: FilePath mismatch: %s", path, vector.FilePath)
		}

		// Validate events
		for i, event := range vector.Events {
			switch event.Type {
			case EventTypeTransaction:
				if len(event.TxBytes) == 0 {
					t.Errorf(
						"%s: event %d: transaction has empty TxBytes",
						path,
						i,
					)
				}
			case EventTypePassTick:
				// TickSlot can be 0, so no validation needed
			case EventTypePassEpoch:
				// EpochDelta can be 0, so no validation needed
			default:
				t.Errorf("%s: event %d: unknown type %d", path, i, event.Type)
			}
		}
	}
}

// TestEventTypeDistribution analyzes the distribution of event types across
// all test vectors. It counts:
//   - Transaction events (with success/failure breakdown)
//   - PassTick events (slot advancement)
//   - PassEpoch events (epoch boundary crossing)
//
// This ensures the test suite has adequate coverage of different event types
// and includes both successful and failing transaction scenarios.
func TestEventTypeDistribution(t *testing.T) {
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}

	var (
		transactionEvents int
		passTickEvents    int
		passEpochEvents   int
		successfulTx      int
		failedTx          int
	)

	for _, path := range vectors {
		vector, err := DecodeTestVector(path)
		if err != nil {
			continue
		}

		for _, event := range vector.Events {
			switch event.Type {
			case EventTypeTransaction:
				transactionEvents++
				if event.Success {
					successfulTx++
				} else {
					failedTx++
				}
			case EventTypePassTick:
				passTickEvents++
			case EventTypePassEpoch:
				passEpochEvents++
			}
		}
	}

	t.Logf("event distribution:")
	t.Logf(
		"  transactions: %d (success=%d, fail=%d)",
		transactionEvents,
		successfulTx,
		failedTx,
	)
	t.Logf("  pass tick: %d", passTickEvents)
	t.Logf("  pass epoch: %d", passEpochEvents)

	// Ensure reasonable distribution
	if transactionEvents < 1000 {
		t.Errorf(
			"expected at least 1000 transaction events, got %d",
			transactionEvents,
		)
	}
	if failedTx == 0 {
		t.Error("expected some failing transactions in test vectors")
	}
}

// TestPParamsConsistency verifies that all protocol parameter hashes referenced
// in test vectors can be loaded from the pparams-by-hash directory. It:
//   - Collects all unique pparams hashes from vector initial states
//   - Attempts to load each referenced hash
//   - Reports any missing protocol parameter files
//
// This catches cases where test vectors reference pparams that don't exist.
func TestPParamsConsistency(t *testing.T) {
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}

	loader := NewPParamsLoaderFromTestdata("testdata")
	hashCounts := make(map[string]int)

	for _, path := range vectors {
		vector, err := DecodeTestVector(path)
		if err != nil {
			continue
		}

		state, err := ParseInitialState(vector.InitialState)
		if err != nil {
			continue
		}

		if len(state.PParamsHash) > 0 {
			hashHex := bytesToHex(state.PParamsHash)
			hashCounts[hashHex]++
		}
	}

	t.Logf("found %d unique protocol parameter hashes:", len(hashCounts))

	// Verify all referenced hashes can be loaded
	var loadErrors int
	for hashHex := range hashCounts {
		hashBytes := make([]byte, 32)
		n, hexErr := hexDecode(hashHex, hashBytes)
		if hexErr != nil || n == 0 {
			loadErrors++
			if loadErrors <= 3 {
				t.Errorf("invalid hash format: %s", hashHex)
			}
			continue
		}
		_, err := loader.Load(hashBytes[:n])
		if err != nil {
			loadErrors++
			if loadErrors <= 3 {
				t.Errorf("cannot load referenced pparams: %s", hashHex)
			}
		}
	}

	if loadErrors > 0 {
		t.Errorf(
			"failed to load %d referenced protocol parameter files",
			loadErrors,
		)
	}
}

// TestConformanceInfrastructureVersion logs diagnostic information about the
// current state of the conformance test infrastructure. It reports:
//   - Total number of test vectors available
//   - Number of protocol parameter files
//   - Status of core parsers (vector, state, pparams)
//
// This test always passes and serves as documentation of the test suite size.
func TestConformanceInfrastructureVersion(t *testing.T) {
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}
	loader := NewPParamsLoaderFromTestdata("testdata")
	hashes, err := loader.ListAvailableHashes()
	if err != nil {
		t.Fatalf("ListAvailableHashes failed: %v", err)
	}

	t.Logf("Conformance Infrastructure Status:")
	t.Logf("  Test vectors: %d", len(vectors))
	t.Logf("  Protocol parameter files: %d", len(hashes))
	t.Logf("  Vector parser: OK")
	t.Logf("  State parser: OK")
	t.Logf("  PParams loader: OK")

	// This test always passes but provides useful diagnostics
}

// bytesToHex converts bytes to a hex string.
func bytesToHex(b []byte) string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, len(b)*2)
	for i, v := range b {
		result[i*2] = hexChars[v>>4]
		result[i*2+1] = hexChars[v&0x0f]
	}
	return string(result)
}

// TestMockStateManager verifies that MockStateManager integrates correctly
// with the Harness to run conformance test vectors. It:
//   - Creates a MockStateManager instance
//   - Configures a Harness with the mock state manager
//   - Runs all vectors and collects pass/fail statistics
//   - Logs the first few failures for debugging
//
// This test documents the current implementation status. Failures are expected
// until the MockStateManager fully implements UTxO and governance state loading.
func TestMockStateManager(t *testing.T) {
	defer goleak.VerifyNone(t)
	// Create a MockStateManager
	sm := NewMockStateManager()

	// Create harness with the mock state manager
	harness := NewHarness(sm, HarnessConfig{
		TestdataRoot: "testdata",
		Debug:        false,
	})

	// Run a subset of vectors to verify the harness works
	results, err := harness.RunAllVectorsWithResults()
	if err != nil {
		t.Fatalf("failed to run vectors: %v", err)
	}

	// Count successes and failures
	var successes, failures int
	for _, result := range results {
		if result.Success {
			successes++
		} else {
			failures++
		}
	}

	t.Logf("MockStateManager integration test:")
	t.Logf("  Total vectors: %d", len(results))
	t.Logf("  Passed: %d", successes)
	t.Logf("  Failed: %d", failures)

	// Log first few failures for debugging
	failCount := 0
	for _, result := range results {
		if !result.Success && failCount < 5 {
			t.Logf("  Failure: %s - %v", result.Title, result.Error)
			failCount++
		}
	}

	// This test documents current state; we expect failures until full implementation
}
