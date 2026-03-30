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

package fixtures_test

import (
	"bytes"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
)

func TestHarnessIntegration(t *testing.T) {
	harness := fixtures.NewHarness(fixtures.HarnessConfig{})
	// fixtures/upstream/manifest.txt is the source of truth for the curated corpus.
	expectedFixtures := expectedFixturesFromManifest(t, harness)
	expectedFixtureCount := len(expectedFixtures)
	expectedCounts := repoCounts(expectedFixtures)
	expectedConsensusBlockCount := countMatchingFixtures(expectedFixtures, fixtures.Filter{
		Repo:   fixtures.RepoOuroborosConsensus,
		Kind:   fixtures.KindBlock,
		Format: fixtures.FormatCBOR,
	})

	t.Run("CollectFixtures", func(t *testing.T) {
		collected, err := harness.Collect()
		if err != nil {
			t.Fatalf("Collect failed: %v", err)
		}
		if len(collected) != expectedFixtureCount {
			t.Fatalf(
				"expected %d fixtures from manifest, got %d",
				expectedFixtureCount,
				len(collected),
			)
		}

		repoCounts := map[fixtures.Repo]int{}
		for _, fixture := range collected {
			repoCounts[fixture.Repo]++
			if fixture.RelPath == "" {
				t.Fatalf("fixture %s has empty relative path", fixture.Path)
			}
			if fixture.SourcePath == "" {
				t.Fatalf("fixture %s has empty source path", fixture.Path)
			}
		}
		if !maps.Equal(repoCounts, expectedCounts) {
			t.Fatalf(
				"unexpected repo counts: got %v want %v",
				repoCounts,
				expectedCounts,
			)
		}
	})

	t.Run("ManifestMatchesOnDiskPaths", func(t *testing.T) {
		root := harness.FixturesRoot()
		manifest, err := fixtures.LoadManifest(root)
		if err != nil {
			t.Fatalf("LoadManifest failed: %v", err)
		}

		onDisk, err := walkFixtureFiles(root)
		if err != nil {
			t.Fatalf("walkFixtureFiles failed: %v", err)
		}

		if !slices.Equal(manifest, onDisk) {
			diffIndex, expectedPath, actualPath := firstSliceDifference(
				manifest,
				onDisk,
			)
			t.Fatalf(
				"manifest mismatch at index %d: expected %q, got %q; expected=%v actual=%v",
				diffIndex,
				expectedPath,
				actualPath,
				manifest,
				onDisk,
			)
		}
	})

	t.Run("RunMatching", func(t *testing.T) {
		var blockCount int
		harness.RunMatching(t, fixtures.Filter{
			Repo:   fixtures.RepoOuroborosConsensus,
			Kind:   fixtures.KindBlock,
			Format: fixtures.FormatCBOR,
		}, func(t *testing.T, fixture fixtures.Fixture) {
			blockCount++
		})
		if blockCount != expectedConsensusBlockCount {
			t.Fatalf(
				"expected %d consensus block fixtures from manifest, got %d",
				expectedConsensusBlockCount,
				blockCount,
			)
		}
	})

	t.Run("RunFixture", func(t *testing.T) {
		var seen string
		harness.RunFixture(
			t,
			"cardano-node/cardano-testnet/files/data/conway/genesis.conway.spec.json",
			func(t *testing.T, fixture fixtures.Fixture) {
				seen = fixture.RelPath
				if fixture.Kind != fixtures.KindGenesis {
					t.Fatalf("expected genesis fixture, got %s", fixture.Kind)
				}
			},
		)
		if seen == "" {
			t.Fatal("fixture callback did not run")
		}
	})

	t.Run("LookupFixture", func(t *testing.T) {
		fixture, err := harness.Fixture(
			"cardano-ledger/eras/alonzo/test-suite/golden/tx.cbor",
		)
		if err != nil {
			t.Fatalf("Fixture failed: %v", err)
		}
		if fixture.Kind != fixtures.KindTransaction {
			t.Fatalf("expected transaction fixture, got %s", fixture.Kind)
		}
	})

	t.Run("ExtractEmbeddedFixtures", func(t *testing.T) {
		fixturesRoot, err := fixtures.ExtractEmbeddedFixtures(t.TempDir())
		if err != nil {
			t.Fatalf("ExtractEmbeddedFixtures failed: %v", err)
		}

		extractedHarness := fixtures.NewHarness(fixtures.HarnessConfig{
			FixturesRoot: fixturesRoot,
		})
		collected, err := extractedHarness.Collect()
		if err != nil {
			t.Fatalf("Collect on extracted fixtures failed: %v", err)
		}
		if len(collected) != expectedFixtureCount {
			t.Fatalf(
				"expected %d extracted fixtures, got %d",
				expectedFixtureCount,
				len(collected),
			)
		}
	})
}

