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
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/blinklabs-io/ouroboros-mock/benchmark"
	"github.com/blinklabs-io/ouroboros-mock/peer"
	"github.com/stretchr/testify/require"
)

func TestBenchmarkExecutableReportsCompletionAndFailures(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "ouroboros-mock")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, ".")
	data, err := build.CombinedOutput()
	require.NoError(t, err, string(data))
	for _, mode := range []string{"complete", "eof", "signal", "writer"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.CopyFS(root, os.DirFS("../../benchmark/testdata/reference")))
			path := filepath.Join(root, "upstream.json")
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			var m benchmark.Manifest
			require.NoError(t, json.Unmarshal(raw, &m))
			m.DurationNS = int64(time.Second)
			m.Roles[0].Initial = m.Roles[0].Initial[:3]
			m.Roles[0].Required = []benchmark.Requirement{}
			if mode == "complete" {
				m.Roles[0].Required = []benchmark.Requirement{{Event: "block_send_call", Items: 3}}
			}
			raw, err = json.Marshal(m)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, raw, 0o600))
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, "benchmark", "--manifest", path, "--run-id", "cli-trial", "--endpoint", "upstream=127.0.0.1:0")
			stdin, err := cmd.StdinPipe()
			require.NoError(t, err)
			stdout, err := cmd.StdoutPipe()
			require.NoError(t, err)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			require.NoError(t, cmd.Start())
			records := make(chan benchmark.Record, 128)
			scanDone := make(chan struct{})
			go func() {
				defer close(scanDone)
				defer close(records)
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					var r benchmark.Record
					if json.Unmarshal(scanner.Bytes(), &r) == nil {
						records <- r
					}
				}
			}()
			next := func(kind string) benchmark.Record {
				for {
					select {
					case r, ok := <-records:
						require.True(t, ok, "missing %s", kind)
						if r.Kind == kind {
							return r
						}
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
				}
			}
			prepared := next("prepared")
			require.NotNil(t, prepared.Build)
			artifact, err := os.ReadFile(binary)
			require.NoError(t, err)
			digest := sha256.Sum256(artifact)
			if runtime.GOOS == "linux" {
				require.Equal(t, hex.EncodeToString(digest[:]), prepared.Build.SHA256)
			} else {
				require.Empty(t, prepared.Build.SHA256)
				require.Equal(t, hex.EncodeToString(digest[:]), prepared.Build.PathSHA256)
			}
			require.Equal(t, "github.com/blinklabs-io/ouroboros-mock", prepared.Build.ModulePath)
			ready := next("ready")
			var follower *peer.Follower
			var followerDone chan struct{}
			if mode == "complete" {
				conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", ready.Endpoint)
				require.NoError(t, err)
				follower, err = peer.NewFollowerContext(ctx, peer.FollowerConfig{Conn: conn, NetworkMagic: 42, FetchBlocks: true})
				require.NoError(t, err)
				defer follower.Close()
				followerDone = make(chan struct{})
				go func() {
					for range follower.Events() {
					}
					close(followerDone)
				}()
			}
			require.NoError(t, json.NewEncoder(stdin).Encode(benchmark.Control{Version: 1, Kind: "start", Run: "cli-trial", Start: time.Now()}))
			next("started")
			switch mode {
			case "eof":
				require.NoError(t, stdin.Close())
			case "signal":
				require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))
			case "writer":
				require.NoError(t, stdout.Close())
			}
			<-scanDone
			err = cmd.Wait()
			if mode == "complete" {
				require.NoError(t, err, stderr.String())
			} else {
				require.Error(t, err)
			}
			var final *bool
			for r := range records {
				if r.Kind == "final" {
					final = r.Complete
				}
			}
			if mode != "writer" {
				require.NotNil(t, final)
				require.Equal(t, mode == "complete", *final)
			}
			if follower != nil {
				require.NoError(t, follower.Close())
				<-followerDone
			}
		})
	}
}
