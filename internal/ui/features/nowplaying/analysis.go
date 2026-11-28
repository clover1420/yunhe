package nowplaying

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"lilt/internal/media/art"
	"lilt/internal/media/lyrics"
	"lilt/internal/media/player"
	"lilt/internal/media/qqmusic"
	"lilt/internal/ui/base/appui"
	"lilt/internal/ui/base/transport"
)

// The analysis pane of Now Playing takes the whole view, laid out as the
// user asked, after a desktop wallpaper they liked: the cover with a
// record half out from behind it on a card of what plays; under it a card
// of what the music is doing, its two channels' spectrum, a spectrogram
// running by, its loudness over the last half minute, the tempo, the
// channels' levels, balance and width, and the readings; and under both,
// the line being sung. The player measures it (player.Analysis) only
// while the pane shows.
//
// It is laid out at a size of its own, anW across, and scaled to the
// window as one, as the wallpaper is to its screen. All that changes as
// the music plays is painted, never built: building Now Playing again
// takes long enough to drop a frame.

const (
	anW      = 1280 // the composition's least width, at scale 1
	anWMost  = 1560 // and its widest, in a wide window
	anH      = 1036 // and height, the lyrics included
	anCover  = 396  // the cover's side
	anInfoY  = 18   // the top card's top, under the cover's
	anInfoH  = 360
	anCardY  = 421 // the analysis card's top
	anCardH  = 420 // taller for a third row of readings
	anLyricY = 898 // the lyrics' top
	anTextX  = 615 // where the top card's text starts
	anLX     = 43  // where the analysis card's left column starts
)

// anCols are where the composition, W wide, has its parts: how wide the
// top card's text is, where the analysis card's right column starts, and
// how wide its left column is. The right column keeps its width; the
// rest grows with W.
func anCols(W float32) (textW, rc, lw float32) {
	rc = W - 361
	return W - anTextX - 56, rc, rc - 48 - anLX
}

// AnalysisState is what the pane shows, eased and kept from frame to frame.
type AnalysisState struct {
	stepped time.Time

	l, r, pkL, pkR [player.AnBands]float32 // the bands, eased, and their peaks
	lvL, lvR       float32                 // the channels' levels, 0 to 1
	holdL, holdR   float32                 // their peaks, held
	holdLAt        time.Time
	holdRAt        time.Time
	bal, wid       float32
	low, mid, high float32
	bright         float32
	corr           float32
	peakBand       int
	peakHz         float32
	live           bool      // measured lately
	liveAt         time.Time // when last

	// The loudness, ten times a second for 30 s: as shown, and linear,
	// for the dynamic range.
	loud, loudLin [300]float32
	loudN, loudI  int
	loudAt        time.Time
	dr            float32

	// What the player measures besides, read five times a second, for the
	// numbers to be read: the loudness, momentary and short-term, in LUFS;
	// the peak of the last moment and of the song so far, and how often it
	// reached full scale; and the key, how sure, over how much of the song.
	// of is the song these are of.
	readAt         time.Time
	lufsM, lufsS   float64
	lufsOK         bool
	peak, peakSong float32
	clips          int
	clipAt         time.Time
	key            string
	keySure        float32
	of             int64
	// paused is 0 playing to 1 paused, eased: paused, all holds as it was,
	// greyed.
	paused float32

	// The tempo: the low end of the chunks played lately, its estimates,
	// and a beat, fading.
	lows    [420]float32
	bpmHist []float32
	bpm     float32
	tempoAt time.Time
	beat    float32
	beatAt  time.Time
	flux    float32 // the low end's rises, on average
	lastLow float32

	// The spectrogram: two canvases, each the last drawn moved a column
	// on, and when it last moved.
	gram   [2]*ui.Canvas
	front  int
	gramAt time.Time

	// The record: its turn, in degrees, how fast it turns, 0 still to 1 at
	// speed, coming up to speed and running down as a turntable's platter
	// does, and when it last turned. It turns in its own drawing, which
	// asks for its frames: turned only in the card's, it moved only when
	// the whole window was painted, a few times a second.
	turn  float64
	spin  float32
	recAt time.Time
	// labels are the record's labels, each a cover cut round, drawn once:
	// turning, a label is an image under a turn, with no clip, which the
	// GPU draws; a clip under a turn falls to the CPU. Two, for the sleeve
	// going as another song comes keeps its own.
	labels [2]anLabel

	// The sleeve, the cover with its record: the song it holds, where it
	// is in the play order, and where it goes as another comes, as the
	// cover in the lyrics' view does; outAt is when the new song's record
	// began to come out of its sleeve, zero when out.
	sleeveTrack int64
	sleeveAt    int
	sleeveExit  *ui.Motion
	outAt       time.Time

	// The lyrics, as the wallpaper has them: the line sung leaves, down and
	// fading, and comes back as the next; the line under it changes on its
	// own, fading over.
	lineTrack int64
	line      int
	main      string // the line sung, as shown
	mainNext  string // what takes its place as it comes back
	mainAt    time.Time
	sub       string // the line under it, as shown
	subWas    string // the one before, fading out
	subAt     time.Time
}

// anLabel is a record's label: the cover of, cut round in canvas, and
// when it was last drawn.
type anLabel struct {
	canvas *ui.Canvas
	of     *ui.Bitmap
	used   time.Time
}

// anGramW is how many columns the spectrogram keeps.
const anGramW = 220

// A new song's record comes out of its sleeve: it waits anOutWait, the
// sleeve coming in, then slides out in anOut, from anTuck further left,
// where the cover hides it whole.
const (
	anOutWait = 220 * time.Millisecond
	anOut     = 620 * time.Millisecond
	anTuck    = 165
)

// anSleeveKey and anWhatKey are the keys of the sleeve and of what plays,
// each a song's own, for another song's to come in as it goes.
type (
	anSleeveKey struct{ track int64 }
	anWhatKey   struct{ track int64 }
)

// anTextSwap is how what plays fades over to another song's.
var anTextSwap = ui.Animation{Duration: 380 * time.Millisecond, Ease: ui.EaseOut}

// The sleeve, the cover and its record, in the composition: from the
// cover's left to the record's right, the record anRecX into it.
const (
	anSleeveX = 32
	anRecX    = 187
	anRecY    = 22
	anRecD    = 352
)

// The lyrics' changes, as the wallpaper times them: the line sung goes in
// anLineOut and comes back in anLineIn; the line under it fades over in
// anSubSwap.
const (
	anLineOut = 200 * time.Millisecond
	anLineIn  = 280 * time.Millisecond
	anSubSwap = 450 * time.Millisecond
)

