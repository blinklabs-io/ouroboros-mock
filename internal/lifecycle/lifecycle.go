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

// Package lifecycle stops and closes gOuroboros values whose Stop and Close
// may or may not take a context, so this module builds against gOuroboros
// releases on either side of that change.
package lifecycle

import (
	"context"
	"time"
)

// Timeout bounds a Stop or Close that waits for goroutines to return.
const Timeout = 5 * time.Second

// Stop calls x.Stop with ctx, further bounded by Timeout, when x's Stop takes
// a context, and x.Stop() otherwise. It returns nil when x has no Stop.
func Stop(ctx context.Context, x any) error {
	switch s := x.(type) {
	case interface{ Stop(context.Context) error }:
		ctx, cancel := context.WithTimeout(ctx, Timeout)
		defer cancel()
		return s.Stop(ctx)
	case interface{ Stop() error }:
		return s.Stop()
	case interface{ Stop() }:
		s.Stop()
	}
	return nil
}

// Close calls x.Close with ctx, further bounded by Timeout, when x's Close
// takes a context, and x.Close() otherwise. It returns nil when x has no
// Close.
func Close(ctx context.Context, x any) error {
	switch c := x.(type) {
	case interface{ Close(context.Context) error }:
		ctx, cancel := context.WithTimeout(ctx, Timeout)
		defer cancel()
		return c.Close(ctx)
	case interface{ Close() error }:
		return c.Close()
	}
	return nil
}
