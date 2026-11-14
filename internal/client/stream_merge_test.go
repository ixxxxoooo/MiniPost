package client

import (
	"testing"

	"minipost/internal/model"
)

func chunksOf(datas ...string) []model.StreamChunk {
	out := []model.StreamChunk{{Kind: "response_start", Data: `{"statusCode":200}`}}
	for _, d := range datas {
		out = append(out, model.StreamChunk{Kind: "data", Data: d})
	}
	return append(out, model.StreamChunk{Kind: "connection_closed"})
}

func TestMergeStreamTextFormats(t *testing.T) {
	cases := []struct {
		name string
		in   []model.StreamChunk
		want string
	}{
		{"OpenAI chat", chunksOf(
			`{"choices":[{"delta":{"role":"assistant"}}]}`,
			`{"choices":[{"delta":{"content":"你"}}]}`,
			`{"choices":[{"delta":{"content":"好"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			`[DONE]`), "你好"},
		{"OpenAI completions", chunksOf(`{"choices":[{"text":"Hel"}]}`, `{"choices":[{"text":"lo"}]}`), "Hello"},
		{"OpenAI responses", chunksOf(
			`{"type":"response.created"}`,
			`{"type":"response.output_text.delta","delta":"Hi "}`,
			`{"type":"response.output_text.delta","delta":"there"}`), "Hi there"},
		{"Anthropic", chunksOf(
			`{"type":"message_start"}`,
			`{"type":"content_block_delta","delta":{"type":"text_delta","text":"A"}}`,
			`{"type":"content_block_delta","delta":{"type":"text_delta","text":"B"}}`), "AB"},
		{"Gemini", chunksOf(
			`{"candidates":[{"content":{"parts":[{"text":"x"},{"text":"y"}]}}]}`,
			`{"candidates":[{"content":{"parts":[{"text":"z"}]}}]}`), "xyz"},
		{"Ollama generate", chunksOf(`{"response":"o","done":false}`, `{"response":"k","done":true}`), "ok"},
		{"Ollama chat", chunksOf(`{"message":{"role":"assistant","content":"hi"}}`, `{"message":{"content":"!"}}`), "hi!"},
	}
	for _, tc := range cases {
		got, ok := MergeStreamText(tc.in)
		if !ok || got != tc.want {
			t.Errorf("%s: got (%q, %v)，想要 (%q, true)", tc.name, got, ok, tc.want)
		}
	}
}

func TestMergeStreamTextNotTokenStream(t *testing.T) {
	// 普通事件日志、非 JSON、心跳都不应被当作 token 流
	if _, ok := MergeStreamText(chunksOf(`{"n":1}`, `hello world`, `{"status":"ok"}`)); ok {
		t.Fatal("普通事件不应被识别为 token 流")
	}
	if _, ok := MergeStreamText(nil); ok {
		t.Fatal("空输入不应识别")
	}
	// 心跳注释被跳过，不影响合并
	in := append(chunksOf(`{"choices":[{"delta":{"content":"a"}}]}`), model.StreamChunk{Kind: "comment", Data: `{"choices":[{"delta":{"content":"X"}}]}`})
	if got, ok := MergeStreamText(in); !ok || got != "a" {
		t.Fatalf("注释应被忽略，实际 (%q,%v)", got, ok)
	}
}
