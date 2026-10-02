# Fixtures

Convenience fixtures imported from upstream Cardano repositories.

The committed fixtures under `fixtures/upstream/` are a curated subset of
official upstream data that is useful for block, header, transaction, genesis,
protocol-parameter, governance-metadata, Byron SSC, and script testing.

Regenerate them with:

```bash
make download-upstream-fixtures
```

The sync target defaults to downloading repository tarballs from GitHub. For
offline or local verification, you can point it at existing checkouts:

```bash
OUROBOROS_CONSENSUS_SRC=/tmp/ouroboros-consensus \
CARDANO_LEDGER_SRC=/tmp/cardano-ledger \
CARDANO_API_SRC=/tmp/cardano-api \
CARDANO_NODE_SRC=/tmp/cardano-node \
make download-upstream-fixtures
```

Current sources:

- `ouroboros-consensus`: raw `Block_*`, `Header_*`, `GenTx_*`, and
  `GenTxId_*` fixtures from `ouroboros-consensus-cardano/golden/cardano/`
- `cardano-ledger`: era protocol parameter goldens, a small Alonzo CBOR
  fixture set, and four Byron SSC annotated CBOR dumps
- `cardano-api`: canonical JSON, protocol-parameter, genesis, and governance
  anchor fixtures, plus paired PlutusV1 hex and binary script captures
- `cardano-node`: testnet genesis spec fixtures

Intentional exclusions:

- Cardano Blueprint conformance vectors are imported into `conformance/testdata/`
  from the pinned `cardano-blueprint` submodule; see `conformance/CORPUS.md`.
- Plutus conformance data is managed separately in `plutigo`
- `SerialisedBlock_*` and `SerialisedHeader_*` placeholder files from
  `ouroboros-consensus` are not imported

### Curated source contracts

The updater pins each upstream revision and copies selected files without
rewriting them. Byron `ssc/CommitmentsMap`, `OpeningsMap`, `SharesMap` and
`VssCertificatesMap` retain their original annotated hexadecimal text. The
decoder checks every hexadecimal offset and chunk before exposing CBOR bytes.
These vectors exercise tag-258 sets and stakeholder-keyed maps used by Byron
SSC decoding, accumulation and proof tests in `gouroboros`. The shared harness
checks the nested wire shapes defined by `Cardano/Chain/Ssc.hs`; proof hashes,
signatures and epoch-state behavior remain downstream responsibilities.

Cardano API's `Script/PlutusScriptV1/alwayssucceeds.txt` and
`alwayssucceeds.bin` contain the same encoded script in hex and binary form.
`Test/Golden/Cardano/Api/Script.hs` supplies the expected PlutusV1 hash
`58503a1d89a21fc9fc53d6a7cccef47341175a8f47636f57ccbdca2d`.
The runner checks that identity and, when both captures are present, requires
their decoded bytes to match. It does not evaluate the program.

The following inspected families stay with their source interfaces:

- Cardano API's six `Script/SimpleV1` and `Script/SimpleV2` goldens pin pretty
  JSON for signature, all/any/threshold, and before/after constructors in
  `Test/Golden/Cardano/Api/Script.hs`. They describe the API's JSON conversion
  contract, rather than the CBOR consumed by Ouroboros. JSON conversion remains
  with that interface; native-script evaluation belongs to downstream ledger
  tests with transaction witnesses and validity intervals.
- `Test/Cardano/Api/Metadata.hs` embeds its transaction metadata JSON examples
  directly and checks schema conversion and round-trips. The separate error
  goldens contain rendered diagnostic strings, rather than serialized metadata
  payloads. Those conversion and diagnostic contracts remain with Cardano API.
  Governance anchor payloads are included in this corpus and validated here.
- Cardano Node's `files/golden/queries` captures are outputs of the CLI against
  a running Conway testnet in `Cardano/Testnet/Test/Cli/Query.hs`. Tip and
  stake/DRep captures replace identifiers or chain positions with
  `<redacted>`; the upstream stake and DRep comparisons are disabled for
  changing reward, expiry or stake values. Constitution, governance state and
  protocol parameters depend on the generated testnet state, and reference
  script size depends on a transaction submitted by that test. CLI rendering
  and query results belong to consumers that reproduce that state, rather than
  the shared ledger-payload validators. Genesis and protocol-parameter inputs
  remain included for deterministic shared checks.

## Public Test Harness

The `github.com/blinklabs-io/ouroboros-mock/fixtures` package exposes this
fixture corpus through a public test harness so downstream projects can reuse
the same inventory, filtering logic, and built-in execution paths.

`RunAllExecutions*` validates fixture contents using the shared checks below.
The built-in runner decodes ledger blocks, headers, and
transactions where possible, cross-checks paired consensus fixtures, derives
protocol parameters from genesis files, applies protocol-parameter updates,
validates governance metadata, and walks every translation corpus case.

