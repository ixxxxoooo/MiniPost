package gui

import (
	"github.com/egoist/mygo/ui"
)

// tabBar 绘制 28px 高的请求标签栏：左侧是各请求标签与“+”，右侧是标签列表与环境选择。
func (a *app) tabBar(c *ui.Context) {
	p := a.p
	ui.Row(c).FillWidth().Height(sizeTabStrip).Shrink(0).Background(p.tabStrip).AlignItems(ui.Stretch).Children(func() {
		if len(a.tabs) == 0 {
			// 无标签时“+”独占一格，右侧带分隔线
			ui.Row(c).Width(35).PaddingX(6).AlignItems(ui.Center).Children(func() {
				a.tabAction(c, "add", a.t("新建请求", "New request"), 13)
			})
			vline(c, p.tabDivider)
		}
		ui.ScrollHorizontal(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
			ui.Row(c).FillHeight().AlignItems(ui.End).Children(func() {
				for i, t := range a.tabs {
					a.tabItem(c, i, t)
				}
				if len(a.tabs) > 0 {
					ui.Row(c).Height(sizeTabStrip).MarginStart(4).AlignItems(ui.Center).Children(func() {
						a.tabAction(c, "add", a.t("新建请求", "New request"), 13)
					})
				}
			})
		})
		vline(c, p.tabDivider)
		ui.Row(c).PaddingX(6).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
			if len(a.tabs) > 0 {
				a.tabListButton(c)
				ui.Box(c).Width(1).Height(12).Background(p.tabDivider).MarginX(2)
			}
			a.envSelector(c)
		})
	})
}

// tabAction 是标签栏里 22×22 的图标按钮。
func (a *app) tabAction(c *ui.Context, name, tip string, size float32) bool {
	p := a.p
	hv := p.hover
	return a.button(c, btnOpts{w: 22, h: 22, radius: radiusBtn - 1, tip: tip, hoverBg: &hv, key: "ta:" + name + tip}, func(hover bool) {
		col := p.muted
		if hover {
			col = p.fg
		}
		icon(c, name, size, col)
	})
}

// tabBleed 是选中标签向下多出并被裁掉的高度。
const tabBleed = 8

// tabHeight 是标签本身的高度（标签栏顶部留 2px 空隙）。
const tabHeight = sizeTabStrip - 2

// tabItem 绘制一个请求标签：方法、标题、未保存圆点与悬停时出现的关闭按钮。
// 选中的标签只在左、上、右画一圈强调色边框，下边不画线，背景和下方内容区一致，看起来与内容区连成一体；
// 为此标签比外层多出一截并被裁掉，底部的圆角和边线就不会露出来。鼠标中键点击标签可关闭它。
func (a *app) tabItem(c *ui.Context, i int, t *tab) {
	p := a.p
	active := i == a.active && !a.showHome
	ui.Column(c.Key("tabw:" + t.draft.ID)).Height(tabHeight).Shrink(0).Clip().Children(func() {
		cell := ui.Row(c.Key("tab:" + t.draft.ID)).Height(tabHeight + tabBleed).MinWidth(148).MaxWidth(230).Shrink(0).
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
		hover := cell.Hovered()
		switch {
		case active:
			cell = cell.Background(p.surface).Radius(7).BorderWidth(1, 1, 0, 1).BorderColor(p.tabActiveBorder)
		case hover:
			cell = cell.Background(p.hover).Radius(5)
		}
		cell.Children(func() {
			ui.Column(c).Grow(1).Basis(0).Height(tabHeight).Children(func() {
				ui.Row(c).Grow(1).Basis(0).FillWidth().PaddingX(8).Gap(4).AlignItems(ui.Center).Children(func() {
					if t.env != nil {
						icon(c, "globe", 11, p.accent).Shrink(0)
					} else {
						txtMono(c, t.draft.Method, 9, p.method(t.draft.Method), 700).Shrink(0)
					}
					col, w := p.fg2, 0
					if active {
						col, w = p.fg, 500
					}
					txtE(c, t.title(), fsV2xs, col, w).Grow(1).Basis(0)
					if t.dirty {
						ui.Box(c).Size(6, 6).Radius(3).Background(p.accent).Shrink(0)
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
	})
	// 标签之间的竖分隔线：选中标签自带边框，它两侧不再画。
	if !active && !(i+1 == a.active && !a.showHome) {
		ui.Box(c.Key("tabdiv:" + t.draft.ID)).Width(1).Height(tabHeight).Shrink(0).Background(p.tabDivider)
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
		if m.Item(a.t("复制 cURL", "Copy as cURL")).Chosen() {
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
	if m.Item(a.t("关闭", "Close")).Shortcut(ui.Cmd, ui.KeyW).Chosen() {
		a.later(func() { a.closeTab(a.tabIndex(t)) })
	}
	if m.Item(a.t("关闭右侧", "Close to the right")).Disabled(i >= len(a.tabs)-1).Chosen() {
		a.later(func() {
			for j := len(a.tabs) - 1; j > a.tabIndex(t); j-- {
				a.closeTab(j)
			}
		})
	}
	if m.Item(a.t("关闭其他", "Close others")).Disabled(len(a.tabs) <= 1).Chosen() {
		a.later(func() {
			keep := a.tabs[i]
			for j := len(a.tabs) - 1; j >= 0; j-- {
				if a.tabs[j] != keep {
					a.closeTab(j)
				}
			}
		})
	}
	if m.Item(a.t("关闭全部", "Close all")).Chosen() {
		a.later(func() {
			for j := len(a.tabs) - 1; j >= 0; j-- {
				a.closeTab(j)
			}
		})
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

// envSelector 是标签栏右侧的环境选择：当前环境名，点击弹出环境列表。
func (a *app) envSelector(c *ui.Context) {
	p := a.p
	name := a.t("无环境", "No environment")
	col := p.muted
	if e := a.env(); e != nil {
		name = e.Name
		col = p.accent
	}
	btn := ui.ButtonBase(c.Key("envsel")).Height(22).Radius(6).PaddingX(8).Gap(4).AlignItems(ui.Center).Shrink(0).
		Tooltip(a.t("切换环境", "Switch environment")).
		PopoverMenuButton(func(m *ui.Menu) { a.envMenu(m) })
	if btn.Hovered() {
		btn = btn.Background(p.hover)
		if a.env() == nil {
			col = p.fg
		}
	}
	btn.Children(func() {
		icon(c, "globe", 11, col)
		txtE(c, name, fsTiny, col, 0).MaxWidth(100)
		icon(c, "arrowDown", 8, col)
	})
}

// envMenu 列出全部环境，当前环境带勾选，末尾可进入环境管理。
func (a *app) envMenu(m *ui.Menu) {
	if m.Item(a.t("无环境", "No environment")).Checked(a.envID == "").Chosen() {
		a.later(func() { a.setEnv("") })
	}
	for _, e := range a.envs {
		id := e.ID
		if m.Item(e.Name).Checked(a.envID == id).Chosen() {
			a.later(func() { a.setEnv(id) })
		}
	}
	m.Separator()
	if m.Item(a.t("管理环境…", "Manage environments…")).Chosen() {
		a.later(a.openEnvManager)
	}
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
