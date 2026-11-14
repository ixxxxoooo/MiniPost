package app

import (
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"minipost/internal/client"
)

func (e *sendEnv) post(path string) *client.Draft {
	d := e.draft(path)
	d.Method = "POST"
	return d
}

// body none：没有请求体，也不带 Content-Type。
func TestSendBodyNone(t *testing.T) {
	e := newSendEnv(t)
	d := e.post("/b")
	d.Body.Type = client.BodyNone
	d.Body.JSON, d.Body.Raw = `{"stale":1}`, "stale"
	r := e.send(t, d)
	if len(r.Body) != 0 || r.Header.Get("Content-Type") != "" {
		t.Fatalf("none 不应有请求体，实际 %q %q", r.Body, r.Header.Get("Content-Type"))
	}
}

// json：原文发送、类型 application/json；注释被去掉但字符串里的 // 保留。
func TestSendBodyJSON(t *testing.T) {
	e := newSendEnv(t)
	d := e.post("/b")
	d.Body.Type = client.BodyJSON
	d.Body.JSON = "{\n  // 注释\n  \"url\": \"https://a.b/c//d\", /* 块 */ \"n\": 1,\n  \"中文\": \"值\"\n}"
	r := e.send(t, d)
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type 应为 application/json，实际 %q", ct)
	}
	want := "{\n  \n  \"url\": \"https://a.b/c//d\",  \"n\": 1,\n  \"中文\": \"值\"\n}"
	if string(r.Body) != want {
		t.Fatalf("请求体不对:\n got %q\nwant %q", r.Body, want)
	}
}

// json 里的环境变量与动态变量被解析；空 JSON 不发请求体。
func TestSendBodyJSONVariablesAndEmpty(t *testing.T) {
	e := newSendEnv(t)
	e.vars = []client.Var{{Key: "name", Value: "alice"}}
	d := e.post("/b")
	d.Body.Type = client.BodyJSON
	d.Body.JSON = `{"name":"{{name}}","id":"{{$randomUUID}}","miss":"{{undefined}}"}`
	body := string(e.send(t, d).Body)
	if !strings.Contains(body, `"name":"alice"`) || strings.Contains(body, "{{$randomUUID}}") || !strings.Contains(body, `"miss":"{{undefined}}"`) {
		t.Fatalf("变量解析不对: %s", body)
	}
	d.Body.JSON = ""
	r := e.send(t, d)
	if len(r.Body) != 0 || r.Header.Get("Content-Type") != "" {
		t.Fatalf("空 JSON 不应发请求体，实际 %q %q", r.Body, r.Header.Get("Content-Type"))
	}
}

// raw：原样发送（含换行、中文、特殊符号），类型 text/plain，变量被解析。
func TestSendBodyRaw(t *testing.T) {
	e := newSendEnv(t)
	e.vars = []client.Var{{Key: "who", Value: "世界"}}
	d := e.post("/b")
	d.Body.Type = client.BodyRaw
	d.Body.Raw = "你好 {{who}}\n<a href=\"x\">&amp;</a>\r\n末尾空格  "
	r := e.send(t, d)
	if string(r.Body) != "你好 世界\n<a href=\"x\">&amp;</a>\r\n末尾空格  " {
		t.Fatalf("raw 内容不对: %q", r.Body)
	}
	if r.Header.Get("Content-Type") != "text/plain" {
		t.Fatalf("Content-Type 实际 %q", r.Header.Get("Content-Type"))
	}
}

// urlencoded：保持行顺序，正确编码，跳过禁用与空键，变量被解析。
func TestSendBodyURLEncoded(t *testing.T) {
	e := newSendEnv(t)
	e.vars = []client.Var{{Key: "pw", Value: "p@ss word&1"}}
	d := e.post("/b")
	d.Body.Type = client.BodyURLEncoded
	d.Body.URLEncoded = []*client.Row{kv("z", "1"), kv("user", "张三"), kv("pw", "{{pw}}"), off("skip", "x"), kv("", "novalue"), kv("a", "2"), client.NewRow()}
	r := e.send(t, d)
	if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		t.Fatalf("Content-Type 实际 %q", ct)
	}
	if want := "z=1&user=%E5%BC%A0%E4%B8%89&pw=p%40ss+word%261&a=2"; string(r.Body) != want {
		t.Fatalf("表单应保持顺序并编码:\n got %q\nwant %q", r.Body, want)
	}
	vals, err := url.ParseQuery(string(r.Body))
	if err != nil || vals.Get("pw") != "p@ss word&1" || vals.Get("user") != "张三" {
		t.Fatalf("服务器解码结果不对: %v %v", vals, err)
	}
}

// urlencoded 同名字段全部保留。
func TestSendBodyURLEncodedDuplicates(t *testing.T) {
	e := newSendEnv(t)
	d := e.post("/b")
	d.Body.Type = client.BodyURLEncoded
	d.Body.URLEncoded = []*client.Row{kv("t", "1"), kv("x", "y"), kv("t", "2"), client.NewRow()}
	if got := string(e.send(t, d).Body); got != "t=1&x=y&t=2" {
		t.Fatalf("实际 %q", got)
	}
}

