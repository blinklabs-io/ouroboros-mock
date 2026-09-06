package event

import (
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"testing"
	"time"
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
	seq := RollbackScenarioEvents(nil, ocommon.Point{Slot: 42, Hash: []byte{1, 2, 3}}, 764824073)
	if len(seq.Events) != 1 || seq.Events[0].Type != TypeRollback {
		t.Fatalf("unexpected sequence: %#v", seq.Events)
	}
	payload := seq.Events[0].Payload.(RollbackEvent)
	if payload.SlotNumber != 42 || payload.BlockHash != "010203" {
		t.Fatalf("unexpected rollback: %#v", payload)
	}
}
