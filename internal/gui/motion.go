package gui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// 动效令牌：与设计稿 docs/design/assets/tokens.css 的 --dur-* / --ease-* 一一对应。
// 修改这里的数值之前，必须先改设计稿的令牌（tokens_sync_test.go 会核对）。
const (
	durFast  = 120 * time.Millisecond // --dur-fast：悬停、颜色变化
	durBase  = 180 * time.Millisecond // --dur-base：标签内容切换、浮层淡入
	durSlow  = 280 * time.Millisecond // --dur-slow：列表行入场
	durXL    = 380 * time.Millisecond // --dur-xl：Toast、徽章弹入
	durSlide = 420 * time.Millisecond // fx.js：滑动指示器（分段滑块、下划线、标签顶线）
)

var (
	easeOut    = ui.CubicBezier(.2, .8, .2, 1)    // --ease-out
	easeEmph   = ui.CubicBezier(.22, 1, .36, 1)   // --ease-emph：入场与指示器，先快后缓
	easeSpring = ui.CubicBezier(.34, 1.4, .64, 1) // --ease-spring：轻微过冲
	easeSoft   = ui.CubicBezier(.3, 1.2, .5, 1)   // --ease-soft：大面积元素
	easeSlide  = ui.CubicBezier(.34, 1.28, .5, 1) // fx.js 里指示器滑动用的曲线
)

// transColors 让元素的底色、边框色在状态变化（悬停、选中）时以 --dur-fast 平滑过渡。
var transColors = ui.ElementTransition{Duration: durFast, Ease: easeOut, Colors: true}

// transSlide 让指示器在选项之间滑动（位置 + 宽高），对应设计稿 fx.js 的 FLIP 滑动。
var transSlide = ui.ElementTransition{Duration: durSlide, Ease: easeSlide, Position: true, Size: true}

// 入场动作：对应设计稿的 popIn 关键帧（浮层、菜单、对话框）。
// paneIn / rowIn / toastIn 暂由各自组件内的渐显与滑入实现（见 toast.go），需要时再在这里补充令牌。
var motionPop = &ui.Motion{Y: 6, Scale: 0.97}

// enterTransition 返回带入场（和可选退场）的过渡；delay 用于列表逐行错开（设计稿 --i × 24ms）。
func enterTransition(in *ui.Motion, dur time.Duration, delay time.Duration) ui.ElementTransition {
	return ui.ElementTransition{
		Duration: dur, Ease: easeOut, Colors: true,
		Animation: &ui.Animation{Duration: dur, Ease: easeOut, Delay: delay},
		Enter:     in,
	}
}

// slideRect 把指示器在容器内的位置换算成相对容器的矩形：active 是活动项、parent 是容器，
// 都取自上一帧的边界（Element.Bounds）；任一为空（首帧）时返回 ok=false，由调用方退回到静态绘制。
func slideRect(parent, active ui.Rect) (r ui.Rect, ok bool) {
	if parent.W <= 0 || active.W <= 0 {
		return ui.Rect{}, false
	}
	return ui.Rect{X: active.X - parent.X, Y: active.Y - parent.Y, W: active.W, H: active.H}, true
}
