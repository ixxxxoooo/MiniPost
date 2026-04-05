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
