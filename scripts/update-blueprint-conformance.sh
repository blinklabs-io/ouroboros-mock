#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(
	cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd
)
BLUEPRINT_DIR="${ROOT_DIR}/fixtures/cardano-blueprint"
ARCHIVE="${BLUEPRINT_DIR}/src/ledger/conformance-test-vectors/vectors.tar.gz"
DEST_DIR="${ROOT_DIR}/conformance/testdata/eras"
EXPECTED_BLUEPRINT_REVISION="0f0c17e1ca24b062c868d216ae50708fc19c83ab"
EXPECTED_ARCHIVE_SHA256="574ff7a17857dfc1f0cf477f7eb9eba1c2a0f901453396a779de4b2392ef6863"

if [[ ! -e "${BLUEPRINT_DIR}/.git" ]]; then
	echo "Cardano Blueprint submodule is not initialized: ${BLUEPRINT_DIR}" >&2
	exit 1
fi

actual_revision=$(git -C "${BLUEPRINT_DIR}" rev-parse HEAD)
if [[ "${actual_revision}" != "${EXPECTED_BLUEPRINT_REVISION}" ]]; then
	echo "unexpected Cardano Blueprint revision: ${actual_revision}" >&2
	echo "expected: ${EXPECTED_BLUEPRINT_REVISION}" >&2
	exit 1
fi

if command -v shasum >/dev/null 2>&1; then
	actual_sha256=$(shasum -a 256 "${ARCHIVE}" | awk '{print $1}')
elif command -v sha256sum >/dev/null 2>&1; then
	actual_sha256=$(sha256sum "${ARCHIVE}" | awk '{print $1}')
else
	echo "neither shasum nor sha256sum is available" >&2
	exit 1
fi
if [[ "${actual_sha256}" != "${EXPECTED_ARCHIVE_SHA256}" ]]; then
	echo "unexpected Cardano Blueprint archive checksum: ${actual_sha256}" >&2
	echo "expected: ${EXPECTED_ARCHIVE_SHA256}" >&2
	exit 1
fi

rm -rf "${DEST_DIR}"
mkdir -p "${DEST_DIR}"
tar --strip-components=1 -xzf "${ARCHIVE}" -C "${DEST_DIR}"

# Normalize paths only; vector bytes and protocol-parameter contents remain
# unchanged.
find "${DEST_DIR}" -depth -execdir bash -c '
	name="${1#./}"
	safe=$(printf "%s" "${name}" | tr -c "[:alnum:]._-" "_" | sed "s/__*/_/g; s/_$//")
	if [[ -z "${safe}" || ( "${name}" != "${safe}" && -e "${safe}" ) ]]; then
		echo "refusing unsafe or colliding normalized path: ${name} -> ${safe}" >&2
		exit 1
	fi
	if [[ "${name}" != "${safe}" ]]; then
		mv -- "${name}" "${safe}"
	fi
' _ {} \;

vector_count=$(find "${DEST_DIR}" -type f ! -path '*/pparams-by-hash/*' | wc -l)
pparams_count=$(find "${DEST_DIR}/conway/impl/dump/pparams-by-hash" -type f | wc -l)
printf 'Imported Cardano Blueprint revision %s (%s)\n' "${EXPECTED_BLUEPRINT_REVISION}" "${EXPECTED_ARCHIVE_SHA256}"
printf 'Ledger vectors: %s; protocol-parameter files: %s\n' "${vector_count}" "${pparams_count}"
