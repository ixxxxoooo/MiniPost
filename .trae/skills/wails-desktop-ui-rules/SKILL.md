---
name: wails-desktop-ui-rules
description: 为基于 Wails + React + TypeScript + Tailwind CSS + shadcn/ui 的 macOS 优先桌面应用提供 UI 设计系统、组件规范、桌面交互规则与代码输出标准。当用户提到 Wails、桌面应用、macOS 原生感、TablePlus 风格、frameless 窗口、自定义 titlebar、或开发同类 HTTP 调试工具/数据库工具/开发者工具时自动激活。
---

# Wails Desktop UI Rules — macOS Native-like 桌面应用开发规范

## 适用范围

当任务满足以下任一条件时自动应用本 Skill：

- 用户提到 `Wails` / `wails`
- 用户要实现 **桌面应用** 或 **macOS 优先应用**
- 用户提到 `frameless` / `custom titlebar` / `自定义标题栏`
- 用户追求 **TablePlus 风格** / **native-like** / **原生感** UI
- 用户使用 `React + Tailwind + shadcn/ui` 技术栈做桌面端
- 开发同类产品：HTTP 调试工具、API 调试器、数据库客户端、代码编辑器、终端工具、开发者工具

## 核心设计哲学

所有实现必须优先保证以下特质：

| 关键词 | 含义 |
|--------|------|
| **lightweight** | 轻量，不臃肿 |
| **fast** | 响应快，启动快 |
| **clear** | 界面清晰，信息层级明确 |
| **native-like** | 有桌面工具的原生质感 |
| **maintainable** | 代码可维护，架构清晰 |

**绝对不要**把产品做成网页后台风格，也不要做成臃肿平台。

---

## 一、技术栈与分层约束

### 技术选型

| 层 | 技术 | 职责 |
|----|------|------|
| 后端 | Go (Wails binding) | 系统能力、网络请求、文件持久化、运行时 API |
| 桌面容器 | Wails v2 | 窗口管理、前后端桥接、原生能力暴露 |
| 前端框架 | React 18 + TypeScript | UI 渲染、前端状态管理 |
| 样式方案 | Tailwind CSS + CSS 变量 | 样式实现，所有尺寸走令牌系统 |
| 组件库 | shadcn/ui (Radix UI primitives) | 基础 UI 原语，需二次封装为 desktop 风格 |
| 状态管理 | Zustand | 轻量状态管理，避免过度抽象 |
| 图标 | HugeIcons / Lucide | 图标统一来源 |

### 严格的职责边界

```
┌─────────────────────────────────────────────┐
│                  Go Layer                   │
│  · 网络请求（HTTP/gRPC/WebSocket）          │
│  · 文件读写 / 持久化                        │
│  · 系统能力（剪贴板/对话框/通知）           │
│  · 加密 / 证书 / 代理                       │
└──────────────────────┬──────────────────────┘
                       │ Wails Binding (结构化 API)
┌──────────────────────▼──────────────────────┐
│                React Layer                  │
│  · UI 渲染与交互                             │
│  · 前端状态 (Zustand stores)                │
│  · 表单验证 / 用户输入处理                  │
│  · 主题切换 / 动画                          │
└─────────────────────────────────────────────┘
```

**关键规则**：
- Go 层负责 IO、网络、文件系统；React 层只负责 UI
- 前后端通信必须通过明确的 service/binding API
- Binding 返回值必须结构化，禁止返回 `any`
- 所有错误必须统一格式返回
- IO 操作必须可恢复、可提示、可记录日志
- **请求发送逻辑优先放在 Go 层**，避免浏览器环境限制（CORS、cookie 等）

---

## 二、设计令牌系统（Design Tokens）

这是整个 UI 规范的基石。所有尺寸、颜色、圆角、间距必须来自 CSS 变量。

### 2.1 圆角令牌（Radius）

```css
:root {
  --radius-window: 12px;    /* 窗口整体圆角 */
  --radius-panel: 12px;     /* 面板、卡片、弹窗 */
  --radius-btn: 7px;        /* 按钮 */
  --radius-input: 8px;      /* 输入框、搜索框、下拉框 */
  --radius-menu: 8px;       /* 菜单、右键菜单、popover */
  --radius-sm: 4px;         /* badge/tag/小标签 */
}
```

**使用原则**：
- 禁止出现"大圆角网页感"，整体是小圆角、克制、精密
- 按钮 7px 是最佳平衡点，既不像网页那么圆润，也不像纯矩形那么硬

