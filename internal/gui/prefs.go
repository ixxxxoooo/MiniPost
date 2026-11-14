package gui

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/pkg/logger"
)

// prefsState 是设置对话框的状态。
type prefsState struct {
	open    bool
	section int // 0 通用 1 外观 2 数据 3 快捷键 4 关于
	timeout string
	maxSize string
	backup  string
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
	{"发送请求", "Send request", "⌘ + Enter"},
	{"保存请求", "Save request", "⌘ + S"},
	{"新建请求", "Create request", "⌘ + N"},
	{"导入 cURL", "Import cURL", "⌘ + I"},
	{"重命名选中请求/文件夹", "Rename selected request/folder", "⌘ + E"},
	{"复制选中请求/文件夹", "Duplicate selected request/folder", "⌘ + D"},
	{"删除选中请求/文件夹", "Delete selected request/folder", "⌫"},
	{"关闭当前标签", "Close current tab", "⌘ + W"},
	{"搜索（响应体/编辑器内）", "Search (response/editor)", "⌘ + F"},
	{"上一个 / 下一个标签", "Previous / next tab", "⌃ + ⇧ + Tab / ⌃ + Tab"},
	{"环境管理", "Environments", "⌘ + ⌥ + E"},
	{"复制为 cURL", "Copy as cURL", "⌘ + ⇧ + C"},
	{"打开设置", "Open settings", "⌘ + ,"},
	{"切换侧边栏", "Toggle sidebar", "⌘ + B"},
}

