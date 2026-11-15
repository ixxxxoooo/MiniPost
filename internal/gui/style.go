package gui

import (
	"embed"
	"fmt"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"minipost/internal/pkg/logger"
)

// 本文件是界面的设计系统：调色板、尺寸常量与基础控件，数值全部取自旧版
// frontend/src/styles/globals.css 与各组件的实际渲染结果（TablePlus 风格）。

//go:embed icons/*.svg icons/logo.png icons/logo-dark.png
var assetFS embed.FS

// 尺寸常量（单位 DIP，与旧版 CSS 像素一致）。
const (
	sizeToolbar   = 46  // 顶部自绘标题栏高度（方案 C：透明，露出极光）
	sizeBottom    = 24  // 状态栏高度
	sizeTabStrip  = 38  // 请求标签栏高度（方案 C 的 --tabh）
	sizeURLBar    = 34  // 地址栏（一体式控件）高度
	sizeMethodCol = 32  // 侧栏树与历史里方法列的固定宽度（徽章宽 = 该值 + 4）
	sizeBtn       = 24  // 工具栏图标按钮
	radiusBtn     = 7   // 按钮圆角（小型图标按钮）
	radiusR       = 10  // 方案 C 的标准圆角（--r）：输入框、按钮、胶囊以外的控件
	radiusInput   = 8   // 输入框圆角
	radiusCard    = 10  // 卡片圆角
	fsBase        = 12  // 基础字号
	fsSmall       = 11  // 次级字号
	fsTiny        = 10  // 微型字号
	fsLarge       = 13  // 大字号（URL、响应标题）
	sizeSideMin   = 180 // 侧栏最小宽度
	sizeSideHead  = 40  // 侧栏顶部分段控件行高（内边距 8/6 + 26）
)

// 旧版用 CSS 变量 --size-font-2xs/xs/sm 表示随「界面字体」设置缩放的字号（N、N+1、N+2，默认 N=12）；
// 其余文本是固定像素，不随设置变化。
var (
	fsV2xs float32 = 12
	fsVXs  float32 = 13
	fsVSm  float32 = 14
)

// setUIFontSize 按设置的界面字体 N（10~16）更新三档可缩放字号。
func setUIFontSize(n int) {
	n = min(max(n, 10), 16)
	fsV2xs, fsVXs, fsVSm = float32(n), float32(n+1), float32(n+2)
}

// palette 是一套完整的颜色令牌，对应 globals.css 的 :root 与 .dark。
type palette struct {
	dark bool

	accent, accentHover, accentFg ui.Color

	surface, surface2, fg, fg2, muted  ui.Color
	border, btnBg, btnBorder, selected ui.Color

	toolbar, sidebar, tabStrip, bottom ui.Color
	sideActive, sideHover, tabActive   ui.Color
	tabDivider, tabActiveBorder        ui.Color

	success, warning, danger, info ui.Color

	get, post, put, patch, del, head, options ui.Color

	jsonKey, jsonString, jsonNumber, jsonBool, jsonBracket ui.Color
	hover                                                  ui.Color

	// 流光（设计稿方案 C）新增令牌，全部由基础色派生：
	// press 为按下底（前景 9%）；acSoft/acSoft2/acLine 为强调色 12%/20%/45%；
	// glow/glowS 为强调色光晕（42%/26%）；glass 为浮层玻璃底（elev 84%）；hair 为发丝线；
	// okSoft 等为语义色的淡底。
	press                               ui.Color
	acSoft, acSoft2, acLine             ui.Color
	glow, glowS                         ui.Color
	glass, hair                         ui.Color
	okSoft, warnSoft, errSoft, infoSoft ui.Color
}

// elevated 对应旧版 --surface-elevated（弹出层背景）：亮色为白色，暗色为 rgb(48,48,48)。
func (p *palette) elevated() ui.Color {
	if p.dark {
		return p.surface2
	}
	return p.surface
}

// shadowMd 给元素加上设计稿的 --shadow-md（浮起的卡片）：1px 描边环 + 大范围柔和投影 + 近处细投影。
func (p *palette) shadowMd(e ui.Element) ui.Element {
	if p.dark {
		return e.Shadow(0, 0, 0, 1, p.border).Shadow(0, 10, 24, -6, ui.RGBA(0, 0, 0, 0.5)).Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.3))
	}
	return e.Shadow(0, 0, 0, 1, p.border).Shadow(0, 8, 22, -6, ui.RGBA(20, 24, 40, 0.14)).Shadow(0, 1, 3, 0, ui.RGBA(20, 24, 40, 0.05))
}

// shadowSm 给元素加上设计稿的 --shadow-sm（分段滑块、选中标签等小块浮起）：细投影 + 0.5px 环。
func (p *palette) shadowSm(e ui.Element) ui.Element {
	if p.dark {
		return e.Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.32)).Shadow(0, 0, 0, 0.5, ui.RGBA(255, 255, 255, 0.04))
	}
	return e.Shadow(0, 1, 2, 0, ui.RGBA(20, 24, 40, 0.06)).Shadow(0, 0, 0, 0.5, ui.RGBA(20, 24, 40, 0.04))
}

// gradAccent 把元素填成设计稿的强调色渐变（--grad-ac：上方 16% 提亮到纯强调色）并加顶部高光；hover 时用悬停色。
func (p *palette) gradAccent(e ui.Element, hover bool) ui.Element {
	base := p.accent
	if hover {
		base = p.accentHover
	}
	top := base.Mix(ui.RGB(255, 255, 255), 0.16)
	return e.Gradient(top, base, 180).InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.24))
}

// shadowLg 对应旧版 --shadow-lg：0 8px 24px，亮色 0.12、暗色 0.4 的黑色。
func (p *palette) shadowLg() ui.Color {
	if p.dark {
		return ui.RGBA(0, 0, 0, 0.4)
	}
	return ui.RGBA(0, 0, 0, 0.12)
}

