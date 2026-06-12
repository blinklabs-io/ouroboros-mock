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

// stubBackend is a ledger.StateProvider that delegates all reads to an inner
// provider supplied at construction time.  This lets GetStateProvider() return
// the custom backend while the conformance vectors still pass.
//
// A real Dingo integration replaces each delegating call with an actual
// database read:
//
//	func (s *stubBackend) UtxoById(id common.TransactionInput) (common.Utxo, error) {
//	    return s.db.GetUtxo(id)   // ← your real database call
//	}
type stubBackend struct {
	// getInner returns the current in-memory state.  Calling it on every
	// method ensures the backend sees state that was written by ApplyTransaction.
	// Replace the delegation with your own database reads to exercise real persistence.
	getInner func() ledger.StateProvider
}

// Verify the stub satisfies the interface at compile time.
var _ ledger.StateProvider = (*stubBackend)(nil)

func (s *stubBackend) NetworkId() uint { return s.getInner().NetworkId() }

func (s *stubBackend) UtxoById(
	id common.TransactionInput,
) (common.Utxo, error) {
	return s.getInner().UtxoById(id)
}

func (s *stubBackend) StakeRegistration(
	hash []byte,
) ([]common.StakeRegistrationCertificate, error) {
	return s.getInner().StakeRegistration(hash)
}

func (s *stubBackend) IsStakeCredentialRegistered(c common.Credential) bool {
	return s.getInner().IsStakeCredentialRegistered(c)
}

func (s *stubBackend) SlotToTime(slot uint64) (time.Time, error) {
	return s.getInner().SlotToTime(slot)
}

func (s *stubBackend) TimeToSlot(t time.Time) (uint64, error) {
	return s.getInner().TimeToSlot(t)
}

func (s *stubBackend) PoolCurrentState(
	id common.PoolKeyHash,
) (*common.PoolRegistrationCertificate, *uint64, error) {
	return s.getInner().PoolCurrentState(id)
}

func (s *stubBackend) IsPoolRegistered(id common.PoolKeyHash) bool {
	return s.getInner().IsPoolRegistered(id)
}

func (s *stubBackend) IsVrfKeyInUse(
	vrf common.Blake2b256,
) (bool, common.PoolKeyHash, error) {
	return s.getInner().IsVrfKeyInUse(vrf)
}

func (s *stubBackend) CalculateRewards(
	pots common.AdaPots,
	snapshot common.RewardSnapshot,
	params common.RewardParameters,
) (*common.RewardCalculationResult, error) {
	return s.getInner().CalculateRewards(pots, snapshot, params)
}

func (s *stubBackend) GetAdaPots() common.AdaPots { return s.getInner().GetAdaPots() }

func (s *stubBackend) UpdateAdaPots(p common.AdaPots) error {
	return s.getInner().UpdateAdaPots(p)
}

func (s *stubBackend) GetRewardSnapshot(epoch uint64) (common.RewardSnapshot, error) {
	return s.getInner().GetRewardSnapshot(epoch)
}

func (s *stubBackend) IsRewardAccountRegistered(c common.Credential) bool {
	return s.getInner().IsRewardAccountRegistered(c)
}

func (s *stubBackend) RewardAccountBalance(
	c common.Credential,
) (*uint64, error) {
	return s.getInner().RewardAccountBalance(c)
}

func (s *stubBackend) CommitteeMember(
	hash common.Blake2b224,
) (*common.CommitteeMember, error) {
	return s.getInner().CommitteeMember(hash)
}

func (s *stubBackend) CommitteeMembers() ([]common.CommitteeMember, error) {
	return s.getInner().CommitteeMembers()
}

func (s *stubBackend) DRepRegistration(
	hash common.Blake2b224,
) (*common.DRepRegistration, error) {
	return s.getInner().DRepRegistration(hash)
}

func (s *stubBackend) DRepRegistrations() ([]common.DRepRegistration, error) {
	return s.getInner().DRepRegistrations()
}

func (s *stubBackend) Constitution() (*common.Constitution, error) {
	return s.getInner().Constitution()
}

func (s *stubBackend) TreasuryValue() (uint64, error) { return s.getInner().TreasuryValue() }

func (s *stubBackend) GovActionById(
	id common.GovActionId,
) (*common.GovActionState, error) {
	return s.getInner().GovActionById(id)
}

func (s *stubBackend) GovActionExists(id common.GovActionId) bool {
	return s.getInner().GovActionExists(id)
}

func (s *stubBackend) CostModels() map[common.PlutusLanguage]common.CostModel {
	return s.getInner().CostModels()
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

func newCustomStateManager() *customStateManager {
	m := &customStateManager{
		inner: conformance.NewMockStateManager(),
	}
	// The stub delegates every read to the inner mock's current state so that
	// conformance vectors pass.  Swap the delegation for real database calls to
	// exercise actual persistence instead.
	m.backend = &stubBackend{getInner: m.inner.GetStateProvider}
	return m
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
// validation queries.  Replace stubBackend with your real database to exercise
// actual persistence instead of the in-memory mock.
func (m *customStateManager) GetStateProvider() conformance.StateProvider {
	return m.backend
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

// TestConformanceWithCustomBackend shows how to wire a custom StateManager into
// the conformance harness.  All reads go through the injected backend
// (stubBackend here); a real Dingo integration replaces the delegation inside
// stubBackend with actual database calls.
func TestConformanceWithCustomBackend(t *testing.T) {
	sm := newCustomStateManager()

	h := conformance.NewHarness(sm, conformance.HarnessConfig{
		TestdataRoot: "testdata",
	})
	h.RunAllVectors(t)
}
