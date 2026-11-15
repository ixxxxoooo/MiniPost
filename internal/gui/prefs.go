package gui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// prefsState 是设置对话框的状态。
type prefsState struct {
	open    bool
	section int // 0 通用 1 外观 2 数据 3 快捷键 4 关于
	timeout string
	maxSize string
	backup  string

	backups       []model.BackupFile // 设置·数据页“最近备份”列表
	backupsLoaded bool               // 列表是否已加载；备份或恢复后置 false 以重新读取
}

func (a *app) openPrefs() {
	a.overlays.prefs = prefsState{
		open:    true,
		timeout: strconv.Itoa(a.set.RequestTimeoutMs),
		maxSize: strconv.Itoa(a.set.MaxResponseSizeMB),
		backup:  strconv.Itoa(a.set.AutoBackupIntervalMinutes),
	}
}

// 设置里各选项与存储值的对应关系。
var (
	httpValues   = []string{"auto", "http1", "http2"}
	detectValues = []string{"auto", "json"}
	themeValues  = []string{"light", "dark", "system"}
	localeValues = []string{localeZH, localeEN}
	fontSizes    = []int{10, 11, 12, 13, 14, 15, 16}
)

// prefsSections 是左侧菜单：图标名与中英文标题。
var prefsSections = []struct{ icon, zh, en string }{
	{"settings", "通用", "General"},
	{"paintBoard", "外观", "Appearance"},
	{"folderOpen", "数据", "Data"},
	{"keyboard", "快捷键", "Shortcuts"},
	{"info", "关于", "About"},
}

// prefsShortcuts 是快捷键列表（描述与按键）。
var prefsShortcuts = []struct{ zh, en, keys string }{
	{"发送请求", "Send request", "⌘ ↵"},
	{"保存", "Save", "⌘ S"},
	{"关闭标签", "Close tab", "⌘ W"},
	{"新建请求", "New request", "⌘ N"},
	{"新建文件夹", "New folder", "⌘ ⇧ N"},
	{"导入 cURL", "Import cURL", "⌘ I"},
	{"导入集合", "Import collection", "⌘ O"},
	{"重命名 / 复制 / 删除", "Rename / duplicate / delete", "⌘E · ⌘D · ⌫"},
	{"在响应中查找", "Find in response", "⌘ F"},
	{"下一个 / 上一个标签", "Next / previous tab", "⌃ Tab · ⌃⇧ Tab"},
	{"复制为 cURL", "Copy as cURL", "⌘ ⇧ C"},
	{"环境", "Environments", "⌘ ⌥ E"},
	{"侧栏 / 控制台", "Sidebar / console", "⌘ B · ⌘ J"},
	{"快速打开（新增）", "Quick open (new)", "⌘ K"},
	{"设置", "Settings", "⌘ ,"},
}