// schemeTokens 是设计稿中一套配色的基础令牌（chrome=标题栏/标签栏/状态栏，side=侧栏，bg=内容区，bg2=次级底，
// bd/bd2=细/粗边框，fg/fg2/mu=主/次/弱文字，ac/ach/acf=强调色/悬停/其上文字）。
type schemeTokens struct {
	chrome, side, bg, bg2, bd, bd2, fg, fg2, mu, ac, ach, acf string
}

// colorSchemes 按设计稿「令牌与通用组件」：mist 雾蓝 / ink 墨白 / jade 松石 / amber 琥珀 / rose 玫瑰，各有 [亮, 暗] 两档。
var colorSchemes = map[string][2]schemeTokens{
	"mist": {
		{"#f1f2f6", "#f5f6f9", "#ffffff", "#eef0f5", "#e2e5ec", "#cfd4de", "#1b1e29", "#4b5266", "#6a7287", "#4f52d9", "#4144c4", "#ffffff"},
		{"#16171c", "#1a1b21", "#1f2027", "#282a32", "#2d2f38", "#3b3e4a", "#eceef4", "#a9adbb", "#8a8fa0", "#8c8fff", "#a3a5ff", "#0f1030"},
	},
	"ink": {
		{"#f2f2f0", "#f6f6f4", "#ffffff", "#efefed", "#e4e4e1", "#d0d0cc", "#17171a", "#4d4d54", "#6c6c74", "#17171a", "#35353b", "#ffffff"},
		{"#121212", "#161616", "#1b1b1c", "#252526", "#2a2a2c", "#3a3a3d", "#f0f0f0", "#a8a8ae", "#86868e", "#f0f0f0", "#ffffff", "#111111"},
	},
	"jade": {
		{"#eff3f2", "#f4f7f6", "#ffffff", "#ebf0ee", "#dde5e2", "#c9d4d0", "#17201e", "#46524f", "#66726f", "#08806f", "#066b5c", "#ffffff"},
		{"#0f1514", "#131a19", "#172020", "#1f2a29", "#243230", "#30403d", "#e8f0ee", "#a1b2ae", "#84968f", "#3fd0b8", "#62dcc7", "#04201b"},
	},
	"amber": {
		{"#f6f3ee", "#f9f7f3", "#ffffff", "#f3efe8", "#e8e2d8", "#d6cdbf", "#231d14", "#5a5042", "#7a6f5f", "#c2570c", "#a84a08", "#ffffff"},
		{"#17140f", "#1b1812", "#211d16", "#2b261d", "#342e23", "#453d2e", "#f3ede2", "#beb4a1", "#968b77", "#ff9a3c", "#ffb066", "#2a1500"},
	},
	"rose": {
		{"#f7f1f3", "#faf5f7", "#ffffff", "#f5edf0", "#ebe0e4", "#dac9cf", "#261a1f", "#5c4750", "#7d6670", "#d6336c", "#b8275a", "#ffffff"},
		{"#181114", "#1c1418", "#221a1e", "#2c2227", "#372a31", "#493943", "#f6ecf0", "#c2aab4", "#9a8590", "#ff7aa5", "#ff9bbb", "#2d0614"},
	},
}

// colorSchemeNames 是设置页里配色方案的顺序，与 colorSchemes 的键一致。
var colorSchemeNames = []string{"mist", "ink", "jade", "amber", "rose"}

// semanticTokens 是各配色共用的语义色（请求方法、状态、JSON 高亮），亮暗各一套。
type semanticTokens struct {
	get, post, put, pat, del, head, opt, ok, warn, err, k, s, n, b, p string
}

var (
	semanticLight = semanticTokens{"#0a7d50", "#b8520a", "#2260c4", "#7e45cc", "#c93030", "#0a7f7f", "#626a7b", "#0a7d50", "#a05a00", "#c93030", "#3e4bc2", "#0a7d50", "#b8520a", "#c93030", "#8a91a2"}
	semanticDark  = semanticTokens{"#4cd694", "#ffa05c", "#6aa6ff", "#c39bff", "#ff7070", "#4fd1d1", "#9aa1b2", "#4cd694", "#f2b84b", "#ff7a7a", "#9fb0ff", "#7fd9a8", "#ffb070", "#ff8a8a", "#7c8394"}
)

// schemePalette 返回指定配色方案的亮/暗调色板；未知方案按雾蓝处理。
func schemePalette(scheme string, dark bool) *palette {
	pair, ok := colorSchemes[scheme]
	if !ok {
		pair = colorSchemes["mist"]
	}
	i, sem := 0, semanticLight
	if dark {
		i, sem = 1, semanticDark
	}
	t := pair[i]
	fg, bg, chrome := ui.Hex(t.fg), ui.Hex(t.bg), ui.Hex(t.chrome)
	ac := ui.Hex(t.ac)
	hov := fg.Alpha(0.05)
	return &palette{
		dark:   dark,
		accent: ac, accentHover: ui.Hex(t.ach), accentFg: ui.Hex(t.acf),
		surface: bg, surface2: ui.Hex(t.bg2),
		fg: fg, fg2: ui.Hex(t.fg2), muted: ui.Hex(t.mu),
		border: ui.Hex(t.bd), btnBg: ui.Hex(t.bg2), btnBorder: ui.Hex(t.bd2), selected: bg.Mix(fg, 0.08),
		toolbar: chrome, sidebar: ui.Hex(t.side), tabStrip: chrome, bottom: chrome,
		sideActive: ac.Alpha(0.12), sideHover: hov, tabActive: bg,
		tabDivider: ui.Hex(t.bd), tabActiveBorder: ac,
		success: ui.Hex(sem.ok), warning: ui.Hex(sem.warn), danger: ui.Hex(sem.err), info: ui.Hex(sem.put),
		get: ui.Hex(sem.get), post: ui.Hex(sem.post), put: ui.Hex(sem.put), patch: ui.Hex(sem.pat),
		del: ui.Hex(sem.del), head: ui.Hex(sem.head), options: ui.Hex(sem.opt),
		jsonKey: ui.Hex(sem.k), jsonString: ui.Hex(sem.s), jsonNumber: ui.Hex(sem.n), jsonBool: ui.Hex(sem.b), jsonBracket: ui.Hex(sem.p),
		hover:  hov,
		press:  fg.Alpha(0.09),
		acSoft: ac.Alpha(0.12), acSoft2: ac.Alpha(0.20), acLine: ac.Alpha(0.45),
		glow: ac.Alpha(0.42), glowS: ac.Alpha(0.26),
		glass: glassBase(ui.Hex(t.bg), ui.Hex(t.bg2), dark).Alpha(0.84), hair: ui.Hex(t.bd).Alpha(0.62),
		okSoft: ui.Hex(sem.ok).Alpha(0.14), warnSoft: ui.Hex(sem.warn).Alpha(0.16),
		errSoft: ui.Hex(sem.err).Alpha(0.13), infoSoft: ui.Hex(sem.put).Alpha(0.14),
	}
}

