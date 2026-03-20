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
	"maps"
	"slices"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
)

func TestHarnessIntegration(t *testing.T) {
	harness := fixtures.NewHarness(fixtures.HarnessConfig{})

	t.Run("CollectFixtures", func(t *testing.T) {
		collected, err := harness.Collect()
		if err != nil {
			t.Fatalf("Collect failed: %v", err)
		}
		if len(collected) != 61 {
			t.Fatalf("expected 61 fixtures, got %d", len(collected))
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

		expectedCounts := map[fixtures.Repo]int{
			fixtures.RepoCardanoAPI:         7,
			fixtures.RepoCardanoLedger:      18,
			fixtures.RepoCardanoNode:        2,
			fixtures.RepoOuroborosConsensus: 34,
		}
		if !maps.Equal(repoCounts, expectedCounts) {
			t.Fatalf("unexpected repo counts: got %v want %v", repoCounts, expectedCounts)
		}
	})

	t.Run("ManifestMatchesCollectedPaths", func(t *testing.T) {
		manifest, err := fixtures.LoadManifest(harness.FixturesRoot())
		if err != nil {
			t.Fatalf("LoadManifest failed: %v", err)
		}

		collected, err := harness.Collect()
		if err != nil {
			t.Fatalf("Collect failed: %v", err)
		}

		collectedPaths := make([]string, 0, len(collected))
		for _, fixture := range collected {
			collectedPaths = append(collectedPaths, fixture.RelPath)
		}

		if !slices.Equal(manifest, collectedPaths) {
			t.Fatalf("manifest mismatch")
		}
	})

	t.Run("RunMatching", func(t *testing.T) {
		var blockCount int
		harness.RunMatching(t, fixtures.Filter{
			Repo: fixtures.RepoOuroborosConsensus,
			Kind: fixtures.KindBlock,
		}, func(t *testing.T, fixture fixtures.Fixture) {
			blockCount++
			if fixture.Format != fixtures.FormatCBOR {
				t.Fatalf("expected cbor block fixture, got %s", fixture.Format)
			}
		})
		if blockCount != 9 {
			t.Fatalf("expected 9 consensus block fixtures, got %d", blockCount)
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
		if len(collected) != 61 {
			t.Fatalf("expected 61 extracted fixtures, got %d", len(collected))
		}
	})
}

func TestFixtureExecutionHarness(t *testing.T) {
	harness := fixtures.NewHarness(fixtures.HarnessConfig{})

	t.Run("RunAllExecutions", func(t *testing.T) {
		harness.RunAllExecutions(t)
	})

	t.Run("RunAllExecutionsWithResults", func(t *testing.T) {
		results, err := harness.RunAllExecutionsWithResults()
		if err != nil {
			t.Fatalf("RunAllExecutionsWithResults failed: %v", err)
		}
		if len(results) != 61 {
			t.Fatalf("expected 61 execution results, got %d", len(results))
		}

		var totalCases int
		for _, result := range results {
			totalCases += result.CaseCount
			if !result.Success {
				t.Fatalf("%s execution failed: %v", result.Fixture.RelPath, result.Error)
			}
			if result.CaseCount == 0 {
				t.Fatalf("%s reported zero execution cases", result.Fixture.RelPath)
			}
		}
		if totalCases <= len(results) {
			t.Fatalf("expected multi-case execution coverage, got %d total cases for %d fixtures", totalCases, len(results))
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
			t.Fatalf("block/header hash mismatch: %s != %s", block.Hash(), header.Hash())
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
			t.Fatalf("DecodeLedgerTransaction on cardano-api canonical tx failed: %v", err)
		}
		if canonicalTx.Type() != int(ledger.TxTypeConway) {
			t.Fatalf("expected Conway transaction type, got %d", canonicalTx.Type())
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
			t.Fatalf("expected Conway protocol-parameter update, got %T", update.Value())
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
			t.Fatal("expected protocol parameters to change after applying update")
		}
	})
}
