package service

import (
	"strings"

	"github.com/google/uuid"

	"minipost/internal/model"
	"minipost/internal/repository"
)

type EnvironmentService struct {
	store *repository.FileStore
}

func NewEnvironmentService(store *repository.FileStore) *EnvironmentService {
	return &EnvironmentService{store: store}
}

func (s *EnvironmentService) ListEnvironments(projectID string) ([]model.Environment, error) {
	return s.store.ListEnvironments(projectID)
}

func (s *EnvironmentService) CreateEnvironment(projectID, name string) (*model.Environment, error) {
	env := &model.Environment{
		ID:        uuid.New().String(),
		Name:      name,
		ProjectID: projectID,
		Variables: []model.Variable{},
	}
	if err := s.store.SaveEnvironment(env); err != nil {
		return nil, err
	}
	return env, nil
}

func (s *EnvironmentService) SaveEnvironment(env *model.Environment) error {
	return s.store.SaveEnvironment(env)
}

func (s *EnvironmentService) DeleteEnvironment(projectID, envID string) error {
	return s.store.DeleteEnvironment(projectID, envID)
}

// ResolveVariables 将 {{variableName}} 替换为对应环境变量的值
func (s *EnvironmentService) ResolveVariables(input string, variables []model.Variable) string {
	result := input
	for _, v := range variables {
		if v.Enabled && v.Key != "" {
			placeholder := "{{" + v.Key + "}}"
			result = strings.ReplaceAll(result, placeholder, v.Value)
		}
	}
	return result
}

// CreateVariable 创建新变量
func CreateVariable(key, value string) model.Variable {
	return model.Variable{
		ID:       uuid.New().String(),
		Key:      key,
		Value:    value,
		Enabled:  true,
		IsSecret: false,
	}
}