// glassBase 返回浮层（对话框、菜单、提示）的底色：亮色为内容底，暗色为次级底（设计稿 --elev）。
func glassBase(bg, bg2 ui.Color, dark bool) ui.Color {
	if dark {
		return bg2
	}
	return bg
}

// withAccent 按项目主题色覆盖强调色及相关令牌，规则同旧版 applyProjectThemeColor。
func (p *palette) withAccent(hex string) *palette {
	if len(hex) != 7 || hex[0] != '#' {
		return p
	}
	base := ui.Hex(hex)
	q := *p
	q.accent = base
	q.tabActiveBorder = base
	q.acSoft, q.acSoft2, q.acLine = base.Alpha(0.12), base.Alpha(0.20), base.Alpha(0.45)
	q.glow, q.glowS = base.Alpha(0.42), base.Alpha(0.26)
	if p.dark {
		q.accentHover = base.Mix(ui.RGB(255, 255, 255), 0.16)
		q.sideActive = base.Alpha(0.24)
		q.selected = base.Alpha(0.2)
	} else {
		q.accentHover = base.Mix(ui.RGB(0, 0, 0), 0.14)
		q.sideActive = base.Alpha(0.14)
		q.selected = base.Alpha(0.13)
	}
	return &q
}

// theme 把调色板转换为 MyGo 主题，让未自绘的内置控件（输入框、下拉框等）也与旧版一致。
func (p *palette) theme(fontSize float32) *ui.Theme {
	th := ui.LightTheme()
	if p.dark {
		th = ui.DarkTheme()
	}
	th.Background = p.surface
	th.Surface = p.btnBg
	th.SurfaceHover = p.selected
	th.SurfacePressed = p.selected
	th.Border = p.border
	th.Text = p.fg
	th.TextMuted = p.muted
	th.Accent = p.accent
	th.AccentHover = p.accentHover
	th.AccentPressed = p.accentHover
	th.AccentText = p.accentFg
	th.InactiveAccent = p.accent
	th.Danger = p.danger
	th.Warning = p.warning
	th.Success = p.success
	th.Selection = p.accent.Alpha(0.35)
	th.Radius = radiusInput
	th.Spacing = 3
	th.FontSize = fontSize
	th.ScrollbarWidth = 6
	return th
}

// methodAbbr 返回树、标签与历史里用的方法缩写（设计稿：DELETE→DEL、OPTIONS→OPT，其余原样），
// 保证最长 5 个字符，能在 32px 的方法列里对齐。
func methodAbbr(m string) string {
	switch strings.ToUpper(m) {
	case "DELETE":
		return "DEL"
	case "OPTIONS":
		return "OPT"
	}
	return m
}

// method 返回 HTTP 方法对应的颜色。
func (p *palette) method(m string) ui.Color {
	switch m {
	case "GET":
		return p.get
	case "POST":
		return p.post
	case "PUT":
		return p.put
	case "PATCH":
		return p.patch
	case "DELETE":
		return p.del
	case "HEAD":
		return p.head
	}
	return p.options
}

// ---- 图标 ----

var (
	svgMu    sync.Mutex
	svgCache = map[string]*ui.SVG{}
)

// iconAlias 把界面代码里沿用的旧图标名映射到设计稿图标集（icons/*.svg 全部取自 docs/design/assets/icons.js）。
var iconAlias = map[string]string{
	"add": "plus", "arrowDown": "chevD", "arrowRight": "chevR", "arrowLeft": "chevL", "clear": "x", "delete": "trash",
	"fileAdd": "filePlus", "folderAdd": "folderPlus", "tick": "check", "tbBack": "arrowL", "tbForward": "arrowR",
	"tbChevDown": "chevD", "tbHome": "home", "tbMoon": "moon", "tbSun": "sun", "tbSettings": "sliders", "tbSidebar": "sidebar",
	"tbImport": "download", "commandLine": "terminal", "settings": "sliders", "paintBoard": "palette", "formatList": "list",
	"wrapText": "wrap", "viewOff": "eyeOff", "sidebarCollapse": "sidebar", "sidebarExpand": "sidebar",
	"toastError": "errorC", "toastSuccess": "okC", "toastWarn": "warn", "fileImport": "download",
	"arrowUpDown": "refresh", "arrowLeftRight": "swap",
}

