package gui

import "testing"

// 底栏右侧只保留布局切换，不再有侧栏开关；侧栏仍可通过标题栏按钮收起和展开。
func TestBottomBarHasNoSidebarToggle(t *testing.T) {
	a, tt, _ := zhApp(t)
	for _, label := range []string{"收起侧边栏", "展开侧边栏"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("底栏不应再有「%s」按钮", label)
		}
	}
	if _, ok := tt.Find("切换到左右布局"); !ok {
		t.Fatal("底栏的布局切换按钮应保留")
	}

	if err := tt.Click("收起侧边栏 (⌘B)"); err != nil {
		t.Fatalf("标题栏的侧栏开关应保留: %v", err)
	}
	tt.Frame()
	if !a.set.SidebarCollapsed {
		t.Fatal("点标题栏开关后侧栏应收起")
	}
	for _, label := range []string{"收起侧边栏", "展开侧边栏"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("侧栏收起后底栏也不应出现「%s」按钮", label)
		}
	}
	if err := tt.Click("展开侧边栏 (⌘B)"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.set.SidebarCollapsed {
		t.Fatal("再点一次标题栏开关后侧栏应展开")
	}
}
