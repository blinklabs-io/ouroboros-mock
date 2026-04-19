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

package fixtures

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	gcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

var errUnsupportedDijkstraRefScriptFields = errors.New(
	"unsupported Dijkstra ref-script protocol parameter fields",
)

// DecodedProtocolParameterUpdate wraps an era-specific protocol-parameter
// update value decoded from a JSON fixture.
type DecodedProtocolParameterUpdate struct {
	Era string

	value any
}

// Value returns the underlying era-specific gouroboros update type.
func (u DecodedProtocolParameterUpdate) Value() any {
	return u.value
}

// Shelley returns the decoded Shelley protocol-parameter update when present.
func (u DecodedProtocolParameterUpdate) Shelley() (*shelley.ShelleyProtocolParameterUpdate, bool) {
	value, ok := u.value.(*shelley.ShelleyProtocolParameterUpdate)
	return value, ok
}

// Alonzo returns the decoded Alonzo protocol-parameter update when present.
func (u DecodedProtocolParameterUpdate) Alonzo() (*alonzo.AlonzoProtocolParameterUpdate, bool) {
	value, ok := u.value.(*alonzo.AlonzoProtocolParameterUpdate)
	return value, ok
}

// Babbage returns the decoded Babbage protocol-parameter update when present.
func (u DecodedProtocolParameterUpdate) Babbage() (*babbage.BabbageProtocolParameterUpdate, bool) {
	value, ok := u.value.(*babbage.BabbageProtocolParameterUpdate)
	return value, ok
}

// Conway returns the decoded Conway protocol-parameter update when present.
func (u DecodedProtocolParameterUpdate) Conway() (*conway.ConwayProtocolParameterUpdate, bool) {
	value, ok := u.value.(*conway.ConwayProtocolParameterUpdate)
	return value, ok
}

// ApplyTo applies the decoded update to the supplied protocol parameters.
func (u DecodedProtocolParameterUpdate) ApplyTo(
	params gcommon.ProtocolParameters,
) error {
	return ApplyProtocolParameterUpdate(params, u)
}

// DecodeProtocolParameters decodes a protocol-parameter JSON fixture into the
// corresponding gouroboros concrete type.
func (f Fixture) DecodeProtocolParameters() (gcommon.ProtocolParameters, error) {
	if f.Kind != KindProtocolParameters {
		return nil, fmt.Errorf(
			"fixture %s is not a protocol-parameters fixture",
			f.RelPath,
		)
	}

	var payload protocolParametersJSON
	if err := decodeFixtureJSON(f, &payload); err != nil {
		return nil, err
	}

	switch {
	case f.Repo == RepoCardanoAPI && f.Name == "LegacyProtocolParameters.json":
		return payload.toAlonzoProtocolParameters()
	case f.Era == "shelley":
		return payload.toShelleyProtocolParameters()
	case f.Era == "alonzo":
		return payload.toAlonzoProtocolParameters()
	case f.Era == "babbage":
		return payload.toBabbageProtocolParameters()
	case f.Era == "conway" ||
		(f.Era != "dijkstra" && f.Name == "conway.json"):
		return payload.toConwayProtocolParameters()
	case f.Era == "dijkstra":
		if err := payload.rejectUnsupportedDijkstraRefScriptFields(); err != nil {
			return nil, fmt.Errorf(
				"failed to decode Dijkstra protocol-parameters fixture %s: %w",
				f.RelPath,
				err,
			)
		}
		return payload.toConwayProtocolParameters()
	default:
		return nil, fmt.Errorf(
			"unsupported protocol-parameters fixture: %s",
			f.RelPath,
		)
	}
}

// DecodeProtocolParameterUpdate decodes a protocol-parameter update JSON
// fixture into the corresponding gouroboros concrete type.
func (f Fixture) DecodeProtocolParameterUpdate() (DecodedProtocolParameterUpdate, error) {
	if f.Kind != KindProtocolParametersUpdate {
		return DecodedProtocolParameterUpdate{}, fmt.Errorf(
			"fixture %s is not a protocol-parameters-update fixture",
			f.RelPath,
		)
	}

	var payload protocolParametersJSON
	if err := decodeFixtureJSON(f, &payload); err != nil {
		return DecodedProtocolParameterUpdate{}, err
	}

	switch f.Era {
	case "shelley":
		value, err := payload.toShelleyProtocolParameterUpdate()
		if err != nil {
			return DecodedProtocolParameterUpdate{}, err
		}
		return DecodedProtocolParameterUpdate{
			Era:   f.Era,
			value: value,
		}, nil
	case "alonzo":
		value, err := payload.toAlonzoProtocolParameterUpdate()
		if err != nil {
			return DecodedProtocolParameterUpdate{}, err
		}
		return DecodedProtocolParameterUpdate{
			Era:   f.Era,
			value: value,
		}, nil
	case "babbage":
		value := payload.toBabbageProtocolParameterUpdate()
		return DecodedProtocolParameterUpdate{
			Era:   f.Era,
			value: value,
		}, nil
	case "conway":
		value := payload.toConwayProtocolParameterUpdate()
		return DecodedProtocolParameterUpdate{
			Era:   f.Era,
			value: value,
		}, nil
	case "dijkstra":
		if err := payload.rejectUnsupportedDijkstraRefScriptFields(); err != nil {
			return DecodedProtocolParameterUpdate{}, fmt.Errorf(
				"failed to decode Dijkstra protocol-parameters update fixture %s: %w",
				f.RelPath,
				err,
			)
		}
		value := payload.toConwayProtocolParameterUpdate()
		return DecodedProtocolParameterUpdate{
			Era:   f.Era,
			value: value,
		}, nil
	default:
		return DecodedProtocolParameterUpdate{}, fmt.Errorf(
			"unsupported protocol-parameters-update fixture: %s",
			f.RelPath,
		)
	}
}

// ApplyProtocolParameterUpdate applies an era-specific update decoded from a
// fixture to the corresponding protocol parameters.
func ApplyProtocolParameterUpdate(
	params gcommon.ProtocolParameters,
	update DecodedProtocolParameterUpdate,
) error {
	switch pp := params.(type) {
	case *shelley.ShelleyProtocolParameters:
		u, ok := update.Shelley()
		if !ok {
			return fmt.Errorf("unexpected Shelley update type %T", update.Value())
		}
		pp.Update(u)
	case *alonzo.AlonzoProtocolParameters:
		u, ok := update.Alonzo()
		if !ok {
			return fmt.Errorf("unexpected Alonzo update type %T", update.Value())
		}
		pp.Update(u)
	case *babbage.BabbageProtocolParameters:
		u, ok := update.Babbage()
		if !ok {
			return fmt.Errorf("unexpected Babbage update type %T", update.Value())
		}
		pp.Update(u)
	case *conway.ConwayProtocolParameters:
		u, ok := update.Conway()
		if !ok {
			return fmt.Errorf("unexpected Conway update type %T", update.Value())
		}
		pp.Update(u)
	default:
		return fmt.Errorf("unsupported protocol parameters type %T", params)
	}
	return nil
}

