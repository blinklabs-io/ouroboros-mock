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

.PHONY: build mod-tidy clean format golines test download-amaru-testdata

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

test: mod-tidy
	go test -v -race ./...

# Build our program binaries
# Depends on GO_FILES to determine when rebuild is needed
$(BINARIES): mod-tidy $(GO_FILES)
	CGO_ENABLED=0 \
	go build \
		$(GO_LDFLAGS) \
		-o $(@) \
		./cmd/$(@)

# Download and update conformance test data from Amaru
# Source: https://github.com/pragma-org/amaru
# Path: crates/amaru-ledger/tests/data/rules-conformance/
download-amaru-testdata:
	@echo "Downloading latest Amaru conformance test data..."
	@rm -rf /tmp/amaru-testdata
	@mkdir -p /tmp/amaru-testdata
	@curl -fsSL https://github.com/pragma-org/amaru/archive/main.tar.gz | tar xz -C /tmp/amaru-testdata
	@rm -rf $(ROOT_DIR)/conformance/testdata/eras
	@mkdir -p $(ROOT_DIR)/conformance/testdata/eras
	@cp -r /tmp/amaru-testdata/amaru-main/crates/amaru-ledger/tests/data/rules-conformance/eras/* $(ROOT_DIR)/conformance/testdata/eras/
	@echo "Sanitizing file paths (Go module zip requires clean paths)..."
	@# Replace any unsafe characters with underscore, collapse multiple underscores, remove trailing underscores
	@# Safe characters: alphanumeric, dot, dash, underscore
	@find $(ROOT_DIR)/conformance/testdata/eras -depth -execdir bash -c 'name="$${1#./}"; safe=$$(printf "%s" "$$name" | tr -c "[:alnum:]._-" "_" | sed "s/__*/_/g; s/_$$//"); [ "$$name" != "$$safe" ] && mv -- "$$name" "$$safe" || true' _ {} \;
	@rm -rf /tmp/amaru-testdata
	@echo "Download complete. Test data is now in conformance/testdata/eras/"
