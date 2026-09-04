package main

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

func TestTransactionUsableAtUsesHalfOpenValidityInterval(t *testing.T) {
	tx := &conway.ConwayTransaction{TxIsValid: true}
	tx.Body.SetValidityIntervalUpperBound(2)
	raw, err := cbor.Encode(tx)
	if err != nil {
		t.Fatalf("encode transaction: %v", err)
	}

	usable, err := transactionUsableAt(raw, 1)
	if err != nil {
		t.Fatalf("inspect transaction at slot 1: %v", err)
	}
	if !usable {
		t.Fatal("transaction should be usable before its upper bound")
	}

	usable, err = transactionUsableAt(raw, 2)
	if err != nil {
		t.Fatalf("inspect transaction at slot 2: %v", err)
	}
	if usable {
		t.Fatal("transaction should be unusable at its upper bound")
	}
}

func TestTransactionUsableAtReportsDecodeErrors(t *testing.T) {
	if _, err := transactionUsableAt([]byte{0xff}, 2); err == nil {
		t.Fatal("expected malformed transaction error")
	}
}