// step brings v to what plays at now, once a frame.
func (v *AnalysisState) step(now time.Time, pl *player.Player) {
	if now == v.stepped {
		return
	}
	dt := float32(now.Sub(v.stepped).Seconds())
	if v.stepped.IsZero() || dt > 0.25 {
		dt = 1.0 / 60
	}
	v.stepped = now
	playing := pl.Playing()
	pk := 1 - expf(-dt*8)
	if playing {
		v.paused -= v.paused * pk
	} else {
		v.paused += (1 - v.paused) * pk
	}
	v.beat = max(0, v.beat-dt*4)
	if cur := pl.Current(); cur != nil && cur.ID != v.of {
		// Another song: what was of the last one goes.
		v.of = cur.ID
		v.peakSong, v.clips, v.key, v.keySure = 0, 0, "", 0
		v.readAt = time.Time{}
	}
	if !playing {
		// Paused, all stays as it was, rather than falling away: still,
		// and greyed, it says so.
		return
	}
	an, ok := pl.Analysis()
	if ok {
		v.live, v.liveAt = true, now
	} else if now.Sub(v.liveAt) > 400*time.Millisecond {
		v.live = false
	}
	up, down := 1-expf(-dt*24), 1-expf(-dt*7)
	ease := func(x *float32, to float32) {
		k := down
		if to > *x {
			k = up
		}
		*x += (to - *x) * k
	}
	var sumL, sumR, diff, wsum, top float32
	var low, mid, high float32
	topI := -1
	for i := range player.AnBands {
		var a, b float32
		if ok {
			a, b = an.L[i], an.R[i]
		}
		ease(&v.l[i], a)
		ease(&v.r[i], b)
		v.pkL[i] = max(v.l[i], v.pkL[i]-dt*0.35)
		v.pkR[i] = max(v.r[i], v.pkR[i]-dt*0.35)
		sumL += v.l[i]
		sumR += v.r[i]
		diff += abs32(v.l[i] - v.r[i])
		mono := (v.l[i] + v.r[i]) / 2
		if mono > top {
			top, topI = mono, i
		}
		wsum += mono * float32(i)
		switch {
		case i < 12:
			low += mono
		case i < 36:
			mid += mono
		default:
			high += mono
		}
	}
	level := func(rms float32) float32 {
		if rms <= 0 {
			return 0
		}
		return clamp01((20*float32(math.Log10(float64(rms))) + 54) / 60)
	}
	var tl, tr float32
	if ok {
		tl, tr = level(an.RmsL), level(an.RmsR)
	}
	ease(&v.lvL, tl)
	ease(&v.lvR, tr)
	hold := func(lv float32, h *float32, at *time.Time) {
		if lv >= *h {
			*h, *at = lv, now
		} else if now.Sub(*at) > 900*time.Millisecond {
			*h = max(lv, *h-dt*0.5)
		}
	}
	hold(v.lvL, &v.holdL, &v.holdLAt)
	hold(v.lvR, &v.holdR, &v.holdRAt)
	s := 1 - expf(-dt*6)
	tot := sumL + sumR
	bal, wid, bright := float32(0), float32(0), float32(0)
	if tot > 0.05 {
		bal = (sumR - sumL) / tot
		wid = clamp01(diff / tot * 2.5)
		bright = clamp01(wsum / (tot / 2) / 40)
	}
	v.bal += (bal - v.bal) * s
	v.wid += (wid - v.wid) * s
	v.bright += (bright - v.bright) * s
	v.low += (clamp01(low/12*1.25) - v.low) * s
	v.mid += (clamp01(mid/24*1.4) - v.mid) * s
	v.high += (clamp01(high/28*1.7) - v.high) * s
	corr := float32(0)
	if ok {
		corr = an.Corr
	}
	v.corr += (corr - v.corr) * s
	v.peakBand = -1
	if top > 0.04 {
		v.peakBand = topI
		v.peakHz = pl.AnalysisBandHz(topI)
	}
	// A beat: the low end rising well beyond how it has been rising.
	if ok {
		rise := max(0, an.Low-v.lastLow)
		v.lastLow = an.Low
		v.flux += (rise - v.flux) * 0.05
		if rise > v.flux*1.8+0.03 && now.Sub(v.beatAt) > 220*time.Millisecond {
			v.beat, v.beatAt = 1, now
		}
	}
	if now.Sub(v.readAt) >= 200*time.Millisecond {
		v.readAt = now
		v.lufsM, v.lufsS, v.lufsOK = pl.AnalysisLoudness()
		v.peak = pl.AnalysisPeak(0.3)
		v.peakSong = max(v.peakSong, v.peak)
		if v.peak >= anFull && now.Sub(v.clipAt) > 500*time.Millisecond {
			v.clips++
			v.clipAt = now
		}
		if ch, secs := pl.AnalysisChroma(); secs >= 8 {
			v.key, v.keySure = keyOf(ch)
		}
	}
	if now.Sub(v.loudAt) >= 100*time.Millisecond {
		v.loudAt = now
		rms := float32(0)
		if ok {
			rms = (an.RmsL + an.RmsR) / 2
		}
		shown := (v.lvL + v.lvR) / 2
		if v.lufsOK {
			shown = lufsShown(v.lufsS)
		}
		v.loud[v.loudI], v.loudLin[v.loudI] = shown, rms
		v.loudI = (v.loudI + 1) % len(v.loud)
		v.loudN = min(v.loudN+1, len(v.loud))
		v.dr = dynamicRange(v.loudLin[:], v.loudI, v.loudN)
	}
	if now.Sub(v.tempoAt) >= time.Second {
		v.tempoAt = now
		n, per := pl.AnalysisLows(v.lows[:])
		if bpm, ok := tempoOf(v.lows[len(v.lows)-n:], per); ok {
			v.bpmHist = append(v.bpmHist, bpm)
			if len(v.bpmHist) > 7 {
				v.bpmHist = v.bpmHist[1:]
			}
			sorted := slices.Clone(v.bpmHist)
			slices.Sort(sorted)
			v.bpm = sorted[len(sorted)/2]
		} else if !v.live && now.Sub(v.liveAt) > 20*time.Second {
			v.bpmHist, v.bpm = v.bpmHist[:0], 0
		}
	}
}

// moving reports whether anything in the pane still moves.
func (v *AnalysisState) moving(playing bool) bool {
	// Paused, all holds: it moves only while greying.
	return playing || v.beat > 0 || v.paused < 0.995
}

func expf(x float32) float32 { return float32(math.Exp(float64(x))) }

// anFull is a peak at full scale, 0 dBFS, as decoded: over it, what was
// mastered clips.
const anFull = 0.999

// lufsShown places a loudness on the graph of the last 30 s: -30 LUFS at
// its foot, -4 at its top, where music is.
func lufsShown(lufs float64) float32 { return clamp01(float32((lufs + 30) / 26)) }

// The key profiles of Krumhansl and Kessler: how much each pitch class,
// from the tonic up, is heard in a major key and in a minor one.
var (
	keyMajor = [12]float64{6.35, 2.23, 3.48, 2.33, 4.38, 4.09, 2.52, 5.19, 2.39, 3.66, 2.29, 2.88}
	keyMinor = [12]float64{6.33, 2.68, 3.52, 5.38, 2.60, 3.53, 2.54, 4.75, 3.98, 2.69, 3.34, 3.17}
	// The tonics as a key's name spells them.
	majorNames = [12]string{"C", "D♭", "D", "E♭", "E", "F", "F♯", "G", "A♭", "A", "B♭", "B"}
	minorNames = [12]string{"C", "C♯", "D", "E♭", "E", "F", "F♯", "G", "G♯", "A", "B♭", "B"}
)

// keyOf names the key the pitch classes chroma, C first, are likeliest
// in, as the Krumhansl-Schmuckler finds it: the profile, of the 24 keys,
// they are likest. sure is by how much it beat the next likeliest.
func keyOf(chroma [12]float32) (name string, sure float32) {
	corr := func(prof *[12]float64, tonic int) float64 {
		var mx, my float64
		for i := range 12 {
			mx += float64(chroma[(tonic+i)%12])
			my += prof[i]
		}
		mx, my = mx/12, my/12
		var sxy, sxx, syy float64
		for i := range 12 {
			x, y := float64(chroma[(tonic+i)%12])-mx, prof[i]-my
			sxy += x * y
			sxx += x * x
			syy += y * y
		}
		if sxx <= 0 || syy <= 0 {
			return 0
		}
		return sxy / math.Sqrt(sxx*syy)
	}
	best, next := -2.0, -2.0
	for tonic := range 12 {
		for minor, prof := range [2]*[12]float64{&keyMajor, &keyMinor} {
			r := corr(prof, tonic)
			if r > best {
				best, next = r, best
				if minor == 0 {
					name = majorNames[tonic] + " 大调"
				} else {
					name = minorNames[tonic] + " 小调"
				}
			} else if r > next {
				next = r
			}
		}
	}
	if best < 0.3 {
		return "", 0
	}
	return name, float32(best - next)
}

