// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package certificates_test

import (
	"bytes"
	"math/big"
	"net"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/certificates"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
	"github.com/stretchr/testify/require"
)

var (
	stakeHash = bytes.Repeat([]byte{0x01}, lcommon.Blake2b224Size)
	poolHash  = bytes.Repeat([]byte{0x02}, lcommon.Blake2b224Size)
	vrfHash   = bytes.Repeat([]byte{0x03}, lcommon.Blake2b256Size)
)

func TestStakeBuildersReturnRoundTrippableCertificates(t *testing.T) {
	registration, err := certificates.NewStakeRegistration().
		WithCredential(stakeHash).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(
		t,
		registration,
		&lcommon.StakeRegistrationCertificate{
			CertType:        0,
			StakeCredential: keyCredential(stakeHash),
		},
	)

	deregistration, err := certificates.NewStakeDeregistration().
		WithScriptCredential(stakeHash).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(
		t,
		deregistration,
		&lcommon.StakeDeregistrationCertificate{
			CertType:        1,
			StakeCredential: scriptCredential(stakeHash),
		},
	)

	delegation, err := certificates.NewStakeDelegation().
		WithCredential(stakeHash).
		WithPoolKeyHash(poolHash).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(
		t,
		delegation,
		&lcommon.StakeDelegationCertificate{
			CertType:        2,
			StakeCredential: new(keyCredential(stakeHash)),
			PoolKeyHash:     lcommon.NewBlake2b224(poolHash),
		},
	)

	registrationWithDeposit, err := certificates.NewRegistration().
		WithScriptCredential(stakeHash).
		WithDeposit(123).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(
		t,
		registrationWithDeposit,
		&lcommon.RegistrationCertificate{
			CertType:        7,
			StakeCredential: scriptCredential(stakeHash),
			Amount:          123,
		},
	)

	deregistrationWithRefund, err := certificates.NewDeregistration().
		WithCredential(stakeHash).
		WithDeposit(123).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(
		t,
		deregistrationWithRefund,
		&lcommon.DeregistrationCertificate{
			CertType:        8,
			StakeCredential: keyCredential(stakeHash),
			Amount:          123,
		},
	)
}

func TestPoolBuildersReturnRoundTrippableCertificates(t *testing.T) {
	hostname := "relay.example.test"
	port := uint32(3001)
	relay := lcommon.PoolRelay{
		Type:     lcommon.PoolRelayTypeSingleHostName,
		Hostname: &hostname,
		Port:     &port,
	}
	registration, err := certificates.NewPoolRegistration(lcommon.AddressNetworkTestnet).
		WithOperator(poolHash).
		WithVrfKeyHash(vrfHash).
		WithRewardAccountKey(stakeHash).
		WithOwners(stakeHash).
		WithMargin(1, 100).
		WithPledge(1000).
		WithCost(500).
		WithRelays(relay).
		WithMetadata("https://example.test/pool", vrfHash).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	expected := &lcommon.PoolRegistrationCertificate{
		CertType:      3,
		Operator:      lcommon.NewBlake2b224(poolHash),
		VrfKeyHash:    lcommon.NewBlake2b256(vrfHash),
		RewardAccount: lcommon.NewBlake2b224(stakeHash),
		PoolOwners:    []lcommon.AddrKeyHash{lcommon.NewBlake2b224(stakeHash)},
		Margin:        cbor.Rat{Rat: big.NewRat(1, 100)},
		Pledge:        1000,
		Cost:          500,
		Relays:        []lcommon.PoolRelay{relay},
		PoolMetadata: &lcommon.PoolMetadata{
			Url:  "https://example.test/pool",
			Hash: lcommon.PoolMetadataHash(vrfHash),
		},
	}
	assertCertificateRoundTrip(t, registration, expected)
	assertPoolRewardAccount(
		t,
		registration,
		uint(lcommon.AddressNetworkTestnet),
		keyCredential(stakeHash),
	)

	retirement, err := certificates.NewPoolRetirement().
		WithPoolKeyHash(poolHash).
		WithEpoch(42).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(
		t,
		retirement,
		&lcommon.PoolRetirementCertificate{
			CertType:    4,
			PoolKeyHash: lcommon.NewBlake2b224(poolHash),
			Epoch:       42,
		},
	)
}

