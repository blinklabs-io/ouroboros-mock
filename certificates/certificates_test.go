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
	registration, err := certificates.NewStakeRegistration().WithCredential(stakeHash).Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(t, registration)

	deregistration, err := certificates.NewStakeDeregistration().WithScriptCredential(stakeHash).Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(t, deregistration)

	delegation, err := certificates.NewStakeDelegation().WithCredential(stakeHash).WithPoolKeyHash(poolHash).Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(t, delegation)
}

func TestPoolBuildersReturnRoundTrippableCertificates(t *testing.T) {
	registration, err := certificates.NewPoolRegistration().
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

	retirement, err := certificates.NewPoolRetirement().WithPoolKeyHash(poolHash).WithEpoch(42).Build()
	if err != nil {
		t.Fatal(err)
	}
	assertCertificateRoundTrip(t, retirement)
}

func TestGovernanceBuildersReturnCertificatesUsableInTransactions(t *testing.T) {
	drepRegistration, err := certificates.NewDRepRegistration().WithCredential(stakeHash).WithDeposit(100).WithAnchor("https://example.test/drep", vrfHash).Build()
	if err != nil {
		t.Fatal(err)
	}
	voteDelegation, err := certificates.NewVoteDelegation().WithCredential(stakeHash).WithDRepKeyHash(poolHash).Build()
	if err != nil {
		t.Fatal(err)
	}
	combined, err := certificates.NewStakeVoteRegistrationDelegation().WithCredential(stakeHash).WithDRepKeyHash(poolHash).WithPoolKeyHash(poolHash).WithDeposit(100).Build()
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
	_, err = tx.Build()
	if err != nil {
		t.Fatalf("transaction rejected certificates: %v", err)
	}
	assertCertificateRoundTrip(t, drepRegistration)
	if _, err := voteDelegation.Utxorpc(); err != nil {
		t.Fatalf("vote delegation conversion: %v", err)
	}
	if _, err := combined.Utxorpc(); err != nil {
		t.Fatalf("combined delegation conversion: %v", err)
	}
}

func TestBuildersRejectInvalidHashes(t *testing.T) {
	if _, err := certificates.NewStakeRegistration().WithCredential([]byte{1}).Build(); err == nil {
		t.Fatal("expected invalid stake credential hash to be rejected")
	}
	if _, err := certificates.NewPoolRegistration().WithOperator(poolHash).WithVrfKeyHash(vrfHash).WithRewardAccountKey(stakeHash).WithMargin(2, 1).Build(); err == nil {
		t.Fatal("expected out-of-range pool margin to be rejected")
	}
	if _, err := certificates.NewDRepRegistration().WithCredential(stakeHash).WithDeposit(^uint64(0)).Build(); err == nil {
		t.Fatal("expected overflowing DRep deposit to be rejected")
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
		t.Fatalf("certificate type mismatch: got %d, want %d", decoded.Type, cert.Type())
	}
}