func dynamicRange(ring []float32, i, n int) float32 {
	k := min(n, 100)
	var peak, sum float32
	for j := 1; j <= k; j++ {
		x := ring[(i-j+len(ring))%len(ring)]
		peak = max(peak, x)
		sum += x
	}
	mean := sum / float32(max(k, 1))
	if k < 30 || mean < 0.003 {
		return 0
	}
	return 20 * float32(math.Log10(float64(peak/mean)))
}

// tempoOf finds the tempo of lows, the low end of what played, per
// chunks a second, oldest first: the lag at which its rises repeat most,
// leaning to tempos near 120. False with too little, or no beat.
func tempoOf(lows []float32, per float64) (float32, bool) {
	n := len(lows)
	if per <= 0 || float64(n) < per*5 {
		return 0, false
	}
	x := make([]float64, n)
	var mean float64
	for i := 1; i < n; i++ {
		x[i] = math.Max(0, float64(lows[i]-lows[i-1]))
		mean += x[i]
	}
	mean /= float64(n)
	var r0 float64
	for i := range x {
		x[i] -= mean
		r0 += x[i] * x[i]
	}
	if r0 < 1e-6 {
		return 0, false
	}
	r0 /= float64(n)
	lagMin := max(2, int(per*60/200))
	lagMax := min(n-20, int(math.Ceil(per*60/60)))
	if lagMax <= lagMin+1 {
		return 0, false
	}
	r := make([]float64, lagMax+2)
	for lag := lagMin - 1; lag <= lagMax+1; lag++ {
		var s float64
		for i := lag; i < n; i++ {
			s += x[i] * x[i-lag]
		}
		r[lag] = s / float64(n-lag)
	}
	best, bestScore := -1, -1.0
	for lag := lagMin; lag <= lagMax; lag++ {
		if r[lag] < r[lag-1] || r[lag] < r[lag+1] {
			continue
		}
		o := math.Log2(per * 60 / float64(lag) / 120)
		if score := r[lag] / r0 * math.Exp(-0.5*(o/0.8)*(o/0.8)); score > bestScore {
			best, bestScore = lag, score
		}
	}
	if best < 0 || r[best]/r0 < 0.1 {
		return 0, false
	}
	a, b, c := r[best-1], r[best], r[best+1]
	off := 0.0
	if den := a - 2*b + c; den != 0 {
		off = math.Max(-0.5, math.Min(0.5, 0.5*(a-c)/den))
	}
	return float32(per * 60 / (float64(best) + off)), true
}

// analysisScale is how large the composition is set in a view w×h with
// top taken: as large as fits, a little larger than its own size at most.
func analysisScale(w, h, top float32) float32 {
	return max(0.4, min(1.1, (w-72)/anW, (h-top-24)/anH))
}

// AnalysisPane fills Now Playing with the analysis of tr playing.
func AnalysisPane(c *ui.Context, d Deps, tr *qqmusic.Track, coverArt *art.Artwork, accent ui.Color, w, h float32) {
	pl := d.Player
	top := d.TopBar(c) + TabsHeight + 16
	s := analysisScale(w, h, top)
	// Set by the height, it takes the width the window has to spare.
	W := max(anW, min(anWMost, (w-96)/s))
	textW, _, _ := anCols(W)
	x0 := (w - W*s) / 2
	y0 := top + max(0, (h-top-anH*s)/2)
	st := d.Analysis
	bmp := bitmap(coverArt)
	S := func(v float32) float32 { return v * s }
	// The progress's row: the buttons, the time, the bar, the time.
	const ctlW = 150
	barX := float32(anTextX + ctlW + 58)
	ui.Box(c.Key("an")).Absolute().Left(x0).Top(y0).Width(S(W)).Height(S(anH)).Children(func() {
		// The top card, the record half out of its sleeve over it, the cover.
		info := ui.Box(c.Key("an-info")).Absolute().Left(0).Top(S(anInfoY)).Width(S(W)).Height(S(anInfoH))
		info.Draw(func(p *ui.Painter, r ui.Rect) {
			st.step(p.Now(), pl)
			anCard(p, r, s)
			// The times either side of the progress, and the little
			// spectrum under it.
			ty := r.Y + S(262) - S(9)
			t := ui.Span{Text: appui.Clock(pl.Position()), Size: max(11, S(13)), Weight: 500, Color: white.Alpha(0.6), Features: "tnum"}
			p.RichText(r.X+S(anTextX+ctlW), ty, S(80), t)
			t.Text = appui.Clock(pl.Duration())
			tw, _ := p.MeasureText(S(80), t)
			p.RichText(r.X+S(anTextX+textW)-tw, ty, S(80), t)
			anMiniSpectrum(p, ui.Rect{X: r.X + S(anTextX), Y: r.Y + S(298), W: S(textW), H: S(40)}, st, s)
			if pl.Playing() {
				// The clock again when its second turns.
				p.After(time.Second - pl.Position()%time.Second)
			}
		})
		// Another song: the sleeve before goes off the way the play order
		// went, the new one comes in from the other side, as the cover in
		// the lyrics' view, and its record comes out of it; what plays
		// fades over.
		var enter, textIn *ui.Motion
		if order, at := pl.PlayOrder(); st.sleeveTrack != tr.ID {
			if st.sleeveExit != nil && st.sleeveTrack != 0 {
				dir := float32(stepDir(st.sleeveAt, at, len(order)))
				// Its copy leaves from this frame on, which first knows the way.
				st.sleeveExit.X = -dir * S(anCover) * 0.5
				enter = &ui.Motion{X: dir * S(anCover) * 0.5, Scale: 0.92}
				textIn = &ui.Motion{Y: S(12)}
				if !c.Preferences().ReduceMotion {
					st.outAt = c.Now()
				}
			}
			st.sleeveTrack, st.sleeveAt, st.sleeveExit = tr.ID, at, &ui.Motion{Scale: 0.92}
		}
		id := tr.ID
		// The sleeves have a parent of their own, and what plays too: the
		// copy of one going goes under its siblings.
		ui.Box(c.Key("an-sleeves")).Absolute().Left(S(anSleeveX)).Top(0).Width(S(anRecX + anRecD)).Height(S(anCover)).Children(func() {
			ui.Box(c.Key(anSleeveKey{id})).Absolute().Left(0).Top(0).Width(S(anRecX + anRecD)).Height(S(anCover)).
				Transition(ui.ElementTransition{Animation: &coverSlide, Colors: true, Enter: enter, Exit: st.sleeveExit}).Children(func() {
				ui.Box(c.Key("an-record")).Absolute().Left(S(anRecX)).Top(S(anRecY)).Width(S(anRecD)).Height(S(anRecD)).Draw(func(p *ui.Painter, r ui.Rect) {
					// The sleeve going draws its record as it was, still.
					st.record(p, r, bmp, pl.Playing(), id != st.sleeveTrack, s)
				})
				ui.Box(c.Key("an-cover")).Absolute().Left(0).Top(0).Width(S(anCover)).Height(S(anCover)).Label("专辑封面").Draw(func(p *ui.Painter, r ui.Rect) {
					p.Shadow(r, S(22), 0, S(16), S(40), 0, ui.RGBA(0, 0, 0, 0.42))
					if bmp == nil {
						p.Fill(r, white.Alpha(0.08), S(22))
						return
					}
					p.Clip(r, S(22), func() { p.Image(bmp, r, ui.Cover) })
				})
			})
		})
		// What plays.
		ui.Box(c.Key("an-whats")).Absolute().Left(S(anTextX)).Top(S(anInfoY + 46)).Width(S(textW)).Children(func() {
			ui.Column(c.Key(anWhatKey{id})).Transition(ui.ElementTransition{Animation: &anTextSwap, Enter: textIn, Exit: &ui.Motion{}}).Width(S(textW)).Gap(S(6)).Children(func() {
				ui.Row(c).Gap(S(8)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Width(S(12)).Height(S(12)).Draw(func(p *ui.Painter, r ui.Rect) {
						// Three bars, the mark of something playing.
						for i, hh := range [3]float32{0.55, 1, 0.75} {
							bw := r.W / 5
							p.Fill(ui.Rect{X: r.X + float32(i)*bw*2, Y: r.Y + r.H*(1-hh), W: bw, H: r.H * hh}, accent, 0)
						}
					})
					ui.Text(c, "正在播放").FontSize(max(11, S(13))).FontWeight(700).LetterSpacing(S(2)).TextColor(white.Alpha(0.6))
					// Where it plays from, at the row's right: its format, level,
					// rate and bits, its bit rate. Painted, for what comes as it
					// loads to show without building the view.
					ui.Box(c.Key("an-source")).Grow(1).Height(S(24)).Label("音源信息").Draw(func(p *ui.Painter, r ui.Rect) {
						anSource(p, r, s, pl)
					})
				})
				ui.Text(c, tr.Name).FontSize(S(40)).FontWeight(800).SingleLine().Margin(S(14), 0, S(4), 0)
				d.ArtistLinks(c, tr, white.Alpha(0.85), white, true).FontSize(S(22)).SingleLine()
				if tr.Album.Name != "" {
					ui.Text(c, tr.Album.Name).FontSize(S(16)).TextColor(white.Alpha(0.5)).SingleLine()
				}
			})
		})
		// The buttons: the song before, play, the next.
		icon := max(16, S(25))
		ui.Row(c.Key("an-transport")).Absolute().Left(S(anTextX - 8)).Top(S(anInfoY+262) - icon).Height(2 * icon).
			Gap(S(2)).AlignItems(ui.Center).Children(func() {
			if anIconButton(c.Key("an-prev"), appui.IconPrev, icon*0.85, white.Alpha(0.85), "上一首").Clicked() {
				pl.Prev()
			}
			if anPlayButton(c.Key("an-play"), pl.Playing(), icon*1.15, "播放/暂停", white).Clicked() {
				pl.TogglePlay()
			}
			if anIconButton(c.Key("an-next"), appui.IconNext, icon*0.85, white.Alpha(0.85), "下一首").Clicked() {
				pl.Next()
			}
		})
		// Where it is: a bar to seek on, between the times painted on
		// the card.
		ui.Box(c.Key("an-progress")).Absolute().Left(S(barX)).Top(S(anInfoY + 262 - 8)).Width(S(anTextX+textW-58) - S(barX)).Height(S(16)).Children(func() {
			transport.Scrubber(c.Key("an-scrub"), pl, d.Settings.ProgressEffectMode(), d.AnalysisScrub, white.Alpha(0.2), accent, S(3), ui.Transparent)
		})
		// The analysis.
		card := ui.Box(c.Key("an-card")).Absolute().Left(0).Top(S(anCardY)).Width(S(W)).Height(S(anCardH))
		card.Draw(func(p *ui.Painter, r ui.Rect) {
			now := p.Now()
			st.step(now, pl)
			anCard(p, r, s)
			anDraw(p, r, s, W, accent, now, pl, st)
			if st.moving(pl.Playing()) {
				p.AnimationFrameRate(60)
			}
		})
		card.Children(func() { anHints(c, s, W) })
		// The line being sung.
		ui.Box(c.Key("an-lyrics")).Absolute().Left(0).Top(S(anLyricY)).Width(S(W)).Height(S(anH - anLyricY)).PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
			anLyrics(p, r, s, tr, pl, st)
		})
	})
}

