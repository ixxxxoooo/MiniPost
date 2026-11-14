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
	ui.Column(c).Fill().Background(a.p.surface).Children(func() {
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
		split := ui.Split(c, &a.set.SidebarWidth, func() { a.sidebar(c) }, right).SplitReset(240).Fill()
		if split.Changed() {
			a.saveSettings()
		}
	})
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
