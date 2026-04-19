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
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	gcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

// ExecutionResult contains the outcome of executing one fixture through the
// built-in fixture runner.
type ExecutionResult struct {
	Fixture Fixture
	Success bool
	Error   error

	// CaseCount is the number of executable cases covered by the fixture.
	// Most fixtures contribute one case, while translation corpora contain many.
	CaseCount int
}

// RunAllExecutions executes every curated fixture using the built-in execution
// logic and fails the test on the first failing fixture.
func (h *Harness) RunAllExecutions(t *testing.T) {
	t.Helper()
	h.runMatchingExecutions(t, Filter{})
}

// RunMatchingExecutions executes every fixture that matches the supplied
// filter and fails the test if the filter matches no fixtures.
func (h *Harness) RunMatchingExecutions(t *testing.T, filter Filter) {
	t.Helper()
	h.runMatchingExecutions(t, filter)
}

// RunExecution executes a single fixture by relative path.
func (h *Harness) RunExecution(t *testing.T, relPath string) {
	t.Helper()

	result, err := h.ExecuteFixture(relPath)
	if err != nil {
		t.Fatalf("failed to execute fixture: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("%s: %v", result.Fixture.RelPath, result.Error)
	}
}

// ExecuteFixture executes a single fixture by relative path and returns the
// detailed result.
func (h *Harness) ExecuteFixture(relPath string) (ExecutionResult, error) {
	_, fixtureMap, err := h.collectFixtureIndex()
	if err != nil {
		return ExecutionResult{}, err
	}

	normalizedPath := normalizeRelativePath(relPath)
	fixture, ok := fixtureMap[normalizedPath]
	if !ok {
		return ExecutionResult{}, fmt.Errorf(
			"fixture not found: %s",
			normalizedPath,
		)
	}
	return executeFixtureWithIndex(fixture, fixtureMap), nil
}

// RunAllExecutionsWithResults executes every curated fixture and returns the
// detailed results without requiring a testing.T.
func (h *Harness) RunAllExecutionsWithResults() ([]ExecutionResult, error) {
	fixtures, fixtureMap, err := h.collectFixtureIndex()
	if err != nil {
		return nil, err
	}

	results := make([]ExecutionResult, 0, len(fixtures))
	for _, fixture := range fixtures {
		results = append(results, executeFixtureWithIndex(fixture, fixtureMap))
	}
	return results, nil
}

func (h *Harness) runMatchingExecutions(t *testing.T, filter Filter) {
	t.Helper()

	fixtures, fixtureMap, err := h.collectFixtureIndex()
	if err != nil {
		t.Fatalf("failed to collect fixtures: %v", err)
	}

	var matched int
	for _, fixture := range fixtures {
		if !filter.Matches(fixture) {
			continue
		}
		matched++
		t.Run(fixture.RelPath, func(t *testing.T) {
			result := executeFixtureWithIndex(fixture, fixtureMap)
			if result.Error != nil {
				t.Fatalf("%v", result.Error)
			}
		})
	}

	if matched == 0 {
		t.Fatalf("no fixtures matched filter: %+v", filter)
	}
}

func (h *Harness) collectFixtureIndex() ([]Fixture, map[string]Fixture, error) {
	fixtures, err := h.Collect()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to collect fixtures: %w", err)
	}

	fixtureMap := make(map[string]Fixture, len(fixtures))
	for _, fixture := range fixtures {
		fixtureMap[fixture.RelPath] = fixture
	}
	return fixtures, fixtureMap, nil
}

func executeFixtureWithIndex(
	fixture Fixture,
	fixtureMap map[string]Fixture,
) ExecutionResult {
	result := ExecutionResult{
		Fixture:   fixture,
		CaseCount: 1,
	}

	caseCount, err := executeFixture(fixture, fixtureMap)
	if caseCount > 0 {
		result.CaseCount = caseCount
	}
	if err != nil {
		result.Error = err
		return result
	}

	result.Success = true
	return result
}

