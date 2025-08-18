.PHONY: build test run clean deps help install-local package binaries generate

BINARY_NAME=launchpad-mcp
BUILD_DIR=./bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.CommitHash=$(COMMIT_HASH) -X main.BuildTime=$(BUILD_TIME)"

# Default target
all: deps build test

# Download and tidy dependencies
deps:
	go mod download
	go mod tidy

# Generate embedded contract files
generate:
	@echo "Generating embedded contract files..."
	go generate ./...
inspect:
	npx -y @modelcontextprotocol/inspector go run cmd/main.go

# Build the project
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/main.go

# Build for multiple architectures
binaries:
	@echo "Building binaries for multiple architectures..."
	./scripts/binaries.sh

# Run tests
test:
	go test -v -p 1 ./...

# Run the MCP server directly (no build)
run:
	go run ./cmd/main.go

# Run the built binary
run-bin: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Install locally to /usr/local/bin
install-local: clean build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "$(BINARY_NAME) installed successfully!"
	@echo "You can now run '$(BINARY_NAME)' from anywhere."

# Package and notarize for distribution
package: build
	@echo "Packaging and notarizing $(BINARY_NAME)..."
	./scripts/sign.sh
	./scripts/package-notarize.sh

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)/
	sudo rm -rf /usr/local/bin/$(BINARY_NAME) 2>/dev/null || true
	go clean


e2e-network:
	killall anvil || true
	anvil

# Run browser E2E tests with chromedp
e2e-browser:
	go test -v ./e2e/api

# Run all E2E tests including browser tests
e2e-all: e2e-network
	go test -v ./e2e
	go test -v ./e2e/api

# Run browser tests in headful mode (with visible browser)
e2e-browser-headful:
	HEADLESS=false go test -v ./e2e/api

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Security scan
sec:
	gosec ./...

# Show help
help:
	@echo "Available commands:"
	@echo "  build       - Build the project"
	@echo "  binaries    - Build for multiple architectures"
	@echo "  test        - Run tests"
	@echo "  run         - Run the MCP server directly"
	@echo "  run-bin     - Build and run the binary"
	@echo "  install-local - Install to /usr/local/bin (requires sudo)"
	@echo "  package     - Package and notarize for distribution"
	@echo "  generate    - Generate embedded contract files"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  clean       - Clean build artifacts"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  sec         - Run security scan"
	@echo "  e2e-network - Start anvil testnet for E2E tests"
	@echo "  e2e-browser - Run browser E2E tests with chromedp"
	@echo "  e2e-browser-headful - Run browser tests in headful mode"
	@echo "  e2e-all     - Run all E2E tests including browser tests"
	@echo "  help        - Show this help message"