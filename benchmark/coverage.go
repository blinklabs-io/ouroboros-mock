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

package benchmark

import "fmt"

// Measurement defines one source boundary and its independently checked cases.
type Measurement struct {
	Name          string   `json:"name"`
	Source        string   `json:"source"`
	Unit          string   `json:"unit"`
	Boundary      string   `json:"boundary"`
	NativeMeaning string   `json:"native_meaning"`
	Cases         []string `json:"cases"`
	// MetricCases links calculation goldens without asserting observable presence.
	MetricCases []string `json:"metric_cases,omitempty"`
}

// Coverage enumerates the complete common-observable and derived-family contract.
type Coverage struct {
	Version     int           `json:"version"`
	Schema      string        `json:"schema"`
	Envelope    string        `json:"envelope"`
	Examples    string        `json:"examples"`
	Observables []Measurement `json:"observables"`
	Derived     []Measurement `json:"derived"`
}

var observableNames = [...]string{
	"BlockFetch.Client.CompletedBlockFetch",
	"BlockFetch.Client.SendFetchRequest",
	"BlockFetch.Server.SendBlock",
	"ChainDB.AddBlockEvent.AddedToCurrentChain",
	"ChainDB.AddBlockEvent.SwitchedToAFork",
	"ChainDB.LedgerEvent.Snapshot.TookSnapshot",
	"ChainSync.Client.DownloadedHeader",
	"ChainSync.ServerHeader.Update",
	"Forge.Loop.AdoptedBlock",
	"Forge.Loop.BlockContext",
	"Forge.Loop.ForgeTickedLedgerState",
	"Forge.Loop.ForgedBlock",
	"Forge.Loop.ForgingMempoolSnapshot",
	"Forge.Loop.LedgerState",
	"Forge.Loop.LedgerView",
	"Forge.Loop.NodeIsLeader",
	"Forge.Loop.NodeNotLeader",
	"Forge.Loop.StartLeadershipCheck",
	"LedgerMetrics",
	"Mempool.AddedTx",
	"Mempool.RejectedTx",
	"Mempool.RemoveTxs",
	"Resources",
	"TxSubmission.TxInbound.Collected",
	"TxSubmission.TxInbound.Processed",
}

var derivedNames = [...]string{
	"Forge-loop tardiness",
	"Block-context acquisition",
	"Ledger-state acquisition",
	"Ledger-view acquisition",
	"Leadership-check duration",
	"Ledger ticking",
	"Mempool snapshotting",
	"Leadership/mempool to signed block",
	"Forged → announced / sending / self-adopted",
	"First peer notice / first fetch",
	"Notice → request",
	"Fetch duration",
	"Fetched → announced / sending / adopted",
	"End-to-end propagation to fraction p",
	"CPU and optional resource rates",
	"Snapshot count",
	"Snapshot duration",
	"Live after major GC inside/outside snapshot windows (R)",
	"Processed / accepted / rejected totals",
	"Reject ratio",
	"CPU per processed transaction",
	"Mempool occupancy / rejection count",
	"Optional ledger size/density",
}

// ObservableNames returns the complete observable catalogue in stable order.
func ObservableNames() []string { return append([]string(nil), observableNames[:]...) }

// DerivedNames returns the complete derived-family catalogue in stable order.
func DerivedNames() []string { return append([]string(nil), derivedNames[:]...) }

func (c Coverage) validate(cases map[string]caseCoverage) error {
	if c.Version != SchemaVersion {
		return fmt.Errorf("unsupported coverage version %d", c.Version)
	}
	for _, group := range []struct {
		got         []Measurement
		want        []string
		observables bool
	}{{c.Observables, observableNames[:], true}, {c.Derived, derivedNames[:], false}} {
		if len(group.got) != len(group.want) {
			return fmt.Errorf("incomplete measurement catalogue: got %d, need %d", len(group.got), len(group.want))
		}
		required := map[string]bool{}
		for _, n := range group.want {
			required[n] = true
		}
		for _, m := range group.got {
			if !required[m.Name] || !name(m.Source) || !name(m.Unit) || !name(m.Boundary) || !name(m.NativeMeaning) || len(m.Cases) == 0 {
				return fmt.Errorf("invalid measurement definition %q", m.Name)
			}
			delete(required, m.Name)
			if group.observables {
				if len(m.MetricCases) == 0 {
					return fmt.Errorf("observable %s lacks calculation cases", m.Name)
				}
				seenMetrics := map[string]bool{}
				for _, id := range m.MetricCases {
					info, ok := cases[id]
					if !ok || seenMetrics[id] || !info.metricFor(m.Name) {
						return fmt.Errorf("case %s does not calculate measurement %s", id, m.Name)
					}
					seenMetrics[id] = true
				}
			} else if len(m.MetricCases) > 0 {
				return fmt.Errorf("derived measurement %s has unexpected observable associations", m.Name)
			}
			seen := map[string]bool{}
			covered := false
			for _, id := range m.Cases {
				c, present := cases[id]
				if !present || seen[id] {
					return fmt.Errorf("unknown/repeated measurement case %q", id)
				}
				if !c.supports(m.Name) && !c.quality {
					return fmt.Errorf("case %s does not cover measurement %s", id, m.Name)
				}
				seen[id] = true
				covered = covered || c.supports(m.Name)
			}
			if !covered {
				return fmt.Errorf("measurement %s has no matching positive golden", m.Name)
			}
		}
	}
	return nil
}
