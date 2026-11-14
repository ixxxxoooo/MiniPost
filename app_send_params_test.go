package main

import (
	"strings"
	"testing"

	"minipost/internal/client"
)

// 在 Params 表里的顺序就是线上的顺序，不会被按字母排序。
func TestSendParamsKeepOrder(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/p")
	d.Params = []*client.Row{kv("z", "1"), kv("a", "2"), kv("m", "3"), client.NewRow()}
	syncURL(d)
	if got := e.send(t, d).query(); got != "z=1&a=2&m=3" {
		t.Fatalf("参数顺序应保持 z,a,m，实际 %q", got)
	}
}

// 特殊字符被正确编码，服务器解码后与输入一致。
func TestSendParamsEncodeSpecialChars(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/p")
	d.Params = []*client.Row{kv("q", "a b&c=d"), kv("name", "张三"), kv("plus", "1+1=2"), kv("pct", "50%"), client.NewRow()}
	syncURL(d)
	r := e.send(t, d)
	vals := parseQuery(t, r.query())
	want := map[string]string{"q": "a b&c=d", "name": "张三", "plus": "1+1=2", "pct": "50%"}
	for k, v := range want {
		if vals.Get(k) != v {
			t.Errorf("参数 %s 服务器收到 %q，应为 %q（原始查询串 %q）", k, vals.Get(k), v, r.query())
		}
	}
	if strings.Count(r.query(), "&") != 3 {
		t.Errorf("应恰好 4 个参数，原始查询串 %q", r.query())
	}
}

// 禁用的行、没有键名的行都不发送。
func TestSendParamsSkipDisabledAndEmptyKey(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/p")
	d.Params = []*client.Row{kv("keep", "1"), off("skip", "2"), kv("", "novalue"), client.NewRow()}
	syncURL(d)
	if got := e.send(t, d).query(); got != "keep=1" {
		t.Fatalf("只应发送 keep=1，实际 %q", got)
	}
}

// 同名参数全部保留并保持顺序。
func TestSendParamsDuplicateKeys(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/p")
	d.Params = []*client.Row{kv("tag", "a"), kv("x", "1"), kv("tag", "b"), client.NewRow()}
	syncURL(d)
	if got := e.send(t, d).query(); got != "tag=a&x=1&tag=b" {
		t.Fatalf("同名参数应都保留，实际 %q", got)
	}
}

// Params 表里写环境变量：只发一次，且是解析后的值（而不是 {{x}} 字面量）。
func TestSendParamsEnvVarFromTable(t *testing.T) {
	e := newSendEnv(t)
	e.vars = []client.Var{{Key: "tk", Value: "SE CRET&1"}}
	d := e.draft("/p")
	d.Params = []*client.Row{kv("token", "{{tk}}"), kv("{{tk}}", "k"), client.NewRow()}
	syncURL(d)
	r := e.send(t, d)
	vals := parseQuery(t, r.query())
	if vals.Get("token") != "SE CRET&1" || len(vals["token"]) != 1 {
		t.Fatalf("token 应只出现一次且已解析，原始查询串 %q", r.query())
	}
	if vals.Get("SE CRET&1") != "k" {
		t.Fatalf("参数名里的变量也应解析，原始查询串 %q", r.query())
	}
	if strings.Contains(r.query(), "7B") {
		t.Fatalf("不应出现未解析的 {{}}，原始查询串 %q", r.query())
	}
}

// 直接在地址栏写变量，同样只发一次。
func TestSendParamsEnvVarTypedInURL(t *testing.T) {
	e := newSendEnv(t)
	e.vars = []client.Var{{Key: "id", Value: "42"}}
	d := e.draft("/users?id={{id}}&x=1")
	d.Params = client.SyncParamsWithURL(d.URL, d.Params)
	if got := e.send(t, d).query(); got != "id=42&x=1" {
		t.Fatalf("实际 %q", got)
	}
}

// 动态变量在参数里每次只求值一次，Params 表与 URL 不会各求一次而重复。
func TestSendParamsDynamicVarSentOnce(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/p")
	d.Params = []*client.Row{kv("ts", "{{$timestamp}}"), kv("id", "{{$randomUUID}}"), client.NewRow()}
	syncURL(d)
	vals := parseQuery(t, e.send(t, d).query())
	if len(vals["ts"]) != 1 || len(vals["id"]) != 1 {
		t.Fatalf("动态变量参数应各出现一次，实际 %v", vals)
	}
	if strings.Contains(vals.Get("ts"), "{") || len(vals.Get("id")) != 36 {
		t.Fatalf("动态变量应已求值，实际 %v", vals)
	}
}

// URL 里已有的查询串与表里没同步的参数合并：前者在前，后者追加。
func TestSendParamsUnsyncedTableAppended(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/p?a=1")
	d.Params = []*client.Row{kv("a", "1"), kv("b", "2"), client.NewRow()}
	if got := e.send(t, d).query(); got != "a=1&b=2" {
		t.Fatalf("应为 a=1&b=2，实际 %q", got)
	}
}

// 路径与主机里的变量、缺省协议、锚点。
func TestSendURLVarsSchemeAndFragment(t *testing.T) {
	e := newSendEnv(t)
	e.vars = []client.Var{{Key: "base", Value: e.srv.URL}, {Key: "id", Value: "7"}}
	d := e.draft("")
	d.URL = "{{base}}/users/{{id}}/posts?x=1#section"
	r := e.send(t, d)
	if r.URI != "/users/7/posts?x=1" {
		t.Fatalf("路径应已解析且不含锚点，实际 %q", r.URI)
	}
	d.URL = strings.TrimPrefix(e.srv.URL, "http://") + "/noscheme"
	if r := e.send(t, d); r.URI != "/noscheme" {
		t.Fatalf("缺省协议应补 http://，实际 %q", r.URI)
	}
}

// 原有的路径编码（空格、中文）不被破坏。
func TestSendURLPathEncoding(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/a b/中文")
	if r := e.send(t, d); r.URI != "/a%20b/%E4%B8%AD%E6%96%87" {
		t.Fatalf("路径应被编码，实际 %q", r.URI)
	}
}