// prefsView 绘制设置对话框（780×520 双栏，设计稿 .set）；任何修改立即生效并保存。
func (a *app) prefsView(c *ui.Context) {
	s := &a.overlays.prefs
	if !s.open {
		return
	}
	p := a.p
	changed := false
	a.dialog(c, &s.open, 780, 520, radiusLg, func() {
		ui.Row(c).FillWidth().FillHeight().AlignItems(ui.Stretch).Children(func() {
			// 左栏：168px，侧栏底色。
			ui.Row(c).Width(168).Shrink(0).Background(p.sidebar).AlignItems(ui.Stretch).Children(func() {
				ui.Column(c).Grow(1).Basis(0).Padding(12, 8).Gap(2).Children(func() {
					txt(c, a.t("设置", "Settings"), fsSmall, p.muted, 600).PaddingX(8).Margin(2, 0, 8, 0)
					for i, sec := range prefsSections {
						active := s.section == i
						b := ui.ButtonBase(c.Key("sec" + strconv.Itoa(i))).FillWidth().Height(30).Radius(radiusR).PaddingX(10).Gap(9).AlignItems(ui.Center).Justify(ui.Start)
						col := p.fg2
						switch {
						case active:
							b = b.Background(p.acSoft)
							col = p.accent
						case b.Hovered():
							b = b.Background(p.btnBg)
						}
						b.Children(func() {
							if active { // 设计稿 .ni.on::before：左缘 3px 强调色竖条 + 柔光
								ui.Box(c.Key("secbar")).Absolute().Left(-8).Top(7).Bottom(7).Width(3).Radius(0, 1.5, 1.5, 0).Background(p.accent).Shadow(0, 0, 8, 0, p.glow)
							}
							ui.Row(c).Grow(1).Gap(8).AlignItems(ui.Center).Children(func() {
								icon(c, sec.icon, 14, col)
								txt(c, a.t(sec.zh, sec.en), fsBase, col, 550)
							})
						})
						if b.Clicked() {
							s.section = i
						}
					}
					ui.Spacer(c)
					txtMono(c, "MiniPost "+a.appVersion(), fsSmall, p.muted, 0).Margin(0, 8, 2, 8)
				})
				vline(c, p.border)
			})
			// 右栏：44px 标题行 + 可滚动内容。
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				ui.Row(c).Height(44).FillWidth().Shrink(0).Padding(0, 12, 0, 20).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
					txt(c, a.t(prefsSections[s.section].zh, prefsSections[s.section].en), 14, p.fg, 650)
					a.ibSm(c, "prefs-close", "x", a.t("关闭", "Close"), false, func() { s.open = false })
				})
				hline(c, p.border)
				ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
					ui.Column(c).FillWidth().Padding(4, 20, 14, 20).Gap(6).Children(func() {
						switch s.section {
						case 0:
							changed = a.prefsGeneral(c, s) || changed
						case 1:
							changed = a.prefsAppearance(c, s) || changed
						case 2:
							changed = a.prefsData(c, s) || changed
						case 3:
							a.prefsShortcutsView(c)
						default:
							a.prefsAbout(c)
						}
					})
				})
				{
					hline(c, p.border)
					ui.Row(c).FillWidth().Height(38).Shrink(0).Padding(0, 16, 0, 20).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
						txt(c, a.t("修改会立即生效并自动保存", "Changes apply immediately and are saved automatically"), fsSmall, p.muted, 0)
						if s.section == 0 && a.btn(c, dbtn{key: "prefs-reset", label: a.t("恢复默认", "Restore defaults"), sm: true}) {
							a.resetGeneralSettings(s)
							changed = true
						}
					})
				}
			})
		})
	})
	if changed {
		a.later(a.applySettings)
	}
}

// settingsRow 是一行设置：左侧标题与说明，右侧控件；last 为 true 时不画下边线。
func (a *app) settingsRow(c *ui.Context, key string, title, desc string, last bool, control func()) {
	p := a.p
	row := ui.Row(c.Key(key)).FillWidth().PaddingY(8.5).Gap(16).AlignItems(ui.Center)
	if !last {
		row = row.BorderWidth(0, 0, 1, 0).BorderColor(p.border.Alpha(0.65))
	}
	row.Children(func() {
		ui.Column(c).Grow(1).Basis(0).Gap(1).Children(func() {
			ui.Text(c, title).FontSize(fsBase+0.5).FontWeight(600).FixedLineHeight(18).TextColor(p.fg).Translate(0, textShiftY)
			ui.Text(c, desc).FontSize(fsSmall).FixedLineHeight(16).TextColor(p.muted).Translate(0, textShiftY)
		})
		ui.Row(c).Shrink(0).AlignItems(ui.Center).Children(control)
	})
}

// numberField 是带单位的数字输入（高 28，右对齐）；返回是否变化。
func (a *app) numberField(c *ui.Context, key string, v *string, unit string, disabled bool) bool {
	p := a.p
	in := ui.Row(c.Key(key)).Height(28).Radius(radiusInput).Border(1, p.btnBorder).Background(p.surface).AlignItems(ui.Center).PaddingEnd(8).Shrink(0)
	changed := false
	in.Children(func() {
		e := ui.TextInputBase(c, v).Width(78).FillHeight().PaddingX(10).FontSize(fsSmall).TextColor(p.fg).TextAlign(ui.End)
		if disabled {
			e = e.Disabled(true)
		}
		changed = e.Changed()
		txt(c, unit, fsSmall, p.muted, 0).Margin(0, 0, 0, 4)
	})
	return changed
}