func TestFixtureExecutionHarness(t *testing.T) {
	harness := fixtures.NewHarness(fixtures.HarnessConfig{})
	expectedFixtureCount := len(expectedFixturesFromManifest(t, harness))

	t.Run("RunAllExecutions", func(t *testing.T) {
		harness.RunAllExecutions(t)
	})

	t.Run("RunAllExecutionsWithResults", func(t *testing.T) {
		results, err := harness.RunAllExecutionsWithResults()
		if err != nil {
			t.Fatalf("RunAllExecutionsWithResults failed: %v", err)
		}
		if len(results) != expectedFixtureCount {
			t.Fatalf(
				"expected %d execution results, got %d",
				expectedFixtureCount,
				len(results),
			)
		}

		var totalCases int
		for _, result := range results {
			totalCases += result.CaseCount
			if !result.Success {
				t.Fatalf(
					"%s execution failed: %v",
					result.Fixture.RelPath,
					result.Error,
				)
			}
			if result.CaseCount == 0 {
				t.Fatalf(
					"%s reported zero execution cases",
					result.Fixture.RelPath,
				)
			}
		}
		if totalCases <= len(results) {
			t.Fatalf(
				"expected multi-case execution coverage, got %d total cases for %d fixtures",
				totalCases,
				len(results),
			)
		}
	})

	t.Run("GenericDecodeAPI", func(t *testing.T) {
		blockFixture, err := harness.Fixture(
			"ouroboros-consensus/ouroboros-consensus-cardano/golden/cardano/CardanoNodeToNodeVersion2/Block_Conway",
		)
		if err != nil {
			t.Fatalf("Fixture failed: %v", err)
		}
		block, err := blockFixture.DecodeLedgerBlock()
		if err != nil {
			t.Fatalf("DecodeLedgerBlock failed: %v", err)
		}

		headerFixture, err := harness.Fixture(
			"ouroboros-consensus/ouroboros-consensus-cardano/golden/cardano/CardanoNodeToNodeVersion2/Header_Conway",
		)
		if err != nil {
			t.Fatalf("Fixture failed: %v", err)
		}
		header, err := headerFixture.DecodeLedgerHeader()
		if err != nil {
			t.Fatalf("DecodeLedgerHeader failed: %v", err)
		}
		if block.Hash() != header.Hash() {
			t.Fatalf(
				"block/header hash mismatch: %s != %s",
				block.Hash(),
				header.Hash(),
			)
		}

		txFixture, err := harness.Fixture(
			"ouroboros-consensus/ouroboros-consensus-cardano/golden/cardano/CardanoNodeToNodeVersion2/GenTx_Conway",
		)
		if err != nil {
			t.Fatalf("Fixture failed: %v", err)
		}
		tx, err := txFixture.DecodeLedgerTransaction()
		if err != nil {
			t.Fatalf("DecodeLedgerTransaction failed: %v", err)
		}

		txIDFixture, err := harness.Fixture(
			"ouroboros-consensus/ouroboros-consensus-cardano/golden/cardano/CardanoNodeToNodeVersion2/GenTxId_Conway",
		)
		if err != nil {
			t.Fatalf("Fixture failed: %v", err)
		}
		txIDBytes, err := txIDFixture.LedgerTransactionIDBytes()
		if err != nil {
			t.Fatalf("LedgerTransactionIDBytes failed: %v", err)
		}
		if got := tx.Hash().Bytes(); !bytes.Equal(got, txIDBytes) {
			t.Fatalf("transaction/txid mismatch")
		}

		canonicalTxFixture, err := harness.Fixture(
			"cardano-api/cardano-api/test/cardano-api-golden/files/tx-canonical.json",
		)
		if err != nil {
			t.Fatalf("Fixture failed: %v", err)
		}
		canonicalTx, err := canonicalTxFixture.DecodeLedgerTransaction()
		if err != nil {
			t.Fatalf(
				"DecodeLedgerTransaction on cardano-api canonical tx failed: %v",
				err,
			)
		}
		if canonicalTx.Type() != int(ledger.TxTypeConway) {
			t.Fatalf(
				"expected Conway transaction type, got %d",
				canonicalTx.Type(),
			)
		}

		baseParamsFixture, err := harness.Fixture(
			"cardano-ledger/eras/conway/impl/golden/pparams.json",
		)
		if err != nil {
			t.Fatalf("Fixture failed: %v", err)
		}
		baseParams, err := baseParamsFixture.DecodeProtocolParameters()
		if err != nil {
			t.Fatalf("DecodeProtocolParameters failed: %v", err)
		}
		conwayParams, ok := baseParams.(*conway.ConwayProtocolParameters)
		if !ok {
			t.Fatalf("expected Conway protocol parameters, got %T", baseParams)
		}

		updateFixture, err := harness.Fixture(
			"cardano-ledger/eras/conway/impl/golden/pparams-update.json",
		)
		if err != nil {
			t.Fatalf("Fixture failed: %v", err)
		}
		update, err := updateFixture.DecodeProtocolParameterUpdate()
		if err != nil {
			t.Fatalf("DecodeProtocolParameterUpdate failed: %v", err)
		}
		conwayUpdate, ok := update.Conway()
		if !ok {
			t.Fatalf(
				"expected Conway protocol-parameter update, got %T",
				update.Value(),
			)
		}
		if conwayUpdate.GovActionDeposit == nil {
			t.Fatal("expected Conway update to include governance deposit")
		}
		beforeGovActionDeposit := conwayParams.GovActionDeposit
		if err := update.ApplyTo(conwayParams); err != nil {
			t.Fatalf("ApplyTo failed: %v", err)
		}
		if conwayParams.GovActionDeposit != *conwayUpdate.GovActionDeposit {
			t.Fatalf(
				"expected updated governance deposit %d, got %d",
				*conwayUpdate.GovActionDeposit,
				conwayParams.GovActionDeposit,
			)
		}
		if conwayParams.GovActionDeposit == beforeGovActionDeposit {
			t.Fatal(
				"expected protocol parameters to change after applying update",
			)
		}
	})
}

