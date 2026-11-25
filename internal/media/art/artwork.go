package art

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"slices"
	"time"
	"weak"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	_ "golang.org/x/image/webp"

	"lilt/internal/media/qqmusic"
)

// Artwork is a cover, decoded and ready to draw, with what the player
// derives from it: its color, and a blurred copy to fill a backdrop.
type Artwork struct {
	BMP      *ui.Bitmap
	Backdrop *ui.Bitmap
	Tint     ui.Color // the cover's most vivid color
	Data     []byte   // the same encoded data the bitmap keeps to restore its pixels
	Ready    Needs    // which derived resources have been made
	Failed   bool
	Used     time.Time // when it was last asked for
	Size     int       // the bytes it keeps: its data, and the backdrop's pixels
}

// Needs are the resources a caller asks of a cover. A plain image needs
// no color measurement or separately decoded image.
type Needs uint8

const (
	Cover    Needs = 0
	Tint     Needs = 1
	Blur     Needs = 2
	Backdrop       = Tint | Blur
)

// A request's needs can grow while its worker runs; only the main thread
// reads or changes them. Extra resources use the same downloaded data.
type artRequest struct {
	want Needs
}

// DiskCache keeps cover data on disk, so covers survive restarts.
type DiskCache interface {
	LookupCover(url string) ([]byte, bool)
	KeepCover(url string, data []byte) bool
	DropCover(url string)
}

// Cache loads covers on goroutines and keeps them, up to artLimit
// bytes: their pixels on the GPU while they show, and on the CPU only
// their data, which takes a fraction of the pixels. Its methods run on
// the main thread; post runs a function there.
type Cache struct {
	art map[string]*Artwork
	// gone are the covers dropped, for as long as something else holds
	// them: Now Playing keeps the cover it shows, which comes back as it
	// was rather than loading again.
	gone    map[string]weak.Pointer[Artwork]
	busy    map[string]*artRequest
	sem     chan struct{}
	post    func(func())
	fetch   func(string) ([]byte, error)
	disk    DiskCache
	release func(*ui.Bitmap)
	now     func() time.Time
	size    int // the bytes the covers in art keep
	limit   int
}

// artLimit budgets encoded cover data and backdrop pixels; decoded pixels,
// GPU textures and covers held outside the cache are not included.
const artLimit = 64 << 20

// artRetry keeps a transient failure from being requested every frame, without
// caching it forever now that successful covers no longer expire by time.
const artRetry = 5 * time.Minute

func New(post func(func())) *Cache {
	return &Cache{
		art: map[string]*Artwork{}, gone: map[string]weak.Pointer[Artwork]{}, busy: map[string]*artRequest{},
		sem: make(chan struct{}, 8), post: post, fetch: qqmusic.Fetch, release: releaseArtPixels,
		now: time.Now, limit: artLimit,
	}
}

// SetDiskCache keeps covers in disk between runs, or stops doing so when nil.
func (c *Cache) SetDiskCache(d DiskCache) { c.disk = d }

// Reserve holds a cover offline until Release, as a request being made.
func (c *Cache) Reserve(url string) { c.busy[url] = &artRequest{} }

// Release frees a cover held by Reserve.
func (c *Cache) Release(url string) { delete(c.busy, url) }

// Clock is the cache's time, which tests set.
func (c *Cache) Clock() time.Time { return c.now() }

// SetClock sets the cache's time, which tests do.
func (c *Cache) SetClock(now func() time.Time) { c.now = now }

// Busy is how many covers are being fetched now.
func (c *Cache) Busy() int { return len(c.busy) }

// Peek is the cover kept for url, without marking it as asked for, or nil.
func (c *Cache) Peek(url string) *Artwork { return c.art[url] }

// Add keeps a cover, and drops those asked for least recently once the
// covers take more than the limit, down to three quarters of it, so that
// it sorts them seldom. Those showing were asked for last, and stay.
func (c *Cache) Add(url string, a *Artwork) {
	if old := c.art[url]; old != nil {
		c.size -= old.Size
	}
	c.art[url] = a
	delete(c.gone, url)
	c.size += a.Size
	c.trim()
}

