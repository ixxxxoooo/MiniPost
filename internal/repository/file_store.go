package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"minipost/internal/model"
)

const (
	schemaVersion  = 1
	maxHistorySize = 500
)

func normalizeIndex(index int, length int) int {
	if index < 0 {
		return 0
	}
	if index > length {
		return length
	}
	return index
}

func sortNodesByOrder(items []model.CollectionNode) []model.CollectionNode {
	sorted := append([]model.CollectionNode(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].SortOrder == sorted[j].SortOrder {
			if sorted[i].NodeType == sorted[j].NodeType {
				return sorted[i].NodeID < sorted[j].NodeID
			}
			return sorted[i].NodeType < sorted[j].NodeType
		}
		return sorted[i].SortOrder < sorted[j].SortOrder
	})
	return sorted
}

func sortFoldersByOrder(items []model.Folder) []model.Folder {
	sorted := append([]model.Folder(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].SortOrder == sorted[j].SortOrder {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].SortOrder < sorted[j].SortOrder
	})
	return sorted
}

func sortRequestsByOrder(items []model.RequestItem) []model.RequestItem {
	sorted := append([]model.RequestItem(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].SortOrder == sorted[j].SortOrder {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].SortOrder < sorted[j].SortOrder
	})
	return sorted
}

func insertNodeAt(items []model.CollectionNode, item model.CollectionNode, index int) []model.CollectionNode {
	index = normalizeIndex(index, len(items))
	items = append(items, model.CollectionNode{})
	copy(items[index+1:], items[index:])
	items[index] = item
	return items
}

func reindexNodes(items []model.CollectionNode) []model.CollectionNode {
	for i := range items {
		items[i].SortOrder = i
	}
	return items
}

func isDescendantFolder(nodes []model.CollectionNode, potentialChildID, ancestorID string) bool {
	if potentialChildID == "" || ancestorID == "" {
		return false
	}
	parentByID := make(map[string]string, len(nodes))
	for _, node := range nodes {
		if node.NodeType == model.CollectionNodeTypeFolder {
			parentByID[node.NodeID] = node.ParentFolderID
		}
	}
	current := potentialChildID
	for current != "" {
		if current == ancestorID {
			return true
		}
		next, ok := parentByID[current]
		if !ok {
			break
		}
		current = next
	}
	return false
}

// FileStore 基于 JSON 文件的本地存储实现
type FileStore struct {
	baseDir string
	mu      sync.RWMutex
}

func NewFileStoreWithDir(baseDir string) (*FileStore, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("数据目录不能为空")
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	return &FileStore{baseDir: baseDir}, nil
}

func NewFileStore() (*FileStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户目录失败: %w", err)
	}
	return NewFileStoreWithDir(filepath.Join(home, ".minipost"))
}

// BaseDir 返回数据存储根目录路径
func (fs *FileStore) BaseDir() string {
	return fs.baseDir
}

// ---- 配置 ----

type AppConfig struct {
	LastProjectID string `json:"lastProjectId,omitempty"`
	Theme         string `json:"theme,omitempty"`
	SchemaVersion int    `json:"schemaVersion"`
}

func (fs *FileStore) LoadConfig() (*AppConfig, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := filepath.Join(fs.baseDir, "config.json")
	cfg := &AppConfig{SchemaVersion: schemaVersion}
	if err := fs.readJSON(path, cfg); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	return cfg, nil
}

func (fs *FileStore) SaveConfig(cfg *AppConfig) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	cfg.SchemaVersion = schemaVersion
	return fs.writeJSON(filepath.Join(fs.baseDir, "config.json"), cfg)
}

// ---- 项目 ----

func (fs *FileStore) ListProjects() ([]model.Project, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	projectsDir := filepath.Join(fs.baseDir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	var projects []model.Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projFile := filepath.Join(projectsDir, entry.Name(), "project.json")
		var proj model.Project
		if err := fs.readJSON(projFile, &proj); err == nil {
			projects = append(projects, proj)
		}
	}
	return projects, nil
}

func (fs *FileStore) GetProject(id string) (*model.Project, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var proj model.Project
	path := filepath.Join(fs.baseDir, "projects", id, "project.json")
	if err := fs.readJSON(path, &proj); err != nil {
		return nil, err
	}
	return &proj, nil
}

