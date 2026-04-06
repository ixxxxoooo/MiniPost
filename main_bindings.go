//go:build bindings

package main

import (
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

// Lightweight entrypoint for bindings generation.
// Avoids running full app initialization/logging during `wails dev` binding step.
func main() {
	app := &App{}

	if err := wails.Run(&options.App{
		Bind: []interface{}{app},
	}); err != nil {
		println("Error:", err.Error())
	}
}
