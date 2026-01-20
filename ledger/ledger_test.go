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

package ledger_test

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
)

// Test helper to create a sample transaction ID
func sampleTxId() []byte {
	return bytes.Repeat([]byte{0xab}, 32)
}

// Test helper to create a sample address (testnet base address)
// Using a valid testnet address for testing purposes
func sampleAddress() string {
	return "addr_test1qz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp"
}

// Test helper to create a sample policy ID
func samplePolicyId() []byte {
	return bytes.Repeat([]byte{0xcd}, 28)
}

// Test helper to create a sample key hash
func sampleKeyHash() []byte {
	return bytes.Repeat([]byte{0xef}, 28)
}

// =============================================================================
// UtxoBuilder Tests
// =============================================================================

func TestNewUtxoBuilder(t *testing.T) {
	builder := ledger.NewUtxoBuilder()
	if builder == nil {
		t.Fatal("NewUtxoBuilder() returned nil")
	}
}

func TestUtxoBuilder_WithTxId(t *testing.T) {
	txId := sampleTxId()
	builder := ledger.NewUtxoBuilder().WithTxId(txId)
	if builder == nil {
		t.Fatal("WithTxId() returned nil")
	}
}

func TestUtxoBuilder_WithIndex(t *testing.T) {
	builder := ledger.NewUtxoBuilder().WithIndex(5)
	if builder == nil {
		t.Fatal("WithIndex() returned nil")
	}
}

func TestUtxoBuilder_WithAddress(t *testing.T) {
	builder := ledger.NewUtxoBuilder().WithAddress(sampleAddress())
	if builder == nil {
		t.Fatal("WithAddress() returned nil")
	}
}

func TestUtxoBuilder_WithLovelace(t *testing.T) {
	builder := ledger.NewUtxoBuilder().WithLovelace(5000000)
	if builder == nil {
		t.Fatal("WithLovelace() returned nil")
	}
}

func TestUtxoBuilder_WithAssets(t *testing.T) {
	assets := []ledger.Asset{
		{
			PolicyId:  samplePolicyId(),
			AssetName: []byte("TestToken"),
			Amount:    1000,
		},
	}
	builder := ledger.NewUtxoBuilder().WithAssets(assets...)
	if builder == nil {
		t.Fatal("WithAssets() returned nil")
	}
}

func TestUtxoBuilder_Build_Success(t *testing.T) {
	txId := sampleTxId()
	addr := sampleAddress()
	lovelace := uint64(5000000)
	index := uint32(0)

	utxo, err := ledger.NewUtxoBuilder().
		WithTxId(txId).
		WithIndex(index).
		WithAddress(addr).
		WithLovelace(lovelace).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	// Verify the UTxO input
	if utxo.Id == nil {
		t.Fatal("Built UTxO has nil Id")
	}
	if utxo.Id.Index() != index {
		t.Errorf("Expected index %d, got %d", index, utxo.Id.Index())
	}

	// Verify the UTxO output
	if utxo.Output == nil {
		t.Fatal("Built UTxO has nil Output")
	}
	if utxo.Output.Amount().Cmp(big.NewInt(int64(lovelace))) != 0 {
		t.Errorf("Expected amount %d, got %s", lovelace, utxo.Output.Amount())
	}
}

func TestUtxoBuilder_Build_MissingTxId(t *testing.T) {
	_, err := ledger.NewUtxoBuilder().
		WithAddress(sampleAddress()).
		WithLovelace(5000000).
		Build()

	if err == nil {
		t.Fatal("Build() should return error when TxId is missing")
	}
}

func TestUtxoBuilder_WithMultipleAssets(t *testing.T) {
	assets := []ledger.Asset{
		{
			PolicyId:  samplePolicyId(),
			AssetName: []byte("Token1"),
			Amount:    100,
		},
		{
			PolicyId:  samplePolicyId(),
			AssetName: []byte("Token2"),
			Amount:    200,
		},
	}

	utxo, err := ledger.NewUtxoBuilder().
		WithTxId(sampleTxId()).
		WithIndex(0).
		WithAddress(sampleAddress()).
		WithLovelace(2000000).
		WithAssets(assets...).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if utxo.Output.Assets() == nil {
		t.Fatal("Expected assets to be set")
	}
}

func TestUtxoBuilder_ChainedMethods(t *testing.T) {
	// Test that all methods can be chained
	utxo, err := ledger.NewUtxoBuilder().
		WithTxId(sampleTxId()).
		WithIndex(1).
		WithAddress(sampleAddress()).
		WithLovelace(10000000).
		WithAssets(ledger.Asset{
			PolicyId:  samplePolicyId(),
			AssetName: []byte("Test"),
			Amount:    50,
		}).
		WithDatum(nil).
		WithDatumHash(nil).
		WithScriptRef(nil).
		Build()

	if err != nil {
		t.Fatalf("Chained Build() returned error: %v", err)
	}

	if utxo.Id.Index() != 1 {
		t.Errorf("Expected index 1, got %d", utxo.Id.Index())
	}
}

// =============================================================================
// LedgerStateBuilder Tests
// =============================================================================

func TestNewLedgerStateBuilder(t *testing.T) {
	builder := ledger.NewLedgerStateBuilder()
	if builder == nil {
		t.Fatal("NewLedgerStateBuilder() returned nil")
	}
}

func TestLedgerStateBuilder_WithNetworkId(t *testing.T) {
	networkId := uint(1) // mainnet
	state := ledger.NewLedgerStateBuilder().
		WithNetworkId(networkId).
		Build()

	if state.NetworkId() != networkId {
		t.Errorf("Expected NetworkId %d, got %d", networkId, state.NetworkId())
	}
}

func TestLedgerStateBuilder_WithAdaPots(t *testing.T) {
	pots := lcommon.AdaPots{
		Reserves: 10000000000000,
		Treasury: 500000000000,
		Rewards:  100000000000,
	}

	state := ledger.NewLedgerStateBuilder().
		WithAdaPots(pots).
		Build()

	result := state.GetAdaPots()
	if result.Reserves != pots.Reserves {
		t.Errorf("Expected Reserves %d, got %d", pots.Reserves, result.Reserves)
	}
	if result.Treasury != pots.Treasury {
		t.Errorf("Expected Treasury %d, got %d", pots.Treasury, result.Treasury)
	}
	if result.Rewards != pots.Rewards {
		t.Errorf("Expected Rewards %d, got %d", pots.Rewards, result.Rewards)
	}
}

