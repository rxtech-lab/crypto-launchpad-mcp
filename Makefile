.PHONY: build test run clean deps help install-local package binaries

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
	go test -v ./...

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
	@echo "  deps        - Download and tidy dependencies"
	@echo "  clean       - Clean build artifacts"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  sec         - Run security scan"
	@echo "  help        - Show this help message"