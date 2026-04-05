package main

import (
	"embed"

	"minipost/internal/pkg/logger"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := logger.Init(); err != nil {
		println("日志系统初始化失败:", err.Error())
	}
	defer logger.Close()

	logger.Info("MiniPost 启动中...")

	app := NewApp()
	logger.Info("App 实例创建完成")

	logger.Info("正在启动 Wails 运行时...")
	err := wails.Run(&options.App{
		Title:            "MiniPost",
		Width:            1280,
		Height:           800,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
		Frameless: true,
		Mac: &mac.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "MiniPost",
				Message: "轻量级 HTTP API 调试工具",
			},
		},
	})

	if err != nil {
		logger.Error("Wails 运行时退出异常", "error", err.Error())
	} else {
		logger.Info("MiniPost 正常退出")
	}
}
