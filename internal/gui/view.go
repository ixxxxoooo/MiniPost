package gui

import (
	"github.com/egoist/mygo/ui"
)

// view 是窗口的根视图，每一帧由 MyGo 调用。
func (a *app) view(c *ui.Context) {
	a.frameNo++
	a.sysLocale = c.Locale()
	a.applyTheme(c)
	if a.native && a.menuLocale != a.locale() {
		a.installMenu()
	}
	a.showToasts(c)
	a.handleShortcuts(c)

	if t := a.activeTab(); t != nil {
		t.refreshDirty() // 标签栏先于编辑区绘制，需要先算出本帧的未保存状态
	}
	winW, winH := c.Size()
	// 根容器是窗口的 chrome 底色，上面先铺极光（绝对定位，不占布局），再放标题栏与工作区。
	ui.Column(c).Fill().Background(a.p.toolbar).Clip().HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputScroll {
			a.noteScroll() // 滚动条自动隐藏：滚动后短暂显示
		}
		return false
	}).Children(func() {
		a.aurora(c, winW, winH)
		// 内容层也用绝对定位铺满，这样按构建顺序画在极光之上（绝对定位的元素总是晚于文档流内的元素绘制）。
		ui.Column(c).Absolute().Top(0).Bottom(0).Left(0).Right(0).Children(func() {
			a.workspace(c)
		})
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
	a.quickView(c)
	// 询问类弹层最后绘制，才能显示在其他对话框之上。
	a.promptView(c)
	a.confirmView(c)
	a.closeView(c)
}

// workspace 是项目工作区（设计稿方案 C「流光」）：透明标题栏在上；下面是半透明圆角侧栏，
// 右边是浮起的圆角内容卡片（标签栏 + 内容 + 控制台 + 状态栏）。主页也在卡片里，侧栏始终显示。
func (a *app) workspace(c *ui.Context) {
	a.titleBar(c)
	card := func() {
		// 内容卡片：圆角 16、1px 描边加柔和投影，与窗口边缘和侧栏各留 shellGap。
		a.p.shadowMd(ui.Column(c).Grow(1).Basis(0).Margin(0, shellGap, shellGap, 0).Radius(radiusLg).Clip().Background(a.p.surface)).Children(func() {
			if a.showHome || a.projectID == "" {
				a.homeView(c)
			} else {
				a.mainPane(c)
			}
			a.bottomArea(c)
		})
	}
	ui.Row(c).Grow(1).Basis(0).AlignItems(ui.Stretch).Children(func() {
		if a.set.SidebarCollapsed {
			ui.Row(c).Grow(1).Basis(0).PaddingStart(shellGap).AlignItems(ui.Stretch).Children(card)
			return
		}
		// 窗口较窄时侧栏不能把右侧请求区挤得太窄：显示宽度按窗口宽度夹紧，
		// 用户设定的宽度原样保留，窗口变宽后自动恢复。
		winW, _ := c.Size()
		lo, hi := sidebarLimits(winW)
		sw := min(max(a.set.SidebarWidth, lo), hi) - 1
		a.side.shown = sw + 1
		// 分隔线只在悬停/拖动时才显示强调色：用边框透明的主题包住分栏，两个窗格里再换回正常主题。
		// Themed 会按主题背景色铺底，这里两个主题都把背景设为透明，才能露出窗口的极光。
		quiet, inner := *a.th, *a.th
		quiet.Border, quiet.Background = ui.Transparent, ui.Transparent
		inner.Background = ui.Transparent
		var split ui.Element
		ui.Themed(c, &quiet).Fill().Children(func() {
			split = ui.Split(c, &sw, func() {
				ui.Themed(c, &inner).Fill().Children(func() { a.sidebarFrame(c) })
			}, func() {
				ui.Themed(c, &inner).Fill().Children(func() {
					ui.Row(c).Fill().PaddingStart(shellGap).AlignItems(ui.Stretch).Children(card)
				})
			}).SplitReset(231).Fill()
		})
		if split.Changed() {
			a.set.SidebarWidth = min(max(sw+1, lo), hi)
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

// 侧栏宽度范围与请求区的最小宽度（窗口最小 900 宽时，侧栏最多约 340）。
const (
	sidebarMinWidth = 180
	sidebarMaxWidth = 500
	mainMinWidth    = 560
)

// sidebarLimits 返回在宽度为 winW 的窗口里侧栏宽度的允许范围：保证右侧请求区至少 mainMinWidth。
func sidebarLimits(winW float32) (lo, hi float32) {
	lo = sidebarMinWidth
	hi = min(sidebarMaxWidth, winW-mainMinWidth)
	return lo, max(lo, hi)
}
