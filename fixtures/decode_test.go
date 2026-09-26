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
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
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

func TestDecodeDijkstraProtocolParametersMapsRefScriptFields(
	t *testing.T,
) {
	harness := NewHarness(HarnessConfig{})

	paramsFixture, err := harness.Fixture(
		"cardano-ledger/eras/dijkstra/impl/golden/pparams.json",
	)
	if err != nil {
		t.Fatalf("Fixture failed: %v", err)
	}
	params, err := paramsFixture.DecodeProtocolParameters()
	if err != nil {
		t.Fatalf("DecodeProtocolParameters failed: %v", err)
	}
	dijkstraParams, ok := params.(*dijkstra.DijkstraProtocolParameters)
	if !ok {
		t.Fatalf("expected Dijkstra parameters, got %T", params)
	}
	if dijkstraParams.MaxRefScriptSizePerBlock == 0 ||
		dijkstraParams.MaxRefScriptSizePerTx == 0 ||
		dijkstraParams.RefScriptCostStride == 0 ||
		dijkstraParams.RefScriptCostMultiplier == nil {
		t.Fatal("expected Dijkstra ref-script fields to be mapped")
	}

	updateFixture, err := harness.Fixture(
		"cardano-ledger/eras/dijkstra/impl/golden/pparams-update.json",
	)
	if err != nil {
		t.Fatalf("Fixture failed: %v", err)
	}
	update, err := updateFixture.DecodeProtocolParameterUpdate()
	if err != nil {
		t.Fatalf("DecodeProtocolParameterUpdate failed: %v", err)
	}
	if _, ok := update.Value().(*dijkstra.DijkstraProtocolParameterUpdate); !ok {
		t.Fatalf("expected Dijkstra update, got %T", update.Value())
	}
}

func TestSetMaxEpochValueSupportsWord64Models(t *testing.T) {
	const want = uint64(math.MaxUint64)

	t.Run("value field", func(t *testing.T) {
		var target struct{ MaxEpoch uint64 }
		if err := setMaxEpochValue(&target, want); err != nil {
			t.Fatalf("setMaxEpochValue failed: %v", err)
		}
		if target.MaxEpoch != want {
			t.Fatalf("MaxEpoch = %d, want %d", target.MaxEpoch, want)
		}
	})

	t.Run("pointer field", func(t *testing.T) {
		var target struct{ MaxEpoch *uint64 }
		if err := setMaxEpochValue(&target, want); err != nil {
			t.Fatalf("setMaxEpochValue failed: %v", err)
		}
		if target.MaxEpoch == nil {
			t.Fatal("MaxEpoch was not set")
		}
		if *target.MaxEpoch != want {
			t.Fatalf("MaxEpoch = %d, want %d", *target.MaxEpoch, want)
		}
	})
}

func TestSetOptionalMaxEpoch(t *testing.T) {
	const want = uint64(math.MaxUint64)
	var target struct{ MaxEpoch *uint64 }

	if err := setOptionalMaxEpoch(&target, nil); err != nil {
		t.Fatalf("setOptionalMaxEpoch with nil value failed: %v", err)
	}
	if target.MaxEpoch != nil {
		t.Fatal("nil MaxEpoch unexpectedly changed the destination")
	}

	if err := setOptionalMaxEpoch(
		&target,
		&jsonUint64{value: want},
	); err != nil {
		t.Fatalf("setOptionalMaxEpoch failed: %v", err)
	}
	if target.MaxEpoch == nil || *target.MaxEpoch != want {
		t.Fatalf("MaxEpoch = %v, want %d", target.MaxEpoch, want)
	}
}