func TestLedgerStateBuilder_WithUtxos(t *testing.T) {
	// Create a UTxO
	utxo, err := ledger.NewUtxoBuilder().
		WithTxId(sampleTxId()).
		WithIndex(0).
		WithAddress(sampleAddress()).
		WithLovelace(5000000).
		Build()
	if err != nil {
		t.Fatalf("Failed to build UTxO: %v", err)
	}

	state := ledger.NewLedgerStateBuilder().
		WithUtxos([]lcommon.Utxo{utxo}).
		Build()

	// Look up the UTxO
	result, err := state.UtxoById(utxo.Id)
	if err != nil {
		t.Fatalf("UtxoById returned error: %v", err)
	}

	if result.Output.Amount().Cmp(utxo.Output.Amount()) != 0 {
		t.Errorf(
			"Expected amount %s, got %s",
			utxo.Output.Amount(),
			result.Output.Amount(),
		)
	}
}

func TestLedgerStateBuilder_WithUtxoById_Callback(t *testing.T) {
	txId := sampleTxId()
	expectedAmount := uint64(7000000)

	callback := func(id lcommon.TransactionInput) (lcommon.Utxo, error) {
		// Return a mock UTxO
		utxo, err := ledger.NewUtxoBuilder().
			WithTxId(txId).
			WithIndex(id.Index()).
			WithAddress(sampleAddress()).
			WithLovelace(expectedAmount).
			Build()
		return utxo, err
	}

	state := ledger.NewLedgerStateBuilder().
		WithUtxoById(callback).
		Build()

	// Create input to look up
	input, _ := ledger.NewTransactionInputBuilder().
		WithTxId(txId).
		WithIndex(0).
		Build()

	result, err := state.UtxoById(input)
	if err != nil {
		t.Fatalf("UtxoById returned error: %v", err)
	}

	if result.Output.Amount().Cmp(big.NewInt(int64(expectedAmount))) != 0 {
		t.Errorf(
			"Expected amount %d, got %s",
			expectedAmount,
			result.Output.Amount(),
		)
	}
}

func TestLedgerStateBuilder_WithSlotToTime(t *testing.T) {
	expectedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	callback := func(slot uint64) (time.Time, error) {
		return expectedTime, nil
	}

	state := ledger.NewLedgerStateBuilder().
		WithSlotToTime(callback).
		Build()

	result, err := state.SlotToTime(12345)
	if err != nil {
		t.Fatalf("SlotToTime returned error: %v", err)
	}

	if !result.Equal(expectedTime) {
		t.Errorf("Expected time %v, got %v", expectedTime, result)
	}
}

func TestLedgerStateBuilder_WithTimeToSlot(t *testing.T) {
	expectedSlot := uint64(54321)

	callback := func(t time.Time) (uint64, error) {
		return expectedSlot, nil
	}

	state := ledger.NewLedgerStateBuilder().
		WithTimeToSlot(callback).
		Build()

	result, err := state.TimeToSlot(time.Now())
	if err != nil {
		t.Fatalf("TimeToSlot returned error: %v", err)
	}

	if result != expectedSlot {
		t.Errorf("Expected slot %d, got %d", expectedSlot, result)
	}
}

func TestLedgerStateBuilder_TreasuryValue(t *testing.T) {
	expectedTreasury := uint64(1000000000000)

	callback := func() (uint64, error) {
		return expectedTreasury, nil
	}

	state := ledger.NewLedgerStateBuilder().
		WithTreasuryValue(callback).
		Build()

	result, err := state.TreasuryValue()
	if err != nil {
		t.Fatalf("TreasuryValue returned error: %v", err)
	}

	if result != expectedTreasury {
		t.Errorf("Expected treasury %d, got %d", expectedTreasury, result)
	}
}

func TestLedgerStateBuilder_TreasuryValue_FromAdaPots(t *testing.T) {
	expectedTreasury := uint64(500000000000)

	state := ledger.NewLedgerStateBuilder().
		WithAdaPots(lcommon.AdaPots{
			Treasury: expectedTreasury,
		}).
		Build()

	result, err := state.TreasuryValue()
	if err != nil {
		t.Fatalf("TreasuryValue returned error: %v", err)
	}

	if result != expectedTreasury {
		t.Errorf("Expected treasury %d, got %d", expectedTreasury, result)
	}
}

func TestLedgerStateBuilder_WithCostModels(t *testing.T) {
	expectedCostModels := map[ledger.PlutusLanguage]ledger.CostModel{
		ledger.PlutusV1: make([]int64, 166),
		ledger.PlutusV2: make([]int64, 175),
	}

	callback := func() map[ledger.PlutusLanguage]ledger.CostModel {
		return expectedCostModels
	}

	state := ledger.NewLedgerStateBuilder().
		WithCostModels(callback).
		Build()

	result := state.CostModels()
	if len(result) != len(expectedCostModels) {
		t.Errorf(
			"Expected %d cost models, got %d",
			len(expectedCostModels),
			len(result),
		)
	}
}

func TestLedgerState_UpdateAdaPots(t *testing.T) {
	state := ledger.NewLedgerStateBuilder().Build()

	newPots := lcommon.AdaPots{
		Reserves: 5000000000000,
		Treasury: 250000000000,
		Rewards:  50000000000,
	}

	err := state.UpdateAdaPots(newPots)
	if err != nil {
		t.Fatalf("UpdateAdaPots returned error: %v", err)
	}

	result := state.GetAdaPots()
	if result.Treasury != newPots.Treasury {
		t.Errorf(
			"Expected Treasury %d, got %d",
			newPots.Treasury,
			result.Treasury,
		)
	}
}

func TestLedgerState_DefaultBehavior(t *testing.T) {
	state := ledger.NewLedgerStateBuilder().Build()

	// Test default UtxoById returns ErrNotFound
	input, _ := ledger.NewTransactionInputBuilder().
		WithTxId(sampleTxId()).
		WithIndex(0).
		Build()
	_, err := state.UtxoById(input)
	if err != ledger.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}

	// Test default StakeRegistration returns empty slice
	certs, err := state.StakeRegistration(sampleKeyHash())
	if err != nil {
		t.Errorf("StakeRegistration should not return error: %v", err)
	}
	if len(certs) != 0 {
		t.Errorf("Expected empty certs, got %d", len(certs))
	}

	// Test default SlotToTime returns zero time
	slotTime, err := state.SlotToTime(0)
	if err != nil {
		t.Errorf("SlotToTime should not return error: %v", err)
	}
	if !slotTime.IsZero() {
		t.Errorf("Expected zero time, got %v", slotTime)
	}

	// Test default CostModels returns empty map
	costModels := state.CostModels()
	if len(costModels) != 0 {
		t.Errorf("Expected empty cost models, got %d", len(costModels))
	}
}

// =============================================================================
// PoolBuilder Tests
// =============================================================================

func TestNewPoolBuilder(t *testing.T) {
	builder := ledger.NewPoolBuilder()
	if builder == nil {
		t.Fatal("NewPoolBuilder() returned nil")
	}
}

