package service

import (
	"github.com/google/uuid"

	"minipost/internal/model"
	"minipost/internal/repository"
)

type HistoryService struct {
	store *repository.FileStore
}

func NewHistoryService(store *repository.FileStore) *HistoryService {
	return &HistoryService{store: store}
}

func (s *HistoryService) GetHistory(projectID string) ([]model.HistoryEntry, error) {
	return s.store.GetHistory(projectID)
}

func (s *HistoryService) AddEntry(projectID string, entry *model.HistoryEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	return s.store.AddHistory(projectID, entry)
}

func (s *HistoryService) ClearHistory(projectID string) error {
	return s.store.ClearHistory(projectID)
}
