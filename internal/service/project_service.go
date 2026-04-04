package service

import (
	"time"

	"github.com/google/uuid"

	"minipost/internal/model"
	"minipost/internal/repository"
)

type ProjectService struct {
	store *repository.FileStore
}

func NewProjectService(store *repository.FileStore) *ProjectService {
	return &ProjectService{store: store}
}

func (s *ProjectService) ListProjects() ([]model.Project, error) {
	return s.store.ListProjects()
}

func (s *ProjectService) CreateProject(name string) (*model.Project, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	project := &model.Project{
		ID:            uuid.New().String(),
		Name:          name,
		CreatedAt:     now,
		UpdatedAt:     now,
		SchemaVersion: 1,
	}
	if err := s.store.SaveProject(project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) RenameProject(id, name string) (*model.Project, error) {
	project, err := s.store.GetProject(id)
	if err != nil {
		return nil, err
	}
	project.Name = name
	project.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.SaveProject(project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) DeleteProject(id string) error {
	return s.store.DeleteProject(id)
}