func TestPoolBuilder_WithOperator(t *testing.T) {
	builder := ledger.NewPoolBuilder().WithOperator(sampleKeyHash())
	if builder == nil {
		t.Fatal("WithOperator() returned nil")
	}
}

func TestPoolBuilder_WithVrfKeyHash(t *testing.T) {
	vrfHash := bytes.Repeat([]byte{0x12}, 32)
	builder := ledger.NewPoolBuilder().WithVrfKeyHash(vrfHash)
	if builder == nil {
		t.Fatal("WithVrfKeyHash() returned nil")
	}
}

func TestPoolBuilder_WithPledge(t *testing.T) {
	builder := ledger.NewPoolBuilder().WithPledge(100000000000)
	if builder == nil {
		t.Fatal("WithPledge() returned nil")
	}
}

func TestPoolBuilder_WithCost(t *testing.T) {
	builder := ledger.NewPoolBuilder().WithCost(340000000)
	if builder == nil {
		t.Fatal("WithCost() returned nil")
	}
}

func TestPoolBuilder_WithMargin(t *testing.T) {
	builder := ledger.NewPoolBuilder().WithMargin(1, 100)
	if builder == nil {
		t.Fatal("WithMargin() returned nil")
	}
}

func TestPoolBuilder_WithOwners(t *testing.T) {
	owners := [][]byte{
		sampleKeyHash(),
		bytes.Repeat([]byte{0x01}, 28),
	}
	builder := ledger.NewPoolBuilder().WithOwners(owners...)
	if builder == nil {
		t.Fatal("WithOwners() returned nil")
	}
}

func TestPoolBuilder_WithMetadata(t *testing.T) {
	url := "https://example.com/pool.json"
	hash := bytes.Repeat([]byte{0x99}, 32)
	builder := ledger.NewPoolBuilder().WithMetadata(url, hash)
	if builder == nil {
		t.Fatal("WithMetadata() returned nil")
	}
}

func TestPoolBuilder_Build(t *testing.T) {
	operatorHash := sampleKeyHash()
	vrfHash := bytes.Repeat([]byte{0x12}, 32)
	pledge := uint64(100000000000)
	cost := uint64(340000000)

	cert, err := ledger.NewPoolBuilder().
		WithOperator(operatorHash).
		WithVrfKeyHash(vrfHash).
		WithPledge(pledge).
		WithCost(cost).
		WithMargin(1, 100).
		WithOwners(operatorHash).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if cert.Pledge != pledge {
		t.Errorf("Expected pledge %d, got %d", pledge, cert.Pledge)
	}

	if cert.Cost != cost {
		t.Errorf("Expected cost %d, got %d", cost, cert.Cost)
	}

	if len(cert.PoolOwners) != 1 {
		t.Errorf("Expected 1 owner, got %d", len(cert.PoolOwners))
	}
}

// =============================================================================
// AdaPotsBuilder Tests
// =============================================================================

func TestNewAdaPotsBuilder(t *testing.T) {
	builder := ledger.NewAdaPotsBuilder()
	if builder == nil {
		t.Fatal("NewAdaPotsBuilder() returned nil")
	}
}

func TestAdaPotsBuilder_WithReserves(t *testing.T) {
	builder := ledger.NewAdaPotsBuilder().WithReserves(10000000000000)
	if builder == nil {
		t.Fatal("WithReserves() returned nil")
	}
}

func TestAdaPotsBuilder_WithTreasury(t *testing.T) {
	builder := ledger.NewAdaPotsBuilder().WithTreasury(500000000000)
	if builder == nil {
		t.Fatal("WithTreasury() returned nil")
	}
}

func TestAdaPotsBuilder_WithRewards(t *testing.T) {
	builder := ledger.NewAdaPotsBuilder().WithRewards(100000000000)
	if builder == nil {
		t.Fatal("WithRewards() returned nil")
	}
}

func TestAdaPotsBuilder_Build(t *testing.T) {
	reserves := uint64(10000000000000)
	treasury := uint64(500000000000)
	rewards := uint64(100000000000)

	pots, err := ledger.NewAdaPotsBuilder().
		WithReserves(reserves).
		WithTreasury(treasury).
		WithRewards(rewards).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if pots.Reserves != reserves {
		t.Errorf("Expected Reserves %d, got %d", reserves, pots.Reserves)
	}

	if pots.Treasury != treasury {
		t.Errorf("Expected Treasury %d, got %d", treasury, pots.Treasury)
	}

	if pots.Rewards != rewards {
		t.Errorf("Expected Rewards %d, got %d", rewards, pots.Rewards)
	}
}

// =============================================================================
// RewardSnapshotBuilder Tests
// =============================================================================

func TestNewRewardSnapshotBuilder(t *testing.T) {
	builder := ledger.NewRewardSnapshotBuilder()
	if builder == nil {
		t.Fatal("NewRewardSnapshotBuilder() returned nil")
	}
}

func TestRewardSnapshotBuilder_WithTotalActiveStake(t *testing.T) {
	builder := ledger.NewRewardSnapshotBuilder().
		WithTotalActiveStake(32000000000000000)
	if builder == nil {
		t.Fatal("WithTotalActiveStake() returned nil")
	}
}

func TestRewardSnapshotBuilder_Build(t *testing.T) {
	totalStake := uint64(32000000000000000)

	snapshot, err := ledger.NewRewardSnapshotBuilder().
		WithTotalActiveStake(totalStake).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if snapshot.TotalActiveStake != totalStake {
		t.Errorf(
			"Expected TotalActiveStake %d, got %d",
			totalStake,
			snapshot.TotalActiveStake,
		)
	}
}

// =============================================================================
// CommitteeMemberBuilder Tests
// =============================================================================

func TestNewCommitteeMemberBuilder(t *testing.T) {
	builder := ledger.NewCommitteeMemberBuilder()
	if builder == nil {
		t.Fatal("NewCommitteeMemberBuilder() returned nil")
	}
}

func TestCommitteeMemberBuilder_WithColdKey(t *testing.T) {
	builder := ledger.NewCommitteeMemberBuilder().WithColdKey(sampleKeyHash())
	if builder == nil {
		t.Fatal("WithColdKey() returned nil")
	}
}

func TestCommitteeMemberBuilder_WithHotKey(t *testing.T) {
	builder := ledger.NewCommitteeMemberBuilder().
		WithColdKey(sampleKeyHash()).
		WithHotKey(bytes.Repeat([]byte{0x01}, 28))
	if builder == nil {
		t.Fatal("WithHotKey() returned nil")
	}
}

func TestCommitteeMemberBuilder_WithExpiryEpoch(t *testing.T) {
	builder := ledger.NewCommitteeMemberBuilder().
		WithColdKey(sampleKeyHash()).
		WithExpiryEpoch(500)
	if builder == nil {
		t.Fatal("WithExpiryEpoch() returned nil")
	}
}

