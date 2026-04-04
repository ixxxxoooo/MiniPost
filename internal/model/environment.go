package model

// Environment 环境变量集合
type Environment struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	ProjectID string     `json:"projectId"`
	Variables []Variable `json:"variables"`
}

// Variable 单个变量
type Variable struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	Enabled  bool   `json:"enabled"`
	IsSecret bool   `json:"isSecret"`
}
