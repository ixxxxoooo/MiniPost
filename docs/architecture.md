# MiniPost 架构与分层边界

MiniPost 是 macOS 专用的轻量 HTTP 调试工具，界面是纯 Go 的 MyGo 原生 UI，没有 WebView 和前端工具链。

## 目录与职责

| 目录 | 职责 | 可以依赖 |
| --- | --- | --- |
| `main.go`、`app*.go`、`backup.go` | 进程入口与后端门面 `App`：实现 `gui.Backend`，只做转发、文件对话框、备份恢复 | `internal/*` |
| `internal/gui` | 全部界面与界面状态；只通过 `Backend` 接口访问后端，不依赖 `main.App` | `internal/client`、`internal/model`、`internal/pkg/logger` |
| `internal/client` | 界面侧的纯逻辑：设置、会话、Cookie 罐、请求草稿与 URL/参数同步、响应体格式化与过滤、错误诊断。不碰界面，也不发网络请求 | `internal/model`、`internal/pkg/*` |
| `internal/service` | 后端业务：HTTP 发送（含 TLS、流式、诊断）、项目/请求/环境/历史、Postman 与 OpenAPI 导入导出 | `internal/repository`、`internal/model`、`internal/pkg/*` |
| `internal/repository` | 文件存储（`~/.minipost`）：JSON 原子写入，集合树的增删改查与移动 | `internal/model` |
| `internal/model` | 纯数据结构，无逻辑依赖 | 无 |
| `internal/pkg/*` | 无业务的小工具：日志与脱敏、cURL 解析、变量替换、JSON 工具 | 无 |

依赖只能从上往下，不允许反向或跨层：`gui` 不直接用 `service`/`repository`，`repository` 不依赖 `service`。

## 线程与状态

- `view` 在主线程运行；后台任务用 `a.async`/`a.post` 回到主线程，改变树结构的变更用 `a.later` 延后到帧末。
- 周期性后台任务（自动备份、检查更新）统一用 `startPeriodic`（`internal/gui/periodic.go`），不要各自起 goroutine 加 ticker。
- `a.async` 和 `startPeriodic` 的后台 goroutine 都带 `recover`，panic 只记日志不会让应用崩溃。
- 原生全局调用（`mygo.Clipboard/Dialog/App` 等）必须经过 `a.native` 判断，无头测试里没有真实窗口。

## 数据与权限

- 数据目录 `~/.minipost`：目录 `0700`、JSON 与备份文件 `0600`（环境变量、Cookie 等私密数据）。
- 日志在 `~/.minipost/logs`，权限同上；日志中不记录请求体、令牌、Cookie，URL 经过脱敏。

## 自动更新

- 更新源、公钥写在 `mygo.json` 的 `updates`；发版用 `scripts/release.sh`（本地构建、签名、上传到 GitHub 草稿 release 后发布），更新说明取自 `CHANGELOG.md` 里对应版本的小节。
- 应用启动 10 秒后和之后每小时在后台检查；发现新版本自动下载（设置 → 关于里可关），下载好后提示「重启更新」。逻辑在 `internal/gui/updates.go`。
- 开发构建（`mygo build -skip-dmg` 不带密钥）没有更新能力，`Updater.Enabled()` 为 false，只有发过 release 的版本才能端到端验证。

## 快捷键与右键菜单

- 菜单栏里的加速键走原生菜单（`menu.go`）；不在菜单里的全局键（⌘E/⌘D/⌫ 作用于侧栏选中节点、⌘F 响应体查找、标签切换）集中在 `internal/gui/shortcuts.go`，输入框有焦点或有对话框时不响应。
- 侧栏的「选中节点」是 `a.side.sel`：点击/右键节点或切换标签时更新，快捷键与菜单都作用于它。
- 右键菜单：请求/文件夹（`nodeMenu`）、标签（`tabMenu`）、历史与环境（`internal/gui/menus_extra.go`）、项目卡片（`projectCardMenu`）；编辑器自带编辑菜单。

## 测试

- CI 跑 `go vet`、`staticcheck` 和 `go test -race`。
- 界面行为用 `ui.NewTester` 加 `fakeBackend` 测试（`internal/gui/*_test.go`）；设置 `MINIPOST_SNAPSHOT_DIR` 可输出截图。
- 后端用临时目录的 `FileStore` 做集成式单元测试；备份恢复的测试在根包（`backup_*_test.go`）。
