package httputil

import (
	"strings"
	"testing"

	"minipost/internal/model"
)

// 以下规则对照 Postman 的 cURL 导入器（postmanlabs/curl-to-postman）：
// 数据系列选项没有 Content-Type 时是 urlencoded 键值行，--data-binary 一律 raw，
// -F 是 form-data 且要删掉 Content-Type 头，等等。

func mustParse(t *testing.T, cmd string) *model.SendRequestInput {
	t.Helper()
	in, err := ParseCurlCommand(cmd)
	if err != nil {
		t.Fatalf("%s: %v", cmd, err)
	}
	return in
}

// bodyText 返回 raw / json 体的文本。
func bodyText(b model.RequestBody) string {
	if b.Type == "json" {
		return b.JSON
	}
	return b.Raw
}

// -d 系列没有 Content-Type：urlencoded 键值行；解码 %xx 与 +；请求头里不凭空多出 Content-Type。
func TestCurlDataDefaultsToURLEncoded(t *testing.T) {
	for _, flag := range []string{"-d", "--data", "--data-raw", "--data-ascii"} {
		in := mustParse(t, `curl https://x.test/a `+flag+` 'user=alice&pw=p%40ss+word&empty='`)
		if !sameForm(in.Body, "user=alice", "pw=p@ss word", "empty=") || in.Method != "POST" {
			t.Errorf("%s: got %+v method=%s", flag, in.Body, in.Method)
		}
		if len(in.Headers) != 0 {
			t.Errorf("%s: 不应凭空添加请求头: %+v", flag, in.Headers)
		}
	}
}

// 不会根据内容猜 JSON；没有 = 的段整段作键、值为空；空键保留；非法编码保留原文。
func TestCurlDataNoGuessing(t *testing.T) {
	cases := map[string][]string{
		`{"a":1}`:     {`{"a":1}=`},
		`plain text`:  {"plain text="},
		`a=1&flag`:    {"a=1", "flag="},
		`=v`:          {"=v"},
		`@/tmp/b.txt`: {"@/tmp/b.txt="},
		`a=%zz`:       {"a=%zz"},
		`a=1&&b=2&`:   {"a=1", "b=2"},
		`k=v=w`:       {"k=v=w"}, // 只按第一个 = 拆开
	}
	for data, want := range cases {
		in := mustParse(t, `curl https://x.test -d '`+data+`'`)
		if !sameForm(in.Body, want...) {
			t.Errorf("%q: got %+v want form %v", data, in.Body, want)
		}
	}
}

// 显式 Content-Type 决定请求体类型；请求头始终保留（只有 form-data 会删）。
func TestCurlDataFollowsExplicitContentType(t *testing.T) {
	cases := []struct {
		name, cmd, typ, body string
	}{
		{"json", `curl https://x.test -H 'Content-Type: application/json' -d '{"a":1}'`, "json", `{"a":1}`},
		{"json 大小写与 charset", `curl https://x.test -H 'content-type: Application/JSON; charset=utf-8' -d '{"a":1}'`, "json", `{"a":1}`},
		{"+json", `curl https://x.test -H 'Content-Type: application/vnd.api+json' -d '{"a":1}'`, "json", `{"a":1}`},
		{"表单（显式）", `curl https://x.test -H 'Content-Type: application/x-www-form-urlencoded' -d 'a=1'`, "form-urlencoded", ""},
		{"表单带 charset", `curl https://x.test -H 'Content-Type: application/x-www-form-urlencoded; charset=UTF-8' -d 'a=1'`, "form-urlencoded", ""},
		{"text/plain", `curl https://x.test -H 'Content-Type: text/plain' -d 'a=1'`, "raw", "a=1"},
		{"xml", `curl https://x.test -H 'Content-Type: application/xml' -d '<a>1</a>'`, "raw", "<a>1</a>"},
		{"显式 text/plain 的 JSON 形状仍是 raw", `curl https://x.test -H 'Content-Type: text/plain' -d '{"a":1}'`, "raw", `{"a":1}`},
	}
	for _, c := range cases {
		in := mustParse(t, c.cmd)
		if _, ok := headerValue(in.Headers, "Content-Type"); !ok {
			t.Errorf("%s: Content-Type 请求头应保留", c.name)
		}
		if in.Body.Type != c.typ || bodyText(in.Body) != c.body {
			t.Errorf("%s: type=%s body=%q, want type=%s body=%q", c.name, in.Body.Type, bodyText(in.Body), c.typ, c.body)
		}
	}
}

