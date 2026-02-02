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
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

// Harness runs conformance test vectors against a StateManager implementation.
type Harness struct {
	// testdataRoot is the root directory containing test vectors.
	testdataRoot string

	// stateManager manages ledger state during test execution.
	stateManager StateManager

	// pparamsLoader loads protocol parameters.
	pparamsLoader *PParamsLoader

	// currentSlot tracks the current slot during test execution.
	currentSlot uint64

	// currentEpoch tracks the current epoch during test execution.
	currentEpoch uint64

	// protocolParams holds the current protocol parameters.
	protocolParams common.ProtocolParameters

	// validator performs pre-validation checks.
	validator *Validator

	// debug enables verbose logging.
	debug bool

	// futureWithdrawals contains cumulative withdrawals from each event to end.
	// Used to compute accurate reward balance at each transaction.
	futureWithdrawals []map[common.Blake2b224]uint64

	// finalStateBalances contains reward balances from the final state.
	finalStateBalances map[common.Blake2b224]uint64

	// epochLength is the number of slots per epoch from the vector config.
	epochLength uint64

	// startSlot is the configured start slot from the vector config.
	startSlot uint64

	// initialEpoch is the epoch at startSlot from the vector's initial state.
	// Used to calculate the actual epoch from slot numbers.
	initialEpoch uint64
}

// HarnessConfig configures the test harness.
type HarnessConfig struct {
	// TestdataRoot is the root directory containing test vectors.
	// Defaults to "testdata" if empty.
	TestdataRoot string

	// Debug enables verbose logging.
	Debug bool
}

// NewHarness creates a new test harness with the given state manager.
func NewHarness(stateManager StateManager, config HarnessConfig) *Harness {
	testdataRoot := config.TestdataRoot
	if testdataRoot == "" {
		testdataRoot = "testdata"
	}

	return &Harness{
		testdataRoot:  testdataRoot,
		stateManager:  stateManager,
		pparamsLoader: NewPParamsLoaderFromTestdata(testdataRoot),
		validator:     NewValidator(),
		debug:         config.Debug,
	}
}

// RunAllVectors runs all conformance test vectors.
func (h *Harness) RunAllVectors(t *testing.T) {
	root := filepath.Join(h.testdataRoot, "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("failed to collect vectors: %v", err)
	}

	for _, path := range vectors {
		t.Run(path, func(t *testing.T) {
			h.runVectorFile(t, path)
		})
	}
}

// RunVector runs a single test vector by path.
func (h *Harness) RunVector(t *testing.T, vectorPath string) {
	h.runVectorFile(t, vectorPath)
}

// runVectorFile loads and runs a single test vector file.
func (h *Harness) runVectorFile(t *testing.T, vectorPath string) {
	// Decode the vector
	vector, err := DecodeTestVector(vectorPath)
	if err != nil {
		t.Fatalf("failed to decode vector: %v", err)
	}

	h.runVector(t, vector)
}

// runVector executes a single test vector.
func (h *Harness) runVector(t *testing.T, vector *TestVector) {
	// Reset state manager for this vector
	if err := h.stateManager.Reset(); err != nil {
		t.Fatalf("failed to reset state: %v", err)
	}

	// Parse initial state
	initialState, err := ParseInitialState(vector.InitialState)
	if err != nil {
		t.Fatalf("failed to parse initial state: %v", err)
	}

	// Load protocol parameters
	pp, err := h.pparamsLoader.LoadForVector(vector, initialState)
	if err != nil {
		t.Fatalf("failed to load protocol parameters: %v", err)
	}
	h.protocolParams = pp

	// Initialize state manager with initial state
	if err := h.stateManager.LoadInitialState(initialState, pp); err != nil {
		t.Fatalf("failed to load initial state: %v", err)
	}

	// Parse epoch length from config (index 2 in the config array)
	h.epochLength = parseEpochLength(vector.Config)
	if h.epochLength == 0 {
		t.Fatalf(
			"failed to parse epoch_length from vector config (got 0, config len=%d)",
			len(vector.Config),
		)
	}

	// Parse start slot from config (index 0 in the config array)
	h.startSlot = parseStartSlot(vector.Config)

	// Initialize epoch and slot
	h.currentEpoch = initialState.CurrentEpoch
	h.initialEpoch = initialState.CurrentEpoch
	h.currentSlot = h.startSlot

	// Extract reward balances from final_state and compute future withdrawals
	// This allows accurate withdrawal validation at each transaction
	h.finalStateBalances = extractFinalStateBalances(vector.FinalState)
	h.futureWithdrawals = h.computeFutureWithdrawals(vector.Events)

	// Process events
	for i, event := range vector.Events {
		// Compute adjusted reward balances for this transaction:
		// balance_at_i = final_state_balance + futureWithdrawals[i+1]
		// We use i+1 because futureWithdrawals[i] includes this TX's withdrawal
		if event.Type == EventTypeTransaction && len(h.finalStateBalances) > 0 {
			adjustedBalances := make(map[common.Blake2b224]uint64)
			for cred, balance := range h.finalStateBalances {
				adjustedBalances[cred] = balance + h.futureWithdrawals[i+1][cred]
			}
			h.stateManager.SetRewardBalances(adjustedBalances)
		}

		if err := h.processEvent(t, i, event); err != nil {
			t.Errorf("event %d failed: %v", i, err)
		}
	}
}

