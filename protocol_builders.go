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
	"errors"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/gouroboros/protocol/localtxmonitor"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
)

// NewPoint returns a point suitable for mini-protocol messages.
func NewPoint(slot uint64, hash []byte) pcommon.Point {
	return pcommon.NewPoint(slot, hash)
}

// OriginPoint returns the chain origin point.
func OriginPoint() pcommon.Point { return pcommon.NewPointOrigin() }

// NewTip returns a tip containing the supplied point and block number.
func NewTip(point pcommon.Point, blockNumber uint64) chainsync.Tip {
	return chainsync.Tip{Point: point, BlockNumber: blockNumber}
}

// ChainSyncRequestNext builds a node-to-node or node-to-client request-next
// input entry. nodeToClient selects the decoder for the negotiated protocol
// mode; the returned entry always describes a request received by the server
// under test.
func ChainSyncRequestNext(nodeToClient bool) ConversationEntryInput {
	protocolID := chainsync.ProtocolIdNtN
	messageFromCbor := chainsync.NewMsgFromCborNtN
	if nodeToClient {
		protocolID = chainsync.ProtocolIdNtC
		messageFromCbor = chainsync.NewMsgFromCborNtC
	}
	return ConversationEntryInput{
		ProtocolId:      protocolID,
		Message:         chainsync.NewMsgRequestNext(),
		MessageType:     chainsync.MessageTypeRequestNext,
		MsgFromCborFunc: messageFromCbor,
		IsResponse:      false,
	}
}

// ChainSyncRollForwardNtN builds a node-to-node roll-forward output entry.
func ChainSyncRollForwardNtN(era, byronType uint, header []byte, tip chainsync.Tip) (ConversationEntryOutput, error) {
	message, err := chainsync.NewMsgRollForwardNtN(era, byronType, header, tip)
	if err != nil {
		return ConversationEntryOutput{}, err
	}
	return ConversationEntryOutput{
		ProtocolId: chainsync.ProtocolIdNtN,
		IsResponse: true,
		Messages: []protocol.Message{
			message,
		},
	}, nil
}

// ChainSyncRollForwardNtC builds a node-to-client roll-forward output entry.
func ChainSyncRollForwardNtC(blockType uint, block []byte, tip chainsync.Tip) (ConversationEntryOutput, error) {
	message, err := chainsync.NewMsgRollForwardNtC(blockType, block, tip)
	if err != nil {
		return ConversationEntryOutput{}, err
	}
	return ConversationEntryOutput{
		ProtocolId: chainsync.ProtocolIdNtC,
		IsResponse: true,
		Messages: []protocol.Message{
			message,
		},
	}, nil
}

// chainSyncMode returns the protocol ID and message decoder for the
// negotiated chain-sync mode. Node-to-node and node-to-client chain-sync
// run on different mini-protocol IDs, so an entry built for one mode is
// not interchangeable with the other.
func chainSyncMode(nodeToClient bool) (uint16, protocol.MessageFromCborFunc) {
	if nodeToClient {
		return chainsync.ProtocolIdNtC, chainsync.NewMsgFromCborNtC
	}
	return chainsync.ProtocolIdNtN, chainsync.NewMsgFromCborNtN
}

// ChainSyncRollBackward builds a rollback output entry. nodeToClient
// selects the mini-protocol the segment is sent on.
func ChainSyncRollBackward(nodeToClient bool, point pcommon.Point, tip chainsync.Tip) ConversationEntryOutput {
	protocolID, _ := chainSyncMode(nodeToClient)
	return ConversationEntryOutput{
		ProtocolId: protocolID,
		IsResponse: true,
		Messages: []protocol.Message{
			chainsync.NewMsgRollBackward(point, tip),
		},
	}
}

// ChainSyncFindIntersect builds a find-intersect input entry. nodeToClient
// selects the mini-protocol and decoder for the negotiated mode.
func ChainSyncFindIntersect(nodeToClient bool, points []pcommon.Point) ConversationEntryInput {
	protocolID, messageFromCbor := chainSyncMode(nodeToClient)
	if points == nil {
		points = []pcommon.Point{}
	}
	return ConversationEntryInput{
		ProtocolId:      protocolID,
		Message:         chainsync.NewMsgFindIntersect(points),
		MessageType:     chainsync.MessageTypeFindIntersect,
		MsgFromCborFunc: messageFromCbor,
	}
}

