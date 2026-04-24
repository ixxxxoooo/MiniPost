#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This script is for macOS only." >&2
  exit 1
fi

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <app-or-dmg-path>"
  echo "Example: $0 dist/macos/MiniPost-1.0.0-macos-arm64.dmg"
  exit 1
fi

TARGET="$1"
if [[ ! -e "$TARGET" ]]; then
  echo "Target not found: $TARGET" >&2
  exit 1
fi

echo "Removing quarantine attribute: $TARGET"
xattr -dr com.apple.quarantine "$TARGET"

echo "Authorization finished."
echo "If needed, you can additionally trust it with:"
echo "sudo spctl --add --label 'MiniPost Local Trust' \"$TARGET\""
