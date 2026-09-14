# Conformance corpus

The ledger corpus is imported from the pinned `cardano-blueprint` submodule at
`fixtures/cardano-blueprint`. The importer verifies both the submodule revision
and the archive checksum, copies the archive to the tracked
`conformance/blueprint-vectors.tar.gz`, and extracts it into the generated
working directory `conformance/testdata/eras`.

`conformance/blueprint-vectors.tar.gz` is byte-identical to the submodule copy
and is the only form of the corpus that reaches downstream modules: Go module
zips contain neither submodule contents nor ignored paths, and
`conformance/testdata/eras` is ignored. `conformance/embed.go` embeds the
archive and `ExtractEmbeddedTestdata` materializes it, so consumers do not need
the submodule.

| Field | Value |
| --- | --- |
| Source | [Cardano Blueprint conformance-test-vectors](https://github.com/cardano-scaling/cardano-blueprint/tree/main/src/ledger/conformance-test-vectors) |
| Blueprint revision | `0f0c17e1ca24b062c868d216ae50708fc19c83ab` |
| `vectors.tar.gz` SHA-256 | `574ff7a17857dfc1f0cf477f7eb9eba1c2a0f901453396a779de4b2392ef6863` |
| Archive size | 1,780,623 bytes |
| Extracted size | 19,920,523 bytes in 2,652 files |
| Ledger vector files | 2,574 |
| Protocol-parameter files | 78 |

Prepare or verify the corpus with:

```sh
make prepare-blueprint-testdata
```

Without an initialized submodule the target extracts the tracked archive after
verifying its checksum. With the submodule initialized it also verifies the
revision and refreshes the tracked archive, which is how the pin is bumped.

Path normalization is implemented twice: in
`scripts/update-blueprint-conformance.sh` for the working copy and in
`conformance/embed.go` for the embedded archive.
`TestEmbeddedErasMatchesWorkingCopy` compares the two trees file-for-file and
byte-for-byte so they cannot drift.

The archive stores one JSON record per transaction. The harness adapter decodes
the `cbor`, `oldLedgerState`, `newLedgerState`, `success`, and `testState`
fields without rewriting the encoded ledger or transaction bytes. Consensus,
wire-protocol, and synthetic rollback fixtures remain separate from this ledger
corpus.

Blueprint JSON omits the legacy event timeline, including execution epochs.
The adapter uses epoch 899 for this pinned corpus and carries the one known
timeline-derived exception in `conformance/vector.go`; update that metadata
from the legacy event envelope when changing the Blueprint pin.

The Blueprint corpus is a ledger-rule corpus, not a complete Cardano
conformance claim. Coverage and failures must be reported by era and rule
family; passing this package's mock backend does not prove a downstream ledger
implementation conforms.
