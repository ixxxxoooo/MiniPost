package main

import (
	"context"

	"minipost/internal/model"
	"minipost/internal/pkg/httputil"
	"minipost/internal/pkg/logger"
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
	logger.Info("正在初始化存储...")
	store, err := repository.NewFileStore()
	if err != nil {
		logger.Fatal("初始化存储失败", "error", err.Error())
	}
	logger.Info("存储初始化完成", "baseDir", store.BaseDir())

	logger.Info("正在初始化各服务...")
	app := &App{
		httpSvc:    service.NewHttpService(),
		projectSvc: service.NewProjectService(store),
		requestSvc: service.NewRequestService(store),
		envSvc:     service.NewEnvironmentService(store),
		historySvc: service.NewHistoryService(store),
		store:      store,
	}
	logger.Info("所有服务初始化完成")

	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logger.Info("Wails OnStartup 回调执行完成, 窗口即将显示")
}

func (a *App) shutdown(ctx context.Context) {
	logger.Info("Wails OnShutdown 回调执行, 应用正在关闭")
}

// ---- HTTP 请求 ----

// SendRequestWithEnv 带环境变量解析的请求发送
func (a *App) SendRequestWithEnv(input model.SendRequestInput, projectID, envID string) (*model.HttpResponse, error) {
	logger.Debug("SendRequestWithEnv 调用",
		"method", input.Method,
		"url", input.URL,
		"projectID", projectID,
		"envID", envID,
	)

	if envID != "" && projectID != "" {
		envs, err := a.envSvc.ListEnvironments(projectID)
		if err == nil {
			for _, env := range envs {
				if env.ID == envID {
					input = httputil.ResolveRequestInput(input, env.Variables)
					logger.Debug("环境变量已解析", "envName", env.Name, "varCount", len(env.Variables))
					break
				}
			}
		}
	}

	resp, err := a.httpSvc.SendRequest(input)
	if err != nil {
		logger.Warn("HTTP 请求失败", "method", input.Method, "url", input.URL, "error", err.Error())
		return nil, err
	}

	logger.Info("HTTP 请求完成",
		"method", input.Method,
		"url", input.URL,
		"status", resp.StatusCode,
		"duration_ms", resp.Duration,
		"size", resp.Size,
	)

	if projectID != "" {
		entry := &model.HistoryEntry{
			Name:       input.URL,
			Method:     input.Method,
			URL:        input.URL,
			StatusCode: resp.StatusCode,
			Duration:   resp.Duration,
			Size:       resp.Size,
		}
		if err := a.historySvc.AddEntry(projectID, entry); err != nil {
			logger.Warn("历史记录保存失败", "projectID", projectID, "error", err.Error())
		}
	}

	return resp, nil
}

func (a *App) SendRequest(input model.SendRequestInput) (*model.HttpResponse, error) {
	logger.Debug("SendRequest 调用", "method", input.Method, "url", input.URL)
	resp, err := a.httpSvc.SendRequest(input)
	if err != nil {
		logger.Warn("HTTP 请求失败", "method", input.Method, "url", input.URL, "error", err.Error())
		return nil, err
	}
	logger.Info("HTTP 请求完成", "method", input.Method, "url", input.URL, "status", resp.StatusCode)
	return resp, nil
}

// ---- 项目管理 ----

func (a *App) ListProjects() ([]model.Project, error) {
	projects, err := a.projectSvc.ListProjects()
	if err != nil {
		logger.Error("列出项目失败", "error", err.Error())
		return nil, err
	}
	logger.Debug("列出项目", "count", len(projects))
	return projects, nil
}

func (a *App) CreateProject(name string) (*model.Project, error) {
	project, err := a.projectSvc.CreateProject(name)
	if err != nil {
		logger.Error("创建项目失败", "name", name, "error", err.Error())
		return nil, err
	}
	logger.Info("创建项目成功", "id", project.ID, "name", project.Name)
	return project, nil
}

func (a *App) RenameProject(id, name string) (*model.Project, error) {
	project, err := a.projectSvc.RenameProject(id, name)
	if err != nil {
		logger.Error("重命名项目失败", "id", id, "name", name, "error", err.Error())
		return nil, err
	}
	logger.Info("重命名项目成功", "id", id, "name", name)
	return project, nil
}

func (a *App) DeleteProject(id string) error {
	if err := a.projectSvc.DeleteProject(id); err != nil {
		logger.Error("删除项目失败", "id", id, "error", err.Error())
		return err
	}
	logger.Info("删除项目成功", "id", id)
	return nil
}

// ---- 文件夹管理 ----

func (a *App) ListFolders(projectID string) ([]model.Folder, error) {
	folders, err := a.requestSvc.ListFolders(projectID)
	if err != nil {
		logger.Error("列出文件夹失败", "projectID", projectID, "error", err.Error())
		return nil, err
	}
	logger.Debug("列出文件夹", "projectID", projectID, "count", len(folders))
	return folders, nil
}

