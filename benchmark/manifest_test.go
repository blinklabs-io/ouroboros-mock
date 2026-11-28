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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func mutableCorpus(t *testing.T) (string, Manifest) {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.CopyFS(root, os.DirFS("testdata/reference")))
	path := filepath.Join(root, "upstream.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var m Manifest
	require.NoError(t, json.Unmarshal(raw, &m))
	return path, m
}

func saveManifest(t *testing.T, path string, m Manifest) {
	t.Helper()
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
}

func TestReferenceCorpusIsCompleteAndImmutable(t *testing.T) {
	for _, role := range []string{"upstream", "follower", "fetcher", "txpeer", "fork"} {
		b, err := Load(filepath.Join("testdata/reference", role+".json"))
		require.NoError(t, err)
		require.Len(t, ObservableNames(), 25)
		require.Len(t, DerivedNames(), 23)
		require.GreaterOrEqual(t, len(b.manifest.Expectations), 46)
		for _, a := range b.manifest.Artifacts {
			if a.Kind == "block" {
				_, err = b.Blocks([]string{a.ID})
				require.NoError(t, err)
			}
			if a.Kind == "transaction" {
				_, err = b.Transactions([]string{a.ID})
				require.NoError(t, err)
			}
		}
		roles := b.Roles()
		roles[0].ID = "changed"
		require.NotEqual(t, "changed", b.Roles()[0].ID)
	}
}

func TestManifestRejectsIncompleteOrAmbiguousExperiments(t *testing.T) {
	cases := map[string]func(*Manifest){
		"version":            func(m *Manifest) { m.Version++ },
		"validity":           func(m *Manifest) { m.Class = "linked" },
		"no cases":           func(m *Manifest) { m.Expectations = nil },
		"no models":          func(m *Manifest) { m.Network.CostModels = nil },
		"no state":           func(m *Manifest) { m.Reset.State = nil },
		"no credentials":     func(m *Manifest) { m.Credentials = nil },
		"duplicate artifact": func(m *Manifest) { m.Artifacts = append(m.Artifacts, m.Artifacts[0]) },
		"escape":             func(m *Manifest) { m.Artifacts[0].Path = "../outside" },
		"wrong digest": func(m *Manifest) {
			m.Artifacts[0].SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
		},
		"wrong size":            func(m *Manifest) { m.Artifacts[0].Bytes++ },
		"missing input":         func(m *Manifest) { m.Expectations[0].Inputs = []string{"absent"} },
		"missing expected case": func(m *Manifest) { m.Expectations[0].ID = "absent" },
		"no oracle":             func(m *Manifest) { m.Expectations[0].Oracle = "absent" },
		"unknown target":        func(m *Manifest) { m.Roles[0].Target = "" },
		"transport":             func(m *Manifest) { m.Roles[0].Transport = "unknown" },
		"no requirements":       func(m *Manifest) { m.Roles[0].Required = nil },
		"unknown requirement":   func(m *Manifest) { m.Roles[0].Required = []Requirement{{Event: "unknown", Events: 1}} },
		"dependency cycle": func(m *Manifest) {
			m.Artifacts[0].DependsOn = []string{m.Artifacts[1].ID}
			m.Artifacts[1].DependsOn = []string{m.Artifacts[0].ID}
		},
		"future action": func(m *Manifest) {
			m.Actions = []Action{{ID: "x", Role: m.Roles[0].ID, Kind: "append", Items: m.Roles[0].Initial, AtNS: 0, After: []string{"absent"}}}
		},
		"bad append": func(m *Manifest) {
			m.Actions = []Action{{ID: "x", Role: m.Roles[0].ID, Kind: "append", Items: m.Roles[0].Initial, AtNS: 0}}
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			path, m := mutableCorpus(t)
			change(&m)
			saveManifest(t, path, m)
			_, err := Load(path)
			require.Error(t, err)
		})
	}
}

