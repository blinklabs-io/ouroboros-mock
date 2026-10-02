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
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
)

const sscRoot = "cardano-ledger/eras/byron/ledger/impl/golden/cbor/ssc/"

func TestSSCFixturesDecodeAndValidate(t *testing.T) {
	harness := fixtures.NewHarness(fixtures.HarnessConfig{})
	for _, test := range []struct {
		name  string
		size  int
		hash  string
		cases int
	}{
		{
			name:  "CommitmentsMap",
			size:  13637,
			hash:  "6249942b814658ca6c17ff9af6c99e36fde449dec402b4e8a39ce8d04ad35586",
			cases: 1,
		},
		{
			name:  "OpeningsMap",
			size:  68,
			hash:  "4454f06699a385585b08cd37a1c86ba14b622b403bb84f2a9e634bf57abd80b7",
			cases: 1,
		},
		{
			name:  "SharesMap",
			size:  564,
			hash:  "ea89eb0f75b873346b4b8d12a98e1790a2d2866a615909422107f7c677f4bbc5",
			cases: 1,
		},
		{
			name:  "VssCertificatesMap",
			size:  692,
			hash:  "64bb4877b8476fed9004c0ddb7fec2b07eae6427ea0cca7de1bd3ee1414d43de",
			cases: 4,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, err := harness.Fixture(sscRoot + test.name)
			require.NoError(t, err)
			require.Equal(t, fixtures.RepoCardanoLedger, fixture.Repo)
			require.Equal(t, fixtures.KindSSC, fixture.Kind)
			require.Equal(t, fixtures.FormatHexDump, fixture.Format)
			require.Equal(t, "byron", fixture.Era)
			raw, err := fixture.DecodeHex()
			require.NoError(t, err)
			require.Len(t, raw, test.size)
			require.Equal(t, test.hash, fmt.Sprintf("%x", sha256.Sum256(raw)))
			result, err := harness.ExecuteFixture(fixture.RelPath)
			require.NoError(t, err)
			require.NoError(t, result.Error)
			require.True(t, result.Success)
			require.Equal(t, test.cases, result.CaseCount)
		})
	}
}

func TestAnnotatedHexRejectsCorruptOffsetsAndChunks(t *testing.T) {
	for _, test := range []struct {
		name string
		text string
		err  string
	}{
		{
			name: "nonzero start",
			text: "01: 00",
			err:  "does not match byte position",
		},
		{
			name: "offset gap",
			text: "00: 000102030405060708090a0b0c0d0e0f\n11: 10",
			err:  "does not match byte position",
		},
		{
			name: "invalid offset",
			text: "zz: 00",
			err:  "invalid hex dump offset",
		},
		{
			name: "missing offset",
			text: "000102",
			err:  "missing an offset",
		},
		{
			name: "invalid hex",
			text: "00: gg",
			err:  "invalid hex dump data",
		},
		{
			name: "empty chunk",
			text: "00:",
			err:  "invalid hex dump chunk length",
		},
		{
			name: "oversized chunk",
			text: "00: 000102030405060708090a0b0c0d0e0f10",
			err:  "invalid hex dump chunk length",
		},
		{
			name: "short interior chunk",
			text: "00: 00\n01: 01",
			err:  "invalid hex dump chunk length",
		},
		{
			name: "trailing garbage",
			text: "00: 000102030405060708090a0b0c0d0e0f\ntrailing",
			err:  "missing an offset",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			harness := fixtureHarness(
				t,
				map[string][]byte{sscRoot + "OpeningsMap": []byte(test.text)},
			)
			fixture, err := harness.Fixture(sscRoot + "OpeningsMap")
			require.NoError(t, err)
			_, err = fixture.DecodeHex()
			require.ErrorContains(t, err, test.err)
			require.ErrorContains(t, err, fixture.RelPath)
		})
	}
}

