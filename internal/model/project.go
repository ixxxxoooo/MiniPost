package model

// Project 项目模型
type Project struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	SchemaVersion int    `json:"schemaVersion"`
}

// CollectionNodeType 集合树节点类型
type CollectionNodeType string

const (
	CollectionNodeTypeFolder  CollectionNodeType = "folder"
	CollectionNodeTypeRequest CollectionNodeType = "request"
)

// CollectionNode 集合树统一节点元数据
type CollectionNode struct {
	NodeID         string             `json:"nodeId"`
	NodeType       CollectionNodeType `json:"nodeType"`
	ProjectID      string             `json:"projectId"`
	ParentFolderID string             `json:"parentFolderId,omitempty"`
	SortOrder      int                `json:"sortOrder"`
}

// CollectionData 项目集合数据
type CollectionData struct {
	Folders []Folder         `json:"folders"`
	Requests []RequestItem   `json:"requests"`
	TreeNodes []CollectionNode `json:"treeNodes"`
}

// Folder 文件夹模型
type Folder struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ProjectID string `json:"projectId"`
	ParentID  string `json:"parentId,omitempty"`
	SortOrder int    `json:"sortOrder"`
}

// RequestItem 请求存储模型
type RequestItem struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Method    string      `json:"method"`
	URL       string      `json:"url"`
	Params    []KeyValue  `json:"params"`
	Headers   []KeyValue  `json:"headers"`
	Body      RequestBody `json:"body"`
	Auth      AuthConfig  `json:"auth"`
	FolderID  string      `json:"folderId,omitempty"`
	SortOrder int         `json:"sortOrder"`
	ProjectID string      `json:"projectId"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
}
