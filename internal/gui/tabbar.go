package gui

import (
	"github.com/egoist/mygo/ui"
)

// tabBar 绘制请求标签栏：方法、名称、未保存标记与关闭按钮。
func (a *app) tabBar(c *ui.Context) {
	th := c.Theme()
	ui.ScrollHorizontal(c).Children(func() {
		ui.Row(c).Gap(2).Padding(4, 8).Children(func() {
			for i, t := range a.tabs {
				i, t := i, t
				row := ui.Row(c.Key(t.draft.ID)).Gap(6).AlignItems(ui.Center).PaddingX(8).Radius(6)
				if i == a.active {
					row = row.Background(th.Surface)
				}
				row.Children(func() {
					ui.Text(c, shortMethod(t.draft.Method)).Bold().FontSize(th.FontSize - 3).TextColor(methodColor(t.draft.Method, th.Dark))
					label := t.title()
					if t.dirty {
						label += " ●"
					}
					ui.Text(c, label).SingleLine().MaxWidth(180)
					if ui.Button(c, "×").Tooltip(a.t("关闭 (⌘W)", "Close (⌘W)")).Clicked() {
						a.later(func() { a.closeTab(i) })
					}
				})
				if row.Clicked() && i != a.active {
					a.activate(i)
				}
			}
		})
	})
}
