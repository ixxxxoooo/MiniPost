// Package gui 实现 MiniPost 的 MyGo 原生界面（macOS）。
package gui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
)

const (
	// trafficLightX 是红绿灯关闭按钮左边缘到窗口左边缘的距离（px）。
	trafficLightX = 12
	// trafficLightY 是关闭按钮上边缘到窗口上边缘的距离（px）。
	// 标题栏内容行高 sizeToolbar=46，图标行垂直中心在 23；系统红绿灯高 14，
	// 上边缘取 16 使其中心在 23，与侧栏开关等图标处于同一水平线。
	trafficLightY = 16
)

// Open 创建主窗口并返回。
func Open(be Backend) *mygo.Window {
	a := newApp(be)
	a.native = true
	a.restartBackupTimer()
	a.startUpdater()
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:     "MiniPost",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		StateKey:  "main",
		// 隐藏系统标题栏，由界面自绘 46px 透明标题栏；红绿灯垂直中心对齐标题栏图标行（见 trafficLightY）。
		TitleBarStyle:        mygo.TitleBarHidden,
		TrafficLightPosition: &mygo.Point{X: trafficLightX, Y: trafficLightY},
		Content:              ui.View(a.view),
	})
	a.win = win
	// 关闭窗口或退出前保存标签会话（含未保存的修改），下次启动时恢复。
	win.OnClose(func(*mygo.CloseEvent) { a.persistSession() })
	// 被第三方软件（如 Manico 的 Option+A）激活时，最小化或隐藏的主窗口要自动还原。
	watchActivation(win)
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		a.persistSession()
		a.stopUpdater()
	})
	logger.Info("原生主窗口已创建")
	return win
}
