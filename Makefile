.PHONY: build build-frontend test run clean clean-frontend deps help install-local package binaries generate

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
	cd frontend/signing && bun install

# Generate embedded contract files
generate:
	@echo "Generating embedded contract files..."
	go generate ./...
inspect:
	npx -y @modelcontextprotocol/inspector go run cmd/main.go

# Build frontend assets first, then the Go binary
build: build-frontend
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/stdio/main.go

# Build frontend assets
build-frontend:
	@echo "Building frontend assets..."
	@if [ -d "frontend/signing" ]; then \
		cd frontend/signing && bun install && bun run build && \
		echo "Copying compiled assets to internal/assets..." && \
		cp dist/app.js ../../internal/assets/signing_app.js && \
		cp dist/app.css ../../internal/assets/signing_app.css && \
		echo "Frontend assets built and copied successfully!"; \
	else \
		echo "Frontend directory not found, skipping frontend build"; \
	fi

# Build for multiple architectures
binaries: build-frontend
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
clean: clean-frontend
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)/
	sudo rm -rf /usr/local/bin/$(BINARY_NAME) 2>/dev/null || true
	go clean

# Clean frontend build artifacts
clean-frontend:
	@echo "Cleaning frontend build artifacts..."
	@if [ -d "frontend/signing/dist" ]; then \
		rm -rf frontend/signing/dist; \
		echo "Frontend dist directory cleaned"; \
	fi


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
	@echo "  build       - Build the project (includes frontend)"
	@echo "  build-frontend - Build frontend assets only"
	@echo "  binaries    - Build for multiple architectures"
	@echo "  test        - Run tests"
	@echo "  run         - Run the MCP server directly"
	@echo "  run-bin     - Build and run the binary"
	@echo "  install-local - Install to /usr/local/bin (requires sudo)"
	@echo "  package     - Package and notarize for distribution"
	@echo "  generate    - Generate embedded contract files"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  clean       - Clean build artifacts (includes frontend)"
	@echo "  clean-frontend - Clean frontend build artifacts only"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  sec         - Run security scan"
	@echo "  e2e-network - Start anvil testnet for E2E tests"
	@echo "  e2e-browser - Run browser E2E tests with chromedp"
	@echo "  e2e-browser-headful - Run browser tests in headful mode"
	@echo "  e2e-all     - Run all E2E tests including browser tests"
	@echo "  help        - Show this help message"