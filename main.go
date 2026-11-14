package main

import (
	"github.com/egoist/mygo"

	"minipost/internal/gui"
	"minipost/internal/pkg/logger"
)

func main() {
	if err := logger.Init(); err != nil {
		println("日志系统初始化失败:", err.Error())
	}
	defer logger.Close()

	logger.Info("MiniPost 启动中...")
	app := NewApp()
	defer app.Shutdown()
	logger.Info("App 实例创建完成")

	mygo.App.WhenReady(func() {
		win := gui.Open(app)
		logger.Info("主窗口已创建", "title", win.Title())
	})

	logger.Info("正在启动 MyGo 运行时...")
	if err := mygo.App.Run(); err != nil {
		logger.Error("MyGo 运行时退出异常", "error", err.Error())
		return
	}
	logger.Info("MiniPost 正常退出")
}