// ChainSyncScenario returns a server-side forward chain-sync conversation.
func ChainSyncScenario(era, byronType uint, headers [][]byte, tip chainsync.Tip) ([]ConversationEntry, error) {
	conversation := []ConversationEntry{ChainSyncRequestNext(false)}
	for _, header := range headers {
		entry, err := ChainSyncRollForwardNtN(era, byronType, header, tip)
		if err != nil {
			return nil, err
		}
		conversation = append(conversation, entry)
		conversation = append(conversation, ChainSyncRequestNext(false))
	}
	return conversation, nil
}

// BlockFetchRequestRange builds a block-fetch request input entry.
func BlockFetchRequestRange(start, end pcommon.Point) ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      blockfetch.ProtocolId,
		Message:         blockfetch.NewMsgRequestRange(start, end),
		MessageType:     blockfetch.MessageTypeRequestRange,
		MsgFromCborFunc: blockfetch.NewMsgFromCbor,
	}
}

// BlockFetchNoBlocks builds a no-blocks response output entry.
func BlockFetchNoBlocks() ConversationEntryOutput {
	return ConversationEntryOutput{
		ProtocolId: blockfetch.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			blockfetch.NewMsgNoBlocks(),
		},
	}
}

// BlockFetchBlock builds a block response output entry.
func BlockFetchBlock(block []byte) ConversationEntryOutput {
	return ConversationEntryOutput{
		ProtocolId: blockfetch.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			blockfetch.NewMsgStartBatch(),
			blockfetch.NewMsgBlock(block),
			blockfetch.NewMsgBatchDone(),
		},
	}
}

// TxSubmissionInit builds the init input entry that opens a
// transaction-submission conversation. The client holds agency in the
// Init state and sends this message before the server may request
// transaction IDs.
func TxSubmissionInit() ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      txsubmission.ProtocolId,
		Message:         txsubmission.NewMsgInit(),
		MessageType:     txsubmission.MessageTypeInit,
		MsgFromCborFunc: txsubmission.NewMsgFromCbor,
	}
}

// TxSubmissionRequestTxIds builds a transaction-ID request output entry.
// The server holds agency in the Idle state, so the mock sends this
// message and the client answers with a reply.
func TxSubmissionRequestTxIds(blocking bool, ack, request uint16) ConversationEntryOutput {
	return ConversationEntryOutput{
		ProtocolId: txsubmission.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			txsubmission.NewMsgRequestTxIds(blocking, ack, request),
		},
	}
}

// TxSubmissionRequestTxs builds a transaction request output entry. The
// server holds agency in the Idle state, so the mock sends this message.
func TxSubmissionRequestTxs(ids []txsubmission.TxId) ConversationEntryOutput {
	return ConversationEntryOutput{
		ProtocolId: txsubmission.ProtocolId,
		IsResponse: true,
		Messages: []protocol.Message{
			txsubmission.NewMsgRequestTxs(ids),
		},
	}
}

// TxSubmissionReplyTxIds builds a transaction-ID reply input entry. The
// client holds agency in both TxIds states, so the mock receives this
// message in answer to a request.
func TxSubmissionReplyTxIds(ids []txsubmission.TxIdAndSize) ConversationEntryInput {
	if ids == nil {
		ids = []txsubmission.TxIdAndSize{}
	}
	return ConversationEntryInput{
		ProtocolId:      txsubmission.ProtocolId,
		Message:         txsubmission.NewMsgReplyTxIds(ids),
		MessageType:     txsubmission.MessageTypeReplyTxIds,
		MsgFromCborFunc: txsubmission.NewMsgFromCbor,
	}
}