// prefsView 绘制设置对话框（880×600 双栏）；任何修改立即生效并保存。
func (a *app) prefsView(c *ui.Context) {
	s := &a.overlays.prefs
	if !s.open {
		return
	}
	p := a.p
	changed := false
	a.dialog(c, &s.open, 880, 600, 14, func() {
		ui.Row(c).FillWidth().FillHeight().AlignItems(ui.Stretch).Children(func() {
			// 左栏：210px，次级背景。
			ui.Row(c).Width(210).Shrink(0).Background(p.surface2).AlignItems(ui.Stretch).Children(func() {
				ui.Column(c).Grow(1).Basis(0).Padding(12).Gap(4).Children(func() {
					txt(c, a.t("设置", "Settings"), fsSmall, p.muted, 600).PaddingX(8).Margin(0, 0, 4, 0)
					for i, sec := range prefsSections {
						active := s.section == i
						b := ui.ButtonBase(c.Key("sec" + strconv.Itoa(i))).FillWidth().Height(36).Radius(9).PaddingX(10).Gap(8).AlignItems(ui.Center).Justify(ui.Start)
						col := p.fg2
						switch {
						case active:
							b = b.Background(p.selected)
							col = p.fg
						case b.Hovered():
							b = b.Background(p.btnBg)
						}
						b.Children(func() {
							ui.Row(c).Grow(1).Gap(8).AlignItems(ui.Center).Children(func() {
								icon(c, sec.icon, 14, col)
								txt(c, a.t(sec.zh, sec.en), fsLarge, col, 0)
							})
						})
						if b.Clicked() {
							s.section = i
						}
					}
					ui.Spacer(c)
					txtMono(c, "MiniPost "+a.appVersion(), fsSmall, p.muted, 0).PaddingX(8)
				})
				vline(c, p.border)
			})
			// 右栏：44px 标题行 + 可滚动内容。
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				ui.Row(c).Height(44).FillWidth().Shrink(0).PaddingX(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
					txt(c, a.t(prefsSections[s.section].zh, prefsSections[s.section].en), 14, p.fg, 600)
					if a.iconButton(c, "clear", a.t("关闭", "Close"), 12, p.muted) {
						s.open = false
					}
				})
				hline(c, p.border)
				ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
					ui.Column(c).FillWidth().PaddingX(16).PaddingY(10).Gap(14).Children(func() {
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
				if s.section == 0 {
					hline(c, p.border)
					ui.Row(c).FillWidth().Height(44).Shrink(0).PaddingX(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
						txt(c, a.t("修改会立即生效并自动保存", "Changes apply immediately and are saved automatically"), fsSmall, p.muted, 0)
						bd := p.btnBorder
						if a.button(c, btnOpts{h: 26, radius: 7, padX: 10, border: &bd, key: "prefs-reset"}, func(bool) {
							txt(c, a.t("恢复默认", "Restore defaults"), fsSmall, p.fg2, 0)
						}) {
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

// section 是带 11px 圆角细边框的设置分组卡片。
func (a *app) settingsCard(c *ui.Context, padX float32, fn func()) {
	ui.Column(c).FillWidth().Radius(11).Border(1, a.p.border).PaddingX(padX).Clip().Children(fn)
}

// settingsRow 是一行设置：左侧标题与说明，右侧控件；last 为 true 时不画下边线。
func (a *app) settingsRow(c *ui.Context, key string, title, desc string, last bool, control func()) {
	p := a.p
	row := ui.Row(c.Key(key)).FillWidth().PaddingY(12).Gap(20).AlignItems(ui.Center)
	if !last {
		row = row.BorderWidth(0, 0, 1, 0).BorderColor(p.border)
	}
	row.Children(func() {
		ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(fsLarge).FontWeight(500).FixedLineHeight(fsLarge*1.5).TextColor(p.fg).Translate(0, textShiftY)
			ui.Text(c, desc).FontSize(fsSmall).FixedLineHeight(20).TextColor(p.fg2).Translate(0, textShiftY)
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
		e := ui.TextInputBase(c, v).Width(92).FillHeight().PaddingX(10).FontSize(fsSmall).TextColor(p.fg).TextAlign(ui.End)
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
	b := ui.ButtonBase(c.Key(key)).Width(124).Height(28).Radius(radiusInput).PaddingX(10).AlignItems(ui.Center).Border(1, p.btnBorder).Background(p.surface)
	b.PopoverMenuButton(func(m *ui.Menu) {
		for i, l := range labels {
			if m.Item(l).Checked(*sel == i).Chosen() {
				i := i
				a.later(func() { *sel = i })
				changed = true
			}
		}
	}).Children(func() {
		txt(c, labels[min(max(*sel, 0), len(labels)-1)], fsBase, p.fg, 0)
		ui.Spacer(c)
		icon(c, "arrowDown", 9, p.muted)
	})
	return changed
}

// prefsGroup 画设置分组：小标题 + 一组无外框、行间细线的设置行。
func (a *app) prefsGroup(c *ui.Context, title string, rows func()) {
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		txt(c, title, fsSmall, a.p.fg2, 600).Margin(2, 0, 4, 0)
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

// previewCard 是外观页的缩略图选项卡：上方迷你窗口，下方名称（可带右侧色点）。选中时 2px 强调色边框。
func (a *app) previewCard(c *ui.Context, key, label string, selected bool, preview func(), dots []ui.Color, labelLeft bool) bool {
	p := a.p
	b := ui.ButtonBase(c.Key(key)).Grow(1).Basis(0).Radius(10).Padding(2)
	bd := p.border
	if selected {
		bd = p.accent
	}
	if b.Hovered() && !selected {
		bd = p.btnBorder
	}
	b = b.Background(p.surface).Border(2, bd)
	b.Children(func() {
		ui.Column(c).FillWidth().Gap(0).Children(func() {
			ui.Row(c).FillWidth().Height(78).Radius(8).Clip().Border(1, p.border).Children(preview)
			ui.Row(c).FillWidth().Height(30).PaddingX(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				if !labelLeft {
					ui.Spacer(c)
				}
				txt(c, label, fsSmall, p.fg, 500)
				if !labelLeft {
					ui.Spacer(c)
				}
				if len(dots) > 0 {
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						for _, d := range dots {
							ui.Box(c).Size(9, 9).Radius(5).Background(d).Border(1, p.border)
						}
					})
				}
			})
		})
	})
	return b.Clicked()
}

// prefsSectionLabel 是外观页分组的小标题（11px 次要色），可带右侧“新增”小徽标。
func (a *app) prefsSectionLabel(c *ui.Context, s, badge string) {
	p := a.p
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(4, 0, 0, 0).Children(func() {
		txt(c, s, fsSmall, p.fg2, 500)
		if badge != "" {
			ui.Row(c).Height(16).PaddingX(7).Radius(8).Background(p.selected).AlignItems(ui.Center).Children(func() {
				txt(c, badge, 10, p.accent, 600)
			})
		}
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
	ui.Row(c.Key("theme")).FillWidth().Gap(12).Children(func() {
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
			if a.previewCard(c, "theme-"+v, names[i], set.Theme == v, preview, nil, false) {
				set.Theme = v
				changed = true
			}
		}
	})
	// 配色：雾蓝 / 墨白 / 松石
	a.prefsSectionLabel(c, a.t("配色", "Color scheme"), a.t("新增", "New"))
	ui.Row(c.Key("scheme")).FillWidth().Gap(12).Children(func() {
		names := map[string]string{"mist": a.t("雾蓝", "Mist"), "ink": a.t("墨白", "Ink"), "jade": a.t("松石", "Jade")}
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
	hline(c, p.border)

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
				b := ui.ButtonBase(c.Key("fs"+strconv.Itoa(f))).Size(30, 28).Radius(7).AlignItems(ui.Center).Justify(ui.Center)
				col, bd, w := p.fg2, p.btnBorder, 500
				switch {
				case active:
					col, bd, w = p.fg, p.accent, 600
				case b.Hovered():
					col = p.fg
				}
				bw := float32(1)
				if active {
					bw = 2
				}
				b = b.Background(p.surface).Border(bw, bd)
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
	a.settingsCard(c, 14, func() {
		a.settingsRow(c, "backup", a.t("自动备份", "Auto backup"), a.t("按固定间隔自动备份项目数据（请求、环境、历史记录等）。", "Automatically backs up project data (requests, environments, history, etc.) on interval."), false, func() {
			if a.toggle(c, "backup-sw", &set.AutoBackupEnabled) {
				changed = true
			}
		})
		a.settingsRow(c, "interval", a.t("备份间隔", "Backup interval"), a.t("单位分钟，范围 5 - 1440。", "In minutes, range 5-1440."), false, func() {
			if a.numberField(c, "interval-in", &s.backup, a.t("分钟", "min"), !set.AutoBackupEnabled) {
				set.AutoBackupIntervalMinutes = atoiOr(s.backup, 5)
				changed = true
			}
		})
		a.settingsRow(c, "now", a.t("立即备份 / 恢复", "Backup / Restore now"), a.t("备份文件默认保存到 ~/.minipost/backups；恢复会覆盖当前数据。", "Backups are saved to ~/.minipost/backups by default; restore will overwrite current data."), true, func() {
			ui.Row(c).Gap(8).Children(func() {
				bd := p.btnBorder
				if a.button(c, btnOpts{h: 28, radius: 7, padX: 10, border: &bd, key: "backup-now"}, func(bool) {
					txt(c, a.t("立即备份", "Backup now"), fsSmall, p.fg2, 0)
				}) {
					a.later(a.backupNow)
				}
				if a.button(c, btnOpts{h: 28, radius: 7, padX: 10, border: &bd, key: "backup-restore"}, func(bool) {
					txt(c, a.t("恢复备份", "Restore backup"), fsSmall, p.fg2, 0)
				}) {
					a.later(a.restoreBackup)
				}
			})
		})
	})
	return changed
}

func (a *app) prefsShortcutsView(c *ui.Context) {
	p := a.p
	ui.Column(c).FillWidth().Radius(11).Border(1, p.border).Clip().Children(func() {
		for i, sc := range prefsShortcuts {
			row := ui.Row(c.Key(sc.keys)).FillWidth().PaddingX(14).PaddingY(10).AlignItems(ui.Center).Justify(ui.SpaceBetween)
			if i > 0 {
				row = row.BorderWidth(1, 0, 0, 0).BorderColor(p.border)
			}
			row.Children(func() {
				txt(c, a.t(sc.zh, sc.en), fsBase, p.fg2, 0)
				ui.Row(c).Radius(6).Border(1, p.btnBorder).Background(p.surface2).PaddingX(8).PaddingY(2).Children(func() {
					txtMono(c, sc.keys, fsTiny, p.muted, 0)
				})
			})
		}
	})
}

func (a *app) prefsAbout(c *ui.Context) {
	p := a.p
	ui.Column(c).FillWidth().Radius(11).Border(1, p.border).Padding(16).Gap(12).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if img := logoImage(p.dark); img != nil {
				ui.Image(c, img).Size(36, 36).Fit(ui.Contain)
			}
			ui.Column(c).Children(func() {
				txt(c, "MiniPost", 14, p.fg, 600)
				txt(c, "Version "+a.appVersion(), fsSmall, p.muted, 0)
			})
		})
		ui.Text(c, a.t("MiniPost 是一个轻量 HTTP 请求调试工具，支持多项目、环境变量、历史记录、响应预览与 Cookie 管理。",
			"MiniPost is a lightweight HTTP debugging client with multi-project support, environment variables, history, response preview, and cookie management.")).
			FontSize(fsBase).FixedLineHeight(24).TextColor(p.fg2)
	})
	a.updateCard(c)
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
			a.overlays.askConfirm(a.t("从备份恢复？", "Restore from backup?"),
				a.t("当前数据会被备份内容替换。", "Current data will be replaced by the backup."), a.t("恢复", "Restore"), func() {
					if err := a.be.RestoreBackup(path); err != nil {
						a.fail(a.t("恢复备份", "Restore backup"), err)
						return
					}
					a.reloadAll()
					a.toastOK(a.t("已从备份恢复", "Restored from backup"))
				})
		}
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
