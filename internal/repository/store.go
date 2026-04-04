package repository

import "minipost/internal/model"

// ProjectStore 项目数据存储接口
type ProjectStore interface {
	ListProjects() ([]model.Project, error)
	GetProject(id string) (*model.Project, error)
	SaveProject(project *model.Project) error
	DeleteProject(id string) error
}

// CollectionStore 请求集合存储接口（文件夹 + 请求）
type CollectionStore interface {
	ListFolders(projectID string) ([]model.Folder, error)
	SaveFolder(folder *model.Folder) error
	DeleteFolder(projectID, folderID string) error

	ListRequests(projectID string) ([]model.RequestItem, error)
	GetRequest(projectID, requestID string) (*model.RequestItem, error)
	SaveRequest(request *model.RequestItem) error
	DeleteRequest(projectID, requestID string) error
}

// EnvironmentStore 环境变量存储接口
type EnvironmentStore interface {
	ListEnvironments(projectID string) ([]model.Environment, error)
	SaveEnvironment(env *model.Environment) error
	DeleteEnvironment(projectID, envID string) error
}

// HistoryStore 历史记录存储接口
type HistoryStore interface {
	GetHistory(projectID string) ([]model.HistoryEntry, error)
	AddHistory(projectID string, entry *model.HistoryEntry) error
	ClearHistory(projectID string) error
}
