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

package conformance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blinklabs-io/ouroboros-mock/conformance"
)

// TestCollectVectorFilesMatchesWholeSegments pins that the directory filters
// match whole path segments.
//
// A substring test excluded every vector whose directory name merely ends in a
// filter word -- the Blueprint corpus's
// Conway.Imp.AlonzoImpSpec.UTXOS.can_use_reference_scripts records were dropped
// by a strings.Contains check for "scripts/", so the suite reported a smaller
// corpus than it held and nothing failed.
func TestCollectVectorFilesMatchesWholeSegments(t *testing.T) {
	root := t.TempDir()

	// A directory whose name ends in a filter word: its vector must be kept.
	kept := filepath.Join(root, "eras", "conway", "UTXOS.can_use_reference_scripts")
	if err := os.MkdirAll(kept, 0o755); err != nil {
		t.Fatalf("mkdir kept: %v", err)
	}
	keptVector := filepath.Join(kept, "0")
	if err := os.WriteFile(keptVector, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write kept vector: %v", err)
	}

	// A directory genuinely named "scripts": its contents must still be skipped.
	skipped := filepath.Join(root, "eras", "conway", "scripts")
	if err := os.MkdirAll(skipped, 0o755); err != nil {
		t.Fatalf("mkdir skipped: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skipped, "helper"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write skipped file: %v", err)
	}

	// A directory whose name ends in the pparams filter word: also kept.
	keptPparams := filepath.Join(root, "eras", "conway", "uses-pparams-by-hash")
	if err := os.MkdirAll(keptPparams, 0o755); err != nil {
		t.Fatalf("mkdir kept pparams: %v", err)
	}
	keptPparamsVector := filepath.Join(keptPparams, "0")
	if err := os.WriteFile(keptPparamsVector, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write kept pparams vector: %v", err)
	}

	// A directory genuinely named "pparams-by-hash": still skipped.
	skippedPparams := filepath.Join(root, "eras", "conway", "pparams-by-hash")
	if err := os.MkdirAll(skippedPparams, 0o755); err != nil {
		t.Fatalf("mkdir skipped pparams: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skippedPparams, "abc"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write skipped pparams file: %v", err)
	}

	// Filter words used as leaf filenames are valid vectors and must be kept.
	leafDir := filepath.Join(root, "eras", "conway", "leaf-vectors")
	if err := os.MkdirAll(leafDir, 0o755); err != nil {
		t.Fatalf("mkdir leaf vectors: %v", err)
	}
	for _, name := range []string{"scripts", "pparams-by-hash"} {
		path := filepath.Join(leafDir, name)
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			t.Fatalf("write leaf vector %s: %v", name, err)
		}
	}

	got, err := conformance.CollectVectorFiles(root)
	if err != nil {
		t.Fatalf("CollectVectorFiles: %v", err)
	}

	found := map[string]bool{}
	for _, path := range got {
		found[filepath.ToSlash(path)] = true
	}

	for _, want := range []string{
		keptVector,
		keptPparamsVector,
		filepath.Join(leafDir, "scripts"),
		filepath.Join(leafDir, "pparams-by-hash"),
	} {
		if !found[filepath.ToSlash(want)] {
			t.Errorf(
				"vector in a directory ending in a filter word was dropped: %s",
				want,
			)
		}
	}
	for _, path := range got {
		for _, segment := range strings.Split(filepath.ToSlash(filepath.Dir(path)), "/") {
			if segment == "scripts" || segment == "pparams-by-hash" {
				t.Errorf("a genuinely filtered directory was collected: %s", path)
			}
		}
	}
}
