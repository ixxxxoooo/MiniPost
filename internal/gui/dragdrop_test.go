package gui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// drag 模拟鼠标左键从 (x1,y1) 按下，分几步移到 (x2,y2) 再松开。
func drag(tt *ui.Tester, x1, y1, x2, y2 float32) {
	tt.Move(x1, y1)
	tt.Frame()
	tt.Press(x1, y1)
	tt.Frame()
	for i := 1; i <= 6; i++ {
		f := float32(i) / 6
		tt.Move(x1+(x2-x1)*f, y1+(y2-y1)*f)
		tt.Frame()
	}
	tt.Release(x2, y2)
	tt.Frame()
	tt.Frame()
}

func center(t *testing.T, tt *ui.Tester, text string) (float32, float32) {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("界面上找不到 %q", text)
	}
	return r.X + r.W/2, r.Y + r.H/2
}

// 把标签拖到另一个标签上，放下后占据它的位置；选中的仍是原来选中的那个标签。
func TestDragTabReorders(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID) // alpha
	a.openRequest(a.coll.Requests[1].ID) // beta（选中）
	tt.Frame()
	y := float32(sizeToolbar + sizeTabStrip/2)
	x0 := a.set.SidebarWidth + 90
	x1 := a.set.SidebarWidth + a.tabs[0].shownW + 40 // 第二个标签（宽度自适应）
	drag(tt, x0, y, x1, y)
	if len(a.tabs) != 2 || a.tabs[0].title() != "beta" || a.tabs[1].title() != "alpha" {
		t.Fatalf("拖动后顺序应为 beta, alpha: %s, %s", a.tabs[0].title(), a.tabs[1].title())
	}
	if a.activeTab().title() != "beta" {
		t.Fatalf("选中的应仍是 beta, 实际 %s", a.activeTab().title())
	}
}

// 轻点标签（没有拖动）仍然是切换标签，不会被当成拖拽。
func TestClickTabStillActivates(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	tt.Frame()
	y := float32(sizeToolbar + sizeTabStrip/2)
	x0 := a.set.SidebarWidth + 90
	tt.Press(x0, y)
	tt.Release(x0, y)
	tt.Frame()
	if a.active != 0 || a.tabs[0].title() != "alpha" {
		t.Fatalf("点击应选中 alpha: active=%d", a.active)
	}
}

func sideDragApp(t *testing.T) (*app, *ui.Tester, *fakeBackend) {
	a, tt, be := zhApp(t)
	f, err := be.CreateFolder(a.projectID, "", "dir")
	if err != nil {
		t.Fatal(err)
	}
	a.reloadCollection()
	a.side.outline.Open.Add(f.ID)
	tt.Frame()
	return a, tt, be
}

func parentOf(a *app, name string) string {
	for id, n := range a.tree.nodes {
		if n.name == name {
			return a.tree.nodes[id].parent
		}
	}
	return "?"
}

// 把请求拖到文件夹上：移进该文件夹；再拖到“根目录”上：回到根目录。
func TestDragRequestIntoFolderAndBackToRoot(t *testing.T) {
	a, tt, _ := sideDragApp(t)
	ax, ay := center(t, tt, "alpha")
	fx, fy := center(t, tt, "dir")
	drag(tt, ax, ay, fx, fy)
	var dir string
	for id, n := range a.tree.nodes {
		if n.name == "dir" {
			dir = id
		}
	}
	if got := parentOf(a, "alpha"); got != dir {
		t.Fatalf("alpha 应在 dir 里, 实际父级 %q", got)
	}
	tt.Frame()
	ax, ay = center(t, tt, "alpha")
	// 根目录没有单独的行：拖到请求树下方的空白区即可（窗口 1200x760，树只有几行）
	drag(tt, ax, ay, 100, 520)
	if got := parentOf(a, "alpha"); got != "" {
		t.Fatalf("alpha 应回到根目录, 实际父级 %q", got)
	}
}

// 把请求拖到同级另一个请求上：插到它之前（重新排序）。
func TestDragRequestReorders(t *testing.T) {
	a, tt, _ := sideDragApp(t)
	order := func() []string {
		var out []string
		for _, id := range a.tree.kids[""] {
			out = append(out, a.tree.nodes[id].name)
		}
		return out
	}
	if got := order(); len(got) != 3 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("前置顺序不对: %v", got)
	}
	bx, by := center(t, tt, "beta")
	ax, ay := center(t, tt, "alpha")
	drag(tt, bx, by, ax, ay)
	if got := order(); got[0] != "beta" || got[1] != "alpha" {
		t.Fatalf("beta 应排到 alpha 之前: %v", got)
	}
}

// 文件夹不能拖进它自己的子文件夹。
func TestDropFolderIntoDescendantIgnored(t *testing.T) {
	a, _, be := sideDragApp(t)
	var dir string
	for id, n := range a.tree.nodes {
		if n.name == "dir" {
			dir = id
		}
	}
	sub, _ := be.CreateFolder(a.projectID, dir, "sub")
	a.reloadCollection()
	a.dropOnNode(dir, sub.ID)
	if a.tree.nodes[dir].parent != "" {
		t.Fatal("文件夹不应被移进自己的子文件夹")
	}
}
