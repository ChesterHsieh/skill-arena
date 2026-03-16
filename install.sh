#!/usr/bin/env bash
# skill-arena installer
# Usage: curl -sSL https://raw.githubusercontent.com/ChesterHsieh/skill-arena/main/install.sh | sh

set -e

REPO="ChesterHsieh/skill-arena"
BINARY="skill-arena"
INSTALL_DIR="/usr/local/bin"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)       ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

# If Go is available, build from source
if command -v go &>/dev/null; then
  echo "Go detected — installing from source..."
  go install github.com/${REPO}@latest
  GOBIN=$(go env GOPATH)/bin
  echo "✓ Installed to ${GOBIN}/skill-arena"
  echo ""
  if ! echo "$PATH" | grep -q "$GOBIN"; then
    echo "  Add this to your shell profile:"
    echo "    export PATH=\"\$PATH:${GOBIN}\""
  fi
  exit 0
fi

# Fetch latest release tag from GitHub API
echo "Fetching latest release..."
LATEST=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed 's/.*"tag_name": *"\(.*\)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Error: could not fetch latest release tag."
  echo "Install Go and run: go install github.com/${REPO}@latest"
  exit 1
fi

# Build download URL (GoReleaser archive naming: skill-arena_os_arch.tar.gz)
ARCHIVE="${BINARY}_${OS}_${ARCH}"
EXT="tar.gz"
[ "$OS" = "windows" ] && EXT="zip"

URL="https://github.com/${REPO}/releases/download/${LATEST}/${ARCHIVE}.${EXT}"
TMPDIR=$(mktemp -d)

echo "Downloading skill-arena ${LATEST} (${OS}/${ARCH})..."
curl -sSfL "$URL" -o "${TMPDIR}/${ARCHIVE}.${EXT}"

# Extract
echo "Extracting..."
if [ "$EXT" = "zip" ]; then
  unzip -q "${TMPDIR}/${ARCHIVE}.${EXT}" -d "${TMPDIR}"
else
  tar -xzf "${TMPDIR}/${ARCHIVE}.${EXT}" -C "${TMPDIR}"
fi

# Install
EXTRACTED="${TMPDIR}/${BINARY}"
[ "$OS" = "windows" ] && EXTRACTED="${TMPDIR}/${BINARY}.exe"

chmod +x "$EXTRACTED"
if [ -w "$INSTALL_DIR" ]; then
  mv "$EXTRACTED" "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "$EXTRACTED" "${INSTALL_DIR}/${BINARY}"
fi

rm -rf "$TMPDIR"

echo "✓ skill-arena ${LATEST} installed to ${INSTALL_DIR}/${BINARY}"
echo "  Run: skill-arena --help"