func TestReadDetectsArtifactMutationAfterLoad(t *testing.T) {
	path, m := mutableCorpus(t)
	b, err := Load(path)
	require.NoError(t, err)
	a := m.Artifacts[0]
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), a.Path))
	require.NoError(t, err)
	raw[0] ^= 1
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(path), a.Path), raw, 0o600))
	_, err = b.Read(a.ID)
	require.ErrorContains(t, err, "changed after validation")
}

func TestManifestRejectsUnknownFieldsTrailingJSONAndEscapingSymlink(t *testing.T) {
	for _, tail := range []string{`,"typo":1}`, `} {}`} {
		t.Run(tail, func(t *testing.T) {
			path, _ := mutableCorpus(t)
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			raw = bytes.TrimSpace(raw)
			raw = append(raw[:len(raw)-1], []byte(tail)...)
			require.NoError(t, os.WriteFile(path, raw, 0o600))
			_, err = Load(path)
			require.Error(t, err)
		})
	}
	path, m := mutableCorpus(t)
	a := m.Artifacts[0]
	target := filepath.Join(t.TempDir(), "outside")
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), a.Path))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(target, raw, 0o600))
	require.NoError(t, os.Remove(filepath.Join(filepath.Dir(path), a.Path)))
	require.NoError(t, os.Symlink(target, filepath.Join(filepath.Dir(path), a.Path)))
	_, err = Load(path)
	require.Error(t, err)
}

func TestCoverageRequiresEveryObservableAndFamilyWithKnownCases(t *testing.T) {
	b, err := Load("testdata/reference/upstream.json")
	require.NoError(t, err)
	raw, err := b.Read(b.manifest.Coverage)
	require.NoError(t, err)
	var c Coverage
	require.NoError(t, decodeJSON(raw, &c))
	cases := b.caseInfo
	require.NoError(t, c.validate(cases))
	c.Observables = c.Observables[1:]
	require.Error(t, c.validate(cases))
	require.NoError(t, decodeJSON(raw, &c))
	c.Derived = c.Derived[1:]
	require.Error(t, c.validate(cases))
	require.NoError(t, decodeJSON(raw, &c))
	c.Observables[0].Cases = []string{"missing"}
	require.Error(t, c.validate(cases))
}

func TestManifestRejectsZeroOrTruncatedModelsEvenWithMatchingHashes(t *testing.T) {
	for _, size := range []int{166, 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			path, m := mutableCorpus(t)
			for i, a := range m.Artifacts {
				if a.ID != "alonzo" && a.ID != "parameters" {
					continue
				}
				p := filepath.Join(filepath.Dir(path), a.Path)
				raw, err := os.ReadFile(p)
				require.NoError(t, err)
				var object map[string]any
				require.NoError(t, json.Unmarshal(raw, &object))
				models := object["costModels"].(map[string]any)
				values := make([]int, size)
				if size == 1 {
					values[0] = 1
				}
				models["PlutusV1"] = values
				raw, err = json.Marshal(object)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(p, raw, 0o600))
				sum := sha256.Sum256(raw)
				m.Artifacts[i].SHA256 = hex.EncodeToString(sum[:])
				m.Artifacts[i].Bytes = int64(len(raw))
			}
			saveManifest(t, path, m)
			_, err := Load(path)
			require.ErrorContains(t, err, "cost model")
		})
	}
}

func replaceArtifact(t *testing.T, path string, m *Manifest, id string, data []byte) {
	t.Helper()
	for i, a := range m.Artifacts {
		if a.ID != id {
			continue
		}
		require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(path), a.Path), data, 0o600))
		hash := sha256.Sum256(data)
		m.Artifacts[i].SHA256 = hex.EncodeToString(hash[:])
		m.Artifacts[i].Bytes = int64(len(data))
		return
	}
	t.Fatalf("missing artifact %s", id)
}

