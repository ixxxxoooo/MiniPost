package gui

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"

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
	fontSizes    = []int{10, 11, 12, 13, 14, 16}
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
	{"复制选中请求/文件夹", "Copy selected request/folder", "⌘ + C"},
	{"Duplicate 选中请求/文件夹", "Duplicate selected request/folder", "⌘ + D"},
	{"删除选中请求/文件夹", "Delete selected request/folder", "⌫"},
	{"关闭当前标签", "Close current tab", "⌘ + W"},
	{"搜索（响应体/编辑器内）", "Search (response/editor)", "⌘ + F"},
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
				ui.Scroll(c).Grow(1).Basis(0).Children(func() {
					ui.Column(c).FillWidth().PaddingX(16).PaddingY(10).Gap(12).Children(func() {
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
			ui.Text(c, title).FontSize(fsLarge).FontWeight(500).TextColor(p.fg)
			ui.Text(c, desc).FontSize(fsSmall).FixedLineHeight(20).TextColor(p.fg2)
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

func (a *app) prefsGeneral(c *ui.Context, s *prefsState) bool {
	changed := false
	set := &a.set
	a.settingsCard(c, 14, func() {
		a.settingsRow(c, "http", a.t("HTTP 版本", "HTTP version"), a.t("设置发送请求时优先使用的 HTTP 版本。", "Preferred HTTP version when sending requests."), false, func() {
			i := indexOf(httpValues, set.HTTPVersion)
			if a.segmented(c, "http-seg", &i, []string{"Auto", "HTTP/1.1", "HTTP/2"}) {
				set.HTTPVersion = httpValues[i]
				changed = true
			}
		})
		a.settingsRow(c, "timeout", a.t("请求超时", "Request timeout"), a.t("单位毫秒，设置为 0 表示不主动超时。", "In milliseconds. Set 0 to disable active timeout."), false, func() {
			if a.numberField(c, "timeout-in", &s.timeout, "ms", false) {
				set.RequestTimeoutMs = atoiOr(s.timeout, 0)
				changed = true
			}
		})
		a.settingsRow(c, "maxsize", a.t("最大响应大小", "Max response size"), a.t("单位 MB，设置为 0 表示不限制。", "In MB. Set 0 for unlimited."), false, func() {
			if a.numberField(c, "maxsize-in", &s.maxSize, "MB", false) {
				set.MaxResponseSizeMB = atoiOr(s.maxSize, 0)
				changed = true
			}
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
		sw("ssl", &set.SSLVerify, a.t("SSL 证书校验", "SSL certificate verification"), a.t("关闭后将跳过证书校验（仅建议在调试环境使用）。", "If disabled, certificate verification is skipped (recommended for debugging only)."), false, nil)
		sw("keylog", &set.SSLTLSKeyLog, "SSL/TLS Key Log", a.t("记录 TLS 会话密钥用于抓包调试。", "Log TLS session keys for packet-capture debugging."), false, nil)
		sw("nocookie", &set.DisableCookies, a.t("禁用 Cookies", "Disable cookies"), a.t("关闭 Cookie Jar 读写，不自动注入或保存 Cookie。", "Disable Cookie Jar read/write, and stop auto injection/persistence."), false, nil)
		sw("nocache", &set.SendNoCacheHeader, a.t("发送 no-cache 请求头", "Send no-cache header"), a.t("参考 Postman「Send no-cache header」：自动追加 Cache-Control: no-cache，强制服务端重新校验缓存。", "Similar to Postman \"Send no-cache header\": adds Cache-Control: no-cache to force revalidation."), false, nil)
		sw("token", &set.SendPostmanTokenHeader, a.t("发送 MiniPost-Token 请求头", "Send MiniPost-Token header"), a.t("参考 Postman「Send Postman Token header」：开启后每次请求自动追加随机 Token（名称为 MiniPost-Token），便于区分请求并辅助排查（默认关闭）。", "Similar to Postman \"Send Postman Token header\": appends a random MiniPost-Token to each request for tracing (off by default)."), false, nil)
		sw("redirect", &set.FollowRedirects, a.t("自动跟随重定向（301/302/307/308）", "Automatically follow redirects (301/302/307/308)"), a.t("对应 Postman：Settings > General > Request > Automatically follow redirects。", "Equivalent to Postman: Settings > General > Request > Automatically follow redirects."), false, nil)
		a.settingsRow(c, "detect", a.t("响应格式检测", "Response format detection"), a.t("Auto 按 Content-Type 自动识别；JSON 会优先用 JSON 视图。", "Auto detects by Content-Type; JSON prefers JSON view."), false, func() {
			i := indexOf(detectValues, set.ResponseFormatDetection)
			if a.segmented(c, "detect-seg", &i, []string{"Auto", "JSON"}) {
				set.ResponseFormatDetection = detectValues[i]
				changed = true
			}
		})
		sw("save", &set.AlwaysSaveUnsaved, a.t("关闭标签时默认保存未保存修改", "Save unsaved changes by default when closing tabs"), a.t("开启后，关闭标签页会自动保存再关闭。开启它会自动关闭“默认丢弃未保存修改”。", "When enabled, closing a tab saves before closing. It turns off default discard automatically."), false, func() {
			if set.AlwaysSaveUnsaved {
				set.AlwaysDiscardUnsaved = false
			}
		})
		sw("discard", &set.AlwaysDiscardUnsaved, a.t("关闭标签时默认丢弃未保存修改", "Discard unsaved changes by default when closing tabs"), a.t("开启后，关闭标签页将不再询问“是否保存更改”，会直接关闭并丢弃未保存内容。你可随时在此处改回。", "When enabled, closing a tab won't ask to save changes. Unsaved changes will be discarded directly."), true, func() {
			if set.AlwaysDiscardUnsaved {
				set.AlwaysSaveUnsaved = false
			}
		})
	})
	return changed
}

// optionCard 是外观页的一张卡片：标题 + 一排等宽选项按钮 + 可选的底部说明。
func (a *app) optionCard(c *ui.Context, key, title, note string, labels []string, cols int, h, radius float32, mono bool, selected int, onPick func(i int)) {
	p := a.p
	ui.Column(c.Key(key)).FillWidth().Radius(11).Border(1, p.border).Padding(14).Gap(8).Children(func() {
		txt(c, title, fsLarge, p.fg, 500)
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			for i, l := range labels {
				active := selected == i
				b := ui.ButtonBase(c.Key(l)).Grow(1).Basis(0).Height(h).Radius(radius).AlignItems(ui.Center).Justify(ui.Center)
				col, bg, bd := p.fg2, p.surface, p.btnBorder
				switch {
				case active:
					col, bd = p.accent, p.accent
					bg = ui.RGBA(p.accent.R, p.accent.G, p.accent.B, 0.12)
				case b.Hovered():
					bg = p.btnBg
				}
				b = b.Background(bg).Border(1, bd)
				b.Children(func() {
					t := txt(c, l, fsSmall, col, 500)
					if mono {
						t.Font(monoFont)
					}
				})
				if b.Clicked() {
					onPick(i)
				}
			}
		})
		if note != "" {
			txt(c, note, fsSmall, p.muted, 0)
		}
	})
}

func (a *app) prefsAppearance(c *ui.Context, s *prefsState) bool {
	changed := false
	set := &a.set
	a.optionCard(c, "theme", a.t("外观主题", "Theme"), "", []string{a.t("浅色", "Light"), a.t("深色", "Dark"), a.t("跟随系统", "System")}, 3, 34, 8, false, indexOf(themeValues, set.Theme), func(i int) {
		set.Theme = themeValues[i]
		changed = true
	})
	cur := indexOf(localeValues, set.Locale)
	if set.Locale == "" {
		cur = 0
		if a.locale() == localeEN {
			cur = 1
		}
	}
	a.optionCard(c, "lang", a.t("语言", "Language"), a.t("立即生效，且会记住你的选择。", "Takes effect immediately and will be remembered."), []string{"中文", "English"}, 2, 34, 8, false, cur, func(i int) {
		set.Locale = localeValues[i]
		changed = true
	})
	labels := make([]string, len(fontSizes))
	sel := -1
	for i, f := range fontSizes {
		labels[i] = strconv.Itoa(f) + "px"
		if f == set.FontSize {
			sel = i
		}
	}
	a.optionCard(c, "font", a.t("界面字体", "UI font size"), a.t("用于调整界面字号（菜单、标签、按钮与面板文本）。", "Adjusts text size for menus, tabs, buttons, and panels."), labels, 6, 28, 7, true, sel, func(i int) {
		set.FontSize = fontSizes[i]
		changed = true
	})
	a.settingsCard(c, 14, func() {
		a.settingsRow(c, "scroll", a.t("滚动条自动隐藏", "Auto-hide scrollbar"), a.t("滚动时显示，静止后自动隐藏。", "Shows while scrolling and hides when idle."), true, func() {
			if a.toggle(c, "scroll-sw", &set.ScrollbarAutoHide) {
				changed = true
			}
		})
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
			if img := logoImage(); img != nil {
				ui.Image(c, img).Size(36, 36).Fit(ui.Contain)
			}
			ui.Column(c).Children(func() {
				txt(c, "MiniPost", 14, p.fg, 600)
				txt(c, "Version 1.0.0", fsSmall, p.muted, 0)
			})
		})
		ui.Text(c, a.t("MiniPost 是一个轻量 HTTP 请求调试工具，支持多项目、环境变量、历史记录、响应预览与 Cookie 管理。",
			"MiniPost is a lightweight HTTP debugging client with multi-project support, environment variables, history, response preview, and cookie management.")).
			FontSize(fsBase).FixedLineHeight(24).TextColor(p.fg2)
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
			a.toast(a.t("备份已创建：", "Backup created: ") + path)
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
					a.toast(a.t("已从备份恢复", "Restored from backup"))
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
	if a.backupStop != nil {
		close(a.backupStop)
		a.backupStop = nil
	}
	if !a.set.AutoBackupEnabled || !a.native {
		return
	}
	stop := make(chan struct{})
	a.backupStop = stop
	interval := time.Duration(a.set.AutoBackupIntervalMinutes) * time.Minute
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if path, err := a.be.CreateBackup(); err != nil {
					logger.Warn("自动备份失败", "error", err.Error())
				} else {
					logger.Info("自动备份完成", "path", path)
				}
			}
		}
	}()
}
