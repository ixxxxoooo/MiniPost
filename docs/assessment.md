# MiniPost 评估与改进路线图

> 文档版本：2026-06-13  
> 定位：轻量版 Postman / Bruno / Insomnia，macOS 优先桌面 API 调试工具  
> 技术栈：Wails + Go + React + TypeScript + Tailwind + shadcn/ui

---

## 1. 执行摘要

MiniPost 已完成 PRD 中定义的 MVP，并在多个维度超出预期：具备完整的 HTTP 调试链路、Postman/OpenAPI 导入导出、SSE 流式响应、TLS/耗时分解、Cookie 管理、自动备份等能力。后端核心模块（HTTP 发送、导入导出、cURL 解析）有较充分的单元测试。

当前主要短板集中在**数据持久化可靠性**、**请求取消与连接复用**、**测试与 CI 覆盖盲区**，以及相对业界产品的**变量体系、认证类型、批量执行、命令面板**等功能缺口。

**建议优先级**：先完成阶段一可靠性修复，再推进性能与功能补齐。

---

## 2. 已实现功能清单

### 2.1 项目管理

| 功能 | 状态 | 实现位置 |
|------|------|----------|
| 新建 / 删除 / 重命名项目 | ✅ | `app.go` → `ProjectService` |
| 项目切换与主题色 | ✅ | `frontend/src/stores/projectStore.ts` |
| 项目描述 | ✅ | `app.go` `UpdateProjectDescription` |
| 本地 JSON 持久化 | ✅ | `internal/repository/file_store.go` |
| 项目导出（Postman v2.1） | ✅ | `internal/service/request_service.go` `ExportProjectJSON` |

### 2.2 集合与请求管理

| 功能 | 状态 | 实现位置 |
|------|------|----------|
| 嵌套文件夹 | ✅ | `file_store.go` + `tree.json` |
| 拖拽排序 / 移动节点 | ✅ | `MoveCollectionNode` |
| 请求 CRUD / 重命名 / 复制 | ✅ | `RequestService` |
| 文件夹复制（含子请求） | ✅ | `DuplicateFolder` |
| 多标签页编辑 | ✅ | `frontend/src/stores/tabStore.ts` |
| 未保存变更检测 | ✅ | `frontend/src/stores/requestStore.ts` `isDirty` |

### 2.3 HTTP 请求能力

| 功能 | 状态 | 说明 |
|------|------|------|
| GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS | ✅ | `frontend/src/lib/constants.ts` |
| URL / Params / Headers / Body / Auth | ✅ | `RequestEditor` 系列组件 |
| Body：none / raw / json / form-urlencoded / form-data（含文件） | ✅ | Go `buildBody` |
| Auth：none / basic / bearer / api-key | ✅ | Go `applyAuth` |
| 请求选项：超时、重定向、SSL、HTTP 版本、响应大小限制 | ✅ | `RequestOptions` + `uiStore` |
| cURL 粘贴解析 | ✅ | `internal/pkg/httputil/curl_parser.go` |
| 环境变量 `{{var}}` 解析 | ✅ | 前后端各有一套 resolver |
| Cookie 自动携带与吸收 | ✅ | `cookieStore` + `httpService.ts` |
| 自动 Header（User-Agent、Accept、Cache-Control、Token） | ✅ | `httpService.ts` |

### 2.4 响应查看

| 功能 | 状态 | 说明 |
|------|------|------|
| 状态码 / 耗时 / 大小 | ✅ | `StatusBar` |
| Timing 分解（DNS/TCP/TLS/TTFB/Download） | ✅ | `http_service.go` `buildTimingBreakdown` |
| 网络详情（协议、地址、证书 CN/颁发者/有效期） | ✅ | `buildNetworkDetails` |
| Headers / Cookies 视图 | ✅ | `ResponseHeaders` / `ResponseCookies` |
| Body：Pretty / Raw / Preview / 流式（SSE） | ✅ | `ResponseBody` / `ResponseStream` |
| 二进制响应识别与下载 | ✅ | `shouldTreatResponseAsBinary` |
| 响应搜索与复制 | ✅ | `ResponseViewer` |
| TLS 证书警告（关闭校验时） | ✅ | `detectTLSWarning` |

### 2.5 环境变量与历史

