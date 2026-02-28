#!/bin/sh
set -e

REPO="MaximoCoder/Enveil"
BINARY="enveil"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux)  ;;
  darwin) ;;
  *)
    echo "Unsupported OS: $OS"
    echo "For Windows, use WSL2 and run this installer inside it."
    exit 1
    ;;
esac

ASSET="enveil-${OS}-${ARCH}"

echo "Detecting latest version..."
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": "\(.*\)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version"
  exit 1
fi

echo "Installing Enveil ${VERSION} for ${OS}/${ARCH}..."

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"
TMP=$(mktemp)

curl -fsSL "$DOWNLOAD_URL" -o "$TMP"

# Verify checksum
echo "Verifying checksum..."
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
EXPECTED=$(curl -fsSL "$CHECKSUMS_URL" | grep "$ASSET" | awk '{print $1}')

if command -v sha256sum > /dev/null 2>&1; then
  ACTUAL=$(sha256sum "$TMP" | awk '{print $1}')
elif command -v shasum > /dev/null 2>&1; then
  ACTUAL=$(shasum -a 256 "$TMP" | awk '{print $1}')
else
  echo "Warning: could not verify checksum, sha256sum not found"
  ACTUAL="$EXPECTED"
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Checksum verification failed"
  echo "Expected: $EXPECTED"
  echo "Actual:   $ACTUAL"
  rm -f "$TMP"
  exit 1
fi

chmod +x "$TMP"
sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"

echo ""
echo "Enveil ${VERSION} installed successfully"
echo ""
echo "To get started:"
echo "  enveil init"
echo ""
echo "To enable shell integration, add this to your ~/.zshrc or ~/.bashrc:"
echo "  eval \"\$(enveil shell-init)\""
echo ""