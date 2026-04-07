package service

import (
	"strings"
	"testing"
	"time"

	"minipost/internal/model"
	"minipost/internal/repository"
)

func newImportTestService(t *testing.T) (*RequestService, *repository.FileStore, string) {
	t.Helper()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	store, err := repository.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	projectID := "proj-import"
	now := time.Now().UTC().Format(time.RFC3339)
	if err := store.SaveProject(&model.Project{
		ID:        projectID,
		Name:      "Import Test",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("SaveProject() error = %v", err)
	}

	return NewRequestService(store), store, projectID
}

func findVariable(variables []model.Variable, key string) *model.Variable {
	for i := range variables {
		if variables[i].Key == key {
			return &variables[i]
		}
	}
	return nil
}

func TestImportPostmanCollection_ImportsRequestsAndVariables(t *testing.T) {
	service, store, projectID := newImportTestService(t)

	raw := []byte(`{
	  "info": { "name": "Demo Collection" },
	  "variable": [
	    { "key": "baseUrl", "value": "https://api.example.com" },
	    { "key": "token", "value": "secret-token", "type": "secret" }
	  ],
	  "item": [
	    {
	      "name": "List Users",
	      "request": {
	        "method": "GET",
	        "url": "{{baseUrl}}/users",
	        "body": {
	          "mode": "raw",
	          "raw": "{\n  // keep comment\n  \"enabled\": true\n}",
	          "options": { "raw": { "language": "json" } }
	        },
	        "header": [
	          { "key": "Authorization", "value": "Bearer {{token}}", "description": "token header" }
	        ]
	      }
	    }
	  ]
	}`)

	if err := service.ImportPostmanCollection(projectID, raw); err != nil {
		t.Fatalf("ImportPostmanCollection() error = %v", err)
	}

	envs, err := store.ListEnvironments(projectID)
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(envs))
	}
	if envs[0].Name != "Demo Collection" {
		t.Fatalf("unexpected environment name: %s", envs[0].Name)
	}
	if got := findVariable(envs[0].Variables, "baseUrl"); got == nil || got.Value != "https://api.example.com" || !got.Enabled {
		t.Fatalf("expected baseUrl variable to be imported, got %+v", envs[0].Variables)
	}
	if got := findVariable(envs[0].Variables, "token"); got == nil || !got.IsSecret || got.Value != "secret-token" {
		t.Fatalf("expected token variable to be imported as secret, got %+v", envs[0].Variables)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].URL != "{{baseUrl}}/users" {
		t.Fatalf("unexpected imported request url: %s", requests[0].URL)
	}
	if len(requests[0].Headers) != 1 || requests[0].Headers[0].Description != "token header" {
		t.Fatalf("expected header description to be imported, got %+v", requests[0].Headers)
	}
	expectedBody := "{\n  // keep comment\n  \"enabled\": true\n}"
	if requests[0].Body.JSON != expectedBody {
		t.Fatalf("expected postman raw json body to be preserved, got %q", requests[0].Body.JSON)
	}
}

func TestImportPostmanCollection_AllowsJSONCAndKeepsBodyComments(t *testing.T) {
	service, store, projectID := newImportTestService(t)

	raw := []byte(`{
	  // comment in collection json
	  "info": { "name": "JSONC Collection" },
	  "item": [
	    {
	      "name": "With Comment Body",
	      "request": {
	        "method": "POST",
	        "url": "https://api.example.com/test",
	        "body": {
	          "mode": "raw",
	          "raw": "{\n  // comment in request body\n  \"a\": 1\n}",
	          "options": { "raw": { "language": "json" } }
	        }
	      }
	    }
	  ]
	}`)

	if err := service.ImportPostmanCollection(projectID, raw); err != nil {
		t.Fatalf("ImportPostmanCollection() error = %v", err)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if !strings.Contains(requests[0].Body.JSON, "// comment in request body") {
		t.Fatalf("expected raw body comments to be preserved, got %s", requests[0].Body.JSON)
	}
}

func TestImportPostmanEnvironment_ImportsVariablesOnly(t *testing.T) {
	service, store, projectID := newImportTestService(t)

	raw := []byte(`{
	  "name": "Development",
	  "_postman_variable_scope": "environment",
	  "values": [
	    { "key": "baseUrl", "value": "https://dev.example.com" },
	    { "key": "featureFlag", "value": "off", "disabled": true }
	  ]
	}`)

	if err := service.ImportPostmanEnvironment(projectID, raw); err != nil {
		t.Fatalf("ImportPostmanEnvironment() error = %v", err)
	}

	envs, err := store.ListEnvironments(projectID)
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(envs))
	}
	if envs[0].Name != "Development" {
		t.Fatalf("unexpected environment name: %s", envs[0].Name)
	}
	if got := findVariable(envs[0].Variables, "featureFlag"); got == nil || got.Enabled {
		t.Fatalf("expected disabled featureFlag variable, got %+v", envs[0].Variables)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("expected no requests from environment import, got %d", len(requests))
	}
}

