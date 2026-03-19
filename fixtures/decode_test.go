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
	txIDFixture.Kind = KindTransaction
	if _, err := txIDFixture.ConsensusTransactionBytes(); err == nil {
		t.Fatal("expected transaction accessor to reject transaction-id envelope")
	} else if !strings.Contains(err.Error(), "expected transaction envelope, got transaction-id") {
		t.Fatalf("unexpected error: %v", err)
	}

	txFixture, err := harness.Fixture(
		"ouroboros-consensus/ouroboros-consensus-cardano/golden/cardano/CardanoNodeToNodeVersion2/GenTx_Conway",
	)
	if err != nil {
		t.Fatalf("Fixture failed: %v", err)
	}
	txFixture.Kind = KindTransactionID
	if _, err := txFixture.ConsensusTransactionIDBytes(); err == nil {
		t.Fatal("expected transaction-id accessor to reject transaction envelope")
	} else if !strings.Contains(err.Error(), "expected transaction-id envelope, got transaction") {
		t.Fatalf("unexpected error: %v", err)
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

func TestDecodeDijkstraProtocolParametersRejectsUnsupportedRefScriptFields(t *testing.T) {
	harness := NewHarness(HarnessConfig{})

	paramsFixture, err := harness.Fixture(
		"cardano-ledger/eras/dijkstra/impl/golden/pparams.json",
	)
	if err != nil {
		t.Fatalf("Fixture failed: %v", err)
	}
	if _, err := paramsFixture.DecodeProtocolParameters(); !errors.Is(err, errUnsupportedDijkstraRefScriptFields) {
		t.Fatalf("expected unsupported Dijkstra ref-script error, got %v", err)
	}

	updateFixture, err := harness.Fixture(
		"cardano-ledger/eras/dijkstra/impl/golden/pparams-update.json",
	)
	if err != nil {
		t.Fatalf("Fixture failed: %v", err)
	}
	if _, err := updateFixture.DecodeProtocolParameterUpdate(); !errors.Is(err, errUnsupportedDijkstraRefScriptFields) {
		t.Fatalf("expected unsupported Dijkstra ref-script error, got %v", err)
	}
}

func TestDecodeProtocolJSONRejectsMalformedNonce(t *testing.T) {
	paramsFixture := writeTempJSONFixture(
		t,
		filepath.Join("cardano-ledger", "eras", "shelley", "impl", "golden", "pparams.json"),
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
		filepath.Join("cardano-ledger", "eras", "shelley", "impl", "golden", "pparams-update.json"),
		`{"extraPraosEntropy": "zz"}`,
	)
	if _, err := updateFixture.DecodeProtocolParameterUpdate(); err == nil {
		t.Fatal("expected protocol-parameters update decode to reject malformed nonce")
	} else if !strings.Contains(err.Error(), "invalid extraPraosEntropy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTempJSONFixture(t *testing.T, relPath string, content string) Fixture {
	t.Helper()

	rootDir := t.TempDir()
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
