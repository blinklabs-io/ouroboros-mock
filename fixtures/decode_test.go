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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger"
)

func TestConsensusEnvelopeKindGuards(t *testing.T) {
	harness := NewHarness(HarnessConfig{})

	txIDFixture, err := harness.Fixture(
		"ouroboros-consensus/ouroboros-consensus-cardano/golden/cardano/CardanoNodeToNodeVersion2/GenTxId_Conway",
	)
	if err != nil {
		t.Fatalf("Fixture failed: %v", err)
	}
	// ConsensusTransactionBytes guards on f.Kind (metadata) rather than envelope.Kind() (binary heuristic)
	// so a fixture with Kind=KindTransactionID is rejected regardless of payload format.
	if _, err := txIDFixture.ConsensusTransactionBytes(); !errors.Is(
		err,
		ErrNotTransactionFixture,
	) {
		t.Fatalf("expected ErrNotTransactionFixture, got %v", err)
	}

	txFixture, err := harness.Fixture(
		"ouroboros-consensus/ouroboros-consensus-cardano/golden/cardano/CardanoNodeToNodeVersion2/GenTx_Conway",
	)
	if err != nil {
		t.Fatalf("Fixture failed: %v", err)
	}
	txFixture.Kind = KindTransactionID
	if _, err := txFixture.ConsensusTransactionIDBytes(); !errors.Is(
		err,
		ErrNotTransactionIDEnvelope,
	) {
		t.Fatalf("expected ErrNotTransactionIDEnvelope, got %v", err)
	}
}

func TestByronLedgerTypesUseConsensusWrappers(t *testing.T) {
	harness := NewHarness(HarnessConfig{})

	blockFixture, err := harness.Fixture(
		"ouroboros-consensus/ouroboros-consensus-cardano/golden/cardano/CardanoNodeToNodeVersion2/Block_Byron_EBB",
	)
	if err != nil {
		t.Fatalf("Fixture failed: %v", err)
	}
	blockFixture.Name = "not-an-ebb"
	blockType, err := blockFixture.LedgerBlockType()
	if err != nil {
		t.Fatalf("LedgerBlockType failed: %v", err)
	}
	if blockType != ledger.BlockTypeByronEbb {
		t.Fatalf("expected Byron EBB block type, got %d", blockType)
	}

	headerFixture, err := harness.Fixture(
		"ouroboros-consensus/ouroboros-consensus-cardano/golden/cardano/CardanoNodeToNodeVersion2/Header_Byron_regular",
	)
	if err != nil {
		t.Fatalf("Fixture failed: %v", err)
	}
	headerFixture.Name = "not-a-main-header"
	headerType, err := headerFixture.LedgerHeaderType()
	if err != nil {
		t.Fatalf("LedgerHeaderType failed: %v", err)
	}
	if headerType != ledger.BlockTypeByronMain {
		t.Fatalf("expected Byron main header type, got %d", headerType)
	}
}

func TestClassifyFixtureCanonicalTransactionUsesConwayEra(t *testing.T) {
	kind, format, era := classifyFixture(
		"cardano-api/cardano-api/test/cardano-api-golden/files/tx-canonical.json",
	)
	if kind != KindTransaction {
		t.Fatalf("expected transaction kind, got %s", kind)
	}
	if format != FormatJSON {
		t.Fatalf("expected JSON format, got %s", format)
	}
	if era != "conway" {
		t.Fatalf("expected conway era, got %q", era)
	}
}

