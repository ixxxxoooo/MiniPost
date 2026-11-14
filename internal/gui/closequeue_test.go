package gui

import "testing"

// 批量关闭多个有未保存修改的标签：逐个询问，互不覆盖。
func TestBatchCloseAsksOneByOne(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	for _, x := range a.tabs {
		x.draft.Name += "*" // 制造未保存修改
	}
	tt.Frame()
	if len(a.tabs) != 2 {
		t.Fatalf("前置条件: 应有 2 个标签, 实际 %d", len(a.tabs))
	}

	a.closeTabs(append([]*tab(nil), a.tabs...))
	tt.Frame()
	if !a.overlays.closing.open {
		t.Fatal("应弹出第一个未保存确认")
	}
	first := a.overlays.closing.tab
	if err := tt.Click("不保存"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if len(a.tabs) != 1 {
		t.Fatalf("第一个标签应已关闭, 剩余 %d", len(a.tabs))
	}
	if !a.overlays.closing.open || a.overlays.closing.tab == first {
		t.Fatal("应继续询问第二个标签，而不是丢失或重复")
	}
	if err := tt.Click("不保存"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if len(a.tabs) != 0 || a.overlays.closing.open {
		t.Fatalf("两个标签都应关闭: tabs=%d open=%v", len(a.tabs), a.overlays.closing.open)
	}
}

// 在批量关闭中点“取消”，剩余标签全部保留。
func TestBatchCloseCancelKeepsRest(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	for _, x := range a.tabs {
		x.draft.Name += "*"
	}
	tt.Frame()
	a.closeTabs(append([]*tab(nil), a.tabs...))
	tt.Frame()
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.tabs) != 2 || len(a.closeQueue) != 0 || a.overlays.closing.open {
		t.Fatalf("取消后应保留全部标签: tabs=%d queue=%d", len(a.tabs), len(a.closeQueue))
	}
}

// 干净标签直接关闭，只对脏标签询问。
func TestBatchCloseSkipsCleanTabs(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	a.tabs[1].draft.Name += "*"
	tt.Frame()
	a.closeTabs(append([]*tab(nil), a.tabs...))
	tt.Frame()
	if len(a.tabs) != 1 || !a.overlays.closing.open || a.overlays.closing.tab != a.tabs[0] {
		t.Fatalf("应只剩脏标签并在询问它: tabs=%d", len(a.tabs))
	}
}
