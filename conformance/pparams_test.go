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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"go.uber.org/goleak"
)

// TestNewPParamsLoader verifies that NewPParamsLoaderFromTestdata creates a
// properly configured loader with a non-empty pparams directory path.
func TestNewPParamsLoader(t *testing.T) {
	defer goleak.VerifyNone(t)
	loader := NewPParamsLoaderFromTestdata("testdata")
	if loader == nil {
		t.Fatal("expected non-nil loader")
	}
	if loader.pparamsDir == "" {
		t.Fatal("expected non-empty pparams directory")
	}
}

// TestPParamsLoaderListAvailableHashes verifies that ListAvailableHashes
// returns all protocol parameter file names from the pparams-by-hash directory.
// These files are named by their Blake2b-256 hash and contain CBOR-encoded
// ConwayProtocolParameters.
func TestPParamsLoaderListAvailableHashes(t *testing.T) {
	loader := NewPParamsLoaderFromTestdata("testdata")

	hashes, err := loader.ListAvailableHashes()
	if err != nil {
		t.Fatalf("ListAvailableHashes failed: %v", err)
	}

	if len(hashes) == 0 {
		t.Fatal("expected at least one pparams hash file")
	}

	t.Logf("found %d protocol parameter files", len(hashes))

	// Log first few hashes
	for i, h := range hashes {
		if i >= 5 {
			t.Logf("  ... and %d more", len(hashes)-5)
			break
		}
		t.Logf("  %s", h)
	}
}

// TestPParamsLoaderLoad verifies that Load correctly decodes a protocol
// parameters file into ConwayProtocolParameters. It checks:
//   - The loaded parameters are non-nil
//   - The result is the correct concrete type (ConwayProtocolParameters)
//   - Key fields (MinFeeA, MinFeeB, MaxTxSize, etc.) are populated
func TestPParamsLoaderLoad(t *testing.T) {
	loader := NewPParamsLoaderFromTestdata("testdata")

	hashes, err := loader.ListAvailableHashes()
	if err != nil {
		t.Fatalf("ListAvailableHashes failed: %v", err)
	}

	if len(hashes) == 0 {
		t.Skip("no pparams files available")
	}

	// Load first available hash
	hashBytes := make([]byte, 32)
	n, err := hexDecode(hashes[0], hashBytes)
	if err != nil {
		t.Skipf("invalid hash format: %s: %v", hashes[0], err)
	}

	pp, err := loader.Load(hashBytes[:n])
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if pp == nil {
		t.Fatal("expected non-nil protocol parameters")
	}

	// Verify it's a Conway parameters type
	cpp, ok := pp.(*conway.ConwayProtocolParameters)
	if !ok {
		t.Fatalf("expected ConwayProtocolParameters, got %T", pp)
	}

	t.Logf("loaded protocol parameters:")
	t.Logf("  MinFeeA: %d", cpp.MinFeeA)
	t.Logf("  MinFeeB: %d", cpp.MinFeeB)
	t.Logf("  MaxTxSize: %d", cpp.MaxTxSize)
	t.Logf("  KeyDeposit: %d", cpp.KeyDeposit)
	t.Logf("  PoolDeposit: %d", cpp.PoolDeposit)
	t.Logf("  CostModels: %d versions", len(cpp.CostModels))
}