func TestDecodeDijkstraProtocolParametersRejectsUnsupportedRefScriptFields(
	t *testing.T,
) {
	harness := NewHarness(HarnessConfig{})

	paramsFixture, err := harness.Fixture(
		"cardano-ledger/eras/dijkstra/impl/golden/pparams.json",
	)
	if err != nil {
		t.Fatalf("Fixture failed: %v", err)
	}
	if _, err := paramsFixture.DecodeProtocolParameters(); !errors.Is(
		err,
		errUnsupportedDijkstraRefScriptFields,
	) {
		t.Fatalf("expected unsupported Dijkstra ref-script error, got %v", err)
	}

	updateFixture, err := harness.Fixture(
		"cardano-ledger/eras/dijkstra/impl/golden/pparams-update.json",
	)
	if err != nil {
		t.Fatalf("Fixture failed: %v", err)
	}
	if _, err := updateFixture.DecodeProtocolParameterUpdate(); !errors.Is(
		err,
		errUnsupportedDijkstraRefScriptFields,
	) {
		t.Fatalf("expected unsupported Dijkstra ref-script error, got %v", err)
	}
}

func TestDecodeDijkstraConwayNamedProtocolParametersRejectsUnsupportedRefScriptFields(
	t *testing.T,
) {
	fixture := writeTempJSONFixture(
		t,
		filepath.Join(
			"cardano-ledger",
			"eras",
			"dijkstra",
			"impl",
			"protocol-parameters",
			"conway.json",
		),
		`{
			"txFeePerByte": 1,
			"maxRefScriptSizePerBlock": 1,
			"txFeeFixed": 2,
			"maxBlockBodySize": 3,
			"maxTxSize": 4,
			"maxBlockHeaderSize": 5,
			"stakeAddressDeposit": 6,
			"stakePoolDeposit": 7,
			"poolRetireMaxEpoch": 8,
			"stakePoolTargetNum": 9,
			"poolPledgeInfluence": 1,
			"monetaryExpansion": 1,
			"treasuryCut": 1,
			"executionUnitPrices": {
				"priceMemory": 1,
				"priceSteps": 1
			},
			"maxTxExecutionUnits": {
				"memory": 1,
				"steps": 1
			},
			"maxBlockExecutionUnits": {
				"memory": 1,
				"steps": 1
			},
			"protocolVersion": {
				"major": 2,
				"minor": 0
			}
		}`,
	)
	if _, err := fixture.DecodeProtocolParameters(); !errors.Is(
		err,
		errUnsupportedDijkstraRefScriptFields,
	) {
		t.Fatalf("expected unsupported Dijkstra ref-script error, got %v", err)
	}
}

func TestExecuteProtocolParametersUpdateFixtureIgnoresUnsupportedDijkstraBaseRefScriptFields(
	t *testing.T,
) {
	rootDir := t.TempDir()
	baseFixture := writeTempJSONFixtureInRoot(
		t,
		rootDir,
		filepath.Join(
			"cardano-ledger",
			"eras",
			"dijkstra",
			"impl",
			"golden",
			"pparams.json",
		),
		`{
			"txFeePerByte": 1,
			"maxRefScriptSizePerBlock": 1,
			"txFeeFixed": 2,
			"maxBlockBodySize": 3,
			"maxTxSize": 4,
			"maxBlockHeaderSize": 5,
			"stakeAddressDeposit": 6,
			"stakePoolDeposit": 7,
			"poolRetireMaxEpoch": 8,
			"stakePoolTargetNum": 9,
			"poolPledgeInfluence": 1,
			"monetaryExpansion": 1,
			"treasuryCut": 1,
			"executionUnitPrices": {
				"priceMemory": 1,
				"priceSteps": 1
			},
			"maxTxExecutionUnits": {
				"memory": 1,
				"steps": 1
			},
			"maxBlockExecutionUnits": {
				"memory": 1,
				"steps": 1
			},
			"protocolVersion": {
				"major": 2,
				"minor": 0
			}
		}`,
	)
	updateFixture := writeTempJSONFixtureInRoot(
		t,
		rootDir,
		filepath.Join(
			"cardano-ledger",
			"eras",
			"dijkstra",
			"impl",
			"golden",
			"pparams-update.json",
		),
		`{"committeeMinSize": 1}`,
	)

	caseCount, err := executeProtocolParametersUpdateFixture(
		updateFixture,
		map[string]Fixture{
			baseFixture.RelPath:   baseFixture,
			updateFixture.RelPath: updateFixture,
		},
	)
	if err != nil {
		t.Fatalf(
			"expected unsupported Dijkstra paired-base fixture to be ignored, got %v",
			err,
		)
	}
	if caseCount != 1 {
		t.Fatalf("expected single execution case, got %d", caseCount)
	}
}

