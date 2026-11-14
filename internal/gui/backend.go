package gui

import (
	"github.com/egoist/mygo"

	"minipost/internal/model"
	"minipost/internal/service"
)

// Backend 是界面依赖的后端能力，由 main 包的 App 实现，测试中可用假对象替换。
// 涉及文件对话框的方法接收父窗口，用于以 sheet 形式展示。
type Backend interface {
	BaseDir() string

	// 项目
	ListProjects() ([]model.Project, error)
	CreateProject(name string) (*model.Project, error)
	RenameProject(id, name string) (*model.Project, error)
	UpdateProjectDescription(id, description string) (*model.Project, error)
	UpdateProjectTheme(id, color string) (*model.Project, error)
	DeleteProject(id string) error

	// 集合
	GetCollectionData(projectID string) (*model.CollectionData, error)
	MoveCollectionNode(projectID, nodeID string, nodeType model.CollectionNodeType, targetParentFolderID string, targetIndex int) error
	CreateFolder(projectID, parentID, name string) (*model.Folder, error)
	RenameFolder(projectID, folderID, name string) error
	DeleteFolder(projectID, folderID string) error
	DuplicateFolder(projectID, folderID string) (*model.Folder, error)
	CreateRequestItem(projectID, folderID, name string) (*model.RequestItem, error)
	SaveRequestItem(request model.RequestItem) error
	DeleteRequestItem(projectID, requestID string) error
	RenameRequest(projectID, requestID, name string) error
	DuplicateRequest(projectID, requestID string) (*model.RequestItem, error)

	// 环境
	ListEnvironments(projectID string) ([]model.Environment, error)
	CreateEnvironment(projectID, name string) (*model.Environment, error)
	SaveEnvironment(env model.Environment) error
	DeleteEnvironment(projectID, envID string) error

	// 发送
	SendRequestWithEnv(input model.SendRequestInput, projectID, envID string) (*model.HttpResponse, error)
	SendRequestWithEnvStream(input model.SendRequestInput, projectID, envID string, emit func(model.StreamChunk)) (*model.HttpResponse, error)
	CancelHTTPRequest(requestID string) bool
	ImportCurl(curlCommand string) (*model.SendRequestInput, error)

	// 历史
	GetHistory(projectID string) ([]model.HistoryEntry, error)
	ClearHistory(projectID string) error

	// 导入导出与备份
	ExportProjectJSON(projectID string) (string, error)
	PreviewImportFromFile(projectID, format, content string) (*service.ImportPreview, error)
	PreviewImportFromURL(projectID, format, sourceURL string) (*service.ImportPreview, error)
	ImportCollectionWithStrategy(projectID, format, content, sourceURL, strategy string) error
	InstallSampleProjects() ([]string, error)
	CreateBackup() (string, error)
	RestoreBackup(backupPath string) error

	// 文件对话框
	OpenFileDialogJSON(parent *mygo.Window) (string, error)
	OpenFileDialogAny(parent *mygo.Window) (string, error)
	SaveFileDialogJSON(parent *mygo.Window, defaultFilename, content string) error
	SaveTextFile(parent *mygo.Window, defaultFilename, content string) error
	SaveResponseToFile(parent *mygo.Window, defaultFilename, content string) error
}
