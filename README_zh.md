# MiniPost

<p align="left">
  <a href="README.md">English</a> | <b>简体中文</b>
</p>

MiniPost 是一款仅支持 macOS 的轻量级 API 调试工具。2.0 起界面使用 [MyGo](https://github.com/egoist/mygo) 编写:纯 Go、GPU 绘制的原生 UI,不再依赖 WebView 与前端工具链。

## 1. 环境要求

- macOS 12 (Monterey) 及以上(由 `mygo.json` 的 `macos.minimumSystemVersion` 指定)
- Go 1.27+
- MyGo CLI:

```bash
go install github.com/egoist/mygo/cmd/mygo@latest
```

## 2. 本地开发

```bash
./scripts/dev.sh        # 构建并启动 "MiniPost Dev"
go test ./...           # 全部测试,含无头 UI 测试(ui.Tester)
```

代码结构:

- `main.go`、`app*.go`:程序入口与后端门面(`main.App`)
- `internal/gui`:原生界面(侧栏、标签页、请求编辑器、响应查看器、各类对话框),通过 `Backend` 接口访问后端,测试中可替换为假实现
- `internal/client`:纯 Go 客户端逻辑(URL/Params 同步、变量、Cookie Jar、自动请求头、cURL、设置与会话持久化)
- `internal/service`、`internal/repository`、`internal/model`、`internal/pkg`:业务逻辑、存储(`~/.minipost`)与工具

## 3. 查看日志

MiniPost 同时把日志输出到终端和本地文件:

- 日志目录:`~/.minipost/logs/`
- 按天日志文件:`~/.minipost/logs/minipost-YYYY-MM-DD.log`
- 实时查看:`tail -f ~/.minipost/logs/minipost-$(date +%F).log`

默认日志级别为 `DEBUG`,可通过环境变量调整:

```bash
MINIPOST_LOG_LEVEL=INFO ./scripts/dev.sh
```

支持级别:`DEBUG`、`INFO`、`WARN`、`ERROR`、`FATAL`。

## 4. 构建

```bash
./scripts/build.sh                            # 当前架构:.app + .dmg
./scripts/build.sh -skip-dmg                  # 仅 .app
./scripts/build.sh -platform darwin/universal # arm64 + amd64 合并为一个 app
```

产物输出到 `build/`:

- `build/darwin-<arch>/MiniPost.app`
- `build/minipost-<version>-<arch>.dmg`

版本号取自 `mygo.json`。

## 5. 签名与公证

默认使用 ad-hoc 签名(本机使用足够)。正式分发时,在 `mygo.json` 中配置 `macos.signingIdentity` 与 `macos.notarize`(或通过 `-sign "Developer ID Application: Your Company (TEAMID)"` 传入),公证凭据配置参见 MyGo 文档。

## 6. 未公证应用的授权(内部分发)

移除隔离属性:

```bash
xattr -dr com.apple.quarantine "build/darwin-arm64/MiniPost.app"
# 或
./scripts/macos-authorize.sh "build/darwin-arm64/MiniPost.app"
```

可选的本机信任(需要管理员权限):

```bash
sudo spctl --add --label "MiniPost Local Trust" "build/darwin-arm64/MiniPost.app"
```