func TestCommitteeMemberBuilder_WithResigned(t *testing.T) {
	builder := ledger.NewCommitteeMemberBuilder().
		WithColdKey(sampleKeyHash()).
		WithResigned(true)
	if builder == nil {
		t.Fatal("WithResigned() returned nil")
	}
}

func TestCommitteeMemberBuilder_Build_Success(t *testing.T) {
	coldKey := sampleKeyHash()
	hotKey := bytes.Repeat([]byte{0x01}, 28)
	expiryEpoch := uint64(500)

	member, err := ledger.NewCommitteeMemberBuilder().
		WithColdKey(coldKey).
		WithHotKey(hotKey).
		WithExpiryEpoch(expiryEpoch).
		WithResigned(false).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if member.ExpiryEpoch != expiryEpoch {
		t.Errorf(
			"Expected ExpiryEpoch %d, got %d",
			expiryEpoch,
			member.ExpiryEpoch,
		)
	}

	if member.Resigned {
		t.Error("Expected Resigned to be false")
	}
}

func TestCommitteeMemberBuilder_Build_MissingColdKey(t *testing.T) {
	_, err := ledger.NewCommitteeMemberBuilder().
		WithExpiryEpoch(500).
		Build()

	if err == nil {
		t.Fatal("Build() should return error when cold key is missing")
	}
}

// =============================================================================
// DRepRegistrationBuilder Tests
// =============================================================================

func TestNewDRepRegistrationBuilder(t *testing.T) {
	builder := ledger.NewDRepRegistrationBuilder()
	if builder == nil {
		t.Fatal("NewDRepRegistrationBuilder() returned nil")
	}
}

func TestDRepRegistrationBuilder_WithCredential(t *testing.T) {
	builder := ledger.NewDRepRegistrationBuilder().
		WithCredential(sampleKeyHash())
	if builder == nil {
		t.Fatal("WithCredential() returned nil")
	}
}

func TestDRepRegistrationBuilder_WithAnchor(t *testing.T) {
	builder := ledger.NewDRepRegistrationBuilder().
		WithCredential(sampleKeyHash()).
		WithAnchor("https://example.com/drep.json", bytes.Repeat([]byte{0x55}, 32))
	if builder == nil {
		t.Fatal("WithAnchor() returned nil")
	}
}

func TestDRepRegistrationBuilder_WithDeposit(t *testing.T) {
	builder := ledger.NewDRepRegistrationBuilder().
		WithCredential(sampleKeyHash()).
		WithDeposit(500000000)
	if builder == nil {
		t.Fatal("WithDeposit() returned nil")
	}
}

func TestDRepRegistrationBuilder_Build_Success(t *testing.T) {
	cred := sampleKeyHash()
	deposit := uint64(500000000)
	anchorUrl := "https://example.com/drep.json"

	cert, err := ledger.NewDRepRegistrationBuilder().
		WithCredential(cred).
		WithDeposit(deposit).
		WithAnchor(anchorUrl, bytes.Repeat([]byte{0x55}, 32)).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if cert.Amount != int64(deposit) {
		t.Errorf("Expected Amount %d, got %d", deposit, cert.Amount)
	}

	if cert.Anchor == nil {
		t.Fatal("Expected Anchor to be set")
	}

	if cert.Anchor.Url != anchorUrl {
		t.Errorf("Expected anchor URL %s, got %s", anchorUrl, cert.Anchor.Url)
	}
}

func TestDRepRegistrationBuilder_Build_MissingCredential(t *testing.T) {
	_, err := ledger.NewDRepRegistrationBuilder().
		WithDeposit(500000000).
		Build()

	if err == nil {
		t.Fatal("Build() should return error when credential is missing")
	}
}

// =============================================================================
// ConstitutionBuilder Tests
// =============================================================================

func TestNewConstitutionBuilder(t *testing.T) {
	builder := ledger.NewConstitutionBuilder()
	if builder == nil {
		t.Fatal("NewConstitutionBuilder() returned nil")
	}
}

func TestConstitutionBuilder_WithAnchor(t *testing.T) {
	builder := ledger.NewConstitutionBuilder().
		WithAnchor("https://example.com/constitution.txt", bytes.Repeat([]byte{0x77}, 32))
	if builder == nil {
		t.Fatal("WithAnchor() returned nil")
	}
}

func TestConstitutionBuilder_WithScriptHash(t *testing.T) {
	builder := ledger.NewConstitutionBuilder().
		WithAnchor("https://example.com/constitution.txt", nil).
		WithScriptHash(sampleKeyHash())
	if builder == nil {
		t.Fatal("WithScriptHash() returned nil")
	}
}

func TestConstitutionBuilder_Build_Success(t *testing.T) {
	anchorUrl := "https://example.com/constitution.txt"
	dataHash := bytes.Repeat([]byte{0x77}, 32)
	scriptHash := sampleKeyHash()

	constitution, err := ledger.NewConstitutionBuilder().
		WithAnchor(anchorUrl, dataHash).
		WithScriptHash(scriptHash).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if constitution.Anchor.Url != anchorUrl {
		t.Errorf(
			"Expected anchor URL %s, got %s",
			anchorUrl,
			constitution.Anchor.Url,
		)
	}

	if !bytes.Equal(constitution.ScriptHash, scriptHash) {
		t.Errorf("Expected ScriptHash to match")
	}
}

func TestConstitutionBuilder_Build_MissingAnchorURL(t *testing.T) {
	_, err := ledger.NewConstitutionBuilder().
		WithScriptHash(sampleKeyHash()).
		Build()

	if err == nil {
		t.Fatal("Build() should return error when anchor URL is missing")
	}
}

// =============================================================================
// GovAnchorBuilder Tests
// =============================================================================

func TestNewGovAnchorBuilder(t *testing.T) {
	builder := ledger.NewGovAnchorBuilder()
	if builder == nil {
		t.Fatal("NewGovAnchorBuilder() returned nil")
	}
}

func TestGovAnchorBuilder_WithURL(t *testing.T) {
	builder := ledger.NewGovAnchorBuilder().
		WithURL("https://example.com/proposal.json")
	if builder == nil {
		t.Fatal("WithURL() returned nil")
	}
}

func TestGovAnchorBuilder_WithDataHash(t *testing.T) {
	builder := ledger.NewGovAnchorBuilder().
		WithURL("https://example.com/proposal.json").
		WithDataHash(bytes.Repeat([]byte{0x88}, 32))
	if builder == nil {
		t.Fatal("WithDataHash() returned nil")
	}
}