// svgIcon 读取内置图标（internal/gui/icons/*.svg），按名称缓存；旧名称先经 iconAlias 转成设计稿图标名。
func svgIcon(name string) *ui.SVG {
	if to, ok := iconAlias[name]; ok {
		name = to
	}
	svgMu.Lock()
	defer svgMu.Unlock()
	if s, ok := svgCache[name]; ok {
		return s
	}
	data, err := assetFS.ReadFile("icons/" + name + ".svg")
	if err != nil {
		logger.Warn("读取图标失败", "name", name, "error", err.Error())
		svgCache[name] = nil
		return nil
	}
	s, err := ui.ParseSVG(data)
	if err != nil {
		logger.Warn("解析图标失败", "name", name, "error", err.Error())
	}
	svgCache[name] = s
	return s
}

var (
	logoOnce [2]sync.Once
	logoImgs [2]ui.ImageSource
)

// logoImage 返回应用图标（空状态、设置-关于使用）：深色主题用深色版，浅色主题用浅色版。
func logoImage(dark bool) ui.ImageSource {
	i, name := 0, "icons/logo.png"
	if dark {
		i, name = 1, "icons/logo-dark.png"
	}
	logoOnce[i].Do(func() {
		data, err := assetFS.ReadFile(name)
		if err == nil {
			logoImgs[i], err = ui.DecodeImage(data)
		}
		if err != nil {
			logger.Warn("加载应用图标失败", "file", name, "error", err.Error())
		}
	})
	return logoImgs[i]
}

// ---- 基础控件 ----

// icon 绘制一个 size×size 的图标。
func icon(c *ui.Context, name string, size float32, col ui.Color) ui.Element {
	s := svgIcon(name)
	if s == nil {
		return ui.Box(c).Size(size, size)
	}
	return ui.Icon(c, s).Size(size, size).TextColor(col)
}

// textShiftY 把文本整体上移的像素数：Chrome 会把基线对齐到整数像素，而 MyGo 按小数定位，
// 实测同一行文字 MyGo 平均偏低约 0.5~0.8px，这里统一补偿以便与旧版逐像素对齐。
var textShiftY float32 = -1

// txt 绘制单行文本，指定字号、颜色与字重（0 为常规）。
func txt(c *ui.Context, s string, size float32, col ui.Color, weight int) ui.Element {
	return txtFlat(c, s, size, col, weight).Translate(0, textShiftY)
}

// txtE 是可截断的单行文本：超出容器宽度时以省略号结尾（txt 为 NoWrap，不会截断）。
func txtE(c *ui.Context, s string, size float32, col ui.Color, weight int) ui.Element {
	return txtEFlat(c, s, size, col, weight).Translate(0, textShiftY)
}

// txtEFlat 同 txtE 但不做基线补偿。
func txtEFlat(c *ui.Context, s string, size float32, col ui.Color, weight int) ui.Element {
	e := ui.Text(c, s).FontSize(size).TextColor(col).FixedLineHeight(size * 1.5).SingleLine().Ellipsis("…")
	if weight > 0 {
		e = e.FontWeight(weight)
	}
	return e
}

// txtEMono 是可截断的等宽单行文本。
func txtEMono(c *ui.Context, s string, size float32, col ui.Color, weight int) ui.Element {
	return txtE(c, s, size, col, weight).Font(monoFont)
}

// txtFlat 与 txt 相同但不做上移补偿，用于旧版文字基线本来就与 MyGo 一致的位置（分段标签、标签页标题等）。
func txtFlat(c *ui.Context, s string, size float32, col ui.Color, weight int) ui.Element {
	e := ui.Text(c, s).FontSize(size).TextColor(col).NoWrap().FixedLineHeight(size * 1.5) // 旧版 Tailwind 默认 line-height: 1.5
	if weight > 0 {
		e = e.FontWeight(weight)
	}
	return e
}

// monoFont 是等宽字体族，同旧版 --font-mono（macOS 上依次回退到 Menlo）。
const monoFont = "SF Mono, Menlo"

// txtMono 绘制等宽单行文本（方法标签、状态码、代码）。
func txtMono(c *ui.Context, s string, size float32, col ui.Color, weight int) ui.Element {
	return txt(c, s, size, col, weight).Font(monoFont)
}

// hline 与 vline 是 1px 的分隔线。
func hline(c *ui.Context, col ui.Color) ui.Element {
	return ui.Box(c).Height(1).FillWidth().Background(col).Shrink(0)
}

func vline(c *ui.Context, col ui.Color) ui.Element {
	return ui.Box(c).Width(1).FillHeight().Background(col).Shrink(0)
}

// btnOpts 描述一个图标/文字按钮的外观。
type btnOpts struct {
	w, h     float32 // 0 表示按内容
	radius   float32
	active   bool      // 选中态（背景加深）
	bg       *ui.Color // 常态背景，nil 为透明
	border   *ui.Color
	tip      string
	disabled bool
	padX     float32
	gap      float32
	key      any
	hoverBg  *ui.Color
	noHover  bool
}

// button 绘制一个自定义外观的按钮，content 绘制其内容，返回是否被点击。
func (a *app) button(c *ui.Context, o btnOpts, content func(hover bool)) bool {
	p := a.p
	if o.key != nil {
		c = c.Key(o.key)
	}
	b := ui.ButtonBase(c).Radius(o.radius).AlignItems(ui.Center).Justify(ui.Center).Shrink(0)
	if o.w > 0 {
		b = b.Width(o.w)
	}
	if o.h > 0 {
		b = b.Height(o.h)
	}
	if o.padX > 0 {
		b = b.PaddingX(o.padX)
	}
	if o.gap > 0 {
		b = b.Gap(o.gap)
	}
	hover := b.Hovered() && !o.disabled
	switch {
	case o.active:
		b = b.Background(p.selected)
	case hover && o.hoverBg != nil:
		b = b.Background(*o.hoverBg)
	case hover && !o.noHover:
		b = b.Background(p.hover)
	case o.bg != nil:
		b = b.Background(*o.bg)
	}
	if o.border != nil {
		b = b.Border(1, *o.border)
	}
	if o.tip != "" {
		b = b.Tooltip(o.tip).Label(o.tip) // 同时作为无障碍名称
	}
	if o.disabled {
		b = b.Disabled(true)
	}
	b.Children(func() { content(hover) })
	return b.Clicked()
}

