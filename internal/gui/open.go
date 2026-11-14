// Package gui 实现 MiniPost 的 MyGo 原生界面（macOS）。
package gui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
)

// Open 创建主窗口并返回。
func Open(be Backend) *mygo.Window {
	m := newModel(be)
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:     "MiniPost",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		Content:   ui.View(m.view),
	})
	m.win = win
	logger.Info("原生主窗口已创建")
	return win
}
