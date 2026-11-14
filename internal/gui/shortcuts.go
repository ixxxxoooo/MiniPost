package gui

import (
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
//   - 侧栏选中节点：⌘E 重命名、⌘D / ⌘C 复制、⌫ / Delete 删除（输入框有焦点时不响应，避免误删）；
//   - ⌘F 在响应体里查找（编辑器有焦点时由编辑器自己处理）；
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
	if c.FocusTakesText() || a.side.tab != 0 {
		return
	}
	id := a.selectedNode()
	if id == "" {
		return
	}
	switch {
	case c.Shortcut(ui.Cmd, ui.KeyE):
		a.later(func() { a.renameNode(id) })
	case c.Shortcut(ui.Cmd, ui.KeyD), c.Shortcut(ui.Cmd, ui.KeyC):
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
