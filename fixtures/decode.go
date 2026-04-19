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
	"fmt"
	"os"
	"strings"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
)

// ConsensusBlock represents a tag-24 wrapped one-era consensus block.
type ConsensusBlock struct {
	Era   uint
	Block cbor.RawMessage
}

// ConsensusEnvelope represents a 2-element ouroboros-consensus envelope where
// the first element is the era and the second element is the wrapped payload.
type ConsensusEnvelope struct {
	Era     uint
	Payload cbor.RawMessage
}

// Read returns the raw fixture file contents from disk.
func (f Fixture) Read() ([]byte, error) {
	return os.ReadFile(f.Path)
}

// DecodeHex decodes a hex-encoded fixture payload.
func (f Fixture) DecodeHex() ([]byte, error) {
	if f.Format != FormatHex {
		return nil, fmt.Errorf("fixture %s is not hex-encoded", f.RelPath)
	}

	data, err := f.Read()
	if err != nil {
		return nil, err
	}

	decoded, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf(
			"failed to decode hex fixture %s: %w",
			f.RelPath,
			err,
		)
	}
	return decoded, nil
}

// LedgerBlockType infers the gouroboros ledger block type for block fixtures.
func (f Fixture) LedgerBlockType() (uint, error) {
	if f.Kind != KindBlock {
		return 0, fmt.Errorf("fixture %s is not a block fixture", f.RelPath)
	}
	if f.Era == "byron" {
		return f.byronLedgerBlockType()
	}
	return ledgerBlockTypeForEra(f.Era)
}

// LedgerHeaderType infers the gouroboros ledger header type for header fixtures.
func (f Fixture) LedgerHeaderType() (uint, error) {
	if f.Kind != KindHeader {
		return 0, fmt.Errorf("fixture %s is not a header fixture", f.RelPath)
	}
	if f.Era == "byron" {
		return f.byronLedgerHeaderType()
	}
	return ledgerBlockTypeForEra(f.Era)
}

func (f Fixture) byronLedgerBlockType() (uint, error) {
	if f.Repo != RepoOuroborosConsensus {
		return 0, fmt.Errorf(
			"cannot determine Byron block type for non-consensus fixture %s",
			f.RelPath,
		)
	}

	block, err := f.ConsensusBlock()
	if err != nil {
		return 0, err
	}
	switch block.Era {
	case ledger.BlockTypeByronEbb, ledger.BlockTypeByronMain:
		return block.Era, nil
	default:
		return 0, fmt.Errorf(
			"unexpected Byron block type %d in %s",
			block.Era,
			f.RelPath,
		)
	}
}

func (f Fixture) byronLedgerHeaderType() (uint, error) {
	if f.Repo != RepoOuroborosConsensus {
		return 0, fmt.Errorf(
			"cannot determine Byron header type for non-consensus fixture %s",
			f.RelPath,
		)
	}

	header, err := f.ConsensusHeader()
	if err != nil {
		return 0, err
	}
	switch header.ByronType() {
	case ledger.BlockTypeByronEbb, ledger.BlockTypeByronMain:
		return header.ByronType(), nil
	default:
		return 0, fmt.Errorf(
			"unexpected Byron header type %d in %s",
			header.ByronType(),
			f.RelPath,
		)
	}
}

func ledgerBlockTypeForEra(era string) (uint, error) {
	switch era {
	case "shelley":
		return ledger.BlockTypeShelley, nil
	case "allegra":
		return ledger.BlockTypeAllegra, nil
	case "mary":
		return ledger.BlockTypeMary, nil
	case "alonzo":
		return ledger.BlockTypeAlonzo, nil
	case "babbage":
		return ledger.BlockTypeBabbage, nil
	case "conway":
		return ledger.BlockTypeConway, nil
	case "dijkstra":
		return ledger.BlockTypeLeiosRanking, nil
	default:
		return 0, fmt.Errorf("unknown block era %q", era)
	}
}

// LedgerTransactionType infers the gouroboros ledger transaction type for transaction fixtures.
func (f Fixture) LedgerTransactionType() (uint, error) {
	if f.Repo == RepoCardanoAPI && f.Name == "tx-canonical.json" {
		return ledger.TxTypeConway, nil
	}
	if f.Kind != KindTransaction {
		return 0, fmt.Errorf(
			"fixture %s is not a transaction fixture",
			f.RelPath,
		)
	}
	return ledgerTransactionTypeForEra(f.Era)
}

