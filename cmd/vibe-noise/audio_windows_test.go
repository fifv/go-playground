package main

import (
	"slices"
	"testing"
)

// Exercise real WinMM buffer ownership without playing nonzero audio. Every
// buffer inspection happens while paused; the only restart uses silent PCM.
func TestPausedVolumeRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("requires a Windows audio output")
	}
	for _, noise := range []string{"pink", "white"} {
		t.Run(noise, func(t *testing.T) {
			o := options{noise: noise, volume: .2, low: 300, high: 3000}
			p, err := openPlayer(outputDevice{waveMapper, "test default output"}, o, 42)
			if err != nil {
				t.Skipf("audio output unavailable: %v", err)
			}
			t.Cleanup(func() {
				if err := p.close(); err != nil {
					t.Error(err)
				}
			})
			reference := newGenerator(o, 42)
			expected := make([]int16, bufferFrames*2)
			for range p.buffers {
				reference.fill(expected)
			}

			// Normal pause/resume must preserve the queued samples and position.
			frames := p.generator.frames
			original := slices.Clone(p.buffers[0].samples)
			if err := p.prepareResume(); err != nil {
				t.Fatal(err)
			}
			if p.generator.frames != frames || !slices.Equal(original, p.buffers[0].samples) {
				t.Fatal("unchanged volume unnecessarily replaced queued audio")
			}

			// Test quieter, louder, muted and unmuted queues, including a ring
			// cursor that has advanced beyond buffer zero before the refresh.
			for _, volume := range []float64{.01, .8, 0, .2, 0} {
				p.nextBuffer = 2
				p.generator.volume = volume
				reference.volume = volume
				if err := p.prepareResume(); err != nil {
					t.Fatal(err)
				}
				if p.nextBuffer != 0 {
					t.Fatal("queue cursor was not reset to the first fresh buffer")
				}
				for i, b := range p.buffers {
					reference.fill(expected)
					if !slices.Equal(b.samples, expected) {
						t.Fatalf("buffer %d retained stale PCM or reset the noise sequence at volume %g", i, volume)
					}
					if b.header.Flags&whdrDone != 0 {
						t.Fatalf("buffer %d was not requeued", i)
					}
				}
			}
			// Only silent buffers may reach playback during this test.
			if err := p.setPaused(false); err != nil {
				t.Fatal(err)
			}
			if err := p.setPaused(true); err != nil {
				t.Fatal(err)
			}
		})
	}
}
