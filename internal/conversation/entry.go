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
	"time"

	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"

	"gopkg.in/yaml.v3"
)

type Entry struct {
	Input  *EntryInput  `yaml:"input"`
	Output *EntryOutput `yaml:"output"`
	Sleep  *EntrySleep  `yaml:"sleep"`
	Close  *EntryClose  `yaml:"close"`
}

type EntryInput struct {
	Response    bool     `yaml:"response"`
	MessageType string   `yaml:"messageType"`
	Message     *Message `yaml:"message"`
	protocolId  uint
	messageType uint
}

func (i *EntryInput) UnmarshalYAML(value *yaml.Node) error {
	var tmpData struct {
		Response    bool     `yaml:"response"`
		MessageType string   `yaml:"messageType"`
		Message     *Message `yaml:"message"`
	}
	if err := value.Decode(&tmpData); err != nil {
		return err
	}
	if tmpData.Message != nil {
		// TODO
	} else if tmpData.MessageType != "" {
		proto, msg, err := splitProtocolMessage(tmpData.MessageType)
		if err != nil {
			return err
		}
		if msgType, ok := protocolMessageTypes[proto][msg]; !ok {
			return fmt.Errorf("unknown message type: %s", tmpData.MessageType)
		} else {
			i.messageType = msgType
			i.protocolId = protocolIds[proto]
		}
	} else {
		return errors.New("no message or message type found")
	}
	return nil
}

func (i EntryInput) ToConversationEntry() ouroboros_mock.ConversationEntry {
	inputFunc := protocolInputFuncs[i.protocolId]
	return inputFunc(i.Response, uint16(i.messageType), i.Message)
}

type EntryOutput struct {
	Response bool     `yaml:"response"`
	Message  *Message `yaml:"message"`
}

func (o *EntryOutput) UnmarshalYAML(value *yaml.Node) error {
	// TODO
	return nil
}

type EntrySleep struct {
	Duration time.Duration `yaml:"duration"`
}

func (s EntrySleep) ToConversationEntry() ouroboros_mock.ConversationEntry {
	return ouroboros_mock.ConversationEntrySleep{
		Duration: s.Duration,
	}
}

type EntryClose struct{}

func (c EntryClose) ToConversationEntry() ouroboros_mock.ConversationEntry {
	return ouroboros_mock.ConversationEntryClose{}
}
