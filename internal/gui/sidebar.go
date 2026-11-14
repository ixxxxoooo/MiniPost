package gui

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"minipost/internal/model"
)

// sidebar 绘制左侧栏：顶部三个标签（请求 / 历史 / 环境）与对应内容，背景与分隔线对齐旧版。
func (a *app) sidebar(c *ui.Context) {
	p := a.p
	ui.Row(c).Fill().Children(func() {
		ui.Column(c).Grow(1).Basis(0).FillHeight().Background(p.sidebar).Children(func() {
			a.sideTabs(c)
			switch a.sideTab {
			case 1:
				a.historyView(c)
			case 2:
				a.envList(c)
			default:
				a.collectionView(c)
			}
		})
	})
}

// sideTabs 是 28px 高的标签条：三等分，当前项有 2px 强调色下划线。
func (a *app) sideTabs(c *ui.Context) {
	p := a.p
	items := []struct{ icon, zh, en string }{
		{"commandLine", "请求", "Requests"},
		{"clock", "历史", "History"},
		{"globe", "环境", "Environments"},
	}
	ui.Row(c).FillWidth().Height(sizeTabStrip - 1).Shrink(0).Children(func() {
		for i, it := range items {
			active := a.sideTab == i
			col, weight := p.muted, 0
			if active {
				col, weight = p.fg, 500
			}
			line := ui.Color{}
			if active {
				line = p.accent
			}
			b := ui.ButtonBase(c.Key("sidetab"+it.en)).Grow(1).Basis(0).Height(26).Gap(6).AlignItems(ui.Center).Justify(ui.Center).
				BorderWidth(0, 0, 2, 0).BorderColor(line)
			b.Children(func() {
				icon(c, it.icon, 12, col)
				txt(c, a.t(it.zh, it.en), fsBase, col, weight)
			})
			if b.Clicked() {
				a.sideTab = i
			}
		}
	})
	hline(c, p.border)
}

// searchRow 是请求标签顶部的搜索框与三个操作按钮（导入、新建文件夹、新建请求）。
func (a *app) searchRow(c *ui.Context) {
	p := a.p
	ui.Row(c).FillWidth().Shrink(0).Padding(7.5, 8, 6, 8).Gap(4).AlignItems(ui.Center).Children(func() {
		a.input(c, &a.sideQuery, a.t("搜索请求...", "Search requests..."), fieldOpts{icon: "search", radius: radiusInput, key: "side-search"})
		if a.iconButton(c, "fileImport", a.t("导入", "Import"), 14, p.fg2) {
			a.later(func() { a.openImport(importFile) })
		}
		if a.iconButton(c, "folderAdd", a.t("新建文件夹", "New folder"), 14, p.fg2) {
			a.later(func() { a.newFolder("") })
		}
		if a.iconButton(c, "add", a.t("新建请求", "New request")+" (⌘N)", 14, p.fg2) {
			a.later(func() { a.newRequest("") })
		}
	})
}

// rootMenu 是“新建”菜单，也是集合空白处的右键菜单。
func (a *app) rootMenu(m *ui.Menu) {
	if m.Item(a.t("添加请求", "Add request")).Chosen() {
		a.later(func() { a.newRequest("") })
	}
	if m.Item(a.t("添加文件夹", "Add folder")).Chosen() {
		a.later(func() { a.newFolder("") })
	}
}

// collectionView 绘制请求标签：搜索行与集合树。
func (a *app) collectionView(c *ui.Context) {
	p := a.p
	a.searchRow(c)
	q := strings.ToLower(strings.TrimSpace(a.sideQuery))
	ui.Scroll(c).Grow(1).Basis(0).ContextMenu(a.rootMenu).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Children(func() {
			if q == "" {
				a.treeRootRow(c)
			}
			switch {
			case len(a.tree.roots()) == 0:
				a.sideHint(c, a.t("暂无请求", "No requests"))
			case q != "":
				n := 0
				for _, id := range a.tree.descendants("")[1:] {
					nd := a.tree.nodes[id]
					if nd.folder || !a.matchNode(nd, q) {
						continue
					}
					n++
					a.requestRow(c, id, 0)
				}
				if n == 0 {
					a.sideHint(c, a.t("无匹配结果", "No results"))
				}
			default:
				for _, id := range a.tree.roots() {
					a.treeNode(c, id, 0)
				}
			}
		})
	})
	_ = p
}

