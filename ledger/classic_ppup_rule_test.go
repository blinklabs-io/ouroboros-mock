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

package ledger_test

import (
	"crypto/ed25519"
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/allegra"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
	"github.com/stretchr/testify/require"
)

type classicPPUPEra struct {
	name        string
	descriptors func() []lcommon.UtxoValidationRuleDescriptor
	pparams     lcommon.ProtocolParameters
	// newTx builds an era transaction proposing an empty update for
	// genesisKey in epoch, witnessed by the given key pairs.
	newTx func(
		genesisKey lcommon.Blake2b224,
		epoch uint64,
		signers ...ed25519.PrivateKey,
	) lcommon.Transaction
}

func signBody(
	hash lcommon.Blake2b256,
	signers []ed25519.PrivateKey,
) []lcommon.VkeyWitness {
	ret := make([]lcommon.VkeyWitness, 0, len(signers))
	for _, signer := range signers {
		ret = append(ret, lcommon.VkeyWitness{
			Vkey:      signer.Public().(ed25519.PublicKey),
			Signature: ed25519.Sign(signer, hash[:]),
		})
	}
	return ret
}

var classicPPUPEras = []classicPPUPEra{
	{
		name:        "Shelley",
		descriptors: shelley.UtxoValidationRuleDescriptors,
		pparams:     &shelley.ShelleyProtocolParameters{ProtocolMajor: 2},
		newTx: func(g lcommon.Blake2b224, epoch uint64, s ...ed25519.PrivateKey) lcommon.Transaction {
			tx := &shelley.ShelleyTransaction{
				Body: shelley.ShelleyTransactionBody{
					Update: &shelley.ShelleyTransactionPparamUpdate{
						ProtocolParamUpdates: map[lcommon.Blake2b224]shelley.ShelleyProtocolParameterUpdate{
							g: {},
						},
						Epoch: epoch,
					},
				},
			}
			tx.WitnessSet.VkeyWitnesses = signBody(tx.Hash(), s)
			return tx
		},
	},
	{
		name:        "Allegra",
		descriptors: allegra.UtxoValidationRuleDescriptors,
		pparams:     &allegra.AllegraProtocolParameters{ProtocolMajor: 3},
		newTx: func(g lcommon.Blake2b224, epoch uint64, s ...ed25519.PrivateKey) lcommon.Transaction {
			tx := &allegra.AllegraTransaction{
				Body: allegra.AllegraTransactionBody{
					Update: &allegra.AllegraTransactionPparamUpdate{
						ProtocolParamUpdates: map[lcommon.Blake2b224]allegra.AllegraProtocolParameterUpdate{
							g: {},
						},
						Epoch: epoch,
					},
				},
			}
			tx.WitnessSet.VkeyWitnesses = signBody(tx.Hash(), s)
			return tx
		},
	},
	{
		name:        "Mary",
		descriptors: mary.UtxoValidationRuleDescriptors,
		pparams:     &mary.MaryProtocolParameters{ProtocolMajor: 4},
		newTx: func(g lcommon.Blake2b224, epoch uint64, s ...ed25519.PrivateKey) lcommon.Transaction {
			tx := &mary.MaryTransaction{Body: mary.MaryTransactionBody{
				Update: &mary.MaryTransactionPparamUpdate{
					ProtocolParamUpdates: map[lcommon.Blake2b224]mary.MaryProtocolParameterUpdate{
						g: {},
					},
					Epoch: epoch,
				},
			}}
			tx.WitnessSet.VkeyWitnesses = signBody(tx.Hash(), s)
			return tx
		},
	},
	{
		name:        "Alonzo",
		descriptors: alonzo.UtxoValidationRuleDescriptors,
		pparams:     &alonzo.AlonzoProtocolParameters{ProtocolMajor: 5},
		newTx: func(g lcommon.Blake2b224, epoch uint64, s ...ed25519.PrivateKey) lcommon.Transaction {
			tx := &alonzo.AlonzoTransaction{
				TxIsValid: true,
				Body: alonzo.AlonzoTransactionBody{
					Update: &alonzo.AlonzoTransactionPparamUpdate{
						ProtocolParamUpdates: map[lcommon.Blake2b224]alonzo.AlonzoProtocolParameterUpdate{
							g: {},
						},
						Epoch: epoch,
					},
				},
			}
			tx.WitnessSet.VkeyWitnesses = signBody(tx.Hash(), s)
			return tx
		},
	},
	{
		name:        "Babbage",
		descriptors: babbage.UtxoValidationRuleDescriptors,
		pparams:     &babbage.BabbageProtocolParameters{ProtocolMajor: 7},
		newTx: func(g lcommon.Blake2b224, epoch uint64, s ...ed25519.PrivateKey) lcommon.Transaction {
			tx := &babbage.BabbageTransaction{
				TxIsValid: true,
				Body: babbage.BabbageTransactionBody{
					Update: &babbage.BabbageTransactionPparamUpdate{
						ProtocolParamUpdates: map[lcommon.Blake2b224]babbage.BabbageProtocolParameterUpdate{
							g: {},
						},
						Epoch: epoch,
					},
				},
			}
			tx.WitnessSet.VkeyWitnesses = signBody(tx.Hash(), s)
			return tx
		},
	},
}

