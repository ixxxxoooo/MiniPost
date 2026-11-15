package gui

import (
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

const longTitle = "一个非常非常非常非常非常长的请求标题用来触发悬浮提示"

// tipApp 打开两个标签，第一个标题很长（一定会被截断）。
func tipApp(t *testing.T) (*app, *ui.Tester) {
	t.Helper()
	be := newFakeBackend(t)
	pid := be.seedProject("Demo", longTitle, "beta")
	a := newApp(be)
	a.set.Locale = localeZH
	a.selectProject(pid)
	tt := ui.NewTester(a.view, 1200, 760)
	tt.Frame()
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	tt.Frame()
	return a, tt
}

// near 判断两个颜色是否几乎相同。
func near(c [3]uint8, want ui.Color, tol int) bool {
	d := func(a, b uint8) int {
		v := int(a) - int(b)
		if v < 0 {
			v = -v
		}
		return v
	}
	return d(c[0], want.R) <= tol && d(c[1], want.G) <= tol && d(c[2], want.B) <= tol
}

// tipBox 在 img 的区域 r 内找出悬浮提示的底色块，返回它的外框；找不到时 ok=false。
// 只认横向连续 40px 以上的底色行（提示是大色块，输入框里的深色文字笔画不会有这么长的纯色行）。
func tipBox(img *image.RGBA, a *app, r image.Rectangle) (box image.Rectangle, ok bool) {
	fill := a.p.tipFill()
	box = image.Rectangle{Min: r.Max, Max: r.Min}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		run := 0
		for x := r.Min.X; x < r.Max.X; x++ {
			if !near(px(img, x, y), fill, 4) {
				run = 0
				continue
			}
			run++
			if run == 40 {
				box.Min.X, box.Min.Y = min(box.Min.X, x-39), min(box.Min.Y, y)
				ok = true
			}
			if run >= 40 {
				box.Max.X, box.Max.Y = max(box.Max.X, x+1), max(box.Max.Y, y+1)
			}
		}
	}
	return box, ok
}

// 回归：侧栏、标签栏所在的子主题把背景设成了透明，缺省的反色文字跟着透明，
// 悬浮提示成了一块没有字的黑块。提示里必须有和底色反差明显的文字像素。
func TestTooltipTextIsVisibleInTabBar(t *testing.T) {
	a, tt := tipApp(t)
	x, y := a.set.SidebarWidth+90, float32(sizeToolbar+sizeTabStrip/2)
	tt.Move(x-20, y)
	tt.Frame()
	tt.Move(x, y)
	tt.Frame()
	tt.Advance(tipDelay + 300*time.Millisecond)
	tt.Frame()

	img := tt.Image()
	// 只在标签栏下方一带找（提示出现在标签正下方），避开标签栏本身
	scan := image.Rect(int(x)-150, sizeToolbar+sizeTabStrip-6, int(x)+450, sizeToolbar+sizeTabStrip+60)
	box, ok := tipBox(img, a, scan)
	if !ok {
		t.Fatal("悬停超过延迟后应出现悬浮提示（找不到提示底色块）")
	}
	if box.Min.Y < sizeToolbar+sizeTabStrip-6 {
		t.Fatalf("提示应在标签正下方，实际顶边 %d", box.Min.Y)
	}
	// 外框内应有文字像素：与底色反差大
	fill := a.p.tipFill()
	textPx := 0
	for yy := box.Min.Y; yy < box.Max.Y; yy++ {
		for xx := box.Min.X; xx < box.Max.X; xx++ {
			if !near(px(img, xx, yy), fill, 90) {
				textPx++
			}
		}
	}
	if textPx < 40 {
		t.Fatalf("提示里没有可见文字（文字像素 %d），文字色可能和底色相同或透明", textPx)
	}
}

// 提示在鼠标停留 tipDelay（设计稿 450ms）后才出现，移开立即消失。
func TestTooltipDelayAndDismiss(t *testing.T) {
	a, tt := tipApp(t)
	x, y := a.set.SidebarWidth+90, float32(sizeToolbar+sizeTabStrip/2)
	scan := image.Rect(int(x)-150, sizeToolbar+sizeTabStrip-6, int(x)+450, sizeToolbar+sizeTabStrip+60)
	tt.Move(x-20, y)
	tt.Frame()
	tt.Move(x, y)
	tt.Frame()
	tt.Advance(tipDelay / 2)
	tt.Frame()
	if _, ok := tipBox(tt.Image(), a, scan); ok {
		t.Fatal("未到延迟不应出现提示")
	}
	tt.Advance(tipDelay)
	tt.Frame()
	if _, ok := tipBox(tt.Image(), a, scan); !ok {
		t.Fatal("超过延迟应出现提示")
	}
	tt.Move(x, 400) // 移到编辑区
	tt.Frame()
	tt.Advance(durBase * 2)
	tt.Frame()
	if _, ok := tipBox(tt.Image(), a, scan); ok {
		t.Fatal("鼠标移开后提示应消失")
	}
}

// 提示色令牌：底色是前景色 92% 混黑，文字色是表面色；主题里的反色也要给出（供 Toast 等使用）。
func TestTooltipThemeTokens(t *testing.T) {
	for _, dark := range []bool{false, true} {
		p := schemePalette("mist", dark)
		th := p.theme(fsBase)
		if th.Inverse != p.tipFill() || th.InverseText != p.surface {
			t.Fatalf("dark=%v 主题反色应等于提示令牌", dark)
		}
		if th.TooltipDelay != tipDelay {
			t.Fatalf("提示延迟应为 %v", tipDelay)
		}
		if th.InverseText == ui.Transparent {
			t.Fatal("反色文字不能是透明")
		}
	}
}

// 量宽用的字重必须与绘制一致：选中标签更粗，粗字更宽，被截断时才不会漏掉提示。
func TestTabTitleWeightMatchesPaint(t *testing.T) {
	if tabTitleWeight(true) <= tabTitleWeight(false) {
		t.Fatal("选中标签标题应比未选中的更粗")
	}
}