func TestSetMaxEpochValueRejectsUnsupportedDestinations(t *testing.T) {
	tests := []struct {
		name   string
		target any
		value  uint64
		want   string
	}{
		{
			name:   "nil destination",
			target: nil,
			want:   "non-nil pointer",
		},
		{
			name:   "non-pointer destination",
			target: struct{ MaxEpoch uint64 }{},
			want:   "non-nil pointer",
		},
		{
			name:   "nil pointer destination",
			target: (*struct{ MaxEpoch uint64 })(nil),
			want:   "non-nil pointer",
		},
		{
			name:   "non-struct pointer destination",
			target: new(uint64),
			want:   "must point to a struct",
		},
		{
			name:   "missing field",
			target: &struct{ Other uint64 }{},
			want:   "no writable MaxEpoch field",
		},
		{
			name:   "unsupported field",
			target: &struct{ MaxEpoch string }{},
			want:   "unsupported type string",
		},
		{
			name:   "unsupported unsigned field",
			target: &struct{ MaxEpoch *uint32 }{},
			want:   "unsupported type *uint32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setMaxEpochValue(tt.target, tt.value)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("setMaxEpochValue error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestSetMaxEpochValueChecksUintOverflow(t *testing.T) {
	const want = uint64(math.MaxUint64)
	var target struct{ MaxEpoch *uint }
	err := setMaxEpochValue(&target, want)
	if reflect.TypeOf(uint(0)).Bits() == 64 {
		if err != nil {
			t.Fatalf("setMaxEpochValue failed: %v", err)
		}
		if target.MaxEpoch == nil || uint64(*target.MaxEpoch) != want {
			t.Fatalf("MaxEpoch = %v, want %d", target.MaxEpoch, want)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), "exceeds uint") {
		t.Fatalf("setMaxEpochValue error = %v, want uint overflow", err)
	}
	if target.MaxEpoch != nil {
		t.Fatalf("MaxEpoch = %d after overflow, want nil", *target.MaxEpoch)
	}
}

func TestDecodeProtocolParametersMapsMaxEpochAcrossEras(t *testing.T) {
	const maxEpoch = uint64(math.MaxUint64)
	const parameters = `{
		"txFeePerByte": 1,
		"txFeeFixed": 2,
		"maxBlockBodySize": 3,
		"maxTxSize": 4,
		"maxBlockHeaderSize": 5,
		"stakeAddressDeposit": 6,
		"stakePoolDeposit": 7,
		"poolRetireMaxEpoch": 18446744073709551615,
		"stakePoolTargetNum": 9,
		"poolPledgeInfluence": 1,
		"monetaryExpansion": 1,
		"treasuryCut": 1,
		"executionUnitPrices": {
			"priceMemory": 1,
			"priceSteps": 1
		},
		"maxTxExecutionUnits": {"memory": 1, "steps": 1},
		"maxBlockExecutionUnits": {"memory": 1, "steps": 1},
		"protocolVersion": {"major": 2, "minor": 0}
	}`
	const update = `{"poolRetireMaxEpoch":18446744073709551615}`

	eras := []struct {
		name        string
		paramsModel any
		updateModel any
	}{
		{
			name:        "shelley",
			paramsModel: (*shelley.ShelleyProtocolParameters)(nil),
			updateModel: (*shelley.ShelleyProtocolParameterUpdate)(nil),
		},
		{
			name:        "alonzo",
			paramsModel: (*alonzo.AlonzoProtocolParameters)(nil),
			updateModel: (*alonzo.AlonzoProtocolParameterUpdate)(nil),
		},
		{
			name:        "babbage",
			paramsModel: (*babbage.BabbageProtocolParameters)(nil),
			updateModel: (*babbage.BabbageProtocolParameterUpdate)(nil),
		},
		{
			name:        "conway",
			paramsModel: (*conway.ConwayProtocolParameters)(nil),
			updateModel: (*conway.ConwayProtocolParameterUpdate)(nil),
		},
		{
			name:        "dijkstra",
			paramsModel: (*dijkstra.DijkstraProtocolParameters)(nil),
			updateModel: (*dijkstra.DijkstraProtocolParameterUpdate)(nil),
		},
	}
	for _, era := range eras {
		t.Run(era.name+" parameters", func(t *testing.T) {
			fixture := writeTempJSONFixture(
				t,
				filepath.Join("cardano-ledger", "eras", era.name, "impl", "golden", "pparams.json"),
				parameters,
			)
			params, err := fixture.DecodeProtocolParameters()
			assertMaxEpochDecodeResult(
				t,
				params,
				err,
				maxEpoch,
				maxEpochFieldBits(t, era.paramsModel),
			)
		})

		t.Run(era.name+" update", func(t *testing.T) {
			fixture := writeTempJSONFixture(
				t,
				filepath.Join("cardano-ledger", "eras", era.name, "impl", "golden", "pparams-update.json"),
				update,
			)
			params, err := fixture.DecodeProtocolParameterUpdate()
			assertMaxEpochDecodeResult(
				t,
				params.Value(),
				err,
				maxEpoch,
				maxEpochFieldBits(t, era.updateModel),
			)
		})
	}
}

func assertMaxEpochDecodeResult(
	t *testing.T,
	target any,
	err error,
	want uint64,
	fieldBits int,
) {
	t.Helper()
	if err != nil {
		wantError := "MaxEpoch value " + strconv.FormatUint(want, 10) + " exceeds uint"
		if fieldBits >= 64 || err.Error() != wantError {
			t.Fatalf("decode error = %v, want uint overflow", err)
		}
		return
	}
	if fieldBits < 64 {
		// A narrow uint model cannot preserve the full JSON Word64 value.
		t.Fatalf("decode unexpectedly accepted a Word64 value into %d-bit field", fieldBits)
	}
	assertMaxEpochValue(t, target, want)
}

func maxEpochFieldBits(t *testing.T, model any) int {
	t.Helper()
	typeOf := reflect.TypeOf(model)
	if typeOf.Kind() != reflect.Pointer {
		t.Fatalf("expected a pointer model type, got %T", model)
	}
	field, ok := typeOf.Elem().FieldByName("MaxEpoch")
	if !ok {
		t.Fatalf("%s has no MaxEpoch field", typeOf.Elem())
	}
	fieldType := field.Type
	if fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}
	if fieldType.Kind() != reflect.Uint && fieldType.Kind() != reflect.Uint64 {
		t.Fatalf("%s has unsupported MaxEpoch type %s", typeOf.Elem(), fieldType)
	}
	return fieldType.Bits()
}

func assertMaxEpochValue(t *testing.T, target any, want uint64) {
	t.Helper()
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		t.Fatalf("expected a non-nil pointer, got %T", target)
	}
	field := value.Elem().FieldByName("MaxEpoch")
	if !field.IsValid() {
		t.Fatalf("%T has no MaxEpoch field", target)
	}
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			t.Fatalf("%T MaxEpoch is nil", target)
		}
		field = field.Elem()
	}
	if field.Kind() != reflect.Uint && field.Kind() != reflect.Uint64 {
		t.Fatalf("%T MaxEpoch has unsupported type %s", target, field.Type())
	}
	if got := field.Uint(); got != want {
		t.Fatalf("%T MaxEpoch = %d, want %d", target, got, want)
	}
}

func TestDecodeDijkstraConwayNamedProtocolParametersMapsRefScriptFields(
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
	params, err := fixture.DecodeProtocolParameters()
	if err != nil {
		t.Fatalf("DecodeProtocolParameters failed: %v", err)
	}
	dijkstraParams, ok := params.(*dijkstra.DijkstraProtocolParameters)
	if !ok || dijkstraParams.MaxRefScriptSizePerBlock != 1 {
		t.Fatalf("expected mapped Dijkstra parameters, got %#v", params)
	}
}

func TestExecuteProtocolParametersUpdateFixtureMapsDijkstraBaseRefScriptFields(
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
			"expected Dijkstra paired-base fixture to execute, got %v",
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
		if err == nil ||
			!strings.Contains(err.Error(), "referenceHash.hashAlgorithm") {
			t.Fatalf(
				"expected optional reference hash to require algorithm, got %v",
				err,
			)
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
		if err == nil ||
			!strings.Contains(err.Error(), "referenceHash.hashDigest") {
			t.Fatalf(
				"expected optional reference hash digest validation, got %v",
				err,
			)
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
