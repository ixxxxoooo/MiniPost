package service

import (
	"net/http"
	"strings"
	"testing"

	"minipost/internal/model"
)

// TestBuildClient_ReusesTransport 验证相同选项下 transport 被缓存复用，
// 不同 sslVerify 选项则使用不同 transport。
func TestBuildClient_ReusesTransport(t *testing.T) {
	svc := NewHttpService()
	opts := svc.extractRequestOptions(nil)

	c1 := svc.buildClient(opts)
	c2 := svc.buildClient(opts)
	if c1.Transport == nil {
		t.Fatal("transport 不应为 nil")
	}
	if c1.Transport != c2.Transport {
		t.Fatal("相同选项应复用同一个 transport 实例")
	}

	insecureOpts := opts
	insecureOpts.sslVerify = false
	c3 := svc.buildClient(insecureOpts)
	if c3.Transport == c1.Transport {
		t.Fatal("不同 sslVerify 选项应使用不同的 transport 实例")
	}
}

// TestApplyAuth_BasicSkipsEmptyCredentials 验证 Basic 账密都为空时不附加 Authorization。
func TestApplyAuth_BasicSkipsEmptyCredentials(t *testing.T) {
	svc := NewHttpService()

	emptyReq, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("构建请求失败: %v", err)
	}
	svc.applyAuth(emptyReq, model.AuthConfig{Type: "basic"})
	if got := emptyReq.Header.Get("Authorization"); got != "" {
		t.Fatalf("空账密不应附加 Authorization, got %q", got)
	}

	authedReq, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("构建请求失败: %v", err)
	}
	svc.applyAuth(authedReq, model.AuthConfig{
		Type:  "basic",
		Basic: model.BasicAuth{Username: "user", Password: "pass"},
	})
	if got := authedReq.Header.Get("Authorization"); !strings.HasPrefix(got, "Basic ") {
		t.Fatalf("有账密时应附加 Basic 认证, got %q", got)
	}
}
