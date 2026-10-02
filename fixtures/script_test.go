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
	"encoding/hex"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
)

const scriptRoot = "cardano-api/cardano-api/test/cardano-api-golden/files/Script/PlutusScriptV1/"

func TestPlutusScriptFixturesPreserveBytesAndHash(t *testing.T) {
	harness := fixtures.NewHarness(fixtures.HarnessConfig{})
	for _, suffix := range []string{"txt", "bin"} {
		t.Run(suffix, func(t *testing.T) {
			f, err := harness.Fixture(scriptRoot + "alwayssucceeds." + suffix)
			require.NoError(t, err)
			require.Equal(t, fixtures.KindScript, f.Kind)
			require.Equal(t, fixtures.RepoCardanoAPI, f.Repo)
			data, err := f.PlutusScriptBytes()
			require.NoError(t, err)
			require.Equal(t, "4701000022220011", hex.EncodeToString(data))
			require.Equal(
				t,
				"58503a1d89a21fc9fc53d6a7cccef47341175a8f47636f57ccbdca2d",
				common.PlutusV1Script(data).Hash().String(),
			)
			result, err := harness.ExecuteFixture(f.RelPath)
			require.NoError(t, err)
			require.NoError(t, result.Error)
			require.True(t, result.Success)
			require.Equal(t, 1, result.CaseCount)
		})
	}
}

func TestScriptExecutionRejectsAlteredPayloadAndPair(t *testing.T) {
	for _, test := range []struct {
		name      string
		text      string
		binary    []byte
		err       string
		container bool
	}{
		{
			name:      "both copies altered",
			text:      "4100",
			binary:    []byte{0x41, 0},
			err:       "script hash",
			container: false,
		},
		{
			name:      "binary copy altered",
			text:      "4701000022220011",
			binary:    []byte{0x41, 0},
			err:       "script bytes differ",
			container: false,
		},
		{
			name:      "array",
			text:      "8100",
			binary:    []byte{0x81, 0},
			err:       "must be a CBOR byte string",
			container: true,
		},
		{
			name:      "null",
			text:      "f6",
			binary:    []byte{0xf6},
			err:       "must be a CBOR byte string",
			container: true,
		},
		{
			name:      "empty program",
			text:      "40",
			binary:    []byte{0x40},
			err:       "byte string is empty",
			container: true,
		},
		{
			name:      "trailing CBOR",
			text:      "410000",
			binary:    []byte{0x41, 0, 0},
			err:       "trailing data",
			container: true,
		},
		{
			name:      "truncated CBOR",
			text:      "42",
			binary:    []byte{0x42},
			err:       "failed to decode Plutus script",
			container: true,
		},
		{
			name:      "invalid hex",
			text:      "not hex",
			binary:    []byte{0x41, 0},
			err:       "failed to decode hex",
			container: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			harness := fixtureHarness(t, map[string][]byte{
				scriptRoot + "alwayssucceeds.txt": []byte(test.text),
				scriptRoot + "alwayssucceeds.bin": test.binary,
			})
			if test.container {
				fixture, err := harness.Fixture(
					scriptRoot + "alwayssucceeds.txt",
				)
				require.NoError(t, err)
				_, err = fixture.PlutusScriptBytes()
				require.ErrorContains(t, err, test.err)
			}
			result, err := harness.ExecuteFixture(
				scriptRoot + "alwayssucceeds.txt",
			)
			require.NoError(t, err)
			require.False(t, result.Success)
			require.ErrorContains(t, result.Error, test.err)
			require.ErrorContains(t, result.Error, "kind=script")
		})
	}
}

func TestScriptExecutionPinsUnpairedCapture(t *testing.T) {
	for _, test := range []struct {
		name, text, err string
	}{
		{name: "upstream hash", text: "4701000022220011"},
		{name: "altered hash", text: "4100", err: "script hash"},
	} {
		t.Run(test.name, func(t *testing.T) {
			harness := fixtureHarness(t, map[string][]byte{
				scriptRoot + "alwayssucceeds.txt": []byte(test.text),
			})
			result, err := harness.ExecuteFixture(
				scriptRoot + "alwayssucceeds.txt",
			)
			require.NoError(t, err)
			if test.err == "" {
				require.NoError(t, result.Error)
				require.True(t, result.Success)
			} else {
				require.ErrorContains(t, result.Error, test.err)
				require.False(t, result.Success)
			}
		})
	}
}
