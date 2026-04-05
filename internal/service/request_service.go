package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"minipost/internal/model"
	"minipost/internal/repository"
)

// ---- Postman Collection v2.1 结构 ----

type postmanCollection struct {
	Info postmanInfo   `json:"info"`
	Item []postmanItem `json:"item"`
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

type postmanReq struct {
	Method string       `json:"method"`
	Header []postmanKV  `json:"header"`
	Body   *postmanBody `json:"body,omitempty"`
	URL    postmanURL   `json:"url"`
}

type postmanURL struct {
	Raw string `json:"raw"`
}

type postmanKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type postmanBody struct {
	Mode       string      `json:"mode"`
	Raw        string      `json:"raw,omitempty"`
	URLEncoded []postmanKV `json:"urlencoded,omitempty"`
	Options    postmanOpt  `json:"options,omitempty"`
}

type postmanOpt struct {
	Raw struct {
		Language string `json:"language"`
	} `json:"raw"`
}

// ---- OpenAPI/Swagger 结构 ----

type openAPIDoc struct {
	Swagger  string                          `json:"swagger,omitempty"`
	OpenAPI  string                          `json:"openapi,omitempty"`
	Host     string                          `json:"host,omitempty"`
	BasePath string                          `json:"basePath,omitempty"`
	Schemes  []string                        `json:"schemes,omitempty"`
	Servers  []openAPIServer                 `json:"servers,omitempty"`
	Paths    map[string]map[string]openAPIOp `json:"paths"`
}

type openAPIServer struct {
	URL string `json:"url"`
}

type openAPIOp struct {
	Tags        []string `json:"tags,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	OperationID string   `json:"operationId,omitempty"`
}

type RequestService struct {
	store *repository.FileStore
}

func NewRequestService(store *repository.FileStore) *RequestService {
	return &RequestService{store: store}
}

func (s *RequestService) GetCollectionData(projectID string) (*model.CollectionData, error) {
	return s.store.GetCollectionData(projectID)
}

func (s *RequestService) ListRequests(projectID string) ([]model.RequestItem, error) {
	return s.store.ListRequests(projectID)
}

func (s *RequestService) CreateRequest(projectID, folderID, name string) (*model.RequestItem, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	req := &model.RequestItem{
		ID:        uuid.New().String(),
		Name:      name,
		Method:    "GET",
		URL:       "",
		Params:    []model.KeyValue{},
		Headers:   []model.KeyValue{},
		Body:      model.RequestBody{Type: "none"},
		Auth:      model.AuthConfig{Type: "none"},
		FolderID:  folderID,
		SortOrder: 0,
		ProjectID: projectID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.SaveRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *RequestService) SaveRequest(request *model.RequestItem) error {
	request.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return s.store.SaveRequest(request)
}

func (s *RequestService) DeleteRequest(projectID, requestID string) error {
	return s.store.DeleteRequest(projectID, requestID)
}

func (s *RequestService) ListFolders(projectID string) ([]model.Folder, error) {
	return s.store.ListFolders(projectID)
}

func (s *RequestService) CreateFolder(projectID, parentID, name string) (*model.Folder, error) {
	folder := &model.Folder{
		ID:        uuid.New().String(),
		Name:      name,
		ProjectID: projectID,
		ParentID:  parentID,
		SortOrder: 0,
	}
	if err := s.store.SaveFolder(folder); err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *RequestService) RenameFolder(projectID, folderID, name string) error {
	folders, err := s.store.ListFolders(projectID)
	if err != nil {
		return err
	}
	for _, f := range folders {
		if f.ID == folderID {
			f.Name = name
			return s.store.SaveFolder(&f)
		}
	}
	return nil
}

func (s *RequestService) MoveCollectionNode(projectID, nodeID string, nodeType model.CollectionNodeType, targetParentID string, targetIndex int) error {
	return s.store.MoveCollectionNode(projectID, nodeID, nodeType, targetParentID, targetIndex)
}

func (s *RequestService) MoveFolder(projectID, folderID, targetParentID string, targetIndex int) error {
	return s.store.MoveFolder(projectID, folderID, targetParentID, targetIndex)
}

func (s *RequestService) MoveRequest(projectID, requestID, targetFolderID string, targetIndex int) error {
	return s.store.MoveRequest(projectID, requestID, targetFolderID, targetIndex)
}

func (s *RequestService) DeleteFolder(projectID, folderID string) error {
	return s.store.DeleteFolder(projectID, folderID)
}

func (s *RequestService) RenameRequest(projectID, requestID, name string) error {
	requests, err := s.store.ListRequests(projectID)
	if err != nil {
		return err
	}
	for _, r := range requests {
		if r.ID == requestID {
			r.Name = name
			r.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			return s.store.SaveRequest(&r)
		}
	}
	return nil
}

func (s *RequestService) DuplicateRequest(projectID, requestID string) (*model.RequestItem, error) {
	requests, err := s.store.ListRequests(projectID)
	if err != nil {
		return nil, err
	}
	for _, r := range requests {
		if r.ID == requestID {
			now := time.Now().UTC().Format(time.RFC3339)
			dup := r
			dup.ID = uuid.New().String()
			dup.Name = r.Name + " (副本)"
			dup.CreatedAt = now
			dup.UpdatedAt = now
			if err := s.store.SaveRequest(&dup); err != nil {
				return nil, err
			}
			return &dup, nil
		}
	}
	return nil, fmt.Errorf("请求 %s 不存在", requestID)
}

func (s *RequestService) DuplicateFolder(projectID, folderID string) (*model.Folder, error) {
	folders, err := s.store.ListFolders(projectID)
	if err != nil {
		return nil, err
	}
	for _, f := range folders {
		if f.ID == folderID {
			dup := &model.Folder{
				ID:        uuid.New().String(),
				Name:      f.Name + " (副本)",
				ProjectID: projectID,
				ParentID:  f.ParentID,
				SortOrder: 0,
			}
			if err := s.store.SaveFolder(dup); err != nil {
				return nil, err
			}
			// 复制文件夹内的请求
			requests, _ := s.store.ListRequests(projectID)
			for _, r := range requests {
				if r.FolderID == folderID {
					now := time.Now().UTC().Format(time.RFC3339)
					dupReq := r
					dupReq.ID = uuid.New().String()
					dupReq.FolderID = dup.ID
					dupReq.CreatedAt = now
					dupReq.UpdatedAt = now
					_ = s.store.SaveRequest(&dupReq)
				}
			}
			return dup, nil
		}
	}
	return nil, fmt.Errorf("文件夹 %s 不存在", folderID)
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
				Key:   header.Key,
				Value: header.Value,
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
				Key:   field.Key,
				Value: field.Value,
			})
		}
		if len(formData) > 0 {
			postmanRequest.Body = &postmanBody{
				Mode:       "urlencoded",
				URLEncoded: formData,
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

// ImportPostmanCollection 导入 Postman Collection v2.1 格式
func (s *RequestService) ImportPostmanCollection(projectID string, raw []byte) error {
	var collection postmanCollection
	if err := json.Unmarshal(raw, &collection); err != nil {
		return fmt.Errorf("解析 Postman JSON 失败: %w", err)
	}
	return s.importPostmanItems(projectID, "", collection.Item)
}

func (s *RequestService) importPostmanItems(projectID, parentFolderID string, items []postmanItem) error {
	for _, item := range items {
		if len(item.Item) > 0 {
			// 这是一个文件夹
			folder, err := s.CreateFolder(projectID, parentFolderID, item.Name)
			if err != nil {
				return err
			}
			if err := s.importPostmanItems(projectID, folder.ID, item.Item); err != nil {
				return err
			}
		} else if item.Request != nil {
			// 这是一个请求
			req, err := s.CreateRequest(projectID, parentFolderID, item.Name)
			if err != nil {
				return err
			}
			req.Method = item.Request.Method
			if item.Request.URL.Raw != "" {
				req.URL = item.Request.URL.Raw
			}
			for _, h := range item.Request.Header {
				req.Headers = append(req.Headers, model.KeyValue{Key: h.Key, Value: h.Value})
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
						formData = append(formData, model.KeyValue{Key: kv.Key, Value: kv.Value})
					}
					req.Body = model.RequestBody{Type: "form-urlencoded", FormUrlEncoded: formData}
				}
			}
			if err := s.SaveRequest(req); err != nil {
				return err
			}
		}
	}
	return nil
}

// ImportSwagger 导入 OpenAPI/Swagger 2.0 或 3.x 格式
func (s *RequestService) ImportSwagger(projectID string, raw []byte) error {
	var doc openAPIDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("解析 OpenAPI JSON 失败: %w", err)
	}

	// 按 tag 分组创建文件夹
	tagFolders := make(map[string]string) // tag -> folderID

	for pathStr, methods := range doc.Paths {
		for method, op := range methods {
			methodUpper := strings.ToUpper(method)
			if methodUpper == "PARAMETERS" || methodUpper == "$REF" {
				continue
			}
			folderID := ""
			if len(op.Tags) > 0 {
				tag := op.Tags[0]
				if fid, ok := tagFolders[tag]; ok {
					folderID = fid
				} else {
					folder, err := s.CreateFolder(projectID, "", tag)
					if err != nil {
						return err
					}
					tagFolders[tag] = folder.ID
					folderID = folder.ID
				}
			}

			name := op.Summary
			if name == "" {
				name = op.OperationID
			}
			if name == "" {
				name = methodUpper + " " + pathStr
			}

			baseURL := ""
			if len(doc.Servers) > 0 {
				baseURL = doc.Servers[0].URL
			} else if doc.Host != "" {
				scheme := "https"
				if len(doc.Schemes) > 0 {
					scheme = doc.Schemes[0]
				}
				baseURL = scheme + "://" + doc.Host
				if doc.BasePath != "" && doc.BasePath != "/" {
					baseURL += doc.BasePath
				}
			}

			req, err := s.CreateRequest(projectID, folderID, name)
			if err != nil {
				return err
			}
			req.Method = methodUpper
			req.URL = baseURL + pathStr
			if err := s.SaveRequest(req); err != nil {
				return err
			}
		}
	}
	return nil
}
