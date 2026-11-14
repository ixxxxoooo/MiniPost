package gui

import (
	"embed"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"minipost/internal/pkg/logger"
)

// 本文件是界面的设计系统：调色板、尺寸常量与基础控件，数值全部取自旧版
// frontend/src/styles/globals.css 与各组件的实际渲染结果（TablePlus 风格）。

//go:embed icons/*.svg icons/logo.png
var assetFS embed.FS

// 尺寸常量（单位 DIP，与旧版 CSS 像素一致）。
const (
	sizeToolbar  = 37  // 顶部自绘标题栏高度
	sizeBottom   = 25  // 底栏高度
	sizeTabStrip = 28  // 侧栏标签条与请求标签栏高度
	sizeBtn      = 24  // 工具栏图标按钮
	radiusBtn    = 7   // 按钮圆角
	radiusInput  = 8   // 输入框圆角
	radiusCard   = 10  // 卡片圆角
	fsBase       = 12  // 基础字号
	fsSmall      = 11  // 次级字号
	fsTiny       = 10  // 微型字号
	fsLarge      = 13  // 大字号（URL、响应标题）
	sizeSideMin  = 180 // 侧栏最小宽度
)

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
}

func lightPalette() *palette {
	return &palette{
		accent: ui.Hex("#007aff"), accentHover: ui.Hex("#0066d6"), accentFg: ui.Hex("#ffffff"),
		surface: ui.Hex("#ffffff"), surface2: ui.Hex("#f9f9f9"),
		fg: ui.Hex("#1d1d1f"), fg2: ui.Hex("#6e6e73"), muted: ui.Hex("#aeaeb2"),
		border: ui.Hex("#ededed"), btnBg: ui.RGB(237, 237, 237), btnBorder: ui.RGB(230, 230, 230), selected: ui.RGB(242, 242, 242),
		toolbar: ui.RGB(250, 250, 250), sidebar: ui.RGB(249, 249, 249), tabStrip: ui.RGB(249, 249, 249), bottom: ui.RGB(249, 249, 249),
		sideActive: ui.RGBA(0, 122, 255, 0.12), sideHover: ui.RGBA(0, 0, 0, 0.04), tabActive: ui.Hex("#ffffff"),
		tabDivider: ui.Hex("#d9d9dd"), tabActiveBorder: ui.Hex("#007aff"),
		success: ui.Hex("#34c759"), warning: ui.Hex("#ff9500"), danger: ui.Hex("#ff3b30"), info: ui.Hex("#5ac8fa"),
		get: ui.Hex("#34c759"), post: ui.Hex("#007aff"), put: ui.Hex("#ff9500"), patch: ui.Hex("#af52de"),
		del: ui.Hex("#ff3b30"), head: ui.Hex("#6e6e73"), options: ui.Hex("#5ac8fa"),
		jsonKey: ui.Hex("#881391"), jsonString: ui.Hex("#1a7f37"), jsonNumber: ui.Hex("#0550ae"), jsonBool: ui.Hex("#cf222e"), jsonBracket: ui.Hex("#6e6e73"),
		hover: ui.RGBA(0, 0, 0, 0.04),
	}
}

