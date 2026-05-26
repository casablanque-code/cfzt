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

TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep '"tag_name"' | sed 's/.*"tag_name": "\(.*\)".*/\1/')

URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY}-${OS}-${ARCH}"
if [[ "$OS" == "windows" ]]; then URL="${URL}.exe"; fi

echo "Installing zt ${TAG} (${OS}/${ARCH})..."
curl -fsSL "$URL" -o "/tmp/${BINARY}"
chmod +x "/tmp/${BINARY}"

if [[ -w "$INSTALL_DIR" ]]; then
  mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo "✓ zt installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Next steps:"
echo "  1. Install cloudflared: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/"
echo "  2. Run: zt init"
echo "  3. Run: zt up <name> <port>"
