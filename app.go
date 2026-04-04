package main

import (
	"context"
	"log"

	"minipost/internal/model"
	"minipost/internal/pkg/httputil"
	"minipost/internal/repository"
	"minipost/internal/service"
)

// App 应用主结构体，Wails 会将其方法暴露给前端
type App struct {
	ctx        context.Context
	httpSvc    *service.HttpService
	projectSvc *service.ProjectService
	requestSvc *service.RequestService
	envSvc     *service.EnvironmentService
	historySvc *service.HistoryService
	store      *repository.FileStore
}

func NewApp() *App {
	store, err := repository.NewFileStore()
	if err != nil {
		log.Fatalf("初始化存储失败: %v", err)
	}

	return &App{
		httpSvc:    service.NewHttpService(),
		projectSvc: service.NewProjectService(store),
		requestSvc: service.NewRequestService(store),
		envSvc:     service.NewEnvironmentService(store),
		historySvc: service.NewHistoryService(store),
		store:      store,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ---- HTTP 请求 ----

// SendRequestWithEnv 带环境变量解析的请求发送
func (a *App) SendRequestWithEnv(input model.SendRequestInput, projectID, envID string) (*model.HttpResponse, error) {
	if envID != "" && projectID != "" {
		envs, err := a.envSvc.ListEnvironments(projectID)
		if err == nil {
			for _, env := range envs {
				if env.ID == envID {
					input = httputil.ResolveRequestInput(input, env.Variables)
					break
				}
			}
		}
	}
	resp, err := a.httpSvc.SendRequest(input)
	if err != nil {
		return nil, err
	}

	// 记录历史
	if projectID != "" {
		entry := &model.HistoryEntry{
			Name:       input.URL,
			Method:     input.Method,
			URL:        input.URL,
			StatusCode: resp.StatusCode,
			Duration:   resp.Duration,
			Size:       resp.Size,
		}
		_ = a.historySvc.AddEntry(projectID, entry)
	}

	return resp, nil
}

func (a *App) SendRequest(input model.SendRequestInput) (*model.HttpResponse, error) {
	return a.httpSvc.SendRequest(input)
}

// ---- 项目管理 ----

func (a *App) ListProjects() ([]model.Project, error) {
	return a.projectSvc.ListProjects()
}

func (a *App) CreateProject(name string) (*model.Project, error) {
	return a.projectSvc.CreateProject(name)
}

func (a *App) RenameProject(id, name string) (*model.Project, error) {
	return a.projectSvc.RenameProject(id, name)
}

func (a *App) DeleteProject(id string) error {
	return a.projectSvc.DeleteProject(id)
}

// ---- 文件夹管理 ----

func (a *App) ListFolders(projectID string) ([]model.Folder, error) {
	return a.requestSvc.ListFolders(projectID)
}

func (a *App) CreateFolder(projectID, parentID, name string) (*model.Folder, error) {
	return a.requestSvc.CreateFolder(projectID, parentID, name)
}

func (a *App) RenameFolder(projectID, folderID, name string) error {
	return a.requestSvc.RenameFolder(projectID, folderID, name)
}

func (a *App) DeleteFolder(projectID, folderID string) error {
	return a.requestSvc.DeleteFolder(projectID, folderID)
}

// ---- 请求管理 ----

func (a *App) ListRequests(projectID string) ([]model.RequestItem, error) {
	return a.requestSvc.ListRequests(projectID)
}

func (a *App) CreateRequestItem(projectID, folderID, name string) (*model.RequestItem, error) {
	return a.requestSvc.CreateRequest(projectID, folderID, name)
}

func (a *App) SaveRequestItem(request model.RequestItem) error {
	return a.requestSvc.SaveRequest(&request)
}

func (a *App) DeleteRequestItem(projectID, requestID string) error {
	return a.requestSvc.DeleteRequest(projectID, requestID)
}

// ---- 环境变量 ----

func (a *App) ListEnvironments(projectID string) ([]model.Environment, error) {
	return a.envSvc.ListEnvironments(projectID)
}

func (a *App) CreateEnvironment(projectID, name string) (*model.Environment, error) {
	return a.envSvc.CreateEnvironment(projectID, name)
}

func (a *App) SaveEnvironment(env model.Environment) error {
	return a.envSvc.SaveEnvironment(&env)
}

func (a *App) DeleteEnvironment(projectID, envID string) error {
	return a.envSvc.DeleteEnvironment(projectID, envID)
}

// ---- 导入 ----

// ImportCurl 解析 cURL 命令为请求数据
func (a *App) ImportCurl(curlCommand string) (*model.SendRequestInput, error) {
	return httputil.ParseCurlCommand(curlCommand)
}

// ---- 历史记录 ----

func (a *App) GetHistory(projectID string) ([]model.HistoryEntry, error) {
	return a.historySvc.GetHistory(projectID)
}

func (a *App) ClearHistory(projectID string) error {
	return a.historySvc.ClearHistory(projectID)
}
