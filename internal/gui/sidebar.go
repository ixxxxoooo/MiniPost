package gui

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"minipost/internal/model"
)

// sidebarFrame 是侧栏的外框（设计稿方案 C）：半透明的侧栏底色透出窗口极光，右侧大圆角，
// 右边缘一条内嵌发丝线，底部与窗口边缘留 shellGap，看起来悬浮在背景上。
func (a *app) sidebarFrame(c *ui.Context) {
	p := a.p
	ui.Column(c).Fill().Margin(0, 0, shellGap, 0).Radius(0, radiusLg, radiusLg, 0).Clip().
		Background(p.sidebar.Alpha(0.55)).InsetShadow(-1, 0, 0, 0, p.border.Alpha(0.7)).Children(func() {
		a.sidebar(c)
	})
}

// sidebar 绘制左侧栏：顶部三个标签（请求 / 历史 / 环境）与对应内容；底色由外框 sidebarFrame 提供。
func (a *app) sidebar(c *ui.Context) {
	ui.Row(c).Fill().Children(func() {
		ui.Column(c).Grow(1).Basis(0).FillHeight().Children(func() {
			a.sideTabs(c)
			// 请求 / 历史 / 环境 切换：旧内容先淡出，新内容再进入
			a.pane(c, "side"+itoa(a.side.tab), func() {
				switch a.side.tab {
				case 1:
					a.historyView(c)
					a.sideFoot(c, "clock", a.t(fmt.Sprintf("%d 条记录", len(a.history)), fmt.Sprintf("%d entries", len(a.history))))
				case 2:
					a.envList(c)
					a.sideFoot(c, "info", a.t("切换后 {{变量}} 立即替换", "Switching replaces {{variables}} immediately"))
				default:
					a.collectionView(c)
					reqs, folders := a.treeCounts()
					a.sideFoot(c, "layers", a.t(fmt.Sprintf("%d 个请求 · %d 个文件夹", reqs, folders), fmt.Sprintf("%d requests · %d folders", reqs, folders)))
				}
			})
		})
	})
}

// kbdHint 是搜索框等处的快捷键提示小键帽（设计稿 .kbd：等宽 10px，圆角 4，细描边），仅作提示、不响应点击。
func (a *app) kbdHint(c *ui.Context, s string) {
	p := a.p
	ui.Row(c.Key("kbd:"+s)).Height(16).PaddingX(5).Margin(0, 5, 0, 0).Radius(4).Background(p.surface.Mix(p.fg, 0.04)).Border(1, p.border).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
		txtMono(c, s, fsTiny, p.muted, 500)
	})
}

// treeCounts 返回当前项目的请求数与文件夹数（不含根目录）。
func (a *app) treeCounts() (reqs, folders int) {
	for _, n := range a.tree.nodes {
		if n.folder {
			folders++
		} else {
			reqs++
		}
	}
	return
}

