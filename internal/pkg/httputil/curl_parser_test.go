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
  --compressed \
  --location \
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

func TestParseCurlCommand_EqualsStyleOptions(t *testing.T) {
	cmd := `curl --request=PUT --url=https://example.com/api --header=content-type: application/json --data-raw='{"ok":true}' --compressed --location`
	input, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("ParseCurlCommand returned error: %v", err)
	}
	if input.Method != "PUT" {
		t.Fatalf("expected PUT, got %s", input.Method)
	}
	if input.URL != "https://example.com/api" {
		t.Fatalf("expected URL to be parsed, got %s", input.URL)
	}
	if input.Body.Type != "json" || input.Body.JSON != `{"ok":true}` {
		t.Fatalf("unexpected body: %+v", input.Body)
	}
	if len(input.Headers) != 1 || input.Headers[0].Key != "content-type" || input.Headers[0].Value != "application/json" {
		t.Fatalf("unexpected headers: %+v", input.Headers)
	}
}

func TestParseCurlCommand_EqualsStyleFormAndUser(t *testing.T) {
	cmd := `curl --url=https://example.com/upload --user=alice:secret --form=name=alice --form=file=@/tmp/avatar.png`
	input, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("ParseCurlCommand returned error: %v", err)
	}
	if input.Method != "POST" {
		t.Fatalf("expected POST, got %s", input.Method)
	}
	if input.Auth.Type != "basic" || input.Auth.Basic.Username != "alice" || input.Auth.Basic.Password != "secret" {
		t.Fatalf("unexpected auth: %+v", input.Auth)
	}
	if input.Body.Type != "form-data" || len(input.Body.FormData) != 2 {
		t.Fatalf("unexpected form body: %+v", input.Body)
	}
	if input.Body.FormData[0].Key != "name" || input.Body.FormData[0].Value != "alice" {
		t.Fatalf("unexpected text field: %+v", input.Body.FormData[0])
	}
	if input.Body.FormData[1].Key != "file" || input.Body.FormData[1].FilePath != "/tmp/avatar.png" {
		t.Fatalf("unexpected file field: %+v", input.Body.FormData[1])
	}
}

func TestParseCurlCommand_MultipleDataSegments(t *testing.T) {
	cmd := `curl https://example.com/form -d foo=1 --data bar=2 --data-raw=baz=3`
	input, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("ParseCurlCommand returned error: %v", err)
	}
	if input.Method != "POST" {
		t.Fatalf("expected POST, got %s", input.Method)
	}
	if input.Body.Type != "raw" || input.Body.Raw != "foo=1&bar=2&baz=3" {
		t.Fatalf("unexpected body: %+v", input.Body)
	}
}

func TestParseCurlCommand_AnsiCQuotedExtendedEscapes(t *testing.T) {
	cmd := "curl https://example.com/api --data-raw $'quote=\\' slash=\\\\ hex=\\x27 uni=\\u4F60 octal=\\141 bad=\\xZ short=\\u12Z'"
	input, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("ParseCurlCommand returned error: %v", err)
	}
	expected := "quote=' slash=\\ hex=' uni=\u4f60 octal=a bad=\\xZ short=\\u12Z"
	if input.Body.Type != "raw" || input.Body.Raw != expected {
		t.Fatalf("unexpected body:\nwant %q\ngot  %q", expected, input.Body.Raw)
	}
}

func TestParseCurlCommand_WindowsCmdLineContinuations(t *testing.T) {
	cmd := "curl \"https://example.com/api?x=1&y=2\" ^\r\n  -H \"accept: application/json\" ^\r\n  --data-raw \"name=alice\" ^\r\n  --compressed"
	input, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("ParseCurlCommand returned error: %v", err)
	}
	if input.Method != "POST" {
		t.Fatalf("expected POST, got %s", input.Method)
	}
	if input.URL != "https://example.com/api?x=1&y=2" {
		t.Fatalf("expected URL to be parsed, got %s", input.URL)
	}
	if input.Body.Type != "raw" || input.Body.Raw != "name=alice" {
		t.Fatalf("unexpected body: %+v", input.Body)
	}
	if len(input.Headers) != 1 || input.Headers[0].Key != "accept" || input.Headers[0].Value != "application/json" {
		t.Fatalf("unexpected headers: %+v", input.Headers)
	}
}
