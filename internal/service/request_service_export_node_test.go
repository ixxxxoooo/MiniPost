package service

import (
	"encoding/json"
	"strings"
	"testing"

	"minipost/internal/model"
)

// 准备：根目录一个请求 Solo；文件夹 Dir 里有 Inner（POST + JSON 请求体）和子文件夹 Sub/Deep。
func seedExportTree(t *testing.T, svc *RequestService, pid string) (dirID, soloID string) {
	t.Helper()
	solo, err := svc.CreateRequest(pid, "", "Solo")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := svc.CreateFolder(pid, "", "Dir")
	if err != nil {
		t.Fatal(err)
	}
	inner, err := svc.CreateRequest(pid, dir.ID, "Inner")
	if err != nil {
		t.Fatal(err)
	}
	inner.Method, inner.URL = "POST", "{{baseUrl}}/inner"
	inner.Body = model.RequestBody{Type: "json", JSON: `{"a":1}`}
	if err := svc.SaveRequest(inner); err != nil {
		t.Fatal(err)
	}
	sub, err := svc.CreateFolder(pid, dir.ID, "Sub")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateRequest(pid, sub.ID, "Deep"); err != nil {
		t.Fatal(err)
	}
	return dir.ID, solo.ID
}

// 导出文件夹：集合名为文件夹名，包含该文件夹本身及全部子项，且不含兄弟节点。
func TestExportNodeJSON_Folder(t *testing.T) {
	svc, _, pid := newImportTestService(t)
	dirID, _ := seedExportTree(t, svc, pid)

	raw, err := svc.ExportNodeJSON(pid, dirID)
	if err != nil {
		t.Fatal(err)
	}
	var col postmanCollection
	if err := json.Unmarshal(raw, &col); err != nil {
		t.Fatal(err)
	}
	if col.Info.Name != "Dir" || len(col.Item) != 1 || col.Item[0].Name != "Dir" {
		t.Fatalf("应只含 Dir 文件夹本身: %+v", col)
	}
	if strings.Contains(string(raw), "Solo") {
		t.Fatal("不应包含兄弟节点 Solo")
	}
	names := []string{}
	var walk func(items []postmanItem)
	walk = func(items []postmanItem) {
		for _, it := range items {
			names = append(names, it.Name)
			walk(it.Item)
		}
	}
	walk(col.Item[0].Item)
	if strings.Join(names, ",") != "Inner,Sub,Deep" {
		t.Fatalf("子项应完整保留层级: %v", names)
	}
}

// 导出单个请求：集合里只有这一个请求。
func TestExportNodeJSON_Request(t *testing.T) {
	svc, _, pid := newImportTestService(t)
	_, soloID := seedExportTree(t, svc, pid)
	raw, err := svc.ExportNodeJSON(pid, soloID)
	if err != nil {
		t.Fatal(err)
	}
	var col postmanCollection
	if err := json.Unmarshal(raw, &col); err != nil {
		t.Fatal(err)
	}
	if len(col.Item) != 1 || col.Item[0].Name != "Solo" || col.Item[0].Request == nil {
		t.Fatalf("应只含 Solo 请求: %+v", col)
	}
}

func TestExportNodeJSON_UnknownNode(t *testing.T) {
	svc, _, pid := newImportTestService(t)
	if _, err := svc.ExportNodeJSON(pid, "nope"); err == nil {
		t.Fatal("不存在的节点应返回错误")
	}
}

// 导出再导入：在新项目里还原出同名文件夹、子文件夹与请求内容。
func TestExportNodeJSON_RoundTripThroughImport(t *testing.T) {
	svc, store, pid := newImportTestService(t)
	dirID, _ := seedExportTree(t, svc, pid)
	raw, err := svc.ExportNodeJSON(pid, dirID)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.SaveProject(&model.Project{ID: "p2", Name: "Other"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.ImportPostmanCollection("p2", raw); err != nil {
		t.Fatal(err)
	}
	data, err := svc.GetCollectionData("p2")
	if err != nil {
		t.Fatal(err)
	}
	folders := map[string]bool{}
	for _, f := range data.Folders {
		folders[f.Name] = true
	}
	if !folders["Dir"] || !folders["Sub"] {
		t.Fatalf("应还原 Dir 与 Sub 文件夹: %v", folders)
	}
	var inner *model.RequestItem
	for i := range data.Requests {
		if data.Requests[i].Name == "Inner" {
			inner = &data.Requests[i]
		}
	}
	if inner == nil || inner.Method != "POST" || inner.Body.JSON != `{"a":1}` || inner.URL != "{{baseUrl}}/inner" {
		t.Fatalf("Inner 的方法/地址/请求体应保持一致: %+v", inner)
	}
}
