package gui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// 提示（toast）类型：决定图标、颜色和停留时间。
const (
	toastInfo    = "info"
	toastSuccess = "success"
	toastWarn    = "warning"
	toastError   = "error"
)

const (
	toastFade     = 180 * time.Millisecond // 进出场渐变时长
	toastSlide    = 14                     // 进场时从右侧滑入的距离（px）
	toastMinW     = 220                    // 提示卡片最小宽度
	toastMaxW     = 380                    // 提示卡片最大宽度，更长的文字自动换行
	toastEdge     = 12                     // 与窗口右边缘的距离
	toastTopGap   = 8                      // 与标题栏下沿的距离
	toastIconSize = 15
	toastRowH     = 22 // 图标、按钮、关闭按钮所在的一行高度
)

// toastNote 是一条待显示的提示。
type toastNote struct {
	msg, kind string
}

// toastAct 是一条带按钮的提示（撤销、更新等）。
type toastAct struct {
	msg, label string
	fn         func()
}

// toastTimeout 返回各类型提示的停留时间：成功最短，错误最长，便于读完。
func toastTimeout(kind string) time.Duration {
	switch kind {
	case toastSuccess:
		return 3 * time.Second
	case toastWarn:
		return 5 * time.Second
	case toastError:
		return 7 * time.Second
	}
	return 4 * time.Second
}

// toast 登记一条普通提示，下一帧显示在窗口右上角。
// 参数：msg 提示文字。
func (a *app) toast(msg string) { a.pushToast(msg, toastInfo) }

// toastOK 登记一条成功提示（已复制、已保存等）。
func (a *app) toastOK(msg string) { a.pushToast(msg, toastSuccess) }

// toastWarn 登记一条警告提示（输入不完整、对象不存在等）。
func (a *app) toastWarn(msg string) { a.pushToast(msg, toastWarn) }

// toastErr 登记一条错误提示（操作失败）。
func (a *app) toastErr(msg string) { a.pushToast(msg, toastError) }

// pushToast 把提示加入队列；非法类型按普通提示处理。
func (a *app) pushToast(msg, kind string) {
	switch kind {
	case toastSuccess, toastWarn, toastError:
	default:
		kind = toastInfo
	}
	a.toastMsgs = append(a.toastMsgs, toastNote{msg, kind})
	a.redraw()
}

// toastAction 登记一条带按钮的提示，下一帧显示；点按钮时在本帧构建结束后执行 fn。
func (a *app) toastAction(msg, label string, fn func()) {
	a.toastActs = append(a.toastActs, toastAct{msg, label, fn})
	a.redraw()
}

// showToasts 把登记的提示交给 MyGo，并按自绘样式在右上角画出所有正在显示的提示。
func (a *app) showToasts(c *ui.Context) {
	for _, n := range a.toastMsgs {
		c.AddToast(ui.Toast{ID: n.msg, Title: n.msg, Type: n.kind, Timeout: toastTimeout(n.kind)})
	}
	a.toastMsgs = nil
	for _, t := range a.toastActs {
		fn := t.fn
		c.AddToast(ui.Toast{ID: t.msg, Title: t.msg, Type: toastInfo, Timeout: 8 * time.Second,
			Action: t.label, OnAction: func() { a.later(fn) }})
	}
	a.toastActs = nil
	a.toastViewport(c)
}

// toastViewport 在标题栏下方的右上角竖向排列提示，最新的在最上面。
func (a *app) toastViewport(c *ui.Context) {
	ui.ToastViewportBase(c, func(vp ui.Element, toasts []ui.Toast) {
		vp.Padding(sizeToolbar+toastTopGap, toastEdge, toastEdge, toastEdge).AlignItems(ui.End).Justify(ui.Start).Gap(8)
		for i := len(toasts) - 1; i >= 0; i-- {
			a.toastCard(c, toasts[i])
		}
	})
}

// toastIcon 返回类型对应的图标名与颜色。
func (a *app) toastIcon(kind string) (string, ui.Color) {
	p := a.p
	switch kind {
	case toastSuccess:
		return "toastSuccess", p.success
	case toastWarn:
		return "toastWarn", p.warning
	case toastError:
		return "toastError", p.danger
	}
	return "info", p.accent
}

// toastCard 绘制一张提示卡片：类型图标、文字（可换行）、可选操作按钮与关闭按钮；
// 进场时从右侧滑入并渐显，结束前渐隐；鼠标停在上面时暂停计时。
func (a *app) toastCard(c *ui.Context, t ui.Toast) {
	p := a.p
	tp := ui.ToastBase(c, t)
	ic, col := a.toastIcon(t.Type)

	opacity, shift := float32(1), float32(0)
	left := tp.Left()
	age := t.Timeout - left // Timeout 总是非零，Left 为 0 表示不会自动关闭
	switch {
	case tp.Root.Hovered():
	case left > 0 && age < toastFade:
		k := float32(age) / float32(toastFade)
		opacity, shift = k, (1-k)*toastSlide
		c.AnimationFrame()
	case left > 0 && left < toastFade:
		opacity = float32(left) / float32(toastFade)
		c.AnimationFrame()
	case left > 0:
		c.After(left - toastFade)
	}

	winW, _ := c.Size()
	// 图标、文字、按钮顶部对齐：长消息换行后图标和关闭按钮仍贴着第一行。
	card := tp.Root.Row().AlignItems(ui.Start).Gap(10).Padding(8, 8, 8, 12).Radius(8).
		MinWidth(toastMinW).MaxWidth(min(toastMaxW, winW-2*toastEdge)).
		Background(p.elevated()).Border(1, p.btnBorder).Shadow(0, 6, 20, 0, p.shadowLg()).
		Opacity(opacity).Translate(shift, 0)
	card.Children(func() {
		ui.Row(c).Width(toastIconSize).Height(toastRowH).Shrink(0).AlignItems(ui.Center).Children(func() {
			icon(c, ic, toastIconSize, col)
		})
		ui.Text(c, t.Title).FontSize(fsSmall).TextColor(p.fg).FixedLineHeight(fsSmall * 1.5).
			Grow(1).Basis(0).MinWidth(0).PaddingY((toastRowH - fsSmall*1.5) / 2)
		if t.Action != "" {
			b := tp.ActionButton().Padding(0, 8).Height(toastRowH).Radius(radiusBtn).Shrink(0).AlignItems(ui.Center)
			if b.Hovered() {
				b = b.Background(p.sideActive)
			}
			b.Children(func() { txt(c, t.Action, fsSmall, p.accent, 600) })
		}
		x := tp.CloseButton().Label(a.t("关闭", "Close")).Size(toastRowH, toastRowH).Radius(radiusBtn).Shrink(0).
			AlignItems(ui.Center).Justify(ui.Center)
		xc := p.muted
		if x.Hovered() {
			x, xc = x.Background(p.sideHover), p.fg2
		}
		x.Children(func() { icon(c, "clear", 10, xc) })
	})
}
