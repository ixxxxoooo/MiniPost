package httputil

import (
	"testing"

	"minipost/internal/model"
)

func TestResolveVariables_ReplacesEnabledVariables(t *testing.T) {
	variables := []model.Variable{
		{Key: "baseUrl", Value: "https://api.example.com", Enabled: true},
		{Key: "token", Value: "secret", Enabled: false},
	}
	got := ResolveVariables("{{baseUrl}}/users/{{token}}", variables)
	want := "https://api.example.com/users/{{token}}"
	if got != want {
		t.Fatalf("解析结果不符合预期: got %q want %q", got, want)
	}
}

func TestResolveKeyValues_PreservesDescription(t *testing.T) {
	input := []model.KeyValue{
		{Key: "{{host}}", Value: "{{path}}", Description: "示例参数"},
	}
	variables := []model.Variable{
		{Key: "host", Value: "api.example.com", Enabled: true},
		{Key: "path", Value: "/v1", Enabled: true},
	}
	got := ResolveKeyValues(input, variables)
	if len(got) != 1 {
		t.Fatalf("期望 1 条记录，实际 %d", len(got))
	}
	if got[0].Key != "api.example.com" || got[0].Value != "/v1" {
		t.Fatalf("键值解析错误: %#v", got[0])
	}
	if got[0].Description != "示例参数" {
		t.Fatalf("Description 应保留，实际 %q", got[0].Description)
	}
}

func TestResolveRequestInput_ResolvesAuthAndBody(t *testing.T) {
	input := model.SendRequestInput{
		Method: "POST",
		URL:    "{{baseUrl}}/login",
		Headers: []model.KeyValue{
			{Key: "Authorization", Value: "Bearer {{token}}"},
		},
		Body: model.RequestBody{
			Type: "json",
			JSON: `{"name":"{{user}}"}`,
		},
		Auth: model.AuthConfig{
			Type: "basic",
			Basic: model.BasicAuth{
				Username: "{{user}}",
				Password: "{{password}}",
			},
		},
	}
	variables := []model.Variable{
		{Key: "baseUrl", Value: "https://api.example.com", Enabled: true},
		{Key: "token", Value: "abc", Enabled: true},
		{Key: "user", Value: "demo", Enabled: true},
		{Key: "password", Value: "pwd", Enabled: true},
	}

	got := ResolveRequestInput(input, variables)
	if got.URL != "https://api.example.com/login" {
		t.Fatalf("URL 解析错误: %q", got.URL)
	}
	if got.Headers[0].Value != "Bearer abc" {
		t.Fatalf("Header 解析错误: %q", got.Headers[0].Value)
	}
	if got.Body.JSON != `{"name":"demo"}` {
		t.Fatalf("Body 解析错误: %q", got.Body.JSON)
	}
	if got.Auth.Basic.Username != "demo" || got.Auth.Basic.Password != "pwd" {
		t.Fatalf("Auth 解析错误: %#v", got.Auth.Basic)
	}
}
