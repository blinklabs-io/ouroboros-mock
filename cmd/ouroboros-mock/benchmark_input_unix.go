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

type descriptorState struct {
	raw   syscall.RawConn
	flags uintptr
}
type streamGroup struct {
	mu        sync.Mutex
	remaining int
	states    []descriptorState
}
type controlInput struct {
	*os.File
	group *streamGroup
	once  sync.Once
	err   error
}

func (r *controlInput) Close() error {
	r.once.Do(func() {
		r.err = r.File.Close()
		r.group.mu.Lock()
		defer r.group.mu.Unlock()
		r.group.remaining--
		if r.group.remaining != 0 {
			return
		}
		r.err = errors.Join(r.err, r.group.restore())
	})
	return r.err
}

func (g *streamGroup) restore() error {
	var result error
	for _, state := range g.states {
		var restoreErr error
		controlErr := state.raw.Control(func(fd uintptr) {
			_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_SETFL, state.flags)
			if errno != 0 {
				restoreErr = errno
			}
		})
		result = errors.Join(result, controlErr, restoreErr)
	}
	return result
}

// Snapshot all inherited flags before Dup changes any shared open-file description.
// Restore them only after both pollable copies have closed.
func ownBenchmarkStreams(files ...*os.File) ([]*controlInput, error) {
	group := &streamGroup{}
	for _, file := range files {
		raw, err := file.SyscallConn()
		if err != nil {
			return nil, err
		}
		state := descriptorState{raw: raw}
		var flagErr error
		err = raw.Control(func(fd uintptr) {
			flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFL, 0)
			if errno != 0 {
				flagErr = errno
			}
			state.flags = flags
		})
		if err = errors.Join(err, flagErr); err != nil {
			return nil, err
		}
		group.states = append(group.states, state)
	}
	streams := make([]*controlInput, 0, len(files))
	for _, state := range group.states {
		var file *os.File
		var setupErr error
		err := state.raw.Control(func(original uintptr) {
			fd, err := syscall.Dup(int(original))
			if err != nil {
				setupErr = err
				return
			}
			if err = syscall.SetNonblock(fd, true); err != nil {
				_ = syscall.Close(fd)
				setupErr = err
				return
			}
			file = os.NewFile(uintptr(fd), "benchmark stream")
		})
		if err = errors.Join(err, setupErr); err != nil {
			for _, stream := range streams {
				_ = stream.File.Close()
			}
			return nil, errors.Join(err, group.restore())
		}
		streams = append(streams, &controlInput{File: file, group: group})
	}
	group.remaining = len(streams)
	return streams, nil
}

func benchmarkStreams(output *os.File) (io.ReadCloser, io.WriteCloser, error) {
	streams, err := ownBenchmarkStreams(os.Stdin, output)
	if err != nil {
		return nil, nil, err
	}
	return streams[0], streams[1], nil
}
