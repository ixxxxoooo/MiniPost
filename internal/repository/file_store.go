package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"minipost/internal/model"
)

const (
	schemaVersion  = 1
	maxHistorySize = 500
)

// FileStore 基于 JSON 文件的本地存储实现
type FileStore struct {
	baseDir string
	mu      sync.RWMutex
}

func NewFileStore() (*FileStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户目录失败: %w", err)
	}

	baseDir := filepath.Join(home, ".minipost")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	return &FileStore{baseDir: baseDir}, nil
}

// ---- 配置 ----

type AppConfig struct {
	LastProjectID string `json:"lastProjectId,omitempty"`
	Theme         string `json:"theme,omitempty"`
	SchemaVersion int    `json:"schemaVersion"`
}

func (fs *FileStore) LoadConfig() (*AppConfig, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := filepath.Join(fs.baseDir, "config.json")
	cfg := &AppConfig{SchemaVersion: schemaVersion}
	if err := fs.readJSON(path, cfg); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	return cfg, nil
}

func (fs *FileStore) SaveConfig(cfg *AppConfig) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	cfg.SchemaVersion = schemaVersion
	return fs.writeJSON(filepath.Join(fs.baseDir, "config.json"), cfg)
}

// ---- 项目 ----

func (fs *FileStore) ListProjects() ([]model.Project, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	projectsDir := filepath.Join(fs.baseDir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	var projects []model.Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projFile := filepath.Join(projectsDir, entry.Name(), "project.json")
		var proj model.Project
		if err := fs.readJSON(projFile, &proj); err == nil {
			projects = append(projects, proj)
		}
	}
	return projects, nil
}

func (fs *FileStore) GetProject(id string) (*model.Project, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var proj model.Project
	path := filepath.Join(fs.baseDir, "projects", id, "project.json")
	if err := fs.readJSON(path, &proj); err != nil {
		return nil, err
	}
	return &proj, nil
}

func (fs *FileStore) SaveProject(project *model.Project) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	project.SchemaVersion = schemaVersion
	projDir := filepath.Join(fs.baseDir, "projects", project.ID)
	if err := os.MkdirAll(projDir, 0755); err != nil {
		return err
	}
	return fs.writeJSON(filepath.Join(projDir, "project.json"), project)
}

func (fs *FileStore) DeleteProject(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return os.RemoveAll(filepath.Join(fs.baseDir, "projects", id))
}

// ---- 文件夹 ----

type foldersFile struct {
	SchemaVersion int            `json:"schemaVersion"`
	Folders       []model.Folder `json:"folders"`
}

func (fs *FileStore) foldersPath(projectID string) string {
	return filepath.Join(fs.baseDir, "projects", projectID, "folders.json")
}

func (fs *FileStore) ListFolders(projectID string) ([]model.Folder, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var data foldersFile
	if err := fs.readJSON(fs.foldersPath(projectID), &data); err != nil {
		if os.IsNotExist(err) {
			return []model.Folder{}, nil
		}
		return nil, err
	}
	return data.Folders, nil
}

func (fs *FileStore) SaveFolder(folder *model.Folder) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.foldersPath(folder.ProjectID)
	var data foldersFile
	_ = fs.readJSON(path, &data)

	found := false
	for i, f := range data.Folders {
		if f.ID == folder.ID {
			data.Folders[i] = *folder
			found = true
			break
		}
	}
	if !found {
		data.Folders = append(data.Folders, *folder)
	}

	data.SchemaVersion = schemaVersion
	return fs.writeJSON(path, &data)
}

func (fs *FileStore) DeleteFolder(projectID, folderID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.foldersPath(projectID)
	var data foldersFile
	if err := fs.readJSON(path, &data); err != nil {
		return err
	}

	var filtered []model.Folder
	for _, f := range data.Folders {
		if f.ID != folderID {
			filtered = append(filtered, f)
		}
	}
	data.Folders = filtered
	data.SchemaVersion = schemaVersion
	return fs.writeJSON(path, &data)
}

// ---- 请求 ----

type requestsFile struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Requests      []model.RequestItem `json:"requests"`
}

func (fs *FileStore) requestsPath(projectID string) string {
	return filepath.Join(fs.baseDir, "projects", projectID, "requests.json")
}

func (fs *FileStore) ListRequests(projectID string) ([]model.RequestItem, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var data requestsFile
	if err := fs.readJSON(fs.requestsPath(projectID), &data); err != nil {
		if os.IsNotExist(err) {
			return []model.RequestItem{}, nil
		}
		return nil, err
	}
	return data.Requests, nil
}

