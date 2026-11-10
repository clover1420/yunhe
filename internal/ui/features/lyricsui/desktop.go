// Package lyricsui owns the windows that show playback outside the main
// window: desktop lyrics and, on Windows, the taskbar's compact lyrics.
package lyricsui

import (
	"math"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"lilt/internal/core/settings"
	"lilt/internal/media/art"
	"lilt/internal/media/player"
)

// Deps are what a lyrics window needs from the app.
type Deps struct {
	Player       *player.Player
	Settings     *settings.Settings
	Art          *art.Cache
	Post         func(func())
	Redraw       func()
	Quitting     func() bool
	MainWindow   func() *mygo.Window
	RefreshTray  func()
	SetEnabled   func(bool)
	SaveSettings func()
}

// DesktopLyricsWidth is the default width of the desktop lyrics panel.
const DesktopLyricsWidth = 760

// Desktop owns a panel independently of the main window's visibility.
// All its state and callbacks belong to MyGo's main thread.
type Desktop struct {
	Deps
	Win           *mygo.Window
	OffDisplays   func()
	Width         float64
	DesiredHeight int
	Revision      uint64

	// WorkAreaOverride adjusts a display's work area, when the platform
	// reports one that covers panels; StopWorkArea stops watching it.
	WorkAreaOverride func(mygo.Rectangle) mygo.Rectangle
	StopWorkArea     func()
}

// SupportsDesktopLyrics reports whether the platform has a desktop lyrics
// panel.
func SupportsDesktopLyrics() bool {
	return runtime.GOOS == "windows" || runtime.GOOS == "darwin" || runtime.GOOS == "linux"
}

// NewDesktop returns a desktop lyrics panel.
func NewDesktop(d Deps) *Desktop {
	return &Desktop{Deps: d, Width: DesktopLyricsWidth}
}

// Sync opens, moves or closes the panel as the settings say.
func (d *Desktop) Sync() {
	if !d.Settings.DesktopLyrics {
		d.Destroy()
		return
	}
	// Tests can exercise actions without opening an operating-system window.
	if d.MainWindow() == nil || d.Win != nil {
		return
	}
	work := mygo.Screen.PrimaryDisplay().WorkArea
	bounds := ClampBounds(mygo.Rectangle{
		X:     work.X + (work.Width-DesktopLyricsWidth)/2,
		Y:     work.Y + work.Height - 220,
		Width: DesktopLyricsWidth, Height: 180,
	}, work)
	w := mygo.NewWindow(mygo.WindowOptions{
		Title: "桌面歌词", Panel: true, Hidden: true, Frameless: true,
		Transparent: true, DisableShadow: true, DisableResize: true,
		X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: bounds.Height,
		StateKey: "desktop-lyrics", Content: ui.View(d.View),
	})
	d.Win = w
	styleDesktopLyricsWindow(w)
	watchDesktopLyricsWorkArea(d)
	d.DesiredHeight = 0
	d.Width = float64(w.Bounds().Width)
	setDesktopLyricsIgnoreMouse(w, d.Settings.DesktopLyricsLocked)
	w.OnReadyToShow(func() {
		if d.Win == w && d.Settings.DesktopLyrics && !d.Quitting() {
			w.ShowInactive()
		}
	})
	w.OnClose(func(*mygo.CloseEvent) {
		if !d.Quitting() {
			d.Settings.DesktopLyrics = false
			d.Settings.Save()
			d.RefreshTray()
			d.Redraw()
		}
	})
	w.OnClosed(func() {
		if d.Win == w {
			d.ClearWorkAreaWatcher()
			d.Win = nil
			if d.OffDisplays != nil {
				d.OffDisplays()
				d.OffDisplays = nil
			}
		}
	})
	d.OffDisplays = mygo.Screen.OnDisplaysChanged(func() {
		if d.Win == nil {
			return
		}
		r := d.Win.Bounds()
		work := d.WorkArea(r)
		d.Win.SetBounds(ClampBounds(r, work))
	})
}