func TestPoolBuilderOwnsMutableInputsAndResults(t *testing.T) {
	port := uint32(3001)
	ipv4 := net.IP{192, 0, 2, 1}
	builder := certificates.NewPoolRegistration(lcommon.AddressNetworkTestnet).
		WithOperator(poolHash).
		WithVrfKeyHash(vrfHash).
		WithRewardAccountKey(stakeHash).
		WithOwners(stakeHash).
		WithMargin(1, 100).
		WithRelays(lcommon.PoolRelay{
			Type: lcommon.PoolRelayTypeSingleHostAddress,
			Port: &port,
			Ipv4: &ipv4,
		}).
		WithMetadata("https://example.test/pool", vrfHash)
	port = 4001
	ipv4[0] = 0xff

	first, err := builder.Build()
	require.NoError(t, err)
	second, err := builder.Build()
	require.NoError(t, err)
	require.Equal(t, uint32(3001), *first.Relays[0].Port)
	require.Equal(t, byte(192), (*first.Relays[0].Ipv4)[0])

	first.Margin.Rat.SetInt64(1)
	first.PoolOwners[0] = lcommon.AddrKeyHash{}
	*first.Relays[0].Port = 5001
	(*first.Relays[0].Ipv4)[0] = 0xfe
	first.PoolMetadata.Hash[0] = 0xfd
	require.Equal(t, big.NewRat(1, 100), second.Margin.Rat)
	require.Equal(t, lcommon.NewBlake2b224(stakeHash), second.PoolOwners[0])
	require.Equal(t, uint32(3001), *second.Relays[0].Port)
	require.Equal(t, byte(192), (*second.Relays[0].Ipv4)[0])
	require.Equal(t, byte(0x03), second.PoolMetadata.Hash[0])

	third, err := builder.Build()
	require.NoError(t, err)
	require.Equal(t, big.NewRat(1, 100), third.Margin.Rat)
	require.Equal(t, uint32(3001), *third.Relays[0].Port)
}

