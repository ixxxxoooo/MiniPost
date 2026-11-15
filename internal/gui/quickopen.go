package gui

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
)

// 快速打开（⌘K）：一个输入框同时搜索请求、环境与命令，↑↓ 选择、回车执行（设计稿“新增交互”）。

// quickState 是快速打开面板的状态。
type quickState struct {
	open  bool
	query string
	sel   int  // 当前高亮行（在扁平化结果里的下标）
	moved bool // 刚用 ↑↓ 移动过高亮：下一帧把高亮行滚入可视区
}

// 各分组最多显示的条数，保证面板不会超出窗口高度。
// quickPanelH 是面板最大高度（设计稿 .qk 的 max-height:440）；结果多时列表区滚动。
const quickPanelH = 440

const (
	quickMaxRequests = 7
	quickMaxEnvs     = 3
)

// quickKind 是结果所属分组。
type quickKind int

const (
	quickRequest quickKind = iota
	quickEnv
	quickCommand
)

// quickItem 是一条可执行的结果。
type quickItem struct {
	kind    quickKind
	method  string // 仅请求：方法（显示缩写）
	label   string
	hint    string // 次要说明，如请求所在文件夹
	accel   string // 命令的快捷键提示
	current bool   // 仅环境：是当前环境，右侧显示“当前”标
	run     func()
}

// openQuick 打开快速打开面板；已在项目工作区之外或已有对话框时不响应。
func (a *app) openQuick() {
	if a.projectID == "" || a.showHome || (a.modalOpen() && !a.overlays.quick.open) {
		return
	}
	a.overlays.quick = quickState{open: true}
	logger.Debug("打开快速打开面板")
}

// quickGroupTitle 返回分组标题。
func (a *app) quickGroupTitle(k quickKind) string {
	switch k {
	case quickEnv:
		return a.t("环境", "Environments")
	case quickCommand:
		return a.t("命令", "Commands")
	}
	return a.t("请求", "Requests")
}

// quickResults 按搜索词收集结果：请求（名称/地址/方法命中）、环境（名称命中）、命令（名称命中）。
// 搜索词为空时请求按侧栏顺序列出前几条，环境与命令全部列出。
func (a *app) quickResults(q string) []quickItem {
	q = strings.ToLower(strings.TrimSpace(q))
	var out []quickItem

	n := 0
	for _, id := range a.tree.descendants("")[1:] {
		nd := a.tree.nodes[id]
		if nd.folder || (q != "" && !a.matchNode(nd, q)) {
			continue
		}
		if n++; n > quickMaxRequests {
			break
		}
		id := id
		// 设计稿：说明只写所在文件夹，根目录下的请求写“根目录”
		hint := a.t("根目录", "Root")
		if pn, ok := a.tree.nodes[nd.parent]; ok && pn.name != "" {
			hint = pn.name
		}
		out = append(out, quickItem{kind: quickRequest, method: nd.method, label: nd.name, hint: hint, run: func() {
			a.showHome = false
			a.openRequest(id)
		}})
	}

	n = 0
	for i := range a.envs {
		e := a.envs[i]
		label := e.Name // 设计稿：行名是环境名，说明是“切换为当前环境”，当前环境右侧有“当前”标
		if q != "" && !strings.Contains(strings.ToLower(label), q) {
			continue
		}
		if n++; n > quickMaxEnvs {
			break
		}
		out = append(out, quickItem{kind: quickEnv, label: label, hint: a.t("切换为当前环境", "Make current environment"), current: a.envID == e.ID, run: func() {
			a.setEnv(e.ID)
			a.toastOK(a.t("已切换到环境「"+e.Name+"」", "Switched to environment “"+e.Name+"”"))
		}})
	}

	for _, cmd := range a.quickCommands() {
		if q != "" && !strings.Contains(strings.ToLower(cmd.label), q) {
			continue
		}
		out = append(out, cmd)
	}
	return out
}

// quickCommands 是可从面板执行的命令。
func (a *app) quickCommands() []quickItem {
	cmds := []quickItem{
		{kind: quickCommand, label: a.t("新建请求", "New request"), accel: "⌘ N", run: func() { a.newRequest("") }},
		{kind: quickCommand, label: a.t("导入 cURL", "Import cURL"), accel: "⌘ I", run: a.openCurlDialog},
	}
	if t := a.activeTab(); t != nil && t.env == nil {
		cmds = append(cmds, quickItem{kind: quickCommand, label: a.t("复制为 cURL", "Copy as cURL"), accel: "⌘ ⇧ C", run: func() { a.copyCurl(t) }})
	}
	return append(cmds,
		quickItem{kind: quickCommand, label: a.t("切换明暗主题", "Toggle light / dark"), run: a.toggleThemeMode},
		quickItem{kind: quickCommand, label: a.t("显示 / 隐藏侧栏", "Toggle sidebar"), accel: "⌘ B", run: a.toggleSidebar},
		quickItem{kind: quickCommand, label: a.t("显示 / 隐藏控制台", "Toggle console"), accel: "⌘ J", run: func() { a.setConsoleOpen(!a.set.ConsoleOpen) }},
		quickItem{kind: quickCommand, label: a.t("环境管理", "Manage environments"), accel: "⌘ ⌥ E", run: a.openEnvManager},
		quickItem{kind: quickCommand, label: "Cookies", accel: "⌘ ⇧ K", run: a.openCookies},
		quickItem{kind: quickCommand, label: a.t("设置", "Settings"), accel: "⌘ ,", run: a.openPrefs},
	)
}

