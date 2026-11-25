package art

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"runtime"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/egoist/mygo/ui"
)

type genericImage struct{ image.Image }

func TestShrinkPreservesSampleColors(t *testing.T) {
	rgba := image.NewRGBA(image.Rect(3, 5, 51, 53))
	nrgba := image.NewNRGBA(rgba.Bounds())
	ycbcr := image.NewYCbCr(rgba.Bounds(), image.YCbCrSubsampleRatio420)
	for y := 5; y < 53; y++ {
		for x := 3; x < 51; x++ {
			c := color.NRGBA{R: uint8(x * 3), G: uint8(y * 4), B: 80, A: uint8(x + y)}
			rgba.Set(x, y, c)
			nrgba.SetNRGBA(x, y, c)
			ycbcr.Y[ycbcr.YOffset(x, y)] = uint8(x * 3)
			i := ycbcr.COffset(x, y)
			ycbcr.Cb[i], ycbcr.Cr[i] = uint8(y*3), 150
		}
	}
	for _, img := range []image.Image{rgba, nrgba, ycbcr} {
		fast, generic := shrink(img, 24), shrink(genericImage{img}, 24)
		if !slices.Equal(fast.Pix, generic.Pix) {
			t.Fatalf("typed sampling changed %T colors", img)
		}
	}
	if allocs := testing.AllocsPerRun(10, func() { shrink(ycbcr, 24) }); allocs > 4 {
		t.Fatalf("YCbCr sampling allocates %.0f objects", allocs)
	}
}

// Lab is how k-means compares colors, so its conversions must round-trip.
func TestLabRoundTrips(t *testing.T) {
	for _, c := range [][3]float32{{0.098, 0.353, 0.863}, {0.8, 0.18, 0.17}, {0.5, 0.5, 0.5}, {0.62, 0.3, 0.48}} {
		l := rgbToLab(c[0], c[1], c[2])
		r, g, b := labToRGB(l)
		if diff := r - c[0]; diff < -0.01 || diff > 0.01 {
			t.Fatalf("%v round-tripped through %+v to %v %v %v", c, l, r, g, b)
		}
		if diff := g - c[1]; diff < -0.01 || diff > 0.01 {
			t.Fatalf("%v round-tripped through %+v to %v %v %v", c, l, r, g, b)
		}
		if diff := b - c[2]; diff < -0.01 || diff > 0.01 {
			t.Fatalf("%v round-tripped through %+v to %v %v %v", c, l, r, g, b)
		}
	}
}

// White is an extreme Palette's filter drops, so the blue minority decides the
// cover's color instead of being washed out by it.
func TestDominantColorFiltersExtremes(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			c := color.RGBA{255, 255, 255, 255}
			if x < 20 {
				c = color.RGBA{25, 90, 220, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	diff := func(a, b uint8) int {
		if a > b {
			return int(a - b)
		}
		return int(b - a)
	}
	got := DominantColor(img)
	if diff(got.R, 25) > 24 || diff(got.G, 90) > 24 || diff(got.B, 220) > 24 {
		t.Fatalf("dominant color = %+v, want the blue", got)
	}
}

// A cover of only grays has no color to take, so its average shows instead.
func TestDominantColorOfGrays(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			v := uint8((x * 4) % 256)
			img.SetRGBA(x, y, color.RGBA{v, v, v, 255})
		}
	}
	got := DominantColor(img)
	if got.R != got.G || got.G != got.B {
		t.Fatalf("gray cover gave %+v", got)
	}
}

// Clusters within ΔE 12 of a heavier one merge into it, their weights added.
func TestMergeSwatchesJoinsCloseColors(t *testing.T) {
	merged := mergeSwatches([]dominantSwatch{
		{rgbToLab(0.12, 0.22, 0.85), 10},
		{rgbToLab(0.80, 0.18, 0.17), 40},
		{rgbToLab(0.78, 0.16, 0.16), 60},
	})
	if len(merged) != 2 {
		t.Fatalf("merged into %d swatches, want 2", len(merged))
	}
	if merged[0].population != 100 {
		t.Fatalf("heaviest swatch population = %v, want 100", merged[0].population)
	}
}

// Palette's HSL scoring favors colors near the middle of saturation and
// lightness over nearly black ones, at the same population.
func TestPaletteScorePrefersMidColors(t *testing.T) {
	mid := dominantSwatch{rgbToLab(0.62, 0.3, 0.48), 10}
	dark := dominantSwatch{rgbToLab(0.1, 0.03, 0.09), 10}
	if paletteScore(mid, 10) <= paletteScore(dark, 10) {
		t.Fatal("a mid color should score higher than a nearly black one")
	}
}