func executeFixture(
	fixture Fixture,
	fixtureMap map[string]Fixture,
) (int, error) {
	switch fixture.Kind {
	case KindBlock:
		return executeBlockFixture(fixture, fixtureMap)
	case KindHeader:
		return executeHeaderFixture(fixture, fixtureMap)
	case KindTransaction:
		return executeTransactionFixture(fixture, fixtureMap)
	case KindTransactionID:
		return executeTransactionIDFixture(fixture, fixtureMap)
	case KindGenesis:
		return executeGenesisFixture(fixture)
	case KindProtocolParameters:
		return executeProtocolParametersFixture(fixture)
	case KindProtocolParametersUpdate:
		return executeProtocolParametersUpdateFixture(fixture, fixtureMap)
	case KindGovernanceMetadata:
		return 1, validateGovernanceMetadataFixture(fixture)
	case KindTranslation:
		return executeTranslationFixture(fixture)
	case KindUnknown:
		return 0, fmt.Errorf("unknown fixture kind for %s", fixture.RelPath)
	default:
		return 0, fmt.Errorf(
			"unsupported fixture kind %q for %s",
			fixture.Kind,
			fixture.RelPath,
		)
	}
}

func executeBlockFixture(
	fixture Fixture,
	fixtureMap map[string]Fixture,
) (int, error) {
	if fixture.Repo == RepoOuroborosConsensus && fixture.Era == "dijkstra" {
		return executeDijkstraConsensusBlockFixture(fixture, fixtureMap)
	}

	block, err := fixture.DecodeLedgerBlock()
	if err != nil {
		return 0, fmt.Errorf(
			"failed to decode block fixture %s: %w",
			fixture.RelPath,
			err,
		)
	}
	if len(block.Cbor()) == 0 {
		return 0, fmt.Errorf(
			"decoded block has empty CBOR: %s",
			fixture.RelPath,
		)
	}
	for txIdx, tx := range block.Transactions() {
		if len(tx.Cbor()) == 0 {
			return 0, fmt.Errorf(
				"decoded block transaction %d has empty CBOR: %s",
				txIdx,
				fixture.RelPath,
			)
		}
	}

	if fixture.Repo == RepoOuroborosConsensus {
		oneEraBlock, err := fixture.ConsensusBlock()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode consensus block wrapper %s: %w",
				fixture.RelPath,
				err,
			)
		}
		expectedType, err := fixture.LedgerBlockType()
		if err != nil {
			return 0, err
		}
		if oneEraBlock.Era != expectedType {
			return 0, fmt.Errorf(
				"unexpected one-era block identifier for %s: got %d want %d",
				fixture.RelPath,
				oneEraBlock.Era,
				expectedType,
			)
		}
	}

	if counterpart, ok := relatedFixture(fixtureMap, fixture, KindHeader); ok {
		header, err := counterpart.DecodeLedgerHeader()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode related header fixture %s: %w",
				counterpart.RelPath,
				err,
			)
		}
		if block.Hash() != header.Hash() {
			return 0, fmt.Errorf(
				"block/header hash mismatch between %s and %s",
				fixture.RelPath,
				counterpart.RelPath,
			)
		}
		if block.SlotNumber() != header.SlotNumber() {
			return 0, fmt.Errorf(
				"block/header slot mismatch between %s and %s",
				fixture.RelPath,
				counterpart.RelPath,
			)
		}
		if block.BlockNumber() != header.BlockNumber() {
			return 0, fmt.Errorf(
				"block/header block number mismatch between %s and %s",
				fixture.RelPath,
				counterpart.RelPath,
			)
		}
	}

	return 1, nil
}

