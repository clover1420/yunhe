package lyricsui

import (
	"github.com/egoist/mygo/ui"

	"lilt/internal/media/art"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
	"lilt/internal/ui/base/transport"
)

type taskbarControlsTrack struct {
	id  int64
	mid string
}

// ControlsView is a small player above the taskbar cover, independent of the
// main window.
func (d *Taskbar) ControlsView(c *ui.Context) {
	th := *c.Theme()
	white := d.Settings.TaskbarLyricsMaterial == TaskbarMaterialWhite
	material := !white && c.Vibrancy()
	// The material follows app appearance, which can differ from the taskbar.
	if white {
		th.Dark = false
	} else if !material {
		th.Dark = d.Dark
	}
	c.SetTheme(&th)
	dark := th.Dark
	background, text := theme.Neutral(dark, 2), theme.Neutral(dark, 12)
	muted := theme.Neutral(dark, 11)
	if material {
		background = ui.Transparent
	}
	c.Root().Background(background)
	c.OnShortcut(0, ui.KeyEscape, d.CloseControls)
	tr := d.Player.Current()
	if !d.ControlsOpen || tr == nil {
		return
	}
	identity := taskbarControlsTrack{id: tr.ID, mid: tr.MID}
	previous := ui.Local(c.Root(), "taskbar-controls-track", func() taskbarControlsTrack { return identity })
	if *previous != identity {
		d.Scrub = transport.State{}
		*previous = identity
	}
	pl := d.Player
	transport.TickTimes(c, pl)
	ui.Column(c.Key(identity)).Fill().Padding(12).Gap(8).TextColor(text).Children(func() {
		ui.Row(c).Height(64).Gap(12).AlignItems(ui.Center).Children(func() {
			cover := ui.Box(c.Key("artwork")).Size(64, 64).Shrink(0).Radius(8).Background(theme.Neutral(dark, 4)).Center()
			bmp := bitmap(d.Art.Get(qqmusic.Sized(tr.Album.PicURL, 120), art.Cover))
			cover.Children(func() {
				if bmp != nil {
					ui.Image(c, bmp).Size(64, 64).Fit(ui.Cover).Radius(8)
				} else {
					ui.Icon(c, appui.IconNote).Size(24, 24).TextColor(muted)
				}
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, tr.Name).FontSize(14).FontWeight(600).LineHeight(1.15).SingleLine().Tooltip(tr.Name)
				artist := tr.ArtistNames()
				ui.Text(c, artist).FontSize(13).LineHeight(1.15).TextColor(muted).SingleLine().Tooltip(artist)
			})
			ui.Row(c).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
				appui.IconButton(c.Key("previous"), appui.IconPrev, 18, text, "上一首", appui.PlayerSurface).Size(28, 28).Shrink(0).
					OnClick(func() { pl.Prev(); d.Redraw() })
				appui.PlayButton(c.Key("play"), pl.Playing(), 20, text, appui.PlayerSurface).Size(36, 36).Shrink(0).
					OnClick(func() { pl.TogglePlay(); d.Redraw() })
				appui.IconButton(c.Key("next"), appui.IconNext, 18, text, "下一首", appui.PlayerSurface).Size(28, 28).Shrink(0).
					OnClick(func() { pl.Next(); d.Redraw() })
			})
		})
		ui.Column(c).Gap(0).Children(func() {
			transport.Scrubber(c, pl, d.Settings.ProgressEffectMode(), &d.Scrub, theme.Neutral(dark, 8), text.Alpha(0.75), 14, theme.Accent(dark, 9)).FillWidth()
			position := pl.Position()
			if d.Scrub.Active || d.Scrub.Held {
				position = transport.Secs(d.Scrub.V)
			}
			ui.Row(c).Height(14).Justify(ui.SpaceBetween).TextColor(muted).Children(func() {
				ui.Text(c, appui.Clock(position)).FontSize(11.5).FontFeatures("tnum").TextColor(muted)
				ui.Text(c, appui.Clock(pl.Duration())).FontSize(11.5).FontFeatures("tnum").TextColor(muted)
			})
		})
	})
}
