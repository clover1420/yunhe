package app

import (
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// Panes are what floats over the app: menus, popovers and dialogs. All take
// their look from pane, so that they are of one material, glass or, with
// settings.GlassPanels off, plain.

// A paneInk holds the colors of what is on a pane.
type paneInk struct {
	text, muted, faint ui.Color
	hover, press, line ui.Color
}

// inkOn returns the colors for a pane, white where it is dark. onDark is for
// a pane over Now Playing, which is dark whatever the theme.
func (a *app) inkOn(onDark bool) paneInk {
	if onDark || a.pal.Dark {
		return paneInk{white, white.Alpha(0.6), white.Alpha(0.42), white.Alpha(0.1), white.Alpha(0.16), white.Alpha(0.1)}
	}
	text := ui.Hex("#1d1d1f")
	return paneInk{text, text.Alpha(0.62), text.Alpha(0.42), text.Alpha(0.07), text.Alpha(0.12), text.Alpha(0.1)}
}

// pane gives e the look of a pane, rounded by radius. Glass lightens what
// shows through in a light theme: a dark pane there takes more of its tint.
func (a *app) pane(e ui.Element, radius float32, onDark bool) ui.Element {
	e.Radius(radius).Shadow(0, 18, 44, -12, ui.RGBA(0, 0, 0, 0.45))
	dark := onDark || a.pal.Dark
	switch {
	case !a.settings.GlassPanels && dark:
		return e.Background(ui.Hex("#232327")).Border(0.5, white.Alpha(0.12))
	case !a.settings.GlassPanels:
		return e.Background(ui.Hex("#f6f6f8")).Border(0.5, ui.RGBA(0, 0, 0, 0.12))
	case a.pal.Dark:
		return e.Material(glass.Glass{Tint: ui.RGBA(14, 14, 18, 0.62)}).Border(0.5, white.Alpha(0.14))
	case onDark:
		return e.Material(glass.Glass{Tint: ui.RGBA(14, 14, 18, 0.86)}).Border(0.5, white.Alpha(0.14))
	}
	return e.Material(glass.Glass{Tint: ui.RGBA(250, 250, 252, 0.58)}).Border(0.5, ui.RGBA(0, 0, 0, 0.1))
}

// paneSolid is pane as plain color, without the glass, for what floats over
// the sidebar: there the system's material fills the window, the scene leaves
// that stretch see-through, and the glass, which samples the scene, tints it
// gray instead of showing what is behind the window.
func (a *app) paneSolid(e ui.Element, radius float32, onDark bool) ui.Element {
	e.Radius(radius).Shadow(0, 18, 44, -12, ui.RGBA(0, 0, 0, 0.35))
	if onDark || a.pal.Dark {
		return e.Background(ui.Hex("#232327")).Border(0.5, white.Alpha(0.12))
	}
	return e.Background(ui.Hex("#f6f6f8")).Border(0.5, ui.RGBA(0, 0, 0, 0.12))
}