// anCard paints a card of the composition at scale s: frosted glass, lit
// along its top.
func anCard(p *ui.Painter, r ui.Rect, s float32) {
	rad := 34 * s
	p.Shadow(r, rad, 0, 18*s, 48*s, 0, ui.RGBA(0, 0, 0, 0.22))
	p.Fill(r, white.Alpha(0.085), rad)
	p.FillGradient(r, ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
		{Color: white.Alpha(0.07), At: 0}, {Color: white.Alpha(0), At: 0.5},
	}}, rad)
	p.Stroke(r, white.Alpha(0.14), rad, 1)
}

// anIconButton is a small round button of the pane's controls.
func anIconButton(c *ui.Context, icon *ui.SVG, size float32, color ui.Color, label string) ui.Element {
	b := ui.ButtonBase(c).Size(max(28, size*1.6), max(28, size*1.6)).Center().Radius(999).
		FocusRing(false).Label(label).Tooltip(label)
	bg := ui.Transparent
	if b.Hovered() || b.FocusVisible() {
		bg = white.Alpha(0.1)
	}
	if b.Pressed() {
		bg = white.Alpha(0.16)
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 140 * time.Millisecond}))
	b.Children(func() { ui.Icon(c, icon).Size(size, size).TextColor(color) })
	return b
}

// anPlayButton is the play and pause button of the pane's controls.
func anPlayButton(c *ui.Context, playing bool, size float32, label string, color ui.Color) ui.Element {
	icon := appui.IconPlay
	if playing {
		icon = appui.IconPause
	}
	b := ui.ButtonBase(c).Size(max(32, size*1.5), max(32, size*1.5)).Center().Radius(999).
		FocusRing(false).Label(label).Tooltip(label)
	bg := white.Alpha(0.08)
	if b.Hovered() || b.FocusVisible() {
		bg = white.Alpha(0.16)
	}
	if b.Pressed() {
		bg = white.Alpha(0.22)
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 140 * time.Millisecond}))
	b.Children(func() { ui.Icon(c, icon).Size(size, size).TextColor(color) })
	return b
}

