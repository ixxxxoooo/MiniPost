package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"minipost/internal/model"
	"minipost/internal/repository"
)

type ProjectService struct {
	store *repository.FileStore
}

var projectThemePalette = []string{
	"#0A84FF",
	"#30D158",
	"#FF9F0A",
	"#FF453A",
	"#BF5AF2",
	"#64D2FF",
	"#5E5CE6",
	"#FF375F",
	"#34C759",
	"#FFD60A",
	"#FF6B35",
	"#00C7BE",
}

func NewProjectService(store *repository.FileStore) *ProjectService {
	return &ProjectService{store: store}
}

func (s *ProjectService) ListProjects() ([]model.Project, error) {
	projects, err := s.store.ListProjects()
	if err != nil {
		return nil, err
	}

	usedColors := make(map[string]struct{}, len(projects))
	for i := range projects {
		normalized := normalizeProjectThemeColor(projects[i].ThemeColor)
		if normalized == "" {
			continue
		}
		projects[i].ThemeColor = normalized
		usedColors[normalized] = struct{}{}
	}

	for i := range projects {
		if projects[i].ThemeColor != "" {
			continue
		}
		projects[i].ThemeColor = pickProjectThemeColor(usedColors)
		usedColors[projects[i].ThemeColor] = struct{}{}
		project := projects[i]
		if err := s.store.SaveProject(&project); err != nil {
			return nil, err
		}
	}

	return projects, nil
}

func (s *ProjectService) CreateProject(name string) (*model.Project, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	projects, err := s.store.ListProjects()
	if err != nil {
		return nil, err
	}
	usedColors := make(map[string]struct{}, len(projects))
	for _, project := range projects {
		normalized := normalizeProjectThemeColor(project.ThemeColor)
		if normalized == "" {
			continue
		}
		usedColors[normalized] = struct{}{}
	}

	project := &model.Project{
		ID:            uuid.New().String(),
		Name:          name,
		ThemeColor:    pickProjectThemeColor(usedColors),
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

func (s *ProjectService) UpdateProjectTheme(id, color string) (*model.Project, error) {
	project, err := s.store.GetProject(id)
	if err != nil {
		return nil, err
	}

	normalized := normalizeProjectThemeColor(color)
	if normalized == "" {
		return nil, fmt.Errorf("无效主题色: %s", color)
	}

	project.ThemeColor = normalized
	project.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.SaveProject(project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) UpdateProjectDescription(id, description string) (*model.Project, error) {
	project, err := s.store.GetProject(id)
	if err != nil {
		return nil, err
	}

	project.Description = strings.TrimSpace(description)
	project.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.SaveProject(project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) DeleteProject(id string) error {
	return s.store.DeleteProject(id)
}

func normalizeProjectThemeColor(color string) string {
	candidate := strings.TrimSpace(color)
	if candidate == "" {
		return ""
	}

	if !strings.HasPrefix(candidate, "#") {
		candidate = "#" + candidate
	}

	if len(candidate) != 7 {
		return ""
	}

	for _, ch := range candidate[1:] {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			continue
		}
		return ""
	}

	return strings.ToUpper(candidate)
}

func pickProjectThemeColor(usedColors map[string]struct{}) string {
	for _, color := range projectThemePalette {
		normalized := normalizeProjectThemeColor(color)
		if _, used := usedColors[normalized]; used {
			continue
		}
		return normalized
	}

	idx := int(time.Now().UnixNano() % int64(len(projectThemePalette)))
	return projectThemePalette[idx]
}