func TestGovernanceBuilderResultsOwnMutableState(t *testing.T) {
	tests := []struct {
		name   string
		build  func() (lcommon.Certificate, error)
		mutate func(lcommon.Certificate)
		check  func(*testing.T, lcommon.Certificate)
	}{
		{
			name: "DRep registration anchor",
			build: func() func() (lcommon.Certificate, error) {
				builder := certificates.NewDRepRegistration().
					WithCredential(stakeHash).
					WithAnchor("https://example.test/drep", vrfHash)
				return func() (lcommon.Certificate, error) { return builder.Build() }
			}(),
			mutate: func(cert lcommon.Certificate) {
				cert.(*certificates.DRepRegistrationCertificate).Anchor.DataHash[0] = 0xff
			},
			check: func(t *testing.T, cert lcommon.Certificate) {
				require.Equal(t, byte(0x03), cert.(*certificates.DRepRegistrationCertificate).Anchor.DataHash[0])
			},
		},
		{
			name: "DRep update anchor",
			build: func() func() (lcommon.Certificate, error) {
				builder := certificates.NewDRepUpdate().
					WithCredential(stakeHash).
					WithAnchor("https://example.test/drep", vrfHash)
				return func() (lcommon.Certificate, error) { return builder.Build() }
			}(),
			mutate: func(cert lcommon.Certificate) {
				cert.(*lcommon.UpdateDrepCertificate).Anchor.DataHash[0] = 0xff
			},
			check: func(t *testing.T, cert lcommon.Certificate) {
				require.Equal(t, byte(0x03), cert.(*lcommon.UpdateDrepCertificate).Anchor.DataHash[0])
			},
		},
		{
			name: "committee resignation anchor",
			build: func() func() (lcommon.Certificate, error) {
				builder := certificates.NewResignCommitteeCold().
					WithColdCredential(stakeHash).
					WithAnchor("https://example.test/committee", vrfHash)
				return func() (lcommon.Certificate, error) { return builder.Build() }
			}(),
			mutate: func(cert lcommon.Certificate) {
				cert.(*lcommon.ResignCommitteeColdCertificate).Anchor.DataHash[0] = 0xff
			},
			check: func(t *testing.T, cert lcommon.Certificate) {
				require.Equal(t, byte(0x03), cert.(*lcommon.ResignCommitteeColdCertificate).Anchor.DataHash[0])
			},
		},
		{
			name: "vote delegation DRep",
			build: func() func() (lcommon.Certificate, error) {
				builder := certificates.NewVoteDelegation().
					WithCredential(stakeHash).
					WithDRepKeyHash(poolHash)
				return func() (lcommon.Certificate, error) { return builder.Build() }
			}(),
			mutate: func(cert lcommon.Certificate) {
				cert.(*lcommon.VoteDelegationCertificate).Drep.Credential[0] = 0xff
			},
			check: func(t *testing.T, cert lcommon.Certificate) {
				require.Equal(t, byte(0x02), cert.(*lcommon.VoteDelegationCertificate).Drep.Credential[0])
			},
		},
		{
			name: "stake vote delegation DRep",
			build: func() func() (lcommon.Certificate, error) {
				builder := certificates.NewStakeVoteDelegation().
					WithCredential(stakeHash).
					WithPoolKeyHash(poolHash).
					WithDRepKeyHash(vrfHash[:28])
				return func() (lcommon.Certificate, error) { return builder.Build() }
			}(),
			mutate: func(cert lcommon.Certificate) {
				cert.(*lcommon.StakeVoteDelegationCertificate).Drep.Credential[0] = 0xff
			},
			check: func(t *testing.T, cert lcommon.Certificate) {
				require.Equal(t, byte(0x03), cert.(*lcommon.StakeVoteDelegationCertificate).Drep.Credential[0])
			},
		},
		{
			name: "vote registration delegation DRep",
			build: func() func() (lcommon.Certificate, error) {
				builder := certificates.NewVoteRegistrationDelegation().
					WithCredential(stakeHash).
					WithDRepKeyHash(poolHash)
				return func() (lcommon.Certificate, error) { return builder.Build() }
			}(),
			mutate: func(cert lcommon.Certificate) {
				cert.(*lcommon.VoteRegistrationDelegationCertificate).Drep.Credential[0] = 0xff
			},
			check: func(t *testing.T, cert lcommon.Certificate) {
				require.Equal(t, byte(0x02), cert.(*lcommon.VoteRegistrationDelegationCertificate).Drep.Credential[0])
			},
		},
		{
			name: "stake vote registration delegation DRep",
			build: func() func() (lcommon.Certificate, error) {
				builder := certificates.NewStakeVoteRegistrationDelegation().
					WithCredential(stakeHash).
					WithPoolKeyHash(poolHash).
					WithDRepKeyHash(vrfHash[:28])
				return func() (lcommon.Certificate, error) { return builder.Build() }
			}(),
			mutate: func(cert lcommon.Certificate) {
				cert.(*lcommon.StakeVoteRegistrationDelegationCertificate).Drep.Credential[0] = 0xff
			},
			check: func(t *testing.T, cert lcommon.Certificate) {
				require.Equal(t, byte(0x03), cert.(*lcommon.StakeVoteRegistrationDelegationCertificate).Drep.Credential[0])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, err := tt.build()
			require.NoError(t, err)
			second, err := tt.build()
			require.NoError(t, err)
			tt.mutate(first)
			tt.check(t, second)
			third, err := tt.build()
			require.NoError(t, err)
			tt.check(t, third)
		})
	}
}

