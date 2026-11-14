package gui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// emptyRecentMax 是欢迎页“最近打开”最多列出的条数。
const emptyRecentMax = 3

// emptyState 是没有打开任何请求时的欢迎页（设计稿）：应用图标与名称、四个快捷入口、最近打开、示例项目提示。
func (a *app) emptyState(c *ui.Context) {
	p := a.p
	ui.Scroll(c.Key("empty-scroll")).Grow(1).Basis(0).Background(p.surface).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Padding(48, 24).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Width(460).MaxWidth(460).Gap(0).Children(func() {
				ui.Column(c).FillWidth().AlignItems(ui.Center).Children(func() {
					if img := logoImage(p.dark); img != nil {
						ui.Image(c, img).Size(48, 48).Fit(ui.Contain)
					}
					ui.Box(c).Height(10)
					txt(c, "MiniPost", 16, p.fg, 600)
					ui.Box(c).Height(4)
					txt(c, a.t("一个简洁、快速的 API 调试工具", "A simple and fast API debugging tool"), fsBase, p.muted, 0)
				})
				ui.Box(c).Height(20)
				// 快捷入口：一张圆角卡片，行与行之间用细线分隔
				ui.Column(c).FillWidth().Radius(radiusCard).Border(1, p.border).Clip().Children(func() {
					a.emptyAction(c, "add", a.t("新建请求", "New request"), "⌘ N", false, func() { a.newRequest("") })
					a.emptyAction(c, "fileImport", a.t("导入 Postman / OpenAPI / cURL", "Import Postman / OpenAPI / cURL"), "⌘ O", true, func() { a.importFromFile() })
					a.emptyAction(c, "commandLine", a.t("粘贴 cURL 命令", "Paste a cURL command"), "⌘ I", true, a.openCurlDialog)
					a.emptyAction(c, "search", a.t("快速打开请求、环境、命令", "Quick open requests, environments, commands"), "⌘ K", true, a.openQuick)
				})
				a.emptyRecent(c)
				ui.Box(c).Height(20)
				ui.Row(c).FillWidth().Padding(8, 12).Gap(8).Radius(radiusInput).Border(1, p.border).AlignItems(ui.Center).Children(func() {
					icon(c, "info", 12, p.muted)
					ui.Text(c, a.t("想先看看效果？在「文件 → 添加示例项目」中创建演示项目，覆盖各种方法、认证、响应格式和流式响应。",
						"Want a quick look? Use File → Add Sample Projects to create demo projects covering methods, auth, response formats and streaming.")).
						FontSize(fsSmall).TextColor(p.fg2).Grow(1).Basis(0)
				})
			})
		})
	})
}

// emptyAction 是欢迎页里的一行入口：左侧图标方块与说明，右侧快捷键。top 为真时顶部画分隔线。
func (a *app) emptyAction(c *ui.Context, iconName, label, keys string, top bool, fn func()) {
	p := a.p
	b := ui.ButtonBase(c.Key("empty:" + keys)).FillWidth().Height(36).PaddingX(10).Gap(10).AlignItems(ui.Center).Justify(ui.Start)
	if top {
		b = b.BorderWidth(1, 0, 0, 0).BorderColor(p.border)
	}
	if b.Hovered() {
		b = b.Background(p.hover)
	}
	b.Children(func() {
		ui.Row(c).Size(22, 22).Radius(6).Background(p.surface2).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
			icon(c, iconName, 12, p.fg2)
		})
		txt(c, label, fsBase, p.fg, 0).Grow(1)
		a.kbdHint(c, keys)
	})
	if b.Clicked() {
		a.later(fn)
	}
}

// emptyRecent 列出最近发送过的请求（取自历史、且请求仍存在），点击在标签里打开。
func (a *app) emptyRecent(c *ui.Context) {
	p := a.p
	var recent []historyItem
	for _, e := range a.history {
		if a.historyRequestID(e) == "" {
			continue
		}
		if recent = append(recent, e); len(recent) == emptyRecentMax {
			break
		}
	}
	if len(recent) == 0 {
		return
	}
	txt(c, a.t("最近打开", "Recent"), fsSmall, p.muted, 600).Margin(18, 0, 6, 2)
	for _, e := range recent {
		e := e
		name := firstNonEmpty(e.Name, e.URL)
		row := ui.ButtonBase(c.Key("recent:" + e.ID)).FillWidth().Height(28).PaddingX(8).Gap(8).Radius(radiusBtn).AlignItems(ui.Center).Justify(ui.Start)
		if row.Hovered() {
			row = row.Background(p.hover)
		}
		row.Children(func() {
			ui.Row(c).Width(sizeMethodCol).Shrink(0).Children(func() {
				txtMono(c, methodAbbr(e.Method), 9, p.method(e.Method), 700)
			})
			txtE(c, name, fsBase, p.fg, 0).Grow(1).Basis(0)
			where := a.relTime(e.Timestamp, time.Now())
			if pr := a.project(); pr != nil {
				where = pr.Name + " · " + where
			}
			txt(c, where, fsSmall, p.muted, 0).Shrink(0)
		})
		if row.Clicked() {
			a.later(func() { a.openFromHistory(e) })
		}
	}
}
