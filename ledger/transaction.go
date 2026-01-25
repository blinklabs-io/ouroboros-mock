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

// TransactionBuilder defines an interface for building mock transactions
type TransactionBuilder interface {
	WithId(txId []byte) TransactionBuilder
	WithInputs(inputs ...lcommon.TransactionInput) TransactionBuilder
	WithOutputs(outputs ...lcommon.TransactionOutput) TransactionBuilder
	WithFee(fee uint64) TransactionBuilder
	WithTTL(ttl uint64) TransactionBuilder
	WithMetadata(metadata []byte) TransactionBuilder
	WithValid(valid bool) TransactionBuilder
	Build() (lcommon.Transaction, error)
}

// TransactionInputBuilder defines an interface for building mock transaction inputs
type TransactionInputBuilder interface {
	WithTxId(txId []byte) TransactionInputBuilder
	WithIndex(idx uint32) TransactionInputBuilder
	Build() (lcommon.TransactionInput, error)
}

// TransactionOutputBuilder defines an interface for building mock transaction outputs
type TransactionOutputBuilder interface {
	WithAddress(addr string) TransactionOutputBuilder
	WithLovelace(amount uint64) TransactionOutputBuilder
	WithAssets(assets ...Asset) TransactionOutputBuilder
	WithDatum(datum []byte) TransactionOutputBuilder
	WithDatumHash(hash []byte) TransactionOutputBuilder
	Build() (lcommon.TransactionOutput, error)
}

// MockTransaction implements lcommon.Transaction interface
type MockTransaction struct {
	cbor.StructAsArray
	cbor.DecodeStoreCbor
	txId        lcommon.Blake2b256
	inputs      []lcommon.TransactionInput
	outputs     []lcommon.TransactionOutput
	fee         uint64
	ttl         uint64
	metadata    lcommon.TransactionMetadatum
	metadataErr error // Stores metadata decoding error for deferred reporting
	valid       bool
	txType      int
	witnesses   *MockTransactionWitnessSet
	leiosHash   lcommon.Blake2b256
	refInputs   []lcommon.TransactionInput
	collateral  []lcommon.TransactionInput
	collReturn  lcommon.TransactionOutput
	totalColl   uint64
	certs       []lcommon.Certificate
	wdrls       map[*lcommon.Address]uint64
	auxHash     *lcommon.Blake2b256
	reqSigners  []lcommon.Blake2b224
	mint        *lcommon.MultiAsset[lcommon.MultiAssetTypeMint]
	scriptHash  *lcommon.Blake2b256
	votingProc  lcommon.VotingProcedures
	proposals   []lcommon.ProposalProcedure
	treasury    int64
	donation    uint64
	ppUpdEpoch  uint64
	ppUpdates   map[lcommon.Blake2b224]lcommon.ProtocolParameterUpdate
	validStart  uint64
}

// NewTransactionBuilder creates a new MockTransaction builder
func NewTransactionBuilder() *MockTransaction {
	return &MockTransaction{
		valid:     true,
		witnesses: NewMockTransactionWitnessSet(),
	}
}

// WithId sets the transaction ID
func (t *MockTransaction) WithId(txId []byte) TransactionBuilder {
	t.txId = lcommon.NewBlake2b256(txId)
	return t
}

// WithInputs sets the transaction inputs
func (t *MockTransaction) WithInputs(
	inputs ...lcommon.TransactionInput,
) TransactionBuilder {
	t.inputs = append(t.inputs, inputs...)
	return t
}

// WithOutputs sets the transaction outputs
func (t *MockTransaction) WithOutputs(
	outputs ...lcommon.TransactionOutput,
) TransactionBuilder {
	t.outputs = append(t.outputs, outputs...)
	return t
}

// WithFee sets the transaction fee
func (t *MockTransaction) WithFee(fee uint64) TransactionBuilder {
	t.fee = fee
	return t
}

// WithTTL sets the transaction time-to-live
func (t *MockTransaction) WithTTL(ttl uint64) TransactionBuilder {
	t.ttl = ttl
	return t
}

