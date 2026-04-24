#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/build-common.sh"

APP_NAME="${APP_NAME:-MiniPost}"
VERSION="${VERSION:-$(default_build_version)}"
OUT_DIR="${OUT_DIR:-$ROOT_DIR/dist/macos}"
MACOS_PLATFORM="${MACOS_PLATFORM:-$( [[ "$(uname -m)" == "arm64" ]] && echo darwin/arm64 || echo darwin/amd64 )}"
WAILS_BUILD_FLAGS="${WAILS_BUILD_FLAGS:-}"
MACOS_ARCH="${MACOS_PLATFORM##*/}"
ARTIFACT_PREFIX="${MACOS_ARTIFACT_PREFIX:-${ARTIFACT_PREFIX:-$(build_artifact_prefix "$APP_NAME" "$VERSION" "macos" "$MACOS_ARCH")}}"

# Optional signing/notarization settings (recommended for distribution)
MACOS_SIGN_IDENTITY="${MACOS_SIGN_IDENTITY:-}"
MACOS_NOTARY_PROFILE="${MACOS_NOTARY_PROFILE:-}"

log() { printf '\n[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }
need_cmd() { command -v "$1" >/dev/null 2>&1 || { echo "Missing command: $1" >&2; exit 1; }; }

need_cmd wails
need_cmd hdiutil
need_cmd ditto

log "Cleaning previous macOS artifacts: $OUT_DIR"
clean_output_dir "$OUT_DIR" "$ROOT_DIR"

log "Building macOS app with Wails (platform: $MACOS_PLATFORM)"
wails build -clean -platform "$MACOS_PLATFORM" ${WAILS_BUILD_FLAGS}

APP_PATH="$ROOT_DIR/build/bin/${APP_NAME}.app"
if [[ ! -d "$APP_PATH" ]]; then
  echo "Expected app not found: $APP_PATH" >&2
  echo "Please check APP_NAME or Wails output." >&2
  exit 1
fi

if [[ -n "$MACOS_SIGN_IDENTITY" ]]; then
  need_cmd codesign
  log "Signing app bundle for DMG packaging: $APP_PATH"
  codesign --force --deep --options runtime --timestamp --sign "$MACOS_SIGN_IDENTITY" "$APP_PATH"
  codesign --verify --deep --strict --verbose=2 "$APP_PATH"
else
  log "No MACOS_SIGN_IDENTITY set. Skip signing (OK for local/internal testing)."
fi

DMG_PATH="$OUT_DIR/${ARTIFACT_PREFIX}.dmg"
DMG_STAGING_DIR="$(mktemp -d)"
trap 'rm -rf "$DMG_STAGING_DIR"' EXIT

log "Preparing DMG staging directory"
ditto "$APP_PATH" "$DMG_STAGING_DIR/${APP_NAME}.app"
ln -s /Applications "$DMG_STAGING_DIR/Applications"

# Add a double-clickable authorization helper in DMG.
AUTH_SCRIPT="$DMG_STAGING_DIR/Authorize ${APP_NAME}.command"
cat > "$AUTH_SCRIPT" <<EOS
#!/usr/bin/env bash
set -Eeuo pipefail

APP_NAME="${APP_NAME}"
SCRIPT_DIR="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"
DMG_APP="\$SCRIPT_DIR/\${APP_NAME}.app"
INSTALLED_APP="/Applications/\${APP_NAME}.app"

echo "== ${APP_NAME} Authorization =="
echo

authorize_target() {
  local target="\$1"
  if [[ -e "\$target" ]]; then
    echo "[1/2] Removing quarantine: \$target"
    xattr -dr com.apple.quarantine "\$target" || true
    echo "Done: \$target"
    echo
  fi
}

authorize_target "\$DMG_APP"
authorize_target "\$INSTALLED_APP"

echo "Authorization completed."
echo "You can now open ${APP_NAME}.app."
echo
read -r -p "Press Enter to exit..."
EOS
chmod +x "$AUTH_SCRIPT"

log "Creating DMG: $DMG_PATH"
rm -f "$DMG_PATH"
hdiutil create -volname "$APP_NAME $VERSION" -srcfolder "$DMG_STAGING_DIR" -ov -format UDZO "$DMG_PATH" >/dev/null

if [[ -n "$MACOS_SIGN_IDENTITY" ]]; then
  log "Signing DMG"
  codesign --force --timestamp --sign "$MACOS_SIGN_IDENTITY" "$DMG_PATH"
  codesign --verify --verbose=2 "$DMG_PATH"
fi

if [[ -n "$MACOS_NOTARY_PROFILE" ]]; then
  need_cmd xcrun
  log "Submitting DMG for notarization (profile: $MACOS_NOTARY_PROFILE)"
  xcrun notarytool submit "$DMG_PATH" --keychain-profile "$MACOS_NOTARY_PROFILE" --wait
  log "Stapling notarization ticket"
  xcrun stapler staple "$DMG_PATH"
else
  log "No MACOS_NOTARY_PROFILE set. Skip notarization."
fi

log "Removing intermediate app bundle: $APP_PATH"
rm -rf "$APP_PATH"

log "Build complete"
echo "DMG: $DMG_PATH"

echo
log "Local authorization command (for internal distribution)"
echo "xattr -dr com.apple.quarantine \"$DMG_PATH\""
