package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func NewFileStoreWithDir(baseDir string) (*FileStore, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("数据目录不能为空")
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	return &FileStore{baseDir: baseDir}, nil
}

func NewFileStore() (*FileStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户目录失败: %w", err)
	}
	return NewFileStoreWithDir(filepath.Join(home, ".minipost"))
}

// BaseDir 返回数据存储根目录路径
func (fs *FileStore) BaseDir() string {
	return fs.baseDir
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
	data.Entries = append([]model.HistoryEntry{*entry}, data.Entries...)
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

	// 先写临时文件再 rename，避免写入中断导致 JSON 损坏。
	tmpPath := path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	writeErr := func(cleanup error) error {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return cleanup
	}

	if _, err := file.Write(data); err != nil {
		return writeErr(err)
	}
	if err := file.Sync(); err != nil {
		return writeErr(err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
