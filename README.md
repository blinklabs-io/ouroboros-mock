# ouroboros-mock
Go library and CLI framework for mocking Ouroboros connections

## Features

- Mock Ouroboros protocol conversations for testing
- Support for positive and negative test cases
- Easy-to-use conversation entry API

## Usage

### Basic Conversation

```go
mockConn := ouroboros_mock.NewConnection(
    ouroboros_mock.ProtocolRoleServer, // Mock acts as server
    []ouroboros_mock.ConversationEntry{
        ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
        ouroboros_mock.ConversationEntryHandshakeNtCResponse,
    },
)
```
### Client Handshake Conversation

When mocking a client that initiates the handshake:

```go
mockConn := ouroboros_mock.NewConnection(
    ouroboros_mock.ProtocolRoleClient, // Mock acts as client
    []ouroboros_mock.ConversationEntry{
        ouroboros_mock.ConversationEntryHandshakeRequestOutput,    // Mock sends ProposeVersions
        ouroboros_mock.ConversationEntryHandshakeNtCResponseInput, // Mock expects AcceptVersion for NtC
    },
)
```

For NtN protocol:

```go
mockConn := ouroboros_mock.NewConnection(
    ouroboros_mock.ProtocolRoleClient, // Mock acts as client
    []ouroboros_mock.ConversationEntry{
        ouroboros_mock.ConversationEntryHandshakeRequestOutput,    // Mock sends ProposeVersions
        ouroboros_mock.ConversationEntryHandshakeNtNResponseInput, // Mock expects AcceptVersion for NtN
    },
)
```

### Protocol message and scenario builders

Message builders describe a server's expected client inputs and server outputs.
Pass `ProtocolRoleClient` to `NewConnection` when the attached gouroboros peer
is the client. Add the handshake entries before a protocol scenario.

```go
start := ouroboros_mock.NewPoint(startSlot, startHash)
end := ouroboros_mock.NewPoint(endSlot, endHash)
entries := []ouroboros_mock.ConversationEntry{
    ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
    ouroboros_mock.ConversationEntryHandshakeNtNResponse,
}
scenario, err := ouroboros_mock.BlockFetchScenario(start, end, blockTypes, blocks)
if err != nil {
    panic(err)
}
entries = append(entries, scenario...)
mockConn := ouroboros_mock.NewConnection(ouroboros_mock.ProtocolRoleClient, entries)
```

`blocks` contains raw ledger block CBOR and `blockTypes` contains each block's
wire block type, as returned by the block's `Type()` method rather than its
ledger era ID. The builder encodes the wrapper required by BlockFetch. An
empty block slice produces `NoBlocks`; otherwise the scenario emits
`StartBatch`, each `Block`, and `BatchDone`. The client finishes with
`ClientDone`. `BlockFetchBlock` retains its existing complete single-block batch
behavior; `BlockFetchBlockResponse` wraps the supplied raw block and emits one response
within an open batch. The existing `BlockFetchBlock` takes an already encoded
block wrapper.

| Flow | Scenario builder |
| --- | --- |
| Forward headers in NtN | `ChainSyncForwardScenarioNtN` |
| Forward blocks in NtC | `ChainSyncForwardScenarioNtC` |
| Rollback in either mode | `ChainSyncRollbackScenario` |
| Intersection found or not found | `ChainSyncIntersectionScenario` |
| Block range or no blocks | `BlockFetchScenario` |
| Transaction IDs, bodies, and acknowledgement | `TxSubmissionScenario` |

These finite scenarios end with the client's protocol termination message and
do not close the connection. `ChainSyncScenario` retains its existing NtN
behavior with a final `RequestNext` awaiting another response. Intersection
success requires a point offered by the client. Transaction ID and body slices
must describe matching transactions in the same order and have equal lengths;
request counts cannot exceed the protocol's `uint16` limit. The final blocking
transaction-ID request permits the client to send `Done`.

Individual builders support ChainSync await/intersection responses, BlockFetch
batch boundaries, LocalTxMonitor acquisition and replies, LocalStateQuery
acquisition targets and results, PeerSharing, and parameterized KeepAlive
cookies. `LocalStateQueryResult` accepts a CBOR-encodable result value;
`LocalStateQueryQuery` preserves the typed query representation required by the
mock's input comparison. Peers and captured transactions retain their protocol
encoding, and callers supply valid ledger block and transaction content.

### Negative Test Cases

To test scenarios where errors are expected, set the `ExpectedError` field on conversation entries:

```go
mockConn := ouroboros_mock.NewConnection(
    ouroboros_mock.ProtocolRoleClient,
    []ouroboros_mock.ConversationEntry{
        ouroboros_mock.ConversationEntryInput{
            ProtocolId:    999, // Invalid protocol ID
            ExpectedError: "input message protocol ID did not match expected value: expected 999, got 0",
        },
    },
)
```

If the entry produces an error matching the `ExpectedError`, the conversation continues without failure. If the error does not match or no error occurs when expected, the mock will report an error.
