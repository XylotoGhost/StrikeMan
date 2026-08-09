package app

// Window sizing. The preferred size is what the layout is designed for; the
// minimum is where it still works after reflowing to one column.

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	PreferredWidth  = 1200
	PreferredHeight = 1000
	MinimumWidth    = 820
	MinimumHeight   = 560
)

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
	width := max(MinimumWidth, min(PreferredWidth, availW-80))
	height := max(MinimumHeight, min(PreferredHeight, availH-120))
	if width == PreferredWidth && height == PreferredHeight {
		return
	}
	runtime.WindowSetSize(ctx, width, height)
	runtime.WindowCenter(ctx)
}
