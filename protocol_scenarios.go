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

package ouroboros_mock

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
)

// ChainSyncForwardScenarioNtN builds a finite node-to-node forward sequence ending with client Done.
func ChainSyncForwardScenarioNtN(
	era, byronType uint,
	headers [][]byte,
	tip chainsync.Tip,
) ([]ConversationEntry, error) {
	entries, err := ChainSyncScenario(era, byronType, headers, tip)
	if err != nil {
		return nil, err
	}
	entries[len(entries)-1] = ChainSyncDone(false)
	return entries, nil
}

// ChainSyncForwardScenarioNtC builds a finite node-to-client forward sequence ending with client Done.
// Each block has its corresponding wire block type in blockTypes.
func ChainSyncForwardScenarioNtC(
	blockTypes []uint,
	blocks [][]byte,
	tip chainsync.Tip,
) ([]ConversationEntry, error) {
	if len(blockTypes) != len(blocks) {
		return nil, errors.New("block types and blocks must have equal lengths")
	}
	entries := make([]ConversationEntry, 0, 2*len(blocks)+1)
	for i, block := range blocks {
		response, err := ChainSyncRollForwardNtC(blockTypes[i], block, tip)
		if err != nil {
			return nil, err
		}
		entries = append(entries, ChainSyncRequestNext(true), response)
	}
	return append(entries, ChainSyncDone(true)), nil
}

// ChainSyncRollbackScenario builds a request, rollback response, and client Done sequence.
func ChainSyncRollbackScenario(
	nodeToClient bool,
	point pcommon.Point,
	tip chainsync.Tip,
) []ConversationEntry {
	return []ConversationEntry{
		ChainSyncRequestNext(nodeToClient),
		ChainSyncRollBackward(nodeToClient, point, tip),
		ChainSyncDone(nodeToClient),
	}
}

// ChainSyncIntersectionScenario builds an intersection exchange followed by client Done.
// A nil intersection returns IntersectNotFound; otherwise the intersection must be one of the requested points.
func ChainSyncIntersectionScenario(
	nodeToClient bool,
	points []pcommon.Point,
	intersection *pcommon.Point,
	tip chainsync.Tip,
) ([]ConversationEntry, error) {
	entries := []ConversationEntry{ChainSyncFindIntersect(nodeToClient, points)}
	if intersection == nil {
		entries = append(entries, ChainSyncIntersectNotFound(nodeToClient, tip))
	} else {
		found := false
		for _, point := range points {
			if point.Slot == intersection.Slot && bytes.Equal(point.Hash, intersection.Hash) {
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New("intersection must be one of the requested points")
		}
		entries = append(entries, ChainSyncIntersectFound(nodeToClient, *intersection, tip))
	}
	return append(entries, ChainSyncDone(nodeToClient)), nil
}

// BlockFetchScenario builds a range request, its complete batch or NoBlocks response, and client Done.
// Each raw ledger block has its corresponding wire block type in blockTypes.
func BlockFetchScenario(
	start, end pcommon.Point,
	blockTypes []uint,
	blocks [][]byte,
) ([]ConversationEntry, error) {
	if len(blockTypes) != len(blocks) {
		return nil, errors.New("block types and blocks must have equal lengths")
	}
	entries := []ConversationEntry{BlockFetchRequestRange(start, end)}
	if len(blocks) == 0 {
		entries = append(entries, BlockFetchNoBlocks())
	} else {
		entries = append(entries, BlockFetchStartBatch())
		for i, block := range blocks {
			response, err := BlockFetchBlockResponse(blockTypes[i], block)
			if err != nil {
				return nil, err
			}
			entries = append(entries, response)
		}
		entries = append(entries, BlockFetchBatchDone())
	}
	return append(entries, BlockFetchClientDone()), nil
}

// TxSubmissionScenario builds an ID exchange, body exchange, acknowledgement, and client Done.
// IDs and bodies describe matching transactions in the same order. The final blocking ID request permits Done.
func TxSubmissionScenario(
	ids []txsubmission.TxIdAndSize,
	txs []txsubmission.TxBody,
) ([]ConversationEntry, error) {
	if len(ids) != len(txs) {
		return nil, errors.New("transaction IDs and bodies must have equal lengths")
	}
	count := len(ids)
	if count > math.MaxUint16 {
		return nil, errors.New("transaction count exceeds the protocol request limit")
	}
	for i, id := range ids {
		if id.TxId.EraId != txs[i].EraId {
			return nil, fmt.Errorf("transaction %d ID and body must have equal era IDs", i)
		}
	}
	requestCount := uint16(count)
	entries := []ConversationEntry{TxSubmissionInit()}
	if len(ids) > 0 {
		requested := make([]txsubmission.TxId, len(ids))
		for i, id := range ids {
			requested[i] = id.TxId
		}
		entries = append(
			entries,
			TxSubmissionRequestTxIds(false, 0, requestCount),
			TxSubmissionReplyTxIds(ids),
			TxSubmissionRequestTxs(requested),
			TxSubmissionReplyTxs(txs),
		)
	}
	entries = append(entries, TxSubmissionRequestTxIds(true, requestCount, 1), TxSubmissionDone())
	return entries, nil
}