// WithMetadata sets the transaction metadata from raw bytes
func (t *MockTransaction) WithMetadata(metadata []byte) TransactionBuilder {
	if metadata != nil {
		decoded, err := lcommon.DecodeAuxiliaryDataToMetadata(metadata)
		if err != nil {
			// Store the error for reporting in Build()
			t.metadataErr = fmt.Errorf("invalid metadata CBOR: %w", err)
		} else {
			t.metadata = decoded
			t.metadataErr = nil
		}
	}
	return t
}

// WithValid sets the transaction validity flag
func (t *MockTransaction) WithValid(valid bool) TransactionBuilder {
	t.valid = valid
	return t
}

// Build constructs a Transaction from the builder state
func (t *MockTransaction) Build() (lcommon.Transaction, error) {
	// Return any stored parsing errors
	if t.metadataErr != nil {
		return nil, t.metadataErr
	}
	if len(t.inputs) == 0 {
		return nil, errors.New("transaction must have at least one input")
	}
	if len(t.outputs) == 0 {
		return nil, errors.New("transaction must have at least one output")
	}
	// Check for nil entries in inputs and outputs
	for i, input := range t.inputs {
		if input == nil {
			return nil, fmt.Errorf("transaction contains nil input at index %d", i)
		}
	}
	for i, output := range t.outputs {
		if output == nil {
			return nil, fmt.Errorf("transaction contains nil output at index %d", i)
		}
	}
	return t, nil
}

// Type returns the transaction type identifier
func (t *MockTransaction) Type() int {
	return t.txType
}

// Cbor returns the CBOR-encoded transaction
func (t *MockTransaction) Cbor() []byte {
	return t.DecodeStoreCbor.Cbor()
}

// Hash returns the transaction hash (ID)
func (t *MockTransaction) Hash() lcommon.Blake2b256 {
	return t.txId
}

// LeiosHash returns the Leios hash of the transaction
func (t *MockTransaction) LeiosHash() lcommon.Blake2b256 {
	return t.leiosHash
}

// Metadata returns the transaction metadata
func (t *MockTransaction) Metadata() lcommon.TransactionMetadatum {
	return t.metadata
}

// AuxiliaryData returns the transaction auxiliary data
func (t *MockTransaction) AuxiliaryData() lcommon.AuxiliaryData {
	return nil
}

// IsValid returns whether the transaction is valid
func (t *MockTransaction) IsValid() bool {
	return t.valid
}

// Consumed returns the transaction inputs that are consumed
func (t *MockTransaction) Consumed() []lcommon.TransactionInput {
	return t.inputs
}

// Produced returns the UTxOs produced by this transaction
func (t *MockTransaction) Produced() []lcommon.Utxo {
	utxos := make([]lcommon.Utxo, 0, len(t.outputs))
	for idx, output := range t.outputs {
		input := &MockTransactionInput{
			txId: t.txId,
			// #nosec G115 - transaction output index is bounded by protocol limits
			index: uint32(idx),
		}
		utxos = append(utxos, lcommon.Utxo{
			Id:     input,
			Output: output,
		})
	}
	return utxos
}

// Witnesses returns the transaction witness set
func (t *MockTransaction) Witnesses() lcommon.TransactionWitnessSet {
	return t.witnesses
}

// Fee returns the transaction fee as *big.Int
func (t *MockTransaction) Fee() *big.Int {
	return new(big.Int).SetUint64(t.fee)
}

// Id returns the transaction ID
func (t *MockTransaction) Id() lcommon.Blake2b256 {
	return t.txId
}

// Inputs returns the transaction inputs
func (t *MockTransaction) Inputs() []lcommon.TransactionInput {
	return t.inputs
}

// Outputs returns the transaction outputs
func (t *MockTransaction) Outputs() []lcommon.TransactionOutput {
	return t.outputs
}

// TTL returns the transaction time-to-live
func (t *MockTransaction) TTL() uint64 {
	return t.ttl
}