func decodeFixtureJSON(f Fixture, v any) error {
	data, err := f.Read()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf(
			"failed to decode JSON fixture %s: %w",
			f.RelPath,
			err,
		)
	}
	return nil
}

type protocolParametersJSON struct {
	TxFeePerByte               *jsonUint                 `json:"txFeePerByte"`
	TxFeeFixed                 *jsonUint                 `json:"txFeeFixed"`
	MaxBlockBodySize           *jsonUint                 `json:"maxBlockBodySize"`
	MaxTxSize                  *jsonUint                 `json:"maxTxSize"`
	MaxBlockHeaderSize         *jsonUint                 `json:"maxBlockHeaderSize"`
	StakeAddressDeposit        *jsonUint                 `json:"stakeAddressDeposit"`
	StakePoolDeposit           *jsonUint                 `json:"stakePoolDeposit"`
	PoolRetireMaxEpoch         *jsonUint                 `json:"poolRetireMaxEpoch"`
	StakePoolTargetNum         *jsonUint                 `json:"stakePoolTargetNum"`
	PoolPledgeInfluence        *jsonRational             `json:"poolPledgeInfluence"`
	MonetaryExpansion          *jsonRational             `json:"monetaryExpansion"`
	TreasuryCut                *jsonRational             `json:"treasuryCut"`
	Decentralization           *jsonRational             `json:"decentralization"`
	ExtraPraosEntropy          json.RawMessage           `json:"extraPraosEntropy"`
	ProtocolVersion            *protocolVersionJSON      `json:"protocolVersion"`
	MinUTxOValue               *jsonUint                 `json:"minUTxOValue"`
	MinPoolCost                *jsonUint64               `json:"minPoolCost"`
	UtxoCostPerByte            *jsonUint64               `json:"utxoCostPerByte"`
	CostModels                 *costModelsJSON           `json:"costModels"`
	ExecutionUnitPrices        *executionUnitPricesJSON  `json:"executionUnitPrices"`
	MaxTxExecutionUnits        *exUnitsJSON              `json:"maxTxExecutionUnits"`
	MaxBlockExecutionUnits     *exUnitsJSON              `json:"maxBlockExecutionUnits"`
	MaxValueSize               *jsonUint                 `json:"maxValueSize"`
	CollateralPercentage       *jsonUint                 `json:"collateralPercentage"`
	MaxCollateralInputs        *jsonUint                 `json:"maxCollateralInputs"`
	PoolVotingThresholds       *poolVotingThresholdsJSON `json:"poolVotingThresholds"`
	DRepVotingThresholds       *dRepVotingThresholdsJSON `json:"dRepVotingThresholds"`
	CommitteeMinSize           *jsonUint                 `json:"committeeMinSize"`
	CommitteeMaxTermLength     *jsonUint64               `json:"committeeMaxTermLength"`
	GovActionLifetime          *jsonUint64               `json:"govActionLifetime"`
	GovActionDeposit           *jsonUint64               `json:"govActionDeposit"`
	DRepDeposit                *jsonUint64               `json:"dRepDeposit"`
	DRepActivity               *jsonUint64               `json:"dRepActivity"`
	MinFeeRefScriptCostPerByte *jsonRational             `json:"minFeeRefScriptCostPerByte"`

	// Dijkstra-era ref-script parameters. Parsed from JSON fixtures but not
	// yet mapped to gouroboros types (no corresponding struct fields exist).
	MaxRefScriptSizePerBlock *jsonUint     `json:"maxRefScriptSizePerBlock"`
	MaxRefScriptSizePerTx    *jsonUint     `json:"maxRefScriptSizePerTx"`
	RefScriptCostStride      *jsonUint     `json:"refScriptCostStride"`
	RefScriptCostMultiplier  *jsonRational `json:"refScriptCostMultiplier"`
}

func (p protocolParametersJSON) toShelleyProtocolParameters() (*shelley.ShelleyProtocolParameters, error) {
	version, err := p.requireProtocolVersion()
	if err != nil {
		return nil, err
	}
	extraEntropy, err := decodeNonce(p.ExtraPraosEntropy)
	if err != nil {
		return nil, fmt.Errorf("invalid extraPraosEntropy: %w", err)
	}
	minFeeA, err := requireUintField("txFeePerByte", p.TxFeePerByte)
	if err != nil {
		return nil, err
	}
	minFeeB, err := requireUintField("txFeeFixed", p.TxFeeFixed)
	if err != nil {
		return nil, err
	}
	maxBlockBodySize, err := requireUintField(
		"maxBlockBodySize",
		p.MaxBlockBodySize,
	)
	if err != nil {
		return nil, err
	}
	maxTxSize, err := requireUintField("maxTxSize", p.MaxTxSize)
	if err != nil {
		return nil, err
	}
	maxBlockHeaderSize, err := requireUintField(
		"maxBlockHeaderSize",
		p.MaxBlockHeaderSize,
	)
	if err != nil {
		return nil, err
	}
	keyDeposit, err := requireUintField(
		"stakeAddressDeposit",
		p.StakeAddressDeposit,
	)
	if err != nil {
		return nil, err
	}
	poolDeposit, err := requireUintField("stakePoolDeposit", p.StakePoolDeposit)
	if err != nil {
		return nil, err
	}
	maxEpoch, err := requireUintField(
		"poolRetireMaxEpoch",
		p.PoolRetireMaxEpoch,
	)
	if err != nil {
		return nil, err
	}
	nOpt, err := requireUintField("stakePoolTargetNum", p.StakePoolTargetNum)
	if err != nil {
		return nil, err
	}
	a0, err := requireRatField("poolPledgeInfluence", p.PoolPledgeInfluence)
	if err != nil {
		return nil, err
	}
	rho, err := requireRatField("monetaryExpansion", p.MonetaryExpansion)
	if err != nil {
		return nil, err
	}
	tau, err := requireRatField("treasuryCut", p.TreasuryCut)
	if err != nil {
		return nil, err
	}
	pp := &shelley.ShelleyProtocolParameters{
		MinFeeA:            minFeeA,
		MinFeeB:            minFeeB,
		MaxBlockBodySize:   maxBlockBodySize,
		MaxTxSize:          maxTxSize,
		MaxBlockHeaderSize: maxBlockHeaderSize,
		KeyDeposit:         keyDeposit,
		PoolDeposit:        poolDeposit,
		MaxEpoch:           maxEpoch,
		NOpt:               nOpt,
		A0:                 a0,
		Rho:                rho,
		Tau:                tau,
		Decentralization:   cloneRat(p.Decentralization),
		ExtraEntropy:       extraEntropy,
		ProtocolMajor:      version.Major,
		ProtocolMinor:      version.Minor,
		MinUtxoValue:       optionalUintValue(p.MinUTxOValue),
	}
	return pp, nil
}

