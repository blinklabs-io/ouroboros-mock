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

package lifecycle_test

import (
	"context"
	"errors"
	"testing"

	"github.com/blinklabs-io/ouroboros-mock/internal/lifecycle"
	"github.com/stretchr/testify/require"
)

var errTest = errors.New("test")

type stopCtx struct{ hadDeadline bool }

func (s *stopCtx) Stop(ctx context.Context) error {
	_, s.hadDeadline = ctx.Deadline()
	return errTest
}

type stopErr struct{ called bool }

func (s *stopErr) Stop() error {
	s.called = true
	return errTest
}

type stopPlain struct{ called bool }

func (s *stopPlain) Stop() { s.called = true }

type closeCtx struct{ hadDeadline bool }

func (c *closeCtx) Close(ctx context.Context) error {
	_, c.hadDeadline = ctx.Deadline()
	return errTest
}

type closeErr struct{ called bool }

func (c *closeErr) Close() error {
	c.called = true
	return errTest
}

func TestStopCallsEitherSignature(t *testing.T) {
	withCtx := &stopCtx{}
	require.ErrorIs(t, lifecycle.Stop(context.Background(), withCtx), errTest)
	require.True(t, withCtx.hadDeadline, "Stop(ctx) got an unbounded context")

	withErr := &stopErr{}
	require.ErrorIs(t, lifecycle.Stop(context.Background(), withErr), errTest)
	require.True(t, withErr.called)

	plain := &stopPlain{}
	require.NoError(t, lifecycle.Stop(context.Background(), plain))
	require.True(t, plain.called)

	require.NoError(t, lifecycle.Stop(context.Background(), struct{}{}))
}

func TestCloseCallsEitherSignature(t *testing.T) {
	withCtx := &closeCtx{}
	require.ErrorIs(t, lifecycle.Close(context.Background(), withCtx), errTest)
	require.True(t, withCtx.hadDeadline, "Close(ctx) got an unbounded context")

	withErr := &closeErr{}
	require.ErrorIs(t, lifecycle.Close(context.Background(), withErr), errTest)
	require.True(t, withErr.called)

	require.NoError(t, lifecycle.Close(context.Background(), struct{}{}))
}