func atoiOr(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

// selectBox 是设置页的下拉选择（高 28、圆角 8、右侧小箭头）；选中新项时返回 true。
func (a *app) selectBox(c *ui.Context, key string, sel *int, labels []string) bool {
	p := a.p
	changed := false
	b := ui.ButtonBase(c.Key(key)).Width(104).Height(28).Radius(radiusInput).PaddingX(10).AlignItems(ui.Center).Border(1, p.btnBorder).Background(p.surface)
	b.PopoverMenuButton(func(m *ui.Menu) {
		for i, l := range labels {
			if m.Item(l).Checked(*sel == i).Chosen() {
				i := i
				a.later(func() { *sel = i })
				changed = true
			}
		}
	}, a.menuOpts()).Children(func() {
		txt(c, labels[min(max(*sel, 0), len(labels)-1)], fsBase, p.fg, 0)
		ui.Spacer(c)
		icon(c, "chevUD", 11, p.muted)
	})
	return changed
}

// prefsGroup 画设置分组：小标题 + 一组无外框、行间细线的设置行。
func (a *app) prefsGroup(c *ui.Context, title string, rows func()) {
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		a.prefsSectionLabel(c, title, "")
		rows()
	})
}

// resetGeneralSettings 把“通用”页的各项恢复为默认值（不动外观、布局和项目相关的设置）。
func (a *app) resetGeneralSettings(s *prefsState) {
	d := client.DefaultSettings()
	set := &a.set
	set.HTTPVersion, set.RequestTimeoutMs, set.MaxResponseSizeMB = d.HTTPVersion, d.RequestTimeoutMs, d.MaxResponseSizeMB
	set.ResponseFormatDetection, set.SSLVerify, set.FollowRedirects = d.ResponseFormatDetection, d.SSLVerify, d.FollowRedirects
	set.SSLTLSKeyLog, set.DisableCookies = d.SSLTLSKeyLog, d.DisableCookies
	set.SendNoCacheHeader, set.SendPostmanTokenHeader = d.SendNoCacheHeader, d.SendPostmanTokenHeader
	set.AlwaysSaveUnsaved, set.AlwaysDiscardUnsaved = d.AlwaysSaveUnsaved, d.AlwaysDiscardUnsaved
	s.timeout, s.maxSize = strconv.Itoa(set.RequestTimeoutMs), strconv.Itoa(set.MaxResponseSizeMB)
	logger.Info("已恢复通用设置默认值")
}

