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
	_ "embed"
	"encoding/json"
)

// TachysPoolVector describes one pool in a leader selection distribution.
type TachysPoolVector struct {
	PoolID string `json:"pool_id"`
	Stake  uint64 `json:"stake"`
}

// TachysDistributionVector describes a complete active stake distribution.
type TachysDistributionVector struct {
	Pools      []TachysPoolVector `json:"pools"`
	TotalStake uint64             `json:"total_stake"`
}

// TachysSelectionVector describes a nonce and slot selection result.
type TachysSelectionVector struct {
	EpochNonce string `json:"epoch_nonce"`
	Slot       uint64 `json:"slot"`
	HashInput  string `json:"hash_input"`
	Sample     string `json:"sample"`
	Leader     string `json:"leader"`
}

// TachysBoundaryVector describes a direct sample boundary result.
type TachysBoundaryVector struct {
	Name   string `json:"name"`
	Sample string `json:"sample"`
	Leader string `json:"leader"`
}

// TachysOrderingVector describes canonicalization of pool input order.
type TachysOrderingVector struct {
	Name            string             `json:"name"`
	Pools           []TachysPoolVector `json:"pools"`
	ExpectedPoolIDs []string           `json:"expected_pool_ids"`
}

// TachysInvalidVector describes input that a consumer must reject.
type TachysInvalidVector struct {
	Name       string             `json:"name"`
	Pools      []TachysPoolVector `json:"pools,omitempty"`
	TotalStake uint64             `json:"total_stake,omitempty"`
	EpochNonce string             `json:"epoch_nonce,omitempty"`
	Slot       uint64             `json:"slot,omitempty"`
	Sample     string             `json:"sample,omitempty"`
}

// TachysLeaderVectors contains reusable CIP-0177 conformance cases.
type TachysLeaderVectors struct {
	FormatVersion int `json:"format_version"`
	Specification struct {
		Name     string `json:"name"`
		Revision string `json:"revision"`
		Source   string `json:"source"`
	} `json:"specification"`
	Provenance struct {
		Generator string `json:"generator"`
		Method    string `json:"method"`
	} `json:"provenance"`
	Encoding       map[string]string        `json:"encoding"`
	Distribution   TachysDistributionVector `json:"distribution"`
	OrderingCases  []TachysOrderingVector   `json:"ordering_cases"`
	SelectionCases []TachysSelectionVector  `json:"selection_cases"`
	BoundaryCases  []TachysBoundaryVector   `json:"boundary_cases"`
	InvalidCases   []TachysInvalidVector    `json:"invalid_cases"`
}

//go:embed tachys/leader_selection.json
var tachysLeaderVectorsJSON []byte

// LoadTachysLeaderVectors decodes the embedded CIP-0177 vector set.
func LoadTachysLeaderVectors() (TachysLeaderVectors, error) {
	var vectors TachysLeaderVectors
	err := json.Unmarshal(tachysLeaderVectorsJSON, &vectors)
	return vectors, err
}
