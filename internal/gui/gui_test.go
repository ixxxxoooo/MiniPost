package gui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// zhApp 创建固定中文界面的测试应用，并选中一个带两条请求的项目。
func zhApp(t *testing.T) (*app, *ui.Tester, *fakeBackend) {
	t.Helper()
	be := newFakeBackend(t)
	pid := be.seedProject("Demo", "alpha", "beta")
	a := newApp(be)
	a.set.Locale = localeZH
	a.selectProject(pid)
	tt := ui.NewTester(a.view, 1200, 760)
	tt.Frame()
	return a, tt, be
}

// snapshot 在设置了 MINIPOST_SNAPSHOT_DIR 时把当前画面保存为 PNG，便于人工查看界面效果。
func snapshot(t *testing.T, tt *ui.Tester, name string) {
	dir := os.Getenv("MINIPOST_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

func TestHomeShownWhenNoProject(t *testing.T) {
	be := newFakeBackend(t)
	a := newApp(be)
	a.set.Locale = localeZH
	tt := ui.NewTester(a.view, 1200, 760)
	if !tt.HasText("项目") || !tt.HasText("新建项目") {
		t.Fatalf("没有项目时应显示项目首页，实际文本 %q", tt.Texts())
	}
}

func TestCreateProjectFromHome(t *testing.T) {
	be := newFakeBackend(t)
	a := newApp(be)
	a.set.Locale = localeZH
	tt := ui.NewTester(a.view, 1200, 760)
	if err := tt.Click("新建项目"); err != nil {
		t.Fatal(err)
	}
	tt.Type("我的项目")
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	if len(be.projects) != 1 || be.projects[0].Name != "我的项目" {
		t.Fatalf("应创建项目，实际 %+v", be.projects)
	}
	if a.projectID != be.projects[0].ID || a.showHome {
		t.Fatalf("创建后应进入新项目")
	}
}

func TestOpenRequestFromSidebar(t *testing.T) {
	a, tt, _ := zhApp(t)
	if !tt.HasText("alpha") || !tt.HasText("beta") {
		t.Fatalf("侧栏应列出请求，实际 %q", tt.Texts())
	}
	if err := tt.Click("alpha"); err != nil {
		t.Fatal(err)
	}
	if len(a.tabs) != 1 || a.tabs[0].draft.Name != "alpha" {
		t.Fatalf("点击后应打开 alpha 标签，实际 %d 个标签", len(a.tabs))
	}
	snapshot(t, tt, "open-request")
}

func TestSendShowsResponse(t *testing.T) {
	a, tt, be := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if len(be.sent) != 1 || be.sent[0].URL != "https://example.com/alpha" {
		t.Fatalf("应发送一次请求，实际 %+v", be.sent)
	}
	if !tt.HasText("200 OK") {
		t.Fatalf("应显示状态码，实际 %q", tt.Texts())
	}
	if len(a.console) != 1 || !a.console[0].done || a.console[0].status != 200 {
		t.Fatalf("控制台应记录完成的请求，实际 %+v", a.console)
	}
	snapshot(t, tt, "sent")
}

func TestSendFailureShowsError(t *testing.T) {
	a, tt, be := zhApp(t)
	be.sendErr = os.ErrDeadlineExceeded
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("请求失败") {
		t.Fatalf("失败时应显示错误，实际 %q", tt.Texts())
	}
	if a.activeTab().sending {
		t.Fatalf("失败后不应保持发送中")
	}
}

func TestEditMarksDirtyAndSave(t *testing.T) {
	a, tt, be := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tab := a.activeTab()
	tab.draft.URL = "https://changed.example"
	tt.Frame()
	if !tab.dirty || !tt.HasText("alpha ●") {
		t.Fatalf("修改后应标记未保存，实际 dirty=%v texts=%q", tab.dirty, tt.Texts())
	}
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	if got := be.colls[a.projectID].Requests[0].URL; got != "https://changed.example" {
		t.Fatalf("保存后后端应收到新地址，实际 %q", got)
	}
	if tab.dirty {
		t.Fatalf("保存后不应再是未保存状态")
	}
}

func TestCloseDirtyTabAsks(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.activeTab().draft.URL = "https://changed.example"
	tt.Frame()
	if err := tt.Click("×"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("不保存") {
		t.Fatalf("关闭未保存标签应询问，实际 %q", tt.Texts())
	}
	if err := tt.Click("不保存"); err != nil {
		t.Fatal(err)
	}
	if len(a.tabs) != 0 {
		t.Fatalf("选择不保存后应关闭标签")
	}
}

func TestCloseCleanTabDirectly(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	if err := tt.Click("×"); err != nil {
		t.Fatal(err)
	}
	if len(a.tabs) != 0 {
		t.Fatalf("没有修改的标签应直接关闭")
	}
}

func TestURLTypingSyncsParams(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tab := a.activeTab()
	tab.draft.URL = "seed"
	tt.Frame()
	if err := tt.Click("URL"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("https://x.test/p?a=1&b=2")
	if got := tab.draft.URL; got != "https://x.test/p?a=1&b=2" {
		t.Fatalf("URL 应被输入，实际 %q", got)
	}
	var keys []string
	for _, r := range tab.draft.Params {
		if !r.IsPlaceholder() {
			keys = append(keys, r.Key+"="+r.Value)
		}
	}
	if len(keys) != 2 || keys[0] != "a=1" || keys[1] != "b=2" {
		t.Fatalf("Params 应与 URL 查询串同步，实际 %v", keys)
	}
}

func TestNewRequestFromSidebar(t *testing.T) {
	a, tt, be := zhApp(t)
	before := len(be.colls[a.projectID].Requests)
	if err := tt.Click("+"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("新建请求"); err != nil {
		t.Fatal(err)
	}
	if got := len(be.colls[a.projectID].Requests); got != before+1 {
		t.Fatalf("应新增一个请求，之前 %d 现在 %d", before, got)
	}
	if len(a.tabs) != 1 {
		t.Fatalf("新建后应自动打开")
	}
}

func TestDeleteRequestClosesTab(t *testing.T) {
	a, tt, be := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	if err := tt.RightClick("beta"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("删除"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("删除"); err != nil {
		t.Fatal(err)
	}
	if len(be.colls[a.projectID].Requests) != 1 {
		t.Fatalf("应删除 beta，实际 %+v", be.colls[a.projectID].Requests)
	}
}

func TestImportCurlCreatesRequest(t *testing.T) {
	a, tt, be := zhApp(t)
	a.overlays.curl = curlState{open: true, text: "curl https://curl.example/x -d hi"}
	tt.Frame()
	if err := tt.Click("导入"); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, r := range be.colls[a.projectID].Requests {
		if r.URL == "https://curl.example/x" && r.Method == "POST" && r.Body.Raw == "hi" {
			found = true
		}
	}
	if !found {
		t.Fatalf("应创建并保存导入的请求，实际 %+v", be.colls[a.projectID].Requests)
	}
}

func TestSessionRestoresUnsavedTab(t *testing.T) {
	a, _, be := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.activeTab().draft.URL = "https://unsaved.example"
	a.persistSession()

	b := newApp(be)
	b.set.Locale = localeZH
	if len(b.tabs) != 1 || b.tabs[0].draft.URL != "https://unsaved.example" {
		t.Fatalf("重启后应恢复未保存的修改，实际 %d 个标签", len(b.tabs))
	}
	b.tabs[0].refreshDirty()
	if !b.tabs[0].dirty {
		t.Fatalf("恢复的标签应仍是未保存状态")
	}
}