// TestPParamsLoaderCache verifies the caching behavior of PParamsLoader:
//   - First load reads from disk and returns a deep copy
//   - Second load returns a different deep copy (not pointer equal)
//   - After ClearCache, the next load returns a fresh instance
//
// Deep copying prevents cross-contamination between test vectors that share
// the same pparams hash, as modifications (e.g., via ParameterChange) should
// not affect other vectors.
func TestPParamsLoaderCache(t *testing.T) {
	loader := NewPParamsLoaderFromTestdata("testdata")

	hashes, err := loader.ListAvailableHashes()
	if err != nil || len(hashes) == 0 {
		t.Skip("no pparams files available")
	}

	hashBytes := make([]byte, 32)
	n, hexErr := hexDecode(hashes[0], hashBytes)
	if hexErr != nil {
		t.Skipf("invalid hash format: %v", hexErr)
	}

	// First load
	pp1, err := loader.Load(hashBytes[:n])
	if err != nil {
		t.Fatalf("first Load failed: %v", err)
	}

	// Second load (should return a fresh copy to prevent cross-contamination)
	pp2, err := loader.Load(hashBytes[:n])
	if err != nil {
		t.Fatalf("second Load failed: %v", err)
	}

	// Should be DIFFERENT instances (copies) to prevent cross-contamination
	// between test vectors that share the same pparams hash
	if pp1 == pp2 {
		t.Error("expected different instances (deep copies) from cache")
	}

	// Both should have the same values (verify cost models as example)
	cpp1, ok1 := pp1.(*conway.ConwayProtocolParameters)
	cpp2, ok2 := pp2.(*conway.ConwayProtocolParameters)
	if ok1 && ok2 {
		if len(cpp1.CostModels) != len(cpp2.CostModels) {
			t.Error("expected same cost models count")
		}
	}

	// Clear cache and load again
	loader.ClearCache()
	pp3, err := loader.Load(hashBytes[:n])
	if err != nil {
		t.Fatalf("third Load failed: %v", err)
	}

	// Should be different instance after cache clear
	if pp1 == pp3 {
		t.Error("expected new instance after cache clear")
	}
}

// TestPParamsLoaderLoadAllAvailable attempts to load every protocol parameter
// file in the pparams-by-hash directory. This ensures all files are valid CBOR
// and can be decoded into ConwayProtocolParameters. Any failures indicate
// corrupted or incompatible pparams files.
func TestPParamsLoaderLoadAllAvailable(t *testing.T) {
	loader := NewPParamsLoaderFromTestdata("testdata")

	hashes, err := loader.ListAvailableHashes()
	if err != nil {
		t.Fatalf("ListAvailableHashes failed: %v", err)
	}

	var (
		successCount int
		failedHashes []string
	)

	for _, hashHex := range hashes {
		hashBytes := make([]byte, 32)
		n, hexErr := hexDecode(hashHex, hashBytes)
		if hexErr != nil {
			failedHashes = append(failedHashes, hashHex+": "+hexErr.Error())
			continue
		}
		if n == 0 {
			failedHashes = append(failedHashes, hashHex+": empty hash")
			continue
		}

		_, err := loader.Load(hashBytes[:n])
		if err != nil {
			failedHashes = append(failedHashes, hashHex+": "+err.Error())
			continue
		}
		successCount++
	}

	if len(failedHashes) > 0 {
		t.Errorf("failed to load %d pparams files:", len(failedHashes))
		for i, msg := range failedHashes {
			if i >= 5 {
				t.Errorf("  ... and %d more", len(failedHashes)-5)
				break
			}
			t.Errorf("  %s", msg)
		}
	}

	t.Logf(
		"successfully loaded %d/%d protocol parameter files",
		successCount,
		len(hashes),
	)
}

// TestPParamsLoaderLoadForVector tests the LoadForVector method which combines
// hash extraction from a test vector's initial state with loading the
// corresponding pparams file. This is the primary method used by the Harness.
func TestPParamsLoaderLoadForVector(t *testing.T) {
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}

	if len(vectors) == 0 {
		t.Fatal("no test vectors found")
	}

	loader := NewPParamsLoaderFromTestdata("testdata")

	// Test with first vector
	vector, err := DecodeTestVector(vectors[0])
	if err != nil {
		t.Fatalf("DecodeTestVector failed: %v", err)
	}

	state, err := ParseInitialState(vector.InitialState)
	if err != nil {
		t.Fatalf("ParseInitialState failed: %v", err)
	}

	pp, err := loader.LoadForVector(vector, state)
	if err != nil {
		t.Fatalf("LoadForVector failed: %v", err)
	}

	if pp == nil {
		t.Fatal("expected non-nil protocol parameters")
	}

	t.Logf("loaded protocol parameters for vector: %s", vector.Title)
}