// iconButton 是 24×24 的图标按钮（工具栏、侧栏操作条）。
func (a *app) iconButton(c *ui.Context, name, tip string, isize float32, col ui.Color) bool {
	return a.button(c, btnOpts{w: sizeBtn, h: sizeBtn, radius: radiusBtn, tip: tip, key: "ib:" + name + tip}, func(bool) {
		icon(c, name, isize, col)
	})
}

// fieldOpts 描述带边框的单行输入框外观。
type fieldOpts struct {
	w, h   float32 // w 为 0 时撑满剩余宽度
	radius float32
	icon   string // 左侧图标名
	padX   float32
	fs     float32
	key    any
	bg     *ui.Color
	flat   bool // 无边框（用于表格单元格）
	focus  bool // 出现时自动获得焦点
	// bd 非 nil 时覆盖默认边框色（弹窗内的输入框用 btnBorder）。
	bd *ui.Color
	// iconSize/iconLeft/iconGap 覆盖搜索图标的尺寸、左边距与到文字的距离，0 表示默认（12/6/6）。
	iconSize, iconLeft, iconGap float32
	// trail 非 nil 时在输入框右端绘制附加内容（如 ⌘K 提示），不参与输入。
	trail func()
	mono  bool // 输入文字使用等宽字体
}

// input 绘制 TablePlus 风格的输入框：圆角边框，获得焦点时边框变为强调色。返回内部输入框以便读取 Changed/Submitted。
func (a *app) input(c *ui.Context, v *string, ph string, o fieldOpts) ui.Element {
	p := a.p
	if o.key != nil {
		c = c.Key(o.key)
	}
	if o.fs == 0 {
		o.fs = fsBase
	}
	if o.h == 0 {
		o.h = 24
	}
	bg := p.surface
	if o.bg != nil {
		bg = *o.bg
	}
	// MinHeight 兜底：放进 Column 时 Grow/Basis(0) 是纵向的，没有它高度会塌成 0。
	wrap := ui.Row(c).Height(o.h).MinHeight(o.h).AlignItems(ui.Center).Radius(o.radius).Background(bg).Shrink(1)
	if o.w > 0 {
		wrap = wrap.Width(o.w).Shrink(0)
	} else {
		wrap = wrap.Grow(1).Basis(0)
	}
	if !o.flat {
		// 设计稿 .inp：1px 描边（bd2），聚焦时描边变强调色并带 3px 淡色光环 + 柔和光晕；侧栏搜索框用细边（bd）
		col := p.btnBorder
		if o.bd != nil {
			col = *o.bd
		}
		if wrap.FocusWithin() {
			col = p.accent
			wrap = wrap.Shadow(0, 0, 0, 3, p.acSoft).Shadow(0, 4, 14, -6, p.glow)
		}
		wrap = wrap.Border(1, col)
	}
	var in ui.Element
	wrap.Children(func() {
		pad := o.padX
		if o.icon != "" {
			isz, ileft, igap := o.iconSize, o.iconLeft, o.iconGap
			if isz == 0 {
				isz = 12
			}
			if ileft == 0 {
				ileft = 6
			}
			if igap == 0 {
				igap = 6
			}
			ui.Box(c).Width(ileft).Shrink(0)
			icon(c, o.icon, isz, p.muted)
			pad = igap
		} else if pad == 0 {
			pad = 8
		}
		in = ui.TextInputBase(c, v).Placeholder(ph).PlaceholderColor(p.muted).FontSize(o.fs).TextColor(p.fg).
			Grow(1).Basis(0).FillHeight().PaddingX(pad)
		if o.mono {
			in = in.Font(monoFont)
		}
		if o.focus {
			in = in.AutoFocus()
		}
		if o.trail != nil {
			o.trail()
		}
	})
	return in
}

// checkbox 绘制 12×12 的复选框：选中时为强调色实心加白色对勾。返回是否被切换。
func (a *app) checkbox(c *ui.Context, key any, on *bool, disabled bool) bool {
	p := a.p
	if key != nil {
		c = c.Key(key)
	}
	// 设计稿 .cb：15px、圆角 4.5、1.5px 描边；选中为强调色渐变 + 光晕 + 粗勾
	b := ui.CheckboxBase(c, on).Size(15, 15).Radius(4.5).Shrink(0).AlignItems(ui.Center).Justify(ui.Center)
	if *on {
		b = p.gradAccent(b, false).Border(1.5, p.accent).Shadow(0, 1, 3, 0, p.glow)
	} else {
		bc := p.btnBorder
		if b.Hovered() {
			bc = p.acLine
		}
		b = b.Background(p.surface).Border(1.5, bc)
	}
	if disabled {
		b = b.Disabled(true).Opacity(0.3)
	}
	b.Children(func() {
		if *on {
			icon(c, "checkBold", 10, p.accentFg)
		}
	})
	return b.Changed()
}

// countZero 传给 underlineTabs 表示「数量为 0 时也显示 0」（响应头 / Cookies 标签）。
const countZero = -1

// tabExtra 是标签名后的附加标记：badge 为等宽小标签，lock 表示显示锁图标，dot 表示显示强调色小圆点（Body / Auth 已配置）。
type tabExtra struct {
	badge string
	lock  bool
	dot   bool
}

