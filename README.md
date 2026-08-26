# MiniPost

<p align="left">
  <b>English</b> | <a href="README_zh.md">简体中文</a>
</p>

MiniPost is a lightweight desktop API debugging tool built with Wails + React + TypeScript.

## 1. Prerequisites

- Go 1.22+
- Node.js 20+
- Wails CLI 2.11+

Install Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 2. Local Development

```bash
wails dev
```

## 3. Viewing Logs

MiniPost outputs backend logs to both the terminal and local log files simultaneously:

- Log directory: `~/.minipost/logs/`
- Daily log file: `~/.minipost/logs/minipost-YYYY-MM-DD.log`
- Real-time monitoring: `tail -f ~/.minipost/logs/minipost-$(date +%F).log`

The default log level is `DEBUG` for easier development and troubleshooting. You can customize it via environment variables:

```bash
MINIPOST_LOG_LEVEL=INFO wails dev
MINIPOST_LOG_LEVEL=WARN ./scripts/build-macos.sh
```

Supported log levels: `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`.

## 4. Build Artifact Directory Conventions

- `build/bin/`: Raw Wails build output
- `dist/macos/`: macOS release artifacts (`.app` + `.dmg`)
- `dist/windows/`: Windows release artifacts (`.exe` / installer)
- Each build will first clean the corresponding platform's `dist` output directory to prevent stale artifacts from mixing into the current release.
- Release artifacts follow a unified naming prefix `MiniPost-<version>-<platform>-<arch>` for clear distinction of versions, operating systems, and architectures.

## 5. Build (Unified Commands)

First, grant execution permissions to the build scripts:

```bash
chmod +x scripts/*.sh
```

### 5.1 Build for macOS (Outputs DMG)

```bash
./scripts/build-macos.sh
```

Default output:

- `dist/macos/MiniPost-<version>-macos-<arch>.app`
- `dist/macos/MiniPost-<version>-macos-<arch>.dmg`
- The DMG includes `Authorize MiniPost.command` (users can double-click to authorize the app)

Optional parameters (Environment Variables):

- `APP_NAME`: Application name (default: `MiniPost`)
- `VERSION`: DMG version (default: automatically retrieved via `git describe`)
- `MACOS_PLATFORM`: `darwin/arm64` or `darwin/amd64`
- `WAILS_BUILD_FLAGS`: Additional Wails build flags
- `OUT_DIR`: Output directory (default: `dist/macos`)
- `MACOS_ARTIFACT_PREFIX` / `ARTIFACT_PREFIX`: Custom release artifact filename prefix

### 5.2 Build for Windows

```bash
./scripts/build-windows.sh
```

Default output:

- `dist/windows/MiniPost-<version>-windows-<arch>-portable.exe`
- `dist/windows/MiniPost-<version>-windows-<arch>-setup.exe`
- `dist/windows/MiniPost-<version>-windows-<arch>-setup.msi` (if applicable)

Optional parameters (Environment Variables):

- `VERSION`: Version string (default: automatically retrieved via `git describe`)
- `WINDOWS_PLATFORM`: Target platform (default: `windows/amd64`)
- `WEBVIEW2_STRATEGY`: WebView2 distribution strategy (default: `download`)
- `WAILS_BUILD_FLAGS`: Additional Wails build flags
- `OUT_DIR`: Output directory (default: `dist/windows`)
- `WINDOWS_ARTIFACT_PREFIX` / `ARTIFACT_PREFIX`: Custom release artifact filename prefix

Notes:

- Cross-compiling Windows binaries on macOS/Linux requires a cross compiler. The script prefers `zig`, followed by `x86_64-w64-mingw32-gcc`.
- Industry best practice recommends using CI native runners for macOS and Windows respectively to minimize cross-compilation inconsistency risks.

### 5.3 Build for Both Platforms

```bash
./scripts/build.sh
```

Optional filters:

```bash
SKIP_WINDOWS=1 ./scripts/build.sh
SKIP_MACOS=1 ./scripts/build.sh
```

## 6. macOS Release Best Practices (Code Signing + Notarization)

For release builds, the following process is recommended:

1. Sign the `.app` and `.dmg` using a `Developer ID Application` certificate
2. Submit the `.dmg` for notarization using `notarytool`
3. Staple the notarization ticket to the `.dmg` (and standalone `.app`)

The build script has built-in support; simply set the following environment variables:

- `MACOS_SIGN_IDENTITY` e.g., `Developer ID Application: Your Company (TEAMID)`
- `MACOS_NOTARY_PROFILE` e.g., `AC_NOTARY_PROFILE`

Example:

```bash
export MACOS_SIGN_IDENTITY="Developer ID Application: Your Company (TEAMID)"
export MACOS_NOTARY_PROFILE="AC_NOTARY_PROFILE"
./scripts/build-macos.sh
```

Configure notary profile (one-time setup):

```bash
xcrun notarytool store-credentials "AC_NOTARY_PROFILE" \
  --apple-id "your-apple-id@example.com" \
  --team-id "TEAMID" \
  --password "app-specific-password"
```

## 7. Direct Authorization Command (For Internal Distribution)

For quarantine flags caused by non-notarized apps or internal network distribution, run:

```bash
xattr -dr com.apple.quarantine "dist/macos/MiniPost-<version>-macos-<arch>.app"
```

Or run the script:

```bash
./scripts/macos-authorize.sh "dist/macos/MiniPost-<version>-macos-<arch>.app"
```

For users distributing via DMG:

- After opening the DMG, double-click `Authorize MiniPost.command` to perform authorization.

Optional local trust enhancement (requires administrator privileges):

```bash
sudo spctl --add --label "MiniPost Local Trust" "dist/macos/MiniPost.app"
```

## 8. Windows Code Signing Best Practices (Optional)

The script supports optional `signtool` signing. Set the following environment variables on a Windows host:

- `WIN_CERT_FILE`: Path to `pfx` certificate
- `WIN_CERT_PASSWORD`: Certificate password
- `WIN_SIGNTOOL`: Sign tool path/binary (default: `signtool`)
- `WIN_TIMESTAMP_URL`: Timestamp server URL (default: `http://timestamp.digicert.com`)

Example:

```bash
WIN_CERT_FILE="C:/certs/release.pfx" \
WIN_CERT_PASSWORD="***" \
./scripts/build-windows.sh
```

## 9. Quick Command Reference

```bash
# Development
wails dev

# macOS release (dmg)
./scripts/build-macos.sh

# Windows release
./scripts/build-windows.sh

# Dual-platform release
./scripts/build.sh
```
