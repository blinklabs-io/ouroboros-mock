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

// Package conformance provides a shared test harness for Cardano ledger
// conformance tests using Amaru test vectors.
package conformance

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/blinklabs-io/gouroboros/cbor"
)

// EventType represents the type of event in a test vector.
type EventType int

const (
	// EventTypeTransaction represents a transaction event.
	// Format: [0, tx_cbor:bytes, success:bool, slot:uint64]
	EventTypeTransaction EventType = 0

	// EventTypePassTick represents a slot advancement event.
	// Format: [1, slot:uint64]
	EventTypePassTick EventType = 1

	// EventTypePassEpoch represents an epoch advancement event.
	// Format: [2, epoch_delta:uint64]
	EventTypePassEpoch EventType = 2
)

// VectorEvent represents an event in a test vector.
type VectorEvent struct {
	Type EventType

	// Transaction event fields (Type == EventTypeTransaction)
	TxBytes []byte
	Success bool
	Slot    uint64

	// PassTick event fields (Type == EventTypePassTick)
	TickSlot uint64

	// PassEpoch event fields (Type == EventTypePassEpoch)
	EpochDelta uint64
}

// TestVector represents a parsed conformance test vector.
type TestVector struct {
	// Title is the test name/path from the vector.
	Title string

	// Config is the raw CBOR-encoded network/protocol configuration.
	Config cbor.RawMessage

	// InitialState is the raw CBOR-encoded NewEpochState before events.
	InitialState cbor.RawMessage

	// FinalState is the raw CBOR-encoded NewEpochState after events.
	FinalState cbor.RawMessage

	// Events is the list of transaction/epoch events to process.
	Events []VectorEvent

	// FilePath is the path to the vector file (for debugging).
	FilePath string
}

