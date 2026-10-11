package main

import (
	"testing"
	"unsafe"
)

func TestKeyboardControls(t *testing.T) {
	if unsafe.Sizeof(inputRecord{}) != 20 {
		t.Fatal("incorrect Windows INPUT_RECORD layout")
	}
	for _, tc := range []struct {
		name   string
		record inputRecord
		want   control
	}{
		{"enter", inputRecord{EventType: 1, KeyDown: 1, VirtualKey: 0x0D, RepeatCount: 1}, control{stop: true}},
		{"up", inputRecord{EventType: 1, KeyDown: 1, VirtualKey: 0x26, RepeatCount: 1}, control{volumeSteps: 1}},
		{"left alt up", inputRecord{EventType: 1, KeyDown: 1, VirtualKey: 0x26, RepeatCount: 1, ControlKeyState: 2}, control{volumeSteps: 10}},
		{"right alt down held", inputRecord{EventType: 1, KeyDown: 1, VirtualKey: 0x28, RepeatCount: 3, ControlKeyState: 1}, control{volumeSteps: -30}},
		{"space", inputRecord{EventType: 1, KeyDown: 1, VirtualKey: 0x20, RepeatCount: 1}, control{togglePause: true}},
		{"space release", inputRecord{EventType: 1, VirtualKey: 0x20}, control{}},
		{"down held", inputRecord{EventType: 1, KeyDown: 1, VirtualKey: 0x28, RepeatCount: 5}, control{volumeSteps: -5}},
		{"release ignored", inputRecord{EventType: 1, VirtualKey: 0x26, RepeatCount: 1}, control{}},
		{"mouse ignored", inputRecord{EventType: 2, KeyDown: 1, VirtualKey: 0x26}, control{}},
		{"other key ignored", inputRecord{EventType: 1, KeyDown: 1, VirtualKey: 0x41}, control{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.record.control(); got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}