| 功能 | 状态 | 说明 |
|------|------|------|
| 多环境（dev/test/prod 等） | ✅ | `environments.json` |
| Secret 类型变量（前端脱敏） | ✅ | `EnvironmentManager` |
| 删除环境确认 | ✅ | `environmentDeleteConfirm.ts` |
| 请求历史（最近 500 条） | ✅ | `history.json` |
| 历史恢复 | ✅ | `HistoryPanel` |

### 2.6 导入导出

| 功能 | 状态 | 说明 |
|------|------|------|
| 导入 Postman Collection v2.1 | ✅ | 含 JSONC 注释支持 |
| 导入 Postman Environment | ✅ | |
| 导入 OpenAPI 3.x / Swagger 2.0 | ✅ | 含 YAML、远程 URL、Swagger UI HTML 解析 |
| 导入冲突预检与策略（update/copy/overwrite） | ✅ | `ImportPreview` + `request_import_strategy.go` |
| 导出 Postman Collection | ✅ | |
| 导出 cURL | ✅ | `SaveTextFile` |
| 从 URL 拉取规范 | ✅ | `fetchRemoteImportContent` |

### 2.7 桌面体验与工程化

| 功能 | 状态 | 说明 |
|------|------|------|
| macOS 原生感 UI / 深色模式 | ✅ | Tailwind + CSS 变量 |
| 侧边栏折叠 / 面板 resize | ✅ | `uiStore` |
| 快捷键（⌘+Enter 发送、⌘+S 保存等） | ✅ | `useKeyboardShortcuts` |
| 窗口位置/大小持久化 | ✅ | `App.tsx` localStorage |
| 手动 / 自动备份（ZIP） | ✅ | `backup.go` + `backupService` |
| 结构化日志 + 敏感信息脱敏 | ✅ | `internal/pkg/logger/` |
| 错误边界 | ✅ | `ErrorBoundary.tsx` |
| i18n（中英） | ✅ | `useI18n` |
| macOS / Windows 构建脚本 | ✅ | `scripts/` |

### 2.8 PRD 预留但未实现

| 功能 | 状态 |
|------|------|
| Command Palette（命令面板） | ❌ 未实现 |
| 云同步 | ❌ PRD 明确不做 |
| OpenAPI 导出 | ❌ 仅导入 |
| HAR 导入 | ❌ |

---

## 3. 业界对标矩阵

对比对象：Postman、Insomnia、Bruno、Hoppscotch。

| 能力维度 | MiniPost | Postman | Insomnia | Bruno | Hoppscotch |
|----------|----------|---------|----------|-------|------------|
| 桌面原生 / 轻量 | ✅ 强 | ❌ 重 | ✅ | ✅ 强 | ❌ Web |
| 多项目 / 文件夹树 | ✅ | ✅ | ✅ | ✅（Git） | ✅ |
| 环境变量 `{{var}}` | ✅ 基础 | ✅ 完整 | ✅ | ✅ | ✅ |
| 动态变量（`$randomUUID` 等） | ❌ | ✅ | ✅ | ✅ | ✅ |
| 响应→变量提取 | ❌ | ✅ | ✅ | ✅ | 部分 |
| 文件夹/集合级 Auth 继承 | ❌ | ✅ | ✅ | ✅ | ❌ |
| 全局变量 | ❌ | ✅ | ✅ | ✅ | ✅ |
| Basic / Bearer / API Key | ✅ | ✅ | ✅ | ✅ | ✅ |
| OAuth2 / Digest / AWS 签名 | ❌ | ✅ | ✅ | 部分 | 部分 |
| GraphQL 专用编辑器 | ❌ | ✅ | ✅ | ✅ | ✅ |
| WebSocket / gRPC | ❌ | ✅ | ✅ | 部分 | ✅ |
| Collection Runner（批量执行） | ❌ | ✅ | ✅ | ✅（CLI） | ❌ |
| 测试脚本（pre/post-request） | ❌ | ✅ | ✅ | ✅ | ❌ |
| Mock Server | ❌ | ✅ | ✅ | ❌ | ❌ |
| Postman 导入导出 | ✅ | — | ✅ | ✅ | ✅ |
| OpenAPI 导入 | ✅ | ✅ | ✅ | ✅ | ✅ |
| OpenAPI 导出 | ❌ | ✅ | ✅ | 部分 | ❌ |
| cURL 导入导出 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 代码生成（多语言） | ❌ | ✅ | ✅ | 部分 | ✅ |
| 代理 / 客户端证书 | ❌ | ✅ | ✅ | 部分 | 部分 |
| 响应对比 / 保存示例 | ❌ | ✅ | ✅ | 部分 | ❌ |
| 命令面板 / 全局搜索 | ❌ | ✅ | ✅ | 部分 | 部分 |
| 协作 / 云同步 | ❌ | ✅ | ✅ | Git | ✅ |
| 本地文件存储 | ✅ | 部分 | ✅ | ✅ | ❌ |
| SSE / 流式响应 | ✅ | 部分 | 部分 | ❌ | 部分 |
| Timing / TLS 详情 | ✅ 详细 | ✅ | 部分 | 部分 | 部分 |
| 自动备份 | ✅ | ❌ | ❌ | Git | ❌ |

