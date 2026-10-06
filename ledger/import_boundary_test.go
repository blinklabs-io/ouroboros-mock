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

package ledger_test

import (
	"os/exec"
	"strings"
	"testing"
)

// gouroboros's in-package ledger/dijkstra tests import this package, so any
// dependency on gouroboros/ledger/dijkstra here is an import cycle that stops
// those tests from building. Dijkstra-typed builders belong in fixtures.
func TestLedgerPackageDoesNotDependOnGouroborosDijkstra(t *testing.T) {
	out, err := exec.Command(
		"go", "list", "-deps", "github.com/blinklabs-io/ouroboros-mock/ledger",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps: %s\n%s", err, out)
	}
	const forbidden = "github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	for _, dep := range strings.Fields(string(out)) {
		if dep == forbidden {
			t.Fatalf("ouroboros-mock/ledger depends on %s", forbidden)
		}
	}
}
