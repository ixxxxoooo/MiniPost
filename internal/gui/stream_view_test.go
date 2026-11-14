package gui

import (
	"fmt"
	"testing"
	"time"

	"minipost/internal/model"
)

func TestStreamOffsetLabel(t *testing.T) {
	base := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	at := func(d time.Duration) string { return base.Add(d).Format(time.RFC3339Nano) }
	cases := []struct {
		ts   string
		want string
	}{
		{at(0), "+0.000s"},
		{at(123 * time.Millisecond), "+0.123s"},
		{at(12400 * time.Millisecond), "+12.40s"},
		{at(65 * time.Second), "+1m 05s"},
		{at(-time.Second), "+0.000s"}, // 时钟回拨不出现负数
		{"bad", "+--"},
	}
	for _, tc := range cases {
		if got := streamOffsetLabel(tc.ts, base); got != tc.want {
			t.Errorf("streamOffsetLabel(%q)=%q，想要 %q", tc.ts, got, tc.want)
		}
	}
	if streamOffsetLabel(at(0), time.Time{}) != "+--" {
		t.Error("没有起点时应显示占位")
	}
}

func TestStreamStatsAndBase(t *testing.T) {
	chunks := []model.StreamChunk{
		{Kind: "response_start", Timestamp: "2026-11-14T10:00:00Z", BytesTotal: 200},
		{Kind: "data", Timestamp: "2026-11-14T10:00:01Z", BytesTotal: 30},
		{Kind: "data", Timestamp: "2026-11-14T10:00:02Z", BytesTotal: 75},
	}
	n, b := streamStats(chunks)
	if n != 2 || b != 75 {
		t.Fatalf("统计应不含 response_start，且取最大已接收字节，实际 %d 条 %d 字节", n, b)
	}
	if base := streamBase(chunks); base.Second() != 1 {
		t.Fatalf("计时起点应是第一条真实消息，实际 %v", base)
	}
}

// 带事件名的消息显示事件名徽标，心跳显示为注释；相对时间出现在行尾。
func TestStreamRowShowsEventAndOffset(t *testing.T) {
	_, tt, rs := streamApp(t)
	rs.onChunk(model.StreamChunk{Kind: "data", Event: "token", ID: "7", Data: `{"t":"a"}`, Timestamp: "2026-11-14T10:00:03.000Z", Sequence: 4})
	rs.onChunk(model.StreamChunk{Kind: "comment", Data: "ping", Timestamp: "2026-11-14T10:00:04.000Z", Sequence: 5})
	tt.Frame()
	for _, want := range []string{"token", "心跳 · ping", "+0.000s", "+0.877s", "+2.877s"} {
		if !tt.HasText(want) {
			t.Errorf("Stream 列表缺少 %q: %q", want, tt.Texts())
		}
	}
}

// 事件列表按时间先后排列：先发生的在上面。
func TestStreamRowsChronological(t *testing.T) {
	_, tt, _ := streamApp(t)
	a, _ := tt.Find(`{"n":1}`)
	b, _ := tt.Find("hello world")
	c, _ := tt.Find("连接已关闭")
	if !(a.Y < b.Y && b.Y < c.Y) {
		t.Fatalf("应按时间先后从上到下排列: %v %v %v", a.Y, b.Y, c.Y)
	}
}

// 大模型 token 流：出现“合并文本”标签，点开后是拼好的完整回答；普通事件流没有这个标签。
func TestStreamMergedTextTab(t *testing.T) {
	a, tt, _, rs := sendWith(t, &model.HttpResponse{StatusCode: 200, ContentType: "text/event-stream"})
	_ = a
	rs.onChunk(model.StreamChunk{Kind: "response_start", Data: `{"statusCode":200,"statusText":"OK","contentType":"text/event-stream"}`})
	for i, w := range []string{"你", "好", "，世界"} {
		rs.onChunk(model.StreamChunk{Kind: "data", Data: fmt.Sprintf(`{"choices":[{"delta":{"content":%q}}]}`, w),
			Timestamp: fmt.Sprintf("2026-11-14T10:00:0%d.000Z", i), Sequence: i + 1})
	}
	rs.onChunk(model.StreamChunk{Kind: "data", Data: "[DONE]", Timestamp: "2026-11-14T10:00:09.000Z", Sequence: 4})
	rs.sub = 3
	tt.Frame()
	if !tt.HasText("合并文本") {
		t.Fatalf("token 流应出现“合并文本”标签: %q", tt.Texts())
	}
	if err := tt.Click("合并文本"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := rs.stream.merged.ed.Buffer.String(); got != "你好，世界" {
		t.Fatalf("合并文本 = %q", got)
	}
	if !tt.HasText("5 字") {
		t.Fatalf("应显示字数: %q", tt.Texts())
	}
	if err := tt.Click("复制"); err != nil {
		t.Fatal(err)
	}
	if a.clipboard != "你好，世界" {
		t.Fatalf("应复制合并文本，实际 %q", a.clipboard)
	}
}

func TestStreamNoMergedTabForPlainEvents(t *testing.T) {
	_, tt, rs := streamApp(t)
	if tt.HasText("合并文本") {
		t.Fatalf("普通事件流不应出现“合并文本”: %q", tt.Texts())
	}
	rs.stream.view = 1 // 即使状态被误设，也要回到事件列表
	tt.Frame()
	if rs.stream.view != 0 {
		t.Fatal("不可合并时应回到事件视图")
	}
}

// 接收中停在底部时自动跟随新消息；用户向上滚动后不再打扰。
func TestStreamFollowsEndUnlessScrolledUp(t *testing.T) {
	_, tt, rs := streamApp(t)
	rs.streaming = true
	add := func(from, to int) {
		for i := from; i <= to; i++ {
			rs.onChunk(model.StreamChunk{Kind: "data", Data: fmt.Sprintf("line-%d", i), Timestamp: "2026-11-14T10:00:05.000Z", Sequence: 10 + i})
		}
		tt.Frame()
		tt.Frame()
	}
	add(1, 60)
	if !tt.HasText("line-60") {
		t.Fatalf("停在底部时应跟随到最新消息: 滚动 %+v", rs.stream.scroll)
	}
	if rs.stream.scroll.MaxY <= 0 {
		t.Fatal("消息足够多时应可滚动")
	}
	if rs.stream.scroll.Y < rs.stream.scroll.MaxY-4 {
		t.Fatalf("应滚动到底部: Y=%v MaxY=%v", rs.stream.scroll.Y, rs.stream.scroll.MaxY)
	}
	// 用户向上滚到顶部后又来新消息：不再被拉回底部
	rs.stream.scroll.Y = 0
	add(61, 70)
	if rs.stream.scroll.Y > 8 {
		t.Fatalf("向上滚动后不应再自动跟随，Y=%v", rs.stream.scroll.Y)
	}
}