// classicPPUPRules returns the era's registered witness, signature and PPUP
// rules in registration order.
func classicPPUPRules(
	t *testing.T,
	era classicPPUPEra,
) []lcommon.UtxoValidationRuleFunc {
	t.Helper()
	want := map[lcommon.UtxoValidationRuleId]bool{
		lcommon.UtxoValidationRuleRequiredVKeyWitnesses:    true,
		lcommon.UtxoValidationRuleSignatures:               true,
		lcommon.UtxoValidationRuleProtocolParameterUpdates: true,
	}
	var rules []lcommon.UtxoValidationRuleFunc
	for _, descriptor := range era.descriptors() {
		if want[descriptor.Id] {
			rules = append(rules, descriptor.Validator)
			delete(want, descriptor.Id)
		}
	}
	require.Empty(t, want, "%s does not register every rule", era.name)
	return rules
}

// TestClassicPPUPRuleUsesMockWindow drives gouroboros'
// ValidateClassicProtocolParameterUpdates through VerifyTransaction with each
// Shelley-family era's registered rules and the mock's genesis delegation and
// voting window. The reference (Shelley PPUP, ppupTransitionNonEmpty) accepts
// only the current epoch before the slot of no return and only the next epoch
// from it on.
func TestClassicPPUPRuleUsesMockWindow(t *testing.T) {
	const (
		epoch       = uint64(3)
		epochLength = uint64(432_000)
		start       = epoch * epochLength
		nextStart   = start + epochLength
		// 2 * ceiling(3 * 2160 / 0.05)
		noReturn = nextStart - 259_200
	)
	_, delegateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	_, otherKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	delegate := lcommon.Blake2b224Hash(delegateKey.Public().(ed25519.PublicKey))
	genesisKey := lcommon.Blake2b224Hash([]byte("genesis key"))
	window := ledger.FixedEpochProtocolParameterUpdateWindow(
		epochLength,
		2160,
		big.NewRat(1, 20),
	)
	state := ledger.NewLedgerStateBuilder().
		WithGenesisDelegation(
			map[lcommon.Blake2b224]lcommon.Blake2b224{genesisKey: delegate},
			1,
		).
		WithProtocolParameterUpdateWindow(window).
		Build()

	slots := []struct {
		name         string
		slot         uint64
		currentEpoch uint64
		forNext      bool
	}{
		{"first slot", start, epoch, false},
		{"last slot before no return", noReturn - 1, epoch, false},
		{"slot of no return", noReturn, epoch, true},
		{"last slot of epoch", nextStart - 1, epoch, true},
		{"first slot of next epoch", nextStart, epoch + 1, false},
	}
	for _, era := range classicPPUPEras {
		rules := classicPPUPRules(t, era)
		verify := func(
			st lcommon.LedgerState,
			tx lcommon.Transaction,
			slot uint64,
		) error {
			return lcommon.VerifyTransaction(tx, slot, st, era.pparams, rules)
		}
		for _, sc := range slots {
			t.Run(era.name+"/"+sc.name, func(t *testing.T) {
				expected := sc.currentEpoch
				wrong := expected + 1
				if sc.forNext {
					expected++
					wrong = sc.currentEpoch
				}
				require.NoError(t, verify(
					state,
					era.newTx(genesisKey, expected, delegateKey),
					sc.slot,
				))
				var epochErr lcommon.ProtocolParameterUpdateEpochError
				require.ErrorAs(t, verify(
					state,
					era.newTx(genesisKey, wrong, delegateKey),
					sc.slot,
				), &epochErr)
				require.Equal(t, sc.currentEpoch, epochErr.Current)
				require.Equal(t, expected, epochErr.Expected)
				require.Equal(t, wrong, epochErr.Proposed)
				require.Equal(t, sc.forNext, epochErr.ForNextEpoch)
			})
		}
		t.Run(era.name+"/unknown genesis key", func(t *testing.T) {
			unknown := lcommon.Blake2b224Hash([]byte("unknown genesis key"))
			var delegateErr lcommon.ProtocolParameterUpdateDelegateError
			require.ErrorAs(t, verify(
				state,
				era.newTx(unknown, epoch, delegateKey),
				start,
			), &delegateErr)
			require.Equal(t, unknown, delegateErr.Delegate)
		})
		t.Run(era.name+"/delegate did not sign", func(t *testing.T) {
			var witnessErr lcommon.ProtocolParameterUpdateWitnessError
			require.ErrorAs(t, verify(
				state,
				era.newTx(genesisKey, epoch, otherKey),
				start,
			), &witnessErr)
		})
		t.Run(era.name+"/no genesis delegation configured", func(t *testing.T) {
			bare := ledger.NewLedgerStateBuilder().
				WithProtocolParameterUpdateWindow(window).
				Build()
			require.ErrorAs(t, verify(
				bare,
				era.newTx(genesisKey, epoch, delegateKey),
				start,
			), &lcommon.GenesisDelegationStateUnavailableError{})
		})
		t.Run(era.name+"/no window configured", func(t *testing.T) {
			noWindow := ledger.NewLedgerStateBuilder().
				WithGenesisDelegation(
					map[lcommon.Blake2b224]lcommon.Blake2b224{
						genesisKey: delegate,
					},
					1,
				).
				Build()
			require.ErrorAs(t, verify(
				noWindow,
				era.newTx(genesisKey, epoch, delegateKey),
				start,
			), &lcommon.ClassicProtocolParameterUpdateWindowStateUnavailableError{})
		})
	}
}

