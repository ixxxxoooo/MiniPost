package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"minipost/internal/model"
)

func newTestApp(t *testing.T) (*App, string) {
	t.Helper()

	t.Setenv("HOME", t.TempDir())
	app := NewApp()
	app.ctx = context.Background()

	projectID := "proj-app-import"
	now := time.Now().UTC().Format(time.RFC3339)
	if err := app.store.SaveProject(&model.Project{
		ID:        projectID,
		Name:      "App Import Test",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("SaveProject() error = %v", err)
	}

	return app, projectID
}

func TestImportFromFile_AutoDetectsPostmanEnvironment(t *testing.T) {
	app, projectID := newTestApp(t)

	raw := `{
	  "name": "Staging",
	  "values": [
	    { "key": "baseUrl", "value": "https://staging.example.com" }
	  ]
	}`

	if err := app.ImportFromFile(projectID, "auto", raw); err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}

	envs, err := app.store.ListEnvironments(projectID)
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v", err)
	}
	if len(envs) != 1 || envs[0].Name != "Staging" {
		t.Fatalf("unexpected imported environments: %+v", envs)
	}
}

func TestImportFromFile_AutoDetectsCommentedPostmanCollection(t *testing.T) {
	app, projectID := newTestApp(t)

	raw := `{
	  // jsonc comment
	  "info": { "name": "Commented Postman" },
	  "item": [
	    {
	      "name": "Ping",
	      "request": {
	        "method": "GET",
	        "url": "https://api.example.com/ping"
	      }
	    }
	  ]
	}`

	if err := app.ImportFromFile(projectID, "auto", raw); err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}

	requests, err := app.store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 1 || requests[0].Name != "Ping" {
		t.Fatalf("unexpected imported requests: %+v", requests)
	}
}

func TestImportFromURL_ResolvesSwaggerUIHTMLToSpec(t *testing.T) {
	app, projectID := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/swagger/index.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><script>SwaggerUIBundle({ url: "doc.json", dom_id: '#swagger-ui' })</script></body></html>`))
		case "/swagger/doc.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
			  "openapi": "3.0.3",
			  "info": { "title": "Remote API" },
			  "servers": [
			    { "url": "https://api.example.com/{stage}", "variables": { "stage": { "default": "prod" } } }
			  ],
			  "paths": {
			    "/health": {
			      "get": { "summary": "Health check" }
			    }
			  }
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if err := app.ImportFromURL(projectID, "auto", server.URL+"/swagger/index.html"); err != nil {
		t.Fatalf("ImportFromURL() error = %v", err)
	}

	requests, err := app.store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 1 || requests[0].URL != "{{baseUrl}}/health" {
		t.Fatalf("unexpected imported requests: %+v", requests)
	}

	envs, err := app.store.ListEnvironments(projectID)
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(envs))
	}
	foundBaseURL := false
	for _, variable := range envs[0].Variables {
		if variable.Key == "baseUrl" && variable.Value == "https://api.example.com/{{stage}}" {
			foundBaseURL = true
		}
	}
	if !foundBaseURL {
		t.Fatalf("expected baseUrl variable to be imported, got %+v", envs[0].Variables)
	}
}

func TestImportFromURL_InfersBaseURLWhenSwagger2HostMissing(t *testing.T) {
	app, projectID := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bw-go/swagger/doc.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
			  "swagger": "2.0",
			  "info": { "title": "BW GO" },
			  "host": "",
			  "basePath": "",
			  "paths": {
			    "/acceleration/start": {
			      "post": { "summary": "启动加速任务" }
			    }
			  }
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if err := app.ImportFromURL(projectID, "auto", server.URL+"/bw-go/swagger/doc.json"); err != nil {
		t.Fatalf("ImportFromURL() error = %v", err)
	}

	requests, err := app.store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 1 || requests[0].URL != "{{baseUrl}}/acceleration/start" {
		t.Fatalf("unexpected imported requests: %+v", requests)
	}

	envs, err := app.store.ListEnvironments(projectID)
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(envs))
	}
	expectedBaseURL := server.URL + "/bw-go"
	for _, variable := range envs[0].Variables {
		if variable.Key == "baseUrl" && variable.Value == expectedBaseURL {
			return
		}
	}
	t.Fatalf("expected inferred baseUrl %s, got %+v", expectedBaseURL, envs[0].Variables)
}
