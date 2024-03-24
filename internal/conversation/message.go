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
	"errors"
	"fmt"

	"github.com/blinklabs-io/gouroboros/protocol"
	"gopkg.in/yaml.v3"
)

type Message struct {
	protocolId uint
	message    protocol.Message
}

func (m *Message) UnmarshalYAML(value *yaml.Node) error {
	var tmpData = map[string]*yaml.Node{}
	if err := value.Decode(&tmpData); err != nil {
		return err
	}
	if len(tmpData) == 0 {
		return errors.New("expected message type")
	}
	if len(tmpData) > 1 {
		return errors.New("found more than one message type")
	}
	var protoMsg string
	var msgNode *yaml.Node
	for k, v := range tmpData {
		protoMsg = k
		msgNode = v
	}
	proto, msg, err := splitProtocolMessage(protoMsg)
	if err != nil {
		return err
	}
	if _, ok := protocolMessageTypes[proto][msg]; !ok {
		return fmt.Errorf("unknown message type: %s", protoMsg)
	}
	m.protocolId = protocolIds[proto]
	// TODO: populate message
	return nil
}
