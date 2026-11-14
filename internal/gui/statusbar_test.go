package gui

import (
	"testing"

	"minipost/internal/model"
)

// 底栏显示当前环境名；没有选环境时显示“无环境”。
func TestStatusBarShowsEnvironment(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.envs = []model.Environment{{ID: "e1", Name: "预发环境"}}
	a.envID = ""
	tt.Frame()
	if a.envLabel() != "无环境" || !tt.HasText("无环境") {
		t.Fatalf("未选环境时应显示“无环境”: %q", tt.Texts())
	}
	a.envID = "e1"
	tt.Frame()
	if a.envLabel() != "预发环境" || !tt.HasText("预发环境") {
		t.Fatalf("应显示当前环境名: %q", tt.Texts())
	}
}

// 底栏显示最近一次请求：成功带状态码与耗时，失败标红，进行中提示。
func TestStatusBarLastRequestSummary(t *testing.T) {
	a, tt, _ := zhApp(t)
	if _, _, ok := a.lastRequestSummary(); ok {
		t.Fatal("没有请求时不应显示摘要")
	}
	a.con.entries = append(a.con.entries, consoleEntry{method: "GET", url: "http://x"})
	if s, _, ok := a.lastRequestSummary(); !ok || s != "请求中…" {
		t.Fatalf("进行中: %q", s)
	}
	a.con.entries[0].status, a.con.entries[0].ms, a.con.entries[0].done = 200, 127.6, true
	s, col, _ := a.lastRequestSummary()
	if s != "200 · 128ms" || col != a.statusTextColor(200) {
		t.Fatalf("成功摘要: %q", s)
	}
	tt.Frame()
	if !tt.HasText("200 · 128ms") {
		t.Fatalf("底栏应显示最近一次请求: %q", tt.Texts())
	}
	// 以最后一条为准：新一条失败
	a.con.entries = append(a.con.entries, consoleEntry{method: "GET", url: "http://y", err: "boom", done: true})
	if s, col, _ := a.lastRequestSummary(); s != "请求失败" || col != a.p.danger {
		t.Fatalf("失败摘要: %q", s)
	}
}

// 点击底栏的环境名：展开侧栏并切到“环境”页。
func TestStatusBarEnvClickOpensEnvPage(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.envs = []model.Environment{{ID: "e1", Name: "预发环境"}}
	a.envID = "e1"
	a.set.SidebarCollapsed = true
	tt.Frame()
	// 标签栏右侧的环境选择器也显示同名文字，所以按底栏按钮的无障碍名称来点。
	if err := tt.Click("当前环境（点击管理环境）"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.set.SidebarCollapsed || a.side.tab != 2 {
		t.Fatalf("应展开侧栏并切到环境页: collapsed=%v tab=%d", a.set.SidebarCollapsed, a.side.tab)
	}
}

// 真实发送一次请求后，底栏出现状态码摘要。
func TestStatusBarUpdatesAfterSend(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.activeTab().draft.URL = "http://example.test/ok"
	tt.Frame()
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	s, _, ok := a.lastRequestSummary()
	if !ok || s == "" || s == "请求中…" {
		t.Fatalf("发送后应有完成状态的摘要: %q ok=%v", s, ok)
	}
}
