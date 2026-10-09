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
	"net"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/gouroboros/protocol/keepalive"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/gouroboros/protocol/localtxmonitor"
	"github.com/blinklabs-io/gouroboros/protocol/peersharing"
	"github.com/blinklabs-io/gouroboros/protocol/txsubmission"
)

func protocolInput(
	id uint16,
	message protocol.Message,
	decoder protocol.MessageFromCborFunc,
) ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      id,
		Message:         message,
		MessageType:     uint(message.Type()),
		MsgFromCborFunc: decoder,
	}
}

func protocolOutput(id uint16, messages ...protocol.Message) ConversationEntryOutput {
	return ConversationEntryOutput{ProtocolId: id, IsResponse: true, Messages: messages}
}

// ChainSyncAwaitReply builds an await-reply response for the negotiated mode.
func ChainSyncAwaitReply(nodeToClient bool) ConversationEntryOutput {
	id, _ := chainSyncMode(nodeToClient)
	return protocolOutput(id, chainsync.NewMsgAwaitReply())
}

// ChainSyncIntersectFound builds a successful intersection response.
func ChainSyncIntersectFound(
	nodeToClient bool,
	point pcommon.Point,
	tip chainsync.Tip,
) ConversationEntryOutput {
	id, _ := chainSyncMode(nodeToClient)
	return protocolOutput(
		id,
		chainsync.NewMsgIntersectFound(clonePoint(point), cloneTip(tip)),
	)
}

// ChainSyncIntersectNotFound builds an unsuccessful intersection response.
func ChainSyncIntersectNotFound(nodeToClient bool, tip chainsync.Tip) ConversationEntryOutput {
	id, _ := chainSyncMode(nodeToClient)
	return protocolOutput(id, chainsync.NewMsgIntersectNotFound(cloneTip(tip)))
}

// ChainSyncDone builds a client termination entry for the negotiated mode.
func ChainSyncDone(nodeToClient bool) ConversationEntryInput {
	id, decoder := chainSyncMode(nodeToClient)
	return protocolInput(id, chainsync.NewMsgDone(), decoder)
}

// BlockFetchStartBatch builds the response that begins a block batch.
func BlockFetchStartBatch() ConversationEntryOutput {
	return protocolOutput(blockfetch.ProtocolId, blockfetch.NewMsgStartBatch())
}

// BlockFetchBlockResponse wraps raw ledger block CBOR with its wire block type and builds one response within an open batch.
func BlockFetchBlockResponse(blockType uint, rawBlock []byte) (ConversationEntryOutput, error) {
	if len(rawBlock) == 0 {
		return ConversationEntryOutput{}, errors.New("raw block CBOR is required")
	}
	wrapped, err := cbor.Encode(
		blockfetch.WrappedBlock{Type: blockType, RawBlock: cbor.RawMessage(rawBlock)},
	)
	if err != nil {
		return ConversationEntryOutput{}, err
	}
	return protocolOutput(blockfetch.ProtocolId, blockfetch.NewMsgBlock(wrapped)), nil
}

// BlockFetchBatchDone builds the response that ends a block batch.
func BlockFetchBatchDone() ConversationEntryOutput {
	return protocolOutput(blockfetch.ProtocolId, blockfetch.NewMsgBatchDone())
}

// BlockFetchClientDone builds a client termination entry.
func BlockFetchClientDone() ConversationEntryInput {
	return protocolInput(
		blockfetch.ProtocolId,
		blockfetch.NewMsgClientDone(),
		blockfetch.NewMsgFromCbor,
	)
}

// TxSubmissionDone builds a client termination entry answering a blocking ID request.
func TxSubmissionDone() ConversationEntryInput {
	return protocolInput(
		txsubmission.ProtocolId,
		txsubmission.NewMsgDone(),
		txsubmission.NewMsgFromCbor,
	)
}

// LocalTxMonitorAcquired builds a response identifying the acquired mempool snapshot.
func LocalTxMonitorAcquired(slot uint64) ConversationEntryOutput {
	return protocolOutput(localtxmonitor.ProtocolId, localtxmonitor.NewMsgAcquired(slot))
}

// LocalTxMonitorReplyNextTx builds a next-transaction response; a nil transaction means none remain.
func LocalTxMonitorReplyNextTx(era uint8, tx []byte) ConversationEntryOutput {
	return protocolOutput(
		localtxmonitor.ProtocolId,
		localtxmonitor.NewMsgReplyNextTx(era, cloneBytes(tx)),
	)
}

// LocalTxMonitorReplyHasTx builds a transaction-presence response.
func LocalTxMonitorReplyHasTx(present bool) ConversationEntryOutput {
	return protocolOutput(localtxmonitor.ProtocolId, localtxmonitor.NewMsgReplyHasTx(present))
}

// LocalTxMonitorReplyGetSizes builds a mempool capacity, size, and transaction-count response.
func LocalTxMonitorReplyGetSizes(capacity, size, count uint32) ConversationEntryOutput {
	return protocolOutput(
		localtxmonitor.ProtocolId,
		localtxmonitor.NewMsgReplyGetSizes(capacity, size, count),
	)
}

