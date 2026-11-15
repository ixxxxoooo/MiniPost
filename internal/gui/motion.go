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

// 入场动作：对应设计稿 v2 的关键帧。
// popIn（浮层、菜单）：自下 8px、缩放 .96 → 原位；dlgIn（对话框）更大一些：自下 14px、缩放 .955；
// itemIn（浮层里的条目）：自下 5px，逐条错开；paneIn（面板切换）：自下 6px。toastIn 在 toast.go 里随卡片自绘。
var (
	motionPop = &ui.Motion{Y: 8, Scale: 0.96, Blur: 3}
	motionDlg = &ui.Motion{Y: 14, Scale: 0.955, Blur: 4}
	// motionQuick 是快速打开面板的入场（设计稿 qkIn）：自顶部落下 10px，缩放 .97，带轻微模糊。
	motionQuick = &ui.Motion{Y: -10, Scale: 0.97, Blur: 4}
	// motionTip 是悬浮提示的入场（设计稿 tipIn）：自上 3px 落下，缩放 .96。
	motionTip  = &ui.Motion{Y: -3, Scale: 0.96}
	motionItem = &ui.Motion{Y: 5}
	motionPane = &ui.Motion{Y: 6}
	// motionPaneOut 是面板退场（设计稿 paneOut：上移 4px 淡出）。
	motionPaneOut = &ui.Motion{Y: -4}
)

// durPaneOut 是面板退场时长；退场结束后新面板才进场（fx.js：退场幽灵 110ms 后 paneIn）。
const durPaneOut = 110 * time.Millisecond

// tipDelay 是鼠标停在元素上多久后出现悬浮提示（设计稿 overlays.js：450ms）。
const tipDelay = 450 * time.Millisecond

// enterTransition 返回带入场（和可选退场）的过渡；delay 用于列表逐行错开（设计稿 --i × 24ms）。
func enterTransition(in *ui.Motion, dur time.Duration, delay time.Duration) ui.ElementTransition {
	return enterWith(in, dur, delay, easeSoft)
}

// enterWith 同 enterTransition，但可指定曲线（popIn 用 ease-soft，itemIn 用 ease-emph）。
func enterWith(in *ui.Motion, dur, delay time.Duration, ease ui.Easing) ui.ElementTransition {
	return ui.ElementTransition{
		Duration: dur, Ease: ease, Colors: true,
		Animation: &ui.Animation{Duration: dur, Ease: ease, Delay: delay},
		Enter:     in,
	}
}

// itemEnter 是浮层里第 i 条的入场（设计稿 itemIn：delay = min(i,10)×18ms + 30ms）。
func itemEnter(i int) ui.ElementTransition {
	return enterWith(motionItem, durSlow, itemDelay(i), easeEmph)
}

// itemDelay 返回第 i 条的错开延迟，供测试核对设计稿公式。
func itemDelay(i int) time.Duration { return time.Duration(min(i, 10)*18+30) * time.Millisecond }

// slideRect 把指示器在容器内的位置换算成相对容器的矩形：active 是活动项、parent 是容器，
// 都取自上一帧的边界（Element.Bounds）；任一为空（首帧）时返回 ok=false，由调用方退回到静态绘制。
func slideRect(parent, active ui.Rect) (r ui.Rect, ok bool) {
	if parent.W <= 0 || active.W <= 0 {
		return ui.Rect{}, false
	}
	return ui.Rect{X: active.X - parent.X, Y: active.Y - parent.Y, W: active.W, H: active.H}, true
}

// paneTransition 是“先退出、再进入”的面板切换（设计稿 fx.js 的 ghostPanes + paneIn）：
// 旧面板用 paneOut 在 110ms 内淡出，新面板等它离开后用 paneIn 淡入上移。只作用于透明度与位移，不动画尺寸。
func paneTransition() ui.ElementTransition {
	if !motionEnabled {
		return ui.ElementTransition{Duration: -1} // 关闭动效：直接显示，不留退场副本
	}
	return ui.ElementTransition{
		Duration: durSlow, Ease: easeEmph, Colors: true,
		Enter: motionPane, Exit: motionPaneOut,
		ExitDuration: durPaneOut, ExitEase: easeOut,
		AfterExits: true,
	}
}

// pane 在 key 对应的容器里绘制一个可切换的面板：key 变化即触发上面的退出 / 进入动画。
func (a *app) pane(c *ui.Context, key string, fn func()) {
	ui.Column(c.Key("pane:" + key)).Grow(1).Basis(0).Transition(paneTransition()).Children(fn)
}

// motionEnabled 控制面板切换“先退后进”与对话框两段式入场是否开启。单元测试里关闭它（切换后立刻要能读到新内容），
// 专门验证动画的测试再单独打开。
var motionEnabled = true