func (p protocolParametersJSON) toAlonzoProtocolParameters() (*alonzo.AlonzoProtocolParameters, error) {
	version, err := p.requireProtocolVersion()
	if err != nil {
		return nil, err
	}
	extraEntropy, err := decodeNonce(p.ExtraPraosEntropy)
	if err != nil {
		return nil, fmt.Errorf("invalid extraPraosEntropy: %w", err)
	}
	minFeeA, err := requireUintField("txFeePerByte", p.TxFeePerByte)
	if err != nil {
		return nil, err
	}
	minFeeB, err := requireUintField("txFeeFixed", p.TxFeeFixed)
	if err != nil {
		return nil, err
	}
	maxBlockBodySize, err := requireUintField(
		"maxBlockBodySize",
		p.MaxBlockBodySize,
	)
	if err != nil {
		return nil, err
	}
	maxTxSize, err := requireUintField("maxTxSize", p.MaxTxSize)
	if err != nil {
		return nil, err
	}
	maxBlockHeaderSize, err := requireUintField(
		"maxBlockHeaderSize",
		p.MaxBlockHeaderSize,
	)
	if err != nil {
		return nil, err
	}
	keyDeposit, err := requireUintField(
		"stakeAddressDeposit",
		p.StakeAddressDeposit,
	)
	if err != nil {
		return nil, err
	}
	poolDeposit, err := requireUintField("stakePoolDeposit", p.StakePoolDeposit)
	if err != nil {
		return nil, err
	}
	maxEpoch, err := requireUintField(
		"poolRetireMaxEpoch",
		p.PoolRetireMaxEpoch,
	)
	if err != nil {
		return nil, err
	}
	nOpt, err := requireUintField("stakePoolTargetNum", p.StakePoolTargetNum)
	if err != nil {
		return nil, err
	}
	a0, err := requireRatField("poolPledgeInfluence", p.PoolPledgeInfluence)
	if err != nil {
		return nil, err
	}
	rho, err := requireRatField("monetaryExpansion", p.MonetaryExpansion)
	if err != nil {
		return nil, err
	}
	tau, err := requireRatField("treasuryCut", p.TreasuryCut)
	if err != nil {
		return nil, err
	}
	executionCosts, err := requireExUnitPriceField(
		"executionUnitPrices",
		p.ExecutionUnitPrices,
	)
	if err != nil {
		return nil, err
	}
	maxTxExUnits, err := requireExUnitsField(
		"maxTxExecutionUnits",
		p.MaxTxExecutionUnits,
	)
	if err != nil {
		return nil, err
	}
	maxBlockExUnits, err := requireExUnitsField(
		"maxBlockExecutionUnits",
		p.MaxBlockExecutionUnits,
	)
	if err != nil {
		return nil, err
	}
	pp := &alonzo.AlonzoProtocolParameters{
		MinFeeA:              minFeeA,
		MinFeeB:              minFeeB,
		MaxBlockBodySize:     maxBlockBodySize,
		MaxTxSize:            maxTxSize,
		MaxBlockHeaderSize:   maxBlockHeaderSize,
		KeyDeposit:           keyDeposit,
		PoolDeposit:          poolDeposit,
		MaxEpoch:             maxEpoch,
		NOpt:                 nOpt,
		A0:                   a0,
		Rho:                  rho,
		Tau:                  tau,
		Decentralization:     cloneRat(p.Decentralization),
		ExtraEntropy:         extraEntropy,
		ProtocolMajor:        version.Major,
		ProtocolMinor:        version.Minor,
		MinUtxoValue:         optionalUintValue(p.MinUTxOValue),
		MinPoolCost:          optionalUint64Value(p.MinPoolCost),
		AdaPerUtxoByte:       optionalUint64Value(p.UtxoCostPerByte),
		CostModels:           cloneCostModels(p.CostModels),
		ExecutionCosts:       executionCosts,
		MaxTxExUnits:         maxTxExUnits,
		MaxBlockExUnits:      maxBlockExUnits,
		MaxValueSize:         optionalUintValue(p.MaxValueSize),
		CollateralPercentage: optionalUintValue(p.CollateralPercentage),
		MaxCollateralInputs:  optionalUintValue(p.MaxCollateralInputs),
	}
	return pp, nil
}

