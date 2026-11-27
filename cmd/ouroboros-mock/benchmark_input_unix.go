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
	"errors"
	"io"
	"os"
	"sync"
	"syscall"
)

type controlInput struct {
	*os.File
	original syscall.RawConn
	flags    uintptr
	once     sync.Once
	err      error
}

func (r *controlInput) Close() error {
	r.once.Do(func() {
		closeErr := r.File.Close()
		var restoreErr error
		controlErr := r.original.Control(func(fd uintptr) {
			_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_SETFL, r.flags)
			if errno != 0 {
				restoreErr = errno
			}
		})
		r.err = errors.Join(closeErr, controlErr, restoreErr)
	})
	return r.err
}

func benchmarkInput() (io.ReadCloser, error) {
	raw, err := os.Stdin.SyscallConn()
	if err != nil {
		return nil, err
	}
	var input *controlInput
	var setupErr error
	controlErr := raw.Control(func(original uintptr) {
		flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, original, syscall.F_GETFL, 0)
		if errno != 0 {
			setupErr = errno
			return
		}
		fd, err := syscall.Dup(int(original))
		if err != nil {
			setupErr = err
			return
		}
		// NewFile registers an already nonblocking descriptor with the poller.
		// Dup shares status flags with the inherited input, so Close restores them.
		if err = syscall.SetNonblock(fd, true); err != nil {
			_ = syscall.Close(fd)
			setupErr = err
			return
		}
		input = &controlInput{File: os.NewFile(uintptr(fd), "benchmark controls"), original: raw, flags: flags}
	})
	if err = errors.Join(controlErr, setupErr); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, errors.New("benchmark input descriptor was not initialized")
	}
	return input, nil
}
