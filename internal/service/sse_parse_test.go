package service

import (
	"strings"
	"testing"

	"minipost/internal/model"
)

func TestParseSSEEvent(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
		want  sseEvent
	}{
		{"仅数据", []string{"data: hello"}, sseEvent{kind: "data", data: "hello", raw: "data: hello"}},
		{"事件名与 id", []string{"event: update", "id: 42", `data: {"a":1}`},
			sseEvent{kind: "data", data: `{"a":1}`, event: "update", id: "42", raw: "event: update\nid: 42\ndata: {\"a\":1}"}},
		{"多行数据", []string{"data: a", "data: b"}, sseEvent{kind: "data", data: "a\nb", raw: "data: a\ndata: b"}},
		{"心跳注释", []string{": ping"}, sseEvent{kind: "comment", data: "ping", raw: ": ping"}},
		{"只有 retry", []string{"retry: 3000"}, sseEvent{kind: "event", data: "retry: 3000", raw: "retry: 3000"}},
		{"data 无空格", []string{"data:x"}, sseEvent{kind: "data", data: "x", raw: "data:x"}},
	}
	for _, tc := range cases {
		if got := parseSSEEvent(tc.lines); got != tc.want {
			t.Errorf("%s:\n got %+v\nwant %+v", tc.name, got, tc.want)
		}
	}
}

// 整段 SSE 流：事件名、id、心跳都应出现在对应的 chunk 里，完整原文保持不变。
func TestReadSSEStreamBodyFields(t *testing.T) {
	stream := ": keep-alive\n\nevent: token\nid: 1\ndata: {\"t\":\"a\"}\n\ndata: second\n\nretry: 1000\n\n"
	var got []model.StreamChunk
	body, err := (&HttpService{}).readSSEStreamBody(strings.NewReader(stream), 0, func(c model.StreamChunk) { got = append(got, c) })
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != stream {
		t.Fatalf("响应体应保持原文，实际 %q", body)
	}
	if len(got) != 4 {
		t.Fatalf("应有 4 个事件，实际 %d: %+v", len(got), got)
	}
	if got[0].Kind != "comment" || got[0].Data != "keep-alive" {
		t.Errorf("心跳: %+v", got[0])
	}
	if got[1].Kind != "data" || got[1].Event != "token" || got[1].ID != "1" || got[1].Data != `{"t":"a"}` {
		t.Errorf("带事件名的数据: %+v", got[1])
	}
	if got[2].Event != "" || got[2].Data != "second" {
		t.Errorf("普通数据: %+v", got[2])
	}
	if got[3].Kind != "event" {
		t.Errorf("只有 retry: %+v", got[3])
	}
	for i, c := range got {
		if c.Sequence != i+1 {
			t.Errorf("序号应递增: %d → %d", i, c.Sequence)
		}
	}
}
