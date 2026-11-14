package gui

import (
	"testing"

	"minipost/internal/model"
)

func seedHistory(a *app, be *fakeBackend) {
	be.history[a.projectID] = []model.HistoryEntry{
		{ID: "h1", Method: "GET", URL: "https://example.com/ok", StatusCode: 200, Timestamp: "2026-11-14T10:00:00Z"},
		{ID: "h2", Method: "POST", URL: "https://example.com/missing", StatusCode: 404, Timestamp: "2026-11-14T10:01:00Z"},
		{ID: "h3", Method: "GET", URL: "https://example.com/boom", StatusCode: 503, Timestamp: "2026-11-14T10:02:00Z"},
	}
	a.reloadHistory()
	a.side.tab = 1
}

func TestHistoryMatch(t *testing.T) {
	e := model.HistoryEntry{Method: "POST", URL: "https://Example.com/Users", Name: "创建用户", StatusCode: 404}
	for q, want := range map[string]bool{
		"": true, "post": true, "POST": true, "users": true, "创建": true, "404": true,
		"4xx": true, "5xx": false, "2xx": false, "delete": false, "  4xx ": true,
	} {
		if got := historyMatch(e, q); got != want {
			t.Errorf("historyMatch(%q)=%v, 想要 %v", q, got, want)
		}
	}
}

// 历史搜索按状态段/方法过滤列表；清空搜索后全部恢复。
func TestHistorySearchFilters(t *testing.T) {
	a, tt, be := zhApp(t)
	seedHistory(a, be)
	tt.Frame()
	for _, u := range []string{"/ok", "/missing", "/boom"} {
		if _, ok := tt.Find(u); !ok {
			t.Fatalf("初始应显示 %s", u)
		}
	}
	a.side.histQuery = "5xx"
	tt.Frame()
	if _, ok := tt.Find("/boom"); !ok {
		t.Fatal("5xx 应保留 503 那条")
	}
	if _, ok := tt.Find("/ok"); ok {
		t.Fatal("5xx 不应显示 200 那条")
	}
	a.side.histQuery = "nothing-matches"
	tt.Frame()
	if _, ok := tt.Find("无匹配结果"); !ok {
		t.Fatal("没有匹配时应提示无匹配结果")
	}
	a.side.histQuery = ""
	tt.Frame()
	if _, ok := tt.Find("/ok"); !ok {
		t.Fatal("清空搜索后应恢复全部记录")
	}
}

// 右键历史记录 → 删除此记录：只删这一条。
func TestHistoryDeleteSingleEntry(t *testing.T) {
	a, tt, be := zhApp(t)
	seedHistory(a, be)
	tt.Frame()
	if err := tt.RightClick("/missing"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("删除此记录"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if len(a.history) != 2 {
		t.Fatalf("应剩 2 条: %d", len(a.history))
	}
	for _, e := range a.history {
		if e.ID == "h2" {
			t.Fatal("h2 应被删除")
		}
	}
	if len(be.history[a.projectID]) != 2 {
		t.Fatal("后端也应只剩 2 条")
	}
}

// 鼠标悬停在历史行上时，右侧出现删除按钮，点击删除该条。
func TestHistoryHoverDeleteButton(t *testing.T) {
	a, tt, be := zhApp(t)
	seedHistory(a, be)
	tt.Frame()
	r, ok := tt.Find("/missing")
	if !ok {
		t.Fatal("找不到历史行")
	}
	y := r.Y + r.H/2
	tt.Move(r.X+10, y)
	tt.Frame()
	tt.Frame()
	x := a.set.SidebarWidth - 4 - 12 - 8 // 行右侧：外边距 4 + 内边距 12 + 图标半宽 8
	tt.ClickAt(x, y)
	tt.Frame()
	tt.Frame()
	if len(a.history) != 2 {
		t.Fatalf("点删除按钮后应剩 2 条: %d", len(a.history))
	}
}
