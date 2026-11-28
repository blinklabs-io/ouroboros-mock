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

package benchmark

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type caseCoverage struct {
	families    map[string]bool
	observables map[string]bool
	quality     bool
}
type evidenceCase struct {
	Inputs         map[string]string `json:"inputs"`
	ExpectedSHA256 string            `json:"expected_sha256"`
}

var resultFields = [][]string{
	{"seconds"},
	{"seconds"},
	{"seconds"},
	{"seconds"},
	{"seconds"},
	{"seconds"},
	{"seconds"},
	{"seconds"},
	{"announced_seconds", "sending_seconds", "adopted_seconds"},
	{"first_notice_seconds", "first_fetch_seconds"},
	{"seconds"},
	{"seconds", "range_length"},
	{"announced_seconds", "sending_seconds", "adopted_seconds"},
	{"eligible", "adopted", "missing", "p50_seconds", "p75_seconds"},
	{"cpu_percent_one_core", "rss_bytes", "threads"},
	{"count"},
	{"seconds"},
	{"inside_live_bytes", "outside_live_bytes"},
	{"processed", "accepted", "rejected"},
	{"ratio"},
	{"cpu_seconds_per_processed"},
	{"peak_count", "peak_bytes", "final_count", "final_bytes", "validation_rejections"},
	{"samples", "last_slot", "last_utxo_size", "last_density"},
}

var observableFamilies = map[string][]int{
	"Resources": {14}, "Forge.Loop.StartLeadershipCheck": {0}, "Forge.Loop.BlockContext": {1}, "Forge.Loop.LedgerState": {2}, "Forge.Loop.LedgerView": {3},
	"Forge.Loop.NodeIsLeader": {4}, "Forge.Loop.NodeNotLeader": {4}, "Forge.Loop.ForgeTickedLedgerState": {5}, "Forge.Loop.ForgingMempoolSnapshot": {6},
	"Forge.Loop.ForgedBlock": {7, 8}, "Forge.Loop.AdoptedBlock": {8}, "ChainSync.Client.DownloadedHeader": {9, 10}, "BlockFetch.Client.SendFetchRequest": {10, 11},
	"BlockFetch.Client.CompletedBlockFetch": {11, 12}, "ChainSync.ServerHeader.Update": {8, 12}, "BlockFetch.Server.SendBlock": {8, 12},
	"ChainDB.AddBlockEvent.AddedToCurrentChain": {13}, "ChainDB.AddBlockEvent.SwitchedToAFork": {13}, "TxSubmission.TxInbound.Collected": {18},
	"TxSubmission.TxInbound.Processed": {18}, "Mempool.AddedTx": {21}, "Mempool.RemoveTxs": {21}, "Mempool.RejectedTx": {21},
	"ChainDB.LedgerEvent.Snapshot.TookSnapshot": {15, 16}, "LedgerMetrics": {22},
}

func (c caseCoverage) supports(measurement string) bool {
	return c.families[measurement] || c.observables[measurement]
}

func (c caseCoverage) metricFor(observable string) bool {
	for _, i := range observableFamilies[observable] {
		if c.families[derivedNames[i]] {
			return true
		}
	}
	return false
}

func (b *Bundle) validateEvidence(e Expectation, o Oracle, value json.RawMessage) (caseCoverage, error) {
	var result map[string]json.RawMessage
	if err := json.Unmarshal(value, &result); err != nil || len(result) == 0 {
		return caseCoverage{}, fmt.Errorf("case %s needs a structured expected outcome", e.ID)
	}
	raw, err := b.Read(o.Result)
	if err != nil {
		return caseCoverage{}, err
	}
	var evidence struct {
		Version  int                     `json:"version"`
		Evidence map[string]evidenceCase `json:"evidence"`
	}
	if err = uniqueJSON(raw); err != nil {
		return caseCoverage{}, err
	}
	if err = json.Unmarshal(raw, &evidence); err != nil {
		return caseCoverage{}, err
	}
	binding, ok := evidence.Evidence[e.ID]
	sum := sha256.Sum256(value)
	if evidence.Version != SchemaVersion || !ok || binding.ExpectedSHA256 != hex.EncodeToString(sum[:]) {
		return caseCoverage{}, fmt.Errorf("case %s lacks matching versioned oracle evidence", e.ID)
	}
	kinds := map[string]bool{}
	inputs := map[string]bool{}
	for _, id := range e.Inputs {
		a := b.artifacts[id]
		kinds[a.Kind] = true
		inputs[id] = true
		if binding.Inputs[id] != a.SHA256 {
			return caseCoverage{}, fmt.Errorf("case %s oracle input %s has no matching digest", e.ID, id)
		}
	}
	if e.Kind == "chain" && !kinds["block"] {
		return caseCoverage{}, errors.New("chain expectation requires block inputs")
	}
	if oneOf(e.Kind, "transaction", "script") && !kinds["transaction"] {
		return caseCoverage{}, errors.New("transaction expectation requires transaction inputs")
	}
	if e.Kind == "script" && !kinds["script"] {
		return caseCoverage{}, errors.New("script expectation requires script inputs")
	}
	if e.Kind != "metric" && b.manifest.Class != "decoder" {
		context := append([]string{b.manifest.Network.Parameters}, b.manifest.Network.Genesis...)
		context = append(context, b.manifest.Reset.State...)
		for _, id := range context {
			if !inputs[id] {
				return caseCoverage{}, fmt.Errorf("case %s omits context input %s", e.ID, id)
			}
		}
	}
	if e.Kind == "chain" {
		if err = b.validateChainOutcome(e, result); err != nil {
			return caseCoverage{}, err
		}
	}
	if e.Kind != "metric" {
		return caseCoverage{}, nil
	}
	observables, err := b.observableEvidence(e, result)
	if err != nil {
		return caseCoverage{}, err
	}
	support := caseCoverage{families: map[string]bool{}, observables: observables}
	for _, id := range e.Inputs {
		if b.artifacts[id].Kind != "golden" {
			continue
		}
		data, err := b.Read(id)
		if err != nil {
			return caseCoverage{}, err
		}
		var source struct {
			Version int `json:"version"`
			Cases   map[string]struct {
				Family   string   `json:"family"`
				Families []string `json:"families"`
			} `json:"cases"`
		}
		if err = json.Unmarshal(data, &source); err != nil {
			continue
		}
		c, ok := source.Cases[e.ID]
		if !ok {
			continue
		}
		if source.Version != SchemaVersion {
			return caseCoverage{}, errors.New("unsupported metric input version")
		}
		families := c.Families
		if c.Family != "" {
			families = append(families, c.Family)
		}
		for _, family := range families {
			if family == "measurement-quality" {
				support.quality = true
				continue
			}
			index := -1
			for i, n := range derivedNames {
				if n == family {
					index = i
					break
				}
			}
			if index < 0 {
				return caseCoverage{}, fmt.Errorf("unknown metric family %s", family)
			}
			if err = validateMetricResult(index, value); err != nil {
				return caseCoverage{}, fmt.Errorf("family %s lacks a typed positive result: %w", family, err)
			}
			support.families[family] = true
		}
	}
	return support, nil
}