**差异化优势**：轻量桌面、本地优先、SSE 流式、详细 timing/TLS、自动备份、导入冲突策略。  
**主要差距**：变量体系深度、认证类型、批量执行、代码生成、命令面板。

---

## 4. 代码可靠性评估

### 4.1 严重问题

#### P0-1：JSON 持久化非原子写入

- **位置**：[internal/repository/file_store.go](internal/repository/file_store.go) 第 863–874 行 `writeJSON`
- **现状**：直接 `os.WriteFile(path, data, 0644)` 覆盖目标文件
- **风险**：写入中途崩溃或断电会导致 `project.json`、`tree.json`、`requests.json` 等损坏，项目数据不可恢复
- **对比**：[backup.go](backup.go) 第 274 行恢复备份时已使用 `os.Rename` 做安全替换，主存储层未采用同样策略
- **建议**：`path.tmp` → `fsync` → `os.Rename`；失败时保留原文件

#### P0-2：后端 HTTP 请求无法真正取消

- **位置**：
  - 后端：[internal/service/http_service.go](internal/service/http_service.go) `sendRequest` / `client.Do(req)`
  - 前端：[frontend/src/components/business/editor/RequestEditor.tsx](frontend/src/components/business/editor/RequestEditor.tsx) 第 281–298 行 `AbortController`
- **现状**：前端取消仅停止 UI 等待；Go 端 `http.Client.Do` 未绑定可取消 `context`，连接与 body 读取仍会继续
- **风险**：用户取消后仍占用连接与带宽；大响应或慢接口无法及时释放资源
- **建议**：按 `streamId` 维护 `context.CancelFunc` 映射；新增 `CancelRequest(streamId)` Wails 绑定；`http.NewRequestWithContext` 贯穿请求生命周期

### 4.2 中等问题

#### P1-1：HTTP Transport 每请求重建

- **位置**：[internal/service/http_service.go](internal/service/http_service.go) 第 737–766 行 `buildClient`
- **现状**：每次请求 `http.DefaultTransport.Clone()`，无法复用 TCP/TLS 连接
- **影响**：高频调试时延迟偏高，无法利用 HTTP keep-alive
- **建议**：按 `(sslVerify, httpVersion, followRedirects)` 签名缓存 `*http.Transport`

#### P1-2：大响应 / 大文件全量进内存

- **位置**：
  - 响应：[internal/service/http_service.go](internal/service/http_service.go) 第 769–792 行 `readResponseBody` 使用 `io.ReadAll`
  - 上传：第 633–675 行 form-data 使用 `bytes.Buffer` 全量读文件
- **现状**：`maxResponseSizeMB` 默认 0 表示无限制（[http_service.go](internal/service/http_service.go) 第 698–701 行）
- **风险**：超大响应或上传大文件可能导致内存飙升甚至 OOM
- **建议**：设置合理默认上限（如 50MB）；大文件上传考虑 `io.Pipe` 流式；响应预览对大 body 做截断提示

#### P1-3：环境 Secret 变量明文落盘

- **位置**：[internal/repository/file_store.go](internal/repository/file_store.go) `environments.json`
- **现状**：`isSecret` 仅控制前端脱敏显示，磁盘 JSON 为明文
- **风险**：本地备份、日志、文件共享时可能泄露 token/password
- **建议**：macOS Keychain / Windows Credential Manager 或本地 AES 加密层

