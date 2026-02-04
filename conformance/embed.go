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
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed testdata
var embeddedTestdata embed.FS

// EmbeddedTestdata returns the embedded testdata filesystem.
// This allows consumers to use the conformance test vectors without
// needing to reference external files.
func EmbeddedTestdata() embed.FS {
	return embeddedTestdata
}

// ExtractEmbeddedTestdata extracts the embedded testdata to a temporary directory
// and returns the path. The caller is responsible for cleaning up the directory
// when done (e.g., using t.TempDir() or defer os.RemoveAll()).
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
	return extractFS(embeddedTestdata, "testdata", destDir)
}

// extractFS extracts files from an embed.FS to a destination directory.
func extractFS(fsys embed.FS, root string, destDir string) (string, error) {
	testdataRoot := filepath.Join(destDir, root)

	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
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
	})
	if err != nil {
		return "", err
	}

	return testdataRoot, nil
}
