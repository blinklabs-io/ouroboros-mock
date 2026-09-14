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
- Byron `GenTxId_*` fixtures do not currently line up with
  `gouroboros`'s Byron transaction hash semantics, so they are validated
  independently rather than as a paired tx/txid round-trip
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
