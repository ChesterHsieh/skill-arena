#!/usr/bin/env bash
# skill-arena installer
# Usage: curl -sSL https://raw.githubusercontent.com/ChesterHsieh/skill-arena/main/install.sh | sh

set -e

REPO="ChesterHsieh/skill-arena"
BINARY="skill-arena"
INSTALL_DIR="/usr/local/bin"

# ── Shell detection ────────────────────────────────────────────────────────────
detect_shell_rc() {
  # Prefer $SHELL, fall back to the parent process name
  local shell_name
  shell_name=$(basename "${SHELL:-$(ps -p $PPID -o comm= 2>/dev/null || echo '')}")

  case "$shell_name" in
    zsh)  echo "$HOME/.zshrc" ;;
    bash) echo "$HOME/.bashrc" ;;
    fish) echo "$HOME/.config/fish/config.fish" ;;
    *)    echo "" ;;  # unknown — skip auto-add
  esac
}

add_to_path() {
  local dir="$1"
  local rc_file
  rc_file=$(detect_shell_rc)

  # Already in PATH — nothing to do
  if echo "$PATH" | tr ':' '\n' | grep -qx "$dir"; then
    return 0
  fi

  if [ -z "$rc_file" ]; then
    echo ""
    echo "  Could not detect shell. Add this manually:"
    echo "    export PATH=\"\$PATH:${dir}\""
    return 0
  fi

  echo ""
  printf "  Add '%s' to PATH in %s? [Y/n] " "$dir" "$rc_file"

  # Read answer — works both interactively and piped (curl | sh defaults to Y)
  local answer="Y"
  if [ -t 0 ]; then
    read -r answer </dev/tty
  fi

  case "${answer:-Y}" in
    [Yy]|"")
      local shell_name
      shell_name=$(basename "${SHELL:-sh}")

      if [ "$shell_name" = "fish" ]; then
        echo "fish_add_path $dir" >> "$rc_file"
      else
        printf '\nexport PATH="$PATH:%s"\n' "$dir" >> "$rc_file"
      fi

      echo "  ✓ Added to $rc_file"
      echo "  Run: source $rc_file  (or open a new terminal)"
      ;;
    *)
      echo "  Skipped. Add manually: export PATH=\"\$PATH:${dir}\""
      ;;
  esac
}

# ── OS / arch detection ────────────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

# ── Install via Go if available ────────────────────────────────────────────────
if command -v go &>/dev/null; then
  echo "Go detected — installing from source..."
  go install github.com/${REPO}@latest
  GOBIN=$(go env GOPATH)/bin
  echo "✓ Installed to ${GOBIN}/skill-arena"
  add_to_path "$GOBIN"
  exit 0
fi

# ── Download pre-built binary ──────────────────────────────────────────────────
echo "Fetching latest release..."
LATEST=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed 's/.*"tag_name": *"\(.*\)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Error: could not fetch latest release."
  echo "Install Go and run: go install github.com/${REPO}@latest"
  exit 1
fi

ARCHIVE="${BINARY}_${OS}_${ARCH}"
EXT="tar.gz"
[ "$OS" = "windows" ] && EXT="zip"

URL="https://github.com/${REPO}/releases/download/${LATEST}/${ARCHIVE}.${EXT}"
TMPDIR=$(mktemp -d)

echo "Downloading skill-arena ${LATEST} (${OS}/${ARCH})..."
curl -sSfL "$URL" -o "${TMPDIR}/${ARCHIVE}.${EXT}"

echo "Extracting..."
if [ "$EXT" = "zip" ]; then
  unzip -q "${TMPDIR}/${ARCHIVE}.${EXT}" -d "${TMPDIR}"
else
  tar -xzf "${TMPDIR}/${ARCHIVE}.${EXT}" -C "${TMPDIR}"
fi

EXTRACTED="${TMPDIR}/${BINARY}"
[ "$OS" = "windows" ] && EXTRACTED="${TMPDIR}/${BINARY}.exe"
chmod +x "$EXTRACTED"

# Try /usr/local/bin first; if not writable use ~/.local/bin
if [ -w "$INSTALL_DIR" ] || sudo -n true 2>/dev/null; then
  if [ -w "$INSTALL_DIR" ]; then
    mv "$EXTRACTED" "${INSTALL_DIR}/${BINARY}"
  else
    sudo mv "$EXTRACTED" "${INSTALL_DIR}/${BINARY}"
  fi
  echo "✓ skill-arena ${LATEST} installed to ${INSTALL_DIR}/${BINARY}"
  # /usr/local/bin is almost always in PATH already — check anyway
  add_to_path "$INSTALL_DIR"
else
  # Fallback: install to ~/.local/bin (no sudo needed)
  LOCAL_BIN="$HOME/.local/bin"
  mkdir -p "$LOCAL_BIN"
  mv "$EXTRACTED" "${LOCAL_BIN}/${BINARY}"
  echo "✓ skill-arena ${LATEST} installed to ${LOCAL_BIN}/${BINARY}"
  add_to_path "$LOCAL_BIN"
fi

rm -rf "$TMPDIR"
echo ""
echo "  Run: skill-arena --help"