#### P1-4：变量解析能力有限

- **位置**：[internal/pkg/httputil/variable_resolver.go](internal/pkg/httputil/variable_resolver.go)
- **问题**：
  - 简单 `strings.ReplaceAll`，不支持嵌套 `{{outer-{{inner}}}}`
  - 无未解析变量检测与 UI 告警
  - `ResolveKeyValues` 丢失 `Description` 字段（第 22–30 行）
  - 前后端各有一套解析逻辑（`frontend/src/lib/variableResolver.ts`），存在漂移风险
- **建议**：统一解析器规格；增加未解析占位符扫描；保留 Description

#### P1-5：错误分类依赖字符串匹配

- **位置**：[internal/service/http_service.go](internal/service/http_service.go) 第 923–961 行 `classifyRequestSendError`
- **现状**：通过 `err.Error()` 子串判断 DNS/超时/TLS 等类型
- **风险**：Go 版本或错误文案变化时分类失效
- **建议**：优先 `errors.As` 匹配 `*net.DNSError`、`*url.Error`、`*tls.CertificateVerificationError` 等

### 4.3 低优先级问题

#### P2-1：Schema 版本无迁移框架

- **位置**：[internal/repository/file_store.go](internal/repository/file_store.go) 第 15–17 行 `schemaVersion = 1`
- **现状**：写入时设置版本号，但无升级/迁移逻辑
- **建议**：引入 `migrate.go`，按版本链式升级

#### P2-2：`app.go` 体量过大

- **位置**：[app.go](app.go)（约 1000 行）
- **现状**：Wails 绑定、导入逻辑、远程拉取、JSONC 处理混在一起
- **建议**：拆分为 `app_http.go`、`app_import.go` 等，便于测试与维护

#### P2-3：集合数据三文件冗余

- **位置**：`folders.json` + `requests.json` + `tree.json`
- **现状**：`syncEntitiesWithTree` 每次读写需同步三份
- **风险**：逻辑复杂，边界情况（如 tree 与 entity 不一致）难排查
- **建议**：长期考虑单一 source of truth；短期加强集成测试

---

## 5. 测试与 CI 现状

### 5.1 后端测试（较好）

| 模块 | 测试文件 | 覆盖要点 |
|------|----------|----------|
| HTTP 发送 | `http_service_test.go` | 重定向、超时、大小限制、timing、cURL |
| Postman 导入 | `request_service_import_test.go` | 集合、环境、JSONC、form-data |
| 导入策略 | `request_import_strategy_test.go` | update/copy/overwrite、冲突预检 |
| Swagger 导入 | `request_service_import_test.go` | OpenAPI3、Swagger2、schema |
| cURL 解析 | `curl_parser_test.go` | 多行、form-data、cookie、转义 |
| 导出 | `request_service_export_test.go` | Postman 树顺序、query 拼接 |
| 日志脱敏 | `logger_test.go` | URL/cURL 脱敏 |
| App 导入 | `app_import_test.go` | 自动检测格式、Swagger UI HTML |

### 5.2 测试盲区（需补齐）

| 模块 | 缺失 | 优先级 |
|------|------|--------|
| `file_store.go` | 无单测：原子写、move/delete 树完整性、并发锁 | P0 |
| `variable_resolver.go` | 无单测：嵌套、未解析、Description 保留 | P1 |
| `environment_service.go` | 无单测 | P2 |
| `history_service.go` | 无单测（500 条截断） | P2 |
| 前端 stores / httpService | 仅 4 个 lib/store 单测，核心发送链路无测 | P1 |
| E2E / 集成 | 无 Wails 端到端测试 | P2 |

### 5.3 CI

- **现状**：仓库无 `.github/workflows`，无自动化构建/测试门禁
- **建议**：最小 CI 跑 `go test ./...` + `npm test`（前端）+ `go vet`

---

## 6. 分阶段改进路线

### 阶段一：可靠性（强烈建议优先）

