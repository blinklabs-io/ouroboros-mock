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

package fixtures

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed upstream
var embeddedFixtures embed.FS

// EmbeddedFixtures returns the embedded upstream fixtures filesystem.
func EmbeddedFixtures() embed.FS {
	return embeddedFixtures
}

// ExtractEmbeddedFixtures extracts the embedded upstream fixtures to destDir
// and returns the extracted root path. The caller is responsible for cleaning
// up destDir when done.
func ExtractEmbeddedFixtures(destDir string) (string, error) {
	return extractFS(embeddedFixtures, "upstream", destDir)
}

// extractFS extracts files from an embed.FS to a destination directory.
func extractFS(fsys embed.FS, root string, destDir string) (string, error) {
	fixturesRoot := filepath.Join(destDir, root)

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

	return fixturesRoot, nil
}
