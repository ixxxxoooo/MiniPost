package gui

import (
	"testing"

	"minipost/internal/client"
)

// spaceUnit 在数字与单位之间补一个空格（设计稿的「128 ms」「1.2 KB」写法），已有空格或没有单位时原样返回。
func TestSpaceUnit(t *testing.T) {
	for in, want := range map[string]string{
		"128ms":  "128 ms",
		"3.87s":  "3.87 s",
		"1.2KB":  "1.2 KB",
		"128 ms": "128 ms",
		"200":    "200",
		"—":      "—",
		"":       "",
	} {
		if got := spaceUnit(in); got != want {
			t.Errorf("spaceUnit(%q)=%q，应为 %q", in, got, want)
		}
	}
}

// 旧图标名都能经 iconAlias 解析到内置的设计稿图标，不会因缺文件而画不出来。
func TestIconAliasTargetsExist(t *testing.T) {
	for from, to := range iconAlias {
		if _, err := assetFS.ReadFile("icons/" + to + ".svg"); err != nil {
			t.Errorf("图标别名 %q → %q 指向的 icons/%s.svg 不存在: %v", from, to, to, err)
		}
		if svgIcon(from) == nil {
			t.Errorf("svgIcon(%q) 解析失败", from)
		}
	}
	// 设计稿新增、界面里直接使用的图标
	for _, name := range []string{"target", "checkBold", "terminal", "dockBottom", "dockRight", "cookie", "globe", "zap", "lock"} {
		if svgIcon(name) == nil {
			t.Errorf("缺少内置图标 %q", name)
		}
	}
}

// 项目卡片网格的列数：每列至少 236，间距 10，左右各留 26；窄到放不下一列时仍至少一列。
func TestHomeGridCols(t *testing.T) {
	for w, want := range map[float32]int{
		100:  1,
		288:  1, // 52 + 236
		533:  1,
		534:  2, // 52 + 236*2 + 10
		779:  2,
		780:  3, // 52 + 236*3 + 20
		1400: 5,
	} {
		if got := homeGridCols(w); got != want {
			t.Errorf("homeGridCols(%v)=%d，应为 %d", w, got, want)
		}
	}
}

