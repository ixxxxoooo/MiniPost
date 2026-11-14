package gui

import (
	"strings"
	"testing"
)

// 请求与文件夹的右键菜单都有“导出 JSON…”，点击后按节点名保存 Postman 集合文件。
func TestExportNodeJSONFromContextMenu(t *testing.T) {
	_, tt, be := sideDragApp(t)
	for _, c := range []struct{ node, file string }{
		{"beta", "beta.postman_collection.json"},
		{"dir", "dir.postman_collection.json"},
	} {
		if err := tt.RightClick(c.node); err != nil {
			t.Fatal(err)
		}
		if menu := strings.Join(tt.Menu(), "|"); !strings.Contains(menu, "导出 JSON…") {
			t.Fatalf("%s 的菜单缺少“导出 JSON…”: %s", c.node, menu)
		}
		if err := tt.ChooseMenuItem("导出 JSON…"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		last := be.texts[len(be.texts)-1]
		if last.name != c.file || !strings.Contains(last.content, `"node"`) {
			t.Fatalf("应导出 %s: %+v", c.file, last)
		}
	}
}

// 文件名里的非法字符会被替换，避免保存对话框拿到不可用的名字。
func TestExportNodeFileNameSanitized(t *testing.T) {
	a, tt, be := sideDragApp(t)
	id := a.coll.Requests[0].ID
	if err := be.RenameRequest(a.projectID, id, `a/b:c`); err != nil {
		t.Fatal(err)
	}
	a.reloadCollection()
	tt.Frame()
	a.exportNodeOf(id)
	if got := be.texts[len(be.texts)-1].name; got != "a-b-c.postman_collection.json" {
		t.Fatalf("文件名应被清洗: %q", got)
	}
}
