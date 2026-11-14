package gui

import "time"

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
