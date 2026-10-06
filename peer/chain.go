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
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/blinklabs-io/gouroboros/ledger"
	gchainsync "github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
)

var (
	// ErrNotLinked is returned when blocks do not chain together through their
	// previous-hash links.
	ErrNotLinked = errors.New("blocks are not linked")
	// ErrNoIntersection is returned when a fork does not attach to the
	// selected chain.
	ErrNoIntersection = errors.New("fork does not attach to the selected chain")
)

// Chain is a fixture chain: a selected sequence of blocks plus every block the
// chain has ever held, so that blocks on an abandoned branch stay fetchable
// after a fork switch. It is safe for concurrent use.
type Chain struct {
	mu       sync.RWMutex
	selected []ledger.Block
	byHash   map[string]ledger.Block
}

// NewChain returns a chain holding blocks, which must be linked through their
// previous-hash fields. The first block may attach to any previous hash.
func NewChain(blocks []ledger.Block) (*Chain, error) {
	c := &Chain{byHash: make(map[string]ledger.Block, len(blocks))}
	if err := c.Append(blocks...); err != nil {
		return nil, err
	}
	return c, nil
}

// Append extends the selected chain. The first block must link to the current
// tip; on an empty chain it may attach to any previous hash.
func (c *Chain) Append(blocks ...ledger.Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(blocks) == 0 {
		return nil
	}
	if len(c.selected) > 0 {
		tip := c.selected[len(c.selected)-1]
		if !bytes.Equal(
			blocks[0].PrevHash().Bytes(),
			tip.Hash().Bytes(),
		) {
			return fmt.Errorf(
				"%w: block %d does not extend the tip",
				ErrNotLinked,
				blocks[0].BlockNumber(),
			)
		}
	}
	if err := checkLinked(blocks); err != nil {
		return err
	}
	for _, b := range blocks {
		c.selected = append(c.selected, b)
		c.byHash[string(b.Hash().Bytes())] = b
	}
	return nil
}

// SwitchFork replaces the selected chain's tail with fork. The first fork
// block's previous hash selects the intersection: a selected block, or the
// chain's base (origin) when it equals the first selected block's previous
// hash. It returns the intersection point, which is the origin point when the
// fork attaches at the base. The abandoned blocks stay fetchable by hash.
func (c *Chain) SwitchFork(fork []ledger.Block) (pcommon.Point, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(fork) == 0 {
		return pcommon.Point{}, fmt.Errorf("%w: empty fork", ErrNotLinked)
	}
	if err := checkLinked(fork); err != nil {
		return pcommon.Point{}, err
	}
	prev := fork[0].PrevHash().Bytes()
	keep := -1
	found := false
	for i, b := range c.selected {
		if bytes.Equal(b.Hash().Bytes(), prev) {
			keep, found = i, true
			break
		}
	}
	if !found {
		if len(c.selected) == 0 ||
			!bytes.Equal(c.selected[0].PrevHash().Bytes(), prev) {
			return pcommon.Point{}, ErrNoIntersection
		}
	}
	intersection := pcommon.NewPointOrigin()
	if keep >= 0 {
		intersection = pointOf(c.selected[keep])
	}
	c.selected = append(
		append([]ledger.Block(nil), c.selected[:keep+1]...),
		fork...,
	)
	for _, b := range fork {
		c.byHash[string(b.Hash().Bytes())] = b
	}
	return intersection, nil
}

func (c *Chain) clone() *Chain {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &Chain{
		selected: append([]ledger.Block(nil), c.selected...),
		byHash:   maps.Clone(c.byHash),
	}
}

// Tip returns the selected chain's tip. An empty chain reports the origin.
func (c *Chain) Tip() gchainsync.Tip {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tipLocked()
}

func (c *Chain) tipLocked() gchainsync.Tip {
	if len(c.selected) == 0 {
		return gchainsync.Tip{}
	}
	last := c.selected[len(c.selected)-1]
	return gchainsync.Tip{
		Point:       pointOf(last),
		BlockNumber: last.BlockNumber(),
	}
}

// Blocks returns a copy of the selected chain.
func (c *Chain) Blocks() []ledger.Block {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]ledger.Block(nil), c.selected...)
}

// indexOfLocked returns the position of point on the selected chain. The
// origin point is position -1.
func (c *Chain) indexOfLocked(point pcommon.Point) (int, bool) {
	if point.Slot == 0 && len(point.Hash) == 0 {
		return -1, true
	}
	for i, b := range c.selected {
		if b.SlotNumber() == point.Slot &&
			bytes.Equal(b.Hash().Bytes(), point.Hash) {
			return i, true
		}
	}
	return 0, false
}

// next describes what follows a client position on the selected chain.
type next struct {
	// block is the next block to roll forward to, or nil when the client is
	// at the tip.
	block ledger.Block
	// rollback is set when the position is no longer on the selected chain.
	rollback bool
	// point is the rollback target when rollback is set.
	point pcommon.Point
	tip   gchainsync.Tip
}

// follow reports what a client holding path (oldest first) should be sent
// next. A path that has left the selected chain resolves to its newest entry
// that is still selected.
func (c *Chain) follow(path []pcommon.Point) (next, []pcommon.Point) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	n := next{tip: c.tipLocked()}
	for i := len(path) - 1; i >= 0; i-- {
		idx, ok := c.indexOfLocked(path[i])
		if !ok {
			continue
		}
		if i != len(path)-1 {
			n.rollback = true
			n.point = path[i]
			return n, path[:i+1]
		}
		if idx+1 < len(c.selected) {
			n.block = c.selected[idx+1]
		}
		return n, path
	}
	origin := pcommon.NewPointOrigin()
	n.rollback = true
	n.point = origin
	return n, []pcommon.Point{origin}
}

// find resolves the first of points on the selected chain.
func (c *Chain) find(
	points []pcommon.Point,
) (pcommon.Point, gchainsync.Tip, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, p := range points {
		if _, ok := c.indexOfLocked(p); ok {
			return p, c.tipLocked(), true
		}
	}
	return pcommon.Point{}, c.tipLocked(), false
}

// rangeOf returns the blocks from start to end inclusive by walking
// previous-hash links back from end, so either branch of a fork can be
// fetched. It reports false when start is not an ancestor of end.
func (c *Chain) rangeOf(start, end pcommon.Point) ([]ledger.Block, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	startAtOrigin := start.Slot == 0 && len(start.Hash) == 0
	var out []ledger.Block
	hash := end.Hash
	for {
		b, ok := c.byHash[string(hash)]
		if !ok {
			return nil, false
		}
		out = append(out, b)
		if bytes.Equal(b.Hash().Bytes(), start.Hash) {
			if b.SlotNumber() != start.Slot {
				return nil, false
			}
			break
		}
		hash = b.PrevHash().Bytes()
		if startAtOrigin {
			if _, ok := c.byHash[string(hash)]; !ok {
				break
			}
		}
	}
	if out[0].SlotNumber() != end.Slot {
		return nil, false
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, true
}

func checkLinked(blocks []ledger.Block) error {
	for i := 1; i < len(blocks); i++ {
		if !bytes.Equal(
			blocks[i].PrevHash().Bytes(),
			blocks[i-1].Hash().Bytes(),
		) {
			return fmt.Errorf(
				"%w: block %d does not follow block %d",
				ErrNotLinked,
				blocks[i].BlockNumber(),
				blocks[i-1].BlockNumber(),
			)
		}
	}
	return nil
}

func pointOf(b ledger.Block) pcommon.Point {
	return pcommon.NewPoint(b.SlotNumber(), b.Hash().Bytes())
}