// --data-binary 一律是 raw（即使带 JSON 的 Content-Type），内容原样，优先级最高。
func TestCurlDataBinaryIsAlwaysRaw(t *testing.T) {
	in := mustParse(t, `curl https://x.test --data-binary 'a=1&b=2'`)
	if in.Body.Type != "raw" || in.Body.Raw != "a=1&b=2" || in.Method != "POST" || len(in.Headers) != 0 {
		t.Fatalf("got %+v method=%s headers=%+v", in.Body, in.Method, in.Headers)
	}
	in = mustParse(t, `curl https://x.test -H 'Content-Type: application/json' --data-binary '{"a":1}'`)
	if in.Body.Type != "raw" || in.Body.Raw != `{"a":1}` {
		t.Fatalf("got %+v", in.Body)
	}
	in = mustParse(t, `curl https://x.test -d 'a=1' -F 'f=1' --data-binary 'bin'`)
	if in.Body.Type != "raw" || in.Body.Raw != "bin" {
		t.Fatalf("--data-binary 应压过 -F 与 -d: %+v", in.Body)
	}
	in = mustParse(t, `curl https://x.test --data-binary 'first' --data-binary 'last'`)
	if in.Body.Raw != "last" {
		t.Fatalf("多次出现取最后一个: %+v", in.Body)
	}
}

// -F 压过 -d；form-data 要删掉 Content-Type 头（否则覆盖带 boundary 的头）。
func TestCurlFormDataWinsAndDropsContentType(t *testing.T) {
	in := mustParse(t, `curl https://x.test -H 'Content-Type: multipart/form-data' -H 'X-A: 1' -F 'a=1' -d 'b=2'`)
	if in.Body.Type != "form-data" || len(in.Body.FormData) != 1 || in.Method != "POST" {
		t.Fatalf("got %+v method=%s", in.Body, in.Method)
	}
	if _, ok := headerValue(in.Headers, "Content-Type"); ok {
		t.Fatalf("form-data 不应保留 Content-Type: %+v", in.Headers)
	}
	if v, _ := headerValue(in.Headers, "X-A"); v != "1" {
		t.Fatalf("其他请求头应保留: %+v", in.Headers)
	}
}

// -F 的细节：无 = 的整段作键；@ 为文件；;type= 后缀被去掉；引号被去掉。
func TestCurlFormFieldDetails(t *testing.T) {
	in := mustParse(t, `curl https://x.test -F 'solo' -F 'f=@/tmp/a.png;type=image/png' -F 't="hello world"' -F 'j={"a":1};type=application/json'`)
	fd := in.Body.FormData
	if len(fd) != 4 {
		t.Fatalf("got %+v", fd)
	}
	if fd[0].Key != "solo" || fd[0].Value != "" || fd[0].Type != "text" {
		t.Errorf("无 = 的段: %+v", fd[0])
	}
	if fd[1].Type != "file" || fd[1].FilePath != "/tmp/a.png" || fd[1].FileName != "a.png" {
		t.Errorf("文件字段: %+v", fd[1])
	}
	if fd[2].Value != "hello world" {
		t.Errorf("引号应被去掉: %+v", fd[2])
	}
	if fd[3].Value != `{"a":1}` {
		t.Errorf(";type= 后缀应被去掉: %+v", fd[3])
	}
}

