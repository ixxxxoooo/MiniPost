package gui

import (
	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// newRequest 在文件夹（空串表示根目录）下创建请求并打开。
func (a *app) newRequest(folderID string) {
	it, err := a.be.CreateRequestItem(a.projectID, folderID, a.t("新请求", "New Request"))
	if err != nil {
		a.fail(a.t("创建请求", "Create request"), err)
		return
	}
	a.reloadCollection()
	a.openRequest(it.ID)
	logger.Info("已创建请求", "requestID", it.ID)
}

func (a *app) newFolder(parentID string) {
	a.overlays.askText(a.t("新建文件夹", "New Folder"), "", a.t("新文件夹", "New Folder"), a.t("创建", "Create"), func(name string) {
		f, err := a.be.CreateFolder(a.projectID, parentID, name)
		if err != nil {
			a.fail(a.t("创建文件夹", "Create folder"), err)
			return
		}
		a.reloadCollection()
		a.side.outline.Open.Add(f.ID)
		logger.Info("已创建文件夹", "folderID", f.ID)
	})
}

func (a *app) renameNode(id string) {
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	a.overlays.askText(a.t("重命名", "Rename"), "", n.name, a.t("确定", "OK"), func(name string) {
		var err error
		if n.folder {
			err = a.be.RenameFolder(a.projectID, id, name)
		} else {
			err = a.be.RenameRequest(a.projectID, id, name)
			if err == nil {
				a.renameTab(id, name)
			}
		}
		if err != nil {
			a.fail(a.t("重命名", "Rename"), err)
			return
		}
		a.reloadCollection()
	})
}

func (a *app) duplicateNode(id string) {
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	if n.folder {
		if _, err := a.be.DuplicateFolder(a.projectID, id); err != nil {
			a.fail(a.t("复制文件夹", "Duplicate folder"), err)
			return
		}
		a.reloadCollection()
		return
	}
	it, err := a.be.DuplicateRequest(a.projectID, id)
	if err != nil {
		a.fail(a.t("复制请求", "Duplicate request"), err)
		return
	}
	a.reloadCollection()
	a.openRequest(it.ID)
}

func (a *app) deleteNode(id string) {
	n, ok := a.tree.nodes[id]
	if !ok {
		return
	}
	title := a.t("删除“", "Delete “") + n.name + a.t("”？", "”?")
	msg := a.t("此操作无法撤销。", "This can't be undone.")
	if n.folder {
		msg = a.t("文件夹内的所有请求也会被删除，此操作无法撤销。", "All requests inside will be deleted too. This can't be undone.")
	}
	a.overlays.askConfirm(title, msg, a.t("删除", "Delete"), func() {
		ids := a.tree.descendants(id)
		var err error
		if n.folder {
			err = a.be.DeleteFolder(a.projectID, id)
		} else {
			err = a.be.DeleteRequestItem(a.projectID, id)
		}
		if err != nil {
			a.fail(a.t("删除", "Delete"), err)
			return
		}
		a.closeTabsFor(ids)
		a.side.outRow = -1
		a.reloadCollection()
		logger.Info("已删除节点", "nodeID", id)
	})
}

// moveNode 把节点移动到目标文件夹（空串表示根目录）的末尾。
func (a *app) moveNode(id, target string) {
	if n, ok := a.tree.nodes[id]; ok && n.parent == target {
		return
	}
	a.moveNodeAt(id, target, -1)
}

// moveNodeAt 把节点移动到目标文件夹的指定位置；idx 为插入后的下标（先摘掉自己再计算），小于 0 表示末尾。
func (a *app) moveNodeAt(id, target string, idx int) {
	n, ok := a.tree.nodes[id]
	if !ok || (n.folder && (target == id || a.tree.isInside(target, id))) {
		return
	}
	if idx < 0 || idx > len(a.tree.kids[target]) {
		idx = len(a.tree.kids[target])
		if n.parent == target {
			idx-- // 自己也在这个列表里，末尾下标要减一
		}
	}
	kind := model.CollectionNodeTypeRequest
	if n.folder {
		kind = model.CollectionNodeTypeFolder
	}
	if err := a.be.MoveCollectionNode(a.projectID, id, kind, target, idx); err != nil {
		a.fail(a.t("移动", "Move"), err)
		return
	}
	a.reloadCollection()
	if target != "" {
		a.side.outline.Open.Add(target)
	}
	logger.Info("已移动节点", "nodeID", id, "target", target, "index", idx)
}

// dropOnNode 处理侧栏里把 src 拖到 target 上：
// 放在文件夹上是移进该文件夹末尾；放在请求上是插到该请求之前（同一文件夹内即重新排序）。
func (a *app) dropOnNode(src, target string) {
	tn, ok := a.tree.nodes[target]
	if !ok || src == target {
		return
	}
	if tn.folder {
		a.moveNode(src, target)
		return
	}
	siblings := a.tree.kids[tn.parent]
	j, k := -1, -1
	for i, id := range siblings {
		switch id {
		case target:
			j = i
		case src:
			k = i
		}
	}
	if j < 0 {
		return
	}
	if k >= 0 && k < j {
		j-- // 自己在目标前面，摘掉自己后目标的下标前移一位
	}
	if k >= 0 && k == j {
		return // 位置没有变化
	}
	a.moveNodeAt(src, tn.parent, j)
}

// dropOnRoot 处理侧栏里把 src 拖到“根目录”上：移到项目根目录末尾。
func (a *app) dropOnRoot(src string) { a.moveNode(src, "") }

// nodeDrag 是拖动侧栏节点时携带的值。
type nodeDrag struct{ id string }
