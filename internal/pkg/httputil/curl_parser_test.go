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

func TestParseCurlCommand_CookieFlag(t *testing.T) {
	cmd := `curl https://example.com/api -b "sid=abc; uid=1001" -H "Accept: application/json"`
	input, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("ParseCurlCommand returned error: %v", err)
	}
	if input.Method != "GET" {
		t.Fatalf("expected GET, got %s", input.Method)
	}
	if input.URL != "https://example.com/api" {
		t.Fatalf("expected URL to be parsed, got %s", input.URL)
	}

	var cookieValue string
	for _, header := range input.Headers {
		if header.Key == "Cookie" {
			cookieValue = header.Value
			break
		}
	}
	if cookieValue != "sid=abc; uid=1001" {
		t.Fatalf("expected cookie header to be parsed, got %q", cookieValue)
	}
}

func TestParseCurlCommand_CookieFlagEquals(t *testing.T) {
	cmd := `curl https://example.com/api --cookie=sid=abc123`
	input, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("ParseCurlCommand returned error: %v", err)
	}

	var cookieValue string
	for _, header := range input.Headers {
		if header.Key == "Cookie" {
			cookieValue = header.Value
			break
		}
	}
	if cookieValue != "sid=abc123" {
		t.Fatalf("expected cookie header from --cookie=..., got %q", cookieValue)
	}
}

func TestParseCurlCommand_ChromeDataRawAnsiCQuotedJSON(t *testing.T) {
	cmd := `curl 'https://bluewhale-lwj-cc.suanshubang.cc/bw-go/sql_query/check_sql?logID=log_5753193900998400' \
  -H 'accept: application/json, text/plain, */*' \
  -H 'content-type: application/json;charset=UTF-8' \
  -b 'ZYBIPSCAS=IPS_token; ZYBIPSUN=user; BL_SS=session' \
  -H 'origin: https://bluewhale-lwj-cc.suanshubang.cc' \
  --data-raw $'{"sql":"select\\n    *\\nfrom\\n    dataware.dwd_change_course\\nwhere\\n    dt = \'20230607\'\\nlimit\\n    100","script_type":2,"queue_id":384}'`

	input, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("ParseCurlCommand returned error: %v", err)
	}
	if input.Method != "POST" {
		t.Fatalf("expected POST, got %s", input.Method)
	}
	if input.URL != "https://bluewhale-lwj-cc.suanshubang.cc/bw-go/sql_query/check_sql?logID=log_5753193900998400" {
		t.Fatalf("expected URL to be parsed, got %s", input.URL)
	}
	if input.Body.Type != "json" {
		t.Fatalf("expected json body type, got %s", input.Body.Type)
	}
	expectedBody := `{"sql":"select\n    *\nfrom\n    dataware.dwd_change_course\nwhere\n    dt = '20230607'\nlimit\n    100","script_type":2,"queue_id":384}`
	if input.Body.JSON != expectedBody {
		t.Fatalf("unexpected body:\nwant %q\ngot  %q", expectedBody, input.Body.JSON)
	}

	var cookieValue string
	var contentType string
	for _, header := range input.Headers {
		switch header.Key {
		case "Cookie":
			cookieValue = header.Value
		case "content-type":
			contentType = header.Value
		}
	}
	if cookieValue != "ZYBIPSCAS=IPS_token; ZYBIPSUN=user; BL_SS=session" {
		t.Fatalf("expected cookie header to be parsed, got %q", cookieValue)
	}
	if contentType != "application/json;charset=UTF-8" {
		t.Fatalf("expected content-type header to be parsed, got %q", contentType)
	}
}