// CollectVectorFiles walks the testdata directory and returns all vector file paths.
// It skips pparams-by-hash directories, scripts directories, and non-vector files.
func CollectVectorFiles(root string) ([]string, error) {
	var vectors []string
	err := filepath.WalkDir(
		root,
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Normalize path separators for cross-platform compatibility
			normalizedPath := filepath.ToSlash(path)
			if entry.IsDir() {
				// Skip protocol parameters directory
				if strings.Contains(normalizedPath, "pparams-by-hash") {
					return filepath.SkipDir
				}
				return nil
			}
			// Skip files in special directories
			if strings.Contains(normalizedPath, "pparams-by-hash") ||
				strings.Contains(normalizedPath, "scripts/") {
				return nil
			}
			// Skip documentation files
			baseName := filepath.Base(path)
			if baseName == "README" || strings.HasSuffix(baseName, ".md") {
				return nil
			}
			vectors = append(vectors, path)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	sort.Strings(vectors)
	return vectors, nil
}

// DecodeTestVector reads and decodes a test vector from a file.
// The vector is a 5-element CBOR array:
//
//	[0] config:        array[13]  - Network/protocol configuration
//	[1] initial_state: array[7]   - NewEpochState before events
//	[2] final_state:   array[7]   - NewEpochState after events
//	[3] events:        array[N]   - Transaction/epoch events
//	[4] title:         string     - Test name/path
func DecodeTestVector(vectorPath string) (*TestVector, error) {
	data, err := os.ReadFile(vectorPath)
	if err != nil {
		return nil, &VectorError{
			Path:    vectorPath,
			Message: "failed to read vector file",
			Err:     err,
		}
	}

	var items []cbor.RawMessage
	if _, err := cbor.Decode(data, &items); err != nil {
		return nil, &VectorError{
			Path:    vectorPath,
			Message: "failed to decode CBOR array",
			Err:     err,
		}
	}

	if len(items) < 5 {
		return nil, &VectorError{
			Path:    vectorPath,
			Message: "unexpected vector structure: expected 5 elements",
		}
	}

	var title string
	if _, err := cbor.Decode(items[4], &title); err != nil {
		return nil, &VectorError{
			Path:    vectorPath,
			Message: "failed to decode title",
			Err:     err,
		}
	}

	events, err := decodeEvents(items[3])
	if err != nil {
		return nil, &VectorError{
			Path:    vectorPath,
			Message: "failed to decode events",
			Err:     err,
		}
	}

	return &TestVector{
		Title:        title,
		Config:       items[0],
		InitialState: items[1],
		FinalState:   items[2],
		Events:       events,
		FilePath:     vectorPath,
	}, nil
}

// decodeEvents decodes the events array from a test vector.
func decodeEvents(raw cbor.RawMessage) ([]VectorEvent, error) {
	var encodedEvents []cbor.RawMessage
	if _, err := cbor.Decode(raw, &encodedEvents); err != nil {
		return nil, &EventError{
			Message: "failed to decode events list",
			Err:     err,
		}
	}

	events := make([]VectorEvent, 0, len(encodedEvents))
	for i, rawEvent := range encodedEvents {
		var payload []any
		if _, err := cbor.Decode(rawEvent, &payload); err != nil {
			return nil, &EventError{
				Index:   i,
				Message: "failed to decode event",
				Err:     err,
			}
		}
		if len(payload) == 0 {
			return nil, &EventError{
				Index:   i,
				Message: "empty event payload",
			}
		}

		variant, ok := payload[0].(uint64)
		if !ok {
			return nil, &EventError{
				Index:   i,
				Message: "unexpected event type",
			}
		}

		//nolint:gosec // variant bounded by CBOR event types
		event, err := decodeEvent(EventType(variant), payload, i)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// decodeEvent decodes a single event from its payload.
func decodeEvent(
	eventType EventType,
	payload []any,
	index int,
) (VectorEvent, error) {
	switch eventType {
	case EventTypeTransaction:
		return decodeTransactionEvent(payload, index)
	case EventTypePassTick:
		return decodePassTickEvent(payload, index)
	case EventTypePassEpoch:
		return decodePassEpochEvent(payload, index)
	default:
		return VectorEvent{}, &EventError{
			Index:   index,
			Message: "unknown event type",
		}
	}
}

// decodeTransactionEvent decodes a transaction event: [0, txBytes, success, slot]
func decodeTransactionEvent(payload []any, index int) (VectorEvent, error) {
	if len(payload) < 4 {
		return VectorEvent{}, &EventError{
			Index:   index,
			Message: "transaction event missing fields",
		}
	}

	txBytes, ok := payload[1].([]byte)
	if !ok {
		return VectorEvent{}, &EventError{
			Index:   index,
			Message: "unexpected tx bytes type",
		}
	}

	success, ok := payload[2].(bool)
	if !ok {
		return VectorEvent{}, &EventError{
			Index:   index,
			Message: "unexpected success flag type",
		}
	}

	slot, ok := payload[3].(uint64)
	if !ok {
		return VectorEvent{}, &EventError{
			Index:   index,
			Message: "unexpected slot type",
		}
	}

	return VectorEvent{
		Type:    EventTypeTransaction,
		TxBytes: txBytes,
		Success: success,
		Slot:    slot,
	}, nil
}

// decodePassTickEvent decodes a pass tick event: [1, slot]
func decodePassTickEvent(payload []any, index int) (VectorEvent, error) {
	if len(payload) < 2 {
		return VectorEvent{}, &EventError{
			Index:   index,
			Message: "PassTick event missing slot field",
		}
	}

	tickSlot, ok := payload[1].(uint64)
	if !ok {
		return VectorEvent{}, &EventError{
			Index:   index,
			Message: "unexpected PassTick slot type",
		}
	}

	return VectorEvent{
		Type:     EventTypePassTick,
		TickSlot: tickSlot,
	}, nil
}

// decodePassEpochEvent decodes a pass epoch event: [2, epochDelta]
func decodePassEpochEvent(payload []any, index int) (VectorEvent, error) {
	if len(payload) < 2 {
		return VectorEvent{}, &EventError{
			Index:   index,
			Message: "PassEpoch event missing epoch field",
		}
	}

	epochDelta, ok := payload[1].(uint64)
	if !ok {
		return VectorEvent{}, &EventError{
			Index:   index,
			Message: "unexpected PassEpoch epoch type",
		}
	}

	return VectorEvent{
		Type:       EventTypePassEpoch,
		EpochDelta: epochDelta,
	}, nil
}

// VectorError represents an error during vector parsing.
type VectorError struct {
	Path    string
	Message string
	Err     error
}

func (e *VectorError) Error() string {
	if e.Err != nil {
		return e.Path + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Path + ": " + e.Message
}

func (e *VectorError) Unwrap() error {
	return e.Err
}

// EventError represents an error during event parsing.
type EventError struct {
	Index   int
	Message string
	Err     error
}

func (e *EventError) Error() string {
	prefix := fmt.Sprintf("event %d: ", e.Index)
	if e.Err != nil {
		return prefix + e.Message + ": " + e.Err.Error()
	}
	return prefix + e.Message
}

func (e *EventError) Unwrap() error {
	return e.Err
}
