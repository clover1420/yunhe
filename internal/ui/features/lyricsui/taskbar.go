package lyricsui

import (
	"log"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"lilt/internal/ui/base/transport"
)

// Taskbar owns the Explorer child used by the taskbar lyrics mode.
// Only the taskbar monitor works off-thread. Window and player state stay on main.
type Taskbar struct {
	Deps
	translationPolicy lyricTranslationPolicy
	Win               *mygo.Window
	CoverWin          *mygo.Window
	Controls          *mygo.Window
	ControlsOpen      bool
	CoverReady        bool
	Scrub             transport.State
	Stop              func()
	Placement         Placement
	Parent            uintptr
	Revision          uint64
	Ready             bool
	Visible           bool
	Dark              bool
	LastError         string
}

// SupportsTaskbarLyrics reports whether the platform has taskbar lyrics.
func SupportsTaskbarLyrics() bool { return runtime.GOOS == "windows" }

// NewTaskbar returns taskbar lyrics for the app.
func NewTaskbar(d Deps) *Taskbar { return &Taskbar{Deps: d} }

// Sync starts the native monitor that places the lyrics in the taskbar.
func (d *Taskbar) Sync() {
	if !SupportsTaskbarLyrics() || !d.Settings.TaskbarLyrics || d.Quitting() {
		d.Destroy()
		return
	}
	// UI tests exercise preferences without creating native windows or COM work.
	if d.MainWindow() == nil || d.Stop != nil {
		return
	}
	d.Stop = startTaskbarLyricsMonitor(func(p Placement) {
		if !d.Settings.TaskbarLyrics || d.Quitting() {
			return
		}
		d.UpdatePlacement(p)
	})
}

func (d *Taskbar) UpdatePlacement(p Placement) {
	changed := d.Placement != p
	d.Placement, d.Dark = p, p.Dark
	if p.Parent == 0 || p.Width <= 0 || p.Height <= 0 {
		d.Hide()
		return
	}
	if d.Win != nil && (d.Win.IsDestroyed() || d.Parent != p.Parent) {
		d.DestroyWindow()
	}
	if d.Win == nil {
		d.newWindow(p)
		return
	}
	if changed || !d.Visible {
		d.present()
	} else if d.Ready {
		// Explorer can reorder its composition bridge without changing geometry.
		// Repair child z-order without requesting another lyric paint.
		if err := placeTaskbarLyricsWindow(d.Win, p); err != nil {
			d.reportError(err)
			d.Hide()
		}
	}
	d.SyncCoverWindow()
}

func (d *Taskbar) newWindow(p Placement) {
	dpi := max(p.DPI, 96)
	w := mygo.NewWindow(mygo.WindowOptions{
		Title: "任务栏歌词", Panel: true, Hidden: true, Frameless: true,
		Transparent: true, DisableShadow: true, DisableResize: true,
		Width: max(1, p.Width*96/int(dpi)), Height: max(1, p.Height*96/int(dpi)),
		Content: ui.View(d.View),
	})
	d.Win, d.Parent, d.Ready = w, p.Parent, false
	w.SetIgnoreMouseEvents(true)
	styleDesktopLyricsWindow(w)
	// Attach while hidden so even the initial transparent frame stays in Explorer.
	if err := placeTaskbarLyricsWindow(w, p); err != nil {
		d.reportError(err)
		d.DestroyWindow()
		return
	}
	w.OnReadyToShow(func() {
		if d.Win == w {
			d.Ready = true
			d.present()
		}
	})
	w.OnClosed(func() {
		if d.Win == w {
			d.Win, d.Ready, d.Visible = nil, false, false
			d.DestroyCoverWindow()
		}
	})
}

func (d *Taskbar) present() {
	p := d.Placement
	if d.Win == nil || !d.Ready || !d.Settings.TaskbarLyrics || d.Quitting() || p.Parent == 0 || p.Width <= 0 || p.Height <= 0 {
		return
	}
	if err := placeTaskbarLyricsWindow(d.Win, p); err != nil {
		d.reportError(err)
		d.Hide()
		return
	}
	d.LastError = ""
	d.Visible = true
	d.Win.Invalidate()
	if !d.Win.IsVisible() {
		d.Win.ShowInactive()
	}
	d.SyncCoverWindow()
}

func (d *Taskbar) reportError(err error) {
	if msg := err.Error(); msg != d.LastError {
		log.Printf("taskbar lyrics: %s", msg)
		d.LastError = msg
	}
}

func (d *Taskbar) Hide() {
	d.Visible = false
	d.CloseControls()
	if d.CoverWin != nil && !d.CoverWin.IsDestroyed() {
		d.CoverWin.Hide()
	}
	if d.Win != nil && !d.Win.IsDestroyed() {
		d.Win.Hide()
	}
}

// ReportError records a native failure, shown by the probe.
func (d *Taskbar) ReportError(err error) { d.reportError(err) }

func (d *Taskbar) DestroyWindow() {
	d.DestroyCoverWindow()
	w := d.Win
	d.Win, d.Ready, d.Visible, d.Parent = nil, false, false, 0
	if w != nil && !w.IsDestroyed() {
		detachTaskbarLyricsWindow(w)
		w.Destroy()
	}
}

func (d *Taskbar) Destroy() {
	if d.Stop != nil {
		d.Stop()
		d.Stop = nil
	}
	d.DestroyWindow()
}
