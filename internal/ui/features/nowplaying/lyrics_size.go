package nowplaying

import (
	"github.com/egoist/mygo/ui"

	"lilt/internal/core/settings"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/theme"
)

const (
	LyricsFontSizeDefault = 30
	LyricsFontSizeMin     = 18
	LyricsFontSizeMax     = 60
)

// LyricsFontSize is the size the scrolling lyrics are set in, kept between
// what the popover offers; zero keeps the original size.
func LyricsFontSize(s *settings.Settings) float64 {
	if s == nil || s.LyricsFontSize <= 0 {
		return LyricsFontSizeDefault
	}
	return min(LyricsFontSizeMax, max(LyricsFontSizeMin, s.LyricsFontSize))
}

// LyricsSizeButton keeps reading preferences beside the lyrics, with the song
// itself as the preview and no separate confirmation step.
func LyricsSizeButton(c *ui.Context, d Deps) {
	b := ui.ButtonBase(c.Key("lyrics-size-button")).Bind(d.LyricsSizeTrigger).
		Size(36, TabsHeight).Shrink(0).Center().Radius(999).Label("歌词字号").Tooltip("歌词字号")
	appui.MediaFeedback(b, theme.Veil(8))
	if *d.LyricsSizeOpen {
		b.Background(theme.Veil(3)).TextColor(theme.Paper)
	}
	b.Children(func() { ui.Text(c, "Aa").FontSize(15).FontWeight(600) })
	if b.Clicked() {
		*d.LyricsSizeOpen = !*d.LyricsSizeOpen
	}
	panel := ui.PopoverBase(c.Key("lyrics-size-popover"), b, d.LyricsSizeOpen, func(panel ui.Element) {
		panel.AttachTo(b, ui.AnchorBottomEnd, ui.AnchorTopEnd).Margin(8, 0, 0, 0).
			Width(240).Padding(16).Gap(12).Radius(12).Background(theme.Neutral(true, 2)).
			Border(0.5, theme.Neutral(true, 6)).Shadow(0, 8, 24, -4, theme.Shade(6)).TextColor(theme.Neutral(true, 12))
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "歌词字号").FontSize(13).FontWeight(600).Grow(1)
			ui.Textf(c, "%.0f", LyricsFontSize(d.Settings)).FontSize(13).FontFeatures("tnum").TextColor(theme.Neutral(true, 11))
		})
		lyricsSizeSlider(c, d)
		reset := ui.ButtonBase(c.Key("reset")).Padding(5, 8).Margin(0, -8, -5, 0).
			AlignSelf(ui.Start).Radius(6).Label("恢复默认字号")
		appui.MediaFeedback(reset, theme.Neutral(true, 11))
		reset.Children(func() { ui.Text(c, "恢复默认").FontSize(12) })
		if reset.Clicked() {
			d.Settings.LyricsFontSize = LyricsFontSizeDefault
			d.LyricsSizeChanged()
		}
	})
	if panel.Dismissed() && !panel.PressedOutside() {
		d.LyricsSizeTrigger.Focus(c)
	}
}

func lyricsSizeSlider(c *ui.Context, d Deps) {
	d.Settings.LyricsFontSize = LyricsFontSize(d.Settings)
	const knob, track = float32(16), float32(4)
	s := ui.SliderBase(c.Key("size"), &d.Settings.LyricsFontSize, LyricsFontSizeMin, LyricsFontSizeMax).
		Height(28).PaddingX(knob / 2).Step(1).AutoFocus().Label("调整歌词字号").Cursor(ui.CursorPointer)
	if s.Changed() {
		d.LyricsSizeChanged()
	}
	value := float32((d.Settings.LyricsFontSize - LyricsFontSizeMin) / (LyricsFontSizeMax - LyricsFontSizeMin))
	fill := theme.Accent(true, 9)
	if s.Hovered() || s.Pressed() {
		fill = theme.Accent(true, 10)
	}
	s.Draw(func(p *ui.Painter, r ui.Rect) {
		left, width := r.X+knob/2, r.W-knob
		x, y := left+width*value, r.Y+r.H/2
		p.Fill(ui.Rect{X: left, Y: y - track/2, W: width, H: track}, theme.Neutral(true, 6), track/2)
		p.Fill(ui.Rect{X: left, Y: y - track/2, W: x - left, H: track}, fill, track/2)
		p.Fill(ui.Rect{X: x - knob/2, Y: y - knob/2, W: knob, H: knob}, theme.Paper, knob/2)
	})
}
