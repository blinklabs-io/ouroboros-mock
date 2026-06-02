#!/usr/bin/env bash

# Copyright 2026 Blink Labs Software
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Thin dispatcher: resolves scenarios/<name>/run.sh and runs it. Each
# scenario owns its own orchestration shape (stack lifecycle, number of
# sidecar invocations, optional composer step). See README.md for the
# rationale and how to add a new scenario.
#
# run.sh validates the composed vector against its scenario's intended shape
# before committing, so a forge that drifts out of shape fails WITHOUT
# overwriting the golden. Because the forge is nondeterministic, this dispatcher
# can re-roll: set CAPTURE_RETRIES=N (default 1) to retry run.sh up to N times
# until it produces a shape-valid vector. Each run.sh invocation tears down with
# `down -v` on exit, so every retry is a fresh forge. capture-all.sh sets a
# generous CAPTURE_RETRIES so a one-shot regeneration re-rolls the lottery
# scenarios (slot-battle tie, exceeds-k deep incumbent) instead of failing.
#
# Usage:
#   ./capture-scenario.sh <scenario-name> [extra args passed to run.sh]
#   CAPTURE_RETRIES=30 ./capture-scenario.sh <scenario-name> [extra args]

set -euo pipefail

SCENARIO="${1:-}"
if [[ -z "${SCENARIO}" ]]; then
    echo "usage: $0 <scenario-name> [extra args]" >&2
    echo "" >&2
    echo "Available scenarios:" >&2
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [[ -d "${SCRIPT_DIR}/scenarios" ]]; then
        for dir in "${SCRIPT_DIR}/scenarios"/*/; do
            [[ -d "${dir}" ]] || continue
            echo "  - $(basename "${dir}")" >&2
        done
    fi
    exit 1
fi
shift

# Reject anything that could escape the scenarios/ subtree before we
# resolve a path or exec under it. First char must be alphanumeric so
# names like ".." or "-foo" can't slip through the character class.
if [[ ! "${SCENARIO}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
    echo "capture-scenario: invalid scenario name: ${SCENARIO}" >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="${SCRIPT_DIR}/scenarios/${SCENARIO}"

if [[ ! -d "${SCENARIO_DIR}" ]]; then
    echo "capture-scenario: scenario not found: ${SCENARIO_DIR}" >&2
    exit 1
fi
if [[ ! -x "${SCENARIO_DIR}/run.sh" ]]; then
    echo "capture-scenario: ${SCENARIO_DIR}/run.sh missing or not executable" >&2
    exit 1
fi

# Validate CAPTURE_RETRIES (positive integer) so a typo can't silently
# disable retries or loop forever.
RETRIES="${CAPTURE_RETRIES:-1}"
if [[ ! "${RETRIES}" =~ ^[1-9][0-9]*$ ]]; then
    echo "capture-scenario: CAPTURE_RETRIES must be a positive integer, got '${RETRIES}'" >&2
    exit 2
fi

attempt=1
while true; do
    if (( RETRIES > 1 )); then
        echo "capture-scenario: ${SCENARIO} attempt ${attempt}/${RETRIES}" >&2
    fi
    # Capture run.sh's exit code in the else branch: `rc=$?` after the `if`
    # compound would read the if-statement's status (0 when the condition
    # fails), so exhausted retries could exit 0 and falsely report success.
    if "${SCENARIO_DIR}/run.sh" "$@"; then
        exit 0
    else
        rc=$?
    fi
    if (( attempt >= RETRIES )); then
        if (( RETRIES > 1 )); then
            echo "capture-scenario: ${SCENARIO} did not produce a valid vector in ${RETRIES} attempt(s)" >&2
        fi
        exit "${rc}"
    fi
    echo "capture-scenario: ${SCENARIO} attempt ${attempt} failed (rc=${rc}); re-rolling the forge" >&2
    attempt=$(( attempt + 1 ))
done
