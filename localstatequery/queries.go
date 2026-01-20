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

package localstatequery

import (
	"time"

	"github.com/blinklabs-io/gouroboros/cbor"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
)

// Query type constants from gouroboros
const (
	QueryTypeBlock       = localstatequery.QueryTypeBlock
	QueryTypeSystemStart = localstatequery.QueryTypeSystemStart
	QueryTypeChainPoint  = localstatequery.QueryTypeChainPoint

	QueryTypeShelley  = localstatequery.QueryTypeShelley
	QueryTypeHardFork = localstatequery.QueryTypeHardFork

	QueryTypeHardForkCurrentEra = localstatequery.QueryTypeHardForkCurrentEra

	QueryTypeShelleyEpochNo                              = localstatequery.QueryTypeShelleyEpochNo
	QueryTypeShelleyCurrentProtocolParams                = localstatequery.QueryTypeShelleyCurrentProtocolParams
	QueryTypeShelleyStakeDistribution                    = localstatequery.QueryTypeShelleyStakeDistribution
	QueryTypeShelleyUtxoByAddress                        = localstatequery.QueryTypeShelleyUtxoByAddress
	QueryTypeShelleyFilteredDelegationsAndRewardAccounts = localstatequery.QueryTypeShelleyFilteredDelegationAndRewardAccounts
)

// Query builders for common query types

// NewCurrentEraQuery builds a query for the current era
func NewCurrentEraQuery() ([]byte, error) {
	// Query structure: [QueryTypeBlock, [QueryTypeHardFork, [QueryTypeHardForkCurrentEra]]]
	query := cbor.IndefLengthList{
		QueryTypeBlock,
		cbor.IndefLengthList{
			QueryTypeHardFork,
			cbor.IndefLengthList{
				QueryTypeHardForkCurrentEra,
			},
		},
	}
	return cbor.Encode(query)
}

// NewEpochNoQuery builds a query for the current epoch number
func NewEpochNoQuery(era int) ([]byte, error) {
	// Query structure: [QueryTypeBlock, [QueryTypeShelley, era, [QueryTypeShelleyEpochNo]]]
	query := cbor.IndefLengthList{
		QueryTypeBlock,
		cbor.IndefLengthList{
			QueryTypeShelley,
			era,
			cbor.IndefLengthList{
				QueryTypeShelleyEpochNo,
			},
		},
	}
	return cbor.Encode(query)
}

// NewSystemStartQuery builds a query for the system start time
func NewSystemStartQuery() ([]byte, error) {
	// Query structure: [QueryTypeSystemStart]
	query := cbor.IndefLengthList{
		QueryTypeSystemStart,
	}
	return cbor.Encode(query)
}

// NewChainPointQuery builds a query for the current chain point
func NewChainPointQuery() ([]byte, error) {
	// Query structure: [QueryTypeChainPoint]
	query := cbor.IndefLengthList{
		QueryTypeChainPoint,
	}
	return cbor.Encode(query)
}

// NewProtocolParamsQuery builds a query for the current protocol parameters
func NewProtocolParamsQuery(era int) ([]byte, error) {
	// Query structure: [QueryTypeBlock, [QueryTypeShelley, era, [QueryTypeShelleyCurrentProtocolParams]]]
	query := cbor.IndefLengthList{
		QueryTypeBlock,
		cbor.IndefLengthList{
			QueryTypeShelley,
			era,
			cbor.IndefLengthList{
				QueryTypeShelleyCurrentProtocolParams,
			},
		},
	}
	return cbor.Encode(query)
}

// NewUTxOByAddressQuery builds a query for UTxOs by address
func NewUTxOByAddressQuery(
	era int,
	addresses []lcommon.Address,
) ([]byte, error) {
	// Convert addresses to byte arrays
	addrBytes := make([][]byte, len(addresses))
	for i, addr := range addresses {
		bytes, err := addr.Bytes()
		if err != nil {
			return nil, err
		}
		addrBytes[i] = bytes
	}

	// Query structure: [QueryTypeBlock, [QueryTypeShelley, era, [QueryTypeShelleyUtxoByAddress, [addresses...]]]]
	query := cbor.IndefLengthList{
		QueryTypeBlock,
		cbor.IndefLengthList{
			QueryTypeShelley,
			era,
			cbor.IndefLengthList{
				QueryTypeShelleyUtxoByAddress,
				addrBytes,
			},
		},
	}
	return cbor.Encode(query)
}

// NewStakeDistributionQuery builds a query for stake distribution
func NewStakeDistributionQuery(era int) ([]byte, error) {
	// Query structure: [QueryTypeBlock, [QueryTypeShelley, era, [QueryTypeShelleyStakeDistribution]]]
	query := cbor.IndefLengthList{
		QueryTypeBlock,
		cbor.IndefLengthList{
			QueryTypeShelley,
			era,
			cbor.IndefLengthList{
				QueryTypeShelleyStakeDistribution,
			},
		},
	}
	return cbor.Encode(query)
}

// Result builders for common query results

// NewCurrentEraResult builds a result for the current era query
func NewCurrentEraResult(era uint) ([]byte, error) {
	return cbor.Encode(era)
}

// NewEpochNoResult builds a result for the epoch number query
func NewEpochNoResult(epochNo uint64) ([]byte, error) {
	return cbor.Encode(epochNo)
}

