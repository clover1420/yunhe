package player

import (
	"math"
	"testing"
)

// A 1 kHz tone at -20 dBFS in both channels is about -20 LUFS (EBU
// R128's reference, the K-weighting near flat there), and its peak is
// its amplitude; an A held is the pitch class A.
func TestAnalysisLoudnessPeakAndPitch(t *testing.T) {
	const rate = 44100
	tone := func(hz, amp float64, frames int) []float32 {
		out := make([]float32, frames*2)
		for i := range frames {
			v := float32(amp * math.Sin(2*math.Pi*hz*float64(i)/rate))
			out[2*i], out[2*i+1] = v, v
		}
		return out
	}
	SetAnalyzing(true)
	defer SetAnalyzing(false)
	s := newSpectrum(rate)
	const chunks = 200 // ~4.6 s, past the short-term's 3 s
	s.feed(0, tone(1000, 0.1, fftSize*chunks))
	an := s.analyzer()
	per := float64(rate) / fftSize
	for _, n := range []int{int(math.Round(0.4 * per)), int(math.Round(3 * per))} {
		ms, ok := an.loudness(chunks-1, n)
		if !ok {
			t.Fatalf("%d chunks not measured", n)
		}
		if lufs := lufsOf(ms); math.Abs(lufs+20) > 0.6 {
			t.Errorf("over %d chunks: %.2f LUFS, want about -20", n, lufs)
		}
	}
	if p := an.peakOver(chunks-1, 40); math.Abs(float64(p)-0.1) > 0.002 {
		t.Errorf("peak %.4f, want 0.1", p)
	}

	s = newSpectrum(rate)
	s.feed(0, tone(440, 0.3, fftSize*64))
	an = s.analyzer()
	if an.keyN == 0 {
		t.Fatal("no pitch measured")
	}
	best := 0
	for c := range an.chroma {
		if an.chroma[c] > an.chroma[best] {
			best = c
		}
	}
	if best != 9 {
		t.Errorf("440 Hz is pitch class %d, want 9 (A): %.2f", best, an.chroma)
	}
}