func TestGovAnchorBuilder_Build_Success(t *testing.T) {
	url := "https://example.com/proposal.json"
	dataHash := bytes.Repeat([]byte{0x88}, 32)

	anchor, err := ledger.NewGovAnchorBuilder().
		WithURL(url).
		WithDataHash(dataHash).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if anchor.Url != url {
		t.Errorf("Expected URL %s, got %s", url, anchor.Url)
	}
}

func TestGovAnchorBuilder_Build_MissingURL(t *testing.T) {
	_, err := ledger.NewGovAnchorBuilder().
		WithDataHash(bytes.Repeat([]byte{0x88}, 32)).
		Build()

	if err == nil {
		t.Fatal("Build() should return error when URL is missing")
	}
}

// =============================================================================
// VoterBuilder Tests
// =============================================================================

func TestNewVoterBuilder(t *testing.T) {
	builder := ledger.NewVoterBuilder()
	if builder == nil {
		t.Fatal("NewVoterBuilder() returned nil")
	}
}

func TestVoterBuilder_WithType(t *testing.T) {
	builder := ledger.NewVoterBuilder().WithType(1) // DRep
	if builder == nil {
		t.Fatal("WithType() returned nil")
	}
}

func TestVoterBuilder_WithHash(t *testing.T) {
	builder := ledger.NewVoterBuilder().
		WithType(1).
		WithHash(sampleKeyHash())
	if builder == nil {
		t.Fatal("WithHash() returned nil")
	}
}

func TestVoterBuilder_Build_Success(t *testing.T) {
	voterType := uint8(1)
	hash := sampleKeyHash()

	voter, err := ledger.NewVoterBuilder().
		WithType(voterType).
		WithHash(hash).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if voter.Type != voterType {
		t.Errorf("Expected Type %d, got %d", voterType, voter.Type)
	}
}

func TestVoterBuilder_Build_MissingHash(t *testing.T) {
	_, err := ledger.NewVoterBuilder().
		WithType(1).
		Build()

	if err == nil {
		t.Fatal("Build() should return error when hash is missing")
	}
}

// =============================================================================
// VotingProcedureBuilder Tests
// =============================================================================

func TestNewVotingProcedureBuilder(t *testing.T) {
	builder := ledger.NewVotingProcedureBuilder()
	if builder == nil {
		t.Fatal("NewVotingProcedureBuilder() returned nil")
	}
}

func TestVotingProcedureBuilder_WithVote(t *testing.T) {
	builder := ledger.NewVotingProcedureBuilder().WithVote(1) // Yes
	if builder == nil {
		t.Fatal("WithVote() returned nil")
	}
}

func TestVotingProcedureBuilder_WithAnchor(t *testing.T) {
	builder := ledger.NewVotingProcedureBuilder().
		WithVote(1).
		WithAnchor("https://example.com/rationale.json", bytes.Repeat([]byte{0x99}, 32))
	if builder == nil {
		t.Fatal("WithAnchor() returned nil")
	}
}

func TestVotingProcedureBuilder_Build_Success(t *testing.T) {
	vote := uint8(1) // Yes
	anchorUrl := "https://example.com/rationale.json"

	procedure, err := ledger.NewVotingProcedureBuilder().
		WithVote(vote).
		WithAnchor(anchorUrl, bytes.Repeat([]byte{0x99}, 32)).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if procedure.Vote != vote {
		t.Errorf("Expected Vote %d, got %d", vote, procedure.Vote)
	}

	if procedure.Anchor == nil {
		t.Fatal("Expected Anchor to be set")
	}

	if procedure.Anchor.Url != anchorUrl {
		t.Errorf(
			"Expected anchor URL %s, got %s",
			anchorUrl,
			procedure.Anchor.Url,
		)
	}
}

func TestVotingProcedureBuilder_Build_WithoutAnchor(t *testing.T) {
	vote := uint8(0) // No

	procedure, err := ledger.NewVotingProcedureBuilder().
		WithVote(vote).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if procedure.Vote != vote {
		t.Errorf("Expected Vote %d, got %d", vote, procedure.Vote)
	}

	if procedure.Anchor != nil {
		t.Error("Expected Anchor to be nil")
	}
}

func TestVotingProcedureBuilder_Build_WithoutVote(t *testing.T) {
	// Build without calling WithVote should return an error
	_, err := ledger.NewVotingProcedureBuilder().Build()

	if err == nil {
		t.Fatal("Build() without WithVote() should return an error")
	}
}

func TestVotingProcedureBuilder_Build_InvalidVote(t *testing.T) {
	// Invalid vote value (>2) should return an error
	_, err := ledger.NewVotingProcedureBuilder().
		WithVote(3).
		Build()

	if err == nil {
		t.Fatal("Build() with invalid vote value should return an error")
	}
}

// =============================================================================
// TransactionBuilder Tests
// =============================================================================

func TestNewTransactionBuilder(t *testing.T) {
	builder := ledger.NewTransactionBuilder()
	if builder == nil {
		t.Fatal("NewTransactionBuilder() returned nil")
	}
}

func TestTransactionBuilder_WithId(t *testing.T) {
	builder := ledger.NewTransactionBuilder().WithId(sampleTxId())
	if builder == nil {
		t.Fatal("WithId() returned nil")
	}
}

func TestTransactionBuilder_WithFee(t *testing.T) {
	builder := ledger.NewTransactionBuilder().WithFee(200000)
	if builder == nil {
		t.Fatal("WithFee() returned nil")
	}
}

func TestTransactionBuilder_WithTTL(t *testing.T) {
	builder := ledger.NewTransactionBuilder().WithTTL(100000000)
	if builder == nil {
		t.Fatal("WithTTL() returned nil")
	}
}

func TestTransactionBuilder_WithValid(t *testing.T) {
	builder := ledger.NewTransactionBuilder().WithValid(true)
	if builder == nil {
		t.Fatal("WithValid() returned nil")
	}
}

func TestTransactionBuilder_Build_MissingInputs(t *testing.T) {
	output, _ := ledger.NewTransactionOutputBuilder().
		WithAddress(sampleAddress()).
		WithLovelace(5000000).
		Build()

	_, err := ledger.NewTransactionBuilder().
		WithId(sampleTxId()).
		WithOutputs(output).
		WithFee(200000).
		Build()

	if err == nil {
		t.Fatal("Build() should return error when inputs are missing")
	}
}

func TestTransactionBuilder_Build_MissingOutputs(t *testing.T) {
	input, _ := ledger.NewTransactionInputBuilder().
		WithTxId(sampleTxId()).
		WithIndex(0).
		Build()

	_, err := ledger.NewTransactionBuilder().
		WithId(sampleTxId()).
		WithInputs(input).
		WithFee(200000).
		Build()

	if err == nil {
		t.Fatal("Build() should return error when outputs are missing")
	}
}

