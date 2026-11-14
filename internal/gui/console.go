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

// consolePanel 绘制底部控制台：按时间列出本次运行中发送的请求。
func (a *app) consolePanel(c *ui.Context) {
	th := c.Theme()
	ui.Column(c).Grow(1).Children(func() {
		ui.Row(c).Padding(6, 10).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, a.t("控制台", "Console")).Bold()
			ui.Spacer(c)
			if ui.Button(c, a.t("清空", "Clear")).Clicked() {
				a.later(func() { a.console = nil })
			}
			if ui.Button(c, "×").Clicked() {
				a.later(func() { a.setConsoleOpen(false) })
			}
		})
		ui.Divider(c)
		if len(a.console) == 0 {
			ui.Text(c, a.t("发送请求后会在这里显示记录", "Requests you send appear here")).Padding(10).TextColor(th.TextMuted)
			return
		}
		a.consoleList.FollowEnd = true
		ui.List(c, &a.consoleList, len(a.console), func(i int) {
			e := a.console[i]
			ui.Row(c).Gap(10).PaddingX(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, e.at.Format("15:04:05")).TextColor(th.TextMuted).Width(64).Shrink(0)
				ui.Text(c, e.method).Bold().Width(60).Shrink(0).TextColor(methodColor(e.method, th.Dark))
				ui.Text(c, e.url).SingleLine().Grow(1).Basis(0)
				switch {
				case !e.done:
					ui.Text(c, "…").TextColor(th.TextMuted)
				case e.err != "":
					ui.Text(c, e.err).SingleLine().MaxWidth(280).TextColor(th.Danger)
				default:
					ui.Textf(c, "%d", e.status).TextColor(statusColor(e.status, th))
					ui.Textf(c, "%.0f ms", e.ms).TextColor(th.TextMuted).Width(70).Shrink(0)
				}
			})
		}).Grow(1)
	})
}

func (a *app) setConsoleOpen(open bool) {
	a.set.ConsoleOpen = open
	a.saveSettings()
}
