package client

import (
	"regexp"
	"strings"
)

// Var 环境变量键值，仅包含已启用的变量。
type Var struct {
	Key   string
	Value string
}

var (
	templatePattern = regexp.MustCompile(`\{\{([^{}]+)\}\}`)
	schemePattern   = regexp.MustCompile(`^[A-Za-z][A-Za-z\d+\-.]*://`)
)

// ResolveTemplate 把 {{name}} 替换为变量值；未定义的变量原样保留。
func ResolveTemplate(input string, vars []Var) string {
	if input == "" || len(vars) == 0 {
		return input
	}
	m := make(map[string]string, len(vars))
	for _, v := range vars {
		if k := strings.TrimSpace(v.Key); k != "" {
			m[k] = v.Value
		}
	}
	return templatePattern.ReplaceAllStringFunc(input, func(token string) string {
		key := strings.TrimSpace(templatePattern.FindStringSubmatch(token)[1])
		if key == "" {
			return token
		}
		if val, ok := m[key]; ok {
			return val
		}
		return token
	})
}

// EnsureProtocol 为缺少协议的地址补 http://；含未解析变量的地址保持不变。
func EnsureProtocol(input string) string {
	value := strings.TrimSpace(input)
	switch {
	case value == "", strings.Contains(value, "{{"), schemePattern.MatchString(value):
		return input
	case strings.HasPrefix(value, "//"):
		return "http:" + value
	}
	return "http://" + value
}