func (a *app) prefsGeneral(c *ui.Context, s *prefsState) bool {
	changed := false
	set := &a.set
	a.prefsGroup(c, a.t("请求", "Request"), func() {
		a.settingsRow(c, "http", a.t("HTTP 版本", "HTTP version"), a.t("发送请求时优先使用的 HTTP 版本。", "Preferred HTTP version when sending requests."), false, func() {
			i := indexOf(httpValues, set.HTTPVersion)
			if a.selectBox(c, "http-sel", &i, []string{a.t("自动", "Auto"), "HTTP/1.1", "HTTP/2"}) {
				set.HTTPVersion = httpValues[i]
				changed = true
			}
		})
		a.settingsRow(c, "timeout", a.t("请求超时", "Request timeout"), a.t("单位毫秒，0 表示不主动超时。", "In milliseconds. 0 disables active timeout."), false, func() {
			if a.numberField(c, "timeout-in", &s.timeout, "ms", false) {
				set.RequestTimeoutMs = atoiOr(s.timeout, 0)
				changed = true
			}
		})
		a.settingsRow(c, "maxsize", a.t("最大响应大小", "Max response size"), a.t("单位 MB，0 表示不限制。", "In MB. 0 means unlimited."), false, func() {
			if a.numberField(c, "maxsize-in", &s.maxSize, "MB", false) {
				set.MaxResponseSizeMB = atoiOr(s.maxSize, 0)
				changed = true
			}
		})
		a.settingsRow(c, "detect", a.t("响应格式检测", "Response format detection"), a.t("自动按 Content-Type 识别，JSON 优先使用 JSON 视图。", "Auto detects by Content-Type; JSON prefers the JSON view."), true, func() {
			i := indexOf(detectValues, set.ResponseFormatDetection)
			if a.selectBox(c, "detect-sel", &i, []string{a.t("自动", "Auto"), "JSON"}) {
				set.ResponseFormatDetection = detectValues[i]
				changed = true
			}
		})
	})
	sw := func(key string, on *bool, title, desc string, last bool, after func()) {
		a.settingsRow(c, key, title, desc, last, func() {
			if a.toggle(c, key+"-sw", on) {
				changed = true
				if after != nil {
					after()
				}
			}
		})
	}
	a.prefsGroup(c, a.t("安全与网络", "Security & network"), func() {
		sw("ssl", &set.SSLVerify, a.t("SSL 证书校验", "SSL certificate verification"), a.t("关闭后会跳过证书校验，仅建议在调试环境使用。", "If disabled, certificate verification is skipped (debugging only)."), false, nil)
		sw("redirect", &set.FollowRedirects, a.t("自动跟随重定向", "Follow redirects"), a.t("301 / 302 / 307 / 308，行为与 Postman 一致。", "301 / 302 / 307 / 308, same behavior as Postman."), false, nil)
		sw("nocookie", &set.DisableCookies, a.t("禁用 Cookies", "Disable cookies"), a.t("不读写 Cookie Jar，不自动注入或保存。", "Do not read or write the Cookie Jar; nothing is injected or saved."), false, nil)
		sw("nocache", &set.SendNoCacheHeader, a.t("发送 no-cache 请求头", "Send no-cache header"), a.t("自动追加 Cache-Control: no-cache。", "Adds Cache-Control: no-cache automatically."), false, nil)
		sw("token", &set.SendPostmanTokenHeader, a.t("发送 MiniPost-Token 请求头", "Send MiniPost-Token header"), a.t("每次请求追加随机 Token（MiniPost-Token），便于区分和排查，默认关闭。", "Appends a random MiniPost-Token to each request for tracing; off by default."), false, nil)
		sw("keylog", &set.SSLTLSKeyLog, "SSL/TLS Key Log", a.t("记录 TLS 会话密钥，用于抓包调试。", "Log TLS session keys for packet-capture debugging."), true, nil)
	})
	a.prefsGroup(c, a.t("标签页", "Tabs"), func() {
		sw("save", &set.AlwaysSaveUnsaved, a.t("关闭标签时默认保存修改", "Save changes by default when closing tabs"), a.t("开启后关闭标签会先自动保存。会自动关闭“默认丢弃”。", "Closing a tab saves first. Turns off default discard."), false, func() {
			if set.AlwaysSaveUnsaved {
				set.AlwaysDiscardUnsaved = false
			}
		})
		sw("discard", &set.AlwaysDiscardUnsaved, a.t("关闭标签时默认丢弃修改", "Discard changes by default when closing tabs"), a.t("开启后关闭标签不再询问，直接丢弃未保存内容。会自动关闭“默认保存”。", "Closing a tab no longer asks; unsaved changes are discarded. Turns off default save."), true, func() {
			if set.AlwaysDiscardUnsaved {
				set.AlwaysSaveUnsaved = false
			}
		})
	})
	return changed
}

// miniWindow 画一个迷你窗口预览（设计稿外观页的缩略图）：左侧栏条 + 右侧内容（强调色条 + 两行灰线）。
// half 为 1 时只画侧栏 + 左半内容，为 2 时只画右半内容（用于“跟随系统”的左浅右深拼接）。
func miniWindow(c *ui.Context, t schemeTokens, half int) {
	if half != 2 {
		ui.Column(c).Width(48).Shrink(0).FillHeight().Padding(8, 6).Gap(4).Background(ui.Hex(t.side)).Children(func() {
			ui.Box(c).Width(22).Height(4).Radius(2).Background(ui.Hex(t.bd2))
		})
	}
	ui.Column(c).Grow(1).Basis(0).FillHeight().Padding(8, 10).Gap(5).Background(ui.Hex(t.bg)).Children(func() {
		ui.Box(c).Width(46).Height(5).Radius(2).Background(ui.Hex(t.ac))
		ui.Box(c).FillWidth().Height(4).Radius(2).Background(ui.Hex(t.bd2))
		ui.Box(c).Width(60).Height(4).Radius(2).Background(ui.Hex(t.bd2))
	})
}

