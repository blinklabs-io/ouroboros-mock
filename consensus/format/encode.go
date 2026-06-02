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

package format

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// EncodeTestVector marshals v to pretty-printed JSON. The vector is
// validated first (see validate); encoding fails if it does not satisfy
// the invariants documented on TestVector.
func EncodeTestVector(v TestVector) ([]byte, error) {
	if err := v.validate(); err != nil {
		return nil, fmt.Errorf("test vector: %w", err)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("test vector: marshal: %w", err)
	}
	// MarshalIndent does not emit a trailing newline; add one so files
	// end in a newline like every other text file in the repo.
	return append(out, '\n'), nil
}

// validate enforces the structural invariants the format documents:
// supported schema version, capture-shape match for the category,
// per-message protocol/msg-type pairs that map to known gouroboros
// IDs.
func (v TestVector) validate() error {
	if v.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf(
			"unsupported schema_version %d (this build supports %d)",
			v.SchemaVersion, CurrentSchemaVersion,
		)
	}
	switch v.Category {
	case CategoryConsensus:
		if v.Capture == nil {
			return errors.New(
				"category=consensus requires capture to be set",
			)
		}
		if err := v.Capture.validate(); err != nil {
			return fmt.Errorf("capture: %w", err)
		}
	default:
		return fmt.Errorf("unknown category %q", v.Category)
	}
	return nil
}

func (c *ConsensusCapture) validate() error {
	if c.LocalTip != nil && len(c.LocalTip.Hash) == 0 {
		return errors.New("local_tip: hash is required when local_tip is set")
	}
	for i, p := range c.Peers {
		for j, m := range p.Served {
			if err := validateServedMessage(m); err != nil {
				return fmt.Errorf(
					"peers[%d].served[%d]: %w", i, j, err,
				)
			}
		}
	}
	for i, m := range c.ExpectedOutput.DownstreamChainSync {
		// The field name is "downstream_chainsync"; reject anything
		// outside the chainsync protocol so a misplaced blockfetch
		// entry fails encoding rather than smuggling into a golden.
		if m.Protocol != ProtocolChainSync {
			return fmt.Errorf(
				"expected_output.downstream_chainsync[%d]: "+
					"protocol %q is not chainsync",
				i, m.Protocol,
			)
		}
		if err := validateServedMessage(m); err != nil {
			return fmt.Errorf(
				"expected_output.downstream_chainsync[%d]: %w", i, err,
			)
		}
	}
	if er := c.ExpectedOutput.ExpectedRollback; er != nil {
		if len(er.Point.Hash) == 0 {
			return errors.New(
				"expected_rollback.point: hash is required",
			)
		}
		if len(er.Tip.Hash) == 0 {
			return errors.New("expected_rollback.tip: hash is required")
		}
		ft := c.ExpectedOutput.FinalTip
		if er.Tip.Slot != ft.Slot ||
			!bytes.Equal(er.Tip.Hash, ft.Hash) ||
			er.Tip.BlockNumber != ft.BlockNumber {
			return errors.New(
				"expected_rollback.tip must equal final_tip",
			)
		}
	}
	return nil
}
