package gui

import (
	"github.com/egoist/mygo/ui/editor"

	"minipost/internal/client"
	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// bodyEditor 是请求体（JSON / 原始文本）使用的代码编辑器及其与草稿的同步状态。
type bodyEditor struct {
	st     *editor.State
	synced uint64
}

func newBodyEditor(text, filename string) *bodyEditor {
	buf := editor.NewBuffer(text)
	if lang := editor.LanguageFor(filename); lang != nil {
		buf.SetLanguage(lang)
	}
	return &bodyEditor{st: &editor.State{Buffer: buf, SoftWrap: true, Placeholder: ""}, synced: buf.Version()}
}

// pull 在编辑器内容变化时写回草稿字段，返回是否有变化。
func (b *bodyEditor) pull(dst *string) bool {
	if v := b.st.Buffer.Version(); v != b.synced {
		b.synced = v
		*dst = b.st.Buffer.String()
		return true
	}
	return false
}

// push 用新内容整体替换编辑器内容，不进入撤销历史；草稿会在下一次 pull 时跟上。
func (b *bodyEditor) push(text string) {
	b.st.Buffer.Reset(text)
}

// tab 是一个打开的请求标签页。
type tab struct {
	draft *client.Draft
	base  *client.Draft // 最近一次保存（或载入）时的内容，用于判断是否有未保存修改
	dirty bool
	sub   int // 请求区子标签：0 Params，1 Headers，2 Body，3 Auth

	autoOpen bool // Headers 页的自动请求头列表是否展开

	jsonEd *bodyEditor
	rawEd  *bodyEditor

	resp     *respState
	sending  bool
	sendID   string
	urlCache string // 上一次与 Params 同步过的 URL

	env *envEdit // 非空表示这是环境编辑页标签，而不是请求标签

	editName      bool   // 面包屑里的请求名是否处于编辑状态
	nameBuf       string // 请求名编辑缓冲
	downloadAfter bool   // 本次发送完成后保存响应到文件（Send and Download）
	unsaved       bool   // 尚未写入项目的草稿标签（cURL 导入），首次保存时才创建请求
}

func newTab(d *client.Draft, base *client.Draft) *tab {
	t := &tab{draft: d, base: base, urlCache: d.URL}
	t.jsonEd = newBodyEditor(d.Body.JSON, "body.json")
	t.rawEd = newBodyEditor(d.Body.Raw, "body.txt")
	t.resp = newRespState()
	t.refreshDirty()
	return t
}

// flush 把编辑器里的内容同步到草稿。
func (t *tab) flush() {
	t.jsonEd.pull(&t.draft.Body.JSON)
	t.rawEd.pull(&t.draft.Body.Raw)
}

func (t *tab) refreshDirty() {
	if t.env != nil {
		t.dirty = t.env.dirty()
		return
	}
	t.flush()
	t.dirty = t.draft.Fingerprint() != t.base.Fingerprint()
}

// title 返回标签显示名。
func (t *tab) title() string {
	if t.env != nil {
		return t.env.env.Name
	}
	if t.draft.Name == "" {
		return "Untitled"
	}
	return t.draft.Name
}

// find 返回打开了指定请求的标签下标，没有则为 -1。
func (a *app) findTab(requestID string) int {
	for i, t := range a.tabs {
		if t.draft.ID == requestID {
			return i
		}
	}
	return -1
}

func (a *app) activeTab() *tab {
	if a.active >= 0 && a.active < len(a.tabs) {
		return a.tabs[a.active]
	}
	return nil
}

func (a *app) requestItem(id string) (model.RequestItem, bool) {
	for _, r := range a.coll.Requests {
		if r.ID == id {
			return r, true
		}
	}
	return model.RequestItem{}, false
}

// openRequest 打开请求：已打开则切换，否则新建标签。
func (a *app) openRequest(id string) {
	if i := a.findTab(id); i >= 0 {
		a.activate(i)
		return
	}
	it, ok := a.requestItem(id)
	if !ok {
		return
	}
	d := client.DraftFromItem(it)
	a.tabs = append(a.tabs, newTab(d, d.Clone()))
	a.activate(len(a.tabs) - 1)
	logger.Info("已打开请求", "requestID", id)
}

func (a *app) activate(i int) {
	a.active = i
	a.syncSel = true
	if a.tabs[i].env != nil {
		return
	}
	a.navRecord(a.tabs[i].draft.ID)
	for p := a.tree.nodes[a.tabs[i].draft.ID].parent; p != ""; p = a.tree.nodes[p].parent {
		a.outline.Open.Add(p)
	}
}

// saveTab 保存标签到磁盘。
func (a *app) saveTab(t *tab) bool {
	if t.env != nil {
		return a.saveEnvTab(t)
	}
	t.flush()
	if t.unsaved {
		if err := a.persistDraft(t); err != nil {
			a.fail(a.t("保存请求", "Save request"), err)
			return false
		}
	}
	if err := a.be.SaveRequestItem(t.draft.ToItem()); err != nil {
		a.fail(a.t("保存请求", "Save request"), err)
		return false
	}
	t.base = t.draft.Clone()
	t.dirty = false
	a.reloadCollection()
	logger.Info("请求已保存", "requestID", t.draft.ID)
	return true
}

// closeTab 关闭标签，必要时先询问是否保存。
func (a *app) closeTab(i int) {
	if i < 0 || i >= len(a.tabs) {
		return
	}
	t := a.tabs[i]
	t.refreshDirty()
	switch {
	case !t.dirty || a.set.AlwaysDiscardUnsaved:
		a.removeTab(i)
	case a.set.AlwaysSaveUnsaved:
		if a.saveTab(t) {
			a.removeTab(i)
		}
	default:
		a.overlays.askClose(i, t.title())
	}
}

func (a *app) removeTab(i int) {
	if t := a.tabs[i]; t.sending {
		a.be.CancelHTTPRequest(t.sendID)
	}
	a.tabs = append(a.tabs[:i], a.tabs[i+1:]...)
	switch {
	case a.active > i, a.active >= len(a.tabs):
		a.active = max(0, a.active-1)
	}
	a.syncSel = true
}

// closeTabsFor 关闭属于给定请求集合的所有标签（请求被删除时使用，不再询问）。
func (a *app) closeTabsFor(ids []string) {
	gone := map[string]bool{}
	for _, id := range ids {
		gone[id] = true
	}
	for i := len(a.tabs) - 1; i >= 0; i-- {
		if gone[a.tabs[i].draft.ID] {
			a.removeTab(i)
		}
	}
}

// renameTab 在请求被重命名后同步已打开标签的名称。
func (a *app) renameTab(id, name string) {
	if i := a.findTab(id); i >= 0 {
		a.tabs[i].draft.Name = name
		a.tabs[i].base.Name = name
	}
}

// persistSession 保存当前项目的标签会话，含未保存的修改。
func (a *app) persistSession() {
	if a.projectID == "" {
		return
	}
	s := a.sessions.Load()
	ps := client.ProjectSession{}
	for i, t := range a.tabs {
		if t.env != nil || t.unsaved {
			continue // 环境编辑页与未保存草稿不进入会话
		}
		if i == a.active {
			ps.Active = len(ps.Tabs)
		}
		t.refreshDirty()
		st := client.SavedTab{RequestID: t.draft.ID}
		if t.dirty {
			it := t.draft.ToItem()
			st.Unsaved = &it
		}
		ps.Tabs = append(ps.Tabs, st)
	}
	s.Projects[a.projectID] = ps
	if err := a.sessions.Save(s); err != nil {
		logger.Error("保存会话失败", "error", err.Error())
	}
}

// restoreSession 恢复项目上次打开的标签。
func (a *app) restoreSession() {
	ps, ok := a.sessions.Load().Projects[a.projectID]
	if !ok {
		return
	}
	for _, st := range ps.Tabs {
		it, ok := a.requestItem(st.RequestID)
		if !ok {
			continue
		}
		base := client.DraftFromItem(it)
		d := base.Clone()
		if st.Unsaved != nil {
			d = client.DraftFromItem(*st.Unsaved)
		}
		a.tabs = append(a.tabs, newTab(d, base))
	}
	if ps.Active >= 0 && ps.Active < len(a.tabs) {
		a.active = ps.Active
	}
}
