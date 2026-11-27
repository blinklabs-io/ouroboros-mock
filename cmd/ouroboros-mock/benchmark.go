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
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/blinklabs-io/ouroboros-mock/benchmark"
)

type endpoints map[string]string

func (e endpoints) String() string { return "role=address" }
func (e endpoints) Set(value string) error {
	role, address, ok := strings.Cut(value, "=")
	if !ok || role == "" || address == "" {
		return errors.New("endpoint must be role=host:port")
	}
	if _, exists := e[role]; exists {
		return fmt.Errorf("duplicate endpoint %q", role)
	}
	e[role] = address
	return nil
}

func runBenchmark(ctx context.Context, args []string, input io.ReadCloser, output io.Writer) error {
	flags := flag.NewFlagSet("benchmark", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	manifest := flags.String("manifest", "", "versioned experiment manifest")
	runID := flags.String("run-id", "", "controller run identity")
	endpoint := endpoints{}
	flags.Var(endpoint, "endpoint", "role=host:port (repeatable)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *manifest == "" || *runID == "" || flags.NArg() != 0 {
		return errors.New("usage: ouroboros-mock benchmark --manifest FILE --run-id ID --endpoint role=host:port")
	}
	bundle, err := benchmark.Load(*manifest)
	if err != nil {
		return err
	}
	wanted := map[string]bool{}
	for _, role := range bundle.Roles() {
		wanted[role.ID] = true
	}
	for role := range endpoint {
		if !wanted[role] {
			return fmt.Errorf("endpoint names unknown role %q", role)
		}
	}
	return benchmark.Serve(ctx, bundle, benchmark.Options{Run: *runID, Endpoints: endpoint}, input, output)
}
