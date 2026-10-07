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
	"os"
	"path/filepath"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
	"github.com/stretchr/testify/require"
)

func TestConsensusDijkstraBlockExecutionValidatesFullBody(t *testing.T) {
	t.Parallel()
	block, err := fixtures.NewDijkstraBlockBuilder().Build()
	require.NoError(t, err)
	encode := func(value any) []byte {
		result, err := cbor.Encode(value)
		require.NoError(t, err)
		return result
	}
	header := block.Header().Cbor()
	headerWire := encode([]any{
		uint(dijkstra.BlockHeaderTypeDijkstra),
		cbor.Tag{Number: 24, Content: header},
	})
	for _, tc := range []struct {
		name    string
		payload []byte
		valid   bool
	}{
		{"healthy", block.Cbor(), true},
		{"null_payload", []byte{0xf6}, false},
		{"missing_body", encode([]any{cbor.RawMessage(header)}), false},
		{"corrupt_body", encode([]any{cbor.RawMessage(header), uint(1)}), false},
		{"truncated_body", block.Cbor()[:len(block.Cbor())-1], false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			relDir := "ouroboros-consensus/generated"
			dir := filepath.Join(root, relDir)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			wrapped := append([]byte{0x82, 0x08}, tc.payload...)
			blockWire := encode(cbor.Tag{Number: 24, Content: wrapped})
			require.NoError(t, os.WriteFile(
				filepath.Join(dir, "Block_Dijkstra"), blockWire, 0o600,
			))
			require.NoError(t, os.WriteFile(
				filepath.Join(dir, "Header_Dijkstra"), headerWire, 0o600,
			))
			manifest := relDir + "/Block_Dijkstra\n" +
				relDir + "/Header_Dijkstra\n"
			require.NoError(t, os.WriteFile(
				filepath.Join(root, "manifest.txt"), []byte(manifest), 0o600,
			))
			harness := fixtures.NewHarness(fixtures.HarnessConfig{FixturesRoot: root})
			result, err := harness.ExecuteFixture(relDir + "/Block_Dijkstra")
			require.NoError(t, err)
			require.Equal(t, tc.valid, result.Success)
			if tc.valid {
				require.NoError(t, result.Error)
			} else {
				require.Error(t, result.Error)
			}
			result, err = harness.ExecuteFixture(relDir + "/Header_Dijkstra")
			require.NoError(t, err)
			require.Equal(t, tc.valid, result.Success)
			if tc.valid {
				require.NoError(t, result.Error)
			} else {
				require.Error(t, result.Error)
			}
		})
	}
}

func TestConsensusDijkstraPraosExampleUsesProducerCodec(t *testing.T) {
	t.Parallel()
	const relPath = "ouroboros-consensus/ouroboros-consensus-cardano/" +
		"golden/cardano/CardanoNodeToNodeVersion2/Header_Dijkstra"
	fixture, err := fixtures.NewHarness(fixtures.HarnessConfig{}).
		Fixture(relPath)
	require.NoError(t, err)
	original, err := fixture.Read()
	require.NoError(t, err)
	wrapped, err := fixture.ConsensusHeader()
	require.NoError(t, err)
	require.Equal(t, uint(7), wrapped.Era)
	_, err = fixture.DecodeLedgerHeader()
	require.ErrorContains(t, err, "expected exactly 12 fields, got 10")

	header := []cbor.RawMessage{}
	_, err = cbor.Decode(wrapped.HeaderCbor(), &header)
	require.NoError(t, err)
	require.Len(t, header, 2)
	var fields []cbor.RawMessage
	_, err = cbor.Decode(header[0], &fields)
	require.NoError(t, err)
	require.Len(t, fields, 10)
	fields[2] = encodeFixtureTestCBOR(t, bytes.Repeat([]byte{0xab}, 31))
	corruptHeader := encodeFixtureTestCBOR(t, []any{
		cbor.RawMessage(encodeFixtureTestCBOR(t, fields)),
		header[1],
	})
	corrupt := encodeFixtureTestCBOR(t, []any{
		uint(7), cbor.Tag{Number: 24, Content: corruptHeader},
	})
	currentBlock, err := fixtures.NewDijkstraBlockBuilder().Build()
	require.NoError(t, err)
	currentHeader := encodeFixtureTestCBOR(t, []any{
		uint(7), cbor.Tag{Number: 24, Content: currentBlock.Header().Cbor()},
	})

	for _, tc := range []struct {
		name  string
		wire  []byte
		valid bool
	}{
		{"producer_example", original, true},
		{"invalid_hash", corrupt, false},
		{"different_header_family", currentHeader, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, relPath)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
			require.NoError(t, os.WriteFile(path, tc.wire, 0o600))
			require.NoError(t, os.WriteFile(
				filepath.Join(root, "manifest.txt"),
				[]byte(relPath+"\n"), 0o600,
			))
			harness := fixtures.NewHarness(fixtures.HarnessConfig{
				FixturesRoot: root,
			})
			result, err := harness.ExecuteFixture(relPath)
			require.NoError(t, err)
			require.Equal(t, tc.valid, result.Success)
			require.False(t, result.ExpectedStrictDecodeRejection)
			if tc.valid {
				require.NoError(t, result.Error)
			} else {
				require.Error(t, result.Error)
			}
		})
	}
}

func encodeFixtureTestCBOR(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := cbor.Encode(value)
	require.NoError(t, err)
	return encoded
}
