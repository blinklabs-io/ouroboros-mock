// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package ouroboros_mock

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
)

func TestProtocolBuildersEncodeMessages(t *testing.T) {
	point := NewPoint(42, make([]byte, common.Blake2b256Size))
	tip := NewTip(point, 7)
	blocks, err := fixtures.GenerateConwayChain(1, common.Blake2b256{}, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	header := blocks[0].Header().Cbor()

	ntn, err := ChainSyncRollForwardNtN(conway.EraIdConway, 0, header, tip)
	if err != nil {
		t.Fatal(err)
	}
	ntc, err := ChainSyncRollForwardNtC(7, []byte{0x80}, tip)
	if err != nil {
		t.Fatal(err)
	}
	entries := []ConversationEntry{
		ChainSyncRequestNext(false),
		ntn,
		ntc,
		ChainSyncRollBackward(point, tip),
		ChainSyncFindIntersect([]pcommon.Point{point}),
		BlockFetchRequestRange(point, point),
		BlockFetchNoBlocks(),
		BlockFetchBlock([]byte{0x80}),
		TxSubmissionRequestTxIds(true, 0, 1),
		TxSubmissionRequestTxs(nil),
		TxSubmissionReplyTxIds(nil),
		TxSubmissionReplyTxs(nil),
		LocalTxMonitorAcquire(),
		LocalTxMonitorNextTx(),
		LocalTxMonitorHasTx(make([]byte, common.Blake2b256Size)),
		LocalTxMonitorGetSizes(),
		LocalTxMonitorRelease(),
		LocalStateQueryAcquire(point),
		LocalStateQueryReAcquire(point),
		LocalStateQueryQuery(nil),
		LocalStateQueryRelease(),
	}
	for i, entry := range entries {
		switch entry := entry.(type) {
		case ConversationEntryInput:
			if entry.Message == nil {
				t.Fatalf("entry %d has no message", i)
			}
			assertMessageEncodes(t, entry.Message)
		case ConversationEntryOutput:
			if len(entry.Messages) == 0 {
				t.Fatalf("entry %d has no output messages", i)
			}
			for _, message := range entry.Messages {
				assertMessageEncodes(t, message)
			}
		}
	}
}

func assertMessageEncodes(t *testing.T, message protocol.Message) {
	t.Helper()
	encoded, err := cbor.Encode(message)
	if err != nil {
		t.Fatalf("encode %T: %v", message, err)
	}
	if len(encoded) == 0 {
		t.Fatalf("encode %T returned empty CBOR", message)
	}
}

func TestChainSyncScenarioUsesCanonicalHeaderCBOR(t *testing.T) {
	blocks, err := fixtures.GenerateConwayChain(1, common.Blake2b256{}, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	conversation, err := ChainSyncScenario(
		conway.EraIdConway,
		0,
		[][]byte{blocks[0].Header().Cbor()},
		NewTip(NewPoint(blocks[0].SlotNumber(), blocks[0].Hash().Bytes()), blocks[0].BlockNumber()),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(conversation) != 3 {
		t.Fatalf("conversation length = %d, want 3", len(conversation))
	}
	entry := conversation[1].(ConversationEntryOutput)
	message := entry.Messages[0].(*chainsync.MsgRollForwardNtN)
	if len(message.WrappedHeader.HeaderCbor()) == 0 {
		t.Fatal("roll-forward header did not retain CBOR")
	}
}
