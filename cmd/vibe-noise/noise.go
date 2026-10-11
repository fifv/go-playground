package main

import (
	"math"
	"math/rand/v2"
)

type biquad struct {
	b0, b1, b2, a1, a2 float64
	z1, z2             float64
}

func newBiquad(hz, q float64, highpass bool) biquad {
	w := 2 * math.Pi * hz / sampleRate
	c, s := math.Cos(w), math.Sin(w)
	alpha := s / (2 * q)
	a0 := 1 + alpha
	b0, b1 := (1-c)/2, 1-c
	if highpass {
		b0, b1 = (1+c)/2, -(1 + c)
	}
	return biquad{b0: b0 / a0, b1: b1 / a0, b2: b0 / a0, a1: -2 * c / a0, a2: (1 - alpha) / a0}
}

func (b *biquad) next(x float64) float64 {
	y := b.b0*x + b.z1
	b.z1 = b.b1*x - b.a1*y + b.z2
	b.z2 = b.b2*x - b.a2*y
	return y
}

type generator struct {
	rng     *rand.Rand
	pink    bool
	volume  float64
	state   [7]float64
	filters [4]biquad
	frames  uint64
}

func newGenerator(o options, seed uint64) *generator {
	g := &generator{rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)), pink: o.noise == "pink", volume: o.volume}
	// Two sections at each edge form fourth-order Butterworth filters.
	for i, q := range []float64{0.541196100146197, 1.306562964876377} {
		g.filters[i] = newBiquad(o.low, q, true)
		g.filters[i+2] = newBiquad(o.high, q, false)
	}
	return g
}

func (g *generator) next() float64 {
	x := 2*g.rng.Float64() - 1
	if g.pink {
		// Paul Kellet's refined pink filter, designed for 44.1 kHz:
		// https://www.firstpr.com.au/dsp/pink-noise/
		b := &g.state
		b[0] = 0.99886*b[0] + x*0.0555179
		b[1] = 0.99332*b[1] + x*0.0750759
		b[2] = 0.96900*b[2] + x*0.1538520
		b[3] = 0.86650*b[3] + x*0.3104856
		b[4] = 0.55000*b[4] + x*0.5329522
		b[5] = -0.7616*b[5] - x*0.0168980
		pink := b[0] + b[1] + b[2] + b[3] + b[4] + b[5] + b[6] + x*0.5362
		b[6] = x * 0.115926
		x = pink * 0.11
	}
	for i := range g.filters {
		x = g.filters[i].next(x)
	}
	// Reserve headroom for filter overshoot; bound the final peak by volume.
	x = max(-1, min(1, x*0.5)) * g.volume
	// A short fade-in avoids an abrupt start.
	if g.frames < sampleRate/20 {
		x *= float64(g.frames) / float64(sampleRate/20)
	}
	g.frames++
	return x
}

func (g *generator) fill(samples []int16) {
	for i := 0; i+1 < len(samples); i += 2 {
		x := int16(math.Round(g.next() * 32767))
		samples[i], samples[i+1] = x, x
	}
}
