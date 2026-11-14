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
			switch a.side.tab {
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

// sideTabs 是 28px 高的标签条：三等分，样式与请求标签一致——方正、选中项只有顶部 2px 强调线、
// 下边不画线并与下方列表连成一体；未选中项底色稍深。顶线由标签条自己画（被选中项的强调线盖住）。
func (a *app) sideTabs(c *ui.Context) {
	p := a.p
	items := []struct{ icon, zh, en string }{
		{"commandLine", "请求", "Requests"},
		{"clock", "历史", "History"},
		{"globe", "环境", "Environments"},
	}
	ui.Row(c).FillWidth().Height(sizeTabStrip).Shrink(0).AlignItems(ui.End).Children(func() {
		stripTopLine(c, p.border)
		for i, it := range items {
			active := a.side.tab == i
			col, weight := p.muted, 0
			if active {
				col, weight = p.fg, 500
			}
			h := float32(sizeTabStrip - 1)
			if active {
				h = sizeTabStrip
			}
			b := ui.ButtonBase(c.Key("sidetab" + it.en)).Grow(1).Basis(0).Height(h).Gap(6).AlignItems(ui.Center).Justify(ui.Center)
			switch {
			case active:
				b = b.Background(p.sidebar).BorderWidth(tabAccentWidth, 0, 0, 0).BorderColor(p.tabActiveBorder)
			case b.Hovered():
				b = b.Background(p.hover)
			default:
				b = b.Background(p.sideHover)
			}
			b.Children(func() {
				icon(c, it.icon, 12, col)
				txt(c, a.t(it.zh, it.en), fsBase, col, weight)
			})
			if b.Clicked() {
				a.side.tab = i
			}
			// 标签之间的竖分隔线。
			if i < len(items)-1 {
				ui.Box(c.Key("sidediv" + it.en)).Width(1).Height(sizeTabStrip - 1).Shrink(0).Background(p.tabDivider)
			}
		}
	})
}

// searchRow 是请求标签顶部的搜索框与三个操作按钮（导入、新建文件夹、新建请求）。
func (a *app) searchRow(c *ui.Context) {
	p := a.p
	ui.Row(c).FillWidth().Shrink(0).Padding(7.5, 8, 6, 8).Gap(4).AlignItems(ui.Center).Children(func() {
		a.input(c, &a.side.query, a.t("搜索请求或内容...", "Search requests..."), fieldOpts{icon: "search", radius: radiusInput, key: "side-search"})
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
	q := strings.ToLower(strings.TrimSpace(a.side.query))
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

// matchNode 判断请求的名称、地址、方法、参数、请求头或请求体是否包含搜索词。
func (a *app) matchNode(n treeNode, q string) bool {
	return a.searchHit(n, q) != hitNone
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
	if d, ok := ui.Drop[nodeDrag](row); ok {
		a.later(func() { a.dropOnRoot(d.id) })
	}
	_, over := ui.DragOver[nodeDrag](row)
	hover := row.Hovered()
	if hover || over {
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
	open := a.side.outline.Open.Has(id)
	row := ui.ButtonBase(c.Key("f:" + id)).Height(28).MarginX(4).PaddingStart(8 + float32(depth)*12).PaddingEnd(8).Radius(radiusBtn).Gap(4).
		AlignItems(ui.Center).Justify(ui.Start).ContextMenu(func(m *ui.Menu) { a.nodeMenu(m, id) }).Drag(nodeDrag{id})
	if d, ok := ui.Drop[nodeDrag](row); ok {
		a.later(func() { a.dropOnNode(d.id, id) })
	}
	d, over := ui.DragOver[nodeDrag](row)
	over = over && d.id != id
	if a.side.reveal == id {
		row = a.revealRow(row)
	}
	hover := row.Hovered()
	switch {
	case over:
		row = row.Background(p.sideActive) // 拖动经过文件夹：高亮表示将放入其中
	case a.side.sel == id:
		row = row.Background(p.sideActive)
	case hover:
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
			a.selectNode(id)
			a.toggleFolder(id)
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
	// 侧栏只高亮一行：选中的节点（点过的文件夹/请求，没有时是当前标签对应的请求）。
	active := a.selectedNode() == id
	row := ui.ButtonBase(c.Key("r:" + id)).Height(28).MarginX(4).PaddingStart(20 + float32(depth)*12).PaddingEnd(8).Radius(radiusBtn).Gap(4).
		AlignItems(ui.Center).Justify(ui.Start).ContextMenu(func(m *ui.Menu) { a.nodeMenu(m, id) }).Drag(nodeDrag{id})
	if d, ok := ui.Drop[nodeDrag](row); ok {
		a.later(func() { a.dropOnNode(d.id, id) })
	}
	d, over := ui.DragOver[nodeDrag](row)
	over = over && d.id != id
	if a.side.reveal == id {
		row = a.revealRow(row)
	}
	hover := row.Hovered()
	switch {
	case active:
		row = row.Background(p.sideActive)
	case hover:
		row = row.Background(p.sideHover)
	}
	row.Children(func() {
		if over {
			// 拖动经过请求：在行顶部画插入线，表示将放到它之前。
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(2).Background(p.accent).ZIndex(1)
		}
		txtMono(c, n.method, 9, p.method(n.method), 700).MinWidth(28).Shrink(0)
		col, w := p.fg, 0
		if active {
			col, w = p.accent, 500
		}
		txtE(c, n.name, fsV2xs, col, w).Grow(1)
		if q := strings.ToLower(strings.TrimSpace(a.side.query)); q != "" && !hover {
			// 搜索结果：名称/地址以外的位置命中时，在行尾标出命中位置
			if label := a.hitLabel(a.searchHit(n, q)); label != "" {
				txt(c, label, 9, p.muted, 0).Shrink(0)
			}
		}
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

func (a *app) nodeMenu(m *ui.Menu, id string) {
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	a.selectNode(id)
	if n.folder {
		if m.Item(a.t("添加请求", "Add request")).Chosen() {
			a.later(func() { a.newRequest(id) })
		}
		if m.Item(a.t("添加文件夹", "Add folder")).Chosen() {
			a.later(func() { a.newFolder(id) })
		}
		open := a.side.outline.Open.Has(id)
		label := a.t("展开", "Expand")
		if open {
			label = a.t("折叠", "Collapse")
		}
		if m.Item(label).Chosen() {
			a.later(func() { a.toggleFolder(id) })
		}
		if m.Item(a.t("导出 JSON…", "Export JSON…")).Chosen() {
			a.later(func() { a.exportNodeOf(id) })
		}
		m.Separator()
	} else {
		if m.Item(a.t("打开", "Open")).Chosen() {
			a.later(func() { a.openRequest(id) })
		}
		if m.Item(a.t("复制 cURL", "Copy as cURL")).Chosen() {
			a.later(func() { a.copyCurlOf(id) })
		}
		if m.Item(a.t("导出 cURL…", "Export cURL…")).Chosen() {
			a.later(func() { a.exportCurlOf(id) })
		}
		if m.Item(a.t("导出 JSON…", "Export JSON…")).Chosen() {
			a.later(func() { a.exportNodeOf(id) })
		}
		m.Separator()
	}
	if m.Item(a.t("重命名", "Rename")).Shortcut(ui.Cmd, ui.KeyE).Chosen() {
		a.later(func() { a.renameNode(id) })
	}
	if m.Item(a.t("复制", "Duplicate")).Shortcut(ui.Cmd, ui.KeyD).Chosen() {
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
	if m.Item(a.t("删除", "Delete")).Danger().Shortcut(0, ui.KeyBackspace).Chosen() {
		a.later(func() { a.deleteNode(id) })
	}
}

// toggleFolder 展开或折叠文件夹。
func (a *app) toggleFolder(id string) {
	if a.side.outline.Open.Has(id) {
		a.side.outline.Open.Remove(id)
	} else {
		a.side.outline.Open.Add(id)
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
	ui.Row(c).FillWidth().Shrink(0).Padding(2, 8, 6, 8).AlignItems(ui.Center).Children(func() {
		a.input(c, &a.side.histQuery, a.t("搜索历史（方法 / 地址 / 状态码 / 4xx）", "Search history (method / URL / status / 4xx)"), fieldOpts{icon: "search", radius: radiusInput, key: "hist-search"})
	})
	groups, order := a.groupHistory()
	if len(order) == 0 {
		a.sideHint(c, a.t("无匹配结果", "No results"))
		return
	}
	ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Children(func() {
			for _, label := range order {
				entries := groups[label]
				collapsed := a.side.histCollapsed[label]
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
						if a.side.histCollapsed == nil {
							a.side.histCollapsed = map[string]bool{}
						}
						a.side.histCollapsed[label] = !collapsed
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
	row := ui.ButtonBase(c.Key("h:" + e.ID)).FillWidth().Height(24).MarginX(4).PaddingStart(24).PaddingEnd(12).Radius(radiusBtn).AlignItems(ui.Center).Justify(ui.Start).
		ContextMenu(func(m *ui.Menu) { a.historyMenu(m, e) })
	hover := row.Hovered()
	if hover {
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
		// 右侧固定宽度：时间在左、删除按钮在右。按钮始终存在（只在悬停时画图标），
		// 这样按下鼠标的那一帧按钮不会因为悬停状态变化而消失；悬停时隐藏时间，宽度不变。
		ui.Row(c).Width(46).Shrink(0).Justify(ui.End).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Width(30).Shrink(0).Justify(ui.End).Children(func() {
				if !hover {
					txt(c, historyClock(e.Timestamp), fsTiny, p.muted, 0)
				}
			})
			if a.button(c, btnOpts{w: 16, h: 20, tip: a.t("删除此记录", "Delete this entry"), noHover: true, key: "hd:" + e.ID}, func(h bool) {
				if hover {
					col := p.muted
					if h {
						col = p.danger
					}
					icon(c, "delete", 12, col)
				}
			}) {
				a.later(func() { a.deleteHistoryEntry(e) })
			}
		})
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
				row := ui.ButtonBase(c.Key("e:" + e.ID)).FillWidth().Height(24).MarginX(4).PaddingX(12).Radius(radiusBtn).AlignItems(ui.Center).Justify(ui.Start).
					ContextMenu(func(m *ui.Menu) { a.sidebarEnvMenu(m, e.ID, e.Name) })
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
		if !historyMatch(e, a.side.histQuery) {
			continue
		}
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

// openFromHistory 点击历史记录：请求仍存在则打开，否则提示。
func (a *app) openFromHistory(e historyItem) {
	if _, ok := a.requestItem(e.RequestID); ok {
		a.openRequest(e.RequestID)
		return
	}
	a.toast(a.t("该请求已被删除", "This request no longer exists"))
}