### 2.2 尺寸令牌（Sizing）

```css
:root {
  /* 控件高度 */
  --size-toolbar: 37px;       /* 工具栏高度 */
  --size-btn: 28px;           /* 默认按钮高度 */
  --size-btn-sm: 24px;        /* 小按钮高度 */
  --size-btn-icon: 16px;      /* 图标按钮内图标尺寸 */
  --size-btn-icon-sm: 14px;   /* 小图标按钮内图标尺寸 */
  --size-input: 32px;         /* 输入框高度 */
  --size-input-sm: 28px;      /* 小输入框高度 */
  --size-tab: 28px;           /* 标签页高度 */

  /* 字号 */
  --size-font-base: 15px;     /* 基础字号 */
  --size-font-sm: 14px;       /* 正文字号（最常用） */
  --size-font-xs: 13px;       /* 次级文字 */
  --size-font-2xs: 12px;      /* 辅助文字/表格 */

  /* 间距 */
  --size-gap: 8px;            /* 常规间距 */
  --size-gap-sm: 4px;         /* 紧凑间距 */
  --size-padding: 12px;       /* 常规内边距 */
  --size-padding-sm: 8px;     /* 紧凑内边距 */
}
```

### 2.3 颜色令牌（Colors）

#### 中性色阶（Light Mode）

```css
:root {
  /* 主色调（品牌色/强调色，默认 iOS 蓝） */
  --accent: #007aff;
  --accent-fg: #ffffff;
  --accent-hover: #0066d6;

  /* 表面色 */
  --surface: #ffffff;
  --surface-secondary: #f9f9f9;
  --surface-elevated: #ffffff;

  /* 前景色 */
  --fg: #1d1d1f;              /* 主文字 */
  --fg-secondary: #6e6e73;    /* 次级文字 */
  --fg-muted: #aeaeb2;        /* 弱化文字/placeholder */

  /* 边框 */
  --border-color: #ededed;
  --border-subtle: #ededed;

  /* 按钮与选中态 */
  --button-bg: rgb(237, 237, 237);
  --button-border: rgb(230, 230, 230);
  --selected-bg: rgb(242, 242, 242);
}
```

#### 侧边栏专用色

```css
:root {
  --sidebar-bg: rgba(249, 249, 249, 0.92);       /* 半透明背景 */
  --sidebar-fg: #1d1d1f;
  --sidebar-border: #ededed;
  --sidebar-accent: #007aff;
  --sidebar-hover: rgba(0, 0, 0, 0.04);          /* hover 极轻 */
  --sidebar-active: rgba(0, 122, 255, 0.12);     /* active 带品牌色 */
}
```

#### 工具栏专用色

```css
:root {
  --toolbar-bg: rgba(249, 249, 249, 0.85);       /* 半透明磨砂感 */
  --toolbar-border: #ededed;
}
```

#### Dark Mode 完整覆盖

```css
.dark {
  --accent: #0a84ff;
  --accent-hover: #409cff;
  --surface: rgb(33, 33, 33);
  --surface-secondary: rgb(48, 48, 48);
  --fg: #f5f5f7;
  --fg-secondary: #b8b8be;
  --fg-muted: #8a8a90;
  --border-color: rgb(48, 48, 48);
  --button-bg: rgb(59, 59, 59);
  --selected-bg: rgb(68, 68, 68);
  --sidebar-bg: rgba(33, 33, 33, 0.92);
  --sidebar-active: rgba(10, 132, 255, 0.18);
  --toolbar-bg: rgba(33, 33, 33, 0.85);
}
```

**Dark Mode 关键原则**：
- 不能一片死黑，必须保持层次（用不同灰度区分 surface/surface-secondary/elevated）
- hover 和 active 态要有明显但不夸张的变化
- 边框不能消失，要用深色边框保持层次感

### 2.4 状态色与语义色

```css
:root {
  /* 功能状态色 */
  --success: #34c759;         /* 成功 / GET 方法 */
  --warning: #ff9500;         /* 警告 / PUT 方法 */
  --danger: #ff3b30;          /* 危险 / DELETE 方法 */
  --info: #5ac8fa;            /* 信息 / OPTIONS 方法 */

  /* HTTP 方法色（用于方法标签） */
  --method-get: #34c759;
  --method-post: #007aff;
  --method-put: #ff9500;
  --method-patch: #af52de;
  --method-delete: #ff3b30;
  --method-head: #6e6e73;
  --method-options: #5ac8fa;

  /* JSON 语法高亮 */
  --json-key: #881391;
  --json-string: #1a7f37;
  --json-number: #0550ae;
  --json-boolean: #cf222e;
  --json-bracket: #6e6e73;
}
```

