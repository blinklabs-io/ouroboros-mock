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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
)

func TestE34EmbeddedIntegrity(t *testing.T) {
	const root = "upstream/e34/"
	embedded := fixtures.EmbeddedFixtures()
	manifest, err := embedded.ReadFile("upstream/manifest.txt")
	require.NoError(t, err)
	listed := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(manifest)), "\n") {
		listed[strings.TrimPrefix(line, "./")] = true
	}

	data, err := embedded.ReadFile(root + "SHA256SUMS")
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(string(data), "\n"))
	checked := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		parts := strings.Split(line, "  ")
		require.Len(t, parts, 2, "malformed checksum row")
		digest, name := parts[0], parts[1]
		rawDigest, err := hex.DecodeString(digest)
		require.NoError(t, err)
		require.Len(t, rawDigest, sha256.Size)
		require.True(t, fs.ValidPath(name))
		require.NotEqual(t, "SHA256SUMS", name)
		require.False(t, checked[name], "duplicate checksum: %s", name)
		payload, err := embedded.ReadFile(root + name)
		require.NoError(t, err)
		require.Equal(t, digest, fmt.Sprintf("%x", sha256.Sum256(payload)), name)
		checked[name] = true
	}
	require.Len(t, checked, 16)
	err = fs.WalkDir(embedded, "upstream/e34", func(
		path string, entry fs.DirEntry, walkErr error,
	) error {
		require.NoError(t, walkErr)
		if entry.IsDir() {
			return nil
		}
		name := strings.TrimPrefix(path, root)
		require.True(t, listed[strings.TrimPrefix(path, "upstream/")], path)
		if name != "SHA256SUMS" {
			require.True(t, checked[name], "missing checksum: %s", name)
		}
		return nil
	})
	require.NoError(t, err)

	// Pin independent source identities, not just a self-consistent manifest.
	pins := map[string]string{
		"pallas/node-vectors.tsv":        "0c2009981a5c31c16fcb943783254351ec910b5ab23d168eda3cce18ec058a1c",
		"jormanager/node-vectors.tsv":    "c0db31169d1811092ad4e56521e802765ee2014945c06f43746847bc96b416cd",
		"reference/cabal.project.freeze": "867325116626ee6841449f09866200528d83a9321faf2df721cd45ef04cd4a69",
	}
	for name, digest := range pins {
		payload, err := embedded.ReadFile(root + name)
		require.NoError(t, err)
		require.Equal(t, digest, fmt.Sprintf("%x", sha256.Sum256(payload)))
	}
}

func TestE34ReferenceDataIsNotExecutable(t *testing.T) {
	harness := fixtures.NewHarness(fixtures.HarnessConfig{})
	result, err := harness.ExecuteFixture("e34/pallas/node-vectors.tsv")
	require.NoError(t, err)
	require.False(t, result.Success)
	require.ErrorContains(t, result.Error, "reference-data fixture is not executable")
}