func TestTransactionBuilder_Build_Success(t *testing.T) {
	txId := sampleTxId()
	fee := uint64(200000)
	ttl := uint64(100000000)

	input, _ := ledger.NewTransactionInputBuilder().
		WithTxId(bytes.Repeat([]byte{0x11}, 32)).
		WithIndex(0).
		Build()

	output, _ := ledger.NewTransactionOutputBuilder().
		WithAddress(sampleAddress()).
		WithLovelace(5000000).
		Build()

	tx, err := ledger.NewTransactionBuilder().
		WithId(txId).
		WithInputs(input).
		WithOutputs(output).
		WithFee(fee).
		WithTTL(ttl).
		WithValid(true).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if tx.Fee().Cmp(big.NewInt(int64(fee))) != 0 {
		t.Errorf("Expected fee %d, got %s", fee, tx.Fee())
	}

	if tx.TTL() != ttl {
		t.Errorf("Expected TTL %d, got %d", ttl, tx.TTL())
	}

	if !tx.IsValid() {
		t.Error("Expected transaction to be valid")
	}

	if len(tx.Inputs()) != 1 {
		t.Errorf("Expected 1 input, got %d", len(tx.Inputs()))
	}

	if len(tx.Outputs()) != 1 {
		t.Errorf("Expected 1 output, got %d", len(tx.Outputs()))
	}
}

// =============================================================================
// TransactionInputBuilder Tests
// =============================================================================

func TestNewTransactionInputBuilder(t *testing.T) {
	builder := ledger.NewTransactionInputBuilder()
	if builder == nil {
		t.Fatal("NewTransactionInputBuilder() returned nil")
	}
}

func TestTransactionInputBuilder_WithTxId(t *testing.T) {
	builder := ledger.NewTransactionInputBuilder().WithTxId(sampleTxId())
	if builder == nil {
		t.Fatal("WithTxId() returned nil")
	}
}

func TestTransactionInputBuilder_WithIndex(t *testing.T) {
	builder := ledger.NewTransactionInputBuilder().
		WithTxId(sampleTxId()).
		WithIndex(5)
	if builder == nil {
		t.Fatal("WithIndex() returned nil")
	}
}

func TestTransactionInputBuilder_Build_Success(t *testing.T) {
	txId := sampleTxId()
	index := uint32(3)

	input, err := ledger.NewTransactionInputBuilder().
		WithTxId(txId).
		WithIndex(index).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if input.Index() != index {
		t.Errorf("Expected index %d, got %d", index, input.Index())
	}
}

func TestTransactionInputBuilder_Build_MissingTxId(t *testing.T) {
	_, err := ledger.NewTransactionInputBuilder().
		WithIndex(0).
		Build()

	if err == nil {
		t.Fatal("Build() should return error when TxId is missing")
	}
}

// =============================================================================
// TransactionOutputBuilder Tests
// =============================================================================

func TestNewTransactionOutputBuilder(t *testing.T) {
	builder := ledger.NewTransactionOutputBuilder()
	if builder == nil {
		t.Fatal("NewTransactionOutputBuilder() returned nil")
	}
}

func TestTransactionOutputBuilder_WithAddress(t *testing.T) {
	builder := ledger.NewTransactionOutputBuilder().WithAddress(sampleAddress())
	if builder == nil {
		t.Fatal("WithAddress() returned nil")
	}
}

func TestTransactionOutputBuilder_WithLovelace(t *testing.T) {
	builder := ledger.NewTransactionOutputBuilder().
		WithAddress(sampleAddress()).
		WithLovelace(5000000)
	if builder == nil {
		t.Fatal("WithLovelace() returned nil")
	}
}

func TestTransactionOutputBuilder_WithAssets(t *testing.T) {
	assets := []ledger.Asset{
		{
			PolicyId:  samplePolicyId(),
			AssetName: []byte("Token"),
			Amount:    100,
		},
	}
	builder := ledger.NewTransactionOutputBuilder().
		WithAddress(sampleAddress()).
		WithLovelace(2000000).
		WithAssets(assets...)
	if builder == nil {
		t.Fatal("WithAssets() returned nil")
	}
}

func TestTransactionOutputBuilder_Build_Success(t *testing.T) {
	addr := sampleAddress()
	lovelace := uint64(5000000)

	output, err := ledger.NewTransactionOutputBuilder().
		WithAddress(addr).
		WithLovelace(lovelace).
		Build()

	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if output.Amount().Cmp(big.NewInt(int64(lovelace))) != 0 {
		t.Errorf("Expected amount %d, got %s", lovelace, output.Amount())
	}
}

func TestTransactionOutputBuilder_Build_WithInvalidAddress(t *testing.T) {
	// Building with invalid address still produces an output
	// The validation in Build() checks if address.String() == ""
	// When an invalid address is provided, WithAddress stores an empty Address{}
	// which may still have a non-empty string representation
	builder := ledger.NewTransactionOutputBuilder().
		WithAddress("invalid-address").
		WithLovelace(5000000)

	// The current implementation is lenient - it allows building
	// even with addresses that failed parsing
	output, err := builder.Build()
	if err != nil {
		// If error is returned, that's acceptable behavior
		return
	}

	// If no error, the output should still have the lovelace amount set
	if output.Amount().Cmp(big.NewInt(5000000)) != 0 {
		t.Errorf("Expected amount 5000000, got %s", output.Amount())
	}
}

// =============================================================================
// Protocol Parameters Tests
// =============================================================================

func TestNewMockByronProtocolParams(t *testing.T) {
	params := ledger.NewMockByronProtocolParams()

	if params.SlotDuration != 20000 {
		t.Errorf("Expected SlotDuration 20000, got %d", params.SlotDuration)
	}

	if params.MaxBlockSize != 2000000 {
		t.Errorf("Expected MaxBlockSize 2000000, got %d", params.MaxBlockSize)
	}

	if params.TxFeePolicy[0] != 155381 {
		t.Errorf(
			"Expected TxFeePolicy[0] 155381, got %d",
			params.TxFeePolicy[0],
		)
	}
}

func TestNewMockShelleyProtocolParams(t *testing.T) {
	params := ledger.NewMockShelleyProtocolParams()

	if params.MinFeeA != 44 {
		t.Errorf("Expected MinFeeA 44, got %d", params.MinFeeA)
	}

	if params.MinFeeB != 155381 {
		t.Errorf("Expected MinFeeB 155381, got %d", params.MinFeeB)
	}

	if params.KeyDeposit != 2000000 {
		t.Errorf("Expected KeyDeposit 2000000, got %d", params.KeyDeposit)
	}

	if params.PoolDeposit != 500000000 {
		t.Errorf("Expected PoolDeposit 500000000, got %d", params.PoolDeposit)
	}

	if params.ProtocolMajor != 2 {
		t.Errorf("Expected ProtocolMajor 2, got %d", params.ProtocolMajor)
	}
}

