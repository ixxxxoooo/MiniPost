package service

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"minipost/internal/model"
)

// 导入冲突处理策略
const (
	ImportStrategyCopy      = "copy"      // 导入副本：冲突时仍然新建请求（历史默认行为）
	ImportStrategyUpdate    = "update"    // 更新已有：合并导入内容并保留用户修改
	ImportStrategyOverwrite = "overwrite" // 覆盖：用导入内容完全替换已有请求
)

// ImportSummary 描述一次导入的处理结果统计
type ImportSummary struct {
	Created     int `json:"created"`
	Updated     int `json:"updated"`
	Overwritten int `json:"overwritten"`
}

// ImportConflict 描述一条与项目中已有请求冲突的导入项
type ImportConflict struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Name   string `json:"name"`
}

func normalizeImportStrategy(strategy string) string {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case ImportStrategyUpdate:
		return ImportStrategyUpdate
	case ImportStrategyOverwrite:
		return ImportStrategyOverwrite
	default:
		return ImportStrategyCopy
	}
}

// requestConflictKey 以“方法 + 去掉查询串的 URL”作为重复判定键，
// 这样用户修改过 query 参数值的请求在重新导入时仍能命中同一接口。
func requestConflictKey(method, rawURL string) string {
	urlPart := strings.TrimSpace(rawURL)
	if idx := strings.Index(urlPart, "?"); idx >= 0 {
		urlPart = urlPart[:idx]
	}
	return strings.ToUpper(strings.TrimSpace(method)) + "\x00" + urlPart
}

// newImportedRequestTemplate 构建一条尚未落库的导入请求模板
func newImportedRequestTemplate(name string) *model.RequestItem {
	return &model.RequestItem{
		Name:    name,
		Method:  "GET",
		URL:     "",
		Params:  []model.KeyValue{},
		Headers: []model.KeyValue{},
		Body:    model.RequestBody{Type: "none"},
		Auth:    model.AuthConfig{Type: "none"},
	}
}

// importApplier 在一次导入过程中按策略落地请求与文件夹
type importApplier struct {
	svc       *RequestService
	projectID string
	strategy  string
	requests  map[string]*model.RequestItem // 冲突键 -> 已有请求
	folders   map[string]string             // 父目录ID+名称 -> 已有文件夹ID
	summary   ImportSummary
}

func (s *RequestService) newImportApplier(projectID, strategy string) (*importApplier, error) {
	applier := &importApplier{
		svc:       s,
		projectID: projectID,
		strategy:  normalizeImportStrategy(strategy),
		requests:  map[string]*model.RequestItem{},
		folders:   map[string]string{},
	}
	if applier.strategy == ImportStrategyCopy {
		return applier, nil
	}

	requests, err := s.store.ListRequests(projectID)
	if err != nil {
		return nil, err
	}
	for index := range requests {
		request := requests[index]
		key := requestConflictKey(request.Method, request.URL)
		if _, ok := applier.requests[key]; !ok {
			applier.requests[key] = &request
		}
	}

	folders, err := s.store.ListFolders(projectID)
	if err != nil {
		return nil, err
	}
	for _, folder := range folders {
		key := folderConflictKey(folder.ParentID, folder.Name)
		if _, ok := applier.folders[key]; !ok {
			applier.folders[key] = folder.ID
		}
	}
	return applier, nil
}

func folderConflictKey(parentID, name string) string {
	return parentID + "\x00" + strings.TrimSpace(name)
}

// ensureFolder 在更新/覆盖策略下复用同名文件夹，避免重复导入产生重复目录
func (a *importApplier) ensureFolder(parentID, name string) (string, error) {
	key := folderConflictKey(parentID, name)
	if a.strategy != ImportStrategyCopy {
		if id, ok := a.folders[key]; ok {
			return id, nil
		}
	}
	folder, err := a.svc.CreateFolder(a.projectID, parentID, name)
	if err != nil {
		return "", err
	}
	a.folders[key] = folder.ID
	return folder.ID, nil
}