// underlineTabs 是设计稿的下划线标签（Params/Headers/Body/Auth、响应体/响应头等 .ut）：
// 每个标签高 h、左右内边距 9，选中项加粗并带 2px 强调色下划线（带光晕，下沿压在下方分隔线上），
// 数量显示为名称后的小徽标（.cnt：选中时为强调色淡底），extras 可为 nil。
func (a *app) underlineTabs(c *ui.Context, key string, sel *int, labels []string, counts []int, extras []tabExtra, h float32) bool {
	p := a.p
	changed := false
	for i, l := range labels {
		n := 0
		if i < len(counts) {
			n = counts[i]
		}
		show := n > 0 || n == countZero
		if n == countZero {
			n = 0
		}
		var ex tabExtra
		if i < len(extras) {
			ex = extras[i]
		}
		active := *sel == i
		b := ui.ButtonBase(c.Key(key + l)).Height(h).PaddingX(9).AlignItems(ui.Center).Shrink(0)
		col, w := p.fg2, 550
		hover := b.Hovered()
		switch {
		case active:
			col, w = p.fg, 650
		case hover:
			col = p.fg
		}
		b.Children(func() {
			if active {
				// 下划线压在行底的分隔线上（-1），左右各缩进 9
				ui.Box(c).Absolute().Left(9).Right(9).Bottom(-1).Height(2).Radius(2, 2, 0, 0).Background(p.accent).Shadow(0, -1, 8, 0, p.glow)
			} else if hover {
				ui.Box(c).Absolute().Left(9).Right(9).Bottom(-1).Height(2).Radius(2, 2, 0, 0).Background(p.btnBorder)
			}
			ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
				txtFlat(c, l, fsBase, col, w)
				if show {
					bg, fc := p.surface2, p.muted
					cnt := ui.Row(c).Height(15).MinWidth(16).PaddingX(4).Radius(8).AlignItems(ui.Center).Justify(ui.Center)
					if active {
						bg, fc = p.acSoft, p.accent
						cnt = cnt.Border(1, p.acSoft2)
					}
					cnt.Background(bg).Children(func() { txtMono(c, itoa(n), fsTiny, fc, 600) })
				}
				if ex.badge != "" {
					ui.Row(c).Height(15).PaddingX(5).Radius(4).Background(p.surface2).AlignItems(ui.Center).Children(func() {
						txtMono(c, ex.badge, 9, p.muted, 500)
					})
				}
				if ex.lock {
					icon(c, "lock", 10, p.accent)
				}
				if ex.dot {
					ui.Box(c).Size(6, 6).Radius(3).Background(p.accent).Shadow(0, 0, 0, 2, p.acSoft).Shrink(0)
				}
			})
		})
		if b.Clicked() && !active {
			*sel = i
			changed = true
		}
	}
	return changed
}

// kindPills 是 Body/Auth 的类型小标签：24px 高的胶囊，选中时为浅强调色底与加粗深色文字，未选中为次要色。
func (a *app) kindPills(c *ui.Context, key string, sel *int, labels []string, padX float32) bool {
	p := a.p
	changed := false
	for i, l := range labels {
		active := *sel == i
		b := ui.ButtonBase(c.Key(key + l)).Height(24).Radius(12).PaddingX(padX + 2).AlignItems(ui.Center).Shrink(0)
		col, w := p.fg2, 0
		switch {
		case active:
			b = b.Background(p.accent.Alpha(0.13))
			col, w = p.fg, 600
		case b.Hovered():
			b = b.Background(p.hover)
			col = p.fg
		}
		b.Children(func() { txt(c, l, fsBase, col, w) })
		if b.Clicked() && !active {
			*sel = i
			changed = true
		}
	}
	return changed
}

// codeView 绘制代码视图（请求体编辑器、响应体）：配色取自旧版 Monaco 的 postman 主题。
func (a *app) codeView(c *ui.Context, st *editor.State) ui.Element {
	if st.Theme == nil || st.Theme != a.edTheme() {
		st.Theme = a.edTheme()
	}
	if st.Font.Size == 0 {
		st.Font = ui.Font{Family: monoFont, Size: 12}
	}
	st.LineHeight = 20.0 / 12.0
	return editor.View(c, st)
}

var edThemes = map[string]*editor.Theme{}

// buildEdTheme 按调色板生成代码配色：背景与内容区一致，高亮色取自 JSON 令牌，当前行/选区由强调色淡化而来。
func buildEdTheme(p *palette) *editor.Theme {
	t := &editor.Theme{}
	t.Background, t.Text = p.surface, p.fg
	t.LineNumber, t.ActiveLineNumber = p.muted, p.fg
	t.ActiveLine, t.Selection = p.surface.Mix(p.accent, 0.06), p.accent.Alpha(0.28)
	t.Cursor = p.fg
	t.Tokens[editor.TokenProperty].Color = p.jsonKey
	t.Tokens[editor.TokenString].Color = p.jsonString
	t.Tokens[editor.TokenNumber].Color = p.jsonNumber
	t.Tokens[editor.TokenConstant].Color = p.jsonBool
	t.Tokens[editor.TokenKeyword].Color = p.jsonBool
	return t
}

// edTheme 返回当前配色对应的代码配色（按键缓存，保证同一配色返回同一指针）。
func (a *app) edTheme() *editor.Theme {
	// 项目主题色会改变强调色，因此缓存键里要带上强调色与明暗（方案不同时调色板的令牌也不同）。
	key := fmt.Sprintf("%s|%v|%v", a.set.ColorScheme, a.p.dark, a.p.accent)
	if t, ok := edThemes[key]; ok {
		return t
	}
	t := buildEdTheme(a.p)
	edThemes[key] = t
	return t
}

// dialog 绘制居中的对话框：半透明遮罩 + 固定宽高、圆角、细边框与大阴影的面板（同旧版弹窗）。
func (a *app) dialog(c *ui.Context, open *bool, w, h, radius float32, fn func()) {
	a.dialogDim(c, open, w, h, radius, 0.35, fn)
}

