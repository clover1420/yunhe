package app

import "lilt/internal/ui/base/theme"

// The app reads the theme through these names while its views still live in
// this package; feature packages import lilt/internal/ui/base/theme directly.
type palette = theme.Palette

var (
	paletteFor = theme.For

	neutral      = theme.Neutral
	accent       = theme.Accent
	accentBase   = theme.AccentBase
	accentPreset = theme.AccentPreset
	warm         = theme.Warm
	veil         = theme.Veil
	shade        = theme.Shade

	ink   = theme.Ink
	paper = theme.Paper

	burstOrange = theme.BurstOrange
	burstPurple = theme.BurstPurple
	burstCyan   = theme.BurstCyan
	burstPink   = theme.BurstPink

	hsv      = theme.HSV
	hsvColor = theme.HSVColor
)
