package nowplaying

import (
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/core/settings"
	"lilt/internal/media/lyrics"
	"lilt/internal/media/player"
)

// The glow reaches further along the line as later words are sung, whether
// the lyrics start at the left or are centered.
func TestGlowSyncsWithWords(t *testing.T) {
	for _, center := range []bool{false, true} {
		name := "left"
		if center {
			name = "center"
		}
		t.Run(name, func(t *testing.T) {
			line := lyrics.Line{At: 0, Timed: true, Words: []lyrics.Word{
				{Text: "甲甲", Start: 0, End: time.Second},
				{Text: "乙乙", Start: time.Second, End: 2 * time.Second},
				{Text: "丙丙", Start: 2 * time.Second, End: 3 * time.Second},
			}}
			red := ui.RGBA(255, 0, 0, 1)
			pl := player.New(func(func()) {})
			at, browse := 0, false
			l := Lyrics{Player: pl, Settings: &settings.Settings{NpLight: true, GlowLevel: 0.5}, Glow: &red,
				Scroll: &ui.ScrollState{}, Glide: &Glide{}, At: &at, Browse: &browse}
			pos := 1500 * time.Millisecond
			tt := ui.NewTester(func(c *ui.Context) {
				c.Root().Background(ui.RGB(0, 0, 0)).Children(func() {
					ui.Box(c).Grow(1).Children(func() {
						scroll := ui.Scroll(c.Key("lyrics")).Grow(1)
						TimedLines(c, l, scroll, []lyrics.Line{line}, pos, 0, 0, LinesStyle{Size: 40, Weight: 800, LineHeight: 1.25, Center: center}, nil)
					})
				})
			}, 600, 300)
			tt.Frame()
			early := glowCentroid(tt.Image())
			pos = 2500 * time.Millisecond
			tt.Frame()
			late := glowCentroid(tt.Image())
			if early < 0 || late < 0 {
				t.Fatal("the glow was not drawn")
			}
			if late <= early {
				t.Fatalf("the glow does not move on with the words: centroid %.1f, then %.1f", early, late)
			}
		})
	}
}

// glowCentroid is the average x of the glow's red pixels, -1 if none.
func glowCentroid(img *image.RGBA) float64 {
	sum, n := 0.0, 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			p := img.RGBAAt(x, y)
			if p.R > 60 && p.G < 40 && p.B < 40 {
				sum += float64(x)
				n++
			}
		}
	}
	if n == 0 {
		return -1
	}
	return sum / float64(n)
}
