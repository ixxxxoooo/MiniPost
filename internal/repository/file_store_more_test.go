package repository

import (
	"os"
	"path/filepath"
	"testing"

	"minipost/internal/model"
)

func saveTestProject(t *testing.T, store *FileStore, id string) {
	t.Helper()
	p := &model.Project{ID: id, Name: "项目 " + id, CreatedAt: "2026-06-13T00:00:00Z", UpdatedAt: "2026-06-13T00:00:00Z"}
	if err := store.SaveProject(p); err != nil {
		t.Fatalf("保存项目失败: %v", err)
	}
}

func testRequest(projectID, id string) *model.RequestItem {
	return &model.RequestItem{ID: id, Name: "Req " + id, Method: "GET", URL: "https://example.com/" + id, ProjectID: projectID,
		Body: model.RequestBody{Type: "none"}, Auth: model.AuthConfig{Type: "none"}}
}

// 数据里有环境变量、Cookie：数据目录、项目目录和 JSON 文件只对当前用户开放，旧目录也会被收紧。
func TestFileStoreUsesPrivatePermissions(t *testing.T) {
	base := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	store, err := NewFileStoreWithDir(base)
	if err != nil {
		t.Fatal(err)
	}
	saveTestProject(t, store, "p1")
	for _, dir := range []string{base, filepath.Join(base, "projects"), filepath.Join(base, "projects", "p1")} {
		fi, err := os.Stat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o700 {
			t.Fatalf("%s 权限应为 0700，实际 %o", dir, fi.Mode().Perm())
		}
	}
	fi, err := os.Stat(filepath.Join(base, "projects", "p1", "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("project.json 权限应为 0600，实际 %o", fi.Mode().Perm())
	}
}

func TestProjectsListAndDelete(t *testing.T) {
	store := newTestFileStore(t)
	saveTestProject(t, store, "a")
	saveTestProject(t, store, "b")
	list, err := store.ListProjects()
	if err != nil || len(list) != 2 {
		t.Fatalf("应有两个项目: %v %v", list, err)
	}
	if err := store.DeleteProject("a"); err != nil {
		t.Fatal(err)
	}
	list, _ = store.ListProjects()
	if len(list) != 1 || list[0].ID != "b" {
		t.Fatalf("删除后应只剩 b: %v", list)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	store := newTestFileStore(t)
	if _, err := store.LoadConfig(); err != nil {
		t.Fatalf("没有配置文件时应返回默认值: %v", err)
	}
	cfg, _ := store.LoadConfig()
	cfg.LastProjectID = "p1"
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadConfig()
	if err != nil || got.LastProjectID != "p1" {
		t.Fatalf("配置应被保存: %+v %v", got, err)
	}
}

func TestEnvironmentsCRUD(t *testing.T) {
	store := newTestFileStore(t)
	saveTestProject(t, store, "p")
	if envs, err := store.ListEnvironments("p"); err != nil || len(envs) != 0 {
		t.Fatalf("初始应为空: %v %v", envs, err)
	}
	env := &model.Environment{ID: "e1", Name: "dev", ProjectID: "p", Variables: []model.Variable{{ID: "v", Key: "k", Value: "1", Enabled: true}}}
	if err := store.SaveEnvironment(env); err != nil {
		t.Fatal(err)
	}
	env.Name = "dev2" // 同 ID 再保存应是更新而不是新增
	if err := store.SaveEnvironment(env); err != nil {
		t.Fatal(err)
	}
	other := &model.Environment{ID: "e2", Name: "prod", ProjectID: "p"}
	if err := store.SaveEnvironment(other); err != nil {
		t.Fatal(err)
	}
	envs, _ := store.ListEnvironments("p")
	if len(envs) != 2 || envs[0].Name != "dev2" {
		t.Fatalf("应更新而非新增: %+v", envs)
	}
	if err := store.DeleteEnvironment("p", "e1"); err != nil {
		t.Fatal(err)
	}
	envs, _ = store.ListEnvironments("p")
	if len(envs) != 1 || envs[0].ID != "e2" {
		t.Fatalf("删除后应只剩 e2: %+v", envs)
	}
}

func TestHistoryAddAndClear(t *testing.T) {
	store := newTestFileStore(t)
	saveTestProject(t, store, "p")
	if err := store.AddHistory("p", &model.HistoryEntry{ID: "h1", Method: "GET", URL: "https://a", StatusCode: 200}); err != nil {
		t.Fatal(err)
	}
	h, err := store.GetHistory("p")
	if err != nil || len(h) != 1 {
		t.Fatalf("应有一条历史: %v %v", h, err)
	}
	if err := store.ClearHistory("p"); err != nil {
		t.Fatal(err)
	}
	if h, _ = store.GetHistory("p"); len(h) != 0 {
		t.Fatalf("清空后应为空: %v", h)
	}
}

func TestRequestsListGetDeleteMove(t *testing.T) {
	store := newTestFileStore(t)
	saveTestProject(t, store, "p")
	if err := store.SaveFolder(&model.Folder{ID: "f1", Name: "F", ProjectID: "p"}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"r1", "r2"} {
		if err := store.SaveRequest(testRequest("p", id)); err != nil {
			t.Fatal(err)
		}
	}
	reqs, err := store.ListRequests("p")
	if err != nil || len(reqs) != 2 {
		t.Fatalf("应有两个请求: %v %v", reqs, err)
	}
	got, err := store.GetRequest("p", "r2")
	if err != nil || got.ID != "r2" {
		t.Fatalf("应取到 r2: %v %v", got, err)
	}
	if _, err := store.GetRequest("p", "missing"); err == nil {
		t.Fatal("不存在的请求应报错")
	}
	// 移进文件夹后，请求的 FolderID 同步更新
	if err := store.MoveRequest("p", "r1", "f1", 0); err != nil {
		t.Fatal(err)
	}
	got, _ = store.GetRequest("p", "r1")
	if got.FolderID != "f1" {
		t.Fatalf("r1 应在 f1 里，实际 %q", got.FolderID)
	}
	if err := store.DeleteRequest("p", "r1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetRequest("p", "r1"); err == nil {
		t.Fatal("删除后不应再取到")
	}
}

func TestMoveFolderIntoDescendantIsRejected(t *testing.T) {
	store := newTestFileStore(t)
	saveTestProject(t, store, "p")
	if err := store.SaveFolder(&model.Folder{ID: "a", Name: "A", ProjectID: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFolder(&model.Folder{ID: "b", Name: "B", ProjectID: "p", ParentID: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := store.MoveFolder("p", "a", "b", 0); err == nil {
		t.Fatal("把文件夹移进自己的子文件夹应报错，否则会形成环")
	}
}
