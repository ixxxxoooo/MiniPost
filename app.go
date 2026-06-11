package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"regexp"
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
const binarySavePrefix = "__MINIPOST_BASE64__:"

var swaggerUIURLPattern = regexp.MustCompile(`(?i)\burl\s*:\s*["']([^"']+)["']`)

type httpStreamEventPayload struct {
	StreamID   string `json:"streamId"`
	Kind       string `json:"kind"`
	Data       string `json:"data"`
	Raw        string `json:"raw,omitempty"`
	Timestamp  string `json:"timestamp"`
	Sequence   int    `json:"sequence"`
	BytesTotal int64  `json:"bytesTotal,omitempty"`
}

func stripJSONComments(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}

	result := make([]byte, 0, len(raw))
	inString := false
	escapeNext := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		next := byte(0)
		hasNext := i+1 < len(raw)
		if hasNext {
			next = raw[i+1]
		}

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				result = append(result, ch)
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && hasNext && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if inString {
			result = append(result, ch)
			if escapeNext {
				escapeNext = false
				continue
			}
			if ch == '\\' {
				escapeNext = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			result = append(result, ch)
			continue
		}

		if ch == '/' && hasNext && next == '/' {
			inLineComment = true
			i++
			continue
		}
		if ch == '/' && hasNext && next == '*' {
			inBlockComment = true
			i++
			continue
		}

		result = append(result, ch)
	}

	return result
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
	parsed.Options = input.Options
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
		logger.Warn("cURL 解析失败", "target", logger.RedactRequestTarget(input.URL), "error", err.Error())
		return nil, err
	}
	input = normalizedInput

	logger.Debug("SendRequestWithEnv 调用",
		"method", input.Method,
		"target", logger.RedactRequestTarget(input.URL),
		"projectID", projectID,
		"envID", envID,
		"streaming", onChunk != nil,
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
		logger.Warn("HTTP 请求失败", "method", input.Method, "target", logger.RedactRequestTarget(input.URL), "error", err.Error())
		return nil, err
	}

	logger.Info("HTTP 请求完成",
		"method", input.Method,
		"target", logger.RedactRequestTarget(input.URL),
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
		logger.Warn("cURL 解析失败", "target", logger.RedactRequestTarget(input.URL), "error", err.Error())
		return nil, err
	}
	input = normalizedInput

	logger.Debug("SendRequest 调用", "method", input.Method, "target", logger.RedactRequestTarget(input.URL))
	resp, err := a.httpSvc.SendRequest(input)
	if err != nil {
		logger.Warn("HTTP 请求失败", "method", input.Method, "target", logger.RedactRequestTarget(input.URL), "error", err.Error())
		return nil, err
	}
	logger.Info("HTTP 请求完成", "method", input.Method, "target", logger.RedactRequestTarget(input.URL), "status", resp.StatusCode)
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

func (a *App) ImportPostmanEnvironment(projectID, jsonStr string) error {
	if err := a.requestSvc.ImportPostmanEnvironment(projectID, []byte(jsonStr)); err != nil {
		logger.Error("导入 Postman Environment 失败", "projectID", projectID, "error", err.Error())
		return err
	}
	logger.Info("导入 Postman Environment 成功", "projectID", projectID)
	return nil
}

func (a *App) ImportSwagger(projectID, jsonStr string) error {
	if err := a.requestSvc.ImportSwaggerWithSource(projectID, []byte(jsonStr), ""); err != nil {
		logger.Error("导入 Swagger 失败", "projectID", projectID, "error", err.Error())
		return err
	}
	logger.Info("导入 Swagger 成功", "projectID", projectID)
	return nil
}

