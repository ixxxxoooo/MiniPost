package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"minipost/internal/model"
)

func keyLogRequest(url string, keyLog bool) model.SendRequestInput {
	return model.SendRequestInput{
		Method: "GET",
		URL:    url,
		Body:   model.RequestBody{Type: "none"},
		Auth:   model.AuthConfig{Type: "none"},
		Options: &model.RequestOptions{
			FollowRedirects: true, SSLVerify: false, SSLTLSKeyLog: keyLog, HTTPVersion: "auto",
		},
	}
}

// 开启「SSL/TLS Key Log」后，握手密钥应追加写入密钥日志文件。
func TestSendRequest_WritesTLSKeyLogWhenEnabled(t *testing.T) {
	t.Setenv("SSLKEYLOGFILE", "")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "logs", "sslkeys.log")
	svc := NewHttpService()
	svc.SetKeyLogPath(path)
	resp, err := svc.SendRequest(keyLogRequest(srv.URL, true))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("请求失败: %v %+v", err, resp)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("应生成密钥日志文件: %v", err)
	}
	if !strings.Contains(string(data), "TRAFFIC_SECRET") && !strings.Contains(string(data), "CLIENT_RANDOM") {
		t.Fatalf("密钥日志内容不符合 NSS 格式: %q", string(data))
	}
}

// 未开启时不应创建密钥日志，也不应影响请求。
func TestSendRequest_NoTLSKeyLogWhenDisabled(t *testing.T) {
	t.Setenv("SSLKEYLOGFILE", "")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "sslkeys.log")
	svc := NewHttpService()
	svc.SetKeyLogPath(path)
	if _, err := svc.SendRequest(keyLogRequest(srv.URL, false)); err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("未开启时不应创建密钥日志，stat err=%v", err)
	}
}
