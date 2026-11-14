package service

import (
	"net/url"
	"strings"

	"minipost/internal/model"
)

// encodePairs 按给定顺序把键值对编码成 a=1&b=2（空格编码为 +）；键名为空白的行被跳过。
// 与 url.Values.Encode 不同，它不会按键名排序，同名键也保持原有位置。
func encodePairs(pairs []model.KeyValue) string {
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		if strings.TrimSpace(p.Key) == "" {
			continue
		}
		parts = append(parts, url.QueryEscape(p.Key)+"="+url.QueryEscape(p.Value))
	}
	return strings.Join(parts, "&")
}

// appendQuery 把 pairs 追加到原始查询串 raw 之后：raw 里已有的部分保持原样（顺序、编码都不动）。
func appendQuery(raw string, pairs []model.KeyValue) string {
	extra := encodePairs(pairs)
	switch {
	case extra == "":
		return raw
	case raw == "":
		return extra
	case strings.HasSuffix(raw, "&"):
		return raw + extra
	}
	return raw + "&" + extra
}

// setQueryParam 在原始查询串 raw 里设置参数 key：已有的同名参数被移除，新值追加到末尾，其余保持原样。
func setQueryParam(raw, key, value string) string {
	var kept []string
	for _, part := range strings.Split(raw, "&") {
		if part == "" {
			continue
		}
		name, _, _ := strings.Cut(part, "=")
		decoded, err := url.QueryUnescape(name)
		if err != nil {
			decoded = name
		}
		if decoded == key {
			continue
		}
		kept = append(kept, part)
	}
	return appendQuery(strings.Join(kept, "&"), []model.KeyValue{{Key: key, Value: value}})
}