func TestSSCExecutionRejectsMalformedWireShapes(t *testing.T) {
	good := fixtures.NewHarness(fixtures.HarnessConfig{})
	load := func(name string) []byte {
		f, err := good.Fixture(sscRoot + name)
		require.NoError(t, err)
		raw, err := f.DecodeHex()
		require.NoError(t, err)
		return raw
	}
	encode := func(value any) []byte {
		data, err := cbor.Encode(value)
		require.NoError(t, err)
		return data
	}
	var tag cbor.RawTag
	_, err := cbor.Decode(load("VssCertificatesMap"), &tag)
	require.NoError(t, err)
	certificates := make([]cbor.RawMessage, 0)
	_, err = cbor.Decode(tag.Content, &certificates)
	require.NoError(t, err)
	require.Len(t, certificates, 4)
	firstCertificate := make([]cbor.RawMessage, 0)
	_, err = cbor.Decode(certificates[0], &firstCertificate)
	require.NoError(t, err)
	require.Len(t, firstCertificate, 4)
	badEpoch := append([]cbor.RawMessage(nil), firstCertificate...)
	badEpoch[1] = encode(nil)
	badCertificate := append([]cbor.RawMessage(nil), firstCertificate...)
	badCertificate[0] = encode(nil)
	set := func(item any) []byte {
		return encode(cbor.Tag{Number: cbor.CborTagSet, Content: []any{item}})
	}
	stakeholder := cbor.NewByteString(make([]byte, 28))
	shortKey := cbor.NewByteString([]byte{0})
	proof := []any{[]byte{}, []byte{}, []byte{}, []any{}}
	commitment := []any{map[cbor.ByteString]any{}, proof}
	signed := []any{[]byte{}, commitment, []byte{}}
	opening := func(value any) []byte {
		return encode(map[cbor.ByteString]any{stakeholder: value})
	}
	shares := func(key cbor.ByteString, value any) []byte {
		return opening(map[cbor.ByteString]any{key: value})
	}
	for _, test := range []struct {
		name    string
		fixture string
		data    []byte
		err     string
	}{
		{
			name:    "untagged commitments",
			fixture: "CommitmentsMap",
			data:    encode([]any{}),
			err:     "SSC set",
		},
		{
			name:    "wrong set tag",
			fixture: "CommitmentsMap",
			data:    encode(cbor.Tag{Number: 259, Content: []any{}}),
			err:     "SSC set tag",
		},
		{
			name:    "commitment width",
			fixture: "CommitmentsMap",
			data:    set(append(signed, nil)),
			err:     "array width",
		},
		{
			name:    "commitment bytes",
			fixture: "CommitmentsMap",
			data:    set([]any{nil, commitment, []byte{}}),
			err:     "byte string",
		},
		{
			name:    "inner commitment width",
			fixture: "CommitmentsMap",
			data:    set([]any{[]byte{}, append(commitment, nil), []byte{}}),
			err:     "array width",
		},
		{
			name:    "secret proof width",
			fixture: "CommitmentsMap",
			data:    set([]any{[]byte{}, []any{map[cbor.ByteString]any{}, append(proof, nil)}, []byte{}}),
			err:     "array width",
		},
		{
			name:    "certificate width",
			fixture: "VssCertificatesMap",
			data:    set(append(append([]cbor.RawMessage(nil), firstCertificate...), encode(nil))),
			err:     "array width",
		},
		{
			name:    "certificate epoch",
			fixture: "VssCertificatesMap",
			data:    set(badEpoch),
			err:     "unsigned integer",
		},
		{
			name:    "certificate bytes",
			fixture: "VssCertificatesMap",
			data:    set(badCertificate),
			err:     "byte string",
		},
		{
			name:    "openings container",
			fixture: "OpeningsMap",
			data:    encode([]any{}),
			err:     "must be a map",
		},
		{
			name:    "null openings",
			fixture: "OpeningsMap",
			data:    encode(nil),
			err:     "must be a map",
		},
		{
			name:    "stakeholder length",
			fixture: "OpeningsMap",
			data:    encode(map[cbor.ByteString]any{shortKey: []byte{0}}),
			err:     "stakeholder key length",
		},
		{
			name:    "opening bytes",
			fixture: "OpeningsMap",
			data:    opening(nil),
			err:     "byte string",
		},
		{
			name:    "inner shares container",
			fixture: "SharesMap",
			data:    opening([]any{}),
			err:     "must be a map",
		},
		{
			name:    "null inner shares",
			fixture: "SharesMap",
			data:    opening(nil),
			err:     "must be a map",
		},
		{
			name:    "inner stakeholder length",
			fixture: "SharesMap",
			data:    shares(shortKey, []any{}),
			err:     "inner stakeholder key length",
		},
		{
			name:    "share list",
			fixture: "SharesMap",
			data:    shares(stakeholder, nil),
			err:     "must be an array",
		},
		{
			name:    "share bytes",
			fixture: "SharesMap",
			data:    shares(stakeholder, []any{nil}),
			err:     "byte string",
		},
		{
			name:    "trailing CBOR",
			fixture: "OpeningsMap",
			data:    append(load("OpeningsMap"), 0),
			err:     "trailing data",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			harness := fixtureHarness(
				t,
				map[string][]byte{
					sscRoot + test.fixture: annotatedHex(test.data),
				},
			)
			result, err := harness.ExecuteFixture(sscRoot + test.fixture)
			require.NoError(t, err)
			require.False(t, result.Success)
			require.ErrorContains(t, result.Error, test.err)
			require.ErrorContains(t, result.Error, "kind=ssc")
			require.ErrorContains(t, result.Error, `era="byron"`)
		})
	}
}

func annotatedHex(raw []byte) []byte {
	var text strings.Builder
	for offset := 0; offset < len(raw); offset += 16 {
		fmt.Fprintf(
			&text,
			"%04x: %x\n",
			offset,
			raw[offset:min(offset+16, len(raw))],
		)
	}
	return []byte(text.String())
}

func fixtureHarness(t *testing.T, files map[string][]byte) *fixtures.Harness {
	t.Helper()
	root := t.TempDir()
	var manifest strings.Builder
	for name, data := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, data, 0o600))
		fmt.Fprintln(&manifest, "./"+name)
	}
	require.NoError(
		t,
		os.WriteFile(
			filepath.Join(root, "manifest.txt"),
			[]byte(manifest.String()),
			0o600,
		),
	)
	return fixtures.NewHarness(fixtures.HarnessConfig{FixturesRoot: root})
}

func TestEmptySSCFixturesReportZeroCases(t *testing.T) {
	for _, name := range []string{"CommitmentsMap", "VssCertificatesMap", "OpeningsMap", "SharesMap"} {
		t.Run(name, func(t *testing.T) {
			value := any(map[cbor.ByteString]any{})
			if name == "CommitmentsMap" || name == "VssCertificatesMap" {
				value = cbor.Tag{Number: cbor.CborTagSet, Content: []any{}}
			}
			data, err := cbor.Encode(value)
			require.NoError(t, err)
			text := fmt.Sprintf("00: %x", data)
			harness := fixtureHarness(t, map[string][]byte{sscRoot + name: []byte(text)})
			result, err := harness.ExecuteFixture(sscRoot + name)
			require.NoError(t, err)
			require.NoError(t, result.Error)
			require.True(t, result.Success)
			require.Zero(t, result.CaseCount)
		})
	}
}
