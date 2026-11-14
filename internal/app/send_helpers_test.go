package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"minipost/internal/client"
)

// recorded 是回显服务器收到的一次请求（按线上实际收到的内容记录）。
type recorded struct {
	Method string
	URI    string // 含查询串的原始请求行 URI，用来检查编码与顺序
	Host   string
	Header http.Header
	Body   []byte
}

// sendEnv 是一次端到端发送的环境：与 GUI 的 sendTab 使用同一条链路。
type sendEnv struct {
	app *App
	pid string
	srv *httptest.Server
	mu  sync.Mutex
	got []recorded
	// vars 是已启用的环境变量，set 是全局设置（默认关闭自动附加的 Cache-Control / Token 头）。
	vars []client.Var
	set  client.Settings
	jar  *client.Jar
	// handler 可自定义响应（默认回 200 ok）。
	handler http.HandlerFunc
}

// newSendEnv 启动回显服务器并准备应用。
func newSendEnv(t *testing.T) *sendEnv {
	t.Helper()
	app, pid := newTestApp(t)
	e := &sendEnv{app: app, pid: pid, set: client.DefaultSettings(), jar: client.NewJar("")}
	e.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		e.mu.Lock()
		e.got = append(e.got, recorded{Method: r.Method, URI: r.RequestURI, Host: r.Host, Header: r.Header.Clone(), Body: body})
		h := e.handler
		e.mu.Unlock()
		if h != nil {
			h(w, r)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(e.srv.Close)
	return e
}

// draft 创建指向回显服务器 path 的 GET 草稿。
func (e *sendEnv) draft(path string) *client.Draft {
	d := client.NewDraft(e.pid, "", "req")
	d.URL = e.srv.URL + path
	return d
}

// send 走完整发送链路（BuildSendInput → SendRequestWithEnv），返回服务器收到的那次请求。
func (e *sendEnv) send(t *testing.T, d *client.Draft) recorded {
	t.Helper()
	in, _ := client.BuildSendInput(d, "rid", client.SendEnv{Vars: e.vars, Settings: e.set, Jar: e.jar})
	if _, err := e.app.SendRequestWithEnv(in, e.pid, ""); err != nil {
		t.Fatalf("SendRequestWithEnv() error = %v", err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.got) == 0 {
		t.Fatalf("服务器没有收到请求")
	}
	return e.got[len(e.got)-1]
}

// kv 创建一个已启用的键值行。
func kv(k, v string) *client.Row {
	r := client.NewRow()
	r.Key, r.Value = k, v
	return r
}

// off 创建一个被禁用的键值行。
func off(k, v string) *client.Row {
	r := kv(k, v)
	r.Enabled = false
	return r
}

// syncURL 模拟界面：在 Params 表里改动后，把查询串写回 URL。
func syncURL(d *client.Draft) { d.URL = client.BuildURLWithParams(d.URL, d.Params) }

// query 返回请求行里 ? 之后的原始查询串。
func (r recorded) query() string {
	for i := 0; i < len(r.URI); i++ {
		if r.URI[i] == '?' {
			return r.URI[i+1:]
		}
	}
	return ""
}

// parseQuery 解析原始查询串，失败即终止测试。
func parseQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	v, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("查询串 %q 无法解析: %v", raw, err)
	}
	return v
}
