#!/usr/bin/env bash
set -euo pipefail

REPO="casablanque-code/cfzt"
BINARY="zt"
INSTALL_DIR="/usr/local/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

# Resolve latest release tag
TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep '"tag_name"' | sed 's/.*"tag_name": "\(.*\)".*/\1/')

if [[ -z "$TAG" ]]; then
  echo "error: could not resolve latest release tag" >&2
  exit 1
fi

FILENAME="${BINARY}-${OS}-${ARCH}"
if [[ "$OS" == "windows" ]]; then FILENAME="${FILENAME}.exe"; fi

BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
BIN_URL="${BASE_URL}/${FILENAME}"
SUM_URL="${BASE_URL}/${FILENAME}.sha256"

echo "Installing zt ${TAG} (${OS}/${ARCH})..."

# Download binary and checksum
curl -fsSL "$BIN_URL" -o "/tmp/${BINARY}"
curl -fsSL "$SUM_URL" -o "/tmp/${BINARY}.sha256"

# Verify checksum
echo "Verifying checksum..."
EXPECTED=$(awk '{print $1}' "/tmp/${BINARY}.sha256")
if command -v sha256sum &>/dev/null; then
  ACTUAL=$(sha256sum "/tmp/${BINARY}" | awk '{print $1}')
elif command -v shasum &>/dev/null; then
  ACTUAL=$(shasum -a 256 "/tmp/${BINARY}" | awk '{print $1}')
else
  echo "warning: no sha256sum or shasum found — skipping checksum verification" >&2
  ACTUAL="$EXPECTED"
fi

if [[ "$ACTUAL" != "$EXPECTED" ]]; then
  echo "error: checksum mismatch" >&2
  echo "  expected: $EXPECTED" >&2
  echo "  got:      $ACTUAL" >&2
  rm -f "/tmp/${BINARY}" "/tmp/${BINARY}.sha256"
  exit 1
fi

echo "Checksum OK"

chmod +x "/tmp/${BINARY}"

if [[ -w "$INSTALL_DIR" ]]; then
  mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

rm -f "/tmp/${BINARY}.sha256"

echo ""
echo "✓ zt ${TAG} installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Next steps:"
echo "  1. Install cloudflared: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/"
echo "  2. Run: zt init"
echo "  3. Run: zt up <name> <port>"
