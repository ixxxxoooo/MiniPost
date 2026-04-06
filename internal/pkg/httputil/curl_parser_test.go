package httputil

import "testing"

func TestParseCurlCommand_MultilineCommand(t *testing.T) {
	cmd := `curl --request POST \
  --url https://example.com/api/v1/users \
  --header 'Content-Type: application/json' \
  --data '{"username":"alice"}'`

	input, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("ParseCurlCommand returned error: %v", err)
	}
	if input.Method != "POST" {
		t.Fatalf("expected POST, got %s", input.Method)
	}
	if input.URL != "https://example.com/api/v1/users" {
		t.Fatalf("expected URL to be parsed, got %s", input.URL)
	}
	if input.Body.Type != "json" {
		t.Fatalf("expected json body type, got %s", input.Body.Type)
	}
	if input.Body.JSON != `{"username":"alice"}` {
		t.Fatalf("unexpected body: %q", input.Body.JSON)
	}
	if len(input.Headers) != 1 || input.Headers[0].Key != "Content-Type" || input.Headers[0].Value != "application/json" {
		t.Fatalf("unexpected headers: %+v", input.Headers)
	}
}

func TestParseCurlCommand_InvalidWhenURLMissing(t *testing.T) {
	_, err := ParseCurlCommand("curl -X POST")
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestParseCurlCommand_UppercaseCurlPrefix(t *testing.T) {
	input, err := ParseCurlCommand("CURL https://example.com/health")
	if err != nil {
		t.Fatalf("ParseCurlCommand returned error: %v", err)
	}
	if input.Method != "GET" {
		t.Fatalf("expected default method GET, got %s", input.Method)
	}
	if input.URL != "https://example.com/health" {
		t.Fatalf("expected URL to be parsed, got %s", input.URL)
	}
}

func TestParseCurlCommand_FormData(t *testing.T) {
	cmd := `curl -X POST https://example.com/upload -F "name=alice" -F "file=@/tmp/avatar.png"`
	input, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("ParseCurlCommand returned error: %v", err)
	}
	if input.Method != "POST" {
		t.Fatalf("expected POST, got %s", input.Method)
	}
	if input.Body.Type != "form-data" {
		t.Fatalf("expected form-data body type, got %s", input.Body.Type)
	}
	if len(input.Body.FormData) != 2 {
		t.Fatalf("expected 2 form-data items, got %d", len(input.Body.FormData))
	}
	if input.Body.FormData[0].Key != "name" || input.Body.FormData[0].Type != "text" || input.Body.FormData[0].Value != "alice" {
		t.Fatalf("unexpected text form-data item: %+v", input.Body.FormData[0])
	}
	if input.Body.FormData[1].Key != "file" || input.Body.FormData[1].Type != "file" || input.Body.FormData[1].FilePath != "/tmp/avatar.png" {
		t.Fatalf("unexpected file form-data item: %+v", input.Body.FormData[1])
	}
}
