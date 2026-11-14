package gui

import (
	"github.com/egoist/mygo/ui"
)

// view 是窗口的根视图，每一帧由 MyGo 调用。
func (a *app) view(c *ui.Context) {
	a.sysLocale = c.Locale()
	a.applyTheme(c)
	if a.native && a.menuLocale != a.locale() {
		a.installMenu()
	}
	for _, m := range a.toastMsgs {
		c.Toast(m)
	}
	a.toastMsgs = nil

	if t := a.activeTab(); t != nil {
		t.refreshDirty() // 标签栏先于编辑区绘制，需要先算出本帧的未保存状态
	}
	ui.Column(c).Fill().Children(func() {
		if a.showHome || a.projectID == "" {
			a.homeView(c)
			return
		}
		a.workspace(c)
	})
	a.overlayViews(c)
	a.runDeferred()
}

// overlayViews 绘制所有弹层，放在主界面之后以显示在最上层。
func (a *app) overlayViews(c *ui.Context) {
	a.projectDialogView(c)
	a.curlDialogView(c)
	a.prefsView(c)
	a.importView(c)
	a.envDialogView(c)
	a.cookiesView(c)
	// 询问类弹层最后绘制，才能显示在其他对话框之上。
	a.promptView(c)
	a.confirmView(c)
	a.closeView(c)
}

func (a *app) workspace(c *ui.Context) {
	a.toolbar(c)
	ui.Divider(c)
	body := func() {
		if a.set.SidebarCollapsed {
			a.mainPane(c)
			return
		}
		split := ui.Split(c, &a.set.SidebarWidth, func() { a.sidebar(c) }, func() { a.mainPane(c) }).SplitReset(260).Grow(1)
		if split.Changed() {
			a.saveSettings()
		}
	}
	if a.set.ConsoleOpen {
		ui.SplitVerticalPercent(c, &a.consolePct, body, func() { a.consolePanel(c) }).Grow(1)
		return
	}
	body()
}

// selectIndex 是按下标选择的下拉框：labels 为选项文案，*idx 为当前选中项。
func selectIndex(c *ui.Context, idx *int, labels []string) ui.Element {
	cur := ""
	if *idx >= 0 && *idx < len(labels) {
		cur = labels[*idx]
	}
	el := ui.Select(c, &cur, labels)
	if el.Changed() {
		for i, l := range labels {
			if l == cur {
				*idx = i
				break
			}
		}
	}
	return el
}

func (a *app) toolbar(c *ui.Context) {
	ui.Toolbar(c, func() {
		if ui.Button(c, "☰").Tooltip(a.t("显示/隐藏侧栏 (⌘B)", "Toggle sidebar (⌘B)")).Clicked() {
			a.toggleSidebar()
		}
		name := a.t("选择项目", "Select project")
		if p := a.project(); p != nil {
			name = p.Name
		}
		ui.MenuButton(c, name, func(m *ui.Menu) { a.projectMenu(m) })
		ui.Spacer(c)

		labels := []string{a.t("无环境", "No environment")}
		cur := 0
		for i, e := range a.envs {
			labels = append(labels, e.Name)
			if e.ID == a.envID {
				cur = i + 1
			}
		}
		if selectIndex(c, &cur, labels).Changed() {
			if cur == 0 {
				a.setEnv("")
			} else {
				a.setEnv(a.envs[cur-1].ID)
			}
		}
		if ui.Button(c, a.t("环境…", "Envs…")).Tooltip(a.t("管理环境变量", "Manage environments")).Clicked() {
			a.later(a.openEnvManager)
		}
		if ui.Button(c, "Cookies").Clicked() {
			a.later(a.openCookies)
		}
		if ui.Button(c, a.t("控制台", "Console")).Clicked() {
			a.later(func() { a.setConsoleOpen(!a.set.ConsoleOpen) })
		}
		if ui.Button(c, a.t("设置", "Settings")).Tooltip("⌘,").Clicked() {
			a.later(a.openPrefs)
		}
	})
}

func (a *app) toggleSidebar() {
	a.set.SidebarCollapsed = !a.set.SidebarCollapsed
	a.saveSettings()
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

// mainPane 是右侧主区域：标签栏与当前请求的编辑器。
func (a *app) mainPane(c *ui.Context) {
	ui.Column(c).Grow(1).Children(func() {
		if len(a.tabs) == 0 {
			a.emptyState(c)
			return
		}
		a.tabBar(c)
		ui.Divider(c)
		if t := a.activeTab(); t != nil {
			a.requestView(c, t)
		}
	})
}

func (a *app) emptyState(c *ui.Context) {
	ui.Column(c).Grow(1).Gap(10).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Text(c, "MiniPost").FontSize(c.Theme().FontSize + 9).Bold()
		ui.Text(c, a.t("从左侧选择一个请求，或新建一个开始", "Pick a request on the left, or create one to start")).TextColor(c.Theme().TextMuted)
		if ui.PrimaryButton(c, a.t("新建请求", "New Request")).Clicked() {
			a.later(func() { a.newRequest("") })
		}
	})
}
