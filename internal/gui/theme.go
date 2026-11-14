package gui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// appliedTheme 记录已应用到窗口的外观，避免每帧重复设置。
type appliedTheme struct {
	source   string
	fontSize int
	dark     bool
	accent   string
	ready    bool
}

// applyTheme 把设置中的主题模式、字号与项目主题色应用到窗口。
func (a *app) applyTheme(c *ui.Context) {
	if a.native && (a.applied.source != a.set.Theme || !a.applied.ready) {
		switch a.set.Theme {
		case "light":
			mygo.Theme.SetSource(mygo.ThemeLight)
		case "dark":
			mygo.Theme.SetSource(mygo.ThemeDark)
		default:
			mygo.Theme.SetSource(mygo.ThemeSystem)
		}
	}
	a.applied.source = a.set.Theme
	dark := c.Theme().Dark
	if a.native {
		dark = mygo.Theme.IsDark()
	}
	accent := ""
	if p := a.project(); p != nil {
		accent = p.ThemeColor
	}
	if a.applied.ready && a.applied.dark == dark && a.applied.fontSize == a.set.FontSize && a.applied.accent == accent {
		return
	}
	th := ui.LightTheme()
	if dark {
		th = ui.DarkTheme()
	}
	if accent != "" {
		th = th.WithAccent(ui.Hex(accent))
	}
	th.FontSize = float32(a.set.FontSize)
	c.SetTheme(th)
	a.applied = appliedTheme{source: a.set.Theme, fontSize: a.set.FontSize, dark: dark, accent: accent, ready: true}
}

// methodColor 返回 HTTP 方法对应的强调色。
func methodColor(method string, dark bool) ui.Color {
	switch method {
	case "GET":
		return pick(dark, "#16a34a", "#4ade80")
	case "POST":
		return pick(dark, "#d97706", "#fbbf24")
	case "PUT":
		return pick(dark, "#2563eb", "#60a5fa")
	case "PATCH":
		return pick(dark, "#7c3aed", "#a78bfa")
	case "DELETE":
		return pick(dark, "#dc2626", "#f87171")
	case "HEAD":
		return pick(dark, "#0d9488", "#2dd4bf")
	}
	return pick(dark, "#6b7280", "#9ca3af")
}

func pick(dark bool, light, darkHex string) ui.Color {
	if dark {
		return ui.Hex(darkHex)
	}
	return ui.Hex(light)
}

// statusColor 按状态码区间返回颜色。
func statusColor(code int, t *ui.Theme) ui.Color {
	switch {
	case code >= 500:
		return t.Danger
	case code >= 400:
		return t.Warning
	case code >= 200 && code < 400:
		return t.Success
	}
	return t.TextMuted
}