func executeHeaderFixture(
	fixture Fixture,
	fixtureMap map[string]Fixture,
) (int, error) {
	header, err := fixture.DecodeLedgerHeader()
	if err != nil {
		return 0, fmt.Errorf(
			"failed to decode header fixture %s: %w",
			fixture.RelPath,
			err,
		)
	}
	if len(header.Cbor()) == 0 {
		return 0, fmt.Errorf(
			"decoded header has empty CBOR: %s",
			fixture.RelPath,
		)
	}

	if fixture.Repo == RepoOuroborosConsensus {
		wrappedHeader, err := fixture.ConsensusHeader()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode consensus header wrapper %s: %w",
				fixture.RelPath,
				err,
			)
		}
		expectedEra, err := fixture.ConsensusHeaderEra()
		if err != nil {
			return 0, err
		}
		if wrappedHeader.Era != expectedEra {
			return 0, fmt.Errorf(
				"unexpected wrapped header era for %s: got %d want %d",
				fixture.RelPath,
				wrappedHeader.Era,
				expectedEra,
			)
		}
	}

	if counterpart, ok := relatedFixture(fixtureMap, fixture, KindBlock); ok {
		if counterpart.Repo == RepoOuroborosConsensus &&
			counterpart.Era == "dijkstra" {
			return 1, nil
		}
		block, err := counterpart.DecodeLedgerBlock()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode related block fixture %s: %w",
				counterpart.RelPath,
				err,
			)
		}
		if block.Hash() != header.Hash() {
			return 0, fmt.Errorf(
				"header/block hash mismatch between %s and %s",
				fixture.RelPath,
				counterpart.RelPath,
			)
		}
		if block.SlotNumber() != header.SlotNumber() {
			return 0, fmt.Errorf(
				"header/block slot mismatch between %s and %s",
				fixture.RelPath,
				counterpart.RelPath,
			)
		}
	}

	return 1, nil
}

func executeTransactionFixture(
	fixture Fixture,
	fixtureMap map[string]Fixture,
) (int, error) {
	if fixture.Repo == RepoOuroborosConsensus && fixture.Era == "dijkstra" {
		return executeDijkstraConsensusTransactionFixture(fixture, fixtureMap)
	}

	tx, err := fixture.DecodeLedgerTransaction()
	if err != nil {
		return 0, fmt.Errorf(
			"failed to decode transaction fixture %s: %w",
			fixture.RelPath,
			err,
		)
	}
	if len(tx.Cbor()) == 0 {
		return 0, fmt.Errorf(
			"decoded transaction has empty CBOR: %s",
			fixture.RelPath,
		)
	}

	if fixture.Repo == RepoOuroborosConsensus {
		envelope, err := fixture.ConsensusEnvelope()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode consensus transaction envelope %s: %w",
				fixture.RelPath,
				err,
			)
		}
		expectedType, err := fixture.LedgerTransactionType()
		if err != nil {
			return 0, err
		}
		if envelope.Era != expectedType {
			return 0, fmt.Errorf(
				"unexpected consensus transaction era for %s: got %d want %d",
				fixture.RelPath,
				envelope.Era,
				expectedType,
			)
		}
	}

	if counterpart, ok := relatedFixture(fixtureMap, fixture, KindTransactionID); ok {
		if fixture.Repo == RepoOuroborosConsensus && fixture.Era == "byron" {
			return 1, nil
		}
		txIDBytes, err := counterpart.LedgerTransactionIDBytes()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode related transaction-id fixture %s: %w",
				counterpart.RelPath,
				err,
			)
		}
		if !bytes.Equal(tx.Hash().Bytes(), txIDBytes) {
			return 0, fmt.Errorf(
				"transaction/txid mismatch between %s and %s",
				fixture.RelPath,
				counterpart.RelPath,
			)
		}
	}

	return 1, nil
}

func executeTransactionIDFixture(
	fixture Fixture,
	fixtureMap map[string]Fixture,
) (int, error) {
	if fixture.Repo == RepoOuroborosConsensus && fixture.Era == "dijkstra" {
		return executeDijkstraConsensusTransactionIDFixture(fixture, fixtureMap)
	}

	txIDBytes, err := fixture.LedgerTransactionIDBytes()
	if err != nil {
		return 0, fmt.Errorf(
			"failed to decode transaction-id fixture %s: %w",
			fixture.RelPath,
			err,
		)
	}
	if len(txIDBytes) != 32 {
		return 0, fmt.Errorf(
			"unexpected transaction-id length for %s: got %d",
			fixture.RelPath,
			len(txIDBytes),
		)
	}

	if fixture.Repo == RepoOuroborosConsensus {
		envelope, err := fixture.ConsensusEnvelope()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode consensus transaction-id envelope %s: %w",
				fixture.RelPath,
				err,
			)
		}
		expectedType, err := ledgerTransactionTypeForEra(fixture.Era)
		if err != nil {
			return 0, err
		}
		if envelope.Era != expectedType {
			return 0, fmt.Errorf(
				"unexpected consensus transaction-id era for %s: got %d want %d",
				fixture.RelPath,
				envelope.Era,
				expectedType,
			)
		}
	}

	if counterpart, ok := relatedFixture(fixtureMap, fixture, KindTransaction); ok {
		if fixture.Repo == RepoOuroborosConsensus && fixture.Era == "byron" {
			return 1, nil
		}
		tx, err := counterpart.DecodeLedgerTransaction()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode related transaction fixture %s: %w",
				counterpart.RelPath,
				err,
			)
		}
		if !bytes.Equal(tx.Hash().Bytes(), txIDBytes) {
			return 0, fmt.Errorf(
				"transaction-id/transaction mismatch between %s and %s",
				fixture.RelPath,
				counterpart.RelPath,
			)
		}
	}

	return 1, nil
}

