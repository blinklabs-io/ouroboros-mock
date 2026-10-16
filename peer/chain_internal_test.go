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

package peer

import (
	"bytes"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
)

func TestChainRangeFromOriginIncludesNonOriginRoot(t *testing.T) {
	t.Parallel()
	prev := common.NewBlake2b256(bytes.Repeat([]byte{7}, 32))
	blocks, err := fixtures.GenerateConwayChain(1, prev, 100, 10, 3)
	require.NoError(t, err)
	chain, err := NewChain(blocks)
	require.NoError(t, err)

	got, ok := chain.rangeOf(
		pcommon.NewPointOrigin(),
		pointOf(blocks[len(blocks)-1]),
	)

	require.True(t, ok)
	require.Equal(t, blocks, got)
}
