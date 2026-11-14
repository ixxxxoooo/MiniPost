package client

import (
	"encoding/json"
	"strings"

	"minipost/internal/model"
	"minipost/internal/pkg/jsonutil"
)

// StreamStart 流式响应开始事件（response_start）携带的响应头信息。
type StreamStart struct {
	StatusCode  int                   `json:"statusCode"`
	StatusText  string                `json:"statusText"`
	Headers     map[string][]string   `json:"headers"`
	ContentType string                `json:"contentType"`
	Protocol    string                `json:"protocol,omitempty"`
	Network     *model.NetworkDetails `json:"network,omitempty"`
	HeaderBytes int64                 `json:"headerBytes"`
}

// ParseStreamStart 解析 response_start 事件的 Data；格式不合法时返回 false。
func ParseStreamStart(data string) (StreamStart, bool) {
	var s StreamStart
	if err := json.Unmarshal([]byte(data), &s); err != nil || s.StatusCode == 0 || s.Headers == nil {
		return StreamStart{}, false
	}
	return s, true
}

// WantsStreaming 判断请求是否应走流式通道：Accept 含 text/event-stream，或 JSON/raw 请求体含 "stream": true。
func WantsStreaming(d *Draft) bool {
	for _, h := range d.Headers {
		if h.Enabled && !IsAutoHeaderMarker(h.Key) && NormalizeHeaderName(h.Key) == "accept" &&
			strings.Contains(strings.ToLower(h.Value), "text/event-stream") {
			return true
		}
	}
	var candidate string
	switch d.Body.Type {
	case BodyJSON:
		candidate = d.Body.JSON
	case BodyRaw:
		candidate = d.Body.Raw
	}
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return false
	}
	var parsed struct {
		Stream any `json:"stream"`
	}
	if err := json.Unmarshal(jsonutil.StripComments([]byte(candidate)), &parsed); err != nil {
		return false
	}
	return parsed.Stream == true
}

// MergeStreamText 把大模型一类“逐 token 推送”的 SSE 事件合并成完整文本。
// 识别 OpenAI（chat/completions/responses）、Anthropic、Ollama、Gemini 的增量格式；
// 只要有任何一个事件被识别，ok 就为 true；完全不是 token 流（如普通事件日志）时 ok 为 false。
func MergeStreamText(chunks []model.StreamChunk) (text string, ok bool) {
	var sb strings.Builder
	for _, ch := range chunks {
		if ch.Kind == "response_start" || ch.Kind == "connection_closed" || ch.Kind == "error" || ch.Kind == "comment" {
			continue
		}
		data := strings.TrimSpace(ch.Data)
		if data == "" || data == "[DONE]" {
			continue
		}
		var v map[string]any
		if err := json.Unmarshal([]byte(data), &v); err != nil {
			continue
		}
		if piece, matched := streamDelta(v); matched {
			ok = true
			sb.WriteString(piece)
		}
	}
	return sb.String(), ok
}

// streamDelta 从一个事件的 JSON 里取出增量文本；matched 表示这个事件是已知的增量格式（即使文本为空）。
func streamDelta(v map[string]any) (string, bool) {
	str := func(x any) (string, bool) { s, ok := x.(string); return s, ok }
	obj := func(x any) map[string]any { m, _ := x.(map[string]any); return m }
	first := func(x any) map[string]any {
		if arr, ok := x.([]any); ok && len(arr) > 0 {
			return obj(arr[0])
		}
		return nil
	}
	// OpenAI chat：choices[0].delta.content；OpenAI completions：choices[0].text
	if ch := first(v["choices"]); ch != nil {
		if d := obj(ch["delta"]); d != nil {
			s, _ := str(d["content"])
			return s, true
		}
		if s, ok := str(ch["text"]); ok {
			return s, true
		}
		return "", true
	}
	// OpenAI responses：type=response.output_text.delta，delta 是字符串
	if t, _ := str(v["type"]); t == "response.output_text.delta" {
		s, _ := str(v["delta"])
		return s, true
	} else if t == "content_block_delta" { // Anthropic：delta.text
		s, _ := str(obj(v["delta"])["text"])
		return s, true
	}
	// Gemini：candidates[0].content.parts[*].text
	if c := first(v["candidates"]); c != nil {
		var sb strings.Builder
		if parts, ok := obj(c["content"])["parts"].([]any); ok {
			for _, p := range parts {
				s, _ := str(obj(p)["text"])
				sb.WriteString(s)
			}
		}
		return sb.String(), true
	}
	// Ollama：generate 的 response；chat 的 message.content
	if s, ok := str(v["response"]); ok {
		return s, true
	}
	if m := obj(v["message"]); m != nil {
		if s, ok := str(m["content"]); ok {
			return s, true
		}
	}
	return "", false
}
