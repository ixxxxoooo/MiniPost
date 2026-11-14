package main

import (
	"net/http"
	"strings"
	"testing"

	"minipost/internal/client"
)

// 自定义 Header 原样发送；禁用行与空键名行不发送。
func TestSendHeadersCustomAndDisabled(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/h")
	d.Headers = []*client.Row{kv("X-One", "1"), off("X-Off", "2"), kv("", "v"), client.NewRow()}
	r := e.send(t, d)
	if r.Header.Get("X-One") != "1" {
		t.Fatalf("X-One 应为 1，实际 %v", r.Header)
	}
	if r.Header.Get("X-Off") != "" {
		t.Fatalf("禁用的 Header 不应发送，实际 %v", r.Header)
	}
}

// 同名 Header（不分大小写）只发最后一个。
func TestSendHeadersDuplicateLastWins(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/h")
	d.Headers = []*client.Row{kv("X-Dup", "first"), kv("x-dup", "second"), client.NewRow()}
	r := e.send(t, d)
	if got := r.Header.Values("X-Dup"); len(got) != 1 || got[0] != "second" {
		t.Fatalf("应只发最后一个，实际 %v", got)
	}
}

// Header 名和值里的环境变量、动态变量都要解析。
func TestSendHeadersVariables(t *testing.T) {
	e := newSendEnv(t)
	e.vars = []client.Var{{Key: "tk", Value: "abc"}, {Key: "hn", Value: "X-From-Var"}}
	d := e.draft("/h")
	d.Headers = []*client.Row{kv("{{hn}}", "v-{{tk}}"), kv("X-Req", "{{$randomUUID}}"), client.NewRow()}
	r := e.send(t, d)
	if r.Header.Get("X-From-Var") != "v-abc" {
		t.Fatalf("变量应解析，实际 %v", r.Header)
	}
	if got := r.Header.Get("X-Req"); len(got) != 36 {
		t.Fatalf("动态变量应求值，实际 %q", got)
	}
}

// 默认补 User-Agent 与 Accept；自定义的优先；关闭标记生效。
func TestSendHeadersDefaults(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/h")
	r := e.send(t, d)
	if r.Header.Get("User-Agent") != "MiniPost/1.0" || r.Header.Get("Accept") != "*/*" {
		t.Fatalf("应补默认 UA 与 Accept，实际 %v", r.Header)
	}
	d.Headers = []*client.Row{kv("User-Agent", "my-agent"), kv("Accept", "application/json"), client.NewRow()}
	r = e.send(t, d)
	if r.Header.Get("User-Agent") != "my-agent" || r.Header.Get("Accept") != "application/json" {
		t.Fatalf("自定义应优先，实际 %v", r.Header)
	}
	d.Headers = nil
	client.SetAutoHeaderEnabled(d, "User-Agent", false)
	client.SetAutoHeaderEnabled(d, "Accept", false)
	d.Headers = []*client.Row{kv("X-Keep", "1"), d.Headers[0], d.Headers[1]}
	r = e.send(t, d)
	if _, ok := r.Header["Accept"]; ok {
		t.Fatalf("关闭 Accept 后不应发送，实际 %v", r.Header)
	}
	if ua := r.Header.Get("User-Agent"); strings.Contains(ua, "MiniPost") {
		t.Fatalf("关闭默认 UA 后不应发送 MiniPost UA，实际 %q", ua)
	}
	if r.Header.Get("X-Keep") != "1" {
		t.Fatalf("其他 Header 应照常发送，实际 %v", r.Header)
	}
}

// 旧数据里标记行可能是启用状态：它只是开关，绝不能作为 Header 发出（否则整个请求会失败）。
func TestSendHeadersLegacyEnabledMarkerNotSent(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/h")
	d.Headers = []*client.Row{kv(client.AutoHeaderMarkerKey("Accept"), ""), kv("X-Keep", "1"), client.NewRow()}
	r := e.send(t, d)
	if _, ok := r.Header["Accept"]; ok || r.Header.Get("X-Keep") != "1" {
		t.Fatalf("标记应生效且不作为 Header 发出，实际 %v", r.Header)
	}
}

