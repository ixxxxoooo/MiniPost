package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"minipost/internal/model"
	"minipost/internal/pkg/jsonutil"
	"minipost/internal/repository"
)

type RequestService struct {
	store *repository.FileStore
}

func NewRequestService(store *repository.FileStore) *RequestService {
	return &RequestService{store: store}
}

func unmarshalJSONPossiblyCommented(raw []byte, target any) error {
	if err := json.Unmarshal(raw, target); err == nil {
		return nil
	}
	cleaned := jsonutil.StripComments(raw)
	return json.Unmarshal(cleaned, target)
}

func normalizeImportedVariables(variables []model.Variable) []model.Variable {
	if len(variables) == 0 {
		return nil
	}

	seen := make(map[string]int, len(variables))
	result := make([]model.Variable, 0, len(variables))
	for _, variable := range variables {
		key := strings.TrimSpace(variable.Key)
		if key == "" {
			continue
		}
		variable.Key = key
		if variable.ID == "" {
			variable.ID = uuid.New().String()
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = len(result)
		result = append(result, variable)
	}
	return result
}

func (s *RequestService) uniqueEnvironmentName(projectID, preferred string) (string, error) {
	base := strings.TrimSpace(preferred)
	if base == "" {
		base = "Imported Environment"
	}

	envs, err := s.store.ListEnvironments(projectID)
	if err != nil {
		return "", err
	}
	existing := make(map[string]struct{}, len(envs))
	for _, env := range envs {
		existing[env.Name] = struct{}{}
	}
	if _, ok := existing[base]; !ok {
		return base, nil
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s (%d)", base, index)
		if _, ok := existing[candidate]; !ok {
			return candidate, nil
		}
	}
}

func (s *RequestService) saveImportedEnvironment(projectID, preferredName string, variables []model.Variable, reuseExisting bool) error {
	variables = normalizeImportedVariables(variables)
	if len(variables) == 0 {
		return nil
	}

	base := strings.TrimSpace(preferredName)
	if reuseExisting && base != "" {
		envs, err := s.store.ListEnvironments(projectID)
		if err != nil {
			return err
		}
		for _, env := range envs {
			// 更新/覆盖策略下同名环境已存在时跳过，保留用户当前的环境配置
			if env.Name == base {
				return nil
			}
		}
	}

	name, err := s.uniqueEnvironmentName(projectID, base)
	if err != nil {
		return err
	}

	return s.store.SaveEnvironment(&model.Environment{
		ID:        uuid.New().String(),
		Name:      name,
		ProjectID: projectID,
		Variables: variables,
	})
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
	return fmt.Errorf("文件夹 %s 不存在", folderID)
}

func (s *RequestService) MoveCollectionNode(projectID, nodeID string, nodeType model.CollectionNodeType, targetParentID string, targetIndex int) error {
	return s.store.MoveCollectionNode(projectID, nodeID, nodeType, targetParentID, targetIndex)
}

// RestoreCollectionSubtree 把删除的文件夹/请求子树放回项目（撤销删除）。
func (s *RequestService) RestoreCollectionSubtree(projectID string, sub model.CollectionData) error {
	return s.store.RestoreCollectionSubtree(projectID, sub)
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
	return fmt.Errorf("请求 %s 不存在", requestID)
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
	data, err := s.store.GetCollectionData(projectID)
	if err != nil {
		return nil, err
	}

	folderByID := make(map[string]model.Folder, len(data.Folders))
	for _, f := range data.Folders {
		folderByID[f.ID] = f
	}
	requestByID := make(map[string]model.RequestItem, len(data.Requests))
	for _, r := range data.Requests {
		requestByID[r.ID] = r
	}

	source, ok := folderByID[folderID]
	if !ok {
		return nil, fmt.Errorf("文件夹 %s 不存在", folderID)
	}

	// 按父节点归类并排序子节点，保证复制后保持原有的层级与顺序。
	childrenByParent := make(map[string][]model.CollectionNode)
	for _, node := range data.TreeNodes {
		childrenByParent[node.ParentFolderID] = append(childrenByParent[node.ParentFolderID], node)
	}
	for parent := range childrenByParent {
		nodes := childrenByParent[parent]
		sort.SliceStable(nodes, func(i, j int) bool {
			if nodes[i].SortOrder == nodes[j].SortOrder {
				if nodes[i].NodeType == nodes[j].NodeType {
					return nodes[i].NodeID < nodes[j].NodeID
				}
				return nodes[i].NodeType < nodes[j].NodeType
			}
			return nodes[i].SortOrder < nodes[j].SortOrder
		})
		childrenByParent[parent] = nodes
	}

	// 递归复制文件夹及其全部后代（子文件夹 + 请求）。
	var duplicateSubtree func(srcFolderID, dstParentID, name string) (*model.Folder, error)
	duplicateSubtree = func(srcFolderID, dstParentID, name string) (*model.Folder, error) {
		newFolder := &model.Folder{
			ID:        uuid.New().String(),
			Name:      name,
			ProjectID: projectID,
			ParentID:  dstParentID,
			SortOrder: 0,
		}
		if err := s.store.SaveFolder(newFolder); err != nil {
			return nil, err
		}

		for _, node := range childrenByParent[srcFolderID] {
			switch node.NodeType {
			case model.CollectionNodeTypeFolder:
				child, exists := folderByID[node.NodeID]
				if !exists {
					continue
				}
				if _, err := duplicateSubtree(node.NodeID, newFolder.ID, child.Name); err != nil {
					return nil, err
				}
			case model.CollectionNodeTypeRequest:
				r, exists := requestByID[node.NodeID]
				if !exists {
					continue
				}
				now := time.Now().UTC().Format(time.RFC3339)
				dupReq := r
				dupReq.ID = uuid.New().String()
				dupReq.FolderID = newFolder.ID
				dupReq.CreatedAt = now
				dupReq.UpdatedAt = now
				if err := s.store.SaveRequest(&dupReq); err != nil {
					return nil, err
				}
			}
		}
		return newFolder, nil
	}

	return duplicateSubtree(folderID, source.ParentID, source.Name+" (副本)")
}