func TestGovernanceBuildersReturnCertificatesUsableInTransactions(
	t *testing.T,
) {
	drepRegistration, err := certificates.NewDRepRegistration().
		WithCredential(stakeHash).
		WithDeposit(100).
		WithAnchor("https://example.test/drep", vrfHash).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	voteDelegation, err := certificates.NewVoteDelegation().
		WithCredential(stakeHash).
		WithDRepKeyHash(poolHash).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	combined, err := certificates.NewStakeVoteRegistrationDelegation().
		WithCredential(stakeHash).
		WithDRepKeyHash(poolHash).
		WithPoolKeyHash(stakeHash).
		WithDeposit(100).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	input, err := ledger.NewTransactionInputBuilder().
		WithTxId(bytes.Repeat([]byte{0x04}, lcommon.Blake2b256Size)).
		WithIndex(0).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	tx := ledger.NewTransactionBuilder()
	tx.WithInputs(input)
	output, err := ledger.NewTransactionOutputBuilder().
		WithAddress("addr_test1qz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp").
		WithLovelace(5_000_000).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	tx.WithOutputs(output)
	tx.WithCertificates(drepRegistration, voteDelegation, combined)
	built, err := tx.Build()
	if err != nil {
		t.Fatalf("transaction rejected certificates: %v", err)
	}
	if got := built.Certificates(); len(got) != 3 {
		t.Fatalf("transaction certificates = %d, want 3", len(got))
	}
	txRPC, err := built.Utxorpc()
	if err != nil {
		t.Fatalf("transaction conversion: %v", err)
	}
	if got := len(txRPC.Certificates); got != 3 {
		t.Fatalf("converted transaction certificates = %d, want 3", got)
	}
	assertCertificateRoundTrip(
		t,
		drepRegistration,
		&certificates.DRepRegistrationCertificate{
			RegistrationDrepCertificate: &lcommon.RegistrationDrepCertificate{
				CertType:       16,
				DrepCredential: keyCredential(stakeHash),
				Amount:         100,
				Anchor: &lcommon.GovAnchor{
					Url:      "https://example.test/drep",
					DataHash: [32]byte(vrfHash),
				},
			},
			Amount: 100,
		},
	)
	assertCertificateRoundTrip(
		t,
		voteDelegation,
		&lcommon.VoteDelegationCertificate{
			CertType:        9,
			StakeCredential: keyCredential(stakeHash),
			Drep:            lcommon.Drep{Type: 0, Credential: poolHash},
		},
	)
	assertCertificateRoundTrip(
		t,
		combined,
		&lcommon.StakeVoteRegistrationDelegationCertificate{
			CertType:        13,
			StakeCredential: keyCredential(stakeHash),
			Drep:            lcommon.Drep{Type: 0, Credential: poolHash},
			PoolKeyHash:     lcommon.NewBlake2b224(stakeHash),
			Amount:          100,
		},
	)
	if _, err := voteDelegation.Utxorpc(); err != nil {
		t.Fatalf("vote delegation conversion: %v", err)
	}
	combinedRPC, err := combined.Utxorpc()
	if err != nil {
		t.Fatalf("combined delegation conversion: %v", err)
	}
	if got := combinedRPC.GetStakeVoteRegDelegCert().GetPoolKeyhash(); !bytes.Equal(
		got,
		stakeHash,
	) {
		t.Fatalf(
			"combined delegation pool key hash = %x, want %x",
			got,
			stakeHash,
		)
	}
}

func TestPoolRegistrationUsesSelectedNetwork(t *testing.T) {
	registration, err := certificates.NewPoolRegistration(lcommon.AddressNetworkMainnet).
		WithOperator(poolHash).
		WithVrfKeyHash(vrfHash).
		WithRewardAccountKey(stakeHash).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertPoolRewardAccount(
		t,
		registration,
		uint(lcommon.AddressNetworkMainnet),
		keyCredential(stakeHash),
	)
	if got, ok := registration.RewardAccountNetworkId(); !ok ||
		got != uint(lcommon.AddressNetworkMainnet) {
		t.Fatalf("reward account network = %d, known %t; want mainnet", got, ok)
	}
}

func TestPoolRegistrationSupportsScriptRewardAccount(t *testing.T) {
	registration, err := certificates.NewPoolRegistration(lcommon.AddressNetworkMainnet).
		WithOperator(poolHash).
		WithVrfKeyHash(vrfHash).
		WithRewardAccountScript(stakeHash).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertPoolRewardAccount(
		t,
		registration,
		uint(lcommon.AddressNetworkMainnet),
		scriptCredential(stakeHash),
	)
	if got, ok := registration.RewardAccountNetworkId(); !ok ||
		got != uint(lcommon.AddressNetworkMainnet) {
		t.Fatalf("reward account network = %d, known %t; want mainnet", got, ok)
	}
	if registration.RewardAccountCredential().CredType != lcommon.CredentialTypeScriptHash {
		t.Fatalf(
			"reward account credential type = %d; want script hash",
			registration.RewardAccountCredential().CredType,
		)
	}
}

func TestCombinedBuildersRejectUnsupportedFields(t *testing.T) {
	if _, err := certificates.NewStakeRegistrationDelegation().
		WithCredential(stakeHash).WithPoolKeyHash(poolHash).
		WithDRepKeyHash(poolHash).Build(); err == nil {
		t.Fatal("expected unsupported DRep field to be rejected")
	}
	if _, err := certificates.NewVoteRegistrationDelegation().
		WithCredential(stakeHash).WithDRepKeyHash(poolHash).
		WithPoolKeyHash(poolHash).Build(); err == nil {
		t.Fatal("expected unsupported pool field to be rejected")
	}
}

