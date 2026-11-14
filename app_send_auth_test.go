package main

import (
	"encoding/base64"
	"testing"

	"minipost/internal/client"
)

// Basic 鉴权：账号密码做 base64；都为空时不发送。
func TestSendAuthBasic(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/a")
	d.Auth = client.AuthDraft{Type: client.AuthBasic, Username: "用户", Password: "p:w d"}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("用户:p:w d"))
	if got := e.send(t, d).Header.Get("Authorization"); got != want {
		t.Fatalf("Authorization 应为 %q，实际 %q", want, got)
	}
	d.Auth = client.AuthDraft{Type: client.AuthBasic}
	if got := e.send(t, d).Header.Get("Authorization"); got != "" {
		t.Fatalf("空账号密码不应发送，实际 %q", got)
	}
}

// Bearer 鉴权，含变量；token 为空时不发送。
func TestSendAuthBearer(t *testing.T) {
	e := newSendEnv(t)
	e.vars = []client.Var{{Key: "tk", Value: "T0K"}}
	d := e.draft("/a")
	d.Auth = client.AuthDraft{Type: client.AuthBearer, Token: "{{tk}}"}
	if got := e.send(t, d).Header.Get("Authorization"); got != "Bearer T0K" {
		t.Fatalf("实际 %q", got)
	}
	d.Auth = client.AuthDraft{Type: client.AuthBearer}
	if got := e.send(t, d).Header.Get("Authorization"); got != "" {
		t.Fatalf("空 token 不应发送，实际 %q", got)
	}
}

// API Key 放 Header（默认）与放查询串。
func TestSendAuthAPIKey(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/a")
	d.Auth = client.AuthDraft{Type: client.AuthAPIKey, APIKeyName: "X-Api-Key", APIKeyValue: "k1"}
	if got := e.send(t, d).Header.Get("X-Api-Key"); got != "k1" {
		t.Fatalf("默认应放 Header，实际 %q", got)
	}
	d.Auth = client.AuthDraft{Type: client.AuthAPIKey, APIKeyName: "api key", APIKeyValue: "a&b", APIKeyLocation: "query"}
	r := e.send(t, d)
	if v := parseQuery(t, r.query()); v.Get("api key") != "a&b" || r.Header.Get("api key") != "" {
		t.Fatalf("应放查询串，实际 %q %v", r.query(), r.Header)
	}
}

// API Key 放查询串时，保留已有参数及其顺序，新参数追加在最后；同名则覆盖。
func TestSendAuthAPIKeyQueryKeepsOrder(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/a")
	d.Params = []*client.Row{kv("z", "1"), kv("a", "2"), client.NewRow()}
	syncURL(d)
	d.Auth = client.AuthDraft{Type: client.AuthAPIKey, APIKeyName: "key", APIKeyValue: "v", APIKeyLocation: "query"}
	if got := e.send(t, d).query(); got != "z=1&a=2&key=v" {
		t.Fatalf("实际 %q", got)
	}
	d.Params = []*client.Row{kv("key", "old"), kv("z", "1"), client.NewRow()}
	d.URL = e.srv.URL + "/a"
	syncURL(d)
	v := parseQuery(t, e.send(t, d).query())
	if got := v["key"]; len(got) != 1 || got[0] != "v" {
		t.Fatalf("同名 key 应被覆盖，实际 %v", v)
	}
}

// API Key 缺名称或缺值时不发送；鉴权为 none 时遗留字段也不发送；鉴权覆盖手写 Authorization。
func TestSendAuthNoneAndOverride(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("/a")
	d.Auth = client.AuthDraft{Type: client.AuthAPIKey, APIKeyName: "X-K"}
	if got := e.send(t, d).Header.Get("X-K"); got != "" {
		t.Fatalf("缺值不应发送，实际 %q", got)
	}
	d.Auth = client.AuthDraft{Type: client.AuthNone, Token: "stale", Username: "u", Password: "p"}
	if got := e.send(t, d).Header.Get("Authorization"); got != "" {
		t.Fatalf("none 不应发送凭证，实际 %q", got)
	}
	d.Headers = []*client.Row{kv("Authorization", "Bearer manual"), client.NewRow()}
	d.Auth = client.AuthDraft{Type: client.AuthBearer, Token: "fromAuth"}
	if got := e.send(t, d).Header.Get("Authorization"); got != "Bearer fromAuth" {
		t.Fatalf("鉴权页应覆盖手写 Authorization，实际 %q", got)
	}
}