func TestImportSwagger_OpenAPI3ImportsServerVariables(t *testing.T) {
	service, store, projectID := newImportTestService(t)

	raw := []byte(`{
	  "openapi": "3.0.3",
	  "info": { "title": "Petstore" },
	  "servers": [
	    {
	      "url": "https://{region}.example.com/api/{version}",
	      "variables": {
	        "region": { "default": "us" },
	        "version": { "default": "v1" }
	      }
	    }
	  ],
	  "paths": {
	    "/pets": {
	      "get": {
	        "summary": "List pets",
	        "tags": ["Pets"]
	      }
	    }
	  }
	}`)

	if err := service.ImportSwagger(projectID, raw); err != nil {
		t.Fatalf("ImportSwagger() error = %v", err)
	}

	envs, err := store.ListEnvironments(projectID)
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(envs))
	}
	if got := findVariable(envs[0].Variables, "baseUrl"); got == nil || got.Value != "https://{{region}}.example.com/api/{{version}}" {
		t.Fatalf("unexpected baseUrl variable: %+v", envs[0].Variables)
	}
	if got := findVariable(envs[0].Variables, "region"); got == nil || got.Value != "us" {
		t.Fatalf("unexpected region variable: %+v", envs[0].Variables)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].URL != "{{baseUrl}}/pets" {
		t.Fatalf("unexpected imported request url: %s", requests[0].URL)
	}
}

func TestImportSwagger_Swagger2ImportsBaseURLVariable(t *testing.T) {
	service, store, projectID := newImportTestService(t)

	raw := []byte(`{
	  "swagger": "2.0",
	  "info": { "title": "Legacy API" },
	  "host": "legacy.example.com",
	  "basePath": "/api/v1",
	  "schemes": ["https"],
	  "paths": {
	    "/users": {
	      "get": {
	        "summary": "List users"
	      }
	    }
	  }
	}`)

	if err := service.ImportSwagger(projectID, raw); err != nil {
		t.Fatalf("ImportSwagger() error = %v", err)
	}

	envs, err := store.ListEnvironments(projectID)
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(envs))
	}
	if got := findVariable(envs[0].Variables, "baseUrl"); got == nil || got.Value != "https://legacy.example.com/api/v1" {
		t.Fatalf("unexpected baseUrl variable: %+v", envs[0].Variables)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 1 || requests[0].URL != "{{baseUrl}}/users" {
		t.Fatalf("unexpected imported requests: %+v", requests)
	}
}

