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
	"bytes"
	"errors"
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

const plutusV1FixtureRoot = "cardano-api/cardano-api/test/cardano-api-golden/files/Script/PlutusScriptV1/"

// PlutusScriptBytes returns the CBOR byte-string encoding used for the imported
// Plutus script's hash, preserving its bytes across hex and binary sources.
// It checks the CBOR container without evaluating the Plutus program.
func (f Fixture) PlutusScriptBytes() ([]byte, error) {
	if f.Kind != KindScript {
		return nil, fmt.Errorf("fixture %s is not a script fixture", f.RelPath)
	}
	var data []byte
	var err error
	switch f.Format {
	case FormatHex:
		data, err = f.DecodeHex()
	case FormatCBOR:
		data, err = f.Read()
	case FormatUnknown, FormatHexDump, FormatJSON, FormatJSONLD:
		return nil, fmt.Errorf("unsupported script format %s", f.Format)
	default:
		return nil, fmt.Errorf("unknown script format %s", f.Format)
	}
	if err != nil {
		return nil, err
	}
	if err := validateArbitraryCbor(data, "Plutus script"); err != nil {
		return nil, err
	}
	if len(data) == 0 || data[0]>>5 != 2 {
		return nil, errors.New("plutus script must be a CBOR byte string")
	}
	var program []byte
	if _, err := cbor.Decode(data, &program); err != nil {
		return nil, err
	}
	if len(program) == 0 {
		return nil, errors.New("plutus script byte string is empty")
	}
	return data, nil
}

func executeScriptFixture(
	fixture Fixture,
	fixtureMap map[string]Fixture,
) error {
	data, err := fixture.PlutusScriptBytes()
	if err != nil {
		return err
	}
	const expectedHash = "58503a1d89a21fc9fc53d6a7cccef47341175a8f47636f57ccbdca2d"
	if fixture.RelPath != plutusV1FixtureRoot+"alwayssucceeds.txt" &&
		fixture.RelPath != plutusV1FixtureRoot+"alwayssucceeds.bin" {
		return fmt.Errorf(
			"unsupported Plutus script fixture %s",
			fixture.RelPath,
		)
	}
	hash := common.PlutusV1Script(data).Hash().String()
	if hash != expectedHash {
		return fmt.Errorf(
			"PlutusV1 script hash: got %s want %s",
			hash,
			expectedHash,
		)
	}
	pairName := "alwayssucceeds.txt"
	if fixture.Format == FormatHex {
		pairName = "alwayssucceeds.bin"
	}
	if paired, ok := fixtureMap[plutusV1FixtureRoot+pairName]; ok {
		pairedBytes, err := paired.PlutusScriptBytes()
		if err != nil {
			return fmt.Errorf("paired script %s: %w", paired.RelPath, err)
		}
		if !bytes.Equal(data, pairedBytes) {
			return fmt.Errorf(
				"script bytes differ from paired fixture %s",
				paired.RelPath,
			)
		}
	}
	return nil
}
