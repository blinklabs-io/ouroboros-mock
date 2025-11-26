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
    ouroboros_mock.ProtocolRoleClient,
    []ouroboros_mock.ConversationEntry{
        ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
        ouroboros_mock.ConversationEntryHandshakeNtCResponse,
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
