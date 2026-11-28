//go:build unix

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
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func rejectsFIFO(t *testing.T, path string, read func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- read() }()
	select {
	case err := <-done:
		require.ErrorContains(t, err, "regular")
	case <-time.After(time.Second):
		// Release a blocking implementation so the failed control does not leak.
		fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_NONBLOCK, 0)
		require.NoError(t, err)
		require.NoError(t, syscall.Close(fd))
		<-done
		t.Fatal("FIFO open blocked before regular-file validation")
	}
}

func TestCorpusRejectsFIFOBeforeReading(t *testing.T) {
	path, m := mutableCorpus(t)
	fifo := filepath.Join(filepath.Dir(path), "manifest.fifo")
	require.NoError(t, syscall.Mkfifo(fifo, 0o600))
	rejectsFIFO(t, fifo, func() error { _, err := Load(fifo); return err })
	b, err := Load(path)
	require.NoError(t, err)
	artifact := m.Artifacts[0]
	target := filepath.Join(filepath.Dir(path), artifact.Path)
	require.NoError(t, os.Remove(target))
	require.NoError(t, syscall.Mkfifo(target, 0o600))
	rejectsFIFO(t, target, func() error { _, err := Load(path); return err })
	rejectsFIFO(t, target, func() error { _, err := b.Read(artifact.ID); return err })
}