func (a *App) CreateFolder(projectID, parentID, name string) (*model.Folder, error) {
	folder, err := a.requestSvc.CreateFolder(projectID, parentID, name)
	if err != nil {
		logger.Error("创建文件夹失败", "projectID", projectID, "name", name, "error", err.Error())
		return nil, err
	}
	logger.Info("创建文件夹成功", "id", folder.ID, "name", folder.Name)
	return folder, nil
}

func (a *App) RenameFolder(projectID, folderID, name string) error {
	if err := a.requestSvc.RenameFolder(projectID, folderID, name); err != nil {
		logger.Error("重命名文件夹失败", "folderID", folderID, "name", name, "error", err.Error())
		return err
	}
	logger.Info("重命名文件夹成功", "folderID", folderID, "name", name)
	return nil
}

func (a *App) DeleteFolder(projectID, folderID string) error {
	if err := a.requestSvc.DeleteFolder(projectID, folderID); err != nil {
		logger.Error("删除文件夹失败", "folderID", folderID, "error", err.Error())
		return err
	}
	logger.Info("删除文件夹成功", "folderID", folderID)
	return nil
}

// ---- 请求管理 ----

func (a *App) ListRequests(projectID string) ([]model.RequestItem, error) {
	requests, err := a.requestSvc.ListRequests(projectID)
	if err != nil {
		logger.Error("列出请求失败", "projectID", projectID, "error", err.Error())
		return nil, err
	}
	logger.Debug("列出请求", "projectID", projectID, "count", len(requests))
	return requests, nil
}

func (a *App) CreateRequestItem(projectID, folderID, name string) (*model.RequestItem, error) {
	req, err := a.requestSvc.CreateRequest(projectID, folderID, name)
	if err != nil {
		logger.Error("创建请求失败", "projectID", projectID, "name", name, "error", err.Error())
		return nil, err
	}
	logger.Info("创建请求成功", "id", req.ID, "name", req.Name)
	return req, nil
}

func (a *App) SaveRequestItem(request model.RequestItem) error {
	if err := a.requestSvc.SaveRequest(&request); err != nil {
		logger.Error("保存请求失败", "id", request.ID, "error", err.Error())
		return err
	}
	logger.Debug("保存请求成功", "id", request.ID, "name", request.Name)
	return nil
}

func (a *App) DeleteRequestItem(projectID, requestID string) error {
	if err := a.requestSvc.DeleteRequest(projectID, requestID); err != nil {
		logger.Error("删除请求失败", "requestID", requestID, "error", err.Error())
		return err
	}
	logger.Info("删除请求成功", "requestID", requestID)
	return nil
}

// ---- 环境变量 ----

func (a *App) ListEnvironments(projectID string) ([]model.Environment, error) {
	envs, err := a.envSvc.ListEnvironments(projectID)
	if err != nil {
		logger.Error("列出环境变量失败", "projectID", projectID, "error", err.Error())
		return nil, err
	}
	logger.Debug("列出环境变量", "projectID", projectID, "count", len(envs))
	return envs, nil
}

func (a *App) CreateEnvironment(projectID, name string) (*model.Environment, error) {
	env, err := a.envSvc.CreateEnvironment(projectID, name)
	if err != nil {
		logger.Error("创建环境变量失败", "projectID", projectID, "name", name, "error", err.Error())
		return nil, err
	}
	logger.Info("创建环境变量成功", "id", env.ID, "name", env.Name)
	return env, nil
}

func (a *App) SaveEnvironment(env model.Environment) error {
	if err := a.envSvc.SaveEnvironment(&env); err != nil {
		logger.Error("保存环境变量失败", "id", env.ID, "error", err.Error())
		return err
	}
	logger.Debug("保存环境变量成功", "id", env.ID, "name", env.Name)
	return nil
}

func (a *App) DeleteEnvironment(projectID, envID string) error {
	if err := a.envSvc.DeleteEnvironment(projectID, envID); err != nil {
		logger.Error("删除环境变量失败", "envID", envID, "error", err.Error())
		return err
	}
	logger.Info("删除环境变量成功", "envID", envID)
	return nil
}

// ---- 导入 ----

// ImportCurl 解析 cURL 命令为请求数据
func (a *App) ImportCurl(curlCommand string) (*model.SendRequestInput, error) {
	logger.Debug("ImportCurl 调用", "cmdLength", len(curlCommand))
	result, err := httputil.ParseCurlCommand(curlCommand)
	if err != nil {
		logger.Error("cURL 解析失败", "error", err.Error())
		return nil, err
	}
	logger.Info("cURL 解析成功", "method", result.Method, "url", result.URL)
	return result, nil
}

// ---- 历史记录 ----

func (a *App) GetHistory(projectID string) ([]model.HistoryEntry, error) {
	entries, err := a.historySvc.GetHistory(projectID)
	if err != nil {
		logger.Error("获取历史记录失败", "projectID", projectID, "error", err.Error())
		return nil, err
	}
	logger.Debug("获取历史记录", "projectID", projectID, "count", len(entries))
	return entries, nil
}

func (a *App) ClearHistory(projectID string) error {
	if err := a.historySvc.ClearHistory(projectID); err != nil {
		logger.Error("清除历史记录失败", "projectID", projectID, "error", err.Error())
		return err
	}
	logger.Info("清除历史记录成功", "projectID", projectID)
	return nil
}
