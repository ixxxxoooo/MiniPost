package gui

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/egoist/mygo"

	"minipost/internal/model"
	"minipost/internal/service"
)

// fakeBackend 是测试用的内存后端，记录调用并返回预设响应。
type fakeBackend struct {
	mu       sync.Mutex
	dir      string
	projects []model.Project
	colls    map[string]*model.CollectionData
	envs     map[string][]model.Environment
	history  map[string][]model.HistoryEntry
	seq      int

	sent     []model.SendRequestInput
	resp     *model.HttpResponse
	sendErr  error
	backups  []model.BackupFile // ListBackups 返回的备份列表
	canceled []string

	importedStrategy string
	saved            []savedFile // SaveResponseToFile 收到的保存请求
	texts            []savedFile // SaveTextFile 收到的保存请求（如导出 cURL）
	sampleNames      []string    // InstallSampleProjects 的返回值
}

// savedFile 记录一次响应保存：默认文件名与内容。
type savedFile struct{ name, content string }

func newFakeBackend(t *testing.T) *fakeBackend {
	return &fakeBackend{
		dir:     t.TempDir(),
		colls:   map[string]*model.CollectionData{},
		envs:    map[string][]model.Environment{},
		history: map[string][]model.HistoryEntry{},
		resp: &model.HttpResponse{
			StatusCode: 200, StatusText: "OK", ContentType: "application/json",
			Headers: map[string][]string{"Content-Type": {"application/json"}},
			Body:    `{"hello":"world"}`, Duration: 12, Size: 17,
		},
	}
}

func (f *fakeBackend) id(prefix string) string {
	f.seq++
	return fmt.Sprintf("%s%d", prefix, f.seq)
}

// seedProject 预置一个带若干请求的项目。
func (f *fakeBackend) seedProject(name string, requestNames ...string) string {
	pid := f.id("p")
	f.projects = append(f.projects, model.Project{ID: pid, Name: name})
	cd := &model.CollectionData{}
	f.colls[pid] = cd
	for i, n := range requestNames {
		rid := f.id("r")
		cd.Requests = append(cd.Requests, model.RequestItem{ID: rid, Name: n, Method: "GET", URL: "https://example.com/" + n, ProjectID: pid, Body: model.RequestBody{Type: "none"}, Auth: model.AuthConfig{Type: "none"}})
		cd.TreeNodes = append(cd.TreeNodes, model.CollectionNode{NodeID: rid, NodeType: model.CollectionNodeTypeRequest, ProjectID: pid, SortOrder: i})
	}
	return pid
}

