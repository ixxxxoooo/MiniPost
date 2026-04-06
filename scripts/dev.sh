#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Default dev mode optimized for faster startup and less noise.
# -skipbindings: avoid running the bindings generation executable every launch
# -m: skip `go mod tidy` on each dev startup
exec wails dev -skipbindings -m "$@"
