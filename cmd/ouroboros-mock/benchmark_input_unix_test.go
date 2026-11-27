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

package main

import (
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBenchmarkControlCloseRestoresInheritedDescriptorFlags(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()
	raw, err := r.SyscallConn()
	require.NoError(t, err)
	require.NoError(t, raw.Control(func(fd uintptr) { require.NoError(t, syscall.SetNonblock(int(fd), false)) }))
	var before uintptr
	require.NoError(t, raw.Control(func(fd uintptr) {
		flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFL, 0)
		require.Zero(t, errno)
		before = flags
	}))
	inherited := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = inherited }()
	input, err := benchmarkInput()
	require.NoError(t, err)
	require.NoError(t, input.Close())
	require.NoError(t, input.Close())
	require.NoError(t, raw.Control(func(fd uintptr) {
		flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFL, 0)
		require.Zero(t, errno)
		require.Equal(t, before, flags)
	}))
}
