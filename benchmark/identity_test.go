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
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildStatusDistinguishesUnknownFromReportedClean(t *testing.T) {
	for _, diagnostic := range []string{"missing", "false", "true", "unknown"} {
		t.Run(diagnostic, func(t *testing.T) {
			info := &debug.BuildInfo{}
			if diagnostic != "missing" {
				info.Settings = []debug.BuildSetting{{Key: "vcs.modified", Value: diagnostic}}
			}
			var identity Identity
			addBuildInfo(&identity, info)
			raw, err := json.Marshal(identity)
			require.NoError(t, err)
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &fields))
			if diagnostic == "true" || diagnostic == "false" {
				require.JSONEq(t, diagnostic, string(fields["go_vcs_modified"]))
			} else {
				require.NotContains(t, fields, "go_vcs_modified")
			}
		})
	}
}