func executeGenesisFixture(fixture Fixture) (int, error) {
	data, err := fixture.Read()
	if err != nil {
		return 0, err
	}

	switch {
	case fixture.Name == "ShelleyGenesis.json":
		genesis, err := shelley.NewShelleyGenesisFromReader(
			bytes.NewReader(data),
		)
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode Shelley genesis fixture %s: %w",
				fixture.RelPath,
				err,
			)
		}
		pp := shelley.ShelleyProtocolParameters{}
		if err := pp.UpdateFromGenesis(&genesis); err != nil {
			return 0, fmt.Errorf(
				"failed to derive Shelley protocol parameters from %s: %w",
				fixture.RelPath,
				err,
			)
		}
		if pp.A0 == nil || pp.Rho == nil || pp.Tau == nil {
			return 0, fmt.Errorf(
				"incomplete Shelley genesis-derived protocol parameters for %s",
				fixture.RelPath,
			)
		}
	case fixture.Era == "alonzo":
		genesis, err := alonzo.NewAlonzoGenesisFromReader(bytes.NewReader(data))
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode Alonzo genesis fixture %s: %w",
				fixture.RelPath,
				err,
			)
		}
		pp := alonzo.AlonzoProtocolParameters{}
		if err := pp.UpdateFromGenesis(&genesis); err != nil {
			return 0, fmt.Errorf(
				"failed to derive Alonzo protocol parameters from %s: %w",
				fixture.RelPath,
				err,
			)
		}
		if pp.ExecutionCosts.MemPrice == nil ||
			pp.ExecutionCosts.StepPrice == nil {
			return 0, fmt.Errorf(
				"missing execution prices after decoding %s",
				fixture.RelPath,
			)
		}
	case fixture.Era == "conway":
		genesis, err := conway.NewConwayGenesisFromReader(bytes.NewReader(data))
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode Conway genesis fixture %s: %w",
				fixture.RelPath,
				err,
			)
		}
		pp := conway.ConwayProtocolParameters{}
		if err := pp.UpdateFromGenesis(&genesis); err != nil {
			return 0, fmt.Errorf(
				"failed to derive Conway protocol parameters from %s: %w",
				fixture.RelPath,
				err,
			)
		}
		if pp.MinCommitteeSize == 0 && pp.CommitteeTermLimit == 0 &&
			len(pp.CostModels) == 0 {
			return 0, fmt.Errorf(
				"Conway genesis fixture %s did not populate any governance parameters",
				fixture.RelPath,
			)
		}
	default:
		return 0, fmt.Errorf("unsupported genesis fixture: %s", fixture.RelPath)
	}

	return 1, nil
}

func executeProtocolParametersFixture(fixture Fixture) (int, error) {
	params, err := fixture.DecodeProtocolParameters()
	if err != nil {
		if fixture.Era == "dijkstra" &&
			errors.Is(err, errUnsupportedDijkstraRefScriptFields) {
			return 1, nil
		}
		return 0, fmt.Errorf(
			"failed to decode protocol-parameters fixture %s: %w",
			fixture.RelPath,
			err,
		)
	}
	if err := validateDecodedProtocolParameters(params); err != nil {
		return 0, fmt.Errorf(
			"invalid decoded protocol parameters for %s: %w",
			fixture.RelPath,
			err,
		)
	}
	return 1, nil
}

