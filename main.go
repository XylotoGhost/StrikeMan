// Command strikeman is a desktop manager for a private Counter-Strike 2
// server. Everything of substance lives in internal/; this only wires the
// application into a Wails window.
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"strikeman/internal/app"
)

//go:embed all:frontend
var assets embed.FS

// version is stamped in by the release build:
//
//	wails build -ldflags "-X main.version=v0.8.1"
//
// A local build stays "dev", which never offers an update.
var version = "dev"

func main() {
	strikeman := app.New(version)
	err := wails.Run(&options.App{
		Title:            "StrikeMan",
		Width:            app.PreferredWidth,
		Height:           app.PreferredHeight,
		MinWidth:         app.MinimumWidth,
		MinHeight:        app.MinimumHeight,
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 21, A: 1},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        strikeman.Startup,
		OnShutdown:       strikeman.Shutdown,
		Bind:             []interface{}{strikeman},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
