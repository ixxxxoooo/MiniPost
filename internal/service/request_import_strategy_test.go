package service

import (
	"testing"

	"minipost/internal/model"
)

const strategyTestCollection = `{
  "info": { "name": "Strategy Collection" },
  "item": [
    {
      "name": "租户列表",
      "request": {
        "method": "POST",
        "url": "{{baseUrl}}/tenant/list",
        "header": [
          { "key": "X-Trace", "value": "", "description": "trace header" }
        ],
        "body": {
          "mode": "raw",
          "raw": "{\n  \"tenantId\": 0\n}",
          "options": { "raw": { "language": "json" } }
        }
      }
    },
    {
      "name": "新接口",
      "request": {
        "method": "GET",
        "url": "{{baseUrl}}/tenant/new",
        "header": []
      }
    }
  ]
}`

// seedExistingTenantRequest 预置一条与导入文档冲突且带“用户修改”的请求
func seedExistingTenantRequest(t *testing.T, service *RequestService, projectID string) *model.RequestItem {
	t.Helper()

	req, err := service.CreateRequest(projectID, "", "我的租户列表")
	if err != nil {
		t.Fatalf("CreateRequest() error = %v", err)
	}
	req.Method = "POST"
	req.URL = "{{baseUrl}}/tenant/list?debug=1"
	req.Headers = []model.KeyValue{{Key: "X-Trace", Value: "user-trace"}}
	req.Body = model.RequestBody{Type: "json", JSON: `{"tenantId": 42}`}
	req.Auth = model.AuthConfig{Type: "bearer", Bearer: model.BearerAuth{Token: "user-token"}}
	if err := service.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}
	return req
}

func TestImportPostmanWithStrategy_UpdatePreservesUserChanges(t *testing.T) {
	service, store, projectID := newImportTestService(t)
	existing := seedExistingTenantRequest(t, service, projectID)

	summary, err := service.ImportPostmanCollectionWithStrategy(projectID, []byte(strategyTestCollection), ImportStrategyUpdate)
	if err != nil {
		t.Fatalf("ImportPostmanCollectionWithStrategy() error = %v", err)
	}
	if summary.Updated != 1 || summary.Created != 1 || summary.Overwritten != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 requests after update import, got %d", len(requests))
	}

	var updated *model.RequestItem
	for i := range requests {
		if requests[i].ID == existing.ID {
			updated = &requests[i]
		}
	}
	if updated == nil {
		t.Fatalf("existing request should be kept, got %+v", requests)
	}
	if updated.Name != "我的租户列表" {
		t.Fatalf("user request name should be preserved, got %s", updated.Name)
	}
	if updated.URL != "{{baseUrl}}/tenant/list?debug=1" {
		t.Fatalf("user url query should be preserved, got %s", updated.URL)
	}
	if len(updated.Headers) != 1 || updated.Headers[0].Value != "user-trace" {
		t.Fatalf("user header value should be preserved, got %+v", updated.Headers)
	}
	if updated.Headers[0].Description != "trace header" {
		t.Fatalf("header description should refresh from import, got %+v", updated.Headers[0])
	}
	if updated.Body.JSON != `{"tenantId": 42}` {
		t.Fatalf("user body should be preserved, got %q", updated.Body.JSON)
	}
	if updated.Auth.Type != "bearer" || updated.Auth.Bearer.Token != "user-token" {
		t.Fatalf("user auth should be preserved, got %+v", updated.Auth)
	}
}

func TestImportPostmanWithStrategy_UpdateFillsEmptyBody(t *testing.T) {
	service, store, projectID := newImportTestService(t)

	req, err := service.CreateRequest(projectID, "", "空 body 请求")
	if err != nil {
		t.Fatalf("CreateRequest() error = %v", err)
	}
	req.Method = "POST"
	req.URL = "{{baseUrl}}/tenant/list"
	if err := service.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}

	if _, err := service.ImportPostmanCollectionWithStrategy(projectID, []byte(strategyTestCollection), ImportStrategyUpdate); err != nil {
		t.Fatalf("ImportPostmanCollectionWithStrategy() error = %v", err)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	for _, item := range requests {
		if item.ID == req.ID {
			if item.Body.Type != "json" || item.Body.JSON == "" {
				t.Fatalf("empty body should be filled from import, got %+v", item.Body)
			}
			return
		}
	}
	t.Fatalf("existing request not found after import")
}

