#!/bin/bash

# Define the list of binaries to process
BINARIES=(
  "launchpad-mcp"
)

# Define the list of architectures
ARCHS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

# Version and build info
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
COMMIT_HASH=${COMMIT_HASH:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
BUILD_TIME=${BUILD_TIME:-$(date -u '+%Y-%m-%d_%H:%M:%S')}

# Build flags
LDFLAGS="-X main.Version=${VERSION} -X main.CommitHash=${COMMIT_HASH} -X main.BuildTime=${BUILD_TIME}"

echo "Building binaries for version: $VERSION"
echo "Commit hash: $COMMIT_HASH"
echo "Build time: $BUILD_TIME"
echo

# Create bin directory if it doesn't exist
mkdir -p bin

# Build for each architecture
for arch in "${ARCHS[@]}"; do
  IFS='/' read -r GOOS GOARCH <<< "$arch"
  
  echo "Building for $GOOS/$GOARCH..."
  
  for binary in "${BINARIES[@]}"; do
    output_name="$binary"
    input_path="./cmd/main.go"
    
    # Add .exe extension for Windows
    if [ "$GOOS" = "windows" ]; then
      output_name="${binary}.exe"
    fi
    
    # Set the output path
    output_path="bin/${binary}_${GOOS}_${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
      output_path="${output_path}.exe"
    fi
    
    # Build the binary
    env GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build \
      -ldflags "$LDFLAGS" \
      -o "$output_path" \
      "$input_path"
    
    if [ $? -eq 0 ]; then
      echo "  ✓ Built $output_path"
    else
      echo "  ✗ Failed to build $output_path"
      exit 1
    fi
  done
  
  echo
done

echo "All binaries built successfully!"
echo "Binaries are located in the bin/ directory:"
ls -la bin/