#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(
	cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd
)
DEST_DIR="${ROOT_DIR}/fixtures/upstream"
WORK_DIR=$(mktemp -d "${TMPDIR:-/tmp}/ouroboros-mock-fixtures.XXXXXX")

OUROBOROS_CONSENSUS_REVISION=${OUROBOROS_CONSENSUS_REVISION:-54765ad9f916793ebf817b74def1ab4e8394ba63}
CARDANO_LEDGER_REVISION=${CARDANO_LEDGER_REVISION:-82a4485f4b34da4752538cda7504aef346b9953b}
CARDANO_API_REVISION=${CARDANO_API_REVISION:-c57b8893544aca0855b70bc07330005b7b514054}
CARDANO_NODE_REVISION=${CARDANO_NODE_REVISION:-126efd54008b8f0f33b236a2fab43a16a1f3b4f1}

trap 'rm -rf "${WORK_DIR}"' EXIT

resolve_override_revision() {
	local name=$1
	local override_path=$2
	local fallback_revision=$3

	if git -C "${override_path}" rev-parse HEAD >/dev/null 2>&1; then
		git -C "${override_path}" rev-parse HEAD
		return
	fi
	if [[ -n "${fallback_revision}" ]]; then
		printf '%s\n' "${fallback_revision}"
		return
	fi

	echo "override path for ${name} is not a git checkout; set a fixed revision" >&2
	exit 1
}

stage_repo() {
	local name=$1
	local url=$2
	local revision=$3
	local override_path=$4
	local staged_path="${WORK_DIR}/${name}-${revision}"

	if [[ -n "${override_path}" ]]; then
		if [[ ! -d "${override_path}" ]]; then
			echo "override path does not exist: ${override_path}" >&2
			exit 1
		fi
		revision=$(resolve_override_revision "${name}" "${override_path}" "${revision}")
		staged_path="${WORK_DIR}/${name}-${revision}"
		cp -a "${override_path}" "${staged_path}"
		printf '%s\n' "${staged_path}"
		return
	fi

	if [[ -z "${revision}" ]]; then
		echo "missing fixed revision for ${name}" >&2
		exit 1
	fi

	curl -fsSL "${url}/archive/${revision}.tar.gz" | \
		tar xz -C "${WORK_DIR}"
	printf '%s\n' "${staged_path}"
}

copy_with_parents() {
	local repo_root=$1
	local repo_name=$2
	shift 2

	local repo_dest="${DEST_DIR}/${repo_name}"
	local relative_path

	mkdir -p "${repo_dest}"
	for relative_path in "$@"; do
		mkdir -p "${repo_dest}/$(dirname "${relative_path}")"
		cp "${repo_root}/${relative_path}" "${repo_dest}/${relative_path}"
	done
}

echo "Syncing curated upstream fixtures into ${DEST_DIR}"
rm -rf "${DEST_DIR}"
mkdir -p "${DEST_DIR}"

consensus_root=$(
	stage_repo \
		"ouroboros-consensus" \
		"https://github.com/IntersectMBO/ouroboros-consensus" \
		"${OUROBOROS_CONSENSUS_REVISION}" \
		"${OUROBOROS_CONSENSUS_SRC:-}"
)
ledger_root=$(
	stage_repo \
		"cardano-ledger" \
		"https://github.com/IntersectMBO/cardano-ledger" \
		"${CARDANO_LEDGER_REVISION}" \
		"${CARDANO_LEDGER_SRC:-}"
)
api_root=$(
	stage_repo \
		"cardano-api" \
		"https://github.com/IntersectMBO/cardano-api" \
		"${CARDANO_API_REVISION}" \
		"${CARDANO_API_SRC:-}"
)
node_root=$(
	stage_repo \
		"cardano-node" \
		"https://github.com/IntersectMBO/cardano-node" \
		"${CARDANO_NODE_REVISION}" \
		"${CARDANO_NODE_SRC:-}"
)

consensus_prefix="ouroboros-consensus-cardano/golden/cardano/CardanoNodeToNodeVersion2"
consensus_files=(
	"${consensus_prefix}/Block_Byron_EBB"
	"${consensus_prefix}/Block_Byron_regular"
	"${consensus_prefix}/Header_Byron_EBB"
	"${consensus_prefix}/Header_Byron_regular"
	"${consensus_prefix}/GenTx_Byron"
	"${consensus_prefix}/GenTxId_Byron"
)

for era in Shelley Allegra Mary Alonzo Babbage Conway Dijkstra; do
	consensus_files+=(
		"${consensus_prefix}/Block_${era}"
		"${consensus_prefix}/Header_${era}"
		"${consensus_prefix}/GenTx_${era}"
		"${consensus_prefix}/GenTxId_${era}"
	)
done

copy_with_parents "${consensus_root}" "ouroboros-consensus" \
	"${consensus_files[@]}"

ledger_files=(
	"eras/shelley/impl/golden/pparams.json"
	"eras/shelley/impl/golden/pparams-update.json"
	"eras/alonzo/test-suite/golden/block.cbor"
	"eras/alonzo/test-suite/golden/hex-block-node-issue-4228.cbor"
	"eras/alonzo/test-suite/golden/mainnet-alonzo-genesis.json"
	"eras/alonzo/test-suite/golden/tx.cbor"
)

for era in alonzo babbage conway dijkstra; do
	ledger_files+=(
		"eras/${era}/impl/golden/pparams.json"
		"eras/${era}/impl/golden/pparams-update.json"
		"eras/${era}/impl/golden/translations.cbor"
	)
done

copy_with_parents "${ledger_root}" "cardano-ledger" "${ledger_files[@]}"

api_files=(
	"cardano-api/test/cardano-api-golden/files/LegacyProtocolParameters.json"
	"cardano-api/test/cardano-api-golden/files/ShelleyGenesis.json"
	"cardano-api/test/cardano-api-golden/files/tx-canonical.json"
	"cardano-api/test/cardano-api-test/files/input/gov-anchor-data/invalid-drep-metadata.jsonld"
	"cardano-api/test/cardano-api-test/files/input/gov-anchor-data/no-confidence.jsonld"
	"cardano-api/test/cardano-api-test/files/input/gov-anchor-data/valid-drep-metadata.jsonld"
	"cardano-api/test/cardano-api-test/files/input/protocol-parameters/conway.json"
)

copy_with_parents "${api_root}" "cardano-api" "${api_files[@]}"

node_files=(
	"cardano-testnet/files/data/alonzo/genesis.alonzo.spec.json"
	"cardano-testnet/files/data/conway/genesis.conway.spec.json"
)

copy_with_parents "${node_root}" "cardano-node" "${node_files[@]}"

manifest_tmp=$(mktemp "${WORK_DIR}/manifest.XXXXXX")
(
	cd "${DEST_DIR}"
	find . -type f ! -name 'manifest.txt' | sort > "${manifest_tmp}"
)
mv "${manifest_tmp}" "${DEST_DIR}/manifest.txt"

echo "Fixture sync complete. Files written to ${DEST_DIR}"
