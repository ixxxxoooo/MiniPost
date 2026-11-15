package gui

import (
	"github.com/egoist/mygo/ui"
)

// 本文件是「流光」外壳（设计稿方案 C）：极光背景、透明标题栏中央的悬浮胶囊、半透明圆角侧栏与浮起的圆角内容卡片。

// 流光的布局令牌（设计稿 .win[data-dir=c]）：卡片圆角 16、行高 27、标题栏 46、标签栏 38、外壳间距 8。
const (
	radiusLg   = 16 // 侧栏右侧与内容卡片的大圆角（--r-lg）
	radiusSm   = 8  // 图标按钮、小块圆角（--r-sm）
	sizeRow    = 27 // 树/列表行高（--row）
	shellGap   = 8  // 侧栏与内容卡片、卡片与窗口边缘的间距
	sizeIb     = 26 // 标题栏图标按钮（.ib）
	sizeIsland = 32 // 悬浮胶囊高度
)

// pinkGlow 是极光里与强调色混合的粉色（设计稿 #ff8bd0）。
var pinkGlow = ui.Hex("#ff8bd0")

// aurora 绘制窗口背景的极光：左上与右下两团大范围渐变光，再叠两团缓慢漂移的光斑。
// 设计稿用椭圆径向渐变（900×520 等），这里用圆形渐变加纵向缩放得到同样的椭圆。
// 它们是绝对定位的图层，不占布局空间，必须放在根容器的第一个子元素位置。
func (a *app) aurora(c *ui.Context, w, h float32) {
	p := a.p
	ac := p.accent
	// glow 画一团以 (cx,cy) 为中心、半径 rx×ry 的椭圆光，中心为 col，边缘透明。
	glow := func(key string, cx, cy, rx, ry float32, col ui.Color) {
		ui.Box(c.Key(key)).Absolute().Left(cx-rx).Top(cy-rx).Size(2*rx, 2*rx).
			RadialGradient(ui.RadialGradient{
				Stops:  []ui.GradientStop{{Color: col, At: 0}, {Color: col.Alpha(0), At: 1}},
				Radius: rx,
				Oklab:  true,
			}).Scale(1, ry/rx)
	}
	// 两团大范围光：900×520 @ (8%,-10%) 与 700×480 @ (100%,110%)，渐变在 62%/60% 处变透明。
	glow("aurora-1", 0.08*w, -0.10*h, 900*0.62, 520*0.62, ac.Alpha(0.26))
	glow("aurora-2", w, 1.10*h, 700*0.60, 480*0.60, ac.Mix(pinkGlow, 0.526).Alpha(0.38))
	// 两团光斑：设计稿加了 blur(46px)、opacity .55，并在 22s/26s 里缓慢漂移；这里取其静止位置的峰值强度。
	glow("aurora-3", 160, 70, 280, 210, ac.Alpha(0.38*0.55))
	glow("aurora-4", w-120, h-50, 260, 210, ac.Mix(pinkGlow, 0.6).Alpha(0.5*0.55))
}

// ib 是标题栏的 26×26 图标按钮（设计稿 .ib）：悬停浅底；on 为强调色淡底；disabled 时变淡且不可点。
// 点击后通过 a.later 在帧外执行 act。
func (a *app) ib(c *ui.Context, key, iconName, tip string, on, disabled bool, act func()) {
	p := a.p
	col := p.fg2
	switch {
	case disabled:
		col = p.fg2.Alpha(0.35)
	case on:
		col = p.accent
	}
	b := ui.ButtonBase(c.Key(key)).Size(sizeIb, sizeIb).Radius(radiusSm).AlignItems(ui.Center).Justify(ui.Center).Shrink(0)
	switch {
	case on:
		b = b.Background(p.acSoft).Border(1, p.acSoft2)
	case b.Hovered() && !disabled:
		b = b.Background(p.hover)
	}
	if tip != "" {
		b = b.Tooltip(tip).Label(tip)
	}
	if disabled {
		b = b.Disabled(true)
	}
	b.Children(func() { icon(c, iconName, 15, col) })
	if b.Clicked() {
		a.later(act)
	}
}

// ibSm 是 22×22 的小图标按钮（设计稿 .ib.sm）：图标 13，悬停浅底；disabled 时图标变淡且不可点。
func (a *app) ibSm(c *ui.Context, key, iconName, tip string, disabled bool, act func()) {
	a.ibSmOn(c, key, iconName, tip, false, disabled, act)
}

// ibSmOn 在 ibSm 基础上支持选中态（on：强调色淡底 + 细描边，图标用强调色，对应 .ib.on）。
func (a *app) ibSmOn(c *ui.Context, key, iconName, tip string, on, disabled bool, act func()) {
	p := a.p
	col := p.fg2
	switch {
	case disabled:
		col = p.fg2.Alpha(0.35)
	case on:
		col = p.accent
	}
	b := ui.ButtonBase(c.Key(key)).Size(22, 22).Radius(radiusSm).AlignItems(ui.Center).Justify(ui.Center).Shrink(0)
	switch {
	case on:
		b = b.Background(p.acSoft).Border(1, p.acSoft2)
	case b.Hovered() && !disabled:
		b = b.Background(p.hover)
	}
	if tip != "" {
		b = b.Tooltip(tip).Label(tip)
	}
	if disabled {
		b = b.Disabled(true)
	}
	b.Children(func() { icon(c, iconName, 13, col) })
	if b.Clicked() {
		a.later(act)
	}
}

// island 是标题栏正中的悬浮胶囊（设计稿 .island）：项目 | 环境 | 搜索。
// 它相对整条标题栏水平居中，而不是夹在左右两组按钮之间。
func (a *app) island(c *ui.Context) {
	p := a.p
	ui.Row(c.Key("island-wrap")).Absolute().Top(0).Bottom(0).Left(0).Right(0).AlignItems(ui.Center).Justify(ui.Center).PassThrough().Children(func() {
		ui.Row(c.Key("island")).Height(sizeIsland).Padding(3).Gap(2).Radius(sizeIsland/2).AlignItems(ui.Center).
			Background(p.surface.Alpha(0.70)).
			Shadow(0, 0, 0, 1, p.border).Shadow(0, 1, 2, 0, ui.RGBA(20, 24, 40, 0.06)).Children(func() {
			a.projectButton(c)
			ui.Box(c).Width(1).Height(14).Background(p.border).Shrink(0).MarginX(3)
			a.envPill(c)
			ui.Box(c).Width(1).Height(14).Background(p.border).Shrink(0).MarginX(3)
			a.searchPill(c)
		})
	})
}

// searchPill 是胶囊里的搜索入口（设计稿 .cmdk2）：150px 宽的浅底胶囊，点击打开快速打开面板。
func (a *app) searchPill(c *ui.Context) {
	p := a.p
	b := ui.ButtonBase(c.Key("tb-quick")).Width(150).Height(sizeIsland - 6).Radius((sizeIsland - 6) / 2).PaddingStart(8).PaddingEnd(10).Gap(6).
		AlignItems(ui.Center).Shrink(0).Tooltip(a.t("快速打开", "Quick open") + " (⌘K)").Label(a.t("搜索", "Search"))
	if b.Hovered() {
		b = b.Background(p.press)
	} else {
		b = b.Background(p.hover)
	}
	b.Children(func() {
		icon(c, "search", 13, p.muted)
		txtE(c, a.t("搜索", "Search"), fsBase, p.muted, 0).Grow(1).Basis(0)
		txtMono(c, "⌘K", fsTiny, p.muted, 0)
	})
	if b.Clicked() {
		a.later(a.openQuick)
	}
}
