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

// Package benchmark loads reproducible experiment data and runs observable peers.
package benchmark

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/ouroboros-mock/peer"
)

// SchemaVersion is the supported experiment and measurement contract version.
const SchemaVersion = 1

// Network fixes the semantic time and protocol context of a corpus.
type Network struct {
	Magic       uint32    `json:"magic"`
	Era         string    `json:"era"`
	SystemStart time.Time `json:"system_start"`
	SlotSeconds float64   `json:"slot_seconds"`
	Genesis     []string  `json:"genesis"`
	CostModels  []string  `json:"cost_models"`
	Parameters  string    `json:"parameters"`
}

// Reset describes how consumers restore equivalent state before a repetition.
type Reset struct {
	Strategy string   `json:"strategy"`
	State    []string `json:"state"`
	Cache    string   `json:"cache"`
}

// Artifact identifies exact bytes, with an explicit wire type for CBOR payloads.
type Artifact struct {
	ID        string   `json:"id"`
	Path      string   `json:"path"`
	SHA256    string   `json:"sha256"`
	Bytes     int64    `json:"bytes"`
	Kind      string   `json:"kind"`
	Type      *uint    `json:"type,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
}

// Oracle identifies independent reference evidence for exact artifact inputs.
type Oracle struct {
	ID      string   `json:"id"`
	Tool    string   `json:"tool"`
	Version string   `json:"version"`
	Source  string   `json:"source"`
	Inputs  []string `json:"inputs"`
	Result  string   `json:"result"`
}

// Expectation links a case's inputs to independent evidence and expected data.
type Expectation struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Inputs   []string `json:"inputs"`
	Oracle   string   `json:"oracle"`
	Expected string   `json:"expected"`
}

// Requirement is a minimum observed event/item population for a peer role.
type Requirement struct {
	Event  string `json:"event"`
	Events uint64 `json:"events"`
	Items  uint64 `json:"items"`
}

// Role selects a reactive peer; addresses are supplied by the process controller.
type Role struct {
	Target    string        `json:"target"`
	Transport string        `json:"transport"`
	Required  []Requirement `json:"required"`
	ID        string        `json:"id"`
	Kind      string        `json:"kind"`
	Initial   []string      `json:"initial"`
}

// Action schedules an operation relative to an explicit experiment start.
type Action struct {
	ID    string   `json:"id"`
	Role  string   `json:"role"`
	Kind  string   `json:"kind"`
	AtNS  int64    `json:"at_ns"`
	Items []string `json:"items"`
	After []string `json:"after"`
}

// Diagnostics selects externally configured node measurements for this experiment.
type Diagnostics struct {
	Snapshots bool `json:"snapshots"`
	Ledger    bool `json:"ledger"`
}

// Manifest describes immutable inputs, their validity class and replay contract.
type Manifest struct {
	Version      int           `json:"version"`
	Diagnostics  *Diagnostics  `json:"diagnostics"`
	ID           string        `json:"id"`
	Class        string        `json:"class"`
	Network      Network       `json:"network"`
	Reset        Reset         `json:"reset"`
	Artifacts    []Artifact    `json:"artifacts"`
	Roles        []Role        `json:"roles"`
	Actions      []Action      `json:"actions"`
	Oracles      []Oracle      `json:"oracles"`
	Expectations []Expectation `json:"expectations"`
	Credentials  []string      `json:"credentials"`
	Coverage     string        `json:"coverage"`
	DurationNS   int64         `json:"duration_ns"`
}

// Bundle holds a validated manifest rooted at its artifact directory.
type Bundle struct {
	// Manifest data remains private so validated schedules cannot change during a run.
	digest    string
	manifest  Manifest
	root      string
	artifacts map[string]Artifact
}

func decodeJSON(data []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON input")
	}
	return nil
}

// Load reads a manifest, verifies every artifact and validates its references.
func Load(path string) (*Bundle, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(f, 4*1024*1024+1))
	if err = errors.Join(readErr, f.Close()); err != nil {
		return nil, err
	}
	if len(data) > 4*1024*1024 {
		return nil, errors.New("manifest exceeds 4 MiB")
	}
	var m Manifest
	if err = decodeJSON(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	sum := sha256.Sum256(data)
	b := &Bundle{digest: hex.EncodeToString(sum[:]), manifest: m, root: filepath.Dir(path), artifacts: make(map[string]Artifact, len(m.Artifacts))}
	if err = b.validate(); err != nil {
		return nil, fmt.Errorf("experiment %q: %w", m.ID, err)
	}
	return b, nil
}

// Read returns verified artifact bytes within the declared corpus root.
func (b *Bundle) Read(id string) ([]byte, error) {
	a, ok := b.artifacts[id]
	if !ok {
		return nil, fmt.Errorf("unknown artifact %q", id)
	}
	if a.Bytes > 128*1024*1024 {
		return nil, fmt.Errorf("artifact %q exceeds the payload read limit", id)
	}
	r, err := os.OpenRoot(b.root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	f, err := r.Open(a.Path)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(f, a.Bytes+1))
	if err = errors.Join(readErr, f.Close()); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	if int64(len(data)) != a.Bytes || hex.EncodeToString(sum[:]) != a.SHA256 {
		return nil, fmt.Errorf("artifact %q changed after validation", id)
	}
	return data, nil
}

// Blocks decodes exact block bytes and verifies their hash linkage.
func (b *Bundle) Blocks(ids []string) ([]ledger.Block, error) {
	blocks := make([]ledger.Block, 0, len(ids))
	for _, id := range ids {
		a, ok := b.artifacts[id]
		if !ok || a.Kind != "block" || a.Type == nil {
			return nil, fmt.Errorf("not a typed block: %q", id)
		}
		raw, err := b.Read(id)
		if err != nil {
			return nil, err
		}
		block, err := ledger.NewBlockFromCbor(*a.Type, raw)
		if err != nil {
			return nil, fmt.Errorf("block %q: %w", id, err)
		}
		blocks = append(blocks, block)
	}
	if _, err := peer.NewChain(blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// Transactions decodes transaction identities without changing their wire bytes.
func (b *Bundle) Transactions(ids []string) ([]peer.Tx, error) {
	txs := make([]peer.Tx, 0, len(ids))
	for _, id := range ids {
		a, ok := b.artifacts[id]
		if !ok || a.Kind != "transaction" || a.Type == nil || *a.Type > 65535 {
			return nil, fmt.Errorf("not a typed transaction: %q", id)
		}
		raw, err := b.Read(id)
		if err != nil {
			return nil, err
		}
		tx, err := ledger.NewTransactionFromCbor(*a.Type, raw)
		if err != nil {
			return nil, fmt.Errorf("transaction %q: %w", id, err)
		}
		txs = append(txs, peer.Tx{EraId: uint16(*a.Type), ID: tx.Hash(), Raw: raw})
	}
	return txs, nil
}

func (b *Bundle) verifyArtifact(a Artifact) error {
	if a.ID == "" || !fs.ValidPath(a.Path) || strings.Contains(a.Path, "\\") || a.Bytes <= 0 || a.Bytes == int64(^uint64(0)>>1) {
		return fmt.Errorf("invalid artifact %q", a.ID)
	}
	digest, err := hex.DecodeString(a.SHA256)
	if err != nil || len(digest) != sha256.Size || strings.ToLower(a.SHA256) != a.SHA256 {
		return fmt.Errorf("invalid digest for %q", a.ID)
	}
	r, err := os.OpenRoot(b.root)
	if err != nil {
		return err
	}
	defer r.Close()
	f, err := r.Open(a.Path)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != a.Bytes {
		_ = f.Close()
		return fmt.Errorf("wrong artifact kind/size: %q", a.ID)
	}
	hash := sha256.New()
	n, readErr := io.Copy(hash, io.LimitReader(f, a.Bytes+1))
	if err = errors.Join(readErr, f.Close()); err != nil {
		return err
	}
	if n != a.Bytes || hex.EncodeToString(hash.Sum(nil)) != a.SHA256 {
		return fmt.Errorf("artifact checksum mismatch: %q", a.ID)
	}
	return nil
}

// Roles returns a copy of the declared role identities.
func (b *Bundle) Roles() []Role {
	out := make([]Role, len(b.manifest.Roles))
	for i, r := range b.manifest.Roles {
		r.Initial = slices.Clone(r.Initial)
		r.Required = slices.Clone(r.Required)
		out[i] = r
	}
	return out
}
