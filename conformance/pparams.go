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
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

// PParamsLoader handles loading protocol parameters from test vector files.
type PParamsLoader struct {
	// pparamsDir is the directory containing pparams-by-hash files.
	pparamsDir string

	// cache stores loaded protocol parameters by hash (hex string).
	cache map[string]common.ProtocolParameters
}

// NewPParamsLoader creates a new protocol parameters loader.
// The pparamsDir should be the path to the pparams-by-hash directory.
func NewPParamsLoader(pparamsDir string) *PParamsLoader {
	return &PParamsLoader{
		pparamsDir: pparamsDir,
		cache:      make(map[string]common.ProtocolParameters),
	}
}

// NewPParamsLoaderFromTestdata creates a loader using the default testdata location.
// The testdataRoot should be the conformance/testdata directory.
func NewPParamsLoaderFromTestdata(testdataRoot string) *PParamsLoader {
	pparamsDir := filepath.Join(
		testdataRoot,
		"eras",
		"conway",
		"impl",
		"dump",
		"pparams-by-hash",
	)
	return NewPParamsLoader(pparamsDir)
}

// Load loads protocol parameters for a given hash.
// Returns a deep copy of cached parameters to prevent cross-contamination
// between test vectors that share the same pparams hash.
func (l *PParamsLoader) Load(hash []byte) (common.ProtocolParameters, error) {
	if len(hash) == 0 {
		return nil, &PParamsError{Message: "empty hash"}
	}

	hashHex := hex.EncodeToString(hash)

	// Check cache - if found, return a deep copy to prevent modifications
	// from affecting other vectors using the same pparams hash
	if pp, ok := l.cache[hashHex]; ok {
		return deepCopyPParams(pp), nil
	}

	// Find file
	filePath, err := l.findFile(hashHex)
	if err != nil {
		return nil, err
	}

	// Load and decode
	pp, err := l.loadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Cache and return a copy (keep original in cache, return copy to caller)
	l.cache[hashHex] = pp
	return deepCopyPParams(pp), nil
}

// LoadForVector loads protocol parameters for a test vector.
// It extracts the hash from the initial state and loads the corresponding file.
// If the vector title contains "No cost model", the cost models are cleared.
func (l *PParamsLoader) LoadForVector(
	vector *TestVector,
	state *ParsedInitialState,
) (common.ProtocolParameters, error) {
	if state == nil {
		return nil, &PParamsError{
			Hash:    nil,
			Message: "nil initial state",
		}
	}
	if len(state.PParamsHash) == 0 {
		return nil, &PParamsError{
			Hash:    nil,
			Message: "no protocol parameters hash in initial state",
		}
	}

	pp, err := l.Load(state.PParamsHash)
	if err != nil {
		return nil, err
	}

	// Handle "No cost model" test cases
	// The Haskell test suite modifies pparams in memory via:
	//   modifyPParams $ ppCostModelsL .~ mempty
	// but the test vector export stores the original pparams hash.
	// We simulate this by clearing the cost models for these specific tests.
	// Check for nil vector before accessing title.
	if vector != nil && matchesNoCostModel(vector.Title) {
		pp = clearCostModels(pp)
	}

	return pp, nil
}

// matchesNoCostModel checks if a title indicates a "no cost model" test case.
// Normalizes the title to handle variants like "No cost model", "NoCostModel",
// "no_cost_model", "no-cost-model", etc.
func matchesNoCostModel(title string) bool {
	// Normalize: lowercase and remove non-alphanumeric characters
	normalized := strings.ToLower(title)
	normalized = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1 // remove character
	}, normalized)
	return strings.Contains(normalized, "nocostmodel")
}

// findFile finds the protocol parameters file for a given hash.
func (l *PParamsLoader) findFile(hashHex string) (string, error) {
	entries, err := os.ReadDir(l.pparamsDir)
	if err != nil {
		return "", &PParamsError{
			Hash:    nil,
			Message: "failed to read pparams directory: " + l.pparamsDir,
			Err:     err,
		}
	}

	// Exact match first (most common case)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if e.Name() == hashHex {
			return filepath.Join(l.pparamsDir, e.Name()), nil
		}
	}

	// Fallback: some datasets may have different naming conventions
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.Contains(e.Name(), hashHex) {
			return filepath.Join(l.pparamsDir, e.Name()), nil
		}
	}

	hashBytes, _ := hex.DecodeString(hashHex)
	return "", &PParamsError{
		Hash:    hashBytes,
		Message: "protocol parameters file not found",
	}
}

