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

package ledger

import (
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

// StateProvider is the read-only surface of ledger state that ouroboros-mock
// exercises during transaction validation and protocol operations. It composes
// the seven gouroboros state interfaces into a single, injectable contract.
//
// The default implementation is MockLedgerState, built via LedgerStateBuilder.
// Custom backends (e.g. a real database or Dingo's ledger subsystem) implement
// this interface so that conformance tests run mocked network protocols against
// real persistence instead of in-memory state.
//
// Example (custom backend):
//
//	type MyBackend struct { db *mydb.DB }
//
//	func (b *MyBackend) UtxoById(id lcommon.TransactionInput) (lcommon.Utxo, error) {
//	    return b.db.GetUtxo(id)
//	}
//	// ... implement remaining six interface methods ...
//
//	var _ ledger.StateProvider = (*MyBackend)(nil) // compile-time check
type StateProvider interface {
	lcommon.LedgerState
	lcommon.UtxoState
	lcommon.CertState
	lcommon.SlotState
	lcommon.PoolState
	lcommon.RewardState
	lcommon.GovState
}