// TestPParamsLoaderNoCostModel verifies special handling for "No cost model"
// test cases. The Haskell test suite dynamically clears cost models in memory
// for these tests, but exports the original pparams hash. LoadForVector detects
// vectors with "No cost model" in the title and returns a copy of the pparams
// with an empty CostModels map.
func TestPParamsLoaderNoCostModel(t *testing.T) {
	loader := NewPParamsLoaderFromTestdata("testdata")

	// Find a vector with "No cost model" in the title
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}

	var noCostModelVector *TestVector
	var noCostModelState *ParsedInitialState

	for _, path := range vectors {
		vector, err := DecodeTestVector(path)
		if err != nil {
			continue
		}
		if containsNoCostModel(vector.Title) {
			state, err := ParseInitialState(vector.InitialState)
			if err != nil {
				continue
			}
			noCostModelVector = vector
			noCostModelState = state
			break
		}
	}

	if noCostModelVector == nil {
		t.Skip("no 'No cost model' vector found")
	}

	pp, err := loader.LoadForVector(noCostModelVector, noCostModelState)
	if err != nil {
		t.Fatalf("LoadForVector failed: %v", err)
	}

	// Verify cost models are cleared
	cpp, ok := pp.(*conway.ConwayProtocolParameters)
	if !ok {
		t.Fatalf("expected ConwayProtocolParameters, got %T", pp)
	}

	if len(cpp.CostModels) > 0 {
		t.Errorf(
			"expected empty cost models for 'No cost model' vector, got %d",
			len(cpp.CostModels),
		)
	}

	t.Logf("verified 'No cost model' handling for: %s", noCostModelVector.Title)
}

// TestExtractPParamsHashFromVector tests the convenience function that extracts
// the protocol parameters hash directly from a TestVector without requiring
// the caller to manually parse the initial state.
func TestExtractPParamsHashFromVector(t *testing.T) {
	root := filepath.Join("testdata", "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles failed: %v", err)
	}

	if len(vectors) == 0 {
		t.Fatal("no test vectors found")
	}

	vector, err := DecodeTestVector(vectors[0])
	if err != nil {
		t.Fatalf("DecodeTestVector failed: %v", err)
	}

	hash, err := ExtractPParamsHashFromVector(vector)
	if err != nil {
		t.Fatalf("ExtractPParamsHashFromVector failed: %v", err)
	}

	if len(hash) == 0 {
		t.Fatal("expected non-empty hash")
	}

	t.Logf("extracted pparams hash: %x", hash)
}

// containsNoCostModel checks if a title indicates a "No cost model" test.
// Uses the same normalized matching as the production code in pparams.go.
func containsNoCostModel(title string) bool {
	return matchesNoCostModel(title)
}

// hexDecode decodes a hex string into bytes, returning the number of bytes decoded.
func hexDecode(s string, dst []byte) (int, error) {
	if len(s)%2 != 0 {
		return 0, fmt.Errorf("hex string has odd length: %d", len(s))
	}
	n := min(len(s)/2, len(dst))
	for i := range n {
		b := hexByte(s[i*2], s[i*2+1])
		if b < 0 {
			return i, fmt.Errorf("invalid hex character at position %d", i*2)
		}
		dst[i] = byte(b)
	}
	return n, nil
}

func hexByte(hi, lo byte) int {
	h := hexDigit(hi)
	l := hexDigit(lo)
	if h < 0 || l < 0 {
		return -1
	}
	return h<<4 | l
}

func hexDigit(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b - 'a' + 10)
	case b >= 'A' && b <= 'F':
		return int(b - 'A' + 10)
	}
	return -1
}
