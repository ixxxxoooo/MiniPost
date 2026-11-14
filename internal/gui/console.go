package gui

import (
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// consoleEntry 是控制台里的一行请求日志。
type consoleEntry struct {
	at     time.Time
	method string
	url    string
	status int
	ms     float64
	err    string
	done   bool
	open   bool // 详情是否展开
}

const maxConsoleEntries = 500

// logRequest 记录一次请求开始。
func (a *app) logRequest(method, url string) {
	a.console = append(a.console, consoleEntry{at: time.Now(), method: method, url: url})
	if len(a.console) > maxConsoleEntries {
		a.console = a.console[len(a.console)-maxConsoleEntries:]
	}
}

// logResponse 把最近一条同请求的日志标记为完成。
func (a *app) logResponse(method, url string, status int, ms float64, errMsg string) {
	for i := len(a.console) - 1; i >= 0; i-- {
		e := &a.console[i]
		if !e.done && e.method == method && e.url == url {
			e.status, e.ms, e.err, e.done = status, ms, errMsg, true
			return
		}
	}
}

// consolePanel 绘制底部控制台：25px 标题行（Console、数量、清空、关闭）+ 请求日志列表。
func (a *app) consolePanel(c *ui.Context) {
	p := a.p
	ui.Column(c).Grow(1).Basis(0).Children(func() {
		head := ui.Row(c).FillWidth().Height(sizeBottom).Shrink(0).PaddingX(12).Background(p.surface2).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			BorderWidth(0, 0, 1, 0).BorderColor(p.border)
		head.Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				txt(c, "Console", fsSmall, p.fg, 500)
				if n := len(a.console); n > 0 {
					txt(c, itoa(n), fsTiny, p.muted, 0)
				}
			})
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				if a.button(c, btnOpts{h: 20, radius: 4, padX: 6, key: "console-clear", tip: a.t("清空", "Clear")}, func(bool) {
					txt(c, a.t("清空", "Clear"), fsTiny, p.muted, 0)
				}) {
					a.later(func() { a.console = nil })
				}
				if a.button(c, btnOpts{w: 20, h: 20, radius: 4, key: "console-close", tip: a.t("关闭", "Close")}, func(bool) {
					icon(c, "clear", 10, p.muted)
				}) {
					a.later(func() { a.setConsoleOpen(false) })
				}
			})
		})
		if len(a.console) == 0 {
			ui.Row(c).Grow(1).Basis(0).FillWidth().Justify(ui.Center).Padding(32, 0, 0, 0).Children(func() {
				txt(c, a.t("暂无请求日志", "No request logs yet"), fsSmall, p.muted, 0)
			})
			return
		}
		a.consoleList.FollowEnd = true
		ui.List(c, &a.consoleList, len(a.console), func(i int) { a.consoleRow(c, i) }).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible())
	})
}

// consoleRow 绘制一条日志：箭头、方法、地址、状态/耗时、时间；点击展开详情。失败行为红底，慢/告警行为橙底。
func (a *app) consoleRow(c *ui.Context, i int) {
	p := a.p
	e := &a.console[i]
	row := ui.Row(c.Key(i)).FillWidth().PaddingX(12).PaddingY(6).Gap(8).AlignItems(ui.Center).Shrink(0)
	switch {
	case e.err != "":
		row = row.Background(p.danger.Alpha(0.1))
	case row.Hovered():
		row = row.Background(p.surface2)
	}
	row.Children(func() {
		arrow := "arrowRight"
		if e.open {
			arrow = "arrowDown"
		}
		icon(c, arrow, 8, p.muted).Shrink(0)
		txtMono(c, strings.ToUpper(e.method), fsSmall, p.method(e.method), 700).Width(44).Shrink(0)
		txtEMono(c, e.url, fsSmall, p.fg, 0).Grow(1).Basis(0).Selectable()
		switch {
		case !e.done:
			txtMono(c, "…", fsSmall, p.muted, 0)
		case e.err != "":
			txtMono(c, "Error", fsTiny, p.danger, 0)
		default:
			txtMono(c, itoa(e.status), fsSmall, a.statusTextColor(e.status), 700)
			txtMono(c, itoa(int(math.Round(e.ms)))+"ms", fsSmall, p.muted, 0)
		}
		txtMono(c, e.at.Format("15:04:05"), fsTiny, p.muted, 0).MarginStart(4)
	})
	if row.Clicked() {
		idx := i
		a.later(func() {
			if idx < len(a.console) {
				a.console[idx].open = !a.console[idx].open
			}
		})
	}
	if !e.open {
		hline(c, p.border.Alpha(0.5))
		return
	}
	bg := p.surface2.Alpha(0.5)
	if e.err != "" {
		bg = p.danger.Alpha(0.06)
	}
	ui.Column(c).FillWidth().Padding(8, 20, 8, 20).Gap(2).Background(bg).Shrink(0).Children(func() {
		txt(c, "Network", fsTiny, p.muted, 0)
		line := func(s string, col ui.Color) {
			ui.Text(c, s).FontSize(fsTiny).Font(monoFont).TextColor(col).Selectable().Margin(0, 0, 0, 8)
		}
		line(e.method+" "+e.url, p.fg2)
		switch {
		case e.err != "":
			line("Error: "+e.err, p.danger)
		case e.done:
			line("Status: "+itoa(e.status), p.fg2)
			line("Duration: "+itoa(int(math.Round(e.ms)))+"ms", p.fg2)
		}
	})
	hline(c, p.border.Alpha(0.5))
}

func (a *app) setConsoleOpen(open bool) {
	a.set.ConsoleOpen = open
	a.saveSettings()
}
