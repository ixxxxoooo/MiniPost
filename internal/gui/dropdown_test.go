package gui

import (
	"image"
	"testing"
	"time"

	"minipost/internal/model"
)

// 项目下拉面板带「全部项目…」入口（设计稿 .pop[type=project] 底部），点击后回到项目主页并收起面板。
func TestProjectDropdownAllProjectsGoesHome(t *testing.T) {
	a, tt, _ := zhApp(t)
	if err := tt.Click("Demo"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("全部项目…") || !tt.HasText("⌘⇧H") {
		t.Fatalf("项目面板应有「全部项目…」与快捷键提示，实际 %q", tt.Texts())
	}
	if err := tt.Click("全部项目…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.showHome || a.projMenu.open {
		t.Fatalf("应回到项目主页并收起面板：showHome=%v open=%v", a.showHome, a.projMenu.open)
	}
}

// 环境下拉面板：展开 → 列出环境与变量数 → 「管理环境…」打开环境页并收起面板。
func TestEnvDropdownManageOpensEnvPage(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.envs = []model.Environment{
		{ID: "e1", Name: "预发环境", Variables: []model.Variable{{ID: "v1", Key: "a"}, {ID: "v2", Key: "b"}}},
	}
	tt.Frame()
	if err := tt.Click("切换环境"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("2 变量") {
		t.Fatalf("环境行应显示变量数，实际 %q", tt.Texts())
	}
	if err := tt.Click("管理环境…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.envOpen {
		t.Fatal("点击「管理环境…」后面板应收起")
	}
}

// 发送按钮右侧的箭头展开设计稿的发送面板；「发送并保存响应…」会按下载模式发送。
func TestSendDropdownSendAndDownload(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tab := a.activeTab()
	tab.draft.URL = "http://127.0.0.1:1/x"
	tt.Frame()
	a.sendMenuOpen = true
	tt.Frame()
	for _, want := range []string{"发送", "发送并保存响应…", "⌘↵"} {
		if !tt.HasText(want) {
			t.Fatalf("发送面板缺少 %q，实际 %q", want, tt.Texts())
		}
	}
	if err := tt.Click("发送并保存响应…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.sendMenuOpen {
		t.Fatal("选择后发送面板应收起")
	}
}

// URL 为空时发送按钮不可用，发送面板不应保持展开。
func TestSendDropdownClosesWhenURLEmpty(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.activeTab().draft.URL = ""
	a.sendMenuOpen = true
	tt.Frame()
	if a.sendMenuOpen {
		t.Fatal("URL 为空时发送面板应自动收起")
	}
}

// 状态码徽标颜色取自主题语义令牌：2xx 成功、3xx 信息、4xx 警告、5xx 危险，且随明暗变化。
func TestStatusBadgeUsesThemeTokens(t *testing.T) {
	for _, dark := range []bool{false, true} {
		a, _, _ := zhApp(t)
		a.p = schemePalette("mist", dark)
		cases := []struct {
			code int
			want [2]interface{}
		}{
			{200, [2]interface{}{a.p.success, a.p.okSoft}},
			{301, [2]interface{}{a.p.info, a.p.infoSoft}},
			{404, [2]interface{}{a.p.warning, a.p.warnSoft}},
			{500, [2]interface{}{a.p.danger, a.p.errSoft}},
		}
		for _, tc := range cases {
			fg, bg := a.statusBadge(tc.code)
			if fg != tc.want[0] || bg != tc.want[1] {
				t.Errorf("dark=%v code=%d 颜色应取语义令牌：fg=%v bg=%v", dark, tc.code, fg, bg)
			}
		}
	}
}

// 快速打开：环境结果用环境名 + “切换为当前环境”，当前环境带「当前」标；请求说明只写所在文件夹。
func TestQuickOpenEnvAndRequestHints(t *testing.T) {
	a, _, _ := zhApp(t)
	a.envs = []model.Environment{{ID: "e1", Name: "预发环境"}, {ID: "e2", Name: "生产环境"}}
	a.setEnv("e2")
	var cur, other quickItem
	for _, it := range a.quickResults("环境") {
		if it.kind != quickEnv {
			continue
		}
		if it.label == "生产环境" {
			cur = it
		}
		if it.label == "预发环境" {
			other = it
		}
	}
	if !cur.current || other.current || cur.hint != "切换为当前环境" {
		t.Fatalf("当前环境应带标记且说明正确：cur=%+v other=%+v", cur, other)
	}
	for _, it := range a.quickResults("alpha") {
		if it.kind == quickRequest && it.hint == "" {
			t.Fatalf("请求结果应有所在文件夹说明: %+v", it)
		}
	}
}

// 动效令牌与设计稿 v2 一致：popIn 自下 8px 缩放 .96，dlgIn 自下 14px 缩放 .955，itemIn 逐条错开 18ms（最多 10 条）。
func TestMotionTokensMatchDesignV2(t *testing.T) {
	if motionPop.Y != 8 || motionPop.Scale != 0.96 || motionDlg.Y != 14 || motionDlg.Scale != 0.955 || motionItem.Y != 5 {
		t.Fatalf("入场动作与设计稿不一致: pop=%+v dlg=%+v item=%+v", motionPop, motionDlg, motionItem)
	}
	if itemDelay(0) != 30*time.Millisecond || itemDelay(3) != 84*time.Millisecond || itemDelay(50) != itemDelay(10) {
		t.Fatalf("itemIn 延迟公式不对: %v %v %v", itemDelay(0), itemDelay(3), itemDelay(50))
	}
}

// 面板切换在开启动效时是“先退后进”：切换后的第一帧旧面板仍作为退场副本留在画面里，退场结束后只剩新面板；
// 关闭动效（测试默认）时立即显示新面板。
func TestPaneSwitchExitsThenEnters(t *testing.T) {
	paneMotion = true // 必须在第一次渲染前打开：退场动画取自旧面板上一帧的过渡设置
	defer func() { paneMotion = false }()
	_, tt, _, rs := sendWith(t, jsonResp())
	rs.sub = 1
	tt.Frame()
	tt.Advance(time.Second)
	if !tt.HasText("Content-Type") {
		t.Fatalf("切到响应头并等动画结束后应显示表头，实际 %q", tt.Texts())
	}

	before := tt.Image() // 响应头面板稳定后的画面
	rs.sub = 0           // 切回响应体：响应头表先退场，响应体等它离开后才出现
	tt.Frame()
	tt.Advance(20 * time.Millisecond)
	mid := tt.Image()
	tt.Advance(durPaneOut + durSlow + 100*time.Millisecond)
	tt.Frame()
	after := tt.Image()
	if tt.HasText("Content-Type") || !tt.HasText("预览") {
		t.Fatalf("动画结束后应只显示响应体面板，实际 %q", tt.Texts())
	}
	// 切换后 20ms：旧面板仍大体在画面里，所以更接近切换前，而不是最终的响应体画面
	if d1, d2 := imgDiff(before, mid), imgDiff(before, after); d1 >= d2 {
		t.Fatalf("退场期间画面应仍接近旧面板：与切换前差 %d 像素，与最终画面差 %d 像素", d1, d2)
	}
}

// imgDiff 统计两张等大图片里不同的像素数。
func imgDiff(a, b *image.RGBA) int {
	n := 0
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				n++
			}
		}
	}
	return n
}
