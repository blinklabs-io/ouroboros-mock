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

package chainsync

import (
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	gchainsync "github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
)

// OriginPoint returns the point representing the origin of the chain, suitable
// for a FindIntersect request that should always match.
func OriginPoint() pcommon.Point {
	return pcommon.NewPointOrigin()
}

// PointOf returns the chain-sync point (slot + block hash) for a block.
func PointOf(block ledger.Block) pcommon.Point {
	return pcommon.NewPoint(block.SlotNumber(), block.Hash().Bytes())
}

// TipOf returns the chain-sync tip (point + block number) for a block, as a
// server callback would report when the block is the current chain tip.
func TipOf(block ledger.Block) gchainsync.Tip {
	return gchainsync.Tip{
		Point:       PointOf(block),
		BlockNumber: block.BlockNumber(),
	}
}

// Chain is a generated sequence of connected blocks together with the
// chain-sync points and tips derived from them. The three slices are parallel:
// Points[i] and Tips[i] describe Blocks[i].
type Chain struct {
	Blocks []ledger.Block
	Points []pcommon.Point
	Tips   []gchainsync.Tip
}

// Tip returns the tip of the whole chain (the point and block number of the
// last block). It returns a zero tip for an empty chain.
func (c Chain) Tip() gchainsync.Tip {
	if len(c.Tips) == 0 {
		return gchainsync.Tip{}
	}
	return c.Tips[len(c.Tips)-1]
}

// Len returns the number of blocks in the chain.
func (c Chain) Len() int {
	return len(c.Blocks)
}

// BuildChain generates count connected Conway blocks and derives the matching
// chain-sync points and tips. It composes
// [github.com/blinklabs-io/ouroboros-mock/fixtures.GenerateConwayChain]; see
// that function for the block layout (slots, block numbers, and prev-hash
// linkage). A count of zero or less returns an empty, non-nil Chain.
func BuildChain(
	startBlockNumber uint64,
	prevHash common.Blake2b256,
	startSlot, slotIncrement uint64,
	count int,
) (Chain, error) {
	blocks, err := fixtures.GenerateConwayChain(
		startBlockNumber,
		prevHash,
		startSlot,
		slotIncrement,
		count,
	)
	if err != nil {
		return Chain{}, err
	}
	chain := Chain{
		Blocks: blocks,
		Points: make([]pcommon.Point, len(blocks)),
		Tips:   make([]gchainsync.Tip, len(blocks)),
	}
	for i, block := range blocks {
		chain.Points[i] = PointOf(block)
		chain.Tips[i] = TipOf(block)
	}
	return chain, nil
}
