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

// Command gen-rollback-vector splices a rollback event into an existing
// conformance test vector to produce a synthetic vector that exercises
// the harness's rollback machinery end-to-end.
//
// The output vector preserves the base's config, initial_state, and
// final_state, and rewrites events to the sequence:
//
//	[tx1@S1, tx2@S2, rollback@(S1+gap), tx2_again@S2]
//
// tx1 and tx2 are the first two successful transaction events found in
// the base vector. The rollback target is chosen so that tx1 is retained
// and tx2 is dropped from the journal. Replaying tx2 after the rollback
// can only succeed if the rollback restored the UTxOs that tx2 consumed,
// so the synthetic vector verifies state reversion through the full
// file-decode + harness execution path.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/ouroboros-mock/conformance"
)

func main() {
	base := flag.String("base", "", "path to base vector file")
	out := flag.String("out", "", "path to write synthetic vector")
	title := flag.String(
		"title",
		"",
		"override title for the synthetic vector "+
			"(default: derived from output path)",
	)
	flag.Parse()
	if *base == "" || *out == "" {
		fmt.Fprintln(
			os.Stderr,
			"usage: gen-rollback-vector -base <vector> -out <path> "+
				"[-title <name>]",
		)
		os.Exit(2)
	}
	if err := generate(*base, *out, *title); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func generate(basePath, outPath, titleOverride string) error {
	baseData, err := os.ReadFile(basePath)
	if err != nil {
		return fmt.Errorf("read base: %w", err)
	}

	var items []cbor.RawMessage
	if _, err := cbor.Decode(baseData, &items); err != nil {
		return fmt.Errorf("decode base CBOR: %w", err)
	}
	if len(items) < 5 {
		return fmt.Errorf(
			"unexpected base structure: got %d top-level elements, want 5",
			len(items),
		)
	}

	vec, err := conformance.DecodeTestVector(basePath)
	if err != nil {
		return fmt.Errorf("decode base vector: %w", err)
	}

	tx1, tx2, err := selectSplicePair(vec.Events)
	if err != nil {
		return err
	}

	// Pick a rollback target strictly between the two tx slots so tx1 is
	// retained and tx2 is dropped from the journal.
	target := tx1.Slot + (tx2.Slot-tx1.Slot)/2
	if target <= tx1.Slot {
		target = tx1.Slot + 1
	}
	if target >= tx2.Slot {
		return fmt.Errorf(
			"cannot pick rollback target strictly between slots %d and %d",
			tx1.Slot,
			tx2.Slot,
		)
	}

	events := []any{
		txEvent(tx1.TxBytes, true, tx1.Slot),
		txEvent(tx2.TxBytes, true, tx2.Slot),
		rollbackEvent(target),
		// Same tx, same slot — verifies that rollback restored the
		// inputs tx2 consumed the first time around.
		txEvent(tx2.TxBytes, true, tx2.Slot),
	}
	eventsCBOR, err := cbor.Encode(events)
	if err != nil {
		return fmt.Errorf("encode events: %w", err)
	}

	title := titleOverride
	if title == "" {
		title = deriveTitle(outPath)
	}

	vector := []any{
		items[0], // config
		items[1], // initial_state
		items[2], // final_state
		cbor.RawMessage(eventsCBOR),
		title,
	}
	outBytes, err := cbor.Encode(vector)
	if err != nil {
		return fmt.Errorf("encode vector: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(outPath, outBytes, 0o600); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	fmt.Fprintf(
		os.Stderr,
		"wrote %s\n  base: %s\n  events: tx1@%d, tx2@%d, rollback@%d, tx2@%d\n",
		outPath, basePath, tx1.Slot, tx2.Slot, target, tx2.Slot,
	)
	return nil
}

// selectSplicePair returns the first two successful transaction events
// whose slots are strictly increasing and differ by at least 2 (so a
// rollback target can fall strictly between them).
func selectSplicePair(
	events []conformance.VectorEvent,
) (conformance.VectorEvent, conformance.VectorEvent, error) {
	var first conformance.VectorEvent
	haveFirst := false
	for _, ev := range events {
		if ev.Type != conformance.EventTypeTransaction || !ev.Success {
			continue
		}
		if !haveFirst {
			first = ev
			haveFirst = true
			continue
		}
		if ev.Slot < first.Slot+2 {
			// Replace the candidate first if the gap is too tight to
			// place a rollback target strictly between them. This lets
			// us tolerate vectors that submit multiple txs at the same
			// slot before the next block boundary.
			first = ev
			continue
		}
		return first, ev, nil
	}
	return conformance.VectorEvent{}, conformance.VectorEvent{}, errors.New(
		"base vector has no pair of successful txs separated by >=2 slots",
	)
}

func txEvent(txBytes []byte, success bool, slot uint64) []any {
	return []any{
		uint64(conformance.EventTypeTransaction),
		txBytes,
		success,
		slot,
	}
}

func rollbackEvent(targetSlot uint64) []any {
	return []any{uint64(conformance.EventTypeRollback), targetSlot}
}

func deriveTitle(outPath string) string {
	stem := filepath.Base(outPath)
	if ext := filepath.Ext(stem); ext != "" {
		stem = stem[:len(stem)-len(ext)]
	}
	return "synthetic/rollback/" + stem
}