func TestNewMockAllegraProtocolParams(t *testing.T) {
	params := ledger.NewMockAllegraProtocolParams()

	// Should inherit Shelley values
	if params.MinFeeA != 44 {
		t.Errorf("Expected MinFeeA 44, got %d", params.MinFeeA)
	}

	// But have Allegra protocol version
	if params.ProtocolMajor != 3 {
		t.Errorf("Expected ProtocolMajor 3, got %d", params.ProtocolMajor)
	}
}

func TestNewMockMaryProtocolParams(t *testing.T) {
	params := ledger.NewMockMaryProtocolParams()

	if params.ProtocolMajor != 4 {
		t.Errorf("Expected ProtocolMajor 4, got %d", params.ProtocolMajor)
	}

	if params.NOpt != 500 {
		t.Errorf("Expected NOpt 500, got %d", params.NOpt)
	}

	if params.MinPoolCost != 340000000 {
		t.Errorf("Expected MinPoolCost 340000000, got %d", params.MinPoolCost)
	}
}

func TestNewMockAlonzoProtocolParams(t *testing.T) {
	params := ledger.NewMockAlonzoProtocolParams()

	if params.ProtocolMajor != 5 {
		t.Errorf("Expected ProtocolMajor 5, got %d", params.ProtocolMajor)
	}

	if params.AdaPerUtxoByte != 4310 {
		t.Errorf("Expected AdaPerUtxoByte 4310, got %d", params.AdaPerUtxoByte)
	}

	// Check PlutusV1 cost model exists
	if _, ok := params.CostModels[0]; !ok {
		t.Error("Expected PlutusV1 cost model to exist")
	}

	if params.MaxCollateralInputs != 3 {
		t.Errorf(
			"Expected MaxCollateralInputs 3, got %d",
			params.MaxCollateralInputs,
		)
	}

	if params.CollateralPercentage != 150 {
		t.Errorf(
			"Expected CollateralPercentage 150, got %d",
			params.CollateralPercentage,
		)
	}
}

func TestNewMockBabbageProtocolParams(t *testing.T) {
	params := ledger.NewMockBabbageProtocolParams()

	if params.ProtocolMajor != 7 {
		t.Errorf("Expected ProtocolMajor 7, got %d", params.ProtocolMajor)
	}

	if params.MaxBlockBodySize != 90112 {
		t.Errorf(
			"Expected MaxBlockBodySize 90112, got %d",
			params.MaxBlockBodySize,
		)
	}

	// Check PlutusV1 and PlutusV2 cost models exist
	if _, ok := params.CostModels[0]; !ok {
		t.Error("Expected PlutusV1 cost model to exist")
	}
	if _, ok := params.CostModels[1]; !ok {
		t.Error("Expected PlutusV2 cost model to exist")
	}

	if params.MinPoolCost != 170000000 {
		t.Errorf("Expected MinPoolCost 170000000, got %d", params.MinPoolCost)
	}
}

