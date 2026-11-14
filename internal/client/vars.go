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

// TemplateRef 是文本里的一处 {{name}} 引用，Start/End 为字节偏移（含花括号）。
type TemplateRef struct {
	Start, End int
	Name       string // 去掉首尾空白后的变量名
	Defined    bool   // 当前环境中是否定义了该变量
}

// FindTemplateRefs 按出现顺序返回 input 中所有 {{name}} 引用，并标注是否已在 vars 中定义。
// 变量名为空白的 {{ }} 不算引用。
func FindTemplateRefs(input string, vars []Var) []TemplateRef {
	if !strings.Contains(input, "{{") {
		return nil
	}
	known := make(map[string]bool, len(vars))
	for _, v := range vars {
		if k := strings.TrimSpace(v.Key); k != "" {
			known[k] = true
		}
	}
	var out []TemplateRef
	for _, m := range templatePattern.FindAllStringSubmatchIndex(input, -1) {
		name := strings.TrimSpace(input[m[2]:m[3]])
		if name == "" {
			continue
		}
		out = append(out, TemplateRef{Start: m[0], End: m[1], Name: name, Defined: known[name]})
	}
	return out
}

// UndefinedVarNames 汇总 inputs 中引用了、但 vars 里没有定义的变量名（去重，保持首次出现顺序）。
func UndefinedVarNames(vars []Var, inputs ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, in := range inputs {
		for _, r := range FindTemplateRefs(in, vars) {
			if !r.Defined && !seen[r.Name] {
				seen[r.Name] = true
				out = append(out, r.Name)
			}
		}
	}
	return out
}
