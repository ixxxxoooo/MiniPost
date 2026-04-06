#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

: "${SKIP_MACOS:=0}"
: "${SKIP_WINDOWS:=0}"

if [[ "$SKIP_MACOS" != "1" ]]; then
  ./scripts/build-macos.sh
fi

if [[ "$SKIP_WINDOWS" != "1" ]]; then
  ./scripts/build-windows.sh
fi
