package gui

import "time"

// scrollbarLinger 对应旧版：滚动后滚动条保留显示的时长（750ms）。
const scrollbarLinger = 750 * time.Millisecond

// noteScroll 记录一次滚动输入；「滚动条自动隐藏」开启时，停止滚动一小段时间后重绘以隐藏滚动条。
func (a *app) noteScroll() {
	if !a.set.ScrollbarAutoHide {
		return
	}
	a.scrollAt = time.Now()
	if a.native && a.win != nil {
		time.AfterFunc(scrollbarLinger+50*time.Millisecond, a.win.Invalidate)
	}
}

// scrollbarsVisible 报告本帧滚动条是否允许显示：
// 关闭自动隐藏时悬停即显示（MyGo 默认行为）；开启时只在滚动后 750ms 内显示。
func (a *app) scrollbarsVisible() bool {
	return !a.set.ScrollbarAutoHide || time.Since(a.scrollAt) < scrollbarLinger
}
