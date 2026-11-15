package gui

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// tabBar 绘制请求标签栏（设计稿方案 C · v3「文件夹标签」）：高 sizeTabStrip=38，顶部留 6、两侧留 8，底边一条分隔线；
// 底色比内容区略深，选中的标签与内容区同色、底部压住分隔线，和下方地址栏连成一体。
// 左侧是各请求标签与“+”，右侧是标签列表。
func (a *app) tabBar(c *ui.Context) {
	p := a.p
	ui.Row(c).FillWidth().Height(sizeTabStrip).Shrink(0).Background(p.toolbar.Mix(p.surface2, 0.42)).Padding(6, 8, 0, 8).Gap(3).AlignItems(ui.End).
		BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
		// 容器向下伸出 1px（盖住标签栏的分隔线）、向左右各伸出 10px（容纳选中标签两侧的凹圆角），否则会被滚动区域裁掉
		ui.ScrollHorizontal(c).Grow(1).Basis(0).Margin(0, 0, -1, -tabFillet).Scrollbars(false).Children(func() {
			ui.Row(c).Height(tabHeight).Gap(tabGap).AlignItems(ui.End).Padding(0, 0, 0, tabFillet).Children(func() {
				w := a.tabCellWidth(c)
				for i, t := range a.tabs {
					a.tabItem(c, i, t, w)
				}
				ui.Row(c).Height(tabHeight).AlignItems(ui.Center).Children(func() { a.newTabButton(c) })
			})
		})
		if len(a.tabs) > 0 {
			ui.Row(c).Height(tabHeight).AlignItems(ui.Center).Children(func() { a.tabListButton(c) })
		}
	})
}

// newTabButton 是标签栏里的“+”新建请求按钮（22×22 的小图标按钮）：悬停浅底；
// 点击在根目录新建一个请求并在新标签中打开（与 ⌘N 一致）。
func (a *app) newTabButton(c *ui.Context) {
	p := a.p
	tip := a.t("新建请求", "New request")
	if a.button(c, btnOpts{w: 22, h: 22, radius: radiusSm, tip: tip, key: "ta:add"}, func(bool) {
		icon(c, "add", 15, p.fg2)
	}) {
		a.later(func() { a.newRequest("") })
	}
}

// tabDrag 是拖动标签时携带的值。
type tabDrag struct{ t *tab }

// 请求标签（设计稿 .tab）：基础宽 tabBaseWidth，标签多了按可用宽度平分，但不小于 tabMinWidth；
// 超出时标签栏横向滚动。tabGap 是标签间距。
const (
	tabBaseWidth = 176
	tabMinWidth  = 104
	tabMaxWidth  = tabBaseWidth
	tabGap       = 2
	tabHeight    = 32 // 选中标签高度；未选中的矮 2（见 tabItem），底边都落在分隔线上
	// 标签里标题之外固定占用的宽度：左内边距 11 + 右内边距 6 + 方法文字与间距 36 + 关闭按钮 18 + 间距 6。
	tabChrome = 11 + 6 + 36 + 18 + 6
)

// tabCellWidth 是每个标签的宽度：可用宽度（窗口宽 - 侧栏 - 外边距 - 内边距 - “+”与标签列表按钮）按标签数平分，夹在 [tabMinWidth, tabBaseWidth] 之间。
func (a *app) tabCellWidth(c *ui.Context) float32 {
	winW, _ := c.Size()
	avail := winW - shellGap - 16 - 22 - 22 - 12
	if !a.set.SidebarCollapsed {
		avail -= a.sidebarShown()
	}
	n := float32(max(len(a.tabs), 1))
	w := (avail - (n-1)*tabGap) / n
	return float32(math.Floor(float64(min(max(w, tabMinWidth), tabBaseWidth))))
}

