package gui

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// sidebar 绘制左侧栏：集合树与历史记录。
func (a *app) sidebar(c *ui.Context) {
	ui.Column(c).Grow(1).Children(func() {
		ui.Row(c).Padding(8).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &a.sideTab, a.t("集合", "Collections"), a.t("历史", "History")).Grow(1)
			if a.sideTab == 0 {
				ui.MenuButton(c, "+", func(m *ui.Menu) { a.rootMenu(m) })
			}
		})
		ui.Divider(c)
		if a.sideTab == 0 {
			a.collectionView(c)
		} else {
			a.historyView(c)
		}
	})
}

// rootMenu 是“新建”菜单，也是集合空白处的右键菜单。
func (a *app) rootMenu(m *ui.Menu) {
	if m.Item(a.t("新建请求", "New Request")).Chosen() {
		a.later(func() { a.newRequest("") })
	}
	if m.Item(a.t("新建文件夹", "New Folder")).Chosen() {
		a.later(func() { a.newFolder("") })
	}
}

func (a *app) collectionView(c *ui.Context) {
	if len(a.tree.roots()) == 0 {
		ui.Column(c).Grow(1).Padding(16).Gap(10).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Text(c, a.t("这个项目还没有请求", "No requests yet")).TextColor(c.Theme().TextMuted)
			if ui.PrimaryButton(c, a.t("新建请求", "New Request")).Clicked() {
				a.later(func() { a.newRequest("") })
			}
		}).ContextMenu(a.rootMenu)
		return
	}
	if a.syncSel {
		a.syncSel = false
		if t := a.activeTab(); t != nil {
			for r := 0; r < a.outline.Rows(); r++ {
				if a.outline.Item(r) == t.draft.ID {
					a.outRow = r
					break
				}
			}
		}
	}
	a.outline.List.Selected = &a.outRow
	ui.Column(c).Grow(1).ContextMenu(a.rootMenu).Children(func() {
		el := ui.Outline(c, &a.outline, a.tree.roots(), a.tree.children, func(id string) {
			a.outlineRow(c, id)
		}).Grow(1)
		if el.Changed() || el.Submitted() {
			a.openSelectedNode(el.Submitted())
		}
	})
}

// openSelectedNode 打开侧栏当前选中的请求；文件夹仅在双击时展开或折叠。
func (a *app) openSelectedNode(submitted bool) {
	if a.outRow < 0 || a.outRow >= a.outline.Rows() {
		return
	}
	id := a.outline.Item(a.outRow)
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	if !n.folder {
		a.openRequest(id)
		return
	}
	if submitted {
		if a.outline.Open.Has(id) {
			a.outline.Open.Remove(id)
		} else {
			a.outline.Open.Add(id)
		}
	}
}

func (a *app) outlineRow(c *ui.Context, id string) {
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	th := c.Theme()
	ui.Row(c).Gap(6).AlignItems(ui.Center).ContextMenu(func(m *ui.Menu) { a.nodeMenu(m, id) }).Children(func() {
		if !n.folder {
			ui.Text(c, shortMethod(n.method)).Bold().FontSize(th.FontSize - 3).Width(34).TextColor(methodColor(n.method, th.Dark))
		}
		ui.Text(c, n.name).SingleLine().Shrink(1)
	})
}

// shortMethod 把方法名缩成侧栏里适合的短标签。
func shortMethod(m string) string {
	switch m {
	case "DELETE":
		return "DEL"
	case "OPTIONS":
		return "OPT"
	case "PATCH":
		return "PATCH"
	}
	return m
}

func (a *app) nodeMenu(m *ui.Menu, id string) {
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	if n.folder {
		if m.Item(a.t("新建请求", "New Request")).Chosen() {
			a.later(func() { a.newRequest(id) })
		}
		if m.Item(a.t("新建文件夹", "New Folder")).Chosen() {
			a.later(func() { a.newFolder(id) })
		}
		m.Separator()
	}
	if m.Item(a.t("重命名…", "Rename…")).Chosen() {
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

func (a *app) historyView(c *ui.Context) {
	if len(a.history) == 0 {
		ui.Column(c).Grow(1).Padding(16).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Text(c, a.t("暂无历史记录", "No history yet")).TextColor(c.Theme().TextMuted)
		})
		return
	}
	th := c.Theme()
	a.histList.Selected = &a.histRow
	list := ui.List(c, &a.histList, len(a.history), func(i int) {
		e := a.history[i]
		ui.Column(c).Gap(2).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, shortMethod(e.Method)).Bold().FontSize(th.FontSize - 3).Width(34).TextColor(methodColor(e.Method, th.Dark))
				ui.Text(c, e.URL).SingleLine().Shrink(1)
			})
			ui.Row(c).Gap(8).Children(func() {
				ui.Textf(c, "%d", e.StatusCode).FontSize(th.FontSize - 2).TextColor(statusColor(e.StatusCode, th))
				ui.Text(c, fmt.Sprintf("%.0f ms", e.Duration)).FontSize(th.FontSize - 2).TextColor(th.TextMuted)
				ui.Text(c, formatTimestamp(e.Timestamp)).FontSize(th.FontSize - 2).TextColor(th.TextMuted)
			})
		})
	}).Grow(1)
	if list.Changed() && a.histRow >= 0 && a.histRow < len(a.history) {
		e := a.history[a.histRow]
		a.histRow = -1
		if _, ok := a.requestItem(e.RequestID); ok {
			a.openRequest(e.RequestID)
		} else {
			a.toast(a.t("该请求已被删除", "This request no longer exists"))
		}
	}
	ui.Row(c).Padding(8).Justify(ui.End).Children(func() {
		if ui.Button(c, a.t("清空历史", "Clear History")).Clicked() {
			a.overlays.askConfirm(a.t("清空历史记录？", "Clear history?"), a.t("此操作无法撤销。", "This can't be undone."), a.t("清空", "Clear"), func() {
				if err := a.be.ClearHistory(a.projectID); err != nil {
					a.fail(a.t("清空历史", "Clear history"), err)
					return
				}
				a.reloadHistory()
			})
		}
	})
}

// formatTimestamp 把 RFC3339 时间格式化为本地短时间，解析失败时原样返回。
func formatTimestamp(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Local().Format("01-02 15:04:05")
}
