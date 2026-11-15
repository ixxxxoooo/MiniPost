package gui

import (
	"image"
	"image/color"
	"strings"
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
	if end := shimmerOffset(shimmerPeriod - time.Millisecond); end < 99 {
		t.Fatalf("终点应扫出容器右侧，实际 %v", end)
	}
	if mid := shimmerOffset(shimmerPeriod / 2); mid <= 0 || mid >= 100 {
		t.Fatalf("中点应在容器内，实际 %v", mid)
	}
	if shimmerOffset(shimmerPeriod+time.Millisecond) > 0 {
		t.Fatal("动画应按周期循环")
	}
}

// 淡入淡出：刚开始透明，随后不透明；请求很快结束时在最短展示时间的末段淡出。
func TestShimmerAlpha(t *testing.T) {
	now := time.Now()
	tb := &tab{sending: true, sendAt: now}
	if a := shimmerAlpha(tb, now); a != 0 {
		t.Fatalf("刚开始应完全透明，实际 %v", a)
	}
	if a := shimmerAlpha(tb, now.Add(shimmerFadeIn/2)); a <= 0 || a >= 1 {
		t.Fatalf("淡入中途应半透明，实际 %v", a)
	}
	if a := shimmerAlpha(tb, now.Add(time.Hour)); a != 1 {
		t.Fatalf("发送中应不透明，实际 %v", a)
	}
	tb.sending = false
	if a := shimmerAlpha(tb, now.Add(shimmerMin-shimmerFadeOut/2)); a <= 0 || a >= 1 {
		t.Fatalf("结束后的淡出中途应半透明，实际 %v", a)
	}
	if a := shimmerAlpha(tb, now.Add(shimmerMin)); a != 0 {
		t.Fatalf("展示结束时应完全透明，实际 %v", a)
	}
}

// 三个圆点依次起伏：亮度始终在 0.25~1 之间，且同一时刻互不相同。
func TestDotPulse(t *testing.T) {
	for ms := 0; ms < int(dotsPeriod/time.Millisecond); ms += 37 {
		el := time.Duration(ms) * time.Millisecond
		for i := 0; i < 3; i++ {
			if v := dotPulse(el, i); v < 0.249 || v > 1.001 {
				t.Fatalf("t=%dms 点 %d 亮度 %v 越界", ms, i, v)
			}
		}
	}
	if dotPulse(300*time.Millisecond, 0) == dotPulse(300*time.Millisecond, 1) {
		t.Fatal("相邻圆点应错开相位")
	}
}

func TestSendingElapsed(t *testing.T) {
	cases := map[time.Duration]string{
		-time.Second:            "0.0s",
		0:                       "0.0s",
		1250 * time.Millisecond: "1.2s",
		59 * time.Second:        "59.0s",
		65 * time.Second:        "1m 05s",
	}
	for in, want := range cases {
		if got := sendingElapsed(in); got != want {
			t.Errorf("sendingElapsed(%v)=%q，想要 %q", in, got, want)
		}
	}
}

// accentPixels 统计响应区顶部那几行里，与空闲帧不同且“偏向 want”的像素数。
func accentPixels(t *testing.T, tt *ui.Tester, busy, idle *image.RGBA, titleY int, want func(c color.RGBA) bool) int {
	t.Helper()
	b := busy.Bounds()
	n := 0
	for y := titleY - 12; y < titleY; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c1, c2 := busy.RGBAAt(x, y), idle.RGBAAt(x, y)
			if c1 != c2 && want(c1) {
				n++
			}
		}
	}
	return n
}

// 发送中响应区顶部应画出扫光线，颜色跟随项目主题色（默认蓝、绿色主题则偏绿），结束后消失。
func TestShimmerDrawnWhileSending(t *testing.T) {
	for _, tc := range []struct {
		name, theme string
		want        func(c color.RGBA) bool
	}{
		{"默认蓝色", "", func(c color.RGBA) bool { return c.B > 200 && c.R < 140 }},
		{"绿色主题", "#16a34a", func(c color.RGBA) bool { return c.G > c.B+30 && c.G > c.R+30 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, tt, _ := zhApp(t)
			a.openRequest(a.coll.Requests[0].ID)
			if tc.theme != "" {
				a.project().ThemeColor = tc.theme
			}
			tb := a.activeTab()
			title := ui.Rect{}
			n := 0
			// 扫光位置取决于挂钟时间；-race 下一帧可能很慢，扫光恰好扫出画面，所以最多重试几次。
			for try := 0; try < 9 && n == 0; try++ {
				phase := time.Duration(1+try) * shimmerPeriod / 10 // 每次换一个相位，覆盖整个扫光周期
				tb.sending, tb.sendAt = true, time.Now().Add(-phase)
				tt.Frame()
				busy := tt.Image()
				if try == 0 {
					snapshot(t, tt, "sending")
					if !tt.HasText("正在等待 example.com 响应首字节…") {
						t.Fatalf("发送中应显示提示，实际 %q", tt.Texts())
					}
				}
				r, ok := tt.Find("响应体")
				if !ok {
					t.Fatal("找不到响应标签")
				}
				title = r
				tb.sending, tb.sendAt = false, time.Now().Add(-time.Hour)
				tt.Frame()
				idle := tt.Image()
				n = accentPixels(t, tt, busy, idle, int(title.Y), tc.want)
			}
			if n == 0 {
				t.Fatalf("发送中响应区顶部应出现主题色扫光线（%s）", tc.name)
			}
		})
	}
}

// 发送中提示里要有实时计时。
func TestSendingIndicatorShowsElapsed(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tb := a.activeTab()
	tb.sending, tb.sendAt = true, time.Now().Add(-2300*time.Millisecond)
	tt.Frame()
	found := false
	for _, s := range tt.Texts() {
		if strings.Contains(s, "发送中… 2.") {
			found = true
		}
	}
	if !found {
		t.Fatalf("应显示约 2.3s 的计时（竞态检测下允许略有偏差），实际 %q", tt.Texts())
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

// 等待动画恰好在“按下”与“松开”之间到期时，点击不能丢：
// 动画元素是响应区的第一个子元素，消失时不能让后面的兄弟元素换位置，否则按钮的按下状态会被丢掉。
// （真实时间的 shimmerMin 窗口在 -race 这类慢环境下曾让响应区的点击测试随机失败。）
func TestClickSurvivesShimmerExpiry(t *testing.T) {
	a, tt, _, rs := sendWith(t, jsonResp())
	tb := a.activeTab()
	tb.sendAt = time.Now() // 动画仍在最短展示期内
	tt.Frame()
	r, ok := tt.Find("过滤")
	if !ok {
		t.Fatalf("应有过滤按钮，实际 %q", tt.Texts())
	}
	x, y := r.X+r.W/2, r.Y+r.H/2
	tb.sendAt = time.Now().Add(-time.Hour) // 动画在下一帧到期并从界面上消失
	tt.Press(x, y)
	tt.Release(x, y)
	if !rs.view.filterOpen {
		t.Fatal("动画在按下与松开之间到期后，点击仍应生效")
	}
}
