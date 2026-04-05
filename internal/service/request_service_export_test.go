package service

import (
	"testing"

	"minipost/internal/model"
)

func TestBuildRequestRawURL_AppendsQueryParams(t *testing.T) {
	req := model.RequestItem{
		URL: "https://api.example.com/users",
		Params: []model.KeyValue{
			{Key: "page", Value: "1"},
			{Key: "search", Value: "mini post"},
		},
	}

	got := buildRequestRawURL(req)
	if got != "https://api.example.com/users?page=1&search=mini+post" &&
		got != "https://api.example.com/users?search=mini+post&page=1" {
		t.Fatalf("unexpected raw URL: %s", got)
	}
}

func TestConvertRequestToPostman_JSONBody(t *testing.T) {
	req := model.RequestItem{
		Name:   "Create User",
		Method: "post",
		URL:    "https://api.example.com/users",
		Headers: []model.KeyValue{
			{Key: "Authorization", Value: "Bearer token"},
			{Key: "", Value: "should_skip"},
		},
		Body: model.RequestBody{
			Type: "json",
			JSON: `{"name":"mini"}`,
		},
	}

	pm := convertRequestToPostman(req)
	if pm.Method != "POST" {
		t.Fatalf("expected POST, got %s", pm.Method)
	}
	if pm.URL.Raw != "https://api.example.com/users" {
		t.Fatalf("unexpected raw url: %s", pm.URL.Raw)
	}
	if len(pm.Header) != 1 {
		t.Fatalf("expected 1 header, got %d", len(pm.Header))
	}
	if pm.Body == nil || pm.Body.Mode != "raw" || pm.Body.Raw != `{"name":"mini"}` {
		t.Fatalf("unexpected body export: %+v", pm.Body)
	}
	if pm.Body.Options.Raw.Language != "json" {
		t.Fatalf("expected json language, got %s", pm.Body.Options.Raw.Language)
	}
}

func TestBuildPostmanItems_PreservesTreeAndOrder(t *testing.T) {
	svc := &RequestService{}
	data := &model.CollectionData{
		Folders: []model.Folder{
			{ID: "f1", Name: "Folder 1"},
		},
		Requests: []model.RequestItem{
			{ID: "r1", Name: "Root Request", Method: "GET", URL: "https://example.com/r1"},
			{ID: "r2", Name: "Folder Request", Method: "POST", URL: "https://example.com/r2", FolderID: "f1"},
		},
		TreeNodes: []model.CollectionNode{
			{NodeID: "r1", NodeType: model.CollectionNodeTypeRequest, ParentFolderID: "", SortOrder: 0},
			{NodeID: "f1", NodeType: model.CollectionNodeTypeFolder, ParentFolderID: "", SortOrder: 1},
			{NodeID: "r2", NodeType: model.CollectionNodeTypeRequest, ParentFolderID: "f1", SortOrder: 0},
		},
	}

	items := svc.buildPostmanItems(data)
	if len(items) != 2 {
		t.Fatalf("expected 2 root items, got %d", len(items))
	}
	if items[0].Name != "Root Request" || items[0].Request == nil {
		t.Fatalf("first item should be root request: %+v", items[0])
	}
	if items[1].Name != "Folder 1" || len(items[1].Item) != 1 || items[1].Item[0].Name != "Folder Request" {
		t.Fatalf("folder item structure mismatch: %+v", items[1])
	}
}