// applyRequest 按策略落地一条导入请求
func (a *importApplier) applyRequest(folderID string, imported *model.RequestItem) error {
	if a.strategy != ImportStrategyCopy {
		if existing, ok := a.requests[requestConflictKey(imported.Method, imported.URL)]; ok {
			merged := *existing
			if a.strategy == ImportStrategyOverwrite {
				overwriteRequestFromImport(&merged, imported)
				a.summary.Overwritten++
			} else {
				mergeRequestFromImport(&merged, imported)
				a.summary.Updated++
			}
			if err := a.svc.SaveRequest(&merged); err != nil {
				return err
			}
			*existing = merged
			return nil
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := *imported
	item.ID = uuid.New().String()
	item.ProjectID = a.projectID
	item.FolderID = folderID
	item.CreatedAt = now
	item.UpdatedAt = now
	if err := a.svc.store.SaveRequest(&item); err != nil {
		return err
	}
	a.summary.Created++
	return nil
}

// mergeRequestFromImport 将导入内容合并进已有请求，保留用户已做的修改：
// - 名称、URL、参数/Header 中用户填写的值、非空 Body、已配置的 Auth 均保留
// - 追加导入文档中新增的参数与 Header，并以导入文档刷新描述信息
func mergeRequestFromImport(existing, imported *model.RequestItem) {
	existing.Method = imported.Method
	if strings.TrimSpace(existing.Name) == "" {
		existing.Name = imported.Name
	}
	existing.Params = mergeKeyValuesPreservingUser(existing.Params, imported.Params)
	existing.Headers = mergeKeyValuesPreservingUser(existing.Headers, imported.Headers)
	if !requestBodyHasContent(existing.Body) {
		existing.Body = imported.Body
	}
	if !authConfigured(existing.Auth) && authConfigured(imported.Auth) {
		existing.Auth = imported.Auth
	}
}

// overwriteRequestFromImport 用导入内容完全替换已有请求（保留 ID、目录位置与创建时间）
func overwriteRequestFromImport(existing, imported *model.RequestItem) {
	existing.Name = imported.Name
	existing.Method = imported.Method
	existing.URL = imported.URL
	existing.Params = imported.Params
	existing.Headers = imported.Headers
	existing.Body = imported.Body
	existing.Auth = imported.Auth
}

// mergeKeyValuesPreservingUser 以已有键值为基础合并导入键值：
// 相同 key 保留用户当前 value，仅刷新描述；新 key 追加到末尾。
func mergeKeyValuesPreservingUser(existing, imported []model.KeyValue) []model.KeyValue {
	result := make([]model.KeyValue, len(existing))
	copy(result, existing)

	index := make(map[string]int, len(result))
	for i, kv := range result {
		key := strings.TrimSpace(kv.Key)
		if key == "" {
			continue
		}
		if _, ok := index[key]; !ok {
			index[key] = i
		}
	}

	for _, kv := range imported {
		key := strings.TrimSpace(kv.Key)
		if key == "" {
			continue
		}
		if i, ok := index[key]; ok {
			if strings.TrimSpace(kv.Description) != "" {
				result[i].Description = kv.Description
			}
			continue
		}
		result = append(result, kv)
		index[key] = len(result) - 1
	}
	return result
}

func requestBodyHasContent(body model.RequestBody) bool {
	switch body.Type {
	case "json":
		return strings.TrimSpace(body.JSON) != ""
	case "raw":
		return strings.TrimSpace(body.Raw) != ""
	case "form-urlencoded":
		for _, kv := range body.FormUrlEncoded {
			if strings.TrimSpace(kv.Key) != "" {
				return true
			}
		}
		return false
	case "form-data":
		for _, item := range body.FormData {
			if strings.TrimSpace(item.Key) != "" {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func authConfigured(auth model.AuthConfig) bool {
	authType := strings.ToLower(strings.TrimSpace(auth.Type))
	return authType != "" && authType != "none"
}

// ---- 导入冲突预检 ----

type importedRequestSignature struct {
	Method string
	URL    string
	Name   string
}

func (s *RequestService) findImportConflicts(projectID string, incoming []importedRequestSignature) ([]ImportConflict, error) {
	requests, err := s.store.ListRequests(projectID)
	if err != nil {
		return nil, err
	}
	existing := make(map[string]struct{}, len(requests))
	for _, request := range requests {
		existing[requestConflictKey(request.Method, request.URL)] = struct{}{}
	}

	conflicts := make([]ImportConflict, 0)
	seen := make(map[string]struct{})
	for _, signature := range incoming {
		if strings.TrimSpace(signature.URL) == "" {
			continue
		}
		key := requestConflictKey(signature.Method, signature.URL)
		if _, ok := existing[key]; !ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		conflicts = append(conflicts, ImportConflict{
			Method: strings.ToUpper(strings.TrimSpace(signature.Method)),
			URL:    strings.TrimSpace(signature.URL),
			Name:   signature.Name,
		})
	}
	return conflicts, nil
}

func collectPostmanRequestSignatures(items []postmanItem, collector *[]importedRequestSignature) {
	for _, item := range items {
		if len(item.Item) > 0 {
			collectPostmanRequestSignatures(item.Item, collector)
			continue
		}
		if item.Request == nil {
			continue
		}
		*collector = append(*collector, importedRequestSignature{
			Method: item.Request.Method,
			URL:    item.Request.URL.Raw,
			Name:   item.Name,
		})
	}
}

// PreviewPostmanConflicts 预检 Postman Collection 导入与已有请求的 URL 冲突
func (s *RequestService) PreviewPostmanConflicts(projectID string, raw []byte) ([]ImportConflict, error) {
	var collection postmanCollection
	if err := unmarshalJSONPossiblyCommented(raw, &collection); err != nil {
		return nil, err
	}
	incoming := make([]importedRequestSignature, 0)
	collectPostmanRequestSignatures(collection.Item, &incoming)
	return s.findImportConflicts(projectID, incoming)
}

// resolveOpenAPIBaseURLTemplate 计算导入后请求 URL 的 base 前缀（与 importOpenAPIEnvironments 保持一致，但无落库副作用）
func resolveOpenAPIBaseURLTemplate(doc openAPIDoc, sourceURL string) string {
	if len(doc.Servers) > 0 {
		return "{{baseUrl}}"
	}
	if strings.TrimSpace(doc.Host) == "" {
		if inferBaseURLFromImportSource(sourceURL) == "" {
			return ""
		}
		return "{{baseUrl}}"
	}
	return "{{baseUrl}}"
}

// PreviewSwaggerConflicts 预检 OpenAPI/Swagger 导入与已有请求的 URL 冲突
func (s *RequestService) PreviewSwaggerConflicts(projectID string, raw []byte, sourceURL string) ([]ImportConflict, error) {
	var doc openAPIDoc
	if err := unmarshalJSONPossiblyCommented(raw, &doc); err != nil {
		return nil, err
	}

	baseURLTemplate := resolveOpenAPIBaseURLTemplate(doc, sourceURL)
	incoming := make([]importedRequestSignature, 0)
	for pathStr, pathItem := range doc.Paths {
		for _, entry := range collectOpenAPIOperations(pathItem) {
			if entry.operation == nil {
				continue
			}
			name := entry.operation.Summary
			if name == "" {
				name = entry.operation.OperationID
			}
			incoming = append(incoming, importedRequestSignature{
				Method: entry.method,
				URL:    joinURLTemplate(baseURLTemplate, replacePathPlaceholders(pathStr)),
				Name:   name,
			})
		}
	}
	return s.findImportConflicts(projectID, incoming)
}