// processEvent processes a single event from a test vector.
func (h *Harness) processEvent(
	t *testing.T,
	eventIdx int,
	event VectorEvent,
) error {
	switch event.Type {
	case EventTypeTransaction:
		return h.processTransactionEvent(t, eventIdx, event)
	case EventTypePassTick:
		return h.processPassTickEvent(event)
	case EventTypePassEpoch:
		return h.processPassEpochEvent(event)
	default:
		return fmt.Errorf("unknown event type: %d", event.Type)
	}
}

// processTransactionEvent processes a transaction event.
func (h *Harness) processTransactionEvent(
	t *testing.T,
	eventIdx int,
	event VectorEvent,
) error {
	// Update slot
	h.currentSlot = event.Slot

	// Decode transaction
	tx, err := h.decodeTransaction(event.TxBytes)
	if err != nil {
		if event.Success {
			return fmt.Errorf("failed to decode transaction: %w", err)
		}
		// Expected failure during decode
		return nil
	}

	// Execute and validate transaction
	success, execErr := h.executeTransaction(tx, event.Slot)

	// Compare result with expected
	if success && !event.Success {
		if h.debug {
			t.Logf(
				"tx %d: expected failure but got success (IsValid=%v)",
				eventIdx,
				tx.IsValid(),
			)
		}
		return errors.New("expected failure but got success")
	}

	if !success && event.Success {
		if h.debug {
			t.Logf(
				"tx %d: expected success but got failure: %v",
				eventIdx,
				execErr,
			)
		}
		return fmt.Errorf("expected success but got failure: %w", execErr)
	}

	// If successful, apply state changes
	if success {
		if err := h.stateManager.ApplyTransaction(tx, event.Slot); err != nil {
			return fmt.Errorf("failed to apply transaction: %w", err)
		}
	}

	return nil
}

// processPassTickEvent processes a pass tick event.
func (h *Harness) processPassTickEvent(event VectorEvent) error {
	h.currentSlot = event.TickSlot
	return nil
}

// processPassEpochEvent processes a pass epoch event.
func (h *Harness) processPassEpochEvent(event VectorEvent) error {
	// Advance epoch
	h.currentEpoch += event.EpochDelta

	// Process epoch boundary in state manager
	if err := h.stateManager.ProcessEpochBoundary(h.currentEpoch); err != nil {
		return fmt.Errorf("failed to process epoch boundary: %w", err)
	}

	// Update protocol parameters in case any ParameterChange proposals were enacted
	h.protocolParams = h.stateManager.GetProtocolParameters()

	return nil
}

// decodeTransaction decodes a transaction from CBOR bytes.
func (h *Harness) decodeTransaction(
	txBytes []byte,
) (common.Transaction, error) {
	tx := &conway.ConwayTransaction{}
	if _, err := cbor.Decode(txBytes, tx); err != nil {
		return nil, fmt.Errorf("failed to decode transaction: %w", err)
	}
	return tx, nil
}

