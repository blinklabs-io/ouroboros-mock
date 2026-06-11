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

package conformance_test

// This file demonstrates how an external project (e.g. Dingo) can plug its
// real database and ledger subsystems into the conformance test harness.
//
// Architecture
// ============
//
//   conformance.Harness
//       │
//       │ StateManager interface
//       ▼
//   CustomStateManager          ← your implementation
//       │ GetStateProvider()
//       ▼
//   CustomBackend               ← wraps your real database
//       │ implements ledger.StateProvider
//       ▼
//   real DB / ledger code
//
// The mocked layer stays as-is (network protocols: handshake, chainsync,
// blockfetch, etc.).  Only the persistence layer changes.
//
// Step-by-step
// ============
//
// 1.  Implement ledger.StateProvider with your real database reads.
// 2.  Implement conformance.StateManager with your real transaction
//     processing / epoch boundary logic.
// 3.  Pass your StateManager to conformance.NewHarness.
//
// The example below uses a stub that delegates everything to the built-in
// MockStateManager so it compiles and runs without a real database.  A real
// integration would replace the stub calls with actual database operations.

import (
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/conformance"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
)

// stubBackend is a trivial ledger.StateProvider that returns empty/zero values
// for every method.  Replace this with your real database implementation.
type stubBackend struct{}

// Verify the stub satisfies the interface at compile time.
var _ ledger.StateProvider = (*stubBackend)(nil)

func (s *stubBackend) NetworkId() uint { return 0 }

func (s *stubBackend) UtxoById(
	_ common.TransactionInput,
) (common.Utxo, error) {
	return common.Utxo{}, ledger.ErrNotFound
}

func (s *stubBackend) StakeRegistration(
	_ []byte,
) ([]common.StakeRegistrationCertificate, error) {
	return nil, nil
}

func (s *stubBackend) IsStakeCredentialRegistered(_ common.Credential) bool {
	return false
}

func (s *stubBackend) SlotToTime(_ uint64) (time.Time, error) {
	return time.Time{}, nil
}

func (s *stubBackend) TimeToSlot(_ time.Time) (uint64, error) {
	return 0, nil
}

func (s *stubBackend) PoolCurrentState(
	_ common.PoolKeyHash,
) (*common.PoolRegistrationCertificate, *uint64, error) {
	return nil, nil, nil
}

func (s *stubBackend) IsPoolRegistered(_ common.PoolKeyHash) bool { return false }

func (s *stubBackend) IsVrfKeyInUse(
	_ common.Blake2b256,
) (bool, common.PoolKeyHash, error) {
	return false, common.PoolKeyHash{}, nil
}

func (s *stubBackend) CalculateRewards(
	pots common.AdaPots,
	snapshot common.RewardSnapshot,
	params common.RewardParameters,
) (*common.RewardCalculationResult, error) {
	return common.CalculateRewards(pots, snapshot, params)
}

func (s *stubBackend) GetAdaPots() common.AdaPots { return common.AdaPots{} }

func (s *stubBackend) UpdateAdaPots(_ common.AdaPots) error { return nil }

func (s *stubBackend) GetRewardSnapshot(_ uint64) (common.RewardSnapshot, error) {
	return common.RewardSnapshot{}, nil
}

func (s *stubBackend) IsRewardAccountRegistered(_ common.Credential) bool {
	return false
}

func (s *stubBackend) RewardAccountBalance(
	_ common.Credential,
) (*uint64, error) {
	return nil, nil
}

func (s *stubBackend) CommitteeMember(
	_ common.Blake2b224,
) (*common.CommitteeMember, error) {
	return nil, nil
}

func (s *stubBackend) CommitteeMembers() ([]common.CommitteeMember, error) {
	return nil, nil
}

func (s *stubBackend) DRepRegistration(
	_ common.Blake2b224,
) (*common.DRepRegistration, error) {
	return nil, nil
}

func (s *stubBackend) DRepRegistrations() ([]common.DRepRegistration, error) {
	return nil, nil
}

func (s *stubBackend) Constitution() (*common.Constitution, error) { return nil, nil }

func (s *stubBackend) TreasuryValue() (uint64, error) { return 0, nil }

func (s *stubBackend) GovActionById(
	_ common.GovActionId,
) (*common.GovActionState, error) {
	return nil, nil
}

func (s *stubBackend) GovActionExists(_ common.GovActionId) bool { return false }

func (s *stubBackend) CostModels() map[common.PlutusLanguage]common.CostModel {
	return nil
}

// customStateManager is a conformance.StateManager that delegates state reads
// to a ledger.StateProvider (here: stubBackend) and state mutations to the
// built-in MockStateManager.
//
// A real Dingo integration would replace MockStateManager with Dingo's own
// transaction application and epoch-boundary logic, and replace stubBackend
// with a read-path that queries Dingo's real database.
type customStateManager struct {
	inner   *conformance.MockStateManager
	backend ledger.StateProvider
}

var _ conformance.StateManager = (*customStateManager)(nil)

func newCustomStateManager(backend ledger.StateProvider) *customStateManager {
	return &customStateManager{
		inner:   conformance.NewMockStateManager(),
		backend: backend,
	}
}

func (m *customStateManager) LoadInitialState(
	state *conformance.ParsedInitialState,
	pp common.ProtocolParameters,
) error {
	// In a real integration: hydrate your database from state and pp.
	// Here we just forward to the in-memory manager so the test runs.
	return m.inner.LoadInitialState(state, pp)
}

func (m *customStateManager) ApplyTransaction(
	tx common.Transaction,
	slot uint64,
) error {
	// In a real integration: call your ledger's transaction application path.
	return m.inner.ApplyTransaction(tx, slot)
}

func (m *customStateManager) ProcessEpochBoundary(newEpoch uint64) error {
	// In a real integration: call your ledger's epoch-boundary logic.
	return m.inner.ProcessEpochBoundary(newEpoch)
}

// GetStateProvider returns the backend that the harness uses for read-only
// validation queries.  Swap stubBackend with your real database here.
func (m *customStateManager) GetStateProvider() conformance.StateProvider {
	// To exercise real Dingo persistence, return your real backend instead:
	//   return m.backend
	return m.inner.GetStateProvider()
}

func (m *customStateManager) GetGovernanceState() *conformance.GovernanceState {
	return m.inner.GetGovernanceState()
}

func (m *customStateManager) SetRewardBalances(
	balances map[common.Blake2b224]uint64,
) {
	m.inner.SetRewardBalances(balances)
}

func (m *customStateManager) GetProtocolParameters() common.ProtocolParameters {
	return m.inner.GetProtocolParameters()
}

func (m *customStateManager) Reset() error {
	return m.inner.Reset()
}

// TestConformanceWithCustomBackend shows how to wire a custom StateManager
// (backed by a real database) into the conformance harness.  All existing
// vectors pass because customStateManager still delegates to MockStateManager;
// switching GetStateProvider() to return m.backend makes tests exercise real
// Dingo persistence instead.
func TestConformanceWithCustomBackend(t *testing.T) {
	backend := &stubBackend{}
	sm := newCustomStateManager(backend)

	h := conformance.NewHarness(sm, conformance.HarnessConfig{
		TestdataRoot: "testdata",
	})
	h.RunAllVectors(t)
}
