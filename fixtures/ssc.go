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
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/blinklabs-io/gouroboros/cbor"
)

const byronSSCFixtureRoot = "cardano-ledger/eras/byron/ledger/impl/golden/cbor/ssc/"

func decodeHexDump(data []byte) ([]byte, error) {
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var decoded []byte
	for idx, line := range lines {
		offsetText, payload, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			return nil, fmt.Errorf(
				"hex dump line %d is missing an offset",
				idx+1,
			)
		}
		offset, err := strconv.ParseUint(offsetText, 16, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid hex dump offset at line %d: %w",
				idx+1,
				err,
			)
		}
		if offset != uint64(len(decoded)) {
			return nil, fmt.Errorf(
				"hex dump line %d offset %x does not match byte position %x",
				idx+1,
				offset,
				len(decoded),
			)
		}
		chunk, err := hex.DecodeString(strings.TrimSpace(payload))
		if err != nil {
			return nil, fmt.Errorf(
				"invalid hex dump data at line %d: %w",
				idx+1,
				err,
			)
		}
		if len(chunk) == 0 || len(chunk) > 16 ||
			(idx < len(lines)-1 && len(chunk) != 16) {
			return nil, fmt.Errorf(
				"invalid hex dump chunk length at line %d: got %d",
				idx+1,
				len(chunk),
			)
		}
		decoded = append(decoded, chunk...)
	}
	return decoded, nil
}

func executeSSCFixture(fixture Fixture) (int, error) {
	data, err := fixture.DecodeHex()
	if err != nil {
		return 0, err
	}
	if err := validateArbitraryCbor(data, "SSC payload"); err != nil {
		return 0, err
	}
	switch fixture.Name {
	case "CommitmentsMap", "VssCertificatesMap":
		var tag cbor.RawTag
		if _, err := cbor.Decode(data, &tag); err != nil {
			return 0, fmt.Errorf("SSC set: %w", err)
		}
		if tag.Number != cbor.CborTagSet {
			return 0, fmt.Errorf(
				"SSC set tag: got %d want %d",
				tag.Number,
				cbor.CborTagSet,
			)
		}
		entries, err := sscArray(tag.Content, -1)
		if err != nil {
			return 0, err
		}
		for idx, entry := range entries {
			if fixture.Name == "CommitmentsMap" {
				err = validateSignedCommitment(entry)
			} else {
				err = validateVssCertificate(entry)
			}
			if err != nil {
				return 0, fmt.Errorf("SSC entry %d: %w", idx, err)
			}
		}
		return len(entries), nil
	case "OpeningsMap", "SharesMap":
		entries, err := sscMap(data)
		if err != nil {
			return 0, err
		}
		for key, value := range entries {
			if len(key.Bytes()) != 28 {
				return 0, fmt.Errorf(
					"SSC stakeholder key length: got %d want 28",
					len(key.Bytes()),
				)
			}
			if fixture.Name == "OpeningsMap" {
				err = sscBytes(value)
			} else {
				err = validateInnerShares(value, true)
			}
			if err != nil {
				return 0, fmt.Errorf("SSC stakeholder %x: %w", key.Bytes(), err)
			}
		}
		return len(entries), nil
	default:
		return 0, fmt.Errorf("unsupported SSC fixture %s", fixture.Name)
	}
}

func sscArray(data []byte, width int) ([]cbor.RawMessage, error) {
	if len(data) == 0 || data[0]>>5 != 4 {
		return nil, errors.New("SSC value must be an array")
	}
	items := make([]cbor.RawMessage, 0)
	if _, err := cbor.Decode(data, &items); err != nil {
		return nil, err
	}
	if width >= 0 && len(items) != width {
		return nil, fmt.Errorf(
			"SSC array width: got %d want %d",
			len(items),
			width,
		)
	}
	return items, nil
}

func sscMap(data []byte) (map[cbor.ByteString]cbor.RawMessage, error) {
	if len(data) == 0 || data[0]>>5 != 5 {
		return nil, errors.New("SSC value must be a map")
	}
	var values map[cbor.ByteString]cbor.RawMessage
	_, err := cbor.Decode(data, &values)
	return values, err
}

func sscBytes(data []byte) error {
	if len(data) == 0 || data[0]>>5 != 2 {
		return errors.New("SSC value must be a byte string")
	}
	var value []byte
	_, err := cbor.Decode(data, &value)
	return err
}

func validateInnerShares(data []byte, stakeholderKeys bool) error {
	values, err := sscMap(data)
	if err != nil {
		return err
	}
	for key, value := range values {
		if stakeholderKeys && len(key.Bytes()) != 28 {
			return fmt.Errorf(
				"SSC inner stakeholder key length: got %d want 28",
				len(key.Bytes()),
			)
		}
		shares, err := sscArray(value, -1)
		if err != nil {
			return err
		}
		for _, share := range shares {
			if err := sscBytes(share); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSignedCommitment(data []byte) error {
	items, err := sscArray(data, 3)
	if err != nil {
		return err
	}
	for _, idx := range []int{0, 2} {
		if err := sscBytes(items[idx]); err != nil {
			return fmt.Errorf("signed commitment field %d: %w", idx, err)
		}
	}
	commitment, err := sscArray(items[1], 2)
	if err != nil {
		return err
	}
	if err := validateInnerShares(commitment[0], false); err != nil {
		return err
	}
	proof, err := sscArray(commitment[1], 4)
	if err != nil {
		return err
	}
	for idx := range 3 {
		if err := sscBytes(proof[idx]); err != nil {
			return fmt.Errorf("secret proof field %d: %w", idx, err)
		}
	}
	commitments, err := sscArray(proof[3], -1)
	if err != nil {
		return err
	}
	for _, value := range commitments {
		if err := sscBytes(value); err != nil {
			return err
		}
	}
	return nil
}

func validateVssCertificate(data []byte) error {
	items, err := sscArray(data, 4)
	if err != nil {
		return err
	}
	for _, idx := range []int{0, 2, 3} {
		if err := sscBytes(items[idx]); err != nil {
			return fmt.Errorf("VSS certificate field %d: %w", idx, err)
		}
	}
	var epoch uint64
	if len(items[1]) == 0 || items[1][0]>>5 != 0 {
		return errors.New("VSS certificate epoch must be an unsigned integer")
	}
	_, err = cbor.Decode(items[1], &epoch)
	return err
}
