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

package conformance

import (
	"maps"
	"slices"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

// parameterUpdatesEqual compares decoded fields without comparing or rewriting
// preserved CBOR. Re-encoding is not a value comparison: the rational encoder
// has narrower magnitude support than the decoder.
func parameterUpdatesEqual(a, b *conway.ConwayProtocolParameterUpdate) bool {
	if a == nil || b == nil {
		return a == b
	}
	return optionalValueEqual(a.MinFeeA, b.MinFeeA) &&
		optionalValueEqual(a.MinFeeB, b.MinFeeB) &&
		optionalValueEqual(a.MaxBlockBodySize, b.MaxBlockBodySize) &&
		optionalValueEqual(a.MaxTxSize, b.MaxTxSize) &&
		optionalValueEqual(a.MaxBlockHeaderSize, b.MaxBlockHeaderSize) &&
		optionalValueEqual(a.KeyDeposit, b.KeyDeposit) &&
		optionalValueEqual(a.PoolDeposit, b.PoolDeposit) &&
		optionalValueEqual(a.MaxEpoch, b.MaxEpoch) &&
		optionalValueEqual(a.NOpt, b.NOpt) &&
		rationalValueEqual(a.A0, b.A0) &&
		rationalValueEqual(a.Rho, b.Rho) &&
		rationalValueEqual(a.Tau, b.Tau) &&
		optionalValueEqual(a.ProtocolVersion, b.ProtocolVersion) &&
		optionalValueEqual(a.MinPoolCost, b.MinPoolCost) &&
		optionalValueEqual(a.AdaPerUtxoByte, b.AdaPerUtxoByte) &&
		(a.CostModels == nil) == (b.CostModels == nil) &&
		maps.EqualFunc(a.CostModels, b.CostModels, func(x, y []int64) bool {
			return (x == nil) == (y == nil) && slices.Equal(x, y)
		}) &&
		executionPricesEqual(a.ExecutionCosts, b.ExecutionCosts) &&
		optionalValueEqual(a.MaxTxExUnits, b.MaxTxExUnits) &&
		optionalValueEqual(a.MaxBlockExUnits, b.MaxBlockExUnits) &&
		optionalValueEqual(a.MaxValueSize, b.MaxValueSize) &&
		optionalValueEqual(a.CollateralPercentage, b.CollateralPercentage) &&
		optionalValueEqual(a.MaxCollateralInputs, b.MaxCollateralInputs) &&
		poolThresholdsEqual(a.PoolVotingThresholds, b.PoolVotingThresholds) &&
		drepThresholdsEqual(a.DRepVotingThresholds, b.DRepVotingThresholds) &&
		optionalValueEqual(a.MinCommitteeSize, b.MinCommitteeSize) &&
		optionalValueEqual(a.CommitteeTermLimit, b.CommitteeTermLimit) &&
		optionalValueEqual(
			a.GovActionValidityPeriod,
			b.GovActionValidityPeriod,
		) &&
		optionalValueEqual(a.GovActionDeposit, b.GovActionDeposit) &&
		optionalValueEqual(a.DRepDeposit, b.DRepDeposit) &&
		optionalValueEqual(a.DRepInactivityPeriod, b.DRepInactivityPeriod) &&
		rationalValueEqual(
			a.MinFeeRefScriptCostPerByte,
			b.MinFeeRefScriptCostPerByte,
		)
}

func optionalValueEqual[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func rationalValueEqual(a, b *cbor.Rat) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Rat == nil || b.Rat == nil {
		return a.Rat == b.Rat
	}
	return a.Cmp(b.Rat) == 0
}

func executionPricesEqual(a, b *common.ExUnitPrice) bool {
	if a == nil || b == nil {
		return a == b
	}
	return rationalValueEqual(a.MemPrice, b.MemPrice) &&
		rationalValueEqual(a.StepPrice, b.StepPrice)
}

func poolThresholdsEqual(a, b *conway.PoolVotingThresholds) bool {
	if a == nil || b == nil {
		return a == b
	}
	return rationalValueEqual(&a.MotionNoConfidence, &b.MotionNoConfidence) &&
		rationalValueEqual(&a.CommitteeNormal, &b.CommitteeNormal) &&
		rationalValueEqual(
			&a.CommitteeNoConfidence,
			&b.CommitteeNoConfidence,
		) &&
		rationalValueEqual(&a.HardForkInitiation, &b.HardForkInitiation) &&
		rationalValueEqual(&a.PpSecurityGroup, &b.PpSecurityGroup)
}

func drepThresholdsEqual(a, b *conway.DRepVotingThresholds) bool {
	if a == nil || b == nil {
		return a == b
	}
	return rationalValueEqual(&a.MotionNoConfidence, &b.MotionNoConfidence) &&
		rationalValueEqual(&a.CommitteeNormal, &b.CommitteeNormal) &&
		rationalValueEqual(
			&a.CommitteeNoConfidence,
			&b.CommitteeNoConfidence,
		) &&
		rationalValueEqual(&a.UpdateToConstitution, &b.UpdateToConstitution) &&
		rationalValueEqual(&a.HardForkInitiation, &b.HardForkInitiation) &&
		rationalValueEqual(&a.PpNetworkGroup, &b.PpNetworkGroup) &&
		rationalValueEqual(&a.PpEconomicGroup, &b.PpEconomicGroup) &&
		rationalValueEqual(&a.PpTechnicalGroup, &b.PpTechnicalGroup) &&
		rationalValueEqual(&a.PpGovGroup, &b.PpGovGroup) &&
		rationalValueEqual(&a.TreasuryWithdrawal, &b.TreasuryWithdrawal)
}
