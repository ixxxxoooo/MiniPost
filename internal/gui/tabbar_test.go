package gui

import (
	"image"
	"testing"

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

// 选中标签的底部没有线、与下方内容区同色；其余位置的标签栏底部也没有分隔线。
func TestActiveTabMergesWithContent(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	snapshot(t, tt, "tabs")
	img := tt.Image()
	stripBottom := sizeToolbar - 1 + sizeTabStrip - 1 // 标签栏最后一行像素（标题栏不再画底线，标签栏从 sizeToolbar-1 开始）
	x := int(a.set.SidebarWidth) + 100                // 选中标签内部、文字右侧（标签最小宽 120）
	below := img.RGBAAt(x, stripBottom+3)             // 内容区
	at := img.RGBAAt(x, stripBottom)
	if at != below {
		t.Fatalf("选中标签底部应与内容区同色（无线）: 标签栏底 %v，内容区 %v", at, below)
	}
	// 标签栏空白处（右侧）底部也没有横线：与上方一行同色。
	far := img.RGBAAt(img.Bounds().Max.X-300, stripBottom)
	above := img.RGBAAt(img.Bounds().Max.X-300, stripBottom-4)
	if far != above {
		t.Fatalf("标签栏底部不应有分隔线: %v vs %v", far, above)
	}
}

// px 取画面上一个像素的 RGB。
func px(img *image.RGBA, x, y int) [3]uint8 {
	c := img.RGBAAt(x, y)
	return [3]uint8{c.R, c.G, c.B}
}

func rgb(c ui.Color) [3]uint8 { return [3]uint8{c.R, c.G, c.B} }

// 标签条顶部只有一条线：选中的标签用强调色上边框盖住它，其余位置是普通边框色；
// 线的下一行不能再有第二条线，上一行（标题栏）也不能有线——不会出现两条线叠在一起。
// 侧栏顶部已改为分段控件，不再画顶线（见 TestSideSegmentedTabs）。
func TestTabStripHasSingleTopLine(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	a.activate(1)
	tt.Frame()
	tt.Frame()
	img := tt.Image()
	p := a.p
	y := sizeToolbar - 1 // 标签条的第一行：顶线所在
	sw := int(a.set.SidebarWidth)
	cases := []struct {
		name string
		x    int
		want [3]uint8
	}{
		{"请求标签条空白处", img.Bounds().Max.X - 300, rgb(p.border)},
		{"未选中的请求标签", sw + 60, rgb(p.border)},
		{"选中的请求标签", sw + int(a.tabs[0].shownW) + 40, rgb(p.tabActiveBorder)},
	}
	for _, c := range cases {
		if got := px(img, c.x, y); got != c.want {
			t.Errorf("%s：顶线颜色 %v，应为 %v", c.name, got, c.want)
		}
		if above, line := px(img, c.x, y-1), px(img, c.x, y); above == line {
			t.Errorf("%s：线的上一行（标题栏）也是线色 %v，出现了两条线", c.name, above)
		}
		// 选中项的强调色边框恰好 tabAccentWidth 行；其余位置的线只有 1 行。
		thick := 1
		if c.want == rgb(p.tabActiveBorder) {
			thick = tabAccentWidth
		}
		for k := 1; k < thick; k++ {
			if px(img, c.x, y+k) != c.want {
				t.Errorf("%s：强调色边框应有 %d 行，第 %d 行颜色不对", c.name, thick, k+1)
			}
		}
		if below := px(img, c.x, y+thick); below == c.want {
			t.Errorf("%s：线比预期的 %d 行更粗", c.name, thick)
		}
	}
}

// 选中标签只有顶部有强调色，左右没有；标签之间的竖分隔线都只有 1px 宽，不会叠成粗线。
func TestTabStripNoSideHighlightAndThinDividers(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	a.activate(0)
	a.openRequest(a.coll.Requests[1].ID) // 选中第二个，前后各有分隔线
	tt.Frame()
	tt.Frame()
	img := tt.Image()
	y := sizeToolbar - 1 + sizeTabStrip - 4 // 标签条靠下的一行，避开文字和顶部强调线
	acc, div := rgb(a.p.tabActiveBorder), rgb(a.p.tabDivider)
	sw := int(a.set.SidebarWidth)
	dividers := 0
	for x := sw; x < img.Bounds().Max.X; x++ {
		got := px(img, x, y)
		if got == acc {
			t.Fatalf("x=%d：选中标签的左右不应有强调色", x)
		}
		if got == div {
			dividers++
			if px(img, x+1, y) == div {
				t.Errorf("x=%d：分隔线比 1px 更粗", x)
			}
		}
	}
	if dividers < 2 {
		t.Errorf("应至少有 2 条标签分隔线，实际 %d", dividers)
	}
}

// 切换选中标签时每个标签的宽度不变（选中项字重更粗，不能让标签突然变宽/变窄）。
func TestTabWidthStableOnSwitch(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	a.tabs[0].draft.Name = "一个很长很长很长很长很长很长很长很长很长的标签标题 (long title)"
	a.tabs[1].draft.Name = "另一个同样很长很长很长很长很长很长很长很长的标题 (long title)"
	sw := int(a.set.SidebarWidth)
	a.activate(0)
	tt.Frame()
	tt.Frame()
	y := sizeToolbar - 1 // 顶部强调线所在行
	img0 := tt.Image()
	// 选中第一个标签时，第二个标签从 sw+tabMaxWidth 开始（第一个标签 + 无分隔线）；
	// 选中第二个标签时，它的左边框也应从同一列开始。
	a.activate(1)
	tt.Frame()
	tt.Frame()
	img1 := tt.Image()
	acc := rgb(a.p.tabActiveBorder)
	// 第一个标签占 [sw, sw+tabMaxWidth)（标题很长，取最大宽度），分隔线在 sw+tabMaxWidth，第二个标签从 sw+tabMaxWidth+1 开始。
	if px(img1, sw+tabMaxWidth+1, y) != acc || px(img1, sw+tabMaxWidth, y) == acc {
		t.Errorf("选中第二个标签时顶部强调线应从 x=%d 开始", sw+tabMaxWidth+1)
	}
	if px(img0, sw+tabMaxWidth-1, y) != acc || px(img0, sw+tabMaxWidth, y) == acc {
		t.Errorf("选中第一个标签时顶部强调线应在 x=%d 结束", sw+tabMaxWidth-1)
	}
}

// 标签栏放不下所有标签时，切换/打开的标签要自动滚进可视区。
func TestActiveTabScrollsIntoView(t *testing.T) {
	a, _, _ := zhApp(t)
	tt := ui.NewTester(a.view, 600, 760) // 窄窗口：标签栏只有约 150px 宽，放不下一个 220px 的标签
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	for i := 0; i < 4; i++ {
		tt.Frame()
	}
	sw, y := int(a.set.SidebarWidth), sizeToolbar-1
	acc := rgb(a.p.tabActiveBorder)
	visible := func() bool {
		img := tt.Image()
		for x := sw; x < sw+120; x++ { // 标签栏左侧 120px 内必须能看到选中标签的强调线
			if px(img, x, y) == acc {
				return true
			}
		}
		return false
	}
	if !visible() {
		t.Fatal("打开第二个标签后，它应滚入可视区")
	}
	a.activate(0)
	for i := 0; i < 4; i++ {
		tt.Frame()
	}
	if !visible() {
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

// 标签宽度随标题自适应：短标题取下限，长标题取上限，中等长度的在两者之间且更长的更宽。
func TestTabWidthAdaptsToTitle(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	names := []string{"a", "中等长度的标题", "中等长度的标题再长一些一些一些"}
	a.tabs[0].draft.Name = names[0]
	a.tabs[1].draft.Name = names[1]
	tt.Frame()
	short, mid := a.tabs[0].shownW, a.tabs[1].shownW
	if short != tabMinWidth {
		t.Fatalf("短标题应取下限 %d，实际 %.0f", tabMinWidth, short)
	}
	if !(mid > short && mid < tabMaxWidth) {
		t.Fatalf("中等标题应在 (%d,%d) 之间，实际 %.0f", tabMinWidth, tabMaxWidth, mid)
	}
	a.tabs[1].draft.Name = names[2]
	tt.Frame()
	if longer := a.tabs[1].shownW; longer <= mid {
		t.Fatalf("更长的标题应更宽: %.0f → %.0f", mid, longer)
	}
	a.tabs[1].draft.Name = "一个很长很长很长很长很长很长很长很长很长很长很长很长的标签标题 (long title)"
	tt.Frame()
	if a.tabs[1].shownW != tabMaxWidth {
		t.Fatalf("超长标题应取上限 %d，实际 %.0f", tabMaxWidth, a.tabs[1].shownW)
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

// 相邻标签首尾相接：第二个标签的左边正好是第一个标签的宽度加 1px 分隔线之后。
func TestAdaptiveTabsAreContiguous(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	a.tabs[0].draft.Name = "中等长度的标题"
	tt.Frame()
	tt.Frame()
	img := tt.Image()
	y := sizeToolbar - 1 // 顶线所在行；选中的第二个标签用强调色盖住它
	sw := int(a.set.SidebarWidth)
	acc := rgb(a.p.tabActiveBorder)
	start := sw + int(a.tabs[0].shownW) + 1
	if px(img, start, y) != acc || px(img, start-2, y) == acc {
		t.Fatalf("选中的第二个标签应从 x=%d 开始", start)
	}
}

// 标签有未保存修改时必须画出主题色圆点：改动草稿（dirty 每帧由草稿指纹重新计算）后，
// 标签栏区域的画面应有差异；改回去后圆点消失、画面恢复。
func TestTabDirtyDotIsDrawn(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
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
	tt.Frame()
	if !a.tabs[0].dirty {
		t.Fatal("改动草稿后应为未保存状态")
	}
	if same(clean, strip()) {
		t.Fatal("标脏后标签栏画面没有变化，未保存圆点没有画出来")
	}
	a.tabs[0].draft.URL = orig
	tt.Frame()
	if !same(clean, strip()) {
		t.Fatal("改回原值后圆点应消失，标签栏应恢复原样")
	}
}
