package gui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSidebarLimits(t *testing.T) {
	cases := []struct{ win, lo, hi float32 }{
		{1280, 180, 500}, // 宽窗口：上限 500
		{1060, 180, 500}, // 恰好 560 给请求区
		{900, 180, 340},  // 最小窗口：侧栏最多 340，请求区至少 560
		{600, 180, 180},  // 极端窄：不低于下限
	}
	for _, c := range cases {
		lo, hi := sidebarLimits(c.win)
		if lo != c.lo || hi != c.hi {
			t.Errorf("窗口 %.0f: 得到 [%.0f,%.0f]，想要 [%.0f,%.0f]", c.win, lo, hi, c.lo, c.hi)
		}
	}
}

// 窗口最窄时，即使设置的侧栏很宽，右侧请求区也至少有 mainMinWidth；
// 设置值原样保留，窗口变宽后恢复。
func TestNarrowWindowClampsSidebarButKeepsSetting(t *testing.T) {
	a, _, _ := zhApp(t)
	a.set.SidebarWidth = 500
	a.openRequest(a.coll.Requests[0].ID)
	tt := ui.NewTester(a.view, 900, 600)
	tt.Frame()
	if a.side.shown > 340 || a.side.shown < 180 {
		t.Fatalf("900 宽窗口里侧栏应夹紧到 ≤340，实际 %.0f", a.side.shown)
	}
	if 900-a.side.shown < mainMinWidth {
		t.Fatalf("请求区宽度 %.0f 小于最小值 %d", 900-a.side.shown, mainMinWidth)
	}
	if a.set.SidebarWidth != 500 {
		t.Fatalf("设置值不应被改写: %.0f", a.set.SidebarWidth)
	}
	// 请求区的内容确实贴着夹紧后的侧栏：Params 标签在主区左侧，离窗口左边不会超过 侧栏宽+边距
	r, ok := tt.Find("Params")
	if !ok {
		t.Fatal("找不到 Params 标签")
	}
	if r.X > a.side.shown+60 {
		t.Fatalf("请求区应紧跟夹紧后的侧栏，Params.X=%.0f 侧栏=%.0f", r.X, a.side.shown)
	}

	wide := ui.NewTester(a.view, 1400, 800)
	wide.Frame()
	if a.side.shown != 500 {
		t.Fatalf("窗口变宽后应恢复到设置的 500，实际 %.0f", a.side.shown)
	}
}
