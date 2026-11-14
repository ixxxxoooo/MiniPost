package client

import (
	"strings"
	"testing"

	"minipost/internal/model"
)

func row(key, value string, enabled bool) *Row {
	r := NewRow()
	r.Key, r.Value, r.Enabled = key, value, enabled
	return r
}

func baseDraft() *Draft {
	d := NewDraft("p1", "", "req")
	d.URL = "https://example.com"
	return d
}

func TestResolveTemplateKeepsUnknownVariables(t *testing.T) {
	vars := []Var{{Key: "host", Value: "api.test"}, {Key: " spaced ", Value: "ok"}}
	got := ResolveTemplate("{{host}}/{{ spaced }}/{{missing}}", vars)
	if got != "api.test/ok/{{missing}}" {
		t.Fatalf("unexpected: %s", got)
	}
}

func TestEnsureProtocol(t *testing.T) {
	cases := map[string]string{
		"example.com/a":        "http://example.com/a",
		"//example.com":        "http://example.com",
		"https://x.test":       "https://x.test",
		"{{base}}/users":       "{{base}}/users",
		"":                     "",
		"ftp://files.test/abc": "ftp://files.test/abc",
	}
	for in, want := range cases {
		if got := EnsureProtocol(in); got != want {
			t.Errorf("EnsureProtocol(%q)=%q want %q", in, got, want)
		}
	}
}

func TestSyncParamsReusesRowsAndKeepsPlaceholder(t *testing.T) {
	existing := row("a", "1", true)
	existing.Description = "keep me"
	placeholder := NewRow()
	got := SyncParamsWithURL("https://x.test?a=1&b=2%203#frag", []*Row{existing, placeholder})
	if len(got) != 3 || got[0].Description != "keep me" || got[1].Key != "b" || got[1].Value != "2 3" || got[2] != placeholder {
		t.Fatalf("unexpected rows: %+v", got)
	}
}

func TestBuildURLWithParamsSkipsDisabledAndKeepsHash(t *testing.T) {
	params := []*Row{row("a", "x y", true), row("skip", "1", false), NewRow()}
	if got := BuildURLWithParams("https://x.test/p?old=1#top", params); got != "https://x.test/p?a=x+y#top" {
		t.Fatalf("unexpected url: %s", got)
	}
	if got := BuildURLWithParams("https://x.test?old=1", nil); got != "https://x.test" {
		t.Fatalf("unexpected url: %s", got)
	}
}

func TestParamsNotInURLRemovesDuplicatesOnce(t *testing.T) {
	params := []model.KeyValue{{Key: "a", Value: "1"}, {Key: "a", Value: "1"}, {Key: "b", Value: "2"}}
	got := ParamsNotInURL("https://x.test?a=1", params)
	if len(got) != 2 || got[0].Value != "1" || got[1].Key != "b" {
		t.Fatalf("unexpected params: %+v", got)
	}
}

func TestMergeCookieHeaders(t *testing.T) {
	if got := MergeCookieHeaders("session=manual; preference=compact", "session=jar; csrf=token"); got != "session=manual; csrf=token; preference=compact" {
		t.Fatalf("unexpected: %s", got)
	}
	if got := MergeCookieHeaders("manual=1; malformed", "jar=2; =empty"); got != "jar=2; manual=1" {
		t.Fatalf("unexpected: %s", got)
	}
}

func TestCurlResolvesVariablesAndQuotes(t *testing.T) {
	d := baseDraft()
	d.Method, d.URL = "POST", "{{baseUrl}}/users"
	d.Params = []*Row{row("{{queryKey}}", "{{queryValue}}", true)}
	d.Headers = []*Row{row("X-Tenant", "{{tenant}}", true)}
	d.Body.Type, d.Body.JSON = BodyJSON, `{"name":"{{name}}"}`
	d.Auth.Type, d.Auth.Token = AuthBearer, "{{token}}"
	vars := []Var{{"baseUrl", "https://api.example.com"}, {"queryKey", "search term"}, {"queryValue", "O'Reilly"},
		{"tenant", "tenant-a"}, {"name", "D'Angelo"}, {"token", "secret-token"}}
	cmd := BuildCurlCommand(d, vars, "")
	for _, want := range []string{
		`'https://api.example.com/users?search%20term=O'"'"'Reilly'`,
		"-H 'X-Tenant: tenant-a'", "-H 'Authorization: Bearer secret-token'", `-d '{"name":"D'"'"'Angelo"}'`,
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("missing %q in\n%s", want, cmd)
		}
	}
	if strings.Contains(cmd, "{{") {
		t.Errorf("variables unresolved: %s", cmd)
	}
}