func TestNewMockConwayProtocolParams(t *testing.T) {
	params := ledger.NewMockConwayProtocolParams()

	if params.ProtocolVersion.Major != 9 {
		t.Errorf(
			"Expected ProtocolVersion.Major 9, got %d",
			params.ProtocolVersion.Major,
		)
	}

	// Check all Plutus cost models exist
	if _, ok := params.CostModels[0]; !ok {
		t.Error("Expected PlutusV1 cost model to exist")
	}
	if _, ok := params.CostModels[1]; !ok {
		t.Error("Expected PlutusV2 cost model to exist")
	}
	if _, ok := params.CostModels[2]; !ok {
		t.Error("Expected PlutusV3 cost model to exist")
	}

	// Check governance parameters
	if params.MinCommitteeSize != 7 {
		t.Errorf("Expected MinCommitteeSize 7, got %d", params.MinCommitteeSize)
	}

	if params.GovActionValidityPeriod != 6 {
		t.Errorf(
			"Expected GovActionValidityPeriod 6, got %d",
			params.GovActionValidityPeriod,
		)
	}

	if params.GovActionDeposit != 100000000000 {
		t.Errorf(
			"Expected GovActionDeposit 100000000000, got %d",
			params.GovActionDeposit,
		)
	}

	if params.DRepDeposit != 500000000 {
		t.Errorf("Expected DRepDeposit 500000000, got %d", params.DRepDeposit)
	}

	if params.DRepInactivityPeriod != 20 {
		t.Errorf(
			"Expected DRepInactivityPeriod 20, got %d",
			params.DRepInactivityPeriod,
		)
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestNewSimpleTransaction(t *testing.T) {
	txId := sampleTxId()
	fee := uint64(200000)

	input, _ := ledger.NewTransactionInputBuilder().
		WithTxId(bytes.Repeat([]byte{0x11}, 32)).
		WithIndex(0).
		Build()

	output, _ := ledger.NewTransactionOutputBuilder().
		WithAddress(sampleAddress()).
		WithLovelace(5000000).
		Build()

	tx, err := ledger.NewSimpleTransaction(
		txId,
		[]lcommon.TransactionInput{input},
		[]lcommon.TransactionOutput{output},
		fee,
	)

	if err != nil {
		t.Fatalf("NewSimpleTransaction returned error: %v", err)
	}

	if tx.Fee().Cmp(big.NewInt(int64(fee))) != 0 {
		t.Errorf("Expected fee %d, got %s", fee, tx.Fee())
	}
}

func TestNewSimpleTransactionInput(t *testing.T) {
	txId := sampleTxId()
	index := uint32(5)

	input, err := ledger.NewSimpleTransactionInput(txId, index)
	if err != nil {
		t.Fatalf("NewSimpleTransactionInput returned error: %v", err)
	}

	if input.Index() != index {
		t.Errorf("Expected index %d, got %d", index, input.Index())
	}
}

func TestNewSimpleTransactionOutput(t *testing.T) {
	addr := sampleAddress()
	lovelace := uint64(10000000)

	output, err := ledger.NewSimpleTransactionOutput(addr, lovelace)
	if err != nil {
		t.Fatalf("NewSimpleTransactionOutput returned error: %v", err)
	}

	if output.Amount().Cmp(big.NewInt(int64(lovelace))) != 0 {
		t.Errorf("Expected amount %d, got %s", lovelace, output.Amount())
	}
}

// =============================================================================
// MockTransactionInput Tests
// =============================================================================

func TestMockTransactionInput_String(t *testing.T) {
	input, err := ledger.NewSimpleTransactionInput(sampleTxId(), 0)
	if err != nil {
		t.Fatalf("NewSimpleTransactionInput() returned error: %v", err)
	}
	str := input.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
}

func TestMockTransactionInput_Utxorpc(t *testing.T) {
	input, err := ledger.NewSimpleTransactionInput(sampleTxId(), 3)
	if err != nil {
		t.Fatalf("NewSimpleTransactionInput() returned error: %v", err)
	}
	utxorpc, err := input.Utxorpc()
	if err != nil {
		t.Fatalf("Utxorpc() returned error: %v", err)
	}
	if utxorpc.OutputIndex != 3 {
		t.Errorf("Expected OutputIndex 3, got %d", utxorpc.OutputIndex)
	}
}

// =============================================================================
// MockTransactionOutput Tests
// =============================================================================

func TestMockTransactionOutput_String(t *testing.T) {
	output, err := ledger.NewSimpleTransactionOutput(sampleAddress(), 5000000)
	if err != nil {
		t.Fatalf("NewSimpleTransactionOutput() returned error: %v", err)
	}
	str := output.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
}

func TestMockTransactionOutput_Utxorpc(t *testing.T) {
	output, err := ledger.NewSimpleTransactionOutput(sampleAddress(), 5000000)
	if err != nil {
		t.Fatalf("NewSimpleTransactionOutput() returned error: %v", err)
	}
	utxorpc, err := output.Utxorpc()
	if err != nil {
		t.Fatalf("Utxorpc() returned error: %v", err)
	}
	if utxorpc.Address == nil {
		t.Error("Expected Address to be set")
	}
}

// =============================================================================
// MockTransaction Method Tests
// =============================================================================

func TestMockTransaction_Produced(t *testing.T) {
	input, err := ledger.NewTransactionInputBuilder().
		WithTxId(bytes.Repeat([]byte{0x11}, 32)).
		WithIndex(0).
		Build()
	if err != nil {
		t.Fatalf("NewTransactionInputBuilder().Build() returned error: %v", err)
	}

	output1, err := ledger.NewTransactionOutputBuilder().
		WithAddress(sampleAddress()).
		WithLovelace(3000000).
		Build()
	if err != nil {
		t.Fatalf(
			"NewTransactionOutputBuilder().Build() returned error: %v",
			err,
		)
	}

	output2, err := ledger.NewTransactionOutputBuilder().
		WithAddress(sampleAddress()).
		WithLovelace(2000000).
		Build()
	if err != nil {
		t.Fatalf(
			"NewTransactionOutputBuilder().Build() returned error: %v",
			err,
		)
	}

	tx, err := ledger.NewTransactionBuilder().
		WithId(sampleTxId()).
		WithInputs(input).
		WithOutputs(output1, output2).
		WithFee(200000).
		Build()
	if err != nil {
		t.Fatalf("NewTransactionBuilder().Build() returned error: %v", err)
	}

	produced := tx.Produced()
	if produced == nil {
		t.Fatal("Produced() returned nil")
	}
	if len(produced) != 2 {
		t.Errorf("Expected 2 produced UTxOs, got %d", len(produced))
	}

	if produced[0].Id.Index() != 0 {
		t.Errorf(
			"Expected first produced UTxO index 0, got %d",
			produced[0].Id.Index(),
		)
	}

	if produced[1].Id.Index() != 1 {
		t.Errorf(
			"Expected second produced UTxO index 1, got %d",
			produced[1].Id.Index(),
		)
	}
}

func TestMockTransaction_Consumed(t *testing.T) {
	input1, err := ledger.NewTransactionInputBuilder().
		WithTxId(bytes.Repeat([]byte{0x11}, 32)).
		WithIndex(0).
		Build()
	if err != nil {
		t.Fatalf("NewTransactionInputBuilder().Build() returned error: %v", err)
	}

	input2, err := ledger.NewTransactionInputBuilder().
		WithTxId(bytes.Repeat([]byte{0x22}, 32)).
		WithIndex(1).
		Build()
	if err != nil {
		t.Fatalf("NewTransactionInputBuilder().Build() returned error: %v", err)
	}

	output, err := ledger.NewTransactionOutputBuilder().
		WithAddress(sampleAddress()).
		WithLovelace(5000000).
		Build()
	if err != nil {
		t.Fatalf(
			"NewTransactionOutputBuilder().Build() returned error: %v",
			err,
		)
	}

	tx, err := ledger.NewTransactionBuilder().
		WithId(sampleTxId()).
		WithInputs(input1, input2).
		WithOutputs(output).
		WithFee(200000).
		Build()
	if err != nil {
		t.Fatalf("NewTransactionBuilder().Build() returned error: %v", err)
	}

	consumed := tx.Consumed()
	if len(consumed) != 2 {
		t.Errorf("Expected 2 consumed inputs, got %d", len(consumed))
	}
}

func TestMockTransaction_Witnesses(t *testing.T) {
	input, err := ledger.NewTransactionInputBuilder().
		WithTxId(bytes.Repeat([]byte{0x11}, 32)).
		WithIndex(0).
		Build()
	if err != nil {
		t.Fatalf("NewTransactionInputBuilder().Build() returned error: %v", err)
	}

	output, err := ledger.NewTransactionOutputBuilder().
		WithAddress(sampleAddress()).
		WithLovelace(5000000).
		Build()
	if err != nil {
		t.Fatalf(
			"NewTransactionOutputBuilder().Build() returned error: %v",
			err,
		)
	}

	tx, err := ledger.NewTransactionBuilder().
		WithId(sampleTxId()).
		WithInputs(input).
		WithOutputs(output).
		WithFee(200000).
		Build()
	if err != nil {
		t.Fatalf("NewTransactionBuilder().Build() returned error: %v", err)
	}

	witnesses := tx.Witnesses()
	if witnesses == nil {
		t.Fatal("Expected Witnesses() to return non-nil value")
	}
}

// =============================================================================
// Plutus Language Constants Tests
// =============================================================================

func TestPlutusLanguageConstants(t *testing.T) {
	if ledger.PlutusV1 != 1 {
		t.Errorf("Expected PlutusV1 to be 1, got %d", ledger.PlutusV1)
	}

	if ledger.PlutusV2 != 2 {
		t.Errorf("Expected PlutusV2 to be 2, got %d", ledger.PlutusV2)
	}

	if ledger.PlutusV3 != 3 {
		t.Errorf("Expected PlutusV3 to be 3, got %d", ledger.PlutusV3)
	}
}

// =============================================================================
// ErrNotFound Tests
// =============================================================================

func TestErrNotFound(t *testing.T) {
	if ledger.ErrNotFound == nil {
		t.Fatal("ErrNotFound should not be nil")
	}

	if ledger.ErrNotFound.Error() != "ledger: not found" {
		t.Errorf(
			"Expected error message 'ledger: not found', got '%s'",
			ledger.ErrNotFound.Error(),
		)
	}
}