func (p protocolParametersJSON) toBabbageProtocolParameters() (*babbage.BabbageProtocolParameters, error) {
	version, err := p.requireProtocolVersion()
	if err != nil {
		return nil, err
	}
	minFeeA, err := requireUintField("txFeePerByte", p.TxFeePerByte)
	if err != nil {
		return nil, err
	}
	minFeeB, err := requireUintField("txFeeFixed", p.TxFeeFixed)
	if err != nil {
		return nil, err
	}
	maxBlockBodySize, err := requireUintField(
		"maxBlockBodySize",
		p.MaxBlockBodySize,
	)
	if err != nil {
		return nil, err
	}
	maxTxSize, err := requireUintField("maxTxSize", p.MaxTxSize)
	if err != nil {
		return nil, err
	}
	maxBlockHeaderSize, err := requireUintField(
		"maxBlockHeaderSize",
		p.MaxBlockHeaderSize,
	)
	if err != nil {
		return nil, err
	}
	keyDeposit, err := requireUintField(
		"stakeAddressDeposit",
		p.StakeAddressDeposit,
	)
	if err != nil {
		return nil, err
	}
	poolDeposit, err := requireUintField("stakePoolDeposit", p.StakePoolDeposit)
	if err != nil {
		return nil, err
	}
	maxEpoch, err := requireUintField(
		"poolRetireMaxEpoch",
		p.PoolRetireMaxEpoch,
	)
	if err != nil {
		return nil, err
	}
	nOpt, err := requireUintField("stakePoolTargetNum", p.StakePoolTargetNum)
	if err != nil {
		return nil, err
	}
	a0, err := requireRatField("poolPledgeInfluence", p.PoolPledgeInfluence)
	if err != nil {
		return nil, err
	}
	rho, err := requireRatField("monetaryExpansion", p.MonetaryExpansion)
	if err != nil {
		return nil, err
	}
	tau, err := requireRatField("treasuryCut", p.TreasuryCut)
	if err != nil {
		return nil, err
	}
	executionCosts, err := requireExUnitPriceField(
		"executionUnitPrices",
		p.ExecutionUnitPrices,
	)
	if err != nil {
		return nil, err
	}
	maxTxExUnits, err := requireExUnitsField(
		"maxTxExecutionUnits",
		p.MaxTxExecutionUnits,
	)
	if err != nil {
		return nil, err
	}
	maxBlockExUnits, err := requireExUnitsField(
		"maxBlockExecutionUnits",
		p.MaxBlockExecutionUnits,
	)
	if err != nil {
		return nil, err
	}
	pp := &babbage.BabbageProtocolParameters{
		MinFeeA:              minFeeA,
		MinFeeB:              minFeeB,
		MaxBlockBodySize:     maxBlockBodySize,
		MaxTxSize:            maxTxSize,
		MaxBlockHeaderSize:   maxBlockHeaderSize,
		KeyDeposit:           keyDeposit,
		PoolDeposit:          poolDeposit,
		MaxEpoch:             maxEpoch,
		NOpt:                 nOpt,
		A0:                   a0,
		Rho:                  rho,
		Tau:                  tau,
		ProtocolMajor:        version.Major,
		ProtocolMinor:        version.Minor,
		MinPoolCost:          optionalUint64Value(p.MinPoolCost),
		AdaPerUtxoByte:       optionalUint64Value(p.UtxoCostPerByte),
		CostModels:           cloneCostModels(p.CostModels),
		ExecutionCosts:       executionCosts,
		MaxTxExUnits:         maxTxExUnits,
		MaxBlockExUnits:      maxBlockExUnits,
		MaxValueSize:         optionalUintValue(p.MaxValueSize),
		CollateralPercentage: optionalUintValue(p.CollateralPercentage),
		MaxCollateralInputs:  optionalUintValue(p.MaxCollateralInputs),
	}
	return pp, nil
}

func (p protocolParametersJSON) toConwayProtocolParameters() (*conway.ConwayProtocolParameters, error) {
	version, err := p.requireProtocolVersion()
	if err != nil {
		return nil, err
	}
	minFeeA, err := requireUintField("txFeePerByte", p.TxFeePerByte)
	if err != nil {
		return nil, err
	}
	minFeeB, err := requireUintField("txFeeFixed", p.TxFeeFixed)
	if err != nil {
		return nil, err
	}
	maxBlockBodySize, err := requireUintField(
		"maxBlockBodySize",
		p.MaxBlockBodySize,
	)
	if err != nil {
		return nil, err
	}
	maxTxSize, err := requireUintField("maxTxSize", p.MaxTxSize)
	if err != nil {
		return nil, err
	}
	maxBlockHeaderSize, err := requireUintField(
		"maxBlockHeaderSize",
		p.MaxBlockHeaderSize,
	)
	if err != nil {
		return nil, err
	}
	keyDeposit, err := requireUintField(
		"stakeAddressDeposit",
		p.StakeAddressDeposit,
	)
	if err != nil {
		return nil, err
	}
	poolDeposit, err := requireUintField("stakePoolDeposit", p.StakePoolDeposit)
	if err != nil {
		return nil, err
	}
	maxEpoch, err := requireUintField(
		"poolRetireMaxEpoch",
		p.PoolRetireMaxEpoch,
	)
	if err != nil {
		return nil, err
	}
	nOpt, err := requireUintField("stakePoolTargetNum", p.StakePoolTargetNum)
	if err != nil {
		return nil, err
	}
	a0, err := requireRatField("poolPledgeInfluence", p.PoolPledgeInfluence)
	if err != nil {
		return nil, err
	}
	rho, err := requireRatField("monetaryExpansion", p.MonetaryExpansion)
	if err != nil {
		return nil, err
	}
	tau, err := requireRatField("treasuryCut", p.TreasuryCut)
	if err != nil {
		return nil, err
	}
	executionCosts, err := requireExUnitPriceField(
		"executionUnitPrices",
		p.ExecutionUnitPrices,
	)
	if err != nil {
		return nil, err
	}
	maxTxExUnits, err := requireExUnitsField(
		"maxTxExecutionUnits",
		p.MaxTxExecutionUnits,
	)
	if err != nil {
		return nil, err
	}
	maxBlockExUnits, err := requireExUnitsField(
		"maxBlockExecutionUnits",
		p.MaxBlockExecutionUnits,
	)
	if err != nil {
		return nil, err
	}
	pp := &conway.ConwayProtocolParameters{
		MinFeeA:              minFeeA,
		MinFeeB:              minFeeB,
		MaxBlockBodySize:     maxBlockBodySize,
		MaxTxSize:            maxTxSize,
		MaxBlockHeaderSize:   maxBlockHeaderSize,
		KeyDeposit:           keyDeposit,
		PoolDeposit:          poolDeposit,
		MaxEpoch:             maxEpoch,
		NOpt:                 nOpt,
		A0:                   a0,
		Rho:                  rho,
		Tau:                  tau,
		ProtocolVersion:      *version,
		MinPoolCost:          optionalUint64Value(p.MinPoolCost),
		AdaPerUtxoByte:       optionalUint64Value(p.UtxoCostPerByte),
		CostModels:           cloneCostModels(p.CostModels),
		ExecutionCosts:       executionCosts,
		MaxTxExUnits:         maxTxExUnits,
		MaxBlockExUnits:      maxBlockExUnits,
		MaxValueSize:         optionalUintValue(p.MaxValueSize),
		CollateralPercentage: optionalUintValue(p.CollateralPercentage),
		MaxCollateralInputs:  optionalUintValue(p.MaxCollateralInputs),
		PoolVotingThresholds: clonePoolVotingThresholdsValue(
			p.PoolVotingThresholds,
		),
		DRepVotingThresholds: cloneDRepVotingThresholdsValue(
			p.DRepVotingThresholds,
		),
		MinCommitteeSize: optionalUintValue(p.CommitteeMinSize),
		CommitteeTermLimit: optionalUint64Value(
			p.CommitteeMaxTermLength,
		),
		GovActionValidityPeriod:    optionalUint64Value(p.GovActionLifetime),
		GovActionDeposit:           optionalUint64Value(p.GovActionDeposit),
		DRepDeposit:                optionalUint64Value(p.DRepDeposit),
		DRepInactivityPeriod:       optionalUint64Value(p.DRepActivity),
		MinFeeRefScriptCostPerByte: cloneRat(p.MinFeeRefScriptCostPerByte),
	}
	return pp, nil
}

