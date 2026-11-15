package gui

import (
	"github.com/egoist/mygo/ui"

	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// titleBarLeftMin 是标题栏内容的最小左边距：红绿灯左边距 + 三颗灯的宽度（14×3 + 间距 6×2）+ 设计稿中的 12px 留白。
const titleBarLeftMin = trafficLightX + 54 + 12

// titleBar 绘制顶部自绘标题栏（设计稿方案 C「流光」：高 sizeToolbar、背景透明，露出窗口的极光）：
// 左侧是红绿灯留白、侧栏开关、后退/前进/主页；正中是悬浮胶囊（项目 | 环境 | 搜索）；右侧是更新提示、明暗切换与设置。
// 红绿灯由系统绘制（窗口使用隐藏标题栏样式），这里只需让出位置并支持拖动窗口。
func (a *app) titleBar(c *ui.Context) {
	left := max(c.TitleBar().Left, titleBarLeftMin)
	ui.Row(c).FillWidth().Height(sizeToolbar).Shrink(0).
		Padding(0, 10, 0, left).Gap(4).AlignItems(ui.Center).DragWindow().Children(func() {
		// 侧栏开关：侧栏展开时为高亮（“on”）态
		tip := a.t("收起侧边栏", "Collapse sidebar")
		if a.set.SidebarCollapsed {
			tip = a.t("展开侧边栏", "Expand sidebar")
		}
		a.ib(c, "tb-sidebar", "tbSidebar", tip+" (⌘B)", !a.set.SidebarCollapsed, false, a.toggleSidebar)
		a.navButtons(c)

		a.island(c)

		ui.Spacer(c)

		a.updateChip(c)
		themeIcon, themeTip := "tbMoon", a.t("切换为深色主题", "Switch to dark theme")
		if a.p.dark {
			themeIcon, themeTip = "tbSun", a.t("切换为浅色主题", "Switch to light theme")
		}
		a.ib(c, "tb-theme", themeIcon, themeTip, false, false, a.toggleThemeMode)
		a.ib(c, "tb-settings", "tbSettings", a.t("设置", "Settings")+" (⌘,)", false, false, a.openPrefs)
	})
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

// navButtons 是标题栏左侧的后退、前进与主页按钮。
func (a *app) navButtons(c *ui.Context) {
	a.ib(c, "nav-back", "tbBack", a.t("后退", "Back"), false, !a.canNav(-1), func() { a.navigate(-1) })
	a.ib(c, "nav-fwd", "tbForward", a.t("前进", "Forward"), false, !a.canNav(1), func() { a.navigate(1) })
	homeTip := a.t("打开主页", "Open home")
	if a.showHome {
		homeTip = a.t("关闭主页", "Close home")
	}
	a.ib(c, "nav-home", "tbHome", homeTip, a.showHome, false, a.toggleHome)
}

// islandChip 是悬浮胶囊里的一枚按钮（设计稿 .island .chip）：26px 高的全圆角，无边框，悬停浅底。
func islandChip(c *ui.Context, key string, open bool, p *palette) ui.Element {
	btn := ui.ButtonBase(c.Key(key)).Height(sizeIsland - 6).Radius((sizeIsland - 6) / 2).PaddingX(9).Gap(6).AlignItems(ui.Center).Shrink(0)
	if btn.Hovered() || open {
		btn = btn.Background(p.hover)
	}
	return btn
}

// projectButton 是项目选择器（设计稿 .island 里的项目 chip）：项目主题色首字头像 + 项目名 + 下箭头，
// 点击弹出项目下拉面板（搜索/新建、切换、导出、删除）。
func (a *app) projectButton(c *ui.Context) {
	p := a.p
	name := a.t("选择项目", "Select project")
	pr := model.Project{Name: name}
	if cur := a.project(); cur != nil {
		pr = *cur
		name = cur.Name
	}
	btn := islandChip(c, "project-btn", a.projMenu.open, p).Tooltip(a.t("切换项目", "Switch project")).Label(a.t("切换项目", "Switch project"))
	btn.Children(func() {
		a.projectAvatar(c, pr, 18, 10)
		txtE(c, name, fsBase, p.fg, 600).MaxWidth(160)
		arrow := icon(c, "tbChevDown", 11, p.muted)
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
		a.projectDropdown(c, btn, 0)
	}
}

// envPill 是胶囊里的环境选择按钮（设计稿 envChip）：地球图标 + 环境名 + 状态点（有环境为绿、无环境为灰）+ 下箭头，
// 点击弹出环境列表，末尾可进入环境管理。
func (a *app) envPill(c *ui.Context) {
	p := a.p
	dot := p.btnBorder
	if a.env() != nil {
		dot = p.success
	}
	btn := islandChip(c, "envsel", false, p).
		Tooltip(a.t("切换环境", "Switch environment")).Label(a.t("切换环境", "Switch environment")).
		PopoverMenuButton(func(m *ui.Menu) { a.envMenu(m) })
	btn.Children(func() {
		icon(c, "globe", 13, p.fg)
		txtE(c, a.envLabel(), fsBase, p.fg, 500).MaxWidth(140)
		ui.Box(c).Size(8, 8).Radius(4).Background(dot).Shrink(0)
		icon(c, "tbChevDown", 11, p.muted)
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
