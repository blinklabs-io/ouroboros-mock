// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

// Package scripts provides deterministic script and witness fixtures.
package scripts

import (
	"encoding/hex"
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

func native(item any) (lcommon.NativeScript, error) {
	raw, err := cbor.Encode(item)
	if err != nil {
		return lcommon.NativeScript{}, err
	}
	var script lcommon.NativeScript
	if err := script.UnmarshalCBOR(raw); err != nil {
		return lcommon.NativeScript{}, err
	}
	return script, nil
}

func NewScriptSig(keyHash lcommon.Blake2b224) (lcommon.NativeScript, error) {
	return native(lcommon.NativeScriptPubkey{Type: 0, Hash: keyHash[:]})
}
func NewScriptAll(scripts ...lcommon.NativeScript) (lcommon.NativeScript, error) {
	return native(lcommon.NativeScriptAll{Type: 1, Scripts: scripts})
}
func NewScriptAny(scripts ...lcommon.NativeScript) (lcommon.NativeScript, error) {
	return native(lcommon.NativeScriptAny{Type: 2, Scripts: scripts})
}
func NewScriptAtLeast(required uint, scripts ...lcommon.NativeScript) (lcommon.NativeScript, error) {
	return native(lcommon.NativeScriptNofK{Type: 3, N: required, Scripts: scripts})
}
func NewInvalidBefore(slot uint64) (lcommon.NativeScript, error) {
	return native(lcommon.NativeScriptInvalidBefore{Type: 4, Slot: slot})
}
func NewInvalidAfter(slot uint64) (lcommon.NativeScript, error) {
	return native(lcommon.NativeScriptInvalidHereafter{Type: 5, Slot: slot})
}

// This is a compact, valid UPLC program wrapped in the CBOR bytestring format
// expected by gouroboros. It is shared by all supported language versions;
// the language tag changes the script hash and evaluator language.
const alwaysSucceedsHex = "587f01010032323232323225333002323232323253" +
	"330073370e900118041baa0011323232533300a3370e900018059baa00513232" +
	"533300f301100214a22c6eb8c03c004c030dd50028b18069807001180600098" +
	"049baa00116300a300b0023009001300900230070013004375400229309b2b2" +
	"b9a5573aaae7955cfaba157441"

func plutus(version uint) (lcommon.Script, error) {
	raw, err := hex.DecodeString(alwaysSucceedsHex)
	if err != nil {
		return nil, err
	}
	switch version {
	case 1:
		return lcommon.PlutusV1Script(raw), nil
	case 2:
		return lcommon.PlutusV2Script(raw), nil
	case 3:
		return lcommon.PlutusV3Script(raw), nil
	default:
		return nil, fmt.Errorf("unsupported Plutus version %d", version)
	}
}
func NewPlutusScript(version uint) (lcommon.Script, error)      { return plutus(version) }
func AlwaysSucceedsScript(version uint) (lcommon.Script, error) { return plutus(version) }

func AlwaysFailsScript(version uint) (lcommon.Script, error) {
	if version < 1 || version > 3 {
		return nil, fmt.Errorf("unsupported Plutus version %d", version)
	}
	// A truncated program is deliberately rejected by the evaluator. Keeping
	// this fixture invalid makes its failure deterministic across language eras.
	raw := []byte{0x00}
	switch version {
	case 1:
		return lcommon.PlutusV1Script(raw), nil
	case 2:
		return lcommon.PlutusV2Script(raw), nil
	case 3:
		return lcommon.PlutusV3Script(raw), nil
	default:
		return nil, fmt.Errorf("unsupported Plutus version %d", version)
	}
}

func ScriptHash(script lcommon.Script) lcommon.ScriptHash {
	if script == nil {
		return lcommon.ScriptHash{}
	}
	return script.Hash()
}
func NewDatum(value data.PlutusData) lcommon.Datum { return lcommon.Datum{Data: value} }
func NewRedeemer(tag lcommon.RedeemerTag, index uint32, value data.PlutusData, exUnits lcommon.ExUnits) (lcommon.RedeemerKey, lcommon.RedeemerValue) {
	return lcommon.RedeemerKey{Tag: tag, Index: index}, lcommon.RedeemerValue{Data: NewDatum(value), ExUnits: exUnits}
}
func ReferenceScript(script lcommon.Script) (lcommon.ScriptRef, error) {
	if script == nil {
		return lcommon.ScriptRef{}, fmt.Errorf("script is nil")
	}
	version, ok := lcommon.PlutusScriptVersion(script)
	if ok {
		return lcommon.ScriptRef{Type: version + 1, Script: script}, nil
	}
	if _, ok := script.(lcommon.NativeScript); ok {
		return lcommon.ScriptRef{Type: 0, Script: script}, nil
	}
	return lcommon.ScriptRef{}, fmt.Errorf("unsupported script type %T", script)
}
