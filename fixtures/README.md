# Fixtures

Convenience fixtures imported from upstream Cardano repositories.

The committed fixtures under `fixtures/upstream/` are a curated subset of
official upstream data that is useful for block, header, transaction, genesis,
protocol-parameter, and governance-metadata testing.

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
- `cardano-ledger`: era protocol parameter goldens plus a small Alonzo CBOR
  fixture set
- `cardano-api`: canonical JSON, protocol-parameter, genesis, and governance
  anchor fixtures
- `cardano-node`: testnet genesis spec fixtures

Intentional exclusions:

- Cardano Blueprint conformance vectors are imported into `conformance/testdata/`
  from the pinned `cardano-blueprint` submodule; see `conformance/CORPUS.md`.
- Plutus conformance data is managed separately in `plutigo`
- `SerialisedBlock_*` and `SerialisedHeader_*` placeholder files from
  `ouroboros-consensus` are not imported

## Public Test Harness

The `github.com/blinklabs-io/ouroboros-mock/fixtures` package exposes this
fixture corpus through a public test harness so downstream projects can reuse
the same inventory, filtering logic, and built-in execution paths.

`RunAllExecutions*` validates actual fixture contents, not just file presence or
serialization. The built-in runner decodes ledger blocks, headers, and
transactions where possible, cross-checks paired consensus fixtures, derives
protocol parameters from genesis files, applies protocol-parameter updates,
validates governance metadata, and walks every translation corpus case.

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
