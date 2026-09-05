#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(
	cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd
)
BLUEPRINT_DIR="${ROOT_DIR}/fixtures/cardano-blueprint"
SUBMODULE_ARCHIVE="${BLUEPRINT_DIR}/src/ledger/conformance-test-vectors/vectors.tar.gz"
# Tracked copy of the archive. Go module zips carry neither submodule contents
# nor ignored paths, so this file is what reaches downstream consumers.
TRACKED_ARCHIVE="${ROOT_DIR}/conformance/blueprint-vectors.tar.gz"
DEST_DIR="${ROOT_DIR}/conformance/testdata/eras"
EXPECTED_BLUEPRINT_REVISION="0f0c17e1ca24b062c868d216ae50708fc19c83ab"
EXPECTED_ARCHIVE_SHA256="574ff7a17857dfc1f0cf477f7eb9eba1c2a0f901453396a779de4b2392ef6863"

sha256_of() {
	if command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "${1}" | awk '{print $1}'
	elif command -v sha256sum >/dev/null 2>&1; then
		sha256sum "${1}" | awk '{print $1}'
	else
		echo "neither shasum nor sha256sum is available" >&2
		exit 1
	fi
}

verify_archive() {
	local archive="${1}"
	local actual_sha256
	actual_sha256=$(sha256_of "${archive}")
	if [[ "${actual_sha256}" != "${EXPECTED_ARCHIVE_SHA256}" ]]; then
		echo "unexpected Cardano Blueprint archive checksum for ${archive}: ${actual_sha256}" >&2
		echo "expected: ${EXPECTED_ARCHIVE_SHA256}" >&2
		exit 1
	fi
}

# Refresh the tracked archive from the submodule when it is available. This is
# the path taken when bumping the Blueprint pin; the submodule revision and the
# upstream archive checksum are both verified before the copy.
if [[ -e "${BLUEPRINT_DIR}/.git" ]]; then
	actual_revision=$(git -C "${BLUEPRINT_DIR}" rev-parse HEAD)
	if [[ "${actual_revision}" != "${EXPECTED_BLUEPRINT_REVISION}" ]]; then
		echo "unexpected Cardano Blueprint revision: ${actual_revision}" >&2
		echo "expected: ${EXPECTED_BLUEPRINT_REVISION}" >&2
		exit 1
	fi
	verify_archive "${SUBMODULE_ARCHIVE}"
	cp "${SUBMODULE_ARCHIVE}" "${TRACKED_ARCHIVE}"
elif [[ ! -f "${TRACKED_ARCHIVE}" ]]; then
	echo "Cardano Blueprint submodule is not initialized and ${TRACKED_ARCHIVE} is missing" >&2
	exit 1
fi

# The tracked archive is the extraction source in every case, so a checkout
# without submodules produces the same corpus as one with them.
verify_archive "${TRACKED_ARCHIVE}"

rm -rf "${DEST_DIR}"
mkdir -p "${DEST_DIR}"
tar --strip-components=1 -xzf "${TRACKED_ARCHIVE}" -C "${DEST_DIR}"

# Normalize paths only; vector bytes and protocol-parameter contents remain
# unchanged. conformance/embed.go applies the same normalization when it
# extracts the embedded archive, and TestEmbeddedErasMatchesWorkingCopy
# asserts the two agree.
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
