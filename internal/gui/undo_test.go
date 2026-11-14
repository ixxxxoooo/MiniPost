package gui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func hasNode(a *app, id string) bool { _, ok := a.tree.nodes[id]; return ok }

// 删除请求后按 ⌘Z 能原样恢复（同一个 ID、同一位置）。
func TestUndoDeleteRequest(t *testing.T) {
	a, tt, _ := zhApp(t)
	beta := a.coll.Requests[1].ID
	a.selectNode(beta)
	tt.Frame()
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if err := tt.Click("删除"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if hasNode(a, beta) {
		t.Fatal("删除后 beta 应该消失")
	}
	if len(a.undoStack) != 1 {
		t.Fatalf("应登记一条撤销记录: %d", len(a.undoStack))
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	tt.Frame()
	tt.Frame()
	if !hasNode(a, beta) {
		t.Fatal("⌘Z 后 beta 应恢复")
	}
	if got := a.tree.kids[""]; len(got) != 2 || got[1] != beta {
		t.Fatalf("恢复后应回到原来的位置: %v", got)
	}
	if len(a.undoStack) != 0 {
		t.Fatal("撤销后记录应被移除")
	}
}

// 删除文件夹会连同里面的请求一起恢复。
func TestUndoDeleteFolderRestoresChildren(t *testing.T) {
	a, tt, be := sideDragApp(t)
	var dir string
	for id, n := range a.tree.nodes {
		if n.name == "dir" {
			dir = id
		}
	}
	child, _ := be.CreateRequestItem(a.projectID, dir, "child")
	a.reloadCollection()
	a.deleteNode(dir)
	tt.Frame()
	if err := tt.Click("删除"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if hasNode(a, dir) {
		t.Fatal("文件夹应被删除")
	}
	a.undoLast()
	tt.Frame()
	if !hasNode(a, dir) || !hasNode(a, child.ID) || a.tree.nodes[child.ID].parent != dir {
		t.Fatal("撤销后文件夹及其请求都应恢复")
	}
}

// 删除环境后可撤销。
func TestUndoDeleteEnvironment(t *testing.T) {
	a, tt, be := zhApp(t)
	e, _ := be.CreateEnvironment(a.projectID, "dev")
	a.reloadEnvs()
	a.deleteEnvironment(e.ID, "dev")
	tt.Frame()
	if err := tt.Click("删除"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	for _, x := range a.envs {
		if x.ID == e.ID {
			t.Fatal("环境应被删除")
		}
	}
	a.undoLast()
	tt.Frame()
	found := false
	for _, x := range a.envs {
		if x.ID == e.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("撤销后环境应恢复")
	}
}