func (p protocolParametersJSON) toShelleyProtocolParameterUpdate() (*shelley.ShelleyProtocolParameterUpdate, error) {
	update := &shelley.ShelleyProtocolParameterUpdate{
		MinFeeA:            cloneUintPtr(p.TxFeePerByte),
		MinFeeB:            cloneUintPtr(p.TxFeeFixed),
		MaxBlockBodySize:   cloneUintPtr(p.MaxBlockBodySize),
		MaxTxSize:          cloneUintPtr(p.MaxTxSize),
		MaxBlockHeaderSize: cloneUintPtr(p.MaxBlockHeaderSize),
		KeyDeposit:         cloneUintPtr(p.StakeAddressDeposit),
		PoolDeposit:        cloneUintPtr(p.StakePoolDeposit),
		MaxEpoch:           cloneUintPtr(p.PoolRetireMaxEpoch),
		NOpt:               cloneUintPtr(p.StakePoolTargetNum),
		A0:                 cloneRat(p.PoolPledgeInfluence),
		Rho:                cloneRat(p.MonetaryExpansion),
		Tau:                cloneRat(p.TreasuryCut),
		Decentralization:   cloneRat(p.Decentralization),
		MinUtxoValue:       cloneUintPtr(p.MinUTxOValue),
	}
	if rawMessagePresent(p.ExtraPraosEntropy) {
		nonce, err := decodeNonce(p.ExtraPraosEntropy)
		if err != nil {
			return nil, fmt.Errorf("invalid extraPraosEntropy: %w", err)
		}
		update.ExtraEntropy = &nonce
	}
	if p.ProtocolVersion != nil {
		version := p.ProtocolVersion.toCommon()
		update.ProtocolVersion = &version
	}
	return update, nil
}

func (p protocolParametersJSON) toAlonzoProtocolParameterUpdate() (*alonzo.AlonzoProtocolParameterUpdate, error) {
	update := &alonzo.AlonzoProtocolParameterUpdate{
		MinFeeA:              cloneUintPtr(p.TxFeePerByte),
		MinFeeB:              cloneUintPtr(p.TxFeeFixed),
		MaxBlockBodySize:     cloneUintPtr(p.MaxBlockBodySize),
		MaxTxSize:            cloneUintPtr(p.MaxTxSize),
		MaxBlockHeaderSize:   cloneUintPtr(p.MaxBlockHeaderSize),
		KeyDeposit:           cloneUintPtr(p.StakeAddressDeposit),
		PoolDeposit:          cloneUintPtr(p.StakePoolDeposit),
		MaxEpoch:             cloneUintPtr(p.PoolRetireMaxEpoch),
		NOpt:                 cloneUintPtr(p.StakePoolTargetNum),
		A0:                   cloneRat(p.PoolPledgeInfluence),
		Rho:                  cloneRat(p.MonetaryExpansion),
		Tau:                  cloneRat(p.TreasuryCut),
		Decentralization:     cloneRat(p.Decentralization),
		MinUtxoValue:         cloneUintPtr(p.MinUTxOValue),
		MinPoolCost:          cloneUint64Ptr(p.MinPoolCost),
		AdaPerUtxoByte:       cloneUint64Ptr(p.UtxoCostPerByte),
		CostModels:           cloneCostModels(p.CostModels),
		ExecutionCosts:       cloneExUnitPrice(p.ExecutionUnitPrices),
		MaxTxExUnits:         cloneExUnitsPtr(p.MaxTxExecutionUnits),
		MaxBlockExUnits:      cloneExUnitsPtr(p.MaxBlockExecutionUnits),
		MaxValueSize:         cloneUintPtr(p.MaxValueSize),
		CollateralPercentage: cloneUintPtr(p.CollateralPercentage),
		MaxCollateralInputs:  cloneUintPtr(p.MaxCollateralInputs),
	}
	if rawMessagePresent(p.ExtraPraosEntropy) {
		nonce, err := decodeNonce(p.ExtraPraosEntropy)
		if err != nil {
			return nil, fmt.Errorf("invalid extraPraosEntropy: %w", err)
		}
		update.ExtraEntropy = &nonce
	}
	if p.ProtocolVersion != nil {
		version := p.ProtocolVersion.toCommon()
		update.ProtocolVersion = &version
	}
	return update, nil
}

func (p protocolParametersJSON) toBabbageProtocolParameterUpdate() *babbage.BabbageProtocolParameterUpdate {
	update := &babbage.BabbageProtocolParameterUpdate{
		MinFeeA:              cloneUintPtr(p.TxFeePerByte),
		MinFeeB:              cloneUintPtr(p.TxFeeFixed),
		MaxBlockBodySize:     cloneUintPtr(p.MaxBlockBodySize),
		MaxTxSize:            cloneUintPtr(p.MaxTxSize),
		MaxBlockHeaderSize:   cloneUintPtr(p.MaxBlockHeaderSize),
		KeyDeposit:           cloneUintPtr(p.StakeAddressDeposit),
		PoolDeposit:          cloneUintPtr(p.StakePoolDeposit),
		MaxEpoch:             cloneUintPtr(p.PoolRetireMaxEpoch),
		NOpt:                 cloneUintPtr(p.StakePoolTargetNum),
		A0:                   cloneRat(p.PoolPledgeInfluence),
		Rho:                  cloneRat(p.MonetaryExpansion),
		Tau:                  cloneRat(p.TreasuryCut),
		MinPoolCost:          cloneUint64Ptr(p.MinPoolCost),
		AdaPerUtxoByte:       cloneUint64Ptr(p.UtxoCostPerByte),
		CostModels:           cloneCostModels(p.CostModels),
		ExecutionCosts:       cloneExUnitPrice(p.ExecutionUnitPrices),
		MaxTxExUnits:         cloneExUnitsPtr(p.MaxTxExecutionUnits),
		MaxBlockExUnits:      cloneExUnitsPtr(p.MaxBlockExecutionUnits),
		MaxValueSize:         cloneUintPtr(p.MaxValueSize),
		CollateralPercentage: cloneUintPtr(p.CollateralPercentage),
		MaxCollateralInputs:  cloneUintPtr(p.MaxCollateralInputs),
	}
	if p.ProtocolVersion != nil {
		version := p.ProtocolVersion.toCommon()
		update.ProtocolVersion = &version
	}
	return update
}

