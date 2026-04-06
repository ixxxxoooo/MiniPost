package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"minipost/internal/model"
	appErrors "minipost/internal/pkg/errors"
	"minipost/internal/pkg/httputil"
	"minipost/internal/pkg/logger"
	"minipost/internal/repository"
	"minipost/internal/service"

	"gopkg.in/yaml.v3"
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

const httpStreamEventName = "minipost:http-stream"

type httpStreamEventPayload struct {
	StreamID   string `json:"streamId"`
	Kind       string `json:"kind"`
	Data       string `json:"data"`
	Raw        string `json:"raw,omitempty"`
	Timestamp  string `json:"timestamp"`
	Sequence   int    `json:"sequence"`
	BytesTotal int64  `json:"bytesTotal,omitempty"`
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

func normalizeCurlInput(input model.SendRequestInput) (model.SendRequestInput, error) {
	fields := strings.Fields(strings.TrimSpace(input.URL))
	if len(fields) == 0 || !strings.EqualFold(fields[0], "curl") {
		return input, nil
	}

	parsed, err := httputil.ParseCurlCommand(input.URL)
	if err != nil {
		return input, appErrors.Wrap("INVALID_CURL", "cURL 命令解析失败", err)
	}
	return *parsed, nil
}

// ---- HTTP 请求 ----

func (a *App) SendRequestWithEnv(input model.SendRequestInput, projectID, envID string) (*model.HttpResponse, error) {
	return a.sendRequestWithEnv(input, projectID, envID, nil)
}

func (a *App) SendRequestWithEnvStream(input model.SendRequestInput, projectID, envID string, streamID string) (*model.HttpResponse, error) {
	lastSequence := 0
	onChunk := func(chunk model.StreamChunk) {
		lastSequence = chunk.Sequence
		a.emitHTTPStreamEvent(streamID, chunk)
	}

	resp, err := a.sendRequestWithEnv(input, projectID, envID, onChunk)
	if err != nil {
		if lastSequence > 0 {
			a.emitHTTPStreamEvent(streamID, model.StreamChunk{
				Kind:      "error",
				Data:      err.Error(),
				Raw:       err.Error(),
				Timestamp: time.Now().Format(time.RFC3339Nano),
				Sequence:  lastSequence + 1,
			})
		}
		return nil, err
	}

	if lastSequence > 0 {
		a.emitHTTPStreamEvent(streamID, model.StreamChunk{
			Kind:      "connection_closed",
			Data:      "Connection closed",
			Raw:       "Connection closed",
			Timestamp: time.Now().Format(time.RFC3339Nano),
			Sequence:  lastSequence + 1,
		})
	}
	return resp, nil
}

func (a *App) sendRequestWithEnv(input model.SendRequestInput, projectID, envID string, onChunk func(model.StreamChunk)) (*model.HttpResponse, error) {
	normalizedInput, err := normalizeCurlInput(input)
	if err != nil {
		logger.Warn("cURL 解析失败", "url", input.URL, "error", err.Error())
		return nil, err
	}
	input = normalizedInput

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

	var resp *model.HttpResponse
	if onChunk != nil {
		resp, err = a.httpSvc.SendRequestStreaming(input, onChunk)
	} else {
		resp, err = a.httpSvc.SendRequest(input)
	}
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

func (a *App) emitHTTPStreamEvent(streamID string, chunk model.StreamChunk) {
	if a.ctx == nil || strings.TrimSpace(streamID) == "" {
		return
	}
	wailsRuntime.EventsEmit(a.ctx, httpStreamEventName, httpStreamEventPayload{
		StreamID:   streamID,
		Kind:       chunk.Kind,
		Data:       chunk.Data,
		Raw:        chunk.Raw,
		Timestamp:  chunk.Timestamp,
		Sequence:   chunk.Sequence,
		BytesTotal: chunk.BytesTotal,
	})
}

func (a *App) SendRequest(input model.SendRequestInput) (*model.HttpResponse, error) {
	normalizedInput, err := normalizeCurlInput(input)
	if err != nil {
		logger.Warn("cURL 解析失败", "url", input.URL, "error", err.Error())
		return nil, err
	}
	input = normalizedInput

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

func (a *App) UpdateProjectTheme(id, color string) (*model.Project, error) {
	project, err := a.projectSvc.UpdateProjectTheme(id, color)
	if err != nil {
		logger.Error("更新项目主题色失败", "id", id, "color", color, "error", err.Error())
		return nil, err
	}
	logger.Info("更新项目主题色成功", "id", id, "themeColor", project.ThemeColor)
	return project, nil
}

func (a *App) UpdateProjectDescription(id, description string) (*model.Project, error) {
	project, err := a.projectSvc.UpdateProjectDescription(id, description)
	if err != nil {
		logger.Error("更新项目描述失败", "id", id, "error", err.Error())
		return nil, err
	}
	logger.Info("更新项目描述成功", "id", id)
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

// ---- 集合管理 ----

func (a *App) GetCollectionData(projectID string) (*model.CollectionData, error) {
	data, err := a.requestSvc.GetCollectionData(projectID)
	if err != nil {
		logger.Error("获取集合数据失败", "projectID", projectID, "error", err.Error())
		return nil, err
	}
	return data, nil
}

func (a *App) MoveCollectionNode(projectID, nodeID string, nodeType model.CollectionNodeType, targetParentFolderID string, targetIndex int) error {
	if err := a.requestSvc.MoveCollectionNode(projectID, nodeID, nodeType, targetParentFolderID, targetIndex); err != nil {
		logger.Error("移动集合节点失败", "nodeID", nodeID, "nodeType", nodeType, "targetParentFolderID", targetParentFolderID, "targetIndex", targetIndex, "error", err.Error())
		return err
	}
	logger.Info("移动集合节点成功", "nodeID", nodeID, "nodeType", nodeType, "targetParentFolderID", targetParentFolderID, "targetIndex", targetIndex)
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

func (a *App) MoveFolder(projectID, folderID, targetParentID string, targetIndex int) error {
	if err := a.requestSvc.MoveFolder(projectID, folderID, targetParentID, targetIndex); err != nil {
		logger.Error("移动文件夹失败", "folderID", folderID, "targetParentID", targetParentID, "targetIndex", targetIndex, "error", err.Error())
		return err
	}
	logger.Info("移动文件夹成功", "folderID", folderID, "targetParentID", targetParentID, "targetIndex", targetIndex)
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

func (a *App) MoveRequestItem(projectID, requestID, targetFolderID string, targetIndex int) error {
	if err := a.requestSvc.MoveRequest(projectID, requestID, targetFolderID, targetIndex); err != nil {
		logger.Error("移动请求失败", "requestID", requestID, "targetFolderID", targetFolderID, "targetIndex", targetIndex, "error", err.Error())
		return err
	}
	logger.Info("移动请求成功", "requestID", requestID, "targetFolderID", targetFolderID, "targetIndex", targetIndex)
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

// ---- 重命名请求 ----

func (a *App) RenameRequest(projectID, requestID, name string) error {
	if err := a.requestSvc.RenameRequest(projectID, requestID, name); err != nil {
		logger.Error("重命名请求失败", "requestID", requestID, "name", name, "error", err.Error())
		return err
	}
	logger.Info("重命名请求成功", "requestID", requestID, "name", name)
	return nil
}

// ---- 复制 ----

func (a *App) DuplicateRequest(projectID, requestID string) (*model.RequestItem, error) {
	req, err := a.requestSvc.DuplicateRequest(projectID, requestID)
	if err != nil {
		logger.Error("复制请求失败", "requestID", requestID, "error", err.Error())
		return nil, err
	}
	logger.Info("复制请求成功", "sourceID", requestID, "newID", req.ID)
	return req, nil
}

func (a *App) DuplicateFolder(projectID, folderID string) (*model.Folder, error) {
	folder, err := a.requestSvc.DuplicateFolder(projectID, folderID)
	if err != nil {
		logger.Error("复制文件夹失败", "folderID", folderID, "error", err.Error())
		return nil, err
	}
	logger.Info("复制文件夹成功", "sourceID", folderID, "newID", folder.ID)
	return folder, nil
}

// ---- 导入导出 ----

func (a *App) ExportProjectJSON(projectID string) (string, error) {
	data, err := a.requestSvc.ExportProjectJSON(projectID)
	if err != nil {
		logger.Error("导出项目失败", "projectID", projectID, "error", err.Error())
		return "", err
	}
	logger.Info("导出项目成功", "projectID", projectID, "size", len(data))
	return string(data), nil
}

func (a *App) ImportPostmanCollection(projectID, jsonStr string) error {
	if err := a.requestSvc.ImportPostmanCollection(projectID, []byte(jsonStr)); err != nil {
		logger.Error("导入 Postman 集合失败", "projectID", projectID, "error", err.Error())
		return err
	}
	logger.Info("导入 Postman 集合成功", "projectID", projectID)
	return nil
}

func (a *App) ImportSwagger(projectID, jsonStr string) error {
	if err := a.requestSvc.ImportSwagger(projectID, []byte(jsonStr)); err != nil {
		logger.Error("导入 Swagger 失败", "projectID", projectID, "error", err.Error())
		return err
	}
	logger.Info("导入 Swagger 成功", "projectID", projectID)
	return nil
}

func (a *App) ImportFromFile(projectID, format, content string) error {
	parseAsObject := func(raw string) (map[string]json.RawMessage, string, error) {
		var data map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &data); err == nil {
			return data, raw, nil
		}

		// fallback: YAML -> JSON
		var yamlObj map[string]interface{}
		if err := yaml.Unmarshal([]byte(raw), &yamlObj); err != nil {
			return nil, "", fmt.Errorf("无法解析 JSON/YAML: %w", err)
		}
		normalized, err := json.Marshal(yamlObj)
		if err != nil {
			return nil, "", fmt.Errorf("YAML 转换失败: %w", err)
		}
		if err := json.Unmarshal(normalized, &data); err != nil {
			return nil, "", fmt.Errorf("转换后的 JSON 解析失败: %w", err)
		}
		return data, string(normalized), nil
	}

	switch format {
	case "postman":
		return a.ImportPostmanCollection(projectID, content)
	case "swagger", "openapi":
		return a.ImportSwagger(projectID, content)
	default:
		// 自动检测格式
		raw, normalizedContent, err := parseAsObject(content)
		if err != nil {
			return err
		}
		if _, ok := raw["info"]; ok {
			if _, ok2 := raw["item"]; ok2 {
				return a.ImportPostmanCollection(projectID, normalizedContent)
			}
		}
		if _, ok := raw["paths"]; ok {
			return a.ImportSwagger(projectID, normalizedContent)
		}
		return fmt.Errorf("无法识别文件格式，请选择正确的导入格式")
	}
}

// ---- 导入 ----

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

// ---- 文件对话框 ----

// OpenFileDialogJSON 打开文件选择对话框（仅 JSON 文件）
func (a *App) OpenFileDialogJSON() (string, error) {
	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择文件",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "JSON 文件 (*.json)", Pattern: "*.json"},
			{DisplayName: "YAML 文件 (*.yaml;*.yml)", Pattern: "*.yaml;*.yml"},
			{DisplayName: "所有文件", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	if selection == "" {
		return "", nil
	}
	data, err := os.ReadFile(selection)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	return string(data), nil
}

// SaveFileDialogJSON 打开保存文件对话框，将内容写入选择的文件
func (a *App) SaveFileDialogJSON(defaultFilename string, content string) error {
	selection, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "保存文件",
		DefaultFilename: defaultFilename,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "JSON 文件 (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return err
	}
	if selection == "" {
		return nil
	}
	return os.WriteFile(selection, []byte(content), 0644)
}

// SaveTextFile 打开保存文件对话框，保存文本内容（用于 cURL 等文本导出）
func (a *App) SaveTextFile(defaultFilename string, content string) error {
	selection, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "保存文件",
		DefaultFilename: defaultFilename,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Shell 脚本 (*.sh)", Pattern: "*.sh"},
			{DisplayName: "Text 文件 (*.txt)", Pattern: "*.txt"},
			{DisplayName: "所有文件", Pattern: "*.*"},
		},
	})
	if err != nil {
		return err
	}
	if selection == "" {
		return nil
	}
	return os.WriteFile(selection, []byte(content), 0644)
}

// OpenFileDialogAny 打开通用文件选择对话框（所有文件类型）
func (a *App) OpenFileDialogAny() (string, error) {
	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择文件",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "所有文件", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// SaveResponseToFile 保存响应体到用户选择的文件
func (a *App) SaveResponseToFile(defaultFilename string, content string) error {
	selection, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "保存响应",
		DefaultFilename: defaultFilename,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "JSON 文件 (*.json)", Pattern: "*.json"},
			{DisplayName: "所有文件", Pattern: "*.*"},
		},
	})
	if err != nil {
		return err
	}
	if selection == "" {
		return nil
	}
	return os.WriteFile(selection, []byte(content), 0644)
}
