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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Only the repo-local synthetic corpus is embedded as loose files. The
// Cardano Blueprint era corpus is embedded as the pinned upstream archive
// below, so the embedded filesystem is identical whether this package is
// built from a checkout or from a downloaded module.
//
//go:embed testdata/synthetic
var embeddedTestdata embed.FS

// blueprintVectors is the Cardano Blueprint conformance archive, byte-identical
// to the copy in the pinned submodule revision. It is tracked in Git because
// Go module zips carry neither submodule contents nor ignored paths, and
// consumers obtain the era corpus only through this module.
//
//go:embed blueprint-vectors.tar.gz
var blueprintVectors []byte

// BlueprintVectorsSHA256 is the SHA-256 of the embedded archive. It matches
// src/ledger/conformance-test-vectors/vectors.tar.gz at the Blueprint revision
// recorded in CORPUS.md.
const BlueprintVectorsSHA256 = "574ff7a17857dfc1f0cf477f7eb9eba1c2a0f901453396a779de4b2392ef6863"

// blueprintArchivePrefix is the single leading path component stripped from
// every archive entry, matching `tar --strip-components=1` in
// scripts/update-blueprint-conformance.sh.
const blueprintArchivePrefix = "eras"

// EmbeddedTestdata returns the embedded testdata filesystem. It contains the
// synthetic corpus only and never contains eras/: the Cardano Blueprint era
// corpus is embedded as an archive and is materialized by
// ExtractEmbeddedTestdata or ExtractBlueprintVectors.
//
// Module builds have behaved this way since the era corpus moved to the
// submodule, because testdata/eras is not tracked. Building from a checkout
// that has run `make prepare-blueprint-testdata` no longer differs, so the
// returned filesystem no longer depends on the build environment.
// Callers that need the era corpus must use ExtractEmbeddedTestdata.
func EmbeddedTestdata() embed.FS {
	return embeddedTestdata
}

// ExtractEmbeddedTestdata extracts the embedded testdata to a temporary directory
// and returns the path. Both the synthetic corpus and the Cardano Blueprint era
// corpus are written, so the result is a complete testdata root. The caller is
// responsible for cleaning up the directory when done (e.g., using t.TempDir()
// or defer os.RemoveAll()).
//
// Usage with testing.T:
//
//	tmpDir := t.TempDir()
//	testdataRoot, err := conformance.ExtractEmbeddedTestdata(tmpDir)
//	if err != nil {
//	    t.Fatal(err)
//	}
//	harness := conformance.NewHarness(sm, conformance.HarnessConfig{
//	    TestdataRoot: testdataRoot,
//	})
func ExtractEmbeddedTestdata(destDir string) (string, error) {
	testdataRoot, err := extractFS(embeddedTestdata, "testdata", destDir)
	if err != nil {
		return "", err
	}
	if err := ExtractBlueprintVectors(filepath.Join(testdataRoot, blueprintArchivePrefix)); err != nil {
		return "", err
	}
	return testdataRoot, nil
}

// ExtractBlueprintVectors writes the embedded Cardano Blueprint era corpus into
// destDir, which is created if needed. Entry names are normalized exactly as
// scripts/update-blueprint-conformance.sh normalizes the working copy under
// conformance/testdata/eras.
func ExtractBlueprintVectors(destDir string) error {
	gz, err := gzip.NewReader(bytes.NewReader(blueprintVectors))
	if err != nil {
		return fmt.Errorf("blueprint vectors: %w", err)
	}
	defer gz.Close()

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	// seen maps a normalized relative path to the archive path it came from,
	// so two distinct entries cannot silently collide after normalization.
	seen := make(map[string]string)
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("blueprint vectors: %w", err)
		}
		rel, err := normalizeBlueprintPath(header.Name)
		if err != nil {
			return err
		}
		if rel == "" {
			continue
		}
		if prev, ok := seen[rel]; ok {
			return fmt.Errorf(
				"blueprint vectors: colliding normalized path %q from %q and %q",
				rel, prev, header.Name,
			)
		}
		seen[rel] = header.Name

		target := filepath.Join(destDir, filepath.FromSlash(rel))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			data, err := io.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("blueprint vectors: %s: %w", header.Name, err)
			}
			if err := os.WriteFile(target, data, 0o600); err != nil {
				return err
			}
		default:
			return fmt.Errorf(
				"blueprint vectors: unsupported entry type %q for %q",
				header.Typeflag, header.Name,
			)
		}
	}
	return nil
}

// normalizeBlueprintPath strips the archive's single leading path component and
// normalizes each remaining component. It returns an empty path for the archive
// root itself.
func normalizeBlueprintPath(name string) (string, error) {
	clean := path.Clean(strings.TrimPrefix(filepath.ToSlash(name), "./"))
	if clean == "." || clean == "/" {
		return "", nil
	}
	parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")
	if parts[0] != blueprintArchivePrefix {
		return "", fmt.Errorf(
			"blueprint vectors: entry %q is outside %q",
			name, blueprintArchivePrefix,
		)
	}
	normalized := make([]string, 0, len(parts)-1)
	for _, part := range parts[1:] {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("blueprint vectors: unsafe entry %q", name)
		}
		safe := normalizeVectorName(part)
		if safe == "" {
			return "", fmt.Errorf(
				"blueprint vectors: entry %q normalizes to an empty name",
				name,
			)
		}
		normalized = append(normalized, safe)
	}
	return strings.Join(normalized, "/"), nil
}

// normalizeVectorName maps a single archive path component to the form used on
// disk: every byte outside [:alnum:], '.', '_' and '-' becomes '_', runs of '_'
// collapse to one, and a single trailing '_' is dropped. This mirrors the
// `tr`/`sed` pipeline in scripts/update-blueprint-conformance.sh.
func normalizeVectorName(name string) string {
	var out strings.Builder
	out.Grow(len(name))
	prevUnderscore := false
	for i := range len(name) {
		c := name[i]
		switch {
		case c >= '0' && c <= '9',
			c >= 'A' && c <= 'Z',
			c >= 'a' && c <= 'z',
			c == '.', c == '_', c == '-':
		default:
			c = '_'
		}
		if c == '_' {
			if prevUnderscore {
				continue
			}
			prevUnderscore = true
		} else {
			prevUnderscore = false
		}
		out.WriteByte(c)
	}
	return strings.TrimSuffix(out.String(), "_")
}

// extractFS extracts files from an embed.FS to a destination directory.
func extractFS(fsys embed.FS, root string, destDir string) (string, error) {
	testdataRoot := filepath.Join(destDir, root)

	err := fs.WalkDir(
		fsys,
		root,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			destPath := filepath.Join(destDir, path)

			if d.IsDir() {
				return os.MkdirAll(destPath, 0o755)
			}

			data, err := fsys.ReadFile(path)
			if err != nil {
				return err
			}

			return os.WriteFile(destPath, data, 0o600)
		},
	)
	if err != nil {
		return "", err
	}

	return testdataRoot, nil
}
