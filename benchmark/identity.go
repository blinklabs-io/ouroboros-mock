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
	"errors"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
)

// Identity records the actual process executable and available Go build metadata.
type Identity struct {
	SHA256        string `json:"executable_sha256,omitempty"`
	PathSHA256    string `json:"executable_path_sha256,omitempty"`
	Source        string `json:"digest_source"`
	GoVersion     string `json:"go_version"`
	ModulePath    string `json:"module_path,omitempty"`
	ModuleVersion string `json:"module_version,omitempty"`
	// GoVCSRevision and GoVCSModified are compiler-reported diagnostics. In a
	// linked worktree they can describe an enclosing checkout, not this module.
	GoVCSRevision string `json:"go_vcs_revision,omitempty"`
	GoVCSModified *bool  `json:"go_vcs_modified,omitempty"`
}

var processIdentity = sync.OnceValues(func() (Identity, error) {
	path, err := os.Executable()
	if err != nil {
		return Identity{}, err
	}
	source := "pathname"
	if runtime.GOOS == "linux" {
		path = "/proc/self/exe"
		source = "running-image"
	}
	f, err := os.Open(path)
	if err != nil {
		return Identity{}, err
	}
	hash := sha256.New()
	_, readErr := io.Copy(hash, f)
	if err = errors.Join(readErr, f.Close()); err != nil {
		return Identity{}, err
	}
	id := Identity{Source: source, GoVersion: runtime.Version()}
	if source == "running-image" {
		id.SHA256 = hex.EncodeToString(hash.Sum(nil))
	} else {
		id.PathSHA256 = hex.EncodeToString(hash.Sum(nil))
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		addBuildInfo(&id, info)
	}
	return id, nil
})

func executableIdentity() (Identity, error) { return processIdentity() }

func addBuildInfo(id *Identity, info *debug.BuildInfo) {
	id.ModulePath = info.Main.Path
	id.ModuleVersion = info.Main.Version
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			id.GoVCSRevision = setting.Value
		case "vcs.modified":
			if setting.Value == "true" || setting.Value == "false" {
				modified := setting.Value == "true"
				id.GoVCSModified = &modified
			}
		}
	}
}
