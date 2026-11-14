package gui

import (
	"strings"

	"minipost/internal/model"
)

// 搜索命中的位置，按优先级排列：先看名称和地址，再看方法、参数、请求头、请求体。
const (
	hitNone   = ""
	hitName   = "name"
	hitURL    = "url"
	hitMethod = "method"
	hitParam  = "param"
	hitHeader = "header"
	hitBody   = "body"
)

// requestHit 返回请求里第一个包含搜索词 q（已转小写）的位置，没有命中返回 hitNone。
// 只搜索名称、地址、方法、参数、请求头和请求体；鉴权信息含密钥，不参与搜索。
func requestHit(it model.RequestItem, q string) string {
	if q == "" {
		return hitNone
	}
	has := func(s string) bool { return strings.Contains(strings.ToLower(s), q) }
	kvHas := func(kvs []model.KeyValue) bool {
		for _, kv := range kvs {
			if has(kv.Key) || has(kv.Value) {
				return true
			}
		}
		return false
	}
	switch {
	case has(it.Name):
		return hitName
	case has(it.URL):
		return hitURL
	case has(it.Method):
		return hitMethod
	case kvHas(it.Params):
		return hitParam
	case kvHas(it.Headers):
		return hitHeader
	}
	b := it.Body
	if has(b.JSON) || has(b.Raw) || kvHas(b.FormUrlEncoded) {
		return hitBody
	}
	for _, f := range b.FormData {
		if has(f.Key) || (f.Type != "file" && has(f.Value)) {
			return hitBody
		}
	}
	return hitNone
}

// hitLabel 是搜索结果行右侧的命中位置提示；名称、地址命中是显而易见的，不加提示。
func (a *app) hitLabel(hit string) string {
	switch hit {
	case hitMethod:
		return a.t("方法", "Method")
	case hitParam:
		return a.t("参数", "Params")
	case hitHeader:
		return a.t("请求头", "Headers")
	case hitBody:
		return a.t("请求体", "Body")
	}
	return ""
}

// searchHit 返回侧栏节点 n 在搜索词 q 下的命中位置。
func (a *app) searchHit(n treeNode, q string) string {
	if strings.Contains(strings.ToLower(n.name), q) {
		return hitName
	}
	if it, ok := a.requestItem(n.id); ok {
		return requestHit(it, q)
	}
	return hitNone
}
