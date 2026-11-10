//go:build !windows

package app

import "github.com/egoist/mygo"

func desktopLyricsNativeSnapshot(a *app) map[string]any {
	out := map[string]any{"enabled": a.settings.DesktopLyrics, "locked": a.settings.DesktopLyricsLocked, "mainVisible": a.win.IsVisible()}
	out["displays"] = mygo.Screen.Displays()
	if d := a.desktopLyrics; d != nil && d.Win != nil {
		out["visible"] = d.Win.IsVisible()
		out["bounds"] = d.Win.Bounds()
		out["workArea"] = d.WorkArea(d.Win.Bounds())
		out["desiredHeight"] = d.DesiredHeight
	}
	return out
}
