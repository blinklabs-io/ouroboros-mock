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

package certificates

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/blinklabs-io/gouroboros/cbor"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

// MoveInstantaneousRewardsBuilder builds a Shelley-family MIR certificate.
// Source is the wire pot: 0 for reserves and 1 for treasury.
type MoveInstantaneousRewardsBuilder struct {
	source     uint
	rewards    map[*lcommon.Credential]*big.Int
	rewardsSet bool
	otherPot   *uint64
	inputErr   error
}

// NewMoveInstantaneousRewards returns a builder for source pot 0 or 1.
// Select a reward-map or opposite-pot target before calling Build.
func NewMoveInstantaneousRewards(source uint) *MoveInstantaneousRewardsBuilder {
	return &MoveInstantaneousRewardsBuilder{source: source}
}

// WithRewards selects a reward map, including an empty map, and copies its
// credentials and signed deltas. Duplicate logical credentials are rejected.
func (b *MoveInstantaneousRewardsBuilder) WithRewards(
	rewards map[*lcommon.Credential]*big.Int,
) *MoveInstantaneousRewardsBuilder {
	b.inputErr = nil
	b.rewards = cloneMIRRewards(rewards)
	b.rewardsSet = true
	return b
}

// WithRewardKey adds a signed delta for a stake key hash.
func (b *MoveInstantaneousRewardsBuilder) WithRewardKey(
	hash []byte, delta *big.Int,
) *MoveInstantaneousRewardsBuilder {
	credential, err := keyCredential(hash)
	return b.withReward(credential, delta, err)
}

// WithRewardScript adds a signed delta for a stake script hash.
func (b *MoveInstantaneousRewardsBuilder) WithRewardScript(
	hash []byte, delta *big.Int,
) *MoveInstantaneousRewardsBuilder {
	credential, err := scriptCredential(hash)
	return b.withReward(credential, delta, err)
}

func (b *MoveInstantaneousRewardsBuilder) withReward(
	credential lcommon.Credential, delta *big.Int, err error,
) *MoveInstantaneousRewardsBuilder {
	if err != nil {
		b.inputErr = err
	}
	if b.rewards == nil {
		b.rewards = make(map[*lcommon.Credential]*big.Int)
	}
	if delta != nil {
		delta = new(big.Int).Set(delta)
	}
	b.rewards[&credential] = delta
	b.rewardsSet = true
	return b
}

// WithOtherPot selects a transfer to the opposite pot, including zero.
// Mixing this target with a reward map is an error.
func (b *MoveInstantaneousRewardsBuilder) WithOtherPot(
	amount uint64,
) *MoveInstantaneousRewardsBuilder {
	b.otherPot = &amount
	return b
}

// Build returns an independently owned certificate after wire validation.
func (b *MoveInstantaneousRewardsBuilder) Build() (
	*lcommon.MoveInstantaneousRewardsCertificate, error,
) {
	if b.inputErr != nil {
		return nil, b.inputErr
	}
	if b.otherPot != nil && b.rewardsSet {
		return nil, errors.New(
			"MIR cannot contain both reward-map and opposite-pot targets",
		)
	}
	if b.otherPot == nil && !b.rewardsSet {
		return nil, errors.New("MIR target is required")
	}
	reward := lcommon.MoveInstantaneousRewardsCertificateReward{
		Source: b.source,
	}
	if b.otherPot != nil {
		reward.OtherPot = *b.otherPot
	} else {
		reward.Rewards = cloneMIRRewards(b.rewards)
	}
	cert := &lcommon.MoveInstantaneousRewardsCertificate{
		CertType: uint(lcommon.CertificateTypeMoveInstantaneousRewards),
		Reward:   reward,
	}
	if _, err := cbor.Encode(cert); err != nil {
		return nil, fmt.Errorf("invalid MIR certificate: %w", err)
	}
	return cert, nil
}

func cloneMIRRewards(
	rewards map[*lcommon.Credential]*big.Int,
) map[*lcommon.Credential]*big.Int {
	copied := make(map[*lcommon.Credential]*big.Int, len(rewards))
	for credential, delta := range rewards {
		if credential != nil {
			value := *credential
			value.SetCbor(nil)
			credential = &value
		}
		if delta != nil {
			delta = new(big.Int).Set(delta)
		}
		copied[credential] = delta
	}
	return copied
}
