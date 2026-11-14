package gui

import (
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
	a.con.entries = append(a.con.entries, consoleEntry{at: time.Now(), method: method, url: url})
	if len(a.con.entries) > maxConsoleEntries {
		a.con.entries = a.con.entries[len(a.con.entries)-maxConsoleEntries:]
	}
}

// logResponse 把最近一条同请求的日志标记为完成。
func (a *app) logResponse(method, url string, status int, ms float64, errMsg string) {
	for i := len(a.con.entries) - 1; i >= 0; i-- {
		e := &a.con.entries[i]
		if !e.done && e.method == method && e.url == url {
			e.status, e.ms, e.err, e.done = status, ms, errMsg, true
			return
		}
	}
}

// 控制台的日志筛选。
const (
	conAll = iota
	conOK
	conErr
)

// isErr 报告这条日志是否算“错误”：请求失败，或响应状态码 ≥ 400。
func (e *consoleEntry) isErr() bool { return e.err != "" || (e.done && e.status >= 400) }

// conVisible 返回符合筛选的日志下标（保持原有顺序）以及错误条数。
func (a *app) conVisible() (idx []int, errs int) {
	for i := range a.con.entries {
		e := &a.con.entries[i]
		if e.isErr() {
			errs++
		}
		switch a.con.filter {
		case conOK:
			if !e.done || e.isErr() {
				continue
			}
		case conErr:
			if !e.isErr() {
				continue
			}
		}
		idx = append(idx, i)
	}
	return idx, errs
}

// consolePanel 绘制底部控制台：25px 标题行（Console、数量、清空、关闭）+ 请求日志列表。
func (a *app) consolePanel(c *ui.Context) {
	p := a.p
	visible, errCount := a.conVisible()
	ui.Column(c).Grow(1).Basis(0).Children(func() {
		head := ui.Row(c).FillWidth().Height(sizeBottom).Shrink(0).PaddingX(12).Background(p.surface2).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			BorderWidth(0, 0, 1, 0).BorderColor(p.border)
		head.Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				txt(c, a.t("控制台", "Console"), fsSmall, p.fg, 500)
				if n := len(a.con.entries); n > 0 {
					txt(c, a.t(itoa(n)+" 条", itoa(n)+" entries"), fsTiny, p.muted, 0)
				}
				a.consoleFilter(c, errCount)
			})
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				if a.button(c, btnOpts{h: 20, radius: 4, padX: 6, key: "console-clear", tip: a.t("清空", "Clear")}, func(bool) {
					txt(c, a.t("清空", "Clear"), fsTiny, p.muted, 0)
				}) {
					a.later(func() { a.con.entries = nil })
				}
				if a.button(c, btnOpts{w: 20, h: 20, radius: 4, key: "console-close", tip: a.t("关闭", "Close")}, func(bool) {
					icon(c, "clear", 10, p.muted)
				}) {
					a.later(func() { a.setConsoleOpen(false) })
				}
			})
		})
		if len(a.con.entries) == 0 {
			ui.Row(c).Grow(1).Basis(0).FillWidth().Justify(ui.Center).Padding(32, 0, 0, 0).Children(func() {
				txt(c, a.t("暂无请求日志", "No request logs yet"), fsSmall, p.muted, 0)
			})
			return
		}
		if len(visible) == 0 {
			ui.Row(c).Grow(1).Basis(0).FillWidth().Justify(ui.Center).Padding(32, 0, 0, 0).Children(func() {
				txt(c, a.t("没有符合筛选的日志", "No logs match the filter"), fsSmall, p.muted, 0)
			})
			return
		}
		a.con.list.FollowEnd = true
		ui.List(c, &a.con.list, len(visible), func(i int) { a.consoleRow(c, visible[i]) }).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible())
	})
}

// consoleFilter 是控制台标题行里的筛选分段：全部 / 成功 / 错误（带错误条数）。
func (a *app) consoleFilter(c *ui.Context, errCount int) {
	p := a.p
	labels := []string{a.t("全部", "All"), a.t("成功", "OK"), a.t("错误", "Errors")}
	ui.Row(c.Key("con-filter")).Height(20).Padding(1).Gap(1).Radius(6).Background(p.hover).AlignItems(ui.Stretch).Margin(0, 0, 0, 6).Children(func() {
		for i, l := range labels {
			on := a.con.filter == i
			b := ui.ButtonBase(c.Key("conf" + itoa(i))).PaddingX(10).Gap(4).Radius(5).AlignItems(ui.Center).Justify(ui.Center)
			switch {
			case on:
				b = b.Background(p.surface)
			case b.Hovered():
				b = b.Background(p.hover)
			}
			b.Children(func() {
				col, w := p.fg2, 0
				if on {
					col, w = p.fg, 600
				}
				txt(c, l, fsTiny, col, w)
				if i == conErr && errCount > 0 {
					txt(c, itoa(errCount), fsTiny, p.danger, 600)
				}
			})
			if b.Clicked() {
				idx := i
				a.later(func() { a.con.filter = idx })
			}
		}
	})
}

// consoleRow 绘制一条日志：箭头、方法、地址、状态/耗时、时间；点击展开详情。失败行为红底，慢/告警行为橙底。
func (a *app) consoleRow(c *ui.Context, i int) {
	p := a.p
	e := &a.con.entries[i]
	row := ui.Row(c.Key(i)).FillWidth().PaddingX(12).PaddingY(4).Gap(10).AlignItems(ui.Center).Shrink(0)
	switch {
	case e.err != "":
		row = row.Background(p.danger.Alpha(0.1))
	case row.Hovered():
		row = row.Background(p.surface2)
	}
	row.Children(func() {
		// 设计稿 13：时间 | 方向箭头 | 方法缩写 | 地址 | 状态（右对齐、加粗）| 耗时
		txtMono(c, e.at.Format("15:04:05"), fsSmall, p.muted, 0).Width(58).Shrink(0)
		dir := "←"
		if !e.done {
			dir = "→"
		}
		txt(c, dir, fsSmall, p.muted, 0).Width(14).Shrink(0)
		txtMono(c, methodAbbr(e.method), fsTiny, p.method(e.method), 600).Width(40).Shrink(0)
		txtEMono(c, e.url, fsSmall, p.fg, 0).Grow(1).Basis(0).Selectable()
		status, col := "", p.muted
		switch {
		case !e.done:
			status = "…"
		case e.err != "":
			status, col = "ERR", p.danger
		default:
			status, col = itoa(e.status), a.statusTextColor(e.status)
		}
		ui.Row(c).Width(44).Shrink(0).Justify(ui.End).Children(func() { txtMono(c, status, fsSmall, col, 700) })
		ms := ""
		if e.done && (e.err == "" || e.ms > 0) {
			ms = formatDuration(e.ms)
		}
		ui.Row(c).Width(64).Shrink(0).Justify(ui.End).Children(func() { txtMono(c, ms, fsSmall, p.muted, 0) })
	})
	if row.Clicked() {
		idx := i
		a.later(func() {
			if idx < len(a.con.entries) {
				a.con.entries[idx].open = !a.con.entries[idx].open
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
			line("Duration: "+formatDuration(e.ms), p.fg2)
		}
	})
	hline(c, p.border.Alpha(0.5))
}

func (a *app) setConsoleOpen(open bool) {
	a.set.ConsoleOpen = open
	a.saveSettings()
}
