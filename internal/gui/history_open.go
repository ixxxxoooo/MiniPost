package gui

import (
	"strings"

	"minipost/internal/client"
	"minipost/internal/pkg/logger"
)

// historyRequestID 找出历史记录对应的集合请求 ID。
// 新记录直接用 RequestID；旧版本写入的记录没有 RequestID，
// 退而求其次按“方法 + 解析变量后的地址”匹配集合里的请求。
// 参数：e 历史记录。返回：请求 ID，找不到时为空串。
//
// @author ixxxxoooo
func (a *app) historyRequestID(e historyItem) string {
	if e.RequestID != "" {
		if _, ok := a.requestItem(e.RequestID); ok {
			return e.RequestID
		}
		return ""
	}
	if strings.TrimSpace(e.URL) == "" {
		return ""
	}
	vars := a.vars()
	for _, r := range a.coll.Requests {
		if strings.EqualFold(r.Method, e.Method) && client.ResolveTemplate(r.URL, vars) == e.URL {
			return r.ID
		}
	}
	return ""
}

// openFromHistory 点击历史记录：对应请求仍存在则打开，否则提示。
// 参数：e 被点击的历史记录。
//
// @author ixxxxoooo
func (a *app) openFromHistory(e historyItem) {
	if id := a.historyRequestID(e); id != "" {
		a.openRequest(id)
		return
	}
	logger.Info("历史记录找不到对应请求", "entryID", e.ID, "hasRequestID", e.RequestID != "")
	a.toastWarn(a.t("该请求已被删除", "This request no longer exists"))
}
