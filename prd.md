你是一个资深桌面应用架构师、前端架构师、Go 工程师、React 工程师和产品设计师。

请帮我实现一个运行在 macOS 上的轻量级 HTTP API 调试工具，定位类似“轻量版 Postman / Bruno / Insomnia”，但比 Postman 更轻、更快、更克制，强调原生感和桌面体验。

# 一、技术栈要求
- Desktop framework: Wails（优先按稳定生产方案设计；如果涉及版本选择，优先给出适合生产的方案）
- Backend: Go
- Frontend: React + TypeScript
- UI: Tailwind CSS + shadcn/ui
- 状态管理：选择简单稳定的方案（如 Zustand）
- 表单处理：选择易维护方案
- JSON 展示：选一个轻量、性能好的 JSON viewer
- 数据持久化：本地文件存储，先不做云同步
- 平台目标：macOS 优先

# 二、产品定位
我要的是一个轻量级 HTTP 请求调试工具，不做臃肿平台。
核心是：
- 多项目管理
- 项目下有文件夹和请求
- 支持常见 HTTP 请求编辑与发送
- 支持常见请求头、参数、Body、认证
- 支持响应查看
- 支持环境变量
- 支持请求历史
- UI 要有 macOS 原生感，轻、快、简洁、克制

# 三、设计目标
请严格遵守以下设计原则：
1. 启动快，交互快，尽量轻量
2. UI 不要像网页后台，要更像 macOS 桌面工具
3. 视觉简洁，层级清晰，留白克制
4. 不做过度拟物，不做厚重阴影，不做太多高饱和颜色
5. 支持浅色/深色模式
6. 组件风格要统一、现代、专业
7. 信息密度适中，适合高频开发使用

# 四、信息架构
整体布局使用 macOS 风格桌面工具布局：
- 顶部 Toolbar
- 左侧 Sidebar：项目、文件夹、请求树
- 中间主编辑区：请求编辑
- 下方或右侧响应区：响应结果
- 使用 split view 思路组织布局
- 支持 resize panel

# 五、核心功能（MVP）
请先实现 MVP，必须包含：

## 1. 项目管理
- 新建项目
- 删除项目
- 重命名项目
- 项目切换
- 项目本地持久化

## 2. 文件夹管理
- 新建文件夹
- 重命名文件夹
- 删除文件夹
- 文件夹支持嵌套
- 拖拽排序

## 3. 请求管理
- 新建请求
- 删除请求
- 重命名请求
- 保存请求
- 请求归属到某个文件夹
- 左侧树中显示 method badge + name

## 4. HTTP 请求能力
支持：
- GET
- POST
- PUT
- PATCH
- DELETE
- HEAD
- OPTIONS

请求编辑必须支持：
- URL
- Query Params
- Headers
- Body
- Auth
- Settings

## 5. Body 类型
- none
- raw text
- JSON
- x-www-form-urlencoded
- form-data

## 6. Auth 类型
- No Auth
- Basic Auth
- Bearer Token
- API Key

## 7. 响应查看
显示：
- HTTP status
- 耗时 duration
- 响应大小 size
- 响应 headers
- 响应 body
- cookies（至少预留结构）

响应 body 视图支持：
- Pretty
- Raw
- Preview（当可预览时）

并且：
- JSON 自动格式化
- 支持复制响应
- 支持搜索响应文本

## 8. 环境变量
- 支持项目下多个 environment，例如 dev/test/prod
- 变量语法使用 {{variableName}}
- 请求发送前自动解析变量
- 支持 secret 类型变量，默认脱敏显示

## 9. 请求历史
- 记录最近请求
- 包括请求名、URL、时间、状态、耗时
- 可点击恢复

# 七、规范要求
实现时尽量遵循：
- HTTP 语义符合 RFC 9110
- 错误响应兼容 application/problem+json 风格
- 数据模型设计要为未来导入 OpenAPI / Postman Collection 预留扩展位
- 导入导出层要可扩展，后续支持：
  - Postman Collection v2.1
  - OpenAPI 3.x
  - cURL

# 八、数据结构要求
请你设计完整的 TypeScript 类型与 Go 端 DTO，包括：
- Project
- Folder
- Request
- Environment
- Variable
- HistoryEntry
- ResponseViewModel
- AuthConfig
- RequestBody

请保证：
- 类型清晰
- 可扩展
- 前后端结构一致
- 字段命名规范统一
- 时间字段统一格式
- ID 生成策略明确

# 九、工程结构要求
请给出完整项目目录设计，前后端分层清晰。

要求：
- React 前端采用 feature-based 或 domain-based 结构
- UI 组件、业务组件、页面、store、hooks、types、services 分开
- Go 后端采用 app/service/repository/model 等清晰分层
- Wails bindings 调用边界清晰
- 前端不要直接耦合底层本地文件逻辑

# 十、UI 细节要求
请严格按照以下视觉方向输出：
- macOS 原生感强
- 侧边栏像 Finder / Xcode / 桌面工具的感觉
- toolbar 简洁
- 输入框、tab、列表、按钮风格克制
- 使用 shadcn/ui 组件，但要进行桌面化二次封装
- 尽量少用大面积边框和重阴影
- 强调灰阶、留白、轻分隔线
- 支持 dark mode
- 支持快捷键
- 支持 command palette（可后做，但先预留）

# 十一、需要你输出的内容
请你一次性输出：
1. 产品功能清单（按 MVP / Next 阶段拆分）
2. 页面与布局设计说明
3. 完整信息架构
4. 前后端项目目录结构
5. TypeScript 类型定义
6. Go 后端模块设计
7. 本地持久化设计
8. 状态管理设计
9. UI 组件清单
10. 关键交互流程
11. 错误处理策略
12. 可扩展点设计
13. 首批开发任务拆分（按迭代拆分）
14. 每个模块的验收标准
15. 输出高质量代码时必须遵守的编码规范

# 十二、代码实现要求
- 代码必须可维护
- 拒绝过度设计
- 优先实现清晰可运行的 MVP
- 组件职责单一
- 类型完整
- 避免魔法字符串
- 所有异步流程都要有 loading / error / empty 状态
- UI 细节完整，不要只做骨架
- 先给出方案，再给出核心代码
- 生成代码时以“可以直接落地开发”为目标，不要只给伪代码

请先输出完整设计方案，然后再按模块逐步生成代码。