func (fs *FileStore) SaveProject(project *model.Project) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	project.SchemaVersion = schemaVersion
	projDir := filepath.Join(fs.baseDir, "projects", project.ID)
	if err := os.MkdirAll(projDir, 0755); err != nil {
		return err
	}
	return fs.writeJSON(filepath.Join(projDir, "project.json"), project)
}

func (fs *FileStore) DeleteProject(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return os.RemoveAll(filepath.Join(fs.baseDir, "projects", id))
}

// ---- 集合树 ----

type treeFile struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Nodes         []model.CollectionNode `json:"nodes"`
}

func (fs *FileStore) treePath(projectID string) string {
	return filepath.Join(fs.baseDir, "projects", projectID, "tree.json")
}

type foldersFile struct {
	SchemaVersion int            `json:"schemaVersion"`
	Folders       []model.Folder `json:"folders"`
}

func (fs *FileStore) foldersPath(projectID string) string {
	return filepath.Join(fs.baseDir, "projects", projectID, "folders.json")
}

type requestsFile struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Requests      []model.RequestItem `json:"requests"`
}

func (fs *FileStore) requestsPath(projectID string) string {
	return filepath.Join(fs.baseDir, "projects", projectID, "requests.json")
}

// collectionFile 合并了文件夹、请求与集合树，作为集合数据的单一权威文件。
// 单文件持久化让一次原子写即可保证三者之间的一致性，避免分散文件之间出现写入不一致。
type collectionFile struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Folders       []model.Folder         `json:"folders"`
	Requests      []model.RequestItem    `json:"requests"`
	Nodes         []model.CollectionNode `json:"nodes"`
}

func (fs *FileStore) collectionPath(projectID string) string {
	return filepath.Join(fs.baseDir, "projects", projectID, "collection.json")
}

func (fs *FileStore) loadFoldersUnlocked(projectID string) ([]model.Folder, error) {
	var data foldersFile
	if err := fs.readJSON(fs.foldersPath(projectID), &data); err != nil {
		if os.IsNotExist(err) {
			return []model.Folder{}, nil
		}
		return nil, err
	}
	return data.Folders, nil
}

func (fs *FileStore) loadRequestsUnlocked(projectID string) ([]model.RequestItem, error) {
	var data requestsFile
	if err := fs.readJSON(fs.requestsPath(projectID), &data); err != nil {
		if os.IsNotExist(err) {
			return []model.RequestItem{}, nil
		}
		return nil, err
	}
	return data.Requests, nil
}

func (fs *FileStore) buildTreeNodesFromLegacy(projectID string, folders []model.Folder, requests []model.RequestItem) []model.CollectionNode {
	folderGroups := make(map[string][]model.Folder)
	for _, folder := range folders {
		folderGroups[folder.ParentID] = append(folderGroups[folder.ParentID], folder)
	}
	for parentID := range folderGroups {
		folderGroups[parentID] = sortFoldersByOrder(folderGroups[parentID])
	}

	requestGroups := make(map[string][]model.RequestItem)
	for _, req := range requests {
		requestGroups[req.FolderID] = append(requestGroups[req.FolderID], req)
	}
	for folderID := range requestGroups {
		requestGroups[folderID] = sortRequestsByOrder(requestGroups[folderID])
	}

	parentSet := make(map[string]struct{})
	for parentID := range folderGroups {
		parentSet[parentID] = struct{}{}
	}
	for parentID := range requestGroups {
		parentSet[parentID] = struct{}{}
	}

	var nodes []model.CollectionNode
	for parentID := range parentSet {
		order := 0
		for _, folder := range folderGroups[parentID] {
			nodes = append(nodes, model.CollectionNode{
				NodeID:         folder.ID,
				NodeType:       model.CollectionNodeTypeFolder,
				ProjectID:      projectID,
				ParentFolderID: parentID,
				SortOrder:      order,
			})
			order++
		}
		for _, req := range requestGroups[parentID] {
			nodes = append(nodes, model.CollectionNode{
				NodeID:         req.ID,
				NodeType:       model.CollectionNodeTypeRequest,
				ProjectID:      projectID,
				ParentFolderID: parentID,
				SortOrder:      order,
			})
			order++
		}
	}

	return sortNodesByOrder(nodes)
}