// ProtocolParameterUpdates returns protocol parameter updates (if any)
func (t *MockTransaction) ProtocolParameterUpdates() (uint64, map[lcommon.Blake2b224]lcommon.ProtocolParameterUpdate) {
	return t.ppUpdEpoch, t.ppUpdates
}

// ValidityIntervalStart returns the validity interval start slot
func (t *MockTransaction) ValidityIntervalStart() uint64 {
	return t.validStart
}

// ReferenceInputs returns the reference inputs
func (t *MockTransaction) ReferenceInputs() []lcommon.TransactionInput {
	return t.refInputs
}

// Collateral returns the collateral inputs
func (t *MockTransaction) Collateral() []lcommon.TransactionInput {
	return t.collateral
}

// CollateralReturn returns the collateral return output
func (t *MockTransaction) CollateralReturn() lcommon.TransactionOutput {
	return t.collReturn
}

// TotalCollateral returns the total collateral amount as *big.Int
func (t *MockTransaction) TotalCollateral() *big.Int {
	return new(big.Int).SetUint64(t.totalColl)
}

// Certificates returns the certificates in the transaction
func (t *MockTransaction) Certificates() []lcommon.Certificate {
	return t.certs
}

// Withdrawals returns the stake withdrawals in the transaction as map[*lcommon.Address]*big.Int
func (t *MockTransaction) Withdrawals() map[*lcommon.Address]*big.Int {
	if t.wdrls == nil {
		return nil
	}
	result := make(map[*lcommon.Address]*big.Int, len(t.wdrls))
	for addr, amount := range t.wdrls {
		result[addr] = new(big.Int).SetUint64(amount)
	}
	return result
}

// AuxDataHash returns the auxiliary data hash
func (t *MockTransaction) AuxDataHash() *lcommon.Blake2b256 {
	return t.auxHash
}

// RequiredSigners returns the required signers for the transaction
func (t *MockTransaction) RequiredSigners() []lcommon.Blake2b224 {
	return t.reqSigners
}

// AssetMint returns the native asset minting/burning in the transaction
func (t *MockTransaction) AssetMint() *lcommon.MultiAsset[lcommon.MultiAssetTypeMint] {
	return t.mint
}

// ScriptDataHash returns the script data hash
func (t *MockTransaction) ScriptDataHash() *lcommon.Blake2b256 {
	return t.scriptHash
}

// VotingProcedures returns the voting procedures in the transaction
func (t *MockTransaction) VotingProcedures() lcommon.VotingProcedures {
	return t.votingProc
}

// ProposalProcedures returns the governance proposals in the transaction
func (t *MockTransaction) ProposalProcedures() []lcommon.ProposalProcedure {
	return t.proposals
}

// CurrentTreasuryValue returns the current treasury value as *big.Int
func (t *MockTransaction) CurrentTreasuryValue() *big.Int {
	return big.NewInt(t.treasury)
}

// Donation returns the donation amount as *big.Int
func (t *MockTransaction) Donation() *big.Int {
	return new(big.Int).SetUint64(t.donation)
}

// Utxorpc returns the UTxO RPC representation of the transaction
func (t *MockTransaction) Utxorpc() (*utxorpc.Tx, error) {
	tx := &utxorpc.Tx{
		Hash: t.txId.Bytes(),
		Fee:  lcommon.ToUtxorpcBigInt(t.fee),
	}

	for _, input := range t.inputs {
		utxorpcInput, err := input.Utxorpc()
		if err != nil {
			return nil, err
		}
		tx.Inputs = append(tx.Inputs, utxorpcInput)
	}

	for _, output := range t.outputs {
		utxorpcOutput, err := output.Utxorpc()
		if err != nil {
			return nil, err
		}
		tx.Outputs = append(tx.Outputs, utxorpcOutput)
	}

	return tx, nil
}

// MockTransactionInputBuilder holds the state for building a transaction input
type MockTransactionInputBuilder struct {
	txId  lcommon.Blake2b256
	index uint32
}

// NewTransactionInputBuilder creates a new transaction input builder
func NewTransactionInputBuilder() *MockTransactionInputBuilder {
	return &MockTransactionInputBuilder{}
}