func (c *Cache) trim() {
	if c.size <= c.limit {
		return
	}
	for key, p := range c.gone {
		if p.Value() == nil {
			delete(c.gone, key)
		}
	}
	keys := make([]string, 0, len(c.art))
	for key := range c.art {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(x, y string) int { return c.art[x].Used.Compare(c.art[y].Used) })
	for _, key := range keys {
		if c.size <= c.limit*3/4 {
			break
		}
		a := c.art[key]
		delete(c.art, key)
		c.size -= a.Size
		c.gone[key] = weak.Make(a)
	}
}

// Get returns the shared cover once the requested resources are ready.
// Plain images need no color measurement or separately decoded image.
func (c *Cache) Get(url string, need Needs) *Artwork {
	if url == "" {
		return nil
	}
	now := c.now()
	a := c.art[url]
	if a == nil {
		a = c.gone[url].Value()
		delete(c.gone, url)
		if a != nil && (!a.Failed || now.Sub(a.Used) < artRetry) {
			if !a.Failed {
				a.Used = now
			}
			c.Add(url, a)
		}
	}
	if a != nil {
		if !a.Failed || now.Sub(a.Used) < artRetry {
			if !a.Failed {
				a.Used = now
			}
			if a.Failed || a.Ready&need == need {
				return a
			}
		} else {
			if c.art[url] == a {
				delete(c.art, url)
				c.size -= a.Size
			}
			a = nil
		}
	}
	if request := c.busy[url]; request != nil {
		request.want |= need
		return nil
	}
	if a != nil {
		need &^= a.Ready
	}
	c.load(url, a, need)
	return nil
}

func (c *Cache) load(url string, a *Artwork, need Needs) {
	request := &artRequest{want: need}
	c.busy[url] = request
	var data []byte
	if a != nil {
		data = a.Data
	}
	fetch, disk := c.fetch, c.disk
	go func() {
		c.sem <- struct{}{}
		var result *Artwork
		var err error
		if a == nil {
			if disk != nil {
				if cached, ok := disk.LookupCover(url); ok {
					result, err = DecodeArt(cached, need)
					if err != nil {
						disk.DropCover(url)
						result, err = nil, nil
					}
				}
			}
			if result == nil {
				data, err = fetch(url)
				if err == nil {
					result, err = DecodeArt(data, need)
					if err == nil && disk != nil {
						disk.KeepCover(url, data)
					}
				}
			}
		} else {
			result, err = deriveArt(data, need)
		}
		<-c.sem
		c.post(func() {
			delete(c.busy, url)
			if err != nil {
				c.Add(url, &Artwork{Failed: true, Used: c.now()})
				return
			}
			if a == nil {
				a = result
				a.Used = c.now()
				c.Add(url, a)
				// Let the first frame and its smaller copies upload before
				// dropping pixels; encoded data can restore them as before.
				c.release(a.BMP)
			} else {
				if need&Tint != 0 {
					a.Tint = result.Tint
				}
				if need&Blur != 0 {
					a.Backdrop = result.Backdrop
				}
				a.Ready |= need
				a.Size += result.Size
				if c.art[url] == a {
					c.size += result.Size
					c.trim()
				} else {
					c.Add(url, a)
				}
			}
			if missing := request.want &^ a.Ready; missing != 0 {
				c.load(url, a, missing)
			}
		})
	}()
}

// artRelease is how long a cover keeps its pixels once loaded: past the
// frame that first shows it, and the smaller copies it shows from there.
const artRelease = 2 * time.Second

func releaseArtPixels(bmp *ui.Bitmap) {
	time.AfterFunc(artRelease, func() { mygo.RunOnMain(bmp.ReleasePixels) })
}

// DecodeArt always keeps the bitmap's compressed restore source. Only
// callers asking for colors or a backdrop need a second decoded image.
func DecodeArt(b []byte, need Needs) (*Artwork, error) {
	size := len(b)
	b = stillCover(b)
	bmp, err := ui.DecodeBitmap(b)
	if err != nil {
		return nil, err
	}
	a, err := deriveArt(b, need)
	if err != nil {
		return nil, err
	}
	a.BMP, a.Data, a.Size = bmp, b, size+a.Size
	return a, nil
}

