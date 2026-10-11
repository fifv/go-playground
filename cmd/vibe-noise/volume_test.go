package main

import (
	"math"
	"strings"
	"testing"
)

func TestVolumeAdjustment(t *testing.T) {
	for _, initial := range []float64{.001, .2, .8} {
		up := adjustVolume(initial, 1)
		if got := volumeDB(up) - volumeDB(initial); math.Abs(got-.1) > 1e-10 {
			t.Errorf("Up changed volume by %g dB", got)
		}
		if got := adjustVolume(up, -1); math.Abs(got-initial) > 1e-12 {
			t.Errorf("Up/Down did not return to %g: %g", initial, got)
		}
		if got := volumeDB(adjustVolume(initial, -5)) - volumeDB(initial); math.Abs(got+.5) > 1e-10 {
			t.Errorf("held Down changed volume by %g dB", got)
		}
	}
	if got := adjustVolume(.99, 100); got != 1 {
		t.Errorf("upper limit: %g", got)
	}
	if got := adjustVolume(0, -1); got != 0 {
		t.Errorf("Down unmuted: %g", got)
	}
	unmuted := adjustVolume(0, 1)
	if got := volumeDB(unmuted); math.Abs(got-minimumVolumeDB) > 1e-10 {
		t.Errorf("unmute level: %g", got)
	}
	if got := adjustVolume(unmuted, -1); got != 0 {
		t.Errorf("could not mute: %g", got)
	}
	if got := volumeDB(adjustVolume(0, 3)); math.Abs(got-(minimumVolumeDB+.2)) > 1e-10 {
		t.Errorf("held Up from mute: %g", got)
	}
	if got := volumeStatus(.2); got != "Volume: 20.00% (-14.0 dB)" {
		t.Errorf("status: %s", got)
	}
	if got := volumeStatus(0); !strings.Contains(got, "-Inf dB") {
		t.Errorf("mute status: %s", got)
	}
}

func TestLiveVolumeKeepsSignalAndChannels(t *testing.T) {
	o := options{noise: "pink", volume: .2, low: 300, high: 3000}
	g, reference := newGenerator(o, 123), newGenerator(o, 123)
	pcm := make([]int16, 4096)
	baseline := make([]int16, len(pcm))
	g.fill(pcm)
	reference.fill(baseline)
	g.volume = adjustVolume(g.volume, 10)
	g.fill(pcm)
	reference.fill(baseline)
	factor := math.Pow(10, 1.0/20)
	for i := 0; i < len(pcm); i += 2 {
		if pcm[i] != pcm[i+1] {
			t.Fatal("live adjustment changed stereo matching")
		}
		if math.Abs(float64(pcm[i])-float64(baseline[i])*factor) > 2 {
			t.Fatal("live adjustment reset or altered the noise sequence")
		}
	}
}