func TestTransactionRejectsNilCertificate(t *testing.T) {
	input, err := ledger.NewTransactionInputBuilder().
		WithTxId(bytes.Repeat([]byte{0x04}, lcommon.Blake2b256Size)).
		WithIndex(0).Build()
	if err != nil {
		t.Fatal(err)
	}
	output, err := ledger.NewTransactionOutputBuilder().
		WithAddress("addr_test1qz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp").
		WithLovelace(5_000_000).Build()
	if err != nil {
		t.Fatal(err)
	}
	tx := ledger.NewTransactionBuilder()
	tx.WithInputs(input)
	tx.WithOutputs(output)
	tx.WithCertificates(nil)
	_, err = tx.Build()
	if err == nil {
		t.Fatal("expected nil certificate to be rejected")
	}
}

func TestBuildersRejectInvalidHashes(t *testing.T) {
	if _, err := certificates.NewStakeRegistration().WithCredential([]byte{1}).Build(); err == nil {
		t.Fatal("expected invalid stake credential hash to be rejected")
	}
	if _, err := certificates.NewPoolRegistration(lcommon.AddressNetworkTestnet).WithOperator(poolHash).WithVrfKeyHash(vrfHash).WithRewardAccountKey(stakeHash).WithMargin(2, 1).Build(); err == nil {
		t.Fatal("expected out-of-range pool margin to be rejected")
	}
	if _, err := certificates.NewPoolRegistration(lcommon.AddressNetworkTestnet).WithOwners([]byte{1}).Build(); err == nil {
		t.Fatal("expected invalid pool owner hash to be rejected")
	}
	if _, err := certificates.NewPoolRegistration(lcommon.AddressNetworkTestnet).
		WithOperator(poolHash).
		WithVrfKeyHash(vrfHash).
		WithRewardAccountKey(stakeHash).
		WithOwners([]byte{1}).
		WithOwners(stakeHash).
		Build(); err != nil {
		t.Fatalf("valid owner retry rejected: %v", err)
	}
	if _, err := certificates.NewPoolRegistration(lcommon.AddressNetworkTestnet).
		WithOperator(poolHash).
		WithVrfKeyHash(vrfHash).
		WithRewardAccountKey(stakeHash).
		WithRelays(lcommon.PoolRelay{Type: 99}).
		Build(); err == nil {
		t.Fatal("expected invalid pool relay to be rejected")
	}
	longURL := strings.Repeat("x", 129)
	if _, err := certificates.NewDRepRegistration().WithCredential(stakeHash).WithAnchor(longURL, nil).Build(); err == nil {
		t.Fatal("expected oversized DRep anchor URL to be rejected")
	}
	if _, err := certificates.NewResignCommitteeCold().WithColdCredential(stakeHash).WithAnchor(longURL, nil).Build(); err == nil {
		t.Fatal("expected oversized committee anchor URL to be rejected")
	}
	if _, err := certificates.NewPoolRegistration(lcommon.AddressNetworkTestnet).
		WithOperator(poolHash).
		WithVrfKeyHash(vrfHash).
		WithRewardAccountKey(stakeHash).
		WithMetadata(longURL, vrfHash).
		Build(); err == nil {
		t.Fatal("expected oversized pool metadata URL to be rejected")
	}
	if _, err := certificates.NewVoteDelegation().WithCredential(stakeHash).WithDRep(lcommon.Drep{Type: 99}).Build(); err == nil {
		t.Fatal("expected unknown DRep type to be rejected")
	}
}

func TestDRepRegistrationBuilderPreservesWord64Deposit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		amount uint64
	}{
		{name: "above int64", amount: uint64(1) << 63},
		{name: "maximum uint64", amount: ^uint64(0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cert, err := certificates.NewDRepRegistration().
				WithCredential(stakeHash).
				WithDeposit(tc.amount).
				Build()
			require.NoError(t, err)
			require.Equal(t, tc.amount, cert.Amount)
			require.Equal(
				t,
				new(big.Int).SetUint64(tc.amount),
				cert.DepositAmount(),
			)

			wantCBOR, err := cbor.Encode([]any{
				uint(lcommon.CertificateTypeRegistrationDrep),
				keyCredential(stakeHash),
				tc.amount,
				nil,
			})
			require.NoError(t, err)
			require.Equal(t, wantCBOR, cert.Cbor())
			encoded, err := cbor.Encode(cert)
			require.NoError(t, err)
			require.Equal(t, wantCBOR, encoded)

			utxoCertificate, err := cert.Utxorpc()
			require.NoError(t, err)
			coin := utxoCertificate.GetRegDrepCert().GetCoin()
			var got big.Int
			if bigUInt := coin.GetBigUInt(); bigUInt != nil {
				got.SetBytes(bigUInt)
			} else {
				got.SetInt64(coin.GetInt())
			}
			require.Equal(t, new(big.Int).SetUint64(tc.amount), &got)
		})
	}
}

