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

# Bulk wrapper around capture-scenario.sh: runs every scenario under
# scenarios/ and writes each produced vector to
# testdata/captured/<scenario-name>.json.
#
# Continues past per-scenario failures and prints a summary at the
# end. Failures don't short-circuit because a single scenario going
# wrong (image build issue, flaky cardano-node startup, etc.) shouldn't
# block regeneration of the other scenarios' goldens.
#
# By default --skip-golden is forwarded to every scenario's run.sh so
# fork_and_select_v1 doesn't abort on the structural-tolerance diff
# against the existing committed golden — the whole point of this
# wrapper is to regenerate the goldens, after all. Scenarios that
# don't have goldens accept --skip-golden as a no-op for uniformity.
#
# Usage:
#   ./capture-all.sh                         # regenerate every scenario
#   ./capture-all.sh --keep-going            # default behavior; explicit
#   ./capture-all.sh --fail-fast             # stop on first scenario failure
#   ./capture-all.sh --only intersect_origin_one_rollforward
#                                            # regenerate one scenario by name

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIOS_DIR="${SCRIPT_DIR}/scenarios"
CAPTURED_DIR="${SCRIPT_DIR}/testdata/captured"
DISPATCHER="${SCRIPT_DIR}/capture-scenario.sh"

FAIL_FAST=false
ONLY=""
# Each scenario's run.sh validates the composed vector against its intended
# shape before committing, so a drifted forge fails without overwriting the
# golden. The forge is nondeterministic — the slot-battle tie and exceeds-k
# deep-incumbent shapes only land on a fraction of runs — so retry generously
# by default; a deterministic scenario succeeds on attempt 1 at no extra cost.
RETRIES="${CAPTURE_RETRIES:-30}"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --fail-fast)   FAIL_FAST=true; shift;;
        --keep-going)  FAIL_FAST=false; shift;;
        --retries)
            if [[ $# -lt 2 || ! "${2:-}" =~ ^[1-9][0-9]*$ ]]; then
                echo "--retries requires a positive integer" >&2
                exit 2
            fi
            RETRIES="$2"
            shift 2
            ;;
        --only)
            if [[ $# -lt 2 || "${2:-}" == --* ]]; then
                echo "--only requires a scenario name" >&2
                exit 2
            fi
            ONLY="$2"
            shift 2
            ;;
        -h|--help)
            cat <<USAGE
usage: $0 [--fail-fast | --keep-going] [--retries N] [--only <scenario>]

Runs every scenarios/<name>/ through capture-scenario.sh and writes
each produced vector to testdata/captured/<name>.json. Each run.sh
validates the vector's shape before committing, so a drift fails
without overwriting the golden; the forge is re-rolled up to N times.

  --fail-fast       Stop on the first scenario failure.
  --keep-going      Continue past failures (default).
  --retries N       Re-roll each scenario's forge up to N times until a
                    shape-valid vector is produced (default 30; or set
                    CAPTURE_RETRIES).
  --only <name>     Run only the named scenario (still writes to
                    testdata/captured/<name>.json).
USAGE
            exit 0;;
        *)
            echo "unknown argument: $1" >&2
            exit 2;;
    esac
done

log()  { echo "[capture-all] $*"; }
warn() { echo "[capture-all] WARNING: $*" >&2; }
die()  { echo "[capture-all] ERROR: $*" >&2; exit 1; }

[[ -x "${DISPATCHER}" ]] || die "dispatcher not found or not executable: ${DISPATCHER}"
[[ -d "${SCENARIOS_DIR}" ]] || die "scenarios dir not found: ${SCENARIOS_DIR}"

mkdir -p "${CAPTURED_DIR}"

# Collect scenarios — alphabetically for stable output ordering, so
# the summary at the end is predictable.
SCENARIOS=()
for dir in "${SCENARIOS_DIR}"/*/; do
    [[ -d "${dir}" ]] || continue
    name="$(basename "${dir}")"
    if [[ -n "${ONLY}" && "${name}" != "${ONLY}" ]]; then
        continue
    fi
    SCENARIOS+=("${name}")
done

if (( ${#SCENARIOS[@]} == 0 )); then
    if [[ -n "${ONLY}" ]]; then
        die "no scenario named '${ONLY}' found under ${SCENARIOS_DIR}"
    fi
    die "no scenarios found under ${SCENARIOS_DIR}"
fi

log "regenerating ${#SCENARIOS[@]} scenario(s) into ${CAPTURED_DIR} (up to ${RETRIES} forge attempt(s) each)"

PASSED=()
FAILED=()
for scenario in "${SCENARIOS[@]}"; do
    out_path="${CAPTURED_DIR}/${scenario}.json"
    log "--- ${scenario} -> ${out_path} ---"
    if CAPTURE_RETRIES="${RETRIES}" "${DISPATCHER}" "${scenario}" -out "${out_path}" --skip-golden; then
        PASSED+=("${scenario}")
        log "${scenario}: OK"
    else
        FAILED+=("${scenario}")
        warn "${scenario}: FAILED"
        if [[ "${FAIL_FAST}" == "true" ]]; then
            warn "--fail-fast set; stopping"
            break
        fi
    fi
done

echo
log "summary:"
for s in "${PASSED[@]}"; do
    echo "  PASS  ${s}"
done
for s in "${FAILED[@]}"; do
    echo "  FAIL  ${s}"
done

if (( ${#FAILED[@]} > 0 )); then
    exit 1
fi
