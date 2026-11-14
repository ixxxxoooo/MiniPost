package gui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// 请求等待动画（同旧版响应区顶部的 shimmer 条）。
const (
	shimmerPeriod = 1800 * time.Millisecond // 一轮扫光时长
	shimmerMin    = 450 * time.Millisecond  // 请求再快也至少展示这么久，避免一闪而过
	shimmerH      = 3                       // 条高
	shimmerWidth  = 34                      // 扫光条宽度占比（%）
)

// shimmerActive 判断标签页此刻是否应显示等待动画。
func shimmerActive(t *tab, now time.Time) bool {
	if t.sending {
		return true
	}
	return !t.sendAt.IsZero() && now.Sub(t.sendAt) < shimmerMin
}

// shimmerOffset 返回扫光条左边缘位置（容器宽度的百分比），随时间做 ease-in-out 循环：
// 从 -1.3 个条宽扫到 3.6 个条宽之外，对应旧版 keyframes。
func shimmerOffset(elapsed time.Duration) float32 {
	phase := float64(elapsed%shimmerPeriod) / float64(shimmerPeriod)
	e := phase * phase * (3 - 2*phase)
	return float32((-1.3 + 4.9*e) * shimmerWidth)
}

// sendShimmer 在响应区顶部绘制 3px 的蓝色扫光条；不在等待中则什么也不画。
func (a *app) sendShimmer(c *ui.Context, t *tab) {
	now := time.Now()
	if !shimmerActive(t, now) {
		return
	}
	c.AnimationFrame()
	p := a.p
	left := shimmerOffset(now.Sub(t.sendAt))
	blue := func(alpha float32) ui.Color { return ui.RGBA(0, 122, 255, alpha) }
	ui.Row(c.Key("shimmer")).Absolute().Top(0).Left(0).Right(0).Height(shimmerH).Clip().Background(p.border).Children(func() {
		ui.Box(c).Absolute().Top(0).Bottom(0).LeftPercent(left).WidthPercent(shimmerWidth).LinearGradient(ui.LinearGradient{
			Angle: 90,
			Stops: []ui.GradientStop{
				{Color: blue(0), At: 0},
				{Color: blue(0.45), At: 0.4},
				{Color: blue(0.95), At: 0.5},
				{Color: blue(0.45), At: 0.6},
				{Color: blue(0), At: 1},
			},
		})
	})
}
