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
	"errors"
	"fmt"
	"math/big"

	"github.com/blinklabs-io/gouroboros/cbor"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
	utxorpc "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
)

// Asset represents a native asset with policy ID, asset name, and amount
type Asset struct {
	PolicyId  []byte
	AssetName []byte
	Amount    uint64
}

// buildMultiAsset constructs a MultiAsset from a slice of Asset.
// Returns nil if assets is empty.
func buildMultiAsset(
	assets []Asset,
) *lcommon.MultiAsset[lcommon.MultiAssetTypeOutput] {
	if len(assets) == 0 {
		return nil
	}
	assetData := make(
		map[lcommon.Blake2b224]map[cbor.ByteString]lcommon.MultiAssetTypeOutput,
	)
	for _, asset := range assets {
		policyId := lcommon.NewBlake2b224(asset.PolicyId)
		if _, ok := assetData[policyId]; !ok {
			assetData[policyId] = make(
				map[cbor.ByteString]lcommon.MultiAssetTypeOutput,
			)
		}
		assetData[policyId][cbor.NewByteString(asset.AssetName)] = new(
			big.Int,
		).SetUint64(asset.Amount)
	}
	multiAsset := lcommon.NewMultiAsset[lcommon.MultiAssetTypeOutput](assetData)
	return &multiAsset
}

// UtxoBuilder defines an interface for building mock UTxOs
type UtxoBuilder interface {
	WithTxId(txId []byte) UtxoBuilder
	WithIndex(idx uint32) UtxoBuilder
	WithAddress(addr string) UtxoBuilder
	WithLovelace(amount uint64) UtxoBuilder
	WithAssets(assets ...Asset) UtxoBuilder
	WithDatum(datum []byte) UtxoBuilder
	WithDatumHash(hash []byte) UtxoBuilder
	WithScriptRef(script []byte) UtxoBuilder
	WithScriptRefLanguage(language PlutusLanguage) UtxoBuilder
	Build() (lcommon.Utxo, error)
}

// MockUtxo holds the state for building a UTxO
type MockUtxo struct {
	txId          lcommon.Blake2b256
	index         uint32
	address       lcommon.Address
	lovelace      uint64
	assets        *lcommon.MultiAsset[lcommon.MultiAssetTypeOutput]
	datum         *lcommon.Datum
	datumHash     *lcommon.Blake2b256
	scriptRef     lcommon.Script
	scriptRefLang PlutusLanguage // Language version for script reference
	addrErr       error          // Stores address parsing error for deferred reporting
	datumErr      error          // Stores datum CBOR decode error for deferred reporting
	scriptRefErr  error          // Stores script ref CBOR decode error for deferred reporting
}

// NewUtxoBuilder creates a new MockUtxo builder
func NewUtxoBuilder() *MockUtxo {
	return &MockUtxo{}
}

// WithTxId sets the transaction ID for the UTxO
func (u *MockUtxo) WithTxId(txId []byte) UtxoBuilder {
	u.txId = lcommon.NewBlake2b256(txId)
	return u
}

// WithIndex sets the output index for the UTxO
func (u *MockUtxo) WithIndex(idx uint32) UtxoBuilder {
	u.index = idx
	return u
}

// WithAddress sets the address for the UTxO
func (u *MockUtxo) WithAddress(addr string) UtxoBuilder {
	parsedAddr, err := lcommon.NewAddress(addr)
	if err != nil {
		// Store the error for reporting in Build()
		u.address = lcommon.Address{}
		u.addrErr = fmt.Errorf("invalid address %q: %w", addr, err)
	} else {
		u.address = parsedAddr
		u.addrErr = nil
	}
	return u
}

// WithLovelace sets the ADA amount in lovelace for the UTxO
func (u *MockUtxo) WithLovelace(amount uint64) UtxoBuilder {
	u.lovelace = amount
	return u
}

// WithAssets sets the native assets for the UTxO
func (u *MockUtxo) WithAssets(assets ...Asset) UtxoBuilder {
	u.assets = buildMultiAsset(assets)
	return u
}

// WithDatum sets the inline datum for the UTxO
func (u *MockUtxo) WithDatum(datum []byte) UtxoBuilder {
	if datum != nil {
		d := lcommon.Datum{}
		if _, err := cbor.Decode(datum, &d); err != nil {
			u.datumErr = fmt.Errorf("failed to decode datum CBOR: %w", err)
		} else {
			u.datum = &d
			u.datumErr = nil
		}
	}
	return u
}

