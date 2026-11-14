package gui

import (
	"github.com/egoist/mygo/ui"
)

// titleBar 绘制顶部自绘标题栏（37px）：红绿灯留白、侧栏开关、前进后退、主页、项目选择与右侧工具图标。
// 红绿灯由系统绘制（窗口使用隐藏标题栏样式），这里只需让出位置并支持拖动窗口。
func (a *app) titleBar(c *ui.Context) {
	p := a.p
	left := max(c.TitleBar().Left, 84) // 旧版红绿灯占 20~68px，侧栏开关从 84px 开始
	ui.Row(c).FillWidth().Height(sizeToolbar).Shrink(0).Background(p.toolbar).
		Padding(0, 10, 0, left).AlignItems(ui.Center).DragWindow().Children(func() {
		// 侧栏开关
		name, tip := "sidebarCollapse", a.t("收起侧边栏", "Collapse sidebar")
		if a.set.SidebarCollapsed {
			name, tip = "sidebarExpand", a.t("展开侧边栏", "Expand sidebar")
		}
		if a.iconButton(c, name, tip+" (⌘B)", 15, p.fg2) {
			a.later(a.toggleSidebar)
		}

		ui.Box(c).Width(4).Shrink(0) // ml-1
		a.workspaceHeader(c)

		ui.Spacer(c)

		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			if a.iconButton(c, "fileImport", a.t("导入 cURL", "Import cURL")+" (⌘I)", 14, p.fg2) {
				a.later(func() { a.overlays.curl = curlState{open: true} })
			}
			themeIcon, themeTip := "moon", a.t("切换为深色主题", "Switch to dark theme")
			if p.dark {
				themeIcon, themeTip = "sun", a.t("切换为浅色主题", "Switch to light theme")
			}
			if a.iconButton(c, themeIcon, themeTip, 14, p.fg2) {
				a.later(func() {
					if p.dark {
						a.set.Theme = "light"
					} else {
						a.set.Theme = "dark"
					}
					a.saveSettings()
				})
			}
			if a.iconButton(c, "settings", a.t("设置", "Settings")+" (⌘,)", 14, p.fg2) {
				a.later(a.openPrefs)
			}
		})
	})
	hline(c, p.border)
}

// workspaceHeader 是标题栏中部：后退、前进、主页按钮与项目选择器。
func (a *app) workspaceHeader(c *ui.Context) {
	p := a.p
	ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
		if a.button(c, btnOpts{w: sizeBtn, h: sizeBtn, radius: radiusBtn, tip: a.t("后退", "Back"), disabled: !a.canNav(-1), key: "nav-back"}, func(bool) {
			col := p.fg2
			if !a.canNav(-1) {
				col = p.muted
			}
			icon(c, "arrowLeft", 14, col)
		}) {
			a.later(func() { a.navigate(-1) })
		}
		if a.button(c, btnOpts{w: sizeBtn, h: sizeBtn, radius: radiusBtn, tip: a.t("前进", "Forward"), disabled: !a.canNav(1), key: "nav-fwd"}, func(bool) {
			col := p.fg2
			if !a.canNav(1) {
				col = p.muted
			}
			icon(c, "arrowRight", 14, col)
		}) {
			a.later(func() { a.navigate(1) })
		}
		homeTip := a.t("打开主页", "Open home")
		if a.showHome {
			homeTip = a.t("关闭主页", "Close home")
		}
		if a.button(c, btnOpts{w: sizeBtn, h: sizeBtn, radius: radiusBtn, tip: homeTip, key: "nav-home", active: a.showHome}, func(bool) {
			icon(c, "home", 15, p.fg2)
		}) {
			a.later(a.toggleHome)
		}
		a.projectButton(c)
	})
}

// projectButton 是项目选择器按钮，点击弹出项目菜单。
func (a *app) projectButton(c *ui.Context) {
	p := a.p
	name := a.t("选择项目", "Select project")
	if pr := a.project(); pr != nil {
		name = pr.Name
	}
	ui.ButtonBase(c.Key("project-btn")).Background(p.surface2).Radius(radiusInput).Height(28).PaddingX(8).Gap(6).AlignItems(ui.Center).
		PopoverMenuButton(func(m *ui.Menu) { a.projectMenu(m) }).
		Children(func() {
			icon(c, "folderShared", 15, p.fg2)
			txt(c, name, fsSmall, p.fg, 500)
			icon(c, "arrowDown", 11, p.muted)
		})
}

func (a *app) toggleSidebar() {
	a.set.SidebarCollapsed = !a.set.SidebarCollapsed
	a.saveSettings()
}

func (a *app) toggleHome() {
	if a.showHome {
		if a.projectID != "" {
			a.showHome = false
		}
		return
	}
	a.persistSession()
	a.showHome = true
}

func (a *app) projectMenu(m *ui.Menu) {
	for _, p := range a.projects {
		id := p.ID
		if m.Item(p.Name).Radio(id == a.projectID).Chosen() {
			a.later(func() { a.selectProject(id) })
		}
	}
	m.Separator()
	if m.Item(a.t("项目首页", "All Projects")).Chosen() {
		a.later(func() {
			a.persistSession()
			a.showHome = true
		})
	}
}