func TestCoverageRejectsUnrelatedAndQualityOnlyGoldens(t *testing.T) {
	b, err := Load("testdata/reference/upstream.json")
	require.NoError(t, err)
	raw, err := b.Read(b.manifest.Coverage)
	require.NoError(t, err)
	for _, id := range []string{"snapshot-count", "source-loss"} {
		// Pick actual positive and quality cases rather than inventing missing IDs.
		for candidate, info := range b.caseInfo {
			if id == "snapshot-count" && info.supports("Snapshot count") {
				id = candidate
				break
			}
			if id == "source-loss" && info.quality {
				id = candidate
				break
			}
		}
		var c Coverage
		require.NoError(t, decodeJSON(raw, &c))
		for i := range c.Derived {
			c.Derived[i].Cases = []string{id}
		}
		require.Error(t, c.validate(b.caseInfo))
	}
}

func TestStrictJSONRejectsDuplicateKeys(t *testing.T) {
	for _, data := range []string{`{"version":1,"version":1}`, `{"a":[{"x":1,"x":2}]}`} {
		var dst map[string]any
		require.ErrorContains(t, decodeJSON([]byte(data), &dst), "duplicate")
	}
	path, _ := mutableCorpus(t)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	raw = append([]byte(`{"version":1,`), bytes.TrimSpace(raw)[1:]...)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
	_, err = Load(path)
	require.ErrorContains(t, err, "duplicate")
	controls := make(chan commandResult, 1)
	readControls(t.Context(), bytes.NewBufferString(`{"version":1,"kind":"start","kind":"stop","run":"trial"}`+"\n"), controls)
	require.ErrorContains(t, (<-controls).err, "duplicate")
}

func TestConsensusNeedsBlocksTypedOutcomeAndMatchingEvidence(t *testing.T) {
	for _, mode := range []string{"config only", "boolean outcome", "unbound outcome", "missing evidence"} {
		t.Run(mode, func(t *testing.T) {
			path, m := mutableCorpus(t)
			for i, e := range m.Expectations {
				if e.Kind != "chain" {
					continue
				}
				if mode == "config only" {
					m.Expectations[i].Inputs = []string{"parameters"}
				}
				if mode == "boolean outcome" || mode == "unbound outcome" {
					var expected map[string]any
					raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), "node-expected.json"))
					require.NoError(t, err)
					require.NoError(t, json.Unmarshal(raw, &expected))
					if mode == "boolean outcome" {
						expected["cases"].(map[string]any)[e.ID] = true
					} else {
						expected["cases"].(map[string]any)[e.ID] = map[string]any{"branches": map[string]any{}}
					}
					raw, err = json.Marshal(expected)
					require.NoError(t, err)
					replaceArtifact(t, path, &m, e.Expected, raw)
				}
				if mode == "missing evidence" {
					replaceArtifact(t, path, &m, "node-oracle", []byte(`{"version":1,"evidence":{}}`))
				}
				break
			}
			saveManifest(t, path, m)
			_, err := Load(path)
			require.Error(t, err)
		})
	}
}

func TestActionEndIsExclusiveForEveryKind(t *testing.T) {
	for _, kind := range []string{"append", "fork", "fetch", "offer"} {
		t.Run(kind, func(t *testing.T) {
			path, m := mutableCorpus(t)
			m.Actions = []Action{{ID: "at-end", Role: m.Roles[0].ID, Kind: kind, AtNS: m.DurationNS, Items: []string{"b1-000"}}}
			saveManifest(t, path, m)
			_, err := Load(path)
			require.ErrorContains(t, err, "invalid action schedule")
		})
	}
}

func TestSchemaRejectsNullDefinitions(t *testing.T) {
	path, m := mutableCorpus(t)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), "observables.schema.json"))
	require.NoError(t, err)
	var schema map[string]any
	require.NoError(t, json.Unmarshal(raw, &schema))
	schema["$defs"].(map[string]any)[observableNames[0]] = json.RawMessage("null")
	raw, err = json.Marshal(schema)
	require.NoError(t, err)
	replaceArtifact(t, path, &m, "common-schema", raw)
	saveManifest(t, path, m)
	_, err = Load(path)
	require.ErrorContains(t, err, "schema")
}

