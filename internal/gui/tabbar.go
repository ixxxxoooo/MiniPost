package gui

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// tabBar 绘制 28px 高的请求标签栏：左侧是各请求标签与“+”，右侧是标签列表。
func (a *app) tabBar(c *ui.Context) {
	p := a.p
	ui.Row(c).FillWidth().Height(sizeTabStrip).Shrink(0).Background(p.tabStrip).AlignItems(ui.Stretch).Children(func() {
		stripTopLine(c, p.border)
		if len(a.tabs) == 0 {
			// 无标签时“+”独占一格，右侧带分隔线
			ui.Row(c).Width(40).PaddingX(6).AlignItems(ui.Center).Children(func() {
				a.newTabButton(c)
			})
			vline(c, p.tabDivider)
		}
		ui.ScrollHorizontal(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
			ui.Row(c).FillHeight().AlignItems(ui.End).Children(func() {
				for i, t := range a.tabs {
					a.tabItem(c, i, t)
				}
				if len(a.tabs) > 0 {
					ui.Row(c).Height(sizeTabStrip - 1).MarginX(6).AlignItems(ui.Center).Children(func() {
						a.newTabButton(c)
					})
				}
			})
		})
		if len(a.tabs) > 0 {
			// 环境选择已移到标题栏，这里只剩“显示所有标签”
			vline(c, p.tabDivider)
			ui.Row(c).PaddingX(6).AlignItems(ui.Center).Shrink(0).Children(func() {
				a.tabListButton(c)
			})
		}
	})
}

// newTabButton 是标签栏里的“+”新建请求按钮：常态就有淡底和前景色图标（不再是不起眼的灰色小图标），
// 悬停时底色加深；点击在根目录新建一个请求并在新标签中打开（与 ⌘N 一致）。
func (a *app) newTabButton(c *ui.Context) {
	p := a.p
	bg, hv := p.hover, p.selected
	tip := a.t("新建请求", "New request")
	if a.button(c, btnOpts{w: 26, h: 22, radius: radiusBtn - 1, tip: tip, bg: &bg, hoverBg: &hv, key: "ta:add"}, func(bool) {
		icon(c, "add", 15, p.fg)
	}) {
		a.later(func() { a.newRequest("") })
	}
}

// tabDrag 是拖动标签时携带的值。
type tabDrag struct{ t *tab }

// 请求标签按标题自适应宽度，限制在 [tabMinWidth, tabMaxWidth]；tabAccentWidth 是选中标签顶部强调线的粗细（与旧版 2px 的强调线一致）。
const (
	tabMinWidth    = 120
	tabMaxWidth    = 220
	tabAccentWidth = 2
	// 标签里标题之外固定占用的宽度：左右内边距 16 + 三处间距 12 + 未保存圆点位 6 + 关闭按钮 16，再留 4 的余量防止刚好被截断。
	tabChrome = 16 + 12 + 6 + 16 + 4
)

// tabLeadWidth 是标题前面的方法标签（或环境标签的地球图标）的宽度。
func (a *app) tabLeadWidth(c *ui.Context, t *tab) float32 {
	if t.env != nil {
		return 11
	}
	w, _ := c.MeasureText(0, ui.Span{Text: t.draft.Method, Font: monoFont, Size: 9, Weight: 700})
	return w
}

// tabWidthFor 计算标签宽度：方法标签 + 标题 + 固定占位，夹在 [tabMinWidth, tabMaxWidth] 之间。
// 不抖动的关键：
//   - 标题一律按选中态的字重（500）量宽，选中/未选中切换时宽度不变；
//   - 未保存圆点和关闭按钮的位置始终预留，修改请求或悬停时宽度不变；
//   - 只有标题、方法变化（改名、改方法）时才会改变宽度。
func (a *app) tabWidthFor(c *ui.Context, t *tab) float32 {
	lead := a.tabLeadWidth(c, t)
	title, _ := c.MeasureText(0, ui.Span{Text: t.title(), Size: fsV2xs, Weight: 500})
	w := min(max(lead+title+tabChrome, tabMinWidth), tabMaxWidth)
	return float32(math.Ceil(float64(w)))
}