// WithTxId sets the transaction ID for the input
func (b *MockTransactionInputBuilder) WithTxId(
	txId []byte,
) TransactionInputBuilder {
	b.txId = lcommon.NewBlake2b256(txId)
	return b
}

// WithIndex sets the output index for the input
func (b *MockTransactionInputBuilder) WithIndex(
	idx uint32,
) TransactionInputBuilder {
	b.index = idx
	return b
}

// Build constructs a TransactionInput from the builder state
func (b *MockTransactionInputBuilder) Build() (lcommon.TransactionInput, error) {
	if b.txId == (lcommon.Blake2b256{}) {
		return nil, errors.New("transaction ID is required")
	}
	return &MockTransactionInput{
		txId:  b.txId,
		index: b.index,
	}, nil
}

// MockTransactionOutputBuilder holds the state for building a transaction output
type MockTransactionOutputBuilder struct {
	address   lcommon.Address
	amount    uint64
	assets    *lcommon.MultiAsset[lcommon.MultiAssetTypeOutput]
	datum     *lcommon.Datum
	datumHash *lcommon.Blake2b256
	addrErr   error // Stores address parsing error for deferred reporting
	datumErr  error // Stores datum decoding error for deferred reporting
}

// NewTransactionOutputBuilder creates a new transaction output builder
func NewTransactionOutputBuilder() *MockTransactionOutputBuilder {
	return &MockTransactionOutputBuilder{}
}

// WithAddress sets the address for the output
func (b *MockTransactionOutputBuilder) WithAddress(
	addr string,
) TransactionOutputBuilder {
	parsedAddr, err := lcommon.NewAddress(addr)
	if err != nil {
		// Store the error for reporting in Build()
		b.address = lcommon.Address{}
		b.addrErr = fmt.Errorf("invalid address %q: %w", addr, err)
	} else {
		b.address = parsedAddr
		b.addrErr = nil
	}
	return b
}

// WithLovelace sets the ADA amount in lovelace for the output
func (b *MockTransactionOutputBuilder) WithLovelace(
	amount uint64,
) TransactionOutputBuilder {
	b.amount = amount
	return b
}

// WithAssets sets the native assets for the output
func (b *MockTransactionOutputBuilder) WithAssets(
	assets ...Asset,
) TransactionOutputBuilder {
	b.assets = buildMultiAsset(assets)
	return b
}

// WithDatum sets the inline datum for the output
func (b *MockTransactionOutputBuilder) WithDatum(
	datum []byte,
) TransactionOutputBuilder {
	if datum != nil {
		d := lcommon.Datum{}
		if _, err := cbor.Decode(datum, &d); err != nil {
			// Store the error for reporting in Build()
			b.datumErr = fmt.Errorf("invalid datum CBOR: %w", err)
		} else {
			b.datum = &d
			b.datumErr = nil
		}
	}
	return b
}

// WithDatumHash sets the datum hash for the output
func (b *MockTransactionOutputBuilder) WithDatumHash(
	hash []byte,
) TransactionOutputBuilder {
	if hash != nil {
		h := lcommon.NewBlake2b256(hash)
		b.datumHash = &h
	}
	return b
}

// Build constructs a TransactionOutput from the builder state
func (b *MockTransactionOutputBuilder) Build() (lcommon.TransactionOutput, error) {
	// Return any stored parsing errors
	if b.addrErr != nil {
		return nil, b.addrErr
	}
	if b.datumErr != nil {
		return nil, b.datumErr
	}
	if b.address.String() == "" {
		return nil, errors.New("address is required")
	}
	return &MockTransactionOutput{
		address:   b.address,
		amount:    b.amount,
		assets:    b.assets,
		datum:     b.datum,
		datumHash: b.datumHash,
	}, nil
}