// TxSubmissionReplyTxs builds a transaction reply input entry. The client
// holds agency in the Txs state, so the mock receives this message.
func TxSubmissionReplyTxs(txs []txsubmission.TxBody) ConversationEntryInput {
	if txs == nil {
		txs = []txsubmission.TxBody{}
	}
	return ConversationEntryInput{
		ProtocolId:      txsubmission.ProtocolId,
		Message:         txsubmission.NewMsgReplyTxs(txs),
		MessageType:     txsubmission.MessageTypeReplyTxs,
		MsgFromCborFunc: txsubmission.NewMsgFromCbor,
	}
}

// LocalTxMonitorAcquire builds an acquire input entry.
func LocalTxMonitorAcquire() ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		Message:         localtxmonitor.NewMsgAcquire(),
		MessageType:     localtxmonitor.MessageTypeAcquire,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
}

// LocalTxMonitorNextTx builds a next-transaction input entry.
func LocalTxMonitorNextTx() ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		Message:         localtxmonitor.NewMsgNextTx(),
		MessageType:     localtxmonitor.MessageTypeNextTx,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
}

// LocalTxMonitorHasTx builds a has-transaction input entry.
func LocalTxMonitorHasTx(txID []byte) ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		Message:         localtxmonitor.NewMsgHasTx(txID),
		MessageType:     localtxmonitor.MessageTypeHasTx,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
}

// LocalTxMonitorGetSizes builds a get-sizes input entry.
func LocalTxMonitorGetSizes() ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		Message:         localtxmonitor.NewMsgGetSizes(),
		MessageType:     localtxmonitor.MessageTypeGetSizes,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
}

// LocalTxMonitorRelease builds a release input entry.
func LocalTxMonitorRelease() ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      localtxmonitor.ProtocolId,
		Message:         localtxmonitor.NewMsgRelease(),
		MessageType:     localtxmonitor.MessageTypeRelease,
		MsgFromCborFunc: localtxmonitor.NewMsgFromCbor,
	}
}

// LocalStateQueryAcquire builds an acquire input entry.
func LocalStateQueryAcquire(point pcommon.Point) ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      localstatequery.ProtocolId,
		Message:         localstatequery.NewMsgAcquire(point),
		MessageType:     localstatequery.MessageTypeAcquire,
		MsgFromCborFunc: localstatequery.NewMsgFromCbor,
	}
}

// LocalStateQueryReAcquire builds a re-acquire input entry.
func LocalStateQueryReAcquire(point pcommon.Point) ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      localstatequery.ProtocolId,
		Message:         localstatequery.NewMsgReAcquire(point),
		MessageType:     localstatequery.MessageTypeReacquire,
		MsgFromCborFunc: localstatequery.NewMsgFromCbor,
	}
}

// LocalStateQueryQuery builds a query input entry. The expected message is
// normalized through CBOR because localstatequery.QueryWrapper keeps its own
// stored CBOR and decodes the payload into a typed query. A message built
// directly from NewMsgQuery carries neither, so it never compares equal to a
// message decoded from the wire.
func LocalStateQueryQuery(query any) (ConversationEntryInput, error) {
	encoded, err := cbor.Encode(localstatequery.NewMsgQuery(query))
	if err != nil {
		return ConversationEntryInput{}, err
	}
	message, err := localstatequery.NewMsgFromCbor(
		localstatequery.MessageTypeQuery,
		encoded,
	)
	if err != nil {
		return ConversationEntryInput{}, err
	}
	if message == nil {
		return ConversationEntryInput{}, errors.New(
			"local state query message type was not recognized",
		)
	}
	message.SetCbor(nil)
	return ConversationEntryInput{
		ProtocolId:      localstatequery.ProtocolId,
		Message:         message,
		MessageType:     localstatequery.MessageTypeQuery,
		MsgFromCborFunc: localstatequery.NewMsgFromCbor,
	}, nil
}

// LocalStateQueryRelease builds a release input entry.
func LocalStateQueryRelease() ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      localstatequery.ProtocolId,
		Message:         localstatequery.NewMsgRelease(),
		MessageType:     localstatequery.MessageTypeRelease,
		MsgFromCborFunc: localstatequery.NewMsgFromCbor,
	}
}