// previewCard 是外观页的缩略图选项卡（设计稿 .thumb）：上方迷你窗口，下方名称（可带右侧色点）。
// 选中时 1.5px 强调色边框 + 3px 淡色光环，并在右上角挂一个渐变小对勾。
func (a *app) previewCard(c *ui.Context, key, label string, selected bool, preview func(), dots []ui.Color, labelLeft bool) bool {
	p := a.p
	b := ui.ButtonBase(c.Key(key)).Grow(1).Basis(0).MinWidth(0).Radius(radiusLg).Padding(6).Background(p.surface)
	bd := p.border
	switch {
	case selected:
		bd = p.accent
		b = b.Shadow(0, 0, 0, 3, p.acSoft)
	case b.Hovered():
		bd = p.btnBorder
	}
	b = b.Border(1.5, bd)
	b.Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			ui.Row(c).FillWidth().Height(54).Radius(7).Clip().Border(1, p.border).Children(preview)
			ui.Row(c).FillWidth().Height(20).Padding(0, 4, 2, 4).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				if !labelLeft {
					ui.Spacer(c)
				}
				txt(c, label, fsSmall+0.5, p.fg, 600)
				if !labelLeft {
					ui.Spacer(c)
				}
				if len(dots) > 0 {
					ui.Row(c).Gap(3).AlignItems(ui.Center).Children(func() {
						for _, d := range dots {
							ui.Box(c).Size(9, 9).Radius(4.5).Background(d).Border(1, ui.RGBA(0, 0, 0, 0.12))
						}
					})
				}
			})
		})
		if selected {
			ui.Row(c).Absolute().Top(-7).Right(-7).Size(17, 17).Radius(8.5).AlignItems(ui.Center).Justify(ui.Center).
				Gradient(p.accent.Mix(ui.RGB(255, 255, 255), 0.16), p.accent, 180).Shadow(0, 0, 0, 2, p.glass.Alpha(1)).Shadow(0, 2, 8, 0, p.glow).Children(func() {
				icon(c, "checkBold", 9, p.accentFg)
			})
		}
	})
	return b.Clicked()
}

// prefsSectionLabel 是设置页分组的小标题（设计稿 .sgrp）：11px 弱色 650 + 可选小徽标 + 向右延伸的细线。
func (a *app) prefsSectionLabel(c *ui.Context, s, badge string) {
	p := a.p
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Margin(14, 0, 0, 0).Children(func() {
		txt(c, s, fsSmall, p.muted, 650)
		if badge != "" {
			ui.Row(c).Height(16).PaddingX(6).Radius(4).Background(p.acSoft).AlignItems(ui.Center).Children(func() {
				txt(c, badge, 10, p.accent, 600)
			})
		}
		ui.Box(c).Grow(1).Height(1).Background(p.border)
	})
}