func (p protocolParametersJSON) toConwayProtocolParameterUpdate() *conway.ConwayProtocolParameterUpdate {
	update := &conway.ConwayProtocolParameterUpdate{
		MinFeeA:              cloneUintPtr(p.TxFeePerByte),
		MinFeeB:              cloneUintPtr(p.TxFeeFixed),
		MaxBlockBodySize:     cloneUintPtr(p.MaxBlockBodySize),
		MaxTxSize:            cloneUintPtr(p.MaxTxSize),
		MaxBlockHeaderSize:   cloneUintPtr(p.MaxBlockHeaderSize),
		KeyDeposit:           cloneUintPtr(p.StakeAddressDeposit),
		PoolDeposit:          cloneUintPtr(p.StakePoolDeposit),
		MaxEpoch:             cloneUintPtr(p.PoolRetireMaxEpoch),
		NOpt:                 cloneUintPtr(p.StakePoolTargetNum),
		A0:                   cloneRat(p.PoolPledgeInfluence),
		Rho:                  cloneRat(p.MonetaryExpansion),
		Tau:                  cloneRat(p.TreasuryCut),
		MinPoolCost:          cloneUint64Ptr(p.MinPoolCost),
		AdaPerUtxoByte:       cloneUint64Ptr(p.UtxoCostPerByte),
		CostModels:           cloneCostModels(p.CostModels),
		ExecutionCosts:       cloneExUnitPrice(p.ExecutionUnitPrices),
		MaxTxExUnits:         cloneExUnitsPtr(p.MaxTxExecutionUnits),
		MaxBlockExUnits:      cloneExUnitsPtr(p.MaxBlockExecutionUnits),
		MaxValueSize:         cloneUintPtr(p.MaxValueSize),
		CollateralPercentage: cloneUintPtr(p.CollateralPercentage),
		MaxCollateralInputs:  cloneUintPtr(p.MaxCollateralInputs),
		PoolVotingThresholds: clonePoolVotingThresholdsPtr(
			p.PoolVotingThresholds,
		),
		DRepVotingThresholds: cloneDRepVotingThresholdsPtr(
			p.DRepVotingThresholds,
		),
		MinCommitteeSize:           cloneUintPtr(p.CommitteeMinSize),
		CommitteeTermLimit:         cloneUint64Ptr(p.CommitteeMaxTermLength),
		GovActionValidityPeriod:    cloneUint64Ptr(p.GovActionLifetime),
		GovActionDeposit:           cloneUint64Ptr(p.GovActionDeposit),
		DRepDeposit:                cloneUint64Ptr(p.DRepDeposit),
		DRepInactivityPeriod:       cloneUint64Ptr(p.DRepActivity),
		MinFeeRefScriptCostPerByte: cloneRat(p.MinFeeRefScriptCostPerByte),
	}
	if p.ProtocolVersion != nil {
		version := p.ProtocolVersion.toCommon()
		update.ProtocolVersion = &version
	}
	return update
}

func (p protocolParametersJSON) rejectUnsupportedDijkstraRefScriptFields() error {
	fields := p.dijkstraRefScriptFields()
	if len(fields) == 0 {
		return nil
	}
	return fmt.Errorf(
		"%w: %s",
		errUnsupportedDijkstraRefScriptFields,
		strings.Join(fields, ", "),
	)
}

func (p protocolParametersJSON) dijkstraRefScriptFields() []string {
	var fields []string
	if p.MaxRefScriptSizePerBlock != nil {
		fields = append(fields, "maxRefScriptSizePerBlock")
	}
	if p.MaxRefScriptSizePerTx != nil {
		fields = append(fields, "maxRefScriptSizePerTx")
	}
	if p.RefScriptCostStride != nil {
		fields = append(fields, "refScriptCostStride")
	}
	if p.RefScriptCostMultiplier != nil {
		fields = append(fields, "refScriptCostMultiplier")
	}
	return fields
}

func (p protocolParametersJSON) requireProtocolVersion() (*gcommon.ProtocolParametersProtocolVersion, error) {
	if p.ProtocolVersion == nil {
		return nil, errors.New("missing protocolVersion")
	}
	version := p.ProtocolVersion.toCommon()
	return &version, nil
}

type protocolVersionJSON struct {
	Major jsonUint `json:"major"`
	Minor jsonUint `json:"minor"`
}

func (p protocolVersionJSON) toCommon() gcommon.ProtocolParametersProtocolVersion {
	return gcommon.ProtocolParametersProtocolVersion{
		Major: p.Major.value,
		Minor: p.Minor.value,
	}
}

type jsonUint struct {
	value uint
}

func (u *jsonUint) UnmarshalJSON(data []byte) error {
	value, err := parseJSONUint(strings.TrimSpace(string(data)))
	if err != nil {
		return err
	}
	u.value = value
	return nil
}

type jsonUint64 struct {
	value uint64
}

func (u *jsonUint64) UnmarshalJSON(data []byte) error {
	value, err := parseJSONUint64(strings.TrimSpace(string(data)))
	if err != nil {
		return err
	}
	u.value = value
	return nil
}

type exUnitsJSON struct {
	Memory int64 `json:"memory"`
	Steps  int64 `json:"steps"`
}

func (e exUnitsJSON) toCommon() gcommon.ExUnits {
	return gcommon.ExUnits{
		Memory: e.Memory,
		Steps:  e.Steps,
	}
}

type executionUnitPricesJSON struct {
	Memory *jsonRational
	Steps  *jsonRational
}

func (e *executionUnitPricesJSON) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if memRaw, ok := raw["priceMemory"]; ok {
		e.Memory = &jsonRational{}
		if err := e.Memory.UnmarshalJSON(memRaw); err != nil {
			return err
		}
	} else if memRaw, ok := raw["prMem"]; ok {
		e.Memory = &jsonRational{}
		if err := e.Memory.UnmarshalJSON(memRaw); err != nil {
			return err
		}
	}
	if stepsRaw, ok := raw["priceSteps"]; ok {
		e.Steps = &jsonRational{}
		if err := e.Steps.UnmarshalJSON(stepsRaw); err != nil {
			return err
		}
	} else if stepsRaw, ok := raw["prSteps"]; ok {
		e.Steps = &jsonRational{}
		if err := e.Steps.UnmarshalJSON(stepsRaw); err != nil {
			return err
		}
	}
	return nil
}