// tabItem 绘制一个请求标签（设计稿 .tab，28 高、圆角 9）：方法徽章、标题、未保存圆点与悬停/选中时出现的关闭按钮。
// 选中的标签是浮起的内容底胶囊（细投影 + 描边），底部带 2px 强调线；鼠标中键点击标签可关闭它。
func (a *app) tabItem(c *ui.Context, i int, t *tab, w float32) {
	p := a.p
	active := i == a.active && !a.showHome
	t.shownW = w
	h := float32(tabHeight - 2)
	if active {
		h = tabHeight
	}
	cell := ui.Row(c.Key("tab:"+t.draft.ID)).Height(h).Width(w).Shrink(0).Radius(10, 10, 0, 0).Drag(tabDrag{t: t}).
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
	// 标题可用宽度 = 标签宽 - 标题之外固定占用的部分
	cell = a.tipIfLong(c, cell, t.title(), fsV2xs, 500, w-tabChrome)
	switch {
	case active:
		// 与内容区同色，压住标签栏的分隔线（-1）；整圈亮轮廓由 tabOutline 绘制
		cell = cell.Background(p.surface).Margin(0, 0, -1, 0).ZIndex(2)
	case hover:
		cell = cell.Background(p.surface.Alpha(0.55))
	}
	cell.Transition(transColors).Children(func() {
		if dropHere {
			ui.Box(c).Absolute().Top(2).Bottom(2).Left(-2).Width(2).Radius(1).Background(p.accent).ZIndex(1)
		}
		if active {
			// 轮廓（两侧凹圆角 + 统一线宽的整圈亮线），在标签之间滑动
			a.tabOutline(c, w)
		} else if !hover && i < len(a.tabs)-1 && !(i+1 == a.active && !a.showHome) {
			// 相邻未选中标签之间的发丝分隔线
			ui.Box(c).Absolute().Right(-1).Top(9).Bottom(9).Width(1).Background(p.border.Alpha(0.8))
		}
		ui.Row(c).Grow(1).Basis(0).FillWidth().Padding(0, 6, 0, 11).Gap(6).AlignItems(ui.Center).Children(func() {
			if t.env != nil {
				icon(c, "globe", 13, p.accent).Shrink(0)
			} else {
				a.methodText(c, t.draft.Method)
			}
			col, wt := p.muted, 500
			if active {
				col, wt = p.fg, 650
			} else if hover {
				col = p.fg
			}
			txtE(c, t.title(), fsV2xs, col, wt).Grow(1).Basis(0)
			// 未保存圆点（7px 强调色）：悬停或选中时让位给关闭按钮。
			if t.dirty && !hover && !active {
				ui.Box(c).Size(7, 7).Radius(3.5).Background(p.accent).Margin(0, 5, 0, 5).Shrink(0)
				return
			}
			closeCol := p.muted
			cb := ui.ButtonBase(c.Key("tc:"+t.draft.ID)).Size(18, 18).Radius(5).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).
				Tooltip(a.t("关闭 (⌘W)", "Close (⌘W)")).Label(a.t("关闭 (⌘W)", "Close (⌘W)"))
			if cb.Hovered() {
				cb = cb.Background(p.press)
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
	if cell.Clicked() && !active {
		a.later(func() {
			a.showHome = false
			a.activate(i)
		})
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

// tabListButton 是“显示所有标签”的下拉按钮：点击展开 tabListDropdown。
func (a *app) tabListButton(c *ui.Context) {
	p := a.p
	btn := ui.ButtonBase(c.Key("tablist")).Size(22, 22).Radius(radiusSm).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).
		Tooltip(a.t("显示所有标签", "Show all tabs")).Label(a.t("显示所有标签", "Show all tabs"))
	if btn.Hovered() || a.tabListOpen {
		btn = btn.Background(p.hover)
	}
	btn.Children(func() { icon(c, "tbChevDown", 12, p.fg2) })
	if btn.Clicked() {
		a.tabListOpen = !a.tabListOpen
		a.tabListReveal = a.tabListOpen
	}
	if a.tabListOpen {
		a.tabListDropdown(c, btn)
	}
}

// tabListPanelW 是标签列表面板宽度（设计稿 .pop[type=tabsAll] 的 320）；tabListMaxH 是列表区最大高度，超出滚动。
const (
	tabListPanelW = 320
	tabListMaxH   = 320
)

// tabListDropdown 是标签列表面板（设计稿 .pop[type=tabsAll]）：顶部“打开的标签 · N”，每行是方法徽标（环境标签是地球图标）、
// 标题、所在文件夹（弱色）、未保存圆点，当前标签整行高亮并带勾选。右对齐在按钮下方，带 popIn 与逐条 itemIn。
func (a *app) tabListDropdown(c *ui.Context, anchor ui.Element) {
	p := a.p
	open := true
	ui.PopoverBase(c, anchor, &open, func(panel ui.Element) {
		a.popPanel(panel.AttachTo(anchor, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(6, 0, 0, 0), tabListPanelW).Clip()
		ui.Column(c).FillWidth().Children(func() {
			ui.Text(c, a.t("打开的标签 · "+itoa(len(a.tabs)), "Open tabs · "+itoa(len(a.tabs)))).FontSize(fsSmall).TextColor(p.muted).Padding(4, 8, 6, 8)
			ui.Scroll(c.Key("tablist-sc")).FillWidth().MaxHeight(tabListMaxH).Scrollbars(a.scrollbarsVisible()).Children(func() {
				ui.Column(c).FillWidth().Children(func() {
					for i, t := range a.tabs {
						i, t := i, t
						row := a.popItem(c, "tl:"+t.draft.ID, i, i == a.active)
						if i == a.active && a.tabListReveal {
							row = row.ScrollIntoView()
						}
						row.Children(func() {
							if t.env != nil {
								ui.Row(c).Size(20, 20).Radius(6).Background(p.surface2).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Margin(0, 7, 0, 7).Children(func() {
									icon(c, "globe", 12, p.fg2)
								})
							} else {
								ui.Row(c).Width(34).Shrink(0).Children(func() { a.methodBadge(c, t.draft.Method, 0) })
							}
							txtE(c, t.title(), fsBase, p.fg, 0).Grow(1).Basis(0)
							txtE(c, a.tabFolderName(t), fsSmall-0.5, p.muted, 0).Shrink(0).MaxWidth(90)
							if t.dirty {
								ui.Box(c).Size(7, 7).Radius(3.5).Background(p.accent).Shrink(0)
							}
							if i == a.active {
								icon(c, "check", 13, p.accent).Shrink(0)
							}
						})
						if row.Clicked() {
							a.later(func() {
								a.tabListOpen = false
								a.showHome = false
								a.activate(i)
							})
						}
					}
				})
			})
		})
	})
	a.tabListReveal = false
	if !open {
		a.tabListOpen = false
	}
}

// tabFolderName 是标签所在文件夹的名称（列表里的弱色说明）：环境标签写“环境管理”，未保存草稿写“未保存”，根目录下的写“根目录”。
func (a *app) tabFolderName(t *tab) string {
	switch {
	case t.env != nil:
		return a.t("环境管理", "Environments")
	case t.unsaved:
		return a.t("未保存", "Unsaved")
	}
	if nd, ok := a.tree.nodes[t.draft.ID]; ok {
		if pn, ok := a.tree.nodes[nd.parent]; ok && pn.name != "" {
			return pn.name
		}
	}
	return a.t("根目录", "Root")
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

// methodText 是标签里的方法名：只有着色的等宽粗体文字，没有色块（比徽章更省地方、更安静），宽度固定 28 让标题对齐。
func (a *app) methodText(c *ui.Context, method string) {
	txtMono(c, methodAbbr(method), 9.5, a.p.method(method), 750).Width(28).Shrink(0)
}

// tabFillet 是选中标签两侧凹圆角的半径（设计稿 .tab.on::after 的 10px）。
const tabFillet = 10

// tabOutline 绘制选中标签的轮廓（设计稿 .tab.on 的轮廓线 + ::after 凹圆角），随选中标签滑动：
//   - 左右各一个凹圆角，底色同内容区，使标签与下方内容连成一体（标签下方没有横线）；
//   - 整圈轮廓取“强调色混边框色”（比普通边框亮），线宽处处统一为 1px：从左凹圆角底部起，经左边、顶部、右边，到右凹圆角底部。
func (a *app) tabOutline(c *ui.Context, w float32) {
	p := a.p
	ln := p.border.Mix(p.accent, 0.58)
	fill := p.surface
	const f, k = float32(tabFillet), 0.5523 // k 为四分之一圆弧的贝塞尔控制点系数
	ui.Box(c.Key("tabline")).PassThrough().Absolute().Left(-f).Right(-f).Top(-1).Bottom(0).
		Transition(ui.ElementTransition{Duration: durSlide, Ease: easeSlide, Match: "tabline"}).
		Draw(func(pt *ui.Painter, r ui.Rect) {
			// 以标签自身左上角为原点：r 比标签向左右各宽 f、向上高 1。
			ox, oy := r.X+f, r.Y+1
			tw, h := r.W-2*f, r.H-1
			X := func(x float32) float32 { return ox + x }
			Y := func(y float32) float32 { return oy + y }
			// 凹圆角底色：弧线与下角之间的小块，与标签同色
			left := (&ui.Path{}).MoveTo(X(-f), Y(h)).LineTo(X(0), Y(h)).LineTo(X(0), Y(h-f)).
				CubeTo(X(0), Y(h-f+f*k), X(-f+f*k), Y(h), X(-f), Y(h)).Close()
			right := (&ui.Path{}).MoveTo(X(tw+f), Y(h)).LineTo(X(tw), Y(h)).LineTo(X(tw), Y(h-f)).
				CubeTo(X(tw), Y(h-f+f*k), X(tw+f-f*k), Y(h), X(tw+f), Y(h)).Close()
			pt.FillPath(left, fill)
			pt.FillPath(right, fill)
			// 整圈轮廓（外缘偏出 0.5px，线宽 1，统一不变）
			o := float32(0.5)
			ring := (&ui.Path{}).MoveTo(X(-f), Y(h)).
				CubeTo(X(-f+f*k), Y(h), X(-o), Y(h-f+f*k), X(-o), Y(h-f)).
				LineTo(X(-o), Y(f)).
				CubeTo(X(-o), Y(f-f*k), X(f-f*k), Y(-o), X(f), Y(-o)).
				LineTo(X(tw-f), Y(-o)).
				CubeTo(X(tw-f+f*k), Y(-o), X(tw+o), Y(f-f*k), X(tw+o), Y(f)).
				LineTo(X(tw+o), Y(h-f)).
				CubeTo(X(tw+o), Y(h-f+f*k), X(tw+f-f*k), Y(h), X(tw+f), Y(h))
			pt.StrokePath(ring, 1, ln)
		})
}
