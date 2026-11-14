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
	a.showToasts(c)

	if t := a.activeTab(); t != nil {
		t.refreshDirty() // 标签栏先于编辑区绘制，需要先算出本帧的未保存状态
	}
	ui.Column(c).Fill().Background(a.p.surface).HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputScroll {
			a.noteScroll() // 滚动条自动隐藏：滚动后短暂显示
		}
		return false
	}).Children(func() {
		a.workspace(c)
	})
	a.overlayViews(c)
	a.runDeferred()
}

// overlayViews 绘制所有弹层，放在主界面之后以显示在最上层。
func (a *app) overlayViews(c *ui.Context) {
	a.prefsView(c)
	a.importView(c)
	a.curlView(c)
	a.cookiesView(c)
	// 询问类弹层最后绘制，才能显示在其他对话框之上。
	a.promptView(c)
	a.confirmView(c)
	a.closeView(c)
}

// workspace 是项目工作区：标题栏在上，左侧侧栏，右侧标签栏 + 内容 + 底栏。
func (a *app) workspace(c *ui.Context) {
	a.titleBar(c)
	ui.Row(c).Grow(1).Basis(0).AlignItems(ui.Stretch).Children(func() {
		if a.showHome || a.projectID == "" {
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				a.homeView(c)
				a.bottomArea(c)
			})
			return
		}
		right := func() {
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				a.mainPane(c)
				a.bottomArea(c)
			})
		}
		if a.set.SidebarCollapsed {
			right()
			return
		}
		// 旧版侧栏宽度（含 1px 右边框）为 SidebarWidth；这里的分隔线就是那条边框，所以窗格宽度减 1。
		sw := a.set.SidebarWidth - 1
		split := ui.Split(c, &sw, func() { a.sidebar(c) }, right).SplitReset(239).Fill()
		if split.Changed() {
			a.set.SidebarWidth = sw + 1
			a.saveSettings()
		}
	})
}

// mainPane 是右侧主区域：标签栏与当前请求的编辑器。
func (a *app) mainPane(c *ui.Context) {
	ui.Column(c).Grow(1).Basis(0).Children(func() {
		a.tabBar(c)
		if t := a.activeTab(); t != nil {
			if t.env != nil {
				a.envPage(c, t)
				return
			}
			a.requestView(c, t)
			return
		}
		a.emptyState(c)
	})
}
