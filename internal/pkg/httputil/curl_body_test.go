package httputil

import "testing"

// parse 解析 cURL 命令，返回请求体类型、请求体文本、Content-Type 头和方法。
func parse(t *testing.T, cmd string) (typ, body, ct, method string) {
	t.Helper()
	in, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("%s: %v", cmd, err)
	}
	ct, _ = headerValue(in.Headers, "Content-Type")
	body = in.Body.Raw
	if in.Body.Type == "json" {
		body = in.Body.JSON
	}
	return in.Body.Type, body, ct, in.Method
}

// 没有 Content-Type 的 -d：与 curl 一致，按 x-www-form-urlencoded 导入为键值行，请求头里不重复显示。
func TestCurlDataDefaultsToURLEncoded(t *testing.T) {
	in, err := ParseCurlCommand(`curl https://x.test/a -d 'user=alice&pw=p%40ss+word&empty='`)
	if err != nil {
		t.Fatal(err)
	}
	if !sameForm(in.Body, "user=alice", "pw=p@ss word", "empty=") || in.Method != "POST" {
		t.Fatalf("got %+v method=%s", in.Body, in.Method)
	}
	if len(in.Headers) != 0 {
		t.Fatalf("表单体自带 Content-Type，不应再留请求头: %+v", in.Headers)
	}
	for _, flag := range []string{"--data", "--data-raw", "--data-binary"} {
		in, _ := ParseCurlCommand(`curl https://x.test/a ` + flag + ` 'a=1'`)
		if !sameForm(in.Body, "a=1") {
			t.Errorf("%s 也应默认按表单导入: %+v", flag, in.Body)
		}
	}
}

// 显式 Content-Type 决定请求体类型。
func TestCurlDataFollowsExplicitContentType(t *testing.T) {
	cases := []struct {
		name, cmd, typ, body string
		keepHeader           bool
	}{
		{"json", `curl https://x.test -H 'Content-Type: application/json' -d '{"a":1}'`, "json", `{"a":1}`, true},
		{"json 带 charset 与大小写", `curl https://x.test -H 'content-type: Application/JSON; charset=utf-8' -d '{"a":1}'`, "json", `{"a":1}`, true},
		{"+json", `curl https://x.test -H 'Content-Type: application/vnd.api+json' -d '{"a":1}'`, "json", `{"a":1}`, true},
		{"表单（显式）", `curl https://x.test -H 'Content-Type: application/x-www-form-urlencoded' -d 'a=1'`, "form-urlencoded", "", false},
		{"表单带 charset 保留请求头", `curl https://x.test -H 'Content-Type: application/x-www-form-urlencoded; charset=UTF-8' -d 'a=1'`, "form-urlencoded", "", true},
		{"text/plain", `curl https://x.test -H 'Content-Type: text/plain' -d 'a=1'`, "raw", "a=1", true},
		{"xml", `curl https://x.test -H 'Content-Type: application/xml' -d '<a>1</a>'`, "raw", "<a>1</a>", true},
		{"显式 text/plain 的 JSON 形状不被猜成 json", `curl https://x.test -H 'Content-Type: text/plain' -d '{"a":1}'`, "raw", `{"a":1}`, true},
	}
	for _, c := range cases {
		in, err := ParseCurlCommand(c.cmd)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		body := in.Body.Raw
		if in.Body.Type == "json" {
			body = in.Body.JSON
		}
		_, hasCT := headerValue(in.Headers, "Content-Type")
		if in.Body.Type != c.typ || body != c.body || hasCT != c.keepHeader {
			t.Errorf("%s: type=%s body=%q keepHeader=%v, want type=%s body=%q keepHeader=%v", c.name, in.Body.Type, body, hasCT, c.typ, c.body, c.keepHeader)
		}
	}
}

