package app

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
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

// 发送时带上集合请求 ID（SourceID）后，历史记录要存成 RequestID，点击历史才能打开原请求。
func TestSendRequestWithEnv_HistoryKeepsSourceRequestID(t *testing.T) {
	app, projectID := newTestApp(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	for _, in := range []model.SendRequestInput{
		{Method: "GET", URL: server.URL + "/a", SourceID: "req-1", RequestID: "send-uuid"},
		{Method: "GET", URL: server.URL + "/b"},
	} {
		if _, err := app.SendRequestWithEnv(in, projectID, ""); err != nil {
			t.Fatalf("SendRequestWithEnv() error = %v", err)
		}
	}
	entries, err := app.historySvc.GetHistory(projectID)
	if err != nil || len(entries) != 2 {
		t.Fatalf("GetHistory() = %v, %v", entries, err)
	}
	got := map[string]string{}
	for _, e := range entries {
		got[strings.TrimPrefix(e.URL, server.URL)] = e.RequestID
	}
	if got["/a"] != "req-1" || got["/b"] != "" {
		t.Fatalf("历史 RequestID 不对: %v", got)
	}
}