// dialogDim 与 dialog 相同，但可指定遮罩的黑色透明度（旧版 cURL 弹窗用 0.5）。
func (a *app) dialogDim(c *ui.Context, open *bool, w, h, radius, dim float32, fn func()) {
	p := a.p
	cw, ch := c.Size()
	ui.DialogBase(c, open, func(back, panel ui.Element) {
		// 设计稿 .dlg：大圆角（--r-lg）、近乎不透明的玻璃底（无法做背景模糊，故用高不透明度）、细边框 + 顶部高光 + 大投影
		back.Background(ui.RGBA(0, 0, 0, dim))
		radius = max(radius, radiusLg)
		panel.Width(w).Radius(radius).Background(p.elevated().Alpha(0.97)).Border(1, p.border).Clip().
			Shadow(0, 14, 40, -8, p.shadowLg()).Shadow(0, 2, 6, 0, p.shadowLg()).InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.18))
		if h > 0 {
			panel.Height(h)
		}
		panel.MaxWidth(cw * 0.92).MaxHeight(ch * 0.88)
		fn()
	})
}

// toggle 是 36×20 的开关：开启时为强调色，旋钮 15px 白色圆。返回是否被切换。
func (a *app) toggle(c *ui.Context, key string, on *bool) bool {
	p := a.p
	bg, bd := p.btnBg, p.btnBorder
	just := ui.Start
	if *on {
		bg, bd, just = p.accent, p.accent, ui.End
	}
	b := ui.ButtonBase(c.Key(key)).Size(36, 20).Radius(10).Background(bg).Border(1, bd).Padding(1).AlignItems(ui.Center).Justify(just).Shrink(0)
	b.Children(func() {
		ui.Box(c).Size(15, 15).Radius(7.5).Background(ui.Hex("#ffffff")).Border(1, ui.RGBA(0, 0, 0, 0.05)).Shrink(0)
	})
	if b.Clicked() {
		*on = !*on
		return true
	}
	return false
}

// segOpts 描述分段控件的外观：full 为撑满宽度（各段等宽），h 为每段高度（默认 22，整体再加上下各 2 的轨道内边距）。
type segOpts struct {
	full bool
	h    float32
	padX float32
	fs   float32
}

// pillSeg 是设计稿方案 C 的分段控件（.seg）：全圆角的浅灰轨道（前景 7%），选中段是浮起的内容色胶囊，
// 文字加粗、图标用强调色。icons 可为 nil（无图标）。返回是否变化。
func (a *app) pillSeg(c *ui.Context, key string, sel *int, labels, icons []string, o segOpts) bool {
	p := a.p
	if o.h == 0 {
		o.h = 22
	}
	if o.padX == 0 {
		o.padX = 10
	}
	if o.fs == 0 {
		o.fs = fsBase
	}
	changed := false
	track := ui.Row(c.Key(key)).Height(o.h + 4).Padding(2).Gap(2).Radius((o.h + 4) / 2).Background(p.fg.Alpha(0.07)).AlignItems(ui.Stretch).Shrink(0)
	if o.full {
		track = track.Grow(1).Basis(0)
	}
	track.Children(func() {
		for i, l := range labels {
			active := *sel == i
			b := ui.ButtonBase(c.Key(key + itoa(i) + l)).Radius(o.h / 2).PaddingX(o.padX).Gap(5).AlignItems(ui.Center).Justify(ui.Center)
			if o.full {
				b = b.Grow(1).Basis(0)
			}
			col, w := p.fg2, 500
			switch {
			case active:
				b = p.shadowSm(b.Background(p.surface))
				col, w = p.fg, 600
			case b.Hovered():
				b = b.Background(p.hover)
				col = p.fg
			}
			b.Children(func() {
				if i < len(icons) && icons[i] != "" {
					ic := p.muted
					if active {
						ic = p.accent
					}
					icon(c, icons[i], 13, ic)
				}
				if l != "" {
					txt(c, l, o.fs, col, w)
				}
			})
			if b.Clicked() && !active {
				*sel = i
				changed = true
			}
		}
	})
	return changed
}

// segmented 是分段控件（设置页等）：等同 pillSeg 的默认样式。
func (a *app) segmented(c *ui.Context, key string, sel *int, labels []string) bool {
	return a.pillSeg(c, key, sel, labels, nil, segOpts{})
}

// 对话框按钮的种类。
const (
	dlgNormal  = iota // 描边按钮
	dlgPrimary        // 强调色实心
	dlgDanger         // 危险色实心
)

// dbtn 描述设计稿 .btn 按钮：kind 取 dlgNormal/dlgPrimary/dlgDanger（见下），ghost 为无边框无底，sm 为 22 高小号。
type dbtn struct {
	key      any
	label    string
	icon     string // 文字前的图标名，空表示没有
	kind     int
	ghost    bool
	sm       bool
	danger   bool // 文字用危险色（kind 为 dlgNormal 时的 .btn.dng）
	disabled bool
	tip      string
	suffix   string // 文字后的等宽小字（如快捷键提示）
	w        float32
	h        float32 // 0 为默认高度（26，小号 22）
	grow     bool    // 在行内等分剩余宽度
}

