#!/bin/bash

# Exit on any error
set -e

# Check if required variables are set
if [ -z "${SIGNING_CERTIFICATE_NAME}" ]; then
  echo "Warning: SIGNING_CERTIFICATE_NAME is not set. Skipping code signing."
  echo "To enable code signing, set SIGNING_CERTIFICATE_NAME environment variable."
  exit 0
fi

# Define the list of binaries to sign
BINARIES=(
  "launchpad-mcp"
)

# Check if we're on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
  echo "Code signing is only supported on macOS. Skipping..."
  exit 0
fi

echo "Starting code signing process..."
echo "Certificate: ${SIGNING_CERTIFICATE_NAME}"

# Sign and verify each binary
for binary in "${BINARIES[@]}"; do
  BINARY_PATH="bin/${binary}"
  
  # Check if binary exists
  if [ ! -f "${BINARY_PATH}" ]; then
    echo "Warning: Binary ${BINARY_PATH} not found. Skipping..."
    continue
  fi
  
  echo "Signing binary: ${BINARY_PATH}"

  # Sign the binary with hardened runtime and entitlements
  codesign --force --options runtime --timestamp \
    --sign "${SIGNING_CERTIFICATE_NAME}" "${BINARY_PATH}"
  
  # Verify signature
  echo "Verifying signature for ${BINARY_PATH}..."
  codesign --verify --verbose "${BINARY_PATH}"
  
  echo "✓ Successfully signed ${BINARY_PATH}"
done

echo "All binaries signed successfully!"