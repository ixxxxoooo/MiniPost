package gui

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// 请求等待动画：响应区顶部一条跟随项目主题色的“彗星”扫光线 + 其下方柔和的光晕，
// 以及空响应区里三个依次起伏的圆点与实时计时。
const (
	shimmerPeriod  = 1700 * time.Millisecond // 一轮扫光时长
	shimmerMin     = 600 * time.Millisecond  // 请求再快也至少展示这么久，避免一闪而过
	shimmerFadeIn  = 220 * time.Millisecond  // 开始时淡入
	shimmerFadeOut = 260 * time.Millisecond  // 结束前淡出
	shimmerH       = 2                       // 扫光线高度
	shimmerGlowH   = 7                       // 光晕高度
	shimmerWidth   = 40                      // 彗星宽度占容器的百分比
	dotsPeriod     = 1200 * time.Millisecond // 三个圆点一个起伏周期
)

// shimmerActive 判断标签页此刻是否应显示等待动画。
func shimmerActive(t *tab, now time.Time) bool {
	if t.sending {
		return true
	}
	return !t.sendAt.IsZero() && now.Sub(t.sendAt) < shimmerMin
}

// shimmerOffset 返回彗星左边缘位置（容器宽度的百分比），随时间做 ease-in-out 循环：
// 从完全在左侧之外扫到完全在右侧之外。
func shimmerOffset(elapsed time.Duration) float32 {
	phase := float64(elapsed%shimmerPeriod) / float64(shimmerPeriod)
	e := phase * phase * (3 - 2*phase)
	return float32((-1 + (100.0/shimmerWidth+1)*e) * shimmerWidth)
}

// shimmerAlpha 是整体不透明度：开始 shimmerFadeIn 内淡入；请求已结束时，在最短展示时间的最后 shimmerFadeOut 内淡出。
func shimmerAlpha(t *tab, now time.Time) float32 {
	el := now.Sub(t.sendAt)
	a := float32(1)
	if el < shimmerFadeIn {
		a = float32(el) / float32(shimmerFadeIn)
	}
	if !t.sending {
		if left := shimmerMin - el; left < shimmerFadeOut {
			a = float32(math.Min(float64(a), float64(left)/float64(shimmerFadeOut)))
		}
	}
	return float32(math.Max(0, math.Min(1, float64(a))))
}

// sendShimmer 在响应区顶部绘制扫光线与光晕；不在等待中则什么也不画。颜色取自当前项目主题色。
func (a *app) sendShimmer(c *ui.Context, t *tab) {
	now := time.Now()
	if !shimmerActive(t, now) {
		return
	}
	c.AnimationFrame()
	acc := a.p.accent
	left := shimmerOffset(now.Sub(t.sendAt))
	comet := func(peak float32, h float32) ui.Element {
		return ui.Box(c).Absolute().Top(0).Height(h).LeftPercent(left).WidthPercent(shimmerWidth).LinearGradient(ui.LinearGradient{
			Angle: 90,
			// 尾巴长而淡，头部亮而集中：像彗星从左向右掠过
			Stops: []ui.GradientStop{
				{Color: acc.Alpha(0), At: 0},
				{Color: acc.Alpha(peak * 0.25), At: 0.45},
				{Color: acc.Alpha(peak), At: 0.92},
				{Color: acc.Alpha(0), At: 1},
			},
		})
	}
	ui.Box(c.Key("shimmer")).Absolute().Top(0).Left(0).Right(0).Height(shimmerGlowH).Clip().Opacity(shimmerAlpha(t, now)).Children(func() {
		// 光晕：线下方一层向下渐隐的主题色
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(shimmerGlowH).LinearGradient(ui.LinearGradient{
			Angle: 0,
			Stops: []ui.GradientStop{{Color: acc.Alpha(0.10), At: 0}, {Color: acc.Alpha(0), At: 1}},
		})
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(shimmerH).Background(acc.Alpha(0.14))
		comet(0.95, shimmerH)
		comet(0.22, shimmerGlowH) // 头部下方的光晕随之移动
	})
}

// dotPulse 返回第 i 个圆点此刻的亮度（0.25~1）：三个点错开 1/6 周期依次起伏。
func dotPulse(elapsed time.Duration, i int) float32 {
	phase := math.Mod(float64(elapsed%dotsPeriod)/float64(dotsPeriod)-float64(i)/6, 1)
	if phase < 0 {
		phase += 1
	}
	return float32(0.25 + 0.75*(0.5-0.5*math.Cos(phase*2*math.Pi)))
}

// sendingIndicator 是响应区中央的等待提示：三个主题色圆点 + 文案 + 已用时间。
func (a *app) sendingIndicator(c *ui.Context, t *tab) {
	p := a.p
	elapsed := time.Since(t.sendAt)
	if t.sendAt.IsZero() {
		elapsed = 0
	}
	c.AnimationFrame()
	ui.Row(c.Key("sending")).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			for i := 0; i < 3; i++ {
				ui.Box(c).Size(7, 7).Radius(3.5).Background(p.accent.Alpha(dotPulse(elapsed, i)))
			}
		})
		txt(c, a.t("发送请求...", "Sending request..."), fsLarge, p.fg2, 0)
		txtMono(c, sendingElapsed(elapsed), fsSmall, p.muted, 0)
	})
}

// sendingElapsed 把已用时间写成 0.0s / 12.3s / 1m 05s，位数稳定，不会让整行抖动。
func sendingElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	s := int(d.Seconds())
	return fmt.Sprintf("%dm %02ds", s/60, s%60)
}