func expectedFixturesFromManifest(
	t *testing.T,
	harness *fixtures.Harness,
) []fixtures.Fixture {
	t.Helper()

	root := harness.FixturesRoot()
	manifest, err := fixtures.LoadManifest(root)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	expected := make([]fixtures.Fixture, 0, len(manifest))
	for _, relPath := range manifest {
		fixture, err := fixtures.NewFixture(
			root,
			filepath.Join(root, filepath.FromSlash(relPath)),
		)
		if err != nil {
			t.Fatalf("NewFixture failed for manifest entry %q: %v", relPath, err)
		}
		expected = append(expected, fixture)
	}
	return expected
}

func repoCounts(collected []fixtures.Fixture) map[fixtures.Repo]int {
	counts := make(map[fixtures.Repo]int, len(collected))
	for _, fixture := range collected {
		counts[fixture.Repo]++
	}
	return counts
}

func countMatchingFixtures(
	collected []fixtures.Fixture,
	filter fixtures.Filter,
) int {
	var count int
	for _, fixture := range collected {
		if filter.Matches(fixture) {
			count++
		}
	}
	return count
}

func walkFixtureFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(
		root,
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if filepath.Base(path) == "manifest.txt" {
				return nil
			}
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(relPath))
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func firstSliceDifference(expected []string, actual []string) (int, string, string) {
	maxIndex := len(expected)
	if len(actual) < maxIndex {
		maxIndex = len(actual)
	}
	for idx := 0; idx < maxIndex; idx++ {
		if expected[idx] != actual[idx] {
			return idx, expected[idx], actual[idx]
		}
	}
	if len(expected) > maxIndex {
		return maxIndex, expected[maxIndex], "<missing>"
	}
	if len(actual) > maxIndex {
		return maxIndex, "<missing>", actual[maxIndex]
	}
	return -1, "", ""
}