// matchNode 判断请求的名称或地址是否包含搜索词。
func (a *app) matchNode(n treeNode, q string) bool {
	if strings.Contains(strings.ToLower(n.name), q) {
		return true
	}
	if it, ok := a.requestItem(n.id); ok {
		return strings.Contains(strings.ToLower(it.URL), q)
	}
	return false
}

// sideHint 是侧栏列表为空时居中的灰色提示。
func (a *app) sideHint(c *ui.Context, s string) {
	ui.Row(c).FillWidth().Padding(24, 0).Justify(ui.Center).Children(func() {
		txt(c, s, 10.4, a.p.muted, 0)
	})
}

// treeRootRow 是“根目录”行：打开状态的文件夹图标，悬停时出现新建请求/文件夹按钮。
func (a *app) treeRootRow(c *ui.Context) {
	p := a.p
	row := ui.Row(c.Key("tree-root")).Height(28).MarginX(4).PaddingX(8).Radius(radiusBtn).Gap(6).AlignItems(ui.Center).
		ContextMenu(a.rootMenu)
	hover := row.Hovered()
	if hover {
		row = row.Background(p.sideHover)
	}
	row.Children(func() {
		icon(c, "folderOpen", 14, p.muted)
		txt(c, a.t("根目录", "Root"), fsBase, p.fg, 500).Grow(1)
		if hover {
			if a.miniIcon(c, "add", a.t("在根目录新建请求", "New request in root")) {
				a.later(func() { a.newRequest("") })
			}
			if a.miniIcon(c, "folderAdd", a.t("在根目录新建文件夹", "New folder in root")) {
				a.later(func() { a.newFolder("") })
			}
		}
	})
}

// miniIcon 是树行右侧 12px 的小图标按钮，返回是否被点击。
func (a *app) miniIcon(c *ui.Context, name, tip string) bool {
	return a.button(c, btnOpts{w: 16, h: 20, tip: tip, noHover: true, key: "mi:" + name + tip}, func(hover bool) {
		col := a.p.muted
		if hover {
			col = a.p.fg
		}
		icon(c, name, 12, col)
	})
}

// treeNode 递归绘制一个节点；文件夹展开时绘制其子节点。
func (a *app) treeNode(c *ui.Context, id string, depth int) {
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	if !n.folder {
		a.requestRow(c, id, depth)
		return
	}
	p := a.p
	open := a.outline.Open.Has(id)
	row := ui.ButtonBase(c.Key("f:" + id)).Height(28).MarginX(4).PaddingStart(8 + float32(depth)*12).PaddingEnd(8).Radius(radiusBtn).Gap(4).
		AlignItems(ui.Center).Justify(ui.Start).ContextMenu(func(m *ui.Menu) { a.nodeMenu(m, id) })
	hover := row.Hovered()
	if hover {
		row = row.Background(p.sideHover)
	}
	row.Children(func() {
		chev := "arrowRight"
		if open {
			chev = "arrowDown"
		}
		icon(c, chev, 12, p.muted)
		fi := "folder"
		if open {
			fi = "folderOpen"
		}
		icon(c, fi, 14, p.muted)
		txtE(c, n.name, fsV2xs, p.fg, 0).PaddingStart(2).Grow(1)
		if hover {
			if a.miniIcon(c, "add", a.t("新建请求", "New request")) {
				a.later(func() { a.newRequest(id) })
			}
			a.nodeMoreButton(c, id)
		}
	})
	if row.Clicked() {
		a.later(func() {
			if open {
				a.outline.Open.Remove(id)
			} else {
				a.outline.Open.Add(id)
			}
		})
	}
	if open {
		for _, k := range a.tree.kids[id] {
			a.treeNode(c, k, depth+1)
		}
	}
}

