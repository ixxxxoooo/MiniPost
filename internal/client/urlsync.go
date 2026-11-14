package client

import (
	"net/url"
	"strings"

	"minipost/internal/model"
)

func splitURL(input string) (head, query, hash string) {
	if i := strings.Index(input, "#"); i >= 0 {
		hash, input = input[i:], input[:i]
	}
	if i := strings.Index(input, "?"); i >= 0 {
		return input[:i], input[i+1:], hash
	}
	return input, "", hash
}

// formEscape 与浏览器 URLSearchParams 的序列化规则一致：空格为 +，保留 *-._ 与字母数字。
func formEscape(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '*', c == '-', c == '.', c == '_':
			b.WriteByte(c)
		case c == ' ':
			b.WriteByte('+')
		default:
			b.WriteString("%" + strings.ToUpper(string("0123456789abcdef"[c>>4])+string("0123456789abcdef"[c&15])))
		}
	}
	return b.String()
}

// readQueryPairs 按出现顺序解析查询串（保留重复键）。
func readQueryPairs(query string) [][2]string {
	var pairs [][2]string
	for _, part := range strings.Split(query, "&") {
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		key, err1 := url.QueryUnescape(k)
		val, err2 := url.QueryUnescape(v)
		if err1 != nil {
			key = k
		}
		if err2 != nil {
			val = v
		}
		pairs = append(pairs, [2]string{key, val})
	}
	return pairs
}

func meaningful(rows []*Row) []*Row { return TrimTrailingPlaceholders(rows) }

// SyncParamsWithURL 以 URL 查询串为准更新 Params：能复用的行（含描述、ID）保留，其余新建。
func SyncParamsWithURL(rawURL string, current []*Row) []*Row {
	_, query, _ := splitURL(rawURL)
	pairs := readQueryPairs(query)
	source := meaningful(current)

	var trailing *Row
	if n := len(current); n > 0 && current[n-1].IsPlaceholder() {
		trailing = current[n-1]
	}

	byKey := map[string][]*Row{}
	for _, r := range source {
		byKey[r.Key] = append(byKey[r.Key], r)
	}

	next := make([]*Row, 0, len(pairs)+1)
	for _, p := range pairs {
		list := byKey[p[0]]
		idx := -1
		for i, r := range list {
			if r.Value == p[1] {
				idx = i
				break
			}
		}
		if idx < 0 && len(list) > 0 {
			idx = 0
		}
		if idx < 0 {
			row := NewRow()
			row.Key, row.Value = p[0], p[1]
			next = append(next, row)
			continue
		}
		reuse := list[idx]
		byKey[p[0]] = append(list[:idx:idx], list[idx+1:]...)
		cp := *reuse
		cp.Key, cp.Value, cp.Enabled = p[0], p[1], true
		next = append(next, &cp)
	}
	if trailing != nil {
		next = append(next, trailing)
	}
	return next
}

// BuildURLWithParams 用已启用且有键名的 Params 重建 URL 的查询串，保留 hash。
func BuildURLWithParams(rawURL string, params []*Row) string {
	head, _, hash := splitURL(rawURL)
	var parts []string
	for _, r := range meaningful(params) {
		if r.Enabled && strings.TrimSpace(r.Key) != "" {
			parts = append(parts, formEscape(r.Key)+"="+formEscape(r.Value))
		}
	}
	if len(parts) == 0 {
		return head + hash
	}
	return head + "?" + strings.Join(parts, "&") + hash
}

// ParamsNotInURL 返回尚未出现在 URL 查询串中的 Params（按键值对多重集合匹配），
// 因为后端会把 Params 再追加到 URL 上，这样可避免同一参数被发送两次。
func ParamsNotInURL(rawURL string, params []model.KeyValue) []model.KeyValue {
	_, query, _ := splitURL(rawURL)
	remaining := map[[2]string]int{}
	for _, p := range readQueryPairs(query) {
		remaining[p]++
	}
	out := make([]model.KeyValue, 0, len(params))
	for _, p := range params {
		k := [2]string{p.Key, p.Value}
		if remaining[k] > 0 {
			remaining[k]--
			continue
		}
		out = append(out, p)
	}
	return out
}
