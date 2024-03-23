package ouroboros_mock

import (
	"testing"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"go.uber.org/goleak"
)

func TestNewConnection_ErrorHandling(t *testing.T) {
	// Create a new Connection with a mock error scenario
	defer goleak.VerifyNone(t)
	mockConn := NewConnection(
		ProtocolRoleClient,
		[]ConversationEntry{
			ConversationEntryHandshakeRequestGeneric,
			ConversationEntryHandshakeNtCResponse,
			{
				Type: 999, // Simulate an unknown entry type
			},
		},
	)

	oConn, err := ouroboros.New(
		ouroboros.WithConnection(mockConn),
		ouroboros.WithNetworkMagic(MockNetworkMagic),
	)
	if err != nil {
		t.Fatalf("unexpected error when creating Ouroboros object: %s", err)
	}

	// Close Ouroboros connection
	if err := oConn.Close(); err != nil {
		t.Fatalf("unexpected error when closing Ouroboros object: %s", err)
	}

	c, ok := mockConn.(*Connection)
	if !ok {
		t.Fatalf("Failed to type assert conn to *Connection")
	}

	select {
	case err := <-c.errorChan:
		if err == nil {
			t.Fatal("Expected an error from errorChan, got nil")
		}
		// Check the error message
		expectedErrMsg := "unknown conversation entry type: 999: ouroboros_mock.ConversationEntry{Type:999, ProtocolId:0x0, IsResponse:false, OutputMessages:[]protocol.Message(nil), InputMessage:protocol.Message(nil), InputMessageType:0x0, MsgFromCborFunc:(protocol.MessageFromCborFunc)(nil), Duration:0}"
		if err.Error() != expectedErrMsg {
			t.Fatalf("unexpected error message: expected %s, got %s", expectedErrMsg, err.Error())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for error from errorChan")
	}

}