func parseImportContentAsObject(raw string) (map[string]json.RawMessage, string, error) {
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &data); err == nil {
		return data, raw, nil
	}
	cleaned := stripJSONComments([]byte(raw))
	if !bytes.Equal(cleaned, []byte(raw)) {
		if err := json.Unmarshal(cleaned, &data); err == nil {
			return data, raw, nil
		}
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

// resolveImportKind 解析导入内容的实际类型（postman / postman-environment / swagger），并返回规范化后的内容
func resolveImportKind(format, content string) (string, string, error) {
	switch format {
	case "postman":
		return "postman", content, nil
	case "postman-environment":
		return "postman-environment", content, nil
	case "swagger", "openapi":
		return "swagger", content, nil
	default:
		// 自动检测格式
		raw, normalizedContent, err := parseImportContentAsObject(content)
		if err != nil {
			return "", "", err
		}
		if _, ok := raw["info"]; ok {
			if _, ok2 := raw["item"]; ok2 {
				return "postman", normalizedContent, nil
			}
		}
		if _, ok := raw["values"]; ok {
			if _, hasItem := raw["item"]; !hasItem {
				return "postman-environment", normalizedContent, nil
			}
		}
		if _, ok := raw["paths"]; ok {
			return "swagger", normalizedContent, nil
		}
		return "", "", fmt.Errorf("无法识别文件格式，请选择正确的导入格式")
	}
}

func (a *App) importCollectionContent(projectID, format, content, sourceURL string) error {
	return a.importCollectionContentWithStrategy(projectID, format, content, sourceURL, service.ImportStrategyCopy)
}

func (a *App) importCollectionContentWithStrategy(projectID, format, content, sourceURL, strategy string) error {
	kind, normalizedContent, err := resolveImportKind(format, content)
	if err != nil {
		return err
	}

	switch kind {
	case "postman":
		summary, err := a.requestSvc.ImportPostmanCollectionWithStrategy(projectID, []byte(normalizedContent), strategy)
		if err != nil {
			logger.Error("导入 Postman 集合失败", "projectID", projectID, "strategy", strategy, "error", err.Error())
			return err
		}
		logger.Info("导入 Postman 集合成功", "projectID", projectID, "strategy", strategy,
			"created", summary.Created, "updated", summary.Updated, "overwritten", summary.Overwritten)
		return nil
	case "postman-environment":
		return a.ImportPostmanEnvironment(projectID, normalizedContent)
	case "swagger":
		summary, err := a.requestSvc.ImportSwaggerWithSourceStrategy(projectID, []byte(normalizedContent), sourceURL, strategy)
		if err != nil {
			logger.Error("导入 Swagger 失败", "projectID", projectID, "strategy", strategy, "error", err.Error())
			return err
		}
		logger.Info("导入 Swagger 成功", "projectID", projectID, "strategy", strategy,
			"created", summary.Created, "updated", summary.Updated, "overwritten", summary.Overwritten)
		return nil
	default:
		return fmt.Errorf("无法识别文件格式，请选择正确的导入格式")
	}
}

// ImportPreview 导入预检结果：包含与已有请求的 URL 冲突信息
type ImportPreview struct {
	Kind          string                   `json:"kind"`
	Content       string                   `json:"content"`
	ResolvedURL   string                   `json:"resolvedURL"`
	ConflictCount int                      `json:"conflictCount"`
	Conflicts     []service.ImportConflict `json:"conflicts"`
}

const importPreviewMaxConflictSamples = 20

func (a *App) buildImportPreview(projectID, format, content, sourceURL string) (*ImportPreview, error) {
	kind, normalizedContent, err := resolveImportKind(format, content)
	if err != nil {
		return nil, err
	}

	var conflicts []service.ImportConflict
	switch kind {
	case "postman":
		conflicts, err = a.requestSvc.PreviewPostmanConflicts(projectID, []byte(normalizedContent))
	case "swagger":
		conflicts, err = a.requestSvc.PreviewSwaggerConflicts(projectID, []byte(normalizedContent), sourceURL)
	default:
		// 环境导入不涉及请求冲突
		conflicts = nil
	}
	if err != nil {
		logger.Error("导入冲突预检失败", "projectID", projectID, "kind", kind, "error", err.Error())
		return nil, err
	}

	preview := &ImportPreview{
		Kind:          kind,
		ResolvedURL:   sourceURL,
		ConflictCount: len(conflicts),
		Conflicts:     conflicts,
	}
	if len(preview.Conflicts) > importPreviewMaxConflictSamples {
		preview.Conflicts = preview.Conflicts[:importPreviewMaxConflictSamples]
	}
	logger.Info("导入冲突预检完成", "projectID", projectID, "kind", kind, "conflictCount", preview.ConflictCount)
	return preview, nil
}

// PreviewImportFromFile 预检文件导入内容与项目中已有请求的 URL 冲突
func (a *App) PreviewImportFromFile(projectID, format, content string) (*ImportPreview, error) {
	return a.buildImportPreview(projectID, format, content, "")
}

// PreviewImportFromURL 拉取远程导入内容并预检 URL 冲突；返回内容供后续导入复用，避免二次拉取
func (a *App) PreviewImportFromURL(projectID, format, sourceURL string) (*ImportPreview, error) {
	trimmedURL := strings.TrimSpace(sourceURL)
	if trimmedURL == "" {
		return nil, fmt.Errorf("导入地址不能为空")
	}

	parsedURL, err := neturl.Parse(trimmedURL)
	if err != nil {
		return nil, fmt.Errorf("导入地址无效: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("仅支持 http/https 导入地址")
	}

	content, resolvedURL, err := fetchRemoteImportContent(a.ctx, trimmedURL)
	if err != nil {
		return nil, err
	}

	preview, err := a.buildImportPreview(projectID, format, content, resolvedURL)
	if err != nil {
		return nil, err
	}
	preview.Content = content
	preview.ResolvedURL = resolvedURL
	return preview, nil
}

// ImportCollectionWithStrategy 按指定冲突策略导入集合内容（strategy: update / copy / overwrite）
func (a *App) ImportCollectionWithStrategy(projectID, format, content, sourceURL, strategy string) error {
	return a.importCollectionContentWithStrategy(projectID, format, content, sourceURL, strategy)
}

func fetchRemoteImportContent(ctx context.Context, sourceURL string) (string, string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	fetch := func(targetURL string) (string, string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return "", "", fmt.Errorf("创建导入请求失败: %w", err)
		}
		req.Header.Set("Accept", "application/json, application/yaml, text/yaml, text/plain, text/html, */*")
		req.Header.Set("User-Agent", "MiniPost/1.0")

		resp, err := client.Do(req)
		if err != nil {
			return "", "", fmt.Errorf("拉取导入地址失败: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", "", fmt.Errorf("拉取导入地址失败: HTTP %d", resp.StatusCode)
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
		if err != nil {
			return "", "", fmt.Errorf("读取导入内容失败: %w", err)
		}

		finalURL := targetURL
		if resp.Request != nil && resp.Request.URL != nil {
			finalURL = resp.Request.URL.String()
		}
		content := strings.TrimSpace(string(body))
		if content == "" {
			return "", "", fmt.Errorf("导入地址返回内容为空")
		}
		return content, finalURL, nil
	}

	content, finalURL, err := fetch(sourceURL)
	if err != nil {
		return "", "", err
	}
	if strings.Contains(strings.ToLower(content), "swaggeruibundle") {
		if matches := swaggerUIURLPattern.FindStringSubmatch(content); len(matches) == 2 {
			baseURL, parseErr := neturl.Parse(finalURL)
			if parseErr != nil {
				return content, finalURL, nil
			}
			refURL, resolveErr := baseURL.Parse(strings.TrimSpace(matches[1]))
			if resolveErr != nil {
				return content, finalURL, nil
			}
			return fetch(refURL.String())
		}
	}

	return content, finalURL, nil
}

func (a *App) ImportFromFile(projectID, format, content string) error {
	return a.importCollectionContent(projectID, format, content, "")
}

func (a *App) ImportFromURL(projectID, format, sourceURL string) error {
	trimmedURL := strings.TrimSpace(sourceURL)
	if trimmedURL == "" {
		return fmt.Errorf("导入地址不能为空")
	}

	parsedURL, err := neturl.Parse(trimmedURL)
	if err != nil {
		return fmt.Errorf("导入地址无效: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("仅支持 http/https 导入地址")
	}

	content, resolvedURL, err := fetchRemoteImportContent(a.ctx, trimmedURL)
	if err != nil {
		return err
	}

	logger.Info("开始从 URL 导入集合", "projectID", projectID, "url", logger.RedactURL(trimmedURL), "resolvedURL", logger.RedactURL(resolvedURL), "size", len(content))
	return a.importCollectionContent(projectID, format, content, resolvedURL)
}

// ---- 导入 ----

func (a *App) ImportCurl(curlCommand string) (*model.SendRequestInput, error) {
	logger.Debug("ImportCurl 调用", "cmdLength", len(curlCommand))
	result, err := httputil.ParseCurlCommand(curlCommand)
	if err != nil {
		logger.Error("cURL 解析失败", "error", err.Error())
		return nil, err
	}
	logger.Info("cURL 解析成功", "method", result.Method, "target", logger.RedactRequestTarget(result.URL))
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
	})
	if err != nil {
		return err
	}
	if selection == "" {
		return nil
	}
	data := []byte(content)
	if strings.HasPrefix(content, binarySavePrefix) {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(content, binarySavePrefix))
		if err != nil {
			return appErrors.Wrap("INVALID_BINARY_PAYLOAD", "下载内容解析失败", err)
		}
		data = decoded
	}
	return os.WriteFile(selection, data, 0644)
}