// WithDatumHash sets the datum hash for the UTxO
func (u *MockUtxo) WithDatumHash(hash []byte) UtxoBuilder {
	if hash != nil {
		h := lcommon.NewBlake2b256(hash)
		u.datumHash = &h
	}
	return u
}

// WithScriptRef sets the script reference for the UTxO
func (u *MockUtxo) WithScriptRef(script []byte) UtxoBuilder {
	if script != nil {
		// Try to decode as a script reference
		var scriptRef lcommon.ScriptRef
		if _, err := cbor.Decode(script, &scriptRef); err != nil {
			u.scriptRefErr = fmt.Errorf(
				"failed to decode script reference CBOR: %w",
				err,
			)
		} else {
			u.scriptRef = scriptRef.Script
			u.scriptRefErr = nil
		}
	}
	return u
}

// WithScriptRefLanguage sets the Plutus language version for the script reference
func (u *MockUtxo) WithScriptRefLanguage(language PlutusLanguage) UtxoBuilder {
	u.scriptRefLang = language
	return u
}

// Build constructs a Utxo from the builder state
func (u *MockUtxo) Build() (lcommon.Utxo, error) {
	// Validate required fields
	if u.txId == (lcommon.Blake2b256{}) {
		return lcommon.Utxo{}, errors.New("transaction ID is required")
	}
	// Return any stored errors from deferred validation
	if u.addrErr != nil {
		return lcommon.Utxo{}, u.addrErr
	}
	if u.datumErr != nil {
		return lcommon.Utxo{}, u.datumErr
	}
	if u.scriptRefErr != nil {
		return lcommon.Utxo{}, u.scriptRefErr
	}
	// Validate address is not zero/invalid
	if u.address.String() == "" {
		return lcommon.Utxo{}, errors.New("address is required")
	}

	// Create the transaction input
	input := &MockTransactionInput{
		txId:  u.txId,
		index: u.index,
	}

	// Create the transaction output
	output := &MockTransactionOutput{
		address:       u.address,
		amount:        u.lovelace,
		assets:        u.assets,
		datum:         u.datum,
		datumHash:     u.datumHash,
		scriptRef:     u.scriptRef,
		scriptRefLang: u.scriptRefLang,
	}

	return lcommon.Utxo{
		Id:     input,
		Output: output,
	}, nil
}

// MockTransactionInput implements lcommon.TransactionInput
type MockTransactionInput struct {
	cbor.StructAsArray
	txId  lcommon.Blake2b256
	index uint32
}

// Id returns the transaction ID
func (i *MockTransactionInput) Id() lcommon.Blake2b256 {
	return i.txId
}

// Index returns the output index
func (i *MockTransactionInput) Index() uint32 {
	return i.index
}

// String returns a string representation of the input
func (i *MockTransactionInput) String() string {
	return fmt.Sprintf("%s#%d", i.txId.String(), i.index)
}

// Utxorpc returns the UTxO RPC representation
func (i *MockTransactionInput) Utxorpc() (*utxorpc.TxInput, error) {
	return &utxorpc.TxInput{
		TxHash:      i.txId.Bytes(),
		OutputIndex: i.index,
	}, nil
}

// ToPlutusData converts the input to Plutus data
func (i *MockTransactionInput) ToPlutusData() data.PlutusData {
	return data.NewConstr(0,
		data.NewByteString(i.txId.Bytes()),
		data.NewInteger(big.NewInt(int64(i.index))),
	)
}

// MockTransactionOutput implements lcommon.TransactionOutput
type MockTransactionOutput struct {
	cbor.StructAsArray
	cbor.DecodeStoreCbor
	address       lcommon.Address
	amount        uint64
	assets        *lcommon.MultiAsset[lcommon.MultiAssetTypeOutput]
	datum         *lcommon.Datum
	datumHash     *lcommon.Blake2b256
	scriptRef     lcommon.Script
	scriptRefLang PlutusLanguage // Language version for script reference in Utxorpc()
}

// Address returns the output address
func (o *MockTransactionOutput) Address() lcommon.Address {
	return o.address
}

// Amount returns the lovelace amount as *big.Int
func (o *MockTransactionOutput) Amount() *big.Int {
	return new(big.Int).SetUint64(o.amount)
}

// Assets returns the native assets
func (o *MockTransactionOutput) Assets() *lcommon.MultiAsset[lcommon.MultiAssetTypeOutput] {
	return o.assets
}

// Datum returns the inline datum
func (o *MockTransactionOutput) Datum() *lcommon.Datum {
	return o.datum
}