// executeTransaction validates a transaction against current state.
func (h *Harness) executeTransaction(
	tx common.Transaction,
	slot uint64,
) (bool, error) {
	govState := h.stateManager.GetGovernanceState()

	// Calculate epoch from slot for accurate expiration checks.
	// The slot-based epoch is more accurate than tracking PassEpoch events,
	// especially for governance action expiration validation.
	// Use epoch length from vector config (parsed in runVector).
	// Add initialEpoch since vectors may start at a non-zero epoch.
	if slot < h.startSlot {
		return false, fmt.Errorf(
			"slot %d is before configured start slot %d",
			slot,
			h.startSlot,
		)
	}
	slotBasedEpoch := h.initialEpoch + (slot-h.startSlot)/h.epochLength

	// Phase 1: Pre-validation checks
	if err := h.validator.ValidateTransaction(tx, slot, slotBasedEpoch, govState, h.protocolParams); err != nil {
		return false, err
	}

	// Phase 2: Core ledger validation
	// Use ConformanceValidationRules which excludes fee/size validation
	// because test vectors have pre-computed values from Haskell
	stateProvider := h.stateManager.GetStateProvider()
	err := common.VerifyTransaction(
		tx,
		slot,
		stateProvider,
		h.protocolParams,
		ConformanceValidationRules,
	)
	if err != nil {
		return false, err
	}

	return true, nil
}

// VectorResult contains the result of running a test vector.
type VectorResult struct {
	// Title is the human-readable title from the test vector.
	Title string

	// Path is the file path of the test vector.
	Path string

	// Success indicates whether all events in the vector passed.
	Success bool

	// Error contains the error if the vector failed, nil otherwise.
	Error error

	// EventCount is the total number of events in the vector.
	EventCount int

	// FailedEvent is the index of the first failed event (-1 if Success is true).
	FailedEvent int
}

// RunAllVectorsWithResults runs all vectors and returns detailed results.
func (h *Harness) RunAllVectorsWithResults() ([]VectorResult, error) {
	root := filepath.Join(h.testdataRoot, "eras")
	vectors, err := CollectVectorFiles(root)
	if err != nil {
		return nil, fmt.Errorf("failed to collect vectors: %w", err)
	}

	results := make([]VectorResult, 0, len(vectors))

	for _, path := range vectors {
		result := h.runVectorWithResult(path)
		results = append(results, result)
	}

	return results, nil
}

// runVectorWithResult runs a single vector and returns the result.
func (h *Harness) runVectorWithResult(vectorPath string) VectorResult {
	result := VectorResult{
		Path:        vectorPath,
		FailedEvent: -1,
	}

	// Decode the vector
	vector, err := DecodeTestVector(vectorPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to decode vector: %w", err)
		return result
	}
	result.Title = vector.Title
	result.EventCount = len(vector.Events)

	// Reset state manager
	if err := h.stateManager.Reset(); err != nil {
		result.Error = fmt.Errorf("failed to reset state: %w", err)
		return result
	}

	// Parse initial state
	initialState, err := ParseInitialState(vector.InitialState)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse initial state: %w", err)
		return result
	}

	// Load protocol parameters
	pp, err := h.pparamsLoader.LoadForVector(vector, initialState)
	if err != nil {
		result.Error = fmt.Errorf("failed to load protocol parameters: %w", err)
		return result
	}
	h.protocolParams = pp

	// Initialize state manager
	if err := h.stateManager.LoadInitialState(initialState, pp); err != nil {
		result.Error = fmt.Errorf("failed to load initial state: %w", err)
		return result
	}

	// Parse epoch length from config (index 2 in the config array)
	h.epochLength = parseEpochLength(vector.Config)
	if h.epochLength == 0 {
		result.Error = fmt.Errorf(
			"failed to parse epoch_length from vector config (got 0, config len=%d)",
			len(vector.Config),
		)
		return result
	}

	// Parse start slot from config (index 0 in the config array)
	h.startSlot = parseStartSlot(vector.Config)

	// Initialize epoch and slot
	h.currentEpoch = initialState.CurrentEpoch
	h.initialEpoch = initialState.CurrentEpoch
	h.currentSlot = h.startSlot

	// Extract reward balances from final_state and compute future withdrawals
	// This allows accurate withdrawal validation at each transaction
	h.finalStateBalances = extractFinalStateBalances(vector.FinalState)
	h.futureWithdrawals = h.computeFutureWithdrawals(vector.Events)

	// Process events
	for i, event := range vector.Events {
		// Compute adjusted reward balances for this transaction:
		// balance_at_i = final_state_balance + futureWithdrawals[i+1]
		// We use i+1 because futureWithdrawals[i] includes this TX's withdrawal
		if event.Type == EventTypeTransaction && len(h.finalStateBalances) > 0 {
			adjustedBalances := make(map[common.Blake2b224]uint64)
			for cred, balance := range h.finalStateBalances {
				adjustedBalances[cred] = balance + h.futureWithdrawals[i+1][cred]
			}
			h.stateManager.SetRewardBalances(adjustedBalances)
		}

		if err := h.processEventWithoutT(i, event); err != nil {
			result.Error = err
			result.FailedEvent = i
			return result
		}
	}

	result.Success = true
	return result
}

