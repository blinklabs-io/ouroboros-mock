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

package ledger

import (
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
)

// ValidityIntervalFixture describes whether each transaction validity bound
// is present and, when present, its slot value. A non-nil pointer to zero
// represents an explicitly encoded zero bound.
type ValidityIntervalFixture struct {
	Name      string
	StartSlot *uint64
	EndSlot   *uint64
}

// ValidityIntervalFixtures returns the complete set of transaction validity
// bound-presence combinations, including explicit zero bounds.
func ValidityIntervalFixtures() []ValidityIntervalFixture {
	slot := func(value uint64) *uint64 {
		return &value
	}
	return []ValidityIntervalFixture{
		{Name: "unbounded"},
		{Name: "upper only", EndSlot: slot(10)},
		{Name: "lower only", StartSlot: slot(5)},
		{
			Name:      "both bounds",
			StartSlot: slot(5),
			EndSlot:   slot(10),
		},
		{
			Name:      "explicit zero lower",
			StartSlot: slot(0),
			EndSlot:   slot(10),
		},
		{Name: "explicit zero upper", EndSlot: slot(0)},
		{
			Name:      "both explicit zero",
			StartSlot: slot(0),
			EndSlot:   slot(0),
		},
	}
}

// AlonzoTransaction returns the fixture decoded as an Alonzo transaction.
func (f ValidityIntervalFixture) AlonzoTransaction() (
	*alonzo.AlonzoTransaction,
	error,
) {
	txCbor, err := f.transactionCbor()
	if err != nil {
		return nil, err
	}
	tx, err := alonzo.NewAlonzoTransactionFromCbor(txCbor)
	if err != nil {
		return nil, fmt.Errorf("decode Alonzo transaction: %w", err)
	}
	return tx, nil
}

// BabbageTransaction returns the fixture decoded as a Babbage transaction.
func (f ValidityIntervalFixture) BabbageTransaction() (
	*babbage.BabbageTransaction,
	error,
) {
	txCbor, err := f.transactionCbor()
	if err != nil {
		return nil, err
	}
	tx, err := babbage.NewBabbageTransactionFromCbor(txCbor)
	if err != nil {
		return nil, fmt.Errorf("decode Babbage transaction: %w", err)
	}
	return tx, nil
}

func (f ValidityIntervalFixture) transactionCbor() ([]byte, error) {
	body := make(map[uint]uint64, 2)
	if f.StartSlot != nil {
		body[8] = *f.StartSlot
	}
	if f.EndSlot != nil {
		body[3] = *f.EndSlot
	}
	bodyCbor, err := cbor.Encode(body)
	if err != nil {
		return nil, fmt.Errorf("encode transaction body: %w", err)
	}
	witnessCbor, err := cbor.Encode(map[uint]any{})
	if err != nil {
		return nil, fmt.Errorf("encode transaction witnesses: %w", err)
	}
	txCbor, err := cbor.Encode([]any{
		cbor.RawMessage(bodyCbor),
		cbor.RawMessage(witnessCbor),
		true,
		nil,
	})
	if err != nil {
		return nil, fmt.Errorf("encode transaction: %w", err)
	}
	return txCbor, nil
}