func deriveArt(data []byte, need Needs) (*Artwork, error) {
	a := &Artwork{Ready: need}
	if need == Cover {
		return a, nil
	}
	img, err := decodeImage(data)
	if err != nil {
		return nil, err
	}
	if need&Tint != 0 {
		a.Tint = DominantColor(img)
	}
	if need&Blur != 0 {
		blurred := BlurImage(img)
		a.Backdrop = ui.NewBitmap(blurred)
		a.Size = len(blurred.Pix)
	}
	return a, nil
}

// Decode returns a cover as a bitmap to draw and an image to measure.
func Decode(b []byte) (*ui.Bitmap, image.Image, error) {
	b = stillCover(b)
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, nil, err
	}
	bmp, err := ui.DecodeBitmap(b)
	return bmp, img, err
}

func FetchImage(url string) (image.Image, error) {
	b, err := qqmusic.Fetch(url)
	if err != nil {
		return nil, err
	}
	return decodeImage(b)
}

func decodeImage(b []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(stillCover(b)))
	return img, err
}

// stillCover returns the data of a cover that the decoders read. Some
// covers are animated WebP, whatever their name says, which they cannot:
// those show their first frame.
func stillCover(b []byte) []byte {
	if still := firstWebPFrame(b); still != nil {
		return still
	}
	return b
}

// firstWebPFrame returns the first frame of an animated WebP as a still
// WebP of its own, or nil if b is not one.
func firstWebPFrame(b []byte) []byte {
	if len(b) < 12 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return nil
	}
	chunk := func(b []byte) (name string, body, rest []byte, ok bool) {
		if len(b) < 8 {
			return "", nil, nil, false
		}
		n := int(binary.LittleEndian.Uint32(b[4:8]))
		if n > len(b)-8 {
			return "", nil, nil, false
		}
		end := min(len(b), 8+n+n&1)
		return string(b[:4]), b[8 : 8+n], b[end:], true
	}
	riff := func(chunks ...[]byte) []byte {
		body := []byte("WEBP")
		for _, c := range chunks {
			body = append(body, c...)
		}
		out := binary.LittleEndian.AppendUint32([]byte("RIFF"), uint32(len(body)))
		return append(out, body...)
	}
	for rest := b[12:]; ; {
		name, body, next, ok := chunk(rest)
		if !ok {
			return nil
		}
		rest = next
		if name != "ANMF" || len(body) < 16 {
			continue
		}
		// A frame: where it sits, its size, how long it shows, then its
		// image as the chunks of a still WebP, its alpha first if any.
		w, h := body[6:9], body[9:12]
		frame := body[16:]
		var alpha []byte
		for len(frame) > 0 {
			name, _, after, ok := chunk(frame)
			if !ok {
				return nil
			}
			whole := frame[:len(frame)-len(after)]
			switch name {
			case "ALPH":
				alpha = whole
			case "VP8L":
				return riff(whole)
			case "VP8 ":
				if alpha == nil {
					return riff(whole)
				}
				// Alpha needs the extended header, which carries the size.
				vp8x := []byte{'V', 'P', '8', 'X', 10, 0, 0, 0, 0x10, 0, 0, 0}
				vp8x = append(append(vp8x, w...), h...)
				return riff(vp8x, alpha, whole)
			}
			frame = after
		}
		return nil
	}
}

// shrink averages img down to n×n pixels.
func shrink(img image.Image, n int) *image.RGBA {
	b := img.Bounds()
	rgba64, _ := img.(image.RGBA64Image)
	out := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := range n {
		for x := range n {
			x0, x1 := b.Min.X+x*b.Dx()/n, b.Min.X+(x+1)*b.Dx()/n
			y0, y1 := b.Min.Y+y*b.Dy()/n, b.Min.Y+(y+1)*b.Dy()/n
			var r, g, bl, cnt uint32
			for yy := y0; yy < max(y1, y0+1); yy += max(1, (y1-y0)/4) {
				for xx := x0; xx < max(x1, x0+1); xx += max(1, (x1-x0)/4) {
					var cr, cg, cb uint32
					if rgba64 != nil {
						p := rgba64.RGBA64At(xx, yy)
						cr, cg, cb = uint32(p.R), uint32(p.G), uint32(p.B)
					} else {
						cr, cg, cb, _ = img.At(xx, yy).RGBA()
					}
					r, g, bl, cnt = r+(cr>>8), g+(cg>>8), bl+(cb>>8), cnt+1
				}
			}
			out.SetRGBA(x, y, color.RGBA{uint8(r / cnt), uint8(g / cnt), uint8(bl / cnt), 255})
		}
	}
	return out
}