// record paints the record in r, its label the cover, bmp, at scale s,
// turning while playing, coming out of its sleeve as v has it. going, it
// is the record of the sleeve going as another comes, still as it was.
func (v *AnalysisState) record(p *ui.Painter, r ui.Rect, bmp *ui.Bitmap, playing, going bool, s float32) {
	now := p.Now()
	moving := false
	if !going {
		if now != v.recAt {
			dt := float32(now.Sub(v.recAt).Seconds())
			if v.recAt.IsZero() || dt > 0.25 {
				dt = 1.0 / 60
			}
			v.recAt = now
			to := float32(0)
			if playing {
				to = 1
			}
			// Up to speed in about half a second, run down in a little
			// more, as a platter's weight has it.
			k := 1 - expf(-dt*6)
			if to < v.spin {
				k = 1 - expf(-dt*3.5)
			}
			v.spin += (to - v.spin) * k
			if abs32(to-v.spin) < 0.002 {
				v.spin = to
			}
			// A turn every 1.8 s at speed, as a record at 33⅓.
			v.turn = math.Mod(v.turn+float64(dt*200*v.spin), 360)
		}
		moving = v.spin > 0
		if !v.outAt.IsZero() {
			k := clamp01(float32(now.Sub(v.outAt)-anOutWait) / float32(anOut))
			if k >= 1 {
				v.outAt = time.Time{}
			} else {
				r.X -= (1 - easeOutCubic(k)) * anTuck * s
				moving = true
			}
		}
	}
	if moving {
		// Its own frames: nothing else here paints it again.
		p.AnimationFrameRate(60)
	}
	R := r.W / 2
	cx, cy := r.X+R, r.Y+R
	p.Shadow(r, R, 0, R*0.04, R*0.12, 0, ui.RGBA(0, 0, 0, 0.45))
	p.Fill(r, ui.RGB(16, 16, 18), R)
	// The grooves.
	for i := range 14 {
		rr := R * (0.42 + 0.56*float32(i)/14)
		g := ui.Rect{X: cx - rr, Y: cy - rr, W: 2 * rr, H: 2 * rr}
		p.Stroke(g, white.Alpha(0.035+0.02*float32(i%3)), rr, 1)
	}
	// Its sheen, which stays as the record turns under it.
	p.FillConicGradient(r, ui.ConicGradient{Angle: 30, Stops: []ui.GradientStop{
		{Color: white.Alpha(0), At: 0}, {Color: white.Alpha(0.08), At: 0.08}, {Color: white.Alpha(0), At: 0.2},
		{Color: white.Alpha(0), At: 0.5}, {Color: white.Alpha(0.06), At: 0.58}, {Color: white.Alpha(0), At: 0.7},
		{Color: white.Alpha(0), At: 1},
	}}, R)
	lr := R * 0.36
	label := ui.Rect{X: cx - lr, Y: cy - lr, W: 2 * lr, H: 2 * lr}
	if bmp == nil {
		p.Fill(label, white.Alpha(0.3), lr)
	} else {
		lb := v.labelFor(p, bmp, now)
		p.Transform(ui.Rotation(float32(v.turn)), cx, cy, func() {
			p.ImageRegion(lb.Bitmap(), ui.Rect{}, label, ui.ImageOptions{})
		})
	}
	hole := R * 0.035
	p.Fill(ui.Rect{X: cx - hole, Y: cy - hole, W: 2 * hole, H: 2 * hole}, ui.RGB(12, 12, 14), hole)
}

// labelFor returns the label of bmp, cut round once, where nothing turns,
// in the label drawn least lately if none is bmp's yet.
func (v *AnalysisState) labelFor(p *ui.Painter, bmp *ui.Bitmap, now time.Time) *ui.Canvas {
	i := -1
	for j := range v.labels {
		if v.labels[j].canvas != nil && v.labels[j].of == bmp {
			i = j
		}
	}
	if i < 0 {
		i = 0
		if v.labels[1].used.Before(v.labels[0].used) {
			i = 1
		}
		lb := &v.labels[i]
		if lb.canvas == nil {
			lb.canvas = ui.NewCanvas(anLabelPx, anLabelPx)
		}
		lb.of = bmp
		lb.canvas.Draw(p, func(cp *ui.Painter) {
			box := ui.Rect{W: anLabelPx, H: anLabelPx}
			cp.Clip(box, anLabelPx/2, func() { cp.Image(bmp, box, ui.Cover) })
		})
	}
	v.labels[i].used = now
	return v.labels[i].canvas
}

// anLabelPx is how many pixels across the record's label is drawn at.
const anLabelPx = 320

// anMiniSpectrum paints the little spectrum under the progress: a dot
// where a band is quiet, a bar as tall as it is loud.
func anMiniSpectrum(p *ui.Painter, r ui.Rect, v *AnalysisState, s float32) {
	const n = 56
	slot := r.W / n
	dot := 2.4 * s
	for i := range n {
		b := i * player.AnBands / n
		lv := (v.l[b] + v.r[b]) / 2
		lv = clamp01(lv * lv * 1.6)
		x := r.X + float32(i)*slot + (slot-dot)/2
		h := max(dot, r.H*lv)
		p.Fill(ui.Rect{X: x, Y: r.Y + r.H - h, W: dot, H: h}, white.Alpha(0.45+0.45*lv), dot/2)
	}
}

