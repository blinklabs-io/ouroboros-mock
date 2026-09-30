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
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type boundedParameter struct {
	jsonName string
	goField  string
	width    string
	limit    uint64
}

var (
	word32Parameters = []boundedParameter{
		{"maxBlockBodySize", "MaxBlockBodySize", "Word32", math.MaxUint32},
		{"maxTxSize", "MaxTxSize", "Word32", math.MaxUint32},
		{"poolRetireMaxEpoch", "MaxEpoch", "Word32", math.MaxUint32},
	}
	word16Parameters = []boundedParameter{
		{"maxBlockHeaderSize", "MaxBlockHeaderSize", "Word16", math.MaxUint16},
		{"stakePoolTargetNum", "NOpt", "Word16", math.MaxUint16},
	}
	alonzoParameters = []boundedParameter{
		{"maxValueSize", "MaxValueSize", "Word32", math.MaxUint32},
		{"collateralPercentage", "CollateralPercentage", "Word16", math.MaxUint16},
		{"maxCollateralInputs", "MaxCollateralInputs", "Word16", math.MaxUint16},
	}
	conwayParameters = []boundedParameter{
		{"committeeMinSize", "MinCommitteeSize", "Word16", math.MaxUint16},
		{"committeeMaxTermLength", "CommitteeTermLimit", "Word32", math.MaxUint32},
		{"govActionLifetime", "GovActionValidityPeriod", "Word32", math.MaxUint32},
		{"dRepActivity", "DRepInactivityPeriod", "Word32", math.MaxUint32},
	}
)

// boundedParametersByEra lists the protocol parameters whose reference type
// (cardano-ledger PParams, and CDDL uint .size 2 / .size 4 or epoch_interval)
// is narrower than the gouroboros field that receives them.
func boundedParametersByEra() map[string][]boundedParameter {
	shelley := append(append([]boundedParameter{}, word32Parameters...), word16Parameters...)
	alonzo := append(append([]boundedParameter{}, shelley...), alonzoParameters...)
	conway := append(append([]boundedParameter{}, alonzo...), conwayParameters...)
	return map[string][]boundedParameter{
		"shelley":  shelley,
		"alonzo":   alonzo,
		"babbage":  alonzo,
		"conway":   conway,
		"dijkstra": conway,
	}
}

func boundedParametersJSON(fields map[string]uint64, update bool) string {
	parts := make([]string, 0, len(fields)+14)
	if !update {
		parts = append(parts,
			`"txFeePerByte": 1`,
			`"txFeeFixed": 2`,
			`"stakeAddressDeposit": 6`,
			`"stakePoolDeposit": 7`,
			`"poolPledgeInfluence": 1`,
			`"monetaryExpansion": 1`,
			`"treasuryCut": 1`,
			`"executionUnitPrices": {"priceMemory": 1, "priceSteps": 1}`,
			`"maxTxExecutionUnits": {"memory": 1, "steps": 1}`,
			`"maxBlockExecutionUnits": {"memory": 1, "steps": 1}`,
			`"protocolVersion": {"major": 2, "minor": 0}`,
		)
	}
	for name, value := range fields {
		parts = append(parts, strconv.Quote(name)+": "+strconv.FormatUint(value, 10))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func decodeBoundedParameterFixture(
	t *testing.T,
	era string,
	update bool,
	fields map[string]uint64,
) (any, error) {
	t.Helper()
	golden := filepath.Join("cardano-ledger", "eras", era, "impl", "golden")
	if update {
		fixture := writeTempJSONFixture(
			t,
			filepath.Join(golden, "pparams-update.json"),
			boundedParametersJSON(fields, true),
		)
		decoded, err := fixture.DecodeProtocolParameterUpdate()
		return decoded.Value(), err
	}
	fixture := writeTempJSONFixture(
		t,
		filepath.Join(golden, "pparams.json"),
		boundedParametersJSON(fields, false),
	)
	return fixture.DecodeProtocolParameters()
}

// decodedUnsignedField reads a named unsigned field, through a pointer when the
// field is optional, from the gouroboros value the fixture decoded into.
func decodedUnsignedField(t *testing.T, value any, name string) uint64 {
	t.Helper()
	target := reflect.ValueOf(value)
	if target.Kind() != reflect.Pointer || target.IsNil() {
		t.Fatalf("decoded value %T is not a non-nil pointer", value)
	}
	field := target.Elem().FieldByName(name)
	if !field.IsValid() {
		t.Fatalf("%T has no field %s", value, name)
	}
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			t.Fatalf("%T field %s is unset", value, name)
		}
		field = field.Elem()
	}
	return field.Uint()
}

func TestDecodeProtocolParametersPreservesReferenceWidthMaximums(t *testing.T) {
	for era, parameters := range boundedParametersByEra() {
		for _, update := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/update=%t", era, update), func(t *testing.T) {
				fields := make(map[string]uint64, len(parameters))
				for _, parameter := range parameters {
					fields[parameter.jsonName] = parameter.limit
				}
				decoded, err := decodeBoundedParameterFixture(t, era, update, fields)
				if err != nil {
					t.Fatalf("decode failed: %v", err)
				}
				for _, parameter := range parameters {
					got := decodedUnsignedField(t, decoded, parameter.goField)
					if got != parameter.limit {
						t.Fatalf(
							"%s = %d, want %d",
							parameter.jsonName,
							got,
							parameter.limit,
						)
					}
				}
			})
		}
	}
}

func TestDecodeProtocolParametersRejectsValuesAboveReferenceWidth(t *testing.T) {
	for era, parameters := range boundedParametersByEra() {
		for _, update := range []bool{false, true} {
			for _, parameter := range parameters {
				name := fmt.Sprintf("%s/update=%t/%s", era, update, parameter.jsonName)
				t.Run(name, func(t *testing.T) {
					fields := make(map[string]uint64, len(parameters))
					for _, other := range parameters {
						fields[other.jsonName] = 1
					}
					fields[parameter.jsonName] = parameter.limit + 1
					decoded, err := decodeBoundedParameterFixture(t, era, update, fields)
					if err == nil {
						t.Fatalf(
							"decode accepted %s = %d into %T, want %s range error",
							parameter.jsonName,
							parameter.limit+1,
							decoded,
							parameter.width,
						)
					}
					if !strings.Contains(err.Error(), "exceeds "+parameter.width+" range") {
						t.Fatalf(
							"decode error = %v, want %s range error",
							err,
							parameter.width,
						)
					}
				})
			}
		}
	}
}
