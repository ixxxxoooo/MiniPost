package gui

import (
	"github.com/egoist/mygo/ui"
)

// emptyState 是没有打开任何请求时的主区域：应用图标、名称与两个快捷入口。
func (a *app) emptyState(c *ui.Context) {
	p := a.p
	ui.Column(c).Grow(1).Basis(0).Background(p.surface).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Column(c).Width(300).Children(func() {
			ui.Column(c).FillWidth().AlignItems(ui.Center).Children(func() {
				if img := logoImage(); img != nil {
					ui.Image(c, img).Size(48, 48).Fit(ui.Contain)
				}
				ui.Box(c).Height(8)
				txt(c, "MiniPost", fsLarge, p.fg2, 0)
				ui.Box(c).Height(4)
				txt(c, a.t("一个简洁、快速的 API 调试工具", "A simple and fast API debugging tool"), fsSmall, p.muted, 0)
			})
			ui.Box(c).Height(16)
			a.emptyAction(c, a.t("新建请求", "Create new"), "N", func() { a.newRequest("") })
			ui.Box(c).Height(2)
			a.emptyAction(c, a.t("导入", "Import"), "O", func() { a.importFromFile() })
		})
	})
}

// emptyAction 是空状态里的一行入口：左侧说明，右侧快捷键方块。
func (a *app) emptyAction(c *ui.Context, label, key string, fn func()) {
	p := a.p
	if a.button(c, btnOpts{h: 32, radius: radiusBtn, padX: 8, hoverBg: &p.surface2, key: "empty:" + key}, func(bool) {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			txt(c, label, fsBase, p.muted, 0)
			ui.Spacer(c)
			ui.Row(c).Gap(4).Children(func() {
				a.kbd(c, "⌘")
				a.kbd(c, key)
			})
		})
	}) {
		a.later(fn)
	}
}
