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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// blueprintLedgerVectorFiles is the number of ledger vector files the pinned
// Blueprint archive contributes, excluding pparams-by-hash entries. It is the
// count reported by `make prepare-blueprint-testdata`.
const blueprintLedgerVectorFiles = 2574

// blueprintCollectedVectors is how many of those files CollectVectorFiles
// returns. It equals the file count: the collector filters whole path
// segments, so a vector directory whose name merely ends in a filter word --
// such as the three ...UTXOS.can_use_reference_scripts/ records -- is
// collected rather than silently dropped.
const blueprintCollectedVectors = blueprintLedgerVectorFiles

// blueprintPParamsCount is the number of protocol-parameter files under
// eras/conway/impl/dump/pparams-by-hash in the pinned Blueprint archive.
const blueprintPParamsCount = 78

func TestBlueprintVectorsChecksum(t *testing.T) {
	sum := sha256.Sum256(blueprintVectors)
	if got := hex.EncodeToString(sum[:]); got != BlueprintVectorsSHA256 {
		t.Fatalf(
			"embedded blueprint archive checksum %s, want %s",
			got,
			BlueprintVectorsSHA256,
		)
	}
}

// TestExtractEmbeddedTestdataShipsEras is the regression test for the corpus
// not reaching module consumers. It uses only embedded data, so it exercises
// exactly what a downstream module build sees.
func TestExtractEmbeddedTestdataShipsEras(t *testing.T) {
	root, err := ExtractEmbeddedTestdata(t.TempDir())
	if err != nil {
		t.Fatalf("failed to extract embedded testdata: %v", err)
	}

	erasRoot := filepath.Join(root, "eras")
	erasFiles := relativeFileSet(t, erasRoot)
	var vectorFiles int
	for rel := range erasFiles {
		if !strings.Contains(filepath.ToSlash(rel), "pparams-by-hash/") {
			vectorFiles++
		}
	}
	if vectorFiles != blueprintLedgerVectorFiles {
		t.Errorf(
			"embedded era corpus has %d vector files, want %d",
			vectorFiles,
			blueprintLedgerVectorFiles,
		)
	}

	erasVectors, err := CollectVectorFiles(erasRoot)
	if err != nil {
		t.Fatalf("failed to collect era vectors: %v", err)
	}
	if len(erasVectors) != blueprintCollectedVectors {
		t.Errorf(
			"embedded era corpus collects %d vectors, want %d",
			len(erasVectors),
			blueprintCollectedVectors,
		)
	}

	pparams, err := os.ReadDir(
		filepath.Join(root, "eras", "conway", "impl", "dump", "pparams-by-hash"),
	)
	if err != nil {
		t.Fatalf("failed to read pparams directory: %v", err)
	}
	if len(pparams) != blueprintPParamsCount {
		t.Errorf(
			"embedded corpus has %d protocol-parameter files, want %d",
			len(pparams),
			blueprintPParamsCount,
		)
	}

	// The harness requires both roots, and a consumer only ever gets them
	// from the embedded data extracted above.
	harness := NewHarness(NewMockStateManager(), HarnessConfig{TestdataRoot: root})
	all, err := harness.collectAllVectors()
	if err != nil {
		t.Fatalf("failed to collect vectors from extracted root: %v", err)
	}
	if len(all) <= len(erasVectors) {
		t.Errorf(
			"harness collected %d vectors, want more than the %d era vectors",
			len(all),
			len(erasVectors),
		)
	}
}

// TestEmbeddedErasMatchesWorkingCopy asserts that the Go extraction path in
// embed.go and the shell extraction path in
// scripts/update-blueprint-conformance.sh produce identical trees, so the two
// normalization implementations cannot drift.
func TestEmbeddedErasMatchesWorkingCopy(t *testing.T) {
	workingCopy := filepath.Join("testdata", "eras")
	if _, err := os.Stat(workingCopy); os.IsNotExist(err) {
		t.Skip("run `make prepare-blueprint-testdata` to compare against the working copy")
	}

	extracted := filepath.Join(t.TempDir(), "eras")
	if err := ExtractBlueprintVectors(extracted); err != nil {
		t.Fatalf("failed to extract blueprint vectors: %v", err)
	}

	want := relativeFileSet(t, workingCopy)
	got := relativeFileSet(t, extracted)

	for rel := range want {
		if _, ok := got[rel]; !ok {
			t.Errorf("extracted corpus is missing %s", rel)
		}
	}
	for rel := range got {
		if _, ok := want[rel]; !ok {
			t.Errorf("extracted corpus has unexpected %s", rel)
			continue
		}
		wantData, err := os.ReadFile(filepath.Join(workingCopy, rel))
		if err != nil {
			t.Fatalf("failed to read %s: %v", rel, err)
		}
		gotData, err := os.ReadFile(filepath.Join(extracted, rel))
		if err != nil {
			t.Fatalf("failed to read %s: %v", rel, err)
		}
		if !bytes.Equal(wantData, gotData) {
			t.Errorf("extracted corpus content differs at %s", rel)
		}
	}
}

func TestNormalizeVectorName(t *testing.T) {
	for _, test := range []struct {
		name string
		want string
	}{
		{name: "Conway.Imp.AllegraImpSpec.UTXOW.InvalidMetadata", want: "Conway.Imp.AllegraImpSpec.UTXOW.InvalidMetadata"},
		{name: "Insufficient collateral", want: "Insufficient_collateral"},
		{name: "ConwayImpSpec - Version 10", want: "ConwayImpSpec_-_Version_10"},
		{name: "trailing space ", want: "trailing_space"},
		{name: "a//b", want: "a_b"},
		{name: "a__b", want: "a_b"},
		{name: "  ", want: ""},
	} {
		if got := normalizeVectorName(test.name); got != test.want {
			t.Errorf("normalizeVectorName(%q) = %q, want %q", test.name, got, test.want)
		}
	}
}

func relativeFileSet(t *testing.T, root string) map[string]struct{} {
	t.Helper()
	files := make(map[string]struct{})
	err := filepath.WalkDir(
		root,
		func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files[rel] = struct{}{}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("failed to walk %s: %v", root, err)
	}
	return files
}
