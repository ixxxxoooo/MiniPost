package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"minipost/internal/client"
)

// 重定向开关：开启时跟随并带上最终响应；关闭时直接返回 302。
func TestSendOptionFollowRedirects(t *testing.T) {
	e := newSendEnv(t)
	e.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/final?x=1", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("final"))
	}
	d := e.draft("/start")

	in, _ := client.BuildSendInput(d, "r1", client.SendEnv{Settings: e.set, Jar: e.jar})
	resp, err := e.app.SendRequestWithEnv(in, e.pid, "")
	if err != nil || resp.StatusCode != 200 || !strings.Contains(resp.Body, "final") {
		t.Fatalf("开启重定向应拿到 200 final，实际 %+v %v", resp, err)
	}

	e.set.FollowRedirects = false
	in, _ = client.BuildSendInput(d, "r2", client.SendEnv{Settings: e.set, Jar: e.jar})
	resp, err = e.app.SendRequestWithEnv(in, e.pid, "")
	if err != nil || resp.StatusCode != http.StatusFound {
		t.Fatalf("关闭重定向应拿到 302，实际 %+v %v", resp, err)
	}
}

// 超时：服务器响应太慢时返回错误，而不是一直等。
func TestSendOptionTimeout(t *testing.T) {
	e := newSendEnv(t)
	e.handler = func(w http.ResponseWriter, r *http.Request) { time.Sleep(600 * time.Millisecond) }
	e.set.RequestTimeoutMs = 100
	d := e.draft("/slow")
	in, _ := client.BuildSendInput(d, "r", client.SendEnv{Settings: e.set, Jar: e.jar})
	start := time.Now()
	if _, err := e.app.SendRequestWithEnv(in, e.pid, ""); err == nil {
		t.Fatalf("应超时报错")
	}
	if time.Since(start) > 450*time.Millisecond {
		t.Fatalf("应在约 100ms 超时，实际 %v", time.Since(start))
	}
}

// 响应状态码、响应头与响应体原样带回。
func TestSendResponseFields(t *testing.T) {
	e := newSendEnv(t)
	e.handler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Out", "yes")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}
	d := e.draft("/r")
	in, _ := client.BuildSendInput(d, "r", client.SendEnv{Settings: e.set, Jar: e.jar})
	resp, err := e.app.SendRequestWithEnv(in, e.pid, "")
	if err != nil || resp.StatusCode != 201 || resp.Body != `{"ok":true}` || resp.Headers["X-Out"][0] != "yes" {
		t.Fatalf("响应不对: %+v %v", resp, err)
	}
}

// 地址栏里直接粘贴 cURL 命令：方法、Header、请求体都被解析并发出。
func TestSendCurlInURLField(t *testing.T) {
	e := newSendEnv(t)
	d := e.draft("")
	d.URL = `curl -X POST '` + e.srv.URL + `/c?x=1' -H 'X-Token: abc' -H 'Content-Type: application/json' -d '{"a":1}'`
	in, _ := client.BuildSendInput(d, "r", client.SendEnv{Settings: e.set, Jar: e.jar})
	if _, err := e.app.SendRequestWithEnv(in, e.pid, ""); err != nil {
		t.Fatal(err)
	}
	r := e.got[len(e.got)-1]
	if r.Method != "POST" || r.URI != "/c?x=1" || r.Header.Get("X-Token") != "abc" || string(r.Body) != `{"a":1}` {
		t.Fatalf("cURL 解析后发送不对: %+v body=%q", r, r.Body)
	}
}

// 导入的 cURL 生成草稿后再发送，与原命令等价（往返一致）。
func TestSendDraftFromImportedCurl(t *testing.T) {
	e := newSendEnv(t)
	cmd := `curl -X PUT '` + e.srv.URL + `/i?b=2&a=1' -H 'X-One: 1' -u 'user:pw' --data-urlencode 'k=v w'`
	in, err := e.app.ImportCurl(cmd)
	if err != nil {
		t.Fatalf("ImportCurl() error = %v", err)
	}
	d := client.DraftFromSendInput(*in, "imp", e.pid, "")
	r := e.send(t, d)
	if r.Method != "PUT" || r.Header.Get("X-One") != "1" || r.Header.Get("Authorization") == "" {
		t.Fatalf("导入后发送不对: %+v", r)
	}
	if got := r.query(); got != "b=2&a=1" {
		t.Fatalf("导入的参数顺序应保持，实际 %q", got)
	}
	if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || string(r.Body) != "k=v+w" {
		t.Fatalf("没有 Content-Type 的 --data-urlencode 应按表单发送，实际 %q %q", r.Header.Get("Content-Type"), r.Body)
	}
}

// 导入 cURL 再发送，线上收到的 Content-Type 与 body 要和 curl 自己发的等价。
func TestSendImportedCurlBodyMatchesCurl(t *testing.T) {
	cases := []struct {
		name, args, wantCT, wantBody string
	}{
		{"默认表单", `-d 'a=1&b=x y'`, "application/x-www-form-urlencoded", "a=1&b=x+y"},
		{"多个 -d", `-d a=1 --data b=2`, "application/x-www-form-urlencoded", "a=1&b=2"},
		{"显式 JSON", `-H 'Content-Type: application/json' -d '{"a":1}'`, "application/json", `{"a":1}`},
		{"显式 text/plain", `-H 'Content-Type: text/plain' -d 'a=1'`, "text/plain", "a=1"},
		{"无法拆成键值行", `-d 'plain text'`, "application/x-www-form-urlencoded", "plain text"},
		{"--json", `--json '{"a":1}'`, "application/json", `{"a":1}`},
	}
	for _, c := range cases {
		e := newSendEnv(t)
		in, err := e.app.ImportCurl(`curl '` + e.srv.URL + `/i' ` + c.args)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		r := e.send(t, client.DraftFromSendInput(*in, "imp", e.pid, ""))
		if r.Method != "POST" || r.Header.Get("Content-Type") != c.wantCT || string(r.Body) != c.wantBody {
			t.Errorf("%s: method=%s ct=%q body=%q，期望 POST %q %q", c.name, r.Method, r.Header.Get("Content-Type"), r.Body, c.wantCT, c.wantBody)
		}
	}
}

// -G 把数据拼进查询串，请求不带 body。
func TestSendImportedCurlGetMode(t *testing.T) {
	e := newSendEnv(t)
	in, err := e.app.ImportCurl(`curl -G '` + e.srv.URL + `/s' -d 'q=go lang' --data-urlencode 'n=a&b'`)
	if err != nil {
		t.Fatal(err)
	}
	r := e.send(t, client.DraftFromSendInput(*in, "imp", e.pid, ""))
	v := parseQuery(t, r.query())
	if r.Method != "GET" || len(r.Body) != 0 || v.Get("q") != "go lang" || v.Get("n") != "a&b" {
		t.Fatalf("method=%s body=%q query=%q", r.Method, r.Body, r.query())
	}
}