// MockTransactionWitnessSet implements lcommon.TransactionWitnessSet
type MockTransactionWitnessSet struct {
	vkeys        []lcommon.VkeyWitness
	native       []lcommon.NativeScript
	bootstrap    []lcommon.BootstrapWitness
	plutusData   []lcommon.Datum
	plutusV1     []lcommon.PlutusV1Script
	plutusV2     []lcommon.PlutusV2Script
	plutusV3     []lcommon.PlutusV3Script
	redeemerData lcommon.TransactionWitnessRedeemers
}

// NewMockTransactionWitnessSet creates a new empty witness set
func NewMockTransactionWitnessSet() *MockTransactionWitnessSet {
	return &MockTransactionWitnessSet{}
}

// Vkey returns the VKey witnesses
func (w *MockTransactionWitnessSet) Vkey() []lcommon.VkeyWitness {
	return w.vkeys
}

// NativeScripts returns the native scripts
func (w *MockTransactionWitnessSet) NativeScripts() []lcommon.NativeScript {
	return w.native
}

// Bootstrap returns the bootstrap witnesses
func (w *MockTransactionWitnessSet) Bootstrap() []lcommon.BootstrapWitness {
	return w.bootstrap
}

// PlutusData returns the Plutus data
func (w *MockTransactionWitnessSet) PlutusData() []lcommon.Datum {
	return w.plutusData
}

// PlutusV1Scripts returns the Plutus V1 scripts
func (w *MockTransactionWitnessSet) PlutusV1Scripts() []lcommon.PlutusV1Script {
	return w.plutusV1
}

// PlutusV2Scripts returns the Plutus V2 scripts
func (w *MockTransactionWitnessSet) PlutusV2Scripts() []lcommon.PlutusV2Script {
	return w.plutusV2
}

// PlutusV3Scripts returns the Plutus V3 scripts
func (w *MockTransactionWitnessSet) PlutusV3Scripts() []lcommon.PlutusV3Script {
	return w.plutusV3
}

// Redeemers returns the transaction redeemers
func (w *MockTransactionWitnessSet) Redeemers() lcommon.TransactionWitnessRedeemers {
	return w.redeemerData
}

// Additional builder methods for MockTransaction to support advanced features

// WithType sets the transaction type
func (t *MockTransaction) WithType(txType int) *MockTransaction {
	t.txType = txType
	return t
}

// WithLeiosHash sets the Leios hash
func (t *MockTransaction) WithLeiosHash(hash []byte) *MockTransaction {
	t.leiosHash = lcommon.NewBlake2b256(hash)
	return t
}

// WithReferenceInputs sets the reference inputs
func (t *MockTransaction) WithReferenceInputs(
	inputs ...lcommon.TransactionInput,
) *MockTransaction {
	t.refInputs = append(t.refInputs, inputs...)
	return t
}

// WithCollateral sets the collateral inputs
func (t *MockTransaction) WithCollateral(
	inputs ...lcommon.TransactionInput,
) *MockTransaction {
	t.collateral = append(t.collateral, inputs...)
	return t
}

// WithCollateralReturn sets the collateral return output
func (t *MockTransaction) WithCollateralReturn(
	output lcommon.TransactionOutput,
) *MockTransaction {
	t.collReturn = output
	return t
}

// WithTotalCollateral sets the total collateral amount
func (t *MockTransaction) WithTotalCollateral(amount uint64) *MockTransaction {
	t.totalColl = amount
	return t
}

// WithCertificates sets the certificates
func (t *MockTransaction) WithCertificates(
	certs ...lcommon.Certificate,
) *MockTransaction {
	t.certs = append(t.certs, certs...)
	return t
}

// WithWithdrawals sets the stake withdrawals
func (t *MockTransaction) WithWithdrawals(
	wdrls map[*lcommon.Address]uint64,
) *MockTransaction {
	t.wdrls = wdrls
	return t
}

// WithAuxDataHash sets the auxiliary data hash
func (t *MockTransaction) WithAuxDataHash(hash []byte) *MockTransaction {
	if hash != nil {
		h := lcommon.NewBlake2b256(hash)
		t.auxHash = &h
	}
	return t
}

