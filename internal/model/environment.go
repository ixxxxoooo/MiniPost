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
	// InitialValue 是初始值：当前值（Value）为空时发送请求使用它。
	InitialValue string `json:"initialValue,omitempty"`
	// Description 是变量的说明文字，只用于界面展示。
	Description string `json:"description,omitempty"`
}

// Effective 返回发送请求时实际使用的值：设置了当前值用当前值，否则用初始值。
func (v Variable) Effective() string {
	if v.Value != "" {
		return v.Value
	}
	return v.InitialValue
}
