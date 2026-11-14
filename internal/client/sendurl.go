package client

import (
	"strings"

	"minipost/internal/pkg/httputil"
)

// IsCurlCommand 判断地址栏内容是否是一条 cURL 命令（以 curl 开头并带空白分隔的参数）。
func IsCurlCommand(s string) bool {
	f := strings.Fields(s)
	return len(f) > 1 && strings.EqualFold(f[0], "curl")
}

// resolveValue 先替换环境变量，再求出内置动态变量（{{$timestamp}} 等）。
func resolveValue(s string, vars []Var) string {
	return httputil.ResolveDynamicVars(ResolveTemplate(s, vars))
}

// buildURLWithQuery 由草稿的地址和已启用的 Params 生成最终地址（未补协议）。
//
// 地址栏里的查询串与 Params 表是同步的，这里先按“未解析”的原值去重（表里多出来的才追加），
// 再逐个解析变量，最后用 escape 重新编码：
//   - 变量解析发生在解码之后，所以表格里写的 {{x}} 不会被编码成 %7B%7Bx%7D%7D 而失效；
//   - 同一个参数只出现一次，动态变量只求值一次；
//   - 参数顺序与表格、地址栏一致（不排序），同名参数全部保留。
func buildURLWithQuery(d *Draft, vars []Var, escape func(string) string) string {
	if IsCurlCommand(d.URL) { // 地址栏里是整条 cURL 命令：交给后端解析，不能拆查询串
		return ResolveTemplate(d.URL, vars)
	}
	head, query, hash := splitURL(d.URL)
	pairs := readQueryPairs(query)
	for _, p := range ParamsNotInURL(d.URL, enabledKV(d.Params)) {
		pairs = append(pairs, [2]string{p.Key, p.Value})
	}
	head = ResolveTemplate(head, vars)
	hash = ResolveTemplate(hash, vars)

	var parts []string
	for _, p := range pairs {
		key := resolveValue(p[0], vars)
		if strings.TrimSpace(key) == "" {
			continue
		}
		parts = append(parts, escape(key)+"="+escape(resolveValue(p[1], vars)))
	}
	if len(parts) == 0 {
		return head + hash
	}
	sep := "?"
	if strings.Contains(head, "?") { // 变量展开后的地址自带查询串
		sep = "&"
		if strings.HasSuffix(head, "?") || strings.HasSuffix(head, "&") {
			sep = ""
		}
	}
	return head + sep + strings.Join(parts, "&") + hash
}