func (e *executionUnitPricesJSON) toCommon() gcommon.ExUnitPrice {
	return gcommon.ExUnitPrice{
		MemPrice:  cloneRat(e.Memory),
		StepPrice: cloneRat(e.Steps),
	}
}

type costModelsJSON struct {
	Models map[uint][]int64
}

func (c *costModelsJSON) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.Models = make(map[uint][]int64)
	for key, value := range raw {
		if key == "Unknown" {
			var unknown map[string][]int64
			if err := json.Unmarshal(value, &unknown); err != nil {
				return err
			}
			for versionKey, model := range unknown {
				version, err := strconv.ParseUint(versionKey, 10, 32)
				if err != nil {
					return fmt.Errorf(
						"invalid cost model version %q: %w",
						versionKey,
						err,
					)
				}
				c.Models[uint(version)] = append([]int64(nil), model...)
			}
			continue
		}

		var model []int64
		if err := json.Unmarshal(value, &model); err != nil {
			return err
		}
		version, ok := knownCostModelVersion(key)
		if !ok {
			parsedVersion, err := strconv.ParseUint(key, 10, 32)
			if err != nil {
				return fmt.Errorf("unsupported cost model key %q", key)
			}
			version = uint(parsedVersion)
		}
		c.Models[version] = append([]int64(nil), model...)
	}
	return nil
}

type jsonRational struct {
	value *cbor.Rat
}

func (r *jsonRational) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		r.value = nil
		return nil
	}

	if strings.HasPrefix(raw, "{") {
		var tmp struct {
			Numerator   json.Number `json:"numerator"`
			Denominator json.Number `json:"denominator"`
		}
		if err := json.Unmarshal(data, &tmp); err != nil {
			return err
		}
		numerator, err := parseBigIntJSONNumber(tmp.Numerator)
		if err != nil {
			return err
		}
		denominator, err := parseBigIntJSONNumber(tmp.Denominator)
		if err != nil {
			return err
		}
		if denominator.Sign() == 0 {
			return errors.New("rational denominator must be non-zero")
		}
		r.value = &cbor.Rat{Rat: new(big.Rat).SetFrac(numerator, denominator)}
		return nil
	}

	rat, err := parseJSONDecimalRat(raw)
	if err != nil {
		return err
	}
	r.value = &cbor.Rat{Rat: rat}
	return nil
}

type poolVotingThresholdsJSON struct {
	MotionNoConfidence    *jsonRational `json:"motionNoConfidence"`
	CommitteeNormal       *jsonRational `json:"committeeNormal"`
	CommitteeNoConfidence *jsonRational `json:"committeeNoConfidence"`
	HardForkInitiation    *jsonRational `json:"hardForkInitiation"`
	PpSecurityGroup       *jsonRational `json:"ppSecurityGroup"`
}

type dRepVotingThresholdsJSON struct {
	MotionNoConfidence    *jsonRational `json:"motionNoConfidence"`
	CommitteeNormal       *jsonRational `json:"committeeNormal"`
	CommitteeNoConfidence *jsonRational `json:"committeeNoConfidence"`
	UpdateToConstitution  *jsonRational `json:"updateToConstitution"`
	HardForkInitiation    *jsonRational `json:"hardForkInitiation"`
	PpNetworkGroup        *jsonRational `json:"ppNetworkGroup"`
	PpEconomicGroup       *jsonRational `json:"ppEconomicGroup"`
	PpTechnicalGroup      *jsonRational `json:"ppTechnicalGroup"`
	PpGovGroup            *jsonRational `json:"ppGovGroup"`
	TreasuryWithdrawal    *jsonRational `json:"treasuryWithdrawal"`
}

func cloneRat(r *jsonRational) *cbor.Rat {
	if r == nil || r.value == nil {
		return nil
	}
	return &cbor.Rat{Rat: new(big.Rat).Set(r.value.Rat)}
}

func cloneCostModels(c *costModelsJSON) map[uint][]int64 {
	if c == nil || len(c.Models) == 0 {
		return nil
	}
	ret := make(map[uint][]int64, len(c.Models))
	for version, model := range c.Models {
		ret[version] = append([]int64(nil), model...)
	}
	return ret
}

func cloneExUnitPrice(e *executionUnitPricesJSON) *gcommon.ExUnitPrice {
	if e == nil {
		return nil
	}
	price := e.toCommon()
	return &price
}

func cloneExUnitsPtr(e *exUnitsJSON) *gcommon.ExUnits {
	if e == nil {
		return nil
	}
	exUnits := e.toCommon()
	return &exUnits
}

func clonePoolVotingThresholdsValue(
	t *poolVotingThresholdsJSON,
) conway.PoolVotingThresholds {
	if t == nil {
		return conway.PoolVotingThresholds{}
	}
	return conway.PoolVotingThresholds{
		MotionNoConfidence:    derefRat(cloneRat(t.MotionNoConfidence)),
		CommitteeNormal:       derefRat(cloneRat(t.CommitteeNormal)),
		CommitteeNoConfidence: derefRat(cloneRat(t.CommitteeNoConfidence)),
		HardForkInitiation:    derefRat(cloneRat(t.HardForkInitiation)),
		PpSecurityGroup:       derefRat(cloneRat(t.PpSecurityGroup)),
	}
}

func clonePoolVotingThresholdsPtr(
	t *poolVotingThresholdsJSON,
) *conway.PoolVotingThresholds {
	if t == nil {
		return nil
	}
	thresholds := clonePoolVotingThresholdsValue(t)
	return &thresholds
}

