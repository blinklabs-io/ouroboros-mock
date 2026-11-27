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
	cases := map[string]bool{}
	for _, e := range b.manifest.Expectations {
		cases[e.ID] = true
	}
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
				require.NoError(t, os.WriteFile(p, raw, 0600))
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