// quickView 绘制快速打开面板：顶部搜索框、分组结果、底部键位提示。
func (a *app) quickView(c *ui.Context) {
	qs := &a.overlays.quick
	if !qs.open {
		return
	}
	p := a.p
	items := a.quickResults(qs.query)
	qs.sel = max(0, min(qs.sel, len(items)-1))

	run := -1
	a.dialogAt(c, &qs.open, 580, 0, radiusLg, 0.35, 70, func() {
		ui.Column(c).FillWidth().MaxHeight(quickPanelH).Children(func() {
			// 搜索行
			ui.Row(c).FillWidth().Height(44).PaddingX(14).Gap(10).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
				icon(c, "search", 14, p.accent)
				in := ui.TextInputBase(c.Key("quick-in"), &qs.query).Placeholder(a.t("搜索请求、环境、命令…", "Search requests, environments, commands…")).PlaceholderColor(p.muted).
					FontSize(14).TextColor(p.fg).CaretColor(p.accent).Grow(1).Basis(0).FillHeight().AutoFocus()
				switch {
				case in.Changed():
					qs.sel = 0
				case in.Shortcut(0, ui.KeyDown):
					if len(items) > 0 {
						qs.sel, qs.moved = (qs.sel+1)%len(items), true
					}
				case in.Shortcut(0, ui.KeyUp):
					if len(items) > 0 {
						qs.sel, qs.moved = (qs.sel+len(items)-1)%len(items), true
					}
				case in.Submitted(), in.Shortcut(ui.Cmd, ui.KeyEnter):
					run = qs.sel
				}
				a.kbdChip(c, "esc")
			})
			// 结果
			ui.Scroll(c.Key("quick-list")).FillWidth().Shrink(1).MaxHeight(quickPanelH - 44 - 30 - 2).Scrollbars(a.scrollbarsVisible()).Children(func() {
				ui.Column(c).FillWidth().Padding(6).Gap(1).Children(func() {
					if len(items) == 0 {
						ui.Row(c).FillWidth().Padding(18, 0).Justify(ui.Center).Children(func() {
							txt(c, a.t("没有匹配的结果", "No results"), fsBase, p.muted, 0)
						})
						return
					}
					last := quickKind(-1)
					for i, it := range items {
						if it.kind != last {
							last = it.kind
							ui.Row(c.Key("qg"+itoa(int(it.kind)))).FillWidth().Height(26).Padding(0, 8, 3, 8).AlignItems(ui.End).Children(func() {
								txt(c, a.quickGroupTitle(it.kind), 10.5, p.muted, 650)
							})
						}
						if a.quickRow(c, i, it, i == qs.sel, qs.moved) {
							run = i
						}
					}
					qs.moved = false
				})
			})
			// 键位提示
			ui.Row(c).FillWidth().Height(30).PaddingX(14).Gap(14).AlignItems(ui.Center).Background(p.surface2.Alpha(0.4)).BorderWidth(1, 0, 0, 0).BorderColor(p.border).Children(func() {
				for _, h := range [][2]string{{"↑↓", a.t("选择", "Select")}, {"↩", a.t("打开", "Open")}, {"esc", a.t("关闭", "Close")}} {
					ui.Row(c.Key("qh" + h[0])).Gap(5).AlignItems(ui.Center).Children(func() {
						a.kbdChip(c, h[0])
						txt(c, h[1], fsSmall, p.muted, 0)
					})
				}
				ui.Spacer(c)
				txt(c, a.t(itoa(len(items))+" 个结果", itoa(len(items))+" results"), fsSmall, p.muted, 0)
			})
		})
	})
	if run >= 0 && run < len(items) {
		it := items[run]
		qs.open = false
		logger.Info("快速打开执行", "kind", int(it.kind), "label", it.label)
		a.later(it.run)
	}
}

// quickRow 绘制一条结果，返回是否被点击。
func (a *app) quickRow(c *ui.Context, i int, it quickItem, selected, reveal bool) bool {
	p := a.p
	row := ui.ButtonBase(c.Key("qr" + itoa(i))).FillWidth().Height(32).PaddingX(10).Gap(10).Radius(radiusR).AlignItems(ui.Center).Justify(ui.Start)
	switch {
	case selected:
		row = row.Background(p.acSoft).InsetShadow(2, 0, 0, 0, p.accent)
	case row.Hovered():
		row = row.Background(p.hover)
	}
	if selected && reveal {
		row.ScrollIntoView()
	}
	row.Children(func() {
		// 左侧 20px 图标块（设计稿 .qk-r .ico）：选中时为淡强调色底
		tile := func(draw func(col ui.Color)) {
			bg, col := p.surface2, p.fg2
			if selected {
				bg, col = p.acSoft2, p.accent
			}
			ui.Row(c).Size(20, 20).Radius(6).Background(bg).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() { draw(col) })
		}
		switch it.kind {
		case quickRequest:
			ui.Row(c).Width(34).Shrink(0).Children(func() { a.methodBadge(c, it.method, 0) })
		case quickEnv:
			tile(func(col ui.Color) { icon(c, "globe", 12, col) })
		default:
			tile(func(col ui.Color) { icon(c, "commandLine", 12, col) })
		}
		nameCol, nameW := p.fg, 0
		if selected {
			nameCol, nameW = p.accent, 600
		}
		txtE(c, it.label, fsLarge, nameCol, nameW).Shrink(0).MaxWidth(300)
		if it.hint != "" {
			txtE(c, it.hint, fsSmall, p.muted, 0).Grow(1)
		} else {
			ui.Spacer(c)
		}
		switch {
		case it.current:
			a.curBadge(c)
		case it.accel != "":
			a.kbdChip(c, it.accel)
		case selected:
			a.kbdChip(c, "↩")
		}
	})
	return row.Clicked()
}