func TestDRepDeregistrationBuilderPreservesWord64Deposit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		amount uint64
	}{
		{name: "above int64", amount: uint64(1) << 63},
		{name: "maximum uint64", amount: ^uint64(0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cert, err := certificates.NewDRepDeregistration().
				WithCredential(stakeHash).
				WithDeposit(tc.amount).
				Build()
			require.NoError(t, err)
			require.Equal(t, tc.amount, cert.Amount)
			require.Equal(
				t,
				new(big.Int).SetUint64(tc.amount),
				cert.DepositAmount(),
			)

			wantCBOR, err := cbor.Encode([]any{
				uint(lcommon.CertificateTypeDeregistrationDrep),
				keyCredential(stakeHash),
				tc.amount,
			})
			require.NoError(t, err)
			require.Equal(t, wantCBOR, cert.Cbor())
			encoded, err := cbor.Encode(cert)
			require.NoError(t, err)
			require.Equal(t, wantCBOR, encoded)

			utxoCertificate, err := cert.Utxorpc()
			require.NoError(t, err)
			coin := utxoCertificate.GetUnregDrepCert().GetCoin()
			var got big.Int
			if bigUInt := coin.GetBigUInt(); bigUInt != nil {
				got.SetBytes(bigUInt)
			} else {
				got.SetInt64(coin.GetInt())
			}
			require.Equal(t, new(big.Int).SetUint64(tc.amount), &got)
		})
	}
}

func TestConwayBuildersSupportScriptCredentials(t *testing.T) {
	tests := []struct {
		name     string
		expected lcommon.Certificate
		build    func() (lcommon.Certificate, error)
	}{
		{
			name: "vote registration delegation",
			expected: &lcommon.VoteRegistrationDelegationCertificate{
				CertType:        12,
				StakeCredential: scriptCredential(stakeHash),
				Drep:            lcommon.Drep{Type: 0, Credential: poolHash},
			},
			build: func() (lcommon.Certificate, error) {
				return certificates.NewVoteRegistrationDelegation().
					WithScriptCredential(stakeHash).
					WithDRepKeyHash(poolHash).
					Build()
			},
		},
		{
			name: "stake vote registration delegation",
			expected: &lcommon.StakeVoteRegistrationDelegationCertificate{
				CertType:        13,
				StakeCredential: scriptCredential(stakeHash),
				PoolKeyHash:     lcommon.NewBlake2b224(poolHash),
				Drep:            lcommon.Drep{Type: 0, Credential: poolHash},
			},
			build: func() (lcommon.Certificate, error) {
				return certificates.NewStakeVoteRegistrationDelegation().
					WithScriptCredential(stakeHash).
					WithDRepKeyHash(poolHash).
					WithPoolKeyHash(poolHash).
					Build()
			},
		},
		{
			name: "committee authorization",
			expected: &lcommon.AuthCommitteeHotCertificate{
				CertType:       14,
				ColdCredential: scriptCredential(stakeHash),
				HotCredential:  scriptCredential(poolHash),
			},
			build: func() (lcommon.Certificate, error) {
				return certificates.NewAuthCommitteeHot().
					WithColdScriptCredential(stakeHash).
					WithHotScriptCredential(poolHash).
					Build()
			},
		},
		{
			name: "committee resignation",
			expected: &lcommon.ResignCommitteeColdCertificate{
				CertType:       15,
				ColdCredential: scriptCredential(stakeHash),
			},
			build: func() (lcommon.Certificate, error) {
				return certificates.NewResignCommitteeCold().
					WithColdScriptCredential(stakeHash).
					Build()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert, err := tt.build()
			if err != nil {
				t.Fatal(err)
			}
			assertCertificateRoundTrip(t, cert, tt.expected)
		})
	}
}