### 2.5 阴影令牌

```css
:root {
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.04);    /* 轻微层次 */
  --shadow-md: 0 4px 12px rgba(0, 0, 0, 0.08);   /* 菜单/浮层 */
  --shadow-lg: 0 8px 24px rgba(0, 0, 0, 0.12);    /* 弹窗/模态 */
}
```

---

## 三、CSS 变量使用铁律

### ✅ 允许的写法

```tsx
// 通过 CSS 变量引用
<h-[var(--size-toolbar)]>
<rounded-[var(--radius-btn)]>
<px-[var(--size-padding-3)]>
<text-[length:var(--size-font-sm)]>
<borders-[var(--border-color)]>
bg-[var(--surface)]
```

### ❌ 禁止的写法

```tsx
// 硬编码像素值
<h-[52px]>           // ❌ 必须用 h-[var(--size-toolbar)]
<rounded-[8px]>      // ❌ 必须用 rounded-[var(--radius-input)]
<px-3>               // ❌ 必须用 px-[var(--size-padding)]
<text-sm>            // ❌ 必须用 text-[length:var(--size-font-sm)]
```

**核心规则**：所有 width/height/padding/margin/gap/border-radius/font-size/icon-size/toolbar-height/sidebar-width/panel-header-height/row-height 都只能使用 CSS 变量和基于变量的 `calc()`。Tailwind class 可以使用，但不得直接写死尺寸值。

---

## 四、Wails 桌面窗口规范

### 4.1 Frameless 自定义标题栏

**必须实现的特性**：

```
┌────────────────────────────────────────────────────────┐
│ [●] [○] [◉] │  项目名 / 页面标题 / 拖拽区    │ [⚙] [🌙] │  ← 52px 高度
├────────────────────────────────────────────────────────┤
│                                                        │
│                    主内容区                            │
│                                                        │
└────────────────────────────────────────────────────────┘
```

**技术要求**：

| 特性 | 实现 |
|------|------|
| 高度 | 固定 `var(--size-toolbar)` (37px) 或自定义 titlebar-height |
| 定位 | `fixed` 顶部层，z-index 最高 |
| 背景 | 透明或半透明磨砂 (`rgba(..., 0.85)`)，不得厚重 |
| 拖拽 | 使用 `-webkit-app-region: drag` |
| 双击最大化 | 双击空白区域触发 `WindowToggleMaximise()` |

### 4.2 拖拽区规则（Critical）

```tsx
// 拖拽区域根节点
<div className="titlebar-drag">  {/* 等同于 -webkit-app-region: drag */}
  {/* 所有可交互元素必须设为 no-drag */}
  <div className="titlebar-no-drag" onMouseDown={e => e.stopPropagation()}>
    <Button>点击我</Button>
    <Input />
  </div>
</div>
```

**关键规则**：
- titlebar 根节点使用 `drag` 区域
- 所有 button/input/select/search/tabs/dropdown trigger 必须 `no-drag`
- 不允许出现"点击按钮导致窗口拖动"的问题
- 不允许出现"输入框无法选中文本"的问题
- `no-drag` 元素内部必须 `onMouseDown={(e) => e.stopPropagation()}` 阻止冒泡

### 4.3 macOS 窗口控制按钮

**必须自绘**，不使用原生 traffic lights：

| 特性 | 要求 |
|------|------|
| 位置 | 左上角 |
| 按钮 | close / minimize / maximize 三个 |
| 尺寸 | 小巧，位置克制（约 12x12px 图标区域） |
| 对齐 | 与 titlebar 垂直居中 |
| 风格 | 保持 macOS 暗示，不做 1:1 复刻 |

**按钮状态设计**：

| 状态 | 视觉表现 |
|------|----------|
| default | 低对比度，几乎融入背景 |
| hover | 提升色彩与图标可见度 |
| pressed | 略微变深 |
| inactive window | 整体降低对比度 |

### 4.4 双击标题栏最大化

