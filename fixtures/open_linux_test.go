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

package fixtures

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFixtureOpenDoesNotBlockOnFIFO(t *testing.T) {
	root := t.TempDir()
	name := "capture.json"
	path := filepath.Join(root, name)
	require.NoError(t, syscall.Mkfifo(path, 0o600))
	for _, read := range []func() error{
		func() error { _, err := openRegularFileInRoot(root, name); return err },
		func() error { _, err := (Fixture{Path: path}).Read(); return err },
	} {
		result := make(chan error, 1)
		go func() { result <- read() }()
		select {
		case err := <-result:
			require.ErrorIs(t, err, errFixtureNotRegular)
		case <-time.After(time.Second):
			// Release a blocking reader before reporting the regression, so the test
			// does not leave a goroutine stranded in the kernel.
			writer, err := syscall.Open(
				path,
				syscall.O_WRONLY|syscall.O_NONBLOCK,
				0,
			)
			require.NoError(t, err)
			require.NoError(t, syscall.Close(writer))
			require.ErrorIs(t, <-result, errFixtureNotRegular)
			t.Error("fixture FIFO open waited for a writer")
		}
	}
}