func TestImportSwagger_MapsParametersAndBodyTemplate(t *testing.T) {
	service, store, projectID := newImportTestService(t)

	raw := []byte(`{
	  "swagger": "2.0",
	  "info": { "title": "Param API" },
	  "host": "api.example.com",
	  "paths": {
	    "/users/{id}": {
	      "post": {
	        "summary": "Update user",
	        "parameters": [
	          { "name": "id", "in": "path", "required": true, "type": "string" },
	          { "name": "verbose", "in": "query", "type": "boolean", "description": "include debug details" },
	          { "name": "X-Trace-Id", "in": "header", "type": "string", "description": "trace id for troubleshooting" },
	          { "name": "payload", "in": "body", "required": true, "schema": { "$ref": "#/definitions/UserPayload" } }
	        ]
	      }
	    }
	  },
	  "definitions": {
	    "UserPayload": {
	      "type": "object",
	      "properties": {
	        "name": { "type": "string", "description": "user name" },
	        "enabled": { "type": "boolean", "description": "whether enabled" }
	      }
	    }
	  }
	}`)

	if err := service.ImportSwagger(projectID, raw); err != nil {
		t.Fatalf("ImportSwagger() error = %v", err)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	request := requests[0]
	if request.URL != "{{baseUrl}}/users/{{id}}" {
		t.Fatalf("unexpected request URL: %s", request.URL)
	}
	if len(request.Params) != 1 || request.Params[0].Key != "verbose" {
		t.Fatalf("unexpected params: %+v", request.Params)
	}
	if request.Params[0].Description != "include debug details" {
		t.Fatalf("expected query description, got %+v", request.Params)
	}
	if len(request.Headers) == 0 || request.Headers[0].Key != "X-Trace-Id" {
		t.Fatalf("unexpected headers: %+v", request.Headers)
	}
	if request.Headers[0].Description != "trace id for troubleshooting" {
		t.Fatalf("expected header description, got %+v", request.Headers)
	}
	if request.Body.Type != "json" {
		t.Fatalf("expected json body type, got %s", request.Body.Type)
	}
	if request.Body.JSON == "" || request.Body.JSON == "{}" {
		t.Fatalf("expected generated JSON body, got %q", request.Body.JSON)
	}
	if !strings.Contains(request.Body.JSON, "// name: user name") {
		t.Fatalf("expected body comments for field descriptions, got %s", request.Body.JSON)
	}
}

func TestImportSwagger_AllowsBooleanAdditionalProperties(t *testing.T) {
	service, store, projectID := newImportTestService(t)

	raw := []byte(`{
	  "swagger": "2.0",
	  "info": { "title": "Boolean Additional Properties" },
	  "host": "api.example.com",
	  "paths": {
	    "/items": {
	      "post": {
	        "summary": "Create items",
	        "parameters": [
	          {
	            "name": "payload",
	            "in": "body",
	            "required": true,
	            "schema": { "$ref": "#/definitions/Payload" }
	          }
	        ]
	      }
	    }
	  },
	  "definitions": {
	    "Payload": {
	      "type": "object",
	      "properties": {
	        "items": {
	          "type": "object",
	          "additionalProperties": false
	        }
	      }
	    }
	  }
	}`)

	if err := service.ImportSwagger(projectID, raw); err != nil {
		t.Fatalf("ImportSwagger() error = %v", err)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].Body.Type != "json" {
		t.Fatalf("expected json body type, got %s", requests[0].Body.Type)
	}
	if requests[0].Body.JSON == "" {
		t.Fatal("expected generated json body to be non-empty")
	}
}

func TestImportSwagger_AllowsNumericEnumInSchema(t *testing.T) {
	service, store, projectID := newImportTestService(t)

	raw := []byte(`{
	  "swagger": "2.0",
	  "info": { "title": "Numeric Enum" },
	  "host": "api.example.com",
	  "paths": {
	    "/enum-demo": {
	      "post": {
	        "summary": "Enum demo",
	        "parameters": [
	          {
	            "name": "payload",
	            "in": "body",
	            "required": true,
	            "schema": { "$ref": "#/definitions/Payload" }
	          }
	        ]
	      }
	    }
	  },
	  "definitions": {
	    "Payload": {
	      "type": "object",
	      "properties": {
	        "enum": {
	          "type": "number",
	          "enum": [1, 2, 3]
	        }
	      }
	    }
	  }
	}`)

	if err := service.ImportSwagger(projectID, raw); err != nil {
		t.Fatalf("ImportSwagger() error = %v", err)
	}

	requests, err := store.ListRequests(projectID)
	if err != nil {
		t.Fatalf("ListRequests() error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].Body.Type != "json" {
		t.Fatalf("expected json body type, got %s", requests[0].Body.Type)
	}
	if !strings.Contains(requests[0].Body.JSON, "\"enum\": 1") {
		t.Fatalf("expected enum numeric default to be generated, got %s", requests[0].Body.JSON)
	}
}
