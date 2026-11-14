package gui

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
)

// modalOpen 报告是否有对话框盖在主界面上；此时全局快捷键不应作用到后面的内容。
func (a *app) modalOpen() bool {
	o := &a.overlays
	return o.prompt.open || o.confirm.open || o.closing.open || o.prefs.open || o.imp.open || o.curl.open || o.cookies.open
}

// selectedNode 是侧栏当前选中的请求/文件夹：最近点击或右键过的节点，没有则取当前标签对应的请求。
func (a *app) selectedNode() string {
	if id := a.side.sel; id != "" {
		if _, ok := a.tree.nodes[id]; ok {
			return id
		}
	}
	if t := a.activeTab(); t != nil && t.env == nil {
		if _, ok := a.tree.nodes[t.draft.ID]; ok {
			return t.draft.ID
		}
	}
	return ""
}

// selectNode 记录侧栏选中的节点。
func (a *app) selectNode(id string) { a.side.sel = id }

// handleShortcuts 处理不在菜单里的全局快捷键（每帧在构建视图时调用）：
//   - 侧栏选中节点：⌘E 重命名、⌘D 复制（⌘C 留给系统的“复制文字”，不再复制节点）、⌫ / Delete 删除（输入框有焦点时不响应，避免误删）；
//   - ⌘F 在响应体里查找（编辑器有焦点时由编辑器自己处理）；
//   - 侧栏键盘导航：↑↓ 选择上/下一行，← 折叠或回到父文件夹，→ 展开或进入第一个子项，回车打开请求/展开折叠文件夹；
//   - 标签切换：⌃Tab / ⌃⇧Tab、⌘⇧] / ⌘⇧[。
func (a *app) handleShortcuts(c *ui.Context) {
	if a.modalOpen() || a.showHome || a.projectID == "" {
		return
	}
	if c.Shortcut(ui.Cmd, ui.KeyF) {
		a.later(a.findInResponse)
	}
	switch {
	case c.Shortcut(ui.Ctrl, ui.KeyTab), c.Shortcut(ui.Cmd|ui.Shift, ui.KeyBracketRight):
		a.later(func() { a.cycleTab(1) })
	case c.Shortcut(ui.Ctrl|ui.Shift, ui.KeyTab), c.Shortcut(ui.Cmd|ui.Shift, ui.KeyBracketLeft):
		a.later(func() { a.cycleTab(-1) })
	}
	if !c.FocusTakesText() && c.Shortcut(ui.Cmd, ui.KeyZ) {
		a.later(a.undoLast) // 输入框有焦点时 ⌘Z 留给输入框自己撤销文字
	}
	if c.FocusTakesText() || a.side.tab != 0 {
		return
	}
	switch {
	case c.Shortcut(0, ui.KeyDown):
		a.later(func() { a.moveSelection(1) })
		return
	case c.Shortcut(0, ui.KeyUp):
		a.later(func() { a.moveSelection(-1) })
		return
	case c.Shortcut(0, ui.KeyLeft):
		a.later(a.selectionLeft)
		return
	case c.Shortcut(0, ui.KeyRight):
		a.later(a.selectionRight)
		return
	case c.Shortcut(0, ui.KeyEnter):
		a.later(a.selectionEnter)
		return
	}
	id := a.selectedNode()
	if id == "" {
		return
	}
	switch {
	case c.Shortcut(ui.Cmd, ui.KeyE):
		a.later(func() { a.renameNode(id) })
	case c.Shortcut(ui.Cmd, ui.KeyD):
		a.later(func() { a.duplicateNode(id) })
	case c.Shortcut(0, ui.KeyBackspace), c.Shortcut(0, ui.KeyDelete):
		a.later(func() { a.deleteNode(id) })
	}
}

// cycleTab 切换到上一个（-1）/下一个（1）标签，首尾相接。
func (a *app) cycleTab(dir int) {
	n := len(a.tabs)
	if n < 2 {
		return
	}
	a.showHome = false
	next := (a.active + dir + n) % n
	logger.Debug("切换标签", "from", a.active, "to", next)
	a.activate(next)
}

// findInResponse 在响应体里打开查找；响应还没有、正在预览或是下载卡片时什么也不做。
func (a *app) findInResponse() {
	t := a.activeTab()
	if t == nil || t.env != nil || t.resp == nil || t.resp.resp == nil || t.resp.bodyEd == nil {
		return
	}
	if v := t.resp.view; v.preview || v.download || v.isImage {
		return
	}
	t.resp.bodyEd.Find("")
}

// visibleNodeIDs 返回侧栏当前可见的节点（按从上到下的顺序）：
// 有搜索词时是匹配的请求，否则是展开状态下的整棵树。
func (a *app) visibleNodeIDs() []string {
	q := strings.ToLower(strings.TrimSpace(a.side.query))
	var out []string
	if q != "" {
		for _, id := range a.tree.descendants("")[1:] {
			if nd := a.tree.nodes[id]; !nd.folder && a.matchNode(nd, q) {
				out = append(out, id)
			}
		}
		return out
	}
	var walk func(ids []string)
	walk = func(ids []string) {
		for _, id := range ids {
			out = append(out, id)
			if a.tree.nodes[id].folder && a.side.outline.Open.Has(id) {
				walk(a.tree.kids[id])
			}
		}
	}
	walk(a.tree.roots())
	return out
}

// selectAndReveal 选中节点并让侧栏在下一帧把它滚进可视区。
func (a *app) selectAndReveal(id string) {
	a.selectNode(id)
	a.side.reveal = id
	a.side.revealFocus = true
}

// moveSelection 把选中项向下（1）或向上（-1）移动一行；没有选中项时从首/末行开始。
func (a *app) moveSelection(delta int) {
	ids := a.visibleNodeIDs()
	if len(ids) == 0 {
		return
	}
	cur := -1
	for i, id := range ids {
		if id == a.selectedNode() {
			cur = i
		}
	}
	next := cur + delta
	if cur < 0 {
		next = 0
		if delta < 0 {
			next = len(ids) - 1
		}
	}
	next = max(0, min(len(ids)-1, next))
	a.selectAndReveal(ids[next])
}

// selectionLeft：展开的文件夹先折叠；否则选中它的父文件夹。
func (a *app) selectionLeft() {
	id := a.selectedNode()
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	if n.folder && a.side.outline.Open.Has(id) {
		a.side.outline.Open.Remove(id)
		return
	}
	if n.parent != "" {
		a.selectAndReveal(n.parent)
	}
}

// selectionRight：折叠的文件夹先展开；已展开的选中它的第一个子项。
func (a *app) selectionRight() {
	id := a.selectedNode()
	n, ok := a.tree.nodes[id]
	if !ok || !n.folder {
		return
	}
	if !a.side.outline.Open.Has(id) {
		a.side.outline.Open.Add(id)
		return
	}
	if kids := a.tree.kids[id]; len(kids) > 0 {
		a.selectAndReveal(kids[0])
	}
}

// selectionEnter：请求在标签里打开；文件夹展开/折叠。
func (a *app) selectionEnter() {
	id := a.selectedNode()
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	if n.folder {
		a.toggleFolder(id)
		return
	}
	a.showHome = false
	a.openRequest(id)
}

// revealRow 在侧栏行构建时消费一次 reveal 请求：滚进可视区，键盘导航时再给它焦点。
func (a *app) revealRow(row ui.Element) ui.Element {
	row = row.ScrollIntoView()
	if a.side.revealFocus {
		row = row.Focus()
	}
	a.side.reveal, a.side.revealFocus = "", false
	return row
}
