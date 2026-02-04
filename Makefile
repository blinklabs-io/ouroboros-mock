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
	@curl -L -s https://github.com/pragma-org/amaru/archive/main.tar.gz | tar xz -C /tmp/amaru-testdata
	@rm -rf $(ROOT_DIR)/conformance/testdata/eras
	@mkdir -p $(ROOT_DIR)/conformance/testdata/eras
	@cp -r /tmp/amaru-testdata/amaru-main/crates/amaru-ledger/tests/data/rules-conformance/* $(ROOT_DIR)/conformance/testdata/eras/
	@echo "Sanitizing file paths (Go module zip requires clean paths)..."
	@# Remove apostrophes from file/directory names
	@find $(ROOT_DIR)/conformance/testdata/eras -depth -name "*'*" -execdir bash -c 'mv "$$1" "$${1//\047/}"' _ {} \;
	@# Replace spaces with underscores
	@find $(ROOT_DIR)/conformance/testdata/eras -depth -name "* *" -execdir bash -c 'mv "$$1" "$${1// /_}"' _ {} \;
	@# Remove trailing underscores (from trailing spaces)
	@find $(ROOT_DIR)/conformance/testdata/eras -depth -name "*_" -execdir bash -c 'mv "$$1" "$${1%_}"' _ {} \;
	@rm -rf /tmp/amaru-testdata
	@echo "Download complete. Test data is now in conformance/testdata/eras/"
