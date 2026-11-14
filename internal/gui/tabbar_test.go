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
	x := int(a.set.SidebarWidth) + 130                // 选中标签内部、文字右侧（标签最小宽 148）
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
// 请求标签条和侧栏标签条都要满足。
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
		{"选中的请求标签", sw + tabWidth + 70, rgb(p.tabActiveBorder)},
		{"侧栏选中的标签", 60, rgb(p.tabActiveBorder)},
		{"侧栏未选中的标签", 150, rgb(p.border)},
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
	// 选中第一个标签时，第二个标签从 sw+tabWidth 开始（第一个标签 + 无分隔线）；
	// 选中第二个标签时，它的左边框也应从同一列开始。
	a.activate(1)
	tt.Frame()
	tt.Frame()
	img1 := tt.Image()
	acc := rgb(a.p.tabActiveBorder)
	// 第一个标签占 [sw, sw+tabWidth)，分隔线在 sw+tabWidth，第二个标签从 sw+tabWidth+1 开始。
	if px(img1, sw+tabWidth+1, y) != acc || px(img1, sw+tabWidth, y) == acc {
		t.Errorf("选中第二个标签时顶部强调线应从 x=%d 开始", sw+tabWidth+1)
	}
	if px(img0, sw+tabWidth-1, y) != acc || px(img0, sw+tabWidth, y) == acc {
		t.Errorf("选中第一个标签时顶部强调线应在 x=%d 结束", sw+tabWidth-1)
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