func (b *Bundle) validateChainOutcome(e Expectation, result map[string]json.RawMessage) error {
	var branches map[string][]struct {
		Artifact string `json:"artifact"`
		Hash     string `json:"hash"`
		Slot     uint64 `json:"slot"`
		Number   uint64 `json:"number"`
	}
	if err := json.Unmarshal(result["branches"], &branches); err != nil || len(branches) == 0 {
		return errors.New("chain outcome requires independently expected branches")
	}
	inputs := map[string]bool{}
	for _, id := range e.Inputs {
		inputs[id] = true
	}
	seen := map[string]bool{}
	for _, branch := range branches {
		ids := make([]string, 0, len(branch))
		for _, point := range branch {
			if !inputs[point.Artifact] {
				return fmt.Errorf("expected branch lacks input %s", point.Artifact)
			}
			ids = append(ids, point.Artifact)
		}
		if len(ids) == 0 {
			return errors.New("empty expected branch")
		}
		blocks, err := b.Blocks(ids)
		if err != nil {
			return err
		}
		for i, block := range blocks {
			point := branch[i]
			if block.Hash().String() != point.Hash || block.SlotNumber() != point.Slot || block.BlockNumber() != point.Number {
				return errors.New("chain outcome does not identify exact input blocks")
			}
			seen[point.Artifact] = true
		}
	}
	for _, id := range e.Inputs {
		if b.artifacts[id].Kind == "block" && !seen[id] {
			return fmt.Errorf("block %s has no expected branch outcome", id)
		}
	}
	return nil
}

func (b *Bundle) observableEvidence(e Expectation, result map[string]json.RawMessage) (map[string]bool, error) {
	support := map[string]bool{}
	if _, present := result["counts"]; !present {
		return support, nil
	}
	var counts map[string]uint64
	var total uint64
	var schema, envelope string
	if json.Unmarshal(result["counts"], &counts) != nil || len(counts) == 0 || json.Unmarshal(result["total"], &total) != nil || total == 0 || json.Unmarshal(result["schema"], &schema) != nil || json.Unmarshal(result["envelope"], &envelope) != nil {
		return nil, errors.New("incomplete observable outcome")
	}
	if schema != b.artifacts[b.coverage.Schema].Path || envelope != b.artifacts[b.coverage.Envelope].Path {
		return nil, errors.New("observable outcome uses different schemas")
	}
	bound := false
	for _, id := range e.Inputs {
		bound = bound || id == b.coverage.Examples
	}
	if !bound {
		return nil, errors.New("observable outcome must bind the canonical records")
	}
	raw, err := b.Read(b.coverage.Examples)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	actual := map[string]uint64{}
	var observed uint64
	for scanner.Scan() {
		if err = uniqueJSON(scanner.Bytes()); err != nil {
			return nil, err
		}
		instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(scanner.Bytes()))
		if err != nil {
			return nil, err
		}
		if err = b.schemas[b.coverage.Schema].Validate(instance); err != nil {
			return nil, fmt.Errorf("canonical observable: %w", err)
		}
		object, ok := instance.(map[string]any)
		if !ok {
			return nil, errors.New("observable must be an object")
		}
		ns, ok := object["ns"].(string)
		if !ok {
			return nil, errors.New("observable namespace required")
		}
		actual[ns]++
		observed++
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	if total != observed || len(counts) != len(actual) {
		return nil, errors.New("observable populations differ from expected")
	}
	for ns, count := range counts {
		if count == 0 || count != actual[ns] {
			return nil, fmt.Errorf("observable population differs for %s", ns)
		}
		for _, known := range observableNames {
			if ns == known {
				support[ns] = true
				break
			}
		}
	}
	return support, nil
}