func TestValidateReferenceValidatesOptionalReferenceHash(t *testing.T) {
	t.Run("MissingHashAlgorithm", func(t *testing.T) {
		err := validateReference(governanceReference{
			Type:  "Link",
			Label: "spec",
			URI:   "https://example.com",
			ReferenceHash: &metadataReferenceHash{
				HashDigest: strings.Repeat("00", 32),
			},
		}, false)
		if err == nil || !strings.Contains(err.Error(), "referenceHash.hashAlgorithm") {
			t.Fatalf("expected optional reference hash to require algorithm, got %v", err)
		}
	})

	t.Run("InvalidHashDigest", func(t *testing.T) {
		err := validateReference(governanceReference{
			Type:  "Link",
			Label: "spec",
			URI:   "https://example.com",
			ReferenceHash: &metadataReferenceHash{
				HashAlgorithm: "sha256",
				HashDigest:    "zz",
			},
		}, false)
		if err == nil || !strings.Contains(err.Error(), "referenceHash.hashDigest") {
			t.Fatalf("expected optional reference hash digest validation, got %v", err)
		}
	})
}

func TestDecodeProtocolJSONRejectsMalformedNonce(t *testing.T) {
	paramsFixture := writeTempJSONFixture(
		t,
		filepath.Join(
			"cardano-ledger",
			"eras",
			"shelley",
			"impl",
			"golden",
			"pparams.json",
		),
		`{
			"txFeePerByte": 1,
			"txFeeFixed": 2,
			"maxBlockBodySize": 3,
			"maxTxSize": 4,
			"maxBlockHeaderSize": 5,
			"stakeAddressDeposit": 6,
			"stakePoolDeposit": 7,
			"poolRetireMaxEpoch": 8,
			"stakePoolTargetNum": 9,
			"poolPledgeInfluence": 1,
			"monetaryExpansion": 1,
			"treasuryCut": 1,
			"extraPraosEntropy": "zz",
			"protocolVersion": {
				"major": 2,
				"minor": 0
			}
		}`,
	)
	if _, err := paramsFixture.DecodeProtocolParameters(); err == nil {
		t.Fatal("expected protocol-parameters decode to reject malformed nonce")
	} else if !strings.Contains(err.Error(), "invalid extraPraosEntropy") {
		t.Fatalf("unexpected error: %v", err)
	}

	updateFixture := writeTempJSONFixture(
		t,
		filepath.Join(
			"cardano-ledger",
			"eras",
			"shelley",
			"impl",
			"golden",
			"pparams-update.json",
		),
		`{"extraPraosEntropy": "zz"}`,
	)
	if _, err := updateFixture.DecodeProtocolParameterUpdate(); err == nil {
		t.Fatal(
			"expected protocol-parameters update decode to reject malformed nonce",
		)
	} else if !strings.Contains(err.Error(), "invalid extraPraosEntropy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTempJSONFixtureInRoot(
	t *testing.T,
	rootDir string,
	relPath string,
	content string,
) Fixture {
	t.Helper()

	path := filepath.Join(rootDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	fixture, err := NewFixture(rootDir, path)
	if err != nil {
		t.Fatalf("NewFixture failed: %v", err)
	}
	return fixture
}

func writeTempJSONFixture(
	t *testing.T,
	relPath string,
	content string,
) Fixture {
	t.Helper()
	rootDir := t.TempDir()
	return writeTempJSONFixtureInRoot(t, rootDir, relPath, content)
}
