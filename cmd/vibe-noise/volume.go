package main

import (
	"fmt"
	"math"
)

const (
	volumeStepDB    = 0.1
	minimumVolumeDB = -80.0
)

func volumeDB(volume float64) float64 {
	return 20 * math.Log10(volume)
}

func volumeStatus(volume float64) string {
	return fmt.Sprintf("Volume: %.2f%% (%+.1f dB)", volume*100, volumeDB(volume))
}

func adjustVolume(volume float64, steps int) float64 {
	if steps == 0 {
		return volume
	}
	db := volumeDB(volume)
	if volume == 0 {
		if steps < 0 {
			return 0
		}
		// The first Up press leaves mute at the bottom of the adjustment range.
		db = minimumVolumeDB
		steps--
	}
	db += float64(steps) * volumeStepDB
	if steps < 0 && db < minimumVolumeDB {
		return 0
	}
	return math.Pow(10, min(0, db)/20)
}
