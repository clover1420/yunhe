package theme

import "github.com/egoist/mygo/ui"

// CoverAccent turns a cover's most vivid color into an accent that keeps
// white text legible. The hue stays, while saturation and value are
// constrained, as Cloudy does. A gray cover does not follow.
func CoverAccent(tint ui.Color) (ui.Color, bool) {
	h, s, v := HSV(float64(tint.R)/255, float64(tint.G)/255, float64(tint.B)/255)
	// A pale cover, a washed-out sky for one, is barely saturated in HSV yet
	// clearly colored: its chroma, v·s, tells it from a gray one.
	if s*v < 0.08 {
		return ui.Color{}, false
	}
	return HSVColor(h, min(max(s, 0.5), 0.8), 0.8), true
}

// AccentVariants are the interactive shades of an accent.
func AccentVariants(base ui.Color) (hover, pressed ui.Color) {
	return base.Mix(Paper, 0.12), base.Mix(Ink, 0.12)
}