// NewSystemStartResult builds a result for the system start query
func NewSystemStartResult(startTime time.Time) ([]byte, error) {
	// System start is encoded as [year, dayOfYear, picoseconds]
	year := startTime.Year()
	dayOfYear := startTime.YearDay()
	// Calculate picoseconds from midnight
	midnight := time.Date(
		startTime.Year(),
		startTime.Month(),
		startTime.Day(),
		0,
		0,
		0,
		0,
		startTime.Location(),
	)
	// #nosec G115 - nanoseconds since midnight are always positive and within uint64 range
	picoseconds := uint64(startTime.Sub(midnight).Nanoseconds()) * 1000

	result := cbor.IndefLengthList{
		year,
		dayOfYear,
		picoseconds,
	}
	return cbor.Encode(result)
}

// NewChainPointResult builds a result for the chain point query
func NewChainPointResult(point pcommon.Point) ([]byte, error) {
	if point.Slot == 0 && len(point.Hash) == 0 {
		// Origin point
		return cbor.Encode(cbor.IndefLengthList{})
	}
	result := cbor.IndefLengthList{
		point.Slot,
		point.Hash,
	}
	return cbor.Encode(result)
}

// ProtocolParamsResult holds simplified protocol parameters for building results
type ProtocolParamsResult struct {
	MinFeeA            uint64
	MinFeeB            uint64
	MaxBlockBodySize   uint64
	MaxTxSize          uint64
	MaxBlockHeaderSize uint64
	KeyDeposit         uint64
	PoolDeposit        uint64
	EMax               uint64
	NOpt               uint64
	ProtocolMajorVer   uint64
	ProtocolMinorVer   uint64
	MinPoolCost        uint64
	CoinsPerUTxOByte   uint64
}

// NewProtocolParamsResult builds a result for the protocol parameters query
func NewProtocolParamsResult(params ProtocolParamsResult) ([]byte, error) {
	// Protocol parameters are encoded as a map
	// This is a simplified version - actual encoding depends on era
	paramsMap := map[uint]any{
		0:  params.MinFeeA,
		1:  params.MinFeeB,
		2:  params.MaxBlockBodySize,
		3:  params.MaxTxSize,
		4:  params.MaxBlockHeaderSize,
		5:  params.KeyDeposit,
		6:  params.PoolDeposit,
		7:  params.EMax,
		8:  params.NOpt,
		14: params.ProtocolMajorVer,
		15: params.ProtocolMinorVer,
		16: params.MinPoolCost,
		17: params.CoinsPerUTxOByte,
	}
	return cbor.Encode(paramsMap)
}

// UTxOResult represents a single UTxO for building results
type UTxOResult struct {
	TxHash      []byte
	OutputIndex uint32
	Address     []byte
	Amount      uint64
	Assets      map[string]map[string]uint64 // PolicyId -> AssetName -> Amount
}

// NewUTxOByAddressResult builds a result for the UTxO by address query
func NewUTxOByAddressResult(utxos []UTxOResult) ([]byte, error) {
	// UTxOs are encoded as a map from TxIn to TxOut
	// TxIn: [txHash, outputIndex]
	// TxOut: [address, value, ...]
	//
	// Implementation note: Per Cardano CDDL, TxIn should be a 2-element array
	// key [txHash, outputIndex], not a byte string. However, Go maps don't
	// support slice or array keys directly, so we pre-encode the TxIn array
	// to CBOR bytes and use ByteString as the map key. This results in the
	// TxIn being double-CBOR-encoded (once here, once in the final Encode).
	// This produces non-standard CBOR but is acceptable for mock/test purposes
	// where clients may not strictly validate the encoding format.
	utxoMap := make(map[cbor.ByteString]any)

	for _, utxo := range utxos {
		// Create TxIn as CBOR bytes
		txIn := cbor.IndefLengthList{utxo.TxHash, utxo.OutputIndex}
		txInBytes, err := cbor.Encode(txIn)
		if err != nil {
			return nil, err
		}

		// Create TxOut value
		var value any
		if len(utxo.Assets) > 0 {
			// Multi-asset value: [lovelace, {policyId: {assetName: amount}}]
			assetsMap := make(map[cbor.ByteString]map[cbor.ByteString]uint64)
			for policyId, assets := range utxo.Assets {
				policyKey := cbor.NewByteString([]byte(policyId))
				assetsMap[policyKey] = make(map[cbor.ByteString]uint64)
				for assetName, amount := range assets {
					assetKey := cbor.NewByteString([]byte(assetName))
					assetsMap[policyKey][assetKey] = amount
				}
			}
			value = cbor.IndefLengthList{utxo.Amount, assetsMap}
		} else {
			// Simple lovelace value
			value = utxo.Amount
		}

		// Create TxOut
		txOut := cbor.IndefLengthList{utxo.Address, value}

		utxoMap[cbor.NewByteString(txInBytes)] = txOut
	}

	return cbor.Encode(utxoMap)
}

// StakeDistributionEntry represents a single stake pool's distribution
type StakeDistributionEntry struct {
	PoolId        []byte
	StakeFraction [2]uint64 // [numerator, denominator]
	VrfKeyHash    []byte
}

// NewStakeDistributionResult builds a result for the stake distribution query
func NewStakeDistributionResult(
	distribution []StakeDistributionEntry,
) ([]byte, error) {
	// Stake distribution is encoded as a map from pool key hash to [rational, vrf key hash]
	distMap := make(map[cbor.ByteString]any)

	for _, entry := range distribution {
		poolKey := cbor.NewByteString(entry.PoolId)
		distMap[poolKey] = cbor.IndefLengthList{
			cbor.IndefLengthList{
				entry.StakeFraction[0],
				entry.StakeFraction[1],
			},
			entry.VrfKeyHash,
		}
	}

	return cbor.Encode(distMap)
}

// NewEmptyResult builds an empty result (used for queries that return nothing)
func NewEmptyResult() ([]byte, error) {
	return cbor.Encode(nil)
}