```tsx
function useTitlebarDoubleClick() {
  const lastClickRef = useRef({ time: 0, x: 0, y: 0 })
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (e.button !== 0) return
    const now = Date.now()
    const last = lastClickRef.current
    if (now - last.time < 400 && Math.abs(e.clientX - last.x) < 5 && Math.abs(e.clientY - last.y) < 5) {
      import("../../../wailsjs/runtime/runtime").then(r => r.WindowToggleMaximise())
      lastClickRef.current = { time: 0, x: 0, y: 0 }
    } else {
      lastClickRef.current = { time: now, x: e.clientX, y: e.clientY }
    }
  }, [])
  return { handleMouseDown }
}
```

**注意**：双击按钮区、输入区、菜单区时不触发最大化。

---

## 五、组件编写规范

### 5.1 组件分层架构

```
components/
├── ui/              ← 基础 UI 组件（shadcn/ui 二次封装）
│   ├── button.tsx
│   ├── input.tsx
│   ├── tabs.tsx
│   ├── select.tsx
│   ├── tooltip.tsx
│   ├── icon.tsx
│   ├── badge.tsx
│   ├── scroll-area.tsx
│   ├── separator.tsx
│   └── CodeEditor.tsx
├── business/        ← 业务组件（领域相关）
│   ├── editor/      （请求编辑器系列）
│   ├── response/    （响应查看器系列）
│   ├── cookie/
│   ├── environment/
│   └── history/
└── layout/          ← 布局组件（页面骨架）
    ├── AppLayout.tsx
    ├── Toolbar.tsx
    ├── Sidebar.tsx
    ├── TabBar.tsx
    ├── BottomBar.tsx
    └── SettingsPanel.tsx
```

### 5.2 Button 组件规范

```tsx
// ✅ 正确示例：所有尺寸来自 token
<button className={cn(
  "inline-flex items-center justify-center rounded-[var(--radius-btn)] font-medium",
  "focus-visible:ring-2 focus-visible:ring-[var(--accent)]",
  "h-[var(--size-btn)] px-3 text-[length:var(--size-font-sm)]",  // size=default
  variant === "default" && "bg-[var(--accent)] text-white hover:bg-[var(--accent-hover)]",
  variant === "ghost" && "hover:bg-[var(--button-bg)] text-[var(--fg)]",
  variant === "outline" && "border border-[var(--button-border)] bg-[var(--button-bg)]",
  variant === "destructive" && "bg-[var(--danger)] text-white",
)}>
```

**Variant 设计原则**：
- `default`: 主操作，轻微品牌色背景
- `ghost`: 次要操作，hover 时才显示背景
- `outline`: 边框按钮，轻背景
- `destructive`: 仅危险操作（删除等），红色
- `link`: 文字链接样式

**Size 规格**：
| Size | 高度 | 字号 | 用途 |
|------|------|------|------|
| default | `--size-btn` (28px) | `--size-font-sm` (14px) | 常规按钮 |
| sm | `--size-btn-sm` (24px) | `--size-font-xs` (13px) | 紧凑/工具栏 |
| lg | calc(`--size-btn`+6px) | `--size-font-base` (15px) | 主要 CTA |
| icon | `--size-btn` × `--size-btn` | — | 图标按钮 |

### 5.3 Input 组件规范

```tsx
<input className={cn(
  "flex h-[var(--size-input)] w-full rounded-[var(--radius-input)]",
  "border border-[var(--border-color)] bg-[var(--surface)] px-3 py-1",
  "text-[var(--fg)] text-[length:var(--size-font-sm)]",
  "placeholder:text-[var(--fg-muted)]",
  "focus-visible:ring-2 focus-visible:ring-[var(--accent)]",  // 细 ring，不用粗蓝边
  "disabled:opacity-50"
)} />
```

**Input 设计原则**：
- 高度统一为 `--size-input` (32px)
- radius 使用 `--radius-input` (8px)
- 背景微微抬起，但不厚重
- focus 态使用细 ring，不用粗蓝边框
- placeholder 低对比度 (`--fg-muted`)

### 5.4 Tabs 组件规范

```tsx
// TabsList — 桌面风格的标签页容器
<TabsList className={cn(
  "inline-flex items-center h-[var(--size-tab)] gap-[var(--size-gap-sm)]",
  "px-[var(--size-padding-sm)] bg-transparent"
)}>

// TabsTrigger — 单个标签
<TabsTrigger className={cn(
  "inline-flex items-center justify-center px-2.5 py-1 rounded-[6px]",
  "text-[11px] font-medium leading-none",
  "text-[var(--fg-muted)] hover:text-[var(--fg-secondary)] hover:bg-[var(--selected-bg)]",
  "data-[state=active]:bg-[var(--selected-bg)] data-[state=active]:text-[var(--fg)]"
)}>
```

