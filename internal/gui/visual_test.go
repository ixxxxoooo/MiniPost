package gui

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/model"
)

// visualApp 构造与旧版参考截图一致的演示数据（项目 Demo、两个文件夹、四个请求），窗口 1280×800。
func visualApp(t *testing.T, dark bool) (*app, *ui.Tester, *fakeBackend) {
	t.Helper()
	be := newFakeBackend(t)
	pid := be.id("p")
	be.projects = append(be.projects, model.Project{ID: pid, Name: "Demo", Description: "演示项目"})
	mk := func(name, method, url, folder string, so int) model.RequestItem {
		return model.RequestItem{ID: be.id("r"), Name: name, Method: method, URL: url, ProjectID: pid, FolderID: folder, SortOrder: so,
			Body: model.RequestBody{Type: "none"}, Auth: model.AuthConfig{Type: "none"}}
	}
	cd := &model.CollectionData{
		Folders: []model.Folder{{ID: "f1", Name: "用户服务", ProjectID: pid, SortOrder: 0}, {ID: "f2", Name: "订单服务", ProjectID: pid, SortOrder: 1}},
	}
	r1 := mk("获取用户列表", "GET", "https://api.example.com/users?page=1&size=20", "f1", 0)
	r1.Params = []model.KeyValue{{Key: "page", Value: "1"}, {Key: "size", Value: "20"}}
	r2 := mk("创建用户", "POST", "https://api.example.com/users", "f1", 1)
	r3 := mk("删除订单", "DELETE", "https://api.example.com/orders/42", "f2", 0)
	r4 := mk("健康检查", "GET", "https://api.example.com/health", "", 2)
	cd.Requests = []model.RequestItem{r1, r2, r3, r4}
	cd.TreeNodes = []model.CollectionNode{
		{NodeID: "f1", NodeType: model.CollectionNodeTypeFolder, ProjectID: pid, SortOrder: 0},
		{NodeID: "f2", NodeType: model.CollectionNodeTypeFolder, ProjectID: pid, SortOrder: 1},
		{NodeID: r4.ID, NodeType: model.CollectionNodeTypeRequest, ProjectID: pid, SortOrder: 2},
		{NodeID: r1.ID, NodeType: model.CollectionNodeTypeRequest, ProjectID: pid, ParentFolderID: "f1", SortOrder: 0},
		{NodeID: r2.ID, NodeType: model.CollectionNodeTypeRequest, ProjectID: pid, ParentFolderID: "f1", SortOrder: 1},
		{NodeID: r3.ID, NodeType: model.CollectionNodeTypeRequest, ProjectID: pid, ParentFolderID: "f2", SortOrder: 0},
	}
	be.colls[pid] = cd
	be.resp = &model.HttpResponse{
		StatusCode: 200, StatusText: "200 OK", ContentType: "application/json", Duration: 128, Size: 100,
		Headers: map[string][]string{"Content-Type": {"application/json"}, "Server": {"nginx"}},
		Body:    `{"code":0,"message":"ok","data":{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}],"total":2}}`,
	}
	a := newApp(be)
	a.set.Locale = localeZH
	a.set.SidebarWidth = 240
	if dark {
		a.set.Theme = "dark"
	}
	a.selectProject(pid)
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	return a, tt, be
}

func TestVisualShell(t *testing.T) {
	_, tt, _ := visualApp(t, false)
	snapshot(t, tt, "v_empty")
	_ = tt.Click("用户服务")
	_ = tt.Click("订单服务")
	tt.Frame()
	snapshot(t, tt, "v_tree")
}

func TestVisualRequest(t *testing.T) {
	a, tt, _ := visualApp(t, false)
	_ = tt.Click("用户服务")
	_ = tt.Click("获取用户列表")
	tt.Frame()
	snapshot(t, tt, "v_req")
	_ = tt.Click("Headers")
	tt.Frame()
	snapshot(t, tt, "v_hdrs")
	_ = tt.Click("Body")
	tt.Frame()
	snapshot(t, tt, "v_body")
	_ = tt.Click("Auth")
	tt.Frame()
	snapshot(t, tt, "v_auth")
	_ = a
}

func TestVisualResponse(t *testing.T) {
	_, tt, _ := visualApp(t, false)
	_ = tt.Click("用户服务")
	_ = tt.Click("获取用户列表")
	snapshot(t, tt, "v_resp_empty")
	_ = tt.Click("Send")
	tt.Frame()
	snapshot(t, tt, "v_resp")
	if r, ok := tt.Find("128ms"); ok {
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
		snapshot(t, tt, "v_resp_time")
	}
	if r, ok := tt.Find("200 200 OK"); ok {
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
		snapshot(t, tt, "v_resp_status")
	}
}

func TestVisualPrefs(t *testing.T) {
	a, tt, _ := visualApp(t, false)
	a.openPrefs()
	tt.Frame()
	snapshot(t, tt, "v_prefs")
	for i, name := range []string{"外观", "数据", "快捷键", "关于"} {
		if err := tt.Click(name); err != nil {
			t.Fatalf("点击 %s: %v", name, err)
		}
		snapshot(t, tt, "v_prefs"+itoa(i+1))
	}
}

func TestVisualEnv(t *testing.T) {
	a, tt, be := visualApp(t, false)
	e, _ := be.CreateEnvironment(a.projectID, "开发环境")
	be.SaveEnvironment(model.Environment{ID: e.ID, Name: "开发环境", ProjectID: a.projectID, Variables: []model.Variable{{ID: "v1", Key: "host", Value: "https://api.example.com", Enabled: true}, {ID: "v2", Key: "token", Value: "abc", Enabled: true}}})
	a.reloadEnvs()
	a.openEnvManagerFor(e.ID)
	tt.Frame()
	snapshot(t, tt, "v_env")
}

func TestVisualCookiesConsole(t *testing.T) {
	a, tt, _ := visualApp(t, false)
	_ = tt.Click("用户服务")
	_ = tt.Click("获取用户列表")
	_ = tt.Click("Send")
	a.jar.Add(client.Cookie{Domain: "api.example.com", Name: "sid", Value: "abc123", Path: "/", Enabled: true, HTTPOnly: true})
	a.jar.Add(client.Cookie{Domain: "example.org", Name: "theme", Value: "dark", Path: "/", Enabled: true})
	a.setConsoleOpen(true)
	a.console[0].open = true
	tt.Frame()
	snapshot(t, tt, "v_console")
	a.openCookies()
	tt.Frame()
	snapshot(t, tt, "v_cookies")
}

func TestVisualHomeDialogs(t *testing.T) {
	a, tt, _ := visualApp(t, false)
	a.showHome = true
	tt.Frame()
	snapshot(t, tt, "v_home")
	a.showHome = false
	a.openImport(importFile)
	tt.Frame()
	snapshot(t, tt, "v_import_file")
	_ = tt.Click("cURL 导入")
	snapshot(t, tt, "v_import_curl")
	a.overlays.imp.open = false
	a.overlays.askConfirm("确认删除", "确定删除请求“获取用户列表”吗?", "删除", func() {})
	tt.Frame()
	snapshot(t, tt, "v_confirm")
	a.overlays.confirm.open = false
	a.overlays.askClose(0, "获取用户列表")
	tt.Frame()
	snapshot(t, tt, "v_close")
}
