package scripts

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

func TestNativeScriptBuildersRoundTrip(t *testing.T) {
	var key lcommon.Blake2b224
	for i := range key {
		key[i] = byte(i)
	}
	sig, err := NewScriptSig(key)
	if err != nil {
		t.Fatal(err)
	}
	all, err := NewScriptAll(sig)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := cbor.Encode(all)
	if err != nil {
		t.Fatal(err)
	}
	var decoded lcommon.NativeScript
	if err := decoded.UnmarshalCBOR(raw); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, decoded.RawScriptBytes()) {
		t.Fatal("native script did not round-trip")
	}
	if decoded.Hash() != all.Hash() {
		t.Fatal("native script hash changed after round-trip")
	}
}

func TestPlutusScriptBuilders(t *testing.T) {
	for version := uint(1); version <= 3; version++ {
		script, err := AlwaysSucceedsScript(version)
		if err != nil {
			t.Fatal(err)
		}
		if len(script.RawScriptBytes()) == 0 {
			t.Fatalf("v%d script is empty", version)
		}
		ref, err := ReferenceScript(script)
		if err != nil {
			t.Fatal(err)
		}
		if ref.Script.Hash() != ScriptHash(script) {
			t.Fatalf("v%d reference hash mismatch", version)
		}
	}
	if _, err := NewPlutusScript(4); err == nil {
		t.Fatal("unsupported version accepted")
	}
}

func TestDatumAndRedeemerBuilders(t *testing.T) {
	datum := NewDatum(data.NewInteger(big.NewInt(7)))
	key, value := NewRedeemer(lcommon.RedeemerTagSpend, 2, datum.Data, lcommon.ExUnits{Memory: 10, Steps: 20})
	if key.Tag != lcommon.RedeemerTagSpend || key.Index != 2 {
		t.Fatalf("unexpected key: %#v", key)
	}
	if value.ExUnits.Memory != 10 || value.Data.Data == nil {
		t.Fatalf("unexpected value: %#v", value)
	}
}