func darkPalette() *palette {
	return &palette{
		dark:   true,
		accent: ui.Hex("#0a84ff"), accentHover: ui.Hex("#409cff"), accentFg: ui.Hex("#ffffff"),
		surface: ui.RGB(33, 33, 33), surface2: ui.RGB(48, 48, 48),
		fg: ui.Hex("#f5f5f7"), fg2: ui.Hex("#b8b8be"), muted: ui.Hex("#8a8a90"),
		border: ui.RGB(48, 48, 48), btnBg: ui.RGB(59, 59, 59), btnBorder: ui.RGB(48, 48, 48), selected: ui.RGB(68, 68, 68),
		toolbar: ui.RGB(33, 33, 33), sidebar: ui.RGB(33, 33, 33), tabStrip: ui.RGB(48, 48, 48), bottom: ui.RGB(48, 48, 48),
		sideActive: ui.RGBA(10, 132, 255, 0.18), sideHover: ui.RGB(48, 48, 48), tabActive: ui.RGB(48, 48, 48),
		tabDivider: ui.RGBA(255, 255, 255, 0.16), tabActiveBorder: ui.Hex("#0a84ff"),
		success: ui.Hex("#30d158"), warning: ui.Hex("#ff9f0a"), danger: ui.Hex("#ff453a"), info: ui.Hex("#64d2ff"),
		get: ui.Hex("#30d158"), post: ui.Hex("#0a84ff"), put: ui.Hex("#ff9f0a"), patch: ui.Hex("#bf5af2"),
		del: ui.Hex("#ff453a"), head: ui.Hex("#8a8a90"), options: ui.Hex("#64d2ff"),
		jsonKey: ui.Hex("#79c0ff"), jsonString: ui.Hex("#a5d6ff"), jsonNumber: ui.Hex("#d2a8ff"), jsonBool: ui.Hex("#ff7b72"), jsonBracket: ui.Hex("#8a8a90"),
		hover: ui.RGB(48, 48, 48),
	}
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
	th.Border = p.btnBorder
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

// svgIcon 读取内置图标（internal/gui/icons/*.svg），按名称缓存。
func svgIcon(name string) *ui.SVG {
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
	logoOnce sync.Once
	logoImg  ui.ImageSource
)

// logoImage 返回应用图标（空状态使用）。
func logoImage() ui.ImageSource {
	logoOnce.Do(func() {
		data, err := assetFS.ReadFile("icons/logo.png")
		if err == nil {
			logoImg, err = ui.DecodeImage(data)
		}
		if err != nil {
			logger.Warn("加载应用图标失败", "error", err.Error())
		}
	})
	return logoImg
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

// txt 绘制单行文本，指定字号、颜色与字重（0 为常规）。
func txt(c *ui.Context, s string, size float32, col ui.Color, weight int) ui.Element {
	e := ui.Text(c, s).FontSize(size).TextColor(col).NoWrap()
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
	w, h       float32 // 0 表示按内容
	radius     float32
	active     bool      // 选中态（背景加深）
	bg         *ui.Color // 常态背景，nil 为透明
	border     *ui.Color
	tip        string
	disabled   bool
	padX       float32
	gap        float32
	key        any
	hoverBg    *ui.Color
	noHover    bool
	hoverCol   *ui.Color
	ghostHover bool
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

// pill 是带底色的圆角小标签（状态码、HTTP 徽标等）。
func (a *app) pill(c *ui.Context, text string, size float32, fg, bg ui.Color, radius, padX, h float32) {
	ui.Row(c).Height(h).PaddingX(padX).Radius(radius).Background(bg).AlignItems(ui.Center).Shrink(0).Children(func() {
		txt(c, text, size, fg, 600)
	})
}

// kbd 是快捷键小方块（⌘ N）。
func (a *app) kbd(c *ui.Context, s string) {
	p := a.p
	ui.Row(c).Width(20).Height(20).Radius(5).Background(p.surface2).Border(1, p.border).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		txt(c, s, fsTiny, p.fg2, 0)
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
	mono   bool
	focus  bool // 出现时自动获得焦点
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
	wrap := ui.Row(c).Height(o.h).AlignItems(ui.Center).Radius(o.radius).Background(bg).Shrink(1)
	if o.w > 0 {
		wrap = wrap.Width(o.w).Shrink(0)
	} else {
		wrap = wrap.Grow(1).Basis(0)
	}
	if !o.flat {
		col := p.border
		if wrap.FocusWithin() {
			col = p.accent
		}
		wrap = wrap.Border(1, col)
	}
	var in ui.Element
	wrap.Children(func() {
		pad := o.padX
		if o.icon != "" {
			ui.Box(c).Width(6).Shrink(0)
			icon(c, o.icon, 12, p.muted)
			pad = 6
		} else if pad == 0 {
			pad = 8
		}
		in = ui.TextInputBase(c, v).Placeholder(ph).PlaceholderColor(p.muted).FontSize(o.fs).TextColor(p.fg).
			Grow(1).Basis(0).FillHeight().PaddingX(pad)
		if o.focus {
			in = in.AutoFocus()
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
	b := ui.CheckboxBase(c, on).Size(12, 12).Radius(3).Shrink(0).AlignItems(ui.Center).Justify(ui.Center)
	if *on {
		b = b.Background(p.accent).Border(1, p.accent)
	} else {
		b = b.Background(p.surface).Border(1, p.btnBorder)
	}
	if disabled {
		b = b.Disabled(true).Opacity(0.3)
	}
	b.Children(func() {
		if *on {
			icon(c, "tick", 9, p.accentFg)
		}
	})
	return b.Changed()
}

// pillTabs 是 TablePlus 风格的分段标签（Params/Headers/Body/Auth、响应体/响应头等）：
// 选中项有浅灰底，数量显示在名称后。返回是否切换了选项。
func (a *app) pillTabs(c *ui.Context, key string, sel *int, labels []string, counts []int) bool {
	p := a.p
	changed := false
	for i, l := range labels {
		n := 0
		if i < len(counts) {
			n = counts[i]
		}
		h := float32(19)
		if n > 0 {
			h = 24
		}
		active := *sel == i
		b := ui.ButtonBase(c.Key(key + l)).Height(h).Radius(6).PaddingX(10).Gap(4).AlignItems(ui.Center).Shrink(0)
		col := p.muted
		hover := b.Hovered()
		switch {
		case active:
			b = b.Background(p.selected)
			col = p.fg
		case hover:
			col = p.fg2
		}
		b.Children(func() {
			txt(c, l, fsSmall, col, 500)
			if n > 0 {
				txt(c, "("+itoa(n)+")", 10.4, p.muted, 500)
			}
		})
		if b.Clicked() && !active {
			*sel = i
			changed = true
		}
	}
	return changed
}

// kindPills 是 Body/Auth 的类型小标签：12px，圆角 4，选中时为浅强调色底与强调色文字。
func (a *app) kindPills(c *ui.Context, key string, sel *int, labels []string) bool {
	p := a.p
	changed := false
	for i, l := range labels {
		active := *sel == i
		b := ui.ButtonBase(c.Key(key + l)).Height(26).Radius(4).PaddingX(10).AlignItems(ui.Center).Shrink(0)
		col, w := p.muted, 0
		switch {
		case active:
			b = b.Background(p.sideActive)
			col, w = p.accent, 500
		case b.Hovered():
			col = p.fg2
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

var (
	edThemeLight = &editor.Theme{}
	edThemeDark  = &editor.Theme{}
	edThemeOnce  sync.Once
)

func initEdThemes() {
	set := func(t *editor.Theme, bg, fg, ln, line, sel, key, str, num, kw string) {
		t.Background, t.Text = ui.Hex(bg), ui.Hex(fg)
		t.LineNumber, t.ActiveLineNumber = ui.Hex(ln), ui.Hex(fg)
		t.ActiveLine, t.Selection = ui.Hex(line), ui.Hex(sel)
		t.Cursor = ui.Hex(fg)
		t.Tokens[editor.TokenProperty].Color = ui.Hex(key)
		t.Tokens[editor.TokenString].Color = ui.Hex(str)
		t.Tokens[editor.TokenNumber].Color = ui.Hex(num)
		t.Tokens[editor.TokenConstant].Color = ui.Hex(kw)
		t.Tokens[editor.TokenKeyword].Color = ui.Hex(kw)
	}
	set(edThemeLight, "#ffffff", "#2f2f33", "#a0a0a8", "#f3f8ff", "#d8e7ff", "#C02F1D", "#0A4FA8", "#0F7B45", "#8F2D56")
	set(edThemeDark, "#212121", "#d3d7de", "#7d828a", "#212121", "#2d4f82", "#F28B82", "#80B7FF", "#8DDF99", "#F59CB5")
}

// edTheme 返回当前亮/暗模式对应的代码配色。
func (a *app) edTheme() *editor.Theme {
	edThemeOnce.Do(initEdThemes)
	if a.p.dark {
		return edThemeDark
	}
	return edThemeLight
}

// dialog 绘制居中的对话框：半透明遮罩 + 固定宽高、圆角、细边框与大阴影的面板（同旧版弹窗）。
func (a *app) dialog(c *ui.Context, open *bool, w, h, radius float32, fn func()) {
	p := a.p
	cw, ch := c.Size()
	ui.DialogBase(c, open, func(back, panel ui.Element) {
		back.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(w).Radius(radius).Background(p.surface).Border(1, p.border).Clip().Shadow(0, 20, 50, 0, ui.RGBA(0, 0, 0, 0.3))
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

// segmented 是分段控件：高 28、圆角 8、细边框，选中段为 selected 背景。返回是否变化。
func (a *app) segmented(c *ui.Context, key string, sel *int, labels []string) bool {
	p := a.p
	changed := false
	ui.Row(c.Key(key)).Height(28).Radius(radiusInput).Border(1, p.btnBorder).Background(p.surface).Clip().Shrink(0).AlignItems(ui.Stretch).Children(func() {
		for i, l := range labels {
			b := ui.ButtonBase(c.Key(l)).PaddingX(10).AlignItems(ui.Center).Justify(ui.Center)
			col := p.fg2
			switch {
			case *sel == i:
				b = b.Background(p.selected)
				col = p.fg
			case b.Hovered():
				b = b.Background(p.btnBg)
			}
			b.Children(func() { txt(c, l, fsSmall, col, 0) })
			if b.Clicked() && *sel != i {
				*sel = i
				changed = true
			}
		}
	})
	return changed
}

// 对话框按钮的种类。
const (
	dlgNormal  = iota // 描边按钮
	dlgPrimary        // 强调色实心
	dlgDanger         // 危险色实心
)

// dlgButton 是对话框底部的 32px 高按钮（圆角 8，12px 文字）。返回是否被点击。
func (a *app) dlgButton(c *ui.Context, key, label string, kind int, disabled bool) bool {
	p := a.p
	o := btnOpts{h: 32, radius: radiusInput, padX: 12, key: key, disabled: disabled}
	col, wt := p.fg, 0
	switch kind {
	case dlgPrimary:
		bg := p.accent
		o.bg, o.hoverBg = &bg, &p.accentHover
		col, wt = p.accentFg, 500
	case dlgDanger:
		bg := p.danger
		o.bg, o.hoverBg = &bg, &bg
		col, wt = ui.Hex("#ffffff"), 500
	default:
		bd := p.btnBorder
		o.border = &bd
	}
	if disabled {
		col = col.Alpha(0.5)
	}
	return a.button(c, o, func(bool) { txt(c, label, fsBase, col, wt) })
}

// stdDialog 绘制标准对话框：head、body 与可选的 foot 自上而下排列，head 与 foot 与正文之间有细分隔线。
func (a *app) stdDialog(c *ui.Context, open *bool, w float32, head, body, foot func()) {
	p := a.p
	a.dialog(c, open, w, 0, 12, func() {
		ui.Column(c).FillWidth().Children(func() {
			ui.Row(c).FillWidth().PaddingX(16).PaddingY(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(head)
			ui.Column(c).FillWidth().Children(body)
			if foot != nil {
				ui.Row(c).FillWidth().PaddingX(16).PaddingY(12).Gap(8).AlignItems(ui.Center).Justify(ui.End).BorderWidth(1, 0, 0, 0).BorderColor(p.border).Children(foot)
			}
		})
	})
}

// dlgTitle 是对话框标题行：16px 粗体标题，closable 时右侧带关闭按钮。
func (a *app) dlgTitle(c *ui.Context, open *bool, title string, closable bool) {
	p := a.p
	txt(c, title, 16, p.fg, 600)
	if closable && a.iconButton(c, "clear", a.t("关闭", "Close"), 12, p.muted) {
		*open = false
	}
}