func ledgerTransactionTypeForEra(era string) (uint, error) {
	switch era {
	case "byron":
		return ledger.TxTypeByron, nil
	case "shelley":
		return ledger.TxTypeShelley, nil
	case "allegra":
		return ledger.TxTypeAllegra, nil
	case "mary":
		return ledger.TxTypeMary, nil
	case "alonzo":
		return ledger.TxTypeAlonzo, nil
	case "babbage":
		return ledger.TxTypeBabbage, nil
	case "conway":
		return ledger.TxTypeConway, nil
	case "dijkstra":
		return ledger.TxTypeLeios, nil
	default:
		return 0, fmt.Errorf("unknown transaction era %q", era)
	}
}

// ConsensusHeaderEra returns the NtN chainsync era identifier for consensus header fixtures.
func (f Fixture) ConsensusHeaderEra() (uint, error) {
	return consensusHeaderTypeForEra(f.Era)
}

func consensusHeaderTypeForEra(era string) (uint, error) {
	switch era {
	case "byron":
		return ledger.BlockHeaderTypeByron, nil
	case "shelley":
		return ledger.BlockHeaderTypeShelley, nil
	case "allegra":
		return ledger.BlockHeaderTypeAllegra, nil
	case "mary":
		return ledger.BlockHeaderTypeMary, nil
	case "alonzo":
		return ledger.BlockHeaderTypeAlonzo, nil
	case "babbage":
		return ledger.BlockHeaderTypeBabbage, nil
	case "conway":
		return ledger.BlockHeaderTypeConway, nil
	case "dijkstra":
		return ledger.BlockHeaderTypeLeios, nil
	default:
		return 0, fmt.Errorf("unknown header era %q", era)
	}
}

// ConsensusBlockBytes unwraps a tag-24 ouroboros-consensus block fixture and returns
// the raw one-era block wrapper bytes.
func (f Fixture) ConsensusBlockBytes() ([]byte, error) {
	if f.Repo != RepoOuroborosConsensus || f.Kind != KindBlock {
		return nil, fmt.Errorf(
			"fixture %s is not an ouroboros-consensus block fixture",
			f.RelPath,
		)
	}

	data, err := f.Read()
	if err != nil {
		return nil, err
	}
	return unwrapTag24(data)
}

// ConsensusBlock decodes an ouroboros-consensus one-era block fixture.
func (f Fixture) ConsensusBlock() (*ConsensusBlock, error) {
	wrappedBytes, err := f.ConsensusBlockBytes()
	if err != nil {
		return nil, err
	}

	var blockItems []cbor.RawMessage
	if _, err := cbor.Decode(wrappedBytes, &blockItems); err != nil {
		return nil, err
	}
	if len(blockItems) != 2 {
		return nil, fmt.Errorf(
			"expected 2-element block wrapper, got %d",
			len(blockItems),
		)
	}

	var era uint
	if _, err := cbor.Decode(blockItems[0], &era); err != nil {
		return nil, fmt.Errorf(
			"failed to decode one-era block identifier: %w",
			err,
		)
	}

	return &ConsensusBlock{
		Era:   era,
		Block: blockItems[1],
	}, nil
}

// ConsensusLedgerBlockBytes returns the actual ledger block bytes from an
// ouroboros-consensus block fixture.
func (f Fixture) ConsensusLedgerBlockBytes() ([]byte, error) {
	block, err := f.ConsensusBlock()
	if err != nil {
		return nil, err
	}
	return block.Block, nil
}

// ConsensusHeader decodes an ouroboros-consensus wrapped header fixture.
func (f Fixture) ConsensusHeader() (*chainsync.WrappedHeader, error) {
	if f.Repo != RepoOuroborosConsensus || f.Kind != KindHeader {
		return nil, fmt.Errorf(
			"fixture %s is not an ouroboros-consensus header fixture",
			f.RelPath,
		)
	}

	data, err := f.Read()
	if err != nil {
		return nil, err
	}

	wrappedHeader := &chainsync.WrappedHeader{}
	if _, err := cbor.Decode(data, wrappedHeader); err != nil {
		return nil, err
	}
	return wrappedHeader, nil
}

// ConsensusHeaderBytes returns the raw header bytes from an ouroboros-consensus wrapped header fixture.
func (f Fixture) ConsensusHeaderBytes() ([]byte, error) {
	wrappedHeader, err := f.ConsensusHeader()
	if err != nil {
		return nil, err
	}
	return wrappedHeader.HeaderCbor(), nil
}

