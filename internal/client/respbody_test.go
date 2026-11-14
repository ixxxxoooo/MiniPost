package client

import (
	"errors"
	"strings"
	"testing"
)

func TestDetectBodyMode(t *testing.T) {
	cases := []struct {
		ct, body, detect, want string
	}{
		{"application/json", "", "auto", ModeJSON},
		{"text/html; charset=utf-8", "", "auto", ModeHTML},
		{"image/svg+xml", "", "auto", ModeXML},
		{"application/javascript", "", "auto", ModeJavaScript},
		{"text/plain", `{"a":1}`, "auto", ModeJSON}, // 按内容识别
		{"", "<!DOCTYPE html><html></html>", "auto", ModeHTML},
		{"", "<?xml version=\"1.0\"?><a/>", "auto", ModeXML},
		{"", "const a = 1", "auto", ModeJavaScript},
		{"text/plain", "hello", "auto", ModeRaw},
		{"text/plain", "hello", "json", ModeJSON}, // 设置里强制按 JSON
	}
	for _, c := range cases {
		if got := DetectBodyMode(c.ct, c.body, c.detect); got != c.want {
			t.Errorf("DetectBodyMode(%q,%q,%q)=%s 期望 %s", c.ct, c.body, c.detect, got, c.want)
		}
	}
}

func TestFormatBody(t *testing.T) {
	if got := FormatBody(ModeJSON, `{"a":1,"b":[1,2]}`); !strings.Contains(got, "\n  \"a\": 1") {
		t.Errorf("JSON 应缩进: %q", got)
	}
	// NDJSON 逐行缩进
	if got := FormatBody(ModeJSON, "{\"a\":1}\n{\"b\":2}\n"); !strings.Contains(got, "\"a\": 1") || !strings.Contains(got, "\"b\": 2") {
		t.Errorf("NDJSON 应逐行缩进: %q", got)
	}
	// 非法 JSON 原样返回
	if got := FormatBody(ModeJSON, "{bad"); got != "{bad" {
		t.Errorf("非法 JSON 应原样: %q", got)
	}
	if got := FormatBody(ModeXML, `<a><b>1</b><c/></a>`); got != "<a>\n  <b>1</b>\n  <c/>\n</a>" {
		t.Errorf("XML 缩进不符: %q", got)
	}
	if got := FormatBody(ModeJavaScript, `function f(){return "a;b";}`); !strings.Contains(got, `"a;b"`) || !strings.Contains(got, "{\n  return") {
		t.Errorf("JS 排版不符（字符串内的分号不应换行）: %q", got)
	}
	if got := FormatBody(ModeHex, "Hi"); got != "48 69" {
		t.Errorf("Hex: %q", got)
	}
	if got := FormatBody(ModeBase64, "Hi"); got != "SGk=" {
		t.Errorf("Base64: %q", got)
	}
	if got := FormatBody(ModeRaw, " x "); got != " x " {
		t.Errorf("Raw 应原样: %q", got)
	}
}

func TestParseJSONPath(t *testing.T) {
	tok, ok := ParseJSONPath(`$.a.b[0]['c d']["e"]`)
	want := []any{"a", "b", 0, "c d", "e"}
	if !ok || len(tok) != len(want) {
		t.Fatalf("解析失败: %v %v", tok, ok)
	}
	for i := range want {
		if tok[i] != want[i] {
			t.Errorf("第 %d 项 = %v 期望 %v", i, tok[i], want[i])
		}
	}
	for _, bad := range []string{"a.b", "$.", "$[", "$[x]", "$['a'", "$.a..b", "$a"} {
		if _, ok := ParseJSONPath(bad); ok {
			t.Errorf("%q 应判为无效", bad)
		}
	}
	if tok, ok := ParseJSONPath("$"); !ok || len(tok) != 0 {
		t.Errorf("$ 应为根")
	}
}

func TestFilterBody(t *testing.T) {
	raw := `{"headers":{"host":"a.com","n":3},"list":[{"id":1},{"id":2}],"flag":true,"nil":null}`
	formatted := FormatBody(ModeJSON, raw)
	check := func(expr, want string) {
		t.Helper()
		got, err := FilterBody(ModeJSON, raw, formatted, expr)
		if err != nil || got != want {
			t.Errorf("FilterBody(%q) = %q, %v 期望 %q", expr, got, err, want)
		}
	}
	check("$.headers.host", "a.com") // 字符串取原文
	check("$.headers.n", "3")
	check("$.flag", "true")
	check("$.nil", "null")
	check("$.list[1]", "{\n  \"id\": 2\n}")
	check("", formatted)

	// 键顺序保持：先 host 后 n
	got, _ := FilterBody(ModeJSON, raw, formatted, "$.headers")
	if strings.Index(got, "host") > strings.Index(got, "\"n\"") {
		t.Errorf("应保持原键顺序: %s", got)
	}

	if _, err := FilterBody(ModeJSON, raw, formatted, "headers"); !errors.Is(err, ErrFilterPath) {
		t.Errorf("无效路径: %v", err)
	}
	if _, err := FilterBody(ModeJSON, raw, formatted, "$.nope"); !errors.Is(err, ErrFilterEmpty) {
		t.Errorf("不存在的路径: %v", err)
	}
	if _, err := FilterBody(ModeJSON, "{bad", "{bad", "$.a"); !errors.Is(err, ErrFilterParse) {
		t.Errorf("非法 JSON: %v", err)
	}

	// 关键字过滤行（不区分大小写）
	text := "Alpha\nbeta\nALPHABET"
	got, err := FilterBody(ModeRaw, text, text, " alpha ")
	if err != nil || got != "Alpha\nALPHABET" {
		t.Errorf("关键字过滤: %q %v", got, err)
	}
	if _, err := FilterBody(ModeRaw, text, text, "zzz"); !errors.Is(err, ErrFilterEmpty) {
		t.Errorf("无匹配应报空: %v", err)
	}
}