**Tabs 设计原则**：
- 更像桌面工具标签，不像网页 tab
- 高度小 (`--size-tab` = 28px)
- 边界清晰但克制
- active 态用浅色背景 + 深色文字，不用下划线或品牌色边框
- 字号偏小 (11px)，紧凑专业

### 5.5 Sidebar 侧边栏规范

**视觉要求**：
- 背景半透明 (`rgba(..., 0.92)`)，与主面板有轻微层次差
- 树形列表适合高频操作（请求树、文件夹树）
- 选中态清楚但不能过蓝（用 `--sidebar-active`）
- hover 极轻（`--sidebar-hover` 仅 4% 不透明度）
- 支持可调节宽度（拖拽调整，范围 180px–500px）

**交互要求**：
- 支持右键上下文菜单
- 支持拖拽排序
- 支持搜索过滤
- 支持行内重命名
- 删除前二次确认

### 5.6 表格/键值对编辑器规范

用于 Headers、Params、Body form-data 等场景：

```tsx
// 行高固定 24px，紧凑高效
<tr className="group hover:bg-[var(--surface-secondary)]/50">
  <td className="h-[24px]">
    <input className="h-[24px] px-2 text-[11px] font-mono" />
  </td>
</tr>
```

**特点**：
- 表格形式，紧凑高效
- 行高 24px，字号 11px monospace
- 自动追加空行（最后一行空行输入时自动新增）
- 支持 bulk edit（批量编辑）模式
- 删除按钮 hover 时才显示

---

## 六、视觉风格总则（TablePlus-inspired）

### 必须做到

- ✅ 以中性灰阶为主色调
- ✅ 强调色仅用于焦点、选中、主操作、小范围状态反馈
- ✅ 边框颜色轻，分隔线明确但存在感低
- ✅ hover 很轻，active 比 hover 稍强但仍克制
- ✅ 靠背景层级、细边框、微妙状态色区分层次
- ✅ 阴影只用于菜单、popover、浮层
- ✅ 控件像桌面工具按钮，不像营销站按钮

### 绝对禁止

- ❌ 花哨渐变
- ❌ 厚重投影
- ❌ 大面积品牌色铺底
- ❌ 明显 SaaS 后台风格
- ❌ 夸张卡片化
- ❌ 强拟物（neumorphism）
- ❌ 过度动画
- ❌ 复杂玻璃拟态（glassmorphism）
- ❌ 大圆角网页感（radius > 12px 的常规元素）
- ❌ 营销站式大按钮
- ❌ 强卡片化后台风格（大白卡片堆叠）

---

## 七、主题系统规范

### 7.1 三种模式支持

```typescript
type Theme = "light" | "dark" | "system"
```

- `light`: 强制浅色
- `dark`: 强制深色
- `system`: 跟随操作系统偏好

### 7.2 主题切换实现

```typescript
function applyTheme(resolved: "light" | "dark") {
  const root = document.documentElement
  if (resolved === "dark") root.classList.add("dark")
  else root.classList.remove("dark")
}
```

### 7.3 项目主题色动态注入

支持用户自定义项目主题色（如 VS Code 的 Color Theme）：

```typescript
export function applyProjectThemeColor(color: string | null, resolved: "light" | "dark") {
  const root = document.documentElement
  if (!color) {
    // 重置为默认
    root.style.removeProperty("--accent")
    root.style.removeProperty("--sidebar-accent")
    root.style.removeProperty("--tab-active-border")
    return
  }

  const hover = mixColor(color, resolved === "dark" ? {r:255,g:255,b:255} : {r:0,g:0,b:0}, 0.14)
  const selectedBg = withAlpha(color, resolved === "dark" ? 0.2 : 0.13)
  const sidebarActive = withAlpha(color, resolved === "dark" ? 0.24 : 0.14)

  root.style.setProperty("--accent", color)
  root.style.setProperty("--accent-hover", hover)
  root.style.setProperty("--sidebar-accent", color)
  root.style.setProperty("--tab-active-border", color)
  root.style.setProperty("--selected-bg", selectedBg)
  root.style.setProperty("--sidebar-active", sidebarActive)
}
```