// anDraw paints the analysis card's insides in r, the card, at scale s.
func anDraw(p *ui.Painter, r ui.Rect, s, W float32, accent ui.Color, now time.Time, pl *player.Player, v *AnalysisState) {
	_, anRC, anLW := anCols(W)
	S := func(x float32) float32 { return x * s }
	// Small type keeps a size it can be read at, however small the rest,
	// but not so large as to crowd the columns in a small window.
	small := func(size float32) float32 { return min(size*0.85, S(size)*1.15) }
	label := func(x, y float32, t string) {
		p.RichText(r.X+S(x), r.Y+S(y), S(400), ui.Span{Text: t, Size: small(12.5), Weight: 700, LetterSpacing: S(1.5), Color: white.Alpha(0.6)})
	}
	right := func(x, y float32, t string, size float32, weight int, color ui.Color, spacing float32) {
		sp := ui.Span{Text: t, Size: size, Weight: weight, Color: color, LetterSpacing: spacing, Features: "tnum"}
		w, _ := p.MeasureText(S(600), sp)
		p.RichText(r.X+S(x)-w, r.Y+S(y), S(600), sp)
	}
	note := func(x, y float32, t string) { right(x, y, t, small(11.5), 600, white.Alpha(0.5), S(1)) }
	tiny := func(x, y float32, t string, align float32) {
		sp := ui.Span{Text: t, Size: small(10), Weight: 600, Color: white.Alpha(0.42), Features: "tnum"}
		w, _ := p.MeasureText(S(200), sp)
		p.RichText(x-w*align, y, S(200), sp)
	}
	lr := anLX + anLW // the left column's right edge

	// The stereo spectrum: the left up, the right down, each band's peak
	// held over it, and the loudest band marked along the top; under it,
	// where the frequencies are.
	label(anLX, 32, "立体声频谱")
	note(lr, 34, "左 ▲ · 右 ▼")
	sp := ui.Rect{X: r.X + S(anLX), Y: r.Y + S(62), W: S(anLW), H: S(112)}
	mid := sp.Y + sp.H/2
	slot := sp.W / player.AnBands
	bw := max(1, slot*0.56)
	cap := max(1, S(2))
	maxH := sp.H/2 - cap - 1
	p.Fill(ui.Rect{X: sp.X, Y: mid, W: sp.W, H: 1}, white.Alpha(0.2), 0)
	shape := func(x float32) float32 { return float32(math.Pow(float64(clamp01(x)), 1.3)) }
	for i := range player.AnBands {
		x := sp.X + float32(i)*slot + (slot-bw)/2
		l, rr := shape(v.l[i]), shape(v.r[i])
		hl, hr := max(1, l*maxH), max(1, rr*maxH)
		p.Fill(ui.Rect{X: x, Y: mid - hl, W: bw, H: hl}, white.Alpha(0.3+0.6*l), 0)
		p.Fill(ui.Rect{X: x, Y: mid + 1, W: bw, H: hr}, white.Alpha(0.2+0.5*rr), 0)
		if pk := shape(v.pkL[i]) * maxH; pk > 2 {
			p.Fill(ui.Rect{X: x, Y: mid - pk - cap - 1, W: bw, H: cap}, accent, 0)
		}
		if pk := shape(v.pkR[i]) * maxH; pk > 2 {
			p.Fill(ui.Rect{X: x, Y: mid + 2 + pk, W: bw, H: cap}, accent.Alpha(0.75), 0)
		}
	}
	if v.peakBand >= 0 {
		p.Fill(ui.Rect{X: sp.X + float32(v.peakBand)*slot, Y: sp.Y - S(6), W: max(2, slot), H: max(2, S(2))}, accent, 0)
	}
	for _, f := range anTicks {
		x := sp.X + sp.W*anPitch(f.hz)
		p.Fill(ui.Rect{X: x, Y: sp.Y + sp.H + S(2), W: 1, H: S(4)}, white.Alpha(0.3), 0)
		tiny(x, sp.Y+sp.H+S(7), f.name, 0.5)
	}

	// The spectrogram, running by: the low bands at its foot, where they
	// are along its right.
	label(anLX, 210, "频谱瀑布")
	note(lr, 212, "近 6 秒")
	gr := ui.Rect{X: r.X + S(anLX), Y: r.Y + S(234), W: S(anLW), H: S(50)}
	anGram(p, now, accent, v, pl)
	p.Fill(gr, white.Alpha(0.04), S(8))
	if g := v.gram[v.front]; g != nil {
		p.Clip(gr, S(8), func() { p.ImageRegion(g.Bitmap(), ui.Rect{}, gr, ui.ImageOptions{}) })
	}
	for _, f := range anTicks {
		if f.hz != 100 && f.hz != 1000 && f.hz != 10000 {
			continue
		}
		y := gr.Y + gr.H*(1-anPitch(f.hz))
		p.Fill(ui.Rect{X: gr.X + gr.W - S(4), Y: y, W: S(4), H: 1}, white.Alpha(0.35), 0)
		tiny(gr.X+gr.W-S(7), y-small(10)*0.65, f.name, 1)
	}

	// The loudness of the last 30 s, short-term, in LUFS; faint lines every
	// 10 LU.
	label(anLX, 300, "响度")
	note(lr, 302, "短期 LUFS · 近 30 秒")
	lg := ui.Rect{X: r.X + S(anLX), Y: r.Y + S(324), W: S(anLW), H: S(68)}
	p.Fill(lg, white.Alpha(0.04), S(8))
	pad := lg.H * 0.08
	yOf := func(lv float32) float32 { return lg.Y + pad + (1-lv)*(lg.H-2*pad) }
	for _, l := range []float64{-10, -20} {
		y := yOf(lufsShown(l))
		p.Fill(ui.Rect{X: lg.X + S(26), Y: y, W: lg.W - S(26), H: 1}, white.Alpha(0.07), 0)
		tiny(lg.X+S(6), y-small(10)*0.65, fmt.Sprint(l), 0)
	}
	if v.loudN >= 2 {
		n := len(v.loud)
		step := lg.W / float32(n-1)
		var line, area ui.Path
		x0 := lg.X + float32(n-v.loudN)*step
		var sum float32
		for i := range v.loudN {
			lv := v.loud[(v.loudI-v.loudN+i+2*n)%n]
			sum += lv
			x, y := x0+float32(i)*step, yOf(lv)
			if i == 0 {
				line.MoveTo(x, y)
				area.MoveTo(x, lg.Y+lg.H)
			}
			line.LineTo(x, y)
			area.LineTo(x, y)
		}
		area.LineTo(lg.X+lg.W, lg.Y+lg.H).Close()
		p.Clip(lg, S(8), func() {
			p.FillPath(&area, white.Alpha(0.12))
			p.StrokePath(&line, max(1, S(1.4)), white.Alpha(0.85))
		})
		mean := sum / float32(v.loudN)
		ay := yOf(mean)
		for dx := float32(0); dx < lg.W; dx += S(8) {
			p.Fill(ui.Rect{X: lg.X + dx, Y: ay, W: S(4), H: 1}, accent, 0)
		}
	}

	// The tempo, a beat lighting as it comes; the key beside it.
	label(anRC, 32, "速度")
	dot := S(8)
	p.Fill(ui.Rect{X: r.X + S(anRC+44), Y: r.Y + S(36), W: dot, H: dot}, accent.Alpha(0.2+0.8*v.beat), dot/2)
	measuring := ui.Span{Text: "测量中", Size: small(16), Weight: 500, Color: white.Alpha(0.5)}
	if v.bpm > 0 {
		bp := ui.Span{Text: fmt.Sprint(int(v.bpm + 0.5)), Size: S(40), Weight: 300, Color: white, Features: "tnum"}
		bw, _ := p.MeasureText(S(300), bp)
		p.RichText(r.X+S(anRC), r.Y+S(52), S(300), bp)
		p.RichText(r.X+S(anRC)+bw+S(6), r.Y+S(76), S(80), ui.Span{Text: "BPM", Size: small(11.5), Weight: 600, Color: white.Alpha(0.5), LetterSpacing: S(1)})
	} else {
		p.RichText(r.X+S(anRC), r.Y+S(66), S(300), measuring)
	}
	kx := anRC + 165
	label(kx, 32, "调性")
	if v.key != "" {
		p.RichText(r.X+S(kx), r.Y+S(56), S(200), ui.Span{Text: v.key, Size: S(30), Weight: 400, Color: white})
		sure := "把握低"
		switch {
		case v.keySure > 0.12:
			sure = "把握高"
		case v.keySure > 0.05:
			sure = "把握中"
		}
		note(W-41, 34, sure)
	} else {
		p.RichText(r.X+S(kx), r.Y+S(66), S(200), measuring)
	}

	// The channels' levels, their peaks held; the balance; the width.
	meter := func(y float32, name string, fill, at float32, hold float32) {
		label(anRC, y-10, name)
		bar := ui.Rect{X: r.X + S(anRC+65), Y: r.Y + S(y), W: S(W - 41 - anRC - 65), H: max(2, S(4))}
		p.Fill(bar, white.Alpha(0.14), bar.H/2)
		if fill >= 0 {
			p.Fill(ui.Rect{X: bar.X, Y: bar.Y, W: bar.W * clamp01(fill), H: bar.H}, white.Alpha(0.85), bar.H/2)
		}
		if hold > 0.01 {
			p.Fill(ui.Rect{X: bar.X + bar.W*clamp01(hold)*0.99, Y: bar.Y - S(3), W: max(1, S(2)), H: bar.H + S(6)}, accent, 0)
		}
		if at >= 0 {
			d := S(10)
			p.Fill(ui.Rect{X: bar.X + bar.W*at - d/2, Y: bar.Y + bar.H/2 - d/2, W: d, H: d}, accent, d/2)
		}
		if name == "平衡" {
			p.Fill(ui.Rect{X: bar.X + bar.W/2, Y: bar.Y - S(3), W: 1, H: bar.H + S(6)}, white.Alpha(0.35), 0)
		}
	}
	for i, m := range anMeters {
		y := float32(anMeterY + anMeterGap*i)
		switch i {
		case 0:
			meter(y, m.name, v.lvL, -1, v.holdL)
		case 1:
			meter(y, m.name, v.lvR, -1, v.holdR)
		case 2:
			meter(y, m.name, -1, 0.5+max(-1, min(1, v.bal*2.5))*0.46, 0)
		case 3:
			meter(y, m.name, v.wid, -1, 0)
		}
	}

	// The readings, three rows of four: the loudness and the peaks; the
	// bands and how bright; the dynamics, how alike the channels are, the
	// loudest band and the time left.
	vals := anReadings(v, pl)
	for i, rd := range anReads {
		col, row := i%4, i/4
		x := anRC + anReadX[col]
		y := float32(anReadY + anReadGap*row)
		val := vals[i]
		p.RichText(r.X+S(x), r.Y+S(y), S(120), ui.Span{Text: val, Size: S(22), Weight: 500, Color: white, Features: "tnum"})
		if i == anReadPeak && v.peak >= anFull && now.Sub(v.clipAt) < 2*time.Second {
			// It reached full scale just now: a red light by it.
			vw, _ := p.MeasureText(S(120), ui.Span{Text: val, Size: S(22), Weight: 500, Features: "tnum"})
			d := S(7)
			p.Fill(ui.Rect{X: r.X + S(x) + vw + S(4), Y: r.Y + S(y+4), W: d, H: d}, ui.Hex("#ff453a"), d/2)
		}
		p.RichText(r.X+S(x), r.Y+S(y+28), S(120), ui.Span{Text: rd.name, Size: small(11.5), Weight: 600, Color: white.Alpha(0.55)})
	}

	// Paused, it all holds, greyed, and says so.
	if k := v.paused; k > 0.01 {
		p.Fill(r, ui.RGBA(18, 18, 22, 0.42*k), S(34))
		pill := ui.Span{Text: "已暂停", Size: small(12.5), Weight: 700, Color: white.Alpha(0.9 * k), LetterSpacing: S(1)}
		pw, ph := p.MeasureText(S(200), pill)
		b := ui.Rect{X: r.X + r.W/2 - pw/2 - S(12), Y: r.Y + S(14), W: pw + S(24), H: ph + S(8)}
		p.Fill(b, white.Alpha(0.14*k), b.H/2)
		p.RichText(b.X+S(12), b.Y+S(4), S(200), pill)
	}
}

