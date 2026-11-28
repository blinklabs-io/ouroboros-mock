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
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var metricSchemas = sync.OnceValues(func() ([]*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	schemas := make([]*jsonschema.Schema, len(resultFields))
	for i, fields := range resultFields {
		properties := map[string]any{"status": map[string]any{"const": "valid"}}
		for _, field := range fields {
			properties[field] = metricFieldType(i, field)
		}
		document := map[string]any{"type": "object", "properties": properties, "required": append([]string{"status"}, fields...)}
		location := fmt.Sprintf("https://benchmark.invalid/metric-%d.json", i)
		encoded, err := json.Marshal(document)
		if err != nil {
			return nil, err
		}
		normalized, err := jsonschema.UnmarshalJSON(bytes.NewReader(encoded))
		if err != nil {
			return nil, err
		}
		if err := compiler.AddResource(location, normalized); err != nil {
			return nil, err
		}
		schema, err := compiler.Compile(location)
		if err != nil {
			return nil, err
		}
		schemas[i] = schema
	}
	return schemas, nil
})

func metricFieldType(index int, field string) map[string]any {
	number := func() map[string]any { return map[string]any{"type": "number", "minimum": 0} }
	integer := func() map[string]any { return map[string]any{"type": "integer", "minimum": 0} }
	switch field {
	case "seconds":
		if index == 11 || index == 16 {
			return map[string]any{"type": "array", "items": number()}
		}
		return number()
	case "inside_live_bytes", "outside_live_bytes":
		return map[string]any{"type": "array", "items": integer()}
	case "p50_seconds", "p75_seconds":
		return map[string]any{"type": []string{"number", "null"}, "minimum": 0}
	case "announced_seconds", "sending_seconds", "adopted_seconds", "first_notice_seconds", "first_fetch_seconds", "cpu_percent_one_core", "cpu_seconds_per_processed":
		return number()
	case "ratio", "last_density":
		result := number()
		result["maximum"] = 1
		return result
	default:
		return integer()
	}
}

func validateMetricResult(index int, raw json.RawMessage) error {
	schemas, err := metricSchemas()
	if err != nil {
		return err
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	return schemas[index].Validate(value)
}