---

## 八、全局样式基础

### 8.1 字体

```css
font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text",
             "SF Pro Display", "Helvetica Neue", Arial, sans-serif;
/* 代码字体 */
--font-mono: "SF Mono", "Cascadia Mono", "Menlo", "Monaco", "Consolas", monospace;
```

### 8.2 滚动条（自定义细滚动条）

```css
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb {
  background: color-mix(in srgb, var(--fg-muted) 60%, transparent);
  border-radius: 3px;
}
```

支持 auto-hide 模式：静止时隐藏，滚动时显示。

### 8.3 按钮点击反馈

```css
button:active:not(:disabled) {
  transform: scale(0.98);
  opacity: 0.92;
  transition: transform 0.05s ease, opacity 0.05s linear;
}
```

轻量按压反馈，让用户感知到交互。可通过 `.no-press-feedback` 禁用。

### 8.4 选择高亮

```css
::selection {
  background: var(--accent);
  color: var(--accent-fg);
}
```

---

## 九、交互规则

### 9.1 键盘快捷键（必备）

| 快捷键 | 功能 |
|--------|------|
| ⌘N | 新建请求/新建项目 |
| ⌘S | 保存 |
| ⌘Enter | 发送请求 |
| ⌘, | 打开设置 |
| ⌘I | 导入 cURL |
| ⌘O | 打开/导入文件 |
| Escape | 关闭弹窗/取消操作 |

### 9.2 通用交互模式

- **高频操作少点击**：常用功能放在触手可及的位置
- **Dirty 状态**：有未保存修改时必须显示视觉提示
- **Loading 反馈**：耗时操作必须有 loading 指示
- **错误反馈**：失败操作显示明确错误信息
- **Destructive 确认**：删除等危险操作必须二次确认（支持 Enter 确认 / Escape 取消）
- **关闭前检查**：未保存时关闭前必须提示用户

---

## 十、工程规范

### 10.1 代码质量

- 所有代码必须 **TypeScript 强类型**，禁止随意使用 `any`
- 函数名语义明确，见名知意
- 单个组件超过 **250 行** 时考虑拆分
- 单个文件职责保持单一
- 类型定义集中管理在 `types/` 目录
- UI 状态和业务状态分离

### 10.2 目录结构建议

```
frontend/src/
├── components/
│   ├── ui/              # 基础组件（desktop 风格封装）
│   ├── business/        # 业务组件
│   └── layout/          # 布局组件
├── hooks/               # 自定义 hooks
├── lib/                 # 工具函数、常量、主题
│   ├── constants.ts     # 枚举、常量
│   ├── utils.ts         # cn() 等
│   ├── projectTheme.ts  # 主题色注入逻辑
│   ├── locale.ts        # i18n
│   └── logger.ts        # 日志
├── services/            # 前端服务层（调 Wails binding）
├── stores/              # Zustand 状态管理
├── types/               # TypeScript 类型定义
├── styles/
│   └── globals.css      # 全局样式 + CSS 变量
├── App.tsx
└── main.tsx
```

### 10.3 输出代码时的检查清单

每次输出代码时必须自查：

- [ ] 是否使用了 CSS 变量而非硬编码尺寸？
- [ ] 是否保持了 Wails 前后端职责边界清晰？
- [ ] 是否避免了随意的 `any`？
- [ ] 是否避免了网页后台/SaaS 风格？
- [ ] titlebar、拖拽区、按钮 no-drag 是否完整实现？
- [ ] 双击 titlebar 最大化是否实现？
- [ ] light/dark 主题是否都适配？
- [ ] 组件是否支持桌面交互（键盘、右键、拖拽）？
- [ ] 是否补全了关键类型定义？
- [ ] 代码是否可直接落地，不给伪代码？

---

## 十一、扩展性预留

架构应为以下能力预留接口空间（第一版可不实现）：

- cURL import / export
- Postman Collection import/export
- OpenAPI / Swagger import
- OAuth 2.0
- Pre-request scripts & Tests
- Proxy settings
- SSL certificates management
- Cookie manager
- Multi-tab / Multi-window
- Command Palette (⌘K)
- Plugin system

---

*本 Skill 基于 MiniPost 项目的实际生产代码提炼而成，涵盖了从设计令牌到桌面交互的完整规范体系。适用于任何基于 Wails + React + TypeScript 的 macOS 优先桌面开发者工具。*