// ConsensusEnvelope decodes a 2-element ouroboros-consensus transaction or transaction-id envelope.
func (f Fixture) ConsensusEnvelope() (*ConsensusEnvelope, error) {
	if f.Repo != RepoOuroborosConsensus {
		return nil, fmt.Errorf(
			"fixture %s is not an ouroboros-consensus fixture",
			f.RelPath,
		)
	}
	if f.Kind != KindTransaction && f.Kind != KindTransactionID {
		return nil, fmt.Errorf(
			"fixture %s is not a consensus envelope fixture",
			f.RelPath,
		)
	}

	data, err := f.Read()
	if err != nil {
		return nil, err
	}

	var envelope []cbor.RawMessage
	if _, err := cbor.Decode(data, &envelope); err != nil {
		return nil, err
	}
	if len(envelope) != 2 {
		return nil, fmt.Errorf(
			"expected 2-element envelope, got %d",
			len(envelope),
		)
	}

	var era uint
	if _, err := cbor.Decode(envelope[0], &era); err != nil {
		return nil, fmt.Errorf("failed to decode envelope era: %w", err)
	}

	return &ConsensusEnvelope{
		Era:     era,
		Payload: envelope[1],
	}, nil
}

// ConsensusTransactionBytes returns the actual ledger transaction bytes from a
// consensus transaction fixture.
func (f Fixture) ConsensusTransactionBytes() ([]byte, error) {
	if f.Kind != KindTransaction {
		return nil, fmt.Errorf("%w: fixture %s", ErrNotTransactionFixture, f.RelPath)
	}

	envelope, err := f.ConsensusEnvelope()
	if err != nil {
		return nil, err
	}

	if f.Era == "byron" {
		var nested []cbor.RawMessage
		if _, err := cbor.Decode(envelope.Payload, &nested); err != nil {
			return nil, err
		}
		if len(nested) != 2 {
			return nil, fmt.Errorf(
				"expected 2-element Byron transaction wrapper, got %d",
				len(nested),
			)
		}
		return nested[1], nil
	}

	// Try tag-24 unwrapping (standard post-Byron eras)
	if tagged, err := unwrapTag24(envelope.Payload); err == nil {
		return tagged, nil
	}

	// Try bytes payload (Dijkstra era uses raw bytes encoding)
	if bytesPayload, err := decodeEnvelopeBytesPayload(envelope.Payload); err == nil {
		return bytesPayload, nil
	}

	return nil, fmt.Errorf(
		"failed to extract transaction bytes from consensus envelope for %s",
		f.RelPath,
	)
}

// ConsensusTransactionIDBytes returns the transaction identifier payload bytes
// from a consensus transaction-id fixture.
func (f Fixture) ConsensusTransactionIDBytes() ([]byte, error) {
	envelope, err := f.ConsensusEnvelope()
	if err != nil {
		return nil, err
	}
	envelopeKind, err := envelope.Kind()
	if err != nil {
		return nil, err
	}
	if envelopeKind != KindTransactionID {
		return nil, fmt.Errorf(
			"%w: got %s in fixture %s",
			ErrNotTransactionIDEnvelope,
			envelopeKind,
			f.RelPath,
		)
	}
	return envelope.BytesPayload()
}

// LedgerBlockBytes returns the actual ledger block bytes for any supported
// block fixture family.
func (f Fixture) LedgerBlockBytes() ([]byte, error) {
	if f.Kind != KindBlock {
		return nil, fmt.Errorf("fixture %s is not a block fixture", f.RelPath)
	}

	switch {
	case f.Repo == RepoOuroborosConsensus:
		return f.ConsensusLedgerBlockBytes()
	case f.Format == FormatHex:
		return f.DecodeHex()
	case f.Format == FormatCBOR:
		return f.Read()
	default:
		return nil, fmt.Errorf(
			"fixture %s does not expose ledger block bytes",
			f.RelPath,
		)
	}
}

// LedgerHeaderBytes returns the actual ledger header bytes for any supported
// header fixture family.
func (f Fixture) LedgerHeaderBytes() ([]byte, error) {
	if f.Kind != KindHeader {
		return nil, fmt.Errorf("fixture %s is not a header fixture", f.RelPath)
	}

	switch {
	case f.Repo == RepoOuroborosConsensus:
		return f.ConsensusHeaderBytes()
	case f.Format == FormatCBOR:
		return f.Read()
	default:
		return nil, fmt.Errorf(
			"fixture %s does not expose ledger header bytes",
			f.RelPath,
		)
	}
}

// LedgerTransactionBytes returns the actual ledger transaction bytes for any
// supported transaction fixture family.
func (f Fixture) LedgerTransactionBytes() ([]byte, error) {
	if f.Kind != KindTransaction {
		return nil, fmt.Errorf(
			"fixture %s is not a transaction fixture",
			f.RelPath,
		)
	}

	switch {
	case f.Repo == RepoOuroborosConsensus:
		return f.ConsensusTransactionBytes()
	case f.Repo == RepoCardanoAPI && f.Name == "tx-canonical.json":
		data, err := f.Read()
		if err != nil {
			return nil, err
		}
		var payload canonicalTransaction
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, fmt.Errorf(
				"failed to decode canonical tx fixture %s: %w",
				f.RelPath,
				err,
			)
		}
		return hex.DecodeString(strings.TrimSpace(payload.CborHex))
	case f.Format == FormatCBOR:
		return f.Read()
	default:
		return nil, fmt.Errorf(
			"fixture %s does not expose ledger transaction bytes",
			f.RelPath,
		)
	}
}

