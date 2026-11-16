package gui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// bottomArea 是主区域底部：控制台面板（打开时）与 25px 底栏。
func (a *app) bottomArea(c *ui.Context) {
	p := a.p
	ui.Column(c).FillWidth().Shrink(0).Children(func() {
		if a.set.ConsoleOpen {
			shown := a.consoleShownHeight(c)
			ui.Column(c).FillWidth().Height(shown).Shrink(0).Background(p.surface).Children(func() {
				hline(c, p.border)
				a.consolePanel(c)
				a.consoleGrip(c, shown)
			})
		}
		a.bottomBar(c)
	})
}

// bottomBar 绘制底栏：左侧 Console 与 Cookies，右侧布局切换。
func (a *app) bottomBar(c *ui.Context) {
	p := a.p
	// 方案 C：状态栏透明、无上边线，直接坐在内容卡片的底部。
	ui.Row(c).FillWidth().Height(sizeBottom).Shrink(0).PaddingX(8).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			if a.footLink(c, "terminal", a.t("控制台", "Console"), len(a.con.entries), a.set.ConsoleOpen) {
				a.later(func() { a.setConsoleOpen(!a.set.ConsoleOpen) })
			}
			label := "Cookies"
			if n := len(a.jar.All()); n > 0 {
				label = fmt.Sprintf("Cookies (%d)", n)
			}
			if a.footLink(c, "cookie", label, 0, false) {
				a.later(a.openCookies)
			}
			a.statusInfo(c)
		})
		ui.Spacer(c)
		ui.Row(c).Gap(2).AlignItems(ui.Center).Children(func() {
			vertical := a.set.LayoutDirection != "horizontal"
			tip := a.t("切换到左右布局", "Switch to horizontal layout")
			glyph := "dockBottom"
			if !vertical {
				tip = a.t("切换到上下布局", "Switch to vertical layout")
				glyph = "dockRight"
			}
			if a.button(c, btnOpts{h: 20, radius: 5, padX: 7, tip: tip, key: "layout"}, func(hover bool) {
				col := p.fg2
				if hover {
					col = p.fg
				}
				icon(c, glyph, 13, col)
			}) {
				a.later(func() {
					if vertical {
						a.set.LayoutDirection = "horizontal"
					} else {
						a.set.LayoutDirection = "vertical"
					}
					a.saveSettings()
				})
			}
		})
	})
}

// envLabel 是底栏里显示的当前环境名；未选择环境时提示“无环境”。
func (a *app) envLabel() string {
	if e := a.env(); e != nil {
		return e.Name
	}
	return a.t("无环境", "No environment")
}

// lastRequestSummary 返回最近一次请求的摘要（状态码与耗时、失败或进行中）与对应颜色；没有请求时 ok 为 false。
func (a *app) lastRequestSummary() (text string, col ui.Color, ok bool) {
	n := len(a.con.entries)
	if n == 0 {
		return "", ui.Color{}, false
	}
	e := a.con.entries[n-1]
	switch {
	case !e.done:
		return a.t("请求中…", "Sending…"), a.p.muted, true
	case e.err != "":
		return a.t("请求失败", "Request failed"), a.p.danger, true
	}
	return fmt.Sprintf("%d · %s", e.status, formatDuration(e.ms)), a.statusTextColor(e.status), true
}

// statusInfo 在底栏左侧 Console/Cookies 之后显示只读状态：当前环境、最近一次请求的状态与耗时。
// 点击环境名切到侧栏的“环境”页。
func (a *app) statusInfo(c *ui.Context) {
	p := a.p
	text, col, ok := a.lastRequestSummary()
	if !ok {
		return
	}
	// 设计稿：「POST /post → 200 · 773ms」，方法着色，路径灰色，状态着色
	ui.Box(c).Width(1).Height(12).Background(p.tabDivider).Shrink(0)
	last := a.con.entries[len(a.con.entries)-1]
	a.tip(c, ui.Row(c.Key("foot:last")).Gap(6).AlignItems(ui.Center).Shrink(1), a.t("最近一次请求", "Last request")).Children(func() {
		txtMono(c, methodAbbr(last.method), fsSmall, p.method(last.method), 600)
		txtEMono(c, urlPathQuery(last.url), fsSmall, p.muted, 0).MaxWidth(280)
		txt(c, "→", fsSmall, p.muted, 0)
		txtMono(c, text, fsSmall, col, 0)
	})
}

// footLink 是底栏左侧的文字按钮（设计稿 .sbtn）：20 高、圆角 5、内边距 7，悬停浅底；
// badge>0 时在右侧显示小圆角计数（.cnt），on 表示面板已打开（强调色淡底）。
func (a *app) footLink(c *ui.Context, iconName, label string, badge int, on bool) bool {
	p := a.p
	return a.button(c, btnOpts{h: 20, radius: 5, padX: 7, gap: 5, key: "foot:" + label, active: on, noHover: false}, func(hover bool) {
		col := p.fg2
		switch {
		case on:
			col = p.accent
		case hover:
			col = p.fg
		}
		icon(c, iconName, 12, col)
		txt(c, label, fsSmall, col, 0)
		if badge > 0 {
			ui.Row(c).Height(14).MinWidth(16).PaddingX(4).Radius(7).Background(p.surface2).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				txtMono(c, fmt.Sprint(badge), fsTiny, p.muted, 600)
			})
		}
	})
}
