package gui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// crumbApp 建一个两层文件夹 outer/inner，里面放请求 deep 并打开，随后把侧栏折叠起来。
func crumbApp(t *testing.T) (a *app, outerID, innerID string, tt *ui.Tester) {
	t.Helper()
	app, tester, be := zhApp(t)
	outer, _ := be.CreateFolder(app.projectID, "", "outer")
	inner, _ := be.CreateFolder(app.projectID, outer.ID, "inner")
	it, _ := be.CreateRequestItem(app.projectID, inner.ID, "deep")
	app.reloadCollection()
	app.openRequest(it.ID)
	tester.Frame()
	// 打开请求时侧栏会展开祖先；这里折叠回去，模拟“侧栏里找不到当前请求”。
	app.side.outline.Open.Clear()
	tester.Frame()
	return app, outer.ID, inner.ID, tester
}

// 点击面包屑里的文件夹段：侧栏展开其所有祖先并选中该文件夹。
func TestBreadcrumbFolderRevealsInSidebar(t *testing.T) {
	a, outer, inner, tt := crumbApp(t)
	if !tt.HasText("inner") {
		t.Fatalf("面包屑应显示文件夹路径")
	}
	if err := tt.Click("inner"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.side.sel != inner {
		t.Fatalf("应选中 inner 文件夹，实际 %q", a.side.sel)
	}
	if !a.side.outline.Open.Has(outer) || !a.side.outline.Open.Has(inner) {
		t.Fatalf("应展开 outer 和 inner")
	}
}

// 侧栏被收起、停在历史页、带搜索词时，点面包屑也要让目标可见。
func TestBreadcrumbRevealResetsSidebarState(t *testing.T) {
	a, _, inner, tt := crumbApp(t)
	a.set.SidebarCollapsed = true
	a.side.tab = 1
	a.side.query = "zzz"
	tt.Frame()
	a.clickCrumb(crumb{name: "inner", folderID: inner})
	tt.Frame()
	if a.set.SidebarCollapsed || a.side.tab != 0 || a.side.query != "" || a.side.sel != inner {
		t.Fatalf("应展开侧栏、回到请求页并清除搜索: collapsed=%v tab=%d query=%q sel=%q",
			a.set.SidebarCollapsed, a.side.tab, a.side.query, a.side.sel)
	}
}

// 点项目段打开项目下拉面板。
func TestBreadcrumbProjectOpensProjectMenu(t *testing.T) {
	a, _, _, tt := crumbApp(t)
	a.clickCrumb(crumb{name: "Demo"})
	tt.Frame()
	if !a.projMenu.open {
		t.Fatal("点项目段应打开项目下拉面板")
	}
}

// 已被删除的文件夹不应导致状态错乱。
func TestBreadcrumbUnknownFolderIgnored(t *testing.T) {
	a, _, _, _ := crumbApp(t)
	before := a.side.sel
	a.clickCrumb(crumb{name: "gone", folderID: "nope"})
	if a.side.sel != before {
		t.Fatalf("未知文件夹应被忽略，sel=%q", a.side.sel)
	}
}