// dominantSize is the cover's edge after shrinking: big enough for k-means to
// tell its regions apart, small enough to keep loading quick.
const dominantSize = 32

const (
	// dominantClusters is how many colors k-means separates before merging.
	dominantClusters = 8
	// dominantIterations bounds the k-means refinement.
	dominantIterations = 24
	// mergeDistance is how close, in CIE76 ΔE, two clusters must be to merge.
	mergeDistance = 12

	// Android Palette's filter drops near-black, near-white and gray pixels,
	// which say nothing about a cover's color.
	filterMinLightness  = 0.15
	filterMaxLightness  = 0.95
	filterMinSaturation = 0.10

	// Palette's HSL scoring: how much a swatch's saturation, lightness and
	// population count toward being the cover's color.
	targetSaturation = 0.5
	targetLightness  = 0.5
	weightSaturation = 0.26
	weightLightness  = 0.30
	weightPopulation = 0.24
)

// labColor is a color in CIE Lab, where distance matches how different two
// colors look.
type labColor struct{ l, a, b float32 }

// dominantSwatch is a cluster of the cover: its center and how many pixels it
// covers.
type dominantSwatch struct {
	lab        labColor
	population float32
}

// DominantColor returns the color a cover is best recognized by. Its pixels
// are filtered as Android's Palette filters them, clustered with k-means in
// CIE Lab, clusters closer than ΔE 12 merged, and the rest scored by Palette's
// HSL model: saturation and lightness near half, weighted by population.
func DominantColor(img image.Image) ui.Color {
	small := shrink(img, dominantSize)
	pixels := make([]labColor, 0, len(small.Pix)/4)
	var gray [3]float64
	n := float64(len(small.Pix) / 4)
	for i := 0; i < len(small.Pix); i += 4 {
		r, g, b := float64(small.Pix[i])/255, float64(small.Pix[i+1])/255, float64(small.Pix[i+2])/255
		gray[0], gray[1], gray[2] = gray[0]+r, gray[1]+g, gray[2]+b
		_, s, l := rgbToHSL(float32(r), float32(g), float32(b))
		if l < filterMinLightness || l > filterMaxLightness || s < filterMinSaturation {
			continue
		}
		pixels = append(pixels, rgbToLab(float32(r), float32(g), float32(b)))
	}
	// A cover of only extremes and grays has no color to take.
	if float64(len(pixels)) < n/10 {
		return ui.RGB(uint8(gray[0]/n*255), uint8(gray[1]/n*255), uint8(gray[2]/n*255))
	}
	swatches := dominantSwatches(pixels)
	var best dominantSwatch
	bestScore, maxPopulation := float32(-1), float32(0)
	for _, sw := range swatches {
		maxPopulation = max(maxPopulation, sw.population)
	}
	for _, sw := range swatches {
		if score := paletteScore(sw, maxPopulation); score > bestScore {
			best, bestScore = sw, score
		}
	}
	r, g, b := labToRGB(best.lab)
	return ui.RGB(uint8(r*255+.5), uint8(g*255+.5), uint8(b*255+.5))
}

