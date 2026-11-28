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
	streams, err := ownBenchmarkStreams(os.Stdin)
	require.NoError(t, err)
	require.NoError(t, streams[0].Close())
	require.NoError(t, streams[0].Close())
	require.NoError(t, raw.Control(func(fd uintptr) {
		flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFL, 0)
		require.Zero(t, errno)
		require.Equal(t, before, flags)
	}))
}

func TestBenchmarkStreamsRestoreSharedFlagsOnlyAfterBothClose(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()
	raw, err := r.SyscallConn()
	require.NoError(t, err)
	require.NoError(t, raw.Control(func(fd uintptr) { require.NoError(t, syscall.SetNonblock(int(fd), false)) }))
	flags := func() uintptr {
		var result uintptr
		require.NoError(t, raw.Control(func(fd uintptr) {
			value, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFL, 0)
			require.Zero(t, errno)
			result = value
		}))
		return result
	}
	before := flags()
	inherited := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = inherited }()
	input, output, err := benchmarkStreams(r)
	require.NoError(t, err)
	require.NoError(t, input.Close())
	require.NotZero(t, flags()&syscall.O_NONBLOCK)
	require.NoError(t, output.Close())
	require.Equal(t, before, flags())
}

func descriptorFlags(t *testing.T, file *os.File) uintptr {
	t.Helper()
	raw, err := file.SyscallConn()
	require.NoError(t, err)
	var result uintptr
	require.NoError(t, raw.Control(func(fd uintptr) {
		flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFL, 0)
		require.Zero(t, errno)
		result = flags
	}))
	return result
}

func TestBenchmarkRejectsUnpollableStreamsAndRestoresFlags(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()
	file, err := os.CreateTemp(t.TempDir(), "unpollable")
	require.NoError(t, err)
	defer file.Close()
	raw, err := r.SyscallConn()
	require.NoError(t, err)
	require.NoError(t, raw.Control(func(fd uintptr) { require.NoError(t, syscall.SetNonblock(int(fd), false)) }))
	inputFlags, outputFlags := descriptorFlags(t, r), descriptorFlags(t, file)
	streams, err := ownBenchmarkStreams(r, file)
	defer func() {
		for _, stream := range streams {
			_ = stream.Close()
		}
	}()
	require.Nil(t, streams)
	require.ErrorContains(t, err, "support polling")
	require.Equal(t, inputFlags, descriptorFlags(t, r))
	require.Equal(t, outputFlags, descriptorFlags(t, file))
	_, err = ownBenchmarkStreams(file)
	require.ErrorContains(t, err, "support polling")
	require.Equal(t, outputFlags, descriptorFlags(t, file))
}