func (a *app) prefsAppearance(c *ui.Context, s *prefsState) bool {
	changed := false
	set := &a.set
	p := a.p
	scheme := set.ColorScheme
	if _, ok := colorSchemes[scheme]; !ok {
		scheme = "mist"
	}
	light, dark := colorSchemes[scheme][0], colorSchemes[scheme][1]

	// 外观主题：浅色 / 深色 / 跟随系统（左浅右深）
	a.prefsSectionLabel(c, a.t("外观主题", "Theme"), "")
	ui.Row(c.Key("theme")).FillWidth().Gap(10).Margin(6, 0, 2, 0).Children(func() {
		names := []string{a.t("浅色", "Light"), a.t("深色", "Dark"), a.t("跟随系统", "System")}
		for i, v := range themeValues {
			i, v := i, v
			preview := func() {
				switch v {
				case "light":
					miniWindow(c, light, 0)
				case "dark":
					miniWindow(c, dark, 0)
				default:
					miniWindow(c, light, 1)
					miniWindow(c, dark, 2)
				}
			}
			if a.previewCard(c, "theme-"+v, names[i], set.Theme == v, preview, nil, true) {
				set.Theme = v
				changed = true
			}
		}
	})
	// 配色：雾蓝 / 墨白 / 松石 / 琥珀 / 玫瑰
	a.prefsSectionLabel(c, a.t("配色", "Color scheme"), a.t("新增 2 套", "2 new"))
	ui.Row(c.Key("scheme")).FillWidth().Gap(8).Margin(6, 0, 2, 0).Children(func() {
		names := map[string]string{"mist": a.t("雾蓝", "Mist"), "ink": a.t("墨白", "Ink"), "jade": a.t("松石", "Jade"), "amber": a.t("琥珀", "Amber"), "rose": a.t("玫瑰", "Rose")}
		for _, name := range colorSchemeNames {
			name := name
			tk := colorSchemes[name][0]
			if p.dark {
				tk = colorSchemes[name][1]
			}
			dots := []ui.Color{ui.Hex(tk.ac), ui.Hex(tk.bg2), ui.Hex(tk.fg)}
			if a.previewCard(c, "scheme-"+name, names[name], scheme == name, func() { miniWindow(c, tk, 0) }, dots, true) {
				set.ColorScheme = name
				logger.Info("已切换配色方案", "scheme", set.ColorScheme)
				changed = true
			}
		}
	})
	txt(c, a.t("项目主题色仍然可用：它会覆盖这里的强调色，只影响选中态、高亮和发送按钮。", "Project theme colors still work: they override the accent here, affecting only selection, highlights and the send button."), fsSmall, p.muted, 0)

	cur := indexOf(localeValues, set.Locale)
	if set.Locale == "" {
		cur = 0
		if a.locale() == localeEN {
			cur = 1
		}
	}
	a.settingsRow(c, "lang", a.t("语言", "Language"), a.t("立即生效，并记住你的选择。", "Takes effect immediately and will be remembered."), true, func() {
		if a.segmented(c, "lang-seg", &cur, []string{"中文", "English"}) {
			set.Locale = localeValues[cur]
			changed = true
		}
	})
	a.settingsRow(c, "font", a.t("界面字体", "UI font size"), a.t("调整菜单、标签、按钮和面板的文字大小。", "Adjusts text size for menus, tabs, buttons, and panels."), true, func() {
		ui.Row(c).Gap(4).Children(func() {
			for _, f := range fontSizes {
				f := f
				active := set.FontSize == f
				b := ui.ButtonBase(c.Key("fs"+strconv.Itoa(f))).Size(28, 26).Radius(radiusSm).AlignItems(ui.Center).Justify(ui.Center)
				col, bd, w := p.fg, p.btnBorder, 550
				switch {
				case active:
					col, bd, w = p.accent, p.accent, 700
					b = b.Background(p.acSoft).Shadow(0, 0, 0, 3, p.acSoft)
				case b.Hovered():
					b = b.Background(p.hover)
				}
				b = b.Border(1, bd)
				b.Children(func() { txt(c, strconv.Itoa(f), fsSmall, col, w) })
				if b.Clicked() {
					set.FontSize = f
					changed = true
				}
			}
		})
	})
	a.settingsRow(c, "scroll", a.t("滚动条自动隐藏", "Auto-hide scrollbar"), a.t("滚动时显示，静止后自动隐藏。", "Shows while scrolling and hides when idle."), true, func() {
		if a.toggle(c, "scroll-sw", &set.ScrollbarAutoHide) {
			changed = true
		}
	})
	return changed
}

