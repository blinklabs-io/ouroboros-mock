// Copyright 2025 Blink Labs Software
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

package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/handshake"
	"github.com/blinklabs-io/gouroboros/protocol/keepalive"
	mock "github.com/blinklabs-io/ouroboros-mock"
	"go.yaml.in/yaml/v3"
)

type configuration struct {
	Listener struct {
		Network yamlString `yaml:"network"`
		Address yamlString `yaml:"address"`
	} `yaml:"listener"`
	Entries []entryConfig `yaml:"entries"`
}
type entryConfig struct {
	Input  *messageConfig `yaml:"input"`
	Output *messageConfig `yaml:"output"`
	Sleep  *yamlString    `yaml:"sleep"`
	Close  *yamlBool      `yaml:"close"`
}
type messageConfig struct {
	Type         yamlString  `yaml:"type"`
	Mode         yamlString  `yaml:"mode"`
	Version      *yamlUint16 `yaml:"version"`
	NetworkMagic *yamlUint32 `yaml:"network-magic"`
	Cookie       *yamlUint16 `yaml:"cookie"`
}

func loadConfiguration(reader io.Reader) (configuration, []mock.ConversationEntry, error) {
	var cfg configuration
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return cfg, nil, errors.New("configuration is empty")
		}
		return cfg, nil, fmt.Errorf("decode configuration: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return cfg, nil, fmt.Errorf("decode trailing document: %w", err)
		}
		return cfg, nil, errors.New("configuration must contain exactly one YAML document")
	}
	if (cfg.Listener.Network != "tcp" && cfg.Listener.Network != "unix") ||
		cfg.Listener.Address == "" {
		return cfg, nil, errors.New("listener requires network tcp or unix and an address")
	}
	if len(cfg.Entries) == 0 {
		return cfg, nil, errors.New("conversation requires entries")
	}
	entries := make([]mock.ConversationEntry, 0, len(cfg.Entries))
	for i, raw := range cfg.Entries {
		entry, err := raw.build()
		if err != nil {
			return cfg, nil, fmt.Errorf("entry %d: %w", i+1, err)
		}
		if raw.Close != nil && i != len(cfg.Entries)-1 {
			return cfg, nil, fmt.Errorf("entry %d: close must be the final entry", i+1)
		}
		entries = append(entries, entry)
	}
	return cfg, entries, nil
}

func (e entryConfig) build() (mock.ConversationEntry, error) {
	count := 0
	for _, present := range []bool{e.Input != nil, e.Output != nil, e.Sleep != nil, e.Close != nil} {
		if present {
			count++
		}
	}
	if count != 1 {
		return nil, errors.New("exactly one of input, output, sleep, close is required")
	}
	if e.Close != nil {
		if !*e.Close {
			return nil, errors.New("close must be true")
		}
		return mock.ConversationEntryClose{}, nil
	}
	if e.Sleep != nil {
		duration, err := time.ParseDuration(string(*e.Sleep))
		if err != nil {
			return nil, fmt.Errorf("sleep: %w", err)
		}
		if duration < 0 {
			return nil, errors.New("sleep must be nonnegative")
		}
		return mock.ConversationEntrySleep{Duration: duration}, nil
	}
	if e.Input != nil {
		return e.Input.build(true)
	}
	return e.Output.build(false)
}

func (m messageConfig) build(input bool) (mock.ConversationEntry, error) {
	switch m.Type {
	case "handshake.propose_versions":
		if !input || m.Mode != "" || m.Version != nil || m.NetworkMagic != nil || m.Cookie != nil {
			return nil, errors.New("handshake.propose_versions requires input and only type")
		}
		return mock.ConversationEntryInput{
			ProtocolId:  handshake.ProtocolId,
			MessageType: handshake.MessageTypeProposeVersions,
		}, nil
	case "handshake.accept_version":
		if input || m.Version == nil || m.NetworkMagic == nil || m.Cookie != nil {
			return nil, errors.New(
				"handshake.accept_version requires output, mode, version, network-magic",
			)
		}
		var version uint16
		var data protocol.VersionData
		switch m.Mode {
		case "node-to-node":
			if *m.Version != 13 {
				return nil, errors.New("node-to-node demo supports version 13")
			}
			version = uint16(*m.Version)
			data = protocol.VersionDataNtN13andUp{
				VersionDataNtN11to12: protocol.VersionDataNtN11to12{
					CborNetworkMagic:                       uint32(*m.NetworkMagic),
					CborInitiatorAndResponderDiffusionMode: protocol.DiffusionModeInitiatorOnly,
					CborPeerSharing:                        protocol.PeerSharingModeNoPeerSharing,
					CborQuery:                              protocol.QueryModeDisabled,
				},
			}
		case "node-to-client":
			if *m.Version != 14 {
				return nil, errors.New("node-to-client demo supports version 14")
			}
			version = uint16(*m.Version) + protocol.ProtocolVersionNtCOffset
			data = protocol.VersionDataNtC9to14(*m.NetworkMagic)
		default:
			return nil, errors.New("handshake mode must be node-to-node or node-to-client")
		}
		return mock.ConversationEntryOutput{
			ProtocolId: handshake.ProtocolId,
			IsResponse: true,
			Messages:   []protocol.Message{handshake.NewMsgAcceptVersion(version, data)},
		}, nil
	case "keepalive.request", "keepalive.response":
		if m.Cookie == nil || m.Mode != "" || m.Version != nil || m.NetworkMagic != nil {
			return nil, errors.New("keepalive requires only type and cookie")
		}
		if m.Type == "keepalive.request" && input {
			return mock.ConversationEntryInput{
				ProtocolId:      keepalive.ProtocolId,
				Message:         keepalive.NewMsgKeepAlive(uint16(*m.Cookie)),
				MsgFromCborFunc: keepalive.NewMsgFromCbor,
			}, nil
		}
		if m.Type == "keepalive.response" && !input {
			return mock.ConversationEntryOutput{
				ProtocolId: keepalive.ProtocolId,
				IsResponse: true,
				Messages:   []protocol.Message{keepalive.NewMsgKeepAliveResponse(uint16(*m.Cookie))},
			}, nil
		}
		return nil, errors.New(
			"keepalive.request requires input; keepalive.response requires output",
		)
	default:
		return nil, fmt.Errorf("unsupported message type %q", m.Type)
	}
}

type (
	yamlUint16 uint16
	yamlUint32 uint32
	yamlBool   bool
	yamlString string
)

func (v *yamlUint16) UnmarshalYAML(node *yaml.Node) error {
	return decodeScalar(node, (*uint16)(v), "!!int")
}

func (v *yamlUint32) UnmarshalYAML(node *yaml.Node) error {
	return decodeScalar(node, (*uint32)(v), "!!int")
}

func (v *yamlBool) UnmarshalYAML(node *yaml.Node) error {
	return decodeScalar(node, (*bool)(v), "!!bool")
}

func (v *yamlString) UnmarshalYAML(node *yaml.Node) error {
	return decodeScalar(node, (*string)(v), "!!str")
}

// YAML decoding into Go integers otherwise truncates fractional values.
func decodeScalar(node *yaml.Node, value any, tag string) error {
	if node.Kind != yaml.ScalarNode || node.ShortTag() != tag {
		return fmt.Errorf("expected YAML %s scalar", tag)
	}
	return node.Decode(value)
}
