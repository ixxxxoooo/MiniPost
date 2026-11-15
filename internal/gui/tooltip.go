package gui

import "github.com/egoist/mygo/ui"

// tip 给元素加悬浮提示（设计稿 .tip）：停留 tipDelay 后出现在元素正下方、居中，下方放不下时自动翻到上方；
// 深色圆角胶囊、11px 字，自上轻落入场。不用 Element.Tooltip，因为它固定摆在元素上方：
// 标签栏、工具栏贴着窗口顶部，提示会盖住标题栏，和设计稿不一致。
// 同时把提示文字作为无障碍描述。s 为空时原样返回。
func (a *app) tip(c *ui.Context, e ui.Element, s string) ui.Element {
	if s == "" {
		return e
	}
	e.Description(s)
	p := a.p
	ui.TooltipBase(c, e, func(t ui.Element) {
		t.MaxWidth(280).Padding(4, 8).Radius(6).Background(p.tipFill()).TextColor(p.surface).FontSize(fsTipText).
			Shadow(0, 4, 14, -2, ui.RGBA(0, 0, 0, 0.35)).
			AttachTo(e, ui.AnchorBottom, ui.AnchorTop).Margin(6, 0, 6, 0).
			Transition(enterWith(motionTip, durBase, 0, easeOut))
		t.Children(func() { ui.Text(c, s) })
	})
	return e
}

// fsTipText 是悬浮提示的字号（设计稿 .tip font-size: 11px）。
const fsTipText float32 = 11

// textOverflows 报告 s 以给定字号和字重单行显示时，是否放不进 avail 宽度（也就是会被省略号截断）。
// 用真实的文字排版量宽，中英文混排也准确。
func (a *app) textOverflows(c *ui.Context, s string, size float32, weight int, avail float32) bool {
	if s == "" || avail <= 0 {
		return s != ""
	}
	w, _ := c.MeasureText(0, ui.Span{Text: s, Size: size, Weight: weight})
	return w > avail
}

// tipIfLong 在文字会被截断时给元素加上显示全文的悬停提示；放得下时原样返回，避免无意义的提示。
func (a *app) tipIfLong(c *ui.Context, e ui.Element, s string, size float32, weight int, avail float32) ui.Element {
	if a.textOverflows(c, s, size, weight, avail) {
		return a.tip(c, e, s)
	}
	return e
}

// 侧栏、标签等行里，名称之外固定占用的水平空间（用来估算名称可用宽度，宁可偏小：多给提示无害）。
const (
	sideRowHoverIcons = 36 // 悬停时右侧出现的“新建”和“···”按钮
	sideFolderChrome  = 60 // 文件夹行：外边距 8 + 内边距 16 + 箭头/图标/间距 36
	sideRequestChrome = 68 // 请求行：外边距 8 + 内边距 28 + 方法标签 28 + 间距 4
	sideDepthIndent   = 21 // 每一层嵌套的缩进（引导线容器左缩进 16 + 线 1 + 行外边距 4）
	sideHistoryChrome = 170
)

// sideNameRoom 返回侧栏行里名称可用的宽度。
func (a *app) sideNameRoom(chrome float32, depth int, hover bool) float32 {
	room := a.sidebarShown() - chrome - float32(depth)*sideDepthIndent
	if hover {
		room -= sideRowHoverIcons
	}
	return room
}

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// sidebarShown 是侧栏当前实际显示的宽度（窗口很窄时比设置值小）。
func (a *app) sidebarShown() float32 {
	if a.side.shown > 0 {
		return a.side.shown
	}
	return a.set.SidebarWidth
}
