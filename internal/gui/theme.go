package gui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// appliedTheme 记录已应用到窗口的外观，避免每帧重复设置。
type appliedTheme struct {
	source string
	scheme string
	dark   bool
	accent string
	ready  bool
}

// applyTheme 按设置中的主题模式（亮/暗/跟随系统）与配色方案（雾蓝/墨白/松石）选择调色板，并把对应的 MyGo 主题应用到窗口。
func (a *app) applyTheme(c *ui.Context) {
	// 设置里的“界面字体”只缩放旧版用 --size-font-* 变量的那几处文本，而不是整个界面。
	setUIFontSize(a.set.FontSize)
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
	dark := c.Theme().Dark
	if a.native {
		dark = mygo.Theme.IsDark()
	}
	switch a.set.Theme {
	case "light":
		dark = false
	case "dark":
		dark = true
	}
	accent := ""
	if pr := a.project(); pr != nil {
		accent = pr.ThemeColor
	}
	if a.applied.ready && a.applied.dark == dark && a.applied.source == a.set.Theme && a.applied.accent == accent && a.applied.scheme == a.set.ColorScheme && a.p != nil && a.th != nil {
		c.SetTheme(a.th) // 主题只对当前帧生效，每帧都要设置
		return
	}
	if dark {
		a.p = schemePalette(a.set.ColorScheme, true).withAccent(accent)
	} else {
		a.p = schemePalette(a.set.ColorScheme, false).withAccent(accent)
	}
	a.th = a.p.theme(fsBase)
	c.SetTheme(a.th)
	a.applied = appliedTheme{source: a.set.Theme, scheme: a.set.ColorScheme, dark: dark, accent: accent, ready: true}
}