// sideFoot 是侧栏底部的信息行（设计稿 .side-f）：图标 + 11px 次级文字，上沿一条细线。
func (a *app) sideFoot(c *ui.Context, iconName, text string) {
	p := a.p
	ui.Row(c.Key("sidefooter")).FillWidth().Height(26).Shrink(0).PaddingX(12).Gap(6).AlignItems(ui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(p.border).Children(func() {
		icon(c, iconName, 12, p.muted)
		txt(c, text, fsSmall, p.muted, 0).Grow(1)
	})
}

// sideTabs 是侧栏顶部的分段控件（设计稿 .side-h）：三个等宽的胶囊分段，选中项是浮起的内容色胶囊，
// 图标用强调色；上下内边距 8/6，整体 40px 高。
func (a *app) sideTabs(c *ui.Context) {
	ui.Row(c).FillWidth().Height(sizeSideHead).Shrink(0).Padding(8, 8, 6, 8).AlignItems(ui.Stretch).Children(func() {
		sel := a.side.tab
		a.pillSeg(c, "sidetabs", &sel, []string{a.t("请求", "Requests"), a.t("历史", "History"), a.t("环境", "Environments")},
			[]string{"layers", "clock", "globe"}, segOpts{full: true})
		a.side.tab = sel
	})
}

// searchRow 是请求标签顶部的搜索框与三个操作按钮（新建请求、新建文件夹、导入），设计稿 .sr：
// 26px 高的搜索框 + 三个 26×26 图标按钮，间距 2，下方留 6。
func (a *app) searchRow(c *ui.Context) {
	ui.Row(c).FillWidth().Shrink(0).Padding(0, 8, 6, 8).Gap(2).AlignItems(ui.Center).Children(func() {
		a.input(c, &a.side.query, a.t("搜索请求…", "Search requests…"), fieldOpts{icon: "search", radius: radiusR, key: "side-search", h: 26, bd: &a.p.border,
			trail: func() {
				if strings.TrimSpace(a.side.query) == "" {
					a.kbdHint(c, "⌘K")
				}
			}})
		a.ib(c, "side-new-req", "fileAdd", a.t("新建请求", "New request")+" (⌘N)", false, false, func() { a.newRequest("") })
		a.ib(c, "side-new-folder", "folderAdd", a.t("新建文件夹", "New folder"), false, false, func() { a.newFolder("") })
		a.ib(c, "side-import", "upload", a.t("导入", "Import"), false, false, func() { a.openImport(importFile) })
	})
}

// rootMenu 是“新建”菜单，也是集合空白处的右键菜单。
func (a *app) rootMenu(m *ui.Menu) {
	if m.Item(a.t("新建请求", "New request")).Shortcut(ui.Cmd, ui.KeyN).Chosen() {
		a.later(func() { a.newRequest("") })
	}
	if m.Item(a.t("新建文件夹", "New folder")).Shortcut(ui.Cmd|ui.Shift, ui.KeyN).Chosen() {
		a.later(func() { a.newFolder("") })
	}
}

// collectionView 绘制请求标签：搜索行与集合树。
func (a *app) collectionView(c *ui.Context) {
	a.searchRow(c)
	q := strings.ToLower(strings.TrimSpace(a.side.query))
	list := ui.Scroll(c).Grow(1).Basis(0).ContextMenu(a.rootMenu).Scrollbars(a.scrollbarsVisible())
	// 整个列表的空白处也是“移回根目录”的落点（树内容不足一屏时，下方的空白属于滚动区而不属于树）
	if d, ok := ui.Drop[nodeDrag](list); ok {
		a.later(func() { a.dropOnRoot(d.id) })
	}
	list.Children(func() {
		ui.Column(c).FillWidth().Padding(2, 0, 8, 0).Children(func() {
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
				a.treeRootDrop(c)
			}
		})
	})
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

// treeRootDrop 是请求树下方的空白拖放区：把节点拖到这里就移回项目根目录；
// 拖动经过时在顶部画一条强调色插入线。设计稿的树里没有“根目录”行，用这块空白承担这个功能。
func (a *app) treeRootDrop(c *ui.Context) {
	p := a.p
	zone := ui.Column(c.Key("tree-root")).FillWidth().MinHeight(48).Grow(1).ContextMenu(a.rootMenu)
	if d, ok := ui.Drop[nodeDrag](zone); ok {
		a.later(func() { a.dropOnRoot(d.id) })
	}
	if _, over := ui.DragOver[nodeDrag](zone); over {
		zone = zone.Background(p.acSoft)
	}
	zone.Children(func() {})
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

// treeRowBox 给树行套上设计稿 .row 的外观：高 sizeRow、圆角 radiusSm；选中时强调色淡底 + 内描边 + 左侧发光竖条，悬停浅底。
// depth 为 0 的行外边距 6，嵌套在引导线里的行外边距左 4 右 6。
func (a *app) treeRowBox(row ui.Element, depth int, selected, over, hover bool) ui.Element {
	p := a.p
	ml := float32(6)
	if depth > 0 {
		ml = 4
	}
	row = row.Height(sizeRow).Margin(0, 6, 0, ml).Radius(radiusSm)
	switch {
	case over:
		return row.Background(p.acSoft2)
	case selected:
		return row.Background(p.accent.Alpha(0.16)).InsetShadow(0, 0, 0, 1, p.accent.Alpha(0.13))
	case hover:
		return row.Background(p.hover)
	}
	return row
}

// selBar 是选中行左边缘的发光竖条（设计稿 .row.sel::after）：贴着侧栏左边，上下各缩 6。
func (a *app) selBar(c *ui.Context, leftOff float32) {
	ui.Box(c).Absolute().Left(-leftOff).Top(6).Bottom(6).Width(3).Radius(0, 3, 3, 0).Background(a.p.accent).Shadow(0, 0, 8, 0, a.p.glow)
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
	row := ui.ButtonBase(c.Key("f:" + id)).PaddingStart(8).PaddingEnd(6).Gap(6).
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
	row = a.tipIfLong(c, row, n.name, fsV2xs, 0, a.sideNameRoom(sideFolderChrome, depth, hover))
	selected := a.side.sel == id
	row = a.treeRowBox(row, depth, selected, over, hover)
	row.Children(func() {
		if selected {
			a.selBar(c, 6)
		}
		chev := "arrowRight"
		if open {
			chev = "arrowDown"
		}
		icon(c, chev, 12, p.muted)
		fi, fc := "folder", p.muted
		if open {
			fi, fc = "folderOpen", p.accent.Mix(p.muted, 0.3)
		}
		if hover {
			fc = p.accent
		}
		icon(c, fi, 14, fc)
		nameCol, nameW := p.fg, 0
		if selected {
			nameCol, nameW = p.accent, 600
		}
		txtE(c, n.name, fsV2xs, nameCol, nameW).Grow(1)
		if !hover {
			// 文件夹右侧的请求数（含子文件夹里的）；悬停时让位给操作按钮
			if cnt := a.tree.requestCount(id); cnt > 0 {
				txt(c, itoa(cnt), fsTiny, p.muted, 0).Shrink(0)
			}
		}
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
	if kids := a.tree.kids[id]; open && len(kids) > 0 {
		// 展开的文件夹：子节点放进左缩进 16、左侧 1px 引导线的容器里
		ui.Column(c.Key("kids:"+id)).FillWidth().Margin(0, 0, 0, 16).BorderWidth(0, 0, 0, 1).BorderColor(p.border).Children(func() {
			for _, k := range kids {
				a.treeNode(c, k, depth+1)
			}
		})
	}
}

// nodeMoreButton 是行右侧的“···”按钮，点击弹出节点菜单。
func (a *app) nodeMoreButton(c *ui.Context, id string) {
	p := a.p
	ui.ButtonBase(c.Key("more:"+id)).Width(16).Height(20).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).
		Tooltip(a.t("更多操作", "More actions")).
		PopoverMenuButton(func(m *ui.Menu) { a.nodeMenu(m, id) }, a.menuOpts()).
		Children(func() { icon(c, "more", 12, p.muted) })
}

// methodBadge 是方法徽章（设计稿 .row .m）：等宽粗体 9.5px，居中于 36×16 的圆角小标签，
// 底色为方法色 12%，描边为方法色约 20%。树、标签、历史、快速打开共用。
func (a *app) methodBadge(c *ui.Context, method string, w float32) ui.Element {
	col := a.p.method(method)
	box := ui.Row(c).Height(16).Radius(4)
	if w > 0 {
		box = box.Width(w)
	} else {
		// 标签里的徽章：设计稿是文字本体 + 向外各扩 3px 的底，这里用 2px 内边距 + 1px 描边折算
		box = box.PaddingX(2)
	}
	return box.Background(col.Alpha(0.12)).Border(1, col.Alpha(0.16)).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
		txtMono(c, methodAbbr(method), 9.5, col, 700)
	})
}

// requestRow 绘制一个请求行：方法徽章与名称；选中的行高亮并带左侧发光竖条。
func (a *app) requestRow(c *ui.Context, id string, depth int) {
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	p := a.p
	// 侧栏只高亮一行：选中的节点（点过的文件夹/请求，没有时是当前标签对应的请求）。
	active := a.selectedNode() == id
	row := ui.ButtonBase(c.Key("r:" + id)).PaddingStart(5).PaddingEnd(6).Gap(3).
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
	row = a.tipIfLong(c, row, n.name, fsV2xs, 0, a.sideNameRoom(sideRequestChrome, depth, hover))
	// 拖动经过请求时在行顶部画插入线（见下），行本身不再换底色。
	row = a.treeRowBox(row, depth, active, false, hover)
	row.Children(func() {
		if over {
			// 拖动经过请求：在行顶部画插入线，表示将放到它之前。
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(2).Background(p.accent).ZIndex(1)
		}
		if active {
			a.selBar(c, 6)
		}
		a.methodBadge(c, n.method, sizeMethodCol+4)
		col, w := p.fg, 0
		if active {
			col, w = p.accent, 600
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
		if m.Item(a.t("新建请求", "New request")).Shortcut(ui.Cmd, ui.KeyN).Chosen() {
			a.later(func() { a.newRequest(id) })
		}
		if m.Item(a.t("新建文件夹", "New folder")).Shortcut(ui.Cmd|ui.Shift, ui.KeyN).Chosen() {
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
		if m.Item(a.t("在新标签页打开", "Open in new tab")).Chosen() {
			a.later(func() { a.openRequest(id) })
		}
		if m.Item(a.t("复制为 cURL", "Copy as cURL")).Shortcut(ui.Cmd|ui.Shift, ui.KeyC).Chosen() {
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

// historyView 绘制历史标签：搜索行（含刷新、清空）与按天分组的记录（设计稿 .sr + .grp + .hrow）。
func (a *app) historyView(c *ui.Context) {
	p := a.p
	ui.Row(c).FillWidth().Shrink(0).Padding(0, 8, 6, 8).Gap(2).AlignItems(ui.Center).Children(func() {
		a.input(c, &a.side.histQuery, a.t("搜索历史…", "Search history…"), fieldOpts{icon: "search", radius: radiusR, key: "hist-search", h: 26, bd: &a.p.border})
		a.ib(c, "hist-refresh", "arrowUpDown", a.t("刷新", "Refresh"), false, false, a.reloadHistory)
		a.ib(c, "hist-clear", "delete", a.t("清空历史", "Clear history"), false, len(a.history) == 0, func() {
			a.overlays.askConfirm(a.t("清空历史记录？", "Clear history?"), a.t("此操作无法撤销。", "This can't be undone."), a.t("清空", "Clear"), func() {
				if err := a.be.ClearHistory(a.projectID); err != nil {
					a.fail(a.t("清空历史", "Clear history"), err)
					return
				}
				a.reloadHistory()
			})
		})
	})
	if len(a.history) == 0 {
		a.sideEmpty(c, "clock", a.t("暂无历史记录", "No history yet"), a.t("发送请求后会自动记录在这里", "Requests you send are recorded here"))
		return
	}
	groups, order := a.groupHistory()
	if len(order) == 0 {
		a.sideHint(c, a.t("无匹配结果", "No results"))
		return
	}
	ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Padding(2, 0, 8, 0).Children(func() {
			for _, label := range order {
				label := label
				entries := groups[label]
				collapsed := a.side.histCollapsed[label]
				head := ui.ButtonBase(c.Key("hg:"+label)).FillWidth().Height(24).Margin(4, 0, 0, 0).PaddingX(12).Gap(4).AlignItems(ui.Center).Justify(ui.Start)
				hover := head.Hovered()
				head.Children(func() {
					col := p.muted
					if hover {
						col = p.fg2
					}
					chev := "chevD"
					if collapsed {
						chev = "chevR"
					}
					icon(c, chev, 11, col)
					txt(c, label, fsSmall, col, 600)
					ui.Row(c).Height(15).MinWidth(16).PaddingX(4).Radius(8).Background(p.surface2).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
						txtMono(c, fmt.Sprint(len(entries)), fsTiny, p.muted, 600)
					})
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

// sideEmpty 是侧栏列表为空时的居中提示：图标、加粗标题与一句说明（设计稿 .empty-s）。
func (a *app) sideEmpty(c *ui.Context, iconName, title, sub string) {
	p := a.p
	ui.Column(c).FillWidth().Padding(28, 16).Gap(8).AlignItems(ui.Center).Children(func() {
		icon(c, iconName, 22, p.muted.Alpha(0.7))
		txt(c, title, fsV2xs, p.fg2, 600)
		txt(c, sub, fsSmall, p.muted, 0)
	})
}

// urlPathQuery 返回地址里的“路径 + 查询”部分（设计稿历史行只显示 /post、/get?page=1）；
// 含 {{变量}} 或无法解析的地址原样返回，没有路径时返回 /。
func urlPathQuery(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.Contains(raw, "{{") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	out := u.EscapedPath()
	if out == "" {
		out = "/"
	}
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	return out
}

// historyRow 绘制一条历史记录（设计稿 .hrow）：方法徽章、等宽路径、状态码与时间；悬停时时间位置换成删除按钮。
func (a *app) historyRow(c *ui.Context, e historyItem) {
	p := a.p
	row := ui.ButtonBase(c.Key("h:" + e.ID)).PaddingStart(5).PaddingEnd(6).Gap(3).AlignItems(ui.Center).Justify(ui.Start).
		ContextMenu(func(m *ui.Menu) { a.historyMenu(m, e) })
	hover := row.Hovered()
	row = a.treeRowBox(row, 0, false, false, hover)
	if full := firstNonEmpty(e.URL, e.Name); full != "" {
		row = a.tipIfLong(c, row, full, fsSmall, 0, a.sidebarShown()-sideHistoryChrome)
	}
	row.Children(func() {
		method := e.Method
		if method == "" {
			method = "GET"
		}
		a.methodBadge(c, method, sizeMethodCol+4)
		u := urlPathQuery(e.URL)
		if u == "" {
			u = e.Name
		}
		if i := strings.Index(u, "?"); i > 0 {
			u = u[:i]
		}
		txtEMono(c, u, fsSmall+0.5, p.fg, 0).Grow(1).Basis(0).MinWidth(0)
		code := fmt.Sprint(e.StatusCode)
		if e.StatusCode == 0 {
			code = "ERR"
		}
		txtMono(c, code, fsSmall, a.statusTextColor(e.StatusCode), 600).MarginEnd(4)
		// 右侧固定 34 宽：时间右对齐；悬停时同一位置换成删除按钮（按钮始终存在，保证按下那一帧不会消失）
		ui.Row(c).Width(34).Shrink(0).Justify(ui.End).AlignItems(ui.Center).Children(func() {
			if a.button(c, btnOpts{w: 34, h: 20, tip: a.t("删除此记录", "Delete this entry"), noHover: true, key: "hd:" + e.ID}, func(h bool) {
				if hover {
					col := p.muted
					if h {
						col = p.danger
					}
					icon(c, "delete", 12, col)
					return
				}
				ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
					txtMono(c, historyClock(e.Timestamp), fsTiny+0.5, p.muted, 0)
				})
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
	case code >= 400 || code == 0:
		return a.p.danger
	}
	return a.p.warning
}

// ---- 环境 ----

// envList 绘制环境标签（设计稿：36 高的环境行 + 末尾“新建环境”行）：地球图标、名称与变量数，
// 当前环境带绿色“当前”徽标；悬停时右侧换成删除按钮。
func (a *app) envList(c *ui.Context) {
	p := a.p
	ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Padding(5, 0, 8, 0).Children(func() {
			for _, e := range a.envs {
				e := e
				active := e.ID == a.envID
				row := ui.ButtonBase(c.Key("e:" + e.ID)).PaddingStart(8).PaddingEnd(6).Gap(6).AlignItems(ui.Center).Justify(ui.Start).
					ContextMenu(func(m *ui.Menu) { a.sidebarEnvMenu(m, e.ID, e.Name) })
				hover := row.Hovered()
				open := a.envTabOpenActive(e.ID)
				row = a.treeRowBox(row, 0, open, false, hover).Height(36)
				row = a.tipIfLong(c, row, e.Name, fsV2xs, 0, a.sidebarShown()-100)
				row.Children(func() {
					if open {
						a.selBar(c, 6)
					}
					ic := p.muted
					if hover {
						ic = p.accent
					}
					icon(c, "globe", 15, ic).Shrink(0)
					ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
						txtE(c, e.Name, fsV2xs, p.fg, 600)
						txtE(c, a.t(fmt.Sprintf("%d 个变量", countEnvVars(e)), fmt.Sprintf("%d variables", countEnvVars(e))), fsSmall-0.5, p.muted, 0)
					})
					switch {
					case hover:
						if a.miniIcon(c, "delete", a.t("删除环境", "Delete environment")) {
							a.later(func() { a.deleteEnvironment(e.ID, e.Name) })
						}
					case active:
						ui.Row(c).Height(16).PaddingX(6).Gap(3).Radius(8).Background(p.success.Alpha(0.14)).Border(1, p.success.Alpha(0.24)).AlignItems(ui.Center).Shrink(0).Children(func() {
							icon(c, "check", 10, p.success)
							txt(c, a.t("当前", "In use"), fsTiny, p.success, 600)
						})
					}
				})
				if row.Clicked() {
					a.later(func() { a.openEnvManagerFor(e.ID) })
				}
			}
			add := ui.ButtonBase(c.Key("env-add")).PaddingStart(8).PaddingEnd(6).Gap(6).AlignItems(ui.Center).Justify(ui.Start)
			add = a.treeRowBox(add, 0, false, false, add.Hovered()).Margin(4, 6, 0, 6)
			add.Tooltip(a.t("新建环境", "New environment")).Children(func() {
				ui.Box(c).Width(17).Shrink(0)
				icon(c, "plus", 14, p.muted)
				txt(c, a.t("新建环境", "New environment"), fsV2xs, p.muted, 0)
			})
			if add.Clicked() {
				a.later(a.newEnvironment)
			}
		})
	})
}

// countEnvVars 返回环境里有名字的变量数（不含末尾空白行）。
func countEnvVars(e model.Environment) int {
	n := 0
	for _, v := range e.Variables {
		if strings.TrimSpace(v.Key) != "" {
			n++
		}
	}
	return n
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
