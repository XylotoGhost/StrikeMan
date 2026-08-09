package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend
var assets embed.FS

// version is stamped in by the release build:
//
//	wails build -ldflags "-X main.version=v0.8.0"
//
// A local build stays "dev", which never offers an update.
var version = "dev"

// Preferred window size, and the smallest the layout still works at.
const (
	preferredWidth  = 1200
	preferredHeight = 1000
	minimumWidth    = 820
	minimumHeight   = 560
)

func main() {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:            "StrikeMan",
		Width:            preferredWidth,
		Height:           preferredHeight,
		MinWidth:         minimumWidth,
		MinHeight:        minimumHeight,
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 21, A: 1},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}

// fitWindowToScreen shrinks the window when the preferred size does not fit
// the display — on a 1366x768 laptop a 1000px tall window would otherwise
// open partly off screen.
func fitWindowToScreen(ctx context.Context) {
	screens, err := runtime.ScreenGetAll(ctx)
	if err != nil || len(screens) == 0 {
		return
	}
	screen := screens[0]
	for _, s := range screens {
		if s.IsCurrent {
			screen = s
			break
		}
	}
	// Size is in logical pixels, the same space WindowSetSize works in.
	availW, availH := screen.Size.Width, screen.Size.Height
	if availW <= 0 || availH <= 0 {
		return
	}
	// Leave room for window chrome and the taskbar.
	width := max(minimumWidth, min(preferredWidth, availW-80))
	height := max(minimumHeight, min(preferredHeight, availH-120))
	if width == preferredWidth && height == preferredHeight {
		return
	}
	runtime.WindowSetSize(ctx, width, height)
	runtime.WindowCenter(ctx)
}
