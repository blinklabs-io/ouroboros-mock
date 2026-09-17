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
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/ouroboros-mock/certificates"
	"github.com/blinklabs-io/ouroboros-mock/ledger"
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
	assertCertificateRoundTrip(t, registration)

	deregistration, err := certificates.NewStakeDeregistration().
		WithScriptCredential(stakeHash).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(t, deregistration)

	delegation, err := certificates.NewStakeDelegation().
		WithCredential(stakeHash).
		WithPoolKeyHash(poolHash).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(t, delegation)

	registrationWithDeposit, err := certificates.NewRegistration().
		WithScriptCredential(stakeHash).
		WithDeposit(123).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(t, registrationWithDeposit)

	deregistrationWithRefund, err := certificates.NewDeregistration().
		WithCredential(stakeHash).
		WithDeposit(123).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(t, deregistrationWithRefund)
}

func TestPoolBuildersReturnRoundTrippableCertificates(t *testing.T) {
	registration, err := certificates.NewPoolRegistration(lcommon.AddressNetworkTestnet).
		WithOperator(poolHash).
		WithVrfKeyHash(vrfHash).
		WithRewardAccountKey(stakeHash).
		WithOwners(stakeHash).
		WithMargin(1, 100).
		WithPledge(1000).
		WithCost(500).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(t, registration)

	retirement, err := certificates.NewPoolRetirement().
		WithPoolKeyHash(poolHash).
		WithEpoch(42).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(t, retirement)
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
	assertCertificateRoundTrip(t, drepRegistration)
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
	if _, err := certificates.NewDRepRegistration().WithCredential(stakeHash).WithDeposit(^uint64(0)).Build(); err == nil {
		t.Fatal("expected overflowing DRep deposit to be rejected")
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

func TestConwayBuildersSupportScriptCredentials(t *testing.T) {
	tests := []struct {
		name  string
		build func() (lcommon.Certificate, error)
	}{
		{
			name: "vote registration delegation",
			build: func() (lcommon.Certificate, error) {
				return certificates.NewVoteRegistrationDelegation().
					WithScriptCredential(stakeHash).
					WithDRepKeyHash(poolHash).
					Build()
			},
		},
		{
			name: "stake vote registration delegation",
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
			build: func() (lcommon.Certificate, error) {
				return certificates.NewAuthCommitteeHot().
					WithColdScriptCredential(stakeHash).
					WithHotScriptCredential(poolHash).
					Build()
			},
		},
		{
			name: "committee resignation",
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
			assertCertificateRoundTrip(t, cert)
		})
	}
}

func assertCertificateRoundTrip(t *testing.T, cert lcommon.Certificate) {
	t.Helper()
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
