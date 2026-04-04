---
name: wails-react-desktop-rules
description: 为基于 Wails + React + TypeScript + Tailwind CSS + shadcn/ui 的 macOS 优先桌面应用提供架构、UI、交互与代码输出规范。用于实现桌面 HTTP 调试工具、API 调试器、请求编辑器、响应查看器，或当用户提到 Wails、React、Tailwind、shadcn/ui、桌面应用、macOS 原生感时自动应用。
---

# Wails React Desktop Rules

## 适用范围

当任务满足以下任一条件时应用本 Skill：

- 用户提到 `Wails`
- 用户提到 `React + Tailwind + shadcn/ui`
- 用户要实现桌面应用或 macOS 优先应用
- 用户要实现 HTTP 调试工具、API 调试器、请求编辑器、响应查看器
- 任务涉及 Go 与前端通过 binding/service 通信

## 核心目标

所有实现都必须优先保证：

- 轻量
- 快
- 清晰
- 原生感
- 可维护

不要把产品做成网页后台，也不要做成臃肿平台。

## 技术与分层约束

- 后端使用 `Go`
- 桌面容器使用 `Wails`
- 前端使用 `React + TypeScript`
- 样式使用 `Tailwind CSS`
- 组件优先使用 `shadcn/ui`
- 保持强类型，不随意使用 `any`
- 避免无意义抽象和过度设计

遵守以下边界：

- Go 层负责系统能力、网络请求、文件持久化、运行时能力
- React 层只负责 UI 与前端状态
- 前端不要直接处理底层文件系统细节
- 前后端通信必须通过明确的 service / binding API
- Binding 返回值必须结构化
- 错误返回必须统一格式
- IO 操作必须可恢复、可提示、可记录日志
- 请求发送逻辑优先放在 Go 层，避免浏览器环境限制

## 架构实现规则

- 按 domain / feature 分层
- 优先模块化，避免大而全单文件
- 组件拆分为基础 UI 组件、业务组件、布局组件
- 类型定义集中管理
- API DTO 与 ViewModel 分离
- Store 不承载过多派生逻辑
- 复杂逻辑提取到 hooks 或 service
- 列表、树、表单应有可复用抽象，但不要过度抽象
- 单个组件超过 250 行时优先考虑拆分
- 单个文件职责保持单一

## UI 与交互规则

- 风格偏 macOS desktop
- 少即是多，避免明显网页感
- 使用轻边框、轻阴影、克制色彩
- 以灰阶和层次为主，品牌色仅点缀
- 统一间距、字号、圆角
- 优先 split view、sidebar、toolbar 桌面布局
- 支持 `light / dark / system`
- 优先键盘可操作性
- 焦点、hover、active 态必须明确但不夸张

交互上必须优先保证：

- 常用操作少点击
- 请求编辑与响应查看尽量在同一主工作区完成
- destructive 操作二次确认
- 耗时操作显示 loading
- 失败操作显示明确信息
- 表单变更显示 dirty 状态
- 关闭前如未保存必须提示

## 数据与响应规则

- 本地持久化结构清晰且支持版本迁移
- 导出 JSON 必须保留 `schemaVersion`
- Secret 字段要有脱敏策略
- 历史记录必须有容量上限
- 删除优先软删除或可恢复（若实现成本低）
- JSON 响应自动格式化
- 大响应优先考虑性能
- 二进制响应不要强行按文本渲染
- HTML / 图片尽量支持预览
- 错误响应尽量识别 `problem+json`
- 响应区至少展示状态码、耗时、大小、headers、body

## 扩展性要求

架构应为以下能力预留空间，但第一版不强制实现：

- cURL import
- Postman Collection import/export
- OpenAPI import
- OAuth 2.0
- Pre-request scripts
- Tests
- Proxy
- SSL certificates
- Cookie manager
- 多标签页
- 多窗口

## 输出要求

每次输出代码时必须按以下顺序组织：

1. 先说明本次实现的模块目标
2. 再说明会新增或修改哪些文件
3. 然后给出尽量可直接落地的完整代码
4. 最后给出如何接入和验证

额外要求：

- 不要只给零散片段，优先给完整可落地实现
- 不要省略关键类型定义
- 不要省略依赖说明
- 注释只解释必要的意图、约束和取舍，不写废话注释
- UI 状态与业务状态分离
- 错误处理与日志处理保持统一

## 实施检查清单

在输出最终实现前自查：

- [ ] 是否保持 Wails 前后端职责边界清晰
- [ ] 是否避免了随意的 `any`
- [ ] 是否避免了明显网页后台风格
- [ ] 是否优先保证了主工作区效率
- [ ] 是否补全了关键类型、错误结构与依赖说明
- [ ] 是否为未来扩展保留了合理接口

## 附加参考

详细规则见 [reference.md](reference.md)