func (a *app) prefsData(c *ui.Context, s *prefsState) bool {
	changed := false
	set := &a.set
	p := a.p
	if !s.backupsLoaded {
		list, err := a.be.ListBackups(4)
		if err != nil {
			logger.Warn("读取备份列表失败", "error", err.Error())
		}
		s.backups, s.backupsLoaded = list, true
	}
	ui.Column(c).FillWidth().Children(func() {
		a.settingsRow(c, "backup", a.t("自动备份", "Auto backup"), a.t("定时备份请求、环境和历史记录。", "Back up requests, environments and history on a schedule."), false, func() {
			if a.toggle(c, "backup-sw", &set.AutoBackupEnabled) {
				changed = true
			}
		})
		a.settingsRow(c, "interval", a.t("备份间隔", "Backup interval"), a.t("范围 5 - 1440 分钟。", "Range 5 - 1440 minutes."), true, func() {
			if a.numberField(c, "interval-in", &s.backup, a.t("分钟", "min"), !set.AutoBackupEnabled) {
				set.AutoBackupIntervalMinutes = atoiOr(s.backup, 5)
				changed = true
			}
		})
	})
	a.prefsGroup(c, a.t("立即操作", "Actions"), func() {
		ui.Row(c).Gap(8).Padding(2, 0, 4, 0).Children(func() {
			bd := p.btnBorder
			if a.button(c, btnOpts{h: 28, radius: 7, padX: 10, gap: 6, border: &bd, key: "backup-now"}, func(bool) {
				icon(c, "download", 12, p.success)
				txt(c, a.t("立即备份", "Back up now"), fsSmall, p.fg, 0)
			}) {
				a.later(a.backupNow)
			}
			if a.button(c, btnOpts{h: 28, radius: 7, padX: 10, gap: 6, border: &bd, key: "backup-restore"}, func(bool) {
				icon(c, "upload", 12, p.success)
				txt(c, a.t("恢复备份", "Restore backup"), fsSmall, p.fg, 0)
			}) {
				a.later(a.restoreBackup)
			}
		})
		txt(c, a.t("恢复会覆盖当前数据，建议先备份一次。", "Restoring replaces current data; back up first."), fsSmall, p.muted, 0)
	})
	a.prefsGroup(c, a.t("最近备份", "Recent backups"), func() {
		if len(s.backups) == 0 {
			txt(c, a.t("还没有备份", "No backups yet"), fsSmall, p.muted, 0).Margin(4, 0)
			return
		}
		ui.Column(c).FillWidth().Gap(6).Padding(2, 0, 0, 0).Children(func() {
			for _, bf := range s.backups {
				bf := bf
				// 设计稿 .bk：淡底圆角行，数据库图标 + 等宽时间 + 大小 + 右侧“恢复”幽灵按钮
				ui.Row(c.Key("bk:" + bf.Name)).FillWidth().Height(34).Radius(radiusInput).PaddingX(10).Gap(10).AlignItems(ui.Center).Background(p.surface2.Alpha(0.7)).Children(func() {
					icon(c, "database", 14, p.fg2)
					txtMono(c, bf.ModTime.Format("2006-01-02 15:04"), fsBase, p.fg, 0).Grow(1)
					txt(c, formatSizeShort(bf.Size), fsSmall, p.muted, 0)
					if a.btn(c, dbtn{key: "bkr:" + bf.Name, label: a.t("恢复", "Restore"), sm: true, ghost: true}) {
						path := bf.Path
						a.later(func() { a.confirmRestore(path) })
					}
				})
			}
		})
	})
	dir := a.be.DataDir()
	ui.Column(c).FillWidth().Gap(4).Children(func() {
		a.prefsSectionLabel(c, a.t("数据位置", "Data location"), "")
		ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
			txtEMono(c, tildePath(dir), fsSmall, p.fg2, 0).Grow(1).Basis(0).Selectable()
			bd := p.btnBorder
			if a.button(c, btnOpts{h: 26, radius: 7, padX: 10, border: &bd, key: "reveal-data"}, func(bool) {
				txt(c, a.t("在 Finder 中显示", "Show in Finder"), fsSmall, p.fg2, 0)
			}) {
				a.later(func() { revealInFileManager(dir) })
			}
		})
	})
	return changed
}

// tildePath 把家目录前缀缩写成 ~，用于展示数据位置。
func tildePath(path string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// revealInFileManager 在系统文件管理器里打开目录（macOS Finder / Linux xdg-open / Windows 资源管理器）。
func revealInFileManager(dir string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dir)
	case "windows":
		cmd = exec.Command("explorer", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		logger.Warn("打开文件管理器失败", "error", err.Error())
		return
	}
	logger.Info("已在文件管理器中显示数据目录")
	go func() { _ = cmd.Wait() }()
}

func (a *app) prefsShortcutsView(c *ui.Context) {
	p := a.p
	ui.Column(c).FillWidth().Children(func() {
		for _, sc := range prefsShortcuts {
			ui.Row(c.Key(sc.keys)).FillWidth().Height(32).Shrink(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
				txt(c, a.t(sc.zh, sc.en), fsBase, p.fg, 0)
				ui.Row(c).Height(18).Radius(5).Background(p.surface2).PaddingX(7).AlignItems(ui.Center).Children(func() {
					txtMono(c, sc.keys, fsTiny, p.muted, 0)
				})
			})
		}
	})
}