// loadFile loads and decodes protocol parameters from a file.
func (l *PParamsLoader) loadFile(
	filePath string,
) (common.ProtocolParameters, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, &PParamsError{
			Message: "failed to read file: " + filePath,
			Err:     err,
		}
	}

	// Try Conway first (most common for conformance tests)
	pp := &conway.ConwayProtocolParameters{}
	if _, err := cbor.Decode(data, pp); err != nil {
		return nil, &PParamsError{
			Message: "failed to decode protocol parameters: " + filePath,
			Err:     err,
		}
	}

	return pp, nil
}

// clearCostModels returns a copy of the protocol parameters with cost models cleared.
// This is used for "No cost model" test cases.
func clearCostModels(pp common.ProtocolParameters) common.ProtocolParameters {
	if cpp, ok := pp.(*conway.ConwayProtocolParameters); ok {
		// Create a shallow copy with cleared cost models
		ppCopy := *cpp
		ppCopy.CostModels = make(map[uint][]int64)
		return &ppCopy
	}
	return pp
}

// deepCopyPParams creates a deep copy of protocol parameters.
// This is essential for isolation between test vectors that share the same
// pparams hash, as modifications (e.g., via ParameterChange governance actions)
// should not affect other test vectors.
func deepCopyPParams(pp common.ProtocolParameters) common.ProtocolParameters {
	if pp == nil {
		return nil
	}

	if cpp, ok := pp.(*conway.ConwayProtocolParameters); ok {
		// Create a shallow copy of the struct
		ppCopy := *cpp

		// Deep copy the CostModels map (most critical - this is what gets modified)
		if cpp.CostModels != nil {
			ppCopy.CostModels = make(map[uint][]int64, len(cpp.CostModels))
			for version, model := range cpp.CostModels {
				modelCopy := make([]int64, len(model))
				copy(modelCopy, model)
				ppCopy.CostModels[version] = modelCopy
			}
		}

		// Deep copy A0, Rho, Tau (cbor.Rat pointers containing *big.Rat)
		if cpp.A0 != nil {
			a0Copy := deepCopyRat(*cpp.A0)
			ppCopy.A0 = &a0Copy
		}
		if cpp.Rho != nil {
			rhoCopy := deepCopyRat(*cpp.Rho)
			ppCopy.Rho = &rhoCopy
		}
		if cpp.Tau != nil {
			tauCopy := deepCopyRat(*cpp.Tau)
			ppCopy.Tau = &tauCopy
		}

		// Deep copy MinFeeRefScriptCostPerByte (cbor.Rat pointer containing *big.Rat)
		if cpp.MinFeeRefScriptCostPerByte != nil {
			refCopy := deepCopyRat(*cpp.MinFeeRefScriptCostPerByte)
			ppCopy.MinFeeRefScriptCostPerByte = &refCopy
		}

		// Deep copy ExecutionCosts (contains cbor.Rat pointers with *big.Rat)
		if cpp.ExecutionCosts.MemPrice != nil {
			memPriceCopy := deepCopyRat(*cpp.ExecutionCosts.MemPrice)
			ppCopy.ExecutionCosts.MemPrice = &memPriceCopy
		}
		if cpp.ExecutionCosts.StepPrice != nil {
			stepPriceCopy := deepCopyRat(*cpp.ExecutionCosts.StepPrice)
			ppCopy.ExecutionCosts.StepPrice = &stepPriceCopy
		}

		// Deep copy PoolVotingThresholds (cbor.Rat contains *big.Rat)
		ppCopy.PoolVotingThresholds.MotionNoConfidence = deepCopyRat(
			cpp.PoolVotingThresholds.MotionNoConfidence,
		)
		ppCopy.PoolVotingThresholds.CommitteeNormal = deepCopyRat(
			cpp.PoolVotingThresholds.CommitteeNormal,
		)
		ppCopy.PoolVotingThresholds.CommitteeNoConfidence = deepCopyRat(
			cpp.PoolVotingThresholds.CommitteeNoConfidence,
		)
		ppCopy.PoolVotingThresholds.HardForkInitiation = deepCopyRat(
			cpp.PoolVotingThresholds.HardForkInitiation,
		)
		ppCopy.PoolVotingThresholds.PpSecurityGroup = deepCopyRat(
			cpp.PoolVotingThresholds.PpSecurityGroup,
		)

		// Deep copy DRepVotingThresholds (cbor.Rat contains *big.Rat)
		ppCopy.DRepVotingThresholds.MotionNoConfidence = deepCopyRat(
			cpp.DRepVotingThresholds.MotionNoConfidence,
		)
		ppCopy.DRepVotingThresholds.CommitteeNormal = deepCopyRat(
			cpp.DRepVotingThresholds.CommitteeNormal,
		)
		ppCopy.DRepVotingThresholds.CommitteeNoConfidence = deepCopyRat(
			cpp.DRepVotingThresholds.CommitteeNoConfidence,
		)
		ppCopy.DRepVotingThresholds.UpdateToConstitution = deepCopyRat(
			cpp.DRepVotingThresholds.UpdateToConstitution,
		)
		ppCopy.DRepVotingThresholds.HardForkInitiation = deepCopyRat(
			cpp.DRepVotingThresholds.HardForkInitiation,
		)
		ppCopy.DRepVotingThresholds.PpNetworkGroup = deepCopyRat(
			cpp.DRepVotingThresholds.PpNetworkGroup,
		)
		ppCopy.DRepVotingThresholds.PpEconomicGroup = deepCopyRat(
			cpp.DRepVotingThresholds.PpEconomicGroup,
		)
		ppCopy.DRepVotingThresholds.PpTechnicalGroup = deepCopyRat(
			cpp.DRepVotingThresholds.PpTechnicalGroup,
		)
		ppCopy.DRepVotingThresholds.PpGovGroup = deepCopyRat(
			cpp.DRepVotingThresholds.PpGovGroup,
		)
		ppCopy.DRepVotingThresholds.TreasuryWithdrawal = deepCopyRat(
			cpp.DRepVotingThresholds.TreasuryWithdrawal,
		)

		return &ppCopy
	}

	// For other types, just return as-is (no deep copy support)
	return pp
}

