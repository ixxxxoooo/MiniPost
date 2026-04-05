package service

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"minipost/internal/model"
	appErrors "minipost/internal/pkg/errors"
)

func newCurlArenaServer() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	handler.HandleFunc("/status/301", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusMovedPermanently)
	})
	handler.HandleFunc("/status/307", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/echo")
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	handler.HandleFunc("/status/500", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "server error")
	})
	handler.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"method":"`+r.Method+`","body":"`+string(body)+`"}`)
	})
	return httptest.NewServer(handler)
}

func TestSendRequest_FollowsRedirect301(t *testing.T) {
	arena := newCurlArenaServer()
	defer arena.Close()

	svc := NewHttpService()
	resp, err := svc.SendRequest(model.SendRequestInput{
		Method: "GET",
		URL:    arena.URL + "/status/301",
		Body:   model.RequestBody{Type: "none"},
		Auth:   model.AuthConfig{Type: "none"},
	})
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Body != `{"ok":true}` {
		t.Fatalf("expected final body, got %q", resp.Body)
	}
}

func TestSendRequest_Follows307AndKeepsMethodAndBody(t *testing.T) {
	arena := newCurlArenaServer()
	defer arena.Close()

	svc := NewHttpService()
	resp, err := svc.SendRequest(model.SendRequestInput{
		Method: "POST",
		URL:    arena.URL + "/status/307",
		Body:   model.RequestBody{Type: "raw", Raw: "hello"},
		Auth:   model.AuthConfig{Type: "none"},
	})
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Body != `{"method":"POST","body":"hello"}` {
		t.Fatalf("expected POST echo response, got %q", resp.Body)
	}
}

func TestSendRequest_CurlCommandInURLField(t *testing.T) {
	arena := newCurlArenaServer()
	defer arena.Close()

	curlCmd := "curl --request POST --url " + arena.URL + "/echo --header 'Content-Type: text/plain' --data 'mini-post'"
	svc := NewHttpService()
	resp, err := svc.SendRequest(model.SendRequestInput{
		Method: "GET",
		URL:    curlCmd,
		Body:   model.RequestBody{Type: "none"},
		Auth:   model.AuthConfig{Type: "none"},
	})
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Body != `{"method":"POST","body":"mini-post"}` {
		t.Fatalf("expected parsed cURL request to be sent, got %q", resp.Body)
	}
}

func TestSendRequest_CurlStatus301Command(t *testing.T) {
	arena := newCurlArenaServer()
	defer arena.Close()

	svc := NewHttpService()
	resp, err := svc.SendRequest(model.SendRequestInput{
		Method: "GET",
		URL:    "curl " + arena.URL + "/status/301",
		Body:   model.RequestBody{Type: "none"},
		Auth:   model.AuthConfig{Type: "none"},
	})
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Body != `{"ok":true}` {
		t.Fatalf("expected final body after redirect, got %q", resp.Body)
	}
}

func TestSendRequest_InvalidCurlCommand(t *testing.T) {
	svc := NewHttpService()
	_, err := svc.SendRequest(model.SendRequestInput{
		Method: "GET",
		URL:    "curl -X POST",
		Body:   model.RequestBody{Type: "none"},
		Auth:   model.AuthConfig{Type: "none"},
	})
	if err == nil {
		t.Fatal("expected error for invalid curl command")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "INVALID_CURL" {
		t.Fatalf("expected INVALID_CURL, got %s", appErr.Code)
	}
}

func TestSendRequest_5xxResponseStillReturnsResponse(t *testing.T) {
	arena := newCurlArenaServer()
	defer arena.Close()

	svc := NewHttpService()
	resp, err := svc.SendRequest(model.SendRequestInput{
		Method: "GET",
		URL:    arena.URL + "/status/500",
		Body:   model.RequestBody{Type: "none"},
		Auth:   model.AuthConfig{Type: "none"},
	})
	if err != nil {
		t.Fatalf("expected no transport error, got %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}
	if resp.Body != "server error" {
		t.Fatalf("expected response body, got %q", resp.Body)
	}
}

func TestSendRequest_DisableFollowRedirectsByOption(t *testing.T) {
	arena := newCurlArenaServer()
	defer arena.Close()

	svc := NewHttpService()
	resp, err := svc.SendRequest(model.SendRequestInput{
		Method: "GET",
		URL:    arena.URL + "/status/301",
		Headers: []model.KeyValue{
			{Key: "X-MiniPost-Option-Follow-Redirects", Value: "0"},
		},
		Body: model.RequestBody{Type: "none"},
		Auth: model.AuthConfig{Type: "none"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("expected status 301 when redirect disabled, got %d", resp.StatusCode)
	}
}

func TestSendRequest_MaxResponseSizeOption(t *testing.T) {
	arena := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, strings.Repeat("a", 64))
	}))
	defer arena.Close()

	svc := NewHttpService()
	_, err := svc.SendRequest(model.SendRequestInput{
		Method: "GET",
		URL:    arena.URL,
		Headers: []model.KeyValue{
			{Key: "X-MiniPost-Option-Max-Response-Size-MB", Value: "0"},
		},
		Body: model.RequestBody{Type: "none"},
		Auth: model.AuthConfig{Type: "none"},
	})
	if err != nil {
		t.Fatalf("expected no error for unlimited body size, got %v", err)
	}

	_, err = svc.SendRequest(model.SendRequestInput{
		Method: "GET",
		URL:    arena.URL,
		Headers: []model.KeyValue{
			{Key: "X-MiniPost-Option-Max-Response-Size-MB", Value: "1"},
		},
		Body: model.RequestBody{Type: "none"},
		Auth: model.AuthConfig{Type: "none"},
	})
	if err != nil {
		// 1MB 限制下 64B 不应触发错误
		t.Fatalf("expected no error with 1MB limit, got %v", err)
	}

	largeArena := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, strings.Repeat("b", 2*1024*1024))
	}))
	defer largeArena.Close()

	_, err = svc.SendRequest(model.SendRequestInput{
		Method: "GET",
		URL:    largeArena.URL,
		Headers: []model.KeyValue{
			{Key: "X-MiniPost-Option-Max-Response-Size-MB", Value: "1"},
		},
		Body: model.RequestBody{Type: "none"},
		Auth: model.AuthConfig{Type: "none"},
	})
	if err == nil {
		t.Fatal("expected max response size exceeded error")
	}
}
