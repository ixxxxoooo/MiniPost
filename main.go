package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

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
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
		Frameless: true,
		Mac: &mac.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "MiniPost",
				Message: "轻量级 HTTP API 调试工具",
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
