package gui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"minipost/internal/model"
	"minipost/internal/service"
)

func TestEnvManagerCreateAndSave(t *testing.T) {
	a, tt, be := zhApp(t)
	a.openEnvManager()
	tt.Frame()
	if err := tt.Click("新建环境"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Prod")
	if err := tt.Click("创建"); err != nil {
		t.Fatal(err)
	}
	envs := be.envs[a.projectID]
	if len(envs) != 1 || envs[0].Name != "Prod" {
		t.Fatalf("应创建环境 Prod，实际 %+v", envs)
	}
	d := a.overlays.envMgr.draft
	if d == nil {
		t.Fatalf("创建后应载入编辑草稿")
	}
	d.Variables[0].Key, d.Variables[0].Value = "host", "api.test"
	tt.Frame()
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	saved := be.envs[a.projectID][0].Variables
	if len(saved) != 1 || saved[0].Key != "host" || saved[0].Value != "api.test" {
		t.Fatalf("应保存变量且不含空白行，实际 %+v", saved)
	}
}

func TestEnvVariablesResolvedOnSend(t *testing.T) {
	a, tt, be := zhApp(t)
	e, _ := be.CreateEnvironment(a.projectID, "Dev")
	be.SaveEnvironment(model.Environment{ID: e.ID, Name: "Dev", ProjectID: a.projectID, Variables: []model.Variable{{ID: "v1", Key: "host", Value: "api.test", Enabled: true}}})
	a.reloadEnvs()
	a.setEnv(e.ID)
	a.openRequest(a.coll.Requests[0].ID)
	a.activeTab().draft.URL = "https://{{host}}/x"
	tt.Frame()
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if len(be.sent) != 1 || be.sent[0].URL != "https://api.test/x" {
		t.Fatalf("发送时应替换环境变量，实际 %+v", be.sent)
	}
	if a.set.EnvByProject[a.projectID] != e.ID {
		t.Fatalf("所选环境应按项目记住")
	}
}

func TestSettingsApplyAndPersist(t *testing.T) {
	a, tt, be := zhApp(t)
	a.set.FontSize = 15
	a.set.Locale = localeEN
	a.applySettings()
	tt.Frame()
	if !tt.HasText("Settings") {
		t.Fatalf("切换为英文后界面应使用英文，实际 %q", tt.Texts())
	}
	loaded := a.setStore.Load()
	if loaded.FontSize != 15 || loaded.Locale != localeEN {
		t.Fatalf("设置应落盘，实际 %+v", loaded)
	}
	b := newApp(be)
	if b.set.FontSize != 15 {
		t.Fatalf("重启后应读取已保存的设置")
	}
}

func TestSettingsDialogOpens(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openPrefs()
	tt.Frame()
	if !tt.HasText("自动跟随重定向") && !tt.HasText("语言") {
		t.Fatalf("设置对话框应显示选项，实际 %q", tt.Texts())
	}
	if err := tt.Click("请求"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("校验 SSL 证书") {
		t.Fatalf("请求页应显示 SSL 选项，实际 %q", tt.Texts())
	}
	snapshot(t, tt, "prefs")
}

func TestCookiesAddAndSent(t *testing.T) {
	a, tt, be := zhApp(t)
	a.openCookies()
	s := &a.overlays.cookies
	s.adding.Domain, s.adding.Name, s.adding.Value = "example.com", "sid", "abc"
	tt.Frame()
	if err := tt.Click("添加"); err != nil {
		t.Fatal(err)
	}
	if got := a.jar.All(); len(got) != 1 || got[0].Name != "sid" {
		t.Fatalf("应添加 Cookie，实际 %+v", got)
	}
	s.open = false
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	var cookie string
	for _, h := range be.sent[0].Headers {
		if strings.EqualFold(h.Key, "cookie") {
			cookie = h.Value
		}
	}
	if !strings.Contains(cookie, "sid=abc") {
		t.Fatalf("发送时应带上 Jar 中的 Cookie，实际 %q", cookie)
	}
}

func TestImportPreviewAppliesStrategy(t *testing.T) {
	a, tt, be := zhApp(t)
	a.showImportPreview(&service.ImportPreview{Kind: "postman", ConflictCount: 1, Conflicts: []service.ImportConflict{{Method: "GET", URL: "https://example.com/alpha", Name: "alpha"}}}, nil)
	tt.Frame()
	if !tt.HasText("GET https://example.com/alpha  (alpha)") {
		t.Fatalf("应列出冲突，实际 %q", tt.Texts())
	}
	a.overlays.imp.strategy = 1
	if err := tt.Click("导入"); err != nil {
		t.Fatal(err)
	}
	if be.importedStrategy != service.ImportStrategyUpdate {
		t.Fatalf("应按所选策略导入，实际 %q", be.importedStrategy)
	}
}

func TestConsolePanelToggle(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.setConsoleOpen(true)
	tt.Frame()
	if !tt.HasText("发送请求后会在这里显示记录") {
		t.Fatalf("打开控制台后应显示面板，实际 %q", tt.Texts())
	}
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("https://example.com/alpha") {
		t.Fatalf("控制台应记录请求地址")
	}
	snapshot(t, tt, "console")
}

func TestHistoryOpensRequest(t *testing.T) {
	a, tt, be := zhApp(t)
	be.history[a.projectID] = []model.HistoryEntry{{ID: "h1", RequestID: a.coll.Requests[1].ID, Method: "GET", URL: "https://example.com/beta", StatusCode: 200, Duration: 5, Timestamp: "2026-11-14T10:00:00Z"}}
	a.reloadHistory()
	if err := tt.Click("历史"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("https://example.com/beta"); err != nil {
		t.Fatal(err)
	}
	if len(a.tabs) != 1 || a.tabs[0].draft.Name != "beta" {
		t.Fatalf("点击历史应打开对应请求，实际 %d 个标签", len(a.tabs))
	}
}

func TestProjectSwitchKeepsTabs(t *testing.T) {
	a, _, be := zhApp(t)
	p1 := a.projectID
	p2 := be.seedProject("Other", "gamma")
	a.loadProjects()
	a.openRequest(a.coll.Requests[0].ID)
	a.selectProject(p2)
	if len(a.tabs) != 0 {
		t.Fatalf("切换项目后应清空标签")
	}
	a.selectProject(p1)
	if len(a.tabs) != 1 || a.tabs[0].draft.Name != "alpha" {
		t.Fatalf("切回项目应恢复标签，实际 %d 个", len(a.tabs))
	}
}

func TestHeadersTabListsAutoHeaders(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	if err := tt.Click("Headers"); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, s := range tt.Texts() {
		if strings.HasPrefix(s, "自动生成的请求头 (") {
			found = true
		}
	}
	if !found {
		t.Fatalf("Headers 页应显示自动请求头折叠区，实际 %q", tt.Texts())
	}
	a.activeTab().autoOpen = true
	tt.Frame()
	if !tt.HasText("User-Agent") || !tt.HasText("Host") {
		t.Fatalf("展开后应列出自动请求头，实际 %q", tt.Texts())
	}
}

func TestFormatJSONBody(t *testing.T) {
	a, _, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tab := a.activeTab()
	tab.draft.Body.Type = "json"
	tab.jsonEd.push(`{"a":1,"b":[1,2]}`)
	a.formatJSON(tab)
	tab.flush()
	if !strings.Contains(tab.draft.Body.JSON, "\n  \"a\": 1") {
		t.Fatalf("应格式化为缩进 JSON，实际 %q", tab.draft.Body.JSON)
	}
	tab.jsonEd.push(`{bad`)
	a.formatJSON(tab)
	if len(a.toastMsgs) == 0 {
		t.Fatalf("非法 JSON 应提示")
	}
}
