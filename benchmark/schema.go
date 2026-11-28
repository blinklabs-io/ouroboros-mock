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
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
)

// Resolve references only from declared, checksummed schema artifacts. Loading a
// bundle must never fetch a remote schema or depend on undeclared adjacent files.
func (b *Bundle) validateSchemaReferences(ids ...string) error {
	byPath := map[string]string{}
	for id, a := range b.artifacts {
		if a.Kind == "config" {
			byPath[a.Path] = id
		}
	}
	documents := map[string]any{}
	seen := map[string]bool{}
	var visit func(string, any) error
	load := func(id string) (any, error) {
		if doc, ok := documents[id]; ok {
			return doc, nil
		}
		raw, err := b.Read(id)
		if err != nil {
			return nil, err
		}
		if err = uniqueJSON(raw); err != nil {
			return nil, err
		}
		var doc any
		if err = json.Unmarshal(raw, &doc); err != nil {
			return nil, err
		}
		documents[id] = doc
		return doc, nil
	}
	visit = func(id string, node any) error {
		switch v := node.(type) {
		case map[string]any:
			if raw, ok := v["$ref"]; ok {
				ref, ok := raw.(string)
				if !ok {
					return fmt.Errorf("schema %s has non-string reference", id)
				}
				key := id + "#" + ref
				if !seen[key] {
					seen[key] = true
					uri, err := url.Parse(ref)
					if err != nil {
						return err
					}
					if uri.IsAbs() || uri.Host != "" || uri.RawQuery != "" || strings.HasPrefix(uri.Path, "/") {
						return fmt.Errorf("schema reference is not corpus-local: %s", ref)
					}
					targetID := id
					if uri.Path != "" {
						targetID = byPath[path.Join(path.Dir(b.artifacts[id].Path), uri.Path)]
					}
					if targetID == "" {
						return fmt.Errorf("schema %s has undeclared reference %s", id, ref)
					}
					target, err := load(targetID)
					if err != nil {
						return err
					}
					if uri.Fragment != "" {
						if !strings.HasPrefix(uri.Fragment, "/") {
							return fmt.Errorf("unsupported schema anchor %s", ref)
						}
						for _, part := range strings.Split(uri.Fragment[1:], "/") {
							part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
							switch container := target.(type) {
							case map[string]any:
								target = container[part]
							case []any:
								index, err := strconv.Atoi(part)
								if err != nil || index < 0 || index >= len(container) {
									return fmt.Errorf("unresolved schema reference %s", ref)
								}
								target = container[index]
							default:
								return fmt.Errorf("unresolved schema reference %s", ref)
							}
						}
					}
					switch target.(type) {
					case map[string]any, bool:
					default:
						return fmt.Errorf("unresolved schema reference %s", ref)
					}
					if err = visit(targetID, target); err != nil {
						return err
					}
				}
			}
			for _, child := range v {
				if err := visit(id, child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range v {
				if err := visit(id, child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, id := range ids {
		doc, err := load(id)
		if err != nil {
			return err
		}
		if err = visit(id, doc); err != nil {
			return err
		}
	}
	return nil
}
