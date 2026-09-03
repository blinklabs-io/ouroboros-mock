# Determine root directory
ROOT_DIR=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

# Gather all .go files for use in dependencies below
GO_FILES=$(shell find $(ROOT_DIR) -name '*.go')

# Gather list of expected binaries
BINARIES=$(shell cd $(ROOT_DIR)/cmd && ls -1)

# Extract Go module name from go.mod
GOMODULE=$(shell grep ^module $(ROOT_DIR)/go.mod | awk '{ print $$2 }')

# Set version strings based on git tag and current ref
GO_LDFLAGS=-ldflags "-s -w"

.PHONY: \
	build \
	mod-tidy \
	clean \
	format \
	golines \
	test \
	download-amaru-testdata \
	prepare-blueprint-testdata \
	download-upstream-fixtures \
	gen-synthetic-vectors \
	capture-consensus-vectors

# Alias for building program binary
build: $(BINARIES)

mod-tidy:
	# Needed to fetch new dependencies and add them to go.mod
	go mod tidy

clean:
	rm -f $(BINARIES)

format: mod-tidy
	go fmt ./...
	gofmt -s -w $(GO_FILES)

golines:
	golines -w --ignore-generated --chain-split-dots --max-len=80 --reformat-tags .

test: prepare-blueprint-testdata mod-tidy
	go test -v -race ./...

# Build our program binaries
# Depends on GO_FILES to determine when rebuild is needed
$(BINARIES): mod-tidy $(GO_FILES)
	CGO_ENABLED=0 \
	go build \
		$(GO_LDFLAGS) \
		-o $(@) \
		./cmd/$(@)

# Prepare the pinned Cardano Blueprint conformance corpus from the submodule.
prepare-blueprint-testdata:
	@bash $(ROOT_DIR)/scripts/update-blueprint-conformance.sh

# Regenerate the locally-authored synthetic conformance vectors. These
# splice rollback events into existing Cardano Blueprint-derived bases to exercise
# harness code paths the upstream corpus does not cover. Re-run this
# target after `make download-amaru-testdata` so the synthetic vectors
# track the refreshed bases. The output lives under
# conformance/testdata/synthetic/ (outside conformance/testdata/eras/)
# so it is preserved across Blueprint-corpus refreshes.
SYNTHETIC_ROLLBACK_BASE=conformance/testdata/eras/conway/impl/dump/Conway.Imp.ConwayImpSpec_-_Version_10.UTXOS.Conway_features_fail_in_Plutusdescribe_v1_and_v2.Unsupported_Fields.CurrentTreasuryValue.V1
SYNTHETIC_ROLLBACK_OUT=conformance/testdata/synthetic/rollback/CurrentTreasuryValue_V1

gen-synthetic-vectors: prepare-blueprint-testdata gen-rollback-vector
	./gen-rollback-vector \
		-base $(SYNTHETIC_ROLLBACK_BASE) \
		-out $(SYNTHETIC_ROLLBACK_OUT) \
		-title "synthetic/rollback/CurrentTreasuryValue-V1"

# Regenerate every committed consensus-conformance vector by running
# the full docker-compose capture stack for each scenario. Each
# scenario writes its output to consensus/testdata/captured/<name>.json,
# overwriting the existing golden. Requires docker.
capture-consensus-vectors:
	$(ROOT_DIR)/consensus/capture-all.sh

# Download and update curated upstream fixtures used for block/message tests.
# Sources:
#   - https://github.com/IntersectMBO/ouroboros-consensus
#   - https://github.com/IntersectMBO/cardano-ledger
#   - https://github.com/IntersectMBO/cardano-api
#   - https://github.com/IntersectMBO/cardano-node
download-upstream-fixtures:
	@bash $(ROOT_DIR)/scripts/update-upstream-fixtures.sh

# Compatibility name for callers of the former Amaru refresh target.
download-amaru-testdata: prepare-blueprint-testdata