func TestRepeatedCostModelsMustAgree(t *testing.T) {
	path, m := mutableCorpus(t)
	var object map[string]any
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), "genesis/genesis.alonzo.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &object))
	values := object["costModels"].(map[string]any)["PlutusV1"].([]any)
	values[0] = values[0].(float64) + 1
	raw, err = json.Marshal(object)
	require.NoError(t, err)
	for _, a := range m.Artifacts {
		if a.ID == "alonzo" {
			copy := a
			copy.ID = "conflicting-model"
			copy.Path = "conflicting-model.json"
			m.Artifacts = append(m.Artifacts, copy)
			break
		}
	}
	replaceArtifact(t, path, &m, "conflicting-model", raw)
	m.Network.CostModels = append(m.Network.CostModels, "conflicting-model")
	saveManifest(t, path, m)
	_, err = Load(path)
	require.ErrorContains(t, err, "conflicting cost model")
}

func TestFetchEndpointsAreDecodedBeforeReady(t *testing.T) {
	path, m := mutableCorpus(t)
	var bad Artifact
	for _, a := range m.Artifacts {
		if a.Kind == "block" {
			bad = a
			break
		}
	}
	bad.ID = "invalid-endpoint"
	bad.Path = "invalid-endpoint.cbor"
	m.Artifacts = append(m.Artifacts, bad)
	replaceArtifact(t, path, &m, bad.ID, []byte{0xff})
	m.Roles = []Role{{ID: "fetch", Kind: "fetcher", Target: "node", Transport: "tcp", Required: []Requirement{}}}
	m.Actions = []Action{{ID: "fetch", Kind: "fetch", Role: "fetch", Items: []string{bad.ID, bad.ID}}}
	saveManifest(t, path, m)
	_, err := Load(path)
	require.ErrorContains(t, err, `block "invalid-endpoint": decode`)
}

func TestSchemasResolveOnlyDeclaredChecksummedReferences(t *testing.T) {
	path, m := mutableCorpus(t)
	for i, a := range m.Artifacts {
		if a.ID == "common-types" {
			m.Artifacts = append(m.Artifacts[:i], m.Artifacts[i+1:]...)
			break
		}
	}
	saveManifest(t, path, m)
	_, err := Load(path)
	require.ErrorContains(t, err, "undeclared reference types.schema.json")
	path, m = mutableCorpus(t)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), "types.schema.json"))
	require.NoError(t, err)
	var schema map[string]any
	require.NoError(t, json.Unmarshal(raw, &schema))
	delete(schema["$defs"].(map[string]any), "Timestamp")
	raw, err = json.Marshal(schema)
	require.NoError(t, err)
	replaceArtifact(t, path, &m, "common-types", raw)
	saveManifest(t, path, m)
	_, err = Load(path)
	require.ErrorContains(t, err, "unresolved schema reference")
}

func TestArtifactSizeLimitAppliesBeforeVerifyAndRead(t *testing.T) {
	root := t.TempDir()
	file, err := os.Create(filepath.Join(root, "oversize"))
	require.NoError(t, err)
	size := maxArtifactBytes + 1
	require.NoError(t, file.Truncate(size))
	require.NoError(t, file.Close())
	hash := sha256.New()
	zero := make([]byte, 1024*1024)
	for left := size; left > 0; {
		n := min(left, int64(len(zero)))
		_, err = hash.Write(zero[:n])
		require.NoError(t, err)
		left -= n
	}
	a := Artifact{ID: "oversize", Path: "oversize", Bytes: size, SHA256: hex.EncodeToString(hash.Sum(nil)), Kind: "config"}
	b := &Bundle{root: root, artifacts: map[string]Artifact{a.ID: a}}
	require.ErrorContains(t, b.verifyArtifact(a), "invalid artifact")
	_, err = b.Read(a.ID)
	require.ErrorContains(t, err, "payload read limit")
}
