// Package gui 实现 MiniPost 的 MyGo 原生界面（macOS）。
package gui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
)

// Open 创建主窗口并返回。
func Open(be Backend) *mygo.Window {
	a := newApp(be)
	a.native = true
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:     "MiniPost",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		StateKey:  "main",
		Content:   ui.View(a.view),
	})
	a.win = win
	// 关闭窗口或退出前保存标签会话（含未保存的修改），下次启动时恢复。
	win.OnClose(func(*mygo.CloseEvent) { a.persistSession() })
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { a.persistSession() })
	logger.Info("原生主窗口已创建")
	return win
}