func TestImportPostmanWithStrategy_OverwriteReplacesRequest(t *testing.T) {
	service, store, projectID := newImportTestService(t)
	existing := seedExistingTenantRequest(t, service, projectID)

	summary, err := service.ImportPostmanCollectionWithStrategy(projectID, []byte(strategyTestCollection), ImportStrategyOverwrite)
	if err != nil {
		t.Fatalf("ImportPostmanCollectionWithStrategy() error = %v", err)
	}
	if summary.Overwritten != 1 || summary.Created != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 requests after overwrite import, got %d", len(requests))
	}
	for _, item := range requests {
		if item.ID != existing.ID {
			continue
		}
		if item.Name != "租户列表" {
			t.Fatalf("overwrite should replace name, got %s", item.Name)
		}
		if item.URL != "{{baseUrl}}/tenant/list" {
			t.Fatalf("overwrite should replace url, got %s", item.URL)
		}
		if item.Headers[0].Value != "" {
			t.Fatalf("overwrite should replace header value, got %+v", item.Headers)
		}
		if item.Body.JSON != "{\n  \"tenantId\": 0\n}" {
			t.Fatalf("overwrite should replace body, got %q", item.Body.JSON)
		}
		if item.Auth.Type != "none" {
			t.Fatalf("overwrite should replace auth, got %+v", item.Auth)
		}
		return
	}
	t.Fatalf("existing request not found after overwrite")
}

func TestImportPostmanWithStrategy_CopyCreatesDuplicate(t *testing.T) {
	service, store, projectID := newImportTestService(t)
	seedExistingTenantRequest(t, service, projectID)

	summary, err := service.ImportPostmanCollectionWithStrategy(projectID, []byte(strategyTestCollection), ImportStrategyCopy)
	if err != nil {
		t.Fatalf("ImportPostmanCollectionWithStrategy() error = %v", err)
	}
	if summary.Created != 2 {
		t.Fatalf("copy strategy should create all requests, got %+v", summary)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 3 {
		t.Fatalf("expected 3 requests after copy import, got %d", len(requests))
	}
}

func TestPreviewPostmanConflicts(t *testing.T) {
	service, _, projectID := newImportTestService(t)
	seedExistingTenantRequest(t, service, projectID)

	conflicts, err := service.PreviewPostmanConflicts(projectID, []byte(strategyTestCollection))
	if err != nil {
		t.Fatalf("PreviewPostmanConflicts() error = %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %+v", conflicts)
	}
	if conflicts[0].Method != "POST" || conflicts[0].URL != "{{baseUrl}}/tenant/list" {
		t.Fatalf("unexpected conflict: %+v", conflicts[0])
	}
}

const strategyTestSwagger = `{
  "openapi": "3.0.0",
  "info": { "title": "Strategy API" },
  "servers": [{ "url": "https://api.example.com" }],
  "paths": {
    "/tenant/list": {
      "post": {
        "tags": ["租户"],
        "summary": "租户列表",
        "requestBody": {
          "content": {
            "application/json": {
              "schema": { "type": "object", "properties": { "tenantId": { "type": "integer" } } }
            }
          }
        }
      }
    }
  }
}`

func TestImportSwaggerWithStrategy_UpdateReusesFoldersAndEnvironments(t *testing.T) {
	service, store, projectID := newImportTestService(t)

	if _, err := service.ImportSwaggerWithSourceStrategy(projectID, []byte(strategyTestSwagger), "", ImportStrategyCopy); err != nil {
		t.Fatalf("first import error = %v", err)
	}
	if _, err := service.ImportSwaggerWithSourceStrategy(projectID, []byte(strategyTestSwagger), "", ImportStrategyUpdate); err != nil {
		t.Fatalf("second import error = %v", err)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("update strategy should not duplicate requests, got %d", len(requests))
	}

	folders, err := store.ListFolders(projectID)
	if err != nil {
		t.Fatalf("ListFolders() error = %v", err)
	}
	if len(folders) != 1 {
		t.Fatalf("update strategy should reuse tag folder, got %d", len(folders))
	}

	envs, err := store.ListEnvironments(projectID)
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("update strategy should reuse environment, got %d", len(envs))
	}
}

func TestPreviewSwaggerConflicts(t *testing.T) {
	service, _, projectID := newImportTestService(t)

	if _, err := service.ImportSwaggerWithSourceStrategy(projectID, []byte(strategyTestSwagger), "", ImportStrategyCopy); err != nil {
		t.Fatalf("import error = %v", err)
	}

	conflicts, err := service.PreviewSwaggerConflicts(projectID, []byte(strategyTestSwagger), "")
	if err != nil {
		t.Fatalf("PreviewSwaggerConflicts() error = %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %+v", conflicts)
	}
	if conflicts[0].URL != "{{baseUrl}}/tenant/list" {
		t.Fatalf("unexpected conflict url: %s", conflicts[0].URL)
	}
}
