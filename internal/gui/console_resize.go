package gui

import (
	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/pkg/logger"
)

const (
	consoleMinMain = 200 // 控制台上方至少留给请求区的高度
	consoleGripH   = 6   // 上边缘拖动手柄的高度（一半在面板内、一半在面板外）
)

// consoleLimits 返回在高度为 winH 的窗口里控制台面板高度的允许范围：
// 下限固定，上限保证标题栏、底栏之外至少还剩 consoleMinMain 给请求区。
// 参数：winH 窗口高度。返回：最小、最大高度（最大不会小于最小）。
//
// @author ixxxxoooo
func consoleLimits(winH float32) (lo, hi float32) {
	lo = client.ConsoleMinHeight
	hi = min(client.ConsoleMaxHeight, winH-sizeToolbar-sizeBottom-consoleMinMain)
	return lo, max(lo, hi)
}

// consoleShownHeight 是本帧控制台实际显示的高度：设置值按窗口高度夹紧。
// 设置值原样保留，窗口变高后自动恢复到用户设定的高度。
func (a *app) consoleShownHeight(c *ui.Context) float32 {
	_, winH := c.Size()
	lo, hi := consoleLimits(winH)
	return min(max(a.set.ConsoleHeight, lo), hi)
}

// consoleGrip 绘制控制台上边缘的拖动手柄：上下拖动调整面板高度（向上拖变高），
// 松手后保存；双击恢复默认高度；悬停或拖动时边缘高亮。
// 参数：shown 本帧面板显示的高度。
func (a *app) consoleGrip(c *ui.Context, shown float32) {
	p := a.p
	grip := ui.Column(c.Key("console-grip")).Absolute().Left(0).Right(0).Top(-consoleGripH / 2).Height(consoleGripH).
		Justify(ui.Center).Cursor(ui.CursorResizeNS)
	_, dy, held := grip.Dragged()
	_, winH := c.Size()
	lo, hi := consoleLimits(winH)

	switch {
	case grip.DoubleClicked():
		a.con.resizing, a.con.over = false, 0
		a.set.ConsoleHeight = min(max(client.ConsoleDefaultHeight, lo), hi)
		a.saveSettings()
		logger.Info("控制台高度已恢复默认", "height", a.set.ConsoleHeight)
	case held:
		a.con.resizing = true
		if dy != 0 {
			want := shown - dy + a.con.over
			got := min(max(want, lo), hi)
			a.con.over = want - got
			a.set.ConsoleHeight = got
		}
	case a.con.resizing:
		a.con.resizing, a.con.over = false, 0
		a.saveSettings()
		logger.Info("控制台高度已调整", "height", a.set.ConsoleHeight)
	}

	// 悬停或拖动时在边缘中间画一条 2px 强调线，提示可以拖动。
	active := held || grip.Hovered()
	grip.Children(func() {
		if active {
			ui.Box(c).FillWidth().Height(2).Background(p.accent.Alpha(0.6))
		}
	})
}