// btn 绘制设计稿 .btn：普通按钮是浮起的浅渐变 + 1px 边 + 细投影；主按钮是强调色渐变 + 光晕；
// 危险主按钮用危险色渐变；ghost 悬停才有底。返回是否被点击。
func (a *app) btn(c *ui.Context, o dbtn) bool {
	p := a.p
	h, padX, fs := float32(26), float32(11), float32(fsBase)
	if o.sm {
		h, padX, fs = 22, 8, fsSmall
	}
	if o.h > 0 {
		h = o.h
	}
	if o.key == nil {
		o.key = "btn:" + o.label
	}
	b := ui.ButtonBase(c.Key(o.key)).Height(h).PaddingX(padX).Gap(6).Radius(10).AlignItems(ui.Center).Justify(ui.Center).Shrink(0)
	if o.w > 0 {
		b = b.Width(o.w)
	}
	if o.grow {
		b = b.Grow(1).Basis(0)
	}
	hover := b.Hovered() && !o.disabled
	col, wt := p.fg, 550
	if o.danger {
		col = p.danger
	}
	switch {
	case o.kind == dlgPrimary:
		b = p.gradAccent(b, hover).Border(1, p.accent.Mix(ui.RGB(0, 0, 0), 0.14))
		if hover {
			b = b.Shadow(0, 5, 16, -4, p.glow)
		} else {
			b = b.Shadow(0, 1, 3, 0, p.glow)
		}
		col = p.accentFg
	case o.kind == dlgDanger:
		top := p.danger.Mix(ui.RGB(255, 255, 255), 0.16)
		b = b.Gradient(top, p.danger, 180).InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.24)).Border(1, p.danger.Mix(ui.RGB(0, 0, 0), 0.14))
		col = ui.RGB(255, 255, 255)
	case o.ghost:
		if hover {
			b = b.Background(p.hover)
		}
	default:
		bg := p.surface
		switch {
		case hover && o.danger:
			bg = p.errSoft
		case hover:
			bg = p.surface2
		}
		b = b.Background(bg).Border(1, p.btnBorder)
		if !hover {
			b = b.Shadow(0, 1, 2, 0, ui.RGBA(20, 24, 40, 0.05))
		}
	}
	if o.tip != "" {
		b = b.Tooltip(o.tip).Label(o.tip)
	}
	if o.disabled {
		b = b.Disabled(true).Opacity(0.45)
	}
	b.Children(func() {
		if o.icon != "" {
			icon(c, o.icon, 13, col)
		}
		if o.label != "" {
			txt(c, o.label, fs, col, wt)
		}
		if o.suffix != "" {
			txtMono(c, o.suffix, fsTiny, col.Alpha(0.75), 0)
		}
	})
	return b.Clicked()
}

// dlgButton 是对话框底部的按钮，外观取设计稿 .btn。返回是否被点击。
func (a *app) dlgButton(c *ui.Context, key, label string, kind int, disabled bool) bool {
	return a.btn(c, dbtn{key: key, label: label, kind: kind, disabled: disabled})
}

// stdDialog 绘制标准对话框（设计稿 .dlg）：head、body 与可选的 foot 自上而下排列；
// 标题行（12/14/10 内边距、顶部淡渐变）和底栏（10/14 内边距、淡底）与正文之间有细分隔线。
func (a *app) stdDialog(c *ui.Context, open *bool, w float32, head, body, foot func()) {
	p := a.p
	a.dialog(c, open, w, 0, radiusLg, func() {
		ui.Column(c).FillWidth().Children(func() {
			ui.Row(c).FillWidth().Padding(12, 14, 10, 14).Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).
				Gradient(p.surface2.Alpha(0.38), p.surface2.Alpha(0), 180).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(head)
			ui.Column(c).FillWidth().Children(body)
			if foot != nil {
				ui.Row(c).FillWidth().Padding(10, 14).Gap(8).AlignItems(ui.Center).Justify(ui.End).
					Background(p.surface2.Alpha(0.55)).BorderWidth(1, 0, 0, 0).BorderColor(p.border).Children(foot)
			}
		})
	})
}

// compactDialog 是设计稿的小号提示框（.dlg.dsm，确认删除、保存更改）：标题行 + 正文 + 底栏，宽 400。
func (a *app) compactDialog(c *ui.Context, open *bool, title string, body, foot func()) {
	a.stdDialog(c, open, 400, func() { a.dlgTitle(c, open, title, true) }, func() {
		ui.Column(c).FillWidth().Padding(14).Gap(12).Children(body)
	}, foot)
}

// dlgMsg 是小号提示框正文里的“图标 + 粗体一行 + 说明”（设计稿 .dsm .msg）：34px 圆形图标带 5px 淡色外环。
func (a *app) dlgMsg(c *ui.Context, kind, title, desc string) {
	p := a.p
	ic, col, soft := a.toastIcon(kind)
	switch kind {
	case toastError:
		ic = "trash"
	case toastWarn:
		ic = "warn"
	}
	ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Start).Children(func() {
		ui.Row(c).Size(34, 34).Shrink(0).Radius(17).Background(soft).Shadow(0, 0, 0, 5, col.Alpha(0.09)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			icon(c, ic, 17, col)
		})
		ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(fsLarge).FontWeight(650).TextColor(p.fg).FixedLineHeight(18)
			if desc != "" {
				ui.Text(c, desc).FontSize(fsBase).TextColor(p.fg2).FixedLineHeight(18)
			}
		})
	})
}

// dlgTitle 是对话框标题行：13.5px 粗体标题，closable 时右侧带关闭按钮。
func (a *app) dlgTitle(c *ui.Context, open *bool, title string, closable bool) {
	txt(c, title, 13.5, a.p.fg, 650)
	if closable {
		a.ibSm(c, "dlg-close:"+title, "x", a.t("关闭", "Close"), false, func() { *open = false })
	}
}

// segTabs 是流式视图等处的分段控件（“事件 / 合并文本”），样式同 pillSeg。
func (a *app) segTabs(c *ui.Context, key string, sel *int, labels []string) bool {
	return a.pillSeg(c, key, sel, labels, nil, segOpts{padX: 12})
}

// dlgIcon 绘制对话框标题行左侧的 28px 图标块（设计稿 .dlg-h .ico：淡强调色渐变底 + 细环 + 强调色图标）。
func (a *app) dlgIcon(c *ui.Context, name string) {
	p := a.p
	ui.Row(c).Size(28, 28).Shrink(0).Radius(radiusSm).Gradient(p.acSoft, p.acSoft.Alpha(0.06), 180).Shadow(0, 0, 0, 1, p.acSoft2).Shadow(0, 2, 8, -2, p.glowS).
		AlignItems(ui.Center).Justify(ui.Center).Children(func() { icon(c, name, 14, p.accent) })
}
