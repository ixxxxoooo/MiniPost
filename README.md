# MiniPost

MiniPost 是一个基于 Wails + React + TypeScript 的桌面 API 调试工具。

## 1. 环境要求

- Go 1.22+
- Node.js 20+
- Wails CLI 2.11+

安装 Wails CLI：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 2. 本地开发

```bash
wails dev
```

## 3. 日志查看

MiniPost 会同时把后端日志输出到终端和本地日志文件：

- 日志目录：`~/.minipost/logs/`
- 当天日志文件：`~/.minipost/logs/minipost-YYYY-MM-DD.log`
- 实时查看：`tail -f ~/.minipost/logs/minipost-$(date +%F).log`

默认日志级别为 `DEBUG`，方便开发和排查问题。可以通过环境变量调整：

```bash
MINIPOST_LOG_LEVEL=INFO wails dev
MINIPOST_LOG_LEVEL=WARN ./scripts/build-macos.sh
```

支持的级别：`DEBUG`、`INFO`、`WARN`、`ERROR`、`FATAL`。

## 4. 产物目录约定

- `build/bin/`：Wails 原始构建输出
- `dist/macos/`：macOS 发布产物（`.app` + `.dmg`）
- `dist/windows/`：Windows 发布产物（`.exe` / 安装包）
- 每次构建会先清空对应平台的 `dist` 输出目录，避免旧产物混入本次发布。
- 发布产物统一命名为 `MiniPost-<version>-<platform>-<arch>` 前缀，便于区分版本、系统和架构。

## 5. 构建（统一一条命令）

先赋予脚本执行权限：

```bash
chmod +x scripts/*.sh
```

### 5.1 构建 macOS（输出 DMG）

```bash
./scripts/build-macos.sh
```

默认输出：

- `dist/macos/MiniPost-<version>-macos-<arch>.app`
- `dist/macos/MiniPost-<version>-macos-<arch>.dmg`
- DMG 内含 `Authorize MiniPost.command`（用户可双击完成授权）

可选参数（环境变量）：

- `APP_NAME`：应用名（默认 `MiniPost`）
- `VERSION`：DMG 版本号（默认自动取 git describe）
- `MACOS_PLATFORM`：`darwin/arm64` 或 `darwin/amd64`
- `WAILS_BUILD_FLAGS`：附加 Wails 构建参数
- `OUT_DIR`：输出目录（默认 `dist/macos`）
- `MACOS_ARTIFACT_PREFIX` / `ARTIFACT_PREFIX`：自定义发布产物文件名前缀

### 5.2 构建 Windows

```bash
./scripts/build-windows.sh
```

默认输出：

- `dist/windows/MiniPost-<version>-windows-<arch>-portable.exe`
- `dist/windows/MiniPost-<version>-windows-<arch>-setup.exe`
- `dist/windows/MiniPost-<version>-windows-<arch>-setup.msi`（如果有）

可选参数（环境变量）：

- `VERSION`：版本号（默认自动取 `git describe`）
- `WINDOWS_PLATFORM`：默认 `windows/amd64`
- `WEBVIEW2_STRATEGY`：默认 `download`
- `WAILS_BUILD_FLAGS`：附加 Wails 构建参数
- `OUT_DIR`：输出目录（默认 `dist/windows`）
- `WINDOWS_ARTIFACT_PREFIX` / `ARTIFACT_PREFIX`：自定义发布产物文件名前缀

说明：

- 在 macOS/Linux 上构建 Windows 需要交叉编译器。脚本会优先使用 `zig`，其次 `x86_64-w64-mingw32-gcc`。
- 业界最佳实践是使用 CI 的原生 runner 分别构建 macOS 和 Windows，以降低交叉编译不一致风险。

### 5.3 一次构建双平台

```bash
./scripts/build.sh
```

可选：

```bash
SKIP_WINDOWS=1 ./scripts/build.sh
SKIP_MACOS=1 ./scripts/build.sh
```

## 6. macOS 发布最佳实践（签名 + 公证）

建议发布版本必须进行以下流程：

1. 使用 `Developer ID Application` 证书签名 `.app` 和 `.dmg`
2. 使用 `notarytool` 对 `.dmg` 提交公证
3. 对 `.dmg`（和可单独分发的 `.app`）执行 `stapler`

脚本已内置支持，设置以下环境变量即可：

- `MACOS_SIGN_IDENTITY` 例如：`Developer ID Application: Your Company (TEAMID)`
- `MACOS_NOTARY_PROFILE` 例如：`AC_NOTARY_PROFILE`

示例：

```bash
export MACOS_SIGN_IDENTITY="Developer ID Application: Your Company (TEAMID)"
export MACOS_NOTARY_PROFILE="AC_NOTARY_PROFILE"
./scripts/build-macos.sh
```

配置 notary profile（一次性）：

```bash
xcrun notarytool store-credentials "AC_NOTARY_PROFILE" \
  --apple-id "your-apple-id@example.com" \
  --team-id "TEAMID" \
  --password "app-specific-password"
```

## 7. “直接授权”命令（内部分发可用）

对于未公证或内网传输导致的隔离标记，可直接执行：

```bash
xattr -dr com.apple.quarantine "dist/macos/MiniPost-<version>-macos-<arch>.app"
```

或使用脚本：

```bash
./scripts/macos-authorize.sh "dist/macos/MiniPost-<version>-macos-<arch>.app"
```

对于 DMG 内分发用户：

- 打开 DMG 后，双击 `Authorize MiniPost.command` 即可执行授权。

可选增强信任（管理员权限）：

```bash
sudo spctl --add --label "MiniPost Local Trust" "dist/macos/MiniPost.app"
```

## 8. Windows 签名最佳实践（可选）

脚本支持可选 `signtool` 签名，请在 Windows 主机设置：

- `WIN_CERT_FILE`：`pfx` 证书路径
- `WIN_CERT_PASSWORD`：证书密码
- `WIN_SIGNTOOL`：默认 `signtool`
- `WIN_TIMESTAMP_URL`：默认 `http://timestamp.digicert.com`

示例：

```bash
WIN_CERT_FILE="C:/certs/release.pfx" \
WIN_CERT_PASSWORD="***" \
./scripts/build-windows.sh
```

## 9. 常用命令速查

```bash
# 开发
wails dev

# macOS 发布（dmg）
./scripts/build-macos.sh

# Windows 发布
./scripts/build-windows.sh

# 双平台
./scripts/build.sh
```
