#!/bin/bash

# Exit on any error
set -e

# Check if we're on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
  echo "Packaging and notarization is only supported on macOS. Skipping..."
  exit 0
fi

# Check if required variables are set for notarization
if [ -z "${INSTALLER_SIGNING_CERTIFICATE_NAME}" ]; then
  echo "Warning: INSTALLER_SIGNING_CERTIFICATE_NAME is not set. Creating unsigned package."
  SKIP_SIGNING=true
fi

if [ -z "${APPLE_ID}" ] || [ -z "${APPLE_ID_PWD}" ] || [ -z "${APPLE_TEAM_ID}" ]; then
  echo "Warning: Apple ID credentials not set (APPLE_ID, APPLE_ID_PWD, APPLE_TEAM_ID). Skipping notarization."
  SKIP_NOTARIZATION=true
fi

# Define binaries and package info
BINARIES=(
  "launchpad-mcp"
)

# Get version for package name
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
PKG_FILE="launchpad-mcp_macOS_arm64_${VERSION}.pkg"
TMP_DIR="tmp_pkg_build"

echo "Creating package: ${PKG_FILE}"

# Create a temporary directory structure for pkgbuild
echo "Creating package structure..."
mkdir -p "${TMP_DIR}/usr/local/bin"
mkdir -p "${TMP_DIR}_scripts"

# Verify and copy each binary
for binary in "${BINARIES[@]}"; do
  BINARY_PATH="bin/${binary}"
  
  # Check if binary exists
  if [ ! -f "${BINARY_PATH}" ]; then
    echo "Error: Binary ${BINARY_PATH} not found. Run 'make build' first."
    exit 1
  fi
  
  # If not skipping signing, verify the binary is signed
  if [ -z "${SKIP_SIGNING}" ]; then
    echo "Verifying binary signature: ${BINARY_PATH}"
    codesign --verify --verbose "${BINARY_PATH}" || {
      echo "Error: Binary ${binary} is not properly signed. Run 'make sign' first."
      exit 1
    }
  fi
  
  # Copy binary to temporary directory
  echo "Adding ${binary} to package..."
  cp "${BINARY_PATH}" "${TMP_DIR}/usr/local/bin/"
  chmod +x "${TMP_DIR}/usr/local/bin/${binary}"
done

# Copy post-install script
echo "Adding post-install script..."
cp "scripts/post-install.sh" "${TMP_DIR}_scripts/postinstall"
chmod +x "${TMP_DIR}_scripts/postinstall"

# Create the pkg file
echo "Building pkg installer..."
if [ -z "${SKIP_SIGNING}" ]; then
  # Signed package
  pkgbuild --root "${TMP_DIR}" \
    --scripts "${TMP_DIR}_scripts" \
    --identifier "com.rxtech-lab.launchpad-mcp" \
    --version "${VERSION}" \
    --sign "${INSTALLER_SIGNING_CERTIFICATE_NAME}" \
    --install-location "/" \
    "${PKG_FILE}"
else
  # Unsigned package
  pkgbuild --root "${TMP_DIR}" \
    --scripts "${TMP_DIR}_scripts" \
    --identifier "com.rxtech-lab.launchpad-mcp" \
    --version "${VERSION}" \
    --install-location "/" \
    "${PKG_FILE}"
fi

# Clean up temporary directories
rm -rf "${TMP_DIR}"
rm -rf "${TMP_DIR}_scripts"

# Notarize the pkg file if credentials are available
if [ -z "${SKIP_NOTARIZATION}" ] && [ -z "${SKIP_SIGNING}" ]; then
  echo "Submitting for notarization..."
  xcrun notarytool submit "${PKG_FILE}" \
    --verbose \
    --apple-id "${APPLE_ID}" \
    --team-id "${APPLE_TEAM_ID}" \
    --password "${APPLE_ID_PWD}" \
    --wait

  # Staple the notarization ticket to the pkg
  echo "Stapling notarization ticket..."
  xcrun stapler staple -v "${PKG_FILE}"
  
  echo "✓ Package created, signed, notarized and stapled successfully: ${PKG_FILE}"
else
  echo "✓ Package created successfully: ${PKG_FILE}"
  if [ ! -z "${SKIP_SIGNING}" ]; then
    echo "  Note: Package is unsigned"
  fi
  if [ ! -z "${SKIP_NOTARIZATION}" ]; then
    echo "  Note: Package is not notarized"
  fi
fi

echo
echo "Package details:"
ls -lh "${PKG_FILE}"