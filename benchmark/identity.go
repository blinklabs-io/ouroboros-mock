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
	SHA256        string `json:"executable_sha256"`
	GoVersion     string `json:"go_version"`
	ModuleVersion string `json:"module_version,omitempty"`
	Revision      string `json:"revision,omitempty"`
	Modified      bool   `json:"modified"`
}

var processIdentity = sync.OnceValues(func() (Identity, error) {
	path, err := os.Executable()
	if err != nil {
		return Identity{}, err
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
	id := Identity{SHA256: hex.EncodeToString(hash.Sum(nil)), GoVersion: runtime.Version()}
	if info, ok := debug.ReadBuildInfo(); ok {
		id.ModuleVersion = info.Main.Version
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				id.Revision = setting.Value
			case "vcs.modified":
				id.Modified = setting.Value == "true"
			}
		}
	}
	return id, nil
})

func executableIdentity() (Identity, error) { return processIdentity() }