func (f *fakeBackend) BaseDir() string { return f.dir }
func (f *fakeBackend) ListProjects() ([]model.Project, error) {
	return append([]model.Project(nil), f.projects...), nil
}
func (f *fakeBackend) CreateProject(name string) (*model.Project, error) {
	p := model.Project{ID: f.id("p"), Name: name}
	f.projects = append(f.projects, p)
	f.colls[p.ID] = &model.CollectionData{}
	return &p, nil
}
func (f *fakeBackend) RenameProject(id, name string) (*model.Project, error) {
	for i := range f.projects {
		if f.projects[i].ID == id {
			f.projects[i].Name = name
			return &f.projects[i], nil
		}
	}
	return nil, errors.New("not found")
}
func (f *fakeBackend) UpdateProjectDescription(id, d string) (*model.Project, error) {
	for i := range f.projects {
		if f.projects[i].ID == id {
			f.projects[i].Description = d
			return &f.projects[i], nil
		}
	}
	return nil, errors.New("not found")
}
func (f *fakeBackend) UpdateProjectTheme(id, c string) (*model.Project, error) {
	for i := range f.projects {
		if f.projects[i].ID == id {
			f.projects[i].ThemeColor = c
			return &f.projects[i], nil
		}
	}
	return nil, errors.New("not found")
}
func (f *fakeBackend) DeleteProject(id string) error {
	for i := range f.projects {
		if f.projects[i].ID == id {
			f.projects = append(f.projects[:i], f.projects[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}
func (f *fakeBackend) GetCollectionData(pid string) (*model.CollectionData, error) {
	cd := *f.colls[pid]
	return &cd, nil
}
func (f *fakeBackend) MoveCollectionNode(pid, nodeID string, kind model.CollectionNodeType, target string, idx int) error {
	cd := f.colls[pid]
	// 与真实实现一致：先摘掉自己，再按 idx 插入目标文件夹的兄弟列表并重排序号。
	var moving model.CollectionNode
	var sibs []model.CollectionNode
	var rest []model.CollectionNode
	for _, n := range cd.TreeNodes {
		switch {
		case n.NodeID == nodeID:
			moving = n
		case n.ParentFolderID == target:
			sibs = append(sibs, n)
		default:
			rest = append(rest, n)
		}
	}
	sort.SliceStable(sibs, func(i, j int) bool { return sibs[i].SortOrder < sibs[j].SortOrder })
	moving.ParentFolderID = target
	if idx < 0 || idx > len(sibs) {
		idx = len(sibs)
	}
	sibs = append(sibs[:idx], append([]model.CollectionNode{moving}, sibs[idx:]...)...)
	for i := range sibs {
		sibs[i].SortOrder = i
	}
	cd.TreeNodes = append(rest, sibs...)
	return nil
}
func (f *fakeBackend) RestoreCollectionSubtree(pid string, sub model.CollectionData) error {
	cd := f.colls[pid]
	have := map[string]bool{}
	for _, n := range cd.TreeNodes {
		have[n.NodeID] = true
	}
	for _, x := range sub.Folders {
		if !have[x.ID] {
			cd.Folders = append(cd.Folders, x)
		}
	}
	for _, x := range sub.Requests {
		if !have[x.ID] {
			cd.Requests = append(cd.Requests, x)
		}
	}
	for _, x := range sub.TreeNodes {
		if !have[x.NodeID] {
			cd.TreeNodes = append(cd.TreeNodes, x)
		}
	}
	return nil
}
func (f *fakeBackend) CreateFolder(pid, parent, name string) (*model.Folder, error) {
	cd := f.colls[pid]
	fo := model.Folder{ID: f.id("f"), Name: name, ProjectID: pid, ParentID: parent}
	cd.Folders = append(cd.Folders, fo)
	cd.TreeNodes = append(cd.TreeNodes, model.CollectionNode{NodeID: fo.ID, NodeType: model.CollectionNodeTypeFolder, ProjectID: pid, ParentFolderID: parent, SortOrder: len(cd.TreeNodes)})
	return &fo, nil
}
func (f *fakeBackend) RenameFolder(pid, id, name string) error {
	cd := f.colls[pid]
	for i := range cd.Folders {
		if cd.Folders[i].ID == id {
			cd.Folders[i].Name = name
		}
	}
	return nil
}
func (f *fakeBackend) DeleteFolder(pid, id string) error { return f.removeNode(pid, id) }
func (f *fakeBackend) DuplicateFolder(pid, id string) (*model.Folder, error) {
	return f.CreateFolder(pid, "", "copy")
}
func (f *fakeBackend) CreateRequestItem(pid, folder, name string) (*model.RequestItem, error) {
	cd := f.colls[pid]
	it := model.RequestItem{ID: f.id("r"), Name: name, Method: "GET", ProjectID: pid, FolderID: folder, Body: model.RequestBody{Type: "none"}, Auth: model.AuthConfig{Type: "none"}}
	cd.Requests = append(cd.Requests, it)
	cd.TreeNodes = append(cd.TreeNodes, model.CollectionNode{NodeID: it.ID, NodeType: model.CollectionNodeTypeRequest, ProjectID: pid, ParentFolderID: folder, SortOrder: len(cd.TreeNodes)})
	return &it, nil
}
func (f *fakeBackend) SaveRequestItem(r model.RequestItem) error {
	cd := f.colls[r.ProjectID]
	for i := range cd.Requests {
		if cd.Requests[i].ID == r.ID {
			cd.Requests[i] = r
			return nil
		}
	}
	return errors.New("request not found")
}
func (f *fakeBackend) DeleteRequestItem(pid, id string) error { return f.removeNode(pid, id) }
func (f *fakeBackend) removeNode(pid, id string) error {
	cd := f.colls[pid]
	for i := range cd.Requests {
		if cd.Requests[i].ID == id {
			cd.Requests = append(cd.Requests[:i], cd.Requests[i+1:]...)
			break
		}
	}
	for i := range cd.Folders {
		if cd.Folders[i].ID == id {
			cd.Folders = append(cd.Folders[:i], cd.Folders[i+1:]...)
			break
		}
	}
	for i := range cd.TreeNodes {
		if cd.TreeNodes[i].NodeID == id {
			cd.TreeNodes = append(cd.TreeNodes[:i], cd.TreeNodes[i+1:]...)
			break
		}
	}
	return nil
}
func (f *fakeBackend) RenameRequest(pid, id, name string) error {
	cd := f.colls[pid]
	for i := range cd.Requests {
		if cd.Requests[i].ID == id {
			cd.Requests[i].Name = name
		}
	}
	return nil
}
func (f *fakeBackend) DuplicateRequest(pid, id string) (*model.RequestItem, error) {
	return f.CreateRequestItem(pid, "", "copy")
}
func (f *fakeBackend) ListEnvironments(pid string) ([]model.Environment, error) {
	return append([]model.Environment(nil), f.envs[pid]...), nil
}
func (f *fakeBackend) CreateEnvironment(pid, name string) (*model.Environment, error) {
	e := model.Environment{ID: f.id("e"), Name: name, ProjectID: pid}
	f.envs[pid] = append(f.envs[pid], e)
	return &e, nil
}
func (f *fakeBackend) SaveEnvironment(env model.Environment) error {
	for i := range f.envs[env.ProjectID] {
		if f.envs[env.ProjectID][i].ID == env.ID {
			f.envs[env.ProjectID][i] = env
			return nil
		}
	}
	f.envs[env.ProjectID] = append(f.envs[env.ProjectID], env) // 与真实实现一致：不存在则新增
	return nil
}
func (f *fakeBackend) DeleteEnvironment(pid, id string) error {
	es := f.envs[pid]
	for i := range es {
		if es[i].ID == id {
			f.envs[pid] = append(es[:i], es[i+1:]...)
		}
	}
	return nil
}
func (f *fakeBackend) SendRequestWithEnv(in model.SendRequestInput, pid, envID string) (*model.HttpResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, in)
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	return f.resp, nil
}
func (f *fakeBackend) SendRequestWithEnvStream(in model.SendRequestInput, pid, envID string, emit func(model.StreamChunk)) (*model.HttpResponse, error) {
	return f.SendRequestWithEnv(in, pid, envID)
}
func (f *fakeBackend) CancelHTTPRequest(id string) bool {
	f.canceled = append(f.canceled, id)
	return true
}
func (f *fakeBackend) ImportCurl(cmd string) (*model.SendRequestInput, error) {
	return &model.SendRequestInput{Method: "POST", URL: "https://curl.example/x", Body: model.RequestBody{Type: "raw", Raw: "hi"}}, nil
}
func (f *fakeBackend) GetHistory(pid string) ([]model.HistoryEntry, error) {
	return f.history[pid], nil
}
func (f *fakeBackend) DeleteHistoryEntry(pid, id string) error {
	var kept []model.HistoryEntry
	for _, e := range f.history[pid] {
		if e.ID != id {
			kept = append(kept, e)
		}
	}
	f.history[pid] = kept
	return nil
}
func (f *fakeBackend) ClearHistory(pid string) error                { f.history[pid] = nil; return nil }
func (f *fakeBackend) ExportProjectJSON(pid string) (string, error) { return "{}", nil }
func (f *fakeBackend) ExportNodeJSON(pid, nodeID string) (string, error) {
	return `{"node":"` + nodeID + `"}`, nil
}
func (f *fakeBackend) PreviewImportFromFile(pid, format, content string) (*service.ImportPreview, error) {
	return &service.ImportPreview{Kind: "postman", Content: content}, nil
}
func (f *fakeBackend) PreviewImportFromURL(pid, format, u string) (*service.ImportPreview, error) {
	return &service.ImportPreview{Kind: "swagger", ResolvedURL: u}, nil
}
func (f *fakeBackend) ImportCollectionWithStrategy(pid, format, content, u, strategy string) error {
	f.importedStrategy = strategy
	return nil
}

// InstallSampleProjects 模拟安装示例：为每个名称创建一个空项目。
func (f *fakeBackend) InstallSampleProjects() ([]string, error) {
	for _, n := range f.sampleNames {
		f.seedProject(n)
	}
	return f.sampleNames, nil
}
func (f *fakeBackend) CreateBackup() (string, error)                     { return "/tmp/backup", nil }
func (f *fakeBackend) ListBackups(limit int) ([]model.BackupFile, error) { return f.backups, nil }
func (f *fakeBackend) DataDir() string                                   { return "/tmp/minipost" }
func (f *fakeBackend) RestoreBackup(path string) error                   { return nil }
func (f *fakeBackend) OpenFileDialogJSON(*mygo.Window) (string, error) {
	return "", nil
}
func (f *fakeBackend) OpenFileDialogAny(*mygo.Window) (string, error) { return "", nil }
func (f *fakeBackend) SaveFileDialogJSON(_ *mygo.Window, name, content string) error {
	f.texts = append(f.texts, savedFile{name, content})
	return nil
}
func (f *fakeBackend) SaveTextFile(_ *mygo.Window, name, content string) error {
	f.texts = append(f.texts, savedFile{name, content})
	return nil
}
func (f *fakeBackend) SaveResponseToFile(_ *mygo.Window, name, content string) error {
	f.saved = append(f.saved, savedFile{name, content})
	return nil
}
