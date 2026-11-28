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
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/blinklabs-io/ouroboros-mock/peer"
)

func oneOf(s string, choices ...string) bool {
	for _, v := range choices {
		if s == v {
			return true
		}
	}
	return false
}
func name(s string) bool { return s != "" && strings.TrimSpace(s) == s }

func (b *Bundle) require(ids []string, kind string) error {
	seen := map[string]bool{}
	for _, id := range ids {
		a, ok := b.artifacts[id]
		if !ok || (kind != "" && a.Kind != kind) || seen[id] {
			return fmt.Errorf("invalid or repeated %s artifact %q", kind, id)
		}
		seen[id] = true
	}
	return nil
}

func (b *Bundle) validate() error {
	m := b.manifest
	if m.Version != SchemaVersion || !name(m.ID) || m.Diagnostics == nil {
		return errors.New("unsupported version or empty experiment identity")
	}
	if !oneOf(m.Class, "decoder", "ledger", "consensus") {
		return errors.New("unknown validity class")
	}
	if m.Network.Magic == 0 || !oneOf(m.Network.Era, "Shelley", "Allegra", "Mary", "Alonzo", "Babbage", "Conway") || m.Network.SystemStart.IsZero() || m.Network.SlotSeconds <= 0 || m.DurationNS <= 0 {
		return errors.New("incomplete network/time context")
	}
	if !oneOf(m.Reset.Strategy, "fresh", "restore") || !oneOf(m.Reset.Cache, "cold", "warm", "steady") {
		return errors.New("explicit reset and cache policy required")
	}
	paths := map[string]bool{}
	for _, a := range m.Artifacts {
		if _, ok := b.artifacts[a.ID]; ok || paths[a.Path] {
			return fmt.Errorf("repeated artifact identity/path %q", a.ID)
		}
		if !oneOf(a.Kind, "genesis", "config", "parameters", "state", "credentials", "block", "transaction", "script", "oracle", "coverage", "golden") {
			return fmt.Errorf("unknown artifact kind %q", a.Kind)
		}
		if (a.Kind == "block" || a.Kind == "transaction") && a.Type == nil {
			return fmt.Errorf("wire type required for %q", a.ID)
		}
		if err := b.verifyArtifact(a); err != nil {
			return err
		}
		b.artifacts[a.ID] = a
		paths[a.Path] = true
	}
	if len(m.Artifacts) == 0 || len(m.Network.Genesis) == 0 {
		return errors.New("genesis and artifacts required")
	}
	if err := b.require(m.Network.Genesis, "genesis"); err != nil {
		return err
	}
	if err := b.require([]string{m.Network.Parameters}, "parameters"); err != nil {
		return err
	}
	if err := b.require(m.Reset.State, ""); err != nil {
		return err
	}
	if (m.Class != "decoder" || m.Reset.Strategy == "restore") && len(m.Reset.State) == 0 {
		return errors.New("initial state required for this replay")
	}
	if len(m.Network.CostModels) == 0 || len(m.Credentials) == 0 {
		return errors.New("cost models and public test credentials required")
	}
	if err := b.require(m.Network.CostModels, "genesis"); err != nil {
		return err
	}
	if err := b.require(m.Credentials, "credentials"); err != nil {
		return err
	}
	if err := b.validateModels(); err != nil {
		return err
	}
	for _, artifact := range m.Artifacts {
		if err := b.require(artifact.DependsOn, ""); err != nil {
			return err
		}
		for _, dep := range artifact.DependsOn {
			if dep == artifact.ID {
				return fmt.Errorf("self dependency: %s", dep)
			}
		}
	}
	if err := b.validateDependencies(); err != nil {
		return err
	}
	oracles := map[string]Oracle{}
	for _, o := range m.Oracles {
		if !name(o.ID) || !name(o.Version) || !name(o.Source) || len(o.Inputs) == 0 || !oneOf(o.Tool, "cardano-node", "cardano-cli", "cardano-ledger", "cardano-blueprint", "manual") {
			return fmt.Errorf("incomplete independent oracle %q", o.ID)
		}
		if _, ok := oracles[o.ID]; ok {
			return fmt.Errorf("repeated oracle %q", o.ID)
		}
		if err := b.require(o.Inputs, ""); err != nil {
			return err
		}
		if err := b.require([]string{o.Result}, "oracle"); err != nil {
			return err
		}
		oracles[o.ID] = o
	}
	cases := map[string]bool{}
	chainCase := false
	ledgerCase := false
	for _, e := range m.Expectations {
		if !name(e.ID) || cases[e.ID] || !oneOf(e.Kind, "transaction", "chain", "script", "metric", "protocol") || len(e.Inputs) == 0 {
			return fmt.Errorf("invalid case %q", e.ID)
		}
		o, ok := oracles[e.Oracle]
		if !ok || (e.Kind != "metric" && o.Tool == "manual") {
			return fmt.Errorf("independent reference oracle required for %q", e.ID)
		}
		if e.Kind == "chain" && m.Class == "consensus" && o.Tool != "cardano-node" {
			return fmt.Errorf("consensus case %q needs a node oracle", e.ID)
		}
		if err := b.require(e.Inputs, ""); err != nil {
			return err
		}
		for _, id := range e.Inputs {
			found := false
			for _, input := range o.Inputs {
				if input == id {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("oracle does not identify case input %q", id)
			}
		}
		if err := b.require([]string{e.Expected}, "golden"); err != nil {
			return err
		}
		data, err := b.Read(e.Expected)
		if err != nil {
			return err
		}
		var expected struct {
			Version int                        `json:"version"`
			Cases   map[string]json.RawMessage `json:"cases"`
		}
		if err = decodeJSON(data, &expected); err != nil {
			return err
		}
		value, present := expected.Cases[e.ID]
		if expected.Version != SchemaVersion || !present || string(value) == "null" || len(value) == 0 {
			return fmt.Errorf("missing expected result for %s", e.ID)
		}
		info, err := b.validateEvidence(e, o, value)
		if err != nil {
			return err
		}
		b.caseInfo[e.ID] = info
		chainCase = chainCase || e.Kind == "chain"
		ledgerCase = ledgerCase || e.Kind == "transaction" || e.Kind == "script"
		cases[e.ID] = true
	}
	if (m.Class == "consensus" && !chainCase) || (m.Class == "ledger" && !ledgerCase) {
		return errors.New("validity class requires matching independent expectations")
	}
	if len(cases) == 0 {
		return errors.New("empty selected case set")
	}
	if err := b.require([]string{m.Coverage}, "coverage"); err != nil {
		return err
	}
	raw, err := b.Read(m.Coverage)
	if err != nil {
		return err
	}
	var coverage Coverage
	if err = decodeJSON(raw, &coverage); err != nil {
		return err
	}
	if err = coverage.validate(b.caseInfo); err != nil {
		return err
	}
	if err = b.require([]string{coverage.Schema, coverage.Envelope}, "config"); err != nil {
		return err
	}
	if err = b.require([]string{coverage.Examples}, "golden"); err != nil {
		return err
	}
	if err = b.validateSchemaReferences(coverage.Schema, coverage.Envelope); err != nil {
		return err
	}
	schemaBytes, err := b.Read(coverage.Schema)
	if err != nil {
		return err
	}
	var schema struct {
		Definitions map[string]json.RawMessage `json:"$defs"`
	}
	if err = json.Unmarshal(schemaBytes, &schema); err != nil {
		return err
	}
	if len(schema.Definitions) != len(observableNames) {
		return errors.New("incomplete observable schema catalogue")
	}
	for _, observable := range observableNames {
		definition := schema.Definitions[observable]
		var object map[string]json.RawMessage
		if err = json.Unmarshal(definition, &object); err != nil || len(object) == 0 {
			return fmt.Errorf("schema lacks %s", observable)
		}
		var kind string
		var required []string
		var properties map[string]json.RawMessage
		if json.Unmarshal(object["type"], &kind) != nil || kind != "object" || (len(object["required"]) > 0 && json.Unmarshal(object["required"], &required) != nil) || json.Unmarshal(object["properties"], &properties) != nil || len(properties) == 0 {
			return fmt.Errorf("invalid observable schema definition %s", observable)
		}
		for _, field := range required {
			if len(properties[field]) == 0 {
				return fmt.Errorf("schema %s lacks required property %s", observable, field)
			}
		}
	}
	roles := map[string]Role{}
	for _, r := range m.Roles {
		if !name(r.ID) || !name(r.Target) || r.Transport != "tcp" || !oneOf(r.Kind, "upstream", "follower", "fetcher", "txpeer") {
			return fmt.Errorf("invalid role %q", r.ID)
		}
		if _, ok := roles[r.ID]; ok {
			return fmt.Errorf("repeated role %q", r.ID)
		}
		if r.Kind != "upstream" && len(r.Initial) != 0 {
			return fmt.Errorf("initial blocks require an upstream role: %q", r.ID)
		}
		if _, err = b.Blocks(r.Initial); err != nil {
			return err
		}
		if r.Required == nil {
			return fmt.Errorf("role %q must declare observation requirements", r.ID)
		}
		seenRequirements := map[string]bool{}
		for _, req := range r.Required {
			known := false
			for _, event := range eventNames {
				if req.Event == event {
					known = true
					break
				}
			}
			if !known || seenRequirements[req.Event] || (req.Events == 0 && req.Items == 0) {
				return fmt.Errorf("invalid event requirement for %q", r.ID)
			}
			seenRequirements[req.Event] = true
		}
		roles[r.ID] = r
	}
	if len(roles) == 0 {
		return errors.New("no peer roles selected")
	}
	chains := map[string]*peer.Chain{}
	for _, r := range m.Roles {
		if r.Kind == "upstream" {
			blocks, err := b.Blocks(r.Initial)
			if err != nil {
				return err
			}
			chains[r.ID], err = peer.NewChain(blocks)
			if err != nil {
				return err
			}
		}
	}
	actions := map[string]Action{}
	last := int64(-1)
	for _, a := range m.Actions {
		r, ok := roles[a.Role]
		if !ok || !name(a.ID) || a.AtNS < 0 || a.AtNS < last || a.AtNS >= m.DurationNS {
			return fmt.Errorf("invalid action schedule %q", a.ID)
		}
		if _, ok = actions[a.ID]; ok {
			return fmt.Errorf("repeated action %q", a.ID)
		}
		for _, dep := range a.After {
			if _, ok = actions[dep]; !ok {
				return fmt.Errorf("action %q has a future/unknown dependency %q", a.ID, dep)
			}
		}
		if len(a.Items) == 0 {
			return fmt.Errorf("empty action %q", a.ID)
		}
		switch a.Kind {
		case "offer":
			if r.Kind != "txpeer" {
				return fmt.Errorf("offer requires a txpeer: %q", a.ID)
			}
			if _, err = b.Transactions(a.Items); err != nil {
				return err
			}
		case "append", "fork":
			if r.Kind != "upstream" {
				return fmt.Errorf("chain action requires an upstream: %q", a.ID)
			}
			blocks, decodeErr := b.Blocks(a.Items)
			if decodeErr != nil {
				return decodeErr
			}
			chain, present := chains[r.ID]
			if !present || chain == nil {
				return fmt.Errorf("missing chain for role %s", r.ID)
			}
			if a.Kind == "append" {
				err = chain.Append(blocks...)
			} else {
				_, err = chain.SwitchFork(blocks)
			}
			if err != nil {
				return fmt.Errorf("action %s: %w", a.ID, err)
			}
		case "fetch":
			if r.Kind != "fetcher" || len(a.Items) != 2 {
				return fmt.Errorf("fetch requires two block endpoints: %q", a.ID)
			}
			for _, id := range a.Items {
				if _, err = b.Blocks([]string{id}); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unknown action %q", a.Kind)
		}
		actions[a.ID] = a
		last = a.AtNS
	}
	return nil
}

func (b *Bundle) validateDependencies() error {
	state := map[string]uint8{}
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("artifact dependency cycle: %s", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, dep := range b.artifacts[id].DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range b.artifacts {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bundle) validateModels() error {
	combined := map[string][]int64{}
	for _, id := range b.manifest.Network.CostModels {
		data, err := b.Read(id)
		if err != nil {
			return err
		}
		var g struct {
			CostModels map[string][]int64 `json:"costModels"`
			V3         []int64            `json:"plutusV3CostModel"`
		}
		if err = json.Unmarshal(data, &g); err != nil {
			return err
		}
		for key, model := range g.CostModels {
			if old, exists := combined[key]; exists && !slices.Equal(old, model) {
				return fmt.Errorf("conflicting cost model %s", key)
			}
			combined[key] = model
		}
		if len(g.V3) > 0 {
			if old, exists := combined["PlutusV3"]; exists && !slices.Equal(old, g.V3) {
				return errors.New("conflicting cost model PlutusV3")
			}
			combined["PlutusV3"] = g.V3
		}
	}
	raw, err := b.Read(b.manifest.Network.Parameters)
	if err != nil {
		return err
	}
	var params struct {
		CostModels map[string][]int64 `json:"costModels"`
	}
	if err = json.Unmarshal(raw, &params); err != nil {
		return err
	}
	if len(combined) == 0 || len(params.CostModels) != len(combined) {
		return errors.New("incomplete cost-model context")
	}
	for name, model := range combined {
		minimum, known := map[string]int{"PlutusV1": 166, "PlutusV2": 175, "PlutusV3": 251}[name]
		if !known || len(model) < minimum {
			return fmt.Errorf("incomplete cost model %s", name)
		}
		nonzero := false
		for _, value := range model {
			nonzero = nonzero || value != 0
		}
		if !nonzero || !slices.Equal(model, params.CostModels[name]) {
			return fmt.Errorf("missing or inconsistent cost model %s", name)
		}
	}
	return nil
}
