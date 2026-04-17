#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/build-common.sh"

APP_NAME="${APP_NAME:-MiniPost}"
VERSION="${VERSION:-$(default_build_version)}"
OUT_DIR="${OUT_DIR:-$ROOT_DIR/dist/windows}"
WINDOWS_PLATFORM="${WINDOWS_PLATFORM:-windows/amd64}"
WEBVIEW2_STRATEGY="${WEBVIEW2_STRATEGY:-download}"
WAILS_BUILD_FLAGS="${WAILS_BUILD_FLAGS:-}"
WINDOWS_ARCH="${WINDOWS_PLATFORM##*/}"
ARTIFACT_PREFIX="${WINDOWS_ARTIFACT_PREFIX:-${ARTIFACT_PREFIX:-$(build_artifact_prefix "$APP_NAME" "$VERSION" "windows" "$WINDOWS_ARCH")}}"

# Optional Windows signing (works on Windows host with signtool available)
WIN_SIGNTOOL="${WIN_SIGNTOOL:-signtool}"
WIN_CERT_FILE="${WIN_CERT_FILE:-}"
WIN_CERT_PASSWORD="${WIN_CERT_PASSWORD:-}"
WIN_TIMESTAMP_URL="${WIN_TIMESTAMP_URL:-http://timestamp.digicert.com}"

log() { printf '\n[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }
need_cmd() { command -v "$1" >/dev/null 2>&1 || { echo "Missing command: $1" >&2; exit 1; }; }

need_cmd wails

setup_cross_compiler_if_needed() {
  local host
  host="$(uname -s)"

  # Native Windows environment generally doesn't need explicit CC.
  if [[ "$host" =~ MINGW|MSYS|CYGWIN ]]; then
    return
  fi

  # Cross-compiling from macOS/Linux: prefer Zig, fallback to MinGW.
  if command -v zig >/dev/null 2>&1; then
    export CC="zig cc -target x86_64-windows-gnu"
    export CXX="zig c++ -target x86_64-windows-gnu"
    log "Using Zig cross-compiler: $CC"
    return
  fi

  if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
    export CC="x86_64-w64-mingw32-gcc"
    export CXX="x86_64-w64-mingw32-g++"
    log "Using MinGW cross-compiler: $CC"
    return
  fi

  echo "No cross-compiler found for Windows build." >&2
  echo "Install one of: zig OR x86_64-w64-mingw32-gcc" >&2
  exit 1
}

setup_cross_compiler_if_needed

log "Cleaning previous Windows artifacts: $OUT_DIR"
clean_output_dir "$OUT_DIR" "$ROOT_DIR"

log "Building Windows app with Wails (platform: $WINDOWS_PLATFORM)"
wails build -clean -platform "$WINDOWS_PLATFORM" -webview2 "$WEBVIEW2_STRATEGY" -nsis ${WAILS_BUILD_FLAGS}

# Copy Windows artifacts from build/bin to dist/windows
log "Collecting artifacts to $OUT_DIR"
find "$ROOT_DIR/build/bin" -maxdepth 1 -type f \( -name "*.exe" -o -name "*.msi" \) -print0 | while IFS= read -r -d '' f; do
  source_name="$(basename "$f")"
  source_name_lower="$(printf '%s' "$source_name" | tr '[:upper:]' '[:lower:]')"

  case "$source_name_lower" in
    *installer*.exe) target_name="${ARTIFACT_PREFIX}-setup.exe" ;;
    *.msi) target_name="${ARTIFACT_PREFIX}-setup.msi" ;;
    *.exe) target_name="${ARTIFACT_PREFIX}-portable.exe" ;;
    *) target_name="${ARTIFACT_PREFIX}-$(sanitize_artifact_part "$source_name")" ;;
  esac

  cp "$f" "$OUT_DIR/$target_name"
done

# Optional signing (best practice for production release)
if [[ -n "$WIN_CERT_FILE" && -n "$WIN_CERT_PASSWORD" ]]; then
  if command -v "$WIN_SIGNTOOL" >/dev/null 2>&1; then
    log "Signing Windows artifacts with signtool"
    find "$OUT_DIR" -maxdepth 1 -type f \( -name "*.exe" -o -name "*.msi" \) -print0 | while IFS= read -r -d '' f; do
      "$WIN_SIGNTOOL" sign /fd SHA256 /td SHA256 /tr "$WIN_TIMESTAMP_URL" /f "$WIN_CERT_FILE" /p "$WIN_CERT_PASSWORD" "$f"
    done
  else
    log "WIN_CERT_FILE/WIN_CERT_PASSWORD provided, but signtool not found. Skip signing."
  fi
else
  log "No Windows signing cert configured. Skip signing (OK for internal testing)."
fi

log "Build complete"
ls -lh "$OUT_DIR" || true