// nodeMoreButton 是行右侧的“···”按钮，点击弹出节点菜单。
func (a *app) nodeMoreButton(c *ui.Context, id string) {
	p := a.p
	ui.ButtonBase(c.Key("more:" + id)).Width(16).Height(20).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).
		Tooltip(a.t("更多操作", "More actions")).
		PopoverMenuButton(func(m *ui.Menu) { a.nodeMenu(m, id) }).
		Children(func() { icon(c, "more", 12, p.muted) })
}

// requestRow 绘制一个请求行：方法标签（9px 粗体，最小宽 28）与名称；当前标签高亮。
func (a *app) requestRow(c *ui.Context, id string, depth int) {
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	p := a.p
	active := false
	if t := a.activeTab(); t != nil && t.draft.ID == id {
		active = true
	}
	row := ui.ButtonBase(c.Key("r:" + id)).Height(28).MarginX(4).PaddingStart(20 + float32(depth)*12).PaddingEnd(8).Radius(radiusBtn).Gap(4).
		AlignItems(ui.Center).Justify(ui.Start).ContextMenu(func(m *ui.Menu) { a.nodeMenu(m, id) })
	hover := row.Hovered()
	switch {
	case active:
		row = row.Background(p.sideActive)
	case hover:
		row = row.Background(p.sideHover)
	}
	row.Children(func() {
		txtMono(c, n.method, 9, p.method(n.method), 700).MinWidth(28).Shrink(0)
		col, w := p.fg, 0
		if active {
			col, w = p.accent, 500
		}
		txtE(c, n.name, fsV2xs, col, w).Grow(1)
		if hover {
			if a.miniIcon(c, "add", a.t("新建请求", "New request")) {
				a.later(func() { a.newRequest(n.parent) })
			}
			a.nodeMoreButton(c, id)
		}
	})
	if row.Clicked() {
		a.later(func() { a.openRequest(id) })
	}
}

// shortMethod 把方法名缩成适合窄列的短标签（兼容旧调用）。
func shortMethod(m string) string {
	switch m {
	case "DELETE":
		return "DEL"
	case "OPTIONS":
		return "OPT"
	}
	return m
}

func (a *app) nodeMenu(m *ui.Menu, id string) {
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	if n.folder {
		if m.Item(a.t("添加请求", "Add request")).Chosen() {
			a.later(func() { a.newRequest(id) })
		}
		if m.Item(a.t("添加文件夹", "Add folder")).Chosen() {
			a.later(func() { a.newFolder(id) })
		}
		m.Separator()
	}
	if m.Item(a.t("重命名", "Rename")).Chosen() {
		a.later(func() { a.renameNode(id) })
	}
	if m.Item(a.t("复制", "Duplicate")).Chosen() {
		a.later(func() { a.duplicateNode(id) })
	}
	m.Submenu(a.t("移动到", "Move to"), func(sm *ui.Menu) {
		if sm.Item(a.t("项目根目录", "Project root")).Disabled(n.parent == "").Chosen() {
			a.later(func() { a.moveNode(id, "") })
		}
		for _, f := range a.tree.folders() {
			if n.folder && a.tree.isInside(f.id, id) {
				continue
			}
			label := fmt.Sprintf("%*s%s", f.depth*2, "", f.name)
			if sm.Item(label).Disabled(f.id == n.parent).Chosen() {
				target := f.id
				a.later(func() { a.moveNode(id, target) })
			}
		}
	})
	m.Separator()
	if m.Item(a.t("删除", "Delete")).Danger().Chosen() {
		a.later(func() { a.deleteNode(id) })
	}
}

// ---- 历史 ----