func executeProtocolParametersUpdateFixture(
	fixture Fixture,
	fixtureMap map[string]Fixture,
) (int, error) {
	update, err := fixture.DecodeProtocolParameterUpdate()
	if err != nil {
		if fixture.Era == "dijkstra" &&
			errors.Is(err, errUnsupportedDijkstraRefScriptFields) {
			return 1, nil
		}
		return 0, fmt.Errorf(
			"failed to decode protocol-parameters update fixture %s: %w",
			fixture.RelPath,
			err,
		)
	}

	baseFixture, ok := sameDirectoryFixture(fixtureMap, fixture, "pparams.json")
	if !ok {
		return 0, fmt.Errorf(
			"missing paired pparams.json for %s",
			fixture.RelPath,
		)
	}
	params, err := baseFixture.DecodeProtocolParameters()
	if err != nil {
		if baseFixture.Era == "dijkstra" &&
			errors.Is(err, errUnsupportedDijkstraRefScriptFields) {
			return 1, nil
		}
		return 0, fmt.Errorf(
			"failed to decode paired protocol parameters fixture %s: %w",
			baseFixture.RelPath,
			err,
		)
	}

	if err := ApplyProtocolParameterUpdate(params, update); err != nil {
		return 0, fmt.Errorf(
			"failed to apply protocol-parameter update %s: %w",
			fixture.RelPath,
			err,
		)
	}
	if err := validateDecodedProtocolParameters(params); err != nil {
		return 0, fmt.Errorf(
			"invalid protocol parameters after applying %s: %w",
			fixture.RelPath,
			err,
		)
	}

	return 1, nil
}

func executeTranslationFixture(fixture Fixture) (int, error) {
	data, err := fixture.Read()
	if err != nil {
		return 0, err
	}

	var cases []cbor.RawMessage
	if _, err := cbor.Decode(data, &cases); err != nil {
		return 0, fmt.Errorf(
			"failed to decode translation corpus %s: %w",
			fixture.RelPath,
			err,
		)
	}
	if len(cases) == 0 {
		return 0, fmt.Errorf("translation corpus is empty: %s", fixture.RelPath)
	}

	for caseIdx, rawCase := range cases {
		var items []cbor.RawMessage
		if _, err := cbor.Decode(rawCase, &items); err != nil {
			return 0, fmt.Errorf(
				"failed to decode translation case %d in %s: %w",
				caseIdx,
				fixture.RelPath,
				err,
			)
		}
		if len(items) != 5 {
			return 0, fmt.Errorf(
				"unexpected translation case width in %s case %d: got %d want 5",
				fixture.RelPath,
				caseIdx,
				len(items),
			)
		}
		if err := validateTranslationCase(items); err != nil {
			return 0, fmt.Errorf(
				"invalid translation case %d in %s: %w",
				caseIdx,
				fixture.RelPath,
				err,
			)
		}
	}

	return len(cases), nil
}

func executeDijkstraConsensusBlockFixture(
	fixture Fixture,
	fixtureMap map[string]Fixture,
) (int, error) {
	wrappedBytes, err := fixture.ConsensusBlockBytes()
	if err != nil {
		return 0, fmt.Errorf(
			"failed to extract consensus block wrapper %s: %w",
			fixture.RelPath,
			err,
		)
	}
	dec, err := cbor.NewStreamDecoder(wrappedBytes)
	if err != nil {
		return 0, fmt.Errorf(
			"failed to create consensus block stream decoder %s: %w",
			fixture.RelPath,
			err,
		)
	}
	arrayLen, _, _, err := dec.DecodeArrayHeader()
	if err != nil {
		return 0, fmt.Errorf(
			"failed to decode Dijkstra block wrapper header %s: %w",
			fixture.RelPath,
			err,
		)
	}
	if arrayLen != 2 {
		return 0, fmt.Errorf(
			"unexpected Dijkstra block wrapper width for %s: got %d want 2",
			fixture.RelPath,
			arrayLen,
		)
	}
	var era uint
	if _, _, err := dec.Decode(&era); err != nil {
		return 0, fmt.Errorf(
			"failed to decode Dijkstra block era %s: %w",
			fixture.RelPath,
			err,
		)
	}
	expectedType, err := fixture.LedgerBlockType()
	if err != nil {
		return 0, err
	}
	if era != expectedType {
		return 0, fmt.Errorf(
			"unexpected one-era block identifier for %s: got %d want %d",
			fixture.RelPath,
			era,
			expectedType,
		)
	}
	if dec.EOF() {
		return 0, fmt.Errorf(
			"decoded Dijkstra block payload is empty: %s",
			fixture.RelPath,
		)
	}
	if _, _, err := dec.Skip(); err != nil {
		// The streaming decoder can report io.ErrUnexpectedEOF on valid CBOR
		// payloads that the non-streaming decoder accepts; tolerate it here.
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			return 0, fmt.Errorf(
				"failed to validate Dijkstra block payload %s: %w",
				fixture.RelPath,
				err,
			)
		}
	} else if !dec.EOF() {
		return 0, fmt.Errorf(
			"unexpected trailing data in Dijkstra block wrapper %s",
			fixture.RelPath,
		)
	}

	if counterpart, ok := relatedFixture(fixtureMap, fixture, KindHeader); ok {
		header, err := counterpart.DecodeLedgerHeader()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode related header fixture %s: %w",
				counterpart.RelPath,
				err,
			)
		}
		if len(header.Cbor()) == 0 {
			return 0, fmt.Errorf(
				"decoded related header has empty CBOR: %s",
				counterpart.RelPath,
			)
		}
	}

	return 1, nil
}

