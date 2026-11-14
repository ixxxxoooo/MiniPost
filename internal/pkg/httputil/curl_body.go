package httputil

import (
	"net/url"
	"strings"

	"minipost/internal/model"
)

const (
	contentTypeHeader = "Content-Type"
	formContentType   = "application/x-www-form-urlencoded"
	jsonContentType   = "application/json"
)

// curlData 汇总 cURL 命令里与请求体相关的选项，解析完所有参数后统一决定请求体与方法。
type curlData struct {
	parts          []string // 每个 -d / --data* / --json 的数据，按出现顺序
	getMode        bool     // -G / --get：数据拼到 URL 查询串，而不是作为请求体
	jsonMode       bool     // --json：自动带 JSON 的 Content-Type 与 Accept
	methodExplicit bool     // 命令里显式给了 -X / --request / -I
}

// headerValue 返回名为 name 的请求头的值（不分大小写，取最后一个），没有则 ok 为 false。
func headerValue(headers []model.KeyValue, name string) (string, bool) {
	value, found := "", false
	for _, h := range headers {
		if strings.EqualFold(strings.TrimSpace(h.Key), name) {
			value, found = strings.TrimSpace(h.Value), true
		}
	}
	return value, found
}

// finalizeCurlBody 按 curl 的语义和业界通行做法决定请求体、请求方法和相关请求头：
//   - -G：数据追加到 URL 查询串，不发请求体，方法保持 GET；
//   - 显式 Content-Type 优先：JSON → json 体；x-www-form-urlencoded → 表单键值行；其他类型 → raw，保留该头；
//   - 没有 Content-Type：curl 实际发的是 x-www-form-urlencoded，能解析成 k=v 就导入为表单键值行；
//     数据像 JSON 时按 json 导入（几乎总是用户本意）；解析不了（如 @文件）则保持 raw 并补上 Content-Type，
//     保证重发时与 curl 等价；
//   - 带数据且未显式指定方法时，方法为 POST。
func finalizeCurlBody(in *model.SendRequestInput, d curlData) {
	if len(d.parts) == 0 || in.Body.Type == "form-data" {
		return
	}
	data := strings.Join(d.parts, "&")
	if d.getMode {
		sep := "?"
		if strings.Contains(in.URL, "?") {
			sep = "&"
			if strings.HasSuffix(in.URL, "?") || strings.HasSuffix(in.URL, "&") {
				sep = ""
			}
		}
		in.URL += sep + data
		return
	}
	if !d.methodExplicit && in.Method == "GET" {
		in.Method = "POST"
	}
	if d.jsonMode {
		if _, ok := headerValue(in.Headers, contentTypeHeader); !ok {
			in.Headers = append(in.Headers, model.KeyValue{Key: contentTypeHeader, Value: jsonContentType})
		}
		if _, ok := headerValue(in.Headers, "Accept"); !ok {
			in.Headers = append(in.Headers, model.KeyValue{Key: "Accept", Value: jsonContentType})
		}
	}

	ct, hasCT := headerValue(in.Headers, contentTypeHeader)
	mediaType := strings.ToLower(strings.TrimSpace(strings.SplitN(ct, ";", 2)[0]))
	switch {
	case hasCT && (mediaType == jsonContentType || strings.HasSuffix(mediaType, "+json")):
		in.Body = model.RequestBody{Type: "json", JSON: data}
	case hasCT && mediaType != formContentType:
		in.Body = model.RequestBody{Type: "raw", Raw: data}
	case !hasCT && looksLikeJSONBody(data):
		in.Body = model.RequestBody{Type: "json", JSON: data}
	default:
		if pairs, ok := parseFormPairs(data); ok {
			in.Body = model.RequestBody{Type: "form-urlencoded", FormUrlEncoded: pairs}
			if mediaType == formContentType && ct == formContentType {
				dropHeader(in, contentTypeHeader) // 表单体会自动带这个头，不再重复显示
			}
			return
		}
		in.Body = model.RequestBody{Type: "raw", Raw: data}
		if !hasCT {
			in.Headers = append(in.Headers, model.KeyValue{Key: contentTypeHeader, Value: formContentType})
		}
	}
}

// dropHeader 移除名为 name 的所有请求头（不分大小写）。
func dropHeader(in *model.SendRequestInput, name string) {
	kept := in.Headers[:0:0]
	for _, h := range in.Headers {
		if !strings.EqualFold(strings.TrimSpace(h.Key), name) {
			kept = append(kept, h)
		}
	}
	in.Headers = kept
}

// parseFormPairs 把 a=1&b=2 解析为键值行（百分号编码与 + 被还原）。
// 只要有一段不是干净的 k=v（无 =、键为空、@文件、编码非法），就整体放弃，由调用方保持原样。
func parseFormPairs(data string) ([]model.KeyValue, bool) {
	var pairs []model.KeyValue
	for _, seg := range strings.Split(data, "&") {
		if seg == "" {
			continue
		}
		k, v, ok := strings.Cut(seg, "=")
		if !ok || strings.HasPrefix(seg, "@") {
			return nil, false
		}
		key, err1 := url.QueryUnescape(k)
		val, err2 := url.QueryUnescape(v)
		if err1 != nil || err2 != nil || strings.TrimSpace(key) == "" {
			return nil, false
		}
		pairs = append(pairs, model.KeyValue{Key: key, Value: val})
	}
	return pairs, len(pairs) > 0
}
