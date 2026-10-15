package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Galaxy Explorer",
		Width:     1600,
		Height:    960,
		MinWidth:  1100,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets:     assets,
			Middleware: app.particleMiddleware,
		},
		BackgroundColour: &options.RGBA{R: 6, G: 8, B: 14, A: 1},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
		Windows: &windows.Options{
			Theme: windows.Dark,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