// 没有 Content-Type 且数据像 JSON：按 json 导入（几乎总是本意）。
func TestCurlDataJSONShapeWithoutContentType(t *testing.T) {
	typ, body, _, method := parse(t, `curl https://x.test -d '{"a":1}'`)
	if typ != "json" || body != `{"a":1}` || method != "POST" {
		t.Fatalf("typ=%s body=%q method=%s", typ, body, method)
	}
}

// 解析不成键值行时保持 raw，并补上 curl 默认的 Content-Type，保证重发与 curl 等价。
func TestCurlDataUnparsableKeepsRawWithDefaultContentType(t *testing.T) {
	for _, data := range []string{"plain text", "a=1&flag", "=v", "@/tmp/body.txt", "a=%zz"} {
		typ, body, ct, _ := parse(t, `curl https://x.test -d '`+data+`'`)
		if typ != "raw" || body != data || ct != formContentType {
			t.Errorf("%q: typ=%s body=%q ct=%q", data, typ, body, ct)
		}
	}
}

// 显式 -X 的方法不被 -d 改成 POST；没有 -X 时带数据为 POST。
func TestCurlDataMethod(t *testing.T) {
	cases := map[string]string{
		`curl https://x.test -d a=1`:                  "POST",
		`curl -X PUT https://x.test -d a=1`:           "PUT",
		`curl -X GET https://x.test -d a=1`:           "GET",
		`curl --request DELETE https://x.test -d a=1`: "DELETE",
	}
	for cmd, want := range cases {
		if _, _, _, m := parse(t, cmd); m != want {
			t.Errorf("%s: method=%s want %s", cmd, m, want)
		}
	}
}

// -G：数据拼到查询串，不发请求体，方法保持 GET。
func TestCurlGetModeAppendsDataToQuery(t *testing.T) {
	in, err := ParseCurlCommand(`curl -G https://x.test/s -d 'q=go' --data-urlencode 'name=a b'`)
	if err != nil {
		t.Fatal(err)
	}
	if in.URL != "https://x.test/s?q=go&name=a%20b" || in.Method != "GET" || in.Body.Type != "none" {
		t.Fatalf("url=%q method=%s body=%+v", in.URL, in.Method, in.Body)
	}
	in, _ = ParseCurlCommand(`curl --get 'https://x.test/s?a=1' -d b=2`)
	if in.URL != "https://x.test/s?a=1&b=2" || in.Method != "GET" {
		t.Fatalf("已有查询串应用 & 连接: %q", in.URL)
	}
}

// --json：数据作为 JSON，自动带 Content-Type 与 Accept，已有的不覆盖。
func TestCurlJSONOption(t *testing.T) {
	in, err := ParseCurlCommand(`curl https://x.test --json '{"a":1}'`)
	if err != nil {
		t.Fatal(err)
	}
	ct, _ := headerValue(in.Headers, "Content-Type")
	accept, _ := headerValue(in.Headers, "Accept")
	if in.Body.Type != "json" || in.Body.JSON != `{"a":1}` || in.Method != "POST" || ct != "application/json" || accept != "application/json" {
		t.Fatalf("got body=%+v method=%s ct=%q accept=%q", in.Body, in.Method, ct, accept)
	}
	in, _ = ParseCurlCommand(`curl https://x.test -H 'Accept: text/html' --json='{"a":1}'`)
	if accept, _ := headerValue(in.Headers, "Accept"); accept != "text/html" {
		t.Fatalf("已有 Accept 不应被覆盖: %q", accept)
	}
}

// -F 表单与 -d 同时出现时，以 -F 为准（curl 本身也不允许混用）。
func TestCurlFormDataWinsOverData(t *testing.T) {
	in, err := ParseCurlCommand(`curl https://x.test -F 'a=1' -d 'b=2'`)
	if err != nil {
		t.Fatal(err)
	}
	if in.Body.Type != "form-data" || len(in.Body.FormData) != 1 {
		t.Fatalf("got %+v", in.Body)
	}
}