// dominantSwatches clusters the pixels in Lab and merges the clusters that
// ended up close, so one region split in two does not count twice.
func dominantSwatches(pixels []labColor) []dominantSwatch {
	centers := labSeeds(pixels, dominantClusters)
	counts := make([]float32, len(centers))
	for range dominantIterations {
		sums := make([]labColor, len(centers))
		counts = make([]float32, len(centers))
		for _, p := range pixels {
			k := nearestLab(centers, p)
			sums[k] = labColor{sums[k].l + p.l, sums[k].a + p.a, sums[k].b + p.b}
			counts[k]++
		}
		next := make([]labColor, 0, len(centers))
		nextCounts := make([]float32, 0, len(centers))
		settled := true
		for k, c := range centers {
			if counts[k] == 0 {
				continue
			}
			moved := labColor{sums[k].l / counts[k], sums[k].a / counts[k], sums[k].b / counts[k]}
			if labDistanceSquared(moved, c) > 0.25*0.25 {
				settled = false
			}
			next = append(next, moved)
			nextCounts = append(nextCounts, counts[k])
		}
		centers, counts = next, nextCounts
		if settled {
			break
		}
	}
	swatches := make([]dominantSwatch, len(centers))
	for i, c := range centers {
		swatches[i] = dominantSwatch{c, counts[i]}
	}
	return mergeSwatches(swatches)
}

// mergeSwatches merges clusters within ΔE mergeDistance of a heavier one,
// heaviest first, so a split region becomes one swatch of its whole weight.
func mergeSwatches(swatches []dominantSwatch) []dominantSwatch {
	slices.SortStableFunc(swatches, func(x, y dominantSwatch) int {
		switch {
		case x.population > y.population:
			return -1
		case x.population < y.population:
			return 1
		}
		return 0
	})
	merged := make([]dominantSwatch, 0, len(swatches))
	for _, sw := range swatches {
		mergeAt := -1
		for i := range merged {
			if labDistanceSquared(merged[i].lab, sw.lab) < mergeDistance*mergeDistance {
				mergeAt = i
				break
			}
		}
		if mergeAt < 0 {
			merged = append(merged, sw)
			continue
		}
		p := merged[mergeAt].population + sw.population
		merged[mergeAt].lab = labColor{
			(merged[mergeAt].lab.l*merged[mergeAt].population + sw.lab.l*sw.population) / p,
			(merged[mergeAt].lab.a*merged[mergeAt].population + sw.lab.a*sw.population) / p,
			(merged[mergeAt].lab.b*merged[mergeAt].population + sw.lab.b*sw.population) / p,
		}
		merged[mergeAt].population = p
	}
	return merged
}

// paletteScore is Palette's score of a swatch: saturation and lightness near
// the middle of their range, plus the swatch's share of the cover's pixels.
func paletteScore(sw dominantSwatch, maxPopulation float32) float32 {
	r, g, b := labToRGB(sw.lab)
	_, s, l := rgbToHSL(r, g, b)
	return weightSaturation*(1-abs32(targetSaturation-s)) +
		weightLightness*(1-abs32(targetLightness-l)) +
		weightPopulation*(sw.population/maxPopulation)
}

// labSeeds picks initial centers spread as far apart as the pixels allow,
// without a random source, so a cover always yields the same accent.
func labSeeds(pixels []labColor, k int) []labColor {
	seeds := make([]labColor, 1, k)
	seeds[0] = pixels[0]
	for len(seeds) < k {
		next, far := -1, float32(0)
		for i, p := range pixels {
			if d := nearestLabDistance(seeds, p); d > far {
				next, far = i, d
			}
		}
		if next < 0 {
			break
		}
		seeds = append(seeds, pixels[next])
	}
	return seeds
}

func nearestLab(centers []labColor, p labColor) int {
	best, bestD := 0, float32(math.MaxFloat32)
	for k, c := range centers {
		if d := labDistanceSquared(p, c); d < bestD {
			best, bestD = k, d
		}
	}
	return best
}

func nearestLabDistance(centers []labColor, p labColor) float32 {
	bestD := float32(math.MaxFloat32)
	for _, c := range centers {
		bestD = min(bestD, labDistanceSquared(p, c))
	}
	return bestD
}