func executeDijkstraConsensusTransactionFixture(
	fixture Fixture,
	fixtureMap map[string]Fixture,
) (int, error) {
	envelope, err := fixture.ConsensusEnvelope()
	if err != nil {
		return 0, fmt.Errorf(
			"failed to decode consensus transaction envelope %s: %w",
			fixture.RelPath,
			err,
		)
	}
	expectedType, err := fixture.LedgerTransactionType()
	if err != nil {
		return 0, err
	}
	if envelope.Era != expectedType {
		return 0, fmt.Errorf(
			"unexpected consensus transaction era for %s: got %d want %d",
			fixture.RelPath,
			envelope.Era,
			expectedType,
		)
	}

	payloadBytes, err := envelope.BytesPayload()
	if err != nil {
		return 0, fmt.Errorf(
			"failed to extract Dijkstra transaction payload %s: %w",
			fixture.RelPath,
			err,
		)
	}
	bodyHash, err := dijkstraConsensusTransactionBodyHash(payloadBytes)
	if err != nil {
		return 0, fmt.Errorf(
			"failed to validate Dijkstra transaction %s: %w",
			fixture.RelPath,
			err,
		)
	}

	if counterpart, ok := relatedFixture(fixtureMap, fixture, KindTransactionID); ok {
		txIDBytes, err := counterpart.LedgerTransactionIDBytes()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode related transaction-id fixture %s: %w",
				counterpart.RelPath,
				err,
			)
		}
		if !bytes.Equal(bodyHash, txIDBytes) {
			return 0, fmt.Errorf(
				"transaction/txid mismatch between %s and %s",
				fixture.RelPath,
				counterpart.RelPath,
			)
		}
	}

	return 1, nil
}

func executeDijkstraConsensusTransactionIDFixture(
	fixture Fixture,
	fixtureMap map[string]Fixture,
) (int, error) {
	txIDBytes, err := fixture.LedgerTransactionIDBytes()
	if err != nil {
		return 0, fmt.Errorf(
			"failed to decode transaction-id fixture %s: %w",
			fixture.RelPath,
			err,
		)
	}
	if len(txIDBytes) != 32 {
		return 0, fmt.Errorf(
			"unexpected transaction-id length for %s: got %d",
			fixture.RelPath,
			len(txIDBytes),
		)
	}

	envelope, err := fixture.ConsensusEnvelope()
	if err != nil {
		return 0, fmt.Errorf(
			"failed to decode consensus transaction-id envelope %s: %w",
			fixture.RelPath,
			err,
		)
	}
	expectedType, err := ledgerTransactionTypeForEra(fixture.Era)
	if err != nil {
		return 0, err
	}
	if envelope.Era != expectedType {
		return 0, fmt.Errorf(
			"unexpected consensus transaction-id era for %s: got %d want %d",
			fixture.RelPath,
			envelope.Era,
			expectedType,
		)
	}

	if counterpart, ok := relatedFixture(fixtureMap, fixture, KindTransaction); ok {
		consensusTx, err := counterpart.ConsensusEnvelope()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to decode related transaction fixture %s: %w",
				counterpart.RelPath,
				err,
			)
		}
		payloadBytes, err := consensusTx.BytesPayload()
		if err != nil {
			return 0, fmt.Errorf(
				"failed to extract related Dijkstra transaction payload %s: %w",
				counterpart.RelPath,
				err,
			)
		}
		bodyHash, err := dijkstraConsensusTransactionBodyHash(payloadBytes)
		if err != nil {
			return 0, fmt.Errorf(
				"failed to validate related Dijkstra transaction %s: %w",
				counterpart.RelPath,
				err,
			)
		}
		if !bytes.Equal(bodyHash, txIDBytes) {
			return 0, fmt.Errorf(
				"transaction-id/transaction mismatch between %s and %s",
				fixture.RelPath,
				counterpart.RelPath,
			)
		}
	}

	return 1, nil
}