func (fs *FileStore) syncEntitiesWithTree(projectID string, folders []model.Folder, requests []model.RequestItem, nodes []model.CollectionNode) ([]model.Folder, []model.RequestItem) {
	folderByID := make(map[string]model.Folder, len(folders))
	for _, folder := range folders {
		folderByID[folder.ID] = folder
	}
	requestByID := make(map[string]model.RequestItem, len(requests))
	for _, req := range requests {
		requestByID[req.ID] = req
	}

	sortedNodes := sortNodesByOrder(nodes)
	orderByParent := make(map[string]int)
	for _, node := range sortedNodes {
		sortOrder := orderByParent[node.ParentFolderID]
		orderByParent[node.ParentFolderID] = sortOrder + 1
		if node.NodeType == model.CollectionNodeTypeFolder {
			folder, ok := folderByID[node.NodeID]
			if !ok {
				continue
			}
			folder.ParentID = node.ParentFolderID
			folder.SortOrder = sortOrder
			folderByID[node.NodeID] = folder
			continue
		}
		req, ok := requestByID[node.NodeID]
		if !ok {
			continue
		}
		req.FolderID = node.ParentFolderID
		req.SortOrder = sortOrder
		requestByID[node.NodeID] = req
	}

	nextFolders := make([]model.Folder, 0, len(folderByID))
	for _, folder := range folderByID {
		nextFolders = append(nextFolders, folder)
	}
	nextRequests := make([]model.RequestItem, 0, len(requestByID))
	for _, req := range requestByID {
		nextRequests = append(nextRequests, req)
	}
	return nextFolders, nextRequests
}

// loadLegacyTreeNodesUnlocked 仅用于迁移：从旧的 tree.json 读取树节点，
// 不存在时由分散的 folders/requests 推导，且不再单独回写 tree.json。
func (fs *FileStore) loadLegacyTreeNodesUnlocked(projectID string, folders []model.Folder, requests []model.RequestItem) ([]model.CollectionNode, error) {
	var data treeFile
	if err := fs.readJSON(fs.treePath(projectID), &data); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		return fs.buildTreeNodesFromLegacy(projectID, folders, requests), nil
	}
	return sortNodesByOrder(data.Nodes), nil
}

// loadCollectionUnlocked 读取集合数据。优先读合并后的 collection.json；
// 若不存在则从旧的分散文件（folders/requests/tree）构建并标记 migrated=true，由调用方落盘迁移。
func (fs *FileStore) loadCollectionUnlocked(projectID string) (data *model.CollectionData, migrated bool, err error) {
	var cf collectionFile
	if readErr := fs.readJSON(fs.collectionPath(projectID), &cf); readErr == nil {
		return &model.CollectionData{
			Folders:   cf.Folders,
			Requests:  cf.Requests,
			TreeNodes: cf.Nodes,
		}, false, nil
	} else if !os.IsNotExist(readErr) {
		return nil, false, readErr
	}

	// collection.json 不存在：从旧的分散文件迁移。
	folders, err := fs.loadFoldersUnlocked(projectID)
	if err != nil {
		return nil, false, err
	}
	requests, err := fs.loadRequestsUnlocked(projectID)
	if err != nil {
		return nil, false, err
	}
	nodes, err := fs.loadLegacyTreeNodesUnlocked(projectID, folders, requests)
	if err != nil {
		return nil, false, err
	}
	return &model.CollectionData{
		Folders:   folders,
		Requests:  requests,
		TreeNodes: nodes,
	}, true, nil
}

// saveCollectionUnlocked 将集合数据以单文件原子写入 collection.json。
func (fs *FileStore) saveCollectionUnlocked(projectID string, data *model.CollectionData) error {
	return fs.writeJSON(fs.collectionPath(projectID), &collectionFile{
		SchemaVersion: schemaVersion,
		Folders:       data.Folders,
		Requests:      data.Requests,
		Nodes:         data.TreeNodes,
	})
}

