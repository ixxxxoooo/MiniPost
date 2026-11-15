package gui

import "github.com/egoist/mygo/ui"

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
		return e.Tooltip(s)
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