// WithRequiredSigners sets the required signers
func (t *MockTransaction) WithRequiredSigners(
	signers ...lcommon.Blake2b224,
) *MockTransaction {
	t.reqSigners = append(t.reqSigners, signers...)
	return t
}

// WithMint sets the asset minting/burning
func (t *MockTransaction) WithMint(
	mint *lcommon.MultiAsset[lcommon.MultiAssetTypeMint],
) *MockTransaction {
	t.mint = mint
	return t
}

// WithScriptDataHash sets the script data hash
func (t *MockTransaction) WithScriptDataHash(hash []byte) *MockTransaction {
	if hash != nil {
		h := lcommon.NewBlake2b256(hash)
		t.scriptHash = &h
	}
	return t
}

// WithVotingProcedures sets the voting procedures
func (t *MockTransaction) WithVotingProcedures(
	procs lcommon.VotingProcedures,
) *MockTransaction {
	t.votingProc = procs
	return t
}

// WithProposalProcedures sets the governance proposals
func (t *MockTransaction) WithProposalProcedures(
	proposals ...lcommon.ProposalProcedure,
) *MockTransaction {
	t.proposals = append(t.proposals, proposals...)
	return t
}

// WithTreasuryValue sets the current treasury value
func (t *MockTransaction) WithTreasuryValue(value int64) *MockTransaction {
	t.treasury = value
	return t
}

// WithDonation sets the donation amount
func (t *MockTransaction) WithDonation(amount uint64) *MockTransaction {
	t.donation = amount
	return t
}

// WithValidityIntervalStart sets the validity interval start slot
func (t *MockTransaction) WithValidityIntervalStart(
	slot uint64,
) *MockTransaction {
	t.validStart = slot
	return t
}

// WithWitnesses sets the witness set
func (t *MockTransaction) WithWitnesses(
	witnesses *MockTransactionWitnessSet,
) *MockTransaction {
	t.witnesses = witnesses
	return t
}

// Witness set builder methods

// WithVkeyWitnesses adds VKey witnesses to the witness set
func (w *MockTransactionWitnessSet) WithVkeyWitnesses(
	vkeys ...lcommon.VkeyWitness,
) *MockTransactionWitnessSet {
	w.vkeys = append(w.vkeys, vkeys...)
	return w
}

// WithNativeScripts adds native scripts to the witness set
func (w *MockTransactionWitnessSet) WithNativeScripts(
	scripts ...lcommon.NativeScript,
) *MockTransactionWitnessSet {
	w.native = append(w.native, scripts...)
	return w
}

// WithBootstrapWitnesses adds bootstrap witnesses to the witness set
func (w *MockTransactionWitnessSet) WithBootstrapWitnesses(
	witnesses ...lcommon.BootstrapWitness,
) *MockTransactionWitnessSet {
	w.bootstrap = append(w.bootstrap, witnesses...)
	return w
}

// WithPlutusData adds Plutus data to the witness set
func (w *MockTransactionWitnessSet) WithPlutusData(
	datum ...lcommon.Datum,
) *MockTransactionWitnessSet {
	w.plutusData = append(w.plutusData, datum...)
	return w
}

// WithPlutusV1Scripts adds Plutus V1 scripts to the witness set
func (w *MockTransactionWitnessSet) WithPlutusV1Scripts(
	scripts ...lcommon.PlutusV1Script,
) *MockTransactionWitnessSet {
	w.plutusV1 = append(w.plutusV1, scripts...)
	return w
}

// WithPlutusV2Scripts adds Plutus V2 scripts to the witness set
func (w *MockTransactionWitnessSet) WithPlutusV2Scripts(
	scripts ...lcommon.PlutusV2Script,
) *MockTransactionWitnessSet {
	w.plutusV2 = append(w.plutusV2, scripts...)
	return w
}

// WithPlutusV3Scripts adds Plutus V3 scripts to the witness set
func (w *MockTransactionWitnessSet) WithPlutusV3Scripts(
	scripts ...lcommon.PlutusV3Script,
) *MockTransactionWitnessSet {
	w.plutusV3 = append(w.plutusV3, scripts...)
	return w
}

