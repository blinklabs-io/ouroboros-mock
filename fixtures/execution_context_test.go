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

package fixtures_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
)

func TestFixtureExecutionErrorsPreserveSourceAndCause(t *testing.T) {
	const source = "cardano-api/test/cardano-api-test/files/input/gov-anchor-data/valid-drep-metadata.jsonld"
	const relPath = "cardano-api/" + source
	for _, test := range []struct {
		name    string
		payload string
		cause   string
		syntax  bool
	}{
		{
			name:    "metadata",
			payload: `{}`,
			cause:   "missing @context",
		},
		{
			name:    "json",
			payload: `{`,
			cause:   "unexpected end of JSON input",
			syntax:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, filepath.FromSlash(relPath))
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, []byte(test.payload), 0o600))
			require.NoError(
				t,
				os.WriteFile(
					filepath.Join(root, "manifest.txt"),
					[]byte("./"+relPath+"\n"),
					0o600,
				),
			)
			harness := fixtures.NewHarness(
				fixtures.HarnessConfig{FixturesRoot: root},
			)
			single, err := harness.ExecuteFixture(relPath)
			require.NoError(t, err)
			all, err := harness.RunAllExecutionsWithResults()
			require.NoError(t, err)
			require.Len(t, all, 1)
			for _, result := range []fixtures.ExecutionResult{single, all[0]} {
				require.False(t, result.Success)
				require.Equal(t, fixtures.Fixture{
					Path:       path,
					RelPath:    relPath,
					Repo:       fixtures.RepoCardanoAPI,
					Kind:       fixtures.KindGovernanceMetadata,
					Format:     fixtures.FormatJSONLD,
					Name:       "valid-drep-metadata.jsonld",
					SourcePath: source,
				}, result.Fixture)
				require.ErrorContains(t, result.Error, relPath)
				require.ErrorContains(t, result.Error, "repo=cardano-api")
				require.ErrorContains(
					t,
					result.Error,
					"kind=governance-metadata",
				)
				require.ErrorContains(t, result.Error, "format=jsonld")
				require.ErrorContains(t, result.Error, `era=""`)
				require.ErrorContains(t, result.Error, "source="+source)
				require.ErrorContains(t, result.Error, test.cause)
				if test.syntax {
					var syntaxError *json.SyntaxError
					require.ErrorAs(t, result.Error, &syntaxError)
				}
			}
		})
	}
}

func TestFixtureManifestRejectsDuplicateCanonicalPaths(t *testing.T) {
	for _, alias := range []string{
		"cardano-api/valid-drep-metadata.jsonld",
		"./cardano-api/valid-drep-metadata.jsonld",
		"cardano-api/subdir/../valid-drep-metadata.jsonld",
	} {
		t.Run(alias, func(t *testing.T) {
			root := t.TempDir()
			manifest := "./cardano-api/valid-drep-metadata.jsonld\n" + alias + "\n"
			require.NoError(
				t,
				os.WriteFile(
					filepath.Join(root, "manifest.txt"),
					[]byte(manifest),
					0o600,
				),
			)
			_, err := fixtures.LoadManifest(root)
			require.ErrorContains(
				t,
				err,
				`duplicate manifest entry "cardano-api/valid-drep-metadata.jsonld"`,
			)
			require.ErrorContains(t, err, "line 2 (first at line 1)")
		})
	}
}

func TestFixtureManifestRequiresRegularFiles(t *testing.T) {
	manifest := "./cardano-api/sample.json\n"
	for _, directory := range []bool{false, true} {
		t.Run(
			map[bool]string{false: "file", true: "directory"}[directory],
			func(t *testing.T) {
				caseRoot := t.TempDir()
				casePath := filepath.Join(
					caseRoot,
					"cardano-api",
					"sample.json",
				)
				require.NoError(t, os.MkdirAll(filepath.Dir(casePath), 0o755))
				if directory {
					require.NoError(t, os.Mkdir(casePath, 0o755))
				} else {
					require.NoError(t, os.WriteFile(casePath, []byte(`{}`), 0o600))
				}
				require.NoError(
					t,
					os.WriteFile(
						filepath.Join(caseRoot, "manifest.txt"),
						[]byte(manifest),
						0o600,
					),
				)
				files, err := fixtures.CollectFixtureFiles(caseRoot)
				if directory {
					require.ErrorContains(
						t,
						err,
						`manifest entry "cardano-api/sample.json" is not a regular file`,
					)
				} else {
					require.NoError(t, err)
					require.Equal(t, []string{casePath}, files)
				}
			},
		)
	}
}
