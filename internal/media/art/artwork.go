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

	"lilt/internal/ui/base/theme"
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
	fetch := c.fetch
	go func() {
		c.sem <- struct{}{}
		var result *Artwork
		var err error
		if a == nil {
			data, err = fetch(url)
			if err == nil {
				result, err = DecodeArt(data, need)
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
	out := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := range n {
		for x := range n {
			x0, x1 := b.Min.X+x*b.Dx()/n, b.Min.X+(x+1)*b.Dx()/n
			y0, y1 := b.Min.Y+y*b.Dy()/n, b.Min.Y+(y+1)*b.Dy()/n
			var r, g, bl, cnt uint32
			for yy := y0; yy < max(y1, y0+1); yy += max(1, (y1-y0)/4) {
				for xx := x0; xx < max(x1, x0+1); xx += max(1, (x1-x0)/4) {
					cr, cg, cb, _ := img.At(xx, yy).RGBA()
					r, g, bl, cnt = cr>>8, cg>>8, cb>>8, cnt+1
				}
			}
			out.SetRGBA(x, y, color.RGBA{uint8(r / cnt), uint8(g / cnt), uint8(bl / cnt), 255})
		}
	}
	return out
}

// DominantColor returns the most vivid color that covers much of img: the
// hue that its saturated pixels share most, at their mean.
func DominantColor(img image.Image) ui.Color {
	small := shrink(img, 24)
	const buckets = 18
	var weight [buckets]float64
	var sum [buckets][3]float64
	var gray [3]float64
	for i := 0; i < len(small.Pix); i += 4 {
		r, g, b := float64(small.Pix[i])/255, float64(small.Pix[i+1])/255, float64(small.Pix[i+2])/255
		h, s, v := theme.HSV(r, g, b)
		gray[0], gray[1], gray[2] = gray[0]+r, gray[1]+g, gray[2]+b
		w := s * s * v
		if s < 0.15 || v < 0.15 {
			continue
		}
		k := int(h/360*buckets) % buckets
		weight[k] += w
		sum[k][0], sum[k][1], sum[k][2] = sum[k][0]+r*w, sum[k][1]+g*w, sum[k][2]+b*w
	}
	best := 0
	for k := range weight {
		if weight[k] > weight[best] {
			best = k
		}
	}
	if weight[best] < 2 {
		n := float64(len(small.Pix) / 4)
		return ui.RGB(uint8(gray[0]/n*255), uint8(gray[1]/n*255), uint8(gray[2]/n*255))
	}
	w := weight[best]
	return ui.RGB(uint8(sum[best][0]/w*255), uint8(sum[best][1]/w*255), uint8(sum[best][2]/w*255))
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