func TestJSONNodeBadgeAndPretty(t *testing.T) {
	n, err := ParseJSONNode(`{"z":[1,2,3],"a":{},"s":"<&>","e":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	if n.Badge() != "{4}" {
		t.Errorf("对象角标 = %s", n.Badge())
	}
	z, _ := n.Child("z")
	a, _ := n.Child("a")
	if z.Badge() != "[3]" || a.Badge() != "{}" {
		t.Errorf("角标: %s %s", z.Badge(), a.Badge())
	}
	if n.Keys[0] != "z" {
		t.Errorf("键顺序应保持: %v", n.Keys)
	}
	p := n.Pretty()
	if !strings.Contains(p, `"s": "<&>"`) || !strings.Contains(p, `"a": {}`) || !strings.Contains(p, `"e": []`) {
		t.Errorf("Pretty 不符: %s", p)
	}
	if _, err := ParseJSONNode(`{"a":1} extra`); err == nil {
		t.Errorf("末尾多余内容应报错")
	}
	if _, ok := n.Resolve([]any{"z", 5}); ok {
		t.Errorf("越界下标应失败")
	}
}

func TestPreferDownload(t *testing.T) {
	cases := []struct {
		name   string
		h      map[string][]string
		ct     string
		binary bool
		want   bool
	}{
		{"二进制", nil, "", true, true},
		{"attachment", map[string][]string{"content-disposition": {"attachment; filename=a.txt"}}, "text/plain", false, true},
		{"JSON", nil, "application/json", false, false},
		{"文本", nil, "text/html", false, false},
		{"图片", nil, "image/png", false, true},
		{"pdf", nil, "application/pdf", false, true},
		{"未知 application", nil, "application/x-foo", false, true},
		{"无类型", nil, "", false, false},
	}
	for _, c := range cases {
		if got := PreferDownload(c.h, c.ct, c.binary); got != c.want {
			t.Errorf("%s: 得到 %v 期望 %v", c.name, got, c.want)
		}
	}
}

func TestSuggestResponseFilename(t *testing.T) {
	cd := func(v string) map[string][]string { return map[string][]string{"Content-Disposition": {v}} }
	cases := []struct {
		name string
		h    map[string][]string
		ct   string
		url  string
		want string
	}{
		{"头里的文件名", cd(`attachment; filename="report.pdf"`), "application/pdf", "https://x.com/a", "report.pdf"},
		{"RFC5987", cd(`attachment; filename*=UTF-8''%E6%8A%A5%E5%91%8A.pdf`), "application/pdf", "", "报告.pdf"},
		{"URL 末段补扩展名", nil, "application/json", "https://x.com/api/users", "users.json"},
		{"已有扩展名不重复补", nil, "application/json", "https://x.com/data.json", "data.json"},
		{"URL 无路径用默认名", nil, "text/plain", "https://x.com/", "response.txt"},
		{"含变量的非法 URL", nil, "", "{{base}}/x", "response"},
		{"非法字符替换", cd(`attachment; filename="a/b:c.txt"`), "", "", "a_b_c.txt"},
		{"未知类型不加扩展名", nil, "application/x-foo", "https://x.com/blob", "blob"},
	}
	for _, c := range cases {
		if got := SuggestResponseFilename(c.h, c.ct, c.url, ""); got != c.want {
			t.Errorf("%s: 得到 %q 期望 %q", c.name, got, c.want)
		}
	}
}

func TestCanPreview(t *testing.T) {
	if !CanPreview(ModeJSON, "", "") || !CanPreview(ModeHTML, "", "") {
		t.Errorf("结构化格式应可预览")
	}
	if CanPreview(ModeRaw, "text/plain", "hello") {
		t.Errorf("纯文本不应可预览")
	}
	if !CanPreview(ModeRaw, "text/html", "x") {
		t.Errorf("HTML 类型应可预览")
	}
}
