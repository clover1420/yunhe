package lyricsui

import (
	"bytes"
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/lyrics"
)

func notchGoldPixels(img *image.RGBA, area image.Rectangle) int {
	count := 0
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			p := img.RGBAAt(x, y)
			if int(p.R) > int(p.G)+15 && int(p.G) > int(p.B)+30 {
				count++
			}
		}
	}
	return count
}

// Render against deliberately uneven word times, including a real silent gap.
// Playback changes only the fill, never the retained sentence geometry.
func TestNotchWordFillAndSeek(t *testing.T) {
	line := lyrics.Build("", "", "[1000,7000]慢(1000,3000)唱(6000,2000)")[0]
	position := 1500 * time.Millisecond
	var layout notchLyricLayout
	tt := ui.NewTester(func(c *ui.Context) {
		layout = makeNotchLyricLayout(c, line, position, "")
		// Render without an opaque box background so Windows ClearType subpixel
		// fringes on white text are not mistaken for gold highlight pixels.
		ui.Box(c).Size(100, 32).Draw(func(p *ui.Painter, r ui.Rect) {
			layout.draw(p, r, ui.Start)
		})
	}, 100, 32)
	tt.Frame()
	initial, firstPixels := layout, append([]byte(nil), tt.Image().Pix...)
	initialGold := notchGoldPixels(tt.Image(), tt.Image().Bounds())
	position = 3500 * time.Millisecond
	tt.Frame()
	if notchGoldPixels(tt.Image(), tt.Image().Bounds()) <= initialGold {
		t.Fatal("the current character did not fill as its real duration elapsed")
	}
	if layout.words != initial.words || layout.fills[0].W <= initial.fills[0].W {
		t.Fatal("word progress reshaped the line or mutated the earlier frame")
	}
	position = 5 * time.Second
	tt.Frame()
	if notchGoldPixels(tt.Image(), tt.Image().Bounds()) != 0 || len(layout.fills) != 0 || layout.colors[0] != notchLyricSung || layout.colors[1] != notchLyricRest {
		t.Fatal("a silent gap must leave completed words white and future words gray")
	}
	position = 7 * time.Second
	tt.Frame()
	if notchGoldPixels(tt.Image(), tt.Image().Bounds()) == 0 || layout.colors[0] != notchLyricSung {
		t.Fatal("the next word did not start its own fill")
	}
	position = 1500 * time.Millisecond
	tt.Frame()
	if !bytes.Equal(tt.Image().Pix, firstPixels) {
		t.Fatal("seeking backwards did not restore the exact earlier progress")
	}
	position = 8 * time.Second
	tt.Frame()
	if len(layout.fills) != 0 || layout.colors[0] != notchLyricSung || layout.colors[1] != notchLyricSung {
		t.Fatal("finished line retained an active word")
	}
}
