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
