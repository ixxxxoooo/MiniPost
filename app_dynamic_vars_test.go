package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"minipost/internal/model"
)

// 没有选择环境时，请求里的内置动态变量也要在发送前被求值。
func TestSendRequestWithEnv_ResolvesDynamicVarsWithoutEnvironment(t *testing.T) {
	app, projectID := newTestApp(t)

	var gotPath, gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		gotHeader = r.Header.Get("X-Request-Id")
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	_, err := app.SendRequestWithEnv(model.SendRequestInput{
		Method:  "GET",
		URL:     server.URL + "/t/{{$timestamp}}?n={{$randomInt}}",
		Headers: []model.KeyValue{{Key: "X-Request-Id", Value: "{{$randomUUID}}"}},
	}, projectID, "")
	if err != nil {
		t.Fatalf("SendRequestWithEnv() error = %v", err)
	}
	if !regexp.MustCompile(`^/t/\d{10}\?n=\d+$`).MatchString(gotPath) {
		t.Fatalf("路径里的动态变量未被求值: %q", gotPath)
	}
	if !regexp.MustCompile(`^[0-9a-f-]{36}$`).MatchString(gotHeader) {
		t.Fatalf("请求头里的动态变量未被求值: %q", gotHeader)
	}
}
