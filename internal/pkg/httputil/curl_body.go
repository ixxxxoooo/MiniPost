package httputil

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"minipost/internal/model"
)

const (
	contentTypeHeader = "Content-Type"
	formContentType   = "application/x-www-form-urlencoded"
	jsonContentType   = "application/json"
)

// jsonMediaType 与 Postman 判断 raw 体语言为 json 的规则一致：application/json 或 application/*+json。
var jsonMediaType = regexp.MustCompile(`^application/(\S+\+)?json`)

// formTypeSuffix 匹配 -F 'file=@a.png;type=image/png' 里的 ;type=... 后缀。
var formTypeSuffix = regexp.MustCompile(`;\s*type=[^\s/;]+/[^;\s]+\s*(?:;.*)?$`)

// curlData 汇总 cURL 命令里与请求体相关的选项，解析完所有参数后统一决定请求体与方法。
type curlData struct {
	parts          []string // 每个 -d / --data* / --json 的数据，按出现顺序
	binary         string   // --data-binary 的数据（多次出现时取最后一个，同 curl）
	hasBinary      bool
	getMode        bool // -G / --get：数据拼到 URL 查询串，而不是作为请求体
	jsonMode       bool // --json：自动带 JSON 的 Content-Type 与 Accept
	head           bool // -I / --head
	methodExplicit bool // 命令里显式给了 -X / --request / -I
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

// stripFormType 去掉 -F 值末尾的 ;type=... 后缀（同 Postman）。
func stripFormType(v string) string { return formTypeSuffix.ReplaceAllString(v, "") }

// finalizeCurlBody 按 Postman 的 cURL 导入规则决定请求体、方法与请求头
// （参考 postmanlabs/curl-to-postman，另保留 curl 自身的语义）：
//   - 优先级：--data-binary（raw）> -F（form-data）> -d/--data/--data-raw/--data-ascii/--data-urlencode；
//   - 数据系列：没有 Content-Type，或是 x-www-form-urlencoded → 表单键值行（每段按第一个 = 拆开，
//     没有 = 的整段作键、值为空，百分号编码与 + 被还原）；JSON 类型 → json 体；其他类型 → raw；
//     不会根据内容“猜” JSON，也不会擅自补 Content-Type；请求头原样保留；
//   - form-data 要删掉 Content-Type 头，否则会覆盖带 boundary 的头；
//   - -G：数据追加到 URL 查询串，方法 GET；
//   - 方法：显式 -X 优先；否则 -I 为 HEAD、-G 为 GET、带数据或 -F 为 POST，其余 GET；
//   - -I 与数据选项同时出现时报错（Postman 同样不支持）。
func finalizeCurlBody(in *model.SendRequestInput, d curlData) error {
	hasData := len(d.parts) > 0 || d.hasBinary
	if d.head && !d.getMode && (hasData || in.Body.Type == "form-data") {
		return errors.New("不支持同时使用 --head/-I 与 --data 系列选项")
	}
	if d.jsonMode {
		if _, ok := headerValue(in.Headers, contentTypeHeader); !ok {
			in.Headers = append(in.Headers, model.KeyValue{Key: contentTypeHeader, Value: jsonContentType})
		}
		if _, ok := headerValue(in.Headers, "Accept"); !ok {
			in.Headers = append(in.Headers, model.KeyValue{Key: "Accept", Value: jsonContentType})
		}
	}

	if d.getMode && hasData {
		in.URL = appendQueryString(in.URL, joinData(d))
		in.Body = model.RequestBody{Type: "none"}
		in.Method = guessedMethod(in.Method, d, "GET")
		return nil
	}
	switch {
	case d.hasBinary:
		in.Body = model.RequestBody{Type: "raw", Raw: d.binary}
	case in.Body.Type == "form-data":
		dropHeader(in, contentTypeHeader)
	case len(d.parts) > 0:
		in.Body = dataBody(strings.Join(d.parts, "&"), in.Headers)
	}
	if hasData || in.Body.Type == "form-data" {
		in.Method = guessedMethod(in.Method, d, "POST")
	}
	return nil
}

// guessedMethod 在没有显式 -X 时采用 guess，否则保持命令里的方法。
func guessedMethod(current string, d curlData, guess string) string {
	if d.methodExplicit {
		return current
	}
	return guess
}

// joinData 把所有数据选项的内容按 curl 的规则用 & 连起来。
func joinData(d curlData) string {
	parts := append([]string{}, d.parts...)
	if d.hasBinary {
		parts = append(parts, d.binary)
	}
	return strings.Join(parts, "&")
}

// appendQueryString 把 query 追加到 rawURL 的查询串后面（锚点之前），已有查询串时用 & 连接。
func appendQueryString(rawURL, query string) string {
	if query == "" {
		return rawURL
	}
	hash := ""
	if i := strings.Index(rawURL, "#"); i >= 0 {
		rawURL, hash = rawURL[:i], rawURL[i:]
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
		if strings.HasSuffix(rawURL, "?") || strings.HasSuffix(rawURL, "&") {
			sep = ""
		}
	}
	return rawURL + sep + query + hash
}

// dataBody 按 Content-Type 把数据系列选项的内容转成请求体（规则见 finalizeCurlBody）。
func dataBody(data string, headers []model.KeyValue) model.RequestBody {
	ct, hasCT := headerValue(headers, contentTypeHeader)
	mediaType := strings.ToLower(strings.TrimSpace(strings.SplitN(ct, ";", 2)[0]))
	switch {
	case !hasCT || mediaType == "" || mediaType == formContentType:
		return model.RequestBody{Type: "form-urlencoded", FormUrlEncoded: parseFormPairs(data)}
	case jsonMediaType.MatchString(mediaType):
		return model.RequestBody{Type: "json", JSON: data}
	}
	return model.RequestBody{Type: "raw", Raw: data}
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

// parseFormPairs 把 a=1&b=2 拆成键值行：每段按第一个 = 拆开，没有 = 的整段作键、值为空；
// 百分号编码与 + 被还原（编码非法的部分保留原文）。空段被跳过。
func parseFormPairs(data string) []model.KeyValue {
	pairs := []model.KeyValue{}
	for _, seg := range strings.Split(strings.TrimSpace(data), "&") {
		if seg == "" {
			continue
		}
		k, v, _ := strings.Cut(seg, "=")
		pairs = append(pairs, model.KeyValue{Key: unescapeForm(k), Value: unescapeForm(v)})
	}
	return pairs
}

// unescapeForm 还原表单编码（+ 为空格）；编码非法时原样返回。
func unescapeForm(s string) string {
	if out, err := url.QueryUnescape(s); err == nil {
		return out
	}
	return s
}
