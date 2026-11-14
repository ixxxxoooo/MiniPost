package gui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func navApp(t *testing.T) (*app, *ui.Tester, string, string) {
	a, tt, be := sideDragApp(t)
	var dir string
	for id, n := range a.tree.nodes {
		if n.name == "dir" {
			dir = id
		}
	}
	child, _ := be.CreateRequestItem(a.projectID, dir, "child")
	a.reloadCollection()
	a.side.outline.Open.Add(dir)
	a.side.sel = ""
	tt.Frame()
	return a, tt, dir, child.ID
}

func selName(a *app) string { return a.tree.nodes[a.selectedNode()].name }

// ↑↓ 在可见节点间移动选中项，首尾不越界。
func TestSidebarArrowKeysMoveSelection(t *testing.T) {
	a, tt, _, _ := navApp(t)
	// 可见顺序：alpha, beta, dir, child
	a.selectNode(a.coll.Requests[0].ID)
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if selName(a) != "beta" {
		t.Fatalf("↓ 应选中 beta: %s", selName(a))
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if selName(a) != "child" {
		t.Fatalf("再 ↓↓ 应到 child: %s", selName(a))
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if selName(a) != "child" {
		t.Fatalf("到底后 ↓ 不应越界: %s", selName(a))
	}
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	if selName(a) != "dir" {
		t.Fatalf("↑ 应回到 dir: %s", selName(a))
	}
}

// ← 先折叠展开的文件夹，再选中父文件夹；→ 先展开，再进入第一个子项。
func TestSidebarLeftRight(t *testing.T) {
	a, tt, dir, child := navApp(t)
	a.selectNode(dir)
	tt.Frame()
	tt.Key(0, ui.KeyLeft)
	tt.Frame()
	if a.side.outline.Open.Has(dir) {
		t.Fatal("← 应折叠已展开的文件夹")
	}
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if !a.side.outline.Open.Has(dir) {
		t.Fatal("→ 应展开折叠的文件夹")
	}
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if a.selectedNode() != child {
		t.Fatalf("再 → 应进入第一个子项: %s", selName(a))
	}
	tt.Key(0, ui.KeyLeft)
	tt.Frame()
	if a.selectedNode() != dir {
		t.Fatalf("子项上 ← 应回到父文件夹: %s", selName(a))
	}
}

// 回车：请求在标签中打开，文件夹展开/折叠。
func TestSidebarEnterOpens(t *testing.T) {
	a, tt, dir, _ := navApp(t)
	beta := a.coll.Requests[1].ID
	a.selectNode(beta)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.findTab(beta) < 0 {
		t.Fatal("回车应打开 beta")
	}
	a.selectNode(dir)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.side.outline.Open.Has(dir) {
		t.Fatal("回车应折叠已展开的文件夹")
	}
}

// 输入框有焦点、对话框打开或不在“请求”页签时，方向键不动侧栏选中项。
func TestSidebarArrowKeysIgnoredWhenBusy(t *testing.T) {
	a, tt, _, _ := navApp(t)
	alpha := a.coll.Requests[0].ID
	a.selectNode(alpha)
	a.openPrefs()
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if a.selectedNode() != alpha {
		t.Fatal("对话框打开时方向键不应移动选中项")
	}
	a.overlays.prefs.open = false
	a.side.tab = 1
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if a.selectedNode() != alpha {
		t.Fatal("在历史页签方向键不应移动请求选中项")
	}
}

// 有搜索词时，导航只在匹配的请求之间进行。
func TestSidebarArrowKeysFollowSearch(t *testing.T) {
	a, _, _, _ := navApp(t)
	a.side.query = "chi"
	ids := a.visibleNodeIDs()
	if len(ids) != 1 || a.tree.nodes[ids[0]].name != "child" {
		t.Fatalf("搜索 chi 应只剩 child: %v", ids)
	}
}

// 先用鼠标点中一行（该行获得焦点），再用方向键移动，回车必须作用在“新选中的行”上，
// 而不是仍然持有焦点的旧行（真实运行时曾出现回车把旧文件夹折叠的问题）。
func TestSidebarEnterAfterClickActsOnNewSelection(t *testing.T) {
	a, tt, dir, child := navApp(t)
	if err := tt.Click("dir"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.side.outline.Open.Has(dir) {
		// 点击会切换展开状态；保证接下来 ↓ 能走到子项
		a.side.outline.Open.Add(dir)
		tt.Frame()
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if a.selectedNode() != child {
		t.Fatalf("↓ 应选中子项: %s", selName(a))
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.findTab(child) < 0 {
		t.Fatalf("回车应打开子项，当前标签数 %d", len(a.tabs))
	}
	if !a.side.outline.Open.Has(dir) {
		t.Fatal("回车不应折叠旧文件夹")
	}
}
