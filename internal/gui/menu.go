package gui

import (
	"github.com/egoist/mygo"

	"minipost/internal/pkg/logger"
)

// installMenu 按当前界面语言构建并设置应用菜单；语言变化时会重新安装。
func (a *app) installMenu() {
	a.menuLocale = a.locale()
	// act 包装菜单动作：只在已进入项目工作区时执行，并在执行后让窗口重绘。
	act := func(needProject bool, fn func()) func(*mygo.MenuItem, *mygo.Window) {
		return func(*mygo.MenuItem, *mygo.Window) {
			if needProject && (a.projectID == "" || a.showHome) {
				return
			}
			fn()
			if a.win != nil {
				a.win.Invalidate()
			}
		}
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: a.t("文件", "File"), Submenu: []*mygo.MenuItem{
			{Label: a.t("新建请求", "New Request"), Accelerator: "CmdOrCtrl+N", Click: act(true, func() { a.newRequest("") })},
			{Label: a.t("新建文件夹", "New Folder"), Accelerator: "CmdOrCtrl+Shift+N", Click: act(true, func() { a.newFolder("") })},
			{Label: a.t("导入 cURL…", "Import cURL…"), Accelerator: "CmdOrCtrl+I", Click: act(true, a.openCurlDialog)},
			{Label: a.t("导入集合…", "Import Collection…"), Accelerator: "CmdOrCtrl+O", Click: act(true, a.importFromFile)},
			{Label: a.t("从 URL 导入…", "Import from URL…"), Click: act(true, a.importFromURL)},
			{Label: a.t("导出项目…", "Export Project…"), Accelerator: "CmdOrCtrl+Shift+E", Click: act(true, a.exportProject)},
			{Label: a.t("添加示例项目", "Add Sample Projects"), Click: act(false, a.installSamples)},
			mygo.Separator(),
			{Label: a.t("保存", "Save"), Accelerator: "CmdOrCtrl+S", Click: act(true, func() {
				if t := a.activeTab(); t != nil {
					a.saveTab(t)
				}
			})},
			{Label: a.t("关闭标签", "Close Tab"), Accelerator: "CmdOrCtrl+W", Click: act(true, func() { a.closeTab(a.active) })},
		}},
		{Label: a.t("请求", "Request"), Submenu: []*mygo.MenuItem{
			{Label: a.t("发送", "Send"), Accelerator: "CmdOrCtrl+Enter", Click: act(true, func() {
				if t := a.activeTab(); t != nil && t.env == nil {
					a.sendTab(t)
				}
			})},
			{Label: a.t("复制为 cURL", "Copy as cURL"), Accelerator: "CmdOrCtrl+Shift+C", Click: act(true, func() {
				if t := a.activeTab(); t != nil && t.env == nil {
					a.copyCurl(t)
				}
			})},
		}},
		{Role: mygo.RoleEditMenu},
		{Label: a.t("视图", "View"), Submenu: []*mygo.MenuItem{
			{Label: a.t("显示/隐藏侧栏", "Toggle Sidebar"), Accelerator: "CmdOrCtrl+B", Click: act(true, a.toggleSidebar)},
			{Label: a.t("显示/隐藏控制台", "Toggle Console"), Accelerator: "CmdOrCtrl+J", Click: act(true, func() { a.setConsoleOpen(!a.set.ConsoleOpen) })},
			mygo.Separator(),
			{Label: a.t("环境管理…", "Environments…"), Accelerator: "CmdOrCtrl+Alt+E", Click: act(true, a.openEnvManager)},
			{Label: "Cookies…", Accelerator: "CmdOrCtrl+Shift+K", Click: act(true, a.openCookies)},
			{Label: a.t("设置…", "Settings…"), Accelerator: "CmdOrCtrl+,", Click: act(false, a.openPrefs)},
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Role: mygo.RoleWindowMenu},
	})
	mygo.App.SetMenu(menu)
	logger.Debug("应用菜单已安装", "locale", a.menuLocale)
}
