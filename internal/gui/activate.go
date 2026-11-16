package gui

import (
	"github.com/egoist/mygo"

	"minipost/internal/pkg/logger"
)

// windowController 是 restoreOnActivate 需要的窗口能力，*mygo.Window 天然满足，
// 单测里用假实现替换。
type windowController interface {
	IsVisible() bool
	IsMinimized() bool
	Show()
	Restore()
	Focus()
}

// restoreOnActivate 在应用被激活（例如第三方切换软件的 Option+A、Dock 图标、Cmd+Tab）时，
// 确保主窗口真的能看到：
//   - 窗口被最小化：还原（macOS 仅"取消隐藏 + 激活"并不会恢复最小化的窗口，
//     结果是应用成了前台却看不到窗口，再按一次反而把它隐藏）；
//   - 窗口被隐藏：重新显示；
//   - 其余情况只把窗口聚焦，保证键盘输入落在主窗口上。
//
// 返回本次做了什么，供日志和测试使用："restored"、"shown"、"focused"。
func restoreOnActivate(w windowController) string {
	switch {
	case w.IsMinimized():
		w.Restore()
		w.Focus()
		return "restored"
	case !w.IsVisible():
		w.Show()
		w.Focus()
		return "shown"
	default:
		w.Focus()
		return "focused"
	}
}

// watchActivation 监听应用激活事件并调用 restoreOnActivate，返回的 off 用于取消监听。
func watchActivation(w windowController) (off func()) {
	return mygo.App.OnDidBecomeActive(func() {
		action := restoreOnActivate(w)
		// 只在真正做了还原/显示时记录日志，避免每次切回应用都刷日志。
		if action != "focused" {
			logger.Info("应用被激活，已恢复主窗口", "action", action)
		}
	})
}
