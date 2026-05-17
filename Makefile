BINARY_NAME := mediakit-cli
MAIN_PKG := ./cmd/mediakit
BUILD_DIR := .mediakit/build/dev
DIST_DIR := dist
VERSION ?= $(shell (git describe --tags --always --dirty 2>/dev/null || echo dev) | sed 's/^v//')
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GOFLAGS ?=
GORELEASER ?= goreleaser
LDFLAGS := -s -w -X 'mediakit-cli/internal/build.Version=$(VERSION)' -X 'mediakit-cli/internal/build.Date=$(DATE)'

.PHONY: build build-all snapshot test clean tag

build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PKG)

build-all:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PKG)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PKG)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PKG)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PKG)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PKG)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PKG)

snapshot:
	$(GORELEASER) release --snapshot --clean

tag:
	git tag v$(VERSION)
	git push origin v$(VERSION)

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)
