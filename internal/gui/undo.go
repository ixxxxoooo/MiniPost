package gui

import (
	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// undoEntry 是一次可撤销的删除：label 用于提示，run 把数据放回去。
type undoEntry struct {
	label string
	run   func()
}

const undoLimit = 20

// pushUndo 登记一次可撤销的删除，并弹出带“撤销”按钮的提示；⌘Z 撤销最近一次。
func (a *app) pushUndo(label string, run func()) {
	e := &undoEntry{label: label, run: run}
	a.undoStack = append(a.undoStack, e)
	if len(a.undoStack) > undoLimit {
		a.undoStack = a.undoStack[len(a.undoStack)-undoLimit:]
	}
	a.toastAction(label+a.t(" 已删除", " deleted"), a.t("撤销", "Undo"), func() { a.runUndo(e) })
}

// runUndo 执行并移除指定的撤销项（已被撤销过的会被忽略）。
func (a *app) runUndo(e *undoEntry) {
	for i, x := range a.undoStack {
		if x == e {
			a.undoStack = append(a.undoStack[:i], a.undoStack[i+1:]...)
			e.run()
			logger.Info("已撤销删除", "label", e.label)
			return
		}
	}
}

// undoLast 撤销最近一次删除（⌘Z）。
func (a *app) undoLast() {
	if n := len(a.undoStack); n > 0 {
		a.runUndo(a.undoStack[n-1])
	}
}

// snapshotSubtree 取出节点及其所有后代的数据，用于删除后原样恢复。
func (a *app) snapshotSubtree(id string) model.CollectionData {
	in := map[string]bool{}
	for _, x := range a.tree.descendants(id) {
		in[x] = true
	}
	var sub model.CollectionData
	for _, f := range a.coll.Folders {
		if in[f.ID] {
			sub.Folders = append(sub.Folders, f)
		}
	}
	for _, r := range a.coll.Requests {
		if in[r.ID] {
			sub.Requests = append(sub.Requests, r)
		}
	}
	for _, n := range a.coll.TreeNodes {
		if in[n.NodeID] {
			sub.TreeNodes = append(sub.TreeNodes, n)
		}
	}
	return sub
}

// restoreSubtree 把删除的子树放回当前项目。
func (a *app) restoreSubtree(projectID string, sub model.CollectionData) {
	if err := a.be.RestoreCollectionSubtree(projectID, sub); err != nil {
		a.fail(a.t("撤销删除", "Undo delete"), err)
		return
	}
	if projectID == a.projectID {
		a.reloadCollection()
	}
}
