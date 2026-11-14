package gui

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/model"
)

// 设计稿 3A：树/历史/标签里的方法用缩写（DELETE→DEL、OPTIONS→OPT），其余原样。
func TestMethodAbbr(t *testing.T) {
	cases := map[string]string{
		"GET": "GET", "POST": "POST", "PUT": "PUT", "PATCH": "PATCH", "HEAD": "HEAD",
		"DELETE": "DEL", "OPTIONS": "OPT", "delete": "DEL",
	}
	for in, want := range cases {
		if got := methodAbbr(in); got != want {
			t.Errorf("methodAbbr(%q)=%q，应为 %q", in, got, want)
		}
	}
}

// 设计稿 3A：侧栏里所有请求的名称从同一列开始（方法列固定 32px），不论方法缩写长短。
func TestSidebarMethodColumnAligned(t *testing.T) {
	a, tt, be := zhApp(t)
	cd := be.colls[a.projectID]
	for i, m := range []string{"POST", "PATCH", "DELETE", "OPTIONS"} {
		rid := be.id("r")
		cd.Requests = append(cd.Requests, model.RequestItem{ID: rid, Name: "mm" + m, Method: m, URL: "https://example.com/" + m, ProjectID: a.projectID, Body: model.RequestBody{Type: "none"}, Auth: model.AuthConfig{Type: "none"}})
		cd.TreeNodes = append(cd.TreeNodes, model.CollectionNode{NodeID: rid, NodeType: model.CollectionNodeTypeRequest, ProjectID: a.projectID, SortOrder: 10 + i})
	}
	a.reloadCollection()
	tt.Frame()
	x0 := -1.0
	for _, name := range []string{"alpha", "mmPOST", "mmPATCH", "mmDELETE", "mmOPTIONS"} {
		r, ok := tt.Find(name)
		if !ok {
			t.Fatalf("侧栏缺少请求 %q：%q", name, tt.Texts())
		}
		if x0 < 0 {
			x0 = float64(r.X)
		}
		if float64(r.X) != x0 {
			t.Errorf("%q 名称起点 X=%.1f，应与第一行 %.1f 对齐", name, r.X, x0)
		}
	}
	for _, abbr := range []string{"DEL", "OPT", "PATCH"} {
		if _, ok := tt.Find(abbr); !ok {
			t.Errorf("侧栏应显示方法缩写 %q：%q", abbr, tt.Texts())
		}
	}
	if _, ok := tt.Find("DELETE"); ok {
		t.Errorf("侧栏不应再显示完整的 DELETE")
	}
}

// 设计稿 2A：方法、地址、发送合成一个控件——发送按钮在地址栏边框内，且按钮比地址栏矮（四周留白）。
func TestURLBarIntegratedSend(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	send, ok := tt.Find("发送")
	if !ok {
		t.Fatalf("地址栏缺少“发送”按钮：%q", tt.Texts())
	}
	params, ok := tt.Find("Params")
	if !ok {
		t.Fatal("缺少 Params 标签")
	}
	// 发送在地址栏这一行：位于参数标签之上，且与标签相距不超过一个地址栏高度。
	if gap := params.Y - (send.Y + send.H); gap < 0 || gap > sizeURLBar {
		t.Fatalf("发送按钮应在地址栏行内（紧邻 Params 标签上方），间距 %.1f", gap)
	}
	// 地址栏右边缘留给窗口边距 12px：发送按钮在其内部，右侧不会贴着窗口边。
	if send.X+send.W > float32(1200-12) {
		t.Fatalf("发送按钮不应超出地址栏边框：右缘 %.0f", send.X+send.W)
	}
	if sizeURLBar != 34 {
		t.Fatalf("设计稿地址栏高度为 34，实际 %d", sizeURLBar)
	}
}