// Destroy closes the panel, keeping the listener's preference.
func (d *Desktop) Destroy() {
	d.ClearWorkAreaWatcher()
	if d.OffDisplays != nil {
		d.OffDisplays()
		d.OffDisplays = nil
	}
	if w := d.Win; w != nil {
		d.Win = nil
		w.Close()
	}
	d.DesiredHeight = 0
}

// ClearWorkAreaWatcher stops watching the platform's panels, if watched.
func (d *Desktop) ClearWorkAreaWatcher() {
	if d.StopWorkArea != nil {
		d.StopWorkArea()
		d.StopWorkArea = nil
	}
	d.WorkAreaOverride = nil
}

// WorkArea is the work area of the display r is on, as the platform's panels
// leave it.
func (d *Desktop) WorkArea(r mygo.Rectangle) mygo.Rectangle {
	work := mygo.Screen.DisplayMatching(r).WorkArea
	if d.WorkAreaOverride != nil {
		return d.WorkAreaOverride(work)
	}
	return work
}

// FitWorkArea keeps the panel inside the work area.
func (d *Desktop) FitWorkArea() {
	if d.Win == nil {
		return
	}
	r := d.Win.Bounds()
	if next := ClampBounds(r, d.WorkArea(r)); next != r {
		d.Win.SetBounds(next)
	}
}

func (d *Desktop) Close() { d.SetEnabled(false) }

// SetLocked locks the panel against the pointer.
func (d *Desktop) SetLocked(locked bool) {
	d.Settings.DesktopLyricsLocked = locked
	d.Settings.Save()
	if d.Win != nil {
		setDesktopLyricsIgnoreMouse(d.Win, locked)
	}
	d.RefreshTray()
	d.Redraw()
}

// FontSize is the size the lyrics show at.
func (d *Desktop) FontSize() float32 {
	size := d.Settings.DesktopLyricsFontSize
	if size == 0 || math.IsNaN(float64(size)) || math.IsInf(float64(size), 0) {
		return 32
	}
	return min(64, max(18, size))
}

// SetFontSize chooses the size the lyrics show at.
func (d *Desktop) SetFontSize(size float32) {
	if math.IsNaN(float64(size)) || math.IsInf(float64(size), 0) {
		return
	}
	d.Settings.DesktopLyricsFontSize = min(64, max(18, size))
	d.Settings.Save()
	d.Redraw()
}

// FitHeight follows measured content once it changes; resizing is deferred
// until the current build finishes, and keeps the lyrics' bottom anchored.
func (d *Desktop) FitHeight(height float64) {
	if d.Win == nil || math.IsNaN(height) || math.IsInf(height, 0) {
		return
	}
	w := d.Win
	work := d.WorkArea(w.Bounds())
	h := min(work.Height, max(100, int(math.Ceil(height))))
	if h == d.DesiredHeight {
		return
	}
	d.DesiredHeight = h
	w.Update(func() {
		if d.Win != w || d.DesiredHeight != h {
			return
		}
		r := w.Bounds()
		if r.Height == h {
			return
		}
		r.Y += r.Height - h
		r.Height = h
		w.SetBounds(ClampBounds(r, work))
	})
}

// ResizeWidth widens or narrows the panel.
func (d *Desktop) ResizeWidth(delta float64) {
	if math.IsNaN(delta) || math.IsInf(delta, 0) {
		return
	}
	if d.Width == 0 {
		d.Width = DesktopLyricsWidth
	}
	if d.Win == nil {
		d.Width = min(1800, max(320, d.Width+delta))
		return
	}
	r := d.Win.Bounds()
	work := d.WorkArea(r)
	r.Width = min(work.Width, max(320, r.Width+int(math.Round(delta))))
	r = ClampBounds(r, work)
	d.Width = float64(r.Width)
	d.Win.SetBounds(r)
}

// ClampBounds keeps a rectangle inside a display's work area.
func ClampBounds(r, work mygo.Rectangle) mygo.Rectangle {
	if work.Width <= 0 || work.Height <= 0 {
		return r
	}
	r.Width = min(work.Width, max(1, r.Width))
	r.Height = min(work.Height, max(1, r.Height))
	r.X = min(work.X+work.Width-r.Width, max(work.X, r.X))
	r.Y = min(work.Y+work.Height-r.Height, max(work.Y, r.Y))
	return r
}
