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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// Harness runs tests against the curated upstream fixture corpus.
type Harness struct {
	fixturesRoot string
}

// HarnessConfig configures the fixture harness.
type HarnessConfig struct {
	// FixturesRoot is the root directory containing the upstream fixtures.
	// When left empty, NewHarness resolves the default "upstream" path
	// relative to the fixtures package directory.
	FixturesRoot string
}

// NewHarness creates a new fixture harness.
func NewHarness(config HarnessConfig) *Harness {
	fixturesRoot := config.FixturesRoot
	if fixturesRoot == "" {
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			panic("fixtures: unable to determine package directory")
		}
		fixturesRoot = filepath.Join(filepath.Dir(thisFile), "upstream")
	}

	return &Harness{
		fixturesRoot: fixturesRoot,
	}
}

// FixturesRoot returns the harness fixture root.
func (h *Harness) FixturesRoot() string {
	return h.fixturesRoot
}

// Collect returns all curated fixtures under the harness root.
func (h *Harness) Collect() ([]Fixture, error) {
	return CollectFixtures(h.fixturesRoot)
}

// Fixture returns a single fixture by relative path.
func (h *Harness) Fixture(relPath string) (Fixture, error) {
	normalizedPath := normalizeRelativePath(relPath)
	fixtures, err := h.Collect()
	if err != nil {
		return Fixture{}, fmt.Errorf("failed to collect fixtures: %w", err)
	}

	for _, fixture := range fixtures {
		if fixture.RelPath == normalizedPath {
			return fixture, nil
		}
	}

	return Fixture{}, fmt.Errorf("fixture not found: %s", normalizedPath)
}

// RunAll runs a test callback against every curated fixture.
func (h *Harness) RunAll(
	t *testing.T,
	testFunc func(t *testing.T, fixture Fixture),
) {
	t.Helper()
	h.runMatching(t, Filter{}, testFunc)
}

// RunMatching runs a test callback against fixtures matching the supplied
// filter. It fails the test if the filter matches no fixtures.
func (h *Harness) RunMatching(
	t *testing.T,
	filter Filter,
	testFunc func(t *testing.T, fixture Fixture),
) {
	t.Helper()
	h.runMatching(t, filter, testFunc)
}

// RunFixture runs a test callback against a single fixture by relative path.
func (h *Harness) RunFixture(
	t *testing.T,
	relPath string,
	testFunc func(t *testing.T, fixture Fixture),
) {
	t.Helper()

	normalizedPath := normalizeRelativePath(relPath)
	fixture, err := h.Fixture(normalizedPath)
	if err != nil {
		t.Fatalf("%v", err)
	}

	t.Run(fixture.RelPath, func(t *testing.T) {
		testFunc(t, fixture)
	})
}

func (h *Harness) runMatching(
	t *testing.T,
	filter Filter,
	testFunc func(t *testing.T, fixture Fixture),
) {
	t.Helper()

	fixtures, err := h.Collect()
	if err != nil {
		t.Fatalf("failed to collect fixtures: %v", err)
	}

	var matched int
	for _, fixture := range fixtures {
		if !filter.Matches(fixture) {
			continue
		}
		matched++
		t.Run(fixture.RelPath, func(t *testing.T) {
			testFunc(t, fixture)
		})
	}

	if matched == 0 {
		t.Fatalf("no fixtures matched filter: %+v", filter)
	}
}

// CollectFixtureFiles reads the committed manifest under root and returns the
// filesystem paths of every listed fixture in sorted order. It returns an
// error if any manifest entry is missing on disk.
func CollectFixtureFiles(root string) ([]string, error) {
	manifest, err := LoadManifest(root)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(manifest))
	for _, relPath := range manifest {
		path := filepath.Join(root, filepath.FromSlash(relPath))
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf(
				"manifest entry %q missing: %w",
				relPath,
				err,
			)
		}
		paths = append(paths, path)
	}

	sort.Strings(paths)
	return paths, nil
}

// CollectFixtures walks the fixture root and returns typed metadata for every
// curated fixture.
func CollectFixtures(root string) ([]Fixture, error) {
	paths, err := CollectFixtureFiles(root)
	if err != nil {
		return nil, err
	}

	fixtures := make([]Fixture, 0, len(paths))
	for _, path := range paths {
		fixture, err := NewFixture(root, path)
		if err != nil {
			return nil, err
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures, nil
}

// LoadManifest reads the committed manifest and returns normalized relative
// fixture paths without the leading "./" prefix.
func LoadManifest(root string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(root, "manifest.txt"))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	manifest := make([]string, 0, len(lines))
	for _, line := range lines {
		line = normalizeRelativePath(line)
		if line == "" {
			continue
		}
		if !filepath.IsLocal(filepath.FromSlash(line)) {
			return nil, fmt.Errorf(
				"manifest entry %q escapes fixture root",
				line,
			)
		}
		manifest = append(manifest, line)
	}
	return manifest, nil
}

// Fixture describes a single curated upstream fixture file.
type Fixture struct {
	// Path is the filesystem path to the fixture.
	Path string

	// RelPath is the slash-normalized fixture path relative to the harness root.
	RelPath string

	// Repo is the upstream repository subtree the fixture came from.
	Repo Repo

	// Kind describes the fixture payload category.
	Kind Kind

	// Format describes the fixture serialization.
	Format Format

	// Era is the fixture's Cardano era when one can be inferred.
	Era string

	// Name is the fixture base filename.
	Name string

	// SourcePath is the path relative to the per-repo fixture subtree.
	SourcePath string
}

// NewFixture constructs fixture metadata for a single file path.
func NewFixture(root string, path string) (Fixture, error) {
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return Fixture{}, fmt.Errorf(
			"failed to compute relative path for %s: %w",
			path,
			err,
		)
	}

	normalizedRelPath := normalizeRelativePath(relPath)
	if normalizedRelPath == ".." ||
		strings.HasPrefix(normalizedRelPath, "../") {
		return Fixture{}, fmt.Errorf(
			"path %s is outside root %s",
			path,
			root,
		)
	}
	parts := strings.Split(normalizedRelPath, "/")
	if len(parts) < 2 {
		return Fixture{}, fmt.Errorf("unexpected fixture path: %s", path)
	}

	repoName := Repo(parts[0])
	sourcePath := strings.Join(parts[1:], "/")
	kind, format, era := classifyFixture(normalizedRelPath)

	return Fixture{
		Path:       path,
		RelPath:    normalizedRelPath,
		Repo:       repoName,
		Kind:       kind,
		Format:     format,
		Era:        era,
		Name:       filepath.Base(path),
		SourcePath: sourcePath,
	}, nil
}

func normalizeRelativePath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	return path
}
