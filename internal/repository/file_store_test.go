package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"minipost/internal/model"

	"github.com/google/uuid"
)

func newTestFileStore(t *testing.T) *FileStore {
	t.Helper()
	store, err := NewFileStoreWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("创建测试存储失败: %v", err)
	}
	return store
}

func TestWriteJSON_AtomicReplacePreservesValidJSON(t *testing.T) {
	store := newTestFileStore(t)
	path := filepath.Join(store.BaseDir(), "sample.json")

	first := map[string]string{"version": "1"}
	if err := store.writeJSON(path, first); err != nil {
		t.Fatalf("首次写入失败: %v", err)
	}

	second := map[string]string{"version": "2", "name": "MiniPost"}
	if err := store.writeJSON(path, second); err != nil {
		t.Fatalf("覆盖写入失败: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取结果失败: %v", err)
	}
	if matches, _ := filepath.Glob(path + ".tmp"); len(matches) > 0 {
		t.Fatalf("不应残留临时文件: %v", matches)
	}

	var decoded map[string]string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("结果不是合法 JSON: %v", err)
	}
	if decoded["version"] != "2" || decoded["name"] != "MiniPost" {
		t.Fatalf("写入内容不符合预期: %#v", decoded)
	}
}

func TestSaveProject_RoundTrip(t *testing.T) {
	store := newTestFileStore(t)
	project := &model.Project{
		ID:        uuid.New().String(),
		Name:      "测试项目",
		CreatedAt: "2026-06-13T00:00:00Z",
		UpdatedAt: "2026-06-13T00:00:00Z",
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("保存项目失败: %v", err)
	}

	loaded, err := store.GetProject(project.ID)
	if err != nil {
		t.Fatalf("读取项目失败: %v", err)
	}
	if loaded.Name != project.Name {
		t.Fatalf("项目名称不一致: got %q want %q", loaded.Name, project.Name)
	}
}

func TestMoveCollectionNode_ReordersSiblings(t *testing.T) {
	store := newTestFileStore(t)
	projectID := uuid.New().String()

	project := &model.Project{
		ID:        projectID,
		Name:      "集合项目",
		CreatedAt: "2026-06-13T00:00:00Z",
		UpdatedAt: "2026-06-13T00:00:00Z",
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("保存项目失败: %v", err)
	}

	folderA := &model.Folder{ID: "folder-a", Name: "A", ProjectID: projectID}
	folderB := &model.Folder{ID: "folder-b", Name: "B", ProjectID: projectID}
	requestA := &model.RequestItem{
		ID:        "request-a",
		Name:      "Req A",
		Method:    "GET",
		URL:       "https://example.com/a",
		ProjectID: projectID,
		Body:      model.RequestBody{Type: "none"},
		Auth:      model.AuthConfig{Type: "none"},
		CreatedAt: "2026-06-13T00:00:00Z",
		UpdatedAt: "2026-06-13T00:00:00Z",
	}
	for _, item := range []*model.Folder{folderA, folderB} {
		if err := store.SaveFolder(item); err != nil {
			t.Fatalf("保存文件夹失败: %v", err)
		}
	}
	if err := store.SaveRequest(requestA); err != nil {
		t.Fatalf("保存请求失败: %v", err)
	}

	if err := store.MoveCollectionNode(projectID, "request-a", model.CollectionNodeTypeRequest, "", 0); err != nil {
		t.Fatalf("移动请求失败: %v", err)
	}

	data, err := store.GetCollectionData(projectID)
	if err != nil {
		t.Fatalf("读取集合数据失败: %v", err)
	}

	rootNodes := make([]string, 0, len(data.TreeNodes))
	for _, node := range data.TreeNodes {
		if node.ParentFolderID == "" {
			rootNodes = append(rootNodes, node.NodeID)
		}
	}
	if len(rootNodes) != 3 {
		t.Fatalf("根节点数量不符合预期: %#v", rootNodes)
	}
	if rootNodes[0] != "request-a" {
		t.Fatalf("请求应位于根节点首位: %#v", rootNodes)
	}
}

func TestDeleteFolder_RemovesDescendants(t *testing.T) {
	store := newTestFileStore(t)
	projectID := uuid.New().String()

	project := &model.Project{
		ID:        projectID,
		Name:      "删除测试",
		CreatedAt: "2026-06-13T00:00:00Z",
		UpdatedAt: "2026-06-13T00:00:00Z",
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("保存项目失败: %v", err)
	}

	parent := &model.Folder{ID: "parent", Name: "Parent", ProjectID: projectID}
	child := &model.Folder{ID: "child", Name: "Child", ProjectID: projectID, ParentID: "parent"}
	req := &model.RequestItem{
		ID:        "child-req",
		Name:      "Child Req",
		Method:    "GET",
		URL:       "https://example.com",
		FolderID:  "child",
		ProjectID: projectID,
		Body:      model.RequestBody{Type: "none"},
		Auth:      model.AuthConfig{Type: "none"},
		CreatedAt: "2026-06-13T00:00:00Z",
		UpdatedAt: "2026-06-13T00:00:00Z",
	}
	for _, folder := range []*model.Folder{parent, child} {
		if err := store.SaveFolder(folder); err != nil {
			t.Fatalf("保存文件夹失败: %v", err)
		}
	}
	if err := store.SaveRequest(req); err != nil {
		t.Fatalf("保存请求失败: %v", err)
	}

	if err := store.DeleteFolder(projectID, "parent"); err != nil {
		t.Fatalf("删除文件夹失败: %v", err)
	}

	data, err := store.GetCollectionData(projectID)
	if err != nil {
		t.Fatalf("读取集合数据失败: %v", err)
	}
	if len(data.Folders) != 0 || len(data.Requests) != 0 {
		t.Fatalf("删除父文件夹后仍残留子节点: folders=%d requests=%d", len(data.Folders), len(data.Requests))
	}
}