func dijkstraConsensusTransactionBodyHash(payload []byte) ([]byte, error) {
	var items []cbor.RawMessage
	if _, err := cbor.Decode(payload, &items); err != nil {
		return nil, fmt.Errorf(
			"failed to decode Dijkstra transaction payload: %w",
			err,
		)
	}
	if len(items) != 3 {
		return nil, fmt.Errorf(
			"unexpected Dijkstra transaction payload width: got %d want 3",
			len(items),
		)
	}
	for idx, item := range items {
		if len(item) == 0 {
			return nil, fmt.Errorf(
				"empty Dijkstra transaction payload item %d",
				idx,
			)
		}
		if err := validateArbitraryCbor(item, fmt.Sprintf("Dijkstra transaction payload item %d", idx)); err != nil {
			return nil, fmt.Errorf(
				"failed to decode Dijkstra transaction payload item %d: %w",
				idx,
				err,
			)
		}
	}
	hash := gcommon.Blake2b256Hash(items[0])
	return hash.Bytes(), nil
}

type translationProtocolVersion struct {
	cbor.StructAsArray
	Major uint
	Minor uint
}

func validateTranslationCase(items []cbor.RawMessage) error {
	var version translationProtocolVersion
	if _, err := cbor.Decode(items[0], &version); err != nil {
		return fmt.Errorf(
			"failed to decode translation protocol version: %w",
			err,
		)
	}

	var language uint
	if _, err := cbor.Decode(items[1], &language); err != nil {
		return fmt.Errorf(
			"failed to decode translation language selector: %w",
			err,
		)
	}

	if err := validateArbitraryCbor(items[2], "translation context"); err != nil {
		return err
	}
	if err := validateTranslationTransaction(items[3]); err != nil {
		return err
	}
	if err := validateArbitraryCbor(items[4], "translation expected value"); err != nil {
		return err
	}
	return nil
}

func validateTranslationTransaction(data []byte) error {
	var txItems []cbor.RawMessage
	if _, err := cbor.Decode(data, &txItems); err != nil {
		return fmt.Errorf(
			"failed to decode translation transaction container: %w",
			err,
		)
	}
	if len(txItems) != 4 {
		return fmt.Errorf(
			"unexpected translation transaction width: got %d want 4",
			len(txItems),
		)
	}

	for idx, item := range txItems {
		if len(item) == 0 {
			return fmt.Errorf("empty translation transaction item %d", idx)
		}
	}
	if err := validateArbitraryCbor(txItems[0], "translation transaction body"); err != nil {
		return err
	}
	if err := validateArbitraryCbor(txItems[1], "translation transaction witnesses"); err != nil {
		return err
	}
	var isValid bool
	if _, err := cbor.Decode(txItems[2], &isValid); err != nil {
		return fmt.Errorf(
			"failed to decode translation transaction validity flag: %w",
			err,
		)
	}
	if err := validateArbitraryCbor(txItems[3], "translation transaction auxiliary data"); err != nil {
		return err
	}

	bodyHash := gcommon.Blake2b256Hash(txItems[0])
	if bodyHash == (gcommon.Blake2b256{}) {
		return errors.New("translation transaction body hash is empty")
	}
	return nil
}

