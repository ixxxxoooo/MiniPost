package service

import (
	"time"

	"github.com/google/uuid"

	"minipost/internal/model"
	"minipost/internal/repository"
)

type RequestService struct {
	store *repository.FileStore
}

func NewRequestService(store *repository.FileStore) *RequestService {
	return &RequestService{store: store}
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

// ListFolders 列出项目下所有文件夹
func (s *RequestService) ListFolders(projectID string) ([]model.Folder, error) {
	return s.store.ListFolders(projectID)
}

// CreateFolder 创建文件夹
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

// RenameFolder 重命名文件夹
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

// DeleteFolder 删除文件夹
func (s *RequestService) DeleteFolder(projectID, folderID string) error {
	return s.store.DeleteFolder(projectID, folderID)
}
