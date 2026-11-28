//go:build linux

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
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdentityHashesRunningImageAfterPathReplacement(t *testing.T) {
	if os.Getenv("MOCK_IDENTITY_CHILD") == "1" {
		fmt.Println("ready")
		var input [1]byte
		_, err := io.ReadFull(os.Stdin, input[:])
		if err != nil {
			os.Exit(2)
		}
		id, err := executableIdentity()
		if err != nil {
			os.Exit(3)
		}
		if json.NewEncoder(os.Stdout).Encode(id) != nil {
			os.Exit(4)
		}
		os.Exit(0)
	}
	self, err := os.Executable()
	require.NoError(t, err)
	data, err := os.ReadFile(self)
	require.NoError(t, err)
	running := filepath.Join(t.TempDir(), "running")
	require.NoError(t, os.WriteFile(running, data, 0o700))
	cmd := exec.CommandContext(t.Context(), running, "-test.run=^TestIdentityHashesRunningImageAfterPathReplacement$")
	cmd.Env = append(os.Environ(), "MOCK_IDENTITY_CHILD=1")
	input, err := cmd.StdinPipe()
	require.NoError(t, err)
	output, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	waited := false
	t.Cleanup(func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	scanner := bufio.NewScanner(output)
	require.True(t, scanner.Scan())
	require.Equal(t, "ready", scanner.Text())
	replacement := running + ".replacement"
	require.NoError(t, os.WriteFile(replacement, []byte("different executable bytes"), 0o700))
	require.NoError(t, os.Rename(replacement, running))
	_, err = input.Write([]byte{1})
	require.NoError(t, err)
	require.NoError(t, input.Close())
	require.True(t, scanner.Scan())
	var id Identity
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &id))
	require.False(t, scanner.Scan())
	require.NoError(t, scanner.Err())
	waitErr := cmd.Wait()
	waited = true
	require.NoError(t, waitErr)
	sum := sha256.Sum256(data)
	require.Equal(t, hex.EncodeToString(sum[:]), id.SHA256)
	require.Equal(t, "running-image", id.Source)
	require.Empty(t, id.PathSHA256)
}
