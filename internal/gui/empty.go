package gui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// emptyRecentMax 是欢迎页“最近打开”最多列出的条数。
const emptyRecentMax = 3

// emptyState 是没有打开任何请求时的欢迎页（设计稿 .welcome）：整页垂直水平居中，背后有两团缓慢的强调色光斑；
// 400 宽的内容依次是 64px 应用图标（带强调色投影）、名称与说明、一张圆角卡片里的四个快捷入口、最近打开。
func (a *app) emptyState(c *ui.Context) {
	p := a.p
	ui.Scroll(c.Key("empty-scroll")).Grow(1).Basis(0).Background(p.surface).Scrollbars(a.scrollbarsVisible()).Children(func() {
		// 光斑：与设计稿 .welcome::before/::after 相同，位于内容之下（绝对定位先画，内容后画）
		glow := func(key string, x, y, r float32, col ui.Color) {
			ui.Box(c.Key(key)).Absolute().PassThrough().Left(x-r).Top(y-r).Size(2*r, 2*r).
				RadialGradient(ui.RadialGradient{Stops: []ui.GradientStop{{Color: col, At: 0}, {Color: col.Alpha(0), At: 1}}, Radius: r, Oklab: true})
		}
		cw, ch := c.Size()
		mid := (float32(cw) - a.sidebarShown()) / 2
		glow("welcome-g1", mid-70, float32(ch)*0.36, 230, p.acSoft2)
		glow("welcome-g2", mid+150, float32(ch)*0.46, 190, p.accent.Mix(pinkGlow, 0.5).Alpha(0.13))
		// 内容约 340 高，在标签栏之下的可用区域里垂直居中（设计稿 .welcome 用 place-items:center + 底部 40 留白）
		top := max(48, (float32(ch)-sizeToolbar-sizeTabStrip-sizeBottom-340)/2-20)
		ui.Column(c).FillWidth().Padding(top, 24, 40, 24).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Width(400).MaxWidth(400).Gap(12).AlignItems(ui.Center).Children(func() {
				if img := logoImage(p.dark); img != nil {
					ui.Row(c).Size(64, 64).Radius(16).Shadow(0, 10, 24, -4, p.accent.Alpha(0.34)).Children(func() {
						ui.Image(c, img).Size(64, 64).Fit(ui.Contain)
					})
				}
				txt(c, "MiniPost", 18, p.fg, 650).Margin(2, 0, 0, 0)
				txt(c, a.t("一个简洁、快速的 API 调试工具", "A simple and fast API debugging tool"), fsBase, p.muted, 0).Margin(-4, 0, 4, 0)
				// 快捷入口：一张圆角卡片里四行，行间有细线
				p.shadowMd(ui.Column(c).FillWidth().Radius(radiusLg).Clip().Background(p.glass.Alpha(0.94))).Children(func() {
					a.emptyAction(c, "plus", a.t("新建请求", "New request"), "⌘ N", false, func() { a.newRequest("") })
					a.emptyAction(c, "upload", a.t("导入 Postman / OpenAPI / cURL", "Import Postman / OpenAPI / cURL"), "⌘ O", true, func() { a.importFromFile() })
					a.emptyAction(c, "code", a.t("粘贴 cURL 命令", "Paste a cURL command"), "⌘ I", true, a.openCurlDialog)
					a.emptyAction(c, "search", a.t("快速打开请求、环境、命令", "Quick open requests, environments, commands"), "⌘ K", true, a.openQuick)
				})
				a.emptyRecent(c)
				ui.Text(c, a.t("想先看看效果？在「文件 → 添加示例项目」中创建演示项目，覆盖各种方法、认证、响应格式和流式响应。",
					"Want a quick look? Use File → Add Sample Projects to create demo projects covering methods, auth, response formats and streaming.")).
					FontSize(fsSmall).TextColor(p.muted).FillWidth().Margin(8, 0, 0, 0).TextAlign(ui.Center)
			})
		})
	})
}

// emptyAction 是欢迎页卡片里的一行入口（设计稿 .acts-l .a，34 高）：22px 图标块、说明（550 字重）、右侧快捷键；
// 悬停时整行淡强调色、图标块变强调色；sep 为真时上方有 1px 分隔线。
func (a *app) emptyAction(c *ui.Context, iconName, label, keys string, sep bool, fn func()) {
	p := a.p
	b := ui.ButtonBase(c.Key("empty:" + keys)).FillWidth().Height(34).PaddingX(12).Gap(10).AlignItems(ui.Center).Justify(ui.Start)
	if sep {
		b = b.BorderWidth(1, 0, 0, 0).BorderColor(p.border)
	}
	hover := b.Hovered()
	if hover {
		b = b.Gradient(p.acSoft, p.acSoft.Alpha(0), 90)
	}
	b.Children(func() {
		bg, fg := p.surface2, p.fg2
		if hover {
			bg, fg = p.acSoft, p.accent
		}
		ui.Row(c).Size(22, 22).Radius(6).Background(bg).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
			icon(c, iconName, 13, fg)
		})
		txt(c, label, fsBase, p.fg, 550).Grow(1)
		a.kbdHint(c, keys)
	})
	if b.Clicked() {
		a.later(fn)
	}
}

// emptyRecent 列出最近发送过的请求（取自历史、且请求仍存在），点击在标签里打开。
func (a *app) emptyRecent(c *ui.Context) {
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
	ui.Column(c.Key("recent")).FillWidth().Gap(2).Children(func() {
		a.emptyRecentRows(c, recent)
	})
}

// emptyRecentRows 画“最近打开”的标题与各行（设计稿 .recent：标题 11px 弱色 600，行高 26、圆角 8）。
func (a *app) emptyRecentRows(c *ui.Context, recent []historyItem) {
	p := a.p
	txt(c, a.t("最近打开", "Recent"), fsSmall, p.muted, 600).Margin(4, 0, 2, 4)
	for _, e := range recent {
		e := e
		name := firstNonEmpty(e.Name, e.URL)
		row := ui.ButtonBase(c.Key("recent:" + e.ID)).FillWidth().Height(26).PaddingX(8).Gap(8).Radius(radiusSm).AlignItems(ui.Center).Justify(ui.Start)
		if row.Hovered() {
			row = row.Background(p.hover)
		}
		row.Children(func() {
			ui.Row(c).Width(sizeMethodCol).Shrink(0).Children(func() {
				a.methodBadge(c, e.Method, 0)
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
