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
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type Conversation struct {
	Name    string  `yaml:"name"`
	Entries []Entry `yaml:"entries"`
}

func NewFromFile(path string) (Conversation, error) {
	f, err := os.Open(path)
	if err != nil {
		return Conversation{}, err
	}
	defer f.Close()
	return NewFromReader(f)
}

func NewFromReader(r io.Reader) (Conversation, error) {
	var ret Conversation
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(&ret); err != nil {
		return Conversation{}, err
	}
	return ret, nil
}
