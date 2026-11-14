// Postman Collection v2.1 / Postman 环境的结构、导入与导出。

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/google/uuid"

	"minipost/internal/model"
)

type postmanCollection struct {
	Info     postmanInfo       `json:"info"`
	Item     []postmanItem     `json:"item"`
	Variable []postmanVariable `json:"variable,omitempty"`
}

type postmanInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema,omitempty"`
}

type postmanItem struct {
	Name        string        `json:"name"`
	Item        []postmanItem `json:"item,omitempty"`
	Request     *postmanReq   `json:"request,omitempty"`
	Description string        `json:"description,omitempty"`
}

type postmanEnvironment struct {
	Name          string            `json:"name"`
	Values        []postmanVariable `json:"values"`
	VariableScope string            `json:"_postman_variable_scope,omitempty"`
}

type postmanReq struct {
	Method string       `json:"method"`
	Header []postmanKV  `json:"header"`
	Body   *postmanBody `json:"body,omitempty"`
	URL    postmanURL   `json:"url"`
}

type postmanURL struct {
	Raw string `json:"raw"`
}

func (u *postmanURL) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*u = postmanURL{}
		return nil
	}

	// Postman 有两种 url 结构：字符串或对象
	if trimmed[0] == '"' {
		var raw string
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			return err
		}
		u.Raw = raw
		return nil
	}

	var parsed struct {
		Raw      string      `json:"raw"`
		Protocol string      `json:"protocol"`
		Host     []string    `json:"host"`
		Path     []string    `json:"path"`
		Query    []postmanKV `json:"query"`
	}
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return err
	}

	if parsed.Raw != "" {
		u.Raw = parsed.Raw
		return nil
	}

	host := strings.Join(parsed.Host, ".")
	path := strings.Join(parsed.Path, "/")
	var b strings.Builder
	if parsed.Protocol != "" {
		b.WriteString(parsed.Protocol)
		b.WriteString("://")
	}
	b.WriteString(host)
	if path != "" {
		if !strings.HasPrefix(path, "/") {
			b.WriteString("/")
		}
		b.WriteString(path)
	}
	if len(parsed.Query) > 0 {
		values := url.Values{}
		for _, q := range parsed.Query {
			if strings.TrimSpace(q.Key) == "" {
				continue
			}
			values.Add(q.Key, q.Value)
		}
		encoded := values.Encode()
		if encoded != "" {
			b.WriteString("?")
			b.WriteString(encoded)
		}
	}

	u.Raw = b.String()
	return nil
}

