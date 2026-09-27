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
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/fixtures"
)

// Every Shelley-and-later era CDDL types the header signature as
// kes_signature = bytes .size 448, and the Shelley-through-Conway block's
// auxiliary_data_set as a map, empty for these generated blocks.
func TestGeneratedBlocksUseEraCDDLShapes(t *testing.T) {
	for _, test := range []struct {
		name     string
		generate func() ([]ledger.Block, error)
	}{
		{"Shelley", func() ([]ledger.Block, error) {
			return fixtures.GenerateShelleyChain(1, common.Blake2b256{}, 1, 1, 2)
		}},
		{"Allegra", func() ([]ledger.Block, error) {
			return fixtures.GenerateAllegraChain(1, common.Blake2b256{}, 1, 1, 2)
		}},
		{"Mary", func() ([]ledger.Block, error) {
			return fixtures.GenerateMaryChain(1, common.Blake2b256{}, 1, 1, 2)
		}},
		{"Alonzo", func() ([]ledger.Block, error) {
			return fixtures.GenerateAlonzoChain(1, common.Blake2b256{}, 1, 1, 2)
		}},
		{"Babbage", func() ([]ledger.Block, error) {
			return fixtures.GenerateBabbageChain(1, common.Blake2b256{}, 1, 1, 2)
		}},
		{"Conway", func() ([]ledger.Block, error) {
			return fixtures.GenerateConwayChain(1, common.Blake2b256{}, 1, 1, 2)
		}},
		{"ConwayWithTransactions", func() ([]ledger.Block, error) {
			return fixtures.GenerateConwayChainWithTransactions(
				1, common.Blake2b256{}, 1, 1, 2,
			)
		}},
		{"Dijkstra", func() ([]ledger.Block, error) {
			return fixtures.GenerateDijkstraChain(1, common.Blake2b256{}, 1, 1, 2)
		}},
		{"ConwayToDijkstra", func() ([]ledger.Block, error) {
			return fixtures.GenerateConwayToDijkstraChain(
				1, common.Blake2b256{}, 1, 1, 2, 2,
			)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			blocks, err := test.generate()
			if err != nil {
				t.Fatalf("generate chain: %s", err)
			}
			if len(blocks) == 0 {
				t.Fatal("generator returned no blocks")
			}
			for i, block := range blocks {
				var blockFields []cbor.RawMessage
				if _, err := cbor.Decode(block.Cbor(), &blockFields); err != nil {
					t.Fatalf("decode block %d: %s", i, err)
				}
				var header []cbor.RawMessage
				if _, err := cbor.Decode(blockFields[0], &header); err != nil {
					t.Fatalf("decode block %d header: %s", i, err)
				}
				if len(header) != 2 {
					t.Fatalf("block %d header has %d elements, want 2", i, len(header))
				}
				var signature []byte
				if _, err := cbor.Decode(header[1], &signature); err != nil {
					t.Fatalf("decode block %d signature: %s", i, err)
				}
				if len(signature) != 448 {
					t.Fatalf(
						"block %d KES signature width = %d, want 448",
						i,
						len(signature),
					)
				}
				if block.Type() == ledger.BlockTypeDijkstra {
					continue
				}
				if len(blockFields) < 4 ||
					!bytes.Equal(blockFields[3], []byte{0xa0}) {
					t.Fatalf(
						"block %d auxiliary_data_set = %x, want empty map",
						i,
						blockFields[3],
					)
				}
			}
		})
	}
}