func (fs *FileStore) getCollectionDataUnlocked(projectID string) (*model.CollectionData, error) {
	data, migrated, err := fs.loadCollectionUnlocked(projectID)
	if err != nil {
		return nil, err
	}

	folders, requests := fs.syncEntitiesWithTree(projectID, data.Folders, data.Requests, data.TreeNodes)
	data.Folders = folders
	data.Requests = requests
	data.TreeNodes = sortNodesByOrder(data.TreeNodes)

	// 仅在首次迁移（collection.json 不存在）时落盘，普通读路径不回写。
	if migrated {
		if err := fs.saveCollectionUnlocked(projectID, data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (fs *FileStore) saveCollectionDataUnlocked(projectID string, data *model.CollectionData) error {
	folders, requests := fs.syncEntitiesWithTree(projectID, data.Folders, data.Requests, data.TreeNodes)
	data.Folders = folders
	data.Requests = requests
	data.TreeNodes = sortNodesByOrder(data.TreeNodes)
	return fs.saveCollectionUnlocked(projectID, data)
}

func (fs *FileStore) ListFolders(projectID string) ([]model.Folder, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 读路径只返回与 tree 同步后的内存数据，不再回写三份 JSON 文件，
	// 避免「读」产生磁盘写副作用（tree 始终是权威源，下次读会重新同步）。
	data, err := fs.getCollectionDataUnlocked(projectID)
	if err != nil {
		return nil, err
	}
	return data.Folders, nil
}

func (fs *FileStore) SaveFolder(folder *model.Folder) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := fs.getCollectionDataUnlocked(folder.ProjectID)
	if err != nil {
		return err
	}

	found := false
	for i, existing := range data.Folders {
		if existing.ID == folder.ID {
			data.Folders[i].Name = folder.Name
			found = true
			break
		}
	}
	if !found {
		data.Folders = append(data.Folders, *folder)
		data.TreeNodes = append(data.TreeNodes, model.CollectionNode{
			NodeID:         folder.ID,
			NodeType:       model.CollectionNodeTypeFolder,
			ProjectID:      folder.ProjectID,
			ParentFolderID: folder.ParentID,
			SortOrder:      len(sortNodesByOrder(filterNodesByParentAndType(data.TreeNodes, folder.ParentID, ""))),
		})
	}

	return fs.saveCollectionDataUnlocked(folder.ProjectID, data)
}

func filterNodesByParentAndType(nodes []model.CollectionNode, parentFolderID string, nodeType model.CollectionNodeType) []model.CollectionNode {
	var filtered []model.CollectionNode
	for _, node := range nodes {
		if node.ParentFolderID != parentFolderID {
			continue
		}
		if nodeType != "" && node.NodeType != nodeType {
			continue
		}
		filtered = append(filtered, node)
	}
	return sortNodesByOrder(filtered)
}

func (fs *FileStore) MoveCollectionNode(projectID, nodeID string, nodeType model.CollectionNodeType, targetParentFolderID string, targetIndex int) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := fs.getCollectionDataUnlocked(projectID)
	if err != nil {
		return err
	}

	movingIndex := -1
	for i := range data.TreeNodes {
		if data.TreeNodes[i].NodeID == nodeID && data.TreeNodes[i].NodeType == nodeType {
			movingIndex = i
			break
		}
	}
	if movingIndex < 0 {
		return fmt.Errorf("节点 %s 不存在", nodeID)
	}

	if nodeType == model.CollectionNodeTypeFolder {
		if targetParentFolderID == nodeID {
			return fmt.Errorf("不能将文件夹移动到自身")
		}
		if isDescendantFolder(data.TreeNodes, targetParentFolderID, nodeID) {
			return fmt.Errorf("不能将文件夹移动到其子文件夹中")
		}
	}

	moving := data.TreeNodes[movingIndex]
	moving.ParentFolderID = targetParentFolderID

	var siblings []model.CollectionNode
	var remaining []model.CollectionNode
	for i, node := range data.TreeNodes {
		if i == movingIndex {
			continue
		}
		if node.ParentFolderID == targetParentFolderID {
			siblings = append(siblings, node)
			continue
		}
		remaining = append(remaining, node)
	}

	siblings = sortNodesByOrder(siblings)
	siblings = insertNodeAt(siblings, moving, targetIndex)
	siblings = reindexNodes(siblings)
	data.TreeNodes = append(remaining, siblings...)
	return fs.saveCollectionDataUnlocked(projectID, data)
}

func (fs *FileStore) MoveFolder(projectID, folderID, targetParentID string, targetIndex int) error {
	return fs.MoveCollectionNode(projectID, folderID, model.CollectionNodeTypeFolder, targetParentID, targetIndex)
}

func collectDescendantFolderIDs(nodes []model.CollectionNode, folderID string) map[string]struct{} {
	ids := map[string]struct{}{folderID: {}}
	changed := true
	for changed {
		changed = false
		for _, node := range nodes {
			if node.NodeType != model.CollectionNodeTypeFolder {
				continue
			}
			if _, ok := ids[node.ParentFolderID]; ok {
				if _, exists := ids[node.NodeID]; !exists {
					ids[node.NodeID] = struct{}{}
					changed = true
				}
			}
		}
	}
	return ids
}

func (fs *FileStore) DeleteFolder(projectID, folderID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := fs.getCollectionDataUnlocked(projectID)
	if err != nil {
		return err
	}

	folderIDs := collectDescendantFolderIDs(data.TreeNodes, folderID)
	requestIDs := make(map[string]struct{})
	for _, node := range data.TreeNodes {
		if node.NodeType == model.CollectionNodeTypeRequest {
			if _, ok := folderIDs[node.ParentFolderID]; ok {
				requestIDs[node.NodeID] = struct{}{}
			}
		}
	}

	var nextFolders []model.Folder
	for _, folder := range data.Folders {
		if _, ok := folderIDs[folder.ID]; ok {
			continue
		}
		nextFolders = append(nextFolders, folder)
	}
	var nextRequests []model.RequestItem
	for _, req := range data.Requests {
		if _, ok := requestIDs[req.ID]; ok {
			continue
		}
		nextRequests = append(nextRequests, req)
	}
	var nextNodes []model.CollectionNode
	for _, node := range data.TreeNodes {
		if _, ok := folderIDs[node.NodeID]; ok && node.NodeType == model.CollectionNodeTypeFolder {
			continue
		}
		if _, ok := requestIDs[node.NodeID]; ok && node.NodeType == model.CollectionNodeTypeRequest {
			continue
		}
		nextNodes = append(nextNodes, node)
	}
	data.Folders = nextFolders
	data.Requests = nextRequests
	data.TreeNodes = nextNodes
	return fs.saveCollectionDataUnlocked(projectID, data)
}

func (fs *FileStore) ListRequests(projectID string) ([]model.RequestItem, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 读路径不回写，仅返回与 tree 同步后的内存数据，详见 ListFolders 说明。
	data, err := fs.getCollectionDataUnlocked(projectID)
	if err != nil {
		return nil, err
	}
	return data.Requests, nil
}

func (fs *FileStore) GetRequest(projectID, requestID string) (*model.RequestItem, error) {
	requests, err := fs.ListRequests(projectID)
	if err != nil {
		return nil, err
	}
	for _, r := range requests {
		if r.ID == requestID {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("请求 %s 不存在", requestID)
}

func (fs *FileStore) SaveRequest(request *model.RequestItem) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := fs.getCollectionDataUnlocked(request.ProjectID)
	if err != nil {
		return err
	}

	found := false
	for i, existing := range data.Requests {
		if existing.ID == request.ID {
			data.Requests[i] = *request
			found = true
			break
		}
	}
	if !found {
		data.Requests = append(data.Requests, *request)
		data.TreeNodes = append(data.TreeNodes, model.CollectionNode{
			NodeID:         request.ID,
			NodeType:       model.CollectionNodeTypeRequest,
			ProjectID:      request.ProjectID,
			ParentFolderID: request.FolderID,
			SortOrder:      len(filterNodesByParentAndType(data.TreeNodes, request.FolderID, "")),
		})
	}

	return fs.saveCollectionDataUnlocked(request.ProjectID, data)
}

func (fs *FileStore) DeleteRequest(projectID, requestID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := fs.getCollectionDataUnlocked(projectID)
	if err != nil {
		return err
	}

	var nextRequests []model.RequestItem
	for _, req := range data.Requests {
		if req.ID == requestID {
			continue
		}
		nextRequests = append(nextRequests, req)
	}
	var nextNodes []model.CollectionNode
	for _, node := range data.TreeNodes {
		if node.NodeType == model.CollectionNodeTypeRequest && node.NodeID == requestID {
			continue
		}
		nextNodes = append(nextNodes, node)
	}
	data.Requests = nextRequests
	data.TreeNodes = nextNodes
	return fs.saveCollectionDataUnlocked(projectID, data)
}

func (fs *FileStore) MoveRequest(projectID, requestID, targetFolderID string, targetIndex int) error {
	return fs.MoveCollectionNode(projectID, requestID, model.CollectionNodeTypeRequest, targetFolderID, targetIndex)
}

func (fs *FileStore) GetCollectionData(projectID string) (*model.CollectionData, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 读路径不回写，仅返回与 tree 同步后的内存数据，详见 ListFolders 说明。
	data, err := fs.getCollectionDataUnlocked(projectID)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ---- 环境变量 ----

type environmentsFile struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Environments  []model.Environment `json:"environments"`
}

func (fs *FileStore) envsPath(projectID string) string {
	return filepath.Join(fs.baseDir, "projects", projectID, "environments.json")
}

func (fs *FileStore) ListEnvironments(projectID string) ([]model.Environment, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var data environmentsFile
	if err := fs.readJSON(fs.envsPath(projectID), &data); err != nil {
		if os.IsNotExist(err) {
			return []model.Environment{}, nil
		}
		return nil, err
	}
	return data.Environments, nil
}

func (fs *FileStore) SaveEnvironment(env *model.Environment) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.envsPath(env.ProjectID)
	var data environmentsFile
	_ = fs.readJSON(path, &data)

	found := false
	for i, e := range data.Environments {
		if e.ID == env.ID {
			data.Environments[i] = *env
			found = true
			break
		}
	}
	if !found {
		data.Environments = append(data.Environments, *env)
	}

	data.SchemaVersion = schemaVersion
	return fs.writeJSON(path, &data)
}

func (fs *FileStore) DeleteEnvironment(projectID, envID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.envsPath(projectID)
	var data environmentsFile
	if err := fs.readJSON(path, &data); err != nil {
		return err
	}

	var filtered []model.Environment
	for _, e := range data.Environments {
		if e.ID != envID {
			filtered = append(filtered, e)
		}
	}
	data.Environments = filtered
	data.SchemaVersion = schemaVersion
	return fs.writeJSON(path, &data)
}

// ---- 历史记录 ----

type historyFile struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Entries       []model.HistoryEntry `json:"entries"`
}

