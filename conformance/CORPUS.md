# Conformance corpus

The ledger corpus is imported from the pinned `cardano-blueprint` submodule at
`fixtures/cardano-blueprint`. The importer verifies both the submodule revision
and the archive checksum before replacing the generated working directory
`conformance/testdata/eras`.

| Field | Value |
| --- | --- |
| Source | [Cardano Blueprint conformance-test-vectors](https://github.com/cardano-scaling/cardano-blueprint/tree/main/src/ledger/conformance-test-vectors) |
| Blueprint revision | `0f0c17e1ca24b062c868d216ae50708fc19c83ab` |
| `vectors.tar.gz` SHA-256 | `574ff7a17857dfc1f0cf477f7eb9eba1c2a0f901453396a779de4b2392ef6863` |
| Ledger vector files | 2,574 |
| Protocol-parameter files | 78 |

Prepare or verify the corpus with:

```sh
make prepare-blueprint-testdata
```

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