// anTick is a frequency marked on the spectra.
type anTick struct {
	hz   float64
	name string
}

var anTicks = []anTick{{50, "50"}, {100, "100"}, {200, "200"}, {500, "500"}, {1000, "1k"}, {2000, "2k"}, {5000, "5k"}, {10000, "10k"}}

// anPitch is where hz is along the analysis's bands, 0 at 30 Hz to 1 at
// 16 kHz, evenly in pitch, as player.Analysis spaces them.
func anPitch(hz float64) float32 {
	return float32(math.Log(hz/30) / math.Log(16000.0/30))
}

// anHint is something the analysis shows, and what it is, said when the
// pointer rests on it, for a listener who does not know the word.
type anHint struct{ name, hint string }

// The meters of the right column, from anMeterY a gap apart.
const (
	anMeterY   = 128
	anMeterGap = 26
)

var anMeters = []anHint{
	{"左", "左声道此刻的电平，竖线是刚才的峰值"},
	{"右", "右声道此刻的电平，竖线是刚才的峰值"},
	{"平衡", "左右哪边更响：圆点偏左是左声道更响，在中线上是一样响"},
	{"宽度", "立体声有多宽：0 是单声道，越大左右两边差得越多"},
}

// The readings, in rows from anReadY, at anReadX across the right column;
// anReadPeak is the peak's.
const (
	anReadY    = 236
	anReadGap  = 60
	anReadPeak = 2
)

var anReadX = [4]float32{0, 82, 165, 247}

var anReads = []anHint{
	{"短期响度", "最近 3 秒的响度，按 EBU R128 标准测量；流媒体上的歌大多在 -14 左右"},
	{"瞬时响度", "最近 0.4 秒的响度（LUFS），跟着每一拍起伏"},
	{"峰值 dB", "最近 0.3 秒里最大的采样值；0 是满幅，到了 0 就可能削波失真，旁边会亮红点"},
	{"削波", "这首歌到现在有几次达到满幅（0 dBFS），次数越多越可能听到失真"},
	{"低频", "约 100 Hz 以下（鼓、贝斯）的能量"},
	{"中频", "约 100 Hz 到 1 kHz（人声、大部分乐器）的能量"},
	{"高频", "约 1 kHz 以上（镲片、齿音、空气感）的能量"},
	{"亮度", "声音的重心偏高还是偏低，越大听起来越明亮"},
	{"动态 dB", "近 30 秒里响的地方和轻的地方差多少 dB，越大起伏越大，压缩得越狠越小"},
	{"相关", "左右声道有多像：1 是完全一样（单声道），0 是毫不相关，负数说明相位相反"},
	{"主频 Hz", "此刻最响的频率"},
	{"剩余", "这首歌还剩多久"},
}

// anReadings are the readings' values, as anReads names them, "--" for
// what is not measured.
func anReadings(v *AnalysisState, pl *player.Player) [12]string {
	var out [12]string
	for i := range out {
		out[i] = "--"
	}
	if v.lufsOK {
		out[0], out[1] = fmt.Sprintf("%.1f", v.lufsS), fmt.Sprintf("%.1f", v.lufsM)
	}
	if v.peak > 0 {
		out[2] = fmt.Sprintf("%.1f", min(0, 20*math.Log10(float64(v.peak))))
	}
	if v.live || v.peakSong > 0 {
		out[3] = fmt.Sprint(v.clips)
	}
	pct := func(x float32) string { return fmt.Sprint(int(x*100 + 0.5)) }
	out[4], out[5], out[6], out[7] = pct(v.low), pct(v.mid), pct(v.high), pct(v.bright)
	if v.dr > 0 {
		out[8] = fmt.Sprintf("%.1f", v.dr)
	}
	if v.live {
		out[9] = fmt.Sprintf("%.2f", v.corr)
	}
	if v.peakBand >= 0 && v.peakHz > 0 {
		if v.peakHz >= 1000 {
			out[10] = fmt.Sprintf("%.1fk", v.peakHz/1000)
		} else {
			out[10] = fmt.Sprint(int(v.peakHz + 0.5))
		}
	}
	if d := pl.Duration(); d > 0 {
		out[11] = "-" + appui.Clock(d-pl.Position())
	}
	return out
}

// anHints puts boxes over what the analysis card paints, in the card r
// wide at scale s, W the composition's width, each saying what it is when
// the pointer rests on it.
func anHints(c *ui.Context, s, W float32) {
	_, rc, lw := anCols(W)
	box := func(key string, x, y, w, h float32, hint string) {
		ui.Box(c.Key(key)).Absolute().Left(x * s).Top(y * s).Width(w * s).Height(h * s).Tooltip(hint)
	}
	box("an-h-spec", anLX, 26, lw, 168, "立体声频谱：每根柱子是一段频率，向上是左声道、向下是右声道，横线是刚才的峰值，左边低音、右边高音")
	box("an-h-gram", anLX, 204, lw, 84, "频谱瀑布：最近 6 秒的频谱从右往左流过，越亮越响，下面低音、上面高音")
	box("an-h-loud", anLX, 294, lw, 100, "响度：最近 30 秒的短期响度（LUFS）")
	box("an-h-bpm", rc, 26, 150, 80, "速度：每分钟多少拍，从低音的节奏里估出来，圆点随拍子闪")
	box("an-h-key", rc+165, 26, 150, 80, "调性：从各个音出现得多少推测的调；转调、无调性或鼓为主的音乐会不准")
	for i, m := range anMeters {
		box("an-h-m"+m.name, rc, float32(anMeterY+anMeterGap*i)-14, W-41-rc, 24, m.hint)
	}
	for i, rd := range anReads {
		box("an-h-r"+rd.name, rc+anReadX[i%4], float32(anReadY+anReadGap*(i/4)), 78, 46, rd.hint)
	}
}

