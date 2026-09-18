package event

import (
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
)

func TestBaseBuildersUseStableEventShapes(t *testing.T) {
	for _, test := range []struct {
		name, kind string
		build      func() Event
	}{
		{"block", TypeBlock, NewBlockEvent},
		{"transaction", TypeTransaction, NewTransactionEvent},
		{"governance", TypeGovernance, NewGovernanceEvent},
		{"rollback", TypeRollback, NewRollbackEvent},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := test.build()
			if e.Type != test.kind {
				t.Fatalf("event type = %q, want %q", e.Type, test.kind)
			}
			if !e.Timestamp.Equal(time.Unix(0, 0).UTC()) {
				t.Fatalf("timestamp = %v", e.Timestamp)
			}
			if e.Payload == nil {
				t.Fatal("payload is nil")
			}
		})
	}
}

func TestRollbackScenarioPreservesOrder(t *testing.T) {
	blocks, err := fixtures.GenerateConwayChainWithTransactions(
		1,
		common.Blake2b256{},
		10,
		10,
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	seq := RollbackScenarioEvents(
		blocks,
		ocommon.Point{Slot: 42, Hash: []byte{1, 2, 3}},
		764824073,
	)
	if len(seq.Events) != 3 || seq.Events[0].Type != TypeBlock ||
		seq.Events[1].Type != TypeTransaction ||
		seq.Events[2].Type != TypeRollback {
		t.Fatalf("unexpected sequence: %#v", seq.Events)
	}
	blockPayload := seq.Events[0].Payload.(BlockEvent)
	if len(blockPayload.BlockCbor) == 0 {
		t.Fatal("block CBOR is empty")
	}
	txPayload := seq.Events[1].Payload.(TransactionEvent)
	if txPayload.BlockHash != blocks[0].Hash().String() ||
		len(txPayload.TransactionCbor) == 0 {
		t.Fatalf("unexpected transaction metadata: %#v", txPayload)
	}
	payload := seq.Events[2].Payload.(RollbackEvent)
	if payload.SlotNumber != 42 || payload.BlockHash != "010203" {
		t.Fatalf("unexpected rollback: %#v", payload)
	}
}