// 控制台一行含大小列；点击行展开详情（状态、耗时、响应大小），再点收起。
func TestConsoleRowShowsSizeAndExpands(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.con.entries = []consoleEntry{
		{method: "GET", url: "https://x/ok", status: 200, ms: 128, size: 1229, done: true},
		{method: "GET", url: "https://x/dns", err: "no such host", done: true},
	}
	a.setConsoleOpen(true)
	tt.Frame()
	for _, want := range []string{"128 ms", "1.2 KB", "ERR", "—"} {
		if !tt.HasText(want) {
			t.Errorf("控制台行缺少 %q：%q", want, tt.Texts())
		}
	}
	if tt.HasText("响应大小: 1.2 KB") {
		t.Fatal("未点击时不应展开详情")
	}
	if err := tt.Click("https://x/ok"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.con.entries[0].open || !tt.HasText("响应大小: 1.2 KB") {
		t.Fatalf("点击后应展开详情：%q", tt.Texts())
	}
	if err := tt.Click("https://x/ok"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.con.entries[0].open {
		t.Fatal("再次点击应收起详情")
	}
}

// logResponse 把响应大小记进对应的日志行。
func TestLogResponseRecordsSize(t *testing.T) {
	a, _, _ := zhApp(t)
	a.logRequest("GET", "https://x/a")
	a.logResponse("GET", "https://x/a", 200, 12, 2048, "")
	e := a.con.entries[len(a.con.entries)-1]
	if !e.done || e.status != 200 || e.size != 2048 || e.ms != 12 {
		t.Fatalf("日志行未正确记录响应：%+v", e)
	}
}

// 控制台默认高度与设计稿一致（190）。
func TestConsoleDefaultHeightMatchesDesign(t *testing.T) {
	if client.ConsoleDefaultHeight != 190 {
		t.Fatalf("控制台默认高度应为 190，实际 %d", client.ConsoleDefaultHeight)
	}
}

// 设置·外观：五套配色卡都在，点「琥珀」立即切换并保存。
func TestPrefsFiveColorSchemesAndPickAmber(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openPrefs()
	a.overlays.prefs.section = 1
	tt.Frame()
	for _, want := range []string{"雾蓝", "墨白", "松石", "琥珀", "玫瑰", "新增 2 套"} {
		if !tt.HasText(want) {
			t.Fatalf("外观页缺少 %q：%q", want, tt.Texts())
		}
	}
	if err := tt.Click("琥珀"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.set.ColorScheme != "amber" {
		t.Fatalf("点击琥珀后应切换配色，实际 %q", a.set.ColorScheme)
	}
}

// 设置页的底栏在每一页都显示自动保存的说明，只有「通用」页带「恢复默认」。
func TestPrefsFooterOnEveryPage(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openPrefs()
	for i := range prefsSections {
		a.overlays.prefs.section = i
		tt.Frame()
		if !tt.HasText("修改会立即生效并自动保存") {
			t.Errorf("第 %d 页缺少底栏说明：%q", i, tt.Texts())
		}
		if has := tt.HasText("恢复默认"); has != (i == 0) {
			t.Errorf("第 %d 页「恢复默认」应为 %v", i, i == 0)
		}
	}
}

// 关闭未保存标签的提示是设计稿的小号提示框：标题「保存更改」、「不保存 / 取消 / 保存并关闭」三个按钮。
func TestCloseDialogMatchesDesign(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	a.overlays.askClose(a.tabs[0])
	tt.Frame()
	for _, want := range []string{"保存更改", "不保存", "取消", "保存并关闭", "有未保存的修改"} {
		if !tt.HasText(want) {
			t.Errorf("关闭提示缺少 %q：%q", want, tt.Texts())
		}
	}
}

// 确认框：标题在标题行，说明在正文；危险操作用红色确认按钮仍可确认。
func TestConfirmDialogRunsAction(t *testing.T) {
	a, tt, _ := zhApp(t)
	done := false
	a.overlays.askConfirm("删除“订单服务”？", "此操作无法撤销。", "删除", func() { done = true })
	tt.Frame()
	for _, want := range []string{"删除“订单服务”？", "此操作无法撤销。", "取消"} {
		if !tt.HasText(want) {
			t.Fatalf("确认框缺少 %q：%q", want, tt.Texts())
		}
	}
	if err := tt.Click("删除"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !done || a.overlays.confirm.open {
		t.Fatalf("点击删除应执行并关闭：done=%v open=%v", done, a.overlays.confirm.open)
	}
}

// 标题栏右侧的按钮不能被居中的悬浮胶囊层盖住（该层铺满标题栏，必须让指针穿透）。
func TestTitleBarRightButtonsClickableThroughIslandLayer(t *testing.T) {
	a, tt, _ := zhApp(t)
	before := a.set.Theme
	if err := tt.Click("切换为深色主题"); err != nil {
		t.Fatalf("右侧主题按钮应可点击: %v", err)
	}
	tt.Frame()
	if a.set.Theme == before {
		t.Fatal("点击后主题应切换")
	}
}

// 快速打开：搜索框聚焦时输入文字能过滤结果。
func TestQuickOpenFiltersResults(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openQuick()
	tt.Frame()
	all := len(a.quickResults(""))
	a.overlays.quick.query = "alpha"
	tt.Frame()
	got := a.quickResults("alpha")
	if len(got) == 0 || len(got) >= all {
		t.Fatalf("搜索 alpha 应只剩部分结果：全部 %d，过滤后 %d", all, len(got))
	}
	if !tt.HasText("alpha") {
		t.Fatalf("结果里应有 alpha：%q", tt.Texts())
	}
}
