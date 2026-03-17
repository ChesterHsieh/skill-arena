#!/usr/bin/env bash
# skill-arena uninstaller
# Usage: curl -sSL https://raw.githubusercontent.com/ChesterHsieh/skill-arena/main/uninstall.sh | sh

set -e

BINARY="skill-arena"
REMOVED=0

remove() {
  local path="$1"
  if [ -f "$path" ]; then
    if [ -w "$path" ]; then
      rm "$path"
    else
      sudo rm "$path"
    fi
    echo "✓ Removed $path"
    REMOVED=1
  fi
}

# All locations the installer may have used
remove "/usr/local/bin/${BINARY}"
remove "$HOME/.local/bin/${BINARY}"
remove "$(go env GOPATH 2>/dev/null)/bin/${BINARY}" 2>/dev/null || true

if [ "$REMOVED" -eq 0 ]; then
  echo "skill-arena not found in standard locations."
  echo "If you installed it elsewhere, remove it manually:"
  echo "  which skill-arena && rm \$(which skill-arena)"
else
  echo ""
  echo "Config and history are kept at:"
  echo "  ~/.skill-arena/   — API config"
  echo "  .skill-arena/     — eval history (project-local)"
  echo ""
  echo "To remove those too:"
  echo "  rm -rf ~/.skill-arena"
fi
