// Copyright 2025 Blink Labs Software
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

package blocks

import (
	"github.com/blinklabs-io/gouroboros/ledger"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

// BlockBuilder defines an interface for building mock blocks
type BlockBuilder interface {
	WithSlot(slot uint64) BlockBuilder
	WithBlockNumber(number uint64) BlockBuilder
	WithHash(hash []byte) BlockBuilder
	WithPrevHash(prevHash []byte) BlockBuilder
	WithTransactions(txs ...TxBuilder) BlockBuilder
	Build() (ledger.Block, error)
	BuildCbor() ([]byte, error)
}

// HeaderBuilder defines an interface for building mock block headers
type HeaderBuilder interface {
	WithSlot(slot uint64) HeaderBuilder
	WithBlockNumber(number uint64) HeaderBuilder
	WithHash(hash []byte) HeaderBuilder
	WithPrevHash(prevHash []byte) HeaderBuilder
	Build() (ledger.BlockHeader, error)
	BuildCbor() ([]byte, error)
}

// TxBuilder defines an interface for building mock transactions
type TxBuilder interface {
	// Build constructs a transaction from the builder state
	Build() (lcommon.Transaction, error)
}
