package gui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestShimmerActive(t *testing.T) {
	now := time.Now()
	tb := &tab{}
	if shimmerActive(tb, now) {
		t.Fatal("从未发送过的标签不应显示等待动画")
	}
	tb.sending, tb.sendAt = true, now.Add(-time.Hour)
	if !shimmerActive(tb, now) {
		t.Fatal("发送中应显示等待动画")
	}
	tb.sending, tb.sendAt = false, now.Add(-100*time.Millisecond)
	if !shimmerActive(tb, now) {
		t.Fatal("请求很快结束时也应至少展示 shimmerMin")
	}
	tb.sendAt = now.Add(-shimmerMin - time.Millisecond)
	if shimmerActive(tb, now) {
		t.Fatal("超过最短展示时间且已结束后应停止动画")
	}
}

func TestShimmerOffset(t *testing.T) {
	start := shimmerOffset(0)
	if start >= 0 {
		t.Fatalf("起点应在容器左侧之外，实际 %v", start)
	}
	if end := shimmerOffset(shimmerPeriod - time.Millisecond); end < 100 {
		t.Fatalf("终点应扫出容器右侧，实际 %v", end)
	}
	if mid := shimmerOffset(shimmerPeriod / 2); mid <= 0 || mid >= 100 {
		t.Fatalf("中点应在容器内，实际 %v", mid)
	}
	if shimmerOffset(shimmerPeriod+time.Millisecond) > 0 {
		t.Fatal("动画应按周期循环")
	}
}

// 发送中响应区顶部应画出蓝色扫光条，结束后消失。
func TestShimmerDrawnWhileSending(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tb := a.activeTab()
	tb.sending, tb.sendAt = true, time.Now().Add(-shimmerPeriod/2)
	tt.Frame()
	busy := tt.Image()
	snapshot(t, tt, "sending")
	if !tt.HasText("发送请求...") {
		t.Fatalf("发送中应显示提示，实际 %q", tt.Texts())
	}
	title, ok := tt.Find("响应")
	if !ok {
		t.Fatal("找不到响应标题")
	}

	tb.sending, tb.sendAt = false, time.Now().Add(-time.Hour)
	tt.Frame()
	idle := tt.Image()

	// 标题上方（响应区最顶端）只可能是扫光条造成差异。
	top := int(title.Y) - 10
	b := busy.Bounds()
	blue := 0
	for y := top; y < int(title.Y); y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c1, c2 := busy.RGBAAt(x, y), idle.RGBAAt(x, y)
			if c1 != c2 && c1.B > 200 && c1.R < 120 {
				blue++
			}
		}
	}
	if blue == 0 {
		t.Fatal("发送中响应区顶部应出现蓝色扫光条")
	}
}

// 项目下拉：鼠标悬停某一行（出现导出/删除按钮）时，其它行位置不能变化。
func TestProjectDropdownHoverNoJitter(t *testing.T) {
	a, tt, be := zhApp(t)
	be.seedProject("Second", "x")
	be.seedProject("Third", "x")
	a.loadProjects()
	a.projMenu.open = true
	tt.Frame()
	second, ok1 := tt.Find("Second")
	third, ok2 := tt.Find("Third")
	if !ok1 || !ok2 {
		t.Fatalf("下拉里应列出各项目，实际 %q", tt.Texts())
	}
	// 悬停到某一行后，该行及其后面的行纵向位置都不应变化（文字被按钮挤窄只影响宽度）。
	same := func(a, b ui.Rect) bool { return a.X == b.X && a.Y == b.Y && a.H == b.H }
	tt.Move(second.X+second.W/2, second.Y+second.H/2)
	tt.Frame()
	second2, _ := tt.Find("Second")
	third2, _ := tt.Find("Third")
	if !same(second, second2) || !same(third, third2) {
		t.Fatalf("悬停后行位置发生抖动: %v→%v, %v→%v", second, second2, third, third2)
	}
	tt.Move(third.X+third.W/2, third.Y+third.H/2)
	tt.Frame()
	second3, _ := tt.Find("Second")
	third3, _ := tt.Find("Third")
	if !same(second, second3) || !same(third, third3) {
		t.Fatalf("移动悬停行后位置发生抖动: %v→%v, %v→%v", second, second3, third, third3)
	}
}

// 项目主页：主题色按钮应保持内容宽度，而不是被拉满整张卡片。
func TestThemeButtonKeepsContentWidth(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.showHome = true
	tt.Frame()
	btn, ok := tt.Find("主题色")
	if !ok {
		t.Fatalf("主页应有主题色按钮，实际 %q", tt.Texts())
	}
	img := tt.Image()
	y := int(btn.Y) + 1
	inside := img.RGBAAt(int(btn.X+btn.W/2), y)
	far := img.RGBAAt(int(btn.X+btn.W)+80, y)
	if inside == far {
		t.Fatalf("按钮右侧 80px 处仍是按钮底色，说明被拉满整行: %v %v", inside, far)
	}
}