// labDistanceSquared is the squared CIE76 distance between two colors.
func labDistanceSquared(x, y labColor) float32 {
	dl, da, db := x.l-y.l, x.a-y.a, x.b-y.b
	return dl*dl + da*da + db*db
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// rgbToHSL returns hue, saturation and lightness of a color in 0..1.
func rgbToHSL(r, g, b float32) (h, s, l float32) {
	hi, lo := max(r, g, b), min(r, g, b)
	l = (hi + lo) / 2
	if hi == lo {
		return 0, 0, l
	}
	d := hi - lo
	if l > 0.5 {
		s = d / (2 - hi - lo)
	} else {
		s = d / (hi + lo)
	}
	switch hi {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h / 6, s, l
}

// rgbToLab converts an sRGB color (D65) to CIE Lab.
func rgbToLab(r, g, b float32) labColor {
	lin := func(c float32) float32 {
		if c <= 0.04045 {
			return c / 12.92
		}
		return float32(math.Pow(float64((c+0.055)/1.055), 2.4))
	}
	rl, gl, bl := lin(r), lin(g), lin(b)
	x := 0.4124*rl + 0.3576*gl + 0.1805*bl
	y := 0.2126*rl + 0.7152*gl + 0.0722*bl
	z := 0.0193*rl + 0.1192*gl + 0.9505*bl
	f := func(t float32) float32 {
		if t > 0.008856 {
			return float32(math.Cbrt(float64(t)))
		}
		return 7.787*t + 16.0/116
	}
	fx, fy, fz := f(x/0.95047), f(y), f(z/1.08883)
	return labColor{116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)}
}

// labToRGB converts a CIE Lab color back to sRGB.
func labToRGB(c labColor) (r, g, b float32) {
	fy := (c.l + 16) / 116
	fx, fz := fy+c.a/500, fy-c.b/200
	inv := func(t float32) float32 {
		if t3 := t * t * t; t3 > 0.008856 {
			return t3
		}
		return (t - 16.0/116) / 7.787
	}
	x, y, z := 0.95047*inv(fx), inv(fy), 1.08883*inv(fz)
	enc := func(v float32) float32 {
		if v <= 0.0031308 {
			return 12.92 * v
		}
		return 1.055*float32(math.Pow(float64(v), 1.0/2.4)) - 0.055
	}
	return clamp01(enc(3.2406*x - 1.5372*y - 0.4986*z)),
		clamp01(enc(-0.9689*x + 1.8758*y + 0.0415*z)),
		clamp01(enc(0.0557*x - 0.2040*y + 1.0570*z))
}

func clamp01(x float32) float32 {
	return min(max(x, 0), 1)
}

// BlurImage returns img shrunk and blurred, more saturated, to fill a
// backdrop behind what plays, scaled up as smooth as it is.
func BlurImage(img image.Image) *image.RGBA {
	const n = 48
	small := shrink(img, n)
	px := make([][3]float64, n*n)
	for i := range px {
		r, g, b := float64(small.Pix[4*i]), float64(small.Pix[4*i+1]), float64(small.Pix[4*i+2])
		// More saturated, as light through colored glass.
		l := 0.3*r + 0.59*g + 0.11*b
		px[i] = [3]float64{l + (r-l)*1.5, l + (g-l)*1.5, l + (b-l)*1.5}
	}
	tmp := make([][3]float64, n*n)
	for range 3 {
		boxBlur(px, tmp, n, 5)
	}
	out := image.NewRGBA(image.Rect(0, 0, n, n))
	for i, p := range px {
		for c := range 3 {
			out.Pix[4*i+c] = uint8(math.Max(0, math.Min(255, p[c])))
		}
		out.Pix[4*i+3] = 255
	}
	return out
}

// boxBlur blurs px of n×n pixels by a box of radius r, across then down,
// clamping at the edges.
func boxBlur(px, tmp [][3]float64, n, r int) {
	at := func(i int) int { return max(0, min(n-1, i)) }
	for y := range n {
		for x := range n {
			var s [3]float64
			for k := -r; k <= r; k++ {
				p := px[y*n+at(x+k)]
				s[0], s[1], s[2] = s[0]+p[0], s[1]+p[1], s[2]+p[2]
			}
			d := float64(2*r + 1)
			tmp[y*n+x] = [3]float64{s[0] / d, s[1] / d, s[2] / d}
		}
	}
	for y := range n {
		for x := range n {
			var s [3]float64
			for k := -r; k <= r; k++ {
				p := tmp[at(y+k)*n+x]
				s[0], s[1], s[2] = s[0]+p[0], s[1]+p[1], s[2]+p[2]
			}
			d := float64(2*r + 1)
			px[y*n+x] = [3]float64{s[0] / d, s[1] / d, s[2] / d}
		}
	}
}
