package player

import (
	"math"
	"math/cmplx"
	"sync"
	"sync/atomic"
)

// The analysis of what plays, for Now Playing's analysis pane: the spectrum
// of each channel in finer bands than the visualizers', how loud each is,
// the peak, how alike the two channels are, and the low end that beats are
// found in. Measured as the song decodes, in the spectrum's chunks, only
// while something asks for it (SetAnalyzing): its ring is made then, so a
// listener who never looks at it keeps no more memory than before.

const (
	// AnBands is how many bands each channel's spectrum is in.
	AnBands = 64
	// anSlots is how many chunks the ring keeps: ~11.9 s at 44.1 kHz,
	// some 80 KB, enough history for the tempo.
	anSlots = 512
)

// analyzing is whether the streams measure the analysis.
var analyzing atomic.Bool

// SetAnalyzing starts or stops measuring the analysis of what plays.
func SetAnalyzing(on bool) { analyzing.Store(on) }

// Analysis is what was measured of a chunk of what plays.
type Analysis struct {
	// L and R are the levels of each channel's bands, low to high,
	// spaced evenly in pitch from 30 Hz to 16 kHz, 0 to 1.
	L, R [AnBands]float32
	// RmsL and RmsR are how loud each channel is, its root mean square,
	// 1 at full scale.
	RmsL, RmsR float32
	// Peak is the largest sample of either channel, 1 at full scale.
	Peak float32
	// Corr is how alike the channels are, as the correlation of their
	// samples: 1 the same (mono), 0 unrelated, -1 opposed in phase.
	Corr float32
	// Low is the low end, under 150 Hz, 0 to 1: where the beats are.
	Low float32
}

// anKeySize is how many frames the pitch classes are measured over: a
// chunk's fftSize is too coarse for the low notes' semitones (43 Hz a bin
// at 44.1 kHz), 4096 resolves them from about 180 Hz.
const anKeySize = 4096

type anSlot struct {
	chunk int64
	l, r  [AnBands]uint8
	rmsL  float32
	rmsR  float32
	peak  float32
	corr  float32
	low   float32
	// kms is the K-weighted mean square of the chunk, the channels summed,
	// as ITU-R BS.1770 weighs them for loudness.
	kms float32
}

type analyzer struct {
	mu     sync.RWMutex
	slots  []anSlot
	edges  [AnBands + 1]int
	fl, fr []complex128
	// The K-weighting of the loudness (loudMeter's filters), each
	// channel's state.
	kPre, kRLB [5]float64
	kState     [2][2][4]float64
	// The pitch classes: the mono samples gathered for the next measure,
	// its window and transform, the pitch class of each bin (-1 for none),
	// and the energy of each class so far, over keyN measures.
	keyAcc   []float32
	keyWin   []float32
	keyFFT   []complex128
	keyClass []int8
	chroma   [12]float64
	keyN     int
	rate     int
}

func newAnalyzer(rate int) *analyzer {
	an := &analyzer{slots: make([]anSlot, anSlots), fl: make([]complex128, fftSize), fr: make([]complex128, fftSize)}
	for i := range an.slots {
		an.slots[i].chunk = -1
	}
	lo, hi := math.Log(30), math.Log(16000)
	for i := range an.edges {
		f := math.Exp(lo + (hi-lo)*float64(i)/AnBands)
		an.edges[i] = max(1, min(fftSize/2-1, int(f*fftSize/float64(rate))))
	}
	for i := 1; i < len(an.edges); i++ {
		an.edges[i] = max(an.edges[i], an.edges[i-1]+1)
	}
	km := newLoudMeter(rate, 2)
	an.kPre, an.kRLB, an.rate = km.preB, km.rlbB, rate
	an.keyAcc = make([]float32, 0, anKeySize)
	an.keyWin = make([]float32, anKeySize)
	an.keyFFT = make([]complex128, anKeySize)
	an.keyClass = make([]int8, anKeySize/2)
	for i := range an.keyWin {
		an.keyWin[i] = float32(0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(anKeySize-1)))
	}
	for k := range an.keyClass {
		an.keyClass[k] = -1
		f := float64(k) * float64(rate) / anKeySize
		// From G3 to C8: under it a bin spans more than a semitone, over
		// it is little but overtones.
		if f < 190 || f > 4200 {
			continue
		}
		midi := 69 + 12*math.Log2(f/440)
		an.keyClass[k] = int8((int(math.Round(midi))%12 + 12) % 12)
	}
	return an
}

// lowBands is how many of the bands are under 150 Hz, at most.
var lowBands = int(math.Ceil(AnBands * math.Log(150.0/30) / math.Log(16000.0/30)))