func (fs *FileStore) historyPath(projectID string) string {
	return filepath.Join(fs.baseDir, "projects", projectID, "history.json")
}

func (fs *FileStore) GetHistory(projectID string) ([]model.HistoryEntry, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var data historyFile
	if err := fs.readJSON(fs.historyPath(projectID), &data); err != nil {
		if os.IsNotExist(err) {
			return []model.HistoryEntry{}, nil
		}
		return nil, err
	}
	return data.Entries, nil
}

func (fs *FileStore) AddHistory(projectID string, entry *model.HistoryEntry) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.historyPath(projectID)
	var data historyFile
	_ = fs.readJSON(path, &data)

	entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	data.Entries = append([]model.HistoryEntry{*entry}, data.Entries...)
	if len(data.Entries) > maxHistorySize {
		data.Entries = data.Entries[:maxHistorySize]
	}

	data.SchemaVersion = schemaVersion
	return fs.writeJSON(path, &data)
}

func (fs *FileStore) ClearHistory(projectID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return fs.writeJSON(fs.historyPath(projectID), &historyFile{SchemaVersion: schemaVersion})
}

// ---- 底层工具方法 ----

func (fs *FileStore) readJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (fs *FileStore) writeJSON(path string, v interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	// 先写临时文件再 rename，避免写入中断导致 JSON 损坏。
	tmpPath := path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	writeErr := func(cleanup error) error {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return cleanup
	}

	if _, err := file.Write(data); err != nil {
		return writeErr(err)
	}
	if err := file.Sync(); err != nil {
		return writeErr(err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