func cloneDRepVotingThresholdsValue(
	t *dRepVotingThresholdsJSON,
) conway.DRepVotingThresholds {
	if t == nil {
		return conway.DRepVotingThresholds{}
	}
	return conway.DRepVotingThresholds{
		MotionNoConfidence:    derefRat(cloneRat(t.MotionNoConfidence)),
		CommitteeNormal:       derefRat(cloneRat(t.CommitteeNormal)),
		CommitteeNoConfidence: derefRat(cloneRat(t.CommitteeNoConfidence)),
		UpdateToConstitution:  derefRat(cloneRat(t.UpdateToConstitution)),
		HardForkInitiation:    derefRat(cloneRat(t.HardForkInitiation)),
		PpNetworkGroup:        derefRat(cloneRat(t.PpNetworkGroup)),
		PpEconomicGroup:       derefRat(cloneRat(t.PpEconomicGroup)),
		PpTechnicalGroup:      derefRat(cloneRat(t.PpTechnicalGroup)),
		PpGovGroup:            derefRat(cloneRat(t.PpGovGroup)),
		TreasuryWithdrawal:    derefRat(cloneRat(t.TreasuryWithdrawal)),
	}
}

func cloneDRepVotingThresholdsPtr(
	t *dRepVotingThresholdsJSON,
) *conway.DRepVotingThresholds {
	if t == nil {
		return nil
	}
	thresholds := cloneDRepVotingThresholdsValue(t)
	return &thresholds
}

func derefRat(r *cbor.Rat) cbor.Rat {
	if r == nil {
		return cbor.Rat{}
	}
	return *r
}

func decodeNonce(data json.RawMessage) (gcommon.Nonce, error) {
	neutralNonce := gcommon.Nonce{Type: gcommon.NonceTypeNeutral}
	if !rawMessagePresent(data) {
		return neutralNonce, nil
	}
	var hexStr string
	if err := json.Unmarshal(data, &hexStr); err != nil {
		return neutralNonce, fmt.Errorf(
			"failed to decode nonce as JSON string: %w",
			err,
		)
	}
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return neutralNonce, fmt.Errorf("failed to decode nonce hex: %w", err)
	}
	if len(decoded) != 32 {
		return neutralNonce, fmt.Errorf(
			"expected 32-byte nonce, got %d bytes",
			len(decoded),
		)
	}
	nonce := gcommon.Nonce{Type: gcommon.NonceTypeNonce}
	copy(nonce.Value[:], decoded)
	return nonce, nil
}

func rawMessagePresent(data json.RawMessage) bool {
	raw := strings.TrimSpace(string(data))
	return raw != "" && raw != "null"
}

func requireUintField(name string, value *jsonUint) (uint, error) {
	if value == nil {
		return 0, fmt.Errorf("missing required uint field %s", name)
	}
	return value.value, nil
}

func requireRatField(name string, value *jsonRational) (*cbor.Rat, error) {
	if value == nil || value.value == nil {
		return nil, fmt.Errorf("missing required rational field %s", name)
	}
	return cloneRat(value), nil
}

func requireExUnitPriceField(
	name string,
	value *executionUnitPricesJSON,
) (gcommon.ExUnitPrice, error) {
	if value == nil {
		return gcommon.ExUnitPrice{}, fmt.Errorf(
			"missing required ex-unit price field %s",
			name,
		)
	}
	return value.toCommon(), nil
}

func requireExUnitsField(
	name string,
	value *exUnitsJSON,
) (gcommon.ExUnits, error) {
	if value == nil {
		return gcommon.ExUnits{}, fmt.Errorf(
			"missing required ex-units field %s",
			name,
		)
	}
	return value.toCommon(), nil
}

func optionalUintValue(value *jsonUint) uint {
	if value == nil {
		return 0
	}
	return value.value
}

func optionalUint64Value(value *jsonUint64) uint64 {
	if value == nil {
		return 0
	}
	return value.value
}

func knownCostModelVersion(name string) (uint, bool) {
	switch name {
	case "PlutusV1":
		return 0, true
	case "PlutusV2":
		return 1, true
	case "PlutusV3":
		return 2, true
	default:
		return 0, false
	}
}

func parseJSONDecimalRat(raw string) (*big.Rat, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("empty rational value")
	}

	sign := 1
	switch raw[0] {
	case '-':
		sign = -1
		raw = raw[1:]
	case '+':
		raw = raw[1:]
	}

	exponent := 0
	if idx := strings.IndexAny(raw, "eE"); idx >= 0 {
		expValue, err := strconv.Atoi(raw[idx+1:])
		if err != nil {
			return nil, fmt.Errorf("invalid rational exponent %q: %w", raw, err)
		}
		exponent = expValue
		raw = raw[:idx]
	}

	intPart := raw
	fracPart := ""
	if idx := strings.IndexByte(raw, '.'); idx >= 0 {
		intPart = raw[:idx]
		fracPart = raw[idx+1:]
	}

	digits := intPart + fracPart
	if digits == "" {
		digits = "0"
	}
	numerator := new(big.Int)
	if _, ok := numerator.SetString(digits, 10); !ok {
		return nil, fmt.Errorf("invalid rational digits %q", digits)
	}
	if sign < 0 {
		numerator.Neg(numerator)
	}

	scale := len(fracPart) - exponent
	if scale <= 0 {
		numerator.Mul(numerator, pow10(-scale))
		return new(big.Rat).SetInt(numerator), nil
	}
	return new(big.Rat).SetFrac(numerator, pow10(scale)), nil
}

func parseBigIntJSONNumber(value json.Number) (*big.Int, error) {
	n := new(big.Int)
	if _, ok := n.SetString(value.String(), 10); !ok {
		return nil, fmt.Errorf("invalid integer value %q", value.String())
	}
	return n, nil
}

func cloneUintPtr(value *jsonUint) *uint {
	if value == nil {
		return nil
	}
	clone := value.value
	return &clone
}

func cloneUint64Ptr(value *jsonUint64) *uint64 {
	if value == nil {
		return nil
	}
	clone := value.value
	return &clone
}

func parseJSONUint(raw string) (uint, error) {
	value, err := parseJSONUint64(raw)
	if err != nil {
		return 0, err
	}
	if value > uint64(^uint(0)) {
		return 0, fmt.Errorf("integer value %q exceeds uint range", raw)
	}
	return uint(value), nil
}

func parseJSONUint64(raw string) (uint64, error) {
	rat, err := parseJSONDecimalRat(raw)
	if err != nil {
		return 0, err
	}
	if rat.Sign() < 0 {
		return 0, fmt.Errorf("integer value %q must be non-negative", raw)
	}
	if !rat.IsInt() {
		return 0, fmt.Errorf("integer value %q must be whole", raw)
	}
	numerator := rat.Num()
	if !numerator.IsUint64() {
		return 0, fmt.Errorf("integer value %q exceeds uint64 range", raw)
	}
	return numerator.Uint64(), nil
}

func pow10(exp int) *big.Int {
	if exp <= 0 {
		return big.NewInt(1)
	}
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil)
}