### Validation ownership

The shared harness owns source classification, canonical manifest paths,
file presence, and the following checks. Manifest entries must identify regular
files and cannot repeat a path, including aliases containing `.` or `..`.
Execution failures retain the fixture repository, source path, payload kind,
format, era, and underlying error for downstream diagnostics.

| Fixture family | Shared harness checks | Downstream implementation checks |
| --- | --- | --- |
| Blocks and headers | Decode supported era payloads; check paired era, hash, slot and block number | Ledger transitions, consensus acceptance, body proofs and cryptographic validity |
| Transactions and IDs | Decode supported payloads; compare paired transaction hashes except explicitly unpaired captures | UTxO rules, fees, witnesses, scripts and transaction acceptance |
| Genesis and protocol parameters | Decode genesis values, derive parameter values, decode and apply parameter updates | Epoch transitions and effects on ledger state |
| Governance metadata | Validate the imported metadata schema and expected invalid metadata failures | Anchor retrieval and governance state transitions |
| Translation corpus | Decode every five-field case, validate CBOR context and expected-value containers, and check the transaction container and body hash | Perform era or Plutus context translation and compare its result with the expected value |
| Byron SSC | Decode annotated hex offsets; validate tag-258 sets, map keys, tuple widths, byte strings, nested shares and unsigned certificate epochs | Signature and proof verification, commitment/opening consistency and epoch-state accumulation |
| Plutus script captures | Check the CBOR byte-string container, pinned PlutusV1 hash and hex/binary byte equality | Program evaluation, budgets and transaction script contexts |

Translation checks inspect the supplied context and expected value independently;
they do not run a translator or establish equality between its output and the
golden result. Downstream projects should reuse these preserved inputs for their
own semantic validators. Encoding contracts remain byte comparisons where the
representation itself is part of the contract.

Basic usage:

```go
tmpDir := t.TempDir()
fixturesRoot, err := fixtures.ExtractEmbeddedFixtures(tmpDir)
if err != nil {
	t.Fatal(err)
}

harness := fixtures.NewHarness(fixtures.HarnessConfig{
	FixturesRoot: fixturesRoot,
})

results, err := harness.RunAllExecutionsWithResults()
if err != nil {
	t.Fatal(err)
}
for _, result := range results {
	if !result.Success {
		t.Fatalf("%s: %v", result.Fixture.RelPath, result.Error)
	}
}
```

You can also decode fixture families through a single public API without
re-implementing the source-specific wrapper handling:

```go
fixture, err := harness.Fixture(
	"ouroboros-consensus/ouroboros-consensus-cardano/golden/cardano/CardanoNodeToNodeVersion2/GenTx_Conway",
)
if err != nil {
	t.Fatal(err)
}

tx, err := fixture.DecodeLedgerTransaction()
if err != nil {
	t.Fatal(err)
}
if tx.Type() != int(ledger.TxTypeConway) {
	t.Fatalf("unexpected tx type: %d", tx.Type())
}
```

Current upstream exceptions are encoded in the harness rather than ignored:

- the current `Block_Dijkstra` consensus payload is truncated upstream, so the
  runner validates the outer wrapper/header path instead of full block decode
- The upstream `GenTx_Byron` and `GenTxId_Byron` files are not a matching
  transaction/ID pair. The ID value matches the GenTx's referenced input ID,
  not its transaction-body hash. Both fixtures are decoded and checked
  independently; pair comparisons use the explicit unpaired-fixture metadata.
- Dijkstra `GenTx_*` fixtures currently validate through payload/body-hash
  semantics because the imported fixture shape is ahead of full
  `gouroboros` transaction decoding support

## Generated block chains

`GenerateShelleyChain`, `GenerateAllegraChain`, `GenerateMaryChain`,
`GenerateAlonzoChain`, `GenerateBabbageChain`, `GenerateConwayChain`, and
`GenerateDijkstraChain` return connected empty blocks whose CBOR round-trips
through `gouroboros`.
`GenerateConwayChainWithTransactions` returns connected Conway blocks containing
one CDDL-shaped transaction with one input and one output per block. The
transactions are suitable for decode, storage, and replay tests but do not
reference spendable ledger UTxOs.
Use `GenerateBabbageChainWithProtocolVersion` when a test needs valid Babbage
bytes with a specific header protocol version, including an unknown version for
fail-closed classification coverage.

### Strict decoder compatibility

The curated upstream corpus is retained byte-for-byte. A small explicit set of
historical consensus captures contains two-byte hash placeholders. Older
decoders accept them, while strict hash decoding rejects them with the recorded
hash-length error. The execution harness continues to execute every such
fixture: it accepts either a complete decode or that exact documented rejection.
Any other failure remains a harness failure, and newly added fixtures are not
included without an explicit review of their decoding contract.