// Host 头要真正改写请求的 Host。
func TestSendHeadersHostOverride(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/h")
	d.Headers = []*client.Row{kv("Host", "api.example.test"), client.NewRow()}
	if r := e.send(t, d); r.Host != "api.example.test" {
		t.Fatalf("Host 应被改写为 api.example.test，实际 %q", r.Host)
	}
}

// Header 值首尾空白被去掉，键名首尾空白被去掉。
func TestSendHeadersTrimWhitespace(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/h")
	d.Headers = []*client.Row{kv("  X-Trim  ", "  v  "), client.NewRow()}
	if got := e.send(t, d).Header.Get("X-Trim"); got != "v" {
		t.Fatalf("应去掉首尾空白，实际 %q", got)
	}
}

// 自定义 Content-Type 覆盖请求体自动设置的类型。
func TestSendHeadersContentTypeOverride(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/h")
	d.Method = "POST"
	d.Body.Type, d.Body.JSON = client.BodyJSON, `{"a":1}`
	d.Headers = []*client.Row{kv("Content-Type", "application/vnd.api+json"), client.NewRow()}
	if got := e.send(t, d).Header.Get("Content-Type"); got != "application/vnd.api+json" {
		t.Fatalf("自定义 Content-Type 应优先，实际 %q", got)
	}
}

// Cookie：自动携带 Jar 里的 Cookie，并与手写的 Cookie 头合并；关闭后不携带。
func TestSendHeadersCookies(t *testing.T) {
	e := newSendEnv(t)
	e.jar.Add(client.Cookie{Domain: "127.0.0.1", Name: "sid", Value: "abc", Path: "/", Enabled: true})
	d := e.draft("/h")
	if got := e.send(t, d).Header.Get("Cookie"); !strings.Contains(got, "sid=abc") {
		t.Fatalf("应自动携带 Cookie，实际 %q", got)
	}
	d.Headers = []*client.Row{kv("Cookie", "mine=1"), client.NewRow()}
	got := e.send(t, d).Header.Get("Cookie")
	if !strings.Contains(got, "mine=1") || !strings.Contains(got, "sid=abc") {
		t.Fatalf("手写 Cookie 应与 Jar 合并，实际 %q", got)
	}
	e.set.DisableCookies = true
	d.Headers = nil
	if got := e.send(t, d).Header.Get("Cookie"); got != "" {
		t.Fatalf("关闭 Cookie 后不应携带，实际 %q", got)
	}
}

// 设置里开启的自动 Header：Cache-Control 与请求标识；用户手写时不重复。
func TestSendHeadersAutoFromSettings(t *testing.T) {
	e := newSendEnv(t)
	e.set.SendNoCacheHeader, e.set.SendPostmanTokenHeader = true, true
	d := e.draft("/h")
	r := e.send(t, d)
	if r.Header.Get("Cache-Control") != "no-cache" || r.Header.Get("MiniPost-Token") == "" {
		t.Fatalf("应自动附加 Cache-Control 与 Token，实际 %v", r.Header)
	}
	d.Headers = []*client.Row{kv("Cache-Control", "max-age=5"), client.NewRow()}
	if got := e.send(t, d).Header.Values("Cache-Control"); len(got) != 1 || got[0] != "max-age=5" {
		t.Fatalf("手写的 Cache-Control 应优先且不重复，实际 %v", got)
	}
}

// 各 HTTP 方法都按原样发出。
func TestSendMethods(t *testing.T) {
	e := newSendEnv(t)
	for _, m := range client.Methods {
		d := e.draft("/m")
		d.Method = m
		if got := e.send(t, d).Method; got != m {
			t.Fatalf("方法应为 %s，实际 %s", m, got)
		}
	}
	_ = http.MethodGet
}