| 任务 | 改动文件 | 验收标准 |
|------|----------|----------|
| 原子 JSON 写入 | `file_store.go` | 写入中断模拟后原文件完好；单测覆盖 |
| 后端请求取消 | `http_service.go`、`app.go`、前端 `httpService.ts` | 点击取消后 Go 端连接在 1s 内断开；取消大响应下载 |
| `file_store` 单测 | `file_store_test.go` | 覆盖 save/move/delete/tree 同步 |
| `variable_resolver` 单测 | `variable_resolver_test.go` | 覆盖基础替换、未解析检测 |
| 最小 CI | `.github/workflows/ci.yml` | PR 自动跑 go test + frontend test |

**预估工作量**：2–3 天

### 阶段二：性能与正确性

| 任务 | 改动文件 | 验收标准 |
|------|----------|----------|
| Transport 连接池复用 | `http_service.go` | 同 host 连续请求复用连接（可用测试 server 验证） |
| 响应大小默认上限 | `http_service.go`、`uiStore` | 默认 50MB；超限友好提示 |
| 未解析变量告警 | `variable_resolver.go`、前端 UrlBar | 发送前高亮 `{{unknown}}` |
| Secret 加密存储 | `file_store.go` 或新 `secrets` 包 | 磁盘无明文 secret；导入导出兼容 |

**预估工作量**：3–5 天

### 阶段三：高价值功能

| 任务 | 验收标准 |
|------|----------|
| 响应→环境变量提取 | 右键 JSON 字段可写入当前环境 |
| 动态变量 | 支持 `{{$timestamp}}`、`{{$randomUUID}}` 等 |
| 文件夹级公共 Header/Auth | 子请求继承并可覆盖 |
| 全局变量 | 跨项目共享变量层 |
| 命令面板 + 全局搜索 | ⌘+K 搜索请求/文件夹/操作 |

**预估工作量**：1–2 周

### 阶段四：按需扩展

- 代码生成（curl / fetch / axios / Python requests）
- OAuth2 Authorization Code / Client Credentials
- GraphQL 专用 Body 编辑器
- Collection Runner（顺序执行 + 简单断言）
- OpenAPI 导出、HAR 导入
- WebSocket 调试（视产品定位决定）

---

## 7. 阶段一任务拆分（已确认并实施）

用户已于 2026-06-13 确认执行阶段一全部项，实施结果如下：

| 任务 | 状态 | 说明 |
|------|------|------|
| 原子 JSON 写入 | ✅ 已完成 | `file_store.writeJSON` 改为 tmp + sync + rename |
| 后端请求取消 | ✅ 已完成 | `HttpService.CancelRequest` + `App.CancelHTTPRequest` + 前端贯通 |
| `file_store` 单测 | ✅ 已完成 | `internal/repository/file_store_test.go` |
| `variable_resolver` 单测 + Description 修复 | ✅ 已完成 | `variable_resolver_test.go`，保留 Description |
| 最小 CI | ✅ 已完成 | `.github/workflows/ci.yml` |

### 原任务拆分（供后续提交参考）

1. **feat: file_store 原子写入 + 单测** — 已合并实施
2. **feat: HTTP 请求后端取消** — 已合并实施
3. **test: variable_resolver 单测 + Description 修复** — 已合并实施
4. **ci: 添加 GitHub Actions 工作流** — 已合并实施

### 阶段二（下一步）

- Transport 连接池复用
- 响应大小默认上限
- 未解析变量告警
- Secret 加密存储

---

## 8. 结论

MiniPost 已具备作为日常 API 调试工具的完整能力，代码结构清晰、后端核心路径测试充分。下一步应**先夯实可靠性**（原子写入、真取消、存储层测试、CI），再按产品定位补齐变量体系与命令面板等高价值功能，避免在数据安全与连接管理上留隐患。

---

## 附录：关键文件索引

| 领域 | 关键文件 |
|------|----------|
| Wails 绑定 | `app.go` |
| HTTP 引擎 | `internal/service/http_service.go` |
| 导入导出 | `internal/service/request_service.go` |
| 本地存储 | `internal/repository/file_store.go` |
| 变量解析 | `internal/pkg/httputil/variable_resolver.go` |
| 前端发送 | `frontend/src/services/httpService.ts` |
| 请求编辑 | `frontend/src/components/business/editor/RequestEditor.tsx` |
| 状态管理 | `frontend/src/stores/*.ts` |
| 备份 | `backup.go` |
| 产品需求 | `prd.md` |
