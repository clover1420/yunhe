package player

import (
	"math"
	"testing"
)

// A tone in the left channel alone shows in the left's spectrum at its
// pitch; the same tone in both is one channel (correlation 1), and in
// opposite phase, -1. Nothing is measured while no one asks.
func TestAnalysisChannelsAndCorrelation(t *testing.T) {
	const rate = 44100
	tone := func(l, r float32) []float32 {
		out := make([]float32, fftSize*4*2)
		for i := 0; i < len(out)/2; i++ {
			v := float32(0.5 * math.Sin(2*math.Pi*440*float64(i)/rate))
			out[2*i], out[2*i+1] = v*l, v*r
		}
		return out
	}
	s := newSpectrum(rate)
	s.feed(0, tone(1, 0))
	if s.analyzer() != nil {
		t.Fatal("measured with no one asking")
	}
	SetAnalyzing(true)
	defer SetAnalyzing(false)
	s.feed(fftSize*4, tone(1, 0))
	an, ok := s.analyzer().at(5)
	if !ok {
		t.Fatal("chunk 5 not measured")
	}
	bin, band := 440*fftSize/rate, 0
	for b := range AnBands {
		if s.analyzer().edges[b] <= bin && bin < s.analyzer().edges[b+1] {
			band = b
		}
	}
	if an.L[band] < 0.6 || an.R[band] > 0.05 {
		t.Errorf("440 Hz in the left: band %d L %.2f R %.2f", band, an.L[band], an.R[band])
	}
	if an.RmsL < 0.3 || an.RmsR != 0 || an.Peak < 0.49 {
		t.Errorf("rms %.2f %.2f peak %.2f", an.RmsL, an.RmsR, an.Peak)
	}
	s.feed(fftSize*8, tone(1, 1))
	if an, _ := s.analyzer().at(9); an.Corr < 0.99 {
		t.Errorf("same in both: correlation %.2f", an.Corr)
	}
	s.feed(fftSize*12, tone(1, -1))
	if an, _ := s.analyzer().at(13); an.Corr > -0.99 {
		t.Errorf("opposed: correlation %.2f", an.Corr)
	}
	lows := make([]float32, 6)
	if n := s.analyzer().lows(15, lows); n != 6 {
		t.Errorf("%d lows measured in a row, want 6", n)
	}
}
