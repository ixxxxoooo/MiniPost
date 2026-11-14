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

// bottomBar 绘制底栏：左侧 Console 与 Cookies，右侧布局切换与侧栏开关。
func (a *app) bottomBar(c *ui.Context) {
	p := a.p
	hline(c, p.border)
	ui.Row(c).FillWidth().Height(sizeBottom - 1).Shrink(0).Background(p.bottom).PaddingX(12).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			if a.footLink(c, "commandLine", "Console", a.errorCount()) {
				a.later(func() { a.setConsoleOpen(!a.set.ConsoleOpen) })
			}
			label := "Cookies"
			if n := len(a.jar.All()); n > 0 {
				label = fmt.Sprintf("Cookies (%d)", n)
			}
			if a.footLink(c, "cookie", label, 0) {
				a.later(a.openCookies)
			}
			a.statusInfo(c)
		})
		ui.Spacer(c)
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			vertical := a.set.LayoutDirection != "horizontal"
			tip := a.t("切换到左右布局", "Switch to horizontal layout")
			if !vertical {
				tip = a.t("切换到上下布局", "Switch to vertical layout")
			}
			if a.button(c, btnOpts{w: 20, h: 20, radius: 6, active: true, tip: tip, key: "layout"}, func(bool) {
				a.layoutGlyph(c, p.fg, func() {
					if vertical {
						ui.Box(c).Width(1).Height(7).Background(p.fg)
					} else {
						ui.Box(c).Width(9).Height(1).Background(p.fg)
					}
				}, ui.Center)
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
			sideTip := a.t("收起侧边栏", "Collapse sidebar")
			if a.set.SidebarCollapsed {
				sideTip = a.t("展开侧边栏", "Expand sidebar")
			}
			if a.button(c, btnOpts{w: 20, h: 20, radius: 6, tip: sideTip, hoverBg: &p.btnBg, key: "sidebar-glyph"}, func(bool) {
				pos := ui.Start
				if a.set.SidebarCollapsed {
					pos = ui.End
				}
				a.layoutGlyph(c, p.fg2, func() {
					ui.Box(c).Width(3).Height(7).Radius(1).Background(p.fg2.Alpha(0.8))
				}, pos)
			}) {
				a.later(a.toggleSidebar)
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
	ui.Row(c.Key("foot:last")).Gap(6).AlignItems(ui.Center).Shrink(1).Tooltip(a.t("最近一次请求", "Last request")).Children(func() {
		txtMono(c, methodAbbr(last.method), fsSmall, p.method(last.method), 600)
		txtEMono(c, urlPathQuery(last.url), fsSmall, p.muted, 0).MaxWidth(280)
		txt(c, "→", fsSmall, p.muted, 0)
		txtMono(c, text, fsSmall, col, 0)
	})
}

// layoutGlyph 绘制底栏里 13×11 的小窗口示意图：圆角边框，内部放一条线或一个色块。
func (a *app) layoutGlyph(c *ui.Context, col ui.Color, inner func(), pos ui.Align) {
	ui.Row(c).Size(13, 11).Radius(2).Border(1, col).AlignItems(ui.Center).Justify(pos).Padding(0, 1).Children(inner)
}

// footLink 是底栏左侧的文字按钮；badge>0 时在右侧显示小圆角计数。
func (a *app) footLink(c *ui.Context, iconName, label string, badge int) bool {
	p := a.p
	return a.button(c, btnOpts{gap: 4, noHover: true, key: "foot:" + label}, func(hover bool) {
		col := p.fg2
		if hover {
			col = p.fg
		}
		icon(c, iconName, 11, col)
		txt(c, label, fsSmall, col, 0)
		if badge > 0 {
			ui.Row(c).Height(14).PaddingX(4).Radius(7).Background(p.surface).Border(1, p.border).AlignItems(ui.Center).Children(func() {
				txt(c, fmt.Sprint(badge), 9, p.muted, 500)
			})
		}
	})
}

// errorCount 返回控制台里失败请求的数量。
func (a *app) errorCount() int {
	n := 0
	for _, e := range a.con.entries {
		if e.err != "" {
			n++
		}
	}
	return n
}
