// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

// Package event provides dependency-free event fixtures for pipeline tests.
// Its exported shapes mirror the event contract consumed by adder without
// importing adder (which already depends on this module).
package event

import (
	"encoding/hex"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger"
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
)

const (
	TypeBlock       = "input.block"
	TypeTransaction = "input.transaction"
	TypeRollback    = "input.rollback"
	TypeGovernance  = "input.governance"
)

type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Context   any       `json:"context,omitempty"`
	Payload   any       `json:"payload"`
	Type      string    `json:"type"`
}

func NewEvent() Event { return Event{Timestamp: time.Unix(0, 0).UTC()} }

func newEvent(kind string, context, payload any) Event {
	e := NewEvent()
	e.Type, e.Context, e.Payload = kind, context, payload
	return e
}

type BlockContext struct {
	Era          string `json:"era"`
	BlockNumber  uint64 `json:"blockNumber"`
	SlotNumber   uint64 `json:"slotNumber"`
	NetworkMagic uint32 `json:"networkMagic"`
}

type BlockEvent struct {
	Block            ledger.Block `json:"-"`
	BlockHash        string       `json:"blockHash"`
	BlockCbor        []byte       `json:"blockCbor,omitempty"`
	BlockBodySize    uint64       `json:"blockBodySize"`
	TransactionCount uint64       `json:"transactionCount"`
}

func NewBlockEvent() Event { return newEvent(TypeBlock, BlockContext{}, BlockEvent{}) }

func BlockEventFromBlock(block ledger.Block, networkMagic uint32) Event {
	ctx := BlockContext{Era: block.Era().Name, BlockNumber: block.BlockNumber(), SlotNumber: block.SlotNumber(), NetworkMagic: networkMagic}
	payload := BlockEvent{Block: block, BlockHash: block.Hash().String(), BlockBodySize: block.BlockBodySize(), TransactionCount: uint64(len(block.Transactions()))}
	return newEvent(TypeBlock, ctx, payload)
}

type TransactionContext struct {
	TransactionHash string `json:"transactionHash"`
	BlockNumber     uint64 `json:"blockNumber"`
	SlotNumber      uint64 `json:"slotNumber"`
	TransactionIdx  uint32 `json:"transactionIdx"`
	NetworkMagic    uint32 `json:"networkMagic"`
}

type TransactionEvent struct {
	Transaction     ledger.Transaction         `json:"-"`
	BlockHash       string                     `json:"blockHash"`
	Inputs          []ledger.TransactionInput  `json:"inputs"`
	Outputs         []ledger.TransactionOutput `json:"outputs"`
	Certificates    []ledger.Certificate       `json:"certificates,omitempty"`
	TransactionCbor []byte                     `json:"transactionCbor,omitempty"`
	Fee             uint64                     `json:"fee"`
	TTL             uint64                     `json:"ttl,omitempty"`
}

func NewTransactionEvent() Event {
	return newEvent(TypeTransaction, TransactionContext{}, TransactionEvent{})
}

func TransactionEventFromTx(tx ledger.Transaction, blockNumber, slot uint64, txIdx uint32, networkMagic uint32) Event {
	ctx := TransactionContext{TransactionHash: tx.Hash().String(), BlockNumber: blockNumber, SlotNumber: slot, TransactionIdx: txIdx, NetworkMagic: networkMagic}
	payload := TransactionEvent{Transaction: tx, Inputs: tx.Inputs(), Outputs: tx.Outputs(), Fee: tx.Fee().Uint64(), TTL: tx.TTL()}
	return newEvent(TypeTransaction, ctx, payload)
}

type GovernanceEvent struct {
	Transaction ledger.Transaction `json:"-"`
	BlockHash   string             `json:"blockHash"`
}

func NewGovernanceEvent() Event { return newEvent(TypeGovernance, nil, GovernanceEvent{}) }

type RollbackEvent struct {
	BlockHash  string `json:"blockHash"`
	SlotNumber uint64 `json:"slotNumber"`
}

func NewRollbackEvent() Event { return newEvent(TypeRollback, nil, RollbackEvent{}) }
func RollbackEventAtSlot(slot uint64, blockHash []byte) Event {
	return newEvent(TypeRollback, nil, RollbackEvent{BlockHash: hex.EncodeToString(blockHash), SlotNumber: slot})
}

type EventSequence struct{ Events []Event }

func NewEventSequence() EventSequence { return EventSequence{} }
func (s *EventSequence) Add(events ...Event) *EventSequence {
	s.Events = append(s.Events, events...)
	return s
}

func EventSequenceFromBlocks(blocks []ledger.Block, networkMagic uint32) EventSequence {
	var seq EventSequence
	for _, block := range blocks {
		seq.Add(BlockEventFromBlock(block, networkMagic))
		for idx, tx := range block.Transactions() {
			seq.Add(TransactionEventFromTx(tx, block.BlockNumber(), block.SlotNumber(), uint32(idx), networkMagic))
		}
	}
	return seq
}

func RollbackScenarioEvents(forwardBlocks []ledger.Block, rollbackTo ocommon.Point, networkMagic uint32) EventSequence {
	seq := EventSequenceFromBlocks(forwardBlocks, networkMagic)
	return *seq.Add(RollbackEventAtSlot(rollbackTo.Slot, rollbackTo.Hash))
}