func TestCurlAPIKeyBasicAndForms(t *testing.T) {
	d := baseDraft()
	d.URL = "api.example.com/{{version}}"
	d.Auth = AuthDraft{Type: AuthAPIKey, APIKeyName: "{{k}}", APIKeyValue: "{{v}}", APIKeyLocation: "query"}
	cmd := BuildCurlCommand(d, []Var{{"k", "access key"}, {"v", "abc/123"}}, "")
	if !strings.Contains(cmd, "'api.example.com/{{version}}?access%20key=abc%2F123'") {
		t.Errorf("query api key: %s", cmd)
	}

	d = baseDraft()
	d.Method = "POST"
	d.Body.Type = BodyURLEncoded
	d.Body.URLEncoded = []*Row{row("{{field}}", "{{value}}", true)}
	d.Auth = AuthDraft{Type: AuthBasic, Username: "{{u}}", Password: "{{p}}"}
	cmd = BuildCurlCommand(d, []Var{{"field", "display name"}, {"value", "Mini Post"}, {"u", "api-user"}, {"p", "p'ass"}}, "")
	if !strings.Contains(cmd, `-u 'api-user:p'"'"'ass'`) || !strings.Contains(cmd, "-d 'display%20name=Mini%20Post'") {
		t.Errorf("basic/urlencoded: %s", cmd)
	}
}

func TestCurlFormDataAndDisabledRows(t *testing.T) {
	d := baseDraft()
	d.Method = "POST"
	d.Body.Type = BodyFormData
	text := NewFormRow()
	text.Key, text.Value = "description", "{{description}}"
	file := NewFormRow()
	file.Key, file.Type, file.FilePath = "{{fileField}}", "file", "{{filePath}}"
	emptyFile := NewFormRow()
	emptyFile.Key, emptyFile.Type = "empty-file", "file"
	off := NewFormRow()
	off.Key, off.Value, off.Enabled = "disabled-field", "no", false
	d.Body.FormData = []*FormRow{text, file, emptyFile, off}
	d.Params = []*Row{row("included", "yes", true), row("disabled-param", "no", false)}
	d.Headers = []*Row{row("X-Included", "yes", true), row("X-Disabled", "no", false)}
	cmd := BuildCurlCommand(d, []Var{{"description", "release artifact"}, {"fileField", "payload"}, {"filePath", "/tmp/release data.json"}}, "")
	for _, want := range []string{"--form-string 'description=release artifact'", "-F 'payload=@/tmp/release data.json'", "included=yes", "X-Included: yes"} {
		if !strings.Contains(cmd, want) {
			t.Errorf("missing %q in\n%s", want, cmd)
		}
	}
	for _, bad := range []string{"empty-file", "disabled-param", "X-Disabled", "disabled-field"} {
		if strings.Contains(cmd, bad) {
			t.Errorf("unexpected %q in\n%s", bad, cmd)
		}
	}
}

func TestCurlCookieMerge(t *testing.T) {
	d := baseDraft()
	d.Headers = []*Row{row("Cookie", "session=manual; preference=compact", true)}
	cmd := BuildCurlCommand(d, nil, "session=jar; csrf=token")
	if !strings.Contains(cmd, "-H 'Cookie: session=manual; csrf=token; preference=compact'") || strings.Count(cmd, "Cookie:") != 1 {
		t.Errorf("cookie merge: %s", cmd)
	}
}

func TestBuildSendInputPipeline(t *testing.T) {
	jar := NewJar("")
	jar.Add(Cookie{Domain: "example.com", Name: "sid", Value: "abc", Path: "/", Enabled: true})

	d := baseDraft()
	d.URL = "{{base}}/items?a=1"
	d.Params = []*Row{row("a", "1", true), row("b", "2", true), row("off", "x", false), NewRow()}
	d.Headers = []*Row{row("X-One", "1", true), row("x-one", "2", true), row(AutoHeaderMarkerKey("User-Agent"), "", true), NewRow()}
	d.Body.Type, d.Body.JSON = BodyJSON, "{\n // c\n \"a\": \"{{base}}\"}"

	set := DefaultSettings()
	set.SendNoCacheHeader, set.SendPostmanTokenHeader, set.RequestTimeoutMs = true, true, 1500
	in, info := BuildSendInput(d, "rid", SendEnv{Vars: []Var{{"base", "https://example.com"}}, Settings: set, Jar: jar})

	if in.URL != "https://example.com/items?a=1" || info.ResolvedURL != in.URL {
		t.Fatalf("url: %s", in.URL)
	}
	if len(in.Params) != 1 || in.Params[0].Key != "b" {
		t.Fatalf("params should only keep b: %+v", in.Params)
	}
	if strings.Contains(in.Body.JSON, "// c") || !strings.Contains(in.Body.JSON, "https://example.com") {
		t.Fatalf("json: %s", in.Body.JSON)
	}
	headers := map[string]string{}
	for _, h := range in.Headers {
		headers[strings.ToLower(h.Key)] = h.Value
	}
	if headers["x-one"] != "2" || headers["cookie"] != "sid=abc" || headers["cache-control"] != "no-cache" || headers["minipost-token"] == "" {
		t.Fatalf("headers: %+v", in.Headers)
	}
	if !info.CookieIncluded || !in.Options.DisableDefaultUserAgent || in.Options.TimeoutMS != 1500 {
		t.Fatalf("options/info: %+v %+v", in.Options, info)
	}
}