// processEventWithoutT processes an event without a testing.T.
func (h *Harness) processEventWithoutT(eventIdx int, event VectorEvent) error {
	switch event.Type {
	case EventTypeTransaction:
		return h.processTransactionEventWithoutT(eventIdx, event)
	case EventTypePassTick:
		return h.processPassTickEvent(event)
	case EventTypePassEpoch:
		return h.processPassEpochEvent(event)
	default:
		return fmt.Errorf("unknown event type: %d", event.Type)
	}
}

// processTransactionEventWithoutT processes a transaction event without a testing.T.
func (h *Harness) processTransactionEventWithoutT(
	eventIdx int,
	event VectorEvent,
) error {
	h.currentSlot = event.Slot

	tx, err := h.decodeTransaction(event.TxBytes)
	if err != nil {
		if event.Success {
			return fmt.Errorf("tx %d: failed to decode: %w", eventIdx, err)
		}
		return nil
	}

	success, execErr := h.executeTransaction(tx, event.Slot)

	if success && !event.Success {
		return fmt.Errorf("tx %d: expected failure but got success", eventIdx)
	}

	if !success && event.Success {
		return fmt.Errorf(
			"tx %d: expected success but got failure: %w",
			eventIdx,
			execErr,
		)
	}

	if success {
		if err := h.stateManager.ApplyTransaction(tx, event.Slot); err != nil {
			return fmt.Errorf("tx %d: failed to apply: %w", eventIdx, err)
		}
	}

	return nil
}

// computeFutureWithdrawals computes cumulative withdrawals from each TX index to the end.
// Returns a slice where futureWithdrawals[i] contains the sum of successful withdrawals
// from events[i] to the end (inclusive). This allows computing balance at TX i as:
// balance_at_i = final_state_balance + futureWithdrawals[i+1]
func (h *Harness) computeFutureWithdrawals(
	events []VectorEvent,
) []map[common.Blake2b224]uint64 {
	n := len(events)
	result := make([]map[common.Blake2b224]uint64, n+1)

	// Initialize the last entry (after all events) to empty
	result[n] = make(map[common.Blake2b224]uint64)

	// Work backwards from the end
	for i := n - 1; i >= 0; i-- {
		// Copy previous (next in order) cumulative
		result[i] = make(map[common.Blake2b224]uint64)
		maps.Copy(result[i], result[i+1])

		event := events[i]
		if event.Type != EventTypeTransaction || !event.Success {
			continue
		}

		tx, err := h.decodeTransaction(event.TxBytes)
		if err != nil || tx == nil {
			continue
		}

		// Skip withdrawals for phase-2 invalid transactions (IsValid=false)
		// These transactions are accepted but their effects are reverted
		if !tx.IsValid() {
			continue
		}

		for addr, amount := range tx.Withdrawals() {
			if amount == nil {
				continue
			}
			withdrawAmount := amount.Uint64()
			if withdrawAmount == 0 {
				continue
			}
			credHash := addr.StakeKeyHash()
			result[i][credHash] += withdrawAmount
		}
	}

	return result
}