func (fs *FileStore) GetRequest(projectID, requestID string) (*model.RequestItem, error) {
	requests, err := fs.ListRequests(projectID)
	if err != nil {
		return nil, err
	}
	for _, r := range requests {
		if r.ID == requestID {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("请求 %s 不存在", requestID)
}

func (fs *FileStore) SaveRequest(request *model.RequestItem) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.requestsPath(request.ProjectID)
	var data requestsFile
	_ = fs.readJSON(path, &data)

	found := false
	for i, r := range data.Requests {
		if r.ID == request.ID {
			data.Requests[i] = *request
			found = true
			break
		}
	}
	if !found {
		data.Requests = append(data.Requests, *request)
	}

	data.SchemaVersion = schemaVersion
	return fs.writeJSON(path, &data)
}

func (fs *FileStore) DeleteRequest(projectID, requestID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.requestsPath(projectID)
	var data requestsFile
	if err := fs.readJSON(path, &data); err != nil {
		return err
	}

	var filtered []model.RequestItem
	for _, r := range data.Requests {
		if r.ID != requestID {
			filtered = append(filtered, r)
		}
	}
	data.Requests = filtered
	data.SchemaVersion = schemaVersion
	return fs.writeJSON(path, &data)
}

// ---- 环境变量 ----

type environmentsFile struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Environments  []model.Environment `json:"environments"`
}

func (fs *FileStore) envsPath(projectID string) string {
	return filepath.Join(fs.baseDir, "projects", projectID, "environments.json")
}

func (fs *FileStore) ListEnvironments(projectID string) ([]model.Environment, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var data environmentsFile
	if err := fs.readJSON(fs.envsPath(projectID), &data); err != nil {
		if os.IsNotExist(err) {
			return []model.Environment{}, nil
		}
		return nil, err
	}
	return data.Environments, nil
}

func (fs *FileStore) SaveEnvironment(env *model.Environment) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.envsPath(env.ProjectID)
	var data environmentsFile
	_ = fs.readJSON(path, &data)

	found := false
	for i, e := range data.Environments {
		if e.ID == env.ID {
			data.Environments[i] = *env
			found = true
			break
		}
	}
	if !found {
		data.Environments = append(data.Environments, *env)
	}

	data.SchemaVersion = schemaVersion
	return fs.writeJSON(path, &data)
}

func (fs *FileStore) DeleteEnvironment(projectID, envID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.envsPath(projectID)
	var data environmentsFile
	if err := fs.readJSON(path, &data); err != nil {
		return err
	}

	var filtered []model.Environment
	for _, e := range data.Environments {
		if e.ID != envID {
			filtered = append(filtered, e)
		}
	}
	data.Environments = filtered
	data.SchemaVersion = schemaVersion
	return fs.writeJSON(path, &data)
}

// ---- 历史记录 ----

type historyFile struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Entries       []model.HistoryEntry `json:"entries"`
}

func (fs *FileStore) historyPath(projectID string) string {
	return filepath.Join(fs.baseDir, "projects", projectID, "history.json")
}

func (fs *FileStore) GetHistory(projectID string) ([]model.HistoryEntry, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var data historyFile
	if err := fs.readJSON(fs.historyPath(projectID), &data); err != nil {
		if os.IsNotExist(err) {
			return []model.HistoryEntry{}, nil
		}
		return nil, err
	}
	return data.Entries, nil
}

func (fs *FileStore) AddHistory(projectID string, entry *model.HistoryEntry) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.historyPath(projectID)
	var data historyFile
	_ = fs.readJSON(path, &data)

	entry.Timestamp = time.Now().UTC().Format(time.RFC3339)

	// 新记录插入到最前面
	data.Entries = append([]model.HistoryEntry{*entry}, data.Entries...)

	// 限制历史记录数量
	if len(data.Entries) > maxHistorySize {
		data.Entries = data.Entries[:maxHistorySize]
	}

	data.SchemaVersion = schemaVersion
	return fs.writeJSON(path, &data)
}

func (fs *FileStore) ClearHistory(projectID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return fs.writeJSON(fs.historyPath(projectID), &historyFile{SchemaVersion: schemaVersion})
}

// ---- 底层工具方法 ----

func (fs *FileStore) readJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (fs *FileStore) writeJSON(path string, v interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