// 方法：显式 -X 优先；否则 -I 为 HEAD、-G 为 GET、带数据或 -F 为 POST。
func TestCurlMethodRules(t *testing.T) {
	cases := map[string]string{
		`curl https://x.test`:                         "GET",
		`curl https://x.test -d a=1`:                  "POST",
		`curl https://x.test -F a=1`:                  "POST",
		`curl https://x.test --data-binary x`:         "POST",
		`curl -X PUT https://x.test -d a=1`:           "PUT",
		`curl -X GET https://x.test -d a=1`:           "GET",
		`curl --request DELETE https://x.test -d a=1`: "DELETE",
		`curl -I https://x.test`:                      "HEAD",
		`curl -G https://x.test -d a=1`:               "GET",
		`curl -X POST -G https://x.test -d a=1`:       "POST",
	}
	for cmd, want := range cases {
		if got := mustParse(t, cmd).Method; got != want {
			t.Errorf("%s: method=%s want %s", cmd, got, want)
		}
	}
}

// -I 与数据选项同时出现：Postman 不支持，报错。
func TestCurlHeadWithDataIsError(t *testing.T) {
	for _, cmd := range []string{`curl -I https://x.test -d a=1`, `curl --head https://x.test --data-binary x`, `curl -I https://x.test -F a=1`} {
		if _, err := ParseCurlCommand(cmd); err == nil {
			t.Errorf("%s: 应报错", cmd)
		}
	}
}

// -G：数据拼到查询串（锚点之前），不留请求体，方法 GET。
func TestCurlGetModeAppendsDataToQuery(t *testing.T) {
	in := mustParse(t, `curl -G https://x.test/s -d 'q=go' --data-urlencode 'name=a b'`)
	if in.URL != "https://x.test/s?q=go&name=a%20b" || in.Method != "GET" || in.Body.Type != "none" {
		t.Fatalf("url=%q method=%s body=%+v", in.URL, in.Method, in.Body)
	}
	in = mustParse(t, `curl --get 'https://x.test/s?a=1#frag' -d b=2`)
	if in.URL != "https://x.test/s?a=1&b=2#frag" {
		t.Fatalf("已有查询串用 &，锚点保留在最后: %q", in.URL)
	}
}

// --json：数据作为 JSON，自动带 Content-Type 与 Accept，已有的不覆盖。
func TestCurlJSONOption(t *testing.T) {
	in := mustParse(t, `curl https://x.test --json '{"a":1}'`)
	ct, _ := headerValue(in.Headers, "Content-Type")
	accept, _ := headerValue(in.Headers, "Accept")
	if in.Body.Type != "json" || in.Body.JSON != `{"a":1}` || in.Method != "POST" || ct != "application/json" || accept != "application/json" {
		t.Fatalf("got body=%+v method=%s ct=%q accept=%q", in.Body, in.Method, ct, accept)
	}
	in = mustParse(t, `curl https://x.test -H 'Accept: text/html' --json='{"a":1}'`)
	if accept, _ := headerValue(in.Headers, "Accept"); accept != "text/html" {
		t.Fatalf("已有 Accept 不应被覆盖: %q", accept)
	}
}

// 请求头：同名只保留先出现的（Cookie 除外）；-H 'X;' 为空值；-A 排在最前；-b 为 Cookie。
func TestCurlHeaderRules(t *testing.T) {
	in := mustParse(t, `curl https://x.test -H 'X-A: first' -H 'X-A: second' -H 'X-Empty;' -H 'Cookie: a=1' -b 'b=2' -A 'my-agent/1.0'`)
	want := []string{"User-Agent=my-agent/1.0", "X-A=first", "X-Empty=", "Cookie=a=1", "Cookie=b=2"}
	var got []string
	for _, h := range in.Headers {
		got = append(got, h.Key+"="+h.Value)
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

// -u：只有用户名时密码为空；密码里可以有冒号（按第一个冒号拆）。
func TestCurlUserOption(t *testing.T) {
	cases := map[string][2]string{
		`curl -u alice https://x.test`:         {"alice", ""},
		`curl -u alice: https://x.test`:        {"alice", ""},
		`curl -u alice:p:w https://x.test`:     {"alice", "p:w"},
		`curl --user 'a b:c d' https://x.test`: {"a b", "c d"},
	}
	for cmd, want := range cases {
		in := mustParse(t, cmd)
		if in.Auth.Type != "basic" || in.Auth.Basic.Username != want[0] || in.Auth.Basic.Password != want[1] {
			t.Errorf("%s: %+v want %v", cmd, in.Auth, want)
		}
	}
}