// analyze measures the chunk of samples l and r, windowed by window.
func (an *analyzer) analyze(chunk int64, l, r []float32, window *[fftSize]float32) {
	var sl, sr, slr float64
	var peak float32
	for i := range fftSize {
		a, b := l[i], r[i]
		sl += float64(a * a)
		sr += float64(b * b)
		slr += float64(a * b)
		peak = max(peak, abs32(a), abs32(b))
		an.fl[i] = complex(float64(a*window[i]), 0)
		an.fr[i] = complex(float64(b*window[i]), 0)
	}
	fft(an.fl)
	fft(an.fr)
	slot := anSlot{chunk: chunk, peak: peak}
	// The loudness: each channel K-weighted, as the ear weighs it.
	var kms float64
	for c, x := range [2][]float32{l[:fftSize], r[:fftSize]} {
		st := &an.kState[c]
		for _, v := range x {
			y := kStage(&an.kRLB, &st[1], kStage(&an.kPre, &st[0], float64(v)))
			kms += y * y
		}
	}
	slot.kms = float32(kms / fftSize)
	an.pitch(l[:fftSize], r[:fftSize])
	slot.rmsL = float32(math.Sqrt(sl / fftSize))
	slot.rmsR = float32(math.Sqrt(sr / fftSize))
	if den := math.Sqrt(sl * sr); den > 1e-9 {
		slot.corr = float32(slr / den)
	}
	level := func(x []complex128, b int) float64 {
		var sum float64
		for k := an.edges[b]; k < an.edges[b+1]; k++ {
			sum += cmplx.Abs(x[k])
		}
		mag := sum / float64(an.edges[b+1]-an.edges[b]) / (fftSize / 4)
		// Tilted up a little as the bands rise, as the ear hears them.
		mag *= 1 + float64(b)/24
		db := 20 * math.Log10(mag+1e-9)
		// A loud master's bands sit about -30 dB, from -62 dB, so that the
		// loud sits about the middle with room over it, and the quiet shows.
		return math.Max(0, math.Min(1, (db+62)/70))
	}
	var low float64
	for b := range AnBands {
		vl, vr := level(an.fl, b), level(an.fr, b)
		slot.l[b], slot.r[b] = uint8(vl*255+0.5), uint8(vr*255+0.5)
		if b < lowBands {
			low += (vl + vr) / 2
		}
	}
	slot.low = float32(low / float64(lowBands))
	an.mu.Lock()
	an.slots[chunk%anSlots] = slot
	an.mu.Unlock()
}

// pitch gathers the chunk l, r for the pitch classes, and measures them
// once it has anKeySize frames: each class's energy, the measure's
// loudest made 1, so that loud and quiet passages count alike.
func (an *analyzer) pitch(l, r []float32) {
	for i := range l {
		an.keyAcc = append(an.keyAcc, (l[i]+r[i])/2)
	}
	if len(an.keyAcc) < anKeySize {
		return
	}
	for i, v := range an.keyAcc {
		an.keyFFT[i] = complex(float64(v*an.keyWin[i]), 0)
	}
	an.keyAcc = an.keyAcc[:0]
	fft(an.keyFFT)
	var cls [12]float64
	var top float64
	for k, c := range an.keyClass {
		if c < 0 {
			continue
		}
		m := cmplx.Abs(an.keyFFT[k])
		cls[c] += m * m
	}
	for _, v := range cls {
		top = max(top, v)
	}
	if top < 1e-6 {
		return // silence says nothing of the key
	}
	an.mu.Lock()
	for c, v := range cls {
		an.chroma[c] += math.Sqrt(v / top)
	}
	an.keyN++
	an.mu.Unlock()
}

// loudness returns the mean square, K-weighted, of the n chunks up to last,
// or false unless all were measured.
func (an *analyzer) loudness(last int64, n int) (float64, bool) {
	an.mu.RLock()
	defer an.mu.RUnlock()
	var sum float64
	for c := last - int64(n) + 1; c <= last; c++ {
		if c < 0 || an.slots[c%anSlots].chunk != c {
			return 0, false
		}
		sum += float64(an.slots[c%anSlots].kms)
	}
	return sum / float64(n), true
}