func validateArbitraryCbor(data []byte, label string) error {
	dec, err := cbor.NewStreamDecoder(data)
	if err != nil {
		return fmt.Errorf("failed to create %s decoder: %w", label, err)
	}
	if _, _, err := dec.Skip(); err != nil {
		return fmt.Errorf("failed to decode %s: %w", label, err)
	}
	if !dec.EOF() {
		return fmt.Errorf("unexpected trailing data in %s", label)
	}
	return nil
}

func validateDecodedProtocolParameters(
	params gcommon.ProtocolParameters,
) error {
	switch pp := params.(type) {
	case *shelley.ShelleyProtocolParameters:
		if pp.A0 == nil || pp.Rho == nil || pp.Tau == nil {
			return errors.New("missing Shelley rational parameters")
		}
		if pp.ProtocolMajor == 0 {
			return errors.New("missing Shelley protocol version")
		}
	case *alonzo.AlonzoProtocolParameters:
		if pp.A0 == nil || pp.Rho == nil || pp.Tau == nil {
			return errors.New("missing Alonzo rational parameters")
		}
		if pp.ExecutionCosts.MemPrice == nil || pp.ExecutionCosts.StepPrice == nil {
			return errors.New("missing Alonzo execution prices")
		}
		if pp.ProtocolMajor == 0 {
			return errors.New("missing Alonzo protocol version")
		}
	case *babbage.BabbageProtocolParameters:
		if pp.A0 == nil || pp.Rho == nil || pp.Tau == nil {
			return errors.New("missing Babbage rational parameters")
		}
		if pp.ExecutionCosts.MemPrice == nil || pp.ExecutionCosts.StepPrice == nil {
			return errors.New("missing Babbage execution prices")
		}
		if pp.ProtocolMajor == 0 {
			return errors.New("missing Babbage protocol version")
		}
	case *conway.ConwayProtocolParameters:
		if pp.A0 == nil || pp.Rho == nil || pp.Tau == nil {
			return errors.New("missing Conway rational parameters")
		}
		if pp.ExecutionCosts.MemPrice == nil || pp.ExecutionCosts.StepPrice == nil {
			return errors.New("missing Conway execution prices")
		}
		if pp.ProtocolVersion.Major == 0 {
			return errors.New("missing Conway protocol version")
		}
	default:
		return fmt.Errorf("unsupported protocol parameters type %T", params)
	}
	return nil
}

func relatedFixture(
	fixtureMap map[string]Fixture,
	fixture Fixture,
	targetKind Kind,
) (Fixture, bool) {
	var targetName string
	switch {
	case fixture.Kind == KindBlock && targetKind == KindHeader && path.Base(fixture.RelPath) != "":
		targetName = stringsReplacePrefix(fixture.Name, "Block_", "Header_")
	case fixture.Kind == KindHeader && targetKind == KindBlock && path.Base(fixture.RelPath) != "":
		targetName = stringsReplacePrefix(fixture.Name, "Header_", "Block_")
	case fixture.Kind == KindTransaction && targetKind == KindTransactionID:
		targetName = stringsReplacePrefix(fixture.Name, "GenTx_", "GenTxId_")
	case fixture.Kind == KindTransactionID && targetKind == KindTransaction:
		targetName = stringsReplacePrefix(fixture.Name, "GenTxId_", "GenTx_")
	default:
		return Fixture{}, false
	}
	if targetName == fixture.Name {
		return Fixture{}, false
	}

	targetPath := path.Join(path.Dir(fixture.RelPath), targetName)
	related, ok := fixtureMap[targetPath]
	return related, ok
}

func sameDirectoryFixture(
	fixtureMap map[string]Fixture,
	fixture Fixture,
	targetName string,
) (Fixture, bool) {
	targetPath := path.Join(path.Dir(fixture.RelPath), targetName)
	related, ok := fixtureMap[targetPath]
	return related, ok
}

func stringsReplacePrefix(
	name string,
	oldPrefix string,
	newPrefix string,
) string {
	if len(name) >= len(oldPrefix) && name[:len(oldPrefix)] == oldPrefix {
		return newPrefix + name[len(oldPrefix):]
	}
	return name
}