// 设计稿 4A：响应状态条是“状态胶囊 + 灰色度量”；状态说明已含状态码时不重复。
func TestRespStatusLabel(t *testing.T) {
	cases := []struct {
		code int
		text string
		want string
	}{
		{200, "OK", "200 OK"},
		{200, "200 OK", "200 OK"},
		{404, "Not Found", "404 Not Found"},
		{204, "", "204"},
	}
	for _, c := range cases {
		if got := respStatusLabel(c.code, c.text); got != c.want {
			t.Errorf("respStatusLabel(%d,%q)=%q，应为 %q", c.code, c.text, got, c.want)
		}
	}
}

// 响应返回后，状态条里显示“200 OK”胶囊、耗时与大小。
func TestRespMetaPill(t *testing.T) {
	_, tt, _ := zhApp(t)
	if err := tt.Click("alpha"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("200 OK"); !ok {
		t.Fatalf("响应状态条应显示“200 OK”：%q", tt.Texts())
	}
}

// 设计稿：侧栏顶部是分段控件，选中项是内容色凸起块，切换后选中项随之变化。
func TestSideSegmentedTabs(t *testing.T) {
	a, tt, _ := zhApp(t)
	for i, label := range []string{"请求", "历史", "环境"} {
		if err := tt.Click(label); err != nil {
			t.Fatalf("点击侧栏标签 %q：%v", label, err)
		}
		tt.Frame()
		if a.side.tab != i {
			t.Fatalf("点击 %q 后 side.tab=%d，应为 %d", label, a.side.tab, i)
		}
	}
	// 选中块使用内容底色（surface），轨道使用 hover 层。
	if err := tt.Click("请求"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	r, ok := tt.Find("请求")
	if !ok {
		t.Fatal("找不到“请求”标签")
	}
	img := tt.Image()
	// 取选中块左缘内侧、文字之外的一个点
	got := px(img, int(r.X)-10, int(r.Y+r.H/2))
	if got != rgb(a.p.surface) {
		t.Fatalf("选中分段应为内容底色 %v，实际 %v", rgb(a.p.surface), got)
	}
}

// 设计稿：侧栏底部显示“N 个请求 · M 个文件夹”。
func TestSideFooterCounts(t *testing.T) {
	a, tt, _ := zhApp(t)
	reqs, folders := a.treeCounts()
	if reqs != 2 || folders != 0 {
		t.Fatalf("统计应为 2 个请求 0 个文件夹，实际 %d/%d", reqs, folders)
	}
	if _, ok := tt.Find("2 个请求 · 0 个文件夹"); !ok {
		t.Fatalf("侧栏底部应显示统计行：%q", tt.Texts())
	}
}

// ⌘K 打开快速打开面板；输入搜索词后只显示匹配的请求，回车打开选中的请求并关闭面板。
func TestQuickOpenSearchAndOpen(t *testing.T) {
	a, tt, _ := zhApp(t)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Frame()
	tt.Frame()
	if !a.overlays.quick.open {
		t.Fatal("⌘K 应打开快速打开面板")
	}
	if _, ok := tt.Find("请求"); !ok {
		t.Fatalf("面板应有“请求”分组：%q", tt.Texts())
	}
	snapshot(t, tt, "d_quick_empty")
	tt.Type("beta")
	tt.Frame()
	snapshot(t, tt, "d_quick_beta")
	items := a.quickResults(a.overlays.quick.query)
	nReq := 0
	for _, it := range items {
		if it.kind == quickRequest {
			nReq++
			if it.label != "beta" {
				t.Fatalf("搜索 beta 只应匹配 beta，实际 %q", it.label)
			}
		}
	}
	if nReq != 1 {
		t.Fatalf("搜索 beta 应有 1 条请求结果，实际 %d", nReq)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if a.overlays.quick.open {
		t.Fatal("回车执行后面板应关闭")
	}
	if tab := a.activeTab(); tab == nil || tab.draft.Name != "beta" {
		t.Fatalf("应打开请求 beta，实际 %+v", a.activeTab())
	}
}

// 快速打开：↑↓ 移动高亮；环境和命令都可被搜索到；无匹配时给出提示。
func TestQuickOpenEnvCommandAndEmpty(t *testing.T) {
	a, tt, be := zhApp(t)
	be.envs[a.projectID] = []model.Environment{{ID: "e1", Name: "预发布", ProjectID: a.projectID}}
	a.reloadEnvs()

	a.openQuick()
	tt.Frame()
	// 环境按名称搜索，执行后切换到该环境
	a.overlays.quick.query = "预发布"
	items := a.quickResults("预发布")
	if len(items) != 1 || items[0].kind != quickEnv || !strings.Contains(items[0].label, "预发布") {
		t.Fatalf("搜索“预发布”应只命中环境，实际 %+v", items)
	}
	items[0].run()
	if a.envID != "e1" {
		t.Fatalf("执行环境结果后应切换到 e1，实际 %q", a.envID)
	}
	// 命令可搜索
	cmds := a.quickResults("cURL")
	found := false
	for _, it := range cmds {
		found = found || (it.kind == quickCommand && strings.Contains(it.label, "cURL"))
	}
	if !found {
		t.Fatalf("搜索 cURL 应命中命令：%+v", cmds)
	}
	// 无匹配
	a.overlays.quick.query = "zzzz-不存在"
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("没有匹配的结果"); !ok {
		t.Fatalf("无匹配时应提示：%q", tt.Texts())
	}
	// ↑↓ 在结果间循环
	a.overlays.quick.query = ""
	a.overlays.quick.sel = 0
	tt.Frame()
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	if want := len(a.quickResults("")) - 1; a.overlays.quick.sel != want {
		t.Fatalf("在首行按 ↑ 应回到末行 %d，实际 %d", want, a.overlays.quick.sel)
	}
}

// 在主页（未进入项目）或已有对话框时 ⌘K 不应打开面板。
func TestQuickOpenBlockedWhenModalOrHome(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openPrefs()
	tt.Frame()
	a.openQuick()
	if a.overlays.quick.open {
		t.Fatal("设置对话框打开时不应再打开快速打开面板")
	}
	a.overlays.prefs.open = false
	a.showHome = true
	a.openQuick()
	if a.overlays.quick.open {
		t.Fatal("在项目主页时不应打开快速打开面板")
	}
}

// homeApp 构造带三个项目的主页场景。
func homeApp(t *testing.T) (*app, *ui.Tester, *fakeBackend) {
	t.Helper()
	be := newFakeBackend(t)
	be.seedProject("示例项目", "a1", "a2")
	be.seedProject("订单服务", "o1")
	be.seedProject("支付网关")
	a := newApp(be)
	a.set.Locale = localeZH
	a.showHome = true
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	return a, tt, be
}

// 设计稿：项目主页有标题、搜索与总数、排序，右侧详情面板有主题色、实时预览与操作按钮。
func TestHomeLayoutMatchesDesign(t *testing.T) {
	_, tt, _ := homeApp(t)
	for _, want := range []string{
		"项目主页", "新建项目", "共 3 个项目", "排序", "最近更新", "项目名称", "主题色", "实时预览",
		"打开项目", "重命名", "导出", "2 个请求 · 0 个文件夹", "1 个请求 · 0 个文件夹",
	} {
		if !tt.HasText(want) {
			t.Errorf("项目主页缺少 %q：%q", want, tt.Texts())
		}
	}
	snapshot(t, tt, "d_home")
}

// 点击卡片选中项目（右侧详情随之切换），“打开项目”进入所选项目。
func TestHomeSelectAndOpen(t *testing.T) {
	a, tt, be := homeApp(t)
	if err := tt.Click("订单服务"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.homeSel != be.projects[1].ID {
		t.Fatalf("点击卡片应选中订单服务，实际 %q", a.homeSel)
	}
	if err := tt.Click("打开项目"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.showHome || a.projectID != be.projects[1].ID {
		t.Fatalf("打开项目后应进入订单服务：showHome=%v projectID=%q", a.showHome, a.projectID)
	}
}

// 搜索词过滤项目，排序方式切换为名称后按名称升序。
func TestHomeSearchAndSort(t *testing.T) {
	a, tt, _ := homeApp(t)
	a.homeQuery = "订单"
	tt.Frame()
	if got := a.homeProjects(); len(got) != 1 || got[0].Name != "订单服务" {
		t.Fatalf("搜索“订单”应只剩订单服务，实际 %+v", got)
	}
	if !tt.HasText("共 1 个项目") {
		t.Fatalf("总数应随搜索更新：%q", tt.Texts())
	}
	a.homeQuery, a.homeSort = "", "name"
	got := a.homeProjects()
	for i := 1; i < len(got); i++ {
		if strings.ToLower(got[i-1].Name) > strings.ToLower(got[i].Name) {
			t.Fatalf("按名称排序应升序：%+v", got)
		}
	}
}

// 详情面板改名：提交后后端项目名更新，空名称与未变化不提交。
func TestHomeRenameCommit(t *testing.T) {
	a, _, be := homeApp(t)
	id := be.projects[2].ID
	a.commitProjectName(id, "")
	a.commitProjectName(id, "支付网关")
	if be.projects[2].Name != "支付网关" {
		t.Fatalf("空名称/未变化不应改名：%q", be.projects[2].Name)
	}
	a.commitProjectName(id, "支付中心")
	if be.projects[2].Name != "支付中心" {
		t.Fatalf("应改名为支付中心，实际 %q", be.projects[2].Name)
	}
}

// 相对时间文案：刚刚 / 分钟 / 小时 / 昨天 / 天 / 上周 / 日期；无法解析显示 --。
func TestRelTime(t *testing.T) {
	a, _, _ := zhApp(t)
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) string { return now.Add(-d).Format(time.RFC3339) }
	cases := []struct {
		in   string
		want string
	}{
		{ago(10 * time.Second), "刚刚"},
		{ago(5 * time.Minute), "5 分钟前"},
		{ago(2 * time.Hour), "2 小时前"},
		{ago(30 * time.Hour), "昨天"},
		{ago(3 * 24 * time.Hour), "3 天前"},
		{ago(9 * 24 * time.Hour), "上周"},
		{"", "--"},
	}
	for _, c := range cases {
		if got := a.relTime(c.in, now); got != c.want {
			t.Errorf("relTime(%q)=%q，应为 %q", c.in, got, c.want)
		}
	}
	if projectInitial("  订单") != "订" || projectInitial("abc") != "A" || projectInitial("") != "?" {
		t.Error("projectInitial 取首字不正确")
	}
}

// 设计稿：欢迎页有四个快捷入口，点击分别触发对应操作；没有历史时不显示“最近打开”。
func TestWelcomePageActions(t *testing.T) {
	a, tt, _ := zhApp(t)
	for _, want := range []string{"新建请求", "导入 Postman / OpenAPI / cURL", "粘贴 cURL 命令", "快速打开请求、环境、命令", "⌘ N", "⌘ K"} {
		if !tt.HasText(want) {
			t.Errorf("欢迎页缺少 %q：%q", want, tt.Texts())
		}
	}
	if tt.HasText("最近打开") {
		t.Error("没有历史记录时不应显示“最近打开”")
	}
	if err := tt.Click("粘贴 cURL 命令"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.overlays.curl.open {
		t.Fatal("点击“粘贴 cURL 命令”应打开 cURL 对话框")
	}
	a.overlays.curl.open = false
	tt.Frame()
	if err := tt.Click("快速打开请求、环境、命令"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.overlays.quick.open {
		t.Fatal("点击“快速打开”应打开快速打开面板")
	}
}

// 有历史记录时，欢迎页列出“最近打开”，点击在标签里打开对应请求。
func TestWelcomeRecentOpens(t *testing.T) {
	a, tt, be := visualApp(t, false)
	be.history[a.projectID][0].RequestID = be.colls[a.projectID].Requests[0].ID
	a.reloadHistory()
	tt.Frame()
	snapshot(t, tt, "d_welcome")
	if !tt.HasText("最近打开") {
		t.Fatalf("有历史记录时应显示“最近打开”：%q", tt.Texts())
	}
	if len(a.tabs) != 0 {
		t.Fatalf("初始不应有标签，实际 %d", len(a.tabs))
	}
	if err := tt.Click("获取用户列表"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.tabs) != 1 || a.tabs[0].draft.Name != "获取用户列表" {
		t.Fatalf("点击最近打开应在标签里打开该请求，实际 %d 个标签", len(a.tabs))
	}
}

// 设计稿：环境页有标题与说明、设为当前/保存按钮、变量表与右侧动态变量面板。
func TestEnvPageMatchesDesign(t *testing.T) {
	a, tt, _ := visualApp(t, false)
	a.openEnvManagerFor("e1")
	tt.Frame()
	tt.Frame()
	for _, want := range []string{
		"设为当前环境", "保存", "变量名", "值", "动态变量", "发送时求值",
		"{{$timestamp}}", "{{$randomUUID}}", "当前时间，秒", "随机 UUID v4",
		"带锁的变量会在界面上遮罩，且不会写入日志。",
	} {
		if !tt.HasText(want) {
			t.Errorf("环境页缺少 %q：%q", want, tt.Texts())
		}
	}
	if tt.HasText("当前使用") {
		t.Fatal("未设为当前环境时不应显示“当前使用”")
	}
	snapshot(t, tt, "d_env")
	if err := tt.Click("设为当前环境"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if a.envID != "e1" {
		t.Fatalf("点击后应切换到 e1，实际 %q", a.envID)
	}
	if !tt.HasText("当前使用") {
		t.Fatalf("设为当前后应显示“当前使用”：%q", tt.Texts())
	}
}

// 环境页：改名后出现未保存状态；变量的密文锁可切换；保存后写回后端。
func TestEnvPageSecretToggleAndSave(t *testing.T) {
	a, tt, be := visualApp(t, false)
	a.openEnvManagerFor("e1")
	tt.Frame()
	tab := a.activeTab()
	if tab == nil || tab.env == nil {
		t.Fatal("应打开环境页")
	}
	ed := tab.env
	if ed.dirty() {
		t.Fatal("刚打开的环境页不应有未保存修改")
	}
	if len(ed.samples) != 5 {
		t.Fatalf("应有 5 个动态变量示例值，实际 %d", len(ed.samples))
	}
	ed.env.Variables[0].IsSecret = true
	if !ed.dirty() {
		t.Fatal("切换密文标记后应有未保存修改")
	}
	if !a.saveEnvTab(tab) {
		t.Fatal("保存应成功")
	}
	got := be.envs[a.projectID][0]
	if len(got.Variables) == 0 || !got.Variables[0].IsSecret {
		t.Fatalf("保存后后端应记录密文标记：%+v", got.Variables)
	}
	if countEnvVars(got) != 1 {
		t.Fatalf("countEnvVars 应为 1，实际 %d", countEnvVars(got))
	}
}

// 环境侧栏：当前环境显示“当前”标记与变量数；底部有说明。
func TestEnvSidebarListShowsActiveAndCount(t *testing.T) {
	a, tt, _ := visualApp(t, false)
	a.setEnv("e1")
	a.side.tab = 2
	tt.Frame()
	for _, want := range []string{"开发环境", "当前", "1", "切换环境后，请求里的 {{变量名}} 会立即换成对应值。"} {
		if !tt.HasText(want) {
			t.Errorf("环境侧栏缺少 %q：%q", want, tt.Texts())
		}
	}
}

// 历史行只显示路径和查询（设计稿：/post、/get?page=1）；带变量或无法解析的地址原样显示。
func TestURLPathQuery(t *testing.T) {
	cases := map[string]string{
		"https://httpbin.org/post":           "/post",
		"https://httpbin.org/get?page=1&x=2": "/get?page=1&x=2",
		"https://httpbin.org":                "/",
		"{{baseUrl}}/post":                   "{{baseUrl}}/post",
		"not a url":                          "not a url",
		"":                                   "",
		"http://localhost:8080/a/b%20c?q=中文": "/a/b%20c?q=中文",
	}
	for in, want := range cases {
		if got := urlPathQuery(in); got != want {
			t.Errorf("urlPathQuery(%q)=%q，应为 %q", in, got, want)
		}
	}
}

// 控制台筛选：全部 / 成功 / 错误；错误包含请求失败与 ≥400 的响应，并显示错误条数。
func TestConsoleFilter(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.con.entries = []consoleEntry{
		{method: "GET", url: "https://x/ok", status: 200, ms: 10, done: true},
		{method: "GET", url: "https://x/401", status: 401, ms: 10, done: true},
		{method: "GET", url: "https://x/dns", err: "no such host", done: true},
		{method: "POST", url: "https://x/pending"},
	}
	a.setConsoleOpen(true)
	tt.Frame()
	if idx, errs := a.conVisible(); len(idx) != 4 || errs != 2 {
		t.Fatalf("全部应 4 条且 2 条错误，实际 %d/%d", len(idx), errs)
	}
	for _, want := range []string{"全部", "成功", "错误", "4 条"} {
		if !tt.HasText(want) {
			t.Errorf("控制台缺少 %q：%q", want, tt.Texts())
		}
	}
	if err := tt.Click("成功"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if idx, _ := a.conVisible(); len(idx) != 1 || a.con.entries[idx[0]].url != "https://x/ok" {
		t.Fatalf("成功只应剩 /ok，实际 %v", idx)
	}
	if err := tt.Click("错误"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if idx, _ := a.conVisible(); len(idx) != 2 {
		t.Fatalf("错误应有 2 条，实际 %v", idx)
	}
	a.con.entries = a.con.entries[:1]
	tt.Frame()
	if !tt.HasText("没有符合筛选的日志") {
		t.Fatalf("筛选后无结果时应提示：%q", tt.Texts())
	}
}

// Auth 页预览文本：密钥一律用圆点遮盖，API Key 区分请求头与查询参数。
func TestAuthPreview(t *testing.T) {
	cases := []struct {
		au   client.AuthDraft
		want string
	}{
		{client.AuthDraft{Type: client.AuthNone}, ""},
		{client.AuthDraft{Type: client.AuthBasic}, "Authorization: Basic •••••"},
		{client.AuthDraft{Type: client.AuthBearer, Token: "secret"}, "Authorization: Bearer •••••"},
		{client.AuthDraft{Type: client.AuthAPIKey, APIKeyName: "X-Key"}, "X-Key: •••••"},
		{client.AuthDraft{Type: client.AuthAPIKey, APIKeyName: "k", APIKeyLocation: "query"}, "?k=•••••"},
	}
	for _, c := range cases {
		got := authPreview(&c.au)
		if got != c.want {
			t.Errorf("authPreview(%v)=%q，应为 %q", c.au.Type, got, c.want)
		}
		if strings.Contains(got, "secret") {
			t.Errorf("预览不应泄露密钥：%q", got)
		}
	}
}

// Auth 页：类型下拉显示当前类型，选 Bearer 后出现 Token 行与请求头预览；Auth 标签带锁。
func TestAuthTabDropdown(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	if err := tt.Click("Auth"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, want := range []string{"认证类型", "None"} {
		if !tt.HasText(want) {
			t.Fatalf("Auth 页缺少 %q：%q", want, tt.Texts())
		}
	}
	tab := a.activeTab()
	tab.draft.Auth.Type = client.AuthBearer
	tab.draft.Auth.Token = "abc"
	tt.Frame()
	for _, want := range []string{"Bearer Token", "Token", "将自动添加请求头", "Authorization: Bearer •••••"} {
		if !tt.HasText(want) {
			t.Fatalf("Bearer 配置后缺少 %q：%q", want, tt.Texts())
		}
	}
	snapshot(t, tt, "auth_bearer")
}

// Body 标签显示类型徽标；格式化按钮为中文。
func TestBodyTabBadgeAndBeautify(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.activeTab().draft.Body.Type = client.BodyJSON
	tt.Frame()
	if !tt.HasText("JSON") {
		t.Fatalf("Body 标签应带 JSON 徽标：%q", tt.Texts())
	}
	if err := tt.Click("Body"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("格式化") || !tt.HasText("urlencoded") {
		t.Fatalf("Body 页应有 格式化 与 urlencoded：%q", tt.Texts())
	}
}

// 响应头表格：行尾有“复制值”按钮，点击把该行的值放进剪贴板。
func TestRespHeadersCopyValue(t *testing.T) {
	a, tt, _, rs := sendWith(t, jsonResp())
	rs.sub = 1
	tt.Frame()
	snapshot(t, tt, "rb_headers")
	if err := tt.Click("复制值"); err != nil {
		t.Fatal(err)
	}
	if a.clipboard == "" {
		t.Fatal("点击复制值后剪贴板不应为空")
	}
}

// 底栏摘要（设计稿 p24）：方法 + 路径 → 状态 · 耗时；底栏不再重复显示环境。
func TestBottomBarRequestSummary(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.con.entries = []consoleEntry{{method: "DELETE", url: "https://api.x.dev/users/1?soft=1", status: 204, ms: 41, done: true}}
	tt.Frame()
	for _, want := range []string{"DEL", "/users/1?soft=1", "→", "204 · 41ms"} {
		if !tt.HasText(want) {
			t.Errorf("底栏应显示 %q：%q", want, tt.Texts())
		}
	}
	if _, ok := tt.Find("当前环境（点击管理环境）"); ok {
		t.Error("环境选择已移到标题栏，底栏不应再有环境按钮")
	}
}

// 设置·通用（设计稿 16）：分组小标题、下拉选择、页脚说明；“恢复默认”只重置通用项。
func TestPrefsGeneralDesignAndReset(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.set.RequestTimeoutMs, a.set.MaxResponseSizeMB = 5000, 7
	a.set.SSLVerify, a.set.DisableCookies = false, true
	a.set.Theme, a.set.FontSize = "dark", 14 // 不属于通用页，不应被重置
	a.openPrefs()
	tt.Frame()
	for _, want := range []string{"请求", "安全与网络", "标签页", "HTTP 版本", "自动", "修改会立即生效并自动保存", "恢复默认"} {
		if !tt.HasText(want) {
			t.Fatalf("设置通用页缺少 %q：%q", want, tt.Texts())
		}
	}
	snapshot(t, tt, "d_prefs_general")
	if err := tt.Click("恢复默认"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	d := client.DefaultSettings()
	if a.set.RequestTimeoutMs != d.RequestTimeoutMs || a.set.MaxResponseSizeMB != d.MaxResponseSizeMB || !a.set.SSLVerify || a.set.DisableCookies {
		t.Fatalf("通用项应恢复默认，实际 %+v", a.set)
	}
	if a.set.Theme != "dark" || a.set.FontSize != 14 {
		t.Fatalf("恢复默认不应动外观设置：theme=%q font=%d", a.set.Theme, a.set.FontSize)
	}
}

// 设置·外观（设计稿 17）：主题与配色为缩略图卡片，字号 10–16 共 7 档；点卡片立即生效。
func TestPrefsAppearanceCards(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openPrefs()
	tt.Frame()
	if err := tt.Click("外观"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, want := range []string{"外观主题", "配色", "新增", "雾蓝", "墨白", "松石", "语言", "界面字体", "10", "15", "16"} {
		if !tt.HasText(want) {
			t.Fatalf("外观页缺少 %q：%q", want, tt.Texts())
		}
	}
	if err := tt.Click("15"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.set.FontSize != 15 {
		t.Fatalf("点击 15 应切换字号，实际 %d", a.set.FontSize)
	}
	if err := tt.Click("墨白"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.set.ColorScheme != "ink" {
		t.Fatalf("点击墨白应切换配色，实际 %q", a.set.ColorScheme)
	}
}