// LedgerTransactionIDBytes returns the transaction identifier bytes for any
// supported transaction-id fixture family.
func (f Fixture) LedgerTransactionIDBytes() ([]byte, error) {
	if f.Kind != KindTransactionID {
		return nil, fmt.Errorf(
			"fixture %s is not a transaction-id fixture",
			f.RelPath,
		)
	}

	if f.Repo == RepoOuroborosConsensus {
		return f.ConsensusTransactionIDBytes()
	}
	return nil, fmt.Errorf(
		"fixture %s does not expose transaction-id bytes",
		f.RelPath,
	)
}

// DecodeLedgerBlock decodes any supported block fixture into a ledger block.
func (f Fixture) DecodeLedgerBlock() (ledger.Block, error) {
	blockType, err := f.LedgerBlockType()
	if err != nil {
		return nil, err
	}
	blockBytes, err := f.LedgerBlockBytes()
	if err != nil {
		return nil, err
	}
	return ledger.NewBlockFromCbor(blockType, blockBytes)
}

// DecodeLedgerHeader decodes any supported header fixture into a ledger block header.
func (f Fixture) DecodeLedgerHeader() (ledger.BlockHeader, error) {
	headerType, err := f.LedgerHeaderType()
	if err != nil {
		return nil, err
	}
	headerBytes, err := f.LedgerHeaderBytes()
	if err != nil {
		return nil, err
	}
	return ledger.NewBlockHeaderFromCbor(headerType, headerBytes)
}

// DecodeLedgerTransaction decodes any supported transaction fixture into a ledger transaction.
func (f Fixture) DecodeLedgerTransaction() (ledger.Transaction, error) {
	txType, err := f.LedgerTransactionType()
	if err != nil {
		return nil, err
	}
	txBytes, err := f.LedgerTransactionBytes()
	if err != nil {
		return nil, err
	}
	return ledger.NewTransactionFromCbor(txType, txBytes)
}

// TaggedPayloadBytes decodes a tag-24 payload from the envelope.
func (e ConsensusEnvelope) TaggedPayloadBytes() ([]byte, error) {
	return unwrapTag24(e.Payload)
}

// txIDHashLen is the length of a Cardano transaction ID (Blake2b-256 hash).
const txIDHashLen = 32

// Kind infers whether the envelope contains a transaction or transaction-id payload.
func (e ConsensusEnvelope) Kind() (Kind, error) {
	if _, err := unwrapTag24(e.Payload); err == nil {
		return KindTransaction, nil
	}
	if payload, err := decodeEnvelopeBytesPayload(e.Payload); err == nil {
		// Transaction IDs are 32-byte Blake2b-256 hashes;
		// byte-string transaction payloads (e.g. Dijkstra GenTx) are larger.
		if len(payload) == txIDHashLen {
			return KindTransactionID, nil
		}
		return KindTransaction, nil
	}

	var nested []cbor.RawMessage
	if _, err := cbor.Decode(e.Payload, &nested); err != nil {
		return KindUnknown, fmt.Errorf(
			"failed to determine consensus envelope kind: %w",
			err,
		)
	}
	if len(nested) != 2 {
		return KindUnknown, fmt.Errorf(
			"failed to determine consensus envelope kind: expected 2-element payload, got %d",
			len(nested),
		)
	}
	return KindTransaction, nil
}

// BytesPayload decodes a byte-string payload from the envelope.
func (e ConsensusEnvelope) BytesPayload() ([]byte, error) {
	return decodeEnvelopeBytesPayload(e.Payload)
}

func decodeEnvelopeBytesPayload(data []byte) ([]byte, error) {
	var payload []byte
	if _, err := cbor.Decode(data, &payload); err != nil {
		var nested []cbor.RawMessage
		if _, nestedErr := cbor.Decode(data, &nested); nestedErr != nil {
			return nil, err
		}
		if len(nested) != 2 {
			return nil, err
		}
		if _, nestedErr := cbor.Decode(nested[1], &payload); nestedErr != nil {
			return nil, err
		}
	}
	return payload, nil
}

type canonicalTransaction struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	CborHex     string `json:"cborHex"`
}

func unwrapTag24(data []byte) ([]byte, error) {
	var tag cbor.Tag
	if _, err := cbor.Decode(data, &tag); err != nil {
		return nil, err
	}
	if tag.Number != 24 {
		return nil, fmt.Errorf("unexpected tag number %d", tag.Number)
	}
	content, ok := tag.Content.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected tag payload type %T", tag.Content)
	}
	return content, nil
}