type postmanKV struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type postmanVariable struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Type     string `json:"type,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

type postmanBody struct {
	Mode       string            `json:"mode"`
	Raw        string            `json:"raw,omitempty"`
	URLEncoded []postmanKV       `json:"urlencoded,omitempty"`
	FormData   []postmanFormData `json:"formdata,omitempty"`
	Options    postmanOpt        `json:"options,omitempty"`
}

type postmanFormData struct {
	Key         string          `json:"key"`
	Value       string          `json:"value,omitempty"`
	Description string          `json:"description,omitempty"`
	Type        string          `json:"type,omitempty"` // text | file
	Src         json.RawMessage `json:"src,omitempty"`  // string or string[]
}

type postmanOpt struct {
	Raw struct {
		Language string `json:"language"`
	} `json:"raw"`
}

func convertPostmanVariables(variables []postmanVariable) []model.Variable {
	result := make([]model.Variable, 0, len(variables))
	for _, variable := range variables {
		key := strings.TrimSpace(variable.Key)
		if key == "" {
			continue
		}
		result = append(result, model.Variable{
			ID:       uuid.New().String(),
			Key:      key,
			Value:    variable.Value,
			Enabled:  !variable.Disabled,
			IsSecret: strings.EqualFold(variable.Type, "secret"),
		})
	}
	return result
}

// ExportProjectJSON 导出项目为 Postman Collection v2.1 JSON
func (s *RequestService) ExportProjectJSON(projectID string) ([]byte, error) {
	data, err := s.store.GetCollectionData(projectID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = &model.CollectionData{}
	}

	projectName := "MiniPost Export"
	if project, err := s.store.GetProject(projectID); err == nil && strings.TrimSpace(project.Name) != "" {
		projectName = project.Name
	}

	collection := postmanCollection{
		Info: postmanInfo{
			Name:   projectName,
			Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item: s.buildPostmanItems(data),
	}

	return json.MarshalIndent(collection, "", "  ")
}

func (s *RequestService) buildPostmanItems(data *model.CollectionData) []postmanItem {
	folderByID := make(map[string]model.Folder, len(data.Folders))
	for _, folder := range data.Folders {
		folderByID[folder.ID] = folder
	}
	requestByID := make(map[string]model.RequestItem, len(data.Requests))
	for _, request := range data.Requests {
		requestByID[request.ID] = request
	}

	children := make(map[string][]model.CollectionNode)
	for _, node := range data.TreeNodes {
		parentID := strings.TrimSpace(node.ParentFolderID)
		children[parentID] = append(children[parentID], node)
	}
	for parentID := range children {
		sort.SliceStable(children[parentID], func(i, j int) bool {
			if children[parentID][i].SortOrder == children[parentID][j].SortOrder {
				if children[parentID][i].NodeType == children[parentID][j].NodeType {
					return children[parentID][i].NodeID < children[parentID][j].NodeID
				}
				return children[parentID][i].NodeType < children[parentID][j].NodeType
			}
			return children[parentID][i].SortOrder < children[parentID][j].SortOrder
		})
	}

	var walk func(parentID string) []postmanItem
	walk = func(parentID string) []postmanItem {
		nodes := children[parentID]
		result := make([]postmanItem, 0, len(nodes))
		for _, node := range nodes {
			if node.NodeType == model.CollectionNodeTypeFolder {
				folder, ok := folderByID[node.NodeID]
				if !ok {
					continue
				}
				result = append(result, postmanItem{
					Name: folder.Name,
					Item: walk(folder.ID),
				})
				continue
			}
			if node.NodeType == model.CollectionNodeTypeRequest {
				request, ok := requestByID[node.NodeID]
				if !ok {
					continue
				}
				result = append(result, postmanItem{
					Name:    request.Name,
					Request: convertRequestToPostman(request),
				})
			}
		}
		return result
	}

	return walk("")
}

func convertRequestToPostman(request model.RequestItem) *postmanReq {
	postmanRequest := &postmanReq{
		Method: strings.ToUpper(strings.TrimSpace(request.Method)),
		URL:    postmanURL{Raw: buildRequestRawURL(request)},
	}
	if postmanRequest.Method == "" {
		postmanRequest.Method = "GET"
	}

	if len(request.Headers) > 0 {
		headers := make([]postmanKV, 0, len(request.Headers))
		for _, header := range request.Headers {
			if strings.TrimSpace(header.Key) == "" {
				continue
			}
			headers = append(headers, postmanKV{
				Key:         header.Key,
				Value:       header.Value,
				Description: header.Description,
			})
		}
		postmanRequest.Header = headers
	}

	switch request.Body.Type {
	case "json":
		body := &postmanBody{
			Mode: "raw",
			Raw:  request.Body.JSON,
		}
		body.Options.Raw.Language = "json"
		postmanRequest.Body = body
	case "raw":
		postmanRequest.Body = &postmanBody{
			Mode: "raw",
			Raw:  request.Body.Raw,
		}
	case "form-urlencoded":
		formData := make([]postmanKV, 0, len(request.Body.FormUrlEncoded))
		for _, field := range request.Body.FormUrlEncoded {
			if strings.TrimSpace(field.Key) == "" {
				continue
			}
			formData = append(formData, postmanKV{
				Key:         field.Key,
				Value:       field.Value,
				Description: field.Description,
			})
		}
		if len(formData) > 0 {
			postmanRequest.Body = &postmanBody{
				Mode:       "urlencoded",
				URLEncoded: formData,
			}
		}
	case "form-data":
		formData := make([]postmanFormData, 0, len(request.Body.FormData))
		for _, field := range request.Body.FormData {
			if strings.TrimSpace(field.Key) == "" {
				continue
			}
			itemType := strings.TrimSpace(field.Type)
			if itemType == "" {
				itemType = "text"
			}
			item := postmanFormData{
				Key:         field.Key,
				Type:        itemType,
				Value:       field.Value,
				Description: field.Description,
			}
			if itemType == "file" {
				src := strings.TrimSpace(field.FilePath)
				if src == "" {
					src = strings.TrimSpace(field.Value)
				}
				if src != "" {
					if encoded, err := json.Marshal(src); err == nil {
						item.Src = encoded
					}
				}
			}
			formData = append(formData, item)
		}
		if len(formData) > 0 {
			postmanRequest.Body = &postmanBody{
				Mode:     "formdata",
				FormData: formData,
			}
		}
	}

	return postmanRequest
}

func buildRequestRawURL(request model.RequestItem) string {
	raw := strings.TrimSpace(request.URL)
	if len(request.Params) == 0 {
		return raw
	}

	values := url.Values{}
	for _, param := range request.Params {
		key := strings.TrimSpace(param.Key)
		if key == "" {
			continue
		}
		values.Add(key, param.Value)
	}
	encoded := values.Encode()
	if encoded == "" {
		return raw
	}
	if strings.Contains(raw, "?") {
		if strings.HasSuffix(raw, "?") || strings.HasSuffix(raw, "&") {
			return raw + encoded
		}
		return raw + "&" + encoded
	}
	return raw + "?" + encoded
}

// ImportPostmanCollection 导入 Postman Collection v2.1 格式（历史行为：冲突时直接新建副本）
func (s *RequestService) ImportPostmanCollection(projectID string, raw []byte) error {
	_, err := s.ImportPostmanCollectionWithStrategy(projectID, raw, ImportStrategyCopy)
	return err
}

// ImportPostmanCollectionWithStrategy 按指定冲突策略导入 Postman Collection v2.1 格式
func (s *RequestService) ImportPostmanCollectionWithStrategy(projectID string, raw []byte, strategy string) (*ImportSummary, error) {
	var collection postmanCollection
	if err := unmarshalJSONPossiblyCommented(raw, &collection); err != nil {
		return nil, fmt.Errorf("解析 Postman JSON 失败: %w", err)
	}
	applier, err := s.newImportApplier(projectID, strategy)
	if err != nil {
		return nil, err
	}
	reuseEnv := applier.strategy != ImportStrategyCopy
	if err := s.saveImportedEnvironment(projectID, strings.TrimSpace(collection.Info.Name), convertPostmanVariables(collection.Variable), reuseEnv); err != nil {
		return nil, err
	}
	if err := s.importPostmanItems(applier, "", collection.Item); err != nil {
		return nil, err
	}
	return &applier.summary, nil
}

func (s *RequestService) ImportPostmanEnvironment(projectID string, raw []byte) error {
	var env postmanEnvironment
	if err := unmarshalJSONPossiblyCommented(raw, &env); err != nil {
		return fmt.Errorf("解析 Postman Environment JSON 失败: %w", err)
	}
	name := strings.TrimSpace(env.Name)
	if name == "" {
		name = "Postman Environment"
	}
	return s.saveImportedEnvironment(projectID, name, convertPostmanVariables(env.Values), false)
}

func parsePostmanFormDataFilePath(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return strings.TrimSpace(single)
	}
	var multiple []string
	if err := json.Unmarshal(raw, &multiple); err == nil {
		for _, path := range multiple {
			trimmed := strings.TrimSpace(path)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func (s *RequestService) importPostmanItems(applier *importApplier, parentFolderID string, items []postmanItem) error {
	for _, item := range items {
		if len(item.Item) > 0 {
			// 这是一个文件夹
			folderID, err := applier.ensureFolder(parentFolderID, item.Name)
			if err != nil {
				return err
			}
			if err := s.importPostmanItems(applier, folderID, item.Item); err != nil {
				return err
			}
		} else if item.Request != nil {
			// 这是一个请求
			req := newImportedRequestTemplate(item.Name)
			req.Method = item.Request.Method
			if item.Request.URL.Raw != "" {
				req.URL = item.Request.URL.Raw
			}
			for _, h := range item.Request.Header {
				req.Headers = append(req.Headers, model.KeyValue{
					Key:         h.Key,
					Value:       h.Value,
					Description: h.Description,
				})
			}
			if item.Request.Body != nil {
				switch item.Request.Body.Mode {
				case "raw":
					req.Body = model.RequestBody{Type: "raw", Raw: item.Request.Body.Raw}
					if strings.EqualFold(item.Request.Body.Options.Raw.Language, "json") {
						req.Body.Type = "json"
						req.Body.JSON = item.Request.Body.Raw
						req.Body.Raw = ""
					}
				case "urlencoded":
					var formData []model.KeyValue
					for _, kv := range item.Request.Body.URLEncoded {
						formData = append(formData, model.KeyValue{
							Key:         kv.Key,
							Value:       kv.Value,
							Description: kv.Description,
						})
					}
					req.Body = model.RequestBody{Type: "form-urlencoded", FormUrlEncoded: formData}
				case "formdata":
					formData := make([]model.FormData, 0, len(item.Request.Body.FormData))
					for _, field := range item.Request.Body.FormData {
						itemType := strings.TrimSpace(field.Type)
						if itemType == "" {
							itemType = "text"
						}
						filePath := ""
						fileName := ""
						if itemType == "file" {
							filePath = parsePostmanFormDataFilePath(field.Src)
							if filePath != "" {
								parts := strings.Split(strings.ReplaceAll(filePath, "\\", "/"), "/")
								fileName = parts[len(parts)-1]
							}
						}
						formData = append(formData, model.FormData{
							Key:         field.Key,
							Value:       field.Value,
							Description: field.Description,
							Type:        itemType,
							FilePath:    filePath,
							FileName:    fileName,
						})
					}
					req.Body = model.RequestBody{Type: "form-data", FormData: formData}
				}
			}
			if err := applier.applyRequest(parentFolderID, req); err != nil {
				return err
			}
		}
	}
	return nil
}
