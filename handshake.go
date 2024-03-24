package ouroboros_mock

import (
	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/handshake"
)

func HandshakeInput(response bool, messageType uint16, message protocol.Message) ConversationEntryInput {
	return ConversationEntryInput{
		ProtocolId:      handshake.ProtocolId,
		MsgFromCborFunc: handshake.NewMsgFromCbor,
		IsResponse:      response,
		MessageType:     uint(messageType),
		Message:         message,
	}
}
