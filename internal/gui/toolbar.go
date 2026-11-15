package gui

import (
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
)

// titleBarLeftMin 是标题栏内容的最小左边距：红绿灯左边距 + 三颗灯的宽度（14×3 + 间距 6×2）+ 设计稿中的 12px 留白。
const titleBarLeftMin = trafficLightX + 54 + 12

// titleBar 绘制顶部自绘标题栏（设计稿「组件 · 标题栏」，高 sizeToolbar，含 1px 底线）：
// 红绿灯留白、侧栏开关、后退/前进/主页、项目选择器，右侧是环境胶囊、导入 cURL、明暗切换与设置。
// 红绿灯由系统绘制（窗口使用隐藏标题栏样式），这里只需让出位置并支持拖动窗口。
func (a *app) titleBar(c *ui.Context) {
	p := a.p
	left := max(c.TitleBar().Left, titleBarLeftMin)
	ui.Row(c).FillWidth().Height(sizeToolbar-1).Shrink(0).Background(p.toolbar).
		Padding(0, 12, 0, left).Gap(4).AlignItems(ui.Center).DragWindow().Children(func() {
		// 侧栏开关：侧栏展开时为高亮（“on”）态
		tip := a.t("收起侧边栏", "Collapse sidebar")
		if a.set.SidebarCollapsed {
			tip = a.t("展开侧边栏", "Expand sidebar")
		}
		a.titleIconButton(c, "tb-sidebar", "tbSidebar", tip+" (⌘B)", !a.set.SidebarCollapsed, false, a.toggleSidebar)

		a.workspaceHeader(c)

		ui.Spacer(c)

		a.updateChip(c)
		a.envPill(c)
		ui.Box(c).Width(1).Height(14).Background(p.btnBorder).Shrink(0).MarginX(6) // 设计稿的竖分隔线
		a.titleIconButton(c, "tb-import", "tbImport", a.t("导入 cURL", "Import cURL")+" (⌘I)", false, false, a.openCurlDialog)
		themeIcon, themeTip := "tbMoon", a.t("切换为深色主题", "Switch to dark theme")
		if p.dark {
			themeIcon, themeTip = "tbSun", a.t("切换为浅色主题", "Switch to light theme")
		}
		a.titleIconButton(c, "tb-theme", themeIcon, themeTip, false, false, a.toggleThemeMode)
		a.titleIconButton(c, "tb-settings", "tbSettings", a.t("设置", "Settings")+" (⌘,)", false, false, a.openPrefs)
	})
	if a.showHome || a.projectID == "" {
		hline(c, p.border) // 其余页面的这条线由标签条和侧栏标签条各自画在顶部
	}
}

// titleIconButton 是标题栏的 24×24 图标按钮（设计稿 .ib）：悬停浅底；on 为高亮态；disabled 时图标变淡且不可点。
// 点击后通过 a.later 在帧外执行 act。
func (a *app) titleIconButton(c *ui.Context, key, iconName, tip string, on, disabled bool, act func()) {
	p := a.p
	col := p.fg2
	switch {
	case disabled:
		col = p.fg2.Alpha(0.4)
	case on:
		col = p.fg
	}
	if a.button(c, btnOpts{w: sizeBtn, h: sizeBtn, radius: radiusBtn - 1, tip: tip, disabled: disabled, active: on, key: key}, func(bool) {
		icon(c, iconName, 14, col)
	}) {
		a.later(act)
	}
}

// toggleThemeMode 在亮色与深色之间切换并保存（跟随系统时按当前显示的明暗反向切换）。
func (a *app) toggleThemeMode() {
	if a.p.dark {
		a.set.Theme = "light"
	} else {
		a.set.Theme = "dark"
	}
	a.saveSettings()
	logger.Info("已切换明暗主题", "theme", a.set.Theme)
}

// workspaceHeader 是标题栏中部：后退、前进、主页按钮与项目选择器。
func (a *app) workspaceHeader(c *ui.Context) {
	a.titleIconButton(c, "nav-back", "tbBack", a.t("后退", "Back"), false, !a.canNav(-1), func() { a.navigate(-1) })
	a.titleIconButton(c, "nav-fwd", "tbForward", a.t("前进", "Forward"), false, !a.canNav(1), func() { a.navigate(1) })
	homeTip := a.t("打开主页", "Open home")
	if a.showHome {
		homeTip = a.t("关闭主页", "Close home")
	}
	a.titleIconButton(c, "nav-home", "tbHome", homeTip, a.showHome, false, a.toggleHome)
	a.projectButton(c)
}

// projectButton 是项目选择器按钮（设计稿 .proj）：项目主题色圆点 + 项目名 + 下箭头，
// 点击弹出项目下拉面板（搜索/新建、切换、导出、删除）。
func (a *app) projectButton(c *ui.Context) {
	p := a.p
	name := a.t("选择项目", "Select project")
	dot := p.accent
	if pr := a.project(); pr != nil {
		name = pr.Name
		if pr.ThemeColor != "" {
			dot = ui.Hex(pr.ThemeColor)
		}
	}
	// 按钮宽度随名称增长，限制在 92~180
	w := min(float32(180), max(float32(92), float32(54+11*utf8.RuneCountInString(name))))
	btn := ui.ButtonBase(c.Key("project-btn")).Width(w).Radius(radiusBtn - 1).Height(sizeBtn).PaddingStart(8).PaddingEnd(6).Gap(7).AlignItems(ui.Center).Shrink(0).
		Tooltip(a.t("切换项目", "Switch project")).Label(a.t("切换项目", "Switch project"))
	if btn.Hovered() || a.projMenu.open {
		btn = btn.Background(p.hover)
	}
	btn.Children(func() {
		ui.Box(c).Size(8, 8).Radius(4).Background(dot).Shrink(0)
		txtE(c, name, fsBase, p.fg, 600).Grow(1).Basis(0)
		arrow := icon(c, "tbChevDown", 12, p.muted)
		if a.projMenu.open {
			arrow.Rotate(180) // 面板展开时箭头翻转
		}
	})
	if btn.Clicked() {
		pm := &a.projMenu
		pm.open = !pm.open
		pm.query = ""
	}
	if a.projMenu.open {
		a.projectDropdown(c, btn, w)
	}
}

// envPill 是标题栏右侧的环境选择按钮：与左侧项目选择器同一风格（无边框、悬停浅底），
// 有环境时绿点 + 环境名，无环境时灰点 + “无环境”，点击弹出环境列表，末尾可进入环境管理。
func (a *app) envPill(c *ui.Context) {
	p := a.p
	dot := p.muted
	if a.env() != nil {
		dot = p.success
	}
	btn := ui.ButtonBase(c.Key("envsel")).Height(sizeBtn).Radius(radiusBtn - 1).PaddingStart(8).PaddingEnd(6).Gap(7).AlignItems(ui.Center).Shrink(0).
		Tooltip(a.t("切换环境", "Switch environment")).Label(a.t("切换环境", "Switch environment")).
		PopoverMenuButton(func(m *ui.Menu) { a.envMenu(m) })
	if btn.Hovered() {
		btn = btn.Background(p.hover)
	}
	btn.Children(func() {
		ui.Box(c).Size(7, 7).Radius(3.5).Background(dot).Shrink(0)
		txtE(c, a.envLabel(), fsBase, p.fg, 500).MaxWidth(140)
		icon(c, "tbChevDown", 12, p.muted)
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
