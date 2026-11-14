package gui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/pkg/logger"
)

// draftFor 返回请求 ID 对应的草稿：已打开标签用其最新内容（含未保存修改），否则从集合里读取。
func (a *app) draftFor(id string) *client.Draft {
	if i := a.findTab(id); i >= 0 {
		t := a.tabs[i]
		t.flush()
		return t.draft
	}
	if it, ok := a.requestItem(id); ok {
		return client.DraftFromItem(it)
	}
	return nil
}

// curlOf 生成请求的 cURL 命令（变量已按当前环境展开）。
func (a *app) curlOf(id string) (string, bool) {
	d := a.draftFor(id)
	if d == nil {
		return "", false
	}
	cmd, _ := client.RuntimeCurl(d, a.vars(), a.set, a.jar)
	return cmd, true
}

// copyCurlOf 把请求的 cURL 命令复制到剪贴板。
func (a *app) copyCurlOf(id string) {
	cmd, ok := a.curlOf(id)
	if !ok {
		return
	}
	a.writeClipboard(cmd)
	a.toast(a.t("已复制 cURL 命令", "cURL command copied"))
	logger.Info("已复制 cURL", "requestID", id)
}

// exportCurlOf 把请求的 cURL 命令导出为 .sh 文件。
func (a *app) exportCurlOf(id string) {
	cmd, ok := a.curlOf(id)
	if !ok {
		return
	}
	name := a.tree.nodes[id].name
	safe := strings.TrimSpace(unsafeFileChars.ReplaceAllString(name, "-"))
	if safe == "" {
		safe = id
	}
	a.async(func() func() {
		err := a.be.SaveTextFile(a.win, safe+".curl.sh", cmd+"\n")
		if err == nil {
			logger.Info("已导出 cURL", "requestID", id)
		}
		return func() { a.fail(a.t("导出 cURL", "Export cURL"), err) }
	})
}

// copyText 复制一段文本并提示。
func (a *app) copyText(s, tip string) {
	a.writeClipboard(s)
	a.toast(tip)
}

// historyMenu 是历史记录行的右键菜单。
func (a *app) historyMenu(m *ui.Menu, e historyItem) {
	if m.Item(a.t("打开", "Open")).Chosen() {
		a.later(func() { a.openFromHistory(e) })
	}
	if m.Item(a.t("复制地址", "Copy URL")).Disabled(e.URL == "").Chosen() {
		a.later(func() { a.copyText(e.URL, a.t("已复制地址", "URL copied")) })
	}
	m.Separator()
	if m.Item(a.t("删除此记录", "Delete this entry")).Danger().Chosen() {
		a.later(func() { a.deleteHistoryEntry(e) })
	}
}

// deleteHistoryEntry 删除单条历史记录并刷新列表。
func (a *app) deleteHistoryEntry(e historyItem) {
	if err := a.be.DeleteHistoryEntry(a.projectID, e.ID); err != nil {
		a.fail(a.t("删除历史记录", "Delete history entry"), err)
		return
	}
	a.reloadHistory()
	logger.Info("已删除历史记录", "entryID", e.ID)
}

// historyMatch 判断历史记录是否匹配搜索词：方法、地址、名称、状态码都可以搜，
// 也支持 2xx / 4xx / 5xx 这样的状态段。
func historyMatch(e historyItem, q string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return true
	}
	if len(q) == 3 && q[1:] == "xx" && q[0] >= '1' && q[0] <= '5' {
		return e.StatusCode/100 == int(q[0]-'0')
	}
	for _, f := range []string{e.Method, e.URL, e.Name, strconv.Itoa(e.StatusCode)} {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

// sidebarEnvMenu 是环境行的右键菜单：编辑、设为当前/取消当前、删除。
func (a *app) sidebarEnvMenu(m *ui.Menu, id, name string) {
	if m.Item(a.t("编辑", "Edit")).Chosen() {
		a.later(func() { a.openEnvManagerFor(id) })
	}
	if a.envID == id {
		if m.Item(a.t("取消使用", "Deactivate")).Chosen() {
			a.later(func() { a.setEnv("") })
		}
	} else if m.Item(a.t("设为当前环境", "Use this environment")).Chosen() {
		a.later(func() { a.setEnv(id) })
	}
	m.Separator()
	if m.Item(a.t("删除", "Delete")).Danger().Chosen() {
		a.later(func() { a.deleteEnvironment(id, name) })
	}
}