func TestGetCollectionData_MigratesLegacyFiles(t *testing.T) {
	store := newTestFileStore(t)
	projectID := uuid.New().String()
	projectDir := filepath.Join(store.BaseDir(), "projects", projectID)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("创建项目目录失败: %v", err)
	}

	// 预置旧版分散文件，模拟历史数据（无 collection.json）。
	foldersJSON := `{"schemaVersion":1,"folders":[{"id":"f1","name":"旧文件夹","projectId":"` + projectID + `","parentId":"","sortOrder":0}]}`
	requestsJSON := `{"schemaVersion":1,"requests":[{"id":"r1","name":"旧请求","method":"GET","url":"https://example.com","folderId":"f1","projectId":"` + projectID + `","body":{"type":"none"},"auth":{"type":"none"},"createdAt":"2026-06-13T00:00:00Z","updatedAt":"2026-06-13T00:00:00Z"}]}`
	if err := os.WriteFile(filepath.Join(projectDir, "folders.json"), []byte(foldersJSON), 0644); err != nil {
		t.Fatalf("写入 folders.json 失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "requests.json"), []byte(requestsJSON), 0644); err != nil {
		t.Fatalf("写入 requests.json 失败: %v", err)
	}

	data, err := store.GetCollectionData(projectID)
	if err != nil {
		t.Fatalf("读取集合数据失败: %v", err)
	}
	if len(data.Folders) != 1 || data.Folders[0].ID != "f1" {
		t.Fatalf("迁移后文件夹不符合预期: %#v", data.Folders)
	}
	if len(data.Requests) != 1 || data.Requests[0].ID != "r1" {
		t.Fatalf("迁移后请求不符合预期: %#v", data.Requests)
	}
	if len(data.TreeNodes) != 2 {
		t.Fatalf("迁移后树节点数量应为 2, 实际 %d", len(data.TreeNodes))
	}

	// 迁移应已生成 collection.json。
	if _, err := os.Stat(filepath.Join(projectDir, "collection.json")); err != nil {
		t.Fatalf("迁移后应生成 collection.json: %v", err)
	}
}

func TestGetCollectionData_DoesNotRewriteOnRead(t *testing.T) {
	store := newTestFileStore(t)
	projectID := uuid.New().String()

	if err := store.SaveProject(&model.Project{
		ID:        projectID,
		Name:      "读不回写",
		CreatedAt: "2026-06-13T00:00:00Z",
		UpdatedAt: "2026-06-13T00:00:00Z",
	}); err != nil {
		t.Fatalf("保存项目失败: %v", err)
	}
	if err := store.SaveFolder(&model.Folder{ID: "f1", Name: "F", ProjectID: projectID}); err != nil {
		t.Fatalf("保存文件夹失败: %v", err)
	}

	// 在 collection.json 注入额外标记字段；写路径的结构化序列化会丢弃它，
	// 因此只要读取后标记仍在，即证明读路径没有回写文件。
	collectionPath := filepath.Join(store.BaseDir(), "projects", projectID, "collection.json")
	raw, err := os.ReadFile(collectionPath)
	if err != nil {
		t.Fatalf("读取 collection.json 失败: %v", err)
	}
	var asMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatalf("解析 collection.json 失败: %v", err)
	}
	asMap["_marker"] = json.RawMessage(`"keep"`)
	marked, err := json.Marshal(asMap)
	if err != nil {
		t.Fatalf("序列化标记后的 collection.json 失败: %v", err)
	}
	if err := os.WriteFile(collectionPath, marked, 0644); err != nil {
		t.Fatalf("写入标记后的 collection.json 失败: %v", err)
	}

	if _, err := store.GetCollectionData(projectID); err != nil {
		t.Fatalf("读取集合数据失败: %v", err)
	}
	if _, err := store.ListFolders(projectID); err != nil {
		t.Fatalf("列出文件夹失败: %v", err)
	}

	after, err := os.ReadFile(collectionPath)
	if err != nil {
		t.Fatalf("再次读取 collection.json 失败: %v", err)
	}
	var afterMap map[string]json.RawMessage
	if err := json.Unmarshal(after, &afterMap); err != nil {
		t.Fatalf("解析读取后的 collection.json 失败: %v", err)
	}
	if _, ok := afterMap["_marker"]; !ok {
		t.Fatal("读取路径不应回写 collection.json（标记字段丢失说明发生了回写）")
	}
}

func TestAddHistory_TruncatesToMaxSize(t *testing.T) {
	store := newTestFileStore(t)
	projectID := uuid.New().String()

	project := &model.Project{
		ID:        projectID,
		Name:      "历史测试",
		CreatedAt: "2026-06-13T00:00:00Z",
		UpdatedAt: "2026-06-13T00:00:00Z",
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("保存项目失败: %v", err)
	}

	for i := 0; i < maxHistorySize+5; i++ {
		entry := &model.HistoryEntry{
			Name:   "req",
			Method: "GET",
			URL:    "https://example.com",
		}
		if err := store.AddHistory(projectID, entry); err != nil {
			t.Fatalf("写入历史失败: %v", err)
		}
	}

	entries, err := store.GetHistory(projectID)
	if err != nil {
		t.Fatalf("读取历史失败: %v", err)
	}
	if len(entries) != maxHistorySize {
		t.Fatalf("历史条数应为 %d，实际 %d", maxHistorySize, len(entries))
	}
}
