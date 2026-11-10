package lyricsui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"lilt/internal/core/settings"
	"lilt/internal/ui/base/transport"
)

const (
	// TaskbarControlsWidth and TaskbarControlsHeight size the taskbar's song controls.
	TaskbarControlsWidth  = 360
	TaskbarControlsHeight = 124

	TaskbarMaterialWhite   = "white"
	TaskbarMaterialAcrylic = "acrylic"
)

// TaskbarControlsVibrancy is the material the taskbar's song controls use.
func TaskbarControlsVibrancy(s *settings.Settings) mygo.Vibrancy {
	if s.TaskbarLyricsMaterial == TaskbarMaterialWhite {
		return mygo.VibrancyNone
	}
	return mygo.VibrancyAcrylic
}

// SetLyricsMaterial switches the taskbar's song controls between the opaque
// white panel and the acrylic backdrop, and keeps the choice.
func (d *Taskbar) SetLyricsMaterial(material string) {
	d.Settings.TaskbarLyricsMaterial = material
	if d.SaveSettings != nil {
		d.SaveSettings()
	}
	if d.Controls != nil && !d.Controls.IsDestroyed() {
		d.Controls.SetVibrancy(TaskbarControlsVibrancy(d.Settings))
	}
	d.Redraw()
}

// Only the cover owns an interactive taskbar rectangle; the lyric host stays
// transparent to input, including clicks sent to Explorer's other threads.
func (d *Taskbar) SyncCoverWindow() {
	if d.MainWindow() == nil {
		return
	}
	tr := d.Player.Current()
	if !d.Visible || !d.Settings.TaskbarLyrics || d.Win == nil || d.Win.IsDestroyed() || tr == nil || tr.Album.PicURL == "" {
		d.DestroyCoverWindow()
		return
	}
	if d.CoverWin == nil || d.CoverWin.IsDestroyed() {
		p := TaskbarCoverPlacement(d.Placement)
		if p.Width <= 0 || p.Height <= 0 {
			return
		}
		dpi := max(p.DPI, 96)
		w := mygo.NewWindow(mygo.WindowOptions{
			Title: "任务栏歌曲封面", Panel: true, Hidden: true, Frameless: true,
			Transparent: true, DisableShadow: true, DisableResize: true,
			Width: max(1, p.Width*96/int(dpi)), Height: max(1, p.Height*96/int(dpi)),
			Content: ui.View(d.CoverView),
		})
		d.CoverWin, d.CoverReady = w, false
		w.SetIgnoreMouseEvents(false)
		styleDesktopLyricsWindow(w)
		if err := placeTaskbarLyricsCover(w, d.Placement); err != nil {
			d.reportError(err)
			d.DestroyCoverWindow()
			return
		}
		w.OnReadyToShow(func() {
			if d.CoverWin == w {
				d.CoverReady = true
				d.SyncCoverWindow()
			}
		})
		w.OnClosed(func() {
			if d.CoverWin == w {
				d.CoverWin, d.CoverReady = nil, false
				d.CloseControls()
			}
		})
		return
	}
	if err := placeTaskbarLyricsCover(d.CoverWin, d.Placement); err != nil {
		d.reportError(err)
		d.DestroyCoverWindow()
		return
	}
	if d.CoverReady && !d.CoverWin.IsVisible() {
		d.CoverWin.Invalidate()
		d.CoverWin.ShowInactive()
	}
}

func (d *Taskbar) DestroyCoverWindow() {
	d.CloseControls()
	w := d.CoverWin
	d.CoverWin, d.CoverReady = nil, false
	if w != nil && !w.IsDestroyed() {
		detachTaskbarLyricsWindow(w)
		w.Destroy()
	}
}

func (d *Taskbar) ToggleControls() {
	if d.ControlsOpen {
		d.CloseControls()
		return
	}
	tr := d.Player.Current()
	if !d.Visible || !d.Settings.TaskbarLyrics || tr == nil || tr.Album.PicURL == "" || d.Quitting() {
		return
	}
	d.ControlsOpen = true
	// The same actions work in Tester without creating native windows.
	if d.CoverWin == nil {
		d.Redraw()
		return
	}
	bounds := d.CoverWin.Bounds()
	w := mygo.NewFlyout(mygo.FlyoutOptions{
		Parent: d.CoverWin, Anchor: mygo.Rectangle{Width: bounds.Width, Height: bounds.Height},
		Placement: mygo.PlacementTopStart, Gap: 16, Width: TaskbarControlsWidth, Height: TaskbarControlsHeight,
		Focusable: true, Shadow: true, Hidden: true, Vibrancy: TaskbarControlsVibrancy(d.Settings), Content: ui.View(d.ControlsView),
	})
	// Windows can enable a backdrop later only when the window was created
	// with material support. Disable it before showing the opaque white panel.
	w.SetVibrancy(TaskbarControlsVibrancy(d.Settings))
	d.Controls = w
	w.OnClosed(func() {
		if d.Controls == w {
			d.Controls, d.ControlsOpen, d.Scrub = nil, false, transport.State{}
			d.Redraw()
		}
	})
	w.OnReadyToShow(func() {
		if d.Controls == w && d.ControlsOpen && d.Visible && !d.Quitting() {
			w.Show()
		}
	})
	d.Redraw()
}

func (d *Taskbar) CloseControls() {
	open := d.ControlsOpen
	w := d.Controls
	d.Controls, d.ControlsOpen, d.Scrub = nil, false, transport.State{}
	if w != nil && !w.IsDestroyed() {
		w.Destroy()
	}
	if open {
		d.Redraw()
	}
}
