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
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
)

// StakeLifecycle returns the certificates that register a stake credential
// with deposit, delegate it to a pool, and deregister it with a refund of the
// same deposit.
func StakeLifecycle(
	credential, poolKeyHash []byte,
	deposit uint64,
) ([]lcommon.Certificate, error) {
	registration, err := NewRegistration().
		WithCredential(credential).
		WithDeposit(deposit).
		Build()
	if err != nil {
		return nil, err
	}
	delegation, err := NewStakeDelegation().
		WithCredential(credential).
		WithPoolKeyHash(poolKeyHash).
		Build()
	if err != nil {
		return nil, err
	}
	deregistration, err := NewDeregistration().
		WithCredential(credential).
		WithDeposit(deposit).
		Build()
	if err != nil {
		return nil, err
	}
	return []lcommon.Certificate{registration, delegation, deregistration}, nil
}

// DRepLifecycle returns the certificates that register a DRep with deposit,
// update it, and deregister it with a refund of the same deposit.
func DRepLifecycle(
	credential []byte,
	deposit uint64,
) ([]lcommon.Certificate, error) {
	registration, err := NewDRepRegistration().
		WithCredential(credential).
		WithDeposit(deposit).
		Build()
	if err != nil {
		return nil, err
	}
	update, err := NewDRepUpdate().WithCredential(credential).Build()
	if err != nil {
		return nil, err
	}
	deregistration, err := NewDRepDeregistration().
		WithCredential(credential).
		WithDeposit(deposit).
		Build()
	if err != nil {
		return nil, err
	}
	return []lcommon.Certificate{registration, update, deregistration}, nil
}

// PoolLifecycle returns the certificates that register a pool and retire it
// at retirementEpoch. The operator's key hash is also the reward account.
func PoolLifecycle(
	network uint,
	operator, vrfKeyHash []byte,
	retirementEpoch uint64,
) ([]lcommon.Certificate, error) {
	registration, err := NewPoolRegistration(network).
		WithOperator(operator).
		WithVrfKeyHash(vrfKeyHash).
		WithRewardAccountKey(operator).
		Build()
	if err != nil {
		return nil, err
	}
	retirement, err := NewPoolRetirement().
		WithPoolKeyHash(operator).
		WithEpoch(retirementEpoch).
		Build()
	if err != nil {
		return nil, err
	}
	return []lcommon.Certificate{registration, retirement}, nil
}
