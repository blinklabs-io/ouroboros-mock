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

package fixtures

import (
	"maps"
	"math/big"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

// newRat creates a new GenesisRat from a numerator and denominator
func newRat(num, denom int64) *lcommon.GenesisRat {
	return &lcommon.GenesisRat{Rat: big.NewRat(num, denom)}
}

// MockByronGenesisConfig provides typical mainnet values for Byron genesis configuration
var MockByronGenesisConfig = byron.ByronGenesis{
	// Mainnet start time: September 23, 2017 21:44:51 UTC (Unix timestamp)
	StartTime: 1506203091,
	ProtocolConsts: byron.ByronGenesisProtocolConsts{
		K:             2160, // Security parameter (number of blocks)
		ProtocolMagic: 764824073,
	},
	BlockVersionData: byron.ByronGenesisBlockVersionData{
		ScriptVersion:     0,
		SlotDuration:      20000, // 20 seconds in milliseconds
		MaxBlockSize:      2000000,
		MaxHeaderSize:     2000000,
		MaxTxSize:         4096,
		MaxProposalSize:   700,
		MpcThd:            20000000000000,
		HeavyDelThd:       300000000000,
		UpdateVoteThd:     1000000000000,
		UpdateProposalThd: 100000000000000,
		UpdateImplicit:    10000,
		SoftforkRule: byron.ByronGenesisBlockVersionDataSoftforkRule{
			InitThd:      900000000000000,
			MinThd:       600000000000000,
			ThdDecrement: 50000000000000,
		},
		TxFeePolicy: byron.ByronGenesisBlockVersionDataTxFeePolicy{
			Summand:    155381,
			Multiplier: 44,
		},
		UnlockStakeEpoch: 18446744073709551615, // Max uint64 (never unlock in Byron)
	},
	AvvmDistr:        map[string]string{},
	NonAvvmBalances:  map[string]string{},
	BootStakeholders: map[string]int{},
	HeavyDelegation:  map[string]byron.ByronGenesisHeavyDelegation{},
	VssCerts:         map[string]byron.ByronGenesisVssCert{},
}

// MockShelleyGenesisConfig provides typical mainnet values for Shelley genesis configuration
var MockShelleyGenesisConfig = shelley.ShelleyGenesis{
	// SystemStart is the blockchain origin time (Byron mainnet start: September 23, 2017)
	// Note: This is NOT when Shelley era started, but when the chain itself started
	SystemStart:  time.Date(2017, 9, 23, 21, 44, 51, 0, time.UTC),
	NetworkMagic: 764824073, // Mainnet magic
	NetworkId:    "Mainnet",
	ActiveSlotsCoeff: lcommon.GenesisRat{
		Rat: big.NewRat(1, 20),
	}, // 0.05 (5% active slots)
	SecurityParam:     2160,                                      // k parameter
	EpochLength:       432000,                                    // 5 days in slots (432000 * 1 second)
	SlotsPerKESPeriod: 129600,                                    // ~1.5 days
	MaxKESEvolutions:  62,                                        // ~93 days max KES validity
	SlotLength:        lcommon.GenesisRat{Rat: big.NewRat(1, 1)}, // 1 second
	UpdateQuorum:      5,
	MaxLovelaceSupply: 45000000000000000, // 45 billion ADA
	ProtocolParameters: shelley.ShelleyGenesisProtocolParams{
		MinFeeA:            44,     // Fee coefficient A
		MinFeeB:            155381, // Fee coefficient B
		MaxBlockBodySize:   65536,  // 64KB
		MaxTxSize:          16384,  // 16KB
		MaxBlockHeaderSize: 1100,
		KeyDeposit:         2000000,         // 2 ADA
		PoolDeposit:        500000000,       // 500 ADA
		MaxEpoch:           18,              // Maximum epoch for pool retirement
		NOpt:               150,             // Target number of pools
		A0:                 newRat(3, 10),   // Pool pledge influence (0.3)
		Rho:                newRat(3, 1000), // Monetary expansion (0.003)
		Tau:                newRat(2, 10),   // Treasury cut (0.2)
		Decentralization: newRat(
			1,
			1,
		), // Fully federated (d=1); d=0 would be fully decentralized
		ProtocolVersion: struct {
			Major uint `json:"major"`
			Minor uint `json:"minor"`
		}{
			Major: 2,
			Minor: 0,
		},
		MinUtxoValue: 1000000,   // 1 ADA minimum UTxO
		MinPoolCost:  340000000, // 340 ADA minimum pool cost
	},
	GenDelegs:    map[string]map[string]string{},
	InitialFunds: map[string]uint64{},
	Staking:      shelley.GenesisStaking{},
}

// MockAlonzoGenesisConfig provides typical mainnet values for Alonzo genesis configuration.
// Alonzo introduced Plutus smart contracts and execution unit pricing.
var MockAlonzoGenesisConfig = alonzo.AlonzoGenesis{
	LovelacePerUtxoWord:  34482, // Cost per word of UTxO storage (~0.034 ADA)
	MaxValueSize:         5000,  // Maximum serialized size of a Value
	CollateralPercentage: 150,   // 150% collateral required for scripts
	MaxCollateralInputs:  3,     // Maximum number of collateral inputs
	ExecutionPrices: alonzo.AlonzoGenesisExecutionPrices{
		Steps: newRat(721, 10000000), // Price per CPU step (~0.0000721 ADA)
		Mem:   newRat(577, 10000),    // Price per memory unit (~0.0577 ADA)
	},
	MaxTxExUnits: alonzo.AlonzoGenesisExUnits{
		Mem:   10000000,    // 10M memory units per transaction
		Steps: 10000000000, // 10B CPU steps per transaction
	},
	MaxBlockExUnits: alonzo.AlonzoGenesisExUnits{
		Mem:   50000000,    // 50M memory units per block
		Steps: 40000000000, // 40B CPU steps per block
	},
	CostModels: map[string]alonzo.CostModel{
		// PlutusV1 cost model with mainnet values (abbreviated - real model has 166 parameters)
		"PlutusV1": {
			"addInteger-cpu-arguments-intercept":          205665,
			"addInteger-cpu-arguments-slope":              812,
			"addInteger-memory-arguments-intercept":       1,
			"addInteger-memory-arguments-slope":           1,
			"appendByteString-cpu-arguments-intercept":    1000,
			"appendByteString-cpu-arguments-slope":        571,
			"appendByteString-memory-arguments-intercept": 0,
			"appendByteString-memory-arguments-slope":     1,
			"appendString-cpu-arguments-intercept":        1000,
			"appendString-cpu-arguments-slope":            24177,
			"appendString-memory-arguments-intercept":     4,
			"appendString-memory-arguments-slope":         1,
			"bData-cpu-arguments":                         1000,
			"bData-memory-arguments":                      32,
			"blake2b_256-cpu-arguments-intercept":         117366,
			"blake2b_256-cpu-arguments-slope":             10475,
			"blake2b_256-memory-arguments":                4,
			"cekApplyCost-exBudgetCPU":                    23000,
			"cekApplyCost-exBudgetMemory":                 100,
			"cekBuiltinCost-exBudgetCPU":                  23000,
			"cekBuiltinCost-exBudgetMemory":               100,
			"cekConstCost-exBudgetCPU":                    23000,
			"cekConstCost-exBudgetMemory":                 100,
			"cekDelayCost-exBudgetCPU":                    23000,
			"cekDelayCost-exBudgetMemory":                 100,
			"cekForceCost-exBudgetCPU":                    23000,
			"cekForceCost-exBudgetMemory":                 100,
			"cekLamCost-exBudgetCPU":                      23000,
			"cekLamCost-exBudgetMemory":                   100,
			"cekStartupCost-exBudgetCPU":                  100,
			"cekStartupCost-exBudgetMemory":               100,
			"cekVarCost-exBudgetCPU":                      23000,
			"cekVarCost-exBudgetMemory":                   100,
		},
	},
}

// MockConwayGenesisConfig provides typical mainnet values for Conway genesis configuration.
// Conway introduced on-chain governance with DReps and constitutional committee.
var MockConwayGenesisConfig = conway.ConwayGenesis{
	PoolVotingThresholds: conway.ConwayGenesisPoolVotingThresholds{
		MotionNoConfidence:    newRat(51, 100), // 51%
		CommitteeNormal:       newRat(51, 100), // 51%
		CommitteeNoConfidence: newRat(51, 100), // 51%
		HardForkInitiation:    newRat(51, 100), // 51%
		PpSecurityGroup:       newRat(51, 100), // 51%
	},
	DRepVotingThresholds: conway.ConwayGenesisDRepVotingThresholds{
		MotionNoConfidence:    newRat(67, 100), // 67%
		CommitteeNormal:       newRat(67, 100), // 67%
		CommitteeNoConfidence: newRat(60, 100), // 60%
		UpdateToConstitution:  newRat(75, 100), // 75%
		HardForkInitiation:    newRat(60, 100), // 60%
		PpNetworkGroup:        newRat(67, 100), // 67%
		PpEconomicGroup:       newRat(67, 100), // 67%
		PpTechnicalGroup:      newRat(67, 100), // 67%
		PpGovGroup:            newRat(75, 100), // 75%
		TreasuryWithdrawal:    newRat(67, 100), // 67%
	},
	MinCommitteeSize:           7,             // Minimum constitutional committee size
	CommitteeTermLimit:         146,           // ~2 years in epochs (73 epochs/year)
	GovActionValidityPeriod:    6,             // 6 epochs (~30 days)
	GovActionDeposit:           100000000000,  // 100,000 ADA
	DRepDeposit:                500000000,     // 500 ADA
	DRepInactivityPeriod:       20,            // 20 epochs (~100 days)
	MinFeeRefScriptCostPerByte: newRat(15, 1), // 15 lovelace per byte
	PlutusV3CostModel:          []int64{},     // Empty for mock, would contain cost model parameters
	Constitution: conway.ConwayGenesisConstitution{
		Anchor: conway.ConwayGenesisConstitutionAnchor{
			Url:      "https://constitution.gov.cardano.org",
			DataHash: "0000000000000000000000000000000000000000000000000000000000000000",
		},
	},
	Committee: conway.ConwayGenesisCommittee{
		Members: map[string]int{},
		Threshold: map[string]int{
			"numerator":   2,
			"denominator": 3,
		}, // 2/3 majority
	},
}

// NewMockByronGenesis returns a copy of the mock Byron genesis configuration.
// This can be modified without affecting the global MockByronGenesisConfig.
func NewMockByronGenesis() byron.ByronGenesis {
	genesis := MockByronGenesisConfig
	// Deep copy maps
	genesis.AvvmDistr = make(map[string]string)
	genesis.NonAvvmBalances = make(map[string]string)
	genesis.BootStakeholders = make(map[string]int)
	genesis.HeavyDelegation = make(map[string]byron.ByronGenesisHeavyDelegation)
	genesis.VssCerts = make(map[string]byron.ByronGenesisVssCert)
	return genesis
}

// NewMockShelleyGenesis returns a copy of the mock Shelley genesis configuration.
// This can be modified without affecting the global MockShelleyGenesisConfig.
func NewMockShelleyGenesis() shelley.ShelleyGenesis {
	genesis := MockShelleyGenesisConfig
	// Deep copy maps
	genesis.GenDelegs = make(map[string]map[string]string)
	genesis.InitialFunds = make(map[string]uint64)
	// Deep copy rationals (they contain pointers)
	genesis.ActiveSlotsCoeff = lcommon.GenesisRat{Rat: big.NewRat(1, 20)}
	genesis.SlotLength = lcommon.GenesisRat{Rat: big.NewRat(1, 1)}
	genesis.ProtocolParameters.A0 = newRat(3, 10)
	genesis.ProtocolParameters.Rho = newRat(3, 1000)
	genesis.ProtocolParameters.Tau = newRat(2, 10)
	genesis.ProtocolParameters.Decentralization = newRat(1, 1)
	return genesis
}

// NewMockAlonzoGenesis returns a copy of the mock Alonzo genesis configuration.
// This can be modified without affecting the global MockAlonzoGenesisConfig.
func NewMockAlonzoGenesis() alonzo.AlonzoGenesis {
	genesis := MockAlonzoGenesisConfig
	// Deep copy maps
	genesis.CostModels = make(map[string]alonzo.CostModel)
	for k, v := range MockAlonzoGenesisConfig.CostModels {
		costModel := make(alonzo.CostModel)
		maps.Copy(costModel, v)
		genesis.CostModels[k] = costModel
	}
	// Deep copy rationals (they contain pointers)
	genesis.ExecutionPrices = alonzo.AlonzoGenesisExecutionPrices{
		Steps: newRat(721, 10000000),
		Mem:   newRat(577, 10000),
	}
	return genesis
}

// NewMockConwayGenesis returns a copy of the mock Conway genesis configuration.
// This can be modified without affecting the global MockConwayGenesisConfig.
func NewMockConwayGenesis() conway.ConwayGenesis {
	genesis := MockConwayGenesisConfig
	// Deep copy maps and slices
	genesis.PlutusV3CostModel = make([]int64, 0)
	genesis.Committee.Members = make(map[string]int)
	genesis.Committee.Threshold = map[string]int{
		"numerator":   2,
		"denominator": 3,
	}
	// Deep copy rationals (they contain pointers)
	genesis.PoolVotingThresholds = conway.ConwayGenesisPoolVotingThresholds{
		MotionNoConfidence:    newRat(51, 100),
		CommitteeNormal:       newRat(51, 100),
		CommitteeNoConfidence: newRat(51, 100),
		HardForkInitiation:    newRat(51, 100),
		PpSecurityGroup:       newRat(51, 100),
	}
	genesis.DRepVotingThresholds = conway.ConwayGenesisDRepVotingThresholds{
		MotionNoConfidence:    newRat(67, 100),
		CommitteeNormal:       newRat(67, 100),
		CommitteeNoConfidence: newRat(60, 100),
		UpdateToConstitution:  newRat(75, 100),
		HardForkInitiation:    newRat(60, 100),
		PpNetworkGroup:        newRat(67, 100),
		PpEconomicGroup:       newRat(67, 100),
		PpTechnicalGroup:      newRat(67, 100),
		PpGovGroup:            newRat(75, 100),
		TreasuryWithdrawal:    newRat(67, 100),
	}
	genesis.MinFeeRefScriptCostPerByte = newRat(15, 1)
	return genesis
}
