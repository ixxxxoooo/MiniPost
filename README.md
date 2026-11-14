# MiniPost

<p align="left">
  <b>English</b> | <a href="README_zh.md">简体中文</a>
</p>

MiniPost is a lightweight, macOS-only API debugging tool. Since 2.0 the UI is written with [MyGo](https://github.com/egoist/mygo): pure Go, GPU-drawn native UI, no WebView and no frontend toolchain.

## 1. Requirements

- macOS 12 (Monterey) or later (set by `macos.minimumSystemVersion` in `mygo.json`)
- Go 1.27+
- MyGo CLI:

```bash
go install github.com/egoist/mygo/cmd/mygo@latest
```

## 2. Local Development

```bash
./scripts/dev.sh        # builds and launches "MiniPost Dev"
go test ./...           # all tests, including headless UI tests (ui.Tester)
```

Code layout:

- `main.go`, `app*.go`: entry point and the backend facade (`main.App`)
- `internal/gui`: native UI (sidebar, tabs, request editor, response viewer, dialogs); talks to the backend through the `Backend` interface so it can be tested with a fake
- `internal/client`: pure-Go client logic (URL/Params sync, variables, cookie jar, auto headers, cURL, settings and session persistence)
- `internal/service`, `internal/repository`, `internal/model`, `internal/pkg`: business logic, storage (`~/.minipost`) and utilities

## 3. Viewing Logs

MiniPost writes logs to both the terminal and local files:

- Log directory: `~/.minipost/logs/`
- Daily log file: `~/.minipost/logs/minipost-YYYY-MM-DD.log`
- Real-time monitoring: `tail -f ~/.minipost/logs/minipost-$(date +%F).log`

The default log level is `DEBUG`. Override it with an environment variable:

```bash
MINIPOST_LOG_LEVEL=INFO ./scripts/dev.sh
```

Supported levels: `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`.

## 4. Build

```bash
./scripts/build.sh                            # current arch: .app + .dmg
./scripts/build.sh -skip-dmg                  # .app only
./scripts/build.sh -platform darwin/universal # arm64 + amd64 in one app
```

Artifacts are written to `build/`:

- `build/darwin-<arch>/MiniPost.app`
- `build/minipost-<version>-<arch>.dmg`

The version comes from `mygo.json`.

## 5. Signing and Notarization

By default the app is ad-hoc signed (fine for local use). For distribution, set `macos.signingIdentity` and `macos.notarize` in `mygo.json` (or pass `-sign "Developer ID Application: Your Company (TEAMID)"`) and follow the MyGo documentation for notarization credentials.

## 6. Authorizing an Unnotarized App (Internal Distribution)

Remove the quarantine flag:

```bash
xattr -dr com.apple.quarantine "build/darwin-arm64/MiniPost.app"
# or
./scripts/macos-authorize.sh "build/darwin-arm64/MiniPost.app"
```

Optional local trust (requires administrator privileges):

```bash
sudo spctl --add --label "MiniPost Local Trust" "build/darwin-arm64/MiniPost.app"
```
