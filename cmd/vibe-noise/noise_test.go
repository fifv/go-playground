package main

import (
	"math"
	"slices"
	"testing"
)

func TestStereoAndVolume(t *testing.T) {
	for _, color := range []string{"pink", "white"} {
		o := options{noise: color, volume: .2, low: 300, high: 3000}
		g := newGenerator(o, 1)
		pcm := make([]int16, sampleRate*2)
		g.fill(pcm)
		nonzero := false
		for i := 0; i < len(pcm); i += 2 {
			if pcm[i] != pcm[i+1] {
				t.Fatalf("%s: channels differ at %d", color, i)
			}
			if math.Abs(float64(pcm[i])) > math.Ceil(.2*32767) {
				t.Fatalf("sample exceeds volume: %d", pcm[i])
			}
			nonzero = nonzero || pcm[i] != 0
		}
		if !nonzero {
			t.Fatalf("%s is silent", color)
		}
		// Different device buffer boundaries must not change the sequence.
		other := newGenerator(o, 1)
		chunked := make([]int16, len(pcm))
		other.fill(chunked[:2000])
		other.fill(chunked[2000:])
		if !slices.Equal(pcm, chunked) {
			t.Fatal("sequence depends on buffer boundaries")
		}
		o.volume = 0
		newGenerator(o, 1).fill(pcm)
		for _, x := range pcm {
			if x != 0 {
				t.Fatal("zero volume is not silent")
			}
		}
	}
}

func TestFilterBand(t *testing.T) {
	response := func(hz float64) float64 {
		g := newGenerator(options{low: 300, high: 3000}, 1)
		var energy float64
		for i := 0; i < sampleRate*2; i++ {
			x := math.Sin(2 * math.Pi * hz * float64(i) / sampleRate)
			for j := range g.filters {
				x = g.filters[j].next(x)
			}
			if i >= sampleRate {
				energy += x * x
			}
		}
		return math.Sqrt(energy / sampleRate)
	}
	pass := response(1000)
	if pass < .65 {
		t.Fatalf("passband attenuated: %g", pass)
	}
	for _, hz := range []float64{50, 10000} {
		if ratio := response(hz) / pass; ratio > .01 {
			t.Errorf("insufficient rejection at %g Hz: %g", hz, ratio)
		}
	}
}

func TestNoiseSpectra(t *testing.T) {
	for _, color := range []string{"pink", "white"} {
		g := newGenerator(options{noise: color, volume: 1, low: 20, high: 20000}, 42)
		for range sampleRate {
			g.next()
		} // Settle filters and fade-in.
		samples := make([]float64, sampleRate*2)
		for i := range samples {
			samples[i] = g.next()
		}
		bandPower := func(center float64) float64 {
			var power float64
			for offset := -100.; offset <= 100; offset += 5 {
				coefficient := 2 * math.Cos(2*math.Pi*(center+offset)/sampleRate)
				var s1, s2 float64
				for _, x := range samples {
					s := x + coefficient*s1 - s2
					s2, s1 = s1, s
				}
				power += s1*s1 + s2*s2 - coefficient*s1*s2
			}
			return power
		}
		ratio := bandPower(600) / bandPower(2400)
		t.Logf("%s power ratio at 600/2400 Hz: %.3f", color, ratio)
		// Equal-width bands: white has equal power; pink has a 1/f slope.
		if color == "white" && (ratio < .5 || ratio > 2) {
			t.Errorf("white spectrum is not flat: %g", ratio)
		}
		if color == "pink" && (ratio < 2.5 || ratio > 7) {
			t.Errorf("pink spectrum lacks 1/f slope: %g", ratio)
		}
	}
}