// stripTopLine 画标签条顶部的 1px 横线（铺满整条，作为最底层）。
// 选中的标签从这一行开始，用自己的强调色上边框把线盖住，所以同一个位置永远只有一条线；
// 其余标签从线下面一行开始。标题栏本身不再画底线，避免与这条线叠成两条。
func stripTopLine(c *ui.Context, col ui.Color) {
	ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(1).Background(col).ZIndex(-1)
}

// tabItem 绘制一个请求标签：方法、标题、未保存圆点与悬停时出现的关闭按钮。
// 标签是方正的：选中的标签只在顶部画一条 2px 强调线，背景和下方内容区一致，看起来与内容区连成一体。
// 相邻标签之间只有一条 1px 竖分隔线。鼠标中键点击标签可关闭它。
func (a *app) tabItem(c *ui.Context, i int, t *tab) {
	p := a.p
	active := i == a.active && !a.showHome
	h := float32(sizeTabStrip - 1) // 未选中的标签在顶线下面
	if active {
		h = sizeTabStrip // 选中的标签从顶线所在行开始，用自己的上边框盖住它
	}
	// 宽度按标题自适应，但用固定字重量宽并预留圆点/关闭按钮的位置，所以切换、编辑、悬停时都不会抖动。
	t.shownW = a.tabWidthFor(c, t)
	cell := ui.Row(c.Key("tab:" + t.draft.ID)).Height(h).Width(t.shownW).Shrink(0).Drag(tabDrag{t: t}).
		ContextMenu(func(m *ui.Menu) { a.tabMenu(m, i) }).
		HandleInput(func(ev ui.InputEvent) bool {
			// 中键按下时接管这次点按，抬起时关闭（和浏览器标签页的习惯一致）。
			switch {
			case ev.Kind == ui.InputPointerDown && ev.Button == 2:
				return true
			case ev.Kind == ui.InputPointerUp && ev.Button == 2 && !ev.Canceled:
				a.later(func() { a.closeTab(a.tabIndex(t)) })
				return true
			}
			return false
		})
	if active && a.revealTab {
		// 只在切换的那一帧滚动一次，之后用户仍可自由滚动标签栏。
		a.revealTab = false
		cell = cell.ScrollIntoView()
	}
	// 拖动标签到另一个标签上：放下后占据它的位置；拖动经过时在左侧画一条插入指示线。
	if d, ok := ui.Drop[tabDrag](cell); ok {
		a.later(func() { a.moveTab(d.t, a.tabIndex(t)) })
	}
	dropHere := false
	if d, ok := ui.DragOver[tabDrag](cell); ok && d.t != t {
		dropHere = true
	}
	hover := cell.Hovered()
	// 标题可用宽度 = 标签宽 - 标题之外固定占用的部分 - 方法标签
	titleRoom := t.shownW - tabChrome - a.tabLeadWidth(c, t)
	cell = a.tipIfLong(c, cell, t.title(), fsV2xs, 500, titleRoom)
	switch {
	case active:
		cell = cell.Background(p.surface).BorderWidth(tabAccentWidth, 0, 0, 0).BorderColor(p.tabActiveBorder)
	case hover:
		cell = cell.Background(p.hover)
	}
	cell.Children(func() {
		if dropHere {
			ui.Box(c).Absolute().Top(0).Bottom(0).Left(0).Width(2).Background(p.accent).ZIndex(1)
		}
		ui.Column(c).Grow(1).Basis(0).FillHeight().Children(func() {
			ui.Row(c).Grow(1).Basis(0).FillWidth().PaddingX(8).Gap(4).AlignItems(ui.Center).Children(func() {
				if t.env != nil {
					icon(c, "globe", 11, p.accent).Shrink(0)
				} else {
					txtMono(c, methodAbbr(t.draft.Method), 9, p.method(t.draft.Method), 700).Shrink(0)
				}
				col, w := p.fg2, 0
				if active {
					col, w = p.fg, 500
				}
				txtE(c, t.title(), fsV2xs, col, w).Grow(1).Basis(0)
				// 未保存圆点：位置始终预留（不脏时透明），避免编辑时标签宽度变化。
				dot := ui.Box(c).Size(6, 6).Radius(3).Shrink(0)
				if t.dirty {
					dot.Background(p.accent) // Element 是节点句柄，原地生效，无需回写
				}
				closeCol := p.muted
				cb := ui.ButtonBase(c.Key("tc:"+t.draft.ID)).Size(16, 16).Radius(4).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).
					Tooltip(a.t("关闭 (⌘W)", "Close (⌘W)")).Label(a.t("关闭 (⌘W)", "Close (⌘W)"))
				if cb.Hovered() {
					cb = cb.Background(p.hover)
					closeCol = p.fg
				}
				cb.Children(func() {
					if hover || active {
						icon(c, "clear", 10, closeCol)
					}
				})
				if cb.Clicked() {
					a.later(func() { a.closeTab(i) })
				}
			})
		})
	})
	if cell.Clicked() && !active {
		a.later(func() {
			a.showHome = false
			a.activate(i)
		})
	}
	// 标签右侧的竖分隔线（选中标签只有顶部强调线，两侧照常画分隔线）。
	{
		ui.Box(c.Key("tabdiv:" + t.draft.ID)).Width(1).Height(sizeTabStrip - 1).Shrink(0).Background(p.tabDivider)
	}
}