// historyView 绘制历史标签：标题行（刷新/清空）与按天分组的记录。
func (a *app) historyView(c *ui.Context) {
	p := a.p
	ui.Row(c).FillWidth().Shrink(0).Padding(6, 12, 4, 12).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			icon(c, "clock", 12, p.muted)
			txt(c, a.t("历史", "History"), 10.4, p.muted, 500)
			if len(a.history) > 0 {
				txt(c, fmt.Sprintf("(%d)", len(a.history)), 10.4, p.muted, 500)
			}
		})
		ui.Spacer(c)
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			if a.textIconButton(c, "arrowUpDown", a.t("刷新", "Refresh"), p.muted, p.fg) {
				a.later(a.reloadHistory)
			}
			if len(a.history) > 0 && a.textIconButton(c, "delete", a.t("清空", "Clear"), p.muted, p.danger) {
				a.later(func() {
					a.overlays.askConfirm(a.t("清空历史记录？", "Clear history?"), a.t("此操作无法撤销。", "This can't be undone."), a.t("清空", "Clear"), func() {
						if err := a.be.ClearHistory(a.projectID); err != nil {
							a.fail(a.t("清空历史", "Clear history"), err)
							return
						}
						a.reloadHistory()
					})
				})
			}
		})
	})
	if len(a.history) == 0 {
		a.sideHint(c, a.t("暂无历史记录", "No history yet"))
		return
	}
	groups, order := a.groupHistory()
	ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Children(func() {
			for _, label := range order {
				entries := groups[label]
				collapsed := a.histCollapsed[label]
				head := ui.ButtonBase(c.Key("hg:" + label)).FillWidth().Height(23).PaddingX(12).Gap(4).AlignItems(ui.Center).Justify(ui.Start)
				if head.Hovered() {
					head = head.Background(p.sideHover)
				}
				head.Children(func() {
					chev := "arrowDown"
					if collapsed {
						chev = "arrowRight"
					}
					icon(c, chev, 8, p.muted)
					txt(c, label, fsTiny, p.fg2, 500)
					txt(c, fmt.Sprintf("(%d)", len(entries)), fsTiny, p.muted, 0)
				})
				if head.Clicked() {
					a.later(func() {
						if a.histCollapsed == nil {
							a.histCollapsed = map[string]bool{}
						}
						a.histCollapsed[label] = !collapsed
					})
				}
				if collapsed {
					continue
				}
				for _, e := range entries {
					a.historyRow(c, e)
				}
			}
		})
	})
}

// textIconButton 是历史标题行里带图标的小文字按钮（20px 高，圆角 4）。
func (a *app) textIconButton(c *ui.Context, iconName, label string, col, hoverCol ui.Color) bool {
	p := a.p
	return a.button(c, btnOpts{h: 20, radius: 4, padX: 6, gap: 2, hoverBg: &p.sideHover, key: "tib:" + label}, func(hover bool) {
		cc := col
		if hover {
			cc = hoverCol
		}
		icon(c, iconName, 10, cc)
		txt(c, label, 10.4, cc, 0)
	})
}

// historyRow 绘制一条历史记录：方法、地址、状态码与时间。
func (a *app) historyRow(c *ui.Context, e historyItem) {
	p := a.p
	row := ui.ButtonBase(c.Key("h:" + e.ID)).FillWidth().Height(24).MarginX(4).PaddingStart(24).PaddingEnd(12).Radius(radiusBtn).AlignItems(ui.Center).Justify(ui.Start)
	if row.Hovered() {
		row = row.Background(p.sideHover)
	}
	row.Children(func() {
		method := e.Method
		if method == "" {
			method = "GET"
		}
		ui.Row(c).Width(40).Shrink(0).Justify(ui.End).Children(func() {
			txtMono(c, method, 9, p.method(method), 700)
		})
		ui.Box(c).Width(6).Shrink(0)
		u := e.URL
		if u == "" {
			u = e.Name
		}
		txtE(c, u, fsSmall, p.fg, 0).Grow(1).Basis(0).MinWidth(0)
		ui.Box(c).Width(4).Shrink(0)
		txtMono(c, fmt.Sprint(e.StatusCode), fsTiny, a.statusTextColor(e.StatusCode), 0)
		ui.Box(c).Width(6).Shrink(0)
		txt(c, historyClock(e.Timestamp), fsTiny, p.muted, 0)
	})
	if row.Clicked() {
		a.later(func() { a.openFromHistory(e) })
	}
}

// statusTextColor 是历史/响应中状态码文字色：2xx 绿，4xx/5xx 红，其余橙。
func (a *app) statusTextColor(code int) ui.Color {
	switch {
	case code >= 200 && code < 300:
		return a.p.success
	case code >= 400:
		return a.p.danger
	}
	return a.p.warning
}

// ---- 环境 ----

