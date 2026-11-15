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
	size   int64 // 响应体字节数
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
func (a *app) logResponse(method, url string, status int, ms float64, size int64, errMsg string) {
	for i := len(a.con.entries) - 1; i >= 0; i-- {
		e := &a.con.entries[i]
		if !e.done && e.method == method && e.url == url {
			e.status, e.ms, e.size, e.err, e.done = status, ms, size, errMsg, true
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

// consolePanel 绘制底部控制台（设计稿 .con）：28px 标题行（图标、控制台、数量、筛选分段、清空、关闭）+ 24px 一行的请求日志。
func (a *app) consolePanel(c *ui.Context) {
	p := a.p
	visible, errCount := a.conVisible()
	ui.Column(c).Grow(1).Basis(0).Children(func() {
		head := ui.Row(c).FillWidth().Height(28).Shrink(0).PaddingStart(12).PaddingEnd(8).Gap(8).Background(p.surface2.Mix(p.surface, 0.5)).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(p.border)
		head.Children(func() {
			icon(c, "terminal", 13, p.fg)
			txt(c, a.t("控制台", "Console"), fsBase, p.fg, 650)
			if n := len(a.con.entries); n > 0 {
				ui.Row(c).Height(15).MinWidth(16).PaddingX(4).Radius(8).Background(p.surface2).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
					txtMono(c, itoa(n), fsTiny, p.muted, 600)
				})
			}
			a.consoleFilter(c, errCount)
			ui.Spacer(c)
			if a.btn(c, dbtn{key: "console-clear", label: a.t("清空", "Clear"), ghost: true, sm: true, tip: a.t("清空", "Clear")}) {
				a.later(func() { a.con.entries = nil })
			}
			a.ibSm(c, "console-close", "x", a.t("关闭", "Close"), false, func() { a.setConsoleOpen(false) })
		})
		if len(a.con.entries) == 0 {
			a.consoleEmpty(c, a.t("暂无请求日志", "No request logs yet"))
			return
		}
		if len(visible) == 0 {
			a.consoleEmpty(c, a.t("没有符合筛选的日志", "No logs match the filter"))
			return
		}
		a.con.list.FollowEnd = true
		ui.List(c, &a.con.list, len(visible), func(i int) { a.consoleRow(c, visible[i]) }).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible())
	})
}

// consoleEmpty 是控制台没有内容时的居中提示。
func (a *app) consoleEmpty(c *ui.Context, text string) {
	p := a.p
	ui.Column(c).Grow(1).Basis(0).FillWidth().Padding(24).Gap(8).AlignItems(ui.Center).Children(func() {
		icon(c, "terminal", 22, p.muted.Alpha(0.7))
		txt(c, text, fsBase, p.muted, 0)
	})
}

// consoleFilter 是控制台标题行里的筛选分段：全部 / 成功 / 错误（带错误条数），设计稿 .seg（20 高的小号胶囊）。
func (a *app) consoleFilter(c *ui.Context, errCount int) {
	labels := []string{a.t("全部", "All"), a.t("成功", "OK"), a.t("错误", "Errors")}
	if errCount > 0 {
		labels[conErr] += " " + itoa(errCount)
	}
	sel := a.con.filter
	if a.pillSeg(c, "con-filter", &sel, labels, nil, segOpts{h: 20, padX: 9, fs: fsBase}) {
		a.con.filter = sel
	}
}

// consoleRow 绘制一条日志（设计稿 .con-r）：时间 | 方向 | 方法徽章 | 地址 | 状态 | 耗时 | 大小；点击展开详情。
// 展开行是次级底色并带左侧强调竖线，悬停是浅竖线。
func (a *app) consoleRow(c *ui.Context, i int) {
	p := a.p
	e := &a.con.entries[i]
	failed := e.err != "" || (e.done && e.status >= 400)
	row := ui.ButtonBase(c.Key(i)).FillWidth().Height(24).PaddingX(12).Gap(6).AlignItems(ui.Center).Justify(ui.Start).Shrink(0)
	switch {
	case e.open:
		row = row.Background(p.surface2).InsetShadow(2, 0, 0, 0, p.accent)
	case row.Hovered():
		row = row.Background(p.hover).InsetShadow(2, 0, 0, 0, p.acLine)
	}
	row.Children(func() {
		txtMono(c, e.at.Format("15:04:05"), fsSmall+0.5, p.muted, 0).Width(62).Shrink(0)
		dir, dcol := "←", p.success
		switch {
		case !e.done:
			dir, dcol = "→", p.muted
		case failed:
			dir, dcol = "✕", p.danger
		}
		txt(c, dir, fsSmall+0.5, dcol, 0).Width(16).Shrink(0)
		txtMono(c, e.method, fsTiny+0.5, p.method(e.method), 600).Width(48).Shrink(0)
		txtEMono(c, e.url, fsSmall+0.5, p.fg, 0).Grow(1).Basis(0)
		status, col := "", p.muted
		switch {
		case !e.done:
			status = "…"
		case e.err != "":
			status, col = "ERR", p.danger
		default:
			status, col = itoa(e.status), a.statusTextColor(e.status)
		}
		ui.Row(c).Width(40).Shrink(0).Children(func() { txtMono(c, status, fsSmall+0.5, col, 600) })
		ms, size := "—", "—"
		if e.done && (e.err == "" || e.ms > 0) {
			ms = spaceUnit(formatDuration(e.ms))
		}
		if e.done && e.err == "" {
			size = formatSizeShort(e.size)
		}
		ui.Row(c).Width(62).Shrink(0).Children(func() { txtMono(c, ms, fsSmall+0.5, p.muted, 0) })
		ui.Row(c).Width(56).Shrink(0).Children(func() { txtMono(c, size, fsSmall+0.5, p.muted, 0) })
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
		return
	}
	// 详情：缩进到地址列（78px），次级底色，等宽 11.5/17
	ui.Column(c).FillWidth().Padding(6, 12, 8, 78).Background(p.surface2).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Shrink(0).Children(func() {
		line := func(s string, col ui.Color) {
			ui.Text(c, s).FontSize(fsSmall + 0.5).Font(monoFont).TextColor(col).Selectable()
		}
		line(e.method+" "+e.url, p.fg2)
		switch {
		case e.err != "":
			line(a.t("错误: ", "Error: ")+e.err, p.danger)
		case e.done:
			line(a.t("状态: ", "Status: ")+itoa(e.status)+a.t(" · 耗时 ", " · duration ")+spaceUnit(formatDuration(e.ms)), p.fg2)
			line(a.t("响应大小: ", "Size: ")+formatSizeShort(e.size), p.fg2)
		}
	})
}

func (a *app) setConsoleOpen(open bool) {
	a.set.ConsoleOpen = open
	a.saveSettings()
}