// peakOver returns the largest peak of the n chunks up to last, of those
// measured.
func (an *analyzer) peakOver(last int64, n int) float32 {
	an.mu.RLock()
	defer an.mu.RUnlock()
	var p float32
	for c := max(0, last-int64(n)+1); c <= last; c++ {
		if s := &an.slots[c%anSlots]; s.chunk == c {
			p = max(p, s.peak)
		}
	}
	return p
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// at returns the analysis of chunk, or false if it was not measured.
func (an *analyzer) at(chunk int64) (Analysis, bool) {
	if chunk < 0 {
		return Analysis{}, false
	}
	an.mu.RLock()
	defer an.mu.RUnlock()
	s := &an.slots[chunk%anSlots]
	if s.chunk != chunk {
		return Analysis{}, false
	}
	out := Analysis{RmsL: s.rmsL, RmsR: s.rmsR, Peak: s.peak, Corr: s.corr, Low: s.low}
	for b := range AnBands {
		out.L[b], out.R[b] = float32(s.l[b])/255, float32(s.r[b])/255
	}
	return out, true
}

// lows fills dst with the low end of the chunks up to last, oldest first,
// and returns how many of the latest were measured without a gap.
func (an *analyzer) lows(last int64, dst []float32) int {
	an.mu.RLock()
	defer an.mu.RUnlock()
	n := 0
	for i := len(dst) - 1; i >= 0; i-- {
		c := last - int64(len(dst)-1-i)
		if c < 0 || an.slots[c%anSlots].chunk != c {
			break
		}
		dst[i] = an.slots[c%anSlots].low
		n++
	}
	return n
}

// Analysis returns the analysis of what plays now, if it was measured.
func (p *Player) Analysis() (Analysis, bool) {
	an, chunk := p.analyzerNow()
	if an == nil {
		return Analysis{}, false
	}
	return an.at(chunk)
}

// AnalysisLows fills dst with the low end of the chunks up to the one
// playing, oldest first, and returns how many of the latest were measured,
// and how many chunks there are a second.
func (p *Player) AnalysisLows(dst []float32) (n int, perSecond float64) {
	an, chunk := p.analyzerNow()
	if an == nil {
		return 0, 0
	}
	return an.lows(chunk, dst), float64(p.stream.SampleRate()) / fftSize
}

func (p *Player) analyzerNow() (*analyzer, int64) {
	if p.stream == nil || p.out == nil || !p.playing {
		return nil, 0
	}
	_, spec := p.stream.source()
	if spec == nil {
		return nil, 0
	}
	an := spec.analyzer()
	if an == nil {
		return nil, 0
	}
	frame := int64(p.Position().Seconds() * float64(p.stream.SampleRate()))
	return an, frame / fftSize
}

// AnalysisBandHz returns the frequency in the middle of band b of the
// analysis of what plays, or 0 if it is not measured.
func (p *Player) AnalysisBandHz(b int) float32 {
	an, _ := p.analyzerNow()
	if an == nil || b < 0 || b >= AnBands {
		return 0
	}
	mid := float32(an.edges[b]+an.edges[b+1]) / 2
	return mid * float32(p.stream.SampleRate()) / fftSize
}

// AnalysisLoudness returns how loud what plays is, in LUFS, as EBU R128
// has it: momentary, over the last 400 ms, and short-term, over the last
// 3 s; false while either has not been measured whole.
func (p *Player) AnalysisLoudness() (momentary, shortTerm float64, ok bool) {
	an, chunk := p.analyzerNow()
	if an == nil {
		return 0, 0, false
	}
	per := float64(an.rate) / fftSize
	m, ok1 := an.loudness(chunk, max(1, int(math.Round(0.4*per))))
	s, ok2 := an.loudness(chunk, min(anSlots-1, int(math.Round(3*per))))
	if !ok1 || !ok2 {
		return 0, 0, false
	}
	lufs := func(ms float64) float64 {
		if ms <= 1e-12 {
			return -70
		}
		return max(-70, lufsOf(ms))
	}
	return lufs(m), lufs(s), true
}

// AnalysisPeak returns the largest sample of the last d of what plays, 1
// at full scale, as decoded, before the volume.
func (p *Player) AnalysisPeak(seconds float64) float32 {
	an, chunk := p.analyzerNow()
	if an == nil {
		return 0
	}
	return an.peakOver(chunk, max(1, int(seconds*float64(an.rate)/fftSize)))
}

// AnalysisChroma returns the energy of each pitch class, C first, over
// what of the song was measured, and how many seconds that was.
func (p *Player) AnalysisChroma() ([12]float32, float64) {
	an, _ := p.analyzerNow()
	var out [12]float32
	if an == nil {
		return out, 0
	}
	an.mu.RLock()
	defer an.mu.RUnlock()
	if an.keyN == 0 {
		return out, 0
	}
	for c, v := range an.chroma {
		out[c] = float32(v / float64(an.keyN))
	}
	return out, float64(an.keyN) * anKeySize / float64(an.rate)
}

// formatOf is a stream that knows its source's format: its rate, and its
// bits a sample.
type formatOf interface {
	format() (rate, bits int)
}

// SourceFormat returns the format of what plays: its sample rate, and its
// bits a sample, 0 where the format has none (lossy ones).
func (p *Player) SourceFormat() (rate, bits int) {
	s := p.stream
	if s == nil {
		return 0, 0
	}
	if f, ok := s.(formatOf); ok {
		return f.format()
	}
	return s.SampleRate(), 0
}