// WithRedeemers sets the redeemers for the witness set
func (w *MockTransactionWitnessSet) WithRedeemers(
	redeemers lcommon.TransactionWitnessRedeemers,
) *MockTransactionWitnessSet {
	w.redeemerData = redeemers
	return w
}

// Ensure MockTransactionOutput implements the Cbor() method required by lcommon.TransactionOutput
// The existing MockTransactionOutput in utxo.go already has most methods, but we need Cbor()

// MockTransactionOutputWithCbor wraps MockTransactionOutput to add CBOR support
type MockTransactionOutputWithCbor struct {
	*MockTransactionOutput
	cborData []byte
}

// Cbor returns the CBOR-encoded output
func (o *MockTransactionOutputWithCbor) Cbor() []byte {
	return o.cborData
}

// NewTransactionOutputFromUtxo creates a transaction output builder from an existing MockTransactionOutput.
// Returns nil if output is nil.
func NewTransactionOutputFromUtxo(
	output *MockTransactionOutput,
) *MockTransactionOutputBuilder {
	if output == nil {
		return nil
	}
	return &MockTransactionOutputBuilder{
		address:   output.address,
		amount:    output.amount,
		assets:    output.assets,
		datum:     output.datum,
		datumHash: output.datumHash,
	}
}

// Helper function to create a simple transaction for testing
func NewSimpleTransaction(
	txId []byte,
	inputs []lcommon.TransactionInput,
	outputs []lcommon.TransactionOutput,
	fee uint64,
) (*MockTransaction, error) {
	tx := NewTransactionBuilder().
		WithId(txId).
		WithFee(fee)

	for _, input := range inputs {
		tx.WithInputs(input)
	}
	for _, output := range outputs {
		tx.WithOutputs(output)
	}

	result, err := tx.Build()
	if err != nil {
		return nil, err
	}
	mockTx, ok := result.(*MockTransaction)
	if !ok {
		return nil, errors.New("unexpected transaction type")
	}
	return mockTx, nil
}

// Helper function to create a simple transaction input
func NewSimpleTransactionInput(
	txId []byte,
	index uint32,
) (*MockTransactionInput, error) {
	input, err := NewTransactionInputBuilder().
		WithTxId(txId).
		WithIndex(index).
		Build()
	if err != nil {
		return nil, err
	}
	mockInput, ok := input.(*MockTransactionInput)
	if !ok {
		return nil, errors.New("unexpected transaction input type")
	}
	return mockInput, nil
}

// Helper function to create a simple transaction output
func NewSimpleTransactionOutput(
	address string,
	lovelace uint64,
) (*MockTransactionOutput, error) {
	output, err := NewTransactionOutputBuilder().
		WithAddress(address).
		WithLovelace(lovelace).
		Build()
	if err != nil {
		return nil, err
	}
	mockOutput, ok := output.(*MockTransactionOutput)
	if !ok {
		return nil, errors.New("unexpected transaction output type")
	}
	return mockOutput, nil
}

// String returns a string representation of the transaction
func (t *MockTransaction) String() string {
	return fmt.Sprintf(
		"Transaction{id=%s, inputs=%d, outputs=%d, fee=%d}",
		t.txId.String(),
		len(t.inputs),
		len(t.outputs),
		t.fee,
	)
}

// ToPlutusData converts the transaction to Plutus data representation
func (t *MockTransaction) ToPlutusData() data.PlutusData {
	// Build inputs list
	inputsList := make([]data.PlutusData, len(t.inputs))
	for i, input := range t.inputs {
		inputsList[i] = input.ToPlutusData()
	}

	// Build outputs list
	outputsList := make([]data.PlutusData, len(t.outputs))
	for i, output := range t.outputs {
		outputsList[i] = output.ToPlutusData()
	}

	return data.NewConstr(0,
		data.NewList(inputsList...),
		data.NewList(outputsList...),
		data.NewInteger(new(big.Int).SetUint64(t.fee)),
	)
}