// extractFinalStateBalances extracts reward account balances from final_state.
// These balances reflect state AFTER all transactions have been applied.
// Structure: final_state[3][1][0][2][0][0] = stake credentials map
func extractFinalStateBalances(
	finalState cbor.RawMessage,
) map[common.Blake2b224]uint64 {
	result := make(map[common.Blake2b224]uint64)

	// Navigate: final_state[3] = begin_epoch_state
	var stateArr []cbor.RawMessage
	if _, err := cbor.Decode(finalState, &stateArr); err != nil {
		return result
	}
	if len(stateArr) < 4 {
		return result
	}

	// bes[1] = ledger_state
	var bes []cbor.RawMessage
	if _, err := cbor.Decode(stateArr[3], &bes); err != nil {
		return result
	}
	if len(bes) < 2 {
		return result
	}

	// ls[0] = cert_state
	var ls []cbor.RawMessage
	if _, err := cbor.Decode(bes[1], &ls); err != nil {
		return result
	}
	if len(ls) < 1 {
		return result
	}

	// cert_state[2] = delegation_state
	var certState []cbor.RawMessage
	if _, err := cbor.Decode(ls[0], &certState); err != nil {
		return result
	}
	if len(certState) < 3 {
		return result
	}

	// dstate = [unified_map_wrapper, ...]
	var dstate []cbor.RawMessage
	if _, err := cbor.Decode(certState[2], &dstate); err != nil {
		return result
	}
	if len(dstate) < 1 {
		return result
	}

	// dstate[0] is an array [stake_creds_map, ...]
	var ds0 []cbor.RawMessage
	if _, err := cbor.Decode(dstate[0], &ds0); err != nil {
		return result
	}
	if len(ds0) < 1 {
		return result
	}

	// ds0[0] is the stake credentials map - parse manually due to non-hashable CBOR map keys
	rawMap := []byte(ds0[0])
	entries := parseStakeCredentialMap(rawMap)

	for _, entry := range entries {
		credHash := common.NewBlake2b224(entry.Hash)
		result[credHash] = entry.Balance
	}

	return result
}

// stakeCredEntry represents a parsed stake credential map entry
type stakeCredEntry struct {
	CredType uint64
	Hash     []byte
	Balance  uint64
}

// parseStakeCredentialMap manually parses a CBOR map with credential keys
// because Go's CBOR library wraps non-hashable keys in pointers
func parseStakeCredentialMap(data []byte) []stakeCredEntry {
	if len(data) == 0 {
		return nil
	}

	pos := 0
	// Check for map major type (0xa0-0xbf for small maps, 0xb9 for 2-byte length)
	major := data[pos] & 0xe0
	if major != 0xa0 {
		return nil
	}

	info := data[pos] & 0x1f
	var mapLen int
	if info < 24 {
		mapLen = int(info)
		pos++
	} else if info == 24 {
		if pos+1 >= len(data) {
			return nil
		}
		mapLen = int(data[pos+1])
		pos += 2
	} else if info == 25 {
		if pos+2 >= len(data) {
			return nil
		}
		mapLen = int(data[pos+1])<<8 | int(data[pos+2])
		pos += 3
	} else {
		// Indefinite length not supported
		return nil
	}

	var entries []stakeCredEntry
	for i := 0; i < mapLen; i++ {
		entry, newPos := parseStakeCredEntry(data, pos)
		if entry != nil {
			entries = append(entries, *entry)
		}
		pos = newPos
	}

	return entries
}

