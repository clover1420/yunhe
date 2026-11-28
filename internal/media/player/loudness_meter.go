package player

import "math"

// loudMeter holds ITU-R BS.1770's K-weighting, which loudness is measured
// through: a high shelf for the head's response to treble, then the high
// pass that leaves out the rumble below hearing. Each stage is one
// biquad, its b and a coefficients in one array.
type loudMeter struct {
	preB, rlbB [5]float64 // b0, b1, b2, a1, a2
}

func newLoudMeter(rate, channels int) *loudMeter {
	return &loudMeter{preB: shelfCoefficients(rate), rlbB: highPassCoefficients(rate)}
}

// shelfCoefficients is BS.1770's high shelf, at rate.
func shelfCoefficients(rate int) [5]float64 {
	const f0, gain, q = 1681.974450955533, 3.999843853973347, 0.7071752369554196
	k := math.Tan(math.Pi * f0 / float64(rate))
	vh := math.Pow(10, gain/20)
	vb := math.Pow(vh, 0.4996667741545416)
	a0 := 1 + k/q + k*k
	return [5]float64{
		(vh + vb*k/q + k*k) / a0,
		2 * (k*k - vh) / a0,
		(vh - vb*k/q + k*k) / a0,
		2 * (k*k - 1) / a0,
		(1 - k/q + k*k) / a0,
	}
}

// highPassCoefficients is BS.1770's high pass, at rate.
func highPassCoefficients(rate int) [5]float64 {
	const f0, q = 38.13547087602444, 0.5003270373238773
	k := math.Tan(math.Pi * f0 / float64(rate))
	a0 := 1 + k/q + k*k
	return [5]float64{
		1 / a0,
		-2 / a0,
		1 / a0,
		2 * (k*k - 1) / a0,
		(1 - k/q + k*k) / a0,
	}
}

// kStage runs x through one biquad of b, its state in s: x1, x2, y1, y2.
func kStage(b *[5]float64, s *[4]float64, x float64) float64 {
	y := b[0]*x + b[1]*s[0] + b[2]*s[1] - b[3]*s[2] - b[4]*s[3]
	s[0], s[1] = x, s[0]
	s[2], s[3] = y, s[2]
	return y
}

// lufsOf returns the loudness of a K-weighted mean square, in LUFS, as
// BS.1770 has it.
func lufsOf(ms float64) float64 {
	if ms <= 0 {
		return -70
	}
	return max(-70, -0.691+10*math.Log10(ms))
}
