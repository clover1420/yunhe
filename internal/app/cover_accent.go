package app

import (
	"github.com/egoist/mygo/ui"

	"lilt/internal/media/art"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/theme"
)

// followCover keeps the cover of the song playing, so its color can accent
// every window. While a new cover loads, the last one's color stays.
func (a *app) followCover() {
	tr := a.pl.Current()
	if tr == nil {
		return
	}
	if tr.Album.PicURL == "" {
		a.coverArt = nil
		return
	}
	if art := a.art.Get(qqmusic.Sized(tr.Album.PicURL, 600), art.Tint); art != nil && !art.Failed {
		a.coverArt = art
	}
}

// coverAccent turns the cover's most vivid color into an accent that keeps
// white text legible. The hue stays, while saturation and value are
// constrained, as Cloudy does. A gray cover does not follow.
func (a *app) coverAccent() (ui.Color, bool) {
	art := a.coverArt
	if art == nil || art.Failed {
		return ui.Color{}, false
	}
	return theme.CoverAccent(art.Tint)
}

// applyTheme sets a window's theme for this frame. Every window calls it, so
// none keeps the fixed red while another follows the cover.
func (a *app) applyTheme(c *ui.Context) palette {
	t := *c.Theme()
	if a.settings.AccentCover {
		a.followCover()
	}
	p := paletteFor(&t)
	base := accentPreset(a.settings.Accent, t.Dark)
	if a.settings.AccentCover {
		if accent, ok := a.coverAccent(); ok {
			base = accent
		}
	}
	p.Accent = base
	a.pal = p
	hover, pressed := theme.AccentVariants(base)
	t.Accent, t.AccentHover, t.AccentPressed = base, hover, pressed
	t.AccentText = p.OnAccent
	t.Selection = base.Alpha(0.3)
	t.Focus = base.Alpha(0.55)
	t.Font = a.fontFamily()
	c.SetTheme(&t)
	return p
}

// progressFill is the played part of the player bar's progress. It takes the
// cover's accent with the rest of the controls, and keeps its neutral look
// while the setting is off.
func (a *app) progressFill(p palette) ui.Color {
	if a.settings.AccentCover {
		if accent, ok := a.coverAccent(); ok {
			return accent
		}
	}
	if a.settings.Accent != "" {
		return accentPreset(a.settings.Accent, p.Dark)
	}
	return p.Text.Alpha(0.75)
}