func TestLedgerState_GenesisDelegation(t *testing.T) {
	unconfigured := ledger.NewLedgerStateBuilder().Build()
	_, err := unconfigured.GenesisDelegateKeyHashes(0)
	require.ErrorAs(t, err, &lcommon.GenesisDelegationStateUnavailableError{})
	_, _, err = unconfigured.GenesisDelegateForGenesisKey(
		lcommon.Blake2b224{},
		0,
	)
	require.ErrorAs(t, err, &lcommon.GenesisDelegationStateUnavailableError{})
	_, err = unconfigured.GenesisUpdateQuorum()
	require.ErrorAs(t, err, &lcommon.GenesisDelegationStateUnavailableError{})

	genesisA := lcommon.Blake2b224{0xa}
	genesisB := lcommon.Blake2b224{0xb}
	delegate1 := lcommon.Blake2b224{0x1}
	delegate2 := lcommon.Blake2b224{0x2}
	delegates := map[lcommon.Blake2b224]lcommon.Blake2b224{
		genesisA: delegate2,
		genesisB: delegate1,
	}
	state := ledger.NewLedgerStateBuilder().
		WithGenesisDelegation(delegates, 2).
		Build()
	delete(delegates, genesisA)

	hashes, err := state.GenesisDelegateKeyHashes(0)
	require.NoError(t, err)
	require.Equal(t, []lcommon.Blake2b224{delegate1, delegate2}, hashes)
	got, ok, err := state.GenesisDelegateForGenesisKey(genesisA, 0)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, delegate2, got)
	_, ok, err = state.GenesisDelegateForGenesisKey(lcommon.Blake2b224{0xc}, 0)
	require.NoError(t, err)
	require.False(t, ok)
	quorum, err := state.GenesisUpdateQuorum()
	require.NoError(t, err)
	require.Equal(t, uint(2), quorum)

	empty := ledger.NewLedgerStateBuilder().
		WithGenesisDelegation(nil, 1).
		Build()
	hashes, err = empty.GenesisDelegateKeyHashes(0)
	require.NoError(t, err)
	require.Empty(t, hashes)
}