// envList 绘制环境标签：标题行（新建按钮）与环境列表，圆点表示当前激活的环境。
func (a *app) envList(c *ui.Context) {
	p := a.p
	ui.Row(c).FillWidth().Shrink(0).Padding(6, 12, 4, 12).AlignItems(ui.Center).Children(func() {
		txt(c, a.t("环境变量", "Environment variables"), 10.4, p.muted, 500)
		ui.Spacer(c)
		if a.button(c, btnOpts{w: 20, h: 20, radius: 4, hoverBg: &p.sideHover, tip: a.t("新建环境", "New environment"), key: "env-add"}, func(bool) {
			icon(c, "add", 12, p.muted)
		}) {
			a.later(a.newEnvironment)
		}
	})
	if len(a.envs) == 0 {
		a.sideHint(c, a.t("暂无环境，点击 + 创建", "No environments yet. Click + to create one."))
		return
	}
	ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Children(func() {
			for _, e := range a.envs {
				e := e
				active := e.ID == a.envID
				row := ui.ButtonBase(c.Key("e:" + e.ID)).FillWidth().Height(24).MarginX(4).PaddingX(12).Radius(radiusBtn).AlignItems(ui.Center).Justify(ui.Start)
				hover := row.Hovered()
				if hover {
					row = row.Background(p.sideHover)
				}
				row.Children(func() {
					// 圆点：激活时为强调色实心并带白色内点
					dot := ui.Row(c.Key("dot:"+e.ID)).Size(12, 12).Radius(6).Shrink(0).AlignItems(ui.Center).Justify(ui.Center)
					if active {
						dot = dot.Background(p.accent).Border(1, p.accent)
					} else {
						dot = dot.Border(1, p.border)
					}
					dot.Children(func() {
						if active {
							ui.Box(c).Size(6, 6).Radius(3).Background(ui.RGB(255, 255, 255))
						}
					})
					if dot.Clicked() {
						a.later(func() {
							if active {
								a.setEnv("")
							} else {
								a.setEnv(e.ID)
							}
						})
					}
					ui.Box(c).Width(8).Shrink(0)
					txtE(c, e.Name, fsV2xs, p.fg, 0).Grow(1)
					if hover {
						if a.miniIcon(c, "delete", a.t("删除环境", "Delete environment")) {
							a.later(func() { a.deleteEnvironment(e.ID, e.Name) })
						}
					}
				})
				if row.Clicked() {
					a.later(func() { a.openEnvManagerFor(e.ID) })
				}
			}
		})
	})
}

// historyItem 是历史记录的别名，避免在多个文件重复引用 model 包。
type historyItem = model.HistoryEntry

// groupHistory 按“今天/昨天/日期”对历史分组，保持原有顺序。
func (a *app) groupHistory() (map[string][]historyItem, []string) {
	groups := map[string][]historyItem{}
	var order []string
	for _, e := range a.history {
		key := historyDay(e.Timestamp, a)
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], e)
	}
	return groups, order
}

func historyDay(ts string, a *app) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	t = t.Local()
	now := time.Now()
	day := func(x time.Time) time.Time { return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, x.Location()) }
	switch {
	case day(t).Equal(day(now)):
		return a.t("今天", "Today")
	case day(t).Equal(day(now).AddDate(0, 0, -1)):
		return a.t("昨天", "Yesterday")
	}
	if a.locale() == localeZH {
		return fmt.Sprintf("%d月%d日%s", int(t.Month()), t.Day(), [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}[t.Weekday()])
	}
	return t.Format("Monday, January 2")
}

// historyClock 把时间戳格式化为 24 小时制 HH:mm。
func historyClock(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Local().Format("15:04")
}

// formatTimestamp 把 RFC3339 时间格式化为本地短时间，解析失败时原样返回。
func formatTimestamp(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Local().Format("01-02 15:04:05")
}

// openFromHistory 点击历史记录：请求仍存在则打开，否则提示。
func (a *app) openFromHistory(e historyItem) {
	if _, ok := a.requestItem(e.RequestID); ok {
		a.openRequest(e.RequestID)
		return
	}
	a.toast(a.t("该请求已被删除", "This request no longer exists"))
}
