#!/usr/bin/env bash

set -euo pipefail

readonly repo_root="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
readonly workflow_dir="${repo_root}/.github/workflows"
readonly fixture_script="${repo_root}/scripts/update-upstream-fixtures.sh"
readonly action_pattern='^[[:space:]]*(-[[:space:]]+)?uses:[[:space:]]+[^@]+@([0-9a-f]{40})([[:space:]]+#.*)?$'

while IFS= read -r action_line; do
	if [[ ! "${action_line}" =~ ${action_pattern} ]]; then
		echo "workflow action is not pinned to an immutable commit: ${action_line}" >&2
		exit 1
	fi
done < <(find "${workflow_dir}" -type f -print0 | xargs -0 grep -hE '^[[:space:]]*(-[[:space:]]+)?uses:')

readonly expected_revisions=(
	OUROBOROS_CONSENSUS_REVISION
	CARDANO_LEDGER_REVISION
	CARDANO_API_REVISION
	CARDANO_NODE_REVISION
)

for revision_name in "${expected_revisions[@]}"; do
	revision_line=$(grep -F "${revision_name}=" "${fixture_script}")
	revision_prefix="${revision_name}=\${${revision_name}:-"
	revision_value=${revision_line#"${revision_prefix}"}
	revision_value=${revision_value%\}}
	if [[ "${revision_line}" != "${revision_prefix}"*\} || ! "${revision_value}" =~ ^[0-9a-f]{40}$ ]]; then
		echo "fixture script is missing a fixed revision for ${revision_name}" >&2
		exit 1
	fi
done

if grep -Eq 'archive/(main|master)(\.tar\.gz)?|/(main|master)/' "${fixture_script}"; then
	echo "fixture script contains a mutable upstream reference" >&2
	exit 1
fi

bash -n "${fixture_script}"
echo "All workflow actions and upstream fixture sources are immutably pinned."
