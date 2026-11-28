package nowplaying

import (
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/core/settings"
	"lilt/internal/media/lyrics"
)

// A short word has no swell, and a word held does only while it is sung,
// highest around its middle and gone by its end: a line left glowing is what
// went too far before.
func TestWordHeld(t *testing.T) {
	short := lyrics.Word{Start: 0, End: 900 * time.Millisecond}
	if v := wordHeld(short, true, 450*time.Millisecond); v != 0 {
		t.Errorf("a short word swells by %v", v)
	}
	long := lyrics.Word{Start: 0, End: 3 * time.Second}
	peak := wordHeld(long, false, 1500*time.Millisecond)
	if peak <= 0 || peak > 0.8 {
		t.Errorf("a word held 3 s swells by %v", peak)
	}
	if v := wordHeld(long, false, 200*time.Millisecond); v >= peak {
		t.Errorf("as it begins it swells by %v, past its middle's %v", v, peak)
	}
	if v := wordHeld(long, false, 3*time.Second); v != 0 {
		t.Errorf("sung, it still swells by %v", v)
	}
	if v := wordHeld(long, true, 1800*time.Millisecond); v <= peak {
		t.Errorf("the last word swells by %v, no more than another's %v", v, peak)
	}
}

// Nothing glows ahead of the singer: words that have not started leave the
// glow layer untouched, while the word already sung lights up.
func TestGlowOnlyWhereSung(t *testing.T) {
	line := lyrics.Line{At: time.Second, End: 9 * time.Second, Text: "Why do the stars", Timed: true, Words: []lyrics.Word{
		{Text: "Why ", Start: time.Second, End: 2 * time.Second},
		{Text: "do the stars", Start: 6 * time.Second, End: 8 * time.Second},
	}}
	frame := func(line lyrics.Line) *image.RGBA {
		white := ui.RGBA(255, 255, 255, 1)
		l := Lyrics{Settings: &settings.Settings{NpLight: true, GlowLevel: 0.5}, Glow: &white}
		tt := ui.NewTester(func(c *ui.Context) {
			c.Root().Background(ui.RGB(0, 0, 0))
			ui.Box(c).Absolute().Left(20).Top(20).Width(500).Height(100).Children(func() {
				l.lineGlow(c, line, 3*time.Second, LinesStyle{Size: 40, Weight: 800, LineHeight: 1.25})
			})
		}, 600, 160)
		return tt.Image()
	}
	actual := frame(line)
	// Removing words that have not started leaves no bright glow where
	// they were: they hold their place, faint, so the line keeps its width.
	line.Words = line.Words[:1]
	expected := frame(line)
	bright := func(img *image.RGBA) int {
		n := 0
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				if img.RGBAAt(x, y).R > 30 {
					n++
				}
			}
		}
		return n
	}
	if got, want := bright(actual), bright(expected); got > want+want/50 {
		t.Fatalf("words that have not started glow: %d bright pixels, without them %d", got, want)
	}
	lit := false
	for y := 0; y < actual.Bounds().Dy(); y++ {
		for x := 0; x < actual.Bounds().Dx(); x++ {
			pixel := actual.RGBAAt(x, y)
			if pixel.R > 0 || pixel.G > 0 || pixel.B > 0 {
				lit = true
			}
		}
	}
	if !lit {
		t.Fatal("the word already sung did not glow")
	}
}

// With the glow off, or turned to nothing, nothing is lit again.
func TestGlows(t *testing.T) {
	off := Lyrics{Settings: &settings.Settings{NpLight: false, GlowLevel: 0.5}}
	if off.glows() {
		t.Error("the glow is drawn with the setting off")
	}
	none := Lyrics{Settings: &settings.Settings{NpLight: true, GlowLevel: 0}}
	if none.glows() {
		t.Error("the glow is drawn at no strength")
	}
	on := Lyrics{Settings: &settings.Settings{NpLight: true, GlowLevel: 0.5}}
	if !on.glows() {
		t.Error("the glow is not drawn with the setting on")
	}
}
