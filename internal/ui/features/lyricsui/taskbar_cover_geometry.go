package lyricsui

import "math"

// The interactive native child covers only the album art. The lyric window
// remains entirely click-through, including the text and surrounding space.
func TaskbarCoverPlacement(p Placement) Placement {
	if p.Parent == 0 || p.Width <= 0 || p.Height <= 0 {
		return Placement{}
	}
	scale := float64(max(p.DPI, 96)) / 96
	size := int(math.Round(min(34*scale, float64(p.Height)-4*scale)))
	left := int(math.Round(TaskbarCoverInset * scale))
	if size <= 0 || left+size > p.Width {
		return Placement{}
	}
	if p.RightAligned {
		p.X += p.Width - left - size
	} else {
		p.X += left
	}
	p.Y += int(math.Round(float64(p.Height-size) / 2))
	p.Width, p.Height = size, size
	return p
}
