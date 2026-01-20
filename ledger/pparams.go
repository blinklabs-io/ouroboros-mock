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

package ledger

import (
	"math/big"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

// Helper function to create a cbor.Rat from numerator and denominator
func newRat(num, denom int64) *cbor.Rat {
	return &cbor.Rat{Rat: big.NewRat(num, denom)}
}

// MockByronProtocolParams represents mock Byron protocol parameters.
// Note: Byron doesn't have a formal ProtocolParameters struct in gouroboros,
// but these values represent typical Byron-era parameters.
type MockByronProtocolParams struct {
	ScriptVersion     uint16
	SlotDuration      uint64 // milliseconds
	MaxBlockSize      uint64
	MaxHeaderSize     uint64
	MaxTxSize         uint64
	MaxProposalSize   uint64
	MpcThd            uint64
	HeavyDelThd       uint64
	UpdateVoteThd     uint64
	UpdateProposalThd uint64
	UpdateImplicit    uint64
	SoftForkRule      [3]uint64 // [initThd, minThd, thdDecrement]
	TxFeePolicy       [2]uint64 // [summand, multiplier] - fee = summand + multiplier * txSize
	UnlockStakeEpoch  uint64
}

// NewMockByronProtocolParams returns mock Byron protocol parameters
// with typical mainnet values
func NewMockByronProtocolParams() MockByronProtocolParams {
	return MockByronProtocolParams{
		ScriptVersion:     0,
		SlotDuration:      20000, // 20 seconds
		MaxBlockSize:      2000000,
		MaxHeaderSize:     2000000,
		MaxTxSize:         8192,
		MaxProposalSize:   700,
		MpcThd:            20000000000000, // 2% of total stake
		HeavyDelThd:       300000000000,   // 0.03% of total stake
		UpdateVoteThd:     1000000000000,  // 0.1% of total stake
		UpdateProposalThd: 100000000000,   // 0.01% of total stake
		UpdateImplicit:    10000,          // slots
		SoftForkRule: [3]uint64{
			900000000000000,
			600000000000000,
			50000000000000,
		},
		TxFeePolicy:      [2]uint64{155381, 44}, // fee = 155381 + 44 * txSize
		UnlockStakeEpoch: 18446744073709551615,  // max uint64 (never)
	}
}

// NewMockShelleyProtocolParams returns mock Shelley protocol parameters
// with typical mainnet values at Shelley launch
func NewMockShelleyProtocolParams() shelley.ShelleyProtocolParameters {
	return shelley.ShelleyProtocolParameters{
		MinFeeA:            44,
		MinFeeB:            155381,
		MaxBlockBodySize:   65536,
		MaxTxSize:          16384,
		MaxBlockHeaderSize: 1100,
		KeyDeposit:         2000000,         // 2 ADA
		PoolDeposit:        500000000,       // 500 ADA
		MaxEpoch:           18,              // pool retirement max epochs
		NOpt:               150,             // desired number of pools
		A0:                 newRat(3, 10),   // pool influence factor 0.3
		Rho:                newRat(3, 1000), // monetary expansion 0.003
		Tau:                newRat(2, 10),   // treasury cut 0.2
		Decentralization: newRat(
			1,
			1,
		), // d=1 means fully federated (Shelley launch)
		ExtraEntropy:  lcommon.Nonce{}, // neutral nonce
		ProtocolMajor: 2,
		ProtocolMinor: 0,
		MinUtxoValue:  1000000, // 1 ADA minimum UTxO
	}
}

// NewMockAllegraProtocolParams returns mock Allegra protocol parameters
// Allegra uses the same parameter structure as Shelley
func NewMockAllegraProtocolParams() shelley.ShelleyProtocolParameters {
	params := NewMockShelleyProtocolParams()
	params.ProtocolMajor = 3
	params.ProtocolMinor = 0
	return params
}

// NewMockMaryProtocolParams returns mock Mary protocol parameters
// with typical mainnet values
func NewMockMaryProtocolParams() mary.MaryProtocolParameters {
	return mary.MaryProtocolParameters{
		MinFeeA:            44,
		MinFeeB:            155381,
		MaxBlockBodySize:   65536,
		MaxTxSize:          16384,
		MaxBlockHeaderSize: 1100,
		KeyDeposit:         2000000,   // 2 ADA
		PoolDeposit:        500000000, // 500 ADA
		MaxEpoch:           18,
		NOpt:               500,             // increased from Shelley
		A0:                 newRat(3, 10),   // 0.3
		Rho:                newRat(3, 1000), // 0.003
		Tau:                newRat(2, 10),   // 0.2
		Decentralization:   newRat(0, 1),    // fully decentralized (d=0)
		ExtraEntropy:       lcommon.Nonce{},
		ProtocolMajor:      4,
		ProtocolMinor:      0,
		MinUtxoValue:       1000000,   // 1 ADA
		MinPoolCost:        340000000, // 340 ADA minimum pool cost
	}
}

// NewMockAlonzoProtocolParams returns mock Alonzo protocol parameters
// with typical mainnet values including Plutus support
func NewMockAlonzoProtocolParams() alonzo.AlonzoProtocolParameters {
	return alonzo.AlonzoProtocolParameters{
		MinFeeA:            44,
		MinFeeB:            155381,
		MaxBlockBodySize:   73728, // increased for smart contracts
		MaxTxSize:          16384,
		MaxBlockHeaderSize: 1100,
		KeyDeposit:         2000000,
		PoolDeposit:        500000000,
		MaxEpoch:           18,
		NOpt:               500,
		A0:                 newRat(3, 10),
		Rho:                newRat(3, 1000),
		Tau:                newRat(2, 10),
		Decentralization:   newRat(0, 1),
		ExtraEntropy:       lcommon.Nonce{},
		ProtocolMajor:      5,
		ProtocolMinor:      0,
		MinUtxoValue:       0, // deprecated, use AdaPerUtxoByte
		MinPoolCost:        340000000,
		AdaPerUtxoByte:     4310,
		CostModels: map[uint][]int64{
			0: mockPlutusV1CostModel(), // PlutusV1
		},
		ExecutionCosts: lcommon.ExUnitPrice{
			MemPrice:  newRat(577, 10000),    // 0.0577 lovelace per memory unit
			StepPrice: newRat(721, 10000000), // 0.0000721 lovelace per step
		},
		MaxTxExUnits: lcommon.ExUnits{
			Memory: 10000000,    // 10M memory units
			Steps:  10000000000, // 10B CPU steps
		},
		MaxBlockExUnits: lcommon.ExUnits{
			Memory: 50000000,    // 50M memory units
			Steps:  40000000000, // 40B CPU steps
		},
		MaxValueSize:         5000,
		CollateralPercentage: 150, // 150% collateral required
		MaxCollateralInputs:  3,
	}
}

// NewMockBabbageProtocolParams returns mock Babbage protocol parameters
// with typical mainnet values including reference scripts
func NewMockBabbageProtocolParams() babbage.BabbageProtocolParameters {
	return babbage.BabbageProtocolParameters{
		MinFeeA:            44,
		MinFeeB:            155381,
		MaxBlockBodySize:   90112, // increased from Alonzo
		MaxTxSize:          16384,
		MaxBlockHeaderSize: 1100,
		KeyDeposit:         2000000,
		PoolDeposit:        500000000,
		MaxEpoch:           18,
		NOpt:               500,
		A0:                 newRat(3, 10),
		Rho:                newRat(3, 1000),
		Tau:                newRat(2, 10),
		ProtocolMajor:      7,
		ProtocolMinor:      0,
		MinPoolCost:        170000000, // reduced to 170 ADA
		AdaPerUtxoByte:     4310,
		CostModels: map[uint][]int64{
			0: mockPlutusV1CostModel(),
			1: mockPlutusV2CostModel(),
		},
		ExecutionCosts: lcommon.ExUnitPrice{
			MemPrice:  newRat(577, 10000),
			StepPrice: newRat(721, 10000000),
		},
		MaxTxExUnits: lcommon.ExUnits{
			Memory: 14000000,
			Steps:  10000000000,
		},
		MaxBlockExUnits: lcommon.ExUnits{
			Memory: 62000000,
			Steps:  40000000000,
		},
		MaxValueSize:         5000,
		CollateralPercentage: 150,
		MaxCollateralInputs:  3,
	}
}

// NewMockConwayProtocolParams returns mock Conway protocol parameters
// with typical mainnet values including governance parameters
func NewMockConwayProtocolParams() conway.ConwayProtocolParameters {
	return conway.ConwayProtocolParameters{
		MinFeeA:            44,
		MinFeeB:            155381,
		MaxBlockBodySize:   90112,
		MaxTxSize:          16384,
		MaxBlockHeaderSize: 1100,
		KeyDeposit:         2000000,
		PoolDeposit:        500000000,
		MaxEpoch:           18,
		NOpt:               500,
		A0:                 newRat(3, 10),
		Rho:                newRat(3, 1000),
		Tau:                newRat(2, 10),
		ProtocolVersion: lcommon.ProtocolParametersProtocolVersion{
			Major: 9,
			Minor: 0,
		},
		MinPoolCost:    170000000,
		AdaPerUtxoByte: 4310,
		CostModels: map[uint][]int64{
			0: mockPlutusV1CostModel(),
			1: mockPlutusV2CostModel(),
			2: mockPlutusV3CostModel(),
		},
		ExecutionCosts: lcommon.ExUnitPrice{
			MemPrice:  newRat(577, 10000),
			StepPrice: newRat(721, 10000000),
		},
		MaxTxExUnits: lcommon.ExUnits{
			Memory: 14000000,
			Steps:  10000000000,
		},
		MaxBlockExUnits: lcommon.ExUnits{
			Memory: 62000000,
			Steps:  40000000000,
		},
		MaxValueSize:         5000,
		CollateralPercentage: 150,
		MaxCollateralInputs:  3,
		// Conway governance parameters
		PoolVotingThresholds: conway.PoolVotingThresholds{
			MotionNoConfidence:    cbor.Rat{Rat: big.NewRat(51, 100)}, // 51%
			CommitteeNormal:       cbor.Rat{Rat: big.NewRat(51, 100)},
			CommitteeNoConfidence: cbor.Rat{Rat: big.NewRat(51, 100)},
			HardForkInitiation:    cbor.Rat{Rat: big.NewRat(51, 100)},
			PpSecurityGroup:       cbor.Rat{Rat: big.NewRat(51, 100)},
		},
		DRepVotingThresholds: conway.DRepVotingThresholds{
			MotionNoConfidence:    cbor.Rat{Rat: big.NewRat(67, 100)}, // 67%
			CommitteeNormal:       cbor.Rat{Rat: big.NewRat(67, 100)},
			CommitteeNoConfidence: cbor.Rat{Rat: big.NewRat(60, 100)}, // 60%
			UpdateToConstitution:  cbor.Rat{Rat: big.NewRat(75, 100)}, // 75%
			HardForkInitiation:    cbor.Rat{Rat: big.NewRat(60, 100)},
			PpNetworkGroup:        cbor.Rat{Rat: big.NewRat(67, 100)},
			PpEconomicGroup:       cbor.Rat{Rat: big.NewRat(67, 100)},
			PpTechnicalGroup:      cbor.Rat{Rat: big.NewRat(67, 100)},
			PpGovGroup:            cbor.Rat{Rat: big.NewRat(75, 100)},
			TreasuryWithdrawal:    cbor.Rat{Rat: big.NewRat(67, 100)},
		},
		MinCommitteeSize:           7,
		CommitteeTermLimit:         146,          // ~2 years in epochs
		GovActionValidityPeriod:    6,            // epochs
		GovActionDeposit:           100000000000, // 100,000 ADA
		DRepDeposit:                500000000,    // 500 ADA
		DRepInactivityPeriod:       20,           // epochs
		MinFeeRefScriptCostPerByte: newRat(15, 1),
	}
}

// mockPlutusV1CostModel returns a mock Plutus V1 cost model (166 parameters)
func mockPlutusV1CostModel() []int64 {
	// These are representative values, not actual mainnet values
	// PlutusV1 has 166 parameters
	costModel := make([]int64, 166)

	// Set some representative values for common operations
	// Integer operations
	costModel[0] = 205665 // addInteger-cpu-arguments-intercept
	costModel[1] = 812    // addInteger-cpu-arguments-slope
	costModel[2] = 1      // addInteger-memory-arguments-intercept
	costModel[3] = 1      // addInteger-memory-arguments-slope

	// Fill remaining with reasonable defaults
	for i := 4; i < 166; i++ {
		costModel[i] = 1000 + int64(i*100)
	}

	return costModel
}

// mockPlutusV2CostModel returns a mock Plutus V2 cost model (175 parameters)
func mockPlutusV2CostModel() []int64 {
	// PlutusV2 has 175 parameters (166 from V1 + 9 new)
	costModel := make([]int64, 175)

	// Copy V1 model as base
	v1Model := mockPlutusV1CostModel()
	copy(costModel, v1Model)

	// Add V2 specific parameters
	for i := 166; i < 175; i++ {
		costModel[i] = 2000 + int64(i*50)
	}

	return costModel
}

// mockPlutusV3CostModel returns a mock Plutus V3 cost model (223 parameters)
func mockPlutusV3CostModel() []int64 {
	// PlutusV3 has 223 parameters
	costModel := make([]int64, 223)

	// Copy V2 model as base
	v2Model := mockPlutusV2CostModel()
	copy(costModel, v2Model)

	// Add V3 specific parameters
	for i := 175; i < 223; i++ {
		costModel[i] = 3000 + int64(i*50)
	}

	return costModel
}