func coverData(t testing.TB) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			img.SetRGBA(x, y, color.RGBA{R: 180, G: 40, B: 80, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func finishArtWork(t *testing.T, posted <-chan func()) {
	t.Helper()
	synctest.Wait()
	select {
	case fn := <-posted:
		fn()
	default:
		t.Fatal("artwork worker did not publish a result")
	}
}

// Display-only covers used to decode and measure a second full image they never used.
func TestArtCoverNeedsNoDerivedImage(t *testing.T) {
	a, err := DecodeArt(coverData(t), Cover)
	if err != nil {
		t.Fatal(err)
	}
	if a.BMP == nil || len(a.Data) == 0 {
		t.Fatal("cover lost its bitmap or compressed restore source")
	}
	if a.Tint != (ui.Color{}) || a.Backdrop != nil || a.Ready != Cover {
		t.Fatal("display-only cover generated unrequested derived resources")
	}
	if w, h := a.BMP.Size(); w != 16 || h != 16 {
		t.Fatalf("cover is %d×%d, want 16×16", w, h)
	}
}

// Display and backdrop requests used separate downloads and bitmaps, even for the same URL.
func TestArtCacheSharesCoverAndBackdrop(t *testing.T) {
	for _, order := range []struct {
		name   string
		first  Needs
		during bool
	}{
		{"cover-then-backdrop", Cover, false},
		{"backdrop-during-cover", Cover, true},
		{"cover-during-backdrop", Backdrop, true},
		{"tint-then-backdrop", Tint, false},
	} {
		t.Run(order.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				data := coverData(t)
				posted := make(chan func(), 1)
				c := New(func(fn func()) { posted <- fn })
				fetches, releases := 0, 0
				c.fetch = func(string) ([]byte, error) {
					fetches++
					return data, nil
				}
				c.release = func(*ui.Bitmap) { releases++ }
				c.Get("cover", order.first)
				if order.during {
					c.Get("cover", Cover)
					c.Get("cover", Backdrop)
				}
				finishArtWork(t, posted)
				base := c.Get("cover", Cover)
				if base == nil || base.Failed || base.BMP == nil {
					t.Fatal("base cover did not load")
				}
				bmp := base.BMP
				if !order.during {
					if c.Get("cover", Backdrop) != nil {
						t.Fatal("returned incomplete backdrop resources")
					}
				}
				for len(c.busy) != 0 {
					finishArtWork(t, posted)
				}
				a := c.Get("cover", Backdrop)
				if a != base || a.BMP != bmp || c.Get("cover", Cover) != a {
					t.Fatal("deriving a backdrop replaced the shared cover or bitmap")
				}
				if a.Backdrop == nil || a.Tint == (ui.Color{}) {
					t.Fatal("backdrop request did not produce colors and a blurred image")
				}
				w, h := a.Backdrop.Size()
				want := len(data) + w*h*4
				if fetches != 1 || releases != 1 || len(c.art) != 1 || c.size != want || a.Size != want {
					t.Fatalf("shared cover: fetches=%d releases=%d entries=%d bytes=%d/%d, want 1/1/1/%d", fetches, releases, len(c.art), c.size, a.Size, want)
				}
			})
		})
	}
}

// Color-only users such as the notch retain their tint without allocating a blurred background.
func TestArtCacheTintNeedsNoBackdrop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		posted := make(chan func(), 1)
		c := New(func(fn func()) { posted <- fn })
		data := coverData(t)
		c.fetch = func(string) ([]byte, error) { return data, nil }
		c.release = func(*ui.Bitmap) {}
		c.Get("cover", Tint)
		finishArtWork(t, posted)
		a := c.Get("cover", Tint)
		if a == nil || a.Failed || a.Tint == (ui.Color{}) || a.Backdrop != nil {
			t.Fatal("color-only request lost its tint or generated an unused backdrop")
		}
		if a.Size != len(data) || c.size != len(data) {
			t.Fatal("color-only request counted unused backdrop pixels")
		}
	})
}

// An upgrade finishing after eviction must revive the same held cover and count its new pixels once.
func TestArtCacheCountsEvictedBackdropUpgrade(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		posted := make(chan func(), 1)
		c := New(func(fn func()) { posted <- fn })
		data := coverData(t)
		fetches := 0
		c.fetch = func(string) ([]byte, error) { fetches++; return data, nil }
		c.release = func(*ui.Bitmap) {}
		now := time.Unix(1e9, 0)
		c.now = func() time.Time { return now }
		c.Get("cover", Cover)
		finishArtWork(t, posted)
		held := c.Get("cover", Cover)
		c.Get("cover", Backdrop)
		c.limit = len(data) * 3 / 2
		now = now.Add(time.Minute)
		c.Add("other", &Artwork{Used: now, Size: len(data)})
		if c.art["cover"] != nil {
			t.Fatal("fixture did not evict the cover while its backdrop was pending")
		}
		c.limit = artLimit
		finishArtWork(t, posted)
		if got := c.Get("cover", Backdrop); got != held || got.Backdrop == nil {
			t.Fatal("upgrade lost a held cover or its derived backdrop")
		}
		want := c.art["other"].Size + held.Size
		if c.size != want || fetches != 1 {
			t.Fatalf("upgraded cache keeps %d bytes after %d fetches, want %d bytes after one fetch", c.size, fetches, want)
		}
	})
}

// Reviving a held cover under pressure must update recency before choosing which entries to evict.
func TestArtCacheRevivesCoverBeforeEviction(t *testing.T) {
	now := time.Unix(1e9, 0)
	c := New(func(func()) {})
	c.now, c.limit = func() time.Time { return now }, 100
	held := &Artwork{Used: now, Size: 30}
	c.Add("held", held)
	now = now.Add(time.Minute)
	c.Add("older", &Artwork{Used: now, Size: 40})
	now = now.Add(time.Minute)
	c.Add("newer", &Artwork{Used: now, Size: 40})
	if c.art["held"] != nil {
		t.Fatal("fixture did not evict the held cover")
	}
	now = now.Add(time.Minute)
	c.Add("fresh", &Artwork{Used: now, Size: 40})
	now = now.Add(time.Minute)
	if got := c.Get("held", Cover); got != held || c.art["held"] != held {
		t.Fatal("revived cover was immediately evicted using its old access time")
	}
}

// Compare display-only decoding with the color measurement ordinary covers previously performed.
func BenchmarkArtDecode(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 800, 800))
	for i := 0; i < len(img.Pix); i += 4 {
		copy(img.Pix[i:i+4], []byte{180, 40, 80, 255})
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 85}); err != nil {
		b.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		need Needs
	}{{"cover", Cover}, {"with-tint", Tint}} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				a, err := DecodeArt(encoded.Bytes(), tc.need)
				if err != nil {
					b.Fatal(err)
				}
				runtime.KeepAlive(a)
			}
		})
	}
}
