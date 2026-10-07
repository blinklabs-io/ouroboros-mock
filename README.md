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

### Certificate Fixtures

The `certificates` package builds every certificate type through Conway.
`NewMoveInstantaneousRewards(source)` uses the Shelley wire pot IDs: 0 for
reserves and 1 for treasury. Select a signed credential reward map with
`WithRewards`, `WithRewardKey`, or `WithRewardScript`, or a coin transfer with
`WithOtherPot`. Reward maps and opposite-pot transfers are mutually exclusive;
empty maps and zero transfers remain distinct. The MIR builder copies its
inputs and returns independent certificate values.

`NewGenerator(seed)` draws certificate types and inputs deterministically
within a Go release, including both MIR targets. Pool reward accounts default
to testnet; `WithNetwork` selects their network ID. `StakeLifecycle`,
`PoolLifecycle`, and `DRepLifecycle` build related registration, delegation or
update, and retirement or deregistration sequences.

## Command line listener

Run `ouroboros-mock demo.yaml` to listen for one connection and execute the
conversation. The process exits after the conversation finishes. Interrupting
the process cancels the listener and active conversation.
The first complete multiplexed message must arrive within the 10-second
node-to-node handshake proposal timeout.

```yaml
listener:
  network: tcp
  address: 127.0.0.1:3001
entries:
  - input:
      type: handshake.propose_versions
  - output:
      type: handshake.accept_version
      mode: node-to-node
      version: 13
      network-magic: 42
  - input:
      type: keepalive.request
      cookie: 123
  - output:
      type: keepalive.response
      cookie: 123
  - close: true
```

The listener accepts `tcp` and `unix` networks. A Unix listener requires an
unused socket path; existing files are preserved, and the socket created by
the listener is removed when it closes.

Each entry contains exactly one of `input`, `output`, `sleep` (a nonnegative Go
duration such as `100ms`), or `close: true`. Close must be the final entry.
Handshake proposals match the incoming message type. Handshake acceptance
requires `mode`, `version`, and `network-magic`; the supported demo versions
are node-to-node 13 and node-to-client 14. Keepalive requests match `cookie`
exactly and responses send the configured cookie. Unknown fields, unsupported
messages, multiple YAML documents, and ambiguous entries are rejected before
opening the listener. Numeric settings require unsigned YAML integers, string
settings require YAML strings, and close requires the literal boolean `true`.
