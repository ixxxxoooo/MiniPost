package httputil

import (
	"strings"

	"minipost/internal/model"
)

// ResolveVariables 在字符串中替换 {{variableName}} 为对应变量值
func ResolveVariables(input string, variables []model.Variable) string {
	result := input
	for _, v := range variables {
		if v.Enabled && v.Key != "" {
			placeholder := "{{" + v.Key + "}}"
			result = strings.ReplaceAll(result, placeholder, v.Value)
		}
	}
	return result
}

// ResolveKeyValues 解析 KeyValue 列表中的变量
func ResolveKeyValues(kvs []model.KeyValue, variables []model.Variable) []model.KeyValue {
	resolved := make([]model.KeyValue, len(kvs))
	for i, kv := range kvs {
		resolved[i] = model.KeyValue{
			Key:   ResolveVariables(kv.Key, variables),
			Value: ResolveVariables(kv.Value, variables),
		}
	}
	return resolved
}

// ResolveRequestInput 对整个请求输入进行变量解析
func ResolveRequestInput(input model.SendRequestInput, variables []model.Variable) model.SendRequestInput {
	resolved := input
	resolved.URL = ResolveVariables(input.URL, variables)
	resolved.Params = ResolveKeyValues(input.Params, variables)
	resolved.Headers = ResolveKeyValues(input.Headers, variables)

	resolved.Body.Raw = ResolveVariables(input.Body.Raw, variables)
	resolved.Body.JSON = ResolveVariables(input.Body.JSON, variables)
	resolved.Body.FormUrlEncoded = ResolveKeyValues(input.Body.FormUrlEncoded, variables)

	resolved.Auth.Basic.Username = ResolveVariables(input.Auth.Basic.Username, variables)
	resolved.Auth.Basic.Password = ResolveVariables(input.Auth.Basic.Password, variables)
	resolved.Auth.Bearer.Token = ResolveVariables(input.Auth.Bearer.Token, variables)
	resolved.Auth.APIKey.Key = ResolveVariables(input.Auth.APIKey.Key, variables)
	resolved.Auth.APIKey.Value = ResolveVariables(input.Auth.APIKey.Value, variables)

	return resolved
}