func (a *app) prefsAbout(c *ui.Context) {
	p := a.p
	ui.Column(c).FillWidth().AlignItems(ui.Center).Gap(8).Padding(8, 0, 0, 0).Children(func() {
		if img := logoImage(p.dark); img != nil {
			ui.Image(c, img).Size(72, 72).Fit(ui.Contain).Shadow(0, 8, 18, 0, p.glowS)
		}
		txt(c, "MiniPost", 16, p.fg, 600)
		txtMono(c, a.t("版本 ", "Version ")+a.appVersion(), fsSmall, p.muted, 0)
		ui.Text(c, a.t("轻量的 HTTP 调试工具。支持多项目、环境变量、历史记录、响应预览与 Cookie 管理。",
			"A lightweight HTTP debugging tool with multi-project support, environment variables, history, response preview and cookie management.")).
			FontSize(fsBase).FixedLineHeight(22).TextColor(p.fg2).TextAlign(ui.Center).MaxWidth(540).Margin(4, 0, 4, 0)
	})
	a.updateCard(c)
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		note := func(ic, zh, en string) {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				icon(c, ic, 12, p.success)
				txt(c, a.t(zh, en), fsSmall, p.fg2, 0)
			})
		}
		note("okC", "数据只保存在本机 ~/.minipost，仅所有者可读写。", "Data stays on this machine in ~/.minipost, readable only by you.")
		note("okC", "日志不记录请求体、令牌和 Cookie。", "Logs never include request bodies, tokens or cookies.")
	})
}

func indexOf(values []string, v string) int {
	for i, x := range values {
		if x == v {
			return i
		}
	}
	return 0
}

// applySettings 规整并保存设置，重建依赖设置的后台任务。
func (a *app) applySettings() {
	a.set.Normalize()
	a.saveSettings()
	a.restartBackupTimer()
	a.applied.ready = false
	logger.Info("设置已更新")
}

func (a *app) backupNow() {
	a.async(func() func() {
		path, err := a.be.CreateBackup()
		return func() {
			if err != nil {
				a.fail(a.t("创建备份", "Back up"), err)
				return
			}
			a.overlays.prefs.backupsLoaded = false
			a.toastOK(a.t("备份已创建：", "Backup created: ") + path)
		}
	})
}

func (a *app) restoreBackup() {
	a.async(func() func() {
		path, err := a.be.OpenFileDialogAny(a.win)
		return func() {
			if err != nil {
				a.fail(a.t("选择备份", "Choose backup"), err)
				return
			}
			if path == "" {
				return
			}
			a.confirmRestore(path)
		}
	})
}

// confirmRestore 弹出确认后用指定备份文件覆盖当前数据。
func (a *app) confirmRestore(path string) {
	a.overlays.askConfirm(a.t("从备份恢复？", "Restore from backup?"),
		a.t("当前数据会被备份内容替换。", "Current data will be replaced by the backup."), a.t("恢复", "Restore"), func() {
			if err := a.be.RestoreBackup(path); err != nil {
				a.fail(a.t("恢复备份", "Restore backup"), err)
				return
			}
			a.overlays.prefs.backupsLoaded = false
			logger.Info("已从备份恢复", "file", filepath.Base(path))
			a.reloadAll()
			a.toastOK(a.t("已从备份恢复", "Restored from backup"))
		})
}

// reloadAll 在数据被外部替换（恢复备份、导入）后重新加载全部内容。
func (a *app) reloadAll() {
	a.loadProjects()
	id := a.projectID
	a.projectID = ""
	for _, p := range a.projects {
		if p.ID == id {
			a.selectProject(id)
			return
		}
	}
	a.selectProject("")
}

// restartBackupTimer 按设置启停定时自动备份。
func (a *app) restartBackupTimer() {
	if a.stopBackup != nil {
		a.stopBackup()
		a.stopBackup = nil
	}
	if !a.set.AutoBackupEnabled || !a.native {
		return
	}
	interval := time.Duration(a.set.AutoBackupIntervalMinutes) * time.Minute
	a.stopBackup = startPeriodic(interval, interval, func() {
		if path, err := a.be.CreateBackup(); err != nil {
			logger.Warn("自动备份失败", "error", err.Error())
		} else {
			logger.Info("自动备份完成", "path", path)
		}
	})
}
