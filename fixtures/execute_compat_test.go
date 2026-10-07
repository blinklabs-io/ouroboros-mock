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

package fixtures

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
)

func TestByronConsensusGenTxFixturesAreUnpaired(t *testing.T) {
	harness := NewHarness(HarnessConfig{})
	allFixtures, err := harness.Collect()
	if err != nil {
		t.Fatalf("failed to collect fixtures: %v", err)
	}
	fixtureMap := make(map[string]Fixture, len(allFixtures))
	for _, fixture := range allFixtures {
		fixtureMap[fixture.RelPath] = fixture
	}
	byronTxPath := consensusV2FixtureRoot + "GenTx_Byron"
	byronTxIDPath := consensusV2FixtureRoot + "GenTxId_Byron"
	byronTx, ok := fixtureMap[byronTxPath]
	if !ok {
		t.Fatalf("missing fixture %s", byronTxPath)
	}
	byronTxID, ok := fixtureMap[byronTxIDPath]
	if !ok {
		t.Fatalf("missing fixture %s", byronTxIDPath)
	}
	if _, ok := relatedFixture(fixtureMap, byronTx, KindTransactionID); ok {
		t.Fatal("Byron GenTx fixture must not pair with the independent ID golden")
	}
	if _, ok := relatedFixture(fixtureMap, byronTxID, KindTransaction); ok {
		t.Fatal("Byron GenTxId fixture must not pair with the independent transaction golden")
	}

	shelleyTxPath := consensusV2FixtureRoot + "GenTx_Shelley"
	shelleyTx := fixtureMap[shelleyTxPath]
	if _, ok := relatedFixture(fixtureMap, shelleyTx, KindTransactionID); !ok {
		t.Fatal("expected Shelley GenTx fixture to pair with its transaction ID")
	}
}

func TestStrictDecodePlaceholderMatcher(t *testing.T) {
	exactStrictDecodeErrors := []string{
		"invalid blake2b-256 hash: expected 32 bytes, got 2",
		"invalid blake2b-256 hash length: expected 32 bytes, got 2",
	}

	allowlisted := Fixture{
		RelPath: "cardano-ledger/eras/alonzo/test-suite/golden/block.cbor",
	}
	if !isStrictDecodePlaceholderFixture(allowlisted) {
		t.Fatal("expected the preserved Alonzo block to be allowlisted")
	}
	if isStrictDecodePlaceholderFixture(Fixture{RelPath: "unrelated.cbor"}) {
		t.Fatal("unexpected allowlist match for unrelated fixture")
	}

	for _, errorText := range exactStrictDecodeErrors {
		exact := errors.New(errorText)
		if !isStrictDecodePlaceholderError(exact) {
			t.Fatalf(
				"expected exact strict decode error %q to match",
				errorText,
			)
		}
		if !isStrictDecodePlaceholderError(
			fmt.Errorf("decode fixture: %w", exact),
		) {
			t.Fatalf(
				"expected wrapped strict decode error %q to match",
				errorText,
			)
		}
	}
	for _, err := range []error{
		errors.New("invalid blake2b-256 hash: expected 32 bytes, got 3"),
		errors.New("invalid blake2b-256 hash length: expected 32 bytes, got 3"),
		errors.New("unrelated error: " + exactStrictDecodeErrors[0]),
		errors.New("unrelated error: " + exactStrictDecodeErrors[1]),
	} {
		if isStrictDecodePlaceholderError(err) {
			t.Fatalf("unexpected strict decode match for %q", err)
		}
	}
}

func TestStrictDecodePlaceholderExecutionsAreAllOrNone(t *testing.T) {
	harness := NewHarness(HarnessConfig{})
	results, err := harness.RunAllExecutionsWithResults()
	if err != nil {
		t.Fatalf("run fixture executions: %s", err)
	}
	var rejectionCount int
	seen := make(map[string]struct{}, len(strictDecodePlaceholderFixtures))
	for _, result := range results {
		if _, allowlisted := strictDecodePlaceholderFixtures[result.Fixture.RelPath]; allowlisted {
			seen[result.Fixture.RelPath] = struct{}{}
			if result.Error != nil {
				t.Fatalf(
					"allowlisted fixture %s failed with an unexpected error: %v",
					result.Fixture.RelPath,
					result.Error,
				)
			}
		}
		if result.ExpectedStrictDecodeRejection {
			rejectionCount++
			if !isStrictDecodePlaceholderFixture(result.Fixture) {
				t.Fatalf(
					"unexpected strict-decode rejection for %s",
					result.Fixture.RelPath,
				)
			}
		}
	}
	if len(seen) != len(strictDecodePlaceholderFixtures) {
		t.Fatalf(
			"strict-decode allowlist covered %d fixtures, want %d",
			len(seen),
			len(strictDecodePlaceholderFixtures),
		)
	}
	if rejectionCount != 0 &&
		rejectionCount != len(strictDecodePlaceholderFixtures) {
		t.Fatalf(
			"strict-decode rejection count = %d, want 0 or %d",
			rejectionCount,
			len(strictDecodePlaceholderFixtures),
		)
	}
}

func TestDijkstraConsensusHeaderChecksRelatedBlock(t *testing.T) {
	harness := NewHarness(HarnessConfig{})
	allFixtures, err := harness.Collect()
	if err != nil {
		t.Fatalf("failed to collect fixtures: %v", err)
	}
	var header Fixture
	for _, fixture := range allFixtures {
		if fixture.RelPath == consensusV2FixtureRoot+"Header_Dijkstra" {
			header = fixture
		}
	}
	if header.RelPath == "" {
		t.Fatal("missing Dijkstra consensus header fixture")
	}

	block, err := NewDijkstraBlockBuilder().Build()
	if err != nil {
		t.Fatalf("build Dijkstra block: %v", err)
	}
	blockType, err := ledgerBlockTypeForEra("dijkstra")
	if err != nil {
		t.Fatalf("Dijkstra block type: %v", err)
	}
	wrapper, err := cbor.Encode([]any{blockType, cbor.RawMessage(block.Cbor())})
	if err != nil {
		t.Fatalf("encode block wrapper: %v", err)
	}
	data, err := cbor.Encode(cbor.Tag{Number: 24, Content: wrapper})
	if err != nil {
		t.Fatalf("encode tag-24 block: %v", err)
	}
	relPath := consensusV2FixtureRoot + "Block_Dijkstra"
	blockPath := filepath.Join(t.TempDir(), filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(blockPath), 0o755); err != nil {
		t.Fatalf("create block directory: %v", err)
	}
	if err := os.WriteFile(blockPath, data, 0o600); err != nil {
		t.Fatalf("write block fixture: %v", err)
	}
	blockFixture := Fixture{
		Path:    blockPath,
		RelPath: relPath,
		Repo:    RepoOuroborosConsensus,
		Kind:    KindBlock,
		Format:  header.Format,
		Era:     "dijkstra",
		Name:    "Block_Dijkstra",
	}
	if _, err := blockFixture.DecodeLedgerBlock(); err != nil {
		t.Fatalf("synthetic Dijkstra block must decode: %v", err)
	}

	_, err = executeHeaderFixture(header, map[string]Fixture{
		header.RelPath: header,
		relPath:        blockFixture,
	})
	if err == nil || !strings.Contains(err.Error(), "header/block hash mismatch") {
		t.Fatalf("expected header/block hash mismatch, got %v", err)
	}
}