// deepCopyRat creates a deep copy of a cbor.Rat.
// cbor.Rat embeds *big.Rat, so a shallow copy shares the underlying big.Rat.
func deepCopyRat(r cbor.Rat) cbor.Rat {
	if r.Rat == nil {
		return cbor.Rat{}
	}
	return cbor.Rat{Rat: new(big.Rat).Set(r.Rat)}
}

// ListAvailableHashes returns all available protocol parameter hashes.
// Useful for debugging and validation.
func (l *PParamsLoader) ListAvailableHashes() ([]string, error) {
	entries, err := os.ReadDir(l.pparamsDir)
	if err != nil {
		return nil, &PParamsError{
			Message: "failed to read pparams directory: " + l.pparamsDir,
			Err:     err,
		}
	}

	var hashes []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		hashes = append(hashes, e.Name())
	}
	return hashes, nil
}

// ClearCache clears the protocol parameters cache.
func (l *PParamsLoader) ClearCache() {
	l.cache = make(map[string]common.ProtocolParameters)
}

// PParamsError represents an error during protocol parameters loading.
type PParamsError struct {
	Hash    []byte
	Message string
	Err     error
}

func (e *PParamsError) Error() string {
	var parts []string
	if len(e.Hash) > 0 {
		parts = append(parts, fmt.Sprintf("hash=%x", e.Hash))
	}
	parts = append(parts, e.Message)
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, ": ")
}

func (e *PParamsError) Unwrap() error {
	return e.Err
}

// ExtractPParamsHashFromVector extracts the protocol parameters hash from a test vector.
// This is a convenience function that parses the initial state and returns the hash.
func ExtractPParamsHashFromVector(vector *TestVector) ([]byte, error) {
	if vector == nil {
		return nil, errors.New("nil test vector")
	}
	state, err := ParseInitialState(vector.InitialState)
	if err != nil {
		return nil, fmt.Errorf("failed to parse initial state: %w", err)
	}
	if len(state.PParamsHash) == 0 {
		return nil, errors.New("no protocol parameters hash in initial state")
	}
	return state.PParamsHash, nil
}