// DatumHash returns the datum hash
func (o *MockTransactionOutput) DatumHash() *lcommon.Blake2b256 {
	return o.datumHash
}

// ScriptRef returns the script reference
func (o *MockTransactionOutput) ScriptRef() lcommon.Script {
	return o.scriptRef
}

// Utxorpc returns the UTxO RPC representation
func (o *MockTransactionOutput) Utxorpc() (*utxorpc.TxOutput, error) {
	addrBytes, err := o.address.Bytes()
	if err != nil {
		return nil, err
	}
	output := &utxorpc.TxOutput{
		Address: addrBytes,
		Coin:    lcommon.ToUtxorpcBigInt(o.amount),
	}

	// Add assets if present
	if o.assets != nil {
		var multiassets []*utxorpc.Multiasset
		for _, policyId := range o.assets.Policies() {
			var assets []*utxorpc.Asset
			for _, assetName := range o.assets.Assets(policyId) {
				amount := o.assets.Asset(policyId, assetName)
				// Convert *big.Int to utxorpc BigInt
				utxorpcAmount := &utxorpc.BigInt{
					BigInt: &utxorpc.BigInt_BigUInt{
						BigUInt: amount.Bytes(),
					},
				}
				assets = append(assets, &utxorpc.Asset{
					Name: assetName,
					Quantity: &utxorpc.Asset_OutputCoin{
						OutputCoin: utxorpcAmount,
					},
				})
			}
			multiassets = append(multiassets, &utxorpc.Multiasset{
				PolicyId: policyId.Bytes(),
				Assets:   assets,
			})
		}
		output.Assets = multiassets
	}

	// Add datum if present (inline datum or datum hash)
	if o.datum != nil {
		output.Datum = &utxorpc.Datum{
			Hash:         o.datum.Hash().Bytes(),
			OriginalCbor: o.datum.Cbor(),
		}
	} else if o.datumHash != nil {
		// Datum hash only (referenced datum)
		output.Datum = &utxorpc.Datum{
			Hash: o.datumHash.Bytes(),
		}
	}

	// Add script reference if present
	if o.scriptRef != nil {
		scriptBytes := o.scriptRef.RawScriptBytes()
		switch o.scriptRefLang {
		case PlutusV1:
			output.Script = &utxorpc.Script{
				Script: &utxorpc.Script_PlutusV1{
					PlutusV1: scriptBytes,
				},
			}
		case PlutusV2:
			output.Script = &utxorpc.Script{
				Script: &utxorpc.Script_PlutusV2{
					PlutusV2: scriptBytes,
				},
			}
		case PlutusV3:
			output.Script = &utxorpc.Script{
				Script: &utxorpc.Script_PlutusV3{
					PlutusV3: scriptBytes,
				},
			}
		default:
			// Default to PlutusV2 for backwards compatibility (e.g., when language is 0/unset)
			output.Script = &utxorpc.Script{
				Script: &utxorpc.Script_PlutusV2{
					PlutusV2: scriptBytes,
				},
			}
		}
	}

	return output, nil
}

// ToPlutusData converts the output to Plutus data
func (o *MockTransactionOutput) ToPlutusData() data.PlutusData {
	// Build address data
	addressPd := o.address.ToPlutusData()

	// Build value data (lovelace + assets)
	var valuePd data.PlutusData
	if o.assets != nil {
		valuePd = data.NewConstr(0,
			data.NewInteger(new(big.Int).SetUint64(o.amount)),
			o.assets.ToPlutusData(),
		)
	} else {
		valuePd = data.NewInteger(new(big.Int).SetUint64(o.amount))
	}

	// Build datum option
	var datumPd data.PlutusData
	if o.datum != nil {
		datumPd = data.NewConstr(2, o.datum.Data)
	} else if o.datumHash != nil {
		datumPd = data.NewConstr(1, data.NewByteString(o.datumHash.Bytes()))
	} else {
		datumPd = data.NewConstr(0)
	}

	// Build script ref option
	var scriptRefPd data.PlutusData
	if o.scriptRef != nil {
		scriptRefPd = data.NewConstr(
			0,
			data.NewByteString(o.scriptRef.Hash().Bytes()),
		)
	} else {
		scriptRefPd = data.NewConstr(1)
	}

	return data.NewConstr(0,
		addressPd,
		valuePd,
		datumPd,
		scriptRefPd,
	)
}

// String returns a string representation of the output
func (o *MockTransactionOutput) String() string {
	return fmt.Sprintf("%s: %d lovelace", o.address.String(), o.amount)
}
