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
		{"选中的请求标签", sw + 148 + 70, rgb(p.tabActiveBorder)},
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
		if below := px(img, c.x, y+1); below == px(img, c.x, y) {
			t.Errorf("%s：线的下一行仍是线色 %v，线变粗了", c.name, below)
		}
	}
}

// 相邻标签之间不会叠出加粗的竖线：沿标签条中部横扫，任何一条竖线都只有 1px 宽。
func TestTabStripNoThickVerticalLines(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	a.activate(0)
	a.openRequest(a.coll.Requests[1].ID) // 三种状态：选中在中间、前后各有未选中
	tt.Frame()
	tt.Frame()
	img := tt.Image()
	y := sizeToolbar - 1 + sizeTabStrip - 4 // 标签条靠下的一行，避开文字
	bg := px(img, img.Bounds().Max.X-300, y)
	sw := int(a.set.SidebarWidth)
	// 选中标签两侧：左右各一条强调色线，且紧邻的像素不是第二条线。
	activeLeft, activeRight := -1, -1
	for x := sw + 2; x < img.Bounds().Max.X; x++ {
		if px(img, x, y) == rgb(a.p.tabActiveBorder) {
			if activeLeft < 0 {
				activeLeft = x
			}
			activeRight = x
		}
	}
	if activeLeft < 0 || activeRight-activeLeft < 100 {
		t.Fatalf("没找到选中标签的左右边框: %d %d", activeLeft, activeRight)
	}
	for _, x := range []int{activeLeft, activeRight} {
		for _, dx := range []int{-1, 1} {
			n := px(img, x+dx, y)
			if n != bg && n != rgb(a.p.surface) {
				t.Errorf("选中标签边框 x=%d 旁边的像素 %v 不是背景色，叠出了第二条线", x, n)
			}
		}
	}
}