func TestBuildSendInputRespectsCookieSwitches(t *testing.T) {
	jar := NewJar("")
	jar.Add(Cookie{Domain: "example.com", Name: "sid", Value: "abc", Enabled: true})
	set := DefaultSettings()
	set.DisableCookies = true
	in, info := BuildSendInput(baseDraft(), "r", SendEnv{Settings: set, Jar: jar})
	if info.CookieIncluded || hasHeader(in.Headers, "Cookie") {
		t.Fatal("cookies disabled globally must not be sent")
	}
	d := baseDraft()
	d.Headers = []*Row{row(AutoHeaderMarkerKey("cookie"), "", true)}
	if _, info := BuildSendInput(d, "r", SendEnv{Settings: DefaultSettings(), Jar: jar}); info.CookieIncluded {
		t.Fatal("suppressed cookie header must not be sent")
	}
}

func TestDraftRoundTripKeepsDisabledRows(t *testing.T) {
	d := baseDraft()
	d.Params = []*Row{row("a", "1", false), NewRow()}
	d.Body.Type = BodyFormData
	f := NewFormRow()
	f.Key, f.Enabled = "k", false
	d.Body.FormData = []*FormRow{f}
	back := DraftFromItem(d.ToItem())
	if back.Params[0].Enabled || back.Body.FormData[0].Enabled {
		t.Fatal("disabled state lost")
	}
	if len(back.Params) != 2 || !back.Params[1].IsPlaceholder() {
		t.Fatalf("trailing placeholder missing: %+v", back.Params)
	}
	if back.Fingerprint() != d.Fingerprint() {
		t.Fatal("fingerprint should be stable across round trip")
	}
}

func TestWantsStreaming(t *testing.T) {
	d := baseDraft()
	if WantsStreaming(d) {
		t.Fatal("plain request must not stream")
	}
	d.Headers = []*Row{row("Accept", "text/event-stream", true)}
	if !WantsStreaming(d) {
		t.Fatal("Accept event-stream should stream")
	}
	d = baseDraft()
	d.Body.Type, d.Body.JSON = BodyJSON, "{ // c\n \"stream\": true }"
	if !WantsStreaming(d) {
		t.Fatal("stream:true body should stream")
	}
}

func TestParseStreamStart(t *testing.T) {
	s, ok := ParseStreamStart(`{"statusCode":200,"statusText":"OK","headers":{"A":["b"]},"contentType":"text/event-stream","headerBytes":12}`)
	if !ok || s.StatusCode != 200 || !IsEventStream(s.ContentType) || s.HeaderBytes != 12 {
		t.Fatalf("parse: %+v %v", s, ok)
	}
	if _, ok := ParseStreamStart("not json"); ok {
		t.Fatal("invalid payload must fail")
	}
}

func TestAutoHeadersListAndToggle(t *testing.T) {
	tr := func(zh, en string) string { return en }
	d := NewDraft("p", "", "r")
	d.Body.Type = BodyJSON
	d.Body.JSON = `{"a":1}`
	set := DefaultSettings()
	set.SendPostmanTokenHeader = true

	has := func(items []AutoHeader, key string) (AutoHeader, bool) {
		for _, it := range items {
			if it.Key == key {
				return it, true
			}
		}
		return AutoHeader{}, false
	}
	items := AutoHeaders(d, set, nil, nil, tr)
	for _, k := range []string{"MiniPost-Token", "Content-Type", "Content-Length", "Host", "User-Agent", "Accept"} {
		if it, ok := has(items, k); !ok || !it.Enabled {
			t.Fatalf("应包含启用的自动 Header %s", k)
		}
	}
	if ct, _ := has(items, "Content-Type"); ct.Value != "application/json" {
		t.Fatalf("Content-Type 应为 application/json，实际 %q", ct.Value)
	}

	// 手写同名 Header 后不再作为自动 Header 列出
	custom := NewRow()
	custom.Key, custom.Value = "accept", "text/html"
	d.Headers = append([]*Row{custom}, d.Headers...)
	if _, ok := has(AutoHeaders(d, set, nil, nil, tr), "Accept"); ok {
		t.Fatalf("手写 Accept 后不应再列出自动 Accept")
	}

	// 关闭与重新开启
	SetAutoHeaderEnabled(d, "User-Agent", false)
	if it, _ := has(AutoHeaders(d, set, nil, nil, tr), "User-Agent"); it.Enabled {
		t.Fatalf("关闭后应标记为未启用")
	}
	if !SuppressedAutoHeaders(d.Headers)["user-agent"] {
		t.Fatalf("应写入关闭标记行")
	}
	SetAutoHeaderEnabled(d, "User-Agent", false)
	count := 0
	for _, h := range d.Headers {
		if IsAutoHeaderMarker(h.Key) {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("重复关闭不应产生多个标记行，实际 %d", count)
	}
	SetAutoHeaderEnabled(d, "User-Agent", true)
	if SuppressedAutoHeaders(d.Headers)["user-agent"] {
		t.Fatalf("重新开启后应移除标记行")
	}
}
