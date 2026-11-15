package gui

import (
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// 鼠标中键点击标签关闭该标签（点的是哪个标签就关哪个，不一定是当前标签）。
func TestTabMiddleClickCloses(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	tt.Frame()
	if len(a.tabs) != 2 {
		t.Fatalf("应有两个标签，实际 %d", len(a.tabs))
	}
	// 第一个标签在标签栏最左：侧栏宽度之后，高度在标题栏下方的标签栏中间。
	x, y := a.set.SidebarWidth+90, float32(sizeToolbar+sizeTabStrip/2)
	tt.PressButton(2, x, y)
	tt.ReleaseButton(2, x, y)
	tt.Frame()
	if len(a.tabs) != 1 || a.tabs[0].title() != "beta" {
		t.Fatalf("中键应关闭 alpha，剩余 %d 个标签", len(a.tabs))
	}
}

// 回归：中键关闭后界面必须立即刷新。以前关闭动作排到帧末才执行，帧已经画完却不再重建，
// 画面和命中区域还停在旧标签上；紧接着的下一次中键点击会落到已关闭的旧标签上被丢弃（表现为“有时候中键点了没反应”）。
func TestTabMiddleClickRepeatedWithoutMove(t *testing.T) {
	motionEnabled = true // 和真实运行一致：开启动效
	t.Cleanup(func() { motionEnabled = false })
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	tt.Frame()
	x, y := a.set.SidebarWidth+90, float32(sizeToolbar+sizeTabStrip/2)
	for i, want := range []int{1, 0} {
		tt.PressButton(2, x, y)
		tt.ReleaseButton(2, x, y)
		if len(a.tabs) != want {
			t.Fatalf("第 %d 次中键后应剩 %d 个标签，实际 %d", i+1, want, len(a.tabs))
		}
	}
}

// 左键按下抬起不会被当成中键关闭。
func TestTabLeftClickDoesNotClose(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	tt.Frame()
	if err := tt.Click("alpha"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.tabs) != 2 || a.active != 0 {
		t.Fatalf("左键只切换标签: tabs=%d active=%d", len(a.tabs), a.active)
	}
}

// px 取画面上一个像素的 RGB。
func px(img *image.RGBA, x, y int) [3]uint8 {
	c := img.RGBAAt(x, y)
	return [3]uint8{c.R, c.G, c.B}
}

func rgb(c ui.Color) [3]uint8 { return [3]uint8{c.R, c.G, c.B} }

// accentRun 在标签条里选中标签的顶部轮廓线（tabOutline：统一 1px、强调色混边框色，两端是圆角）上取样：
// 在标签顶部附近逐行扫描轮廓色像素，取最长的一段连续轮廓色，返回它的起点 x 与长度；没有则返回 -1, 0。
// 轮廓会在标签之间滑动（Match 过渡），先让动画走完再取样。函数名沿用旧版（那时顶线是强调色）。
func accentRun(a *app, tt *ui.Tester) (x, n int) {
	for i := 0; i < 8; i++ {
		tt.Advance(100 * time.Millisecond)
	}
	img := tt.Image()
	acc := rgb(a.p.border.Mix(a.p.accent, 0.58))
	x = -1
	for y := sizeToolbar + 2; y <= sizeToolbar+14; y++ {
		start, run := -1, 0
		for i := 0; i <= img.Bounds().Dx(); i++ {
			if i < img.Bounds().Dx() && px(img, i, y) == acc {
				if start < 0 {
					start = i
				}
				run++
				continue
			}
			if run > n {
				x, n = start, run
			}
			start, run = -1, 0
		}
	}
	if x < 0 {
		return -1, 0
	}
	return x, n
}

// 选中的标签是浮起的内容底胶囊：胶囊内部与内容区同色，其余位置的标签栏底色不同（设计稿 .tab.on）。
func TestActiveTabIsRaisedPill(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	tt.Frame()
	snapshot(t, tt, "tabs")
	x, n := accentRun(a, tt)
	if n == 0 {
		t.Fatal("选中的标签底部应有强调线")
	}
	img := tt.Image()
	// 顶部轮廓线是直线段，两端各让出 10px 圆角，标签右缘约在 x+n+10；取右缘内侧 14px、上沿下方 12px 的位置（避开文字）。
	inside := px(img, x+n-14, sizeToolbar+6+12)
	if inside != rgb(a.p.surface) {
		t.Fatalf("选中标签内部应是内容底色 %v，实际 %v", rgb(a.p.surface), inside)
	}
	outside := px(img, x+n+13+tabGap+40, sizeToolbar+6+12) // 标签右侧的空白标签栏
	if outside == rgb(a.p.surface) {
		t.Fatalf("标签栏空白处不应与内容底同色（标签栏是 chrome 与内容底的混合色）: %v", outside)
	}
}

// 切换选中标签时每个标签的宽度不变（选中项字重更粗，不能让标签突然变宽/变窄）。
func TestTabWidthStableOnSwitch(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	a.tabs[0].draft.Name = "一个很长很长很长很长很长很长很长很长很长的标签标题 (long title)"
	a.tabs[1].draft.Name = "另一个同样很长很长很长很长很长很长很长很长的标题 (long title)"
	a.activate(0)
	tt.Frame()
	tt.Frame()
	x0, n0 := accentRun(a, tt)
	a.activate(1)
	tt.Frame()
	tt.Frame()
	x1, n1 := accentRun(a, tt)
	// 顶部轮廓是直线段，两端各让出 10px 圆角，所以整段比标签窄约 20px（抗锯齿再吃掉几像素）
	if n0 != n1 || n0 > tabMaxWidth-16 || n0 < tabMaxWidth-28 {
		t.Errorf("两个标签的顶部轮廓应一样长，约 %d（标签宽 %d），实际 %d、%d", tabMaxWidth-20, tabMaxWidth, n0, n1)
	}
	if x1-x0 != tabMaxWidth+tabGap {
		t.Errorf("第二个标签应从第一个标签右边 %d（标签宽 + 间距）处开始，实际相差 %d", tabMaxWidth+tabGap, x1-x0)
	}
}

// 标签栏放不下所有标签时，切换/打开的标签要自动滚进可视区。
func TestActiveTabScrollsIntoView(t *testing.T) {
	a, _, _ := zhApp(t)
	tt := ui.NewTester(a.view, 600, 760) // 窄窗口：标签栏只有约 90px 宽，放不下两个标签
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	for i := 0; i < 4; i++ {
		tt.Frame()
	}
	// 标签栏有滚动裁剪：标签滚出可视区时，它的强调线也跟着被裁掉。
	if _, n := accentRun(a, tt); n == 0 {
		t.Fatal("打开第二个标签后，它应滚入可视区")
	}
	a.activate(0)
	for i := 0; i < 4; i++ {
		tt.Frame()
	}
	if _, n := accentRun(a, tt); n == 0 {
		t.Fatal("切回第一个标签后，它应滚入可视区")
	}
}

// 标签栏里的“+”：有标签时点击新建请求并在新标签打开（曾经点了没有任何反应）。
func TestTabBarPlusCreatesRequest(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	before, reqs := len(a.tabs), len(a.coll.Requests)
	if err := tt.Click("新建请求"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.tabs) != before+1 || len(a.coll.Requests) != reqs+1 {
		t.Fatalf("点“+”应新增一个请求和标签: tabs %d→%d reqs %d→%d", before, len(a.tabs), reqs, len(a.coll.Requests))
	}
	if a.active != len(a.tabs)-1 {
		t.Fatalf("新标签应被激活: active=%d", a.active)
	}
}

// 没有任何标签时，标签栏左侧独占一格的“+”同样可以新建请求。
func TestTabBarPlusWorksWithoutTabs(t *testing.T) {
	a, tt, _ := zhApp(t)
	if len(a.tabs) != 0 {
		t.Fatalf("前置条件：应没有标签，实际 %d", len(a.tabs))
	}
	reqs := len(a.coll.Requests)
	if err := tt.Click("新建请求"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.tabs) != 1 || len(a.coll.Requests) != reqs+1 {
		t.Fatalf("无标签时点“+”应新建并打开: tabs=%d reqs %d→%d", len(a.tabs), reqs, len(a.coll.Requests))
	}
}

// “+”按钮常态就有淡底（不再是透明的灰色小图标）：像素颜色应与标签栏底色不同。
func TestTabBarPlusHasVisibleBackground(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	r, ok := tt.Find("新建请求")
	if !ok {
		t.Fatal("找不到“+”按钮")
	}
	img := tt.Image()
	// 取按钮左上角内侧一点（避开图标本身）
	px := img.RGBAAt(int(r.X)+3, int(r.Y)+3)
	strip := img.RGBAAt(int(r.X)-3, int(r.Y)+3)
	if px == strip {
		t.Fatalf("“+”按钮常态应有可见底色，与标签栏背景不同: %v", px)
	}
}

// 标签宽度与标题无关：宽敞时都是基础宽度；标签栏变窄后按可用宽度平分，但不小于下限。
func TestTabWidthIgnoresTitleAndShrinksWhenCrowded(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	a.tabs[0].draft.Name = "a"
	a.tabs[1].draft.Name = "一个很长很长很长很长很长很长很长很长很长很长很长很长的标签标题 (long title)"
	tt.Frame()
	if a.tabs[0].shownW != tabBaseWidth || a.tabs[1].shownW != tabBaseWidth {
		t.Fatalf("宽敞时标签应都是基础宽度 %d，实际 %.0f、%.0f", tabBaseWidth, a.tabs[0].shownW, a.tabs[1].shownW)
	}
	narrow := ui.NewTester(a.view, 560, 760) // 可用宽度约 250：两个标签各约 120
	narrow.Frame()
	w := a.tabs[0].shownW
	if !(w >= tabMinWidth && w < tabBaseWidth) || a.tabs[1].shownW != w {
		t.Fatalf("变窄后两个标签应平分且在 [%d,%d) 之间，实际 %.0f、%.0f", tabMinWidth, tabBaseWidth, w, a.tabs[1].shownW)
	}
}

// 不抖动：切换选中、出现未保存圆点、鼠标悬停/移开，所有标签的宽度都不变。
func TestTabWidthsDoNotJitter(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	a.tabs[0].draft.Name = "中等长度的标题"
	a.tabs[1].draft.Name = "另一个中等长度标题"
	a.tabs[0].base.Name, a.tabs[1].base.Name = a.tabs[0].draft.Name, a.tabs[1].draft.Name // 视为已保存
	tt.Frame()
	widths := func() [2]float32 { return [2]float32{a.tabs[0].shownW, a.tabs[1].shownW} }
	base := widths()
	check := func(what string) {
		t.Helper()
		if got := widths(); got != base {
			t.Fatalf("%s后标签宽度变了: %v → %v", what, base, got)
		}
	}

	a.activate(0)
	tt.Frame()
	check("切换到第一个标签")
	a.activate(1)
	tt.Frame()
	check("切换到第二个标签")

	// 未保存圆点出现（不改标题和方法，只标脏）
	orig := a.tabs[1].draft.URL
	a.tabs[1].draft.URL = orig + "?changed=1" // dirty 每帧由草稿指纹重新计算，必须真的改草稿
	tt.Frame()
	if !a.tabs[1].dirty {
		t.Fatal("改动草稿后应为未保存状态")
	}
	check("出现未保存圆点")
	a.tabs[1].draft.URL = orig
	tt.Frame()
	check("圆点消失")

	// 悬停到标签上再移开（关闭按钮图标出现/消失）
	y := float32(sizeToolbar + sizeTabStrip/2)
	tt.Move(a.set.SidebarWidth+30, y)
	tt.Frame()
	check("鼠标悬停在第一个标签")
	tt.Move(600, 500)
	tt.Frame()
	check("鼠标移开")
}

// 相邻标签首尾相接：第二个标签的左边正好是第一个标签的宽度加标签间距之后。
func TestAdaptiveTabsAreContiguous(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	a.activate(0)
	tt.Frame()
	tt.Frame()
	x0, _ := accentRun(a, tt)
	a.activate(1)
	tt.Frame()
	tt.Frame()
	x1, _ := accentRun(a, tt)
	if want := int(a.tabs[0].shownW) + tabGap; x1-x0 != want {
		t.Fatalf("第二个标签应在第一个标签右边 %d 处开始，实际 %d", want, x1-x0)
	}
}

// 标签有未保存修改时必须画出主题色圆点：改动草稿（dirty 每帧由草稿指纹重新计算）后，
// 标签栏区域的画面应有差异；改回去后圆点消失、画面恢复。
func TestTabDirtyDotIsDrawn(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID) // 当前标签是第二个：未保存圆点只画在非当前、非悬停的标签上
	tt.Frame()
	if a.tabs[0].dirty {
		t.Fatal("刚打开的请求不应是未保存状态")
	}
	strip := func() *image.RGBA {
		img := tt.Image()
		r := image.Rect(int(a.set.SidebarWidth), sizeToolbar, img.Bounds().Dx(), sizeToolbar+sizeTabStrip)
		dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		for y := 0; y < r.Dy(); y++ {
			for x := 0; x < r.Dx(); x++ {
				dst.Set(x, y, img.At(r.Min.X+x, r.Min.Y+y))
			}
		}
		return dst
	}
	same := func(x, y *image.RGBA) bool { return string(x.Pix) == string(y.Pix) }

	clean := strip()
	orig := a.tabs[0].draft.URL
	a.tabs[0].draft.URL = orig + "?changed=1"
	a.tabs[0].refreshDirty() // 只有当前标签每帧刷新未保存状态，这里手动刷新非当前的第一个标签
	tt.Frame()
	if !a.tabs[0].dirty {
		t.Fatal("改动草稿后应为未保存状态")
	}
	if same(clean, strip()) {
		t.Fatal("标脏后标签栏画面没有变化，未保存圆点没有画出来")
	}
	a.tabs[0].draft.URL = orig
	a.tabs[0].refreshDirty()
	tt.Frame()
	if !same(clean, strip()) {
		t.Fatal("改回原值后圆点应消失，标签栏应恢复原样")
	}
}
