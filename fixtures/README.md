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
A single capture remains independently valid only when it has the pinned
upstream hash; a missing counterpart does not bypass that check.

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
Admission and reads resolve paths within the fixture root; symlinks that leave
it are rejected. `NewFixture` preserves that root through `Path` and `RelPath`.
A directly constructed fixture with only `Path` uses its parent directory as
the root. Reads reject inconsistent path metadata and non-regular files.
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

`NewBlockBuilder(era)` builds one block for any era in `SupportedEras`. Set the
block number, slot, previous hash, issuer key, and protocol version, and add
transactions with `WithTransactions`: each carries its era's transaction CBOR
(for example a `conway.ConwayTransaction`), and a transaction whose validity
flag is false is listed as invalid. The header's body size and hash are derived
from the encoded body, and `Build` returns the block as `gouroboros` decodes it;
`BuildHeader` returns its header. Byron builds an epoch boundary block, or a
regular block with `WithByronMainBlock`; Byron blocks carry no transactions.
Dijkstra blocks use the pinned CDDL shape described below. `NewSequence(era)`
builds connected blocks one at a time with `Next` or in bulk with `Blocks`,
`GenesisBlock` returns the first block of an era, and `RandomBlock(era, seed)`
returns a block whose fields are drawn from the seed. The `Generate*Chain`
functions are built on these.

`NewDijkstraBlockBuilder` builds one Dijkstra block in the pinned Dijkstra CDDL
shape: a non-segregated body with the optional Leios and Peras certificate
slots, and a 12-field header body whose `block_body_contains_leios_cert` flag
follows the body's Leios certificate and whose `eb_announcement` is set with
`WithEbAnnouncement`. It derives the body size and hash and decodes its output
before returning it. A transaction that carries `block_transaction` CBOR, such
as one from `NewDijkstraTransactionBuilder`, keeps those bytes; any other
transaction is encoded as the transaction builder encodes it. Both builders
reject a transaction whose encoding lacks a required body key, carries an
empty sub-transaction set, or uses a witness set key outside 0-7. `GenerateConwayToDijkstraChain` builds a connected chain
spanning the PV12 era boundary from the same builder. `GenerateDijkstraChain`
keeps the 10-field Babbage header body, which `ledger.DetermineBlockType`
classifies; that function rejects the 12-field Dijkstra header body.

Use `NewDijkstraTransactionBuilder` to construct Dijkstra block
transactions with guards, subtransactions, redeemers, metadata, and the
`TxIsValid` flag. The builder encodes the `block_transaction` form, including
the required transaction-body keys gouroboros omits when empty, and decodes it
through the Dijkstra block-body decoder, where that validity flag is encoded.
The Dijkstra witness set has no Plutus V4 script field. The CDDL permits V4
scripts as output reference scripts or in `auxiliary_data_map` key 5; this
builder exposes metadata but does not construct script-bearing auxiliary data.
The builder lives in `fixtures`
rather than `ledger` because gouroboros's in-package `ledger/dijkstra` tests
import this module's `ledger` package, which therefore cannot import
`gouroboros/ledger/dijkstra`; gouroboros uses the builder from external
`dijkstra_test` test packages.

```go
tx, err := fixtures.NewDijkstraTransactionBuilder().
	WithTxGuards(guards).
	WithRedeemers(redeemers).
	WithTxIsValid(false).
	Build()
if err != nil {
	t.Fatal(err)
}

block, err := fixtures.NewDijkstraBlockBuilder().
	WithBlockNumber(42).
	WithSlot(1200).
	WithTransactions(*tx).
	Build()
if err != nil {
	t.Fatal(err)
}
```

### Strict decoder compatibility

The curated upstream corpus is retained byte-for-byte. A small explicit set of
historical consensus captures contains two-byte hash placeholders. Older
decoders accept them, while strict hash decoding rejects them with the recorded
hash-length error. The execution harness continues to execute every such
fixture: it accepts either a complete decode or that exact documented rejection.
Any other failure remains a harness failure, and newly added fixtures are not
included without an explicit review of their decoding contract.

Fixture reads derive their confinement root from the public `Path` and
`RelPath` metadata; a directly constructed fixture with only `Path` uses its
parent directory. Read and execution errors retain their underlying causes,
including resource cleanup failures, and execution errors include metadata for
every fixture kind.

Script execution is limited to the classified captures with pinned upstream
hashes. `PlutusScriptBytes` checks the byte-string container independently;
structural validity alone does not supply an expected upstream hash. Empty SSC
containers validate successfully and report zero executable cases.