func parseStakeCredEntry(data []byte, pos int) (*stakeCredEntry, int) {
	if pos >= len(data) {
		return nil, pos
	}

	// Parse key (credential = [type, hash])
	if data[pos]&0xe0 != 0x80 { // Array major type
		return nil, skipCborItem(data, pos)
	}
	keyArrayLen := int(data[pos] & 0x1f)
	pos++
	if keyArrayLen != 2 {
		for range keyArrayLen {
			pos = skipCborItem(data, pos)
		}
		pos = skipCborItem(data, pos)
		return nil, pos
	}

	// Parse credential type (uint)
	credType, n := parseCborUint(data, pos)
	pos += n

	// Parse hash (byte string)
	hash, n := parseCborBytes(data, pos)
	pos += n
	if len(hash) != 28 {
		pos = skipCborItem(data, pos)
		return nil, pos
	}

	// Parse value (array: [rewards, deposit, drep_delegatee, pool_delegatee])
	if pos >= len(data) {
		return nil, pos
	}
	if data[pos]&0xe0 != 0x80 {
		pos = skipCborItem(data, pos)
		return nil, pos
	}
	valArrayLen := int(data[pos] & 0x1f)
	pos++
	if valArrayLen < 1 {
		return nil, pos
	}

	// First element is rewards: [[epoch, balance], ...]
	if pos >= len(data) {
		return nil, pos
	}
	if data[pos]&0xe0 != 0x80 {
		for range valArrayLen {
			pos = skipCborItem(data, pos)
		}
		return nil, pos
	}
	rewardsLen := int(data[pos] & 0x1f)
	pos++

	var balance uint64
	if rewardsLen > 0 {
		// Parse first reward entry [epoch, balance]
		if pos >= len(data) {
			return nil, pos
		}
		if data[pos]&0xe0 == 0x80 {
			rewardEntryLen := int(data[pos] & 0x1f)
			pos++
			if rewardEntryLen >= 2 {
				// Skip epoch
				_, n := parseCborUint(data, pos)
				pos += n
				// Parse balance
				balance, n = parseCborUint(data, pos)
				pos += n
				// Skip remaining elements
				for k := 2; k < rewardEntryLen; k++ {
					pos = skipCborItem(data, pos)
				}
			}
		}
		// Skip remaining reward entries
		for j := 1; j < rewardsLen; j++ {
			pos = skipCborItem(data, pos)
		}
	}

	// Skip remaining value elements (deposit, drep_delegatee, pool_delegatee)
	for j := 1; j < valArrayLen; j++ {
		pos = skipCborItem(data, pos)
	}

	return &stakeCredEntry{
		CredType: credType,
		Hash:     hash,
		Balance:  balance,
	}, pos
}

func parseCborUint(data []byte, pos int) (uint64, int) {
	if pos >= len(data) {
		return 0, 0
	}
	major := data[pos] & 0xe0
	info := data[pos] & 0x1f

	// Check for tag (e.g., tag 258 for set)
	if major == 0xc0 {
		var tagLen int
		switch info {
		case 24:
			tagLen = 2
		case 25:
			tagLen = 3
		default:
			tagLen = 1
		}
		// Ensure we have enough bytes for the tag before recursing
		if pos+tagLen > len(data) {
			return 0, 0
		}
		v, n := parseCborUint(data, pos+tagLen)
		return v, tagLen + n
	}

	if major != 0x00 {
		return 0, 1
	}

	if info < 24 {
		return uint64(info), 1
	} else if info == 24 {
		if pos+2 > len(data) {
			return 0, 0
		}
		return uint64(data[pos+1]), 2
	} else if info == 25 {
		if pos+3 > len(data) {
			return 0, 0
		}
		return uint64(data[pos+1])<<8 | uint64(data[pos+2]), 3
	} else if info == 26 {
		if pos+5 > len(data) {
			return 0, 0
		}
		return uint64(data[pos+1])<<24 | uint64(data[pos+2])<<16 | uint64(data[pos+3])<<8 | uint64(data[pos+4]), 5
	} else if info == 27 {
		if pos+9 > len(data) {
			return 0, 0
		}
		var v uint64
		for i := range 8 {
			v = v<<8 | uint64(data[pos+1+i])
		}
		return v, 9
	}
	return 0, 1
}

func parseCborBytes(data []byte, pos int) ([]byte, int) {
	if pos >= len(data) {
		return nil, 1 // Return 1 to allow forward progress on malformed data
	}
	major := data[pos] & 0xe0
	info := data[pos] & 0x1f

	if major != 0x40 {
		return nil, 1
	}

	var length int
	var headerLen int
	if info < 24 {
		length = int(info)
		headerLen = 1
	} else if info == 24 {
		if pos+2 > len(data) {
			return nil, 1 // Return 1 to allow forward progress
		}
		length = int(data[pos+1])
		headerLen = 2
	} else if info == 25 {
		if pos+3 > len(data) {
			return nil, 1 // Return 1 to allow forward progress
		}
		length = int(data[pos+1])<<8 | int(data[pos+2])
		headerLen = 3
	} else {
		return nil, 1
	}

	if pos+headerLen+length > len(data) {
		return nil, headerLen // Return header length to skip past malformed item
	}

	return data[pos+headerLen : pos+headerLen+length], headerLen + length
}