// anGram moves the spectrogram on a column when the music has moved on,
// the new column the bands as they are, colored from accent, faint where
// quiet, to white where loud.
func anGram(p *ui.Painter, now time.Time, accent ui.Color, v *AnalysisState, pl *player.Player) {
	if v.gram[0] == nil {
		v.gram[0], v.gram[1] = ui.NewCanvas(anGramW, player.AnBands), ui.NewCanvas(anGramW, player.AnBands)
	}
	// Paused, it stands still, as all of the card does.
	if now.Sub(v.gramAt) < 28*time.Millisecond || !v.live && v.lvL+v.lvR < 0.01 || !pl.Playing() {
		return
	}
	v.gramAt = now
	back, front := v.gram[1-v.front], v.gram[v.front]
	back.Draw(p, func(cp *ui.Painter) {
		cp.ImageRegion(front.Bitmap(), ui.Rect{X: 1, W: anGramW - 1, H: player.AnBands}, ui.Rect{W: anGramW - 1, H: player.AnBands}, ui.ImageOptions{})
		for y := range player.AnBands {
			b := player.AnBands - 1 - y
			lv := float32(math.Pow(float64(clamp01((v.l[b]+v.r[b])/2)), 1.8))
			if lv < 0.02 {
				continue
			}
			cp.Fill(ui.Rect{X: anGramW - 1, Y: float32(y), W: 1, H: 1}, accent.Mix(white, lv).Alpha(min(1, lv*1.2)), 0)
		}
	})
	v.front = 1 - v.front
}

// anLyrics paints, under the cards, in r at scale s, the line being sung,
// large, and under it its translation, or else the next line, faint. As
// the wallpaper has them: the line sung, giving way, goes down a little
// and fades, and the next comes back up into its place; the line under it
// fades over to its next on its own.
func anLyrics(p *ui.Painter, r ui.Rect, s float32, tr *qqmusic.Track, pl *player.Player, v *AnalysisState) {
	lines, loaded := pl.Lyrics()
	pos := pl.Position()
	cur, now, next := -1, "", ""
	if len(lines) == 0 {
		if loaded {
			now = "纯音乐，请欣赏"
		}
	} else {
		cur = lyrics.Index(lines, pos)
		if cur >= 0 {
			now = lines[cur].Text
			next = lines[cur].Translation
		}
		if next == "" {
			for j := cur + 1; j < len(lines); j++ {
				if lines[j].Text != "" {
					next = lines[j].Text
					break
				}
			}
		}
	}
	t := p.Now()
	if v.lineTrack != tr.ID {
		// A song's first lines come in as they are.
		v.lineTrack, v.line, v.main, v.mainNext, v.mainAt, v.sub, v.subWas, v.subAt = tr.ID, cur, now, now, time.Time{}, next, "", time.Time{}
	}
	if cur != v.line || now != v.mainNext {
		v.line, v.mainNext = cur, now
		if v.mainAt.IsZero() || t.Sub(v.mainAt) >= anLineOut {
			// Going again, or still: it goes from where it is.
			v.mainAt = t
		}
	}
	if next != v.sub {
		v.subWas, v.sub, v.subAt = v.sub, next, t
	}
	animating := false
	// The line sung: going, then back as the next.
	mainText, mainAlpha, mainDy := v.main, float32(1), float32(0)
	drop := 21 * s
	if !v.mainAt.IsZero() {
		gone := t.Sub(v.mainAt)
		switch {
		case gone < anLineOut:
			k := anSmooth(float32(gone) / float32(anLineOut))
			mainAlpha, mainDy = 1-k, k*drop
			animating = true
		case gone < anLineOut+anLineIn:
			v.main = v.mainNext
			k := easeOutCubic(clamp01(float32(gone-anLineOut) / float32(anLineIn)))
			mainText, mainAlpha, mainDy = v.main, k, (1-k)*drop
			animating = true
		default:
			v.main, v.mainAt = v.mainNext, time.Time{}
			mainText = v.main
		}
	}
	cx := r.X + r.W/2
	line := func(text string, y, size float32, weight int, color ui.Color) {
		if text == "" || color.A == 0 {
			return
		}
		sp := ui.Span{Text: text, Size: size, Weight: weight, Color: color}
		w, _ := p.MeasureText(r.W*2, sp)
		if w > r.W {
			// Too long for the width: smaller, rather than cut.
			sp.Size *= r.W / w
			w, _ = p.MeasureText(r.W*2, sp)
		}
		p.RichText(cx-w/2, y, r.W*2, sp)
	}
	line(mainText, r.Y+mainDy, 42*s, 800, white.Alpha(mainAlpha))
	// The line under it, a fixed way down, for the line sung, as long or
	// short as it is, not to move it.
	subY := r.Y + 42*s*1.45 + 10*s
	k := float32(1)
	if !v.subAt.IsZero() {
		k = clamp01(float32(t.Sub(v.subAt)) / float32(anSubSwap))
		if k < 1 {
			animating = true
		}
	}
	line(v.subWas, subY, 24*s, 500, white.Alpha(0.55*(1-k)))
	line(v.sub, subY, 24*s, 500, white.Alpha(0.55*k))
	if animating {
		p.AnimationFrame()
	} else if cur+1 < len(lines) && pl.Playing() {
		// Painted again when the next line comes.
		p.After(lines[cur+1].At - pos)
	}
}

// anSmooth eases k, 0 to 1, in and out.
func anSmooth(k float32) float32 {
	k = clamp01(k)
	return k * k * (3 - 2*k)
}

// anLevels name QQ Music's levels as its app does.
var anLevels = map[string]string{
	"standard": "标准", "higher": "较高", "exhigh": "极高", "lossless": "无损", "hires": "Hi-Res",
}

// anSourceChips are what is known of where the song plays from, a chip
// each, and whether more is to come (the stream loading).
func anSourceChips(pl *player.Player) (chips []string, more bool) {
	st := pl.Stream()
	if st == nil {
		return nil, true
	}
	if kind := strings.ToUpper(st.Type); kind != "" {
		chips = append(chips, kind)
	}
	if l := anLevels[st.Level]; l != "" {
		chips = append(chips, l)
	}
	rate, bits := pl.SourceFormat()
	switch {
	case rate > 0:
		hz := strconv.FormatFloat(float64(rate)/1000, 'f', -1, 64) + " kHz"
		if bits > 0 {
			hz += fmt.Sprintf(" · %d bit", bits)
		}
		chips = append(chips, hz)
	default:
		more = true
	}
	if d := pl.Duration().Seconds(); st.Size > 0 && d > 1 {
		chips = append(chips, fmt.Sprintf("%d kbps", int(float64(st.Size)*8/d/1000+0.5)))
	}
	return chips, more
}

// anSource paints where what plays comes from, as chips at the right of
// r, at scale s, as many as fit.
func anSource(p *ui.Painter, r ui.Rect, s float32, pl *player.Player) {
	chips, more := anSourceChips(pl)
	if more {
		// Asked again until all is known.
		p.After(time.Second)
	}
	gap := 6 * s
	spans := make([]ui.Span, 0, len(chips))
	var ws []float32
	total := -gap
	th := float32(0)
	for _, t := range chips {
		sp := ui.Span{Text: t, Size: max(10.5, 12.5*s), Weight: 600, Color: white.Alpha(0.78), Features: "tnum"}
		tw, h := p.MeasureText(r.W, sp)
		if total+gap+tw+16*s > r.W {
			break
		}
		total += gap + tw + 16*s
		spans, ws, th = append(spans, sp), append(ws, tw+16*s), h
	}
	x := r.X + r.W - total
	for i, sp := range spans {
		b := ui.Rect{X: x, Y: r.Y + (r.H-th-8*s)/2, W: ws[i], H: th + 8*s}
		p.Fill(b, white.Alpha(0.1), 6*s)
		p.RichText(b.X+8*s, b.Y+4*s, r.W, sp)
		x += b.W + gap
	}
}
