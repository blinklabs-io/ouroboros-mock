// Copyright 2024 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package conversation

import (
	"fmt"
	"strings"

	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"

	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/handshake"
)

var protocolIds = map[string]uint{
	"handshake": handshake.ProtocolId,
}

var protocolMessageTypes = map[string]map[string]uint{
	"handshake": map[string]uint{
		"ProposeVersions": handshake.MessageTypeProposeVersions,
		"AcceptVersion":   handshake.MessageTypeAcceptVersion,
		"Refuse":          handshake.MessageTypeRefuse,
	},
}

var protocolInputFuncs = map[uint]func(bool, uint16, protocol.Message) ouroboros_mock.ConversationEntryInput{
	handshake.ProtocolId: ouroboros_mock.HandshakeInput,
}

var protocolMessages = map[string]map[string]any{
	"handshake": map[string]any{
		// TODO
	},
}

func splitProtocolMessage(input string) (string, string, error) {
	inputParts := strings.SplitN(input, `.`, 2)
	if len(inputParts) != 2 {
		return "", "", fmt.Errorf("malformed message name: %s", input)
	}
	proto, msg := inputParts[0], inputParts[1]
	if _, ok := protocolMessageTypes[proto]; !ok {
		return "", "", fmt.Errorf("unknown protocol: %s", proto)
	}
	return proto, msg, nil
}