// formPart 是 multipart 里已读出内容的一个字段。
type formPart struct{ name, file, data string }

// parseMultipart 解析服务器收到的 multipart 请求体（遍历时即读出内容，NextPart 后前一个 part 不可再读）。
func parseMultipart(t *testing.T, r recorded) []formPart {
	t.Helper()
	mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/form-data" || params["boundary"] == "" {
		t.Fatalf("Content-Type 应为带 boundary 的 multipart/form-data，实际 %q", r.Header.Get("Content-Type"))
	}
	mr := multipart.NewReader(strings.NewReader(string(r.Body)), params["boundary"])
	var parts []formPart
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			return parts
		}
		if err != nil {
			t.Fatalf("解析 multipart 失败: %v", err)
		}
		data, _ := io.ReadAll(p)
		parts = append(parts, formPart{p.FormName(), p.FileName(), string(data)})
	}
}

// form-data：文本字段与文件字段，保持顺序；禁用与空键名的行不发送。
func TestSendBodyFormData(t *testing.T) {
	e := newSendEnv(t)
	e.vars = []client.Var{{Key: "who", Value: "世界"}}
	dir := t.TempDir()
	file := filepath.Join(dir, "报告.txt")
	if err := os.WriteFile(file, []byte("file-content\n第二行"), 0o600); err != nil {
		t.Fatal(err)
	}
	text := client.NewFormRow()
	text.Key, text.Value = "greeting", "你好 {{who}}"
	fileRow := client.NewFormRow()
	fileRow.Key, fileRow.Type, fileRow.FilePath = "upload", "file", file
	disabled := client.NewFormRow()
	disabled.Key, disabled.Value, disabled.Enabled = "skip", "x", false
	empty := client.NewFormRow()
	empty.Value = "novalue"
	last := client.NewFormRow()
	last.Key, last.Value = "last", "z"

	d := e.post("/b")
	d.Body.Type = client.BodyFormData
	d.Body.FormData = []*client.FormRow{text, fileRow, disabled, empty, last, client.NewFormRow()}
	parts := parseMultipart(t, e.send(t, d))
	if len(parts) != 3 {
		t.Fatalf("应有 3 个字段（禁用与空键名不发送），实际 %d", len(parts))
	}
	if p := parts[0]; p.name != "greeting" || p.data != "你好 世界" {
		t.Fatalf("文本字段不对: %+v", p)
	}
	if p := parts[1]; p.name != "upload" || p.file != "报告.txt" || p.data != "file-content\n第二行" {
		t.Fatalf("文件字段不对: %+v", p)
	}
	if p := parts[2]; p.name != "last" || p.data != "z" {
		t.Fatalf("顺序应保持，第三个应是 last: %+v", p)
	}
}

// form-data 文件路径不存在：返回明确的错误，不发请求。
func TestSendBodyFormDataMissingFile(t *testing.T) {
	e := newSendEnv(t)
	row := client.NewFormRow()
	row.Key, row.Type, row.FilePath = "f", "file", filepath.Join(t.TempDir(), "nope.bin")
	d := e.post("/b")
	d.Body.Type = client.BodyFormData
	d.Body.FormData = []*client.FormRow{row}
	in, _ := client.BuildSendInput(d, "rid", client.SendEnv{Settings: e.set, Jar: e.jar})
	if _, err := e.app.SendRequestWithEnv(in, e.pid, ""); err == nil || !strings.Contains(err.Error(), "form-data") {
		t.Fatalf("应返回 form-data 文件错误，实际 %v", err)
	}
	if len(e.got) != 0 {
		t.Fatalf("不应发出请求")
	}
}

// 只发当前类型的请求体：切换类型后遗留在其他类型里的内容不会被发送。
func TestSendBodyOnlyActiveType(t *testing.T) {
	e := newSendEnv(t)
	d := e.post("/b")
	d.Body.JSON, d.Body.Raw = `{"j":1}`, "raw-text"
	d.Body.URLEncoded = []*client.Row{kv("u", "1"), client.NewRow()}
	d.Body.Type = client.BodyRaw
	if got := string(e.send(t, d).Body); got != "raw-text" {
		t.Fatalf("应只发 raw，实际 %q", got)
	}
	d.Body.Type = client.BodyURLEncoded
	if got := string(e.send(t, d).Body); got != "u=1" {
		t.Fatalf("应只发 urlencoded，实际 %q", got)
	}
}

// 各方法都能带请求体（GET/DELETE 也可以），HEAD 不带。
func TestSendBodyWithVariousMethods(t *testing.T) {
	e := newSendEnv(t)
	for _, m := range []string{"POST", "PUT", "PATCH", "DELETE", "GET"} {
		d := e.draft("/b")
		d.Method = m
		d.Body.Type, d.Body.JSON = client.BodyJSON, `{"m":"`+m+`"}`
		r := e.send(t, d)
		if r.Method != m || string(r.Body) != `{"m":"`+m+`"}` {
			t.Fatalf("%s 请求体不对: %q", m, r.Body)
		}
	}
}