// LocalTxMonitorDone builds a client termination entry after releasing its snapshot.
func LocalTxMonitorDone() ConversationEntryInput {
	return protocolInput(
		localtxmonitor.ProtocolId,
		localtxmonitor.NewMsgDone(),
		localtxmonitor.NewMsgFromCbor,
	)
}

// LocalStateQueryAcquireVolatileTip builds an acquire-volatile-tip request.
func LocalStateQueryAcquireVolatileTip() ConversationEntryInput {
	return protocolInput(
		localstatequery.ProtocolId,
		localstatequery.NewMsgAcquireVolatileTip(),
		localstatequery.NewMsgFromCbor,
	)
}

// LocalStateQueryAcquireImmutableTip builds an acquire-immutable-tip request.
func LocalStateQueryAcquireImmutableTip() ConversationEntryInput {
	return protocolInput(
		localstatequery.ProtocolId,
		localstatequery.NewMsgAcquireImmutableTip(),
		localstatequery.NewMsgFromCbor,
	)
}

// LocalStateQueryReAcquireVolatileTip builds a reacquire-volatile-tip request.
func LocalStateQueryReAcquireVolatileTip() ConversationEntryInput {
	return protocolInput(
		localstatequery.ProtocolId,
		localstatequery.NewMsgReAcquireVolatileTip(),
		localstatequery.NewMsgFromCbor,
	)
}

// LocalStateQueryReAcquireImmutableTip builds a reacquire-immutable-tip request.
func LocalStateQueryReAcquireImmutableTip() ConversationEntryInput {
	return protocolInput(
		localstatequery.ProtocolId,
		localstatequery.NewMsgReAcquireImmutableTip(),
		localstatequery.NewMsgFromCbor,
	)
}

// LocalStateQueryAcquired builds a successful acquisition response.
func LocalStateQueryAcquired() ConversationEntryOutput {
	return protocolOutput(localstatequery.ProtocolId, localstatequery.NewMsgAcquired())
}

// LocalStateQueryFailure builds an acquisition-failure response.
func LocalStateQueryFailure(reason uint8) ConversationEntryOutput {
	return protocolOutput(localstatequery.ProtocolId, localstatequery.NewMsgFailure(reason))
}

// LocalStateQueryDone builds a client termination entry in the idle state.
func LocalStateQueryDone() ConversationEntryInput {
	return protocolInput(
		localstatequery.ProtocolId,
		localstatequery.NewMsgDone(),
		localstatequery.NewMsgFromCbor,
	)
}

// PeerSharingRequest builds a request for peer addresses.
func PeerSharingRequest(amount uint8) ConversationEntryInput {
	return protocolInput(
		peersharing.ProtocolId,
		peersharing.NewMsgShareRequest(amount),
		peersharing.NewMsgFromCbor,
	)
}

// PeerSharingDone builds a client termination entry.
func PeerSharingDone() ConversationEntryInput {
	return protocolInput(
		peersharing.ProtocolId,
		peersharing.NewMsgDone(),
		peersharing.NewMsgFromCbor,
	)
}

// KeepAliveRequest builds a request containing the supplied cookie.
func KeepAliveRequest(cookie uint16) ConversationEntryInput {
	return protocolInput(
		keepalive.ProtocolId,
		keepalive.NewMsgKeepAlive(cookie),
		keepalive.NewMsgFromCbor,
	)
}

// KeepAliveResponse builds a response containing the supplied cookie.
func KeepAliveResponse(cookie uint16) ConversationEntryOutput {
	return protocolOutput(keepalive.ProtocolId, keepalive.NewMsgKeepAliveResponse(cookie))
}

// KeepAliveDone builds a client termination entry.
func KeepAliveDone() ConversationEntryInput {
	return protocolInput(keepalive.ProtocolId, keepalive.NewMsgDone(), keepalive.NewMsgFromCbor)
}

// LocalStateQueryResult builds a result response from a CBOR-encodable query result.
func LocalStateQueryResult(result any) (ConversationEntryOutput, error) {
	data, err := cbor.Encode(result)
	if err != nil {
		return ConversationEntryOutput{}, err
	}
	return protocolOutput(localstatequery.ProtocolId, localstatequery.NewMsgResult(data)), nil
}

// PeerSharingPeers builds a peer-address response. A nil slice describes no peers.
func PeerSharingPeers(peers []peersharing.PeerAddress) ConversationEntryOutput {
	return protocolOutput(peersharing.ProtocolId, peersharing.NewMsgSharePeers(clonePeers(peers)))
}

func clonePeers(peers []peersharing.PeerAddress) []peersharing.PeerAddress {
	ret := make([]peersharing.PeerAddress, len(peers))
	for i, peer := range peers {
		ret[i] = peer
		ret[i].IP = append(net.IP(nil), peer.IP...)
	}
	return ret
}
