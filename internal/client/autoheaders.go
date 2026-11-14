package client

import "strings"

// AutoHeaderDisabledPrefix 标记"用户关闭了某个自动 Header"的特殊 Header 键前缀（与旧版数据兼容）。
const AutoHeaderDisabledPrefix = "__minipost_auto_disabled__:"

// NormalizeHeaderName 统一 Header 名的比较形式。
func NormalizeHeaderName(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

// AutoHeaderMarkerKey 生成关闭某个自动 Header 的标记键。
func AutoHeaderMarkerKey(name string) string {
	return AutoHeaderDisabledPrefix + NormalizeHeaderName(name)
}

// IsAutoHeaderMarker 判断键是否为自动 Header 关闭标记。
func IsAutoHeaderMarker(key string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(key)), AutoHeaderDisabledPrefix)
}

// SuppressedAutoHeaders 返回被用户关闭的自动 Header 名集合。
func SuppressedAutoHeaders(headers []*Row) map[string]bool {
	out := map[string]bool{}
	for _, h := range headers {
		k := strings.ToLower(strings.TrimSpace(h.Key))
		if strings.HasPrefix(k, AutoHeaderDisabledPrefix) {
			if name := strings.TrimSpace(strings.TrimPrefix(k, AutoHeaderDisabledPrefix)); name != "" {
				out[name] = true
			}
		}
	}
	return out
}
