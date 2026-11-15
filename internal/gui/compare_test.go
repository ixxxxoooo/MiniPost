package gui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"minipost/internal/model"
)

// 本文件是「设计稿对照截图」：用与 docs/design 原型一致的演示数据，在 1180×740 的窗口里复现设计稿的各个屏幕，
// 设置 MINIPOST_COMPARE_DIR 后把每个屏幕保存为 <屏幕 ID>.png，再与原型（方案 C）的同名截图逐屏并排对比。
// 屏幕 ID 与 docs/design/assets/screens.js 的 SCREENS 一致。未设置环境变量时只跑一遍场景，保证它们不会崩溃。

const (
	compareW = 1180
	compareH = 740
)

// designProject 往假后端里写入与设计稿一致的演示数据（项目「Acme 开放平台」、四个文件夹、十二个请求、开发环境等）。
func designProject(be *fakeBackend) string {
	pid := be.id("p")
	now := time.Now().Format(time.RFC3339)
	be.projects = append(be.projects,
		model.Project{ID: pid, Name: "Acme 开放平台", Description: "电商后台 API 联调", CreatedAt: now, UpdatedAt: now},
		model.Project{ID: be.id("p"), Name: "支付网关", Description: "三方支付回调与对账", ThemeColor: "#30d158", CreatedAt: now, UpdatedAt: now},
		model.Project{ID: be.id("p"), Name: "AI 对话服务", Description: "SSE 流式接口与鉴权", ThemeColor: "#ff9f0a", CreatedAt: now, UpdatedAt: now},
		model.Project{ID: be.id("p"), Name: "内部运营后台", ThemeColor: "#ff375f", CreatedAt: now, UpdatedAt: now},
		model.Project{ID: be.id("p"), Name: "Mock 沙盒", Description: "前端联调用的假数据", ThemeColor: "#00c7be", CreatedAt: now, UpdatedAt: now},
	)
	mk := func(id, name, method, url, folder string, so int) model.RequestItem {
		return model.RequestItem{ID: id, Name: name, Method: method, URL: url, ProjectID: pid, FolderID: folder, SortOrder: so,
			Body: model.RequestBody{Type: "none"}, Auth: model.AuthConfig{Type: "none"}}
	}
	cd := &model.CollectionData{Folders: []model.Folder{
		{ID: "f1", Name: "用户服务", ProjectID: pid, SortOrder: 0},
		{ID: "f2", Name: "订单服务", ProjectID: pid, SortOrder: 1},
		{ID: "f3", Name: "流式 / AI", ProjectID: pid, SortOrder: 2},
		{ID: "f4", Name: "认证", ProjectID: pid, SortOrder: 3},
	}}
	r1 := mk("r1", "获取用户列表", "GET", "{{host}}/users?page=1&size=20", "f1", 0)
	r1.Params = []model.KeyValue{{Key: "page", Value: "1", Description: "页码，从 1 开始"}, {Key: "size", Value: "20", Description: "每页条数，最大 100"}, {Key: "keyword", Disabled: true}}
	r1.Headers = []model.KeyValue{{Key: "Accept", Value: "application/json"}, {Key: "X-Request-Id", Value: "{{$randomUUID}}", Description: "链路追踪"}, {Key: "Accept-Language", Value: "zh-CN", Disabled: true}}
	r1.Auth = model.AuthConfig{Type: "bearer", Bearer: model.BearerAuth{Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.demo"}}
	r2 := mk("r2", "创建用户", "POST", "{{host}}/users", "f1", 1)
	r2.Headers = []model.KeyValue{{Key: "Content-Type", Value: "application/json"}, {Key: "Accept", Value: "application/json"}}
	r2.Body = model.RequestBody{Type: "json", JSON: "{\n  \"name\": \"Alice\",\n  \"email\": \"alice@example.com\",\n  \"roles\": [\"admin\", \"editor\"],\n  \"profile\": {\n    \"age\": 28,\n    \"active\": true,\n    \"avatar\": null\n  }\n}"}
	r2.Auth = r1.Auth
	r3 := mk("r3", "更新用户资料", "PUT", "{{host}}/users/1024", "f1", 2)
	r3.Headers = []model.KeyValue{{Key: "Content-Type", Value: "multipart/form-data"}}
	r3.Body = model.RequestBody{Type: "form-data", FormData: []model.FormData{
		{Key: "name", Value: "Alice", Type: "text"},
		{Key: "avatar", Type: "file", FileName: "avatar.png", FilePath: "/tmp/avatar.png", Description: "头像，最大 2 MB"},
		{Key: "note", Type: "text", Disabled: true},
	}}
	r3.Auth = r1.Auth
	r4 := mk("r4", "删除用户", "DELETE", "{{host}}/users/1024", "f1", 3)
	r5 := mk("r5", "订单列表", "GET", "{{host}}/orders?status=paid", "f2", 0)
	r6 := mk("r6", "创建订单", "POST", "{{host}}/orders", "f2", 1)
	r9 := mk("r9", "修改收货地址", "PATCH", "{{host}}/orders/8801/address", "f2", 2)
	r7 := mk("r7", "Chat 流式补全", "POST", "{{host}}/v1/chat/stream", "f3", 0)
	r10 := mk("r10", "SSE 事件订阅", "GET", "{{host}}/events?count=5", "f3", 1)
	r11 := mk("r11", "登录获取 Token", "POST", "{{host}}/auth/login", "f4", 0)
	r12 := mk("r12", "刷新 Token", "POST", "{{host}}/auth/refresh", "f4", 1)
	r8 := mk("r8", "健康检查", "GET", "{{host}}/healthz", "", 4)
	r11.Headers = []model.KeyValue{{Key: "Content-Type", Value: "application/x-www-form-urlencoded"}}
	r11.Body = model.RequestBody{Type: "form-urlencoded", FormUrlEncoded: []model.KeyValue{{Key: "username", Value: "alice@example.com"}, {Key: "password", Value: "{{password}}"}, {Key: "grant_type", Value: "password"}}}
	r12.Headers = []model.KeyValue{{Key: "Content-Type", Value: "text/plain"}}
	r12.Body = model.RequestBody{Type: "raw", Raw: "refresh_token=rt_2f8a91c0b7d4\nclient_id=minipost-desktop"}
	cd.Requests = []model.RequestItem{r1, r2, r3, r4, r5, r6, r9, r7, r10, r11, r12, r8}
	cd.TreeNodes = []model.CollectionNode{
		{NodeID: "f1", NodeType: model.CollectionNodeTypeFolder, ProjectID: pid, SortOrder: 0},
		{NodeID: "f2", NodeType: model.CollectionNodeTypeFolder, ProjectID: pid, SortOrder: 1},
		{NodeID: "f3", NodeType: model.CollectionNodeTypeFolder, ProjectID: pid, SortOrder: 2},
		{NodeID: "f4", NodeType: model.CollectionNodeTypeFolder, ProjectID: pid, SortOrder: 3},
		{NodeID: "r8", NodeType: model.CollectionNodeTypeRequest, ProjectID: pid, SortOrder: 4},
	}
	for _, r := range cd.Requests {
		if r.FolderID != "" {
			cd.TreeNodes = append(cd.TreeNodes, model.CollectionNode{NodeID: r.ID, NodeType: model.CollectionNodeTypeRequest, ProjectID: pid, ParentFolderID: r.FolderID, SortOrder: r.SortOrder})
		}
	}
	be.colls[pid] = cd
	be.envs[pid] = []model.Environment{
		{ID: "e1", Name: "开发环境", ProjectID: pid, Variables: []model.Variable{
			{ID: "v1", Key: "host", Value: "https://api.dev.acme.io", Enabled: true},
			{ID: "v2", Key: "token", Value: "dev-9c1f2a7b-3d4e", Enabled: true, IsSecret: true},
			{ID: "v3", Key: "tenant", Value: "acme-dev", Enabled: true},
			{ID: "v4", Key: "debug", Value: "true"},
		}},
		{ID: "e2", Name: "预发布", ProjectID: pid, Variables: []model.Variable{{ID: "v5", Key: "host", Value: "https://api.stg.acme.io", Enabled: true}, {ID: "v6", Key: "token", Value: "stg-••••", Enabled: true, IsSecret: true}}},
		{ID: "e3", Name: "生产环境", ProjectID: pid, Variables: []model.Variable{{ID: "v7", Key: "host", Value: "https://api.acme.io", Enabled: true}, {ID: "v8", Key: "token", Value: "prod-••••", Enabled: true, IsSecret: true}}},
	}
	ts := func(m int) string { return time.Now().Add(-time.Duration(m) * time.Minute).Format(time.RFC3339) }
	be.history[pid] = []model.HistoryEntry{
		{ID: "h1", Name: "获取用户列表", Method: "GET", URL: "https://api.dev.acme.io/users?page=1&size=20", StatusCode: 200, Duration: 128, Size: 1200, Timestamp: ts(5)},
		{ID: "h2", Name: "创建用户", Method: "POST", URL: "https://api.dev.acme.io/users", StatusCode: 201, Duration: 246, Size: 312, Timestamp: ts(13)},
		{ID: "h3", Name: "创建订单", Method: "POST", URL: "https://api.dev.acme.io/orders", StatusCode: 422, Duration: 91, Size: 208, Timestamp: ts(27)},
		{ID: "h4", Name: "健康检查", Method: "GET", URL: "https://api.dev.acme.io/healthz", StatusCode: 0, Timestamp: ts(44)},
	}
	be.resp = &model.HttpResponse{
		StatusCode: 200, StatusText: "200 OK", ContentType: "application/json; charset=utf-8", Duration: 128, Size: 1186,
		Headers: map[string][]string{
			"Content-Type": {"application/json; charset=utf-8"}, "Content-Length": {"1186"}, "Server": {"nginx/1.25.3"}, "Cache-Control": {"no-store"},
			"X-Request-Id": {"9f2c41ab-77de-4c0a-b1d5-2e8f3a6c91d0"},
			"Date":         {"Sun, 15 Nov 2026 01:41:07 GMT"},
			"Set-Cookie":   {"sid=a1b2c3d4e5f6; Path=/; Secure; HttpOnly", "theme=dark; Path=/; Expires=Mon, 15 Nov 2027 00:00:00 GMT"},
		},
		Body: "{\n  \"code\": 0,\n  \"message\": \"ok\",\n  \"data\": {\n    \"total\": 128,\n    \"page\": 1,\n    \"users\": [\n      {\n        \"id\": 1024,\n        \"name\": \"Alice\",\n        \"email\": \"alice@example.com\",\n        \"roles\": [\"admin\", \"editor\"],\n        \"active\": true,\n        \"balance\": 1280.5,\n        \"lastLogin\": \"2026-11-15T09:41:07Z\"\n      },\n      {\n        \"id\": 1025,\n        \"name\": \"Bob\",\n        \"email\": \"bob@example.com\",\n        \"roles\": [\"viewer\"],\n        \"active\": false,\n        \"balance\": 0,\n        \"lastLogin\": null\n      }\n    ]\n  }\n}",
	}
	return pid
}

// compareApp 构造与设计稿初始状态一致的应用：已打开项目、选择开发环境、侧栏 240 宽，窗口 1180×740。
func compareApp(t *testing.T, dark bool, scheme string) (*app, *ui.Tester, *fakeBackend) {
	t.Helper()
	be := newFakeBackend(t)
	pid := designProject(be)
	a := newApp(be)
	a.set.Locale = localeZH
	a.set.Theme = "light"
	if dark {
		a.set.Theme = "dark"
	}
	if scheme != "" {
		a.set.ColorScheme = scheme
	}
	a.set.SidebarWidth = 232
	a.selectProject(pid)
	a.setEnv("e1")
	tt := ui.NewTester(a.view, compareW, compareH)
	tt.Frame()
	return a, tt, be
}

// openTabs 依次打开若干请求并把第 active 个设为当前标签（与设计稿默认的三标签一致）。
func openTabs(a *app, tt *ui.Tester, active int, ids ...string) {
	for _, id := range ids {
		a.openRequest(id)
		tt.Frame()
	}
	a.active = active
	a.side.sel = a.tabs[active].draft.ID
	tt.Frame()
}

// compareScene 描述一个与设计稿对应的屏幕：id 同 screens.js，setup 把应用带到对应状态。
type compareScene struct {
	id    string
	setup func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend)
}

// defaultTabs 是设计稿大多数屏幕的初始标签：获取用户列表（当前）、创建用户（有未保存修改）、订单列表。
func defaultTabs(t *testing.T, a *app, tt *ui.Tester) {
	openTabs(a, tt, 0, "r1", "r2", "r5")
	a.tabs[1].draft.Name += " " // 第二个标签带未保存修改（设计稿里的小圆点）
	a.tabs[1].refreshDirty()
	if err := tt.Click("发送"); err != nil { // 当前标签已发送，响应区有内容
		t.Fatal(err)
	}
	tt.Frame()
}

var compareScenes = []compareScene{
	{"welcome", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {}},
	{"req-params", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) { defaultTabs(t, a, tt) }},
	{"resp-ok", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) { defaultTabs(t, a, tt) }},
	{"req-headers", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		_ = tt.Click("Headers")
	}},
	{"req-body", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		openTabs(a, tt, 1, "r1", "r2", "r5")
		_ = tt.Click("Body")
	}},
	{"req-body-form", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		openTabs(a, tt, 0, "r3", "r1", "r5")
		_ = tt.Click("Body")
	}},
	{"req-body-urlenc", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		openTabs(a, tt, 0, "r11", "r1", "r5")
		_ = tt.Click("Body")
	}},
	{"req-body-raw", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		openTabs(a, tt, 0, "r12", "r1", "r5")
		_ = tt.Click("Body")
	}},
	{"tabs-list", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		tt.Press(1151, 67) // 右上角“显示所有标签”图标按钮（无文字，只能按坐标点）
		for i := 0; i < 30; i++ {
			tt.Advance(100 * time.Millisecond)
			tt.Frame()
		}
	}},
	{"req-auth", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		_ = tt.Click("Auth")
	}},
	{"resp-headers", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		_ = tt.Click("响应头")
	}},
	{"resp-cookies", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		_ = tt.Click("Cookies")
	}},
	{"resp-empty", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		openTabs(a, tt, 0, "r1", "r2", "r5")
	}},
	{"hist", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		_ = tt.Click("历史")
	}},
	{"env-list", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		_ = tt.Click("环境")
	}},
	{"env-page", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		_ = tt.Click("环境")
		a.openEnvManagerFor("e1")
	}},
	{"home-grid", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.showHome = true
	}},
	{"home-list", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.showHome, a.homeList = true, true
	}},
	{"home-empty", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		be.projects = nil
		a.loadProjects()
		a.showHome = true
	}},
	{"console", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.setConsoleOpen(true)
	}},
	{"cookies", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.openCookies()
	}},
	{"quick", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.openQuick()
	}},
	{"set-general", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.openPrefs()
	}},
	{"set-appearance", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.openPrefs()
		a.overlays.prefs.section = 1
	}},
	{"set-data", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.openPrefs()
		a.overlays.prefs.section = 2
		// 与设计稿一致的两条备份记录
		a.overlays.prefs.backups = []model.BackupFile{
			{Name: "a.zip", Path: "/tmp/a.zip", Size: 48 << 10, ModTime: time.Date(2026, 11, 15, 10, 42, 0, 0, time.Local)},
			{Name: "b.zip", Path: "/tmp/b.zip", Size: 46 << 10, ModTime: time.Date(2026, 11, 14, 22, 10, 0, 0, time.Local)},
		}
		a.overlays.prefs.backupsLoaded = true
	}},
	{"set-shortcuts", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.openPrefs()
		a.overlays.prefs.section = 3
	}},
	{"set-about", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.openPrefs()
		a.overlays.prefs.section = 4
	}},
	{"dlg-save", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.overlays.askClose(a.tabs[1])
	}},
	{"dlg-delete", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.overlays.askConfirm("删除“创建用户”？", "此操作无法撤销。", "删除", func() {})
		a.overlays.confirm.danger = true
	}},
	{"dlg-prompt", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.overlays.askText("重命名", "名称", "创建用户", "确定", func(string) {})
	}},
	{"toast-all", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.toastErr("请求失败：ECONNREFUSED，无法连接到 127.0.0.1:8080")
		a.toastWarn("响应较大（8.2 MB），仅显示前 2 MB")
		a.toastOK("已复制为 cURL")
		a.toast("已切换到「开发环境」")
		tt.Frame()
		tt.Frame()
	}},
	{"quick-q", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.openQuick()
		a.overlays.quick.query = "user"
	}},
	{"side-search", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.side.query = "user"
	}},
	{"side-collapsed", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.set.SidebarCollapsed = true
	}},
	{"imp-curl", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.openImport(2)
	}},
	{"imp-file", func(t *testing.T, a *app, tt *ui.Tester, be *fakeBackend) {
		defaultTabs(t, a, tt)
		a.openImport(0)
	}},
}

// TestCompareScenes 逐个复现设计稿屏幕；设置 MINIPOST_COMPARE_DIR 时输出截图（亮/暗各一套，文件名带 light/dark）。
func TestCompareScenes(t *testing.T) {
	dir := os.Getenv("MINIPOST_COMPARE_DIR")
	scheme := os.Getenv("MINIPOST_COMPARE_SCHEME")
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []string{"light", "dark"} {
		for _, sc := range compareScenes {
			sc := sc
			t.Run(mode+"/"+sc.id, func(t *testing.T) {
				a, tt, be := compareApp(t, mode == "dark", scheme)
				sc.setup(t, a, tt, be)
				tt.Frame()
				for i := 0; i < 6; i++ { // 走完弹层/遮罩的入场动画再截图
					tt.Advance(100 * time.Millisecond)
					tt.Frame()
				}
				if dir == "" {
					return
				}
				f, err := os.Create(filepath.Join(dir, "app-"+mode+"-"+sc.id+".png"))
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				if err := png.Encode(f, tt.Image()); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
