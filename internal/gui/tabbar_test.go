package gui

import (
	"testing"
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
	stripBottom := sizeToolbar + sizeTabStrip - 1 // 标签栏最后一行像素
	x := int(a.set.SidebarWidth) + 130            // 选中标签内部、文字右侧（标签最小宽 148）
	below := img.RGBAAt(x, stripBottom+3)         // 内容区
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