func skipCborItem(data []byte, pos int) int {
	if pos >= len(data) {
		return pos
	}
	major := data[pos] & 0xe0
	info := data[pos] & 0x1f

	switch major {
	case 0x00, 0x20: // Positive/negative int
		if info < 24 {
			return pos + 1
		} else if info == 24 {
			return pos + 2
		} else if info == 25 {
			return pos + 3
		} else if info == 26 {
			return pos + 5
		} else if info == 27 {
			return pos + 9
		}
		return pos + 1
	case 0x40, 0x60: // Byte/text string
		var length, headerLen int
		if info < 24 {
			length = int(info)
			headerLen = 1
		} else if info == 24 {
			if pos+1 >= len(data) {
				return pos + 1
			}
			length = int(data[pos+1])
			headerLen = 2
		} else if info == 25 {
			if pos+2 >= len(data) {
				return pos + 1
			}
			length = int(data[pos+1])<<8 | int(data[pos+2])
			headerLen = 3
		} else {
			return pos + 1
		}
		if pos+headerLen+length > len(data) {
			return pos + 1
		}
		return pos + headerLen + length
	case 0x80: // Array
		pos++
		var length int
		if info < 24 {
			length = int(info)
		} else if info == 24 {
			if pos >= len(data) {
				return pos
			}
			length = int(data[pos])
			pos++
		}
		for i := 0; i < length; i++ {
			pos = skipCborItem(data, pos)
		}
		return pos
	case 0xa0: // Map
		pos++
		var length int
		if info < 24 {
			length = int(info)
		} else if info == 24 {
			if pos >= len(data) {
				return pos
			}
			length = int(data[pos])
			pos++
		}
		for i := 0; i < length*2; i++ {
			pos = skipCborItem(data, pos)
		}
		return pos
	case 0xc0: // Tag
		// Skip tag and its content
		if info < 24 {
			return skipCborItem(data, pos+1)
		} else if info == 24 {
			return skipCborItem(data, pos+2)
		} else if info == 25 {
			return skipCborItem(data, pos+3)
		}
		return skipCborItem(data, pos+1)
	case 0xe0: // Simple/float
		if info < 24 {
			return pos + 1
		} else if info == 24 {
			return pos + 2
		} else if info == 25 {
			return pos + 3
		} else if info == 26 {
			return pos + 5
		} else if info == 27 {
			return pos + 9
		}
		return pos + 1
	}
	return pos + 1
}

// parseEpochLength extracts the epoch_length from the config array.
// Config structure: [start_slot, slot_length, epoch_length, ...]
// Returns 0 if parsing fails.
func parseEpochLength(config []byte) uint64 {
	if len(config) < 2 {
		return 0
	}
	// Check for array header
	if config[0]&0xe0 != 0x80 {
		return 0
	}
	pos := 1
	info := config[0] & 0x1f
	switch info {
	case 24:
		pos = 2
	case 25:
		pos = 3
	}

	// Skip first 2 elements (start_slot, slot_length)
	for range 2 {
		pos = skipCborItem(config, pos)
		if pos >= len(config) {
			return 0
		}
	}

	// Parse epoch_length at index 2
	epochLength, _ := parseCborUint(config, pos)
	return epochLength
}

// parseStartSlot extracts the start_slot from the config array.
// Config structure: [start_slot, slot_length, epoch_length, ...]
// Returns 0 if parsing fails.
func parseStartSlot(config []byte) uint64 {
	if len(config) < 2 {
		return 0
	}
	// Check for array header
	if config[0]&0xe0 != 0x80 {
		return 0
	}
	pos := 1
	info := config[0] & 0x1f
	switch info {
	case 24:
		pos = 2
	case 25:
		pos = 3
	}

	startSlot, _ := parseCborUint(config, pos)
	return startSlot
}