func assertCertificateRoundTrip(
	t *testing.T,
	cert, expected lcommon.Certificate,
) {
	t.Helper()
	require.EqualExportedValues(t, expected, cert, "built certificate fields")
	wire, err := cbor.Encode(cert)
	if err != nil {
		t.Fatalf("encode certificate: %v", err)
	}
	var decoded lcommon.CertificateWrapper
	if _, err := cbor.Decode(wire, &decoded); err != nil {
		t.Fatalf("decode certificate: %v", err)
	}
	if decoded.Type != cert.Type() {
		t.Fatalf(
			"certificate type mismatch: got %d, want %d",
			decoded.Type,
			cert.Type(),
		)
	}
	// Pool registration can re-encode its retained CBOR without consulting
	// decoded fields. Expectations come from the builder inputs.
	decodedExpected := expected
	if drepRegistration, ok := expected.(*certificates.DRepRegistrationCertificate); ok {
		decodedExpected = drepRegistration.RegistrationDrepCertificate
	}
	if drepDeregistration, ok := expected.(*certificates.DRepDeregistrationCertificate); ok {
		decodedExpected = drepDeregistration.DeregistrationDrepCertificate
	}
	require.EqualExportedValues(
		t,
		decodedExpected,
		decoded.Certificate,
		"decoded certificate fields",
	)
	if want, ok := expected.(*lcommon.PoolRegistrationCertificate); ok {
		got := decoded.Certificate.(*lcommon.PoolRegistrationCertificate)
		require.Equal(
			t,
			want.Margin.Rat,
			cert.(*lcommon.PoolRegistrationCertificate).Margin.Rat,
			"built pool margin",
		)
		require.Equal(t, want.Margin.Rat, got.Margin.Rat, "decoded pool margin")
	}
	decodedWire, err := cbor.Encode(decoded.Certificate)
	if err != nil {
		t.Fatalf("re-encode decoded certificate: %v", err)
	}
	if !bytes.Equal(decodedWire, wire) {
		t.Fatalf(
			"certificate body changed during round trip: got %x, want %x",
			decodedWire,
			wire,
		)
	}
}

func keyCredential(hash []byte) lcommon.Credential {
	return lcommon.Credential{
		CredType:   lcommon.CredentialTypeAddrKeyHash,
		Credential: lcommon.NewBlake2b224(hash),
	}
}

func scriptCredential(hash []byte) lcommon.Credential {
	return lcommon.Credential{
		CredType:   lcommon.CredentialTypeScriptHash,
		Credential: lcommon.NewBlake2b224(hash),
	}
}

func assertPoolRewardAccount(
	t *testing.T,
	cert *lcommon.PoolRegistrationCertificate,
	network uint,
	credential lcommon.Credential,
) {
	t.Helper()
	wire, err := cbor.Encode(cert)
	require.NoError(t, err)
	var decoded lcommon.CertificateWrapper
	_, err = cbor.Decode(wire, &decoded)
	require.NoError(t, err)
	for _, got := range []*lcommon.PoolRegistrationCertificate{cert, decoded.Certificate.(*lcommon.PoolRegistrationCertificate)} {
		id, known := got.RewardAccountNetworkId()
		require.True(t, known)
		require.Equal(t, network, id, "reward account network")
		require.EqualExportedValues(
			t,
			credential,
			got.RewardAccountCredential(),
			"reward account credential",
		)
	}
}

