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
	"path/filepath"
	"strings"
)

// Repo identifies the upstream repository that produced a fixture.
type Repo string

const (
	RepoCardanoAPI         Repo = "cardano-api"
	RepoCardanoLedger      Repo = "cardano-ledger"
	RepoCardanoNode        Repo = "cardano-node"
	RepoOuroborosConsensus Repo = "ouroboros-consensus"
)

// Kind identifies the fixture payload category.
type Kind string

const (
	KindUnknown                  Kind = "unknown"
	KindBlock                    Kind = "block"
	KindGenesis                  Kind = "genesis"
	KindGovernanceMetadata       Kind = "governance-metadata"
	KindHeader                   Kind = "header"
	KindProtocolParameters       Kind = "protocol-parameters"
	KindProtocolParametersUpdate Kind = "protocol-parameters-update"
	KindTransaction              Kind = "transaction"
	KindTransactionID            Kind = "transaction-id"
	KindTranslation              Kind = "translation"
)

// Format identifies the fixture serialization format.
type Format string

const (
	FormatUnknown Format = "unknown"
	FormatCBOR    Format = "cbor"
	FormatHex     Format = "hex"
	FormatJSON    Format = "json"
	FormatJSONLD  Format = "jsonld"
)

// Filter selects fixtures for RunMatching.
type Filter struct {
	Repo       Repo
	Kind       Kind
	Format     Format
	Era        string
	PathPrefix string
}

// Matches reports whether the fixture matches the filter.
func (f Filter) Matches(fixture Fixture) bool {
	if f.Repo != "" && fixture.Repo != f.Repo {
		return false
	}
	if f.Kind != "" && fixture.Kind != f.Kind {
		return false
	}
	if f.Format != "" && fixture.Format != f.Format {
		return false
	}
	if f.Era != "" && !strings.EqualFold(fixture.Era, f.Era) {
		return false
	}
	if f.PathPrefix != "" {
		prefix := normalizeRelativePath(f.PathPrefix)
		if !strings.HasPrefix(fixture.RelPath, prefix) {
			return false
		}
		// Enforce directory boundary so "foo/bar" does not match "foo/bar-baz/…"
		if len(fixture.RelPath) > len(prefix) &&
			!strings.HasSuffix(prefix, "/") &&
			fixture.RelPath[len(prefix)] != '/' {
			return false
		}
	}
	return true
}

func classifyFixture(relPath string) (Kind, Format, string) {
	normalizedPath := filepath.ToSlash(relPath)
	baseName := filepath.Base(normalizedPath)

	switch {
	case strings.HasPrefix(baseName, "Block_"):
		return KindBlock, FormatCBOR, normalizeConsensusEra(
			strings.TrimPrefix(baseName, "Block_"),
		)
	case strings.HasPrefix(baseName, "Header_"):
		return KindHeader, FormatCBOR, normalizeConsensusEra(
			strings.TrimPrefix(baseName, "Header_"),
		)
	case strings.HasPrefix(baseName, "GenTxId_"):
		return KindTransactionID, FormatCBOR, normalizeConsensusEra(
			strings.TrimPrefix(baseName, "GenTxId_"),
		)
	case strings.HasPrefix(baseName, "GenTx_"):
		return KindTransaction, FormatCBOR, normalizeConsensusEra(
			strings.TrimPrefix(baseName, "GenTx_"),
		)
	case baseName == "pparams.json":
		return KindProtocolParameters, FormatJSON, eraFromPath(normalizedPath)
	case baseName == "pparams-update.json":
		return KindProtocolParametersUpdate, FormatJSON, eraFromPath(
			normalizedPath,
		)
	case baseName == "translations.cbor":
		return KindTranslation, FormatCBOR, eraFromPath(normalizedPath)
	case strings.HasPrefix(baseName, "hex-block-"):
		return KindBlock, FormatHex, eraFromPath(normalizedPath)
	case baseName == "block.cbor":
		return KindBlock, FormatCBOR, eraFromPath(normalizedPath)
	case baseName == "tx.cbor":
		return KindTransaction, FormatCBOR, eraFromPath(normalizedPath)
	case baseName == "tx-canonical.json":
		return KindTransaction, formatFromFilename(baseName), "conway"
	case baseName == "LegacyProtocolParameters.json":
		return KindProtocolParameters, FormatJSON, "alonzo"
	case strings.HasSuffix(baseName, ".jsonld"):
		return KindGovernanceMetadata, FormatJSONLD, eraFromPath(normalizedPath)
	case isGenesisName(baseName):
		return KindGenesis, FormatJSON, eraFromPath(normalizedPath)
	case strings.Contains(normalizedPath, "protocol-parameters/"):
		return KindProtocolParameters, formatFromFilename(
				baseName,
			), eraFromPath(
				normalizedPath,
			)
	default:
		return KindUnknown, formatFromFilename(
				baseName,
			), eraFromPath(
				normalizedPath,
			)
	}
}

func formatFromFilename(name string) Format {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".cbor":
		return FormatCBOR
	case ".json":
		return FormatJSON
	case ".jsonld":
		return FormatJSONLD
	case ".hex":
		return FormatHex
	default:
		return FormatUnknown
	}
}

func eraFromPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for idx, part := range parts {
		if part == "eras" && idx+1 < len(parts) {
			return normalizeEra(parts[idx+1])
		}
	}
	for _, part := range parts {
		switch part {
		case "alonzo", "babbage", "conway", "dijkstra", "shelley":
			return part
		}
	}

	baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	for _, token := range strings.FieldsFunc(baseName, splitEraTokens) {
		switch strings.ToLower(token) {
		case "alonzo", "babbage", "conway", "dijkstra", "shelley":
			return strings.ToLower(token)
		}
	}
	return ""
}

func isGenesisName(name string) bool {
	lowerName := strings.ToLower(name)
	return strings.Contains(lowerName, "genesis")
}

func normalizeEra(era string) string {
	return strings.ToLower(era)
}

func normalizeConsensusEra(era string) string {
	lowerEra := strings.ToLower(era)
	switch {
	case strings.HasPrefix(lowerEra, "byron_"):
		return "byron"
	case lowerEra == "dijkstra":
		return "dijkstra"
	default:
		return lowerEra
	}
}

func splitEraTokens(r rune) bool {
	switch r {
	case '-', '_', '.', ' ':
		return true
	default:
		return false
	}
}
