// Copyright 2026 Blink Labs Software
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

package benchmark

import (
	"bytes"
	"fmt"
	"net/url"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type corpusSchemas struct {
	bundle    *Bundle
	artifacts map[string]string
}

func schemaURL(path string) string {
	return (&url.URL{Scheme: "https", Host: "benchmark.invalid", Path: "/" + path}).String()
}

// Load never accesses the network or undeclared adjacent files.
func (l corpusSchemas) Load(location string) (any, error) {
	id, ok := l.artifacts[location]
	if !ok {
		return nil, fmt.Errorf("undeclared schema resource %s", location)
	}
	raw, err := l.bundle.Read(id)
	if err != nil {
		return nil, err
	}
	if err = uniqueJSON(raw); err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(raw))
}

func (b *Bundle) prepareCoverage() (Coverage, error) {
	var c Coverage
	if err := b.require([]string{b.manifest.Coverage}, "coverage"); err != nil {
		return c, err
	}
	raw, err := b.Read(b.manifest.Coverage)
	if err != nil {
		return c, err
	}
	if err = decodeJSON(raw, &c); err != nil {
		return c, err
	}
	if err = b.require([]string{c.Schema, c.Envelope}, "config"); err != nil {
		return c, err
	}
	if err = b.require([]string{c.Examples}, "golden"); err != nil {
		return c, err
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	loader := corpusSchemas{bundle: b, artifacts: map[string]string{}}
	for id, a := range b.artifacts {
		if a.Kind == "config" {
			loader.artifacts[schemaURL(a.Path)] = id
		}
	}
	compiler.UseLoader(loader)
	b.schemas = map[string]*jsonschema.Schema{}
	for _, id := range []string{c.Schema, c.Envelope} {
		schema, err := compiler.Compile(schemaURL(b.artifacts[id].Path))
		if err != nil {
			return c, fmt.Errorf("invalid or unresolved schema reference: %w", err)
		}
		b.schemas[id] = schema
	}
	return c, nil
}