func TestGovernanceBuildersPreserveRequestedFields(t *testing.T) {
	anchor := &lcommon.GovAnchor{
		Url:      "https://example.test/governance",
		DataHash: [32]byte(vrfHash),
	}
	tests := []struct {
		name     string
		build    func() (lcommon.Certificate, error)
		expected lcommon.Certificate
	}{
		{
			name: "DRep deregistration",
			build: func() (lcommon.Certificate, error) {
				return certificates.NewDRepDeregistration().
					WithScriptCredential(stakeHash).
					WithDeposit(321).
					Build()
			},
			expected: &certificates.DRepDeregistrationCertificate{
				DeregistrationDrepCertificate: &lcommon.DeregistrationDrepCertificate{
					CertType:       17,
					DrepCredential: scriptCredential(stakeHash),
					Amount:         321,
				},
				Amount: 321,
			},
		},
		{
			name: "DRep update",
			build: func() (lcommon.Certificate, error) {
				return certificates.NewDRepUpdate().
					WithScriptCredential(stakeHash).
					WithAnchor(anchor.Url, vrfHash).
					Build()
			},
			expected: &lcommon.UpdateDrepCertificate{
				CertType:       18,
				DrepCredential: scriptCredential(stakeHash),
				Anchor:         anchor,
			},
		},
		{
			name: "stake and vote delegation",
			build: func() (lcommon.Certificate, error) {
				return certificates.NewStakeVoteDelegation().
					WithScriptCredential(stakeHash).
					WithPoolKeyHash(poolHash).
					WithDRepScriptHash(vrfHash[:28]).
					Build()
			},
			expected: &lcommon.StakeVoteDelegationCertificate{
				CertType:        10,
				StakeCredential: scriptCredential(stakeHash),
				PoolKeyHash:     lcommon.NewBlake2b224(poolHash),
				Drep: lcommon.Drep{
					Type:       1,
					Credential: vrfHash[:28],
				},
			},
		},
		{
			name: "stake registration and delegation",
			build: func() (lcommon.Certificate, error) {
				return certificates.NewStakeRegistrationDelegation().
					WithScriptCredential(stakeHash).
					WithPoolKeyHash(poolHash).
					WithDeposit(456).
					Build()
			},
			expected: &lcommon.StakeRegistrationDelegationCertificate{
				CertType:        11,
				StakeCredential: scriptCredential(stakeHash),
				PoolKeyHash:     lcommon.NewBlake2b224(poolHash),
				Amount:          456,
			},
		},
		{
			name: "vote registration and delegation with deposit",
			build: func() (lcommon.Certificate, error) {
				return certificates.NewVoteRegistrationDelegation().
					WithCredential(stakeHash).
					WithDRepScriptHash(poolHash).
					WithDeposit(789).
					Build()
			},
			expected: &lcommon.VoteRegistrationDelegationCertificate{
				CertType:        12,
				StakeCredential: keyCredential(stakeHash),
				Drep:            lcommon.Drep{Type: 1, Credential: poolHash},
				Amount:          789,
			},
		},
		{
			name: "committee resignation anchor",
			build: func() (lcommon.Certificate, error) {
				return certificates.NewResignCommitteeCold().
					WithColdCredential(stakeHash).
					WithAnchor(anchor.Url, vrfHash).
					Build()
			},
			expected: &lcommon.ResignCommitteeColdCertificate{
				CertType:       15,
				ColdCredential: keyCredential(stakeHash),
				Anchor:         anchor,
			},
		},
		{
			name: "committee key authorization",
			build: func() (lcommon.Certificate, error) {
				return certificates.NewAuthCommitteeHot().
					WithColdCredential(stakeHash).
					WithHotCredential(poolHash).
					Build()
			},
			expected: &lcommon.AuthCommitteeHotCertificate{
				CertType:       14,
				ColdCredential: keyCredential(stakeHash),
				HotCredential:  keyCredential(poolHash),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert, err := tt.build()
			require.NoError(t, err)
			assertCertificateRoundTrip(t, cert, tt.expected)
		})
	}
}

func TestVoteDelegationPreservesDRepVariants(t *testing.T) {
	tests := []struct {
		name string
		drep lcommon.Drep
	}{
		{
			name: "key",
			drep: lcommon.Drep{
				Type:       lcommon.DrepTypeAddrKeyHash,
				Credential: poolHash,
			},
		},
		{
			name: "script",
			drep: lcommon.Drep{
				Type:       lcommon.DrepTypeScriptHash,
				Credential: poolHash,
			},
		},
		{name: "abstain", drep: lcommon.Drep{Type: lcommon.DrepTypeAbstain}},
		{
			name: "no confidence",
			drep: lcommon.Drep{Type: lcommon.DrepTypeNoConfidence},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert, err := certificates.NewVoteDelegation().
				WithScriptCredential(stakeHash).
				WithDRep(tt.drep).
				Build()
			require.NoError(t, err)
			assertCertificateRoundTrip(
				t,
				cert,
				&lcommon.VoteDelegationCertificate{
					CertType:        9,
					StakeCredential: scriptCredential(stakeHash),
					Drep:            tt.drep,
				},
			)
		})
	}
}