// tabMenu 是标签的右键菜单：关闭、关闭其他、关闭全部。
func (a *app) tabMenu(m *ui.Menu, i int) {
	if i < 0 || i >= len(a.tabs) {
		return
	}
	t := a.tabs[i]
	if m.Item(a.t("保存", "Save")).Shortcut(ui.Cmd, ui.KeyS).Disabled(!t.dirty && !t.unsaved).Chosen() {
		a.later(func() { a.saveTab(t) })
	}
	if t.env == nil {
		id := t.draft.ID
		if m.Item(a.t("复制为 cURL", "Copy as cURL")).Shortcut(ui.Cmd|ui.Shift, ui.KeyC).Chosen() {
			a.later(func() { a.copyCurlOf(id) })
		}
		if m.Item(a.t("复制地址", "Copy URL")).Disabled(t.draft.URL == "").Chosen() {
			a.later(func() { a.copyText(t.draft.URL, a.t("已复制地址", "URL copied")) })
		}
		if m.Item(a.t("在侧栏中定位", "Reveal in sidebar")).Disabled(t.unsaved).Chosen() {
			a.later(func() { a.showHome = false; a.side.tab = 0; a.activate(a.tabIndex(t)) })
		}
	}
	m.Separator()
	if m.Item(a.t("关闭其他标签", "Close other tabs")).Disabled(len(a.tabs) <= 1).Chosen() {
		a.later(func() {
			var rest []*tab
			for _, x := range a.tabs {
				if x != t {
					rest = append(rest, x)
				}
			}
			a.closeTabs(rest)
		})
	}
	if m.Item(a.t("关闭右侧标签", "Close tabs to the right")).Disabled(i >= len(a.tabs)-1).Chosen() {
		a.later(func() {
			if k := a.tabIndex(t); k >= 0 {
				a.closeTabs(append([]*tab(nil), a.tabs[k+1:]...))
			}
		})
	}
	if m.Item(a.t("关闭", "Close")).Shortcut(ui.Cmd, ui.KeyW).Chosen() {
		a.later(func() { a.closeTab(a.tabIndex(t)) })
	}
	if m.Item(a.t("关闭全部", "Close all")).Chosen() {
		a.later(func() { a.closeTabs(append([]*tab(nil), a.tabs...)) })
	}
}

// tabListButton 是“显示所有标签”的下拉按钮。
func (a *app) tabListButton(c *ui.Context) {
	p := a.p
	ui.ButtonBase(c.Key("tablist")).Size(22, 22).Radius(radiusBtn - 1).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).
		Tooltip(a.t("显示所有标签", "Show all tabs")).
		PopoverMenuButton(func(m *ui.Menu) {
			for i, t := range a.tabs {
				i := i
				label := t.draft.Method + "  " + t.title()
				if t.env != nil {
					label = a.t("环境  ", "Env  ") + t.title()
				}
				if t.dirty {
					label += "  ●"
				}
				if m.Item(label).Checked(i == a.active).Chosen() {
					a.later(func() { a.activate(i) })
				}
			}
		}).
		Children(func() { icon(c, "arrowDown", 11, p.muted) })
}

// tabIndex 返回标签当前的下标，找不到（已关闭）返回 -1。
func (a *app) tabIndex(t *tab) int {
	for i, x := range a.tabs {
		if x == t {
			return i
		}
	}
	return -1
}
